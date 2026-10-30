package provider_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/elvn5/sai/services/payments/internal/provider"
)

// The signature from GoPay's own shell example (printf nonce, data | openssl).
func TestGoPaySignature(t *testing.T) {
	got := provider.GoPaySignature("secret", "abc", []byte(`{"order_id":"ORDER-001","amount":"150.00"}`))
	if got != "3D66CCD9F4A2EC31B1D499A3CE32ADB73CDE98B01FDDE0FB3DFD6AA9A0101F5FC76FEC3097CB68EAA69ED6A543EC92E640AB1377DAFFBADE3FAEC18BB5ADC41C" {
		t.Fatalf("signature %q", got)
	}
	g := &provider.GoPay{WebhookSecret: "secret"}
	body := []byte(`{"order_id":"ORDER-001","amount":"150.00"}`)
	if !g.VerifyWebhook("abc", strings.ToLower(got), body) {
		t.Error("valid signature refused")
	}
	if g.VerifyWebhook("abd", got, body) || g.VerifyWebhook("abc", got, append(body, ' ')) || g.VerifyWebhook("abc", "", body) {
		t.Error("bad signature accepted")
	}
	if (&provider.GoPay{}).VerifyWebhook("abc", got, body) {
		t.Error("accepted without a webhook secret")
	}
}

func TestGoPay(t *testing.T) {
	var lastPath string
	var lastBody map[string]any
	answer := map[string]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || r.Header.Get("GoPay-Api-Key") != "key" ||
			r.Header.Get("GoPay-Signature") != provider.GoPaySignature("secret", r.Header.Get("GoPay-Nonce"), raw) {
			_, _ = w.Write([]byte(`{"status":"FAIL","code":"4001","error_message":"Invalid signature"}`))
			return
		}
		lastPath, lastBody = r.URL.Path, nil
		_ = json.Unmarshal(raw, &lastBody)
		_, _ = w.Write([]byte(answer[r.URL.Path]))
	}))
	defer srv.Close()
	g := &provider.GoPay{BaseURL: srv.URL, APIKey: "key", SecretKey: "secret"}
	ctx := context.Background()
	orderID := "0b7e5a1c-2d3e-4f50-8a6b-7c8d9e0f1a2b"
	compact := "0b7e5a1c2d3e4f508a6b7c8d9e0f1a2b"

	answer["/v1/payments"] = `{"status":"OK","code":"0000","data":{"payment_id":"p1","order_id":"` + compact + `","amount":"24000.00",
		"status":"CREATED","testing_mode":true,"checkout_url":"https://pay.gopay.kg/p/p1/","created_at":"2026-10-30T10:00:00",
		"expires_at":"2026-10-30T10:30:00","committed_at":null}}`
	checkout, err := g.CreatePayment(ctx, provider.Payment{TransactionID: orderID, Amount: 24000,
		Comment: strings.Repeat("я", 300), ReturnURL: "https://ermekten.com/payments/return?order=" + orderID, TTL: 2 * time.Minute})
	if err != nil || checkout.URL != "https://pay.gopay.kg/p/p1/" {
		t.Fatalf("create: %+v %v", checkout, err)
	}
	d := checkout.Details
	if d == nil || d.PaymentID != "p1" || d.Status != "CREATED" || !d.Testing || d.ExpiresAt == nil || d.ExpiresAt.Minute() != 30 {
		t.Errorf("details: %+v", d)
	}
	if lastBody["order_id"] != compact || lastBody["amount"] != "24000.00" || lastBody["lifetime"] != float64(300) ||
		len([]rune(lastBody["description"].(string))) != 255 || lastBody["success_url"] == nil {
		t.Errorf("request: %v", lastBody)
	}
	// A local site has no https return address.
	_, _ = g.CreatePayment(ctx, provider.Payment{TransactionID: orderID, Amount: 100, ReturnURL: "http://localhost:3012/x", TTL: time.Hour})
	if _, ok := lastBody["success_url"]; ok || lastBody["lifetime"] != float64(3600) {
		t.Errorf("local request: %v", lastBody)
	}

	for status, want := range map[string]provider.State{
		"CREATED": provider.StatePending, "COMMITTED": provider.StatePaid, "FAILED": provider.StateFailed,
		"CANCELLED": provider.StateFailed, "EXPIRED": provider.StateExpired,
	} {
		answer["/v1/payments/query"] = `{"status":"OK","code":"0000","data":{"payment_id":"p1","order_id":"` + compact +
			`","amount":"24000.00","status":"` + status + `","committed_at":"2026-10-30T10:05:00","bank_op_date":"2026-10-30"}}`
		res, err := g.Status(ctx, orderID)
		if err != nil || res.State != want || res.Amount != 24000 || res.Details.Status != status || res.Details.Testing {
			t.Errorf("%s: %+v %v", status, res, err)
		}
		if lastPath != "/v1/payments/query" || lastBody["order_id"] != compact {
			t.Errorf("query: %s %v", lastPath, lastBody)
		}
	}

	answer["/v1/payments/query"] = `{"status":"FAIL","code":"4041","error_message":"Payment not found"}`
	var apiErr *provider.GoPayError
	if _, err := g.Status(ctx, orderID); !errors.As(err, &apiErr) || !apiErr.NotFound() {
		t.Errorf("not found: %v", err)
	}

	answer["/v1/payments/cancel"] = `{"status":"OK","code":"0000","data":{"payment_id":"p1","order_id":"` + compact +
		`","amount":"24000.00","status":"CANCELLED"}}`
	if res, err := g.Cancel(ctx, orderID); err != nil || res.State != provider.StateFailed || res.Details.Status != "CANCELLED" {
		t.Errorf("cancel: %+v %v", res, err)
	}
	answer["/v1/payments/cancel"] = `{"status":"FAIL","code":"0013","error_message":"Payment is already COMMITTED"}`
	if _, err := g.Cancel(ctx, orderID); !errors.Is(err, provider.ErrNotCancellable) {
		t.Errorf("cancel a paid one: %v", err)
	}

	bad := &provider.GoPay{BaseURL: srv.URL, APIKey: "key", SecretKey: "wrong"}
	if _, err := bad.Status(ctx, orderID); !errors.As(err, &apiErr) || apiErr.Code != "4001" || strings.Contains(err.Error(), "wrong") {
		t.Errorf("bad secret: %v", err)
	}
}

func TestParseGoPayEvent(t *testing.T) {
	e, err := provider.ParseGoPayEvent([]byte(`{"event":"payment.committed","data":{"payment_id":"p1","order_id":"o1",
		"status":"COMMITTED","amount":"100.00","testing_mode":true},"created_at":"2026-09-03T04:00:00Z"}`))
	if err != nil || e.Event != "payment.committed" || e.PaymentID != "p1" || e.OrderID != "o1" || e.Status != "COMMITTED" || !e.Testing {
		t.Errorf("typed: %+v %v", e, err)
	}
	e, err = provider.ParseGoPayEvent([]byte(`{"payment_id":"p1","order_id":"o1","status":"COMMITTED","amount":"100.00"}`))
	if err != nil || e.Event != "payment.committed" || e.PaymentID != "p1" || e.Testing {
		t.Errorf("legacy: %+v %v", e, err)
	}
	if e, err := provider.ParseGoPayEvent([]byte(`{"event":"test.ping","data":{}}`)); err != nil || e.PaymentID != "" {
		t.Errorf("ping: %+v %v", e, err)
	}
	if _, err := provider.ParseGoPayEvent([]byte(`{"hello":1}`)); err == nil {
		t.Error("junk accepted")
	}
}
