// Package provider talks to the payment gateway: Bakai Bank Open Banking,
// or a fake for development and staging.
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
}

type Provider interface {
	Name() string
	// CreatePayment returns the URL of the gateway's payment page.
	CreatePayment(ctx context.Context, p Payment) (string, error)
	Status(ctx context.Context, transactionID string) (Result, error)
}
