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

func scanReview(row pgx.Row, extra ...any) (models.Review, error) {
	var r models.Review
	var created time.Time
	var reply *string
	var repliedAt *time.Time
	err := row.Scan(append([]any{&r.ID, &r.Rating, &r.Text, &r.Photos, &r.UserID, &created, &reply, &repliedAt}, extra...)...)
	r.CreatedAt, r.Photos = jsontime.New(created), nonNil(r.Photos)
	if reply != nil && repliedAt != nil {
		r.Reply = &models.ReviewReply{Text: *reply, RepliedAt: jsontime.New(*repliedAt)}
	}
	return r, err
}

const reviewColumns = `id, rating, text, photos, user_id::text, created_at, reply, replied_at`

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

// ReviewTargetOwner returns the owner of the listing a review is about
// (published or not).
func (s *Store) ReviewTargetOwner(ctx context.Context, reviewID string) (*string, error) {
	var targetType, targetID string
	err := s.db.QueryRow(ctx, `SELECT target_type, target_id::text FROM reviews WHERE id = $1`, reviewID).
		Scan(&targetType, &targetID)
	if err != nil {
		return nil, notFound(err)
	}
	var owner *string
	err = s.db.QueryRow(ctx, `SELECT user_id::text FROM `+targetTables[targetType]+` WHERE id = $1`, targetID).Scan(&owner)
	return owner, notFound(err)
}

// SetReviewReply stores the owner's answer to a review; nil removes it.
func (s *Store) SetReviewReply(ctx context.Context, id string, reply *string) (models.Review, error) {
	r, err := scanReview(s.db.QueryRow(ctx, `UPDATE reviews SET reply = $2,
		replied_at = CASE WHEN $2::text IS NULL THEN NULL ELSE now() END
		WHERE id = $1 RETURNING `+reviewColumns, id, reply))
	return r, notFound(err)
}

// ReviewsOfOwner lists reviews of a user's listings, newest first;
// unanswered keeps only those without a reply.
func (s *Store) ReviewsOfOwner(ctx context.Context, userID string, unanswered bool, limit, offset int) ([]models.PartnerReview, int, error) {
	const from = `FROM reviews r JOIN (` + ownedListingsSQL + `) l
		ON l.target_type = r.target_type AND l.id = r.target_id
		WHERE ($2 = false OR r.reply IS NULL)`
	rows, err := s.db.Query(ctx, `SELECT r.id, r.rating, r.text, r.photos, r.user_id::text, r.created_at,
		r.reply, r.replied_at, l.target_type, l.id::text, l.title, l.slug, l.locale, l.is_published `+from+`
		ORDER BY r.created_at DESC LIMIT $3 OFFSET $4`, userID, unanswered, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.PartnerReview, error) {
		var pr models.PartnerReview
		l := &pr.Listing
		var err error
		pr.Review, err = scanReview(row, &l.TargetType, &l.ID, &l.Title, &l.Slug, &l.Locale, &l.IsPublished)
		return pr, err
	})
	if err != nil {
		return nil, 0, err
	}
	var total int
	err = s.db.QueryRow(ctx, `SELECT count(*) `+from, userID, unanswered).Scan(&total)
	return items, total, err
}
