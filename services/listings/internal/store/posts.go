package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/pkg/jsontime"
)

const postColumns = `id, text, photos, user_id::text, created_at`

func scanPost(row pgx.Row) (models.ListingPost, error) {
	var p models.ListingPost
	var created time.Time
	err := row.Scan(&p.ID, &p.Text, &p.Photos, &p.UserID, &created)
	p.CreatedAt, p.Photos = jsontime.New(created), nonNil(p.Photos)
	return p, err
}

func (s *Store) Posts(ctx context.Context, targetType, targetID string, limit, offset int) ([]models.ListingPost, int, error) {
	rows, err := s.db.Query(ctx, `SELECT `+postColumns+` FROM listing_posts
		WHERE target_type = $1 AND target_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		targetType, targetID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.ListingPost, error) { return scanPost(r) })
	if err != nil {
		return nil, 0, err
	}
	var total int
	err = s.db.QueryRow(ctx, `SELECT count(*) FROM listing_posts WHERE target_type = $1 AND target_id = $2`,
		targetType, targetID).Scan(&total)
	return items, total, err
}

func (s *Store) InsertPost(ctx context.Context, targetType, targetID, userID, text string, photos []string) (models.ListingPost, error) {
	return scanPost(s.db.QueryRow(ctx, `INSERT INTO listing_posts (target_type, target_id, text, photos, user_id)
		VALUES ($1, $2, $3, $4, $5) RETURNING `+postColumns, targetType, targetID, text, nonNil(photos), userID))
}

func (s *Store) PostOwner(ctx context.Context, id string) (string, error) {
	var userID string
	err := s.db.QueryRow(ctx, `SELECT user_id::text FROM listing_posts WHERE id = $1`, id).Scan(&userID)
	return userID, notFound(err)
}

func (s *Store) DeletePost(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM listing_posts WHERE id = $1`, id)
	return err
}
