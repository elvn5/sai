package handlers

import (
	"net/http"

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
	Phone              validation.Field[string] `json:"phone" binding:"omitempty,max=32,phone"`
	Whatsapp           validation.Field[string] `json:"whatsapp" binding:"omitempty,max=32,phone"`
	Telegram           validation.Field[string] `json:"telegram" binding:"omitempty,max=33,telegram"`
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
	contactFields{r.Phone, r.Whatsapp, r.Telegram}.apply(&t.Phone, &t.Whatsapp, &t.Telegram)
}

type Tours struct{ Deps }

func (h *Tours) List(c *gin.Context) {
	q := validation.NewQuery(c)
	locale := q.RequiredEnum("locale", models.Locales)
	destination := q.Enum("destination", models.Destinations)
	if !q.OK() {
		return
	}
	tours, err := h.Store.PublishedTours(c.Request.Context(), locale, destination)
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
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

func (h *Tours) GetBySlug(c *gin.Context) {
	tour, err := h.Store.PublishedTourBySlug(c.Request.Context(), c.Param("locale"), c.Param("slug"))
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, tour)
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
