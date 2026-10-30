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
	// ErrClosed: the tour is over or isn't published.
	ErrClosed = errors.New("tour is closed")
)

// A participant takes a place while "new" or "confirmed" (activeSQL);
// "waitlist" waits for one, "cancelled" is kept for the partner's history.
const (
	StatusNew       = "new"
	StatusConfirmed = "confirmed"
	StatusWaitlist  = "waitlist"
	StatusCancelled = "cancelled"

	activeSQL = "status IN ('new', 'confirmed')"
)

// JoinedTours lists the published tours userID has a place on, latest
// start first.
func (s *Store) JoinedTours(ctx context.Context, userID string) ([]models.Tour, error) {
	return s.collectTours(ctx, `SELECT `+tourColumns+` FROM tours WHERE is_published
		AND id IN (SELECT tour_id FROM tour_participants WHERE user_id = $1 AND `+activeSQL+`)
		ORDER BY start_date DESC LIMIT 100`, userID)
}

// Join signs userID up for a published tour that isn't over (endsAt is
// after now; a late traveler can still catch up with the group): with a place while there are any, else on the waitlist.
// It returns the status they got. The row lock keeps two last-place
// sign-ups from both getting it. The tour's partner gets them as a client.
func (s *Store) Join(ctx context.Context, tourID, userID string, endsAt, now time.Time) (string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var capacity *int
	var published bool
	var owner, current *string
	var taken int
	err = tx.QueryRow(ctx, `SELECT capacity, is_published, user_id::text,
		(SELECT count(*) FROM tour_participants p WHERE p.tour_id = t.id AND p.`+activeSQL+`)::int,
		(SELECT status FROM tour_participants p WHERE p.tour_id = t.id AND p.user_id = $2)
		FROM tours t WHERE id = $1 FOR UPDATE`, tourID, userID).Scan(&capacity, &published, &owner, &taken, &current)
	if err != nil {
		return "", notFound(err)
	}
	if current != nil && *current != StatusCancelled {
		return "", ErrDuplicate
	}
	if !published || !endsAt.After(now) {
		return "", ErrClosed
	}
	status := StatusNew
	if capacity != nil && taken >= *capacity {
		status = StatusWaitlist
	}
	if _, err := tx.Exec(ctx, `INSERT INTO tour_participants (tour_id, user_id, status) VALUES ($1, $2, $3)
		ON CONFLICT (tour_id, user_id) WHERE user_id IS NOT NULL
		DO UPDATE SET status = EXCLUDED.status, joined_at = now(), arrived_at = NULL`,
		tourID, userID, status); err != nil {
		return "", duplicate(err)
	}
	if owner != nil {
		if _, err := tx.Exec(ctx, `INSERT INTO crm_clients (partner_id, user_id, source) VALUES ($1, $2, 'ermekten')
			ON CONFLICT (partner_id, user_id) DO NOTHING`, *owner, userID); err != nil {
			return "", err
		}
		if err := winLeads(ctx, tx, *owner, userID, tourID); err != nil {
			return "", err
		}
	}
	return status, tx.Commit(ctx)
}

// Leave drops userID's sign-up (a place or the waitlist). A freed place
// goes to the first on the waitlist, returned (nil if none).
func (s *Store) Leave(ctx context.Context, tourID, userID string) (*Promoted, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	err = tx.QueryRow(ctx, `DELETE FROM tour_participants WHERE tour_id = $1 AND user_id = $2
		AND status <> 'cancelled' RETURNING status`, tourID, userID).Scan(&status)
	if err != nil {
		return nil, notFound(err)
	}
	var promoted *Promoted
	if status == StatusNew || status == StatusConfirmed {
		if promoted, err = promote(ctx, tx, tourID); err != nil {
			return nil, err
		}
	}
	return promoted, tx.Commit(ctx)
}

// Promoted is the participant who got a freed place: an account (UserID)
// or a client the partner added (UserID nil).
type Promoted struct {
	ParticipantID string
	UserID        *string
}

// promote gives a free place on the tour to the first on its waitlist.
// Call it in the transaction that freed the place.
func promote(ctx context.Context, tx pgx.Tx, tourID string) (*Promoted, error) {
	if _, err := tx.Exec(ctx, `SELECT 1 FROM tours WHERE id = $1 FOR UPDATE`, tourID); err != nil {
		return nil, err
	}
	var p Promoted
	err := tx.QueryRow(ctx, `UPDATE tour_participants SET status = 'new' WHERE id = (
		SELECT p.id FROM tour_participants p JOIN tours t ON t.id = p.tour_id
		WHERE p.tour_id = $1 AND p.status = 'waitlist'
			AND (t.capacity IS NULL OR (SELECT count(*) FROM tour_participants a
				WHERE a.tour_id = t.id AND a.`+activeSQL+`) < t.capacity)
		ORDER BY p.joined_at LIMIT 1)
		RETURNING id::text, user_id::text`, tourID).Scan(&p.ParticipantID, &p.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// MyStatus is userID's status on the tour ("" when they haven't signed up).
func (s *Store) MyStatus(ctx context.Context, tourID, userID string) (string, error) {
	var status string
	err := s.db.QueryRow(ctx, `SELECT status FROM tour_participants WHERE tour_id = $1 AND user_id = $2`,
		tourID, userID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return status, err
}

// Participant is an account with a place on the tour: when they signed
// up, and when the organizer marked them at the meeting point.
type Participant struct {
	UserID    string
	JoinedAt  time.Time
	ArrivedAt *time.Time
}

// Participants lists the accounts with a place, in sign-up order (clients
// the partner added by hand are only in the CRM).
func (s *Store) Participants(ctx context.Context, tourID string) ([]Participant, error) {
	rows, err := s.db.Query(ctx, `SELECT user_id::text, joined_at, arrived_at FROM tour_participants
		WHERE tour_id = $1 AND user_id IS NOT NULL AND `+activeSQL+` ORDER BY joined_at`, tourID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Participant, error) {
		var p Participant
		return p, r.Scan(&p.UserID, &p.JoinedAt, &p.ArrivedAt)
	})
}

// ParticipantIDs lists the accounts with a place, in sign-up order.
func (s *Store) ParticipantIDs(ctx context.Context, tourID string) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT user_id::text FROM tour_participants
		WHERE tour_id = $1 AND user_id IS NOT NULL AND `+activeSQL+` ORDER BY joined_at`, tourID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// WaitlistCount is how many wait for a place on the tour.
func (s *Store) WaitlistCount(ctx context.Context, tourID string) (int, error) {
	var n int
	err := s.db.QueryRow(ctx, `SELECT count(*)::int FROM tour_participants
		WHERE tour_id = $1 AND status = 'waitlist'`, tourID).Scan(&n)
	return n, err
}

// SetArrived marks a participant as arrived (at now) or not.
func (s *Store) SetArrived(ctx context.Context, tourID, userID string, arrived bool, now time.Time) error {
	var at *time.Time
	if arrived {
		at = &now
	}
	tag, err := s.db.Exec(ctx, `UPDATE tour_participants
		SET arrived_at = CASE WHEN $3::timestamptz IS NULL THEN NULL ELSE coalesce(arrived_at, $3) END
		WHERE tour_id = $1 AND user_id = $2 AND `+activeSQL, tourID, userID, at)
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
		WHERE tour_id = $1 AND arrived_at IS NOT NULL AND `+activeSQL+`)`, tourID).Scan(&marked)
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
