// Package email sends verification codes and password reset links through
// Resend.
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/elvn5/sai/services/pkg/telegram"
	"github.com/elvn5/sai/services/users/internal/resetmsg"
)

// ErrNotConfigured: RESEND_API_KEY is missing.
var ErrNotConfigured = errors.New("email service is not configured")

type Sender interface {
	SendVerificationCode(ctx context.Context, to, code string) error
	// SendPasswordReset emails the link for setting a new password, in l.
	SendPasswordReset(ctx context.Context, to string, l telegram.Lang, link string) error
}

type Resend struct {
	APIKey, From, BaseURL string
	HTTP                  *http.Client
}

func NewResend(apiKey, from, baseURL string) *Resend {
	return &Resend{APIKey: apiKey, From: from, BaseURL: baseURL, HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func (r *Resend) SendVerificationCode(ctx context.Context, to, code string) error {
	return r.send(ctx, to, "Код подтверждения Ermekten",
		"<p>Ваш код подтверждения: <strong>"+code+"</strong></p><p>Код действителен 10 минут.</p>")
}

func (r *Resend) SendPasswordReset(ctx context.Context, to string, l telegram.Lang, link string) error {
	subject, body := resetmsg.Email(l, link)
	return r.send(ctx, to, subject, body)
}

func (r *Resend) send(ctx context.Context, to, subject, html string) error {
	if r.APIKey == "" {
		return ErrNotConfigured
	}
	body, err := json.Marshal(map[string]any{
		"from":    r.From,
		"to":      []string{to},
		"subject": subject,
		"html":    html,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.BaseURL+"/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.APIKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := r.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("resend request: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return fmt.Errorf("resend API error (%d): %s", res.StatusCode, detail)
	}
	return nil
}
