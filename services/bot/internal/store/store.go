// Package store is the bot's database: blog subscribers and the posts the
// blog feed has seen.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) IsBlogSubscriber(ctx context.Context, chatID int64) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM blog_subscribers WHERE chat_id = $1)`, chatID).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("blog subscriber: %w", err)
	}
	return ok, nil
}

// SetBlogSubscriber subscribes or unsubscribes a chat (idempotent).
func (s *Store) SetBlogSubscriber(ctx context.Context, chatID int64, on bool) error {
	query := `DELETE FROM blog_subscribers WHERE chat_id = $1`
	if on {
		query = `INSERT INTO blog_subscribers (chat_id) VALUES ($1) ON CONFLICT DO NOTHING`
	}
	if _, err := s.db.Exec(ctx, query, chatID); err != nil {
		return fmt.Errorf("set blog subscriber: %w", err)
	}
	return nil
}

func (s *Store) BlogSubscribers(ctx context.Context) ([]int64, error) {
	rows, err := s.db.Query(ctx, `SELECT chat_id FROM blog_subscribers ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("blog subscribers: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowTo[int64])
}

// SeenPosts returns the posts the feed has seen; empty before its first pass.
func (s *Store) SeenPosts(ctx context.Context) (map[string]bool, error) {
	rows, err := s.db.Query(ctx, `SELECT post_id FROM blog_seen`)
	if err != nil {
		return nil, fmt.Errorf("seen posts: %w", err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("seen posts: %w", err)
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		seen[id] = true
	}
	return seen, nil
}

func (s *Store) MarkSeen(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := s.db.Exec(ctx, `INSERT INTO blog_seen (post_id) SELECT unnest($1::text[]) ON CONFLICT DO NOTHING`, ids)
	if err != nil {
		return fmt.Errorf("mark seen: %w", err)
	}
	return nil
}
