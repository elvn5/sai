package store

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/pkg/jsontime"
)

// Reviews and posts come back without authors; handlers attach those from
// the users service.

func scanReview(row pgx.Row) (models.Review, error) {
	var r models.Review
	var created time.Time
	err := row.Scan(&r.ID, &r.Rating, &r.Text, &r.Photos, &r.UserID, &created)
	r.CreatedAt, r.Photos = jsontime.New(created), nonNil(r.Photos)
	return r, err
}

const reviewColumns = `id, rating, text, photos, user_id::text, created_at`

func (s *Store) Reviews(ctx context.Context, targetType, targetID string, limit, offset int) ([]models.Review, int, error) {
	rows, err := s.db.Query(ctx, `SELECT `+reviewColumns+` FROM reviews
		WHERE target_type = $1 AND target_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		targetType, targetID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.Review, error) { return scanReview(r) })
	if err != nil {
		return nil, 0, err
	}
	var total int
	err = s.db.QueryRow(ctx, `SELECT count(*) FROM reviews WHERE target_type = $1 AND target_id = $2`,
		targetType, targetID).Scan(&total)
	return items, total, err
}

func (s *Store) ReviewSummary(ctx context.Context, targetType, targetID string) (models.ReviewSummary, error) {
	rows, err := s.db.Query(ctx, `SELECT rating, count(*) FROM reviews
		WHERE target_type = $1 AND target_id = $2 GROUP BY rating`, targetType, targetID)
	if err != nil {
		return models.ReviewSummary{}, err
	}
	defer rows.Close()

	summary := models.ReviewSummary{Distribution: map[string]int{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0}}
	sum := 0
	for rows.Next() {
		var rating, n int
		if err := rows.Scan(&rating, &n); err != nil {
			return models.ReviewSummary{}, err
		}
		summary.Distribution[strconv.Itoa(rating)] = n
		summary.Count += n
		sum += rating * n
	}
	if summary.Count > 0 {
		summary.Average = math.Round(float64(sum)/float64(summary.Count)*100) / 100
	}
	return summary, rows.Err()
}

func (s *Store) HasReviewed(ctx context.Context, targetType, targetID, userID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM reviews
		WHERE target_type = $1 AND target_id = $2 AND user_id = $3)`, targetType, targetID, userID).Scan(&exists)
	return exists, err
}

func (s *Store) InsertReview(ctx context.Context, targetType, targetID, userID string, rating int, text string, photos []string) (models.Review, error) {
	r, err := scanReview(s.db.QueryRow(ctx, `INSERT INTO reviews (target_type, target_id, rating, text, photos, user_id)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+reviewColumns, targetType, targetID, rating, text, nonNil(photos), userID))
	return r, duplicate(err)
}

// ReviewOwner returns a review's author and target.
func (s *Store) ReviewOwner(ctx context.Context, id string) (userID, targetType, targetID string, err error) {
	err = s.db.QueryRow(ctx, `SELECT user_id::text, target_type, target_id::text FROM reviews WHERE id = $1`, id).
		Scan(&userID, &targetType, &targetID)
	return userID, targetType, targetID, notFound(err)
}

func (s *Store) DeleteReview(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM reviews WHERE id = $1`, id)
	return err
}
