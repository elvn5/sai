package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/elvn5/sai/services/payments/internal/provider"
)

func TestBakai(t *testing.T) {
	var logins atomic.Int32
	var linkBody, state string
	var created map[string]any
	expireToken := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/Auth/Login":
			var req map[string]string
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req["login"] != "ermekten" || req["password"] != "secret" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			n := logins.Add(1)
			_, _ = w.Write([]byte(`{"token":"tok` + strconv.Itoa(int(n)) + `"}`))
			return
		}
		if expireToken && r.Header.Get("Authorization") == "Bearer tok1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/PayLink/CreatePayLink":
			_ = json.NewDecoder(r.Body).Decode(&created)
			_, _ = w.Write([]byte(linkBody))
		case "/api/Qr/GetStateCustomQr":
			if r.URL.Query().Get("qrType") != "PayLink" || r.URL.Query().Get("transactionID") != "order-1" {
				t.Errorf("status query: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(state))
		}
	}))
	defer srv.Close()
	b := &provider.Bakai{BaseURL: srv.URL, Login: "ermekten", Password: "secret", QRType: "PayLink"}
	ctx := context.Background()
	payment := provider.Payment{TransactionID: "order-1", Amount: 24000, Comment: "Тур", ReturnURL: "https://x.kg/payments/return?order=order-1", TTL: 30 * time.Minute}

	for body, want := range map[string]string{
		`"https://pay.bakai.kg/l/abc"`:                         "https://pay.bakai.kg/l/abc",
		`https://pay.bakai.kg/l/abc`:                           "https://pay.bakai.kg/l/abc",
		`{"payLink":"https://pay.bakai.kg/l/abc"}`:             "https://pay.bakai.kg/l/abc",
		`{"data":{"url":"https://pay.bakai.kg/l/abc"},"ok":1}`: "https://pay.bakai.kg/l/abc",
	} {
		linkBody = body
		got, err := b.CreatePayment(ctx, payment)
		if err != nil || got.URL != want {
			t.Errorf("%s: %q %v", body, got.URL, err)
		}
	}
	if created["amount"] != float64(24000) || created["transactionID"] != "order-1" || created["ttl"] != float64(30) ||
		created["ttlUnits"] != float64(1) || !strings.Contains(created["redirectURL"].(string), "order-1") {
		t.Errorf("request: %v", created)
	}
	linkBody = `{"message":"ok"}`
	if _, err := b.CreatePayment(ctx, payment); err == nil {
		t.Error("missing link accepted")
	}
	if logins.Load() != 1 {
		t.Errorf("logged in %d times", logins.Load())
	}

	for body, want := range map[string]provider.State{
		`{"state":"Success","amount":24000}`:                     provider.StatePaid,
		`{"state":"Processed","amount":0}`:                       provider.StatePending,
		`{"state":"Error","amount":0,"errorMessage":"Declined"}`: provider.StateFailed,
	} {
		state = body
		res, err := b.Status(ctx, "order-1")
		if err != nil || res.State != want {
			t.Errorf("%s: %+v %v", body, res, err)
		}
	}

	// An expired token is replaced once.
	expireToken = true
	state = `{"state":"Success","amount":24000}`
	if res, err := b.Status(ctx, "order-1"); err != nil || res.State != provider.StatePaid || logins.Load() != 2 {
		t.Errorf("re-login: %+v %v (logins %d)", res, err, logins.Load())
	}

	bad := &provider.Bakai{BaseURL: srv.URL, Login: "ermekten", Password: "wrong"}
	if _, err := bad.Status(ctx, "order-1"); err == nil || strings.Contains(err.Error(), "wrong") {
		t.Errorf("bad credentials: %v", err)
	}
}
