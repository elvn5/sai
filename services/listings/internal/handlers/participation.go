package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
)

const (
	// The organizer marks arrivals from a day before the start until a
	// week after it.
	checkInBefore = 24 * time.Hour
	checkInAfter  = 7 * 24 * time.Hour
	// Participants rate a tour from its start until a month after its end.
	ratingWindow = 30 * 24 * time.Hour
)

// formRequest is a participant's form (models.ParticipantForm).
type formRequest struct {
	EmergencyName  *validation.Trimmed      `json:"emergencyName" binding:"required,min=1,max=120"`
	EmergencyPhone *string                  `json:"emergencyPhone" binding:"required,phone"`
	Experience     *string                  `json:"experience" binding:"required,oneof=none hiking trekking mountaineering"`
	Health         validation.Field[string] `json:"health" binding:"omitempty,max=2000"`
	Notes          validation.Field[string] `json:"notes" binding:"omitempty,max=2000"`
}

func (r *formRequest) model() *models.ParticipantForm {
	if r == nil {
		return nil
	}
	text := func(f validation.Field[string]) *string {
		if !f.Valid || strings.TrimSpace(f.Value) == "" {
			return nil
		}
		v := strings.TrimSpace(f.Value)
		return &v
	}
	return &models.ParticipantForm{
		EmergencyName: string(*r.EmergencyName), EmergencyPhone: strings.TrimSpace(*r.EmergencyPhone),
		Experience: *r.Experience, Health: text(r.Health), Notes: text(r.Notes),
	}
}

// joinRequest: the form and the rules can come with the sign-up (the
// site); the bot signs up without a body.
type joinRequest struct {
	Form         *formRequest `json:"form"`
	AcceptWaiver bool         `json:"acceptWaiver"`
}

type ratingRequest struct {
	Rating *int                     `json:"rating" binding:"required,gte=1,lte=5"`
	Text   validation.Field[string] `json:"text" binding:"omitempty,max=1000"`
}

// Participation serves signing up for tours: travelers take a place while
// there are any, the partner sees who signed up and marks who came, and
// participants rate the tour after it (their review of it).
type Participation struct{ Deps }

// Get is one tour as the visitor sees it: whether they signed up, and for
// the organizer the roster. The owner and admins also see an unpublished
// tour.
func (h *Participation) Get(c *gin.Context) {
	tour, ok := h.load(c)
	if !ok {
		return
	}
	if !tour.IsPublished && !ownedBy(tour.UserID, currentUserID(c)) && !isAdmin(c) {
		httperr.Abort(c, http.StatusNotFound, tourNotFound)
		return
	}
	h.respondDetail(c, tour)
}

// Joined lists the tours the signed-in user signed up for.
func (h *Participation) Joined(c *gin.Context) {
	ctx := c.Request.Context()
	tours, err := h.Store.JoinedTours(ctx, currentUserID(c))
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	for i := range tours {
		tours[i].IsJoined = true
	}
	(&Tours{Deps: h.Deps}).withOrganizers(ctx, tours)
	respond.JSON(c, http.StatusOK, tours)
}

func (h *Participation) Join(c *gin.Context) {
	var req joinRequest
	if c.Request.ContentLength != 0 && !validation.BindJSON(c, &req) {
		return
	}
	tour, ok := h.load(c)
	if !ok {
		return
	}
	me := currentUserID(c)
	if ownedBy(tour.UserID, me) {
		httperr.Abort(c, http.StatusBadRequest, "You organize this tour")
		return
	}
	status, err := h.Store.Join(c.Request.Context(), tour.ID, me, tour.StartsAt(), h.now())
	switch {
	case errors.Is(err, store.ErrClosed):
		httperr.Abort(c, http.StatusConflict, "Sign-up for this tour is closed")
		return
	case errors.Is(err, store.ErrDuplicate):
		httperr.Abort(c, http.StatusConflict, "You have already signed up")
		return
	case err != nil:
		h.fail(c, err, tourNotFound, "")
		return
	}
	if req.Form != nil || req.AcceptWaiver {
		if err := h.Store.SaveForm(c.Request.Context(), tour.ID, me, req.Form.model(), req.AcceptWaiver, h.now()); err != nil {
			h.fail(c, err, tourNotFound, "")
			return
		}
	}
	// The organizer hears of those who got a place.
	if updated, ok := h.reload(c, tour.ID); ok && h.Notify != nil && status != store.StatusWaitlist {
		h.Notify.Joined(c.Request.Context(), updated, me)
	}
}

func (h *Participation) Leave(c *gin.Context) {
	tour, ok := h.load(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	me := currentUserID(c)
	status, err := h.Store.MyStatus(ctx, tour.ID, me)
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	promoted, err := h.Store.Leave(ctx, tour.ID, me)
	if err != nil {
		h.fail(c, err, "You haven't signed up for this tour", "")
		return
	}
	// Dropping out of a tour that took place, or off the waitlist, isn't
	// news.
	updated, ok := h.reload(c, tour.ID)
	if ok && h.Notify != nil && updated.StartsAt().After(h.now()) && status != store.StatusWaitlist {
		h.Notify.Left(ctx, updated, me)
	}
	h.notifyPromoted(c, updated, promoted)
}

// notifyPromoted tells the one who got a freed place (an account) about it.
func (d Deps) notifyPromoted(c *gin.Context, t models.Tour, promoted *store.Promoted) {
	if promoted != nil && promoted.UserID != nil && d.Notify != nil {
		d.Notify.Promoted(c.Request.Context(), t, *promoted.UserID)
	}
}

// SaveForm fills in (or changes) the signed-in participant's form and
// accepts the rules, after signing up.
func (h *Participation) SaveForm(c *gin.Context) {
	var req joinRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	tour, ok := h.load(c)
	if !ok {
		return
	}
	if req.Form == nil && !req.AcceptWaiver {
		httperr.Abort(c, http.StatusBadRequest, "Nothing to save")
		return
	}
	if err := h.Store.SaveForm(c.Request.Context(), tour.ID, currentUserID(c), req.Form.model(), req.AcceptWaiver, h.now()); err != nil {
		h.fail(c, err, "You haven't signed up for this tour", "")
		return
	}
	h.reload(c, tour.ID)
}

// SetArrival marks a participant as arrived at the meeting point (PUT) or
// not (DELETE): the organizer's check-in.
func (h *Participation) SetArrival(c *gin.Context) {
	tour, ok := h.load(c)
	if !ok {
		return
	}
	if !ownedBy(tour.UserID, currentUserID(c)) {
		httperr.Abort(c, http.StatusForbidden, "Only the organizer can mark arrivals")
		return
	}
	now := h.now()
	switch {
	case now.Before(tour.StartsAt().Add(-checkInBefore)):
		httperr.Abort(c, http.StatusConflict, "Check-in opens a day before the start")
		return
	case now.After(tour.StartsAt().Add(checkInAfter)):
		httperr.Abort(c, http.StatusConflict, "Check-in is over")
		return
	}
	participant := c.Param("userId")
	if !validID(participant) {
		httperr.Abort(c, http.StatusNotFound, "Participant not found")
		return
	}
	arrived := c.Request.Method == http.MethodPut
	if err := h.Store.SetArrived(c.Request.Context(), tour.ID, participant, arrived, now); err != nil {
		h.fail(c, err, "Participant not found", "")
		return
	}
	h.reload(c, tour.ID)
}

// rateRefusal says why userID can't rate the tour now ("" when they can):
// they signed up, it started and ended less than a month ago, and — once
// the organizer marked arrivals — they were marked.
func (h *Participation) rateRefusal(c *gin.Context, t models.Tour, userID string, roster []store.Participant) (string, error) {
	var mine *store.Participant
	marked := false
	for i, p := range roster {
		if p.UserID == userID {
			mine = &roster[i]
		}
		marked = marked || p.ArrivedAt != nil
	}
	if mine == nil {
		return "Only participants can rate this tour", nil
	}
	now := h.now()
	switch {
	case now.Before(t.StartsAt()):
		return "The tour hasn't started yet", nil
	case now.After(t.EndsAt().Add(ratingWindow)):
		return "Rating is closed", nil
	case marked && mine.ArrivedAt == nil:
		return "Only those who came can rate this tour", nil
	}
	return "", nil
}

// Rate stores the signed-in participant's 1–5 rating of a tour with an
// optional comment, as their review of it; rating again changes it.
func (h *Participation) Rate(c *gin.Context) {
	var req ratingRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	tour, ok := h.load(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	me := currentUserID(c)
	roster, err := h.Store.Participants(ctx, tour.ID)
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	refusal, err := h.rateRefusal(c, tour, me, roster)
	if err != nil {
		h.fail(c, err, tourNotFound, "")
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
	text := ""
	if req.Text.Valid {
		text = strings.TrimSpace(req.Text.Value)
	}
	if err := h.Store.RateTour(ctx, tour.ID, me, *req.Rating, text); err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	h.reload(c, tour.ID)
}

func (h *Participation) load(c *gin.Context) (models.Tour, bool) {
	id := c.Param("id")
	if !validID(id) {
		h.fail(c, store.ErrNotFound, tourNotFound, "")
		return models.Tour{}, false
	}
	tour, err := h.Store.Tour(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return models.Tour{}, false
	}
	return tour, true
}

// reload answers with the tour's current state and returns it.
func (h *Participation) reload(c *gin.Context, id string) (models.Tour, bool) {
	tour, err := h.Store.Tour(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return tour, false
	}
	return tour, h.respondDetail(c, tour)
}

// respondDetail adds the organizer, the visitor's place and rating, and
// for the organizer (or an admin) the roster.
func (h *Participation) respondDetail(c *gin.Context, t models.Tour) bool {
	ctx := c.Request.Context()
	roster, err := h.Store.Participants(ctx, t.ID)
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return false
	}
	me := currentUserID(c)
	t.IsOrganizer = me != "" && ownedBy(t.UserID, me)
	if me != "" {
		if t.MyStatus, err = h.Store.MyStatus(ctx, t.ID, me); err != nil {
			h.fail(c, err, tourNotFound, "")
			return false
		}
		if t.MyStatus == store.StatusCancelled {
			t.MyStatus = ""
		}
		t.IsJoined = t.MyStatus == store.StatusNew || t.MyStatus == store.StatusConfirmed
		if t.MyStatus != "" {
			var waiverAt *time.Time
			if t.MyForm, waiverAt, err = h.Store.MyForm(ctx, t.ID, me); err != nil {
				h.fail(c, err, tourNotFound, "")
				return false
			}
			t.MyWaiverAt = jsontime.Ptr(waiverAt)
		}
	}
	tours := []models.Tour{t}
	(&Tours{Deps: h.Deps}).withOrganizers(ctx, tours)
	t = tours[0]
	if t.IsOrganizer || isAdmin(c) {
		if t.Waitlist, err = h.Store.WaitlistCount(ctx, t.ID); err != nil {
			h.fail(c, err, tourNotFound, "")
			return false
		}
	}
	if (t.IsOrganizer || isAdmin(c)) && len(roster) > 0 {
		ids := make([]string, len(roster))
		for i, p := range roster {
			ids[i] = p.UserID
		}
		people, err := h.Authors.Authors(ctx, ids)
		if err != nil {
			h.Log.WarnContext(ctx, "participants' profiles", slog.Any("error", err))
		}
		t.Roster = make([]models.RosterEntry, 0, len(roster))
		for _, p := range roster {
			t.Roster = append(t.Roster, models.RosterEntry{
				Author: authorFor(people, p.UserID), JoinedAt: jsontime.New(p.JoinedAt), ArrivedAt: jsontime.Ptr(p.ArrivedAt),
			})
		}
	}
	if t.IsJoined {
		refusal, err := h.rateRefusal(c, t, me, roster)
		if err == nil {
			t.MyRating, err = h.Store.MyTourRating(ctx, t.ID, me)
		}
		if err != nil {
			h.fail(c, err, tourNotFound, "")
			return false
		}
		t.CanRate = refusal == ""
	}
	respond.JSON(c, http.StatusOK, t)
	return true
}

// notifyParticipants calls send with the tour's participants, if
// notifications are on and anyone signed up.
func (h *Tours) notifyParticipants(c *gin.Context, tourID string, send func(ids []string)) {
	if h.Notify == nil {
		return
	}
	ids, err := h.Store.ParticipantIDs(c.Request.Context(), tourID)
	if err != nil {
		h.Log.ErrorContext(c.Request.Context(), "could not load participants to notify", slog.Any("error", err))
		return
	}
	if len(ids) > 0 {
		send(ids)
	}
}
