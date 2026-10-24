package routes_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/elvn5/sai/services/payments/internal/provider"
	"github.com/elvn5/sai/services/payments/internal/store"
)

func TestConcurrentPayouts(t *testing.T) {
	e := setup(t, &provider.Fake{SiteURL: "https://ermekten.kg"})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, body := e.do("POST", "/orders", order(tourID, 1), tourist)
	id := obj(t, body)["id"].(string)
	e.fake.Complete(id, true)
	e.worker.Tick(ctx)

	// Hold the order so both requests reach the locking query before either
	// can finish. The old implementation inserted two payouts before waiting.
	tx, err := e.db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT id FROM orders WHERE id = $1::uuid FOR UPDATE`, id); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := e.worker.Store.CreatePayout(ctx, partner, nil)
			results <- err
		}()
	}
	for {
		var waiting int
		if err := e.db.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database() AND wait_event_type = 'Lock'`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting >= 2 {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("payout requests did not reach the lock")
		case <-time.After(10 * time.Millisecond):
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var succeeded, conflicted int
	for range 2 {
		switch err := <-results; {
		case err == nil:
			succeeded++
		case errors.Is(err, store.ErrConflict):
			conflicted++
		default:
			t.Fatal(err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("successes=%d conflicts=%d", succeeded, conflicted)
	}
	var count, amount int
	if err := e.db.QueryRow(ctx, `SELECT count(*), sum(amount) FROM payouts`).Scan(&count, &amount); err != nil {
		t.Fatal(err)
	}
	if count != 1 || amount != 10800 {
		t.Fatalf("payouts=%d amount=%d", count, amount)
	}
}

func TestLatePaymentKeepsReservedPlaces(t *testing.T) {
	e := setup(t, &provider.Fake{SiteURL: "https://ermekten.kg"})
	_, body := e.do("POST", "/orders", order(smallTour, 3), tourist)
	id := obj(t, body)["id"].(string)
	e.fake.Complete(id, true)
	*e.clock = e.clock.Add(45 * time.Minute)
	if status, body := e.do("POST", "/orders", order(smallTour, 1), tourist); status != 409 {
		t.Fatalf("sold a reserved place: %d %s", status, body)
	}
	e.worker.Tick(context.Background())
	if _, body := e.do("GET", "/orders/"+id, "", tourist); obj(t, body)["status"] != "paid" {
		t.Fatalf("late payment: %s", body)
	}
	if _, body := e.do("GET", "/orders/places?ids="+smallTour, "", ""); body != `{"`+smallTour+`":3}` {
		t.Fatalf("paid places: %s", body)
	}
}

func TestMissingPaymentLinkIsReconciled(t *testing.T) {
	for _, paid := range []bool{false, true} {
		e := setup(t, &provider.Fake{SiteURL: "https://ermekten.kg"})
		_, body := e.do("POST", "/orders", order(smallTour, 3), tourist)
		id := obj(t, body)["id"].(string)
		// Simulate a crash before the bank's URL was saved.
		if _, err := e.db.Exec(context.Background(), `UPDATE orders SET payment_url = NULL WHERE id = $1::uuid`, id); err != nil {
			t.Fatal(err)
		}
		want := "expired"
		if paid {
			e.fake.Complete(id, true)
			want = "paid"
		}
		*e.clock = e.clock.Add(45 * time.Minute)
		e.worker.Tick(context.Background())
		if _, body := e.do("GET", "/orders/"+id, "", tourist); obj(t, body)["status"] != want {
			t.Fatalf("missing link: %s, want %s", body, want)
		}
	}
}
