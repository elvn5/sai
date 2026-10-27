package handlers_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/handlers"
	"github.com/elvn5/sai/services/bot/internal/screens"
	"github.com/elvn5/sai/services/bot/internal/users"
)

var now = time.Date(2026, 10, 20, 4, 0, 0, 0, time.UTC)

// fakeContent stands in for the gateway: trips are the tours as each user
// sees them by id (signed up or not), tours the public upcoming list.
type fakeContent struct {
	mu        sync.Mutex
	log       []string
	trips     map[string]api.Tour
	tours     []api.Tour
	favorites map[string]bool
	alerts    []string
	posts     []api.BlogPost
	joinErr   error
	// down fails every read, as an unreachable gateway would.
	down bool
}

var errDown = errors.New("gateway down")

func (f *fakeContent) record(s string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log = append(f.log, s)
}

func (f *fakeContent) calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.log...)
}

func (f *fakeContent) JoinedTours(context.Context, string) ([]api.Tour, error) {
	if f.down {
		return nil, errDown
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []api.Tour
	for _, t := range f.trips {
		if t.IsJoined {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeContent) TourByID(_ context.Context, _, id string) (api.Tour, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t, ok := f.trips[id]; ok {
		return t, nil
	}
	for _, t := range f.tours {
		if t.ID == id {
			t.Content = "<p>Описание тура " + t.Title + "</p>"
			return t, nil
		}
	}
	return api.Tour{}, &api.Error{Status: 404, Message: "Tour not found"}
}

func (f *fakeContent) JoinTour(_ context.Context, userID, id string) (api.Tour, error) {
	f.record("join " + userID + " " + id)
	if f.joinErr != nil {
		return api.Tour{}, f.joinErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	t := f.trips[id]
	t.IsJoined, t.Taken = true, t.Taken+1
	f.trips[id] = t
	return t, nil
}

func (f *fakeContent) LeaveTour(_ context.Context, userID, id string) (api.Tour, error) {
	f.record("leave " + userID + " " + id)
	f.mu.Lock()
	defer f.mu.Unlock()
	t := f.trips[id]
	t.IsJoined, t.Taken = false, t.Taken-1
	f.trips[id] = t
	return t, nil
}

func (f *fakeContent) UpcomingTours(context.Context, string) ([]api.Tour, error) { return f.tours, nil }

func (f *fakeContent) FavoriteTours(context.Context, string) ([]api.Tour, error) {
	if f.down {
		return nil, errDown
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []api.Tour
	for _, t := range f.tours {
		if f.favorites[t.ID] {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeContent) SetFavoriteTour(_ context.Context, _, tourID string, saved bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.favorites[tourID] = saved
	return nil
}

func (f *fakeContent) TourAlerts(context.Context, string) ([]string, error) {
	if f.down {
		return nil, errDown
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.alerts...), nil
}

func (f *fakeContent) SetAllToursAlert(_ context.Context, _ string, on bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.alerts = nil
	if on {
		f.alerts = []string{""}
	}
	return nil
}

func (f *fakeContent) BlogPosts(context.Context, string) ([]api.BlogPost, error) {
	if f.down {
		return nil, errDown
	}
	return f.posts, nil
}

func (f *fakeContent) BlogPost(_ context.Context, locale, slug string) (api.BlogPost, error) {
	for _, p := range f.posts {
		if p.Slug == slug {
			p.Content = "<h2>Рюкзак</h2><p>Литров на 30.</p>"
			return p, nil
		}
	}
	return api.BlogPost{}, &api.Error{Status: 404}
}

type fakeSubscribers struct {
	mu   sync.Mutex
	subs map[int64]bool
}

func (f *fakeSubscribers) IsBlogSubscriber(_ context.Context, chatID int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.subs[chatID], nil
}

func (f *fakeSubscribers) SetBlogSubscriber(_ context.Context, chatID int64, on bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.subs[chatID] = on
	return nil
}

func newSections(t *testing.T, configure ...func(*handlers.Handlers)) (*harness, *fakeContent, *fakeSubscribers) {
	t.Helper()
	content := &fakeContent{
		trips: map[string]api.Tour{
			"o1": {ID: "o1", Title: "Пик Учитель", StartDate: "2026-10-24", EndDate: "2026-10-24", MeetingTime: str("10:00"),
				Capacity: num(10), Taken: 7, IsJoined: true},
			"o2": {ID: "o2", Title: "Барскоон", StartDate: "2026-10-30", EndDate: "2026-10-31", Capacity: num(12), Taken: 4},
		},
		tours: []api.Tour{{ID: "t1", Slug: "son-kol", Title: "Сон-Кёль", StartDate: "2026-11-14", EndDate: "2026-11-15"},
			{ID: "t2", Slug: "ala-kol", Title: "Ала-Кёль"}},
		favorites: map[string]bool{},
		posts:     []api.BlogPost{{ID: "p1", Slug: "checklist", Title: "Чек-лист"}},
	}
	subs := &fakeSubscribers{subs: map[int64]bool{}}
	h := newHarness(t, miniApp, func(hs *handlers.Handlers) {
		hs.Content, hs.Subscribers = content, subs
		hs.Links = screens.Links{Bot: "ermekten_bot"}
		hs.Now = func() time.Time { return now }
		for _, c := range configure {
			c(hs)
		}
	})
	h.users.account = users.Account{ID: "u1"}
	return h, content, subs
}

func TestStartWithSections(t *testing.T) {
	h, _, _ := newSections(t)
	h.message("/start")
	msg := h.expect("sendMessage")[0]
	if !strings.HasPrefix(msg.text(), "🏔 <b>Здравствуйте, Айгерим!</b>") ||
		!strings.Contains(msg.text(), "Ближайший тур: <b>Пик Учитель</b>, сб, 24 окт., 10:00") ||
		msg.params["parse_mode"] != "HTML" {
		t.Errorf("greeting: %q", msg.text())
	}
	if m := msg.markup(); !strings.Contains(m, `[[{"text":"🎒 Мои туры"},{"text":"🧭 Туры"}],[{"text":"📰 Блог"},{"text":"🔔 Уведомления"}],[{"text":"👤 Профиль"},{"text":"🆘 Поддержка"}],`) {
		t.Errorf("main menu: %s", m)
	}

	h.message("/start tour_o2")
	card := h.expect("sendMessage")[0]
	if !strings.HasPrefix(card.text(), "🧭 <b>Барскоон</b>") || !strings.Contains(card.markup(), `"callback_data":"o:join:u:o2"`) {
		t.Errorf("deep link: %q %s", card.text(), card.markup())
	}
	h.message("/start tour_nope")
	if text := h.expect("sendMessage")[0].text(); text != "Такого тура нет — возможно, его удалили." {
		t.Errorf("unknown tour: %q", text)
	}
}

func TestMyToursFlow(t *testing.T) {
	h, content, _ := newSections(t)
	h.message("🎒 Мои туры")
	list := h.expect("sendMessage")[0]
	if !strings.Contains(list.text(), "<b>1. Пик Учитель</b>") || strings.Contains(list.text(), "Барскоон") ||
		!strings.Contains(list.markup(), `"callback_data":"o:view:u:o1"`) {
		t.Errorf("my tours: %q %s", list.text(), list.markup())
	}
	h.press("o:join:t:o2")
	calls := h.expect("answerCallbackQuery", "editMessageText")
	if calls[0].params["text"] != "✅ Вы записаны! Организатор получит уведомление." ||
		!strings.Contains(calls[1].text(), "✅ Вы записаны") || !strings.Contains(calls[1].markup(), `"o:ask:t:o2"`) {
		t.Errorf("join: %v / %q", calls[0].params, calls[1].text())
	}

	h.press("o:ask:t:o2")
	calls = h.expect("answerCallbackQuery", "editMessageText")
	if !strings.HasPrefix(calls[1].text(), "Точно отменить запись на «Барскоон»?") {
		t.Errorf("ask: %q", calls[1].text())
	}
	h.press("o:leave:t:o2")
	calls = h.expect("answerCallbackQuery", "editMessageText")
	if calls[0].params["text"] != "Запись отменена" || !strings.Contains(calls[1].markup(), `"o:join:t:o2"`) {
		t.Errorf("leave: %v / %s", calls[0].params, calls[1].markup())
	}

	content.joinErr = &api.Error{Status: 409, Message: "No places left"}
	h.press("o:join:u:o2")
	if calls = h.expect("answerCallbackQuery", "editMessageText"); calls[0].params["text"] != "Мест больше нет 😔" {
		t.Errorf("full: %v", calls[0].params)
	}
	if got := strings.Join(content.calls(), ","); got != "join u1 o2,leave u1 o2,join u1 o2" {
		t.Errorf("content calls: %s", got)
	}

	// A gone tour falls back to the list.
	h.press("o:view:u:gone")
	if calls = h.expect("answerCallbackQuery", "editMessageText"); calls[0].params["text"] != "Этого тура больше нет" ||
		!strings.Contains(calls[1].text(), "Мои туры") {
		t.Errorf("gone: %v", calls[0].params)
	}
	// From a notification: the card as a message of its own.
	h.press("o:view:n:o1")
	if card := h.expect("answerCallbackQuery", "sendMessage")[1]; !strings.HasPrefix(card.text(), "🧭 <b>Пик Учитель</b>") {
		t.Errorf("from a notice: %q", card.text())
	}
}

// Buttons of what is gone (an old draft, a trail report)
// say so.
func TestStaleButtons(t *testing.T) {
	h, content, _ := newSections(t)
	for _, data := range []string{"d:new", "c:s:o1", "c:d:o1:2:1"} {
		h.press(data)
		if text, _ := h.expect("answerCallbackQuery")[0].params["text"].(string); !strings.HasPrefix(text, "Эта кнопка устарела") {
			t.Errorf("%s: %q", data, text)
		}
		h.quiet()
	}
	if len(content.calls()) != 0 {
		t.Errorf("stale buttons changed something: %v", content.calls())
	}
}

// Reading a tour's description offers signing up for it.
func TestTourDescriptionSignUp(t *testing.T) {
	h, _, _ := newSections(t)
	h.press("t:more:n:0:0")
	more := h.expect("answerCallbackQuery", "editMessageText")[1]
	if !strings.Contains(more.text(), "Описание тура Сон-Кёль") || !strings.Contains(more.markup(), `"o:join:t:t1"`) {
		t.Errorf("description: %q %s", more.text(), more.markup())
	}
}

func TestToursFlow(t *testing.T) {
	h, content, _ := newSections(t)
	// No favorites yet: the upcoming tab.
	h.message("/tours")
	msg := h.expect("sendMessage")[0]
	if !strings.HasPrefix(msg.text(), "🧭 <b>Туры</b> · Ближайшие 1 из 2") || !strings.Contains(msg.markup(), `"t:save:n:0:t1"`) {
		t.Errorf("tours: %q %s", msg.text(), msg.markup())
	}
	h.press("t:save:n:0:t1")
	calls := h.expect("answerCallbackQuery", "editMessageText")
	if calls[0].params["text"] != "❤️ Добавлено в избранное" || !strings.Contains(calls[1].markup(), `"t:drop:n:0:t1"`) {
		t.Errorf("save: %v %s", calls[0].params, calls[1].markup())
	}
	h.press("t:alert:n:0")
	calls = h.expect("answerCallbackQuery", "editMessageText")
	if calls[0].params["text"] != "🔔 Сообщим о новых турах" || !strings.Contains(calls[1].markup(), "🔕 Не сообщать о новых турах") {
		t.Errorf("alert: %v", calls[0].params)
	}
	if !content.favorites["t1"] || len(content.alerts) != 1 {
		t.Errorf("state: %v %v", content.favorites, content.alerts)
	}
	h.press("t:f:0")
	if text := h.expect("answerCallbackQuery", "editMessageText")[1].text(); !strings.HasPrefix(text, "🧭 <b>Туры</b> · Избранное 1 из 1") {
		t.Errorf("favorites: %q", text)
	}
}

func TestBlogAndNotifications(t *testing.T) {
	h, _, subs := newSections(t)
	h.message("📰 Блог")
	if m := h.expect("sendMessage")[0].markup(); !strings.Contains(m, `"b:sub:0"`) {
		t.Errorf("blog: %s", m)
	}
	h.press("b:sub:0")
	calls := h.expect("answerCallbackQuery", "editMessageText")
	if calls[0].params["text"] != "🔔 Пришлём новые статьи" || !subs.subs[chatID] || !strings.Contains(calls[1].markup(), `"b:unsub:0"`) {
		t.Errorf("subscribe: %v", calls[0].params)
	}

	h.message("🔔 Уведомления")
	if m := h.expect("sendMessage")[0].markup(); !strings.Contains(m, "📰 Статьи блога: ✅ вкл") || !strings.Contains(m, "🧭 Новые туры: ⬜ выкл") {
		t.Errorf("settings: %s", m)
	}
	h.press("n:tours")
	calls = h.expect("answerCallbackQuery", "editMessageText")
	if calls[0].params["text"] != "Новые туры: включено" || !strings.Contains(calls[1].markup(), "🧭 Новые туры: ✅ вкл") {
		t.Errorf("tours toggle: %v %s", calls[0].params, calls[1].markup())
	}

	// "Don't send" under a post that came as a notification.
	h.press("n:blog_off")
	calls = h.expect("answerCallbackQuery")
	if subs.subs[chatID] || !strings.HasPrefix(calls[0].params["text"].(string), "Больше не присылаем статьи.") {
		t.Errorf("blog off: %v", calls[0].params)
	}
	h.quiet()

	h.press("noop")
	h.expect("answerCallbackQuery")
}

const loadFailed = "😔 Не получилось загрузить данные. Попробуйте ещё раз чуть позже."

// With the gateway down every section says so instead of going silent.
func TestSectionsDown(t *testing.T) {
	h, content, _ := newSections(t)
	content.down = true
	for _, text := range []string{"🎒 Мои туры", "🧭 Туры", "📰 Блог", "🔔 Уведомления"} {
		h.message(text)
		if got := h.expect("sendMessage")[0].text(); got != loadFailed {
			t.Errorf("%s: %q", text, got)
		}
	}
	for _, data := range []string{"o:list:u", "t:f:0", "t:alert:n:0", "b:0", "n:tours"} {
		h.press(data)
		if got := h.expect("answerCallbackQuery")[0].params["text"]; got != loadFailed {
			t.Errorf("%s: %v", data, got)
		}
	}
	h.quiet()
}

func TestScreenEditFallbacks(t *testing.T) {
	h, _, _ := newSections(t)

	// Too old to edit: the screen comes anew.
	h.editError.Store("Bad Request: message can't be edited")
	h.press("o:list:u")
	if calls := h.expect("answerCallbackQuery", "editMessageText", "sendMessage"); !strings.Contains(calls[2].text(), "Мои туры") {
		t.Errorf("resend: %q", calls[2].text())
	}

	// Pressing the current tab changes nothing: no new message.
	h.editError.Store("Bad Request: message is not modified: specified new message content and reply markup are exactly the same")
	h.press("o:list:u")
	h.expect("answerCallbackQuery", "editMessageText")
	h.quiet()
	h.editError.Store("")

	// Under a photo (a blog post notification), or a message Telegram lost: a new message.
	h.pressOn("b:0", &telego.Message{MessageID: 9, Chat: telego.Chat{ID: chatID}, Photo: []telego.PhotoSize{{FileID: "f"}}})
	h.expect("answerCallbackQuery", "sendMessage")
	h.pressOn("b:0", nil)
	if msg := h.expect("answerCallbackQuery", "sendMessage")[1]; msg.params["chat_id"] != float64(chatID) {
		t.Errorf("chat: %v", msg.params)
	}
}

func TestCommands(t *testing.T) {
	h, _, _ := newSections(t)
	h.users.support = users.Support{Message: "Напишите нам"}
	for cmd, want := range map[string]string{
		"/mytours": "🎒 <b>Мои туры</b>", "/blog": "📰 <b>Блог Ermekten</b>",
		"/notifications": "🔔 <b>Уведомления</b>", "/profile": "Ваш профиль:", "/support": "Напишите нам",
	} {
		h.message(cmd)
		if got := h.expect("sendMessage")[0].text(); !strings.HasPrefix(got, want) {
			t.Errorf("%s: %q", cmd, got)
		}
	}
}

// Stale or malformed button data only gets the press acknowledged.
func TestUnknownButtons(t *testing.T) {
	h, content, _ := newSections(t)
	for _, data := range []string{"o:weird", "o:a:b:c:d", "t:save:n:0", "t:x", "t:alert:n", "t:alert:x:0", "t:f:0:extra"} {
		h.press(data)
		if c := h.expect("answerCallbackQuery")[0]; c.params["text"] != nil {
			t.Errorf("%s: %v", data, c.params)
		}
		h.quiet()
	}
	if len(content.alerts) != 0 || len(content.calls()) != 0 {
		t.Errorf("malformed buttons changed something: %v %v", content.alerts, content.calls())
	}
}

func TestMyToursEdges(t *testing.T) {
	h, content, _ := newSections(t)

	// Asked to cancel a sign-up the user hasn't got: just the card.
	h.press("o:ask:t:o2")
	if text := h.expect("answerCallbackQuery", "editMessageText")[1].text(); !strings.HasPrefix(text, "🧭 <b>Барскоон</b>") {
		t.Errorf("ask, not joined: %q", text)
	}
	// Signing up for a tour deleted meanwhile: back to the list.
	content.joinErr = &api.Error{Status: 404, Message: "Tour not found"}
	h.press("o:join:t:gone")
	if c := h.expect("answerCallbackQuery", "editMessageText"); c[0].params["text"] != "Этого тура больше нет" {
		t.Errorf("join gone: %v", c[0].params)
	}
	// Any other refusal is an error.
	content.joinErr = &api.Error{Status: 500}
	h.press("o:join:t:o2")
	if c := h.expect("answerCallbackQuery")[0]; c.params["text"] != loadFailed {
		t.Errorf("join failed: %v", c.params)
	}
	h.quiet()

	h.press("o:list:p")
	if text := h.expect("answerCallbackQuery", "editMessageText")[1].text(); !strings.Contains(text, "Прошедших туров пока нет.") {
		t.Errorf("past: %q", text)
	}
}

func TestToggles(t *testing.T) {
	h, content, subs := newSections(t)
	content.alerts = []string{""}
	h.press("t:alert:n:0")
	if c := h.expect("answerCallbackQuery", "editMessageText"); c[0].params["text"] != "Больше не сообщаем о новых турах" ||
		len(content.alerts) != 0 {
		t.Errorf("alert off: %v %v", c[0].params, content.alerts)
	}
	content.favorites["t1"] = true
	h.press("t:drop:f:0:t1")
	if c := h.expect("answerCallbackQuery", "editMessageText"); c[0].params["text"] != "Убрано из избранного" ||
		!strings.Contains(c[1].text(), "В избранном пока пусто") {
		t.Errorf("drop: %v %q", c[0].params, c[1].text())
	}

	subs.subs[chatID] = true
	h.press("b:unsub:0")
	if c := h.expect("answerCallbackQuery", "editMessageText"); c[0].params["text"] != "Больше не присылаем статьи" || subs.subs[chatID] {
		t.Errorf("unsubscribe: %v", c[0].params)
	}
	h.press("n:blog")
	if c := h.expect("answerCallbackQuery", "editMessageText"); c[0].params["text"] != "Статьи блога: включено" || !subs.subs[chatID] {
		t.Errorf("blog on: %v", c[0].params)
	}
	h.press("n:blog")
	if c := h.expect("answerCallbackQuery", "editMessageText"); c[0].params["text"] != "Статьи блога: выключено" {
		t.Errorf("blog off: %v", c[0].params)
	}

	// Destinations followed on the site: the button turns on every new tour.
	content.alerts = []string{"karakol", "naryn"}
	h.message("🔔 Уведомления")
	msg := h.expect("sendMessage")[0]
	if !strings.Contains(msg.text(), "по 2 направлениям, выбранным на сайте") || !strings.Contains(msg.markup(), "🧭 Новые туры: включить все") {
		t.Errorf("partial: %q %s", msg.text(), msg.markup())
	}
	h.press("n:tours")
	if c := h.expect("answerCallbackQuery", "editMessageText"); c[0].params["text"] != "Новые туры: включено" {
		t.Errorf("all tours: %v", c[0].params)
	}
}

// Without the API the sections don't exist: their buttons are plain text.
func TestWithoutSections(t *testing.T) {
	h := newHarness(t, miniApp)
	h.message("🎒 Мои туры")
	h.press("o:list:u")
	h.quiet()
	h.message("/start tour_o1")
	if text := h.expect("sendMessage")[0].text(); !strings.HasPrefix(text, "Здравствуйте, Айгерим! Я помогу") {
		t.Errorf("deep link without sections: %q", text)
	}
}

// The users service down: logged, no half-done reply.
func TestSectionsUsersDown(t *testing.T) {
	h, _, _ := newSections(t)
	h.users.err = errors.New("unavailable")
	h.message("🎒 Мои туры")
	h.message("/start tour_o1")
	h.quiet()
}
