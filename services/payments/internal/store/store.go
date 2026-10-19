// Package store reads and writes the payments database.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elvn5/sai/services/payments/internal/models"
	"github.com/elvn5/sai/services/pkg/jsontime"
)

var (
	ErrNotFound = errors.New("not found")
	// ErrConflict: the order isn't in a state that allows the change.
	ErrConflict = errors.New("conflict")
	// ErrNoPlaces: the item hasn't enough free places left.
	ErrNoPlaces = errors.New("no places")
)

type Store struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

const orderColumns = `id::text, kind, item_id::text, item_title, item_slug, item_locale, item_date::text, quantity,
	unit_price, amount, fee, payee_amount, user_id::text, payee_id::text, contact_name, contact_phone, comment,
	status, provider, payment_url, provider_error, expires_at, paid_at, refund_note, payout_id::text, created_at, item_image_url`

func scanOrder(row pgx.Row) (models.Order, error) {
	var o models.Order
	var expires, created time.Time
	var paid *time.Time
	err := row.Scan(&o.ID, &o.Kind, &o.ItemID, &o.ItemTitle, &o.ItemSlug, &o.ItemLocale, &o.ItemDate, &o.Quantity,
		&o.UnitPrice, &o.Amount, &o.Fee, &o.PayeeAmount, &o.UserID, &o.PayeeID, &o.ContactName, &o.ContactPhone,
		&o.Comment, &o.Status, &o.Provider, &o.PaymentURL, &o.ProviderError, &expires, &paid, &o.RefundNote,
		&o.PayoutID, &created, &o.ItemImageURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return o, ErrNotFound
	}
	o.ExpiresAt, o.CreatedAt, o.PaidAt = jsontime.New(expires), jsontime.New(created), jsontime.Ptr(paid)
	return o, err
}

func (s *Store) collect(ctx context.Context, query string, args ...any) ([]models.Order, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.Order, error) { return scanOrder(r) })
}

// InsertOrder saves a new order. With a capacity, it first checks under a
// per-item lock that the places are still free (ErrNoPlaces otherwise), so
// two bookings at once can't both take the last places.
func (s *Store) InsertOrder(ctx context.Context, o models.Order, capacity *int, now time.Time) (models.Order, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return o, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if capacity != nil {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, o.ItemID); err != nil {
			return o, err
		}
		var taken int
		if err := tx.QueryRow(ctx, `SELECT coalesce(sum(quantity), 0) FROM orders WHERE item_id::text = $1 AND `+
			holdsPlaces, o.ItemID, now).Scan(&taken); err != nil {
			return o, err
		}
		if taken+o.Quantity > *capacity {
			return o, ErrNoPlaces
		}
	}
	saved, err := scanOrder(tx.QueryRow(ctx, `INSERT INTO orders (kind, item_id, item_title, item_slug, item_locale,
		item_date, quantity, unit_price, amount, fee, payee_amount, user_id, payee_id, contact_name, contact_phone,
		comment, provider, expires_at, item_image_url)
		VALUES ($1, $2, $3, $4, $5, $6::date, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING `+orderColumns,
		o.Kind, o.ItemID, o.ItemTitle, o.ItemSlug, o.ItemLocale, o.ItemDate, o.Quantity, o.UnitPrice, o.Amount,
		o.Fee, o.PayeeAmount, o.UserID, o.PayeeID, o.ContactName, o.ContactPhone, o.Comment, o.Provider,
		o.ExpiresAt.Time, o.ItemImageURL))
	if err != nil {
		return saved, err
	}
	return saved, tx.Commit(ctx)
}

// An order holds its places once paid, or while its payment is pending ($2:
// now).
const holdsPlaces = `(status = 'paid' OR (status = 'pending' AND expires_at > $2))`

// TakenPlaces counts the places held on each of the items.
func (s *Store) TakenPlaces(ctx context.Context, itemIDs []string, now time.Time) (map[string]int, error) {
	rows, err := s.db.Query(ctx, `SELECT item_id::text, sum(quantity)::int FROM orders
		WHERE item_id::text = ANY($1) AND `+holdsPlaces+` GROUP BY item_id`, itemIDs, now)
	if err != nil {
		return nil, err
	}
	taken := make(map[string]int, len(itemIDs))
	var id string
	var n int
	_, err = pgx.ForEachRow(rows, []any{&id, &n}, func() error {
		taken[id] = n
		return nil
	})
	return taken, err
}

func (s *Store) Order(ctx context.Context, id string) (models.Order, error) {
	return scanOrder(s.db.QueryRow(ctx, `SELECT `+orderColumns+` FROM orders WHERE id::text = $1`, id))
}

// SetPaymentLink stores the bank's payment page, or the error creating it
// (which fails the order).
func (s *Store) SetPaymentLink(ctx context.Context, id string, url *string, providerError *string) (models.Order, error) {
	return scanOrder(s.db.QueryRow(ctx, `UPDATE orders SET payment_url = $2, provider_error = $3,
		status = CASE WHEN $2::text IS NULL THEN 'failed' ELSE status END, updated_at = now()
		WHERE id::text = $1 RETURNING `+orderColumns, id, url, providerError))
}

func (s *Store) OrdersOfUser(ctx context.Context, userID string) ([]models.Order, error) {
	return s.collect(ctx, `SELECT `+orderColumns+` FROM orders WHERE user_id::text = $1
		ORDER BY created_at DESC LIMIT 100`, userID)
}

// OrdersForPayee lists bookings a partner received (paid or refunded).
func (s *Store) OrdersForPayee(ctx context.Context, payeeID string) ([]models.Order, error) {
	return s.collect(ctx, `SELECT `+orderColumns+` FROM orders WHERE payee_id::text = $1
		AND status IN ('paid', 'refunded') ORDER BY paid_at DESC LIMIT 200`, payeeID)
}

func (s *Store) AllOrders(ctx context.Context, status *string, limit int) ([]models.Order, error) {
	return s.collect(ctx, `SELECT `+orderColumns+` FROM orders WHERE ($1::text IS NULL OR status = $1)
		ORDER BY created_at DESC LIMIT $2`, status, limit)
}

// PendingOrders lists orders awaiting payment, least recently checked first.
func (s *Store) PendingOrders(ctx context.Context, limit int) ([]models.Order, error) {
	return s.collect(ctx, `SELECT `+orderColumns+` FROM orders WHERE status = 'pending' AND payment_url IS NOT NULL
		ORDER BY checked_at NULLS FIRST LIMIT $1`, limit)
}

// Resolve moves a pending order to its outcome. It's a no-op (and returns
// ErrConflict) if another check got there first.
func (s *Store) Resolve(ctx context.Context, id, status string, providerError *string, now time.Time) (models.Order, error) {
	o, err := scanOrder(s.db.QueryRow(ctx, `UPDATE orders SET status = $2, provider_error = $3,
		paid_at = CASE WHEN $2 = 'paid' THEN $4::timestamptz END, checked_at = $4, updated_at = now()
		WHERE id::text = $1 AND status = 'pending' RETURNING `+orderColumns, id, status, providerError, now))
	if errors.Is(err, ErrNotFound) {
		return o, ErrConflict
	}
	return o, err
}

func (s *Store) MarkChecked(ctx context.Context, id string, now time.Time) error {
	_, err := s.db.Exec(ctx, `UPDATE orders SET checked_at = $2 WHERE id::text = $1`, id, now)
	return err
}

// PaidUnnotified lists paid orders whose Telegram messages haven't gone out.
func (s *Store) PaidUnnotified(ctx context.Context, limit int) ([]models.Order, error) {
	return s.collect(ctx, `SELECT `+orderColumns+` FROM orders WHERE status = 'paid' AND notified_at IS NULL
		ORDER BY paid_at LIMIT $1`, limit)
}

func (s *Store) MarkNotified(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `UPDATE orders SET notified_at = now() WHERE id::text = $1`, id)
	return err
}

// Refund marks a paid order not yet paid out as refunded (the money goes
// back to the customer outside the system).
func (s *Store) Refund(ctx context.Context, id string, note *string) (models.Order, error) {
	o, err := scanOrder(s.db.QueryRow(ctx, `UPDATE orders SET status = 'refunded', refund_note = $2, updated_at = now()
		WHERE id::text = $1 AND status = 'paid' AND payout_id IS NULL RETURNING `+orderColumns, id, note))
	if errors.Is(err, ErrNotFound) {
		if _, lookup := s.Order(ctx, id); lookup == nil {
			return o, ErrConflict
		}
	}
	return o, err
}

// Balances sums paid, not-yet-paid-out orders per partner.
func (s *Store) Balances(ctx context.Context) ([]models.Balance, error) {
	rows, err := s.db.Query(ctx, `SELECT payee_id::text, sum(payee_amount)::int, count(*)::int FROM orders
		WHERE status = 'paid' AND payout_id IS NULL GROUP BY payee_id ORDER BY sum(payee_amount) DESC`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.Balance, error) {
		var b models.Balance
		err := r.Scan(&b.Payee.ID, &b.Amount, &b.Orders)
		return b, err
	})
}

// CreatePayout records paying a partner everything they're owed and ties
// those orders to it. ErrConflict: nothing is owed.
func (s *Store) CreatePayout(ctx context.Context, payeeID string, note *string) (models.Payout, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return models.Payout{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var p models.Payout
	var created time.Time
	err = tx.QueryRow(ctx, `INSERT INTO payouts (payee_id, amount, orders, note)
		SELECT $1::uuid, sum(payee_amount), count(*), $2 FROM orders
		WHERE payee_id = $1::uuid AND status = 'paid' AND payout_id IS NULL
		HAVING count(*) > 0
		RETURNING id::text, payee_id::text, amount, orders, note, created_at`, payeeID, note).
		Scan(&p.ID, &p.Payee.ID, &p.Amount, &p.Orders, &p.Note, &created)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrConflict
	}
	if err != nil {
		return p, err
	}
	if _, err := tx.Exec(ctx, `UPDATE orders SET payout_id = $2::uuid, updated_at = now()
		WHERE payee_id = $1::uuid AND status = 'paid' AND payout_id IS NULL`, payeeID, p.ID); err != nil {
		return p, err
	}
	p.CreatedAt = jsontime.New(created)
	return p, tx.Commit(ctx)
}

// Finances sums a partner's paid bookings: owed, paid out and since
// monthStart. Payouts are left for PayoutsOf.
func (s *Store) Finances(ctx context.Context, payeeID string, monthStart time.Time) (models.Finances, error) {
	var f models.Finances
	m := &f.Month
	err := s.db.QueryRow(ctx, `SELECT
		COALESCE(sum(payee_amount) FILTER (WHERE payout_id IS NULL), 0)::int,
		count(*) FILTER (WHERE payout_id IS NULL)::int,
		COALESCE(sum(payee_amount) FILTER (WHERE payout_id IS NOT NULL), 0)::int,
		count(*) FILTER (WHERE paid_at >= $2)::int,
		COALESCE(sum(quantity) FILTER (WHERE paid_at >= $2), 0)::int,
		COALESCE(sum(amount) FILTER (WHERE paid_at >= $2), 0)::int,
		COALESCE(sum(fee) FILTER (WHERE paid_at >= $2), 0)::int,
		COALESCE(sum(payee_amount) FILTER (WHERE paid_at >= $2), 0)::int
		FROM orders WHERE payee_id::text = $1 AND status = 'paid'`, payeeID, monthStart).
		Scan(&f.Owed, &f.OwedOrders, &f.PaidOut, &m.Orders, &m.Guests, &m.Revenue, &m.Fee, &m.Earned)
	return f, err
}

// PayoutsOf lists a partner's payouts, newest first.
func (s *Store) PayoutsOf(ctx context.Context, payeeID string, limit int) ([]models.Payout, error) {
	return s.payouts(ctx, `WHERE payee_id::text = $2`, limit, payeeID)
}

func (s *Store) Payouts(ctx context.Context, limit int) ([]models.Payout, error) {
	return s.payouts(ctx, "", limit)
}

func (s *Store) payouts(ctx context.Context, where string, limit int, args ...any) ([]models.Payout, error) {
	rows, err := s.db.Query(ctx, `SELECT id::text, payee_id::text, amount, orders, note, created_at FROM payouts
		`+where+` ORDER BY created_at DESC LIMIT $1`, append([]any{limit}, args...)...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.Payout, error) {
		var p models.Payout
		var created time.Time
		err := r.Scan(&p.ID, &p.Payee.ID, &p.Amount, &p.Orders, &p.Note, &created)
		p.CreatedAt = jsontime.New(created)
		return p, err
	})
}
