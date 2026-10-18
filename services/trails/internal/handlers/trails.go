package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/respond"
	"github.com/elvn5/sai/services/pkg/sanitize"
	"github.com/elvn5/sai/services/pkg/validation"
	"github.com/elvn5/sai/services/trails/internal/models"
	"github.com/elvn5/sai/services/trails/internal/store"
)

const (
	trailNotFound = "Route not found"
	slugTaken     = "This address is already used by another route"
	maxTrack      = 5000
)

// trailRequest is the editable part of a route; create additionally
// requires slug, name, area, kind and grade (see Refine).
type trailRequest struct {
	Slug            *string                   `json:"slug" binding:"omitempty,min=1,max=120,slug"`
	Name            *validation.Trimmed       `json:"name" binding:"omitempty,min=1,max=160"`
	Summary         validation.Field[string]  `json:"summary" binding:"omitempty,max=300"`
	Description     *string                   `json:"description" binding:"omitempty,max=20000"`
	Area            *string                   `json:"area" binding:"omitempty,oneof=ala_archa alamedin chunkurchak issyk_ata kegeti sokuluk chon_kemin karakol jeti_oguz suusamyr song_kol kel_suu alay central_tien_shan other"`
	Kind            *string                   `json:"kind" binding:"omitempty,oneof=hike trek climb ski_tour ice_climb rock_climb"`
	Grade           *string                   `json:"grade" binding:"omitempty,oneof=trail 1a 1b 2a 2b 3a 3b 4a 4b 5a 5b 6a 6b"`
	Seasons         []string                  `json:"seasons" binding:"omitempty,max=4,dive,oneof=spring summer autumn winter"`
	Gear            []string                  `json:"gear" binding:"omitempty,max=16,dive,oneof=helmet crampons microspikes ice_axe harness rope ice_screws trekking_poles headlamp tent sleeping_bag stove avalanche_kit snowshoes gaiters sunglasses"`
	SummitElevation validation.Field[int]     `json:"summitElevation" binding:"omitempty,gte=500,lte=7500"`
	StartElevation  validation.Field[int]     `json:"startElevation" binding:"omitempty,gte=300,lte=7500"`
	ElevationGain   validation.Field[int]     `json:"elevationGain" binding:"omitempty,gte=0,lte=7000"`
	DistanceKm      validation.Field[float64] `json:"distanceKm" binding:"omitempty,gt=0,lte=500"`
	DurationHours   validation.Field[float64] `json:"durationHours" binding:"omitempty,gt=0,lte=500"`
	Days            *int                      `json:"days" binding:"omitempty,gte=1,lte=60"`
	Segments        []models.Segment          `json:"segments" binding:"omitempty,max=20,dive"`
	GettingThere    validation.Field[string]  `json:"gettingThere" binding:"omitempty,max=2000"`
	StartLatitude   validation.Field[float64] `json:"startLatitude" binding:"omitempty,gte=-90,lte=90"`
	StartLongitude  validation.Field[float64] `json:"startLongitude" binding:"omitempty,gte=-180,lte=180"`
	Track           []models.TrackPoint       `json:"track"`
	GpxURL          validation.Field[string]  `json:"gpxUrl" binding:"omitempty,max=2000,mediaurl"`
	CoverImageURL   validation.Field[string]  `json:"coverImageUrl" binding:"omitempty,max=2000,mediaurl"`
	Photos          []string                  `json:"photos" binding:"omitempty,max=20,dive,max=2000,mediaurl"`
	IsPublished     *bool                     `json:"isPublished"`
	creating        bool
}

// Refine checks what tags can't: required fields on create and the track.
func (r trailRequest) Refine() map[string]string {
	errs := map[string]string{}
	if r.creating {
		required := map[string]bool{"slug": r.Slug == nil, "name": r.Name == nil, "area": r.Area == nil, "kind": r.Kind == nil, "grade": r.Grade == nil}
		for field, missing := range required {
			if missing {
				errs[field] = "Required"
			}
		}
	}
	if len(r.Track) > maxTrack {
		errs["track"] = "The track has too many points; simplify it"
	}
	for _, p := range r.Track {
		if p[0] == nil || p[1] == nil || *p[0] < -90 || *p[0] > 90 || *p[1] < -180 || *p[1] > 180 {
			errs["track"] = "Invalid track point"
			break
		}
	}
	return errs
}

func (r trailRequest) apply(t *models.Trail) {
	if r.Slug != nil {
		t.Slug = *r.Slug
	}
	if r.Name != nil {
		t.Name = string(*r.Name)
	}
	applyField(&t.Summary, r.Summary)
	if r.Description != nil {
		t.Description = sanitize.HTML(*r.Description)
	}
	for dst, src := range map[*string]*string{&t.Area: r.Area, &t.Kind: r.Kind, &t.Grade: r.Grade} {
		if src != nil {
			*dst = *src
		}
	}
	if r.Seasons != nil {
		t.Seasons = r.Seasons
	}
	if r.Gear != nil {
		t.Gear = r.Gear
	}
	applyField(&t.SummitElevation, r.SummitElevation)
	applyField(&t.StartElevation, r.StartElevation)
	applyField(&t.ElevationGain, r.ElevationGain)
	applyField(&t.DistanceKm, r.DistanceKm)
	applyField(&t.DurationHours, r.DurationHours)
	if r.Days != nil {
		t.Days = *r.Days
	}
	if r.Segments != nil {
		t.Segments = r.Segments
	}
	applyField(&t.GettingThere, r.GettingThere)
	applyField(&t.StartLatitude, r.StartLatitude)
	applyField(&t.StartLongitude, r.StartLongitude)
	if r.Track != nil {
		t.Track = r.Track
	}
	applyField(&t.GpxURL, r.GpxURL)
	applyField(&t.CoverImageURL, r.CoverImageURL)
	if r.Photos != nil {
		t.Photos = r.Photos
	}
	if r.IsPublished != nil {
		t.IsPublished = *r.IsPublished
	}
	// A route drawn from a track starts where the track does.
	if t.StartLatitude == nil && len(t.Track) > 0 {
		t.StartLatitude, t.StartLongitude = t.Track[0][0], t.Track[0][1]
	}
}

// Trails serves /trails: the route catalog. Approved partners (and the
// admin) publish routes; everyone reads them.
type Trails struct{ Deps }

func (h *Trails) List(c *gin.Context) {
	q := validation.NewQuery(c)
	f := store.TrailFilter{
		Area:   q.Enum("area", models.Areas),
		Kind:   q.Enum("kind", models.Kinds),
		Season: q.Enum("season", models.Seasons),
		Grades: q.List("grade", models.Grades),
		Search: q.String("search", 100),
		Sort:   q.EnumDefault("sort", models.TrailSorts, "popular"),
	}
	if !q.OK() {
		return
	}
	trails, err := h.Store.PublishedTrails(c.Request.Context(), f, h.now())
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, trails)
}

func (h *Trails) Areas(c *gin.Context) {
	counts, err := h.Store.AreaCounts(c.Request.Context())
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, counts)
}

// Get shows a published route by slug (drafts to their author and the
// admin).
func (h *Trails) Get(c *gin.Context) {
	trail, err := h.Store.TrailBySlug(c.Request.Context(), c.Param("key"), h.now())
	if err == nil && !trail.IsPublished && !h.canEdit(c, trail) {
		err = store.ErrNotFound
	}
	if err != nil {
		h.fail(c, err, trailNotFound)
		return
	}
	respond.JSON(c, http.StatusOK, trail)
}

func (h *Trails) Mine(c *gin.Context) {
	trails, err := h.Store.TrailsByUser(c.Request.Context(), currentUserID(c), h.now())
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, trails)
}

func (h *Trails) AdminAll(c *gin.Context) {
	trails, err := h.Store.AllTrails(c.Request.Context(), h.now())
	if err != nil {
		h.internalError(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, trails)
}

func (h *Trails) Create(c *gin.Context) {
	req := trailRequest{creating: true}
	if !validation.BindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	if taken, err := h.Store.SlugTaken(ctx, *req.Slug, ""); err != nil || taken {
		h.slugConflict(c, err)
		return
	}
	trail := models.Trail{Days: 1, IsPublished: true, Seasons: []string{}, Gear: []string{}, Photos: []string{},
		Segments: []models.Segment{}, Track: []models.TrackPoint{}}
	if !isAdmin(c) {
		userID := currentUserID(c)
		trail.UserID = &userID
	}
	req.apply(&trail)
	saved, err := h.Store.InsertTrail(ctx, trail, h.now())
	if err != nil {
		h.slugConflict(c, err)
		return
	}
	respond.JSON(c, http.StatusCreated, saved)
}

func (h *Trails) Update(c *gin.Context) {
	var req trailRequest
	if !validation.BindJSON(c, &req) {
		return
	}
	trail, ok := h.editable(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	if req.Slug != nil && *req.Slug != trail.Slug {
		if taken, err := h.Store.SlugTaken(ctx, *req.Slug, trail.ID); err != nil || taken {
			h.slugConflict(c, err)
			return
		}
	}
	req.apply(&trail)
	saved, err := h.Store.UpdateTrail(ctx, trail, h.now())
	if err != nil {
		h.slugConflict(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, saved)
}

func (h *Trails) Delete(c *gin.Context) {
	trail, ok := h.editable(c)
	if !ok {
		return
	}
	if err := h.Store.DeleteTrail(c.Request.Context(), trail.ID); err != nil {
		h.fail(c, err, trailNotFound)
		return
	}
	respond.Empty(c, http.StatusOK)
}

// editable loads the route in the path; partners may only touch their own.
func (h *Trails) editable(c *gin.Context) (models.Trail, bool) {
	id := c.Param("key")
	if !validID(id) {
		h.fail(c, store.ErrNotFound, trailNotFound)
		return models.Trail{}, false
	}
	trail, err := h.Store.Trail(c.Request.Context(), id, h.now())
	if err != nil {
		h.fail(c, err, trailNotFound)
		return models.Trail{}, false
	}
	if !h.canEdit(c, trail) {
		httperr.Abort(c, http.StatusForbidden, "You can only manage your own routes")
		return models.Trail{}, false
	}
	return trail, true
}

func (h *Trails) canEdit(c *gin.Context, t models.Trail) bool {
	return isAdmin(c) || (t.UserID != nil && *t.UserID == currentUserID(c) && currentUserID(c) != "")
}

func (h *Trails) slugConflict(c *gin.Context, err error) {
	if err == nil || errors.Is(err, store.ErrDuplicate) {
		httperr.Abort(c, http.StatusConflict, slugTaken)
		return
	}
	h.internalError(c, err)
}

// trailByKey finds a route by id or slug.
func (d Deps) trailByKey(c *gin.Context, key string) (models.Trail, error) {
	if validID(key) {
		return d.Store.Trail(c.Request.Context(), key, d.now())
	}
	return d.Store.TrailBySlug(c.Request.Context(), strings.ToLower(key), d.now())
}
