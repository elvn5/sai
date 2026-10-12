package routes_test

import (
	"context"
	"strings"
	"testing"
)

func TestTelegramLinkEndpoints(t *testing.T) {
	e := setup(t)
	session, id := e.signUp("link@x.kg")

	if r := e.do("POST", "/auth/user/telegram/link", ""); r.status != 401 {
		t.Errorf("anonymous: %d", r.status)
	}
	r := e.do("POST", "/auth/user/telegram/link", "", session)
	link := decode[struct {
		Token     string
		ExpiresAt string
	}](t, r.body)
	if r.status != 201 || len(link.Token) != 48 || link.ExpiresAt == "" {
		t.Fatalf("start: %d %s", r.status, r.body)
	}
	if r := e.do("DELETE", "/auth/user/telegram", "", session); r.status != 409 || !strings.Contains(r.body, "not linked") {
		t.Errorf("unlink unlinked: %d %s", r.status, r.body)
	}

	// Once linked (by the bot), a second link is refused and unlinking
	// works for an account with a password.
	if _, err := e.store.LinkTelegram(context.Background(), id, "4242", nil, ""); err != nil {
		t.Fatal(err)
	}
	if r := e.do("GET", "/auth/user/me", "", session); !strings.Contains(r.body, `"telegramId":"4242"`) {
		t.Errorf("me after link: %s", r.body)
	}
	if r := e.do("POST", "/auth/user/telegram/link", "", session); r.status != 409 {
		t.Errorf("link twice: %d", r.status)
	}
	if r := e.do("DELETE", "/auth/user/telegram", "", session); r.status != 200 || !strings.Contains(r.body, `"telegramId":null`) {
		t.Errorf("unlink: %d %s", r.status, r.body)
	}
}
