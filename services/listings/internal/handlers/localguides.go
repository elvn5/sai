package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/sanitize"
	"github.com/elvn5/sai/services/pkg/validation"
)

const localGuideNotFound = "Guide not found"

type localGuideRequest struct {
	Slug            *string                              `json:"slug" binding:"omitempty,min=1,max=255,slug"`
	Name            *validation.Trimmed                  `json:"name" binding:"omitempty,min=1,max=255"`
	Headline        validation.Field[validation.Trimmed] `json:"headline" binding:"omitempty,max=160"`
	Bio             *string                              `json:"bio" binding:"omitempty,min=1,max=20000"`
	AvatarURL       validation.Field[string]             `json:"avatarUrl" binding:"omitempty,max=2000,mediaurl"`
	CoverImageURL   validation.Field[string]             `json:"coverImageUrl" binding:"omitempty,max=2000,mediaurl"`
	Photos          []string                             `json:"photos" binding:"omitempty,max=20,dive,max=2000,mediaurl"`
	Languages       []string                             `json:"languages" binding:"omitempty,min=1,max=14,dive,oneof=ru ky en de fr es it zh ja ko tr ar kk uz"`
	Specializations []string                             `json:"specializations" binding:"omitempty,min=1,max=10,dive,oneof=trekking horse_riding mountaineering culture nomad_life city photo ski jeep food"`
	Destinations    []string                             `json:"destinations" binding:"omitempty,min=1,dive,oneof=issyk_kul karakol bishkek ala_archa song_kol naryn tash_rabat kel_suu osh arslanbob sary_chelek talas batken chuy"`
	PricePerDay     validation.Field[int]                `json:"pricePerDay" binding:"omitempty,gte=0"`
	ExperienceYears validation.Field[int]                `json:"experienceYears" binding:"omitempty,gte=0,lte=80"`
	HasVehicle      *bool                                `json:"hasVehicle"`
	Phone           validation.Field[string]             `json:"phone" binding:"omitempty,max=32,phone"`
	Whatsapp        validation.Field[string]             `json:"whatsapp" binding:"omitempty,max=32,phone"`
	Telegram        validation.Field[string]             `json:"telegram" binding:"omitempty,max=33,telegram"`
}

type createLocalGuideRequest struct {
	Slug            *string                              `json:"slug" binding:"required,min=1,max=255,slug"`
	Name            *validation.Trimmed                  `json:"name" binding:"required,min=1,max=255"`
	Headline        validation.Field[validation.Trimmed] `json:"headline" binding:"omitempty,max=160"`
	Bio             *string                              `json:"bio" binding:"required,min=1,max=20000"`
	AvatarURL       validation.Field[string]             `json:"avatarUrl" binding:"omitempty,max=2000,mediaurl"`
	CoverImageURL   validation.Field[string]             `json:"coverImageUrl" binding:"omitempty,max=2000,mediaurl"`
	Photos          []string                             `json:"photos" binding:"omitempty,max=20,dive,max=2000,mediaurl"`
	Languages       []string                             `json:"languages" binding:"required,min=1,max=14,dive,oneof=ru ky en de fr es it zh ja ko tr ar kk uz"`
	Specializations []string                             `json:"specializations" binding:"required,min=1,max=10,dive,oneof=trekking horse_riding mountaineering culture nomad_life city photo ski jeep food"`
	Destinations    []string                             `json:"destinations" binding:"required,min=1,dive,oneof=issyk_kul karakol bishkek ala_archa song_kol naryn tash_rabat kel_suu osh arslanbob sary_chelek talas batken chuy"`
	PricePerDay     validation.Field[int]                `json:"pricePerDay" binding:"omitempty,gte=0"`
	ExperienceYears validation.Field[int]                `json:"experienceYears" binding:"omitempty,gte=0,lte=80"`
	HasVehicle      *bool                                `json:"hasVehicle"`
	Phone           validation.Field[string]             `json:"phone" binding:"omitempty,max=32,phone"`
	Whatsapp        validation.Field[string]             `json:"whatsapp" binding:"omitempty,max=32,phone"`
	Telegram        validation.Field[string]             `json:"telegram" binding:"omitempty,max=33,telegram"`
}

func (r createLocalGuideRequest) asUpdate() localGuideRequest {
	// Same fields, different validation tags.
	return localGuideRequest(r)
}

func (r localGuideRequest) apply(g *models.LocalGuide) {
	if r.Slug != nil {
		g.Slug = *r.Slug
	}
	if r.Name != nil {
		g.Name = string(*r.Name)
	}
	if r.Bio != nil {
		g.Bio = sanitize.HTML(*r.Bio)
	}
	if r.Photos != nil {
		g.Photos = r.Photos
	}
	if r.Languages != nil {
		g.Languages = r.Languages
	}
	if r.Specializations != nil {
		g.Specializations = r.Specializations
	}
	if r.Destinations != nil {
		g.Destinations = r.Destinations
	}
	if r.HasVehicle != nil {
		g.HasVehicle = *r.HasVehicle
	}
	applyTrimmed(&g.Headline, r.Headline)
	applyField(&g.AvatarURL, r.AvatarURL)
	applyField(&g.CoverImageURL, r.CoverImageURL)
	applyField(&g.PricePerDay, r.PricePerDay)
	applyField(&g.ExperienceYears, r.ExperienceYears)
	contactFields{r.Phone, r.Whatsapp, r.Telegram}.apply(&g.Phone, &g.Whatsapp, &g.Telegram)
}

type LocalGuides struct{ Deps }

func (h *LocalGuides) List(c *gin.Context) {
	q := validation.NewQuery(c)
	f := store.LocalGuideFilter{
		Limit:          q.IntDefault("limit", 20, validation.Bound(1), validation.Bound(50)),
		Offset:         q.IntDefault("offset", 0, validation.Bound(0), nil),
		Destination:    q.Enum("destination", models.Destinations),
		Language:       q.Enum("language", models.GuideLanguages),
		Specialization: q.Enum("specialization", models.GuideSpecializations),
		Search:         q.String("search", 100),
		Sort:           q.EnumDefault("sort", models.LocalGuideSorts, "rating"),
		HasVehicle:     boolValue(q.Bool("hasVehicle")),
	}
	if n := q.Number("maxPrice", validation.NumberRule{Integer: true, Min: validation.Bound(0)}); n != nil {
		v := int(*n)
		f.MaxPrice = &v
	}
	f.MinRating = q.Number("minRating", validation.NumberRule{Min: validation.Bound(0), Max: validation.Bound(5)})
	if !q.OK() {
		return
	}

	page, err := h.Store.PublishedLocalGuides(c.Request.Context(), f)
	if err != nil {
		h.fail(c, err, localGuideNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, page)
}

// Mine answers {"guide": ...} — wrapped so "no profile yet" is an explicit
// null.
func (h *LocalGuides) Mine(c *gin.Context) {
	guide, err := h.Store.LocalGuideByUser(c.Request.Context(), currentUserID(c))
	switch {
	case errors.Is(err, store.ErrNotFound):
		respond.JSON(c, http.StatusOK, gin.H{"guide": nil})
	case err != nil:
		h.fail(c, err, localGuideNotFound, "")
	default:
		respond.JSON(c, http.StatusOK, gin.H{"guide": guide})
	}
}

func (h *LocalGuides) GetBySlug(c *gin.Context) {
	guide, err := h.Store.PublishedLocalGuideBySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		h.fail(c, err, localGuideNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, guide)
}

func (h *LocalGuides) Create(c *gin.Context) {
	var req createLocalGuideRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	userID := currentUserID(c)
	if _, err := h.Store.LocalGuideByUser(ctx, userID); err == nil {
		httperr.Abort(c, http.StatusConflict, "You already have a guide profile")
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		h.fail(c, err, localGuideNotFound, "")
		return
	}
	if taken, err := h.Store.LocalGuideSlugTaken(ctx, *req.Slug, ""); err != nil || taken {
		h.fail(c, orDuplicate(err), localGuideNotFound, slugTaken)
		return
	}

	guide := models.LocalGuide{UserID: &userID, IsPublished: true}
	req.asUpdate().apply(&guide)
	saved, err := h.Store.SaveLocalGuide(ctx, guide)
	if err != nil {
		h.fail(c, err, localGuideNotFound, slugTaken)
		return
	}
	respond.JSON(c, http.StatusCreated, saved)
}

func (h *LocalGuides) Update(c *gin.Context) {
	var req localGuideRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	guide, ok := h.owned(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	if req.Slug != nil && *req.Slug != guide.Slug {
		if taken, err := h.Store.LocalGuideSlugTaken(ctx, *req.Slug, guide.ID); err != nil || taken {
			h.fail(c, orDuplicate(err), localGuideNotFound, slugTaken)
			return
		}
	}
	req.apply(&guide)
	saved, err := h.Store.SaveLocalGuide(ctx, guide)
	if err != nil {
		h.fail(c, err, localGuideNotFound, slugTaken)
		return
	}
	respond.JSON(c, http.StatusOK, saved)
}

func (h *LocalGuides) Delete(c *gin.Context) {
	guide, ok := h.owned(c)
	if !ok {
		return
	}
	if err := h.Store.DeleteListing(c.Request.Context(), "local_guide", guide.ID); err != nil {
		h.fail(c, err, localGuideNotFound, "")
		return
	}
	respond.Empty(c, http.StatusOK)
}

func (h *LocalGuides) owned(c *gin.Context) (models.LocalGuide, bool) {
	id := c.Param("id")
	if !validID(id) {
		h.fail(c, store.ErrNotFound, localGuideNotFound, "")
		return models.LocalGuide{}, false
	}
	guide, err := h.Store.LocalGuide(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err, localGuideNotFound, "")
		return models.LocalGuide{}, false
	}
	if !ownedBy(guide.UserID, currentUserID(c)) {
		httperr.Abort(c, http.StatusForbidden, "You can only manage your own guide profile")
		return models.LocalGuide{}, false
	}
	return guide, true
}
