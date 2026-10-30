package provider

import (
	"context"
	"errors"
)

// ErrDisabled: online payment is switched off (PAYMENTS_PROVIDER=off).
var ErrDisabled = errors.New("online payment is disabled")

// Disabled is the gateway while online payment is off: no new payments,
// and the site hides its booking form (Online). Orders from before the
// switch expire on schedule.
type Disabled struct{}

func (Disabled) Name() string { return "off" }

func (Disabled) CreatePayment(context.Context, Payment) (Checkout, error) {
	return Checkout{}, ErrDisabled
}

func (Disabled) Status(context.Context, string) (Result, error) { return Result{}, ErrDisabled }

// Online reports whether p takes payments.
func Online(p Provider) bool {
	_, off := p.(Disabled)
	return !off
}
