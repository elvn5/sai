package routes_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/elvn5/sai/services/pkg/auth"
	"github.com/elvn5/sai/services/users/internal/handlers"
	"github.com/elvn5/sai/services/users/internal/routes"
	"github.com/elvn5/sai/services/users/internal/session"
	"github.com/elvn5/sai/services/users/internal/store"
	"github.com/elvn5/sai/services/users/internal/testdb"
)

const (
	secret   = "test-secret-at-least-16-chars"
	botToken = "123456:TEST-bot-token"
)

type fakeEmail struct {
	mu    sync.Mutex
	codes map[string]string
	fail  bool
}

func (f *fakeEmail) SendVerificationCode(_ context.Context, to, code string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return context.DeadlineExceeded
	}
	f.codes[to] = code
	return nil
}

type env struct {
	t      *testing.T
	router *gin.Engine
	email  *fakeEmail
	clock  *time.Time
	store  *store.Store
}

func setup(t *testing.T) *env {
	t.Helper()
	db, _ := testdb.New(t, true)
	signer, err := auth.NewSigner(secret, "12h")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	e := &env{t: t, email: &fakeEmail{codes: map[string]string{}}, clock: &now, store: store.New(db)}
	gin.SetMode(gin.TestMode)
	e.router = routes.Setup(handlers.Deps{
		Store: e.store, Signer: signer, Cookies: session.NewCookies(true), Email: e.email,
		AdminPassword: "admin-pass", BotToken: botToken, Log: slog.New(slog.DiscardHandler),
		Now: func() time.Time { return *e.clock },
	}, secret)
	return e
}

type response struct {
	status    int
	body      string
	setCookie []string
}

func (e *env) do(method, path, body string, cookies ...*http.Cookie) response {
	e.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return response{rec.Code, rec.Body.String(), rec.Header().Values("Set-Cookie")}
}

func (r response) cookie(t *testing.T, name string) *http.Cookie {
	t.Helper()
	for _, raw := range r.setCookie {
		if strings.HasPrefix(raw, name+"=") {
			value := strings.TrimPrefix(strings.SplitN(raw, ";", 2)[0], name+"=")
			return &http.Cookie{Name: name, Value: value}
		}
	}
	t.Fatalf("no %s cookie in %v", name, r.setCookie)
	return nil
}

func TestEmailRegistrationFlow(t *testing.T) {
	e := setup(t)
	r := e.do("POST", "/auth/user/register", `{"email":"aibek@example.kg","password":"secret123","firstName":"Aibek"}`)
	if r.status != 200 || r.body != `{"verification":{"identifier":"aibek@example.kg","method":"email"}}` {
		t.Fatalf("register: %d %s", r.status, r.body)
	}
	code := e.email.codes["aibek@example.kg"]
	if !regexp.MustCompile(`^\d{6}$`).MatchString(code) {
		t.Fatalf("code = %q", code)
	}

	if r := e.do("POST", "/auth/user/register", `{"email":"aibek@example.kg","password":"secret123"}`); r.status != 409 || !strings.Contains(r.body, "Email is already registered") {
		t.Errorf("duplicate: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/auth/user/login", `{"identifier":"aibek@example.kg","password":"secret123"}`); r.body != `{"verification":{"identifier":"aibek@example.kg","method":"email"}}` || len(r.setCookie) != 0 {
		t.Errorf("unverified login: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/auth/user/resend-code", `{"identifier":"aibek@example.kg"}`); r.status != 429 || r.body != `{"statusCode":429,"message":"Please wait before requesting another code"}` {
		t.Errorf("resend cooldown: %d %s", r.status, r.body)
	}
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	if r := e.do("POST", "/auth/user/verify", `{"identifier":"aibek@example.kg","code":"`+wrong+`"}`); r.status != 401 || !strings.Contains(r.body, "Invalid code") {
		t.Errorf("wrong code: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/auth/user/verify", `{"identifier":"aibek@example.kg","code":"12ab"}`); r.status != 400 || !strings.Contains(r.body, `"code":["Code must be 6 digits"]`) {
		t.Errorf("code format: %d %s", r.status, r.body)
	}

	r = e.do("POST", "/auth/user/verify", `{"identifier":"aibek@example.kg","code":"`+code+`"}`)
	if r.status != 200 || !strings.Contains(r.body, `"email":"aibek@example.kg"`) || !strings.Contains(r.body, `"partnerStatus":"none"`) {
		t.Fatalf("verify: %d %s", r.status, r.body)
	}
	if !regexp.MustCompile(`^user_token=[^;]+; Max-Age=2592000; Path=/; Expires=[^;]+ GMT; HttpOnly; Secure; SameSite=Lax$`).MatchString(r.setCookie[0]) {
		t.Errorf("cookie format: %q", r.setCookie[0])
	}
	session := r.cookie(t, "user_token")
	claims := jwt.MapClaims{}
	if _, err := jwt.ParseWithClaims(session.Value, claims, func(*jwt.Token) (any, error) { return []byte(secret), nil }); err != nil {
		t.Fatal(err)
	}
	if exp, iat := claims["exp"].(float64), claims["iat"].(float64); exp-iat != 12*3600 || claims["role"] != "user" {
		t.Errorf("claims: %v", claims)
	}

	if r := e.do("GET", "/auth/user/me", "", session); r.status != 200 || !strings.Contains(r.body, `"firstName":"Aibek"`) || strings.Contains(r.body, "password") {
		t.Errorf("me: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/auth/user/verify", `{"identifier":"aibek@example.kg","code":"`+code+`"}`); r.status != 409 {
		t.Errorf("verify twice: %d", r.status)
	}
	if r := e.do("POST", "/auth/user/login", `{"identifier":"aibek@example.kg","password":"wrong-pass"}`); r.status != 401 || !strings.Contains(r.body, "Invalid credentials") {
		t.Errorf("bad password: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/auth/user/login", `{"identifier":"aibek@example.kg","password":"secret123"}`); r.status != 200 || len(r.setCookie) != 1 {
		t.Errorf("login: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/auth/user/logout", ""); r.setCookie[0] != "user_token=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT" {
		t.Errorf("logout cookie: %v", r.setCookie)
	}
}

func TestVerificationLimits(t *testing.T) {
	e := setup(t)
	e.do("POST", "/auth/user/register", `{"email":"a@b.co","password":"secret123"}`)
	for range 5 {
		e.do("POST", "/auth/user/verify", `{"identifier":"a@b.co","code":"999999"}`)
	}
	if r := e.do("POST", "/auth/user/verify", `{"identifier":"a@b.co","code":"`+e.email.codes["a@b.co"]+`"}`); r.status != 400 || !strings.Contains(r.body, "Too many attempts") {
		t.Errorf("attempt limit: %d %s", r.status, r.body)
	}
	*e.clock = e.clock.Add(11 * time.Minute)
	if r := e.do("POST", "/auth/user/resend-code", `{"identifier":"a@b.co"}`); r.status != 200 {
		t.Fatalf("resend: %d %s", r.status, r.body)
	}
	*e.clock = e.clock.Add(11 * time.Minute)
	if r := e.do("POST", "/auth/user/verify", `{"identifier":"a@b.co","code":"`+e.email.codes["a@b.co"]+`"}`); r.status != 400 || !strings.Contains(r.body, "Code has expired") {
		t.Errorf("expired: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/auth/user/verify", `{"identifier":"nobody@b.co","code":"123456"}`); r.status != 404 || !strings.Contains(r.body, "Account not found") {
		t.Errorf("unknown: %d %s", r.status, r.body)
	}
}

func TestRegistrationValidationAndEmailFailure(t *testing.T) {
	e := setup(t)
	cases := map[string]string{
		`{"password":"secret123"}`:                                   `{"formErrors":[],"fieldErrors":{"email":["Email or phone is required"]}}`,
		`{"email":"not an email","password":"short","firstName":""}`: `{"formErrors":[],"fieldErrors":{"email":["Invalid email"],"firstName":["String must contain at least 1 character(s)"],"password":["String must contain at least 8 character(s)"]}}`,
		`{"email":"a..b@x.com","password":"secret123"}`:              `{"formErrors":[],"fieldErrors":{"email":["Invalid email"]}}`,
		`{"phone":"123","password":5}`:                               `{"formErrors":[],"fieldErrors":{"password":["Expected string, received number"],"phone":["String must contain at least 5 character(s)"]}}`,
	}
	for body, want := range cases {
		if r := e.do("POST", "/auth/user/register", body); r.status != 400 || r.body != want {
			t.Errorf("%s\n got: %d %s\nwant: %s", body, r.status, r.body, want)
		}
	}

	e.email.fail = true
	if r := e.do("POST", "/auth/user/register", `{"email":"x@y.kg","password":"secret123"}`); r.status != 500 || !strings.Contains(r.body, "Could not send verification email") {
		t.Errorf("email failure: %d %s", r.status, r.body)
	}
	if taken, _ := e.store.EmailTaken(context.Background(), "x@y.kg"); taken {
		t.Error("account kept after failed email")
	}
}

func TestPhoneRegistrationViaTelegramToken(t *testing.T) {
	e := setup(t)
	r := e.do("POST", "/auth/user/register", `{"phone":"+996555123456","password":"secret123"}`)
	var res struct{ Verification handlers.PendingVerification }
	_ = json.Unmarshal([]byte(r.body), &res)
	if res.Verification.Method != "telegram" || len(res.Verification.Token) != 48 {
		t.Fatalf("register: %s", r.body)
	}
	if r := e.do("GET", "/auth/user/verification-status?identifier=%2B996555123456", ""); r.body != `{"verified":false}` {
		t.Errorf("status: %s", r.body)
	}
	if r := e.do("GET", "/auth/user/verification-status", ""); r.status != 400 || r.body != `{"formErrors":[],"fieldErrors":{"identifier":["Required"]}}` {
		t.Errorf("status validation: %d %s", r.status, r.body)
	}
	if r := e.do("GET", "/auth/user/verification-status?identifier=nope", ""); r.status != 404 {
		t.Errorf("status unknown: %d", r.status)
	}
}

func signWidget(fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var lines []string
	for _, k := range keys {
		lines = append(lines, k+"="+fields[k])
	}
	key := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, key[:])
	mac.Write([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestTelegramWidget(t *testing.T) {
	e := setup(t)
	authDate := strconv.FormatInt(e.clock.Unix(), 10)
	fields := map[string]string{"id": "987654321", "first_name": "Anna", "username": "anna_k", "auth_date": authDate}
	body := `{"id":987654321,"first_name":"Anna","username":"anna_k","auth_date":` + authDate + `,"hash":"` + signWidget(fields) + `"}`

	r := e.do("POST", "/auth/telegram/widget", body)
	if r.status != 200 || !strings.Contains(r.body, `"telegramId":"987654321"`) || len(r.setCookie) != 1 {
		t.Fatalf("widget: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/auth/telegram/widget", strings.Replace(body, "anna_k", "evil", 1)); r.status != 401 || !strings.Contains(r.body, "Invalid Telegram signature") {
		t.Errorf("tampered: %d %s", r.status, r.body)
	}
	*e.clock = e.clock.Add(25 * time.Hour)
	if r := e.do("POST", "/auth/telegram/widget", body); r.status != 401 || !strings.Contains(r.body, "Telegram auth data has expired") {
		t.Errorf("stale: %d %s", r.status, r.body)
	}
}

func TestTelegramMiniApp(t *testing.T) {
	e := setup(t)
	user := `{"id":555,"first_name":"Bakyt","last_name":"","photo_url":"https://t.me/i/u.jpg"}`
	values := url.Values{"auth_date": {strconv.FormatInt(e.clock.Unix(), 10)}, "query_id": {"AAH"}, "user": {user}}
	var lines []string
	for k, v := range values {
		lines = append(lines, k+"="+v[0])
	}
	sort.Strings(lines)
	secretMac := hmac.New(sha256.New, []byte("WebAppData"))
	secretMac.Write([]byte(botToken))
	mac := hmac.New(sha256.New, secretMac.Sum(nil))
	mac.Write([]byte(strings.Join(lines, "\n")))
	values.Set("hash", hex.EncodeToString(mac.Sum(nil)))
	initData, _ := json.Marshal(map[string]string{"initData": values.Encode()})

	r := e.do("POST", "/auth/telegram/miniapp", string(initData))
	if r.status != 200 || !strings.Contains(r.body, `"telegramId":"555"`) || !strings.Contains(r.body, `"lastName":""`) || !strings.Contains(r.body, `"avatarUrl":"https://t.me/i/u.jpg"`) {
		t.Fatalf("miniapp: %d %s", r.status, r.body)
	}
	// A second sign-in reuses the account.
	r2 := e.do("POST", "/auth/telegram/miniapp", string(initData))
	if r.body != r2.body {
		t.Errorf("second sign-in differs:\n%s\n%s", r.body, r2.body)
	}
	if r := e.do("POST", "/auth/telegram/miniapp", `{"initData":"user=x&hash=00"}`); r.status != 401 || !strings.Contains(r.body, "Invalid Telegram Mini App signature") {
		t.Errorf("bad initData: %d %s", r.status, r.body)
	}
}

func TestAdminAndPartners(t *testing.T) {
	e := setup(t)
	if r := e.do("POST", "/auth/login", `{"password":""}`); r.status != 400 || r.body != `{"formErrors":[],"fieldErrors":{"password":["Password is required"]}}` {
		t.Errorf("empty password: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/auth/login", `{"password":"nope"}`); r.status != 401 || !strings.Contains(r.body, "Invalid password") {
		t.Errorf("wrong password: %d %s", r.status, r.body)
	}
	r := e.do("POST", "/auth/login", `{"password":"admin-pass"}`)
	if r.status != 200 || r.body != `{"success":true}` || !strings.Contains(r.setCookie[0], "Max-Age=43200") {
		t.Fatalf("admin login: %d %s %v", r.status, r.body, r.setCookie)
	}
	admin := r.cookie(t, "admin_token")
	if r := e.do("GET", "/auth/me", "", admin); r.body != `{"authenticated":true}` {
		t.Errorf("admin me: %s", r.body)
	}

	// A site user applies, the admin approves.
	e.do("POST", "/auth/user/register", `{"email":"p@x.kg","password":"secret123"}`)
	r = e.do("POST", "/auth/user/verify", `{"identifier":"p@x.kg","code":"`+e.email.codes["p@x.kg"]+`"}`)
	user := r.cookie(t, "user_token")
	if r := e.do("GET", "/users", "", user); r.status != 401 {
		t.Errorf("user listing users: %d", r.status)
	}
	if r := e.do("POST", "/partners/apply", "", user); r.status != 201 || !strings.Contains(r.body, `"partnerStatus":"pending"`) {
		t.Fatalf("apply: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/partners/apply", "", user); r.status != 409 {
		t.Errorf("apply twice: %d", r.status)
	}
	r = e.do("GET", "/partners/requests", "", admin)
	var pending []map[string]any
	_ = json.Unmarshal([]byte(r.body), &pending)
	if len(pending) != 1 || pending[0]["partnerRequestedAt"] == nil || pending[0]["verificationAttempts"] != float64(0) {
		t.Fatalf("requests: %s", r.body)
	}
	id := pending[0]["id"].(string)
	if r := e.do("POST", "/partners/"+id+"/approve", "", admin); r.status != 201 || !strings.Contains(r.body, `"partnerStatus":"approved"`) {
		t.Errorf("approve: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/partners/not-a-uuid/reject", "", admin); r.status != 404 {
		t.Errorf("reject unknown: %d", r.status)
	}
	if r := e.do("GET", "/users/"+id, "", admin); !strings.Contains(r.body, `"partnerStatus":"approved"`) || strings.Contains(r.body, "Hash") {
		t.Errorf("get user: %s", r.body)
	}
	if r := e.do("POST", "/auth/logout", ""); r.setCookie[0] != "admin_token=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT" {
		t.Errorf("admin logout: %v", r.setCookie)
	}
}

func TestSupport(t *testing.T) {
	e := setup(t)
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"role": "admin"}).SignedString([]byte(secret))
	admin := &http.Cookie{Name: "admin_token", Value: tok}
	r := e.do("GET", "/support", "", admin)
	if r.status != 200 || !strings.Contains(r.body, "Служба поддержки пока не настроена") {
		t.Fatalf("default: %d %s", r.status, r.body)
	}
	if r := e.do("PUT", "/support", `{"message":"Пишите нам","phone":"+996 312 000","email":"help@ermekten.com"}`, admin); r.status != 200 || !strings.Contains(r.body, `"phone":"+996 312 000"`) {
		t.Errorf("update: %d %s", r.status, r.body)
	}
	if r := e.do("PUT", "/support", `{"message":"Пишите нам","phone":null}`, admin); !strings.Contains(r.body, `"phone":null`) || !strings.Contains(r.body, `"email":"help@ermekten.com"`) {
		t.Errorf("partial update: %s", r.body)
	}
	if r := e.do("PUT", "/support", `{"message":"","email":"bad"}`, admin); r.status != 400 || r.body != `{"formErrors":[],"fieldErrors":{"email":["Invalid email"],"message":["String must contain at least 1 character(s)"]}}` {
		t.Errorf("validation: %d %s", r.status, r.body)
	}
}
