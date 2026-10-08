// Package routes wires the media service's router.
package routes

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/media/internal/config"
	"github.com/elvn5/sai/services/media/internal/handlers"
	"github.com/elvn5/sai/services/media/internal/storage"
	"github.com/elvn5/sai/services/pkg/auth"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/middleware"
)

// Setup builds the router. It sits behind the gateway, which handles CORS.
func Setup(cfg config.Config, r2 *storage.R2, log *slog.Logger) *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(middleware.Recovery(log), middleware.RequestLogger(log))
	r.NoRoute(httperr.RouteNotFound)

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	requireSession := middleware.RequireAdminOrUser(auth.NewVerifier(cfg.JWTSecret))
	uploads := &handlers.Uploads{Dir: cfg.UploadsDir, Log: log}
	files := &handlers.Files{Dir: cfg.UploadsDir}
	store := &handlers.Storage{R2: r2, Log: log}

	r.GET("/uploads/*filepath", files.Serve)
	r.HEAD("/uploads/*filepath", files.Serve)
	r.POST("/uploads/image", requireSession, uploads.Image)
	r.POST("/uploads/file", requireSession, uploads.File)

	r.POST("/storage/upload-url", requireSession, store.UploadURL)
	r.DELETE("/storage/file", requireSession, store.DeleteFile)

	return r
}
