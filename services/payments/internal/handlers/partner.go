package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/respond"
)

// Partner serves the partner cabinet's money pages. Every route runs
// behind RequirePartner and shows only the signed-in partner's own data.
type Partner struct{ Deps }

// Finances sums what the partner earned and is owed, with their payouts.
func (h *Partner) Finances(c *gin.Context) {
	ctx := c.Request.Context()
	me := currentUserID(c)
	now := h.now().In(bishkek)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, bishkek)
	f, err := h.Store.Finances(ctx, me, monthStart)
	if err != nil {
		h.internalError(c, err)
		return
	}
	if f.Payouts, err = h.Store.PayoutsOf(ctx, me, 100); err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, f)
}
