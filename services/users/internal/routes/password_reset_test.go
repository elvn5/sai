package routes_test

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"
)

// registerVerified creates a verified email account and returns its id.
func (e *env) registerVerified(email, password string) string {
	e.t.Helper()
	e.do("POST", "/auth/user/register", `{"email":"`+email+`","password":"`+password+`"}`)
	r := e.do("POST", "/auth/user/verify", `{"identifier":"`+email+`","code":"`+e.email.codes[email]+`"}`)
	var out struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal([]byte(r.body), &out); err != nil || out.User.ID == "" {
		e.t.Fatalf("verify %s: %d %s", email, r.status, r.body)
	}
	return out.User.ID
}

// resetToken is the token in the last link emailed to email ("" if none).
func (e *env) resetToken(email string) string {
	_, token, _ := strings.Cut(e.email.resets[email], "#")
	return token
}

func (e *env) resetWith(token, password string) response {
	return e.do("POST", "/auth/user/password/reset", `{"token":"`+token+`","password":"`+password+`"}`)
}

func TestPasswordResetByEmail(t *testing.T) {
	e := setup(t)
	e.registerVerified("aida@example.kg", "old-secret")
	oldSession := e.do("POST", "/auth/user/login", `{"identifier":"aida@example.kg","password":"old-secret"}`).cookie(t, "user_token")

	// An unknown email gets the same answer as a known one, and no mail.
	if r := e.do("POST", "/auth/user/password/forgot", `{"identifier":"nobody@example.kg"}`); r.status != 200 ||
		r.body != `{"channel":"email"}` || len(e.email.resets) != 0 {
		t.Errorf("unknown: %d %s %v", r.status, r.body, e.email.resets)
	}
	if r := e.do("POST", "/auth/user/password/forgot", `{"identifier":"aida@example.kg","lang":"en"}`); r.status != 200 ||
		r.body != `{"channel":"email"}` {
		t.Fatalf("forgot: %d %s", r.status, r.body)
	}
	if sent := e.email.resets["aida@example.kg"]; !strings.HasPrefix(sent, "en https://ermekten.test/en/reset-password#") {
		t.Fatalf("link: %q", sent)
	}
	token := e.resetToken("aida@example.kg")

	if r := e.resetWith("made-up-token", "new-secret"); r.status != 400 || !strings.Contains(r.body, "Reset link is invalid or has expired") {
		t.Errorf("unknown token: %d %s", r.status, r.body)
	}
	if r := e.resetWith(token, "short"); r.status != 400 || !strings.Contains(r.body, `"password"`) {
		t.Errorf("short password: %d %s", r.status, r.body)
	}
	// The old (code-based) request shape is refused.
	if r := e.do("POST", "/auth/user/password/reset", `{"identifier":"aida@example.kg","code":"123456","password":"new-secret"}`); r.status != 400 {
		t.Errorf("old shape: %d %s", r.status, r.body)
	}

	// Sessions issued before the reset end; the reset's own one works.
	time.Sleep(1100 * time.Millisecond) // token times are whole seconds
	r := e.resetWith(token, "new-secret")
	if r.status != 200 || len(r.setCookie) != 1 {
		t.Fatalf("reset: %d %s", r.status, r.body)
	}
	if me := e.do("GET", "/auth/user/me", "", oldSession); me.status != 401 {
		t.Errorf("old session still works: %d %s", me.status, me.body)
	}
	if me := e.do("GET", "/auth/user/me", "", r.cookie(t, "user_token")); me.status != 200 {
		t.Errorf("new session: %d %s", me.status, me.body)
	}
	if r := e.do("POST", "/auth/user/login", `{"identifier":"aida@example.kg","password":"old-secret"}`); r.status != 401 {
		t.Errorf("old password still works: %d", r.status)
	}
	if r := e.do("POST", "/auth/user/login", `{"identifier":"aida@example.kg","password":"new-secret"}`); r.status != 200 || len(r.setCookie) != 1 {
		t.Errorf("new password: %d %s", r.status, r.body)
	}
	// One use only.
	if r := e.resetWith(token, "another-one"); r.status != 400 {
		t.Errorf("token reused: %d %s", r.status, r.body)
	}
	// Tokens are stored hashed.
	var stored int
	if err := e.storeDB().QueryRow(context.Background(), `SELECT count(*) FROM password_reset_tokens WHERE token_hash = $1`, token).Scan(&stored); err != nil || stored != 0 {
		t.Errorf("plain token stored: %d %v", stored, err)
	}
}

func TestPasswordResetByTelegram(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	id := e.registerVerified("bek@example.kg", "old-secret")

	// No Telegram linked: the same answer, nothing sent.
	if r := e.do("POST", "/auth/user/password/forgot", `{"identifier":"bek@example.kg","channel":"telegram"}`); r.status != 200 ||
		r.body != `{"channel":"telegram"}` || len(e.tg.sent) != 0 {
		t.Fatalf("unlinked: %d %s %v", r.status, r.body, e.tg.sent)
	}
	if _, err := e.store.LinkTelegram(ctx, id, "777001", nil, ""); err != nil {
		t.Fatal(err)
	}
	if err := e.store.SetTelegramLanguage(ctx, 777001, "ky"); err != nil {
		t.Fatal(err)
	}
	*e.clock = e.clock.Add(2 * time.Minute)
	// The site is in English, but the chat reads Kyrgyz: the message and
	// the link follow the chat.
	if r := e.do("POST", "/auth/user/password/forgot", `{"identifier":"bek@example.kg","channel":"telegram","lang":"en"}`); r.status != 200 {
		t.Fatalf("forgot: %d %s", r.status, r.body)
	}
	if len(e.tg.sent) != 1 || len(e.email.resets) != 0 {
		t.Fatalf("sent: %v, emails %v", e.tg.sent, e.email.resets)
	}
	msg := e.tg.sent[0]
	if msg.chatID != 777001 || !strings.Contains(msg.text, "Сырсөздү калыбына келтирүү") ||
		!strings.HasPrefix(msg.button.URL, "https://ermekten.test/ky/reset-password#") {
		t.Fatalf("message: %+v", msg)
	}
	_, token, _ := strings.Cut(msg.button.URL, "#")
	if r := e.resetWith(token, "new-secret"); r.status != 200 {
		t.Fatalf("reset: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/auth/user/login", `{"identifier":"bek@example.kg","password":"new-secret"}`); r.status != 200 {
		t.Errorf("new password: %d %s", r.status, r.body)
	}
}

// A phone account resets through its Telegram (the default for a phone);
// a phone can't get an email.
func TestPasswordResetForPhoneAccount(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	if r := e.do("POST", "/auth/user/password/forgot", `{"identifier":"+996700000000","channel":"email"}`); r.status != 422 {
		t.Errorf("email to a phone: %d %s", r.status, r.body)
	}
	e.do("POST", "/auth/user/register", `{"phone":"+996700111222","password":"old-secret"}`)
	user, err := e.store.ByIdentifier(ctx, "+996700111222")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.store.LinkTelegram(ctx, user.ID, "777002", nil, ""); err != nil {
		t.Fatal(err)
	}
	if r := e.do("POST", "/auth/user/password/forgot", `{"identifier":"+996700111222"}`); r.status != 200 || r.body != `{"channel":"telegram"}` {
		t.Fatalf("forgot: %d %s", r.status, r.body)
	}
	if len(e.tg.sent) != 1 || !strings.HasPrefix(e.tg.sent[0].button.URL, "https://ermekten.test/reset-password#") {
		t.Fatalf("sent: %+v", e.tg.sent)
	}
	_, token, _ := strings.Cut(e.tg.sent[0].button.URL, "#")
	if r := e.resetWith(token, "new-secret"); r.status != 200 {
		t.Fatalf("reset: %d %s", r.status, r.body)
	}
	// The link proved the Telegram, so the account is verified now.
	if r := e.do("POST", "/auth/user/login", `{"identifier":"+996700111222","password":"new-secret"}`); r.status != 200 || len(r.setCookie) != 1 {
		t.Errorf("login: %d %s", r.status, r.body)
	}
}

// Requests are limited per identifier, known or not, with the same answer;
// a new link replaces the old one, and links expire.
func TestPasswordResetLimits(t *testing.T) {
	e := setup(t)
	e.registerVerified("cholpon@example.kg", "old-secret")
	forgot := func(identifier string) int {
		return e.do("POST", "/auth/user/password/forgot", `{"identifier":"`+identifier+`"}`).status
	}
	for _, identifier := range []string{"cholpon@example.kg", "ghost@example.kg"} {
		if status := forgot(identifier); status != 200 {
			t.Fatalf("%s: first request %d", identifier, status)
		}
		if status := forgot(identifier); status != 429 {
			t.Errorf("%s: again within a minute %d", identifier, status)
		}
	}
	first := e.resetToken("cholpon@example.kg")
	for range 4 {
		*e.clock = e.clock.Add(2 * time.Minute)
		if status := forgot("cholpon@example.kg"); status != 200 {
			t.Fatalf("spaced request: %d", status)
		}
	}
	*e.clock = e.clock.Add(2 * time.Minute)
	if status := forgot("cholpon@example.kg"); status != 429 {
		t.Errorf("sixth within an hour: %d", status)
	}
	if status := forgot("CHOLPON@example.kg"); status != 429 {
		t.Errorf("same email, other case: %d", status)
	}
	last := e.resetToken("cholpon@example.kg")
	if first == last {
		t.Fatal("the link didn't change")
	}
	if r := e.resetWith(first, "new-secret"); r.status != 400 {
		t.Errorf("replaced link: %d %s", r.status, r.body)
	}
	*e.clock = e.clock.Add(31 * time.Minute)
	if r := e.resetWith(last, "new-secret"); r.status != 400 {
		t.Errorf("expired link: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/auth/user/login", `{"identifier":"cholpon@example.kg","password":"old-secret"}`); r.status != 200 {
		t.Errorf("password changed: %d", r.status)
	}
}

func TestChangePassword(t *testing.T) {
	e := setup(t)
	e.registerVerified("bek@example.kg", "old-secret")
	login := func(password string) response {
		return e.do("POST", "/auth/user/login", `{"identifier":"bek@example.kg","password":"`+password+`"}`)
	}
	session := login("old-secret").cookie(t, "user_token")
	other := login("old-secret").cookie(t, "user_token")
	if me := e.do("GET", "/auth/user/me", "", session); !strings.Contains(me.body, `"hasPassword":true`) {
		t.Fatalf("me: %s", me.body)
	}

	if r := e.do("POST", "/auth/user/password/change", `{"currentPassword":"old-secret","password":"new-secret"}`); r.status != 401 {
		t.Errorf("signed out: %d", r.status)
	}
	if r := e.do("POST", "/auth/user/password/change", `{"currentPassword":"wrong","password":"new-secret"}`, session); r.status != 400 ||
		!strings.Contains(r.body, "Current password is incorrect") {
		t.Errorf("wrong current: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/auth/user/password/change", `{"password":"new-secret"}`, session); r.status != 400 {
		t.Errorf("no current: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/auth/user/password/change", `{"currentPassword":"old-secret","password":"short"}`, session); r.status != 400 ||
		!strings.Contains(r.body, `"password"`) {
		t.Errorf("short: %d %s", r.status, r.body)
	}

	// Other sessions end; this one goes on with a new cookie.
	time.Sleep(1100 * time.Millisecond) // token times are whole seconds
	r := e.do("POST", "/auth/user/password/change", `{"currentPassword":"old-secret","password":"new-secret"}`, session)
	if r.status != 200 || len(r.setCookie) != 1 {
		t.Fatalf("change: %d %s", r.status, r.body)
	}
	if me := e.do("GET", "/auth/user/me", "", other); me.status != 401 {
		t.Errorf("another session still works: %d", me.status)
	}
	if me := e.do("GET", "/auth/user/me", "", r.cookie(t, "user_token")); me.status != 200 {
		t.Errorf("new cookie: %d", me.status)
	}
	if login("old-secret").status != 401 || login("new-secret").status != 200 {
		t.Error("the new password isn't the one that works")
	}
}

func TestSetFirstPassword(t *testing.T) {
	e := setup(t)
	authDate := strconv.FormatInt(e.clock.Unix(), 10)
	fields := map[string]string{"id": "123456789", "first_name": "Ulan", "auth_date": authDate}
	body := `{"id":123456789,"first_name":"Ulan","auth_date":` + authDate + `,"hash":"` + signWidget(fields) + `"}`
	r := e.do("POST", "/auth/telegram/widget", body)
	if !strings.Contains(r.body, `"hasPassword":false`) {
		t.Fatalf("telegram account: %s", r.body)
	}
	// Nothing to sign in with but Telegram: no password to set.
	if r := e.do("POST", "/auth/user/password/change", `{"password":"first-secret"}`, r.cookie(t, "user_token")); r.status != 409 {
		t.Errorf("without email or phone: %d %s", r.status, r.body)
	}
}
