package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/elvn5/sai/services/payments/internal/models"
	"github.com/elvn5/sai/services/payments/internal/provider"
	"github.com/elvn5/sai/services/payments/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
)

// Most a webhook body may weigh.
const maxWebhookBody = 1 << 20

// GoPayWebhook receives GoPay's events_url deliveries. A delivery is only
// a nudge: once its signature checks out, the order is settled from GoPay's
// own answer to a status query (as the worker does), never from the body.
// A repeated delivery of a processed event is acknowledged and skipped.
type GoPayWebhook struct {
	Deps
	GoPay *provider.GoPay
}

func (h *GoPayWebhook) Receive(c *gin.Context) {
	ctx := c.Request.Context()
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxWebhookBody+1))
	if err != nil || len(body) > maxWebhookBody {
		httperr.Abort(c, http.StatusBadRequest, "Invalid body")
		return
	}
	if !h.GoPay.VerifyWebhook(c.GetHeader("GoPay-Nonce"), c.GetHeader("GoPay-Signature"), body) {
		h.Log.WarnContext(ctx, "gopay webhook with a bad signature", slog.String("ip", c.ClientIP()))
		httperr.Abort(c, http.StatusUnauthorized, "Invalid signature")
		return
	}
	event, err := provider.ParseGoPayEvent(body)
	if err != nil {
		httperr.Abort(c, http.StatusBadRequest, "Invalid event")
		return
	}
	if event.PaymentID == "" { // test.ping and events about other things
		c.Status(http.StatusNoContent)
		return
	}

	var orderID *string
	if id, err := uuid.Parse(event.OrderID); err == nil {
		s := id.String()
		orderID = &s
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, body); err != nil {
		httperr.Abort(c, http.StatusBadRequest, "Invalid event")
		return
	}
	eventID, done, err := h.Store.RecordEvent(ctx, store.Event{
		Provider: h.GoPay.Name(), Key: event.Event + ":" + event.PaymentID, Event: event.Event,
		PaymentID: event.PaymentID, OrderID: orderID, Testing: event.Testing, Payload: compact.Bytes(),
	})
	if err != nil {
		h.internalError(c, err)
		return
	}
	if done {
		c.Status(http.StatusNoContent)
		return
	}
	finish := func(processed bool, problem string) {
		var p *string
		if problem != "" {
			p = &problem
		}
		if err := h.Store.FinishEvent(ctx, eventID, processed, p); err != nil {
			h.Log.ErrorContext(ctx, "could not mark the webhook", slog.Any("error", err))
		}
	}

	if orderID == nil {
		finish(true, "Unknown order")
		c.Status(http.StatusNoContent)
		return
	}
	o, err := h.Store.Order(ctx, *orderID)
	if errors.Is(err, store.ErrNotFound) {
		finish(true, "Unknown order")
		c.Status(http.StatusNoContent)
		return
	}
	if err != nil {
		finish(false, err.Error())
		h.internalError(c, err)
		return
	}
	checked, err := h.Checker.Check(ctx, o)
	if err != nil {
		// Not acknowledged: GoPay delivers it again later.
		h.Log.WarnContext(ctx, "gopay webhook: status check failed", slog.String("order", o.ID), slog.Any("error", err))
		finish(false, err.Error())
		httperr.Abort(c, http.StatusBadGateway, "The payment could not be checked")
		return
	}
	problem := ""
	if event.Status == "COMMITTED" && checked.Status != models.StatusPaid && checked.Status != models.StatusRefunded {
		problem = "The order is " + checked.Status
	}
	finish(true, problem)
	c.Status(http.StatusNoContent)
}
