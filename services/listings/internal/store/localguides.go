package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/listings/internal/models"
)

const localGuideColumns = `id, slug, name, headline, bio, avatar_url, cover_image_url, photos, languages,
	specializations, destinations, price_per_day, experience_years, has_vehicle, phone, whatsapp,
	telegram, rating_average, review_count, is_published, user_id::text, created_at, updated_at`

func scanLocalGuide(row pgx.Row) (models.LocalGuide, error) {
	var g models.LocalGuide
	var created, updated time.Time
	err := row.Scan(&g.ID, &g.Slug, &g.Name, &g.Headline, &g.Bio, &g.AvatarURL, &g.CoverImageURL,
		&g.Photos, &g.Languages, &g.Specializations, &g.Destinations, &g.PricePerDay, &g.ExperienceYears,
		&g.HasVehicle, &g.Phone, &g.Whatsapp, &g.Telegram, &g.RatingAverage, &g.ReviewCount,
		&g.IsPublished, &g.UserID, &created, &updated)
	g.CreatedAt, g.UpdatedAt = times(created, updated)
	g.Photos, g.Languages = nonNil(g.Photos), nonNil(g.Languages)
	g.Specializations, g.Destinations = nonNil(g.Specializations), nonNil(g.Destinations)
	return g, err
}

func (s *Store) collectLocalGuides(ctx context.Context, query string, args ...any) ([]models.LocalGuide, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.LocalGuide, error) { return scanLocalGuide(r) })
}

type LocalGuideFilter struct {
	Destination, Language, Specialization, Search *string
	MaxPrice                                      *int
	MinRating                                     *float64
	HasVehicle                                    bool
	Sort                                          string
	Limit, Offset                                 int
}

func (s *Store) PublishedLocalGuides(ctx context.Context, f LocalGuideFilter) (models.Page[models.LocalGuide], error) {
	w := &where{}
	w.addRaw("is_published")
	if f.Destination != nil {
		w.add("? = ANY(destinations)", *f.Destination)
	}
	if f.Language != nil {
		w.add("? = ANY(languages)", *f.Language)
	}
	if f.Specialization != nil {
		w.add("? = ANY(specializations)", *f.Specialization)
	}
	if f.MaxPrice != nil {
		w.add("price_per_day <= ?", *f.MaxPrice)
	}
	if f.MinRating != nil {
		w.add("rating_average >= ?", *f.MinRating)
	}
	if f.HasVehicle {
		w.addRaw("has_vehicle")
	}
	if f.Search != nil && *f.Search != "" {
		w.add("(name ILIKE ? OR headline ILIKE ?)", "%"+*f.Search+"%")
	}

	total, err := s.count(ctx, "local_guides", w)
	if err != nil {
		return models.Page[models.LocalGuide]{}, err
	}
	query := `SELECT ` + localGuideColumns + ` FROM local_guides WHERE ` + w.sql() + ` ORDER BY ` + orderBy(f.Sort, "price_per_day")
	items, err := s.collectLocalGuides(ctx, query+w.page(f.Limit, f.Offset), w.args...)
	return models.Page[models.LocalGuide]{Items: items, Total: total}, err
}

// LocalGuideByUser returns the user's profile (each user has at most one).
func (s *Store) LocalGuideByUser(ctx context.Context, userID string) (models.LocalGuide, error) {
	g, err := scanLocalGuide(s.db.QueryRow(ctx, `SELECT `+localGuideColumns+` FROM local_guides WHERE user_id = $1`, userID))
	return g, notFound(err)
}

func (s *Store) PublishedLocalGuideBySlug(ctx context.Context, slug string) (models.LocalGuide, error) {
	g, err := scanLocalGuide(s.db.QueryRow(ctx, `SELECT `+localGuideColumns+` FROM local_guides WHERE slug = $1 AND is_published`, slug))
	return g, notFound(err)
}

func (s *Store) LocalGuide(ctx context.Context, id string) (models.LocalGuide, error) {
	g, err := scanLocalGuide(s.db.QueryRow(ctx, `SELECT `+localGuideColumns+` FROM local_guides WHERE id = $1`, id))
	return g, notFound(err)
}

func (s *Store) LocalGuideSlugTaken(ctx context.Context, slug, excludeID string) (bool, error) {
	var taken bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM local_guides WHERE slug = $1 AND ($2 = '' OR id::text <> $2))`,
		slug, excludeID).Scan(&taken)
	return taken, err
}

func (s *Store) SaveLocalGuide(ctx context.Context, g models.LocalGuide) (models.LocalGuide, error) {
	args := []any{g.Slug, g.Name, g.Headline, g.Bio, g.AvatarURL, g.CoverImageURL, nonNil(g.Photos),
		nonNil(g.Languages), nonNil(g.Specializations), nonNil(g.Destinations), g.PricePerDay,
		g.ExperienceYears, g.HasVehicle, g.Phone, g.Whatsapp, g.Telegram}
	var row pgx.Row
	if g.ID == "" {
		row = s.db.QueryRow(ctx, `INSERT INTO local_guides (slug, name, headline, bio, avatar_url,
			cover_image_url, photos, languages, specializations, destinations, price_per_day,
			experience_years, has_vehicle, phone, whatsapp, telegram, is_published, user_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,true,$17)
			RETURNING `+localGuideColumns, append(args, g.UserID)...)
	} else {
		row = s.db.QueryRow(ctx, `UPDATE local_guides SET slug=$1, name=$2, headline=$3, bio=$4,
			avatar_url=$5, cover_image_url=$6, photos=$7, languages=$8, specializations=$9,
			destinations=$10, price_per_day=$11, experience_years=$12, has_vehicle=$13, phone=$14,
			whatsapp=$15, telegram=$16, updated_at=now()
			WHERE id=$17 RETURNING `+localGuideColumns, append(args, g.ID)...)
	}
	saved, err := scanLocalGuide(row)
	return saved, duplicate(notFound(err))
}
