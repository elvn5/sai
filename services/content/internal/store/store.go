// Package store reads and writes the content database.
package store

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elvn5/sai/services/pkg/jsontime"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrSlugTaken = errors.New("slug is already used for this locale")
)

type Store struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// notFound maps "no rows" to ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// uniqueViolation maps a (locale, slug) unique-constraint violation — a
// concurrent insert that slipped past the slug check — to ErrSlugTaken.
func uniqueViolation(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrSlugTaken
	}
	return err
}

func timePtr(t *time.Time) *jsontime.Time { return jsontime.Ptr(t) }
