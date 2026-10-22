// Package store is the bot's database: blog subscribers, the posts the
// blog feed has seen and the language of each chat.
package store

import (
	"context"
	"errors"
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

// Subscriber is a chat that gets new blog posts, in its language.
type Subscriber struct {
	ChatID int64
	Lang   string
}

// BlogSubscribers lists the subscribed chats with their languages
// (Russian for a chat that has none yet).
func (s *Store) BlogSubscribers(ctx context.Context) ([]Subscriber, error) {
	rows, err := s.db.Query(ctx, `
		SELECT b.chat_id, COALESCE(l.lang, 'ru') FROM blog_subscribers b
		LEFT JOIN chat_languages l ON l.chat_id = b.chat_id
		ORDER BY b.created_at`)
	if err != nil {
		return nil, fmt.Errorf("blog subscribers: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Subscriber])
}

// Language is the chat's language; a chat without one gets fallback (and
// keeps it).
func (s *Store) Language(ctx context.Context, chatID int64, fallback string) (string, error) {
	var lang string
	err := s.db.QueryRow(ctx, `
		INSERT INTO chat_languages (chat_id, lang) VALUES ($1, $2)
		ON CONFLICT (chat_id) DO UPDATE SET chat_id = EXCLUDED.chat_id
		RETURNING lang`, chatID, fallback).Scan(&lang)
	if err != nil {
		return "", fmt.Errorf("chat language: %w", err)
	}
	return lang, nil
}

// KnownLanguage is the chat's language, "" if it has none yet.
func (s *Store) KnownLanguage(ctx context.Context, chatID int64) (string, error) {
	var lang string
	err := s.db.QueryRow(ctx, `SELECT lang FROM chat_languages WHERE chat_id = $1`, chatID).Scan(&lang)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("chat language: %w", err)
	}
	return lang, nil
}

// SetLanguage keeps the language the user picked.
func (s *Store) SetLanguage(ctx context.Context, chatID int64, lang string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO chat_languages (chat_id, lang) VALUES ($1, $2)
		ON CONFLICT (chat_id) DO UPDATE SET lang = EXCLUDED.lang, updated_at = now()`, chatID, lang)
	if err != nil {
		return fmt.Errorf("set chat language: %w", err)
	}
	return nil
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
