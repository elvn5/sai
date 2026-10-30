package store

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/listings/internal/models"
)

// AdminTours lists every tour, published or not, with its group's counts:
// the panel's view, latest departure first.
func (s *Store) AdminTours(ctx context.Context) ([]models.AdminTour, error) {
	rows, err := s.db.Query(ctx, `SELECT `+tourColumns+`, g.waitlist, g.cancelled, g.arrived, g.forms, g.waivers, g.due, g.paid
		FROM tours LEFT JOIN LATERAL (
			SELECT count(*) FILTER (WHERE p.status = 'waitlist')::int AS waitlist,
				count(*) FILTER (WHERE p.status = 'cancelled')::int AS cancelled,
				count(*) FILTER (WHERE p.`+activeSQL+` AND p.arrived_at IS NOT NULL)::int AS arrived,
				count(*) FILTER (WHERE p.`+activeSQL+` AND (p.form IS NOT NULL OR EXISTS (
					SELECT 1 FROM participant_profiles pp WHERE pp.user_id = p.user_id)))::int AS forms,
				count(*) FILTER (WHERE p.`+activeSQL+` AND p.waiver_at IS NOT NULL)::int AS waivers,
				coalesce(sum(coalesce(p.amount_due, tours.price, 0)) FILTER (WHERE p.`+activeSQL+`), 0)::int AS due,
				coalesce(sum(`+paidSQL+`) FILTER (WHERE p.`+activeSQL+`), 0)::int AS paid
			FROM tour_participants p WHERE p.tour_id = tours.id) g ON true
		ORDER BY start_date DESC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.AdminTour, error) {
		var a models.AdminTour
		var err error
		a.Tour, err = scanTourWith(r, &a.Counts.Waitlist, &a.Counts.Cancelled, &a.Counts.Arrived,
			&a.Counts.Forms, &a.Counts.Waivers, &a.Counts.Due, &a.Counts.Paid)
		a.Counts.Active = a.Tour.Taken
		return a, err
	})
}

// SetTourPublished shows a tour in the catalog or hides it (the panel's
// moderation); it keeps the first publication date.
func (s *Store) SetTourPublished(ctx context.Context, id string, published bool) error {
	tag, err := s.db.Exec(ctx, `UPDATE tours SET is_published = $2,
		published_at = CASE WHEN $2 THEN coalesce(published_at, now()) ELSE published_at END, updated_at = now()
		WHERE id = $1`, id, published)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
