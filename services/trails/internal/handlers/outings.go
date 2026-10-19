package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
	"github.com/elvn5/sai/services/trails/internal/models"
	"github.com/elvn5/sai/services/trails/internal/store"
)

const (
	outingNotFound = "Outing not found"
	// How far ahead an outing can be planned.
	maxPlanAhead = 180 * 24 * time.Hour
)

type outingRequest struct {
	TrailID       validation.Field[string] `json:"trailId" binding:"omitempty,uuid"`
	Title         *validation.Trimmed      `json:"title" binding:"omitempty,min=1,max=160"`
	StartsAt      *string                  `json:"startsAt" binding:"omitempty,datetime"`
	ReturnsAt     validation.Field[string] `json:"returnsAt" binding:"omitempty,datetime"`
	MeetingPoint  *validation.Trimmed      `json:"meetingPoint" binding:"omitempty,min=1,max=200"`
	Transport     *string                  `json:"transport" binding:"omitempty,oneof=car bus arranged"`
	Seats         *int                     `json:"seats" binding:"omitempty,gte=1,lte=50"`
	CostPerPerson validation.Field[int]    `json:"costPerPerson" binding:"omitempty,gte=0,lte=1000000"`
	Level         *string                  `json:"level" binding:"omitempty,oneof=beginner intermediate advanced"`
	Description   validation.Field[string] `json:"description" binding:"omitempty,max=2000"`
	Contact       validation.Field[string] `json:"contact" binding:"omitempty,max=120"`
	creating      bool
}

func (r outingRequest) Refine() map[string]string {
	errs := map[string]string{}
	if r.creating {
		for field, missing := range map[string]bool{
			"title": r.Title == nil, "startsAt": r.StartsAt == nil, "meetingPoint": r.MeetingPoint == nil,
			"transport": r.Transport == nil, "seats": r.Seats == nil, "level": r.Level == nil,
		} {
			if missing {
				errs[field] = "Required"
			}
		}
	}
	return errs
}

// Outings serves group outings: anyone signed in organizes one, others
// join while seats last. The organizer's contact is shown to participants.
type Outings struct{ Deps }

func (h *Outings) List(c *gin.Context) {
	q := validation.NewQuery(c)
	limit := q.IntDefault("limit", 30, validation.Bound(1), validation.Bound(100))
	var trailID *string
	if key := c.Query("trail"); key != "" {
		trail, err := h.trailByKey(c, key)
		if err != nil {
			h.fail(c, err, trailNotFound)
			return
		}
		trailID = &trail.ID
	}
	outings, err := h.Store.UpcomingOutings(c.Request.Context(), trailID, limit, h.now())
	if err != nil {
		h.internalError(c, err)
		return
	}
	h.respondList(c, outings)
}

// Mine lists what the signed-in user organizes or joined.
func (h *Outings) Mine(c *gin.Context) {
	outings, err := h.Store.OutingsOf(c.Request.Context(), currentUserID(c))
	if err != nil {
		h.internalError(c, err)
		return
	}
	h.respondList(c, outings)
}

// Summary: how many outings the visitor has been on and how far and high
// they took them (the profile's numbers).
func (h *Outings) Summary(c *gin.Context) {
	sum, err := h.Store.OutingsSummary(c.Request.Context(), currentUserID(c), h.now())
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, sum)
}

func (h *Outings) Get(c *gin.Context) {
	outing, ok := h.load(c)
	if ok {
		h.respondDetail(c, outing, http.StatusOK)
	}
}

func (h *Outings) Create(c *gin.Context) {
	req := outingRequest{creating: true}
	if !validation.BindJSON(c, &req) {
		return
	}
	outing := models.Outing{UserID: currentUserID(c), Status: "open"}
	if !h.apply(c, req, &outing, 0) {
		return
	}
	outing.Schedule(h.now())
	saved, err := h.Store.InsertOuting(c.Request.Context(), outing)
	if err != nil {
		h.internalError(c, err)
		return
	}
	h.respondDetail(c, saved, http.StatusCreated)
}

func (h *Outings) Update(c *gin.Context) {
	var req outingRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	outing, ok := h.organized(c)
	if !ok {
		return
	}
	before := outing
	if !h.apply(c, req, &outing, outing.Taken) {
		return
	}
	outing.Schedule(h.now())
	saved, err := h.Store.UpdateOuting(c.Request.Context(), outing)
	if err != nil {
		h.internalError(c, err)
		return
	}
	h.notifyParticipants(c, saved, func(ids []string) { h.Notify.Changed(c.Request.Context(), before, saved, ids) })
	h.respondDetail(c, saved, http.StatusOK)
}

func (h *Outings) Cancel(c *gin.Context) {
	outing, ok := h.organized(c)
	if !ok {
		return
	}
	outing.Status = "cancelled"
	saved, err := h.Store.UpdateOuting(c.Request.Context(), outing)
	if err != nil {
		h.internalError(c, err)
		return
	}
	h.notifyParticipants(c, saved, func(ids []string) { h.Notify.Cancelled(c.Request.Context(), saved, ids) })
	h.respondDetail(c, saved, http.StatusOK)
}

func (h *Outings) Join(c *gin.Context) {
	outing, ok := h.load(c)
	if !ok {
		return
	}
	if outing.UserID == currentUserID(c) {
		httperr.Abort(c, http.StatusBadRequest, "You organize this outing")
		return
	}
	switch err := h.Store.Join(c.Request.Context(), outing.ID, currentUserID(c), h.now()); {
	case errors.Is(err, store.ErrFull):
		httperr.Abort(c, http.StatusConflict, "No seats left")
		return
	case errors.Is(err, store.ErrClosed):
		httperr.Abort(c, http.StatusConflict, "This outing is closed")
		return
	case errors.Is(err, store.ErrDuplicate):
		httperr.Abort(c, http.StatusConflict, "You have already joined")
		return
	case err != nil:
		h.fail(c, err, outingNotFound)
		return
	}
	if updated, ok := h.reload(c, outing.ID); ok && h.Notify != nil {
		h.Notify.Joined(c.Request.Context(), updated, currentUserID(c))
	}
}

func (h *Outings) Leave(c *gin.Context) {
	outing, ok := h.load(c)
	if !ok {
		return
	}
	if err := h.Store.Leave(c.Request.Context(), outing.ID, currentUserID(c)); err != nil {
		h.fail(c, err, "You haven't joined this outing")
		return
	}
	// Dropping out of a cancelled or finished outing isn't news.
	if updated, ok := h.reload(c, outing.ID); ok && h.Notify != nil && updated.Status == "open" && updated.StartsAt.After(h.now()) {
		h.Notify.Left(c.Request.Context(), updated, currentUserID(c))
	}
}

// apply validates req against the outing being created or edited (taken:
// seats already filled) and copies it over.
func (h *Outings) apply(c *gin.Context, req outingRequest, o *models.Outing, taken int) bool {
	now := h.now()
	fieldError := func(field, msg string) bool {
		c.AbortWithStatusJSON(http.StatusBadRequest, validation.Flattened{
			FormErrors: []string{}, FieldErrors: map[string][]string{field: {msg}},
		})
		return false
	}
	if req.TrailID.Set {
		o.Trail = nil
		if req.TrailID.Valid {
			trail, err := h.Store.Trail(c.Request.Context(), req.TrailID.Value, h.now())
			if errors.Is(err, store.ErrNotFound) || (err == nil && !trail.IsPublished) {
				return fieldError("trailId", "Route not found")
			}
			if err != nil {
				h.internalError(c, err)
				return false
			}
			o.Trail = &models.TrailRef{ID: trail.ID, Slug: trail.Slug, Name: trail.Name, Area: trail.Area}
		}
	}
	if req.Title != nil {
		o.Title = string(*req.Title)
	}
	if req.StartsAt != nil {
		starts, _ := time.Parse(time.RFC3339, *req.StartsAt)
		if !starts.After(now) || starts.Sub(now) > maxPlanAhead {
			return fieldError("startsAt", "The start must be in the next 6 months")
		}
		o.StartsAt = jsontime.New(starts)
	}
	if req.ReturnsAt.Set {
		o.ReturnsAt = nil
		if req.ReturnsAt.Valid {
			returns, _ := time.Parse(time.RFC3339, req.ReturnsAt.Value)
			v := jsontime.New(returns)
			o.ReturnsAt = &v
		}
	}
	if o.ReturnsAt != nil && !o.ReturnsAt.After(o.StartsAt.Time) {
		return fieldError("returnsAt", "The return must be after the start")
	}
	if req.MeetingPoint != nil {
		o.MeetingPoint = string(*req.MeetingPoint)
	}
	if req.Transport != nil {
		o.Transport = *req.Transport
	}
	if req.Seats != nil {
		if *req.Seats < taken {
			return fieldError("seats", "More people have already joined")
		}
		o.Seats = *req.Seats
	}
	applyField(&o.CostPerPerson, req.CostPerPerson)
	if req.Level != nil {
		o.Level = *req.Level
	}
	applyField(&o.Description, req.Description)
	applyField(&o.Contact, req.Contact)
	return true
}

func (h *Outings) load(c *gin.Context) (models.Outing, bool) {
	id := c.Param("id")
	if !validID(id) {
		h.fail(c, store.ErrNotFound, outingNotFound)
		return models.Outing{}, false
	}
	outing, err := h.Store.Outing(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err, outingNotFound)
		return models.Outing{}, false
	}
	return outing, true
}

// organized loads an outing the signed-in user organizes that hasn't
// started yet.
func (h *Outings) organized(c *gin.Context) (models.Outing, bool) {
	outing, ok := h.load(c)
	if !ok {
		return outing, false
	}
	if outing.UserID != currentUserID(c) {
		httperr.Abort(c, http.StatusForbidden, "Only the organizer can change this outing")
		return outing, false
	}
	if outing.Status != "open" || !outing.StartsAt.After(h.now()) {
		httperr.Abort(c, http.StatusConflict, "This outing is closed")
		return outing, false
	}
	return outing, true
}

// reload answers with the outing's current state and returns it.
func (h *Outings) reload(c *gin.Context, id string) (models.Outing, bool) {
	outing, err := h.Store.Outing(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err, outingNotFound)
		return outing, false
	}
	h.respondDetail(c, outing, http.StatusOK)
	return outing, true
}

// notifyParticipants calls send with the outing's participants, if
// notifications are on and anyone joined.
func (h *Outings) notifyParticipants(c *gin.Context, o models.Outing, send func(ids []string)) {
	if h.Notify == nil || o.Taken == 0 {
		return
	}
	ids, err := h.Store.ParticipantIDs(c.Request.Context(), o.ID)
	if err != nil {
		h.Log.ErrorContext(c.Request.Context(), "could not load participants to notify", slog.Any("error", err))
		return
	}
	send(ids)
}

// respondList names organizers and hides contacts.
func (h *Outings) respondList(c *gin.Context, outings []models.Outing) {
	ids := make([]string, 0, len(outings))
	for _, o := range outings {
		ids = append(ids, o.UserID)
	}
	people := h.people(c.Request.Context(), ids)
	me := currentUserID(c)
	for i := range outings {
		outings[i].Organizer = people[outings[i].UserID]
		outings[i].IsOrganizer = me != "" && outings[i].UserID == me
		if !outings[i].IsOrganizer {
			outings[i].Contact = nil
		}
	}
	respond.JSON(c, http.StatusOK, outings)
}

// respondDetail adds participants, and the contact for those who joined.
func (h *Outings) respondDetail(c *gin.Context, o models.Outing, status int) {
	ctx := c.Request.Context()
	participantIDs, err := h.Store.ParticipantIDs(ctx, o.ID)
	if err != nil {
		h.internalError(c, err)
		return
	}
	people := h.people(ctx, append([]string{o.UserID}, participantIDs...))
	me := currentUserID(c)
	o.Organizer = people[o.UserID]
	o.Participants = make([]models.Person, 0, len(participantIDs))
	for _, id := range participantIDs {
		o.Participants = append(o.Participants, people[id])
		if id == me {
			o.IsJoined = true
		}
	}
	o.IsOrganizer = me != "" && o.UserID == me
	if !o.IsOrganizer && !o.IsJoined && !isAdmin(c) {
		o.Contact = nil
	}
	respond.JSON(c, status, o)
}
