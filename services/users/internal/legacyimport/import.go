// Package legacyimport copies accounts and support contacts out of the
// NestJS API's database into the users service's own database, once — on
// the first start, before serving. Ids are kept, so sessions and every
// user id stored by the other services stay valid. The NestJS tables are
// only read.
package legacyimport

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const importName = "nestjs-users"

func Run(ctx context.Context, db *pgxpool.Pool, legacyDSN string, log *slog.Logger) error {
	var done bool
	if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM legacy_imports WHERE name = $1)`, importName).Scan(&done); err != nil {
		return fmt.Errorf("check import marker: %w", err)
	}
	if done {
		return nil
	}

	legacy, err := pgx.Connect(ctx, legacyDSN)
	if err != nil {
		return fmt.Errorf("connect to legacy database: %w", err)
	}
	defer func() { _ = legacy.Close(ctx) }()

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, s := range steps {
		n, err := s.run(ctx, legacy, tx)
		if err != nil {
			return fmt.Errorf("import %s: %w", s.table, err)
		}
		log.Info("imported legacy rows", slog.String("table", s.table), slog.Int("rows", n))
	}
	if _, err := tx.Exec(ctx, `INSERT INTO legacy_imports (name) VALUES ($1)`, importName); err != nil {
		return fmt.Errorf("record import: %w", err)
	}
	return tx.Commit(ctx)
}

type step struct{ table, query, insert string }

var steps = []step{
	{
		table: "users",
		query: `SELECT id, "telegramId", email, "passwordHash", username, "firstName", "lastName", phone,
			"avatarUrl", "isVerified", "verificationCodeHash", "verificationCodeExpiresAt",
			"verificationCodeSentAt", "verificationAttempts", "partnerStatus"::text, "partnerRequestedAt",
			"createdAt", "updatedAt" FROM users`,
		insert: `INSERT INTO users (id, telegram_id, email, password_hash, username, first_name, last_name,
			phone, avatar_url, is_verified, verification_code_hash, verification_code_expires_at,
			verification_code_sent_at, verification_attempts, partner_status, partner_requested_at,
			created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18) ON CONFLICT DO NOTHING`,
	},
	{
		table: "support_info",
		query: `SELECT id, message, phone, "telegramContact", email, "workingHours", "updatedAt" FROM support_info`,
		insert: `INSERT INTO support_info (id, message, phone, telegram_contact, email, working_hours, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`,
	},
}

func (s step) run(ctx context.Context, legacy *pgx.Conn, tx pgx.Tx) (int, error) {
	var exists bool
	if err := legacy.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+s.table).Scan(&exists); err != nil {
		return 0, err
	}
	if !exists {
		return 0, nil
	}
	rows, err := legacy.Query(ctx, s.query)
	if err != nil {
		return 0, err
	}
	var all [][]any
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			rows.Close()
			return 0, err
		}
		all = append(all, values)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, values := range all {
		if _, err := tx.Exec(ctx, s.insert, values...); err != nil {
			return 0, err
		}
	}
	return len(all), nil
}
