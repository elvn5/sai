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

	"github.com/elvn5/sai/services/pkg/legacy"
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

	conn, ok, err := legacy.Connect(ctx, legacyDSN)
	if err != nil {
		return err
	}
	if !ok {
		log.Info("no legacy database; nothing to import")
		return nil
	}
	defer func() { _ = conn.Close(ctx) }()

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, s := range steps {
		n, err := s.run(ctx, conn, tx)
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

type step struct {
	table  string
	cols   []legacy.Column
	insert string
}

// Fallbacks are the defaults the NestJS migrations that added those
// columns applied, for databases still at an older NestJS release.
var steps = []step{
	{
		table: "users",
		cols: []legacy.Column{
			legacy.Col("id"), legacy.Col("telegramId"),
			{Name: "email", Fallback: "NULL"}, {Name: "passwordHash", Fallback: "NULL"},
			legacy.Col("username"), legacy.Col("firstName"), legacy.Col("lastName"), legacy.Col("phone"),
			{Name: "avatarUrl", Fallback: "NULL"},
			{Name: "isVerified", Fallback: "true"},
			{Name: "verificationCodeHash", Fallback: "NULL"},
			{Name: "verificationCodeExpiresAt", Fallback: "NULL::timestamp"},
			{Name: "verificationCodeSentAt", Fallback: "NULL::timestamp"},
			{Name: "verificationAttempts", Fallback: "0"},
			{Name: "partnerStatus", Cast: "::text", Fallback: "'none'"},
			{Name: "partnerRequestedAt", Fallback: "NULL::timestamp"},
			legacy.Col("createdAt"), legacy.Col("updatedAt"),
		},
		insert: `INSERT INTO users (id, telegram_id, email, password_hash, username, first_name, last_name,
			phone, avatar_url, is_verified, verification_code_hash, verification_code_expires_at,
			verification_code_sent_at, verification_attempts, partner_status, partner_requested_at,
			created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18) ON CONFLICT DO NOTHING`,
	},
	{
		table: "support_info",
		cols: []legacy.Column{
			legacy.Col("id"), legacy.Col("message"), legacy.Col("phone"), legacy.Col("telegramContact"),
			legacy.Col("email"), legacy.Col("workingHours"), legacy.Col("updatedAt"),
		},
		insert: `INSERT INTO support_info (id, message, phone, telegram_contact, email, working_hours, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`,
	},
}

func (s step) run(ctx context.Context, conn *pgx.Conn, tx pgx.Tx) (int, error) {
	exists, err := legacy.TableExists(ctx, conn, s.table)
	if err != nil || !exists {
		return 0, err
	}
	query, err := legacy.Select(ctx, conn, s.table, s.cols)
	if err != nil {
		return 0, err
	}
	rows, err := conn.Query(ctx, query)
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
