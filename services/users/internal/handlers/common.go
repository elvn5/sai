// Package handlers holds the users service's HTTP handlers: sign-in
// (password, Telegram, admin), registration and its verification,
// partner applications, and the admin user list and support contacts.
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

	"github.com/elvn5/sai/services/pkg/auth"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/middleware"
	"github.com/elvn5/sai/services/pkg/validation"
	"github.com/elvn5/sai/services/users/internal/email"
	"github.com/elvn5/sai/services/users/internal/session"
	"github.com/elvn5/sai/services/users/internal/store"
)

// zod's z.string().email() (v3.25), minus the lookaheads Go's regexp
// lacks, which are checked separately.
var zodEmail = regexp.MustCompile(`(?i)^[A-Z0-9_'+\-.]*[A-Z0-9_+-]@([A-Z0-9][A-Z0-9\-]*\.)+[A-Z]{2,}$`)

func init() {
	validation.RegisterRule("zodemail", func(s string) bool {
		return !strings.HasPrefix(s, ".") && !strings.Contains(s, "..") && zodEmail.MatchString(s)
	}, "Invalid email")
	validation.RegisterPattern("sixdigits", regexp.MustCompile(`^\d{6}$`), "Code must be 6 digits")
	validation.RegisterRule("adminpassword", func(s string) bool { return s != "" }, "Password is required")
}

// Deps is what the handlers share.
type Deps struct {
	Store         *store.Store
	Signer        *auth.Signer
	Cookies       *session.Cookies
	Email         email.Sender
	AdminPassword string
	BotToken      string
	Log           *slog.Logger
	Now           func() time.Time
}

func (d Deps) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

// internalError logs err and answers NestJS's generic 500.
func (d Deps) internalError(c *gin.Context, err error) {
	d.Log.ErrorContext(c.Request.Context(), "request failed", slog.String("path", c.Request.URL.Path), slog.Any("error", err))
	c.AbortWithStatusJSON(http.StatusInternalServerError, httperr.InternalServerError())
}

// Lookup lets the shared auth middleware check sessions against this
// service's own store.
type Lookup struct{ Store *store.Store }

func (l Lookup) LookupUser(ctx context.Context, id string) (middleware.SessionUser, error) {
	u, err := l.Store.ByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return middleware.SessionUser{}, middleware.ErrUserNotFound
	}
	if err != nil {
		return middleware.SessionUser{}, err
	}
	return middleware.SessionUser{ID: u.ID, PartnerStatus: u.PartnerStatus}, nil
}

func currentUserID(c *gin.Context) string { return c.GetString(middleware.ContextUserID) }

func ptr[T any](v T) *T { return &v }
