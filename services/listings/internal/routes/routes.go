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

// HistoryModules maps the paths whose admin changes go into the
// panel's history to the panel module they belong to.
var HistoryModules = map[string]string{"/tours": "tours"}

// Setup builds the router. It sits behind the gateway, which handles CORS.
func Setup(d handlers.Deps, users middleware.UserLookup, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(middleware.Recovery(d.Log), middleware.RequestLogger(d.Log), d.History.Middleware())
	r.NoRoute(httperr.RouteNotFound)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	verifier := auth.NewVerifier(jwtSecret)
	requireUser := middleware.RequireUser(verifier, users)
	optionalUser := middleware.OptionalUser(verifier, users)
	requirePartner := middleware.RequirePartner()
	requireOrganizer := middleware.RequireOrganizer()
	requireAdmin := middleware.RequireAdmin(verifier)

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
	r.GET("/tours/organizers/:userId", tours.Organizer)
	r.GET("/tours/:locale/:slug", tours.GetBySlug)
	r.POST("/tours", requireUser, requireOrganizer, tours.Create)
	r.PUT("/tours/:id", requireUser, tours.Update)
	r.DELETE("/tours/:id", requireUser, tours.Delete)

	// The panel: every tour with its group.
	admin := &handlers.Admin{Deps: d}
	r.GET("/tours/admin", requireAdmin, admin.Tours)
	r.GET("/tours/admin/history", requireAdmin, d.History.History("tours"))
	r.GET("/tours/admin/:id", requireAdmin, admin.Tour)
	r.PATCH("/tours/admin/:id/participants/:pid", requireAdmin, admin.UpdateParticipant)
	r.PUT("/tours/admin/:id/published", requireAdmin, admin.SetPublished)

	// Signing up for a tour, the organizer's roster and check-in, and
	// participants' ratings.
	trips := &handlers.Participation{Deps: d}
	r.GET("/tours/joined", requireUser, trips.Joined)
	r.GET("/tours/by-id/:id", optionalUser, trips.Get)
	r.POST("/tours/:id/join", requireUser, trips.Join)
	r.DELETE("/tours/:id/join", requireUser, trips.Leave)
	r.PUT("/tours/:id/form", requireUser, trips.SaveForm)
	r.PUT("/tours/:id/participants/:userId/arrival", requireUser, trips.SetArrival)
	r.DELETE("/tours/:id/participants/:userId/arrival", requireUser, trips.SetArrival)
	r.PUT("/tours/:id/rating", requireUser, trips.Rate)

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
	r.POST("/listing-stats/events", middleware.OptionalUser(verifier, users), stats.Record)
	r.GET("/listing-stats/mine", requireUser, requirePartner, stats.Mine)

	// The partner's CRM: clients, each tour's group, payments.
	crm := &handlers.CRM{Deps: d}
	partnerOnly := r.Group("/crm", requireUser, requirePartner)
	partnerOnly.GET("/summary", crm.Summary)
	partnerOnly.GET("/clients", crm.Clients)
	partnerOnly.GET("/clients/tags", crm.Tags)
	partnerOnly.POST("/clients", crm.CreateClient)
	partnerOnly.GET("/clients/:id", crm.Client)
	partnerOnly.PUT("/clients/:id", crm.UpdateClient)
	partnerOnly.DELETE("/clients/:id", crm.DeleteClient)
	partnerOnly.GET("/tours/:id/participants", crm.Group)
	partnerOnly.POST("/tours/:id/participants", crm.AddParticipant)
	partnerOnly.PATCH("/tours/:id/participants/:pid", crm.UpdateParticipant)
	partnerOnly.DELETE("/tours/:id/participants/:pid", crm.DeleteParticipant)
	partnerOnly.PUT("/tours/:id/participants/:pid/form", crm.SetForm)
	partnerOnly.POST("/tours/:id/participants/:pid/payments", crm.AddPayment)
	partnerOnly.DELETE("/payments/:id", crm.DeletePayment)
	partnerOnly.POST("/tours/:id/expenses", crm.AddExpense)
	partnerOnly.DELETE("/expenses/:id", crm.DeleteExpense)
	partnerOnly.POST("/tours/:id/staff", crm.AddStaff)
	partnerOnly.DELETE("/staff/:id", crm.DeleteStaff)
	partnerOnly.GET("/leads", crm.Leads)
	partnerOnly.POST("/leads", crm.CreateLead)
	partnerOnly.PATCH("/leads/:id", crm.UpdateLead)
	partnerOnly.DELETE("/leads/:id", crm.DeleteLead)
	partnerOnly.GET("/tasks", crm.Tasks)
	partnerOnly.POST("/tasks", crm.CreateTask)
	partnerOnly.PATCH("/tasks/:id", crm.UpdateTask)
	partnerOnly.DELETE("/tasks/:id", crm.DeleteTask)
	partnerOnly.GET("/templates", crm.Templates)
	partnerOnly.POST("/templates", crm.SaveTemplate)
	partnerOnly.PUT("/templates/:id", crm.SaveTemplate)
	partnerOnly.DELETE("/templates/:id", crm.DeleteTemplate)
	partnerOnly.GET("/broadcasts", crm.Broadcasts)
	partnerOnly.POST("/broadcasts", crm.Broadcast)
	partnerOnly.GET("/reports", crm.Report)

	// Organizers find drivers and ask them for transport for a tour.
	drivers := &handlers.Drivers{Deps: d}
	organizerOnly := r.Group("/crm", requireUser, requireOrganizer)
	organizerOnly.GET("/drivers", drivers.Catalog)
	organizerOnly.GET("/tours/:id/transport", drivers.Transport)
	organizerOnly.POST("/tours/:id/transport", drivers.RequestTransport)
	organizerOnly.DELETE("/transport/:id", drivers.CancelTransport)

	// The driver's CRM; their clients, tasks, templates and broadcasts are
	// the shared /crm ones.
	driverOnly := r.Group("/driver", requireUser, middleware.RequireDriver())
	driverOnly.GET("/summary", drivers.Summary)
	driverOnly.GET("/profile", drivers.Profile)
	driverOnly.PUT("/profile", drivers.SaveProfile)
	driverOnly.GET("/vehicles", drivers.Vehicles)
	driverOnly.POST("/vehicles", drivers.AddVehicle)
	driverOnly.PUT("/vehicles/:id", drivers.UpdateVehicle)
	driverOnly.DELETE("/vehicles/:id", drivers.DeleteVehicle)
	driverOnly.GET("/trips", drivers.Trips)
	driverOnly.POST("/trips", drivers.AddTrip)
	driverOnly.GET("/trips/:id", drivers.Trip)
	driverOnly.PUT("/trips/:id", drivers.UpdateTrip)
	driverOnly.PUT("/trips/:id/status", drivers.SetStatus)
	driverOnly.DELETE("/trips/:id", drivers.DeleteTrip)
	driverOnly.POST("/trips/:id/payments", drivers.AddPayment)
	driverOnly.DELETE("/payments/:id", drivers.DeletePayment)
	driverOnly.GET("/expenses", drivers.Expenses)
	driverOnly.POST("/expenses", drivers.AddExpense)
	driverOnly.DELETE("/expenses/:id", drivers.DeleteExpense)
	driverOnly.GET("/reports", drivers.Report)

	// Chats between partners and travelers.
	chats := &handlers.Chats{Deps: d}
	r.GET("/chats", requireUser, chats.List)
	r.GET("/chats/unread", requireUser, chats.Unread)
	r.POST("/chats", requireUser, chats.Open)
	r.GET("/chats/:id/messages", requireUser, chats.Messages)
	r.POST("/chats/:id/messages", requireUser, chats.Send)
	r.PUT("/chats/:id/read", requireUser, chats.Read)

	return r
}
