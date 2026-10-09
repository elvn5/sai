package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/users/internal/models"
)

// DefaultSupportMessage is shown until an admin fills in support contacts.
const DefaultSupportMessage = "Служба поддержки пока не настроена. Обратитесь к администратору."

const supportColumns = `id::text, message, phone, telegram_contact, email, working_hours, updated_at`

func scanSupport(row pgx.Row) (models.SupportInfo, error) {
	var s models.SupportInfo
	var updated time.Time
	err := row.Scan(&s.ID, &s.Message, &s.Phone, &s.TelegramContact, &s.Email, &s.WorkingHours, &updated)
	s.UpdatedAt = jsontime.New(updated)
	return s, err
}

// Support returns the support contacts, or ErrNotFound before any exist.
func (s *Store) Support(ctx context.Context) (models.SupportInfo, error) {
	info, err := scanSupport(s.db.QueryRow(ctx, `SELECT `+supportColumns+` FROM support_info ORDER BY updated_at LIMIT 1`))
	if errors.Is(err, pgx.ErrNoRows) {
		return info, ErrNotFound
	}
	return info, err
}

// EnsureSupport returns the support contacts, creating the default row on
// first use (as the NestJS admin endpoint did).
func (s *Store) EnsureSupport(ctx context.Context) (models.SupportInfo, error) {
	info, err := s.Support(ctx)
	if !errors.Is(err, ErrNotFound) {
		return info, err
	}
	return scanSupport(s.db.QueryRow(ctx, `INSERT INTO support_info (message) VALUES ($1) RETURNING `+supportColumns, DefaultSupportMessage))
}

func (s *Store) SaveSupport(ctx context.Context, info models.SupportInfo) (models.SupportInfo, error) {
	return scanSupport(s.db.QueryRow(ctx, `UPDATE support_info SET message = $2, phone = $3,
		telegram_contact = $4, email = $5, working_hours = $6, updated_at = now()
		WHERE id = $1 RETURNING `+supportColumns,
		info.ID, info.Message, info.Phone, info.TelegramContact, info.Email, info.WorkingHours))
}
