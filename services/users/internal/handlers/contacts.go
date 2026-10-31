package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
	"github.com/elvn5/sai/services/users/internal/models"
	"github.com/elvn5/sai/services/users/internal/store"
	"github.com/elvn5/sai/services/users/internal/verification"
)

// Contacts lets a signed-in account without an email (one made by
// signing in with Telegram) add one, confirmed by a code sent to it. An
// email that already belongs to another account is that person's: the
// Telegram account is merged into it and the session moves there. Runs
// behind RequireUser.
type Contacts struct{ Deps }

type addEmailRequest struct {
	Email *string `json:"email" binding:"required,zodemail,max=255"`
}

type confirmEmailRequest struct {
	Code *string `json:"code" binding:"required,sixdigits"`
}

// AddEmail handles POST /auth/user/email: sends a code to the email.
func (h *Contacts) AddEmail(c *gin.Context) {
	var req addEmailRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	user, ok := h.currentUser(c)
	if !ok {
		return
	}
	if user.Email != nil {
		httperr.Abort(c, http.StatusConflict, "The account already has an email")
		return
	}
	address := strings.TrimSpace(*req.Email)
	owner, err := h.Store.ByIdentifier(ctx, address)
	switch {
	case errors.Is(err, store.ErrNotFound):
	case err != nil:
		h.internalError(c, err)
		return
	case store.CanMerge(user, owner) != nil:
		httperr.Abort(c, http.StatusConflict, "Email is already registered")
		return
	}
	if user.VerificationCodeSentAt != nil && h.now().Sub(*user.VerificationCodeSentAt) < verification.ResendCooldown {
		httperr.AbortPlain(c, http.StatusTooManyRequests, "Please wait before requesting another code")
		return
	}

	code, err := verification.Code()
	if err != nil {
		h.internalError(c, err)
		return
	}
	store.SetVerification(&user, verification.Hash(code), verification.CodeTTL, h.now())
	user.PendingEmail = &address
	if _, err := h.Store.Save(ctx, user); err != nil {
		h.internalError(c, err)
		return
	}
	if err := h.Email.SendVerificationCode(ctx, address, code); err != nil {
		h.Log.ErrorContext(ctx, "failed to send the email code", slog.Any("error", err))
		httperr.Abort(c, http.StatusServiceUnavailable, "Could not send the code. Please try again later.")
		return
	}
	respond.JSON(c, http.StatusOK, gin.H{"verification": PendingVerification{Identifier: address, Method: "email"}})
}

// ConfirmEmail handles POST /auth/user/email/confirm: with the right code
// the pending email becomes the account's, or, when another account has
// it, this account is merged into that one (merged: true; the session
// cookie is now that account's).
func (h *Contacts) ConfirmEmail(c *gin.Context) {
	var req confirmEmailRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	user, ok := h.currentUser(c)
	if !ok {
		return
	}
	switch {
	case user.PendingEmail == nil || user.VerificationCodeHash == nil || user.VerificationCodeExpiresAt == nil:
		httperr.Abort(c, http.StatusBadRequest, "No verification code pending. Please request a new one.")
		return
	case user.VerificationCodeExpiresAt.Before(h.now()):
		httperr.Abort(c, http.StatusBadRequest, "Code has expired. Please request a new one.")
		return
	case user.VerificationAttempts >= verification.MaxAttempts:
		httperr.Abort(c, http.StatusBadRequest, "Too many attempts. Please request a new code.")
		return
	}
	if !verification.Matches(*req.Code, *user.VerificationCodeHash) {
		user.VerificationAttempts++
		if _, err := h.Store.Save(ctx, user); err != nil {
			h.internalError(c, err)
			return
		}
		httperr.Abort(c, http.StatusUnauthorized, "Invalid code")
		return
	}

	address := *user.PendingEmail
	store.ClearVerification(&user)
	user.PendingEmail = nil
	owner, err := h.Store.ByIdentifier(ctx, address)
	if errors.Is(err, store.ErrNotFound) {
		user.Email = &address
		saved, err := h.Store.Save(ctx, user)
		if errors.Is(err, store.ErrDuplicate) {
			httperr.Abort(c, http.StatusConflict, "Email is already registered")
			return
		} else if err != nil {
			h.internalError(c, err)
			return
		}
		respond.JSON(c, http.StatusOK, gin.H{"user": saved.Public(), "merged": false})
		return
	} else if err != nil {
		h.internalError(c, err)
		return
	}

	if _, err := h.Store.Save(ctx, user); err != nil {
		h.internalError(c, err)
		return
	}
	merged, err := h.Store.Merge(ctx, user.ID, owner.ID, store.MergeEmail)
	if errors.Is(err, store.ErrNotMergeable) || errors.Is(err, store.ErrSameAccount) {
		httperr.Abort(c, http.StatusConflict, "Email is already registered")
		return
	} else if err != nil {
		h.internalError(c, err)
		return
	}
	h.Log.InfoContext(ctx, "accounts merged by email", slog.String("from", user.ID), slog.String("to", merged.ID))
	h.startMergedSession(c, merged)
}

// startMergedSession signs the browser in to the account it was merged
// into.
func (h *Contacts) startMergedSession(c *gin.Context, user models.User) {
	token, err := h.Signer.UserToken(user.ID)
	if err != nil {
		h.internalError(c, err)
		return
	}
	h.Cookies.SetUser(c, token)
	respond.JSON(c, http.StatusOK, gin.H{"user": user.Public(), "merged": true})
}

func (h *Contacts) currentUser(c *gin.Context) (models.User, bool) {
	user, err := h.Store.ByID(c.Request.Context(), currentUserID(c))
	if errors.Is(err, store.ErrNotFound) {
		httperr.Abort(c, http.StatusUnauthorized, "User not found")
		return user, false
	} else if err != nil {
		h.internalError(c, err)
		return user, false
	}
	return user, true
}
