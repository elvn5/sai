package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/elvn5/sai/services/bot/internal/handlers"
	"github.com/elvn5/sai/services/bot/internal/session"
	"github.com/elvn5/sai/services/bot/internal/users"
)

const (
	token   = "123456789:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	chatID  = 42
	miniApp = "https://ermekten.com"
)

// call is one request the bot made to the Bot API.
type call struct {
	method string
	params map[string]any
}

func (c call) text() string { s, _ := c.params["text"].(string); return s }

// markup returns the reply_markup as JSON (keys sorted), for comparisons.
func (c call) markup() string {
	b, _ := json.Marshal(c.params["reply_markup"])
	return string(b)
}

// fakeUsers stands in for the users service.
type fakeUsers struct {
	mu       sync.Mutex
	log      []string
	account  users.Account
	verified *users.Account
	link     users.LinkResult
	support  users.Support
	err      error
	phoneErr error
}

func (f *fakeUsers) record(s string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log = append(f.log, s)
}

func (f *fakeUsers) calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.log...)
}

func (f *fakeUsers) FindOrCreate(_ context.Context, p users.Profile) (users.Account, error) {
	f.record("find " + p.FirstName + " @" + p.Username)
	return f.account, f.err
}

func (f *fakeUsers) ConfirmVerification(_ context.Context, token string) (*users.Account, error) {
	f.record("verify " + token)
	return f.verified, f.err
}

func (f *fakeUsers) Link(_ context.Context, token string, _ int64, _ string, move bool) (users.LinkResult, error) {
	if move {
		f.record("link " + token + " move")
	} else {
		f.record("link " + token)
	}
	return f.link, f.err
}

func (f *fakeUsers) UpdateFirstName(_ context.Context, userID, firstName string) error {
	f.record("name " + userID + " " + firstName)
	return f.err
}

func (f *fakeUsers) UpdatePhone(_ context.Context, userID, phone string) error {
	f.record("phone " + userID + " " + phone)
	return f.phoneErr
}

func (f *fakeUsers) Support(context.Context) (users.Support, error) {
	f.record("support")
	return f.support, f.err
}

type harness struct {
	// editError, when set, is the Bot API's answer to editMessageText.
	editError atomic.Value
	t         *testing.T
	calls     chan call
	updates   chan telego.Update
	users     *fakeUsers
	nextID    int
}

// newHarness runs the handlers against a fake Bot API that records every
// request.
func newHarness(t *testing.T, miniAppURL string, configure ...func(*handlers.Handlers)) *harness {
	t.Helper()
	h := &harness{t: t, calls: make(chan call, 32), updates: make(chan telego.Update), users: &fakeUsers{}}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		params := map[string]any{}
		_ = json.Unmarshal(body, &params)
		method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		h.calls <- call{method: method, params: params}
		w.Header().Set("Content-Type", "application/json")
		if desc, _ := h.editError.Load().(string); desc != "" && method == "editMessageText" {
			_, _ = io.WriteString(w, `{"ok":false,"error_code":400,"description":"`+desc+`"}`)
			return
		}
		switch method {
		case "sendMessage", "editMessageReplyMarkup", "editMessageText":
			_, _ = io.WriteString(w, `{"ok":true,"result":{"message_id":7,"date":0,"chat":{"id":42,"type":"private"}}}`)
		default:
			_, _ = io.WriteString(w, `{"ok":true,"result":true}`)
		}
	}))
	t.Cleanup(api.Close)

	bot, err := telego.NewBot(token, telego.WithAPIServer(api.URL), telego.WithDiscardLogger())
	if err != nil {
		t.Fatal(err)
	}
	bh, err := th.NewBotHandler(bot, h.updates)
	if err != nil {
		t.Fatal(err)
	}
	handler := &handlers.Handlers{Users: h.users, Sessions: session.New(), MiniAppURL: miniAppURL, Log: slog.New(slog.DiscardHandler)}
	for _, c := range configure {
		c(handler)
	}
	handler.Register(bh)
	go func() { _ = bh.Start() }()
	t.Cleanup(func() { _ = bh.Stop() })
	return h
}

var sender = telego.User{ID: chatID, FirstName: "Айгерим", Username: "aigerim"}

func (h *harness) message(text string) {
	h.nextID++
	h.updates <- telego.Update{UpdateID: h.nextID, Message: &telego.Message{
		MessageID: h.nextID, Chat: telego.Chat{ID: chatID, Type: "private"}, From: &sender, Text: text,
	}}
}

func (h *harness) press(data string) {
	h.pressOn(data, &telego.Message{MessageID: 5, Chat: telego.Chat{ID: chatID, Type: "private"}})
}

// pressOn presses a button under msg (nil: a message Telegram no longer has).
func (h *harness) pressOn(data string, msg *telego.Message) {
	h.nextID++
	q := &telego.CallbackQuery{ID: "q1", From: sender, Data: data}
	if msg != nil {
		q.Message = msg
	}
	h.updates <- telego.Update{UpdateID: h.nextID, CallbackQuery: q}
}

// expect waits for the next Bot API calls, in order, by method.
func (h *harness) expect(methods ...string) []call {
	h.t.Helper()
	got := make([]call, 0, len(methods))
	for _, method := range methods {
		select {
		case c := <-h.calls:
			if c.method != method {
				h.t.Fatalf("call %d: got %s %v, want %s", len(got), c.method, c.params, method)
			}
			got = append(got, c)
		case <-time.After(2 * time.Second):
			h.t.Fatalf("timed out waiting for %s (got %d of %v)", method, len(got), methods)
		}
	}
	return got
}

// quiet checks that the bot didn't call the Bot API.
func (h *harness) quiet() {
	h.t.Helper()
	select {
	case c := <-h.calls:
		h.t.Fatalf("unexpected call %s %v", c.method, c.params)
	case <-time.After(150 * time.Millisecond):
	}
}

func str(s string) *string { return &s }

func TestStart(t *testing.T) {
	h := newHarness(t, miniApp)
	h.message("/start")
	msg := h.expect("sendMessage")[0]
	if msg.text() != "Здравствуйте, Айгерим! Я помогу с гайдами и маршрутами по Кыргызстану и отвечу на вопросы. Выберите раздел в меню ниже." {
		t.Errorf("greeting: %q", msg.text())
	}
	if m := msg.markup(); !strings.Contains(m, `[[{"text":"👤 Профиль"},{"text":"🆘 Поддержка"}],[{"text":"🏔 Открыть Ermekten","web_app":{"url":"https://ermekten.com"}}]]`) ||
		!strings.Contains(m, `"resize_keyboard":true`) {
		t.Errorf("main menu: %s", m)
	}
	if got := h.users.calls(); len(got) != 1 || got[0] != "find Айгерим @aigerim" {
		t.Errorf("users calls: %v", got)
	}
}

func TestStartWithoutMiniApp(t *testing.T) {
	h := newHarness(t, "")
	h.message("/start")
	if m := h.expect("sendMessage")[0].markup(); strings.Contains(m, "web_app") {
		t.Errorf("menu without a Mini App: %s", m)
	}
}

func TestVerificationDeepLink(t *testing.T) {
	h := newHarness(t, miniApp)
	h.users.verified = &users.Account{ID: "u1", FirstName: str("Айгерим")}
	h.message("/start verify_abc123")
	if text := h.expect("sendMessage")[0].text(); text != "✅ Аккаунт Ermekten подтверждён, Айгерим! Возвращайтесь на сайт — вход выполнится автоматически." {
		t.Errorf("verified: %q", text)
	}

	h.users.verified = nil
	h.message("/start verify_old")
	if text := h.expect("sendMessage")[0].text(); text != "❌ Ссылка недействительна или истекла. Запросите новую ссылку на сайте Ermekten." {
		t.Errorf("invalid: %q", text)
	}
	// Verification never creates a Telegram account.
	if got := h.users.calls(); strings.Join(got, ",") != "verify abc123,verify old" {
		t.Errorf("users calls: %v", got)
	}
}

func TestLinkDeepLink(t *testing.T) {
	h := newHarness(t, miniApp)
	for _, tc := range []struct {
		result users.LinkResult
		text   string
		markup string
	}{
		{users.LinkResult{Status: users.Linked, Account: users.Account{FirstName: str("Айгерим")}},
			"✅ Telegram привязан к вашему аккаунту Ermekten, Айгерим! Теперь сюда будут приходить уведомления о выходах, а войти на сайт можно через Telegram.",
			`"keyboard"`},
		{users.LinkResult{Status: users.LinkInvalid},
			"❌ Ссылка недействительна или истекла. Нажмите «Привязать Telegram» в профиле на сайте ещё раз.", "null"},
		{users.LinkResult{Status: users.LinkTaken},
			"Этот Telegram уже привязан к другому аккаунту Ermekten (с почтой или телефоном). Войдите на сайт через Telegram, чтобы пользоваться тем аккаунтом, или отвяжите Telegram в его профиле.", "null"},
		{users.LinkResult{Status: users.LinkNeedsMove},
			"У этого Telegram уже есть свой аккаунт Ermekten, созданный через бота. Привязать Telegram к аккаунту с сайта? Войти в старый Telegram-аккаунт после этого будет нельзя.",
			`{"inline_keyboard":[[{"callback_data":"link_move:tok","text":"Привязать"},{"callback_data":"link_cancel","text":"Отмена"}]]}`},
	} {
		h.users.link = tc.result
		h.message("/start link_tok")
		msg := h.expect("sendMessage")[0]
		if msg.text() != tc.text || !strings.Contains(msg.markup(), tc.markup) {
			t.Errorf("status %d: %q %s", tc.result.Status, msg.text(), msg.markup())
		}
	}
	for _, c := range h.users.calls() {
		if c != "link tok" {
			t.Errorf("linking must not create an account: %v", h.users.calls())
		}
	}
}

func TestLinkButtons(t *testing.T) {
	h := newHarness(t, miniApp)
	h.users.link = users.LinkResult{Status: users.Linked, Account: users.Account{}}
	h.press("link_move:tok")
	calls := h.expect("answerCallbackQuery", "editMessageReplyMarkup", "sendMessage")
	if _, hasText := calls[0].params["text"]; hasText || calls[0].params["callback_query_id"] != "q1" {
		t.Errorf("answer: %v", calls[0].params)
	}
	if calls[1].params["message_id"] != float64(5) || calls[1].params["reply_markup"] != nil {
		t.Errorf("drop buttons: %v", calls[1].params)
	}
	if !strings.HasPrefix(calls[2].text(), "✅ Telegram привязан к вашему аккаунту Ermekten! ") {
		t.Errorf("linked without a name: %q", calls[2].text())
	}
	if got := h.users.calls(); len(got) != 1 || got[0] != "link tok move" {
		t.Errorf("users calls: %v", got)
	}

	h.press("link_cancel")
	calls = h.expect("answerCallbackQuery", "editMessageReplyMarkup", "sendMessage")
	if calls[0].params["text"] != "Отменено" || calls[2].text() != "Хорошо, Telegram не привязан. Ссылка на сайте действует 15 минут." {
		t.Errorf("cancel: %v / %q", calls[0].params, calls[2].text())
	}
}

func TestProfileAndSupport(t *testing.T) {
	h := newHarness(t, miniApp)
	h.users.account = users.Account{ID: "u1", FirstName: str("Айгерим"), Username: str("aigerim")}
	h.message("👤 Профиль")
	msg := h.expect("sendMessage")[0]
	if msg.text() != "Ваш профиль:\n\nИмя: Айгерим\nUsername: @aigerim\nТелефон: не указан" ||
		msg.markup() != `{"inline_keyboard":[[{"callback_data":"profile_edit_name","text":"✏️ Изменить имя"}],[{"callback_data":"profile_edit_phone","text":"✏️ Изменить телефон"}]]}` {
		t.Errorf("profile: %q %s", msg.text(), msg.markup())
	}

	h.users.account = users.Account{ID: "u1"}
	h.message("👤 Профиль")
	if text := h.expect("sendMessage")[0].text(); text != "Ваш профиль:\n\nИмя: не указано\nUsername: не указан\nТелефон: не указан" {
		t.Errorf("empty profile: %q", text)
	}

	h.users.support = users.Support{Message: "Напишите нам", Phone: str("+996 555 000 111"), Email: str(""), WorkingHours: str("9–18")}
	h.message("🆘 Поддержка")
	if text := h.expect("sendMessage")[0].text(); text != "Напишите нам\nТелефон: +996 555 000 111\nЧасы работы: 9–18" {
		t.Errorf("support: %q", text)
	}
}

func TestEditProfile(t *testing.T) {
	h := newHarness(t, miniApp)
	h.users.account = users.Account{ID: "u1"}

	// Text with nothing being edited is ignored.
	h.message("привет")
	h.quiet()

	h.press("profile_edit_name")
	calls := h.expect("answerCallbackQuery", "sendMessage")
	if calls[1].text() != "Введите новое имя:" ||
		calls[1].markup() != `{"inline_keyboard":[[{"callback_data":"profile_edit_cancel","text":"Отмена"}]]}` {
		t.Errorf("name prompt: %q %s", calls[1].text(), calls[1].markup())
	}
	h.message("  Бакыт  ")
	if msg := h.expect("sendMessage")[0]; msg.text() != "Имя обновлено." || !strings.Contains(msg.markup(), "👤 Профиль") {
		t.Errorf("name saved: %q %s", msg.text(), msg.markup())
	}

	h.press("profile_edit_phone")
	if text := h.expect("answerCallbackQuery", "sendMessage")[1].text(); text != "Введите новый номер телефона:" {
		t.Errorf("phone prompt: %q", text)
	}
	h.users.phoneErr = users.ErrPhoneTaken
	h.message("+996 700 123 456")
	if text := h.expect("sendMessage")[0].text(); text != "Этот номер уже привязан к другому аккаунту." {
		t.Errorf("phone taken: %q", text)
	}

	h.users.phoneErr = nil
	h.press("profile_edit_phone")
	h.expect("answerCallbackQuery", "sendMessage")
	h.message("+996 700 123 457")
	if text := h.expect("sendMessage")[0].text(); text != "Телефон обновлен." {
		t.Errorf("phone saved: %q", text)
	}

	h.press("profile_edit_name")
	h.expect("answerCallbackQuery", "sendMessage")
	h.message("   ")
	if text := h.expect("sendMessage")[0].text(); text != "Пустое значение не сохранено." {
		t.Errorf("empty value: %q", text)
	}

	h.press("profile_edit_name")
	h.expect("answerCallbackQuery", "sendMessage")
	h.press("profile_edit_cancel")
	calls = h.expect("answerCallbackQuery", "sendMessage")
	if calls[0].params["text"] != "Отменено" || calls[1].text() != "Редактирование отменено." {
		t.Errorf("cancel: %v %q", calls[0].params, calls[1].text())
	}
	h.message("Не имя")
	h.quiet()

	want := "name u1 Бакыт,phone u1 +996 700 123 456,phone u1 +996 700 123 457"
	var saved []string
	for _, c := range h.users.calls() {
		if strings.HasPrefix(c, "name ") || strings.HasPrefix(c, "phone ") {
			saved = append(saved, c)
		}
	}
	if strings.Join(saved, ",") != want {
		t.Errorf("saved: %v", saved)
	}
}

func TestUsersServiceDown(t *testing.T) {
	h := newHarness(t, miniApp)
	h.users.err = errors.New("unavailable")
	// The error is logged; the user gets no half-done reply.
	h.message("/start")
	h.message("🆘 Поддержка")
	h.quiet()
}
