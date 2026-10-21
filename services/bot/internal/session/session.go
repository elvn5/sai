// Package session remembers which profile field a chat is about to edit:
// the bot asks for a new value, and the next text message is that value.
package session

import (
	"sync"
	"time"
)

type Field string

const (
	FieldName  Field = "name"
	FieldPhone Field = "phone"
	// A message from an outing's organizer to its participants: being
	// written (the argument is the outing), then written (outing and text).
	FieldBroadcast      Field = "broadcast"
	FieldBroadcastReady Field = "broadcast_ready"
)

// ttl drops edits a user walked away from.
const ttl = 10 * time.Minute

type pending struct {
	field   Field
	arg     string
	expires time.Time
}

// Store is safe for concurrent use: updates are handled in parallel. It
// lives in memory, enough for a single bot instance.
type Store struct {
	mu    sync.Mutex
	edits map[int64]pending
	// Now is overridable in tests.
	Now func() time.Time
}

func New() *Store {
	return &Store{edits: map[int64]pending{}, Now: time.Now}
}

func (s *Store) Start(chatID int64, field Field) { s.StartWith(chatID, field, "") }

// StartWith remembers a field with what it is about (an outing's id).
func (s *Store) StartWith(chatID int64, field Field, arg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.Now()
	for id, p := range s.edits {
		if now.After(p.expires) {
			delete(s.edits, id)
		}
	}
	s.edits[chatID] = pending{field: field, arg: arg, expires: now.Add(ttl)}
}

// Consume returns and forgets the chat's pending edit, if any.
func (s *Store) Consume(chatID int64) (Field, bool) {
	field, _, ok := s.ConsumeWith(chatID)
	return field, ok
}

// ConsumeWith is Consume with the field's argument.
func (s *Store) ConsumeWith(chatID int64) (Field, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.edits[chatID]
	delete(s.edits, chatID)
	if !ok || s.Now().After(p.expires) {
		return "", "", false
	}
	return p.field, p.arg, true
}

func (s *Store) Cancel(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.edits, chatID)
}
