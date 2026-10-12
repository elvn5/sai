package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/trails/internal/models"
)

const outingColumns = `o.id::text, t.id::text, t.slug, t.name, t.area, o.title, o.starts_at, o.returns_at,
	o.meeting_point, o.transport, o.seats,
	(SELECT count(*) FROM outing_participants p WHERE p.outing_id = o.id)::int,
	o.cost_per_person, o.level, o.description, o.contact, o.status, o.user_id::text, o.created_at,
	o.remind_at, o.followup_at
	FROM outings o LEFT JOIN trails t ON t.id = o.trail_id`

func scanOuting(row pgx.Row) (models.Outing, error) {
	var o models.Outing
	var trailID, slug, name, area *string
	var starts, created time.Time
	var returns *time.Time
	err := row.Scan(&o.ID, &trailID, &slug, &name, &area, &o.Title, &starts, &returns, &o.MeetingPoint,
		&o.Transport, &o.Seats, &o.Taken, &o.CostPerPerson, &o.Level, &o.Description, &o.Contact, &o.Status,
		&o.UserID, &created, &o.RemindAt, &o.FollowUpAt)
	if err != nil {
		return o, notFound(err)
	}
	if trailID != nil {
		o.Trail = &models.TrailRef{ID: *trailID, Slug: *slug, Name: *name, Area: *area}
	}
	o.StartsAt, o.ReturnsAt, o.CreatedAt = jsontime.New(starts), timePtr(returns), jsontime.New(created)
	o.Organizer.ID = o.UserID
	return o, nil
}

func (s *Store) collectOutings(ctx context.Context, query string, args ...any) ([]models.Outing, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.Outing, error) { return scanOuting(r) })
}

// UpcomingOutings lists open outings that haven't started (with a few
// hours' grace), soonest first; trailID narrows it to one route.
func (s *Store) UpcomingOutings(ctx context.Context, trailID *string, limit int) ([]models.Outing, error) {
	return s.collectOutings(ctx, `SELECT `+outingColumns+` WHERE o.status = 'open'
		AND o.starts_at > now() - interval '3 hours' AND ($1::text IS NULL OR o.trail_id::text = $1)
		ORDER BY o.starts_at LIMIT $2`, trailID, limit)
}

// OutingsOf lists what userID organizes or joined, newest first.
func (s *Store) OutingsOf(ctx context.Context, userID string) ([]models.Outing, error) {
	return s.collectOutings(ctx, `SELECT `+outingColumns+` WHERE o.user_id::text = $1
		OR EXISTS (SELECT 1 FROM outing_participants p WHERE p.outing_id = o.id AND p.user_id::text = $1)
		ORDER BY o.starts_at DESC LIMIT 100`, userID)
}

func (s *Store) Outing(ctx context.Context, id string) (models.Outing, error) {
	return scanOuting(s.db.QueryRow(ctx, `SELECT `+outingColumns+` WHERE o.id::text = $1`, id))
}

// ParticipantIDs lists who joined, in join order.
func (s *Store) ParticipantIDs(ctx context.Context, outingID string) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT user_id::text FROM outing_participants
		WHERE outing_id::text = $1 ORDER BY joined_at`, outingID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (s *Store) InsertOuting(ctx context.Context, o models.Outing) (models.Outing, error) {
	var id string
	err := s.db.QueryRow(ctx, `INSERT INTO outings (trail_id, title, starts_at, returns_at, meeting_point,
		transport, seats, cost_per_person, level, description, contact, user_id, remind_at, followup_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) RETURNING id::text`,
		trailIDArg(o.Trail), o.Title, o.StartsAt.Time, returnsArg(o.ReturnsAt), o.MeetingPoint, o.Transport,
		o.Seats, o.CostPerPerson, o.Level, o.Description, o.Contact, o.UserID, o.RemindAt, o.FollowUpAt).Scan(&id)
	if err != nil {
		return models.Outing{}, err
	}
	return s.Outing(ctx, id)
}

func (s *Store) UpdateOuting(ctx context.Context, o models.Outing) (models.Outing, error) {
	_, err := s.db.Exec(ctx, `UPDATE outings SET trail_id = $2, title = $3, starts_at = $4, returns_at = $5,
		meeting_point = $6, transport = $7, seats = $8, cost_per_person = $9, level = $10, description = $11,
		contact = $12, status = $13, remind_at = $14, followup_at = $15,
		-- A moved outing gets its reminder and follow-up again.
		reminder_sent_at = CASE WHEN starts_at = $4 THEN reminder_sent_at END,
		followup_sent_at = CASE WHEN starts_at = $4 AND returns_at IS NOT DISTINCT FROM $5 THEN followup_sent_at END,
		updated_at = now() WHERE id::text = $1`,
		o.ID, trailIDArg(o.Trail), o.Title, o.StartsAt.Time, returnsArg(o.ReturnsAt), o.MeetingPoint, o.Transport,
		o.Seats, o.CostPerPerson, o.Level, o.Description, o.Contact, o.Status, o.RemindAt, o.FollowUpAt)
	if err != nil {
		return models.Outing{}, err
	}
	return s.Outing(ctx, o.ID)
}

// Join adds userID to an open, not-yet-started outing with a free seat.
// The row lock keeps two last-seat joins from both succeeding.
func (s *Store) Join(ctx context.Context, outingID, userID string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var seats, taken int
	var open bool
	err = tx.QueryRow(ctx, `SELECT seats, status = 'open' AND starts_at > now(),
		(SELECT count(*) FROM outing_participants p WHERE p.outing_id = o.id)::int
		FROM outings o WHERE id::text = $1 FOR UPDATE`, outingID).Scan(&seats, &open, &taken)
	if err != nil {
		return notFound(err)
	}
	if !open {
		return ErrClosed
	}
	if taken >= seats {
		return ErrFull
	}
	if _, err := tx.Exec(ctx, `INSERT INTO outing_participants (outing_id, user_id) VALUES ($1, $2)`,
		outingID, userID); err != nil {
		return duplicate(err)
	}
	return tx.Commit(ctx)
}

func (s *Store) Leave(ctx context.Context, outingID, userID string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM outing_participants WHERE outing_id::text = $1 AND user_id::text = $2`,
		outingID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) IsParticipant(ctx context.Context, outingID, userID string) (bool, error) {
	var joined bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM outing_participants
		WHERE outing_id::text = $1 AND user_id::text = $2)`, outingID, userID).Scan(&joined)
	return joined, err
}

func trailIDArg(t *models.TrailRef) *string {
	if t == nil {
		return nil
	}
	return &t.ID
}

func returnsArg(t *jsontime.Time) *time.Time {
	if t == nil {
		return nil
	}
	return &t.Time
}
