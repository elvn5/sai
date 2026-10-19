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

	verifier := auth.NewVerifier(jwtSecret)
	requireUser := middleware.RequireUser(verifier, users)
	requirePartner := middleware.RequirePartner()

	hotels := &handlers.Hotels{Deps: d}
	r.GET("/hotels", hotels.List)
	r.GET("/hotels/mine", requireUser, hotels.ListMine)
	r.GET("/hotels/:slug", hotels.GetBySlug)
	r.POST("/hotels", requireUser, requirePartner, hotels.Create)
	r.PUT("/hotels/:id", requireUser, hotels.Update)
	r.DELETE("/hotels/:id", requireUser, hotels.Delete)

	guides := &handlers.LocalGuides{Deps: d}
	r.GET("/local-guides", guides.List)
	r.GET("/local-guides/mine", requireUser, guides.Mine)
	r.GET("/local-guides/:slug", guides.GetBySlug)
	r.POST("/local-guides", requireUser, requirePartner, guides.Create)
	r.PUT("/local-guides/:id", requireUser, guides.Update)
	r.DELETE("/local-guides/:id", requireUser, guides.Delete)

	tours := &handlers.Tours{Deps: d}
	r.GET("/tours", tours.List)
	r.GET("/tours/mine", requireUser, tours.ListMine)
	r.GET("/tours/:locale/:slug", tours.GetBySlug)
	r.POST("/tours", requireUser, requirePartner, tours.Create)
	r.PUT("/tours/:id", requireUser, tours.Update)
	r.DELETE("/tours/:id", requireUser, tours.Delete)

	reviews := &handlers.Reviews{Deps: d}
	r.GET("/reviews", reviews.List)
	r.GET("/reviews/mine", requireUser, requirePartner, reviews.Mine)
	r.POST("/reviews", requireUser, reviews.Create)
	r.DELETE("/reviews/:id", requireUser, reviews.Delete)
	r.PUT("/reviews/:id/reply", requireUser, reviews.Reply)
	r.DELETE("/reviews/:id/reply", requireUser, reviews.DeleteReply)

	posts := &handlers.Posts{Deps: d}
	r.GET("/listing-posts", posts.List)
	r.POST("/listing-posts", requireUser, posts.Create)
	r.DELETE("/listing-posts/:id", requireUser, posts.Delete)

	// A traveler's saved listings, and "tell me about new tours".
	favorites := &handlers.Favorites{Deps: d}
	r.GET("/favorites/mine", requireUser, favorites.Mine)
	r.GET("/favorites/mine/ids", requireUser, favorites.IDs)
	r.PUT("/favorites/:type/:id", requireUser, favorites.Add)
	r.DELETE("/favorites/:type/:id", requireUser, favorites.Remove)
	alerts := &handlers.TourAlerts{Deps: d}
	r.GET("/tour-alerts/mine", requireUser, alerts.Mine)
	r.PUT("/tour-alerts", requireUser, alerts.Add)
	r.DELETE("/tour-alerts", requireUser, alerts.Remove)

	// Partner cabinet: visits and contact clicks of listings.
	stats := &handlers.Stats{Deps: d}
	r.POST("/listing-stats/events", middleware.OptionalUser(verifier), stats.Record)
	r.GET("/listing-stats/mine", requireUser, requirePartner, stats.Mine)

	return r
}
