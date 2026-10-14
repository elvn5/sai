package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// targetTables maps a review/post target type to its table. Values are
// fixed, never taken from input.
var targetTables = map[string]string{
	"hotel":       "hotels",
	"local_guide": "local_guides",
	"tour":        "tours",
}

// TargetOwner returns the owner of a published listing, or
// ErrTargetMissing.
func (s *Store) TargetOwner(ctx context.Context, targetType, id string) (*string, error) {
	table, ok := targetTables[targetType]
	if !ok {
		return nil, ErrTargetMissing
	}
	var owner *string
	err := s.db.QueryRow(ctx, fmt.Sprintf(`SELECT user_id::text FROM %s WHERE id = $1 AND is_published`, table), id).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTargetMissing
	}
	return owner, err
}

// RefreshRating recomputes a listing's denormalized rating from its reviews.
func (s *Store) RefreshRating(ctx context.Context, targetType, id string) error {
	table := targetTables[targetType]
	_, err := s.db.Exec(ctx, fmt.Sprintf(`UPDATE %s SET
		rating_average = COALESCE((SELECT round(avg(rating)::numeric, 2) FROM reviews WHERE target_type = $1 AND target_id = $2), 0),
		review_count = (SELECT count(*) FROM reviews WHERE target_type = $1 AND target_id = $2)
		WHERE id = $2`, table), targetType, id)
	return err
}

// DeleteListing removes a listing with its reviews and posts (they
// reference it polymorphically, so there's no cascade).
func (s *Store) DeleteListing(ctx context.Context, targetType, id string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, q := range []string{
		`DELETE FROM reviews WHERE target_type = $1 AND target_id = $2`,
		`DELETE FROM listing_posts WHERE target_type = $1 AND target_id = $2`,
		`DELETE FROM listing_stats_daily WHERE target_type = $1 AND target_id = $2`,
	} {
		if _, err := tx.Exec(ctx, q, targetType, id); err != nil {
			return err
		}
	}
	tag, err := tx.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, targetTables[targetType]), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}
