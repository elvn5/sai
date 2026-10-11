// Package partnerpurge removes, once, what accounts that aren't approved
// partners published before publishing became partner-only.
package partnerpurge

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Marker is the legacy_imports row recording that the purge ran.
const Marker = "non-partner-content-purge"

// StatusSource reports partner statuses (the users service); unknown ids
// are absent from the result.
type StatusSource interface {
	PartnerStatuses(ctx context.Context, ids []string) (map[string]string, error)
}

// Service is the part each service supplies.
type Service struct {
	// Owners lists the distinct user ids owning purgeable rows.
	Owners func(ctx context.Context, db *pgxpool.Pool) ([]string, error)
	// Purge deletes the rows those users own, inside tx, and reports what
	// it removed (for the log).
	Purge func(ctx context.Context, tx pgx.Tx, userIDs []string) (map[string]int64, error)
	// AfterCommit runs once the deletion is committed (e.g. removing
	// stored files); optional.
	AfterCommit func(ctx context.Context)
}

// RetryInterval is how long Run waits before retrying when the users
// service can't be reached.
var RetryInterval = 15 * time.Second

// Run purges once, retrying until the users service answers or ctx ends.
// Only users the users service knows and who aren't approved partners lose
// content: an unreachable or not-yet-imported account is never treated as
// "not a partner".
func Run(ctx context.Context, db *pgxpool.Pool, users StatusSource, svc Service, log *slog.Logger) {
	for {
		err := runOnce(ctx, db, users, svc, log)
		if err == nil || ctx.Err() != nil {
			return
		}
		log.WarnContext(ctx, "non-partner content purge failed; retrying", slog.Any("error", err))
		select {
		case <-ctx.Done():
			return
		case <-time.After(RetryInterval):
		}
	}
}

func runOnce(ctx context.Context, db *pgxpool.Pool, users StatusSource, svc Service, log *slog.Logger) error {
	done, err := markerExists(ctx, db)
	if err != nil || done {
		return err
	}
	owners, err := svc.Owners(ctx, db)
	if err != nil {
		return fmt.Errorf("list owners: %w", err)
	}
	statuses, err := users.PartnerStatuses(ctx, owners)
	if err != nil {
		return err
	}
	var targets []string
	for _, id := range owners {
		if status, known := statuses[id]; known && status != "approved" {
			targets = append(targets, id)
		}
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	removed := map[string]int64{}
	if len(targets) > 0 {
		if removed, err = svc.Purge(ctx, tx, targets); err != nil {
			return fmt.Errorf("purge: %w", err)
		}
	}
	tag, err := tx.Exec(ctx, `INSERT INTO legacy_imports (name) VALUES ($1) ON CONFLICT DO NOTHING`, Marker)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil // another instance got there first; roll back
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	attrs := []any{slog.Int("users", len(targets))}
	for table, n := range removed {
		attrs = append(attrs, slog.Int64(table, n))
	}
	log.InfoContext(ctx, "removed content published by non-partners", attrs...)
	if svc.AfterCommit != nil && len(targets) > 0 {
		svc.AfterCommit(ctx)
	}
	return nil
}

func markerExists(ctx context.Context, db *pgxpool.Pool) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM legacy_imports WHERE name = $1)`, Marker).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return exists, err
}
