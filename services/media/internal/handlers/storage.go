package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/media/internal/storage"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/validation"
)

func init() {
	validation.RegisterPattern("storagefolder", regexp.MustCompile(`^[a-z0-9-]+$`),
		"folder must be lowercase alphanumeric with hyphens")
}

type presignRequest struct {
	Filename    *string `json:"filename" binding:"required,min=1,max=255"`
	ContentType *string `json:"contentType" binding:"required,oneof=image/jpeg image/png image/webp"`
	Folder      *string `json:"folder" binding:"omitempty,max=100,storagefolder"`
}

type deleteRequest struct {
	Key *string `json:"key" binding:"required,min=1,max=500"`
}

// Storage exposes R2 to the frontends: presigned direct uploads and
// deletes.
type Storage struct {
	R2  *storage.R2
	Log *slog.Logger
}

// UploadURL handles POST /storage/upload-url.
func (h *Storage) UploadURL(c *gin.Context) {
	var req presignRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	folder := ""
	if req.Folder != nil {
		folder = *req.Folder
	}

	upload, err := h.R2.PresignUpload(c.Request.Context(), *req.Filename, *req.ContentType, folder)
	switch {
	case errors.Is(err, storage.ErrNotConfigured):
		h.Log.ErrorContext(c.Request.Context(), "R2 is not configured", slog.Any("error", err))
		httperr.Abort(c, http.StatusInternalServerError, "Storage service is not configured")
	case err != nil:
		h.Log.ErrorContext(c.Request.Context(), "failed to create presigned upload URL", slog.Any("error", err))
		httperr.Abort(c, http.StatusInternalServerError, "Could not create upload URL")
	default:
		c.JSON(http.StatusCreated, upload)
	}
}

// DeleteFile handles DELETE /storage/file.
func (h *Storage) DeleteFile(c *gin.Context) {
	var req deleteRequest
	if !validation.BindJSON(c, &req) {
		return
	}

	err := h.R2.Delete(c.Request.Context(), *req.Key)
	switch {
	case errors.Is(err, storage.ErrNotConfigured):
		h.Log.ErrorContext(c.Request.Context(), "R2 is not configured", slog.Any("error", err))
		httperr.Abort(c, http.StatusInternalServerError, "Storage service is not configured")
	case err != nil:
		h.Log.ErrorContext(c.Request.Context(), "failed to delete R2 object", slog.String("key", *req.Key), slog.Any("error", err))
		httperr.Abort(c, http.StatusInternalServerError, "Could not delete file")
	default:
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}
