package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/usersclient"
	"github.com/elvn5/sai/services/pkg/validation"
)

const (
	clientNotFound      = "Client not found"
	participantNotFound = "Participant not found"
	paymentNotFound     = "Payment not found"
)

type clientRequest struct {
	Name     validation.Field[validation.Trimmed] `json:"name" binding:"omitempty,min=1,max=120"`
	Phone    validation.Field[string]             `json:"phone" binding:"omitempty,phone"`
	Telegram validation.Field[string]             `json:"telegram" binding:"omitempty,telegram"`
	Note     validation.Field[string]             `json:"note" binding:"omitempty,max=4000"`
	Tags     []string                             `json:"tags" binding:"omitempty,max=20,dive,min=1,max=40"`
	Source   validation.Field[string]             `json:"source" binding:"omitempty,oneof=ermekten instagram whatsapp telegram phone referral other"`
	creating bool
}

// Refine: a client added by hand needs a name.
func (r clientRequest) Refine() map[string]string {
	if r.creating && !r.Name.Valid {
		return map[string]string{"name": "Required"}
	}
	return nil
}

func (r clientRequest) apply(in *store.ClientInput) {
	applyTrimmed(&in.Name, r.Name)
	applyField(&in.Phone, r.Phone)
	applyField(&in.Note, r.Note)
	applyField(&in.Telegram, r.Telegram)
	applyField(&in.Source, r.Source)
	if in.Telegram != nil {
		stripped := strings.TrimPrefix(*in.Telegram, "@")
		in.Telegram = &stripped
	}
	if r.Tags != nil {
		in.Tags = normalizeTags(r.Tags)
	}
}

// normalizeTags trims tags and drops empty and repeated ones.
func normalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t != "" && !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out
}

type addParticipantRequest struct {
	ClientID *string `json:"clientId" binding:"required,uuid"`
	Status   *string `json:"status" binding:"omitempty,oneof=new confirmed waitlist"`
}

type participantRequest struct {
	Status    *string               `json:"status" binding:"omitempty,oneof=new confirmed waitlist cancelled"`
	AmountDue validation.Field[int] `json:"amountDue" binding:"omitempty,gte=0,lte=100000000"`
	Arrived   *bool                 `json:"arrived"`
}

type paymentRequest struct {
	Kind   *string                  `json:"kind" binding:"omitempty,oneof=payment refund"`
	Amount *int                     `json:"amount" binding:"required,gt=0,lte=100000000"`
	Method *string                  `json:"method" binding:"required,oneof=cash transfer card other"`
	PaidOn *string                  `json:"paidOn" binding:"required,isodate"`
	Note   validation.Field[string] `json:"note" binding:"omitempty,max=500"`
}

// CRM is the partner's client base: their clients (accounts that signed
// up for their tours and people they added by hand), each tour's group
// with statuses and the waitlist, and payments received.
type CRM struct{ Deps }

// accounts looks up the profiles of clients with an account.
func (h *CRM) accounts(ctx context.Context, ids []string) map[string]usersclient.Author {
	if len(ids) == 0 {
		return nil
	}
	people, err := h.Authors.Authors(ctx, ids)
	if err != nil {
		h.Log.WarnContext(ctx, "clients' profiles", slog.Any("error", err))
	}
	return people
}

func accountOf(people map[string]usersclient.Author, id *string) *usersclient.Author {
	if id == nil {
		return nil
	}
	a := authorFor(people, *id)
	return &a
}

// displayName is what a client is found by: the name the partner typed,
// else the account's.
func displayName(c models.CRMClient) string {
	parts := []string{}
	for _, s := range []*string{c.Name, c.Phone, c.Telegram, c.Note} {
		if s != nil {
			parts = append(parts, *s)
		}
	}
	if c.Account != nil {
		for _, s := range []*string{c.Account.FirstName, c.Account.Username} {
			if s != nil {
				parts = append(parts, *s)
			}
		}
	}
	return strings.ToLower(strings.Join(parts, " "))
}

// Clients lists the partner's clients, newest first, found by name,
// phone, Telegram or note (q) and by tag.
func (h *CRM) Clients(c *gin.Context) {
	q := validation.NewQuery(c)
	search := q.String("q", 100)
	tag := q.String("tag", 40)
	limit := q.IntDefault("limit", 50, validation.Bound(1), validation.Bound(200))
	offset := q.IntDefault("offset", 0, validation.Bound(0), nil)
	if !q.OK() {
		return
	}
	ctx := c.Request.Context()
	clients, err := h.Store.CRMClients(ctx, currentUserID(c))
	if err != nil {
		h.fail(c, err, clientNotFound, "")
		return
	}
	var ids []string
	for _, cl := range clients {
		if cl.UserID != nil {
			ids = append(ids, *cl.UserID)
		}
	}
	people := h.accounts(ctx, ids)
	found := make([]models.CRMClient, 0, len(clients))
	for _, cl := range clients {
		cl.Account = accountOf(people, cl.UserID)
		if search != nil && !strings.Contains(displayName(cl), strings.ToLower(strings.TrimSpace(*search))) {
			continue
		}
		if tag != nil && !slices.Contains(cl.Tags, *tag) {
			continue
		}
		found = append(found, cl)
	}
	page := models.Page[models.CRMClient]{Items: []models.CRMClient{}, Total: len(found)}
	if offset < len(found) {
		page.Items = found[offset:min(offset+limit, len(found))]
	}
	respond.JSON(c, http.StatusOK, page)
}

// Tags lists the tags the partner uses, for the filter.
func (h *CRM) Tags(c *gin.Context) {
	clients, err := h.Store.CRMClients(c.Request.Context(), currentUserID(c))
	if err != nil {
		h.fail(c, err, clientNotFound, "")
		return
	}
	tags := []string{}
	for _, cl := range clients {
		for _, t := range cl.Tags {
			if !slices.Contains(tags, t) {
				tags = append(tags, t)
			}
		}
	}
	slices.Sort(tags)
	respond.JSON(c, http.StatusOK, tags)
}

// Client is one client's card with their tours and the chat with them.
func (h *CRM) Client(c *gin.Context) {
	h.respondClient(c, http.StatusOK, c.Param("id"))
}

func (h *CRM) respondClient(c *gin.Context, status int, id string) {
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, clientNotFound)
		return
	}
	ctx := c.Request.Context()
	me := currentUserID(c)
	cl, err := h.Store.CRMClient(ctx, me, id)
	if err != nil {
		h.fail(c, err, clientNotFound, "")
		return
	}
	if cl.UserID != nil {
		cl.Account = accountOf(h.accounts(ctx, []string{*cl.UserID}), cl.UserID)
		chat, err := h.Store.ChatID(ctx, me, *cl.UserID)
		if err != nil {
			h.fail(c, err, clientNotFound, "")
			return
		}
		cl.Chat = chat
	}
	if cl.Leads, err = h.Store.Leads(ctx, me, cl.ID); err != nil {
		h.fail(c, err, clientNotFound, "")
		return
	}
	if cl.Tasks, err = h.Store.Tasks(ctx, me, cl.ID, false); err != nil {
		h.fail(c, err, clientNotFound, "")
		return
	}
	respond.JSON(c, status, cl)
}

func (h *CRM) CreateClient(c *gin.Context) {
	req := clientRequest{creating: true}
	if !validation.BindJSON(c, &req) {
		return
	}
	var in store.ClientInput
	req.apply(&in)
	id, err := h.Store.CreateClient(c.Request.Context(), currentUserID(c), in)
	if err != nil {
		h.fail(c, err, clientNotFound, "")
		return
	}
	h.respondClient(c, http.StatusCreated, id)
}

func (h *CRM) UpdateClient(c *gin.Context) {
	var req clientRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, clientNotFound)
		return
	}
	ctx := c.Request.Context()
	me := currentUserID(c)
	cl, err := h.Store.CRMClient(ctx, me, id)
	if err != nil {
		h.fail(c, err, clientNotFound, "")
		return
	}
	in := store.ClientInput{Name: cl.Name, Phone: cl.Phone, Telegram: cl.Telegram, Note: cl.Note, Tags: cl.Tags, Source: cl.Source}
	req.apply(&in)
	if cl.UserID == nil && in.Name == nil {
		httperr.Abort(c, http.StatusBadRequest, "A client added by hand needs a name")
		return
	}
	if err := h.Store.UpdateClient(ctx, me, id, in); err != nil {
		h.fail(c, err, clientNotFound, "")
		return
	}
	h.respondClient(c, http.StatusOK, id)
}

func (h *CRM) DeleteClient(c *gin.Context) {
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, clientNotFound)
		return
	}
	switch err := h.Store.DeleteClient(c.Request.Context(), currentUserID(c), id); {
	case errors.Is(err, store.ErrNotManual):
		httperr.Abort(c, http.StatusConflict, "Clients with an account can't be deleted")
	case err != nil:
		h.fail(c, err, clientNotFound, "")
	default:
		c.Status(http.StatusNoContent)
	}
}

// ownTour loads one of the partner's tours (others are not found).
func (h *CRM) ownTour(c *gin.Context, id string) (models.Tour, bool) {
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, tourNotFound)
		return models.Tour{}, false
	}
	tour, err := h.Store.Tour(c.Request.Context(), id)
	if err == nil && !ownedBy(tour.UserID, currentUserID(c)) {
		err = store.ErrNotFound
	}
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return tour, false
	}
	return tour, true
}

// ownParticipant loads the tour of a participant on one of the partner's
// tours.
func (h *CRM) ownParticipant(c *gin.Context) (string, models.Tour, bool) {
	id := c.Param("pid")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, participantNotFound)
		return "", models.Tour{}, false
	}
	tourID, _, err := h.Store.ParticipantTour(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err, participantNotFound, "")
		return "", models.Tour{}, false
	}
	if c.Param("id") != "" && c.Param("id") != tourID {
		httperr.Abort(c, http.StatusNotFound, participantNotFound)
		return "", models.Tour{}, false
	}
	tour, ok := h.ownTour(c, tourID)
	return id, tour, ok
}

// Group is a tour's group: places, the waitlist and the cancelled, with
// payments and totals.
func (h *CRM) Group(c *gin.Context) {
	tour, ok := h.ownTour(c, c.Param("id"))
	if ok {
		h.respondGroup(c, http.StatusOK, tour)
	}
}

func (h *CRM) respondGroup(c *gin.Context, status int, tour models.Tour) {
	ctx := c.Request.Context()
	if fresh, err := h.Store.Tour(ctx, tour.ID); err == nil {
		tour = fresh
	}
	group, err := h.Store.Group(ctx, tour.ID)
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	var ids []string
	for _, p := range group {
		if p.UserID != nil {
			ids = append(ids, *p.UserID)
		}
	}
	people := h.accounts(ctx, ids)
	out := models.CRMGroup{Tour: tour, Participants: group}
	if out.Expenses, err = h.Store.TourExpenses(ctx, tour.ID); err == nil {
		out.Staff, err = h.Store.TourStaff(ctx, tour.ID)
	}
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	for _, e := range out.Expenses {
		out.Totals.Expenses += e.Amount
	}
	for i, p := range group {
		out.Participants[i].Account = accountOf(people, p.UserID)
		switch p.Status {
		case store.StatusNew, store.StatusConfirmed:
			out.Totals.Active++
			out.Totals.Due += p.Due
			out.Totals.Paid += p.Paid
			if p.ArrivedAt != nil {
				out.Totals.Arrived++
			}
			if p.Form != nil {
				out.Totals.Forms++
			}
			if p.WaiverAt != nil {
				out.Totals.Waivers++
			}
		case store.StatusWaitlist:
			out.Totals.Waitlist++
		}
	}
	out.Totals.Margin = out.Totals.Paid - out.Totals.Expenses
	if out.Participants == nil {
		out.Participants = []models.CRMParticipant{}
	}
	respond.JSON(c, status, out)
}

// AddParticipant puts one of the partner's clients on a tour.
func (h *CRM) AddParticipant(c *gin.Context) {
	var req addParticipantRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	tour, ok := h.ownTour(c, c.Param("id"))
	if !ok {
		return
	}
	status := store.StatusConfirmed
	if req.Status != nil {
		status = *req.Status
	}
	_, err := h.Store.AddParticipant(c.Request.Context(), tour.ID, currentUserID(c), *req.ClientID, status)
	switch {
	case errors.Is(err, store.ErrFull):
		httperr.Abort(c, http.StatusConflict, "No places left")
		return
	case errors.Is(err, store.ErrDuplicate):
		httperr.Abort(c, http.StatusConflict, "The client is already on this tour")
		return
	case err != nil:
		h.fail(c, err, clientNotFound, "")
		return
	}
	h.respondGroup(c, http.StatusCreated, tour)
}

// UpdateParticipant changes a participant's status, what they owe or
// whether they came.
func (h *CRM) UpdateParticipant(c *gin.Context) {
	var req participantRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	id, tour, ok := h.ownParticipant(c)
	if !ok {
		return
	}
	change := store.ParticipantChange{Status: req.Status, Arrived: req.Arrived,
		SetAmountDue: req.AmountDue.Set, AmountDue: req.AmountDue.Ptr()}
	promoted, err := h.Store.UpdateParticipant(c.Request.Context(), id, change, h.now())
	if errors.Is(err, store.ErrFull) {
		httperr.Abort(c, http.StatusConflict, "No places left")
		return
	}
	if err != nil {
		h.fail(c, err, participantNotFound, "")
		return
	}
	h.notifyPromoted(c, tour, promoted)
	h.respondGroup(c, http.StatusOK, tour)
}

// DeleteParticipant takes a client the partner added off the tour;
// accounts that signed up can only be cancelled.
func (h *CRM) DeleteParticipant(c *gin.Context) {
	id, tour, ok := h.ownParticipant(c)
	if !ok {
		return
	}
	promoted, err := h.Store.DeleteParticipant(c.Request.Context(), id)
	if errors.Is(err, store.ErrNotManual) {
		httperr.Abort(c, http.StatusConflict, "Participants with an account can only be cancelled")
		return
	}
	if err != nil {
		h.fail(c, err, participantNotFound, "")
		return
	}
	h.notifyPromoted(c, tour, promoted)
	h.respondGroup(c, http.StatusOK, tour)
}

type participantFormRequest struct {
	// null clears it.
	Form *formRequest `json:"form"`
}

// SetForm fills in a participant's form for them (a client added by
// hand, or what an account told the partner).
func (h *CRM) SetForm(c *gin.Context) {
	var req participantFormRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	id, tour, ok := h.ownParticipant(c)
	if !ok {
		return
	}
	if err := h.Store.SetParticipantForm(c.Request.Context(), id, req.Form.model(), h.now()); err != nil {
		h.fail(c, err, participantNotFound, "")
		return
	}
	h.respondGroup(c, http.StatusOK, tour)
}

func (h *CRM) AddPayment(c *gin.Context) {
	var req paymentRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	id, tour, ok := h.ownParticipant(c)
	if !ok {
		return
	}
	var note *string
	if req.Note.Valid && strings.TrimSpace(req.Note.Value) != "" {
		trimmed := strings.TrimSpace(req.Note.Value)
		note = &trimmed
	}
	kind := "payment"
	if req.Kind != nil {
		kind = *req.Kind
	}
	if _, err := h.Store.AddPayment(c.Request.Context(), id, store.PaymentInput{
		Kind: kind, Amount: *req.Amount, Method: *req.Method, PaidOn: *req.PaidOn, Note: note,
	}); err != nil {
		h.fail(c, err, participantNotFound, "")
		return
	}
	h.respondGroup(c, http.StatusCreated, tour)
}

func (h *CRM) DeletePayment(c *gin.Context) {
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, paymentNotFound)
		return
	}
	if err := h.Store.DeletePayment(c.Request.Context(), currentUserID(c), id); err != nil {
		h.fail(c, err, paymentNotFound, "")
		return
	}
	c.Status(http.StatusNoContent)
}

// Summary is the partner's dashboard: clients, money and the next tours.
func (h *CRM) Summary(c *gin.Context) {
	summary, err := h.Store.CRMSummary(c.Request.Context(), currentUserID(c), h.today())
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, summary)
}
