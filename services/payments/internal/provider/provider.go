// Package provider talks to the payment gateway: GoPay (ELQR), Bakai Bank
// Open Banking, or a fake for development and staging.
package provider

import (
	"context"
	"time"
)

// State is a payment's state at the gateway.
type State string

const (
	StatePending State = "pending"
	StatePaid    State = "paid"
	StateFailed  State = "failed"
	// The gateway closed the payment unpaid once its time ran out.
	StateExpired State = "expired"
)

// Payment is what's charged: TransactionID ties the gateway's payment to
// our order.
type Payment struct {
	TransactionID string
	Amount        int // som
	Comment       string
	ReturnURL     string
	TTL           time.Duration
}

// Result is a payment's state; Amount is what the gateway reports as paid.
type Result struct {
	State   State
	Amount  float64
	Message string
	// The gateway's own record, when it gives one (GoPay).
	Details *Details
}

// Checkout is a created payment: where the customer pays.
type Checkout struct {
	URL     string
	Details *Details
}

// Details is the gateway's record of a payment, kept as the order's
// transaction (provider_payments) for the admin panel.
type Details struct {
	PaymentID string
	// The gateway's status as it names it (CREATED, COMMITTED…).
	Status      string
	Amount      string
	Testing     bool
	CheckoutURL *string
	CreatedAt   *time.Time
	ExpiresAt   *time.Time
	CommittedAt *time.Time
	BankOpDate  *string
}

type Provider interface {
	Name() string
	CreatePayment(ctx context.Context, p Payment) (Checkout, error)
	Status(ctx context.Context, transactionID string) (Result, error)
}

// Canceler is a gateway that can cancel an unpaid payment (GoPay).
type Canceler interface {
	Cancel(ctx context.Context, transactionID string) (Result, error)
}
