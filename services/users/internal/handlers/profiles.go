package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/users/internal/store"
)

// Profiles serves public profiles: an organizer's page on the site.
type Profiles struct{ Deps }

// publicProfile is what anyone may see about a user: never contacts.
type publicProfile struct {
	ID        string  `json:"id"`
	FirstName *string `json:"firstName"`
	Username  *string `json:"username"`
	AvatarURL *string `json:"avatarUrl"`
	// An approved partner: the site shows them as verified.
	IsVerified  bool      `json:"isVerified"`
	MemberSince time.Time `json:"memberSince"`
}

func (h *Profiles) Public(c *gin.Context) {
	id := c.Param("id")
	if uuid.Validate(id) != nil {
		httperr.Abort(c, http.StatusNotFound, "User not found")
		return
	}
	user, err := h.Store.ByID(c.Request.Context(), id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		httperr.Abort(c, http.StatusNotFound, "User not found")
		return
	case err != nil:
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, publicProfile{
		ID: user.ID, FirstName: user.FirstName, Username: user.Username, AvatarURL: user.AvatarURL,
		IsVerified: user.PartnerStatus == "approved", MemberSince: user.CreatedAt,
	})
}
