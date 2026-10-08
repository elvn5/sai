package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/content/internal/models"
	"github.com/elvn5/sai/services/content/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/sanitize"
	"github.com/elvn5/sai/services/pkg/validation"
)

const guideNotFound = "Guide not found"

type guideRequest struct {
	Locale             *string                  `json:"locale" binding:"omitempty,oneof=ru ky en de"`
	Slug               *string                  `json:"slug" binding:"omitempty,min=1,max=255,slug"`
	Title              *string                  `json:"title" binding:"omitempty,min=1,max=255"`
	Excerpt            validation.Field[string] `json:"excerpt" binding:"omitempty,max=500"`
	Content            *string                  `json:"content" binding:"omitempty,min=1,max=20000"`
	Difficulty         *string                  `json:"difficulty" binding:"omitempty,oneof=easy moderate hard"`
	Season             *string                  `json:"season" binding:"omitempty,oneof=summer winter all_year"`
	EquipmentChecklist []string                 `json:"equipmentChecklist" binding:"omitempty,max=50,dive,min=1,max=255"`
	CoverImageURL      validation.Field[string] `json:"coverImageUrl" binding:"omitempty,max=2000"`
	Attachments        []models.Attachment      `json:"attachments" binding:"omitempty,max=10,dive"`
}

// createGuideRequest is guideRequest with the fields a new guide must have
// marked required. (Not embedded: validator would put the embedded type's
// name into error paths.)
type createGuideRequest struct {
	Locale             *string                  `json:"locale" binding:"required,oneof=ru ky en de"`
	Slug               *string                  `json:"slug" binding:"required,min=1,max=255,slug"`
	Title              *string                  `json:"title" binding:"required,min=1,max=255"`
	Excerpt            validation.Field[string] `json:"excerpt" binding:"omitempty,max=500"`
	Content            *string                  `json:"content" binding:"required,min=1,max=20000"`
	Difficulty         *string                  `json:"difficulty" binding:"omitempty,oneof=easy moderate hard"`
	Season             *string                  `json:"season" binding:"omitempty,oneof=summer winter all_year"`
	EquipmentChecklist []string                 `json:"equipmentChecklist" binding:"omitempty,max=50,dive,min=1,max=255"`
	CoverImageURL      validation.Field[string] `json:"coverImageUrl" binding:"omitempty,max=2000"`
	Attachments        []models.Attachment      `json:"attachments" binding:"omitempty,max=10,dive"`
}

func (r createGuideRequest) options() guideRequest {
	return guideRequest{Difficulty: r.Difficulty, Season: r.Season, EquipmentChecklist: r.EquipmentChecklist, Attachments: r.Attachments}
}

// Guides serves /guides: written route guides that any signed-in user can
// publish and manage their own of.
type Guides struct {
	Store *store.Store
	Log   *slog.Logger
}

func (h *Guides) ListPublished(c *gin.Context) {
	locale, ok := optionalEnumQuery(c, "locale", models.Locales)
	if !ok {
		respond.JSON(c, http.StatusOK, []models.Guide{})
		return
	}
	guides, err := h.Store.PublishedGuides(c.Request.Context(), locale)
	if err != nil {
		fail(c, h.Log, err, guideNotFound)
		return
	}
	respond.JSON(c, http.StatusOK, guides)
}

func (h *Guides) ListMine(c *gin.Context) {
	guides, err := h.Store.GuidesByUser(c.Request.Context(), currentUserID(c))
	if err != nil {
		fail(c, h.Log, err, guideNotFound)
		return
	}
	respond.JSON(c, http.StatusOK, guides)
}

func (h *Guides) GetBySlug(c *gin.Context) {
	guide, err := h.Store.PublishedGuideBySlug(c.Request.Context(), c.Param("locale"), c.Param("slug"))
	if err != nil {
		fail(c, h.Log, err, guideNotFound)
		return
	}
	respond.JSON(c, http.StatusOK, guide)
}

func (h *Guides) Create(c *gin.Context) {
	var req createGuideRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	if taken, err := h.Store.GuideSlugTaken(ctx, *req.Locale, *req.Slug, ""); err != nil || taken {
		fail(c, h.Log, orSlugTaken(err), guideNotFound)
		return
	}

	userID := currentUserID(c)
	now := jsontime.New(time.Now())
	guide := models.Guide{
		Locale: *req.Locale, Slug: *req.Slug, Title: *req.Title,
		Content:    sanitize.HTML(*req.Content),
		Excerpt:    req.Excerpt.Ptr(),
		Difficulty: "easy", Season: "all_year",
		EquipmentChecklist: []string{}, Attachments: []models.Attachment{},
		CoverImageURL: req.CoverImageURL.Ptr(),
		IsPublished:   true, PublishedAt: &now, UserID: &userID,
	}
	applyGuideOptions(&guide, req.options())

	saved, err := h.Store.InsertGuide(ctx, guide)
	if err != nil {
		fail(c, h.Log, err, guideNotFound)
		return
	}
	respond.JSON(c, http.StatusCreated, saved)
}

func (h *Guides) Update(c *gin.Context) {
	var req guideRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	guide, ok := h.owned(c)
	if !ok {
		return
	}

	locale, slug := guide.Locale, guide.Slug
	if req.Locale != nil {
		locale = *req.Locale
	}
	if req.Slug != nil {
		slug = *req.Slug
	}
	if locale != guide.Locale || slug != guide.Slug {
		if taken, err := h.Store.GuideSlugTaken(ctx, locale, slug, guide.ID); err != nil || taken {
			fail(c, h.Log, orSlugTaken(err), guideNotFound)
			return
		}
	}

	guide.Locale, guide.Slug = locale, slug
	if req.Title != nil {
		guide.Title = *req.Title
	}
	if req.Content != nil {
		guide.Content = sanitize.HTML(*req.Content)
	}
	applyField(&guide.Excerpt, req.Excerpt)
	applyField(&guide.CoverImageURL, req.CoverImageURL)
	applyGuideOptions(&guide, req)

	saved, err := h.Store.UpdateGuide(ctx, guide)
	if err != nil {
		fail(c, h.Log, err, guideNotFound)
		return
	}
	respond.JSON(c, http.StatusOK, saved)
}

func (h *Guides) Delete(c *gin.Context) {
	guide, ok := h.owned(c)
	if !ok {
		return
	}
	if err := h.Store.DeleteGuide(c.Request.Context(), guide.ID); err != nil {
		fail(c, h.Log, err, guideNotFound)
		return
	}
	respond.Empty(c, http.StatusOK)
}

// owned loads the guide in the path and checks the signed-in user owns it.
func (h *Guides) owned(c *gin.Context) (models.Guide, bool) {
	id := c.Param("id")
	if !validID(id) {
		fail(c, h.Log, store.ErrNotFound, guideNotFound)
		return models.Guide{}, false
	}
	guide, err := h.Store.Guide(c.Request.Context(), id)
	if err != nil {
		fail(c, h.Log, err, guideNotFound)
		return models.Guide{}, false
	}
	if guide.UserID == nil || *guide.UserID != currentUserID(c) {
		httperr.Abort(c, http.StatusForbidden, "You can only manage your own guides")
		return models.Guide{}, false
	}
	return guide, true
}

func applyGuideOptions(g *models.Guide, req guideRequest) {
	if req.Difficulty != nil {
		g.Difficulty = *req.Difficulty
	}
	if req.Season != nil {
		g.Season = *req.Season
	}
	if req.EquipmentChecklist != nil {
		g.EquipmentChecklist = req.EquipmentChecklist
	}
	if req.Attachments != nil {
		g.Attachments = req.Attachments
	}
}
