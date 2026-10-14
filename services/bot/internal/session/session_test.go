package session_test

import (
	"testing"
	"time"

	"github.com/elvn5/sai/services/bot/internal/session"
)

func TestStore(t *testing.T) {
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	s := session.New()
	s.Now = func() time.Time { return now }

	if _, ok := s.Consume(1); ok {
		t.Fatal("empty store returned an edit")
	}
	s.Start(1, session.FieldName)
	s.Start(2, session.FieldPhone)
	if field, ok := s.Consume(1); !ok || field != session.FieldName {
		t.Errorf("consume: %q %v", field, ok)
	}
	if _, ok := s.Consume(1); ok {
		t.Error("an edit is consumed once")
	}

	s.Cancel(2)
	if _, ok := s.Consume(2); ok {
		t.Error("cancelled edit still pending")
	}

	s.Start(3, session.FieldPhone)
	now = now.Add(11 * time.Minute)
	if _, ok := s.Consume(3); ok {
		t.Error("expired edit still pending")
	}
}
