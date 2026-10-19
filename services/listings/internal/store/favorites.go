package store

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/listings/internal/models"
)

// AddFavorite saves a listing for a user (again is fine).
func (s *Store) AddFavorite(ctx context.Context, userID, targetType, targetID string) error {
	_, err := s.db.Exec(ctx, `INSERT INTO favorites (user_id, target_type, target_id) VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`, userID, targetType, targetID)
	return err
}

func (s *Store) RemoveFavorite(ctx context.Context, userID, targetType, targetID string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM favorites WHERE user_id = $1 AND target_type = $2 AND target_id = $3`,
		userID, targetType, targetID)
	return err
}

// FavoriteIDs lists the ids a user saved, by type (for the hearts on cards).
func (s *Store) FavoriteIDs(ctx context.Context, userID string) (map[string][]string, error) {
	rows, err := s.db.Query(ctx, `SELECT target_type, target_id::text FROM favorites WHERE user_id = $1
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	ids := map[string][]string{"hotel": {}, "local_guide": {}, "tour": {}}
	var targetType, id string
	_, err = pgx.ForEachRow(rows, []any{&targetType, &id}, func() error {
		ids[targetType] = append(ids[targetType], id)
		return nil
	})
	return ids, err
}

// FavoriteTours lists a user's saved published tours, latest saved first.
func (s *Store) FavoriteTours(ctx context.Context, userID string) ([]models.Tour, error) {
	return s.collectTours(ctx, `SELECT `+tourColumns+` FROM tours
		WHERE is_published AND id IN (SELECT target_id FROM favorites WHERE target_type = 'tour' AND user_id = $1)
		ORDER BY (SELECT f.created_at FROM favorites f
			WHERE f.target_type = 'tour' AND f.target_id = tours.id AND f.user_id = $1) DESC`, userID)
}

// TourAlerts lists the destinations a user follows (” for any).
func (s *Store) TourAlerts(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT destination FROM tour_alerts WHERE user_id = $1 ORDER BY destination`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (s *Store) AddTourAlert(ctx context.Context, userID, destination string) error {
	_, err := s.db.Exec(ctx, `INSERT INTO tour_alerts (user_id, destination) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, destination)
	return err
}

func (s *Store) RemoveTourAlert(ctx context.Context, userID, destination string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM tour_alerts WHERE user_id = $1 AND destination = $2`, userID, destination)
	return err
}

// TourAlertSubscribers lists who follows a new tour's destination or any.
func (s *Store) TourAlertSubscribers(ctx context.Context, destination *string) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT DISTINCT user_id::text FROM tour_alerts
		WHERE destination = '' OR destination = coalesce($1, '')`, destination)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// FavoriteHotels lists a user's saved published hotels, latest saved first.
func (s *Store) FavoriteHotels(ctx context.Context, userID string) ([]models.Hotel, error) {
	return s.collectHotels(ctx, `SELECT `+hotelColumns+` FROM hotels
		WHERE is_published AND id IN (SELECT target_id FROM favorites WHERE target_type = 'hotel' AND user_id = $1)
		ORDER BY (SELECT f.created_at FROM favorites f
			WHERE f.target_type = 'hotel' AND f.target_id = hotels.id AND f.user_id = $1) DESC`, userID)
}

// FavoriteLocalGuides lists a user's saved published guides, latest saved first.
func (s *Store) FavoriteLocalGuides(ctx context.Context, userID string) ([]models.LocalGuide, error) {
	return s.collectLocalGuides(ctx, `SELECT `+localGuideColumns+` FROM local_guides
		WHERE is_published AND id IN (SELECT target_id FROM favorites WHERE target_type = 'local_guide' AND user_id = $1)
		ORDER BY (SELECT f.created_at FROM favorites f
			WHERE f.target_type = 'local_guide' AND f.target_id = local_guides.id AND f.user_id = $1) DESC`, userID)
}
