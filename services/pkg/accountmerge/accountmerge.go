// Package accountmerge moves what one account owns onto another when the
// users service merges two accounts of the same person (a Telegram-only
// account into the one with their confirmed email or phone). The users
// service records each merge; every other service polls the list and
// rewrites its own references.
package accountmerge

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Columns are the names of the uuid columns, in any table of a service's
// database, that hold an account's id. A new table referencing users
// needs one of these names to follow merges.
var Columns = []string{"user_id", "partner_id", "driver_id", "organizer_id", "sender_id", "awarded_by"}

// Merge is one account (From) merged into another (To). IDs grow.
type Merge struct {
	ID       int64
	From, To string
}

// Source lists the merges recorded after a given id, oldest first (the
// users service).
type Source interface {
	AccountMerges(ctx context.Context, after int64) ([]Merge, error)
}

// Interval is how often Run asks for new merges.
var Interval = 10 * time.Second

// Reassign moves every reference to from onto to, inside tx. A row that
// can't move because to already has its twin (both accounts favorited the
// same tour, joined the same group…) is dropped: to's own row wins.
// Tables listed in skip are left alone. Reports the rows moved or dropped
// per table.column.
func Reassign(ctx context.Context, tx pgx.Tx, from, to string, skip ...string) (map[string]int64, error) {
	rows, err := tx.Query(ctx, `SELECT table_name, column_name FROM information_schema.columns
		WHERE table_schema = current_schema() AND data_type = 'uuid' AND column_name = ANY($1)
			AND NOT (table_name = ANY($2))
			AND table_name IN (SELECT table_name FROM information_schema.tables
				WHERE table_schema = current_schema() AND table_type = 'BASE TABLE')
		ORDER BY table_name, column_name`, Columns, append(skip, ""))
	if err != nil {
		return nil, err
	}
	type column struct{ table, name string }
	columns, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (column, error) {
		var c column
		return c, r.Scan(&c.table, &c.name)
	})
	if err != nil {
		return nil, err
	}

	moved := map[string]int64{}
	for _, c := range columns {
		n, err := reassignColumn(ctx, tx, pgx.Identifier{c.table}.Sanitize(), pgx.Identifier{c.name}.Sanitize(), from, to)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", c.table, c.name, err)
		}
		if n > 0 {
			moved[c.table+"."+c.name] = n
		}
	}
	return moved, nil
}

// reassignColumn moves the whole column at once, or row by row when a
// unique key is in the way.
func reassignColumn(ctx context.Context, tx pgx.Tx, table, column, from, to string) (int64, error) {
	update := `UPDATE ` + table + ` SET ` + column + ` = $2 WHERE ` + column + ` = $1`
	n, err := savepoint(ctx, tx, func() (int64, error) {
		tag, err := tx.Exec(ctx, update, from, to)
		return tag.RowsAffected(), err
	})
	if !isUniqueViolation(err) {
		return n, err
	}

	rows, err := tx.Query(ctx, `SELECT ctid::text FROM `+table+` WHERE `+column+` = $1`, from)
	if err != nil {
		return 0, err
	}
	ctids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return 0, err
	}
	for _, ctid := range ctids {
		_, err := savepoint(ctx, tx, func() (int64, error) {
			tag, err := tx.Exec(ctx, `UPDATE `+table+` SET `+column+` = $2 WHERE ctid = $1::tid`, ctid, to)
			return tag.RowsAffected(), err
		})
		if isUniqueViolation(err) {
			_, err = tx.Exec(ctx, `DELETE FROM `+table+` WHERE ctid = $1::tid`, ctid)
		}
		if err != nil {
			return 0, err
		}
	}
	return int64(len(ctids)), nil
}

// savepoint runs fn so that its failure doesn't abort tx.
func savepoint(ctx context.Context, tx pgx.Tx, fn func() (int64, error)) (int64, error) {
	if _, err := tx.Exec(ctx, `SAVEPOINT account_merge`); err != nil {
		return 0, err
	}
	n, err := fn()
	if err != nil {
		if _, rbErr := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT account_merge`); rbErr != nil {
			return 0, rbErr
		}
		return 0, err
	}
	_, err = tx.Exec(ctx, `RELEASE SAVEPOINT account_merge`)
	return n, err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Run applies the users service's merges to db as they come, until ctx
// ends. Applied merges are kept in account_merges_applied, so each is
// applied once.
func Run(ctx context.Context, db *pgxpool.Pool, src Source, log *slog.Logger) {
	if _, err := db.Exec(ctx, `CREATE TABLE IF NOT EXISTS account_merges_applied (
		merge_id bigint PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		log.ErrorContext(ctx, "account merges: can't create the table", slog.Any("error", err))
		return
	}
	for {
		if err := applyNew(ctx, db, src, log); err != nil && ctx.Err() == nil {
			log.WarnContext(ctx, "account merges: not applied; retrying", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(Interval):
		}
	}
}

func applyNew(ctx context.Context, db *pgxpool.Pool, src Source, log *slog.Logger) error {
	var last int64
	if err := db.QueryRow(ctx, `SELECT coalesce(max(merge_id), 0) FROM account_merges_applied`).Scan(&last); err != nil {
		return err
	}
	merges, err := src.AccountMerges(ctx, last)
	if err != nil {
		return err
	}
	for _, m := range merges {
		if err := Apply(ctx, db, m, log); err != nil {
			return err
		}
	}
	return nil
}

// Apply rewrites db's references for one merge and records it as applied.
func Apply(ctx context.Context, db *pgxpool.Pool, m Merge, log *slog.Logger) error {
	return pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		moved, err := Reassign(ctx, tx, m.From, m.To, "account_merges_applied")
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO account_merges_applied (merge_id) VALUES ($1)
			ON CONFLICT DO NOTHING`, m.ID); err != nil {
			return err
		}
		log.InfoContext(ctx, "account merge applied", slog.Int64("merge", m.ID),
			slog.String("from", m.From), slog.String("to", m.To), slog.Any("rows", moved))
		return nil
	})
}
