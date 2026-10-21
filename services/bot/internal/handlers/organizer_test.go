package handlers_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/handlers"
)

// fakeOrganizer stands in for the gateway's organizer calls and the users
// service's chats; arrivals change the outings of content.
type fakeOrganizer struct {
	mu      sync.Mutex
	content *fakeContent
	log     []string
	tours   []api.Tour
	orders  []api.Order
	chats   map[string]int64
	// arrivalErr refuses marking.
	arrivalErr error
}

func (f *fakeOrganizer) record(s string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log = append(f.log, s)
}

func (f *fakeOrganizer) calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.log...)
}

func (f *fakeOrganizer) SetArrival(_ context.Context, _, outingID, participantID string, arrived bool) (api.Outing, error) {
	f.record("arrival " + outingID + " " + participantID + map[bool]string{true: " on", false: " off"}[arrived])
	if f.arrivalErr != nil {
		return api.Outing{}, f.arrivalErr
	}
	f.content.mu.Lock()
	defer f.content.mu.Unlock()
	o := f.content.outings[outingID]
	roster := append([]api.RosterEntry(nil), o.Roster...)
	for i := range roster {
		if roster[i].ID == participantID {
			roster[i].ArrivedAt = nil
			if arrived {
				at := now
				roster[i].ArrivedAt = &at
			}
		}
	}
	o.Roster = roster
	f.content.outings[outingID] = o
	return o, nil
}

func (f *fakeOrganizer) MyTours(context.Context, string) ([]api.Tour, error) { return f.tours, nil }

func (f *fakeOrganizer) ListingStats(context.Context, string) (api.ListingStats, error) {
	return api.ListingStats{}, nil
}

func (f *fakeOrganizer) ReceivedOrders(context.Context, string) ([]api.Order, error) {
	return f.orders, nil
}

func (f *fakeOrganizer) TelegramChats(_ context.Context, ids []string) (map[string]int64, error) {
	out := map[string]int64{}
	for _, id := range ids {
		if chat, ok := f.chats[id]; ok {
			out[id] = chat
		}
	}
	return out, nil
}

func newOrganizer(t *testing.T) (*harness, *fakeContent, *fakeOrganizer) {
	t.Helper()
	content := &fakeContent{outings: map[string]api.Outing{}, favorites: map[string]bool{}}
	people := []api.Person{{ID: "p-anya-0001", FirstName: str("Аня")}, {ID: "p-bob-00002", Username: str("bob")}, {ID: "p-gone-0003"}}
	o := api.Outing{ID: "o9", Title: "Пик Учитель", StartsAt: now.Add(3 * time.Hour), Status: "open", Seats: 10, Taken: 3,
		MeetingPoint: "Вефа", IsOrganizer: true, Participants: people}
	for _, p := range people {
		o.Roster = append(o.Roster, api.RosterEntry{Person: p, JoinedAt: now.Add(-48 * time.Hour)})
	}
	content.outings["o9"] = o
	org := &fakeOrganizer{content: content, chats: map[string]int64{"p-anya-0001": 501, "p-bob-00002": 502},
		tours: []api.Tour{{ID: "t1", Title: "Сон-Кёль", StartDate: "2027-06-12", EndDate: "2027-06-14",
			Program: []api.ProgramDay{{Title: "Выезд", Text: "В 8:00"}, {Title: "Озеро"}}}},
		orders: []api.Order{{Kind: "tour", ItemID: "t1", Quantity: 2, Amount: 100, ContactName: "Айбек", ContactPhone: "+996", Status: "paid"}},
	}
	h := newHarness(t, miniApp, func(hs *handlers.Handlers) {
		hs.Content, hs.Subscribers = content, &fakeSubscribers{subs: map[int64]bool{}}
		hs.Organizer, hs.Chats = org, org
		hs.Now = func() time.Time { return now }
	})
	h.users.account.ID = "u1"
	return h, content, org
}

func TestOrganizerRoomAndArrivals(t *testing.T) {
	h, content, org := newOrganizer(t)
	h.message("/organizer")
	home := h.expect("sendMessage")[0]
	if !strings.Contains(home.text(), "✅ на месте 0 из 3") || !strings.Contains(home.markup(), `"g:r:o9"`) ||
		!strings.Contains(home.markup(), `"g:t:0"`) {
		t.Errorf("room: %q %s", home.text(), home.markup())
	}

	h.press("g:r:o9")
	roster := h.expect("answerCallbackQuery", "editMessageText")[1]
	if !strings.Contains(roster.text(), `1. ⬜ <a href="tg://user?id=501">Аня</a>`) || !strings.Contains(roster.markup(), `"g:a:o9:p-anya-0"`) {
		t.Errorf("roster: %q %s", roster.text(), roster.markup())
	}

	h.press("g:a:o9:p-anya-0")
	calls := h.expect("answerCallbackQuery", "editMessageText")
	if calls[0].params["text"] != "✅ Аня на месте" || !strings.Contains(calls[1].text(), "1. ✅") || !strings.Contains(calls[1].text(), "на месте 1 из 3") {
		t.Errorf("mark: %v %q", calls[0].params, calls[1].text())
	}
	h.press("g:a:o9:p-anya-0")
	if calls = h.expect("answerCallbackQuery", "editMessageText"); calls[0].params["text"] != "Аня: отметка снята" {
		t.Errorf("unmark: %v", calls[0].params)
	}

	h.press("g:all:o9")
	calls = h.expect("answerCallbackQuery", "editMessageText")
	if calls[0].params["text"] != "✅ Все на месте — хорошего похода!" || strings.Contains(calls[1].markup(), "Все на месте") {
		t.Errorf("all: %v %s", calls[0].params, calls[1].markup())
	}
	if got := strings.Join(org.calls(), ","); got != "arrival o9 p-anya-0001 on,arrival o9 p-anya-0001 off,"+
		"arrival o9 p-anya-0001 on,arrival o9 p-bob-00002 on,arrival o9 p-gone-0003 on" {
		t.Errorf("arrival calls: %s", got)
	}

	org.arrivalErr = &api.Error{Status: 409, Message: "Check-in opens a day before the start"}
	h.press("g:a:o9:p-bob-00")
	if calls = h.expect("answerCallbackQuery", "editMessageText"); calls[0].params["text"] != "Отмечать прибывших можно за сутки до сбора" {
		t.Errorf("refused: %v", calls[0].params)
	}

	// From the reminder: the roster comes as a message of its own.
	h.press("g:rn:o9")
	h.expect("answerCallbackQuery", "sendMessage")

	// Someone else's outing.
	content.mu.Lock()
	o := content.outings["o9"]
	o.IsOrganizer = false
	content.outings["o9"] = o
	content.mu.Unlock()
	h.press("g:r:o9")
	if calls = h.expect("answerCallbackQuery"); calls[0].params["text"] != "Это может только организатор выхода" {
		t.Errorf("not organizer: %v", calls[0].params)
	}
}

func TestOrganizerBroadcast(t *testing.T) {
	h, _, _ := newOrganizer(t)
	h.press("g:m:o9")
	prompt := h.expect("answerCallbackQuery", "editMessageText")[1]
	if !strings.HasPrefix(prompt.text(), "📣 <b>Сообщение участникам</b>") {
		t.Errorf("prompt: %q", prompt.text())
	}
	h.message("Сбор в 6:30 <у ворот>")
	preview := h.expect("sendMessage")[0]
	if !strings.Contains(preview.text(), "<blockquote>Сбор в 6:30 &lt;у ворот&gt;</blockquote>") || !strings.Contains(preview.markup(), `"g:ms:o9"`) {
		t.Errorf("preview: %q %s", preview.text(), preview.markup())
	}
	// Typing again replaces it.
	h.message("Сбор в 7:00")
	h.expect("sendMessage")

	h.press("g:ms:o9")
	calls := h.expect("answerCallbackQuery", "sendMessage", "sendMessage", "editMessageText")
	for i, chat := range []float64{501, 502} {
		m := calls[1+i]
		if m.params["chat_id"] != chat || !strings.HasSuffix(m.text(), "\n\nСбор в 7:00") || !strings.Contains(m.markup(), `"o:view:n:o9"`) {
			t.Errorf("delivered %d: %v %q", i, m.params["chat_id"], m.text())
		}
	}
	if got := calls[3].text(); !strings.HasPrefix(got, "✅ Сообщение отправлено: 2 из 3.") {
		t.Errorf("report: %q", got)
	}

	// Sent once: pressing again doesn't resend.
	h.press("g:ms:o9")
	if calls = h.expect("answerCallbackQuery", "editMessageReplyMarkup"); calls[0].params["text"] != "Сообщение устарело — напишите его заново" {
		t.Errorf("resend: %v", calls[0].params)
	}

	// Cancelled while writing: the next text is no message.
	h.press("g:m:o9")
	h.expect("answerCallbackQuery", "editMessageText")
	h.press("g:mx:o9")
	if calls = h.expect("answerCallbackQuery", "editMessageText"); calls[0].params["text"] != "Сообщение не отправлено" {
		t.Errorf("abort: %v", calls[0].params)
	}
	h.message("просто текст")
	h.quiet()
}

func TestOrganizerTours(t *testing.T) {
	h, _, _ := newOrganizer(t)
	h.press("g:t:0")
	card := h.expect("answerCallbackQuery", "editMessageText")[1]
	if !strings.HasPrefix(card.text(), "🧭 <b>Сон-Кёль</b>") || !strings.Contains(card.text(), "🎫 Онлайн-заявок: 1 · 2 человека") ||
		!strings.Contains(card.markup(), `"g:tp:0:0"`) || !strings.Contains(card.markup(), `"g:tb:0"`) {
		t.Errorf("tour: %q %s", card.text(), card.markup())
	}
	h.press("g:tp:0:1")
	if day := h.expect("answerCallbackQuery", "editMessageText")[1]; !strings.Contains(day.text(), "<b>День 2. Озеро</b>") {
		t.Errorf("program: %q", day.text())
	}
	h.press("g:tb:0")
	if list := h.expect("answerCallbackQuery", "editMessageText")[1]; !strings.Contains(list.text(), "1. Айбек · 2 человека · +996 · 100 сом") {
		t.Errorf("bookings: %q", list.text())
	}
	h.press("g:t:5")
	calls := h.expect("answerCallbackQuery", "editMessageText")
	if calls[0].params["text"] != "Этого тура больше нет" || !strings.Contains(calls[1].text(), "Кабинет организатора") {
		t.Errorf("gone tour: %v", calls[0].params)
	}
}
