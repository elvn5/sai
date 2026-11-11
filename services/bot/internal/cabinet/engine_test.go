package cabinet

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"
	"unicode/utf16"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/screens"
	"github.com/elvn5/sai/services/pkg/auth"
)

const tourID = "11111111-1111-4111-8111-111111111111"
const participantID = "22222222-2222-4222-8222-222222222222"
const clientID = "33333333-3333-4333-8333-333333333333"
const reviewID = "44444444-4444-4444-8444-444444444444"

type call struct {
	method, path, user string
	body               record
}
type harness struct {
	t      *testing.T
	e      *Engine
	screen screens.Screen
	calls  []call
	mu     sync.Mutex
	kind   string
	fail   bool
	now    time.Time
	lang   i18n.Lang
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{t: t, kind: "organizer", now: time.Date(2026, 11, 10, 10, 0, 0, 0, time.UTC), lang: i18n.RU}
	verifier := auth.NewVerifier("cabinet-testing-secret")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		defer h.mu.Unlock()
		cookie, err := r.Cookie(auth.UserCookie)
		if err != nil {
			t.Error("missing user cookie")
			w.WriteHeader(401)
			return
		}
		user, err := verifier.UserID(cookie.Value)
		if err != nil {
			t.Error(err)
			w.WriteHeader(401)
			return
		}
		var body record
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		h.calls = append(h.calls, call{r.Method, r.URL.RequestURI(), user, body})
		w.Header().Set("Content-Type", "application/json")
		if r.Method != "GET" {
			if h.fail {
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"fieldErrors":{"phone":["Invalid phone"]}}`))
				return
			}
			if r.Method == "DELETE" || r.URL.Path == "/listing-stats/events" {
				w.WriteHeader(204)
				return
			}
			_, _ = w.Write([]byte(`{"id":"` + tourID + `","sent":2,"skipped":1}`))
			return
		}
		var result any
		switch r.URL.Path {
		case "/auth/user/me":
			result = record{"user": record{"partnerStatus": "approved", "partnerKind": h.kind}}
		case "/tours/mine", "/tours":
			result = []record{{"id": tourID, "title": "Tour <one>", "content": "<p>Description</p>", "locale": "ru", "startDate": "2026-12-01", "endDate": "2026-12-02", "photos": []string{"/uploads/old.webp"}, "program": []record{{"title": "Day 1", "text": "Walk"}}, "phone": "+996555000111", "isPublished": true}}
		case "/tours/by-id/" + tourID:
			result = record{"id": tourID, "title": "Public tour", "isJoined": true, "organizer": record{"id": clientID, "firstName": "Guide"}, "myForm": record{"emergencyName": "Mother", "emergencyPhone": "+996555000111", "experience": "hiking"}}
		case "/crm/clients":
			items := []record{}
			offset := 0
			if r.URL.Query().Get("offset") == "6" {
				offset = 6
			}
			for i := offset; i < min(offset+6, 8); i++ {
				items = append(items, record{"id": fmt.Sprintf("33333333-3333-4333-8333-%012d", i), "name": fmt.Sprintf("Client %d", i)})
			}
			result = record{"items": items, "total": 8}
		case "/crm/tours/" + tourID + "/participants":
			result = record{"participants": []record{{"id": participantID, "clientId": clientID, "name": "Traveler", "status": "confirmed", "amountDue": 1200, "form": record{"emergencyName": "Mother", "emergencyPhone": "+996555000111", "experience": "none"}, "payments": []record{{"id": reviewID, "amount": 400, "method": "cash", "paidOn": "2026-11-10"}}}}, "totals": record{"paid": 400, "due": 800}, "expenses": []record{}, "staff": []record{}}
		case "/reviews/mine", "/reviews":
			result = record{"items": []record{{"id": reviewID, "text": "A good tour!", "rating": 5, "author": record{"id": "me", "firstName": "Author"}, "reply": record{"text": "Thank you"}}}, "total": 1}
		case "/listing-posts":
			result = record{"items": []record{{"id": reviewID, "text": "News", "author": record{"id": "me"}}}, "total": 1}
		case "/driver/trips/" + tourID:
			result = record{"id": tourID, "routeFrom": "Bishkek", "routeTo": "Karakol", "startsAt": "2026-12-01T06:00:00+06:00", "status": "requested", "payments": []any{}}
		case "/driver/profile":
			result = record{"name": "Driver", "published": true}
		case "/chats/" + tourID + "/messages":
			result = record{"id": tourID, "peer": record{"firstName": "Friend"}, "messages": []record{{"senderId": "me", "body": "Hello", "createdAt": "2026-11-10T08:00:00Z"}, {"senderId": clientID, "body": "Hi", "createdAt": "2026-11-10T09:00:00Z"}}}
		default:
			if strings.HasPrefix(r.URL.Path, "/crm/clients/") {
				result = record{"id": clientID, "name": "Client card", "note": "Private note"}
			} else if strings.HasSuffix(r.URL.Path, "/summary") || strings.HasSuffix(r.URL.Path, "/reports") || strings.HasPrefix(r.URL.Path, "/listing-stats") || r.URL.Path == "/payouts/mine" {
				result = record{"paid": 400, "due": 800}
			} else {
				result = []record{}
			}
		}
		_ = json.NewEncoder(w).Encode(result)
	}))
	t.Cleanup(srv.Close)
	client, err := api.New(srv.URL, "cabinet-testing-secret")
	if err != nil {
		t.Fatal(err)
	}
	h.e = New(client, screens.Links{PublicAPI: "https://api.example", Bot: "test_bot"})
	h.e.Now = func() time.Time { return h.now }
	return h
}
func (h *harness) open(data string) {
	h.t.Helper()
	s, err := h.e.Handle(context.Background(), "me", 10, h.lang, data)
	if err != nil {
		h.t.Fatal(err)
	}
	h.screen = s
	h.check()
}
func (h *harness) token(label string) string {
	h.t.Helper()
	for _, row := range h.screen.Markup.InlineKeyboard {
		for _, b := range row {
			if b.Text == label || strings.TrimSuffix(b.Text, " *") == label {
				return strings.TrimPrefix(b.CallbackData, "k:")
			}
		}
	}
	h.t.Fatalf("button %q missing: %+v\n%s", label, h.screen.Markup, h.screen.Text)
	return ""
}
func (h *harness) press(label string) { h.t.Helper(); h.open(h.token(label)) }
func (h *harness) text(text string) {
	h.t.Helper()
	s, ok, err := h.e.Text(context.Background(), "me", 10, h.lang, text)
	if err != nil || !ok {
		h.t.Fatalf("text handled=%v err=%v", ok, err)
	}
	h.screen = s
	h.check()
}
func (h *harness) fill(label, text string) { h.t.Helper(); h.press(label); h.text(text) }
func (h *harness) check() {
	h.t.Helper()
	if len(utf16.Encode([]rune(html.UnescapeString(regexp.MustCompile(`<[^>]*>`).ReplaceAllString(h.screen.Text, ""))))) > 4096 {
		h.t.Fatalf("message exceeds Telegram limit: %d", len([]rune(h.screen.Text)))
	}
	if h.screen.Markup == nil {
		h.t.Fatal("no navigation")
	}
	for _, row := range h.screen.Markup.InlineKeyboard {
		for _, b := range row {
			if len(b.CallbackData) > 64 {
				h.t.Fatal("oversize callback")
			}
		}
	}
}
func (h *harness) writes() []call {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := []call{}
	for _, c := range h.calls {
		if c.method != "GET" && c.path != "/listing-stats/events" {
			out = append(out, c)
		}
	}
	return out
}

func TestCreateTourRequiresPreviewAndConfirmation(t *testing.T) {
	h := newHarness(t)
	h.open("home")
	h.press("Мои объявления")
	h.press("➕ Добавить")
	h.fill("Название", "My <new> tour")
	h.fill("Описание", "Walk & climb")
	h.fill("Дата начала", "2026-12-01")
	h.fill("Дата окончания", "2026-12-02")
	h.press("Проверить и сохранить")
	if len(h.writes()) != 0 {
		t.Fatal("wrote before confirmation")
	}
	if !strings.Contains(h.screen.Text, "My &lt;new&gt; tour") {
		t.Fatal("preview not escaped")
	}
	token := h.token("Подтвердить")
	h.open(token)
	h.open(token)
	calls := h.writes()
	if len(calls) != 1 || calls[0].path != "/tours" || calls[0].method != "POST" || calls[0].user != "me" {
		t.Fatalf("duplicate or wrong request: %+v", calls)
	}
	body := calls[0].body
	if body["content"] != "<p>Walk &amp; climb</p>" || body["locale"] != "ru" || !strings.HasPrefix(str(body["slug"]), "tour-") {
		t.Fatalf("tour payload: %+v", body)
	}
}
func TestEditPreservesPhotosAndProgramAndClearsOptionalFields(t *testing.T) {
	h := newHarness(t)
	h.open("home")
	h.press("Мои объявления")
	h.press("Tour <one>")
	h.press("Изменить")
	h.fill("Название", "Updated")
	h.press("Дальше ▶️")
	h.press("Дальше ▶️")
	h.press("Телефон")
	h.press("Очистить поле")
	h.press("Проверить и сохранить")
	h.press("Подтвердить")
	calls := h.writes()
	if len(calls) != 1 || calls[0].method != "PUT" || calls[0].path != "/tours/"+tourID {
		t.Fatalf("update: %+v", calls)
	}
	b := calls[0].body
	if b["phone"] != nil || len(b["photos"].([]any)) != 1 || len(b["program"].([]any)) != 1 || b["title"] != "Updated" {
		t.Fatalf("lost data: %+v", b)
	}
}
func TestNestedParticipantFormAndPayments(t *testing.T) {
	h := newHarness(t)
	h.open("group:" + tourID)
	h.press("Traveler")
	h.press("Анкета участника")
	h.fill("Контакт на случай ЧП", "Father")
	h.press("Проверить и сохранить")
	h.press("Подтвердить")
	call := h.writes()[0]
	if call.path != "/crm/tours/"+tourID+"/participants/"+participantID+"/form" || object(call.body["form"])["emergencyName"] != "Father" || call.body["acceptWaiver"] != nil {
		t.Fatalf("form contract: %+v", call)
	}
	h.open("group:" + tourID)
	h.press("Traveler")
	h.press("Оплаты участника")
	h.press("➕ Добавить")
	h.press("Тип операции")
	h.press("Оплата")
	h.fill("Сумма, сом", "500")
	h.press("Способ оплаты")
	h.press("Наличные")
	h.fill("Дата оплаты", "2026-11-10")
	h.press("Проверить и сохранить")
	token := h.token("Подтвердить")
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() { _, _ = h.e.Handle(context.Background(), "me", 10, i18n.RU, token) })
	}
	wg.Wait()
	writes := h.writes()
	if len(writes) != 2 || writes[1].path != "/crm/tours/"+tourID+"/participants/"+participantID+"/payments" || writes[1].body["amount"] != float64(500) {
		t.Fatalf("payment/replay: %+v", writes)
	}
}
func TestActionsBoundToAccountChatAndExpiry(t *testing.T) {
	h := newHarness(t)
	h.open("home")
	token := h.token("Мои объявления")
	for _, tc := range []struct {
		user string
		chat int64
	}{{"other", 10}, {"me", 11}} {
		s, err := h.e.Handle(context.Background(), tc.user, tc.chat, i18n.RU, token)
		if err != nil || !strings.Contains(s.Text, "устарела") {
			t.Fatalf("cross-user token accepted: %s %v", s.Text, err)
		}
	}
	h.open("home")
	token = h.token("Мои объявления")
	h.now = h.now.Add(lifetime + time.Second)
	h.open(token)
	if !strings.Contains(h.screen.Text, "устарела") {
		t.Fatal("expired action accepted")
	}
	h.open("home")
	h.press("Клиенты")
	h.press("➕ Добавить")
	h.press("Имя")
	h.now = h.now.Add(lifetime + time.Second)
	h.text("Should not save")
	if !strings.Contains(h.screen.Text, "истёк") || len(h.writes()) != 0 {
		t.Fatal("expired form accepted")
	}
}
func TestBackendFailureKeepsDraftWithoutAutomaticRetry(t *testing.T) {
	h := newHarness(t)
	h.open("home")
	h.press("Клиенты")
	h.press("➕ Добавить")
	h.fill("Имя", "Aida")
	h.press("Проверить и сохранить")
	h.fail = true
	h.press("Подтвердить")
	if !strings.Contains(h.screen.Text, "Телефон") || len(h.writes()) != 1 {
		t.Fatalf("failure: %s", h.screen.Text)
	}
	h.press("Вернуться к черновику")
	if !strings.Contains(h.screen.Text, "Aida") {
		t.Fatal("draft lost")
	}
	h.fail = false
	h.press("Проверить и сохранить")
	h.press("Подтвердить")
	if len(h.writes()) != 2 {
		t.Fatal("retry missing")
	}
}
func TestPaginationAndReferencePicker(t *testing.T) {
	h := newHarness(t)
	h.open("home")
	h.press("Клиенты")
	h.press("Дальше ▶️")
	h.token("Client 6")
	h.open("group:" + tourID)
	h.press("➕ Добавить")
	h.press("Клиент")
	h.press("Дальше ▶️")
	h.press("Client 7")
	h.press("Проверить и сохранить")
	if !strings.Contains(h.screen.Text, "Client 7") || strings.Contains(h.screen.Text, "33333333-") {
		t.Fatalf("selection preview: %s", h.screen.Text)
	}
	h.press("Подтвердить")
	if h.writes()[0].body["clientId"] != "33333333-3333-4333-8333-000000000007" {
		t.Fatal("wrong selection")
	}
}
func TestReviewReplyDeleteAndOwnership(t *testing.T) {
	h := newHarness(t)
	h.open("home")
	// Exercise the retained review workflow independently of the home menu.
	v := &view{e: h.e}
	h.e.state(10).actions["review-list"] = v.listAction("reviews", nil, nil, 0)
	h.open("review-list")
	h.press("A good tour!")
	h.press("Ответить")
	if !strings.Contains(h.screen.Text, "Thank you") {
		t.Fatal("reply object not decoded")
	}
	h.fill("Ответ", "You are welcome")
	h.press("Проверить и сохранить")
	h.press("Подтвердить")
	if h.writes()[0].path != "/reviews/"+reviewID+"/reply" {
		t.Fatal("reply route")
	}
	h.open("tour:" + tourID)
	h.press("Отзывы")
	h.press("A good tour!")
	h.press("Удалить отзыв")
	if len(h.writes()) != 1 {
		t.Fatal("deleted without confirmation")
	}
	token := h.token("Да, удалить")
	h.open(token)
	h.open(token)
	if len(h.writes()) != 2 || h.writes()[1].method != "DELETE" {
		t.Fatal("delete confirmation/replay")
	}
}
func TestTravelerWaiverOnlyOnExplicitAcceptance(t *testing.T) {
	h := newHarness(t)
	h.open("tour:" + tourID)
	h.press("Анкета участника")
	h.press("Проверить и сохранить")
	h.press("Подтвердить")
	if h.writes()[0].body["acceptWaiver"] != nil || h.writes()[0].body["form"] == nil {
		t.Fatal("implicit waiver")
	}
	h.open("tour:" + tourID)
	h.press("Правила участия и риски")
	if !strings.Contains(h.screen.Text, "страховку") {
		t.Fatal("missing rules")
	}
	// Labels longer than Telegram's practical button width are shortened.
	token := ""
	for _, row := range h.screen.Markup.InlineKeyboard {
		if strings.HasPrefix(row[0].Text, "Я прочитал") {
			token = strings.TrimPrefix(row[0].CallbackData, "k:")
		}
	}
	if token == "" {
		t.Fatal("accept missing")
	}
	h.open(token)
	if h.writes()[1].body["acceptWaiver"] != true {
		t.Fatal("waiver not saved")
	}
}
func TestDriverAccessAndChatHistory(t *testing.T) {
	h := newHarness(t)
	h.kind = "driver"
	h.open("home")
	h.token("Автомобили")
	h.open("trip:" + tourID)
	h.press("Статус поездки")
	h.press("Статус")
	h.press("Принято")
	h.press("Проверить и сохранить")
	h.press("Подтвердить")
	if h.writes()[0].path != "/driver/trips/"+tourID+"/status" {
		t.Fatal("driver status route")
	}
	h.open("group:" + tourID)
	if !strings.Contains(h.screen.Text, "отклонено") {
		t.Fatal("driver saw organizer group")
	}
	h.open("chat:" + tourID)
	if !strings.Contains(h.screen.Text, "Friend") || !strings.Contains(h.screen.Text, "Вы") {
		t.Fatal("message authors not shown")
	}
	h.press("Написать сообщение")
	h.fill("Сообщение", "Reply <unsafe>")
	h.press("Проверить и сохранить")
	if len(h.writes()) != 1 {
		t.Fatal("message sent before confirmation")
	}
	h.press("Подтвердить")
	if h.writes()[1].body["body"] != "Reply <unsafe>" {
		t.Fatal("message changed")
	}
}
func TestPhotosPersistWithoutTelegramToken(t *testing.T) {
	h := newHarness(t)
	h.open("home")
	h.press("Мои объявления")
	h.press("Tour <one>")
	h.press("Изменить")
	h.press("Дальше ▶️")
	h.press("Фотографии")
	s, handled, err := h.e.Photo(context.Background(), "me", 10, i18n.RU, func() (string, error) { return "/uploads/new.webp", nil })
	if !handled || err != nil {
		t.Fatal(err)
	}
	h.screen = s
	h.press("Проверить и сохранить")
	h.press("Подтвердить")
	photos := h.writes()[0].body["photos"].([]any)
	if len(photos) != 2 || photos[1] != "/uploads/new.webp" {
		t.Fatalf("photos: %v", photos)
	}
}
func TestLanguageAndLongTextLimits(t *testing.T) {
	for _, l := range []i18n.Lang{i18n.EN, i18n.DE, i18n.KY} {
		t.Run(string(l), func(t *testing.T) {
			h := newHarness(t)
			h.lang = l
			h.open("home")
			for _, row := range h.screen.Markup.InlineKeyboard {
				for _, b := range row {
					if l != i18n.KY {
						for _, r := range b.Text {
							if unicode.Is(unicode.Cyrillic, r) {
								t.Fatalf("untranslated: %s", b.Text)
							}
						}
					}
				}
			}
		})
	}
	h := newHarness(t)
	s := h.e.state(10)
	v := &view{e: h.e, s: s, ctx: context.Background(), user: "me", l: i18n.RU}
	text := strings.Repeat("<&long> ", 1500)
	screen, err := v.card(resource{title: "Сообщение"}, nil, record{"body": text}, func(v *view) (screens.Screen, error) { return v.home() }, 0)
	if err != nil {
		t.Fatal(err)
	}
	h.screen = screen
	h.check()
	h.token("Дальше ▶️")
}
func TestCatalogFiltersAndInputValidation(t *testing.T) {
	items := []record{{"price": 10, "durationDays": 2, "cancellationPolicy": "free"}, {"price": 20, "durationDays": 1, "cancellationPolicy": "free"}, {"price": 5, "durationDays": 3, "cancellationPolicy": "strict"}, {"durationDays": 3, "cancellationPolicy": "free"}}
	got := filterTours(items, record{"free": true, "duration": "2-3", "sort": "price_asc"})
	if len(got) != 2 || got[0]["price"] != 10 || got[1]["price"] != nil {
		t.Fatalf("filters: %v", got)
	}
	for _, tc := range []struct {
		f    field
		text string
	}{{number("capacity", "", true, 1, 500), "501"}, {date("startDate", "", true), "2026-02-30"}, {special("meetingTime", "", "time", 5), "25:00"}, {textField("phone", "", false, 32), "bad"}, {special("photos", "", "photos", 20), "javascript:alert(1)"}, {ref("clientId", "", "clients", true), clientID}} {
		if _, err := parseField(tc.f, tc.text); err == nil {
			t.Fatalf("accepted invalid %s", tc.f.key)
		}
	}
	if got, err := parseField(special("startsAt", "", "datetime", 0), "2026-11-10 09:30"); err != nil || got != "2026-11-10T09:30:00+06:00" {
		t.Fatalf("time zone: %v %v", got, err)
	}
}
