// Package handlers holds the trails service's HTTP handlers.
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

	"github.com/elvn5/sai/services/pkg/audit"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/middleware"
	"github.com/elvn5/sai/services/pkg/usersclient"
	"github.com/elvn5/sai/services/pkg/validation"
	"github.com/elvn5/sai/services/trails/internal/models"
	"github.com/elvn5/sai/services/trails/internal/store"
)

func init() {
	validation.RegisterPattern("slug", regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`),
		"Slug must be lowercase, alphanumeric, hyphen-separated")
	validation.RegisterPattern("mediaurl", regexp.MustCompile(`^(https?://|/uploads/)`), "Must be an uploaded file URL")
	validation.RegisterRule("isodate", func(s string) bool {
		_, err := time.Parse(time.DateOnly, s)
		return err == nil && len(s) == len(time.DateOnly)
	}, "Invalid date")
	validation.RegisterRule("datetime", func(s string) bool {
		_, err := time.Parse(time.RFC3339, s)
		return err == nil
	}, "Invalid date and time")
}

// AuthorSource resolves public profiles (the users service).
type AuthorSource interface {
	Authors(ctx context.Context, ids []string) (map[string]usersclient.Author, error)
}

// Deps is what the handlers share.
type Deps struct {
	Store   *store.Store
	Authors AuthorSource
	Log     *slog.Logger
	Now     func() time.Time
	// The admin panel's history; nil records nothing.
	History *audit.Log
}

func (d Deps) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

// fail answers a store error with notFoundMessage for a missing row.
func (d Deps) fail(c *gin.Context, err error, notFoundMessage string) {
	if errors.Is(err, store.ErrNotFound) {
		httperr.Abort(c, http.StatusNotFound, notFoundMessage)
		return
	}
	d.internalError(c, err)
}

func (d Deps) internalError(c *gin.Context, err error) {
	d.Log.ErrorContext(c.Request.Context(), "request failed", slog.String("path", c.Request.URL.Path), slog.Any("error", err))
	c.AbortWithStatusJSON(http.StatusInternalServerError, httperr.InternalServerError())
}

// people resolves user ids to public profiles; unknown ids keep just the
// id. A users-service outage degrades to bare ids rather than failing.
func (d Deps) people(ctx context.Context, ids []string) map[string]models.Person {
	out := make(map[string]models.Person, len(ids))
	authors, err := d.Authors.Authors(ctx, ids)
	if err != nil {
		d.Log.WarnContext(ctx, "could not load profiles", slog.Any("error", err))
	}
	for _, id := range ids {
		a, ok := authors[id]
		if !ok {
			out[id] = models.Person{ID: id}
			continue
		}
		out[id] = models.Person{ID: a.ID, FirstName: a.FirstName, Username: a.Username, AvatarURL: a.AvatarURL}
	}
	return out
}

func validID(id string) bool { return uuid.Validate(id) == nil }

func currentUserID(c *gin.Context) string { return c.GetString(middleware.ContextUserID) }

func isAdmin(c *gin.Context) bool { return c.GetBool(middleware.ContextIsAdmin) }

func applyField[T any](dst **T, f validation.Field[T]) {
	if f.Set {
		*dst = f.Ptr()
	}
}
