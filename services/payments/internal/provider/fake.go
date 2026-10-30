package provider

import (
	"context"
	"sync"
)

// Fake stands in for the bank: the payment page is the site's own test
// page (SiteURL/payments/test?order=…), where the tester picks the outcome
// (Complete). State lives in memory, so it's for one instance only.
type Fake struct {
	SiteURL string

	mu     sync.Mutex
	states map[string]Result
}

func (f *Fake) Name() string { return "fake" }

func (f *Fake) CreatePayment(_ context.Context, p Payment) (Checkout, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.states == nil {
		f.states = map[string]Result{}
	}
	f.states[p.TransactionID] = Result{State: StatePending, Amount: float64(p.Amount)}
	return Checkout{URL: f.SiteURL + "/payments/test?order=" + p.TransactionID}, nil
}

func (f *Fake) Status(_ context.Context, transactionID string) (Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.states[transactionID]; ok {
		return r, nil
	}
	return Result{State: StatePending}, nil
}

// Complete settles a test payment; false if it isn't known.
func (f *Fake) Complete(transactionID string, success bool) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.states[transactionID]
	if !ok {
		return false
	}
	if success {
		r.State = StatePaid
	} else {
		r.State, r.Message = StateFailed, "Declined on the test page"
	}
	f.states[transactionID] = r
	return true
}
