package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/pkg/jsontime"
)

// Chat is one chat between a partner and a traveler.
type Chat struct {
	ID        string
	PartnerID string
	UserID    string
}

// Other is the one in the chat who isn't userID.
func (c Chat) Other(userID string) string {
	if c.PartnerID == userID {
		return c.UserID
	}
	return c.PartnerID
}

// Has reports whether userID is in the chat.
func (c Chat) Has(userID string) bool { return c.PartnerID == userID || c.UserID == userID }

// ChatID is the id of the chat between a partner and a traveler, nil if
// they haven't written yet.
func (s *Store) ChatID(ctx context.Context, partnerID, userID string) (*string, error) {
	var id string
	err := s.db.QueryRow(ctx, `SELECT id::text FROM chats WHERE partner_id = $1 AND user_id = $2`,
		partnerID, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// OpenChat returns the chat between a partner and a traveler, starting it
// if there's none.
func (s *Store) OpenChat(ctx context.Context, partnerID, userID string) (string, error) {
	var id string
	err := s.db.QueryRow(ctx, `INSERT INTO chats (partner_id, user_id) VALUES ($1, $2)
		ON CONFLICT (partner_id, user_id) DO UPDATE SET partner_id = EXCLUDED.partner_id
		RETURNING id::text`, partnerID, userID).Scan(&id)
	return id, err
}

func (s *Store) Chat(ctx context.Context, id string) (Chat, error) {
	var c Chat
	err := s.db.QueryRow(ctx, `SELECT id::text, partner_id::text, user_id::text FROM chats WHERE id = $1`, id).
		Scan(&c.ID, &c.PartnerID, &c.UserID)
	return c, notFound(err)
}

// unreadSQL counts the messages to $1 in chat ch after their last read.
const unreadSQL = `(SELECT count(*)::int FROM chat_messages m WHERE m.chat_id = ch.id AND m.sender_id <> $1
	AND m.created_at > coalesce(CASE WHEN ch.partner_id = $1 THEN ch.partner_read_at ELSE ch.user_read_at END, '-infinity'))`

// Chats lists userID's chats (as the partner or the traveler) with
// messages, the latest first, with the last message and unread count.
func (s *Store) Chats(ctx context.Context, userID string) ([]models.ChatSummary, error) {
	rows, err := s.db.Query(ctx, `SELECT ch.id::text, ch.partner_id::text, ch.user_id::text,
		m.id::text, m.body, m.sender_id::text, m.created_at, `+unreadSQL+`
		FROM chats ch JOIN LATERAL (SELECT id, body, sender_id, created_at FROM chat_messages
			WHERE chat_id = ch.id ORDER BY created_at DESC LIMIT 1) m ON true
		WHERE ch.partner_id = $1 OR ch.user_id = $1
		ORDER BY ch.last_message_at DESC LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.ChatSummary, error) {
		var c models.ChatSummary
		var at time.Time
		err := r.Scan(&c.ID, &c.PartnerID, &c.UserID, &c.LastMessage.ID, &c.LastMessage.Body, &c.LastMessage.SenderID, &at, &c.Unread)
		c.LastMessage.CreatedAt = jsontime.New(at)
		return c, err
	})
}

// UnreadChats is how many of userID's chats have messages they haven't
// read.
func (s *Store) UnreadChats(ctx context.Context, userID string) (int, error) {
	var n int
	err := s.db.QueryRow(ctx, `SELECT count(*)::int FROM chats ch
		WHERE (ch.partner_id = $1 OR ch.user_id = $1) AND `+unreadSQL+` > 0`, userID).Scan(&n)
	return n, err
}

// Messages lists a chat's messages before a time (nil: the latest),
// oldest first.
func (s *Store) Messages(ctx context.Context, chatID string, before *time.Time, limit int) ([]models.ChatMessage, error) {
	rows, err := s.db.Query(ctx, `SELECT * FROM (SELECT id::text, sender_id::text, body, created_at FROM chat_messages
		WHERE chat_id = $1 AND ($2::timestamptz IS NULL OR created_at < $2)
		ORDER BY created_at DESC LIMIT $3) latest ORDER BY created_at`, chatID, before, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.ChatMessage, error) {
		var m models.ChatMessage
		var at time.Time
		err := r.Scan(&m.ID, &m.SenderID, &m.Body, &at)
		m.CreatedAt = jsontime.New(at)
		return m, err
	})
}

// ReadAt is when the other side of the chat last read it (nil: never).
func (s *Store) ReadAt(ctx context.Context, c Chat, reader string) (*time.Time, error) {
	var at *time.Time
	err := s.db.QueryRow(ctx, `SELECT CASE WHEN partner_id = $2 THEN partner_read_at ELSE user_read_at END
		FROM chats WHERE id = $1`, c.ID, reader).Scan(&at)
	return at, err
}

// AddMessage stores a message from senderID, who has read the chat by
// then.
func (s *Store) AddMessage(ctx context.Context, c Chat, senderID, body string, now time.Time) (models.ChatMessage, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return models.ChatMessage{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	m := models.ChatMessage{SenderID: senderID, Body: body, CreatedAt: jsontime.New(now)}
	if err := tx.QueryRow(ctx, `INSERT INTO chat_messages (chat_id, sender_id, body, created_at) VALUES ($1, $2, $3, $4)
		RETURNING id::text`, c.ID, senderID, body, now).Scan(&m.ID); err != nil {
		return m, err
	}
	if _, err := tx.Exec(ctx, `UPDATE chats SET last_message_at = $2,
		partner_read_at = CASE WHEN partner_id = $3 THEN $2 ELSE partner_read_at END,
		user_read_at = CASE WHEN user_id = $3 THEN $2 ELSE user_read_at END WHERE id = $1`, c.ID, now, senderID); err != nil {
		return m, err
	}
	return m, tx.Commit(ctx)
}

// MarkChatRead records that readerID has read the chat up to now.
func (s *Store) MarkChatRead(ctx context.Context, c Chat, readerID string, now time.Time) error {
	_, err := s.db.Exec(ctx, `UPDATE chats SET
		partner_read_at = CASE WHEN partner_id = $2 THEN $3 ELSE partner_read_at END,
		user_read_at = CASE WHEN user_id = $2 THEN $3 ELSE user_read_at END WHERE id = $1`, c.ID, readerID, now)
	return err
}

// OrganizesTours reports whether userID runs a published tour: travelers
// can write to them.
func (s *Store) OrganizesTours(ctx context.Context, userID string) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tours WHERE user_id = $1 AND is_published)`, userID).Scan(&ok)
	return ok, err
}
