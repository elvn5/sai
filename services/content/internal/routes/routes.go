// Package routes wires the content service's router.
package routes

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/content/internal/handlers"
	"github.com/elvn5/sai/services/content/internal/store"
	"github.com/elvn5/sai/services/pkg/audit"
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
	// The admin panel's history; nil records nothing.
	History *audit.Log
}

// HistoryModules maps the paths whose admin changes go into the
// panel's history to the panel module they belong to.
var HistoryModules = map[string]string{"/blog": "blog", "/page-seo": "seo"}

// Setup builds the router. It sits behind the gateway, which handles CORS.
func Setup(d Deps) *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(middleware.Recovery(d.Log), middleware.RequestLogger(d.Log), d.History.Middleware())
	r.NoRoute(httperr.RouteNotFound)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	verifier := auth.NewVerifier(d.JWTSecret)
	requireUser := middleware.RequireUser(verifier, d.Users)
	requireAdmin := middleware.RequireAdmin(verifier)
	requirePartner := middleware.RequirePartner()
	requireAdminOrPartner := middleware.RequireAdminOrPartner(verifier, d.Users)

	blog := &handlers.Blog{Store: d.Store, Log: d.Log}
	r.GET("/blog", blog.ListPublished)
	r.GET("/blog/admin/all", requireAdmin, blog.ListAll)
	r.GET("/blog/admin/:id", requireAdmin, blog.GetAdmin)
	r.GET("/blog/mine", requireUser, requirePartner, blog.ListMine)
	r.GET("/blog/:locale/:slug", blog.GetBySlug)
	r.POST("/blog", requireAdminOrPartner, blog.Create)
	r.PUT("/blog/:id", requireAdminOrPartner, blog.Update)
	r.DELETE("/blog/:id", requireAdminOrPartner, blog.Delete)

	guides := &handlers.Guides{Store: d.Store, Log: d.Log}
	r.GET("/guides", guides.ListPublished)
	r.GET("/guides/mine", requireUser, guides.ListMine)
	r.GET("/guides/:locale/:slug", guides.GetBySlug)
	r.POST("/guides", requireUser, requirePartner, guides.Create)
	r.PUT("/guides/:id", requireUser, guides.Update)
	r.DELETE("/guides/:id", requireUser, guides.Delete)

	gallery := &handlers.Gallery{Store: d.Store, Media: d.Media, Log: d.Log}
	r.GET("/gallery", gallery.ListPublished)
	r.GET("/gallery/locations", gallery.Locations)
	r.GET("/gallery/mine", requireUser, gallery.ListMine)
	r.POST("/gallery", requireUser, requirePartner, gallery.Create)
	r.PUT("/gallery/:id", requireUser, gallery.Update)
	r.DELETE("/gallery/:id", requireUser, gallery.Delete)

	pageSeo := &handlers.PageSeo{Store: d.Store, Log: d.Log}
	r.GET("/page-seo/admin/all", requireAdmin, pageSeo.ListAll)
	r.GET("/page-seo/:page/:locale", pageSeo.Get)
	r.PUT("/page-seo/:page/:locale", requireAdmin, pageSeo.Upsert)

	r.GET("/blog/admin/history", requireAdmin, d.History.History("blog"))
	r.GET("/page-seo/admin/history", requireAdmin, d.History.History("seo"))

	return r
}
