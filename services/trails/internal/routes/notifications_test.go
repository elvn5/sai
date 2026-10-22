package routes_test

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/telegram"
	"github.com/elvn5/sai/services/pkg/usersclient"
	"github.com/elvn5/sai/services/trails/internal/notify"
	"github.com/elvn5/sai/services/trails/internal/routes"
)

type sentMessage struct {
	chat   int64
	text   string
	button *telegram.Button
}

// fakeTelegram records messages; chats in `blocked` refuse them, and
// `failNext` makes the next sends fail with a server error.
type fakeTelegram struct {
	mu       sync.Mutex
	sent     []sentMessage
	blocked  map[int64]bool
	failNext int
}

func (f *fakeTelegram) SendMessage(_ context.Context, chat int64, text string, button *telegram.Button) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext > 0 {
		f.failNext--
		return context.DeadlineExceeded
	}
	if f.blocked[chat] {
		return telegram.ErrUndeliverable
	}
	f.sent = append(f.sent, sentMessage{chat, text, button})
	return nil
}

func (f *fakeTelegram) take() []sentMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.sent
	f.sent = nil
	return out
}

// usersWithChats adds Telegram chats to the fake users service: hiker and
// partner have Telegram, friend signed up by email. Chats read the bot in
// Russian unless langs says otherwise.
type usersWithChats struct {
	fakeUsers
	langs map[string]telegram.Lang
}

var chatOf = map[string]int64{hiker: 1001, partner: 1002, other: 1003}

func (u usersWithChats) TelegramRecipients(_ context.Context, ids []string) (map[string]usersclient.TelegramChat, error) {
	out := map[string]usersclient.TelegramChat{}
	for _, id := range ids {
		if chat, ok := chatOf[id]; ok {
			out[id] = usersclient.TelegramChat{ID: chat, Lang: telegram.ParseLang(string(u.langs[id]))}
		}
	}
	return out, nil
}

func TestOutingNotifications(t *testing.T) {
	clock := now
	deps := newDeps(t)
	deps.Now = func() time.Time { return clock }
	tg := &fakeTelegram{blocked: map[int64]bool{}}
	notifier := &notify.Notifier{Store: deps.Store, Users: usersWithChats{fakeUsers: users}, Sender: tg,
		Log: slog.New(slog.DiscardHandler), Now: func() time.Time { return clock }}
	deps.Notify = notifier
	gin.SetMode(gin.TestMode)
	c := client{t, routes.Setup(deps, users, secret)}
	ctx := context.Background()

	_, body := c.do("POST", "/trails", komsomolets, partner)
	trailID := obj(t, body)["id"].(string)
	// hiker organizes: Saturday 06:00 Bishkek, a week ahead of the clock.
	_, body = c.do("POST", "/outings", `{"trailId":"`+trailID+`","title":"Комсомолец <тест>","startsAt":"2026-10-17T06:00:00+06:00",
		"meetingPoint":"Южные ворота","transport":"car","seats":3,"level":"beginner","contact":"@hiker_kg"}`, hiker)
	id := obj(t, body)["id"].(string)

	// Joins and leaves go to the organizer; friend has no Telegram.
	c.do("POST", "/outings/"+id+"/join", "", partner)
	c.do("POST", "/outings/"+id+"/join", "", friend)
	c.do("DELETE", "/outings/"+id+"/join", "", friend)
	notifier.Tick(ctx)
	sent := tg.take()
	if len(sent) != 3 || sent[0].chat != 1001 || !strings.Contains(sent[0].text, "<b>User 1111</b> присоединяется") ||
		!strings.Contains(sent[0].text, "«Комсомолец &lt;тест&gt;» (сб, 17 окт., 06:00)") || !strings.Contains(sent[0].text, "Занято мест: 1 из 3") ||
		!strings.Contains(sent[2].text, "не сможет пойти") || sent[0].button.Data != "o:view:n:"+id || sent[0].button.URL != "" {
		t.Fatalf("join/leave: %+v", sent)
	}

	// Edits that matter reach participants; cosmetic ones don't.
	c.do("PUT", "/outings/"+id, `{"description":"Берите воду"}`, hiker)
	c.do("PUT", "/outings/"+id, `{"startsAt":"2026-10-17T07:30:00+06:00","meetingPoint":"Асанбай"}`, hiker)
	notifier.Tick(ctx)
	sent = tg.take()
	if len(sent) != 1 || sent[0].chat != 1002 || !strings.Contains(sent[0].text, "Сбор: <b>сб, 17 окт., 07:30</b> (было сб, 17 окт., 06:00)") ||
		!strings.Contains(sent[0].text, "Место сбора: <b>Асанбай</b>") {
		t.Fatalf("changed: %+v", sent)
	}

	// The reminder: not yet a day ahead, then 12 hours before, once, with
	// the contact for participants and the head count for the organizer.
	clock = time.Date(2026, 10, 16, 6, 0, 0, 0, time.UTC) // 12:00 Bishkek, 19.5 h before
	notifier.Tick(ctx)
	if sent := tg.take(); len(sent) != 0 {
		t.Fatalf("early reminder: %+v", sent)
	}
	clock = time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC) // 20:00 Bishkek, 11.5 h before
	notifier.Tick(ctx)
	notifier.Tick(ctx)
	sent = tg.take()
	if len(sent) != 2 {
		t.Fatalf("reminders: %+v", sent)
	}
	for _, m := range sent {
		switch m.chat {
		case 1001:
			if !strings.Contains(m.text, "Участников: 1") || m.button == nil || m.button.Data != "g:rn:"+id {
				t.Errorf("organizer reminder: %s %+v", m.text, m.button)
			}
		case 1002:
			if !strings.Contains(m.text, "Организатор: @hiker_kg") || !strings.Contains(m.text, "📍 Асанбай") {
				t.Errorf("participant reminder: %s", m.text)
			}
		default:
			t.Errorf("reminder to %d", m.chat)
		}
	}

	// After the outing: the organizer reports on the route, the
	// participant rates the outing — both answered in the bot.
	clock = time.Date(2026, 10, 17, 14, 0, 0, 0, time.UTC) // start + 10 h + 2 h
	notifier.Tick(ctx)
	sent = tg.take()
	if len(sent) != 2 {
		t.Fatalf("follow-up: %+v", sent)
	}
	for _, m := range sent {
		switch {
		case m.chat == 1001 && m.button.Data == "c:s:"+id && strings.Contains(m.text, "Как прошёл выход"):
		case m.chat == 1002 && m.button.Data == "r:s:"+id && strings.Contains(m.text, "Оцените его"):
		default:
			t.Errorf("follow-up to %d: %q %+v", m.chat, m.text, m.button)
		}
	}
	notifier.Tick(ctx)
	if sent := tg.take(); len(sent) != 0 {
		t.Fatalf("follow-up repeated: %+v", sent)
	}
}

func TestNotificationDelivery(t *testing.T) {
	clock := now
	deps := newDeps(t)
	deps.Now = func() time.Time { return clock }
	tg := &fakeTelegram{blocked: map[int64]bool{1003: true}}
	notifier := &notify.Notifier{Store: deps.Store, Users: usersWithChats{fakeUsers: users}, Sender: tg,
		Log: slog.New(slog.DiscardHandler), Now: func() time.Time { return clock }}
	deps.Notify = notifier
	gin.SetMode(gin.TestMode)
	c := client{t, routes.Setup(deps, users, secret)}
	ctx := context.Background()

	_, body := c.do("POST", "/outings", `{"title":"Хайк","startsAt":"2026-10-17T06:00:00+06:00","meetingPoint":"Ала-Арча","transport":"bus","seats":3,"level":"beginner"}`, hiker)
	id := obj(t, body)["id"].(string)
	c.do("POST", "/outings/"+id+"/join", "", other)   // other blocked the bot
	c.do("POST", "/outings/"+id+"/join", "", partner) // delivered after a failure

	// A transient failure is retried after a backoff; a blocked chat isn't.
	tg.failNext = 1
	notifier.Tick(ctx)
	if sent := tg.take(); len(sent) != 1 {
		t.Fatalf("first pass: %+v", sent)
	}
	notifier.Tick(ctx)
	if sent := tg.take(); len(sent) != 0 {
		t.Fatalf("retried before backoff: %+v", sent)
	}
	clock = clock.Add(2 * time.Minute)
	notifier.Tick(ctx)
	if sent := tg.take(); len(sent) != 1 || sent[0].chat != 1001 {
		t.Fatalf("retry: %+v", sent)
	}

	// Cancelling tells participants who can be reached.
	c.do("POST", "/outings/"+id+"/cancel", "", hiker)
	notifier.Tick(ctx)
	sent := tg.take()
	if len(sent) != 1 || sent[0].chat != 1002 || !strings.Contains(sent[0].text, "отменён организатором") {
		t.Fatalf("cancelled: %+v", sent)
	}
}

func TestMovedOutingIsRemindedAgain(t *testing.T) {
	clock := now
	deps := newDeps(t)
	deps.Now = func() time.Time { return clock }
	tg := &fakeTelegram{blocked: map[int64]bool{}}
	notifier := &notify.Notifier{Store: deps.Store, Users: usersWithChats{fakeUsers: users}, Sender: tg,
		Log: slog.New(slog.DiscardHandler), Now: func() time.Time { return clock }}
	deps.Notify = notifier
	gin.SetMode(gin.TestMode)
	c := client{t, routes.Setup(deps, users, secret)}
	ctx := context.Background()

	_, body := c.do("POST", "/outings", `{"title":"Хайк","startsAt":"2026-10-17T06:00:00+06:00","meetingPoint":"Ала-Арча","transport":"bus","seats":3,"level":"beginner"}`, hiker)
	id := obj(t, body)["id"].(string)

	clock = time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC) // inside the window
	notifier.Tick(ctx)
	if sent := tg.take(); len(sent) != 1 || !strings.Contains(sent[0].text, "сб, 17 окт., 06:00") {
		t.Fatalf("first reminder: %+v", sent)
	}

	// Moved a week later: a new reminder when the new window opens.
	c.do("PUT", "/outings/"+id, `{"startsAt":"2026-10-24T06:00:00+06:00"}`, hiker)
	notifier.Tick(ctx)
	tg.take()
	clock = time.Date(2026, 10, 23, 14, 0, 0, 0, time.UTC)
	notifier.Tick(ctx)
	if sent := tg.take(); len(sent) != 1 || !strings.Contains(sent[0].text, "сб, 24 окт., 06:00") {
		t.Fatalf("reminder after the move: %+v", sent)
	}
	notifier.Tick(ctx)
	if sent := tg.take(); len(sent) != 0 {
		t.Fatalf("reminded twice: %+v", sent)
	}
}

// Each message comes in the recipient's language; callback data stays the
// same.
func TestNotificationLanguages(t *testing.T) {
	type expected struct {
		joined, changed, cancelled, organizerReminder, checkIn, participantReminder, followUp, report, rate, rateButton, openButton string
	}
	for lang, want := range map[telegram.Lang]expected{
		telegram.KY: {
			joined:  "<b>User 1111</b> «Sunday hike» (иш, 17-окт., 06:00) чыгуусуна кошулат.\nЭэленген орундар: 1 / 3.",
			changed: "🕕 Чогулуу: <b>иш, 17-окт., 07:30</b> (мурда иш, 17-окт., 06:00)", cancelled: "чыгуусун уюштуруучу жокко чыгарды",
			organizerReminder: "👥 Катышуучулар: 1", checkIn: "📋 Келгендерди белгилөө", participantReminder: "💬 Уюштуруучу: @hiker_kg",
			followUp: "чыгуусу кандай өттү?", report: "📝 Маршрут тууралуу айтуу", rate: "Баа бериңиз", rateButton: "⭐ Чыгууга баа берүү",
			openButton: "Чыгууну ачуу",
		},
		telegram.EN: {
			joined:  "<b>User 1111</b> is joining the outing “Sunday hike” (Sat, 17 Oct, 06:00).\nSeats taken: 1 of 3.",
			changed: "🕕 Meeting time: <b>Sat, 17 Oct, 07:30</b> (was Sat, 17 Oct, 06:00)", cancelled: "The organizer cancelled the outing “Sunday hike”",
			organizerReminder: "👥 Participants: 1", checkIn: "📋 Check people in", participantReminder: "💬 Organizer: @hiker_kg",
			followUp: "How was the outing “Sunday hike”?", report: "📝 Report on the route", rate: "Rate it", rateButton: "⭐ Rate the outing",
			openButton: "Open the outing",
		},
		telegram.DE: {
			joined:  "<b>User 1111</b> nimmt am Ausflug „Sunday hike“ (Sa., 17. Okt., 06:00) teil.\nBelegte Plätze: 1 von 3.",
			changed: "🕕 Treffen: <b>Sa., 17. Okt., 07:30</b> (vorher Sa., 17. Okt., 06:00)", cancelled: "wurde vom Organisator abgesagt",
			organizerReminder: "👥 Teilnehmende: 1", checkIn: "📋 Anwesenheit erfassen", participantReminder: "💬 Organisator: @hiker_kg",
			followUp: "Wie war der Ausflug „Sunday hike“?", report: "📝 Route bewerten", rate: "Bewerten Sie ihn", rateButton: "⭐ Ausflug bewerten",
			openButton: "Ausflug öffnen",
		},
	} {
		t.Run(string(lang), func(t *testing.T) {
			clock := now
			deps := newDeps(t)
			deps.Now = func() time.Time { return clock }
			tg := &fakeTelegram{blocked: map[int64]bool{}}
			notifier := &notify.Notifier{Store: deps.Store, Users: usersWithChats{fakeUsers: users, langs: map[string]telegram.Lang{hiker: lang, partner: lang}},
				Sender: tg, Log: slog.New(slog.DiscardHandler), Now: func() time.Time { return clock }}
			deps.Notify = notifier
			gin.SetMode(gin.TestMode)
			c := client{t, routes.Setup(deps, users, secret)}
			ctx := context.Background()
			// No Russian words outside the ones users wrote (the route's
			// name) in English and German.
			russian := func(m sentMessage) bool {
				text := strings.ReplaceAll(m.text, "Пик Комсомолец", "")
				return lang != telegram.KY && strings.ContainsFunc(text+m.button.Text, func(r rune) bool { return r >= 'а' && r <= 'я' })
			}

			_, body := c.do("POST", "/trails", komsomolets, partner)
			trailID := obj(t, body)["id"].(string)
			_, body = c.do("POST", "/outings", `{"trailId":"`+trailID+`","title":"Sunday hike","startsAt":"2026-10-17T06:00:00+06:00",
				"meetingPoint":"South gate","transport":"car","seats":3,"level":"beginner","contact":"@hiker_kg"}`, hiker)
			id := obj(t, body)["id"].(string)

			c.do("POST", "/outings/"+id+"/join", "", partner)
			notifier.Tick(ctx)
			sent := tg.take()
			if len(sent) != 1 || !strings.Contains(sent[0].text, want.joined) || sent[0].button.Text != want.openButton ||
				sent[0].button.Data != "o:view:n:"+id || russian(sent[0]) {
				t.Fatalf("joined: %+v", sent)
			}

			c.do("PUT", "/outings/"+id, `{"startsAt":"2026-10-17T07:30:00+06:00"}`, hiker)
			notifier.Tick(ctx)
			sent = tg.take()
			if len(sent) != 1 || !strings.Contains(sent[0].text, want.changed) || russian(sent[0]) {
				t.Fatalf("changed: %+v", sent)
			}

			clock = time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC)
			notifier.Tick(ctx)
			for _, m := range tg.take() {
				switch {
				case m.chat == 1001 && strings.Contains(m.text, want.organizerReminder) && m.button.Text == want.checkIn && m.button.Data == "g:rn:"+id:
				case m.chat == 1002 && strings.Contains(m.text, want.participantReminder) && m.button.Data == "o:view:n:"+id:
				default:
					t.Errorf("reminder to %d: %q %+v", m.chat, m.text, m.button)
				}
				if russian(m) {
					t.Errorf("reminder in Russian: %q", m.text)
				}
			}

			clock = time.Date(2026, 10, 17, 16, 0, 0, 0, time.UTC)
			notifier.Tick(ctx)
			sent = tg.take()
			if len(sent) != 2 {
				t.Fatalf("follow-up: %+v", sent)
			}
			for _, m := range sent {
				switch {
				case m.chat == 1001 && strings.Contains(m.text, want.followUp) && m.button.Text == want.report && m.button.Data == "c:s:"+id:
				case m.chat == 1002 && strings.Contains(m.text, want.rate) && m.button.Text == want.rateButton && m.button.Data == "r:s:"+id:
				default:
					t.Errorf("follow-up to %d: %q %+v", m.chat, m.text, m.button)
				}
				if russian(m) {
					t.Errorf("follow-up in Russian: %q", m.text)
				}
			}

			c.do("POST", "/outings/"+id+"/cancel", "", hiker)
			notifier.Tick(ctx)
			if sent := tg.take(); len(sent) == 1 && (!strings.Contains(sent[0].text, want.cancelled) || russian(sent[0])) {
				t.Errorf("cancelled: %+v", sent)
			}
		})
	}
}
