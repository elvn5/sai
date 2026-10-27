package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/telegram"
	"github.com/elvn5/sai/services/pkg/validation"
	"github.com/elvn5/sai/services/users/internal/resetmsg"
	"github.com/elvn5/sai/services/users/internal/store"
	"github.com/elvn5/sai/services/users/internal/verification"
)

const (
	// PasswordResetTTL is how long a reset link works (the messages say so).
	PasswordResetTTL = 30 * time.Minute
	// How long sending a reset link may take after the request is answered.
	resetDeliveryTimeout = 30 * time.Second

	ResetChannelEmail    = "email"
	ResetChannelTelegram = "telegram"
)

var (
	// Reset requests per typed email or phone, whether or not it has an
	// account: once a minute, five an hour.
	resetLimitsPerIdentifier = []store.ResetLimit{{Max: 1, Window: time.Minute}, {Max: 5, Window: time.Hour}}
	resetLimitsPerIP         = []store.ResetLimit{{Max: 20, Window: time.Hour}}
)

type forgotPasswordRequest struct {
	Identifier *string `json:"identifier" binding:"required,min=1,max=255"`
	// Where the link goes; by default email for an email, Telegram for a
	// phone.
	Channel *string `json:"channel" binding:"omitnil,oneof=email telegram"`
	// The site's language, for the email and the link.
	Lang *string `json:"lang" binding:"omitnil,oneof=ru ky en de"`
}

type resetPasswordRequest struct {
	Token    *string `json:"token" binding:"required,min=1,max=128"`
	Password *string `json:"password" binding:"required,min=8,bcryptpassword"`
}

// ForgotPassword sends a one-time link for setting a new password: by
// email, or to the Telegram chat linked to the account. The answer is the
// same whether or not there is such an account (or a linked Telegram), and
// the link goes out after the answer, so neither the reply nor its timing
// tells who has an account. Requests are limited per typed identifier and
// per IP (429).
func (h *Credentials) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	identifier := strings.TrimSpace(*req.Identifier)
	channel := ResetChannelTelegram
	if strings.Contains(identifier, "@") {
		channel = ResetChannelEmail
	}
	if req.Channel != nil {
		channel = *req.Channel
	}
	if channel == ResetChannelEmail && !strings.Contains(identifier, "@") {
		httperr.Abort(c, http.StatusUnprocessableEntity, "A reset link can only be emailed to an email address")
		return
	}
	allowed, err := h.Store.AllowResetRequest(ctx, verification.Hash(strings.ToLower(identifier)), c.ClientIP(),
		h.now(), resetLimitsPerIdentifier, resetLimitsPerIP)
	if err != nil {
		h.internalError(c, err)
		return
	}
	if !allowed {
		httperr.AbortPlain(c, http.StatusTooManyRequests, "Please wait before requesting another link")
		return
	}
	lang := telegram.RU
	if req.Lang != nil {
		lang = telegram.ParseLang(*req.Lang)
	}
	detached := context.WithoutCancel(ctx)
	h.background(func() {
		ctx, cancel := context.WithTimeout(detached, resetDeliveryTimeout)
		defer cancel()
		h.sendPasswordReset(ctx, identifier, channel, lang)
	})
	respond.JSON(c, http.StatusOK, gin.H{"channel": channel})
}

// sendPasswordReset issues a link for the account behind identifier and
// sends it over channel, if the account can receive it there; anything
// else is only logged.
func (h *Credentials) sendPasswordReset(ctx context.Context, identifier, channel string, lang telegram.Lang) {
	user, err := h.Store.ByIdentifier(ctx, identifier)
	if errors.Is(err, store.ErrNotFound) {
		return
	} else if err != nil {
		h.Log.ErrorContext(ctx, "password reset: account lookup failed", slog.Any("error", err))
		return
	}
	var chat store.TelegramChat
	switch channel {
	case ResetChannelEmail:
		if user.Email == nil {
			return
		}
	case ResetChannelTelegram:
		if user.TelegramID == nil {
			return
		}
		if h.Telegram == nil {
			h.Log.WarnContext(ctx, "password reset: Telegram is not configured")
			return
		}
		chats, err := h.Store.TelegramChats(ctx, []string{user.ID})
		if err != nil {
			h.Log.ErrorContext(ctx, "password reset: chat lookup failed", slog.Any("error", err))
			return
		}
		chat = chats[user.ID]
		// The recipient's language: the chat's, not the site's.
		lang = telegram.ParseLang(chat.Lang)
	}

	token, err := verification.Token()
	if err != nil {
		h.Log.ErrorContext(ctx, "password reset: token failed", slog.Any("error", err))
		return
	}
	if err := h.Store.CreatePasswordReset(ctx, user.ID, verification.Hash(token), channel, h.now().Add(PasswordResetTTL)); err != nil {
		h.Log.ErrorContext(ctx, "password reset: saving the token failed", slog.Any("error", err))
		return
	}
	link := resetmsg.Link(h.SiteURL, lang, token)
	if channel == ResetChannelEmail {
		err = h.Email.SendPasswordReset(ctx, *user.Email, lang, link)
	} else {
		text, button := resetmsg.Telegram(lang, link)
		err = h.Telegram.SendMessage(ctx, chat.ID, text, &button)
	}
	if err != nil {
		h.Log.ErrorContext(ctx, "password reset: sending the link failed",
			slog.String("channel", channel), slog.String("userId", user.ID), slog.Any("error", err))
		return
	}
	h.Log.InfoContext(ctx, "password reset link sent", slog.String("channel", channel), slog.String("userId", user.ID))
}

// ResetPassword sets a new password with the token from a reset link and
// signs in. The token works once; every session issued before the reset
// stops working, on every device.
func (h *Credentials) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	tokenHash := verification.Hash(*req.Token)
	// Cheap check first, so made-up tokens don't cost a bcrypt hash.
	if valid, err := h.Store.PasswordResetValid(ctx, tokenHash, h.now()); err != nil {
		h.internalError(c, err)
		return
	} else if !valid {
		httperr.Abort(c, http.StatusBadRequest, "Reset link is invalid or has expired")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), passwordCost)
	if err != nil {
		h.internalError(c, err)
		return
	}
	// Session tokens carry the real time (whole seconds) as iat, so the
	// cut-off is taken from the same clock; the session started below,
	// issued within this second, stays valid.
	validAfter := time.Now().Truncate(time.Second)
	user, err := h.Store.ResetPassword(ctx, tokenHash, string(hash), h.now(), validAfter)
	if errors.Is(err, store.ErrResetInvalid) {
		httperr.Abort(c, http.StatusBadRequest, "Reset link is invalid or has expired")
		return
	} else if err != nil {
		h.internalError(c, err)
		return
	}
	h.Log.InfoContext(ctx, "password reset", slog.String("userId", user.ID))
	h.startSession(c, user)
}
