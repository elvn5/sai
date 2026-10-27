// Package routes wires the trails service's HTTP router.
package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/auth"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/middleware"
	"github.com/elvn5/sai/services/trails/internal/handlers"
)

// Setup builds the router. It sits behind the gateway, which handles CORS.
func Setup(d handlers.Deps, users middleware.UserLookup, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(middleware.Recovery(d.Log), middleware.RequestLogger(d.Log))
	r.NoRoute(httperr.RouteNotFound)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	verifier := auth.NewVerifier(jwtSecret)
	optionalUser := middleware.OptionalUser(verifier, users)
	requireUser := middleware.RequireUser(verifier, users)
	requireAdmin := middleware.RequireAdmin(verifier)
	requireAdminOrPartner := middleware.RequireAdminOrPartner(verifier, users)
	requireAdminOrUser := middleware.RequireAdminOrUser(verifier, users)

	trails := &handlers.Trails{Deps: d}
	conditions := &handlers.Conditions{Deps: d}
	r.GET("/trails", trails.List)
	r.GET("/trails/areas", trails.Areas)
	r.GET("/trails/mine", requireUser, middleware.RequirePartner(), trails.Mine)
	r.GET("/trails/admin/all", requireAdmin, trails.AdminAll)
	// :key is a slug for reading and an id for changes.
	r.GET("/trails/:key", optionalUser, trails.Get)
	r.GET("/trails/:key/conditions", conditions.ForTrail)
	r.POST("/trails/:key/conditions", requireUser, conditions.Create)
	r.POST("/trails", requireAdminOrPartner, trails.Create)
	r.PUT("/trails/:key", requireAdminOrPartner, trails.Update)
	r.DELETE("/trails/:key", requireAdminOrPartner, trails.Delete)

	r.GET("/conditions", conditions.Recent)
	r.DELETE("/conditions/:id", requireAdminOrUser, conditions.Delete)

	return r
}
