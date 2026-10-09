package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
	"github.com/elvn5/sai/services/users/internal/models"
	"github.com/elvn5/sai/services/users/internal/store"
)

type supportRequest struct {
	Message         *string                  `json:"message" binding:"required,min=1,max=4000"`
	Phone           validation.Field[string] `json:"phone" binding:"omitempty,max=50"`
	TelegramContact validation.Field[string] `json:"telegramContact" binding:"omitempty,max=255"`
	Email           validation.Field[string] `json:"email" binding:"omitempty,zodemail,max=255"`
	WorkingHours    validation.Field[string] `json:"workingHours" binding:"omitempty,max=255"`
}

// Admin serves the admin panel's user list, partner requests and support
// contacts. Every route runs behind RequireAdmin.
type Admin struct{ Deps }

func (h *Admin) ListUsers(c *gin.Context) {
	users, err := h.Store.All(c.Request.Context())
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, models.AdminUsers(users))
}

func (h *Admin) GetUser(c *gin.Context) {
	user, ok := h.user(c, c.Param("id"))
	if ok {
		respond.JSON(c, http.StatusOK, user.Admin())
	}
}

func (h *Admin) PartnerRequests(c *gin.Context) {
	users, err := h.Store.PendingPartners(c.Request.Context())
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, models.AdminUsers(users))
}

func (h *Admin) ApprovePartner(c *gin.Context) { h.setPartnerStatus(c, "approved") }
func (h *Admin) RejectPartner(c *gin.Context)  { h.setPartnerStatus(c, "rejected") }

func (h *Admin) setPartnerStatus(c *gin.Context, status string) {
	user, ok := h.user(c, c.Param("userId"))
	if !ok {
		return
	}
	user.PartnerStatus = status
	saved, err := h.Store.Save(c.Request.Context(), user)
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusCreated, saved.SavedAdmin())
}

func (h *Admin) GetSupport(c *gin.Context) {
	info, err := h.Store.EnsureSupport(c.Request.Context())
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, info)
}

func (h *Admin) UpdateSupport(c *gin.Context) {
	var req supportRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	info, err := h.Store.EnsureSupport(ctx)
	if err != nil {
		h.internalError(c, err)
		return
	}
	info.Message = *req.Message
	for dst, f := range map[**string]validation.Field[string]{
		&info.Phone: req.Phone, &info.TelegramContact: req.TelegramContact,
		&info.Email: req.Email, &info.WorkingHours: req.WorkingHours,
	} {
		if f.Set {
			*dst = f.Ptr()
		}
	}
	saved, err := h.Store.SaveSupport(ctx, info)
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, saved)
}

// user loads an account for an admin route; a malformed id is just not
// found (NestJS answered 500).
func (h *Admin) user(c *gin.Context, id string) (models.User, bool) {
	user, err := h.Store.ByID(c.Request.Context(), id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		httperr.Abort(c, http.StatusNotFound, "User not found")
		return user, false
	case err != nil:
		h.internalError(c, err)
		return user, false
	}
	return user, true
}
