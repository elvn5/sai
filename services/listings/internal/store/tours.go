package store

import (
	"context"
	"math"
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
	is_published, published_at, user_id::text, created_at, updated_at, photos, program, capacity,
	meeting_point, meeting_time, (SELECT count(*)::int FROM tour_participants p WHERE p.tour_id = tours.id),
	remind_at, followup_at`

func scanTour(row pgx.Row) (models.Tour, error) {
	var t models.Tour
	var publishedAt *time.Time
	var created, updated time.Time
	err := row.Scan(&t.ID, &t.Locale, &t.Slug, &t.Title, &t.Excerpt, &t.Content, &t.Price,
		&t.DurationDays, &t.Location, &t.StartDate, &t.EndDate, &t.Destination, &t.CancellationPolicy,
		&t.CoverImageURL, &t.Phone, &t.Whatsapp, &t.Telegram, &t.RatingAverage, &t.ReviewCount,
		&t.IsPublished, &publishedAt, &t.UserID, &created, &updated, &t.Photos, &t.Program, &t.Capacity,
		&t.MeetingPoint, &t.MeetingTime, &t.Taken, &t.RemindAt, &t.FollowUpAt)
	t.PublishedAt = jsontime.Ptr(publishedAt)
	t.Photos = nonNil(t.Photos)
	if t.Program == nil {
		t.Program = []models.ProgramDay{}
	}
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

// TourFilter narrows the public list: a destination, and departures from
// / until a date (YYYY-MM-DD, inclusive).
type TourFilter struct {
	Destination, From, To *string
}

func (s *Store) PublishedTours(ctx context.Context, locale string, f TourFilter) ([]models.Tour, error) {
	return s.collectTours(ctx, `SELECT `+tourColumns+` FROM tours
		WHERE is_published AND locale = $1 AND ($2::text IS NULL OR destination = $2)
		AND ($3::date IS NULL OR start_date >= $3::date) AND ($4::date IS NULL OR start_date <= $4::date)
		ORDER BY published_at DESC, created_at DESC`, locale, f.Destination, f.From, f.To)
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
	program := t.Program
	if program == nil {
		program = []models.ProgramDay{}
	}
	args := []any{t.Locale, t.Slug, t.Title, t.Excerpt, t.Content, t.Price, t.DurationDays, t.Location,
		t.StartDate, t.EndDate, t.Destination, t.CancellationPolicy, t.CoverImageURL, t.Phone, t.Whatsapp, t.Telegram,
		nonNil(t.Photos), program, t.Capacity, t.MeetingPoint, t.MeetingTime, t.RemindAt, t.FollowUpAt}
	var row pgx.Row
	if t.ID == "" {
		row = s.db.QueryRow(ctx, `INSERT INTO tours (locale, slug, title, excerpt, content, price,
			duration_days, location, start_date, end_date, destination, cancellation_policy,
			cover_image_url, phone, whatsapp, telegram, photos, program, capacity, meeting_point, meeting_time,
			remind_at, followup_at, is_published, published_at, user_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::date,$10::date,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,
			true,now(),$24)
			RETURNING `+tourColumns, append(args, t.UserID)...)
	} else {
		// A moved tour gets its reminder and follow-up again.
		row = s.db.QueryRow(ctx, `UPDATE tours SET locale=$1, slug=$2, title=$3, excerpt=$4, content=$5,
			price=$6, duration_days=$7, location=$8, start_date=$9::date, end_date=$10::date,
			destination=$11, cancellation_policy=$12, cover_image_url=$13, phone=$14, whatsapp=$15,
			telegram=$16, photos=$17, program=$18, capacity=$19, meeting_point=$20, meeting_time=$21,
			reminder_sent_at = CASE WHEN remind_at IS NOT DISTINCT FROM $22 THEN reminder_sent_at END,
			followup_sent_at = CASE WHEN followup_at IS NOT DISTINCT FROM $23 THEN followup_sent_at END,
			remind_at=$22, followup_at=$23, updated_at=now()
			WHERE id=$24 RETURNING `+tourColumns, append(args, t.ID)...)
	}
	saved, err := scanTour(row)
	return saved, duplicate(notFound(err))
}

// OrganizerTours is what an organizer's public page shows about their
// tours in a language: how many already took place, the upcoming ones, the
// rating of all their tours and the latest reviews (authors left for the
// handler).
func (s *Store) OrganizerTours(ctx context.Context, userID, locale string, today time.Time, reviews int) (models.OrganizerTours, error) {
	var out models.OrganizerTours
	day := today.Format(time.DateOnly)
	err := s.db.QueryRow(ctx, `SELECT count(*)::int, coalesce(sum((SELECT count(*) FROM tour_participants p
		WHERE p.tour_id = tours.id)), 0)::int FROM tours WHERE user_id = $1 AND is_published
		AND locale = $2 AND start_date <= $3::date`, userID, locale, day).Scan(&out.Held, &out.Participants)
	if err != nil {
		return out, err
	}
	out.Upcoming, err = s.collectTours(ctx, `SELECT `+tourColumns+` FROM tours WHERE user_id = $1 AND is_published
		AND locale = $2 AND start_date >= $3::date ORDER BY start_date LIMIT 20`, userID, locale, day)
	if err != nil {
		return out, err
	}
	var avg *float64
	if err := s.db.QueryRow(ctx, `SELECT avg(r.rating)::float8, count(*)::int FROM reviews r
		JOIN tours t ON r.target_type = 'tour' AND t.id = r.target_id WHERE t.user_id = $1`, userID).
		Scan(&avg, &out.Rating.Count); err != nil {
		return out, err
	}
	if avg != nil {
		out.Rating.Average = math.Round(*avg*100) / 100
	}
	rows, err := s.db.Query(ctx, `SELECT r.id, r.rating, r.text, r.photos, r.user_id::text, r.created_at,
		r.reply, r.replied_at, 'tour', t.id::text, t.title, t.slug, t.locale, t.is_published
		FROM reviews r JOIN tours t ON r.target_type = 'tour' AND t.id = r.target_id
		WHERE t.user_id = $1 AND t.is_published ORDER BY r.created_at DESC LIMIT $2`, userID, reviews)
	if err != nil {
		return out, err
	}
	out.Reviews, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.PartnerReview, error) {
		var pr models.PartnerReview
		l := &pr.Listing
		var err error
		pr.Review, err = scanReview(row, &l.TargetType, &l.ID, &l.Title, &l.Slug, &l.Locale, &l.IsPublished)
		return pr, err
	})
	return out, err
}
