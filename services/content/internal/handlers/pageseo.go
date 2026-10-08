package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/content/internal/models"
	"github.com/elvn5/sai/services/content/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
)

type upsertPageSeoRequest struct {
	Title       validation.Field[string] `json:"title" binding:"omitempty,max=255"`
	Description validation.Field[string] `json:"description" binding:"omitempty,max=500"`
	OgImageURL  validation.Field[string] `json:"ogImageUrl" binding:"omitempty,max=2000"`
}

// PageSeo serves /page-seo: per-page, per-locale SEO overrides edited in
// the admin panel.
type PageSeo struct {
	Store *store.Store
	Log   *slog.Logger
}

func (h *PageSeo) ListAll(c *gin.Context) {
	items, err := h.Store.AllPageSeo(c.Request.Context())
	if err != nil {
		fail(c, h.Log, err, "")
		return
	}
	respond.JSON(c, http.StatusOK, items)
}

// Get answers with the override, or — like NestJS returning null — an
// empty 200 when there is none.
func (h *PageSeo) Get(c *gin.Context) {
	page, locale := c.Param("page"), c.Param("locale")
	if !slices.Contains(models.SeoPages, page) || !slices.Contains(models.Locales, locale) {
		respond.Empty(c, http.StatusOK)
		return
	}
	item, err := h.Store.PageSeo(c.Request.Context(), page, locale)
	switch {
	case errors.Is(err, store.ErrNotFound):
		respond.Empty(c, http.StatusOK)
	case err != nil:
		fail(c, h.Log, err, "")
	default:
		respond.JSON(c, http.StatusOK, item)
	}
}

func (h *PageSeo) Upsert(c *gin.Context) {
	var req upsertPageSeoRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	page, locale := c.Param("page"), c.Param("locale")
	if !slices.Contains(models.SeoPages, page) || !slices.Contains(models.Locales, locale) {
		httperr.Abort(c, http.StatusBadRequest, "Unknown page or locale")
		return
	}

	ctx := c.Request.Context()
	item, err := h.Store.PageSeo(ctx, page, locale)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		fail(c, h.Log, err, "")
		return
	}
	item.Page, item.Locale = page, locale
	applyField(&item.Title, req.Title)
	applyField(&item.Description, req.Description)
	applyField(&item.OgImageURL, req.OgImageURL)

	saved, err := h.Store.UpsertPageSeo(ctx, item)
	if err != nil {
		fail(c, h.Log, err, "")
		return
	}
	respond.JSON(c, http.StatusOK, saved)
}
