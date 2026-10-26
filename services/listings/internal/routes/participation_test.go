package routes_test

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/listings/internal/handlers"
	"github.com/elvn5/sai/services/listings/internal/notify"
	"github.com/elvn5/sai/services/listings/internal/routes"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/telegram"
	"github.com/elvn5/sai/services/pkg/usersclient"
)

type sentMessage struct {
	chat   int64
	text   string
	button *telegram.Button
}

// recordingTelegram keeps what was sent.
type recordingTelegram struct {
	mu   sync.Mutex
	sent []sentMessage
}

func (f *recordingTelegram) SendMessage(_ context.Context, chat int64, text string, button *telegram.Button) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, sentMessage{chat, text, button})
	return nil
}

func (f *recordingTelegram) take() []sentMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.sent
	f.sent = nil
	return out
}

// usersWithChats: everyone but regular has Telegram; the partner's chat
// reads the bot in partnerLang.
type usersWithChats struct {
	fakeUsers
	partnerLang telegram.Lang
}

var chatOf = map[string]int64{owner: 1001, guest: 1002, partner: 1003}

func (u usersWithChats) TelegramRecipients(_ context.Context, ids []string) (map[string]usersclient.TelegramChat, error) {
	out := map[string]usersclient.TelegramChat{}
	for _, id := range ids {
		if chat, ok := chatOf[id]; ok {
			lang := telegram.RU
			if id == partner && u.partnerLang != "" {
				lang = u.partnerLang
			}
			out[id] = usersclient.TelegramChat{ID: chat, Lang: lang}
		}
	}
	return out, nil
}

func setupTrips(t *testing.T, partnerLang telegram.Lang) (client, *notify.Notifier, *recordingTelegram, *time.Time) {
	t.Helper()
	db, _ := createDatabase(t, true)
	st := store.New(db)
	clock := time.Date(2026, 10, 10, 6, 0, 0, 0, time.UTC)
	now := func() time.Time { return clock }
	users := usersWithChats{fakeUsers: testUsers, partnerLang: partnerLang}
	tg := &recordingTelegram{}
	log := slog.New(slog.DiscardHandler)
	notifier := &notify.Notifier{Store: st, Users: users, Sender: tg, Log: log, Now: now}
	gin.SetMode(gin.TestMode)
	deps := handlers.Deps{Store: st, Authors: users, Log: log, Notify: notifier, Now: now}
	return client{t, routes.Setup(deps, users, secret)}, notifier, tg, &clock
}

const tripTour = `{"locale":"ru","slug":"ala-kul","title":"Ала-Куль <тест>","content":"<p>Озеро</p>",
	"startDate":"2026-10-17","endDate":"2026-10-18","meetingTime":"06:00","meetingPoint":"Южные ворота",
	"capacity":2,"telegram":"@partner_kg"}`

func TestSigningUpForTours(t *testing.T) {
	c, notifier, tg, clock := setupTrips(t, telegram.RU)
	ctx := context.Background()
	status, body := c.do("POST", "/tours", tripTour, partner)
	tour := obj(t, body)
	if status != 201 || tour["meetingTime"] != "06:00" || tour["taken"] != float64(0) {
		t.Fatalf("create: %d %s", status, body)
	}
	id := tour["id"].(string)

	// Places run out; the organizer can't sign up for their own tour.
	if status, body := c.do("POST", "/tours/"+id+"/join", "", guest); status != 200 || obj(t, body)["isJoined"] != true || obj(t, body)["taken"] != float64(1) {
		t.Fatalf("join: %d %s", status, body)
	}
	c.do("POST", "/tours/"+id+"/join", "", owner)
	for user, want := range map[string]string{regular: "No places left", partner: "You organize this tour", guest: "already signed up"} {
		if _, body := c.do("POST", "/tours/"+id+"/join", "", user); !strings.Contains(body, want) {
			t.Errorf("join as %s: %s", user, body)
		}
	}
	notifier.Tick(ctx)
	sent := tg.take()
	if len(sent) != 2 || sent[0].chat != 1003 || !strings.Contains(sent[0].text, "<b>User 2222</b> записывается на тур «Ала-Куль &lt;тест&gt;» (сб, 17 окт., 06:00)") ||
		!strings.Contains(sent[1].text, "Занято мест: 2 из 2.") || sent[0].button.Data != "o:view:n:"+id {
		t.Fatalf("joined: %+v", sent)
	}
	c.do("DELETE", "/tours/"+id+"/join", "", owner)
	notifier.Tick(ctx)
	if sent := tg.take(); len(sent) != 1 || !strings.Contains(sent[0].text, "отменяет запись") || !strings.Contains(sent[0].text, "Занято мест: 1 из 2.") {
		t.Fatalf("left: %+v", sent)
	}

	// The organizer sees the roster; participants and visitors don't.
	if _, body := c.do("GET", "/tours/by-id/"+id, "", partner); !strings.Contains(body, `"roster":[{"id":"`+guest) || obj(t, body)["isOrganizer"] != true {
		t.Errorf("organizer's view: %s", body)
	}
	if _, body := c.do("GET", "/tours/by-id/"+id, "", guest); strings.Contains(body, "roster") || obj(t, body)["isJoined"] != true {
		t.Errorf("participant's view: %s", body)
	}
	if _, body := c.do("GET", "/tours/joined", "", guest); !strings.Contains(body, id) {
		t.Errorf("joined: %s", body)
	}

	// A new meeting time reaches participants; the capacity can't drop
	// below those signed up.
	c.do("PUT", "/tours/"+id, `{"meetingTime":"07:30","excerpt":"Берите воду"}`, partner)
	notifier.Tick(ctx)
	if sent := tg.take(); len(sent) != 1 || sent[0].chat != 1002 || !strings.Contains(sent[0].text, "Сбор: <b>сб, 17 окт., 07:30</b> (было сб, 17 окт., 06:00)") {
		t.Fatalf("changed: %+v", sent)
	}
	if status, _ := c.do("PUT", "/tours/"+id, `{"capacity":1}`, partner); status != 200 {
		t.Errorf("capacity to taken: %d", status)
	}

	// Check-in opens a day before; rating once it started.
	if _, body := c.do("PUT", "/tours/"+id+"/participants/"+guest+"/arrival", "", partner); !strings.Contains(body, "Check-in opens") {
		t.Errorf("early check-in: %s", body)
	}
	if _, body := c.do("PUT", "/tours/"+id+"/rating", `{"rating":5}`, guest); !strings.Contains(body, "hasn't started") {
		t.Errorf("early rating: %s", body)
	}

	// The reminder, 12 hours before the start.
	*clock = time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC) // 20:00 Bishkek
	notifier.Tick(ctx)
	notifier.Tick(ctx)
	sent = tg.take()
	if len(sent) != 2 {
		t.Fatalf("reminders: %+v", sent)
	}
	for _, m := range sent {
		switch m.chat {
		case 1003:
			if !strings.Contains(m.text, "Участников: 1") || m.button.Data != "g:rn:"+id {
				t.Errorf("organizer reminder: %s %+v", m.text, m.button)
			}
		case 1002:
			if !strings.Contains(m.text, "Организатор: @partner_kg") || !strings.Contains(m.text, "📍 Южные ворота") {
				t.Errorf("participant reminder: %s", m.text)
			}
		default:
			t.Errorf("reminder to %d", m.chat)
		}
	}
	if status, _ := c.do("PUT", "/tours/"+id+"/participants/"+guest+"/arrival", "", guest); status != 403 {
		t.Errorf("check-in by a participant: %d", status)
	}
	if _, body := c.do("PUT", "/tours/"+id+"/participants/"+guest+"/arrival", "", partner); !strings.Contains(body, `"arrivedAt":"2026-10-16`) {
		t.Errorf("check-in: %s", body)
	}

	// After it: participants are asked to rate it, and the rating is their
	// review of the tour.
	*clock = time.Date(2026, 10, 18, 14, 0, 0, 0, time.UTC) // the last day, 20:00 Bishkek
	notifier.Tick(ctx)
	if sent := tg.take(); len(sent) != 1 || sent[0].chat != 1002 || sent[0].button.Data != "r:s:"+id || !strings.Contains(sent[0].text, "Как прошёл тур") {
		t.Fatalf("follow-up: %+v", sent)
	}
	if _, body := c.do("PUT", "/tours/"+id+"/rating", `{"rating":5,"text":"  Отлично  "}`, guest); !strings.Contains(body, `"myRating":{"rating":5,"text":"Отлично"}`) {
		t.Errorf("rate: %s", body)
	}
	c.do("PUT", "/tours/"+id+"/rating", `{"rating":4}`, guest)
	if _, body := c.do("GET", "/tours/ru/ala-kul", "", ""); obj(t, body)["reviewCount"] != float64(1) || obj(t, body)["ratingAverage"] != float64(4) {
		t.Errorf("tour rating: %s", body)
	}
	if status, _ := c.do("PUT", "/tours/"+id+"/rating", `{"rating":4}`, regular); status != 403 {
		t.Errorf("rating by a stranger: %d", status)
	}

	// Deleting a tour that hasn't taken place tells who signed up.
	_, body = c.do("POST", "/tours", strings.Replace(strings.Replace(tripTour, "ala-kul", "son-kul", 1), "2026-10-1", "2026-11-1", 2), partner)
	next := obj(t, body)["id"].(string)
	c.do("POST", "/tours/"+next+"/join", "", guest)
	tg.take()
	c.do("DELETE", "/tours/"+next, "", partner)
	notifier.Tick(ctx)
	notifier.Tick(ctx)
	var cancelled []sentMessage
	for _, m := range tg.take() {
		if m.chat == 1002 {
			cancelled = append(cancelled, m)
		}
	}
	if len(cancelled) != 1 || !strings.Contains(cancelled[0].text, "отменён организатором") || cancelled[0].button != nil {
		t.Fatalf("cancelled: %+v", cancelled)
	}
}

// The organizer reads sign-ups in their own language.
func TestSignUpNotificationLanguage(t *testing.T) {
	for lang, want := range map[telegram.Lang]string{
		telegram.EN: "<b>User 2222</b> is signing up for the tour “Ала-Куль &lt;тест&gt;” (Sat, 17 Oct, 06:00).\nPlaces taken: 1 of 2.",
		telegram.DE: "Belegte Plätze: 1 von 2.",
		telegram.KY: "турына жазылат.\nЭэленген орундар: 1 / 2.",
	} {
		c, notifier, tg, _ := setupTrips(t, lang)
		_, body := c.do("POST", "/tours", tripTour, partner)
		id := obj(t, body)["id"].(string)
		c.do("POST", "/tours/"+id+"/join", "", guest)
		notifier.Tick(context.Background())
		if sent := tg.take(); len(sent) != 1 || !strings.Contains(sent[0].text, want) || sent[0].button.Data != "o:view:n:"+id {
			t.Errorf("%s: %+v", lang, sent)
		}
	}
}
