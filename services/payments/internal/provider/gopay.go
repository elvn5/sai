package provider

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// GoPay is GoPay's ELQR payments API (https://doc.gopay.kg/v1/): a dynamic
// QR payment per order with a checkout page, its state read through
// /v1/payments/query and pushed to the events_url webhook. Test and Live
// share one host; the key pair picks the mode (Test keys on staging, Live
// keys in production). The API has no refunds: only an unpaid payment can
// be cancelled.
type GoPay struct {
	BaseURL   string
	APIKey    string
	SecretKey string
	// WebhookSecret signs events_url deliveries (Developer → Webhooks).
	WebhookSecret string
	HTTP          *http.Client
}

func (g *GoPay) Name() string { return "gopay" }

// GoPay's limits on a payment.
const (
	gopayMaxOrderID     = 32
	gopayMaxDescription = 255
	gopayMinLifetime    = 300
	gopayMaxLifetime    = 86400
)

// Codes in GoPay's answers.
const (
	gopayOK = "0000"
	// The payment is in a terminal state (or the bank is settling it).
	gopayCannotCancel = "0013"
	gopayNotFound     = "4041"
)

// ErrNotCancellable: the payment can't be cancelled any more (it's paid or
// already closed).
var ErrNotCancellable = errors.New("the payment can't be cancelled")

// GoPayOrderID is the order id GoPay knows an order by: GoPay allows 32
// characters, so it's the order's uuid without dashes.
func GoPayOrderID(orderID string) string { return strings.ReplaceAll(orderID, "-", "") }

func (g *GoPay) CreatePayment(ctx context.Context, p Payment) (Checkout, error) {
	orderID := GoPayOrderID(p.TransactionID)
	if len(orderID) > gopayMaxOrderID {
		return Checkout{}, fmt.Errorf("gopay: order id %q is too long", orderID)
	}
	body := map[string]any{
		"order_id":    orderID,
		"amount":      strconv.Itoa(p.Amount) + ".00",
		"description": truncate(p.Comment, gopayMaxDescription),
		"lifetime":    min(max(int(math.Ceil(p.TTL.Seconds())), gopayMinLifetime), gopayMaxLifetime),
	}
	// GoPay only takes https return addresses (not a local site).
	if strings.HasPrefix(p.ReturnURL, "https://") {
		body["success_url"], body["failure_url"] = p.ReturnURL, p.ReturnURL
	}
	var out gopayPayment
	if err := g.call(ctx, "/v1/payments", body, &out); err != nil {
		return Checkout{}, err
	}
	if out.CheckoutURL == "" {
		return Checkout{}, errors.New("gopay: no checkout_url in the response")
	}
	details := out.details()
	return Checkout{URL: out.CheckoutURL, Details: &details}, nil
}

func (g *GoPay) Status(ctx context.Context, transactionID string) (Result, error) {
	var out gopayPayment
	if err := g.call(ctx, "/v1/payments/query", map[string]any{"order_id": GoPayOrderID(transactionID)}, &out); err != nil {
		return Result{}, err
	}
	return out.result(), nil
}

// Cancel closes an unpaid payment (its QR stops working); ErrNotCancellable
// once it's paid or closed.
func (g *GoPay) Cancel(ctx context.Context, transactionID string) (Result, error) {
	var out gopayPayment
	err := g.call(ctx, "/v1/payments/cancel", map[string]any{"order_id": GoPayOrderID(transactionID)}, &out)
	var apiErr *GoPayError
	if errors.As(err, &apiErr) && apiErr.Code == gopayCannotCancel {
		return Result{}, fmt.Errorf("%w: %s", ErrNotCancellable, apiErr.Message)
	}
	if err != nil {
		return Result{}, err
	}
	return out.result(), nil
}

// GoPayError is a FAIL answer from GoPay.
type GoPayError struct {
	Code    string
	Message string
}

func (e *GoPayError) Error() string { return "gopay: " + e.Code + " " + e.Message }

// NotFound reports whether GoPay has no such payment.
func (e *GoPayError) NotFound() bool { return e.Code == gopayNotFound }

// gopayPayment is GoPay's payment object (PaymentOutput).
type gopayPayment struct {
	PaymentID   string  `json:"payment_id"`
	OrderID     string  `json:"order_id"`
	Amount      string  `json:"amount"`
	Status      string  `json:"status"`
	TestingMode bool    `json:"testing_mode"`
	CheckoutURL string  `json:"checkout_url"`
	CreatedAt   string  `json:"created_at"`
	ExpiresAt   string  `json:"expires_at"`
	CommittedAt *string `json:"committed_at"`
	BankOpDate  *string `json:"bank_op_date"`
}

func (p gopayPayment) details() Details {
	d := Details{
		PaymentID: p.PaymentID, Status: strings.ToUpper(p.Status), Amount: p.Amount, Testing: p.TestingMode,
		CreatedAt: parseGoPayTime(p.CreatedAt), ExpiresAt: parseGoPayTime(p.ExpiresAt), BankOpDate: p.BankOpDate,
	}
	if p.CheckoutURL != "" {
		d.CheckoutURL = &p.CheckoutURL
	}
	if p.CommittedAt != nil {
		d.CommittedAt = parseGoPayTime(*p.CommittedAt)
	}
	return d
}

func (p gopayPayment) result() Result {
	amount, _ := strconv.ParseFloat(p.Amount, 64)
	d := p.details()
	r := Result{Amount: amount, Details: &d}
	switch d.Status {
	case "COMMITTED":
		r.State = StatePaid
	case "FAILED":
		r.State, r.Message = StateFailed, "Declined by the bank"
	case "CANCELLED":
		r.State, r.Message = StateFailed, "Cancelled"
	case "EXPIRED":
		r.State = StateExpired
	default: // CREATED, PENDING: still waiting
		r.State = StatePending
	}
	return r
}

// parseGoPayTime reads GoPay's times: UTC with Z in webhooks, and without a
// zone in create/query answers (UTC as well).
func parseGoPayTime(s string) *time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}

// call signs and sends a request; GoPay answers HTTP 200 with
// {status, code, data, error_message}.
func (g *GoPay) call(ctx context.Context, path string, body any, out any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	nonceHex := hex.EncodeToString(nonce)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(g.BaseURL, "/")+path, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("GoPay-Api-Key", g.APIKey)
	req.Header.Set("GoPay-Nonce", nonceHex)
	req.Header.Set("GoPay-Signature", GoPaySignature(g.SecretKey, nonceHex, encoded))
	client := g.HTTP
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("gopay %s: %w", path, err)
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode >= 300 {
		return fmt.Errorf("gopay %s: HTTP %d: %.200s", path, res.StatusCode, raw)
	}
	var envelope struct {
		Status       string          `json:"status"`
		Code         string          `json:"code"`
		ErrorMessage string          `json:"error_message"`
		Data         json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("gopay %s: unexpected response: %.200s", path, raw)
	}
	if envelope.Status != "OK" || (envelope.Code != "" && envelope.Code != gopayOK) {
		return &GoPayError{Code: envelope.Code, Message: envelope.ErrorMessage}
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("gopay %s: unexpected data: %.200s", path, envelope.Data)
	}
	return nil
}

// GoPaySignature is GoPay's request and webhook signature:
// HMAC-SHA512(secret, nonce + "\n" + body + "\n") in uppercase hex.
func GoPaySignature(secret, nonce string, body []byte) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write([]byte(nonce + "\n"))
	mac.Write(body)
	mac.Write([]byte("\n"))
	return strings.ToUpper(hex.EncodeToString(mac.Sum(nil)))
}

// VerifyWebhook checks an events_url delivery's GoPay-Signature.
func (g *GoPay) VerifyWebhook(nonce, signature string, body []byte) bool {
	if g.WebhookSecret == "" || nonce == "" || signature == "" {
		return false
	}
	want := GoPaySignature(g.WebhookSecret, nonce, body)
	return hmac.Equal([]byte(want), []byte(strings.ToUpper(strings.TrimSpace(signature))))
}

// GoPayEvent is an events_url delivery ({event, data, created_at}); the
// legacy callback_url body (the payment itself) reads as one too.
type GoPayEvent struct {
	Event     string
	PaymentID string
	OrderID   string
	Status    string
	Testing   bool
}

// ParseGoPayEvent reads a webhook body.
func ParseGoPayEvent(body []byte) (GoPayEvent, error) {
	var envelope struct {
		Event string        `json:"event"`
		Data  *gopayPayment `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return GoPayEvent{}, err
	}
	data := envelope.Data
	if envelope.Event == "" {
		var legacy gopayPayment
		if err := json.Unmarshal(body, &legacy); err != nil {
			return GoPayEvent{}, err
		}
		if legacy.PaymentID == "" {
			return GoPayEvent{}, errors.New("not a GoPay event")
		}
		data = &legacy
		envelope.Event = "payment." + strings.ToLower(legacy.Status)
	}
	e := GoPayEvent{Event: envelope.Event}
	if data != nil {
		e.PaymentID, e.OrderID, e.Status, e.Testing = data.PaymentID, data.OrderID, strings.ToUpper(data.Status), data.TestingMode
	}
	return e, nil
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}
