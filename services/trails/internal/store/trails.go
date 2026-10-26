package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/trails/internal/models"
)

// trailColumns leaves out the track (large); detail queries add it. Run
// queries that use it through atNow.
const trailColumns = `t.id::text, t.slug, t.name, t.summary, t.description, t.area, t.kind, t.grade,
	t.seasons, t.gear, t.summit_elevation, t.start_elevation, t.elevation_gain, t.distance_km::float8,
	t.duration_hours::float8, t.days, t.segments, t.getting_there, t.start_latitude, t.start_longitude,
	t.gpx_url, t.cover_image_url, t.photos, t.is_published, t.user_id::text, t.created_at, t.updated_at,
	lc.observed_on::text, lc.trail_state, lc.traction`

// trailJoins adds the newest condition report.
const trailJoins = ` FROM trails t
	LEFT JOIN LATERAL (SELECT observed_on, trail_state, traction FROM condition_reports c
		WHERE c.trail_id = t.id ORDER BY observed_on DESC, created_at DESC LIMIT 1) lc ON TRUE`

func scanTrail(row pgx.Row, withTrack bool) (models.Trail, error) {
	var t models.Trail
	var created, updated time.Time
	var observedOn, state, traction *string
	dest := []any{&t.ID, &t.Slug, &t.Name, &t.Summary, &t.Description, &t.Area, &t.Kind, &t.Grade,
		&t.Seasons, &t.Gear, &t.SummitElevation, &t.StartElevation, &t.ElevationGain, &t.DistanceKm,
		&t.DurationHours, &t.Days, &t.Segments, &t.GettingThere, &t.StartLatitude, &t.StartLongitude,
		&t.GpxURL, &t.CoverImageURL, &t.Photos, &t.IsPublished, &t.UserID, &created, &updated,
		&observedOn, &state, &traction}
	if withTrack {
		dest = append(dest, &t.Track)
	}
	if err := row.Scan(dest...); err != nil {
		return t, notFound(err)
	}
	t.CreatedAt, t.UpdatedAt = jsontime.New(created), jsontime.New(updated)
	t.Seasons, t.Gear, t.Photos, t.Segments = nonNil(t.Seasons), nonNil(t.Gear), nonNil(t.Photos), nonNil(t.Segments)
	if observedOn != nil {
		t.LatestCondition = &models.ConditionSummary{ObservedOn: *observedOn, TrailState: *state, Traction: *traction}
	}
	if withTrack {
		t.Track = nonNil(t.Track)
		t.TrackStats = models.ComputeTrackStats(t.Track)
	}
	return t, nil
}

func (s *Store) collectTrails(ctx context.Context, now time.Time, query string, args ...any) ([]models.Trail, error) {
	query, args = atNow(query, now, args)
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.Trail, error) { return scanTrail(r, false) })
}

type TrailFilter struct {
	Area, Kind, Season, Search *string
	Grades                     []string
	Sort                       string
}

// PublishedTrails lists the catalog.
func (s *Store) PublishedTrails(ctx context.Context, f TrailFilter, now time.Time) ([]models.Trail, error) {
	w := &where{}
	w.add("t.is_published = ?", true)
	if f.Area != nil {
		w.add("t.area = ?", *f.Area)
	}
	if f.Kind != nil {
		w.add("t.kind = ?", *f.Kind)
	}
	if f.Season != nil {
		w.add("? = ANY(t.seasons)", *f.Season)
	}
	if len(f.Grades) > 0 {
		w.add("t.grade = ANY(?)", f.Grades)
	}
	if f.Search != nil {
		p := w.next("%" + *f.Search + "%")
		w.conds = append(w.conds, "(t.name ILIKE "+p+" OR t.summary ILIKE "+p+")")
	}
	order := map[string]string{
		"name":      "t.name",
		"elevation": "t.summit_elevation DESC NULLS LAST, t.name",
		"newest":    "t.created_at DESC",
	}[f.Sort]
	if order == "" {
		// Most talked about: fresh reports first.
		order = `(SELECT count(*) FROM condition_reports c WHERE c.trail_id = t.id AND c.observed_on > @now - interval '60 days') DESC, t.name`
	}
	return s.collectTrails(ctx, now, `SELECT `+trailColumns+trailJoins+` WHERE `+w.sql()+` ORDER BY `+order+` LIMIT 200`, w.args...)
}

// AreaCounts counts published routes per area.
func (s *Store) AreaCounts(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.Query(ctx, `SELECT area, count(*)::int FROM trails WHERE is_published GROUP BY area`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var area string
		var n int
		if err := rows.Scan(&area, &n); err != nil {
			return nil, err
		}
		out[area] = n
	}
	return out, rows.Err()
}

// TrailBySlug loads a route with its track (published or not; callers
// decide who may see drafts).
func (s *Store) TrailBySlug(ctx context.Context, slug string, now time.Time) (models.Trail, error) {
	query, args := atNow(`SELECT `+trailColumns+`, t.track`+trailJoins+` WHERE t.slug = $1`, now, []any{slug})
	return scanTrail(s.db.QueryRow(ctx, query, args...), true)
}

func (s *Store) Trail(ctx context.Context, id string, now time.Time) (models.Trail, error) {
	query, args := atNow(`SELECT `+trailColumns+`, t.track`+trailJoins+` WHERE t.id::text = $1`, now, []any{id})
	return scanTrail(s.db.QueryRow(ctx, query, args...), true)
}

func (s *Store) TrailsByUser(ctx context.Context, userID string, now time.Time) ([]models.Trail, error) {
	return s.collectTrails(ctx, now, `SELECT `+trailColumns+trailJoins+` WHERE t.user_id::text = $1 ORDER BY t.created_at DESC`, userID)
}

func (s *Store) AllTrails(ctx context.Context, now time.Time) ([]models.Trail, error) {
	return s.collectTrails(ctx, now, `SELECT `+trailColumns+trailJoins+` ORDER BY t.created_at DESC`)
}

func (s *Store) SlugTaken(ctx context.Context, slug, excludeID string) (bool, error) {
	var taken bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM trails WHERE slug = $1 AND ($2 = '' OR id::text <> $2))`,
		slug, excludeID).Scan(&taken)
	return taken, err
}

const trailWriteColumns = `slug, name, summary, description, area, kind, grade, seasons, gear, summit_elevation,
	start_elevation, elevation_gain, distance_km, duration_hours, days, segments, getting_there, start_latitude,
	start_longitude, track, gpx_url, cover_image_url, photos, is_published`

func trailWriteArgs(t models.Trail) []any {
	return []any{t.Slug, t.Name, t.Summary, t.Description, t.Area, t.Kind, t.Grade, t.Seasons, t.Gear,
		t.SummitElevation, t.StartElevation, t.ElevationGain, t.DistanceKm, t.DurationHours, t.Days,
		t.Segments, t.GettingThere, t.StartLatitude, t.StartLongitude, nonNil(t.Track), t.GpxURL,
		t.CoverImageURL, t.Photos, t.IsPublished}
}

func (s *Store) InsertTrail(ctx context.Context, t models.Trail, now time.Time) (models.Trail, error) {
	var id string
	err := s.db.QueryRow(ctx, `INSERT INTO trails (`+trailWriteColumns+`, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
		RETURNING id::text`, append(trailWriteArgs(t), t.UserID)...).Scan(&id)
	if err != nil {
		return models.Trail{}, duplicate(err)
	}
	return s.Trail(ctx, id, now)
}

func (s *Store) UpdateTrail(ctx context.Context, t models.Trail, now time.Time) (models.Trail, error) {
	tag, err := s.db.Exec(ctx, `UPDATE trails SET (`+trailWriteColumns+`, updated_at) =
		($2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, now())
		WHERE id::text = $1`, append([]any{t.ID}, trailWriteArgs(t)...)...)
	if err != nil {
		return models.Trail{}, duplicate(err)
	}
	if tag.RowsAffected() == 0 {
		return models.Trail{}, ErrNotFound
	}
	return s.Trail(ctx, t.ID, now)
}

func (s *Store) DeleteTrail(ctx context.Context, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM trails WHERE id::text = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
