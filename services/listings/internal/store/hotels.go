package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/listings/internal/models"
)

const hotelColumns = `id, slug, name, type, description, destination, address, latitude, longitude,
	price_from, stars, amenities, cancellation_policy, check_in_time, check_out_time, cover_image_url,
	photos, phone, whatsapp, telegram, website, rating_average, review_count, is_published,
	user_id::text, created_at, updated_at`

func scanHotel(row pgx.Row) (models.Hotel, error) {
	var h models.Hotel
	var created, updated time.Time
	err := row.Scan(&h.ID, &h.Slug, &h.Name, &h.Type, &h.Description, &h.Destination, &h.Address,
		&h.Latitude, &h.Longitude, &h.PriceFrom, &h.Stars, &h.Amenities, &h.CancellationPolicy,
		&h.CheckInTime, &h.CheckOutTime, &h.CoverImageURL, &h.Photos, &h.Phone, &h.Whatsapp,
		&h.Telegram, &h.Website, &h.RatingAverage, &h.ReviewCount, &h.IsPublished, &h.UserID,
		&created, &updated)
	h.CreatedAt, h.UpdatedAt = times(created, updated)
	h.Amenities, h.Photos = nonNil(h.Amenities), nonNil(h.Photos)
	return h, err
}

func (s *Store) collectHotels(ctx context.Context, query string, args ...any) ([]models.Hotel, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.Hotel, error) { return scanHotel(r) })
}

type HotelFilter struct {
	Destination, Type, Search *string
	MaxPrice                  *int
	MinRating                 *float64
	Amenities                 []string
	FreeCancellation          bool
	Sort                      string
	Limit, Offset             int
}

func (s *Store) PublishedHotels(ctx context.Context, f HotelFilter) (models.Page[models.Hotel], error) {
	w := &where{}
	w.addRaw("is_published")
	if f.Destination != nil {
		w.add("destination = ?", *f.Destination)
	}
	if f.Type != nil {
		w.add("type = ?", *f.Type)
	}
	if f.MaxPrice != nil {
		w.add("price_from <= ?", *f.MaxPrice)
	}
	if f.MinRating != nil {
		w.add("rating_average >= ?", *f.MinRating)
	}
	if f.FreeCancellation {
		w.add("cancellation_policy = ?", "free")
	}
	if f.Search != nil && *f.Search != "" {
		w.add("(name ILIKE ? OR address ILIKE ?)", "%"+*f.Search+"%")
	}
	if len(f.Amenities) > 0 {
		w.add("amenities @> ?::text[]", f.Amenities)
	}

	total, err := s.count(ctx, "hotels", w)
	if err != nil {
		return models.Page[models.Hotel]{}, err
	}
	query := `SELECT ` + hotelColumns + ` FROM hotels WHERE ` + w.sql() + ` ORDER BY ` + orderBy(f.Sort, "price_from")
	items, err := s.collectHotels(ctx, query+w.page(f.Limit, f.Offset), w.args...)
	return models.Page[models.Hotel]{Items: items, Total: total}, err
}

func (s *Store) HotelsByUser(ctx context.Context, userID string) ([]models.Hotel, error) {
	return s.collectHotels(ctx, `SELECT `+hotelColumns+` FROM hotels WHERE user_id = $1 ORDER BY created_at DESC`, userID)
}

func (s *Store) PublishedHotelBySlug(ctx context.Context, slug string) (models.Hotel, error) {
	h, err := scanHotel(s.db.QueryRow(ctx, `SELECT `+hotelColumns+` FROM hotels WHERE slug = $1 AND is_published`, slug))
	return h, notFound(err)
}

func (s *Store) Hotel(ctx context.Context, id string) (models.Hotel, error) {
	h, err := scanHotel(s.db.QueryRow(ctx, `SELECT `+hotelColumns+` FROM hotels WHERE id = $1`, id))
	return h, notFound(err)
}

func (s *Store) HotelSlugTaken(ctx context.Context, slug, excludeID string) (bool, error) {
	var taken bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM hotels WHERE slug = $1 AND ($2 = '' OR id::text <> $2))`,
		slug, excludeID).Scan(&taken)
	return taken, err
}

func (s *Store) SaveHotel(ctx context.Context, h models.Hotel) (models.Hotel, error) {
	args := []any{h.Slug, h.Name, h.Type, h.Description, h.Destination, h.Address, h.Latitude, h.Longitude,
		h.PriceFrom, h.Stars, nonNil(h.Amenities), h.CancellationPolicy, h.CheckInTime, h.CheckOutTime,
		h.CoverImageURL, nonNil(h.Photos), h.Phone, h.Whatsapp, h.Telegram, h.Website}
	var row pgx.Row
	if h.ID == "" {
		row = s.db.QueryRow(ctx, `INSERT INTO hotels (slug, name, type, description, destination, address,
			latitude, longitude, price_from, stars, amenities, cancellation_policy, check_in_time,
			check_out_time, cover_image_url, photos, phone, whatsapp, telegram, website, is_published, user_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,true,$21)
			RETURNING `+hotelColumns, append(args, h.UserID)...)
	} else {
		row = s.db.QueryRow(ctx, `UPDATE hotels SET slug=$1, name=$2, type=$3, description=$4,
			destination=$5, address=$6, latitude=$7, longitude=$8, price_from=$9, stars=$10, amenities=$11,
			cancellation_policy=$12, check_in_time=$13, check_out_time=$14, cover_image_url=$15, photos=$16,
			phone=$17, whatsapp=$18, telegram=$19, website=$20, updated_at=now()
			WHERE id=$21 RETURNING `+hotelColumns, append(args, h.ID)...)
	}
	saved, err := scanHotel(row)
	return saved, duplicate(notFound(err))
}
