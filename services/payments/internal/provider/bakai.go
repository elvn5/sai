package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Bakai is Bakai Bank's Open Banking API
// (https://openbanking-api.bakai.kg/swagger): a payment link
// (PayLink/CreatePayLink) per order, its state polled through
// Qr/GetStateCustomQr — the API has no callbacks.
type Bakai struct {
	BaseURL  string
	Login    string
	Password string
	// qrType of payment links for GetStateCustomQr, as given by the bank.
	QRType string
	HTTP   *http.Client

	mu    sync.Mutex
	token string
}

func (b *Bakai) Name() string { return "bakai" }

// errUnauthorized makes a call log in again once.
var errUnauthorized = errors.New("bakai: unauthorized")

// ttlUnitMinutes is QrTtlUnits.Minutes.
const ttlUnitMinutes = 1

func (b *Bakai) CreatePayment(ctx context.Context, p Payment) (Checkout, error) {
	body := map[string]any{
		"amount":        p.Amount,
		"transactionID": p.TransactionID,
		"comment":       p.Comment,
		"redirectURL":   p.ReturnURL,
		"ttlUnits":      ttlUnitMinutes,
		"ttl":           int(math.Ceil(p.TTL.Minutes())),
		"qrVisible":     true,
	}
	raw, err := b.call(ctx, http.MethodPost, "/api/PayLink/CreatePayLink", body)
	if err != nil {
		return Checkout{}, err
	}
	link := findLink(raw)
	if link == "" {
		return Checkout{}, fmt.Errorf("bakai: no payment link in the response: %.200s", raw)
	}
	return Checkout{URL: link}, nil
}

func (b *Bakai) Status(ctx context.Context, transactionID string) (Result, error) {
	q := url.Values{"qrType": {b.QRType}, "transactionID": {transactionID}}
	raw, err := b.call(ctx, http.MethodGet, "/api/Qr/GetStateCustomQr?"+q.Encode(), nil)
	if err != nil {
		return Result{}, err
	}
	var res struct {
		State        string  `json:"state"`
		Amount       float64 `json:"amount"`
		ErrorMessage string  `json:"errorMessage"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return Result{}, fmt.Errorf("bakai: unexpected status response: %.200s", raw)
	}
	switch strings.ToLower(res.State) {
	case "success":
		return Result{State: StatePaid, Amount: res.Amount}, nil
	case "error":
		return Result{State: StateFailed, Amount: res.Amount, Message: res.ErrorMessage}, nil
	default: // "Processed": still in progress
		return Result{State: StatePending, Amount: res.Amount}, nil
	}
}

// call sends an authorized request, logging in first (and again once if
// the token has expired).
func (b *Bakai) call(ctx context.Context, method, path string, body any) ([]byte, error) {
	for attempt := 0; ; attempt++ {
		token, err := b.authToken(ctx)
		if err != nil {
			return nil, err
		}
		raw, err := b.do(ctx, method, path, body, token)
		if errors.Is(err, errUnauthorized) && attempt == 0 {
			b.mu.Lock()
			b.token = ""
			b.mu.Unlock()
			continue
		}
		return raw, err
	}
}

func (b *Bakai) authToken(ctx context.Context) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.token != "" {
		return b.token, nil
	}
	raw, err := b.do(ctx, http.MethodPost, "/Auth/Login", map[string]string{"login": b.Login, "password": b.Password}, "")
	if err != nil {
		return "", fmt.Errorf("bakai login: %w", err)
	}
	var res struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &res); err != nil || res.Token == "" {
		return "", errors.New("bakai login: no token in the response")
	}
	b.token = res.Token
	return b.token, nil
}

func (b *Bakai) do(ctx context.Context, method, path string, body any, token string) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(b.BaseURL, "/")+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := b.HTTP
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bakai %s: %w", path, err)
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	switch {
	case res.StatusCode == http.StatusUnauthorized:
		return nil, errUnauthorized
	case res.StatusCode >= 300:
		return nil, fmt.Errorf("bakai %s: HTTP %d: %.200s", path, res.StatusCode, raw)
	}
	return raw, nil
}

// linkKeys are where a payment link may sit in the CreatePayLink response,
// whose shape the API doesn't document.
var linkKeys = []string{"payLink", "paylink", "link", "url", "payUrl", "paymentUrl", "paymentLink", "qrLink"}

// findLink extracts the payment page URL from the CreatePayLink response:
// a bare or JSON string, or an object (possibly wrapped in data/result).
func findLink(raw []byte) string {
	text := strings.TrimSpace(string(raw))
	if strings.HasPrefix(text, "http") {
		return text
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return ""
	}
	return linkIn(v, 0)
}

func linkIn(v any, depth int) string {
	switch x := v.(type) {
	case string:
		if strings.HasPrefix(x, "http") {
			return x
		}
	case map[string]any:
		for _, key := range linkKeys {
			for k, val := range x {
				if strings.EqualFold(k, key) {
					if s, ok := val.(string); ok && strings.HasPrefix(s, "http") {
						return s
					}
				}
			}
		}
		if depth < 2 {
			for _, wrapper := range []string{"data", "result", "value"} {
				for k, val := range x {
					if strings.EqualFold(k, wrapper) {
						if link := linkIn(val, depth+1); link != "" {
							return link
						}
					}
				}
			}
		}
	}
	return ""
}
