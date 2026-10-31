package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/pkg/accountmerge"
	"github.com/elvn5/sai/services/users/internal/models"
)

// Why two accounts were merged (account_merges.reason).
const (
	MergeEmail    = "email"    // the Telegram account confirmed the other's email
	MergePhone    = "phone"    // … shared the other's phone number in the bot
	MergeTelegram = "telegram" // the site account linked the other's Telegram
	MergeAdmin    = "admin"    // merged from the admin panel
)

var (
	// ErrNotMergeable: the account to merge away has its own way to sign
	// in (an email or a password), or the two have different Telegram
	// accounts.
	ErrNotMergeable = errors.New("accounts can't be merged")
	// ErrSameAccount: an account can't be merged into itself.
	ErrSameAccount = errors.New("same account")
)

// CanMerge reports whether from may be merged into to: from signs in only
// through Telegram (no email, no password: a merge never takes a way to
// sign in from anyone), and the two don't hold different Telegram ids.
func CanMerge(from, to models.User) error {
	switch {
	case from.ID == to.ID:
		return ErrSameAccount
	case from.Email != nil || from.PasswordHash != nil:
		return ErrNotMergeable
	case from.TelegramID != nil && to.TelegramID != nil && *from.TelegramID != *to.TelegramID:
		return ErrNotMergeable
	}
	return nil
}

// Merge folds the account fromID into toID: to takes over from's Telegram
// account and fills its empty profile fields (and partnership) from it,
// whatever from owned in this database moves to to, from is deleted, and
// the merge is recorded for the other services. Returns to as saved.
func (s *Store) Merge(ctx context.Context, fromID, toID, reason string) (models.User, error) {
	var merged models.User
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		lock := `SELECT ` + userColumns + ` FROM users WHERE id::text = $1 FOR UPDATE`
		from, err := scanUser(tx.QueryRow(ctx, lock, fromID))
		if err != nil {
			return err
		}
		to, err := scanUser(tx.QueryRow(ctx, lock, toID))
		if err != nil {
			return err
		}
		if err := CanMerge(from, to); err != nil {
			return err
		}

		// from lets go of its unique values before to takes them.
		var awardCode *string
		if err := tx.QueryRow(ctx, `SELECT award_code FROM users WHERE id = $1`, from.ID).Scan(&awardCode); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET telegram_id = NULL, phone = NULL, award_code = NULL
			WHERE id = $1`, from.ID); err != nil {
			return err
		}
		partner := to.PartnerStatus == "none" && from.PartnerStatus != "none"
		if _, err := tx.Exec(ctx, `UPDATE users SET
			telegram_id = coalesce(telegram_id, $2::bigint), username = coalesce(username, $3),
			first_name = coalesce(first_name, $4), last_name = coalesce(last_name, $5),
			phone = coalesce(phone, $6), avatar_url = coalesce(avatar_url, $7),
			award_code = coalesce(award_code, $8), is_verified = true,
			partner_status = CASE WHEN $9 THEN $10 ELSE partner_status END,
			partner_requested_at = CASE WHEN $9 THEN $11 ELSE partner_requested_at END,
			partner_kind = CASE WHEN $9 THEN $12 ELSE partner_kind END,
			updated_at = now()
			WHERE id = $1`,
			to.ID, from.TelegramID, from.Username, from.FirstName, from.LastName, from.Phone, from.AvatarURL,
			awardCode, partner, from.PartnerStatus, from.PartnerRequestedAt, from.PartnerKind); err != nil {
			return duplicate(err)
		}

		if _, err := tx.Exec(ctx, `DELETE FROM password_reset_tokens WHERE user_id = $1`, from.ID); err != nil {
			return err
		}
		if _, err := accountmerge.Reassign(ctx, tx, from.ID, to.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO account_merges (from_id, to_id, reason) VALUES ($1, $2, $3)`,
			from.ID, to.ID, reason); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, from.ID); err != nil {
			return err
		}
		merged, err = scanUser(tx.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, to.ID))
		return err
	})
	return merged, err
}

// AccountMerges lists the merges recorded after id, oldest first.
func (s *Store) AccountMerges(ctx context.Context, after int64, limit int) ([]accountmerge.Merge, error) {
	rows, err := s.db.Query(ctx, `SELECT id, from_id::text, to_id::text FROM account_merges
		WHERE id > $1 ORDER BY id LIMIT $2`, after, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (accountmerge.Merge, error) {
		var m accountmerge.Merge
		return m, r.Scan(&m.ID, &m.From, &m.To)
	})
}

// ByPhone finds the account with this phone number, however it was
// typed: digits are compared, and a Kyrgyz number matches its local form
// (996555123456 = 0555123456).
func (s *Store) ByPhone(ctx context.Context, phone string) (models.User, error) {
	return scanUser(s.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users
		WHERE regexp_replace(phone, '\D', '', 'g') = ANY($1) ORDER BY created_at LIMIT 1`, PhoneForms(phone)))
}

// PhoneForms are the digit strings phone may be stored as.
func PhoneForms(phone string) []string {
	digits := Digits(phone)
	forms := []string{digits}
	switch {
	case len(digits) == 12 && digits[:3] == "996":
		forms = append(forms, "0"+digits[3:])
	case len(digits) == 10 && digits[0] == '0':
		forms = append(forms, "996"+digits[1:])
	}
	return forms
}

// Digits keeps only phone's digits.
func Digits(phone string) string {
	out := make([]byte, 0, len(phone))
	for i := range len(phone) {
		if phone[i] >= '0' && phone[i] <= '9' {
			out = append(out, phone[i])
		}
	}
	return string(out)
}
