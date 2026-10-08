// Package store reads accounts from the users table.
package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("user not found")

type User struct {
	ID            string
	FirstName     *string
	Username      *string
	AvatarURL     *string
	PartnerStatus string
}

type Store struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// The table and its camelCase columns are still defined by the NestJS
// (TypeORM) schema.
const userColumns = `id::text, "firstName", username, "avatarUrl", "partnerStatus"::text`

func scan(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.FirstName, &u.Username, &u.AvatarURL, &u.PartnerStatus)
	return u, err
}

func (s *Store) User(ctx context.Context, id string) (User, error) {
	u, err := scan(s.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id::text = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) Users(ctx context.Context, ids []string) ([]User, error) {
	rows, err := s.db.Query(ctx, `SELECT `+userColumns+` FROM users WHERE id::text = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (User, error) { return scan(row) })
}
