// Package store reads and writes the listings database.
package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elvn5/sai/services/pkg/jsontime"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrDuplicate     = errors.New("duplicate")
	ErrTargetMissing = errors.New("listing not found")
)

type Store struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// duplicate maps a unique-constraint violation (a race past an explicit
// check) to ErrDuplicate.
func duplicate(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicate
	}
	return err
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func times(created, updated time.Time) (jsontime.Time, jsontime.Time) {
	return jsontime.New(created), jsontime.New(updated)
}

// where accumulates SQL conditions with numbered parameters.
type where struct {
	conds []string
	args  []any
}

func (w *where) add(cond string, arg any) {
	w.args = append(w.args, arg)
	w.conds = append(w.conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(w.args))))
}

func (w *where) addRaw(cond string) { w.conds = append(w.conds, cond) }

func (w *where) sql() string {
	if len(w.conds) == 0 {
		return "TRUE"
	}
	return strings.Join(w.conds, " AND ")
}

// page appends LIMIT/OFFSET placeholders.
func (w *where) page(limit, offset int) string {
	w.args = append(w.args, limit, offset)
	n := len(w.args)
	return fmt.Sprintf(" LIMIT $%d OFFSET $%d", n-1, n)
}

func (s *Store) count(ctx context.Context, table string, w *where) (int, error) {
	var n int
	err := s.db.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE `+w.sql(), w.args...).Scan(&n)
	return n, err
}

// orderBy maps a list sort to SQL; the default is by rating.
func orderBy(sort, priceColumn string) string {
	switch sort {
	case "price_asc":
		return priceColumn + " ASC NULLS LAST, created_at DESC"
	case "price_desc":
		return priceColumn + " DESC NULLS LAST, created_at DESC"
	case "experience":
		return "experience_years DESC NULLS LAST, created_at DESC"
	case "newest":
		return "created_at DESC"
	default:
		return "rating_average DESC, review_count DESC, created_at DESC"
	}
}
