package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/users/internal/models"
)

// ErrResetInvalid: the reset token is unknown, used or expired.
var ErrResetInvalid = errors.New("password reset token is invalid or expired")

// ResetLimit is at most Max requests within Window.
type ResetLimit struct {
	Max    int
	Window time.Duration
}

// AllowResetRequest records a reset request for identifierHash from ip,
// unless one of the limits is already used up (false: refused, nothing
// recorded). Known and unknown identifiers count alike. Requests from the
// same identifier are serialized, so two at once can't both slip through.
func (s *Store) AllowResetRequest(ctx context.Context, identifierHash, ip string, now time.Time,
	perIdentifier, perIP []ResetLimit,
) (bool, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, identifierHash); err != nil {
		return false, err
	}
	for _, check := range []struct {
		column, value string
		limits        []ResetLimit
	}{{"identifier_hash", identifierHash, perIdentifier}, {"ip", ip, perIP}} {
		for _, limit := range check.limits {
			var count int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM password_reset_requests
				WHERE `+check.column+` = $1 AND created_at > $2`, check.value, now.Add(-limit.Window)).Scan(&count); err != nil {
				return false, err
			}
			if count >= limit.Max {
				return false, nil
			}
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO password_reset_requests (identifier_hash, ip, created_at) VALUES ($1, $2, $3)`,
		identifierHash, ip, now); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM password_reset_requests WHERE created_at < $1`, now.Add(-24*time.Hour)); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// CreatePasswordReset stores a new reset token for userID; the account's
// earlier tokens stop working.
func (s *Store) CreatePasswordReset(ctx context.Context, userID, tokenHash, channel string, expiresAt time.Time) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM password_reset_tokens WHERE user_id::text = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO password_reset_tokens (token_hash, user_id, channel, expires_at)
		VALUES ($1, $2::uuid, $3, $4)`, tokenHash, userID, channel, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ResetPassword spends a reset token: sets the password hash, ends the
// sessions issued before validAfter, and drops the account's reset tokens
// and pending verification code (the link proved the email or Telegram,
// so the account counts as verified). ErrResetInvalid when the token is
// unknown or expired.
func (s *Store) ResetPassword(ctx context.Context, tokenHash, passwordHash string, now, validAfter time.Time) (models.User, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return models.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID string
	err = tx.QueryRow(ctx, `DELETE FROM password_reset_tokens WHERE token_hash = $1 AND expires_at > $2
		RETURNING user_id::text`, tokenHash, now).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, ErrResetInvalid
	} else if err != nil {
		return models.User{}, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM password_reset_tokens WHERE user_id::text = $1`, userID); err != nil {
		return models.User{}, err
	}
	u, err := scanUser(tx.QueryRow(ctx, `UPDATE users SET password_hash = $2, sessions_valid_after = $3,
		is_verified = true, verification_code_hash = NULL, verification_code_expires_at = NULL,
		verification_code_sent_at = NULL, verification_attempts = 0, updated_at = now()
		WHERE id::text = $1 RETURNING `+userColumns, userID, passwordHash, validAfter))
	if err != nil {
		return u, err
	}
	return u, tx.Commit(ctx)
}

// PasswordResetValid: the token exists and hasn't expired.
func (s *Store) PasswordResetValid(ctx context.Context, tokenHash string, now time.Time) (bool, error) {
	var found bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM password_reset_tokens
		WHERE token_hash = $1 AND expires_at > $2)`, tokenHash, now).Scan(&found)
	return found, err
}
