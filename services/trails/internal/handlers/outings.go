package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
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
	// The organizer marks arrivals from a day before the start until a
	// week after it.
	checkInBefore = 24 * time.Hour
	checkInAfter  = 7 * 24 * time.Hour
	// Participants rate an outing from its start for this long.
	ratingWindow = 30 * 24 * time.Hour
	// Reviews on an organizer's page.
	organizerReviews = 20
)

type ratingRequest struct {
	Rating *int                     `json:"rating" binding:"required,gte=1,lte=5"`
	Text   validation.Field[string] `json:"text" binding:"omitempty,max=1000"`
}

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

// SetArrival marks a participant as arrived at the meeting point (PUT) or
// not (DELETE): the organizer's check-in.
func (h *Outings) SetArrival(c *gin.Context) {
	outing, ok := h.load(c)
	if !ok {
		return
	}
	if outing.UserID != currentUserID(c) {
		httperr.Abort(c, http.StatusForbidden, "Only the organizer can mark arrivals")
		return
	}
	now := h.now()
	switch {
	case outing.Status != "open":
		httperr.Abort(c, http.StatusConflict, "This outing is closed")
		return
	case now.Before(outing.StartsAt.Add(-checkInBefore)):
		httperr.Abort(c, http.StatusConflict, "Check-in opens a day before the start")
		return
	case now.After(outing.StartsAt.Add(checkInAfter)):
		httperr.Abort(c, http.StatusConflict, "Check-in is over")
		return
	}
	participant := c.Param("userId")
	if !validID(participant) {
		httperr.Abort(c, http.StatusNotFound, "Participant not found")
		return
	}
	arrived := c.Request.Method == http.MethodPut
	if err := h.Store.SetArrived(c.Request.Context(), outing.ID, participant, arrived, now); err != nil {
		h.fail(c, err, "Participant not found")
		return
	}
	h.reload(c, outing.ID)
}

// rateRefusal says why userID can't rate the outing now ("" when they can):
// they joined it, it took place (open, started) less than a month ago, and
// — once the organizer marked arrivals — they were marked.
func (h *Outings) rateRefusal(c *gin.Context, o models.Outing, userID string) (string, error) {
	ctx := c.Request.Context()
	joined, err := h.Store.IsParticipant(ctx, o.ID, userID)
	if err != nil || !joined {
		return "Only participants can rate this outing", err
	}
	now := h.now()
	switch {
	case o.Status != "open":
		return "This outing was cancelled", nil
	case now.Before(o.StartsAt.Time):
		return "The outing hasn't started yet", nil
	case now.After(o.StartsAt.Add(ratingWindow)):
		return "Rating is closed", nil
	}
	marked, err := h.Store.AnyArrived(ctx, o.ID)
	if err != nil || !marked {
		return "", err
	}
	arrived, err := h.Store.Arrived(ctx, o.ID, userID)
	if err != nil || arrived {
		return "", err
	}
	return "Only those who came can rate this outing", nil
}

// Rate stores the signed-in participant's 1–5 rating of an outing (and so
// of its organizer), with an optional comment; rating again changes it.
func (h *Outings) Rate(c *gin.Context) {
	var req ratingRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	outing, ok := h.load(c)
	if !ok {
		return
	}
	me := currentUserID(c)
	refusal, err := h.rateRefusal(c, outing, me)
	if err != nil {
		h.internalError(c, err)
		return
	}
	if refusal != "" {
		status := http.StatusConflict
		if strings.HasPrefix(refusal, "Only") {
			status = http.StatusForbidden
		}
		httperr.Abort(c, status, refusal)
		return
	}
	var text *string
	if req.Text.Valid {
		if t := strings.TrimSpace(req.Text.Value); t != "" {
			text = &t
		}
	}
	if err := h.Store.RateOuting(c.Request.Context(), outing, me, *req.Rating, text); err != nil {
		h.internalError(c, err)
		return
	}
	h.reload(c, outing.ID)
}

// Organizer is an organizer's public record: outings held, participants,
// rating, upcoming outings and the latest reviews.
func (h *Outings) Organizer(c *gin.Context) {
	id := c.Param("userId")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, "Organizer not found")
		return
	}
	ctx := c.Request.Context()
	out, err := h.Store.OrganizerOutings(ctx, id, organizerReviews, h.now())
	if err != nil {
		h.internalError(c, err)
		return
	}
	ids := []string{id}
	for _, r := range out.Reviews {
		ids = append(ids, r.UserID)
	}
	people := h.people(ctx, ids)
	for i := range out.Upcoming {
		out.Upcoming[i].Organizer, out.Upcoming[i].Contact = people[id], nil
	}
	for i := range out.Reviews {
		out.Reviews[i].Author = people[out.Reviews[i].UserID]
	}
	if out.Upcoming == nil {
		out.Upcoming = []models.Outing{}
	}
	if out.Reviews == nil {
		out.Reviews = []models.OutingReview{}
	}
	respond.JSON(c, http.StatusOK, out)
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

// respondDetail adds participants, and the contact for those who joined;
// the organizer also gets the roster (when each joined and arrived).
func (h *Outings) respondDetail(c *gin.Context, o models.Outing, status int) {
	ctx := c.Request.Context()
	participants, err := h.Store.Participants(ctx, o.ID)
	if err != nil {
		h.internalError(c, err)
		return
	}
	ids := make([]string, 0, len(participants)+1)
	ids = append(ids, o.UserID)
	for _, p := range participants {
		ids = append(ids, p.UserID)
	}
	people := h.people(ctx, ids)
	me := currentUserID(c)
	o.Organizer = people[o.UserID]
	o.IsOrganizer = me != "" && o.UserID == me
	o.Participants = make([]models.Person, 0, len(participants))
	for _, p := range participants {
		o.Participants = append(o.Participants, people[p.UserID])
		if p.UserID == me {
			o.IsJoined = true
		}
		if o.IsOrganizer || isAdmin(c) {
			o.Roster = append(o.Roster, models.RosterEntry{
				Person: people[p.UserID], JoinedAt: jsontime.New(p.JoinedAt), ArrivedAt: arrivedAt(p.ArrivedAt),
			})
		}
	}
	if !o.IsOrganizer && !o.IsJoined && !isAdmin(c) {
		o.Contact = nil
	}
	if o.IsJoined {
		refusal, err := h.rateRefusal(c, o, me)
		if err == nil {
			o.MyRating, err = h.Store.MyRating(ctx, o.ID, me)
		}
		if err != nil {
			h.internalError(c, err)
			return
		}
		o.CanRate = refusal == ""
	}
	respond.JSON(c, status, o)
}

func arrivedAt(t *time.Time) *jsontime.Time {
	if t == nil {
		return nil
	}
	v := jsontime.New(*t)
	return &v
}
