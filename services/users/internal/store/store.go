// Package store reads and writes the users database.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elvn5/sai/services/users/internal/models"
)

var (
	ErrNotFound = errors.New("user not found")
	// ErrDuplicate: the email, phone or Telegram id belongs to another account.
	ErrDuplicate = errors.New("duplicate")
)

type Store struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

const userColumns = `id::text, telegram_id::text, email, password_hash, username, first_name, last_name,
	phone, avatar_url, is_verified, verification_code_hash, verification_code_expires_at,
	verification_code_sent_at, verification_attempts, partner_status, partner_requested_at,
	created_at, updated_at`

func scanUser(row pgx.Row) (models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.TelegramID, &u.Email, &u.PasswordHash, &u.Username, &u.FirstName,
		&u.LastName, &u.Phone, &u.AvatarURL, &u.IsVerified, &u.VerificationCodeHash,
		&u.VerificationCodeExpiresAt, &u.VerificationCodeSentAt, &u.VerificationAttempts,
		&u.PartnerStatus, &u.PartnerRequestedAt, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func (s *Store) collect(ctx context.Context, query string, args ...any) ([]models.User, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.User, error) { return scanUser(r) })
}

func duplicate(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicate
	}
	return err
}

// ByID treats a malformed id as not found.
func (s *Store) ByID(ctx context.Context, id string) (models.User, error) {
	return scanUser(s.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id::text = $1`, id))
}

func (s *Store) ByIDs(ctx context.Context, ids []string) ([]models.User, error) {
	return s.collect(ctx, `SELECT `+userColumns+` FROM users WHERE id::text = ANY($1)`, ids)
}

// ByIdentifier finds the account whose email or phone is identifier.
func (s *Store) ByIdentifier(ctx context.Context, identifier string) (models.User, error) {
	return scanUser(s.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users
		WHERE email = $1 OR phone = $1 ORDER BY created_at LIMIT 1`, identifier))
}

func (s *Store) ByTelegramID(ctx context.Context, telegramID string) (models.User, error) {
	return scanUser(s.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE telegram_id::text = $1`, telegramID))
}

func (s *Store) ByVerificationHash(ctx context.Context, hash string) (models.User, error) {
	return scanUser(s.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE verification_code_hash = $1`, hash))
}

func (s *Store) EmailTaken(ctx context.Context, email string) (bool, error) {
	return s.exists(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`, email)
}

func (s *Store) PhoneTaken(ctx context.Context, phone string) (bool, error) {
	return s.exists(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE phone = $1)`, phone)
}

func (s *Store) exists(ctx context.Context, query string, arg any) (bool, error) {
	var found bool
	err := s.db.QueryRow(ctx, query, arg).Scan(&found)
	return found, err
}

// All lists every account, newest first, for the admin panel.
func (s *Store) All(ctx context.Context) ([]models.User, error) {
	return s.collect(ctx, `SELECT `+userColumns+` FROM users ORDER BY created_at DESC`)
}

// PendingPartners lists partner applications, oldest first.
func (s *Store) PendingPartners(ctx context.Context) ([]models.User, error) {
	return s.collect(ctx, `SELECT `+userColumns+` FROM users
		WHERE partner_status = 'pending' ORDER BY partner_requested_at ASC`)
}

// Insert creates an account and returns the stored row.
func (s *Store) Insert(ctx context.Context, u models.User) (models.User, error) {
	saved, err := scanUser(s.db.QueryRow(ctx, `INSERT INTO users (telegram_id, email, password_hash,
		username, first_name, last_name, phone, avatar_url, is_verified)
		VALUES ($1::bigint, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING `+userColumns,
		u.TelegramID, u.Email, u.PasswordHash, u.Username, u.FirstName, u.LastName, u.Phone, u.AvatarURL, u.IsVerified))
	return saved, duplicate(err)
}

// Save writes every mutable column back.
func (s *Store) Save(ctx context.Context, u models.User) (models.User, error) {
	saved, err := scanUser(s.db.QueryRow(ctx, `UPDATE users SET username = $2, first_name = $3,
		last_name = $4, phone = $5, avatar_url = $6, is_verified = $7, verification_code_hash = $8,
		verification_code_expires_at = $9, verification_code_sent_at = $10, verification_attempts = $11,
		partner_status = $12, partner_requested_at = $13, updated_at = now()
		WHERE id = $1 RETURNING `+userColumns,
		u.ID, u.Username, u.FirstName, u.LastName, u.Phone, u.AvatarURL, u.IsVerified, u.VerificationCodeHash,
		u.VerificationCodeExpiresAt, u.VerificationCodeSentAt, u.VerificationAttempts, u.PartnerStatus,
		u.PartnerRequestedAt))
	return saved, duplicate(err)
}

// SetPassword replaces the password hash.
func (s *Store) SetPassword(ctx context.Context, id, hash string) error {
	_, err := s.db.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, id, hash)
	return err
}

func (s *Store) Delete(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

// ClearVerification marks an account verified and drops its pending code.
func ClearVerification(u *models.User) {
	u.IsVerified = true
	u.VerificationCodeHash = nil
	u.VerificationCodeExpiresAt = nil
	u.VerificationCodeSentAt = nil
	u.VerificationAttempts = 0
}

// SetVerification stores a new pending code (as its hash).
func SetVerification(u *models.User, hash string, ttl time.Duration, now time.Time) {
	expires := now.Add(ttl)
	u.VerificationCodeHash = &hash
	u.VerificationCodeExpiresAt = &expires
	u.VerificationCodeSentAt = &now
	u.VerificationAttempts = 0
}

// TelegramChat is a user's private chat with the bot.
type TelegramChat struct {
	ID int64
	// The chat's language, "" until the bot reports one.
	Lang string
}

// TelegramChats maps the given users that have a Telegram account to
// their Telegram id (the chat id of their private chat with the bot) and
// the chat's language.
func (s *Store) TelegramChats(ctx context.Context, ids []string) (map[string]TelegramChat, error) {
	rows, err := s.db.Query(ctx, `SELECT u.id::text, u.telegram_id, COALESCE(l.lang, '') FROM users u
		LEFT JOIN telegram_languages l ON l.telegram_id = u.telegram_id
		WHERE u.id::text = ANY($1) AND u.telegram_id IS NOT NULL`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]TelegramChat{}
	for rows.Next() {
		var id string
		var chat TelegramChat
		if err := rows.Scan(&id, &chat.ID, &chat.Lang); err != nil {
			return nil, err
		}
		out[id] = chat
	}
	return out, rows.Err()
}

// SetTelegramLanguage keeps the language of a Telegram chat.
func (s *Store) SetTelegramLanguage(ctx context.Context, telegramID int64, lang string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO telegram_languages (telegram_id, lang) VALUES ($1, $2)
		ON CONFLICT (telegram_id) DO UPDATE SET lang = EXCLUDED.lang, updated_at = now()`, telegramID, lang)
	return err
}
