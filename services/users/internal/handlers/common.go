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
	"github.com/elvn5/sai/services/pkg/telegram"
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
	validation.RegisterRule("bcryptpassword", func(s string) bool { return len(s) <= 72 }, "Password must contain at most 72 UTF-8 bytes")
}

// TelegramSender messages a Telegram chat as the bot (pkg/telegram.Client).
type TelegramSender interface {
	SendMessage(ctx context.Context, chatID int64, text string, button *telegram.Button) error
}

// Deps is what the handlers share.
type Deps struct {
	Store   *store.Store
	Signer  *auth.Signer
	Cookies *session.Cookies
	Email   email.Sender
	// Nil without BOT_TOKEN: no reset links in Telegram.
	Telegram      TelegramSender
	SiteURL       string
	AdminPassword string
	BotToken      string
	Log           *slog.Logger
	Now           func() time.Time
	// Background runs work that outlives the request (sending a reset
	// link); nil means a goroutine. Tests run it inline.
	Background func(func())
}

func (d Deps) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

func (d Deps) background(work func()) {
	if d.Background != nil {
		d.Background(work)
		return
	}
	go work()
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
	out := middleware.SessionUser{ID: u.ID, PartnerStatus: u.PartnerStatus, PartnerKind: u.PartnerKind}
	if u.SessionsValidAfter != nil {
		out.SessionsValidAfter = *u.SessionsValidAfter
	}
	return out, nil
}

func currentUserID(c *gin.Context) string { return c.GetString(middleware.ContextUserID) }

func ptr[T any](v T) *T { return &v }
