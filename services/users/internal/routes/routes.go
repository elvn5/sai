// Package routes wires the users service's HTTP router.
package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/auth"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/middleware"
	"github.com/elvn5/sai/services/users/internal/handlers"
)

// Setup builds the router. It sits behind the gateway, which handles CORS.
func Setup(d handlers.Deps, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(middleware.Recovery(d.Log), middleware.RequestLogger(d.Log))
	r.NoRoute(httperr.RouteNotFound)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	verifier := auth.NewVerifier(jwtSecret)
	requireUser := middleware.RequireUser(verifier, handlers.Lookup{Store: d.Store})
	requireAdmin := middleware.RequireAdmin(verifier)

	adminAuth := &handlers.AdminAuth{Deps: d}
	r.POST("/auth/login", adminAuth.Login)
	r.POST("/auth/logout", adminAuth.Logout)
	r.GET("/auth/me", requireAdmin, adminAuth.Me)

	creds := &handlers.Credentials{Deps: d}
	r.POST("/auth/user/register", creds.Register)
	r.POST("/auth/user/login", creds.Login)
	r.POST("/auth/user/verify", creds.Verify)
	r.POST("/auth/user/resend-code", creds.ResendCode)
	r.GET("/auth/user/verification-status", creds.VerificationStatus)
	r.POST("/auth/user/password/forgot", creds.ForgotPassword)
	r.POST("/auth/user/password/reset", creds.ResetPassword)

	account := &handlers.Account{Deps: d}
	r.GET("/auth/user/me", requireUser, account.Me)
	r.POST("/auth/user/logout", account.Logout)
	r.POST("/partners/apply", requireUser, account.ApplyForPartnership)

	telegramLink := &handlers.TelegramLink{Deps: d}
	r.POST("/auth/user/telegram/link", requireUser, telegramLink.Start)
	r.DELETE("/auth/user/telegram", requireUser, telegramLink.Unlink)

	telegram := &handlers.Telegram{Deps: d}
	r.POST("/auth/telegram/widget", telegram.Widget)
	r.POST("/auth/telegram/miniapp", telegram.MiniApp)

	admin := &handlers.Admin{Deps: d}
	r.GET("/users", requireAdmin, admin.ListUsers)
	r.GET("/users/:id", requireAdmin, admin.GetUser)
	r.GET("/users/:id/public", (&handlers.Profiles{Deps: d}).Public)
	r.GET("/partners/requests", requireAdmin, admin.PartnerRequests)
	r.POST("/partners/:userId/approve", requireAdmin, admin.ApprovePartner)
	r.POST("/partners/:userId/reject", requireAdmin, admin.RejectPartner)
	r.GET("/support", requireAdmin, admin.GetSupport)
	r.PUT("/support", requireAdmin, admin.UpdateSupport)

	achievements := &handlers.Achievements{Deps: d}
	requirePartner := middleware.RequirePartner()
	r.GET("/hall-of-fame", achievements.HallOfFame)
	r.GET("/achievements", achievements.Catalog)
	r.GET("/achievements/me", requireUser, achievements.Mine)
	r.POST("/achievements/me/seen", requireUser, achievements.MarkSeen)
	r.GET("/achievements/recipients/:code", requireUser, requirePartner, achievements.Recipient)
	r.POST("/achievements/awards", requireUser, requirePartner, achievements.Award)
	r.GET("/achievements/awards/given", requireUser, requirePartner, achievements.Given)
	r.GET("/achievements/admin/all", requireAdmin, achievements.AdminList)
	r.GET("/achievements/admin/awards", requireAdmin, achievements.AdminAwards)
	r.DELETE("/achievements/awards/:id", requireAdmin, achievements.Revoke)
	r.POST("/achievements", requireAdmin, achievements.Create)
	r.PUT("/achievements/:id", requireAdmin, achievements.Update)
	r.DELETE("/achievements/:id", requireAdmin, achievements.Delete)

	return r
}
