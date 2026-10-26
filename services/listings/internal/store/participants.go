package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/listings/internal/models"
)

var (
	// ErrFull: no places left on the tour.
	ErrFull = errors.New("tour is full")
	// ErrClosed: the tour has started or isn't published.
	ErrClosed = errors.New("tour is closed")
)

// JoinedTours lists the published tours userID signed up for, latest
// start first.
func (s *Store) JoinedTours(ctx context.Context, userID string) ([]models.Tour, error) {
	return s.collectTours(ctx, `SELECT `+tourColumns+` FROM tours WHERE is_published
		AND id IN (SELECT tour_id FROM tour_participants WHERE user_id = $1)
		ORDER BY start_date DESC LIMIT 100`, userID)
}

// Join signs userID up for a published tour that hasn't started (startsAt
// is after now) with a free place. The row lock keeps two last-place
// sign-ups from both succeeding.
func (s *Store) Join(ctx context.Context, tourID, userID string, startsAt, now time.Time) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var capacity *int
	var published, joined bool
	var taken int
	err = tx.QueryRow(ctx, `SELECT capacity, is_published,
		(SELECT count(*) FROM tour_participants p WHERE p.tour_id = t.id)::int,
		EXISTS (SELECT 1 FROM tour_participants p WHERE p.tour_id = t.id AND p.user_id = $2)
		FROM tours t WHERE id = $1 FOR UPDATE`, tourID, userID).Scan(&capacity, &published, &taken, &joined)
	if err != nil {
		return notFound(err)
	}
	if joined {
		return ErrDuplicate
	}
	if !published || !startsAt.After(now) {
		return ErrClosed
	}
	if capacity != nil && taken >= *capacity {
		return ErrFull
	}
	if _, err := tx.Exec(ctx, `INSERT INTO tour_participants (tour_id, user_id) VALUES ($1, $2)`,
		tourID, userID); err != nil {
		return duplicate(err)
	}
	return tx.Commit(ctx)
}

func (s *Store) Leave(ctx context.Context, tourID, userID string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM tour_participants WHERE tour_id = $1 AND user_id = $2`, tourID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Participant is one who signed up: when, and when the organizer marked
// them at the meeting point.
type Participant struct {
	UserID    string
	JoinedAt  time.Time
	ArrivedAt *time.Time
}

// Participants lists who signed up, in sign-up order.
func (s *Store) Participants(ctx context.Context, tourID string) ([]Participant, error) {
	rows, err := s.db.Query(ctx, `SELECT user_id::text, joined_at, arrived_at FROM tour_participants
		WHERE tour_id = $1 ORDER BY joined_at`, tourID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Participant, error) {
		var p Participant
		return p, r.Scan(&p.UserID, &p.JoinedAt, &p.ArrivedAt)
	})
}

// ParticipantIDs lists who signed up, in sign-up order.
func (s *Store) ParticipantIDs(ctx context.Context, tourID string) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT user_id::text FROM tour_participants
		WHERE tour_id = $1 ORDER BY joined_at`, tourID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// SetArrived marks a participant as arrived (at now) or not.
func (s *Store) SetArrived(ctx context.Context, tourID, userID string, arrived bool, now time.Time) error {
	var at *time.Time
	if arrived {
		at = &now
	}
	tag, err := s.db.Exec(ctx, `UPDATE tour_participants
		SET arrived_at = CASE WHEN $3::timestamptz IS NULL THEN NULL ELSE coalesce(arrived_at, $3) END
		WHERE tour_id = $1 AND user_id = $2`, tourID, userID, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AnyArrived reports whether the organizer marked anyone as arrived.
func (s *Store) AnyArrived(ctx context.Context, tourID string) (bool, error) {
	var marked bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tour_participants
		WHERE tour_id = $1 AND arrived_at IS NOT NULL)`, tourID).Scan(&marked)
	return marked, err
}

// MyTourRating is userID's review of a tour, as a rating; nil if none.
func (s *Store) MyTourRating(ctx context.Context, tourID, userID string) (*models.RatingMine, error) {
	var r models.RatingMine
	var text string
	err := s.db.QueryRow(ctx, `SELECT rating, text FROM reviews
		WHERE target_type = 'tour' AND target_id = $1 AND user_id = $2`, tourID, userID).Scan(&r.Rating, &text)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if text != "" {
		r.Text = &text
	}
	return &r, nil
}

// RateTour stores a participant's rating of a tour as their review of it
// (text may be empty), replacing an earlier one, and refreshes the tour's
// rating.
func (s *Store) RateTour(ctx context.Context, tourID, userID string, rating int, text string) error {
	if _, err := s.db.Exec(ctx, `INSERT INTO reviews (target_type, target_id, rating, text, user_id)
		VALUES ('tour', $1, $2, $3, $4) ON CONFLICT (target_type, target_id, user_id)
		DO UPDATE SET rating = EXCLUDED.rating, text = EXCLUDED.text, updated_at = now()`,
		tourID, rating, text, userID); err != nil {
		return err
	}
	return s.RefreshRating(ctx, "tour", tourID)
}
