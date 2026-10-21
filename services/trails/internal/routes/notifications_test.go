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
// partner have Telegram, friend signed up by email.
type usersWithChats struct{ fakeUsers }

var chatOf = map[string]int64{hiker: 1001, partner: 1002, other: 1003}

func (usersWithChats) TelegramChats(_ context.Context, ids []string) (map[string]int64, error) {
	out := map[string]int64{}
	for _, id := range ids {
		if chat, ok := chatOf[id]; ok {
			out[id] = chat
		}
	}
	return out, nil
}

func TestOutingNotifications(t *testing.T) {
	clock := now
	deps := newDeps(t)
	deps.Now = func() time.Time { return clock }
	tg := &fakeTelegram{blocked: map[int64]bool{}}
	notifier := &notify.Notifier{Store: deps.Store, Users: usersWithChats{users}, Sender: tg,
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
			if !strings.Contains(m.text, "Участников: 1") {
				t.Errorf("organizer reminder: %s", m.text)
			}
		case 1002:
			if !strings.Contains(m.text, "Организатор: @hiker_kg") || !strings.Contains(m.text, "📍 Асанбай") {
				t.Errorf("participant reminder: %s", m.text)
			}
		default:
			t.Errorf("reminder to %d", m.chat)
		}
	}

	// After the outing: a request for a condition report, answered in the bot.
	clock = time.Date(2026, 10, 17, 14, 0, 0, 0, time.UTC) // start + 10 h + 2 h
	notifier.Tick(ctx)
	sent = tg.take()
	if len(sent) != 2 || !strings.Contains(sent[0].text, "Как прошёл выход") || sent[0].button.Data != "c:s:"+id {
		t.Fatalf("follow-up: %+v", sent)
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
	notifier := &notify.Notifier{Store: deps.Store, Users: usersWithChats{users}, Sender: tg,
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
	notifier := &notify.Notifier{Store: deps.Store, Users: usersWithChats{users}, Sender: tg,
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
