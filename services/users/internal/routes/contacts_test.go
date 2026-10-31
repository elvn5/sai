package routes_test

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

// telegramSession signs in with the Telegram widget, which makes a
// Telegram-only account on first contact.
func (e *env) telegramSession(id, username string) response {
	e.t.Helper()
	authDate := strconv.FormatInt(e.clock.Unix(), 10)
	fields := map[string]string{"id": id, "first_name": "Ermek", "username": username, "auth_date": authDate}
	body := `{"id":` + id + `,"first_name":"Ermek","username":"` + username + `","auth_date":` + authDate + `,"hash":"` + signWidget(fields) + `"}`
	r := e.do("POST", "/auth/telegram/widget", body)
	if r.status != 200 {
		e.t.Fatalf("telegram sign-in: %d %s", r.status, r.body)
	}
	return r
}

func TestAddEmailToTelegramAccount(t *testing.T) {
	e := setup(t)
	session := e.telegramSession("111", "solo").cookie(t, "user_token")

	if r := e.do("POST", "/auth/user/email", `{"email":"solo@x.kg"}`); r.status != 401 {
		t.Errorf("anonymous: %d", r.status)
	}
	if r := e.do("POST", "/auth/user/email/confirm", `{"code":"123456"}`, session); r.status != 400 || !strings.Contains(r.body, "No verification code pending") {
		t.Errorf("confirm without a code: %d %s", r.status, r.body)
	}
	r := e.do("POST", "/auth/user/email", `{"email":"solo@x.kg"}`, session)
	if r.status != 200 || !strings.Contains(r.body, `"method":"email"`) || e.email.codes["solo@x.kg"] == "" {
		t.Fatalf("add email: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/auth/user/email", `{"email":"solo@x.kg"}`, session); r.status != 429 {
		t.Errorf("resend at once: %d", r.status)
	}
	if r := e.do("POST", "/auth/user/email/confirm", `{"code":"000000"}`, session); r.status != 401 {
		t.Errorf("wrong code: %d %s", r.status, r.body)
	}
	r = e.do("POST", "/auth/user/email/confirm", `{"code":"`+e.email.codes["solo@x.kg"]+`"}`, session)
	if r.status != 200 || !strings.Contains(r.body, `"email":"solo@x.kg"`) || !strings.Contains(r.body, `"merged":false`) || !strings.Contains(r.body, `"telegramId":"111"`) {
		t.Fatalf("confirm: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/auth/user/email", `{"email":"other@x.kg"}`, session); r.status != 409 {
		t.Errorf("a second email: %d %s", r.status, r.body)
	}
}

// The duplicate from the admin panel: an email account and a Telegram
// account of the same person. Confirming the email from the Telegram
// account merges it into the email one.
func TestEmailOfAnotherAccountMerges(t *testing.T) {
	e := setup(t)
	_, siteID := e.signUp("ermek@x.kg")
	session := e.telegramSession("434413828", "zetxc").cookie(t, "user_token")

	if r := e.do("POST", "/auth/user/email", `{"email":"ermek@x.kg"}`, session); r.status != 200 {
		t.Fatalf("add taken email: %d %s", r.status, r.body)
	}
	r := e.do("POST", "/auth/user/email/confirm", `{"code":"`+e.email.codes["ermek@x.kg"]+`"}`, session)
	if r.status != 200 || !strings.Contains(r.body, `"merged":true`) || !strings.Contains(r.body, `"id":"`+siteID+`"`) ||
		!strings.Contains(r.body, `"telegramId":"434413828"`) || !strings.Contains(r.body, `"username":"zetxc"`) {
		t.Fatalf("merge: %d %s", r.status, r.body)
	}
	merged := r.cookie(t, "user_token")
	if r := e.do("GET", "/auth/user/me", "", merged); !strings.Contains(r.body, `"id":"`+siteID+`"`) {
		t.Errorf("session after merge: %s", r.body)
	}
	// The Telegram account is gone: its old session no longer works, and
	// signing in with Telegram now opens the email account.
	if r := e.do("GET", "/auth/user/me", "", session); r.status != 401 {
		t.Errorf("old session: %d %s", r.status, r.body)
	}
	*e.clock = e.clock.Add(time.Minute)
	if r := e.telegramSession("434413828", "zetxc"); !strings.Contains(r.body, `"id":"`+siteID+`"`) {
		t.Errorf("telegram sign-in after merge: %s", r.body)
	}

	// An account with a password is never merged away by an email.
	other, _ := e.signUp("other@x.kg")
	if r := e.do("POST", "/auth/user/email", `{"email":"ermek@x.kg"}`, other); r.status != 409 {
		t.Errorf("full account: %d %s", r.status, r.body)
	}
}

func TestAdminMergesDuplicates(t *testing.T) {
	e := setup(t)
	_, siteID := e.signUp("dup@x.kg")
	shell := e.telegramSession("777", "dup")
	shellID := between(shell.body, `"id":"`, `"`)
	admin := e.do("POST", "/auth/login", `{"password":"admin-pass"}`).cookie(t, "admin_token")

	if r := e.do("POST", "/users/"+siteID+"/merge", `{"intoId":"`+shellID+`"}`, admin); r.status != 409 {
		t.Errorf("merging the email account away: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/users/"+shellID+"/merge", `{"intoId":"`+shellID+`"}`, admin); r.status != 400 {
		t.Errorf("into itself: %d %s", r.status, r.body)
	}
	r := e.do("POST", "/users/"+shellID+"/merge", `{"intoId":"`+siteID+`"}`, admin)
	if r.status != 200 || !strings.Contains(r.body, `"telegramId":"777"`) || !strings.Contains(r.body, `"email":"dup@x.kg"`) {
		t.Fatalf("merge: %d %s", r.status, r.body)
	}
	if r := e.do("GET", "/users/"+shellID, "", admin); r.status != 404 {
		t.Errorf("merged account still listed: %d", r.status)
	}
}

func between(s, start, end string) string {
	_, after, _ := strings.Cut(s, start)
	value, _, _ := strings.Cut(after, end)
	return value
}
