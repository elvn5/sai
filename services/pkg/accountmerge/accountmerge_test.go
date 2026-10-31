package accountmerge_test

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

	"github.com/elvn5/sai/services/pkg/accountmerge"
	"github.com/elvn5/sai/services/pkg/database"
)

const (
	from  = "11111111-1111-1111-1111-111111111111"
	to    = "22222222-2222-2222-2222-222222222222"
	other = "33333333-3333-3333-3333-333333333333"
)

// testDB is a fresh database on the server in TEST_POSTGRES_DSN.
func testDB(t *testing.T) *pgxpool.Pool {
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
	name := "accountmerge_test_" + hex.EncodeToString(suffix)
	ctx := context.Background()
	admin, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	_ = admin.Close(ctx)
	db, err := pgxpool.New(ctx, database.DSN(cfg.Host, strconv.Itoa(int(cfg.Port)), cfg.User, cfg.Password, name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		if conn, err := pgx.ConnectConfig(ctx, cfg); err == nil {
			_, _ = conn.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)")
			_ = conn.Close(ctx)
		}
	})
	return db
}

func TestApply(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	if _, err := db.Exec(ctx, `
		CREATE TABLE favorites (user_id uuid NOT NULL, tour text NOT NULL, PRIMARY KEY (user_id, tour));
		CREATE TABLE chats (partner_id uuid NOT NULL, user_id uuid NOT NULL, note text, UNIQUE (partner_id, user_id));
		CREATE TABLE posts (id serial PRIMARY KEY, user_id uuid, title text);
		CREATE TABLE refs (id uuid, user_id text);
		CREATE TABLE account_merges_applied (merge_id bigint PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now());`); err != nil {
		t.Fatal(err)
	}
	for _, insert := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO favorites VALUES ($1, 'ala-kul'), ($1, 'sary-chelek'), ($2, 'ala-kul')`, []any{from, to}},
		{`INSERT INTO chats VALUES ($3, $1, 'from'), ($3, $2, 'to'), ($1, $3, 'as partner')`, []any{from, to, other}},
		{`INSERT INTO posts (user_id, title) VALUES ($1, 'a'), ($1, 'b'), (NULL, 'c')`, []any{from}},
		{`INSERT INTO refs VALUES ($1, $2)`, []any{from, from}},
	} {
		if _, err := db.Exec(ctx, insert.sql, insert.args...); err != nil {
			t.Fatal(err)
		}
	}

	merge := accountmerge.Merge{ID: 1, From: from, To: to}
	if err := accountmerge.Apply(ctx, db, merge, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatal(err)
	}

	count := func(query string, args ...any) int {
		t.Helper()
		var n int
		if err := db.QueryRow(ctx, query, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := count(`SELECT count(*) FROM favorites WHERE user_id = $1`, to); n != 2 {
		t.Errorf("favorites of to: %d, want 2 (the shared tour once)", n)
	}
	if n := count(`SELECT count(*) FROM chats WHERE note = 'to' AND user_id = $1`, to); n != 1 {
		t.Errorf("to's own chat with the partner should win")
	}
	if n := count(`SELECT count(*) FROM chats WHERE partner_id = $1`, to); n != 1 {
		t.Errorf("chats as partner moved: %d", n)
	}
	if n := count(`SELECT count(*) FROM posts WHERE user_id = $1`, to); n != 2 {
		t.Errorf("posts moved: %d", n)
	}
	// Only uuid columns with a known name are touched.
	if n := count(`SELECT count(*) FROM refs WHERE id = $1 AND user_id = $2`, from, from); n != 1 {
		t.Errorf("unrelated columns changed")
	}
	for _, table := range []string{"favorites", "chats", "posts"} {
		if n := count(`SELECT count(*) FROM `+table+` WHERE user_id = $1`, from); n != 0 {
			t.Errorf("%s still has from: %d", table, n)
		}
	}
	if n := count(`SELECT count(*) FROM account_merges_applied`); n != 1 {
		t.Errorf("applied: %d", n)
	}
}
