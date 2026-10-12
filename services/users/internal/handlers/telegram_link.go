package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/users/internal/store"
	"github.com/elvn5/sai/services/users/internal/verification"
)

// TelegramLinkTTL is how long a link token stays valid.
const TelegramLinkTTL = 15 * time.Minute

// TelegramLink lets a signed-in site user link their Telegram account: the
// site gets a token for a t.me/<bot>?start=link_<token> deep link, and the
// bot completes the link (the LinkTelegram RPC). Opening the bot this way
// also lets it message the user. Runs behind RequireUser.
type TelegramLink struct{ Deps }

// Start issues a fresh link token.
func (h *TelegramLink) Start(c *gin.Context) {
	ctx := c.Request.Context()
	user, err := h.Store.ByID(ctx, currentUserID(c))
	if err != nil {
		h.userError(c, err)
		return
	}
	if user.TelegramID != nil {
		httperr.Abort(c, http.StatusConflict, "Telegram is already linked")
		return
	}
	token, err := verification.Token()
	if err != nil {
		h.internalError(c, err)
		return
	}
	expires := h.now().Add(TelegramLinkTTL)
	if err := h.Store.SetTelegramLinkToken(ctx, user.ID, verification.Hash(token), expires); err != nil {
		h.userError(c, err)
		return
	}
	respond.JSON(c, http.StatusCreated, gin.H{"token": token, "expiresAt": jsontime.New(expires)})
}

// Unlink removes the Telegram account, unless it's the only way to sign in.
func (h *TelegramLink) Unlink(c *gin.Context) {
	ctx := c.Request.Context()
	user, err := h.Store.ByID(ctx, currentUserID(c))
	if err != nil {
		h.userError(c, err)
		return
	}
	if user.TelegramID == nil {
		httperr.Abort(c, http.StatusConflict, "Telegram is not linked")
		return
	}
	if user.PasswordHash == nil {
		httperr.Abort(c, http.StatusConflict, "Telegram is your only way to sign in")
		return
	}
	saved, err := h.Store.UnlinkTelegram(ctx, user.ID)
	if err != nil {
		h.userError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, gin.H{"user": saved.Public()})
}

func (h *TelegramLink) userError(c *gin.Context, err error) {
	if errors.Is(err, store.ErrNotFound) {
		httperr.Abort(c, http.StatusUnauthorized, "User not found")
		return
	}
	h.internalError(c, err)
}
