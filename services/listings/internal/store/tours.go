package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/pkg/jsontime"
)

// Dates are rendered as YYYY-MM-DD strings, as TypeORM returned "date"
// columns.
const tourColumns = `id, locale, slug, title, excerpt, content, price, duration_days, location,
	to_char(start_date, 'YYYY-MM-DD'), to_char(end_date, 'YYYY-MM-DD'), destination,
	cancellation_policy, cover_image_url, phone, whatsapp, telegram, rating_average, review_count,
	is_published, published_at, user_id::text, created_at, updated_at`

func scanTour(row pgx.Row) (models.Tour, error) {
	var t models.Tour
	var publishedAt *time.Time
	var created, updated time.Time
	err := row.Scan(&t.ID, &t.Locale, &t.Slug, &t.Title, &t.Excerpt, &t.Content, &t.Price,
		&t.DurationDays, &t.Location, &t.StartDate, &t.EndDate, &t.Destination, &t.CancellationPolicy,
		&t.CoverImageURL, &t.Phone, &t.Whatsapp, &t.Telegram, &t.RatingAverage, &t.ReviewCount,
		&t.IsPublished, &publishedAt, &t.UserID, &created, &updated)
	t.PublishedAt = jsontime.Ptr(publishedAt)
	t.CreatedAt, t.UpdatedAt = times(created, updated)
	return t, err
}

func (s *Store) collectTours(ctx context.Context, query string, args ...any) ([]models.Tour, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.Tour, error) { return scanTour(r) })
}

func (s *Store) PublishedTours(ctx context.Context, locale string, destination *string) ([]models.Tour, error) {
	return s.collectTours(ctx, `SELECT `+tourColumns+` FROM tours
		WHERE is_published AND locale = $1 AND ($2::text IS NULL OR destination = $2)
		ORDER BY published_at DESC, created_at DESC`, locale, destination)
}

func (s *Store) ToursByUser(ctx context.Context, userID string) ([]models.Tour, error) {
	return s.collectTours(ctx, `SELECT `+tourColumns+` FROM tours WHERE user_id = $1 ORDER BY created_at DESC`, userID)
}

func (s *Store) PublishedTourBySlug(ctx context.Context, locale, slug string) (models.Tour, error) {
	t, err := scanTour(s.db.QueryRow(ctx, `SELECT `+tourColumns+` FROM tours
		WHERE locale = $1 AND slug = $2 AND is_published`, locale, slug))
	return t, notFound(err)
}

func (s *Store) Tour(ctx context.Context, id string) (models.Tour, error) {
	t, err := scanTour(s.db.QueryRow(ctx, `SELECT `+tourColumns+` FROM tours WHERE id = $1`, id))
	return t, notFound(err)
}

func (s *Store) TourSlugTaken(ctx context.Context, locale, slug, excludeID string) (bool, error) {
	var taken bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tours
		WHERE locale = $1 AND slug = $2 AND ($3 = '' OR id::text <> $3))`, locale, slug, excludeID).Scan(&taken)
	return taken, err
}

func (s *Store) SaveTour(ctx context.Context, t models.Tour) (models.Tour, error) {
	args := []any{t.Locale, t.Slug, t.Title, t.Excerpt, t.Content, t.Price, t.DurationDays, t.Location,
		t.StartDate, t.EndDate, t.Destination, t.CancellationPolicy, t.CoverImageURL, t.Phone, t.Whatsapp, t.Telegram}
	var row pgx.Row
	if t.ID == "" {
		row = s.db.QueryRow(ctx, `INSERT INTO tours (locale, slug, title, excerpt, content, price,
			duration_days, location, start_date, end_date, destination, cancellation_policy,
			cover_image_url, phone, whatsapp, telegram, is_published, published_at, user_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::date,$10::date,$11,$12,$13,$14,$15,$16,true,now(),$17)
			RETURNING `+tourColumns, append(args, t.UserID)...)
	} else {
		row = s.db.QueryRow(ctx, `UPDATE tours SET locale=$1, slug=$2, title=$3, excerpt=$4, content=$5,
			price=$6, duration_days=$7, location=$8, start_date=$9::date, end_date=$10::date,
			destination=$11, cancellation_policy=$12, cover_image_url=$13, phone=$14, whatsapp=$15,
			telegram=$16, updated_at=now()
			WHERE id=$17 RETURNING `+tourColumns, append(args, t.ID)...)
	}
	saved, err := scanTour(row)
	return saved, duplicate(notFound(err))
}
