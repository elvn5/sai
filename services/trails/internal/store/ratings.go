package store

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/trails/internal/models"
)

// RateOuting stores (or changes) userID's rating of an outing.
func (s *Store) RateOuting(ctx context.Context, o models.Outing, userID string, rating int, text *string) error {
	_, err := s.db.Exec(ctx, `INSERT INTO outing_ratings (outing_id, user_id, organizer_id, rating, text)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (outing_id, user_id) DO UPDATE SET rating = $4, text = $5, updated_at = now()`,
		o.ID, userID, o.UserID, rating, text)
	return err
}

// MyRating is userID's rating of an outing, nil if none.
func (s *Store) MyRating(ctx context.Context, outingID, userID string) (*models.RatingMine, error) {
	var r models.RatingMine
	err := s.db.QueryRow(ctx, `SELECT rating, text FROM outing_ratings WHERE outing_id::text = $1 AND user_id::text = $2`,
		outingID, userID).Scan(&r.Rating, &r.Text)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// AnyArrived: the organizer marked at least one arrival.
func (s *Store) AnyArrived(ctx context.Context, outingID string) (bool, error) {
	var marked bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM outing_participants
		WHERE outing_id::text = $1 AND arrived_at IS NOT NULL)`, outingID).Scan(&marked)
	return marked, err
}

// Arrived: userID was marked at the meeting point.
func (s *Store) Arrived(ctx context.Context, outingID, userID string) (bool, error) {
	var arrived bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM outing_participants
		WHERE outing_id::text = $1 AND user_id::text = $2 AND arrived_at IS NOT NULL)`, outingID, userID).Scan(&arrived)
	return arrived, err
}

// OrganizerOutings adds up an organizer's held outings (open, started),
// their participants and ratings, with the upcoming outings and the latest
// reviews (authors left for the handler).
func (s *Store) OrganizerOutings(ctx context.Context, userID string, reviews int, now time.Time) (models.OrganizerOutings, error) {
	var out models.OrganizerOutings
	err := s.db.QueryRow(ctx, `SELECT count(*)::int, coalesce(sum(
			(SELECT count(*) FROM outing_participants p WHERE p.outing_id = o.id)), 0)::int
		FROM outings o WHERE o.user_id::text = $1 AND o.status = 'open' AND o.starts_at <= $2`, userID, now).
		Scan(&out.Hosted, &out.Participants)
	if err != nil {
		return out, err
	}
	var avg *float64
	if err := s.db.QueryRow(ctx, `SELECT avg(rating)::float8, count(*)::int FROM outing_ratings
		WHERE organizer_id::text = $1`, userID).Scan(&avg, &out.Rating.Count); err != nil {
		return out, err
	}
	if avg != nil {
		out.Rating.Average = math.Round(*avg*100) / 100
	}
	out.Upcoming, err = s.collectOutings(ctx, `SELECT `+outingColumns+` WHERE o.user_id::text = $1
		AND o.status = 'open' AND o.starts_at > $2::timestamptz - interval '3 hours' ORDER BY o.starts_at LIMIT 20`, userID, now)
	if err != nil {
		return out, err
	}
	rows, err := s.db.Query(ctx, `SELECT r.rating, r.text, r.created_at, r.user_id::text, o.id::text, o.title, o.starts_at
		FROM outing_ratings r JOIN outings o ON o.id = r.outing_id
		WHERE r.organizer_id::text = $1 ORDER BY r.created_at DESC LIMIT $2`, userID, reviews)
	if err != nil {
		return out, err
	}
	out.Reviews, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.OutingReview, error) {
		var r models.OutingReview
		var created, starts time.Time
		err := row.Scan(&r.Rating, &r.Text, &created, &r.UserID, &r.Outing.ID, &r.Outing.Title, &starts)
		r.CreatedAt, r.Outing.StartsAt = jsontime.New(created), jsontime.New(starts)
		return r, err
	})
	return out, err
}
