package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/payments/internal/provider"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
)

type testPaymentRequest struct {
	Success *bool `json:"success" binding:"required"`
}

// TestPayments backs the site's test payment page when the fake provider
// is on (PAYMENTS_PROVIDER=fake); the routes aren't registered otherwise.
type TestPayments struct {
	Deps
	Fake *provider.Fake
}

// Complete settles a test payment as paid or declined, as the bank would.
func (h *TestPayments) Complete(c *gin.Context) {
	var req testPaymentRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	o, ok := (&Orders{Deps: h.Deps}).visible(c)
	if !ok {
		return
	}
	if o.UserID != currentUserID(c) {
		httperr.Abort(c, http.StatusForbidden, "Only the customer can pay for this order")
		return
	}
	if !h.Fake.Complete(o.ID, *req.Success) {
		httperr.Abort(c, http.StatusConflict, "Unknown test payment")
		return
	}
	respond.JSON(c, http.StatusOK, gin.H{"returnUrl": h.SiteURL + "/payments/return?order=" + o.ID})
}
