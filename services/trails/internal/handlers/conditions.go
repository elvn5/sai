package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/validation"
	"github.com/elvn5/sai/services/trails/internal/models"
	"github.com/elvn5/sai/services/trails/internal/store"
)

const (
	conditionNotFound = "Report not found"
	// Reports describe recent trips: up to a month back.
	maxReportAgeDays = 30
)

// Asia/Bishkek: "today" for observation dates.
var bishkek = time.FixedZone("Asia/Bishkek", 6*60*60)

type conditionRequest struct {
	ObservedOn    *string                  `json:"observedOn" binding:"required,isodate"`
	TrailState    *string                  `json:"trailState" binding:"required,oneof=clear wet patchy_snow snow deep_snow ice"`
	Traction      *string                  `json:"traction" binding:"required,oneof=none microspikes crampons"`
	AvalancheRisk validation.Field[string] `json:"avalancheRisk" binding:"omitempty,oneof=low moderate considerable high"`
	SnowLine      validation.Field[int]    `json:"snowLine" binding:"omitempty,gte=500,lte=7500"`
	Reached       validation.Field[string] `json:"reached" binding:"omitempty,max=120"`
	Text          validation.Field[string] `json:"text" binding:"omitempty,max=2000"`
	Photos        []string                 `json:"photos" binding:"omitempty,max=6,dive,max=2000,mediaurl"`
	today         string
	earliest      string
}

func (r conditionRequest) Refine() map[string]string {
	if r.ObservedOn != nil && (*r.ObservedOn > r.today || *r.ObservedOn < r.earliest) {
		return map[string]string{"observedOn": "The date must be within the last 30 days"}
	}
	return nil
}

// Conditions serves trail condition reports: what the route looked like
// on a given day. Anyone signed in can file one.
type Conditions struct{ Deps }

// Recent is the feed of the latest reports across routes.
func (h *Conditions) Recent(c *gin.Context) {
	limit := validation.NewQuery(c).IntDefault("limit", 20, validation.Bound(1), validation.Bound(50))
	reports, err := h.Store.RecentConditions(c.Request.Context(), limit)
	if err != nil {
		h.internalError(c, err)
		return
	}
	h.respondReports(c, reports)
}

// ForTrail lists a route's reports.
func (h *Conditions) ForTrail(c *gin.Context) {
	trail, err := h.trailByKey(c, c.Param("key"))
	if err != nil {
		h.fail(c, err, trailNotFound)
		return
	}
	limit := validation.NewQuery(c).IntDefault("limit", 10, validation.Bound(1), validation.Bound(50))
	reports, err := h.Store.TrailConditions(c.Request.Context(), trail.ID, limit)
	if err != nil {
		h.internalError(c, err)
		return
	}
	h.respondReports(c, reports)
}

func (h *Conditions) Create(c *gin.Context) {
	now := h.now().In(bishkek)
	req := conditionRequest{
		today:    now.Format(time.DateOnly),
		earliest: now.AddDate(0, 0, -maxReportAgeDays).Format(time.DateOnly),
	}
	if !validation.BindJSON(c, &req) {
		return
	}
	trail, err := h.trailByKey(c, c.Param("key"))
	if err == nil && !trail.IsPublished {
		err = store.ErrNotFound
	}
	if err != nil {
		h.fail(c, err, trailNotFound)
		return
	}
	report := models.ConditionReport{
		Trail: models.TrailRef{ID: trail.ID}, UserID: currentUserID(c), ObservedOn: *req.ObservedOn,
		TrailState: *req.TrailState, Traction: *req.Traction, AvalancheRisk: req.AvalancheRisk.Ptr(),
		SnowLine: req.SnowLine.Ptr(), Reached: req.Reached.Ptr(), Text: req.Text.Ptr(), Photos: req.Photos,
	}
	saved, err := h.Store.InsertCondition(c.Request.Context(), report)
	if err != nil {
		h.internalError(c, err)
		return
	}
	saved.Author = h.people(c.Request.Context(), []string{saved.UserID})[saved.UserID]
	respond.JSON(c, http.StatusCreated, saved)
}

// Delete removes a report: its author or the admin.
func (h *Conditions) Delete(c *gin.Context) {
	id := c.Param("id")
	if !validID(id) {
		h.fail(c, store.ErrNotFound, conditionNotFound)
		return
	}
	ctx := c.Request.Context()
	report, err := h.Store.Condition(ctx, id)
	if err != nil {
		h.fail(c, err, conditionNotFound)
		return
	}
	if !isAdmin(c) && report.UserID != currentUserID(c) {
		httperr.Abort(c, http.StatusForbidden, "You can only delete your own reports")
		return
	}
	if err := h.Store.DeleteCondition(ctx, id); err != nil {
		h.fail(c, err, conditionNotFound)
		return
	}
	respond.Empty(c, http.StatusOK)
}

func (h *Conditions) respondReports(c *gin.Context, reports []models.ConditionReport) {
	ids := make([]string, 0, len(reports))
	for _, r := range reports {
		ids = append(ids, r.UserID)
	}
	people := h.people(c.Request.Context(), ids)
	for i := range reports {
		reports[i].Author = people[reports[i].UserID]
	}
	respond.JSON(c, http.StatusOK, reports)
}
