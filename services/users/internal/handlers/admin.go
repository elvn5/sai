package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/audit"
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
	audit.SetSubject(c, displayName(user))
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

type createPartnerRequest struct {
	UserID *string `json:"userId" binding:"required,uuid"`
	Kind   string  `json:"kind" binding:"omitempty,oneof=organizer driver"`
}

type updatePartnerRequest struct {
	Kind      *string                  `json:"kind" binding:"omitnil,oneof=organizer driver"`
	Status    *string                  `json:"status" binding:"omitnil,oneof=pending approved rejected"`
	FirstName validation.Field[string] `json:"firstName" binding:"omitempty,max=255"`
	LastName  validation.Field[string] `json:"lastName" binding:"omitempty,max=255"`
	Phone     validation.Field[string] `json:"phone" binding:"omitempty,max=50"`
}

// Partners lists partners for the panel: ?status=pending|approved|rejected,
// or every account with a partner status.
func (h *Admin) Partners(c *gin.Context) {
	status := c.Query("status")
	switch status {
	case "", "pending", "approved", "rejected":
	default:
		httperr.Abort(c, http.StatusBadRequest, "status must be one of pending, approved, rejected")
		return
	}
	users, err := h.Store.Partners(c.Request.Context(), status)
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, models.AdminUsers(users))
}

// CreatePartner makes an existing account an approved partner.
func (h *Admin) CreatePartner(c *gin.Context) {
	var req createPartnerRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	user, ok := h.user(c, *req.UserID)
	if !ok {
		return
	}
	if user.PartnerStatus == "approved" {
		httperr.Abort(c, http.StatusConflict, "This user is already a partner")
		return
	}
	if req.Kind == "" {
		req.Kind = "organizer"
	}
	user.PartnerStatus = "approved"
	user.PartnerKind = req.Kind
	if user.PartnerRequestedAt == nil {
		now := h.now()
		user.PartnerRequestedAt = &now
	}
	h.savePartner(c, user, http.StatusCreated)
}

// UpdatePartner changes a partner's kind, status or contact details.
func (h *Admin) UpdatePartner(c *gin.Context) {
	var req updatePartnerRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	user, ok := h.partner(c)
	if !ok {
		return
	}
	if req.Kind != nil {
		user.PartnerKind = *req.Kind
	}
	if req.Status != nil {
		user.PartnerStatus = *req.Status
	}
	for dst, f := range map[**string]validation.Field[string]{
		&user.FirstName: req.FirstName, &user.LastName: req.LastName, &user.Phone: req.Phone,
	} {
		if f.Set {
			*dst = f.Ptr()
		}
	}
	h.savePartner(c, user, http.StatusOK)
}

// DeletePartner takes partnership away; the account stays, as an ordinary
// traveler who can apply again.
func (h *Admin) DeletePartner(c *gin.Context) {
	user, ok := h.partner(c)
	if !ok {
		return
	}
	user.PartnerStatus = "none"
	user.PartnerKind = "organizer"
	user.PartnerRequestedAt = nil
	h.savePartner(c, user, http.StatusOK)
}

// partner loads the account in :userId, which must have a partner status.
func (h *Admin) partner(c *gin.Context) (models.User, bool) {
	user, ok := h.user(c, c.Param("userId"))
	if ok && user.PartnerStatus == "none" {
		httperr.Abort(c, http.StatusNotFound, "Partner not found")
		return user, false
	}
	return user, ok
}

func (h *Admin) savePartner(c *gin.Context, user models.User, status int) {
	audit.SetSubject(c, displayName(user))
	saved, err := h.Store.Save(c.Request.Context(), user)
	switch {
	case errors.Is(err, store.ErrDuplicate):
		httperr.Abort(c, http.StatusConflict, "This phone belongs to another account")
		return
	case err != nil:
		h.internalError(c, err)
		return
	}
	respond.JSON(c, status, saved.Admin())
}

// displayName is how the history names an account: its name, else its
// email, phone or Telegram id.
func displayName(u models.User) string {
	var parts []string
	for _, p := range []*string{u.FirstName, u.LastName} {
		if p != nil && *p != "" {
			parts = append(parts, *p)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	for _, p := range []*string{u.Email, u.Phone, u.Username} {
		if p != nil && *p != "" {
			return *p
		}
	}
	if u.TelegramID != nil {
		return "tg:" + *u.TelegramID
	}
	return ""
}
