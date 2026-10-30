package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/audit"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
)

// Admin is the panel's view of tours (behind RequireAdmin): every
// departure with its group, the organizer's tools to change a
// participant's place, and hiding a tour from the catalog.
type Admin struct{ Deps }

// adminTours is the panel's list, with how many tours each phase has.
type adminTours struct {
	Tours  []models.AdminTour `json:"tours"`
	Counts map[string]int     `json:"counts"`
}

// Tours lists tours: ?phase= (upcoming, current, past; all when unset)
// and ?q= (a title, slug, place or the organizer's name).
func (h *Admin) Tours(c *gin.Context) {
	ctx := c.Request.Context()
	all, err := h.Store.AdminTours(ctx)
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	plain := make([]models.Tour, len(all))
	for i, a := range all {
		plain[i] = a.Tour
	}
	(&Tours{Deps: h.Deps}).withOrganizers(ctx, plain)

	now, phase := h.now(), c.Query("phase")
	q := strings.ToLower(strings.TrimSpace(c.Query("q")))
	out := adminTours{Tours: []models.AdminTour{},
		Counts: map[string]int{models.PhaseUpcoming: 0, models.PhaseCurrent: 0, models.PhasePast: 0}}
	for i, a := range all {
		a.Tour = plain[i]
		a.Phase = a.Tour.Phase(now)
		if q != "" && !strings.Contains(adminSearchText(a.Tour), q) {
			continue
		}
		out.Counts[a.Phase]++
		if phase == "" || phase == a.Phase {
			out.Tours = append(out.Tours, a)
		}
	}
	// Tours on the way and ahead come soonest first.
	if phase == models.PhaseUpcoming || phase == models.PhaseCurrent {
		for i, j := 0, len(out.Tours)-1; i < j; i, j = i+1, j-1 {
			out.Tours[i], out.Tours[j] = out.Tours[j], out.Tours[i]
		}
	}
	respond.JSON(c, http.StatusOK, out)
}

// adminSearchText is what the panel's search looks in.
func adminSearchText(t models.Tour) string {
	parts := []string{t.Title, t.Slug}
	for _, s := range []*string{t.Location, t.MeetingPoint, t.Telegram, t.Phone} {
		if s != nil {
			parts = append(parts, *s)
		}
	}
	if t.Organizer != nil {
		for _, s := range []*string{t.Organizer.FirstName, t.Organizer.Username} {
			if s != nil {
				parts = append(parts, *s)
			}
		}
	}
	return strings.ToLower(strings.Join(parts, " "))
}

// Tour is one tour's group: every participant with their form, check-in
// and payments, the totals, expenses and staff.
func (h *Admin) Tour(c *gin.Context) {
	if tour, ok := h.tour(c, c.Param("id")); ok {
		h.respondTour(c, tour)
	}
}

func (h *Admin) tour(c *gin.Context, id string) (models.Tour, bool) {
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, tourNotFound)
		return models.Tour{}, false
	}
	tour, err := h.Store.Tour(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return tour, false
	}
	return tour, true
}

func (h *Admin) respondTour(c *gin.Context, tour models.Tour) {
	ctx := c.Request.Context()
	group, err := h.group(ctx, tour)
	if err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	out := models.AdminGroup{CRMGroup: group, Phase: group.Tour.Phase(h.now())}
	if tour.UserID != nil {
		out.Organizer = accountOf(h.accounts(ctx, []string{*tour.UserID}), tour.UserID)
	}
	respond.JSON(c, http.StatusOK, out)
}

type adminParticipantRequest struct {
	Status  *string `json:"status" binding:"omitempty,oneof=new confirmed waitlist cancelled"`
	Arrived *bool   `json:"arrived"`
}

// UpdateParticipant changes a participant's place (as the organizer can:
// a place needs a free one, a freed place goes to the waitlist) or marks
// them arrived.
func (h *Admin) UpdateParticipant(c *gin.Context) {
	var req adminParticipantRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	tour, ok := h.tour(c, c.Param("id"))
	if !ok {
		return
	}
	id := c.Param("pid")
	if !validID(id) {
		httperr.Abort(c, http.StatusNotFound, participantNotFound)
		return
	}
	ctx := c.Request.Context()
	if tourID, _, err := h.Store.ParticipantTour(ctx, id); err != nil || tourID != tour.ID {
		httperr.Abort(c, http.StatusNotFound, participantNotFound)
		return
	}
	audit.SetSubject(c, tour.Title)
	promoted, err := h.Store.UpdateParticipant(ctx, id, store.ParticipantChange{Status: req.Status, Arrived: req.Arrived}, h.now())
	if errors.Is(err, store.ErrFull) {
		httperr.Abort(c, http.StatusConflict, "No places left")
		return
	}
	if err != nil {
		h.fail(c, err, participantNotFound, "")
		return
	}
	h.notifyPromoted(c, tour, promoted)
	h.respondTour(c, tour)
}

type publishRequest struct {
	Published *bool `json:"published" binding:"required"`
}

// SetPublished shows the tour in the catalog or hides it.
func (h *Admin) SetPublished(c *gin.Context) {
	var req publishRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	tour, ok := h.tour(c, c.Param("id"))
	if !ok {
		return
	}
	audit.SetSubject(c, tour.Title)
	if err := h.Store.SetTourPublished(c.Request.Context(), tour.ID, *req.Published); err != nil {
		h.fail(c, err, tourNotFound, "")
		return
	}
	h.respondTour(c, tour)
}
