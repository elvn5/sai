package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/content/internal/models"
	"github.com/elvn5/sai/services/content/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
)

const (
	galleryItemNotFound = "Gallery item not found"
	galleryDefaultLimit = 20
	galleryMaxLimit     = 50
)

type galleryItemRequest struct {
	ImageURL *string                  `json:"imageUrl" binding:"omitempty,min=1,max=2000"`
	Location validation.Field[string] `json:"location" binding:"omitempty,max=255"`
	Caption  validation.Field[string] `json:"caption" binding:"omitempty,max=500"`
	AltText  validation.Field[string] `json:"altText" binding:"omitempty,max=255"`
}

type createGalleryItemRequest struct {
	ImageURL *string                  `json:"imageUrl" binding:"required,min=1,max=2000"`
	Location validation.Field[string] `json:"location" binding:"omitempty,max=255"`
	Caption  validation.Field[string] `json:"caption" binding:"omitempty,max=500"`
	AltText  validation.Field[string] `json:"altText" binding:"omitempty,max=255"`
}

// PhotoDeleter removes a photo's file from storage (the media service).
type PhotoDeleter interface {
	DeleteObject(ctx context.Context, publicURL string) error
}

// Gallery serves /gallery: community photos.
type Gallery struct {
	Store *store.Store
	// Optional; without it deleted photos' files stay in storage.
	Media PhotoDeleter
	Log   *slog.Logger
}

func (h *Gallery) ListPublished(c *gin.Context) {
	limit := galleryDefaultLimit
	if n, ok := jsParseInt(c.Query("limit")); ok && n != 0 {
		limit = min(max(n, 1), galleryMaxLimit)
	}
	offset := 0
	if n, ok := jsParseInt(c.Query("offset")); ok {
		offset = max(n, 0)
	}

	items, err := h.Store.PublishedGalleryItems(c.Request.Context(), store.GalleryFilter{
		Location: c.Query("location"), Search: c.Query("search"), Limit: limit, Offset: offset,
	})
	if err != nil {
		fail(c, h.Log, err, galleryItemNotFound)
		return
	}
	respond.JSON(c, http.StatusOK, items)
}

func (h *Gallery) Locations(c *gin.Context) {
	locations, err := h.Store.GalleryLocations(c.Request.Context())
	if err != nil {
		fail(c, h.Log, err, galleryItemNotFound)
		return
	}
	respond.JSON(c, http.StatusOK, locations)
}

func (h *Gallery) ListMine(c *gin.Context) {
	items, err := h.Store.GalleryItemsByUser(c.Request.Context(), currentUserID(c))
	if err != nil {
		fail(c, h.Log, err, galleryItemNotFound)
		return
	}
	respond.JSON(c, http.StatusOK, items)
}

func (h *Gallery) Create(c *gin.Context) {
	var req createGalleryItemRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	userID := currentUserID(c)
	saved, err := h.Store.InsertGalleryItem(c.Request.Context(), models.GalleryItem{
		ImageURL: *req.ImageURL, Location: req.Location.Ptr(), Caption: req.Caption.Ptr(),
		AltText: req.AltText.Ptr(), SortOrder: 0, IsPublished: true, UserID: &userID,
	})
	if err != nil {
		fail(c, h.Log, err, galleryItemNotFound)
		return
	}
	respond.JSON(c, http.StatusCreated, saved)
}

func (h *Gallery) Update(c *gin.Context) {
	var req galleryItemRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	item, ok := h.owned(c)
	if !ok {
		return
	}
	if req.ImageURL != nil {
		item.ImageURL = *req.ImageURL
	}
	applyField(&item.Location, req.Location)
	applyField(&item.Caption, req.Caption)
	applyField(&item.AltText, req.AltText)

	saved, err := h.Store.UpdateGalleryItem(c.Request.Context(), item)
	if err != nil {
		fail(c, h.Log, err, galleryItemNotFound)
		return
	}
	respond.JSON(c, http.StatusOK, saved)
}

// Delete removes the item, then (best effort) its photo from R2 — a
// failed cleanup is logged, not reported, as in the NestJS API.
func (h *Gallery) Delete(c *gin.Context) {
	item, ok := h.owned(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	if err := h.Store.DeleteGalleryItem(ctx, item.ID); err != nil {
		fail(c, h.Log, err, galleryItemNotFound)
		return
	}
	if h.Media != nil {
		if err := h.Media.DeleteObject(ctx, item.ImageURL); err != nil {
			h.Log.ErrorContext(ctx, "failed to delete gallery photo from storage",
				slog.String("imageUrl", item.ImageURL), slog.Any("error", err))
		}
	}
	respond.Empty(c, http.StatusOK)
}

func (h *Gallery) owned(c *gin.Context) (models.GalleryItem, bool) {
	id := c.Param("id")
	if !validID(id) {
		fail(c, h.Log, store.ErrNotFound, galleryItemNotFound)
		return models.GalleryItem{}, false
	}
	item, err := h.Store.GalleryItem(c.Request.Context(), id)
	if err != nil {
		fail(c, h.Log, err, galleryItemNotFound)
		return models.GalleryItem{}, false
	}
	if item.UserID == nil || *item.UserID != currentUserID(c) {
		httperr.Abort(c, http.StatusForbidden, "You can only manage your own gallery items")
		return models.GalleryItem{}, false
	}
	return item, true
}
