// Package handlers holds the listings service's HTTP handlers.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/elvn5/sai/services/listings/internal/alerts"
	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/middleware"
	"github.com/elvn5/sai/services/pkg/usersclient"
	"github.com/elvn5/sai/services/pkg/validation"
)

// Field rules shared by the listing request bodies.
func init() {
	validation.RegisterPattern("slug", regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`),
		"Slug must be lowercase, alphanumeric, hyphen-separated")
	validation.RegisterPattern("mediaurl", regexp.MustCompile(`^(https?://|/uploads/)`), "Must be an uploaded file URL")
	validation.RegisterPattern("phone", regexp.MustCompile(`^\+?[0-9\s\-()]{5,32}$`), "Invalid phone number")
	validation.RegisterPattern("telegram", regexp.MustCompile(`^@?[A-Za-z0-9_]{5,32}$`), "Invalid Telegram username")
	validation.RegisterPattern("hhmm", regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`), "Expected HH:MM")
	// z.string().date(): a real calendar date in YYYY-MM-DD form.
	validation.RegisterRule("isodate", func(s string) bool {
		_, err := time.Parse(time.DateOnly, s)
		return err == nil && len(s) == len(time.DateOnly)
	}, "Invalid date")
}

// AuthorSource resolves public author profiles (the users service).
type AuthorSource interface {
	Authors(ctx context.Context, ids []string) (map[string]usersclient.Author, error)
}

// TourNotifier queues Telegram notifications about sign-ups for tours
// (internal/notify); optional.
type TourNotifier interface {
	Joined(ctx context.Context, t models.Tour, userID string)
	Left(ctx context.Context, t models.Tour, userID string)
	Changed(ctx context.Context, before, after models.Tour, participantIDs []string)
	Cancelled(ctx context.Context, t models.Tour, participantIDs []string)
	// Promoted tells someone on the waitlist they got a place.
	Promoted(ctx context.Context, t models.Tour, userID string)
	// Message tells the other side of a chat about a new message.
	Message(ctx context.Context, chat store.Chat, senderID, body string)
}

// Deps is what the handlers share.
type Deps struct {
	Store   *store.Store
	Authors AuthorSource
	// Telegram messages about new tours; nil when the bot isn't configured.
	Alerts *alerts.Notifier
	// Telegram messages about sign-ups; nil when the bot isn't configured.
	Notify TourNotifier
	Log    *slog.Logger
	// Now is overridable in tests.
	Now func() time.Time
}

// today is the current time in Bishkek, whose calendar tours follow.
func (d Deps) today() time.Time {
	if d.Now != nil {
		return d.Now().In(bishkek)
	}
	return time.Now().In(bishkek)
}

func (d Deps) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

// fail answers a store error. Messages for "not found" and duplicates
// differ per endpoint, as in the NestJS API.
func (d Deps) fail(c *gin.Context, err error, notFoundMessage, duplicateMessage string) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		httperr.Abort(c, http.StatusNotFound, notFoundMessage)
	case errors.Is(err, store.ErrDuplicate) && duplicateMessage != "":
		httperr.Abort(c, http.StatusConflict, duplicateMessage)
	default:
		d.Log.ErrorContext(c.Request.Context(), "request failed", slog.String("path", c.Request.URL.Path), slog.Any("error", err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, httperr.InternalServerError())
	}
}

// validID: NestJS passed malformed ids to Postgres (500); here they're
// simply not found.
func validID(id string) bool { return uuid.Validate(id) == nil }

func currentUserID(c *gin.Context) string { return c.GetString(middleware.ContextUserID) }

func isAdmin(c *gin.Context) bool { return c.GetBool(middleware.ContextIsAdmin) }

func ownedBy(owner *string, userID string) bool { return owner != nil && *owner == userID }

func applyField[T any](dst **T, f validation.Field[T]) {
	if f.Set {
		*dst = f.Ptr()
	}
}

func applyTrimmed(dst **string, f validation.Field[validation.Trimmed]) {
	if !f.Set {
		return
	}
	if !f.Valid {
		*dst = nil
		return
	}
	s := string(f.Value)
	*dst = &s
}

// contactFields is contactFieldsSchema: optional, nullable phone numbers and
// a Telegram username stored without its leading @. Request structs
// declare these three fields themselves (with the same tags) rather than
// embedding — validator would put an embedded type's name into error paths.
type contactFields struct {
	Phone, Whatsapp, Telegram validation.Field[string]
}

func (f contactFields) apply(phone, whatsapp, telegram **string) {
	applyField(phone, f.Phone)
	applyField(whatsapp, f.Whatsapp)
	applyField(telegram, f.Telegram)
	if *telegram != nil {
		stripped := strings.TrimPrefix(**telegram, "@")
		*telegram = &stripped
	}
}

// authorFor fills in a missing profile with just the id.
func authorFor(authors map[string]usersclient.Author, id string) usersclient.Author {
	if a, ok := authors[id]; ok {
		return a
	}
	return usersclient.Author{ID: id}
}
