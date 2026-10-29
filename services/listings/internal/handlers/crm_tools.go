package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
)

const (
	leadNotFound     = "Lead not found"
	taskNotFound     = "Task not found"
	templateNotFound = "Template not found"
	// A broadcast reaches at most this many clients.
	maxBroadcast = 500
	// Reports cover at most this long.
	maxReportDays = 366
)

type expenseRequest struct {
	Category *string                  `json:"category" binding:"required,oneof=transport guides food accommodation permits gear other"`
	Amount   *int                     `json:"amount" binding:"required,gt=0,lte=100000000"`
	Note     validation.Field[string] `json:"note" binding:"omitempty,max=500"`
}

type staffRequest struct {
	Role  *string                  `json:"role" binding:"required,oneof=guide assistant driver cook other"`
	Name  *validation.Trimmed      `json:"name" binding:"required,min=1,max=120"`
	Phone validation.Field[string] `json:"phone" binding:"omitempty,phone"`
	Note  validation.Field[string] `json:"note" binding:"omitempty,max=500"`
}

type leadRequest struct {
	ClientID   *string                  `json:"clientId" binding:"omitempty,uuid"`
	TourID     validation.Field[string] `json:"tourId" binding:"omitempty,uuid"`
	Stage      *string                  `json:"stage" binding:"omitempty,oneof=new working offer won lost"`
	LostReason validation.Field[string] `json:"lostReason" binding:"omitempty,max=300"`
	Note       validation.Field[string] `json:"note" binding:"omitempty,max=2000"`
	creating   bool
}

func (r leadRequest) Refine() map[string]string {
	if r.creating && r.ClientID == nil {
		return map[string]string{"clientId": "Required"}
	}
	return nil
}

type taskRequest struct {
	Title    *validation.Trimmed `json:"title" binding:"omitempty,min=1,max=200"`
	DueOn    *string             `json:"dueOn" binding:"omitempty,isodate"`
	ClientID *string             `json:"clientId" binding:"omitempty,uuid"`
	TourID   *string             `json:"tourId" binding:"omitempty,uuid"`
	Done     *bool               `json:"done"`
	creating bool
}

func (r taskRequest) Refine() map[string]string {
	errs := map[string]string{}
	if r.creating && r.Title == nil {
		errs["title"] = "Required"
	}
	if r.creating && r.DueOn == nil {
		errs["dueOn"] = "Required"
	}
	return errs
}

type templateRequest struct {
	Title *validation.Trimmed `json:"title" binding:"required,min=1,max=100"`
	Body  *validation.Trimmed `json:"body" binding:"required,min=1,max=4000"`
}

type broadcastRequest struct {
	// "all", "tour:<id>" or "tag:<tag>".
	Audience *string             `json:"audience" binding:"required,min=3,max=80"`
	Body     *validation.Trimmed `json:"body" binding:"required,min=1,max=4000"`
}

// optText trims an optional text; empty is none.
func optText(f validation.Field[string]) *string {
	if !f.Valid || strings.TrimSpace(f.Value) == "" {
		return nil
	}
	v := strings.TrimSpace(f.Value)
	return &v
}

// --- A tour's expenses and staff ---

func (h *CRM) AddExpense(c *gin.Context) {
	var req expenseRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	tour, ok := h.ownTour(c, c.Param("id"))
	if !ok {
		return
	}
	if err := h.Store.AddExpense(c.Request.Context(), tour.ID, *req.Category, *req.Amount, optText(req.Note)); err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	h.respondGroup(c, http.StatusCreated, tour)
}

func (h *CRM) DeleteExpense(c *gin.Context) {
	h.deleteFromTour(c, h.Store.DeleteExpense, "Expense not found")
}

func (h *CRM) AddStaff(c *gin.Context) {
	var req staffRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	tour, ok := h.ownTour(c, c.Param("id"))
	if !ok {
		return
	}
	m := models.CRMStaff{Role: *req.Role, Name: string(*req.Name), Phone: optText(req.Phone), Note: optText(req.Note)}
	if err := h.Store.AddStaff(c.Request.Context(), tour.ID, m); err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	h.respondGroup(c, http.StatusCreated, tour)
}

func (h *CRM) DeleteStaff(c *gin.Context) {
	h.deleteFromTour(c, h.Store.DeleteStaff, "Staff member not found")
}

// deleteFromTour removes a row of one of the partner's tours and answers
// with the tour's group.
func (h *CRM) deleteFromTour(c *gin.Context, remove func(ctx context.Context, partnerID, id string) (string, error), notFound string) {
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, notFound)
		return
	}
	tourID, err := remove(c.Request.Context(), currentUserID(c), id)
	if err != nil {
		h.fail(c, err, notFound, "")
		return
	}
	if tour, ok := h.ownTour(c, tourID); ok {
		h.respondGroup(c, http.StatusOK, tour)
	}
}

// --- Leads ---

// withRefs fills in the profiles of clients with an account.
func (h *CRM) withRefs(ctx context.Context, refs ...*models.CRMClientRef) {
	var ids []string
	for _, r := range refs {
		if r != nil && r.UserID != nil {
			ids = append(ids, *r.UserID)
		}
	}
	people := h.accounts(ctx, ids)
	for _, r := range refs {
		if r != nil {
			r.Account = accountOf(people, r.UserID)
		}
	}
}

// Leads lists the partner's leads (?clientId= for one client's).
func (h *CRM) Leads(c *gin.Context) {
	q := validation.NewQuery(c)
	clientID := q.String("clientId", 40)
	if !q.OK() {
		return
	}
	ctx := c.Request.Context()
	leads, err := h.Store.Leads(ctx, currentUserID(c), deref(clientID))
	if err != nil {
		h.fail(c, err, leadNotFound, "")
		return
	}
	refs := make([]*models.CRMClientRef, len(leads))
	for i := range leads {
		refs[i] = &leads[i].Client
	}
	h.withRefs(ctx, refs...)
	respond.JSON(c, http.StatusOK, leads)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (h *CRM) respondLead(c *gin.Context, status int, id string) {
	ctx := c.Request.Context()
	lead, err := h.Store.Lead(ctx, currentUserID(c), id)
	if err != nil {
		h.fail(c, err, leadNotFound, "")
		return
	}
	h.withRefs(ctx, &lead.Client)
	respond.JSON(c, status, lead)
}

func (r leadRequest) input() store.LeadInput {
	in := store.LeadInput{ClientID: r.ClientID, Stage: r.Stage,
		SetTour: r.TourID.Set, SetLost: r.LostReason.Set, SetNote: r.Note.Set,
		LostReason: optText(r.LostReason), Note: optText(r.Note)}
	if r.TourID.Valid {
		in.TourID = &r.TourID.Value
	}
	return in
}

func (h *CRM) CreateLead(c *gin.Context) {
	req := leadRequest{creating: true}
	if !validation.BindJSON(c, &req) {
		return
	}
	id, err := h.Store.CreateLead(c.Request.Context(), currentUserID(c), req.input())
	if err != nil {
		h.fail(c, err, clientNotFound, "")
		return
	}
	h.respondLead(c, http.StatusCreated, id)
}

func (h *CRM) UpdateLead(c *gin.Context) {
	var req leadRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, leadNotFound)
		return
	}
	if err := h.Store.UpdateLead(c.Request.Context(), currentUserID(c), id, req.input()); err != nil {
		h.fail(c, err, leadNotFound, "")
		return
	}
	h.respondLead(c, http.StatusOK, id)
}

func (h *CRM) DeleteLead(c *gin.Context) {
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, leadNotFound)
		return
	}
	if err := h.Store.DeleteLead(c.Request.Context(), currentUserID(c), id); err != nil {
		h.fail(c, err, leadNotFound, "")
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Tasks ---

// Tasks lists the partner's open tasks by due day (?all=true with the
// done ones, ?clientId= for one client's).
func (h *CRM) Tasks(c *gin.Context) {
	q := validation.NewQuery(c)
	clientID := q.String("clientId", 40)
	all := q.Bool("all")
	if !q.OK() {
		return
	}
	h.respondTasks(c, http.StatusOK, deref(clientID), all != nil && *all)
}

func (h *CRM) respondTasks(c *gin.Context, status int, clientID string, all bool) {
	ctx := c.Request.Context()
	tasks, err := h.Store.Tasks(ctx, currentUserID(c), clientID, all)
	if err != nil {
		h.fail(c, err, taskNotFound, "")
		return
	}
	refs := make([]*models.CRMClientRef, len(tasks))
	for i := range tasks {
		refs[i] = tasks[i].Client
	}
	h.withRefs(ctx, refs...)
	respond.JSON(c, status, tasks)
}

func (r taskRequest) input() store.TaskInput {
	in := store.TaskInput{DueOn: r.DueOn, ClientID: r.ClientID, TourID: r.TourID, Done: r.Done}
	if r.Title != nil {
		title := string(*r.Title)
		in.Title = &title
	}
	return in
}

// CreateTask adds a task and answers with the open tasks.
func (h *CRM) CreateTask(c *gin.Context) {
	req := taskRequest{creating: true}
	if !validation.BindJSON(c, &req) {
		return
	}
	if err := h.Store.CreateTask(c.Request.Context(), currentUserID(c), req.input()); err != nil {
		h.fail(c, err, clientNotFound, "")
		return
	}
	h.respondTasks(c, http.StatusCreated, "", false)
}

// UpdateTask changes a task (done, title, due day) and answers with the
// open tasks.
func (h *CRM) UpdateTask(c *gin.Context) {
	var req taskRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, taskNotFound)
		return
	}
	if err := h.Store.UpdateTask(c.Request.Context(), currentUserID(c), id, req.input(), h.now()); err != nil {
		h.fail(c, err, taskNotFound, "")
		return
	}
	h.respondTasks(c, http.StatusOK, "", false)
}

func (h *CRM) DeleteTask(c *gin.Context) {
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, taskNotFound)
		return
	}
	if err := h.Store.DeleteTask(c.Request.Context(), currentUserID(c), id); err != nil {
		h.fail(c, err, taskNotFound, "")
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Templates ---

func (h *CRM) Templates(c *gin.Context) {
	list, err := h.Store.Templates(c.Request.Context(), currentUserID(c))
	if err != nil {
		h.fail(c, err, templateNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, list)
}

// SaveTemplate creates a template (POST) or changes one (PUT /:id) and
// answers with them all.
func (h *CRM) SaveTemplate(c *gin.Context) {
	var req templateRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	id := c.Param("id")
	if id != "" && !validID(id) {
		httperr.Abort(c, http.StatusNotFound, templateNotFound)
		return
	}
	if err := h.Store.SaveTemplate(c.Request.Context(), currentUserID(c), id, string(*req.Title), string(*req.Body)); err != nil {
		h.fail(c, err, templateNotFound, "")
		return
	}
	status := http.StatusOK
	if id == "" {
		status = http.StatusCreated
	}
	list, err := h.Store.Templates(c.Request.Context(), currentUserID(c))
	if err != nil {
		h.fail(c, err, templateNotFound, "")
		return
	}
	respond.JSON(c, status, list)
}

func (h *CRM) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, templateNotFound)
		return
	}
	if err := h.Store.DeleteTemplate(c.Request.Context(), currentUserID(c), id); err != nil {
		h.fail(c, err, templateNotFound, "")
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Broadcasts ---

func (h *CRM) Broadcasts(c *gin.Context) {
	list, err := h.Store.Broadcasts(c.Request.Context(), currentUserID(c))
	if err != nil {
		h.fail(c, err, clientNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, list)
}

// Broadcast sends a message to a tour's group, clients with a tag or all
// clients: each with an account gets it in their chat with the partner
// (and the bot tells them); clients added by hand are skipped.
func (h *CRM) Broadcast(c *gin.Context) {
	var req broadcastRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	me := currentUserID(c)
	var audience store.Audience
	kind, value, _ := strings.Cut(*req.Audience, ":")
	switch {
	case kind == "all" && value == "":
	case kind == "tag" && value != "":
		audience.Tag = value
	case kind == "tour" && validID(value):
		if _, ok := h.ownTour(c, value); !ok {
			return
		}
		audience.TourID = value
	default:
		httperr.Abort(c, http.StatusBadRequest, "Unknown audience")
		return
	}
	ids, skipped, err := h.Store.Recipients(ctx, me, audience)
	if err != nil {
		h.fail(c, err, clientNotFound, "")
		return
	}
	if len(ids) > maxBroadcast {
		httperr.Abort(c, http.StatusBadRequest, "Too many recipients")
		return
	}
	body := string(*req.Body)
	sent := 0
	for _, id := range ids {
		chatID, err := h.Store.OpenChat(ctx, me, id)
		if err == nil {
			chat := store.Chat{ID: chatID, PartnerID: me, UserID: id}
			if _, err = h.Store.AddMessage(ctx, chat, me, body, h.now()); err == nil && h.Notify != nil {
				h.Notify.Message(ctx, chat, me, body)
			}
		}
		if err != nil {
			h.Log.ErrorContext(ctx, "broadcast message failed", slog.Any("error", err))
			skipped++
			continue
		}
		sent++
	}
	if err := h.Store.SaveBroadcast(ctx, me, *req.Audience, body, sent, skipped); err != nil {
		h.fail(c, err, clientNotFound, "")
		return
	}
	respond.JSON(c, http.StatusCreated, gin.H{"sent": sent, "skipped": skipped})
}

// --- Reports ---

// Report sums up departures starting from–to (?from=&to=, YYYY-MM-DD; the
// last 90 days by default).
func (h *CRM) Report(c *gin.Context) {
	today := h.today()
	from := c.DefaultQuery("from", today.AddDate(0, 0, -90).Format(time.DateOnly))
	to := c.DefaultQuery("to", today.Format(time.DateOnly))
	start, err1 := time.Parse(time.DateOnly, from)
	end, err2 := time.Parse(time.DateOnly, to)
	if err1 != nil || err2 != nil || end.Before(start) || end.Sub(start) > maxReportDays*24*time.Hour {
		httperr.Abort(c, http.StatusBadRequest, "Invalid period")
		return
	}
	report, err := h.Store.CRMReport(c.Request.Context(), currentUserID(c), from, to)
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	respond.JSON(c, http.StatusOK, report)
}
