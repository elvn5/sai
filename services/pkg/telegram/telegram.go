// Package telegram sends messages through the Telegram Bot API.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DefaultAPIURL is the Bot API endpoint.
const DefaultAPIURL = "https://api.telegram.org"

// ErrUndeliverable: the chat can't receive messages from the bot (the user
// blocked it, never started it, or the chat is gone). Retrying won't help.
var ErrUndeliverable = errors.New("telegram chat unreachable")

// RateLimitedError: Telegram asked to retry after a pause.
type RateLimitedError struct{ After time.Duration }

func (e *RateLimitedError) Error() string {
	return fmt.Sprintf("telegram rate limit, retry after %s", e.After)
}

// Button is an inline keyboard button that opens a URL.
type Button struct {
	Text string
	URL  string
}

type Client struct {
	token  string
	apiURL string
	http   *http.Client
}

// New returns a client for the bot with token. apiURL "" means the real
// Bot API (tests point it at a fake server).
func New(token, apiURL string) *Client {
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}
	return &Client{token: token, apiURL: strings.TrimRight(apiURL, "/"), http: &http.Client{Timeout: 15 * time.Second}}
}

type sendMessageRequest struct {
	ChatID             int64           `json:"chat_id"`
	Text               string          `json:"text"`
	ParseMode          string          `json:"parse_mode"`
	LinkPreviewOptions map[string]bool `json:"link_preview_options"`
	ReplyMarkup        *inlineKeyboard `json:"reply_markup,omitempty"`
}

type inlineKeyboard struct {
	InlineKeyboard [][]inlineButton `json:"inline_keyboard"`
}

type inlineButton struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
	Parameters  struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

// SendMessage sends HTML-formatted text (escape user input with
// EscapeHTML), with an optional button.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, button *Button) error {
	req := sendMessageRequest{
		ChatID: chatID, Text: text, ParseMode: "HTML",
		LinkPreviewOptions: map[string]bool{"is_disabled": true},
	}
	if button != nil {
		req.ReplyMarkup = &inlineKeyboard{InlineKeyboard: [][]inlineButton{{{Text: button.Text, URL: button.URL}}}}
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL+"/bot"+c.token+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(httpReq)
	if err != nil {
		// The URL contains the token; never let it reach the logs.
		return errors.New("telegram request failed")
	}
	defer func() { _ = res.Body.Close() }()
	var out apiResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return fmt.Errorf("telegram: HTTP %d", res.StatusCode)
	}
	switch {
	case out.OK:
		return nil
	case out.ErrorCode == http.StatusTooManyRequests:
		return &RateLimitedError{After: time.Duration(out.Parameters.RetryAfter) * time.Second}
	case out.ErrorCode == http.StatusForbidden ||
		(out.ErrorCode == http.StatusBadRequest && strings.Contains(out.Description, "chat not found")):
		return fmt.Errorf("%w: %s", ErrUndeliverable, out.Description)
	default:
		return fmt.Errorf("telegram: %d %s", out.ErrorCode, out.Description)
	}
}

var htmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// EscapeHTML makes text safe inside an HTML-mode message.
func EscapeHTML(s string) string { return htmlEscaper.Replace(s) }
