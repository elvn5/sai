package handlers

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/payments/internal/listingsclient"
	"github.com/elvn5/sai/services/payments/internal/models"
	"github.com/elvn5/sai/services/payments/internal/provider"
	"github.com/elvn5/sai/services/payments/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
)

const (
	orderNotFound = "Order not found"
	maxQuantity   = 20
)

// Bishkek: tours start on Bishkek calendar days.
var bishkek = time.FixedZone("Asia/Bishkek", 6*60*60)

type createOrderRequest struct {
	Kind         *string                  `json:"kind" binding:"required,oneof=tour"`
	ItemID       *string                  `json:"itemId" binding:"required,uuid"`
	Quantity     *int                     `json:"quantity" binding:"required,gte=1,lte=20"`
	ContactName  *validation.Trimmed      `json:"contactName" binding:"required,min=1,max=120"`
	ContactPhone *string                  `json:"contactPhone" binding:"required,phone"`
	Comment      validation.Field[string] `json:"comment" binding:"omitempty,max=1000"`
}

// Orders serves bookings paid online: customers create and track their
// orders; partners see what they received.
type Orders struct{ Deps }

// Create prices the booking from the item itself (never from the client),
// creates the gateway payment and returns where to pay.
func (h *Orders) Create(c *gin.Context) {
	if !provider.Online(h.Provider) {
		httperr.Abort(c, http.StatusServiceUnavailable, "Online payment is not available")
		return
	}
	var req createOrderRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	tour, err := h.Tours.Tour(ctx, *req.ItemID)
	switch {
	case errors.Is(err, listingsclient.ErrNotFound) || (err == nil && !tour.IsPublished):
		httperr.Abort(c, http.StatusNotFound, "Tour not found")
		return
	case err != nil:
		h.internalError(c, err)
		return
	}
	today := h.now().In(bishkek).Format(time.DateOnly)
	switch {
	case tour.Price == nil || *tour.Price <= 0:
		httperr.Abort(c, http.StatusConflict, "This tour can't be booked online")
		return
	case tour.UserID == nil:
		httperr.Abort(c, http.StatusConflict, "This tour can't be booked online")
		return
	case tour.StartDate < today:
		httperr.Abort(c, http.StatusConflict, "This tour has already started")
		return
	case *tour.UserID == currentUserID(c):
		httperr.Abort(c, http.StatusBadRequest, "You can't book your own tour")
		return
	}

	amount := *tour.Price * *req.Quantity
	fee := int(math.Round(float64(amount) * h.FeePercent / 100))
	order, err := h.Store.InsertOrder(ctx, models.Order{
		Kind: "tour", ItemID: tour.ID, ItemTitle: tour.Title, ItemSlug: &tour.Slug, ItemLocale: &tour.Locale,
		ItemDate: &tour.StartDate, ItemImageURL: tour.CoverImageURL, Quantity: *req.Quantity, UnitPrice: *tour.Price, Amount: amount, Fee: fee,
		PayeeAmount: amount - fee, UserID: currentUserID(c), PayeeID: *tour.UserID,
		ContactName: string(*req.ContactName), ContactPhone: *req.ContactPhone, Comment: req.Comment.Ptr(),
		Provider: h.Provider.Name(), ExpiresAt: jsontime.New(h.now().Add(h.PaymentTTL)),
	}, tour.Capacity)
	if errors.Is(err, store.ErrNoPlaces) {
		httperr.Abort(c, http.StatusConflict, "Not enough free places on this tour")
		return
	}
	if err != nil {
		h.internalError(c, err)
		return
	}

	link, err := h.Provider.CreatePayment(ctx, provider.Payment{
		TransactionID: order.ID, Amount: amount, TTL: h.PaymentTTL,
		Comment:   fmt.Sprintf("Ermekten: %s, %d чел.", tour.Title, *req.Quantity),
		ReturnURL: h.SiteURL + "/payments/return?order=" + order.ID,
	})
	var linkPtr, errPtr *string
	if err != nil {
		h.Log.ErrorContext(ctx, "could not create a payment", "order", order.ID, "error", err)
		msg := "The payment service is unavailable"
		errPtr = &msg
	} else {
		linkPtr = &link
	}
	order, saveErr := h.Store.SetPaymentLink(ctx, order.ID, linkPtr, errPtr)
	if saveErr != nil {
		h.internalError(c, saveErr)
		return
	}
	if err != nil {
		httperr.Abort(c, http.StatusBadGateway, "The payment service is unavailable, please try again later")
		return
	}
	respond.JSON(c, http.StatusCreated, order)
}

// Most items one /orders/places request may ask about.
const maxPlacesItems = 50

// Places handles GET /orders/places?ids=a,b: how many places bookings hold
// on each item (paid, or pending payment), for "N places left". Public.
func (h *Orders) Places(c *gin.Context) {
	ids := strings.Split(c.Query("ids"), ",")
	if len(ids) > maxPlacesItems {
		httperr.Abort(c, http.StatusBadRequest, "Too many ids")
		return
	}
	valid := ids[:0]
	for _, id := range ids {
		if validID(id) {
			valid = append(valid, id)
		}
	}
	taken, err := h.Store.TakenPlaces(c.Request.Context(), valid)
	if err != nil {
		h.internalError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	respond.JSON(c, http.StatusOK, taken)
}

type paymentsConfig struct {
	Online bool `json:"online"`
}

// Config handles GET /payments/config: whether the site offers online
// booking (false with PAYMENTS_PROVIDER=off).
func (h *Orders) Config(c *gin.Context) {
	c.JSON(http.StatusOK, paymentsConfig{Online: provider.Online(h.Provider)})
}

// Mine lists the signed-in customer's orders.
func (h *Orders) Mine(c *gin.Context) {
	orders, err := h.Store.OrdersOfUser(c.Request.Context(), currentUserID(c))
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, orders)
}

// Received lists the paid bookings of the signed-in partner's tours.
func (h *Orders) Received(c *gin.Context) {
	orders, err := h.Store.OrdersForPayee(c.Request.Context(), currentUserID(c))
	if err != nil {
		h.internalError(c, err)
		return
	}
	h.withPeople(c.Request.Context(), orders)
	respond.JSON(c, http.StatusOK, orders)
}

// Get shows an order to its customer, its partner (once paid) or the admin.
func (h *Orders) Get(c *gin.Context) {
	o, ok := h.visible(c)
	if ok {
		respond.JSON(c, http.StatusOK, o)
	}
}

// Check asks the gateway about a pending order right away (the customer
// is back from the payment page) instead of waiting for the worker.
func (h *Orders) Check(c *gin.Context) {
	o, ok := h.visible(c)
	if !ok {
		return
	}
	if o.UserID != currentUserID(c) {
		httperr.Abort(c, http.StatusForbidden, "Only the customer can check this order")
		return
	}
	if o.Status == models.StatusPending {
		checked, err := h.Checker.Check(c.Request.Context(), o)
		if err != nil {
			h.Log.WarnContext(c.Request.Context(), "payment check failed", "order", o.ID, "error", err)
		} else {
			o = checked
		}
	}
	respond.JSON(c, http.StatusOK, o)
}

func (h *Orders) visible(c *gin.Context) (models.Order, bool) {
	id := c.Param("id")
	if !validID(id) {
		h.fail(c, errNotFound, orderNotFound)
		return models.Order{}, false
	}
	o, err := h.Store.Order(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err, orderNotFound)
		return o, false
	}
	me := currentUserID(c)
	partnerSees := o.PayeeID == me && (o.Status == models.StatusPaid || o.Status == models.StatusRefunded)
	if !isAdmin(c) && o.UserID != me && !partnerSees {
		h.fail(c, errNotFound, orderNotFound)
		return o, false
	}
	return o, true
}
