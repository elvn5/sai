package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/payments/internal/models"
	"github.com/elvn5/sai/services/payments/internal/provider"
	"github.com/elvn5/sai/services/payments/internal/store"
	"github.com/elvn5/sai/services/pkg/audit"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
)

const transactionNotFound = "Transaction not found"

// Transactions serves the admin panel's online payments: the gateway's
// payments (GoPay) with filters, a payment's card, checking it at the
// gateway and cancelling an unpaid one. GoPay's API has no refunds: money
// goes back outside it, and the order is marked refunded (Admin.Refund).
type Transactions struct{ Deps }

type gatewayInfo struct {
	Provider string `json:"provider"`
	Online   bool   `json:"online"`
	// The gateway can cancel unpaid payments.
	CanCancel bool `json:"canCancel"`
}

// Gateway handles GET /payments/admin/gateway: which gateway takes payments.
func (h *Transactions) Gateway(c *gin.Context) {
	_, canCancel := h.Provider.(provider.Canceler)
	c.JSON(http.StatusOK, gatewayInfo{Provider: h.Provider.Name(), Online: provider.Online(h.Provider), CanCancel: canCancel})
}

// List handles GET /payments/admin/transactions?status=&from=&to=.
func (h *Transactions) List(c *gin.Context) {
	q := validation.NewQuery(c)
	status := q.Enum("status", models.TransactionStatuses)
	if !q.OK() {
		return
	}
	list, err := h.Store.Transactions(c.Request.Context(), models.TransactionFilter{
		Status: status, From: isoDate(c, "from"), To: isoDate(c, "to"), Limit: 300,
	})
	if err != nil {
		h.internalError(c, err)
		return
	}
	h.withTransactionPeople(c, list)
	respond.JSON(c, http.StatusOK, list)
}

// Get handles GET /payments/admin/transactions/:id (the order's id).
func (h *Transactions) Get(c *gin.Context) {
	if t, ok := h.load(c); ok {
		h.respond(c, t)
	}
}

// Check asks the gateway about the payment now; a pending order is settled.
func (h *Transactions) Check(c *gin.Context) {
	t, ok := h.load(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	audit.SetSubject(c, t.Order.ItemTitle)
	if _, err := h.Checker.Check(ctx, *t.Order); err != nil {
		h.Log.WarnContext(ctx, "payment check failed", slog.String("order", t.OrderID), slog.Any("error", err))
		httperr.Abort(c, http.StatusBadGateway, "The payment service is unavailable, please try again later")
		return
	}
	h.reply(c, t.OrderID)
}

// Cancel closes an unpaid payment at the gateway (its QR stops working)
// and fails the order, freeing its places.
func (h *Transactions) Cancel(c *gin.Context) {
	canceler, ok := h.Provider.(provider.Canceler)
	if !ok {
		httperr.Abort(c, http.StatusConflict, "This payment service can't cancel payments")
		return
	}
	t, ok := h.load(c)
	if !ok {
		return
	}
	if t.Order.Status != models.StatusPending {
		httperr.Abort(c, http.StatusConflict, "Only an unpaid payment can be cancelled")
		return
	}
	ctx := c.Request.Context()
	audit.SetSubject(c, t.Order.ItemTitle)
	res, err := canceler.Cancel(ctx, t.OrderID)
	if errors.Is(err, provider.ErrNotCancellable) {
		// Paid or closed meanwhile: settle it from the gateway's answer.
		_, _ = h.Checker.Check(ctx, *t.Order)
		httperr.Abort(c, http.StatusConflict, "The payment can't be cancelled any more")
		return
	}
	if err != nil {
		h.Log.WarnContext(ctx, "payment cancel failed", slog.String("order", t.OrderID), slog.Any("error", err))
		httperr.Abort(c, http.StatusBadGateway, "The payment service is unavailable, please try again later")
		return
	}
	if res.Details != nil {
		if err := h.Store.SaveTransaction(ctx, t.OrderID, h.Provider.Name(), t.ProviderOrderID, *res.Details); err != nil {
			h.internalError(c, err)
			return
		}
	}
	msg := "Cancelled by the admin"
	if _, err := h.Store.Resolve(ctx, t.OrderID, models.StatusFailed, &msg, h.now()); err != nil && !errors.Is(err, store.ErrConflict) {
		h.internalError(c, err)
		return
	}
	h.reply(c, t.OrderID)
}

func (h *Transactions) load(c *gin.Context) (models.Transaction, bool) {
	id := c.Param("id")
	if !validID(id) {
		h.fail(c, errNotFound, transactionNotFound)
		return models.Transaction{}, false
	}
	t, err := h.Store.Transaction(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err, transactionNotFound)
		return t, false
	}
	return t, true
}

func (h *Transactions) reply(c *gin.Context, id string) {
	t, err := h.Store.Transaction(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err, transactionNotFound)
		return
	}
	h.respond(c, t)
}

func (h *Transactions) respond(c *gin.Context, t models.Transaction) {
	list := []models.Transaction{t}
	h.withTransactionPeople(c, list)
	respond.JSON(c, http.StatusOK, list[0])
}

func (h *Transactions) withTransactionPeople(c *gin.Context, list []models.Transaction) {
	orders := make([]models.Order, len(list))
	for i := range list {
		orders[i] = *list[i].Order
	}
	h.withPeople(c.Request.Context(), orders)
	for i := range list {
		list[i].Order = &orders[i]
	}
}

// isoDate reads a YYYY-MM-DD query parameter; nil when unset or invalid.
func isoDate(c *gin.Context, name string) *string {
	value := c.Query(name)
	if _, err := time.Parse(time.DateOnly, value); err != nil || len(value) != len(time.DateOnly) {
		return nil
	}
	return &value
}
