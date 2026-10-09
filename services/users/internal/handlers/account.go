package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/auth"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/users/internal/session"
	"github.com/elvn5/sai/services/users/internal/store"
)

// Account serves the signed-in site user's own session and partner
// application.
type Account struct{ Deps }

// Me runs behind RequireUser.
func (h *Account) Me(c *gin.Context) {
	user, err := h.Store.ByID(c.Request.Context(), currentUserID(c))
	if errors.Is(err, store.ErrNotFound) {
		httperr.Abort(c, http.StatusUnauthorized, "User not found")
		return
	} else if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, gin.H{"user": user.Public()})
}

func (h *Account) Logout(c *gin.Context) {
	session.Clear(c, auth.UserCookie)
	respond.JSON(c, http.StatusOK, gin.H{"success": true})
}

// ApplyForPartnership runs behind RequireUser.
func (h *Account) ApplyForPartnership(c *gin.Context) {
	ctx := c.Request.Context()
	user, err := h.Store.ByID(ctx, currentUserID(c))
	if errors.Is(err, store.ErrNotFound) {
		httperr.Abort(c, http.StatusNotFound, "User not found")
		return
	} else if err != nil {
		h.internalError(c, err)
		return
	}
	if user.PartnerStatus == "pending" || user.PartnerStatus == "approved" {
		httperr.Abort(c, http.StatusConflict, "Partner request already submitted")
		return
	}
	user.PartnerStatus = "pending"
	user.PartnerRequestedAt = ptr(h.now())
	saved, err := h.Store.Save(ctx, user)
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusCreated, gin.H{"user": saved.Public()})
}
