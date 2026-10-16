// Package routes wires the media service's router.
package routes

import (
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/media/internal/config"
	"github.com/elvn5/sai/services/media/internal/handlers"
	"github.com/elvn5/sai/services/media/internal/storage"
	"github.com/elvn5/sai/services/pkg/auth"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/middleware"
)

// Setup builds the router. It sits behind the gateway, which handles CORS.
func Setup(cfg config.Config, r2 *storage.R2, users middleware.UserLookup, log *slog.Logger) *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(middleware.Recovery(log), middleware.RequestLogger(log))
	r.NoRoute(httperr.RouteNotFound)

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	requireSession := middleware.RequireAdminOrUser(auth.NewVerifier(cfg.JWTSecret), users)
	uploads := &handlers.Uploads{Dir: cfg.UploadsDir, Log: log}
	files := &handlers.Files{Dir: cfg.UploadsDir}
	store := &handlers.Storage{R2: r2, Log: log}
	thumbsDir := cfg.ThumbsDir
	if thumbsDir == "" {
		thumbsDir = filepath.Join(cfg.UploadsDir, ".thumbs")
	}
	images := &handlers.Images{
		UploadsDir:  cfg.UploadsDir,
		CacheDir:    thumbsDir,
		R2PublicURL: cfg.R2PublicURL,
		Client:      &http.Client{},
		Log:         log,
		Slots:       make(chan struct{}, max(1, cfg.ThumbConcurrency)),
	}

	r.GET("/uploads/*filepath", files.Serve)
	r.HEAD("/uploads/*filepath", files.Serve)
	r.GET("/images/:width/*source", images.Serve)
	r.HEAD("/images/:width/*source", images.Serve)
	r.POST("/uploads/image", requireSession, uploads.Image)
	r.POST("/uploads/file", requireSession, uploads.File)

	r.POST("/storage/upload-url", requireSession, store.UploadURL)
	r.DELETE("/storage/file", requireSession, store.DeleteFile)

	return r
}
