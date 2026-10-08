// Package database connects a service to its own PostgreSQL database and
// keeps its schema migrated.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Connect ensures the database named in url exists, applies the embedded
// goose migrations from migrationsFS (SQL files at its root), and returns
// a connection pool.
func Connect(ctx context.Context, url string, migrationsFS fs.FS, log *slog.Logger) (*pgxpool.Pool, error) {
	if err := ensureDatabase(ctx, url, log); err != nil {
		return nil, err
	}

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	db := stdlib.OpenDBFromPool(pool)
	defer func() { _ = db.Close() }()
	if err := migrate(ctx, db, migrationsFS, log); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func migrate(ctx context.Context, db *sql.DB, migrationsFS fs.FS, log *slog.Logger) error {
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrationsFS)
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	for _, r := range results {
		log.Info("applied migration", slog.String("file", r.Source.Path), slog.Duration("took", r.Duration))
	}
	return nil
}

// ensureDatabase creates the service's database on first start, connecting
// to the server's "postgres" maintenance database with the same
// credentials (which therefore need CREATEDB).
func ensureDatabase(ctx context.Context, url string, log *slog.Logger) error {
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		return fmt.Errorf("parse database URL: %w", err)
	}
	name := cfg.Database
	if name == "" {
		return errors.New("database URL has no database name")
	}

	admin := cfg.Copy()
	admin.Database = "postgres"
	conn, err := pgx.ConnectConfig(ctx, admin)
	if err != nil {
		return fmt.Errorf("connect to maintenance database: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	var exists bool
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists); err != nil {
		return fmt.Errorf("check database %q: %w", name, err)
	}
	if exists {
		return nil
	}
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		return fmt.Errorf("create database %q: %w", name, err)
	}
	log.Info("created database", slog.String("name", name))
	return nil
}

// DSN builds a keyword/value connection string, quoting values so that
// passwords containing URL-special characters work.
func DSN(host, port, user, password, dbname string) string {
	quote := func(v string) string {
		v = strings.ReplaceAll(v, `\`, `\\`)
		return "'" + strings.ReplaceAll(v, "'", `\'`) + "'"
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s",
		quote(host), quote(port), quote(user), quote(password), quote(dbname))
}
