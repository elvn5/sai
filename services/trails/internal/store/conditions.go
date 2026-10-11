package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/trails/internal/models"
)

const conditionColumns = `c.id::text, t.id::text, t.slug, t.name, t.area, c.user_id::text, c.observed_on::text,
	c.trail_state, c.traction, c.avalanche_risk, c.snow_line, c.reached, c.text, c.photos, c.created_at
	FROM condition_reports c JOIN trails t ON t.id = c.trail_id`

func scanCondition(row pgx.Row) (models.ConditionReport, error) {
	var r models.ConditionReport
	var created time.Time
	err := row.Scan(&r.ID, &r.Trail.ID, &r.Trail.Slug, &r.Trail.Name, &r.Trail.Area, &r.UserID, &r.ObservedOn,
		&r.TrailState, &r.Traction, &r.AvalancheRisk, &r.SnowLine, &r.Reached, &r.Text, &r.Photos, &created)
	r.CreatedAt, r.Photos, r.Author.ID = jsontime.New(created), nonNil(r.Photos), r.UserID
	return r, notFound(err)
}

func (s *Store) collectConditions(ctx context.Context, query string, args ...any) ([]models.ConditionReport, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.ConditionReport, error) { return scanCondition(r) })
}

// TrailConditions lists a route's reports, newest observation first.
func (s *Store) TrailConditions(ctx context.Context, trailID string, limit int) ([]models.ConditionReport, error) {
	return s.collectConditions(ctx, `SELECT `+conditionColumns+` WHERE c.trail_id::text = $1
		ORDER BY c.observed_on DESC, c.created_at DESC LIMIT $2`, trailID, limit)
}

// RecentConditions lists the latest reports on published routes.
func (s *Store) RecentConditions(ctx context.Context, limit int) ([]models.ConditionReport, error) {
	return s.collectConditions(ctx, `SELECT `+conditionColumns+` WHERE t.is_published
		ORDER BY c.observed_on DESC, c.created_at DESC LIMIT $1`, limit)
}

func (s *Store) Condition(ctx context.Context, id string) (models.ConditionReport, error) {
	return scanCondition(s.db.QueryRow(ctx, `SELECT `+conditionColumns+` WHERE c.id::text = $1`, id))
}

func (s *Store) InsertCondition(ctx context.Context, r models.ConditionReport) (models.ConditionReport, error) {
	var id string
	err := s.db.QueryRow(ctx, `INSERT INTO condition_reports
		(trail_id, user_id, observed_on, trail_state, traction, avalanche_risk, snow_line, reached, text, photos)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id::text`,
		r.Trail.ID, r.UserID, r.ObservedOn, r.TrailState, r.Traction, r.AvalancheRisk, r.SnowLine, r.Reached,
		r.Text, nonNil(r.Photos)).Scan(&id)
	if err != nil {
		return models.ConditionReport{}, err
	}
	return s.Condition(ctx, id)
}

func (s *Store) DeleteCondition(ctx context.Context, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM condition_reports WHERE id::text = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
