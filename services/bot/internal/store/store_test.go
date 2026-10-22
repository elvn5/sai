package store_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/bot/internal/migrations"
	"github.com/elvn5/sai/services/bot/internal/store"
	"github.com/elvn5/sai/services/pkg/database"
)

// newStore migrates a throwaway database on the server in
// TEST_POSTGRES_DSN (skips without it).
func newStore(t *testing.T) *store.Store {
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
	name := "bot_test_" + hex.EncodeToString(suffix)
	ctx := context.Background()
	db, err := database.Connect(ctx, database.DSN(cfg.Host, strconv.Itoa(int(cfg.Port)), cfg.User, cfg.Password, name),
		migrations.FS, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		admin := cfg.Copy()
		admin.Database = "postgres"
		if conn, err := pgx.ConnectConfig(ctx, admin); err == nil {
			_, _ = conn.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
			_ = conn.Close(ctx)
		}
	})
	return store.New(db)
}

func TestBlogSubscribers(t *testing.T) {
	s, ctx := newStore(t), context.Background()
	if on, err := s.IsBlogSubscriber(ctx, 42); err != nil || on {
		t.Fatalf("fresh chat: %v, %v", on, err)
	}
	for _, chat := range []int64{42, 7, 42} { // subscribing twice is fine
		if err := s.SetBlogSubscriber(ctx, chat, true); err != nil {
			t.Fatal(err)
		}
	}
	if on, err := s.IsBlogSubscriber(ctx, 42); err != nil || !on {
		t.Errorf("subscribed: %v, %v", on, err)
	}
	if chats, err := s.BlogSubscribers(ctx); err != nil || !slices.Equal(chats, []store.Subscriber{{ChatID: 42, Lang: "ru"}, {ChatID: 7, Lang: "ru"}}) {
		t.Errorf("subscribers: %v, %v", chats, err)
	}
	for range 2 { // unsubscribing twice too
		if err := s.SetBlogSubscriber(ctx, 42, false); err != nil {
			t.Fatal(err)
		}
	}
	if chats, err := s.BlogSubscribers(ctx); err != nil || !slices.Equal(chats, []store.Subscriber{{ChatID: 7, Lang: "ru"}}) {
		t.Errorf("after unsubscribing: %v, %v", chats, err)
	}
	if err := s.SetLanguage(ctx, 7, "en"); err != nil {
		t.Fatal(err)
	}
	if chats, err := s.BlogSubscribers(ctx); err != nil || !slices.Equal(chats, []store.Subscriber{{ChatID: 7, Lang: "en"}}) {
		t.Errorf("with a language: %v, %v", chats, err)
	}
}

func TestLanguages(t *testing.T) {
	s, ctx := newStore(t), context.Background()
	if lang, err := s.KnownLanguage(ctx, 5); err != nil || lang != "" {
		t.Fatalf("unknown chat: %q, %v", lang, err)
	}
	// The first language a chat gets stays, until the user picks another.
	for _, fallback := range []string{"de", "en"} {
		if lang, err := s.Language(ctx, 5, fallback); err != nil || lang != "de" {
			t.Errorf("Language(%s) = %q, %v", fallback, lang, err)
		}
	}
	if err := s.SetLanguage(ctx, 5, "ky"); err != nil {
		t.Fatal(err)
	}
	if lang, err := s.KnownLanguage(ctx, 5); err != nil || lang != "ky" {
		t.Errorf("picked: %q, %v", lang, err)
	}
}

func TestSeenPosts(t *testing.T) {
	s, ctx := newStore(t), context.Background()
	if seen, err := s.SeenPosts(ctx); err != nil || len(seen) != 0 {
		t.Fatalf("fresh: %v, %v", seen, err)
	}
	if err := s.MarkSeen(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkSeen(ctx, []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkSeen(ctx, []string{"b", "c"}); err != nil {
		t.Fatal(err)
	}
	seen, err := s.SeenPosts(ctx)
	if err != nil || len(seen) != 3 || !seen["a"] || !seen["c"] {
		t.Errorf("seen: %v, %v", seen, err)
	}
}
