package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
)

const (
	vehicleNotFound = "Vehicle not found"
	tripNotFound    = "Trip not found"
	driverNotFound  = "Driver not found"
	expenseNotFound = "Expense not found"
)

type driverProfileRequest struct {
	Name      *validation.Trimmed      `json:"name" binding:"required,min=1,max=120"`
	Phone     validation.Field[string] `json:"phone" binding:"omitempty,phone"`
	Region    validation.Field[string] `json:"region" binding:"omitempty,max=120"`
	About     validation.Field[string] `json:"about" binding:"omitempty,max=2000"`
	Published *bool                    `json:"published"`
}

type vehicleRequest struct {
	Kind            *string                  `json:"kind" binding:"required,oneof=sedan minivan minibus bus offroad other"`
	Model           *validation.Trimmed      `json:"model" binding:"required,min=1,max=120"`
	Plate           validation.Field[string] `json:"plate" binding:"omitempty,max=20"`
	Seats           *int                     `json:"seats" binding:"required,gte=1,lte=80"`
	Year            validation.Field[int]    `json:"year" binding:"omitempty,gte=1950,lte=2100"`
	InsuranceUntil  validation.Field[string] `json:"insuranceUntil" binding:"omitempty,isodate"`
	InspectionUntil validation.Field[string] `json:"inspectionUntil" binding:"omitempty,isodate"`
	Note            validation.Field[string] `json:"note" binding:"omitempty,max=500"`
	Active          *bool                    `json:"active"`
}

func (r vehicleRequest) vehicle(id string) models.DriverVehicle {
	v := models.DriverVehicle{
		ID: id, Kind: *r.Kind, Model: string(*r.Model), Plate: optText(r.Plate), Seats: *r.Seats, Year: r.Year.Ptr(),
		InsuranceUntil: optText(r.InsuranceUntil), InspectionUntil: optText(r.InspectionUntil), Note: optText(r.Note), Active: true,
	}
	if r.Active != nil {
		v.Active = *r.Active
	}
	return v
}

// tripRequest is a trip the driver writes down or changes, and what an
// organizer asks for (then only the route, time, size, price and note
// count).
type tripRequest struct {
	// Only when an organizer asks: the driver.
	DriverID      validation.Field[string]    `json:"driverId" binding:"omitempty,uuid"`
	VehicleID     validation.Field[string]    `json:"vehicleId" binding:"omitempty,uuid"`
	ClientID      validation.Field[string]    `json:"clientId" binding:"omitempty,uuid"`
	Status        *string                     `json:"status" binding:"omitempty,oneof=requested accepted declined done cancelled"`
	StartsAt      *time.Time                  `json:"startsAt" binding:"required"`
	EndsAt        validation.Field[time.Time] `json:"endsAt"`
	RouteFrom     *validation.Trimmed         `json:"routeFrom" binding:"required,min=1,max=200"`
	RouteTo       *validation.Trimmed         `json:"routeTo" binding:"required,min=1,max=200"`
	Passengers    validation.Field[int]       `json:"passengers" binding:"omitempty,gte=1,lte=80"`
	Price         validation.Field[int]       `json:"price" binding:"omitempty,gte=0,lte=100000000"`
	Note          validation.Field[string]    `json:"note" binding:"omitempty,max=2000"`
	DeclineReason validation.Field[string]    `json:"declineReason" binding:"omitempty,max=300"`
}

func (r tripRequest) Refine() map[string]string {
	if r.EndsAt.Valid && r.StartsAt != nil && r.EndsAt.Value.Before(*r.StartsAt) {
		return map[string]string{"endsAt": "Must be after the start"}
	}
	return nil
}

func (r tripRequest) input() store.TripInput {
	return store.TripInput{
		VehicleID: optText(r.VehicleID), ClientID: optText(r.ClientID), StartsAt: *r.StartsAt, EndsAt: r.EndsAt.Ptr(),
		RouteFrom: string(*r.RouteFrom), RouteTo: string(*r.RouteTo), Passengers: r.Passengers.Ptr(), Price: r.Price.Ptr(),
		Note: optText(r.Note), Decline: optText(r.DeclineReason),
	}
}

// tripStatusRequest answers a request or moves a trip along.
type tripStatusRequest struct {
	Status        *string                  `json:"status" binding:"required,oneof=accepted declined done cancelled"`
	VehicleID     validation.Field[string] `json:"vehicleId" binding:"omitempty,uuid"`
	Price         validation.Field[int]    `json:"price" binding:"omitempty,gte=0,lte=100000000"`
	DeclineReason validation.Field[string] `json:"declineReason" binding:"omitempty,max=300"`
}

type driverPaymentRequest struct {
	Amount *int                     `json:"amount" binding:"required,gt=0,lte=100000000"`
	Method *string                  `json:"method" binding:"required,oneof=cash transfer card other"`
	PaidOn *string                  `json:"paidOn" binding:"required,isodate"`
	Note   validation.Field[string] `json:"note" binding:"omitempty,max=500"`
}

type driverExpenseRequest struct {
	Category  *string                  `json:"category" binding:"required,oneof=fuel repair maintenance wash parking fines insurance other"`
	Amount    *int                     `json:"amount" binding:"required,gt=0,lte=100000000"`
	SpentOn   *string                  `json:"spentOn" binding:"required,isodate"`
	VehicleID validation.Field[string] `json:"vehicleId" binding:"omitempty,uuid"`
	TripID    validation.Field[string] `json:"tripId" binding:"omitempty,uuid"`
	Note      validation.Field[string] `json:"note" binding:"omitempty,max=500"`
}

// moves are the statuses a trip may go to from each status.
var moves = map[string][]string{
	"requested": {"accepted", "declined"},
	"accepted":  {"done", "cancelled"},
	"done":      {"accepted"},
	"cancelled": {"accepted"},
}

// Drivers serves the driver's CRM (/driver/*: drivers only) and, for tour
// organizers, the catalog of drivers and the transport asked for a tour.
type Drivers struct{ Deps }

// --- The driver's card ---

func (h *Drivers) Profile(c *gin.Context) {
	p, err := h.Store.DriverProfile(c.Request.Context(), currentUserID(c))
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		h.fail(c, err, driverNotFound, "")
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		p = models.DriverProfile{UserID: currentUserID(c)}
	}
	respond.JSON(c, http.StatusOK, p)
}

func (h *Drivers) SaveProfile(c *gin.Context) {
	var req driverProfileRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	p := models.DriverProfile{
		UserID: currentUserID(c), Name: string(*req.Name), Phone: optText(req.Phone), Region: optText(req.Region),
		About: optText(req.About), Published: req.Published != nil && *req.Published,
	}
	if err := h.Store.SaveDriverProfile(c.Request.Context(), p); err != nil {
		h.fail(c, err, driverNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, p)
}

// --- Vehicles ---

func (h *Drivers) Vehicles(c *gin.Context) {
	out, err := h.Store.Vehicles(c.Request.Context(), currentUserID(c), false)
	if err != nil {
		h.fail(c, err, vehicleNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, out)
}

func (h *Drivers) AddVehicle(c *gin.Context) {
	var req vehicleRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	id, err := h.Store.AddVehicle(ctx, currentUserID(c), req.vehicle(""))
	if err != nil {
		h.fail(c, err, vehicleNotFound, "")
		return
	}
	v, err := h.Store.Vehicle(ctx, currentUserID(c), id)
	if err != nil {
		h.fail(c, err, vehicleNotFound, "")
		return
	}
	respond.JSON(c, http.StatusCreated, v)
}

func (h *Drivers) UpdateVehicle(c *gin.Context) {
	var req vehicleRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, vehicleNotFound)
		return
	}
	ctx := c.Request.Context()
	if err := h.Store.UpdateVehicle(ctx, currentUserID(c), req.vehicle(id)); err != nil {
		h.fail(c, err, vehicleNotFound, "")
		return
	}
	v, err := h.Store.Vehicle(ctx, currentUserID(c), id)
	if err != nil {
		h.fail(c, err, vehicleNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, v)
}

func (h *Drivers) DeleteVehicle(c *gin.Context) {
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, vehicleNotFound)
		return
	}
	if err := h.Store.DeleteVehicle(c.Request.Context(), currentUserID(c), id); err != nil {
		h.fail(c, err, vehicleNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, gin.H{"success": true})
}

// --- Trips ---

// Trips lists the driver's trips: ?status=, ?clientId=, ?from=&to=
// (YYYY-MM-DD).
func (h *Drivers) Trips(c *gin.Context) {
	f := store.TripFilter{Status: c.Query("status"), ClientID: c.Query("clientId"), From: c.Query("from"), To: c.Query("to")}
	for _, d := range []string{f.From, f.To} {
		if d != "" {
			if _, err := time.Parse(time.DateOnly, d); err != nil {
				httperr.Abort(c, http.StatusBadRequest, "Invalid date")
				return
			}
		}
	}
	if f.ClientID != "" && !validID(f.ClientID) {
		httperr.Abort(c, http.StatusBadRequest, "Invalid clientId")
		return
	}
	ctx := c.Request.Context()
	trips, err := h.Store.Trips(ctx, currentUserID(c), f)
	if err != nil {
		h.fail(c, err, tripNotFound, "")
		return
	}
	h.withAccounts(c, trips)
	respond.JSON(c, http.StatusOK, trips)
}

// withAccounts fills in the profiles of clients with an account.
func (h *Drivers) withAccounts(c *gin.Context, trips []models.DriverTrip) {
	var ids []string
	for _, t := range trips {
		if t.Client != nil && t.Client.UserID != nil {
			ids = append(ids, *t.Client.UserID)
		}
	}
	people := (&CRM{Deps: h.Deps}).accounts(c.Request.Context(), ids)
	for i := range trips {
		if cl := trips[i].Client; cl != nil {
			cl.Account = accountOf(people, cl.UserID)
		}
	}
}

func (h *Drivers) Trip(c *gin.Context) {
	h.respondTrip(c, http.StatusOK, c.Param("id"))
}

func (h *Drivers) respondTrip(c *gin.Context, status int, id string) {
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, tripNotFound)
		return
	}
	ctx := c.Request.Context()
	me := currentUserID(c)
	trip, err := h.Store.Trip(ctx, me, id)
	if err != nil {
		h.fail(c, err, tripNotFound, "")
		return
	}
	if trip.Payments, err = h.Store.TripPayments(ctx, id); err != nil {
		h.fail(c, err, tripNotFound, "")
		return
	}
	if trip.ExpenseList, err = h.Store.DriverExpenses(ctx, me, store.ExpenseFilter{TripID: id}); err != nil {
		h.fail(c, err, tripNotFound, "")
		return
	}
	if trip.Client != nil && trip.Client.UserID != nil {
		if trip.ChatID, err = h.Store.ChatID(ctx, me, *trip.Client.UserID); err != nil {
			h.fail(c, err, tripNotFound, "")
			return
		}
	}
	trips := []models.DriverTrip{trip}
	h.withAccounts(c, trips)
	respond.JSON(c, status, trips[0])
}

// AddTrip writes down a trip the driver took on outside Ermekten.
func (h *Drivers) AddTrip(c *gin.Context) {
	var req tripRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	in := req.input()
	in.Status = "accepted"
	if req.Status != nil && *req.Status != "requested" && *req.Status != "declined" {
		in.Status = *req.Status
	}
	id, err := h.Store.AddTrip(c.Request.Context(), currentUserID(c), in)
	if err != nil {
		h.fail(c, err, vehicleNotFound, "")
		return
	}
	h.respondTrip(c, http.StatusCreated, id)
}

// UpdateTrip changes a trip's details (not its status: SetStatus does).
func (h *Drivers) UpdateTrip(c *gin.Context) {
	var req tripRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	id := c.Param("id")
	trip, ok := h.ownTrip(c, id)
	if !ok {
		return
	}
	in := req.input()
	in.Status = trip.Status
	if trip.Tour != nil && trip.Client != nil {
		// An organizer's request stays theirs.
		in.ClientID = &trip.Client.ID
	}
	if in.Decline == nil {
		in.Decline = trip.DeclineReason
	}
	if err := h.Store.UpdateTrip(c.Request.Context(), currentUserID(c), id, in); err != nil {
		h.fail(c, err, vehicleNotFound, "")
		return
	}
	h.respondTrip(c, http.StatusOK, id)
}

func (h *Drivers) ownTrip(c *gin.Context, id string) (models.DriverTrip, bool) {
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, tripNotFound)
		return models.DriverTrip{}, false
	}
	trip, err := h.Store.Trip(c.Request.Context(), currentUserID(c), id)
	if err != nil {
		h.fail(c, err, tripNotFound, "")
		return trip, false
	}
	return trip, true
}

// SetStatus answers an organizer's request (accepted, declined) or moves
// a trip along (done, cancelled, back to accepted). The organizer hears
// about it.
func (h *Drivers) SetStatus(c *gin.Context) {
	var req tripStatusRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	id := c.Param("id")
	trip, ok := h.ownTrip(c, id)
	if !ok {
		return
	}
	next := *req.Status
	allowed := false
	for _, s := range moves[trip.Status] {
		allowed = allowed || s == next
	}
	if !allowed {
		httperr.Abort(c, http.StatusConflict, "A "+trip.Status+" trip can't become "+next)
		return
	}
	in := store.TripInput{
		Status: next, StartsAt: trip.StartsAt.Time, RouteFrom: trip.RouteFrom, RouteTo: trip.RouteTo,
		Passengers: trip.Passengers, Price: trip.Price, Note: trip.Note, Decline: trip.DeclineReason,
	}
	if trip.EndsAt != nil {
		in.EndsAt = &trip.EndsAt.Time
	}
	if trip.Vehicle != nil {
		in.VehicleID = &trip.Vehicle.ID
	}
	if trip.Client != nil {
		in.ClientID = &trip.Client.ID
	}
	if req.VehicleID.Set {
		in.VehicleID = optText(req.VehicleID)
	}
	if req.Price.Set {
		in.Price = req.Price.Ptr()
	}
	if next == "declined" || next == "cancelled" {
		in.Decline = optText(req.DeclineReason)
	}
	ctx := c.Request.Context()
	if err := h.Store.UpdateTrip(ctx, currentUserID(c), id, in); err != nil {
		h.fail(c, err, vehicleNotFound, "")
		return
	}
	if trip.Tour != nil && h.Notify != nil && (next == "accepted" || next == "declined" || next == "cancelled") {
		if tour, err := h.Store.Tour(ctx, trip.Tour.ID); err == nil {
			name := ""
			if p, err := h.Store.DriverProfile(ctx, currentUserID(c)); err == nil {
				name = p.Name
			}
			h.Notify.TransportAnswered(ctx, tour, name, next)
		}
	}
	h.respondTrip(c, http.StatusOK, id)
}

func (h *Drivers) DeleteTrip(c *gin.Context) {
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, tripNotFound)
		return
	}
	if err := h.Store.DeleteTrip(c.Request.Context(), currentUserID(c), id); err != nil {
		h.fail(c, err, tripNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, gin.H{"success": true})
}

// --- Payments and expenses ---

func (h *Drivers) AddPayment(c *gin.Context) {
	var req driverPaymentRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, tripNotFound)
		return
	}
	p := models.DriverPayment{Amount: *req.Amount, Method: *req.Method, PaidOn: *req.PaidOn, Note: optText(req.Note)}
	if err := h.Store.AddTripPayment(c.Request.Context(), currentUserID(c), id, p); err != nil {
		h.fail(c, err, tripNotFound, "")
		return
	}
	h.respondTrip(c, http.StatusCreated, id)
}

func (h *Drivers) DeletePayment(c *gin.Context) {
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, paymentNotFound)
		return
	}
	tripID, err := h.Store.DeleteTripPayment(c.Request.Context(), currentUserID(c), id)
	if err != nil {
		h.fail(c, err, paymentNotFound, "")
		return
	}
	h.respondTrip(c, http.StatusOK, tripID)
}

// Expenses lists the driver's expenses: ?from=&to= (YYYY-MM-DD).
func (h *Drivers) Expenses(c *gin.Context) {
	f := store.ExpenseFilter{From: c.Query("from"), To: c.Query("to")}
	for _, d := range []string{f.From, f.To} {
		if d != "" {
			if _, err := time.Parse(time.DateOnly, d); err != nil {
				httperr.Abort(c, http.StatusBadRequest, "Invalid date")
				return
			}
		}
	}
	out, err := h.Store.DriverExpenses(c.Request.Context(), currentUserID(c), f)
	if err != nil {
		h.fail(c, err, expenseNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, out)
}

func (h *Drivers) AddExpense(c *gin.Context) {
	var req driverExpenseRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	e := models.DriverExpense{
		Category: *req.Category, Amount: *req.Amount, SpentOn: *req.SpentOn, Note: optText(req.Note),
		VehicleID: optText(req.VehicleID), TripID: optText(req.TripID),
	}
	if err := h.Store.AddDriverExpense(c.Request.Context(), currentUserID(c), e); err != nil {
		h.fail(c, err, vehicleNotFound, "")
		return
	}
	respond.JSON(c, http.StatusCreated, gin.H{"success": true})
}

func (h *Drivers) DeleteExpense(c *gin.Context) {
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, expenseNotFound)
		return
	}
	if err := h.Store.DeleteDriverExpense(c.Request.Context(), currentUserID(c), id); err != nil {
		h.fail(c, err, expenseNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, gin.H{"success": true})
}

// --- Dashboard and report ---

func (h *Drivers) Summary(c *gin.Context) {
	ctx := c.Request.Context()
	out, err := h.Store.DriverSummary(ctx, currentUserID(c), h.today())
	if err != nil {
		h.fail(c, err, driverNotFound, "")
		return
	}
	h.withAccounts(c, out.Upcoming)
	respond.JSON(c, http.StatusOK, out)
}

func (h *Drivers) Report(c *gin.Context) {
	today := h.today()
	from := c.DefaultQuery("from", today.AddDate(0, -5, -today.Day()+1).Format(time.DateOnly))
	to := c.DefaultQuery("to", today.Format(time.DateOnly))
	start, err1 := time.Parse(time.DateOnly, from)
	end, err2 := time.Parse(time.DateOnly, to)
	if err1 != nil || err2 != nil || end.Before(start) || end.Sub(start) > maxReportDays*24*time.Hour {
		httperr.Abort(c, http.StatusBadRequest, "Invalid period")
		return
	}
	report, err := h.Store.DriverReport(c.Request.Context(), currentUserID(c), from, to)
	if err != nil {
		h.fail(c, err, driverNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, report)
}

// --- The organizers' side ---

// Catalog lists the drivers an organizer can ask: ?date=YYYY-MM-DD marks
// those busy that day, ?seats= hides vehicles too small for the group.
func (h *Drivers) Catalog(c *gin.Context) {
	day := c.Query("date")
	if day != "" {
		if _, err := time.Parse(time.DateOnly, day); err != nil {
			httperr.Abort(c, http.StatusBadRequest, "Invalid date")
			return
		}
	}
	seats, _ := strconv.Atoi(c.Query("seats"))
	ctx := c.Request.Context()
	out, err := h.Store.Drivers(ctx, day, max(seats, 1))
	if err != nil {
		h.fail(c, err, driverNotFound, "")
		return
	}
	ids := make([]string, len(out))
	for i, d := range out {
		ids[i] = d.UserID
	}
	people := (&CRM{Deps: h.Deps}).accounts(ctx, ids)
	for i := range out {
		out[i].Account = accountOf(people, &out[i].UserID)
	}
	respond.JSON(c, http.StatusOK, out)
}

// Transport lists the trips asked for one of the organizer's tours.
func (h *Drivers) Transport(c *gin.Context) {
	tour, ok := (&CRM{Deps: h.Deps}).ownTour(c, c.Param("id"))
	if !ok {
		return
	}
	h.respondTransport(c, http.StatusOK, tour.ID)
}

func (h *Drivers) respondTransport(c *gin.Context, status int, tourID string) {
	out, err := h.Store.TransportRequests(c.Request.Context(), tourID)
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	respond.JSON(c, status, out)
}

// RequestTransport asks a driver for a trip for one of the organizer's
// tours; the driver hears about it in Telegram.
func (h *Drivers) RequestTransport(c *gin.Context) {
	var req tripRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	driverID := optText(req.DriverID)
	if driverID == nil {
		httperr.Abort(c, http.StatusNotFound, driverNotFound)
		return
	}
	tour, ok := (&CRM{Deps: h.Deps}).ownTour(c, c.Param("id"))
	if !ok {
		return
	}
	if *driverID == currentUserID(c) {
		httperr.Abort(c, http.StatusNotFound, driverNotFound)
		return
	}
	in := req.input()
	ctx := c.Request.Context()
	id, err := h.Store.RequestTransport(ctx, currentUserID(c), tour.ID, *driverID, in)
	if err != nil {
		h.fail(c, err, driverNotFound, "")
		return
	}
	if h.Notify != nil {
		h.Notify.TransportRequested(ctx, tour, id, *driverID, in.StartsAt)
	}
	h.respondTransport(c, http.StatusCreated, tour.ID)
}

// CancelTransport calls off a trip asked for one of the organizer's tours.
func (h *Drivers) CancelTransport(c *gin.Context) {
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, tripNotFound)
		return
	}
	ctx := c.Request.Context()
	tourID, driverID, err := h.Store.CancelTransport(ctx, currentUserID(c), id)
	if err != nil {
		h.fail(c, err, tripNotFound, "")
		return
	}
	if h.Notify != nil {
		if tour, err := h.Store.Tour(ctx, tourID); err == nil {
			h.Notify.TransportCancelled(ctx, tour, id, driverID)
		}
	}
	h.respondTransport(c, http.StatusOK, tourID)
}
