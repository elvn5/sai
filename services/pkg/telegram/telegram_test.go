package telegram_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/elvn5/sai/services/pkg/telegram"
)

func TestSendMessage(t *testing.T) {
	var got map[string]any
	reply := `{"ok":true}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/botTOKEN/sendMessage" {
			t.Errorf("path %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(reply))
	}))
	defer srv.Close()
	c := telegram.New("TOKEN", srv.URL)
	ctx := context.Background()

	if err := c.SendMessage(ctx, 42, "<b>Hi</b> "+telegram.EscapeHTML("a<b>&c"), &telegram.Button{Text: "Open", URL: "https://x.kg/o/1"}); err != nil {
		t.Fatal(err)
	}
	if got["chat_id"] != float64(42) || got["text"] != "<b>Hi</b> a&lt;b&gt;&amp;c" || got["parse_mode"] != "HTML" {
		t.Errorf("request: %v", got)
	}
	if kb := got["reply_markup"].(map[string]any)["inline_keyboard"].([]any)[0].([]any)[0].(map[string]any); kb["url"] != "https://x.kg/o/1" || kb["callback_data"] != nil {
		t.Errorf("button: %v", kb)
	}

	// A button pressing back to the bot carries callback data, no URL.
	if err := c.SendMessage(ctx, 42, "x", &telegram.Button{Text: "Open", Data: "o:view:n:1"}); err != nil {
		t.Fatal(err)
	}
	if kb := got["reply_markup"].(map[string]any)["inline_keyboard"].([]any)[0].([]any)[0].(map[string]any); kb["callback_data"] != "o:view:n:1" || kb["url"] != nil {
		t.Errorf("callback button: %v", kb)
	}

	reply = `{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`
	if err := c.SendMessage(ctx, 42, "x", nil); !errors.Is(err, telegram.ErrUndeliverable) {
		t.Errorf("blocked: %v", err)
	}
	reply = `{"ok":false,"error_code":400,"description":"Bad Request: chat not found"}`
	if err := c.SendMessage(ctx, 42, "x", nil); !errors.Is(err, telegram.ErrUndeliverable) {
		t.Errorf("chat not found: %v", err)
	}
	reply = `{"ok":false,"error_code":429,"description":"Too Many Requests","parameters":{"retry_after":7}}`
	var rl *telegram.RateLimitedError
	if err := c.SendMessage(ctx, 42, "x", nil); !errors.As(err, &rl) || rl.After != 7*time.Second {
		t.Errorf("rate limit: %v", err)
	}
	reply = `{"ok":false,"error_code":500,"description":"Internal"}`
	if err := c.SendMessage(ctx, 42, "x", nil); err == nil || errors.Is(err, telegram.ErrUndeliverable) {
		t.Errorf("server error: %v", err)
	}
}
