package users_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/elvn5/sai/services/bot/internal/users"
)

type memoryLanguages map[int64]string

func (m memoryLanguages) Language(_ context.Context, chatID int64, fallback string) (string, error) {
	if l, ok := m[chatID]; ok {
		return l, nil
	}
	m[chatID] = fallback
	return fallback, nil
}

func (m memoryLanguages) KnownLanguage(_ context.Context, chatID int64) (string, error) {
	return m[chatID], nil
}

func (m memoryLanguages) SetLanguage(_ context.Context, chatID int64, lang string) error {
	m[chatID] = lang
	return nil
}

type reports struct {
	mu   sync.Mutex
	got  []string
	fail bool
	sent chan struct{}
}

func (r *reports) SetLanguage(_ context.Context, telegramID int64, lang string) error {
	r.mu.Lock()
	defer func() {
		r.mu.Unlock()
		r.sent <- struct{}{}
	}()
	if r.fail {
		return errors.New("users service down")
	}
	r.got = append(r.got, lang)
	return nil
}

func (r *reports) wait(t *testing.T) {
	t.Helper()
	select {
	case <-r.sent:
	case <-time.After(2 * time.Second):
		t.Fatal("no report")
	}
}

func TestLanguagesReportToUsers(t *testing.T) {
	r := &reports{sent: make(chan struct{}, 10)}
	l := &users.Languages{Store: memoryLanguages{}, Users: r, Log: slog.New(slog.DiscardHandler)}
	ctx := context.Background()

	// First contact: the detected language is reported once.
	if got, _ := l.Language(ctx, 7, "en"); got != "en" {
		t.Fatalf("language: %q", got)
	}
	r.wait(t)
	_, _ = l.Language(ctx, 7, "ru")
	// A pick is reported right away; the same one again isn't.
	if err := l.SetLanguage(ctx, 7, "de"); err != nil {
		t.Fatal(err)
	}
	r.wait(t)
	_ = l.SetLanguage(ctx, 7, "de")
	if known, _ := l.KnownLanguage(ctx, 7); known != "de" {
		t.Errorf("known: %q", known)
	}
	r.mu.Lock()
	if len(r.got) != 2 || r.got[0] != "en" || r.got[1] != "de" {
		t.Errorf("reports: %v", r.got)
	}
	r.mu.Unlock()

	// A failed report is retried on the chat's next message.
	r.mu.Lock()
	r.fail = true
	r.mu.Unlock()
	_ = l.SetLanguage(ctx, 8, "ky")
	r.wait(t)
	r.mu.Lock()
	r.fail = false
	r.mu.Unlock()
	_, _ = l.Language(ctx, 8, "ru")
	r.wait(t)
	r.mu.Lock()
	if last := r.got[len(r.got)-1]; last != "ky" {
		t.Errorf("retry: %v", r.got)
	}
	r.mu.Unlock()
}
