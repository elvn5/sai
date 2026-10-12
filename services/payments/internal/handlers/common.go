// Package handlers holds the payments service's HTTP handlers.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/elvn5/sai/services/payments/internal/listingsclient"
	"github.com/elvn5/sai/services/payments/internal/models"
	"github.com/elvn5/sai/services/payments/internal/provider"
	"github.com/elvn5/sai/services/payments/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/middleware"
	"github.com/elvn5/sai/services/pkg/usersclient"
	"github.com/elvn5/sai/services/pkg/validation"
)

func init() {
	validation.RegisterPattern("phone", regexp.MustCompile(`^\+?[0-9\s\-()]{5,32}$`), "Invalid phone number")
}

// Tours looks up tours (the listings service).
type Tours interface {
	Tour(ctx context.Context, id string) (listingsclient.Tour, error)
}

// Authors resolves public profiles (the users service).
type Authors interface {
	Authors(ctx context.Context, ids []string) (map[string]usersclient.Author, error)
}

// Checker settles a pending order against the gateway (internal/worker).
type Checker interface {
	Check(ctx context.Context, o models.Order) (models.Order, error)
}

// Deps is what the handlers share.
type Deps struct {
	Store      *store.Store
	Provider   provider.Provider
	Tours      Tours
	Authors    Authors
	Checker    Checker
	FeePercent float64
	PaymentTTL time.Duration
	SiteURL    string
	Log        *slog.Logger
	Now        func() time.Time
}

func (d Deps) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

func (d Deps) fail(c *gin.Context, err error, notFoundMessage string) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		httperr.Abort(c, http.StatusNotFound, notFoundMessage)
	case errors.Is(err, store.ErrConflict):
		httperr.Abort(c, http.StatusConflict, "The order can't be changed in its current state")
	default:
		d.internalError(c, err)
	}
}

func (d Deps) internalError(c *gin.Context, err error) {
	d.Log.ErrorContext(c.Request.Context(), "request failed", slog.String("path", c.Request.URL.Path), slog.Any("error", err))
	c.AbortWithStatusJSON(http.StatusInternalServerError, httperr.InternalServerError())
}

// withPeople fills in payer and payee profiles (best effort).
func (d Deps) withPeople(ctx context.Context, orders []models.Order) {
	ids := make([]string, 0, 2*len(orders))
	for _, o := range orders {
		ids = append(ids, o.UserID, o.PayeeID)
	}
	authors, err := d.Authors.Authors(ctx, ids)
	if err != nil {
		d.Log.WarnContext(ctx, "could not load profiles", slog.Any("error", err))
	}
	person := func(id string) *models.Person {
		a := authors[id]
		return &models.Person{ID: id, FirstName: a.FirstName, Username: a.Username}
	}
	for i := range orders {
		orders[i].Payer, orders[i].Payee = person(orders[i].UserID), person(orders[i].PayeeID)
	}
}

func validID(id string) bool { return uuid.Validate(id) == nil }

func currentUserID(c *gin.Context) string { return c.GetString(middleware.ContextUserID) }

func isAdmin(c *gin.Context) bool { return c.GetBool(middleware.ContextIsAdmin) }

var errNotFound = store.ErrNotFound

var errConflict = store.ErrConflict
