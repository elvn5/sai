package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/sanitize"
	"github.com/elvn5/sai/services/pkg/validation"
)

const (
	tourNotFound  = "Tour not found"
	tourSlugTaken = "Slug is already used for this locale"
)

// tourRequest is updateTourSchema. Like NestJS, an update doesn't re-check
// that endDate >= startDate (the refine is on the create schema only).
type tourRequest struct {
	Locale             *string                  `json:"locale" binding:"omitempty,oneof=ru ky en de"`
	Slug               *string                  `json:"slug" binding:"omitempty,min=1,max=255,slug"`
	Title              *string                  `json:"title" binding:"omitempty,min=1,max=255"`
	Excerpt            validation.Field[string] `json:"excerpt" binding:"omitempty,max=500"`
	Content            *string                  `json:"content" binding:"omitempty,min=1,max=20000"`
	Price              validation.Field[int]    `json:"price" binding:"omitempty,gte=0"`
	DurationDays       validation.Field[int]    `json:"durationDays" binding:"omitempty,gt=0"`
	Location           validation.Field[string] `json:"location" binding:"omitempty,max=255"`
	StartDate          *string                  `json:"startDate" binding:"omitempty,isodate"`
	EndDate            *string                  `json:"endDate" binding:"omitempty,isodate"`
	Destination        validation.Field[string] `json:"destination" binding:"omitempty,oneof=issyk_kul karakol bishkek ala_archa song_kol naryn tash_rabat kel_suu osh arslanbob sary_chelek talas batken chuy"`
	CancellationPolicy validation.Field[string] `json:"cancellationPolicy" binding:"omitempty,oneof=free flexible strict non_refundable"`
	CoverImageURL      validation.Field[string] `json:"coverImageUrl" binding:"omitempty,max=2000"`
	Photos             []string                 `json:"photos" binding:"omitempty,max=20,dive,max=2000,mediaurl"`
	Program            []programDay             `json:"program" binding:"omitempty,max=30,dive"`
	Capacity           validation.Field[int]    `json:"capacity" binding:"omitempty,gt=0,lte=500"`
	Phone              validation.Field[string] `json:"phone" binding:"omitempty,max=32,phone"`
	Whatsapp           validation.Field[string] `json:"whatsapp" binding:"omitempty,max=32,phone"`
	Telegram           validation.Field[string] `json:"telegram" binding:"omitempty,max=33,telegram"`
}

type createTourRequest struct {
	Locale             *string                  `json:"locale" binding:"required,oneof=ru ky en de"`
	Slug               *string                  `json:"slug" binding:"required,min=1,max=255,slug"`
	Title              *string                  `json:"title" binding:"required,min=1,max=255"`
	Excerpt            validation.Field[string] `json:"excerpt" binding:"omitempty,max=500"`
	Content            *string                  `json:"content" binding:"required,min=1,max=20000"`
	Price              validation.Field[int]    `json:"price" binding:"omitempty,gte=0"`
	DurationDays       validation.Field[int]    `json:"durationDays" binding:"omitempty,gt=0"`
	Location           validation.Field[string] `json:"location" binding:"omitempty,max=255"`
	StartDate          *string                  `json:"startDate" binding:"required,isodate"`
	EndDate            *string                  `json:"endDate" binding:"required,isodate"`
	Destination        validation.Field[string] `json:"destination" binding:"omitempty,oneof=issyk_kul karakol bishkek ala_archa song_kol naryn tash_rabat kel_suu osh arslanbob sary_chelek talas batken chuy"`
	CancellationPolicy validation.Field[string] `json:"cancellationPolicy" binding:"omitempty,oneof=free flexible strict non_refundable"`
	CoverImageURL      validation.Field[string] `json:"coverImageUrl" binding:"omitempty,max=2000"`
	Photos             []string                 `json:"photos" binding:"omitempty,max=20,dive,max=2000,mediaurl"`
	Program            []programDay             `json:"program" binding:"omitempty,max=30,dive"`
	Capacity           validation.Field[int]    `json:"capacity" binding:"omitempty,gt=0,lte=500"`
	Phone              validation.Field[string] `json:"phone" binding:"omitempty,max=32,phone"`
	Whatsapp           validation.Field[string] `json:"whatsapp" binding:"omitempty,max=32,phone"`
	Telegram           validation.Field[string] `json:"telegram" binding:"omitempty,max=33,telegram"`
}

type programDay struct {
	Title string `json:"title" binding:"required,min=1,max=120"`
	Text  string `json:"text" binding:"max=2000"`
}

func (r createTourRequest) asUpdate() tourRequest {
	// Same fields, different validation tags.
	return tourRequest(r)
}

func (r tourRequest) apply(t *models.Tour) {
	if r.Locale != nil {
		t.Locale = *r.Locale
	}
	if r.Slug != nil {
		t.Slug = *r.Slug
	}
	if r.Title != nil {
		t.Title = *r.Title
	}
	if r.Content != nil {
		t.Content = sanitize.HTML(*r.Content)
	}
	if r.StartDate != nil {
		t.StartDate = *r.StartDate
	}
	if r.EndDate != nil {
		t.EndDate = *r.EndDate
	}
	applyField(&t.Excerpt, r.Excerpt)
	applyField(&t.Price, r.Price)
	applyField(&t.DurationDays, r.DurationDays)
	applyField(&t.Location, r.Location)
	applyField(&t.Destination, r.Destination)
	applyField(&t.CancellationPolicy, r.CancellationPolicy)
	applyField(&t.CoverImageURL, r.CoverImageURL)
	if r.Photos != nil {
		t.Photos = r.Photos
	}
	if r.Program != nil {
		t.Program = make([]models.ProgramDay, 0, len(r.Program))
		for _, day := range r.Program {
			t.Program = append(t.Program, models.ProgramDay{
				Title: strings.TrimSpace(day.Title), Text: strings.TrimSpace(day.Text),
			})
		}
	}
	applyField(&t.Capacity, r.Capacity)
	contactFields{r.Phone, r.Whatsapp, r.Telegram}.apply(&t.Phone, &t.Whatsapp, &t.Telegram)
}

type Tours struct{ Deps }

func (h *Tours) List(c *gin.Context) {
	q := validation.NewQuery(c)
	locale := q.RequiredEnum("locale", models.Locales)
	filter := store.TourFilter{Destination: q.Enum("destination", models.Destinations)}
	filter.From, filter.To = isoDate(c, "from"), isoDate(c, "to")
	if !q.OK() {
		return
	}
	ctx := c.Request.Context()
	tours, err := h.Store.PublishedTours(ctx, locale, filter)
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	h.withOrganizers(ctx, tours)
	respond.JSON(c, http.StatusOK, tours)
}

func (h *Tours) ListMine(c *gin.Context) {
	tours, err := h.Store.ToursByUser(c.Request.Context(), currentUserID(c))
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, tours)
}

// organizerReviews: reviews on an organizer's page.
const organizerReviews = 20

// Organizer is an organizer's public record in a language: tours held, the
// upcoming ones (without their long texts), the rating of all their tours
// and the latest reviews.
func (h *Tours) Organizer(c *gin.Context) {
	q := validation.NewQuery(c)
	locale := q.RequiredEnum("locale", models.Locales)
	if !q.OK() {
		return
	}
	id := c.Param("userId")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, "Organizer not found")
		return
	}
	ctx := c.Request.Context()
	out, err := h.Store.OrganizerTours(ctx, id, locale, h.today(), organizerReviews)
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	for i := range out.Upcoming {
		out.Upcoming[i].Content, out.Upcoming[i].Program = "", nil
	}
	h.withOrganizers(ctx, out.Upcoming)
	reviews := make([]models.Review, len(out.Reviews))
	for i := range out.Reviews {
		reviews[i] = out.Reviews[i].Review
	}
	authors, err := h.Authors.Authors(ctx, reviewAuthorIDs(reviews))
	if err != nil {
		h.Log.WarnContext(ctx, "review authors", "error", err)
	}
	for i := range out.Reviews {
		out.Reviews[i].Author = authorFor(authors, out.Reviews[i].UserID)
	}
	if out.Upcoming == nil {
		out.Upcoming = []models.Tour{}
	}
	if out.Reviews == nil {
		out.Reviews = []models.PartnerReview{}
	}
	respond.JSON(c, http.StatusOK, out)
}

func reviewAuthorIDs(reviews []models.Review) []string {
	ids := make([]string, len(reviews))
	for i, r := range reviews {
		ids[i] = r.UserID
	}
	return ids
}

func (h *Tours) GetBySlug(c *gin.Context) {
	ctx := c.Request.Context()
	tour, err := h.Store.PublishedTourBySlug(ctx, c.Param("locale"), c.Param("slug"))
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	tours := []models.Tour{tour}
	h.withOrganizers(ctx, tours)
	respond.JSON(c, http.StatusOK, tours[0])
}

func (h *Tours) Create(c *gin.Context) {
	var req createTourRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	// createTourSchema's refine, which zod only runs once the fields pass.
	if *req.EndDate < *req.StartDate {
		c.AbortWithStatusJSON(http.StatusBadRequest, validation.Flattened{
			FormErrors:  []string{},
			FieldErrors: map[string][]string{"endDate": {"endDate must be on or after startDate"}},
		})
		return
	}
	ctx := c.Request.Context()
	if taken, err := h.Store.TourSlugTaken(ctx, *req.Locale, *req.Slug, ""); err != nil || taken {
		h.fail(c, orDuplicate(err), tourNotFound, tourSlugTaken)
		return
	}
	userID := currentUserID(c)
	tour := models.Tour{UserID: &userID}
	req.asUpdate().apply(&tour)
	saved, err := h.Store.SaveTour(ctx, tour)
	if err != nil {
		h.fail(c, err, tourNotFound, tourSlugTaken)
		return
	}
	// Published at once: tell its destination's followers.
	if h.Alerts != nil {
		go h.Alerts.TourPublished(context.WithoutCancel(ctx), saved)
	}
	respond.JSON(c, http.StatusCreated, saved)
}

func (h *Tours) Update(c *gin.Context) {
	var req tourRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	tour, ok := h.owned(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	locale, slug := tour.Locale, tour.Slug
	if req.Locale != nil {
		locale = *req.Locale
	}
	if req.Slug != nil {
		slug = *req.Slug
	}
	if locale != tour.Locale || slug != tour.Slug {
		if taken, err := h.Store.TourSlugTaken(ctx, locale, slug, tour.ID); err != nil || taken {
			h.fail(c, orDuplicate(err), tourNotFound, tourSlugTaken)
			return
		}
	}
	req.apply(&tour)
	saved, err := h.Store.SaveTour(ctx, tour)
	if err != nil {
		h.fail(c, err, tourNotFound, tourSlugTaken)
		return
	}
	respond.JSON(c, http.StatusOK, saved)
}

func (h *Tours) Delete(c *gin.Context) {
	tour, ok := h.owned(c)
	if !ok {
		return
	}
	if err := h.Store.DeleteListing(c.Request.Context(), "tour", tour.ID); err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	respond.Empty(c, http.StatusOK)
}

// withOrganizers adds the partners' public profiles (best effort: a tour
// without one still shows).
func (h *Tours) withOrganizers(ctx context.Context, tours []models.Tour) {
	ids := make([]string, 0, len(tours))
	for _, t := range tours {
		if t.UserID != nil {
			ids = append(ids, *t.UserID)
		}
	}
	if len(ids) == 0 {
		return
	}
	authors, err := h.Authors.Authors(ctx, ids)
	if err != nil {
		h.Log.WarnContext(ctx, "tour organizers", "error", err)
		return
	}
	for i := range tours {
		if tours[i].UserID == nil {
			continue
		}
		if a, ok := authors[*tours[i].UserID]; ok {
			tours[i].Organizer = &a
		}
	}
}

// isoDate reads an optional YYYY-MM-DD query parameter; a malformed one
// is ignored, as the filter is only a narrowing.
func isoDate(c *gin.Context, name string) *string {
	value := c.Query(name)
	if _, err := time.Parse(time.DateOnly, value); err != nil || len(value) != len(time.DateOnly) {
		return nil
	}
	return &value
}

func (h *Tours) owned(c *gin.Context) (models.Tour, bool) {
	id := c.Param("id")
	if !validID(id) {
		h.fail(c, store.ErrNotFound, tourNotFound, "")
		return models.Tour{}, false
	}
	tour, err := h.Store.Tour(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return models.Tour{}, false
	}
	if !ownedBy(tour.UserID, currentUserID(c)) {
		httperr.Abort(c, http.StatusForbidden, "You can only manage your own tours")
		return models.Tour{}, false
	}
	return tour, true
}
