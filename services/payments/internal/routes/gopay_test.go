package routes_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/elvn5/sai/services/payments/internal/provider"
)

// fakeGoPay is GoPay's API: payments by order_id, their status set by the test.
type fakeGoPay struct {
	mu       sync.Mutex
	statuses map[string]string
	queries  int
}

func (f *fakeGoPay) set(orderID, status string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.statuses[provider.GoPayOrderID(orderID)] = status
}

func (f *fakeGoPay) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var req map[string]any
	_ = json.NewDecoder(r.Body).Decode(&req)
	id, _ := req["order_id"].(string)
	switch r.URL.Path {
	case "/v1/payments":
		f.statuses[id] = "CREATED"
	case "/v1/payments/query":
		f.queries++
	case "/v1/payments/cancel":
		if f.statuses[id] != "CREATED" {
			_, _ = io.WriteString(w, `{"status":"FAIL","code":"0013","error_message":"Payment is already `+f.statuses[id]+`"}`)
			return
		}
		f.statuses[id] = "CANCELLED"
	}
	status, ok := f.statuses[id]
	if !ok {
		_, _ = io.WriteString(w, `{"status":"FAIL","code":"4041","error_message":"Payment not found"}`)
		return
	}
	amount := "24000.00"
	if a, ok := req["amount"].(string); ok {
		amount = a
	}
	_, _ = io.WriteString(w, `{"status":"OK","code":"0000","data":{"payment_id":"pay-`+id[:8]+`","order_id":"`+id+
		`","amount":"`+amount+`","status":"`+status+`","testing_mode":true,"checkout_url":"https://pay.gopay.kg/p/`+id+
		`/","created_at":"2026-10-12T06:00:00","expires_at":"2026-10-12T06:30:00","committed_at":null}}`)
}

const webhookSecret = "whsec-test"

func (e *env) webhook(body, signature string) int {
	e.t.Helper()
	req := httptest.NewRequest("POST", "/payments/gopay/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("GoPay-Nonce", "n1")
	if signature == "" {
		signature = provider.GoPaySignature(webhookSecret, "n1", []byte(body))
	}
	req.Header.Set("GoPay-Signature", signature)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec.Code
}

func committed(orderID string) string {
	id := provider.GoPayOrderID(orderID)
	return `{"event":"payment.committed","data":{"payment_id":"pay-` + id[:8] + `","order_id":"` + id +
		`","status":"COMMITTED","amount":"24000.00","testing_mode":true},"created_at":"2026-10-12T06:05:00Z"}`
}

func TestGoPayWebhookAndAdmin(t *testing.T) {
	api := &fakeGoPay{statuses: map[string]string{}}
	srv := httptest.NewServer(api)
	defer srv.Close()
	e := setup(t, &provider.GoPay{BaseURL: srv.URL, APIKey: "key", SecretKey: "secret", WebhookSecret: webhookSecret})

	status, body := e.do("POST", "/orders", order(tourID, 2), tourist)
	if status != 201 || obj(t, body)["paymentUrl"] == nil || !strings.HasPrefix(obj(t, body)["paymentUrl"].(string), "https://pay.gopay.kg/") {
		t.Fatalf("create: %d %s", status, body)
	}
	orderID := obj(t, body)["id"].(string)

	if status, _ := e.do("GET", "/payments/admin/transactions", "", tourist); status != 401 {
		t.Errorf("a user sees transactions: %d", status)
	}
	status, body = e.do("GET", "/payments/admin/transactions", "", "admin")
	var list []map[string]any
	_ = json.Unmarshal([]byte(body), &list)
	if status != 200 || len(list) != 1 || list[0]["status"] != "CREATED" || list[0]["testing"] != true ||
		list[0]["providerOrderId"] != provider.GoPayOrderID(orderID) || list[0]["order"].(map[string]any)["payer"] == nil {
		t.Fatalf("list: %d %s", status, body)
	}

	// A forged or broken delivery changes nothing.
	if code := e.webhook(committed(orderID), "BAD"); code != 401 {
		t.Errorf("bad signature: %d", code)
	}
	if code := e.webhook(`{"event":"test.ping","data":{}}`, ""); code != 204 {
		t.Errorf("ping: %d", code)
	}

	// The webhook settles the order from GoPay's own answer.
	api.set(orderID, "COMMITTED")
	if code := e.webhook(committed(orderID), ""); code != 204 {
		t.Fatalf("webhook: %d", code)
	}
	_, body = e.do("GET", "/orders/"+orderID, "", tourist)
	if obj(t, body)["status"] != "paid" {
		t.Fatalf("not paid: %s", body)
	}
	queries := api.queries
	if code := e.webhook(committed(orderID), ""); code != 204 || api.queries != queries {
		t.Errorf("a repeated delivery was processed again: %d (queries %d → %d)", code, queries, api.queries)
	}
	status, body = e.do("GET", "/payments/admin/transactions/"+orderID, "", "admin")
	card := obj(t, body)
	events, _ := card["events"].([]any)
	if status != 200 || card["status"] != "COMMITTED" || len(events) != 1 || events[0].(map[string]any)["attempts"] != float64(2) ||
		events[0].(map[string]any)["processedAt"] == nil || card["order"].(map[string]any)["status"] != "paid" {
		t.Fatalf("card: %d %s", status, body)
	}
	// A paid one can't be cancelled.
	if status, _ := e.do("POST", "/payments/admin/transactions/"+orderID+"/cancel", "", "admin"); status != 409 {
		t.Errorf("cancel a paid one: %d", status)
	}

	// The admin cancels an unpaid one, which frees its places.
	_, body = e.do("POST", "/orders", order(tourID, 1), tourist)
	second := obj(t, body)["id"].(string)
	status, body = e.do("POST", "/payments/admin/transactions/"+second+"/cancel", "", "admin")
	if status != 200 || obj(t, body)["status"] != "CANCELLED" || obj(t, body)["order"].(map[string]any)["status"] != "failed" {
		t.Fatalf("cancel: %d %s", status, body)
	}

	// Checking at GoPay: an expired payment expires the order.
	_, body = e.do("POST", "/orders", order(tourID, 1), tourist)
	third := obj(t, body)["id"].(string)
	api.set(third, "EXPIRED")
	status, body = e.do("POST", "/payments/admin/transactions/"+third+"/check", "", "admin")
	if status != 200 || obj(t, body)["status"] != "EXPIRED" || obj(t, body)["order"].(map[string]any)["status"] != "expired" {
		t.Fatalf("check: %d %s", status, body)
	}

	for query, want := range map[string]int{"?status=COMMITTED": 1, "?status=CANCELLED": 1, "?from=2099-01-01": 0, "": 3} {
		_, body = e.do("GET", "/payments/admin/transactions"+query, "", "admin")
		list = nil
		_ = json.Unmarshal([]byte(body), &list)
		if len(list) != want {
			t.Errorf("%q: %d, want %d", query, len(list), want)
		}
	}
	if status, _ := e.do("GET", "/payments/admin/transactions?status=nope", "", "admin"); status != 400 {
		t.Errorf("bad status filter: %d", status)
	}
	status, body = e.do("GET", "/payments/admin/gateway", "", "admin")
	if status != 200 || obj(t, body)["provider"] != "gopay" || obj(t, body)["canCancel"] != true {
		t.Errorf("gateway: %d %s", status, body)
	}
}
