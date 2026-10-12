package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/users/internal/models"
)

// SetTelegramLinkToken records a pending Telegram link for userID.
func (s *Store) SetTelegramLinkToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	tag, err := s.db.Exec(ctx, `UPDATE users SET telegram_link_token_hash = $2, telegram_link_expires_at = $3,
		updated_at = now() WHERE id::text = $1`, userID, tokenHash, expiresAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ByTelegramLinkToken finds the account with a pending, unexpired link
// token.
func (s *Store) ByTelegramLinkToken(ctx context.Context, tokenHash string, now time.Time) (models.User, error) {
	return scanUser(s.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users
		WHERE telegram_link_token_hash = $1 AND telegram_link_expires_at > $2`, tokenHash, now))
}

// LinkTelegram sets userID's Telegram id (and username, if it has none)
// and clears the link token. With moveFrom set, that account's Telegram id
// is cleared first, in the same transaction.
func (s *Store) LinkTelegram(ctx context.Context, userID, telegramID string, username *string, moveFrom string) (models.User, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return models.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if moveFrom != "" {
		if _, err := tx.Exec(ctx, `UPDATE users SET telegram_id = NULL, updated_at = now() WHERE id::text = $1`, moveFrom); err != nil {
			return models.User{}, err
		}
	}
	u, err := scanUser(tx.QueryRow(ctx, `UPDATE users SET telegram_id = $2::bigint, username = coalesce(username, $3),
		telegram_link_token_hash = NULL, telegram_link_expires_at = NULL, updated_at = now()
		WHERE id::text = $1 RETURNING `+userColumns, userID, telegramID, username))
	if err != nil {
		return u, duplicate(err)
	}
	return u, tx.Commit(ctx)
}

// UnlinkTelegram clears userID's Telegram id.
func (s *Store) UnlinkTelegram(ctx context.Context, userID string) (models.User, error) {
	u, err := scanUser(s.db.QueryRow(ctx, `UPDATE users SET telegram_id = NULL, updated_at = now()
		WHERE id::text = $1 RETURNING `+userColumns, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}
