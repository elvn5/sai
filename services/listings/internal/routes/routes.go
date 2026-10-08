// Package routes wires the listings service's router.
package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/listings/internal/handlers"
	"github.com/elvn5/sai/services/pkg/auth"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/middleware"
)

// Setup builds the router. It sits behind the gateway, which handles CORS.
func Setup(d handlers.Deps, users middleware.UserLookup, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(middleware.Recovery(d.Log), middleware.RequestLogger(d.Log))
	r.NoRoute(httperr.RouteNotFound)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	requireUser := middleware.RequireUser(auth.NewVerifier(jwtSecret), users)

	hotels := &handlers.Hotels{Deps: d}
	r.GET("/hotels", hotels.List)
	r.GET("/hotels/mine", requireUser, hotels.ListMine)
	r.GET("/hotels/:slug", hotels.GetBySlug)
	r.POST("/hotels", requireUser, hotels.Create)
	r.PUT("/hotels/:id", requireUser, hotels.Update)
	r.DELETE("/hotels/:id", requireUser, hotels.Delete)

	guides := &handlers.LocalGuides{Deps: d}
	r.GET("/local-guides", guides.List)
	r.GET("/local-guides/mine", requireUser, guides.Mine)
	r.GET("/local-guides/:slug", guides.GetBySlug)
	r.POST("/local-guides", requireUser, guides.Create)
	r.PUT("/local-guides/:id", requireUser, guides.Update)
	r.DELETE("/local-guides/:id", requireUser, guides.Delete)

	tours := &handlers.Tours{Deps: d}
	r.GET("/tours", tours.List)
	r.GET("/tours/mine", requireUser, tours.ListMine)
	r.GET("/tours/:locale/:slug", tours.GetBySlug)
	r.POST("/tours", requireUser, middleware.RequirePartner(), tours.Create)
	r.PUT("/tours/:id", requireUser, tours.Update)
	r.DELETE("/tours/:id", requireUser, tours.Delete)

	reviews := &handlers.Reviews{Deps: d}
	r.GET("/reviews", reviews.List)
	r.POST("/reviews", requireUser, reviews.Create)
	r.DELETE("/reviews/:id", requireUser, reviews.Delete)

	posts := &handlers.Posts{Deps: d}
	r.GET("/listing-posts", posts.List)
	r.POST("/listing-posts", requireUser, posts.Create)
	r.DELETE("/listing-posts/:id", requireUser, posts.Delete)

	return r
}
