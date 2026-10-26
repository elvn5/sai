package handlers_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/elvn5/sai/services/bot/internal/analytics"
	"github.com/elvn5/sai/services/bot/internal/handlers"
)

// fakeTracker records the analytics events.
type fakeTracker struct {
	mu     sync.Mutex
	events []analytics.Event
}

func (f *fakeTracker) Track(e analytics.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)
}

// names lists the events as "<name> <main param> <lang>", comma-separated.
func (f *fakeTracker) names() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, e := range f.events {
		param := ""
		for _, key := range []string{"deep_link", "section", "tour_id", "post_id"} {
			if v, ok := e.Params[key]; ok {
				param = fmt.Sprint(v)
				break
			}
		}
		if e.ChatID != chatID {
			param += fmt.Sprintf(" (chat %d)", e.ChatID)
		}
		out = append(out, strings.Join(strings.Fields(e.Name+" "+param+" "+e.Lang), " "))
	}
	return strings.Join(out, ",")
}

func withTracker(t *fakeTracker) func(*handlers.Handlers) {
	return func(h *handlers.Handlers) { h.Analytics = t }
}

// What a user does in the chat is reported, in the chat's language, and
// deep link tokens never are.
func TestAnalyticsEvents(t *testing.T) {
	events := &fakeTracker{}
	langs := &fakeLanguages{langs: map[int64]string{chatID: "ky"}}
	h, _, _ := newSections(t, withTracker(events), withLanguages(langs))

	h.message("/start")
	h.expect("sendMessage")
	h.message("/start tour_o2")
	h.expect("sendMessage")
	h.message("/start verify_secret-token")
	h.expect("sendMessage")
	h.message("🎒 Мои туры")
	h.expect("sendMessage")
	h.message("/tours")
	h.expect("sendMessage")
	h.press("o:join:t:o2")
	h.expect("answerCallbackQuery", "editMessageText")
	h.press("o:leave:t:o2")
	h.expect("answerCallbackQuery", "editMessageText")
	h.press("t:more:n:1:0")
	h.expect("answerCallbackQuery", "editMessageText")
	h.press("t:more:n:1:1") // the description's next page: not a new view
	h.expect("answerCallbackQuery", "editMessageText")
	h.press("t:open:t1")
	h.expect("answerCallbackQuery", "sendMessage")
	h.press("b:read:p1:0")
	h.expect("answerCallbackQuery", "editMessageText")
	h.press("l:de")
	h.expect("answerCallbackQuery", "editMessageReplyMarkup", "setChatMenuButton", "sendMessage")

	want := "bot_start none ky,bot_start tour ky,bot_start verify ky,menu_select my_tours ky,menu_select tours ky," +
		"join_tour o2 ky,leave_tour o2 ky,view_tour t2 ky,view_tour t1 ky,view_article p1 ky,language_change de"
	if got := events.names(); got != want {
		t.Errorf("events:\n%s\nwant:\n%s", got, want)
	}
	for _, e := range events.events {
		if strings.Contains(fmt.Sprint(e.Params), "secret") {
			t.Errorf("a deep link token reported: %v", e)
		}
	}
	if e := events.events[8]; e.Params["source"] != "alert" {
		t.Errorf("alert source: %v", e.Params)
	}
}
