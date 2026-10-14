// Package routes wires the payments service's HTTP router.
package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/payments/internal/handlers"
	"github.com/elvn5/sai/services/payments/internal/provider"
	"github.com/elvn5/sai/services/pkg/auth"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/middleware"
)

// Setup builds the router. It sits behind the gateway, which handles CORS.
// fake is set when the test payment page is on.
func Setup(d handlers.Deps, users middleware.UserLookup, jwtSecret string, fake *provider.Fake) *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(middleware.Recovery(d.Log), middleware.RequestLogger(d.Log))
	r.NoRoute(httperr.RouteNotFound)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	verifier := auth.NewVerifier(jwtSecret)
	requireUser := middleware.RequireUser(verifier, users)
	requireAdmin := middleware.RequireAdmin(verifier)

	orders := &handlers.Orders{Deps: d}
	admin := &handlers.Admin{Deps: d}
	r.POST("/orders", requireUser, orders.Create)
	r.GET("/orders/mine", requireUser, orders.Mine)
	r.GET("/orders/received", requireUser, middleware.RequirePartner(), orders.Received)
	r.GET("/orders/admin", requireAdmin, admin.Orders)
	r.GET("/orders/:id", middleware.RequireAdminOrUser(verifier, users), orders.Get)
	r.POST("/orders/:id/check", requireUser, orders.Check)
	r.POST("/orders/:id/refund", requireAdmin, admin.Refund)

	partner := &handlers.Partner{Deps: d}
	r.GET("/payouts/mine", requireUser, middleware.RequirePartner(), partner.Finances)
	r.GET("/payouts", requireAdmin, admin.Payouts)
	r.GET("/payouts/balances", requireAdmin, admin.Balances)
	r.POST("/payouts", requireAdmin, admin.CreatePayout)

	if fake != nil {
		test := &handlers.TestPayments{Deps: d, Fake: fake}
		r.POST("/payments/test/:id", requireUser, test.Complete)
	}
	return r
}
