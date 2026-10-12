package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/payments/internal/models"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
)

type refundRequest struct {
	Note validation.Field[string] `json:"note" binding:"omitempty,max=500"`
}

type payoutRequest struct {
	PayeeID *string                  `json:"payeeId" binding:"required,uuid"`
	Note    validation.Field[string] `json:"note" binding:"omitempty,max=500"`
}

// Admin serves the admin panel: all orders, what's owed to each partner,
// and recording payouts and refunds. Every route runs behind RequireAdmin.
type Admin struct{ Deps }

func (h *Admin) Orders(c *gin.Context) {
	q := validation.NewQuery(c)
	status := q.Enum("status", models.Statuses)
	if !q.OK() {
		return
	}
	orders, err := h.Store.AllOrders(c.Request.Context(), status, 300)
	if err != nil {
		h.internalError(c, err)
		return
	}
	h.withPeople(c.Request.Context(), orders)
	respond.JSON(c, http.StatusOK, orders)
}

// Refund records that a paid order's money was returned to the customer
// (outside the system). Not possible once the partner was paid out.
func (h *Admin) Refund(c *gin.Context) {
	var req refundRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	if !validID(c.Param("id")) {
		h.fail(c, errNotFound, orderNotFound)
		return
	}
	o, err := h.Store.Refund(c.Request.Context(), c.Param("id"), req.Note.Ptr())
	if err != nil {
		h.fail(c, err, orderNotFound)
		return
	}
	respond.JSON(c, http.StatusOK, o)
}

// Balances lists what's owed to each partner.
func (h *Admin) Balances(c *gin.Context) {
	ctx := c.Request.Context()
	balances, err := h.Store.Balances(ctx)
	if err != nil {
		h.internalError(c, err)
		return
	}
	ids := make([]string, len(balances))
	for i, b := range balances {
		ids[i] = b.Payee.ID
	}
	authors, _ := h.Authors.Authors(ctx, ids)
	for i := range balances {
		a := authors[balances[i].Payee.ID]
		balances[i].Payee.FirstName, balances[i].Payee.Username = a.FirstName, a.Username
	}
	respond.JSON(c, http.StatusOK, balances)
}

// CreatePayout records that a partner was paid everything they're owed.
func (h *Admin) CreatePayout(c *gin.Context) {
	var req payoutRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	p, err := h.Store.CreatePayout(c.Request.Context(), *req.PayeeID, req.Note.Ptr())
	if err != nil {
		if errors.Is(err, errConflict) {
			httperr.Abort(c, http.StatusConflict, "Nothing is owed to this partner")
			return
		}
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusCreated, p)
}

func (h *Admin) Payouts(c *gin.Context) {
	ctx := c.Request.Context()
	payouts, err := h.Store.Payouts(ctx, 200)
	if err != nil {
		h.internalError(c, err)
		return
	}
	ids := make([]string, len(payouts))
	for i, p := range payouts {
		ids[i] = p.Payee.ID
	}
	authors, _ := h.Authors.Authors(ctx, ids)
	for i := range payouts {
		a := authors[payouts[i].Payee.ID]
		payouts[i].Payee.FirstName, payouts[i].Payee.Username = a.FirstName, a.Username
	}
	respond.JSON(c, http.StatusOK, payouts)
}
