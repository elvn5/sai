package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/payments/internal/models"
	"github.com/elvn5/sai/services/payments/internal/provider"
	"github.com/elvn5/sai/services/pkg/jsontime"
)

const transactionColumns = `t.order_id::text, t.provider, t.payment_id, t.provider_order_id, t.status, t.amount::text,
	t.testing, t.checkout_url, t.gateway_created_at, t.expires_at, t.committed_at, t.bank_op_date::text, t.last_error,
	t.checked_at, t.created_at, t.updated_at`

// SaveTransaction keeps what the gateway reported about an order's payment.
func (s *Store) SaveTransaction(ctx context.Context, orderID, gateway, gatewayOrderID string, d provider.Details) error {
	_, err := s.db.Exec(ctx, `INSERT INTO provider_payments (order_id, provider, payment_id, provider_order_id, status,
		amount, testing, checkout_url, gateway_created_at, expires_at, committed_at, bank_op_date)
		VALUES ($1::uuid, $2, NULLIF($3, ''), $4, $5, $6::numeric, $7, $8, $9, $10, $11, $12::date)
		ON CONFLICT (order_id) DO UPDATE SET payment_id = COALESCE(EXCLUDED.payment_id, provider_payments.payment_id),
			status = EXCLUDED.status, amount = EXCLUDED.amount, testing = EXCLUDED.testing,
			checkout_url = COALESCE(EXCLUDED.checkout_url, provider_payments.checkout_url),
			gateway_created_at = COALESCE(EXCLUDED.gateway_created_at, provider_payments.gateway_created_at),
			expires_at = COALESCE(EXCLUDED.expires_at, provider_payments.expires_at),
			committed_at = COALESCE(EXCLUDED.committed_at, provider_payments.committed_at),
			bank_op_date = COALESCE(EXCLUDED.bank_op_date, provider_payments.bank_op_date),
			last_error = NULL, checked_at = now(), updated_at = now()`,
		orderID, gateway, d.PaymentID, gatewayOrderID, d.Status, nonEmpty(d.Amount, "0"), d.Testing, d.CheckoutURL,
		d.CreatedAt, d.ExpiresAt, d.CommittedAt, d.BankOpDate)
	return err
}

// TransactionError notes a failed request to the gateway about a payment.
func (s *Store) TransactionError(ctx context.Context, orderID, message string) error {
	_, err := s.db.Exec(ctx, `UPDATE provider_payments SET last_error = $2, checked_at = now(), updated_at = now()
		WHERE order_id::text = $1`, orderID, message)
	return err
}

func nonEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func scanTransaction(row pgx.Row, withOrder bool) (models.Transaction, error) {
	var t models.Transaction
	var created, expires, committed *time.Time
	var checked, createdAt, updated time.Time
	dest := []any{&t.OrderID, &t.Provider, &t.PaymentID, &t.ProviderOrderID, &t.Status, &t.Amount, &t.Testing,
		&t.CheckoutURL, &created, &expires, &committed, &t.BankOpDate, &t.LastError, &checked, &createdAt, &updated}
	var o models.Order
	var oExpires, oCreated time.Time
	var oPaid *time.Time
	if withOrder {
		dest = append(dest, &o.ID, &o.Kind, &o.ItemID, &o.ItemTitle, &o.ItemSlug, &o.ItemLocale, &o.ItemDate, &o.Quantity,
			&o.UnitPrice, &o.Amount, &o.Fee, &o.PayeeAmount, &o.UserID, &o.PayeeID, &o.ContactName, &o.ContactPhone,
			&o.Comment, &o.Status, &o.Provider, &o.PaymentURL, &o.ProviderError, &oExpires, &oPaid, &o.RefundNote,
			&o.PayoutID, &oCreated, &o.ItemImageURL)
	}
	err := row.Scan(dest...)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	t.GatewayCreated, t.ExpiresAt, t.CommittedAt = jsontime.Ptr(created), jsontime.Ptr(expires), jsontime.Ptr(committed)
	t.CheckedAt, t.CreatedAt, t.UpdatedAt = jsontime.New(checked), jsontime.New(createdAt), jsontime.New(updated)
	if withOrder {
		o.ExpiresAt, o.CreatedAt, o.PaidAt = jsontime.New(oExpires), jsontime.New(oCreated), jsontime.Ptr(oPaid)
		t.Order = &o
	}
	return t, err
}

// orderColumnsOf is orderColumns of the joined orders table o.
const orderColumnsOf = `o.id::text, o.kind, o.item_id::text, o.item_title, o.item_slug, o.item_locale, o.item_date::text,
	o.quantity, o.unit_price, o.amount, o.fee, o.payee_amount, o.user_id::text, o.payee_id::text, o.contact_name,
	o.contact_phone, o.comment, o.status, o.provider, o.payment_url, o.provider_error, o.expires_at, o.paid_at,
	o.refund_note, o.payout_id::text, o.created_at, o.item_image_url`

// Transactions lists the gateway's payments with their orders, newest first.
func (s *Store) Transactions(ctx context.Context, f models.TransactionFilter) ([]models.Transaction, error) {
	rows, err := s.db.Query(ctx, `SELECT `+transactionColumns+`, `+orderColumnsOf+`
		FROM provider_payments t JOIN orders o ON o.id = t.order_id
		WHERE ($1::text IS NULL OR t.status = $1)
		AND ($2::date IS NULL OR t.created_at >= ($2::date::timestamp AT TIME ZONE 'Asia/Bishkek'))
		AND ($3::date IS NULL OR t.created_at < (($3::date + 1)::timestamp AT TIME ZONE 'Asia/Bishkek'))
		ORDER BY t.created_at DESC LIMIT $4`, f.Status, f.From, f.To, f.Limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.Transaction, error) { return scanTransaction(r, true) })
}

// Transaction is one payment with its order and webhook deliveries.
func (s *Store) Transaction(ctx context.Context, orderID string) (models.Transaction, error) {
	t, err := scanTransaction(s.db.QueryRow(ctx, `SELECT `+transactionColumns+`, `+orderColumnsOf+`
		FROM provider_payments t JOIN orders o ON o.id = t.order_id WHERE t.order_id::text = $1`, orderID), true)
	if err != nil {
		return t, err
	}
	rows, err := s.db.Query(ctx, `SELECT id, event, payment_id, testing, attempts, received_at, processed_at, error
		FROM provider_events WHERE order_id::text = $1 ORDER BY received_at DESC LIMIT 50`, orderID)
	if err != nil {
		return t, err
	}
	t.Events, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.ProviderEvent, error) {
		var e models.ProviderEvent
		var received time.Time
		var processed *time.Time
		err := r.Scan(&e.ID, &e.Event, &e.PaymentID, &e.Testing, &e.Attempts, &received, &processed, &e.Error)
		e.ReceivedAt, e.ProcessedAt = jsontime.New(received), jsontime.Ptr(processed)
		return e, err
	})
	if t.Events == nil {
		t.Events = []models.ProviderEvent{}
	}
	return t, err
}

// Event is a webhook delivery to record.
type Event struct {
	Provider  string
	Key       string
	Event     string
	PaymentID string
	OrderID   *string
	Testing   bool
	Payload   []byte
}

// RecordEvent stores a webhook delivery and returns its id; done is true
// when the same event was already processed (a repeated delivery).
func (s *Store) RecordEvent(ctx context.Context, e Event) (id int64, done bool, err error) {
	var processed *time.Time
	err = s.db.QueryRow(ctx, `INSERT INTO provider_events (provider, event_key, event, payment_id, order_id, testing, payload)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5::uuid, $6, $7::jsonb)
		ON CONFLICT (provider, event_key) DO UPDATE SET attempts = provider_events.attempts + 1
		RETURNING id, processed_at`, e.Provider, e.Key, e.Event, e.PaymentID, e.OrderID, e.Testing, string(e.Payload)).
		Scan(&id, &processed)
	return id, processed != nil, err
}

// FinishEvent marks a delivery processed, or notes why it failed (it's
// then processed again on the next delivery).
func (s *Store) FinishEvent(ctx context.Context, id int64, processed bool, problem *string) error {
	_, err := s.db.Exec(ctx, `UPDATE provider_events SET error = $3,
		processed_at = CASE WHEN $2 THEN now() END WHERE id = $1`, id, processed, problem)
	return err
}
