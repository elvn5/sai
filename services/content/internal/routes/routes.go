// Package routes wires the content service's router.
package routes

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/content/internal/handlers"
	"github.com/elvn5/sai/services/content/internal/store"
	"github.com/elvn5/sai/services/pkg/auth"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/middleware"
)

type Deps struct {
	Store     *store.Store
	Media     handlers.PhotoDeleter // optional
	Users     middleware.UserLookup // optional
	JWTSecret string
	Log       *slog.Logger
}

// Setup builds the router. It sits behind the gateway, which handles CORS.
func Setup(d Deps) *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(middleware.Recovery(d.Log), middleware.RequestLogger(d.Log))
	r.NoRoute(httperr.RouteNotFound)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	verifier := auth.NewVerifier(d.JWTSecret)
	requireUser := middleware.RequireUser(verifier, d.Users)
	requireAdmin := middleware.RequireAdmin(verifier)

	blog := &handlers.Blog{Store: d.Store, Log: d.Log}
	r.GET("/blog", blog.ListPublished)
	r.GET("/blog/admin/all", requireAdmin, blog.ListAll)
	r.GET("/blog/admin/:id", requireAdmin, blog.GetAdmin)
	r.GET("/blog/:locale/:slug", blog.GetBySlug)
	r.POST("/blog", requireAdmin, blog.Create)
	r.PUT("/blog/:id", requireAdmin, blog.Update)
	r.DELETE("/blog/:id", requireAdmin, blog.Delete)

	guides := &handlers.Guides{Store: d.Store, Log: d.Log}
	r.GET("/guides", guides.ListPublished)
	r.GET("/guides/mine", requireUser, guides.ListMine)
	r.GET("/guides/:locale/:slug", guides.GetBySlug)
	r.POST("/guides", requireUser, guides.Create)
	r.PUT("/guides/:id", requireUser, guides.Update)
	r.DELETE("/guides/:id", requireUser, guides.Delete)

	gallery := &handlers.Gallery{Store: d.Store, Media: d.Media, Log: d.Log}
	r.GET("/gallery", gallery.ListPublished)
	r.GET("/gallery/locations", gallery.Locations)
	r.GET("/gallery/mine", requireUser, gallery.ListMine)
	r.POST("/gallery", requireUser, gallery.Create)
	r.PUT("/gallery/:id", requireUser, gallery.Update)
	r.DELETE("/gallery/:id", requireUser, gallery.Delete)

	pageSeo := &handlers.PageSeo{Store: d.Store, Log: d.Log}
	r.GET("/page-seo/admin/all", requireAdmin, pageSeo.ListAll)
	r.GET("/page-seo/:page/:locale", pageSeo.Get)
	r.PUT("/page-seo/:page/:locale", requireAdmin, pageSeo.Upsert)

	return r
}
