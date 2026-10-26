package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Notification is a queued Telegram message for one user.
type Notification struct {
	ID         string
	UserID     string
	Kind       string
	TourID     *string
	Text       string
	ButtonText *string
	// Callback data the bot answers.
	ButtonData *string
	DedupKey   *string
	Attempts   int
}

// Enqueue adds notifications, due from sendAfter; ones whose dedup key is
// already queued are dropped.
func (s *Store) Enqueue(ctx context.Context, items []Notification, sendAfter time.Time) error {
	if len(items) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, n := range items {
		batch.Queue(`INSERT INTO notifications (user_id, kind, tour_id, text, button_text, button_data, dedup_key, send_after)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT (dedup_key) DO NOTHING`,
			n.UserID, n.Kind, n.TourID, n.Text, n.ButtonText, n.ButtonData, n.DedupKey, sendAfter)
	}
	return s.db.SendBatch(ctx, batch).Close()
}

// DueNotifications returns up to limit pending notifications ready to send.
func (s *Store) DueNotifications(ctx context.Context, now time.Time, limit int) ([]Notification, error) {
	rows, err := s.db.Query(ctx, `SELECT id::text, user_id::text, kind, tour_id::text, text, button_text,
		button_data, dedup_key, attempts FROM notifications
		WHERE status = 'pending' AND send_after <= $1 ORDER BY created_at LIMIT $2`, now, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Notification, error) {
		var n Notification
		err := r.Scan(&n.ID, &n.UserID, &n.Kind, &n.TourID, &n.Text, &n.ButtonText, &n.ButtonData, &n.DedupKey, &n.Attempts)
		return n, err
	})
}

// MarkNotification records a delivery outcome: status "sent", "skipped",
// "failed", or "pending" with a later sendAfter for a retry.
func (s *Store) MarkNotification(ctx context.Context, id, status string, attempts int, lastError *string, sendAfter time.Time) error {
	_, err := s.db.Exec(ctx, `UPDATE notifications SET status = $2, attempts = $3, last_error = $4, send_after = $5,
		sent_at = CASE WHEN $2 = 'sent' THEN now() END WHERE id::text = $1`, id, status, attempts, lastError, sendAfter)
	return err
}

// ToursDueForReminder: published, not-yet-started tours with someone
// signed up whose reminder is due and unsent.
func (s *Store) ToursDueForReminder(ctx context.Context, now time.Time) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT id::text FROM tours WHERE is_published AND reminder_sent_at IS NULL
		AND remind_at <= $1 AND remind_at + interval '12 hours' > $1
		AND EXISTS (SELECT 1 FROM tour_participants p WHERE p.tour_id = tours.id)`, now)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// ToursDueForFollowUp: published tours with someone signed up whose
// follow-up is due and unsent; ones that became due before staleBefore are
// skipped (too late to ask).
func (s *Store) ToursDueForFollowUp(ctx context.Context, now, staleBefore time.Time) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT id::text FROM tours WHERE is_published AND followup_sent_at IS NULL
		AND followup_at <= $1 AND followup_at > $2
		AND EXISTS (SELECT 1 FROM tour_participants p WHERE p.tour_id = tours.id)`, now, staleBefore)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (s *Store) MarkReminderSent(ctx context.Context, tourID string) error {
	_, err := s.db.Exec(ctx, `UPDATE tours SET reminder_sent_at = now() WHERE id::text = $1`, tourID)
	return err
}

func (s *Store) MarkFollowUpSent(ctx context.Context, tourID string) error {
	_, err := s.db.Exec(ctx, `UPDATE tours SET followup_sent_at = now() WHERE id::text = $1`, tourID)
	return err
}
