// Package testdb creates throwaway databases for tests on the server in
// TEST_POSTGRES_DSN (tests skip without it).
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elvn5/sai/services/pkg/database"
	"github.com/elvn5/sai/services/users/internal/migrations"
)

// New returns a pool on a fresh database (migrated when migrate is set)
// and its DSN.
func New(t *testing.T, migrate bool) (*pgxpool.Pool, string) {
	t.Helper()
	base := os.Getenv("TEST_POSTGRES_DSN")
	if base == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}
	cfg, err := pgx.ParseConfig(base)
	if err != nil {
		t.Fatal(err)
	}
	suffix := make([]byte, 4)
	_, _ = rand.Read(suffix)
	name := "users_test_" + hex.EncodeToString(suffix)
	dsn := database.DSN(cfg.Host, strconv.Itoa(int(cfg.Port)), cfg.User, cfg.Password, name)
	admin := cfg.Copy()
	admin.Database = "postgres"
	ctx := context.Background()

	var db *pgxpool.Pool
	if migrate {
		db, err = database.Connect(ctx, dsn, migrations.FS, slog.New(slog.DiscardHandler))
	} else {
		var conn *pgx.Conn
		if conn, err = pgx.ConnectConfig(ctx, admin); err == nil {
			_, err = conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
			_ = conn.Close(ctx)
		}
		if err == nil {
			db, err = pgxpool.New(ctx, dsn)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		if conn, err := pgx.ConnectConfig(ctx, admin); err == nil {
			_, _ = conn.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
			_ = conn.Close(ctx)
		}
	})
	return db, dsn
}
