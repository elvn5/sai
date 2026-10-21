// Package notify sends Telegram notifications about outings: to the
// organizer when people join or leave, to participants when the outing
// changes or is cancelled, a reminder before it starts and a request for a
// condition report after it. Messages go through an outbox table and a
// background worker, so requests never wait on Telegram and failed sends
// are retried.
package notify

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/elvn5/sai/services/pkg/telegram"
	"github.com/elvn5/sai/services/pkg/usersclient"
	"github.com/elvn5/sai/services/trails/internal/models"
	"github.com/elvn5/sai/services/trails/internal/store"
)

const (
	// A follow-up that couldn't go out within this long is dropped.
	followUpStale = 48 * time.Hour

	batchSize   = 50
	maxAttempts = 5
)

// Sender delivers one message (pkg/telegram).
type Sender interface {
	SendMessage(ctx context.Context, chatID int64, text string, button *telegram.Button) error
}

// Users resolves profiles and Telegram chats (the users service).
type Users interface {
	Authors(ctx context.Context, ids []string) (map[string]usersclient.Author, error)
	TelegramChats(ctx context.Context, ids []string) (map[string]int64, error)
}

type Notifier struct {
	Store  *store.Store
	Users  Users
	Sender Sender
	Log    *slog.Logger
	Now    func() time.Time
	// Interval between worker passes.
	Interval time.Duration
}

func (n *Notifier) now() time.Time {
	if n.Now != nil {
		return n.Now()
	}
	return time.Now()
}

// Buttons press back to the bot (services/bot answers this callback data):
// everything is done in the chat.
func botButton(text, data string) (*string, *string) { return &text, &data }

// outingButton opens the outing's card in the bot, as a new message.
func outingButton(o models.Outing) (*string, *string) {
	return botButton("Открыть выход", "o:view:n:"+o.ID)
}

// checkInButton opens the organizer's roster in the bot, to mark who
// arrived.
func checkInButton(o models.Outing) (*string, *string) {
	return botButton("📋 Отметить прибывших", "g:rn:"+o.ID)
}

func (n *Notifier) message(userID, kind string, o models.Outing, text string) store.Notification {
	btnText, btnData := outingButton(o)
	return store.Notification{UserID: userID, Kind: kind, OutingID: &o.ID, Text: text, ButtonText: btnText, ButtonData: btnData}
}

// enqueue queues notifications, logging rather than failing the request.
func (n *Notifier) enqueue(ctx context.Context, items []store.Notification) {
	if err := n.Store.Enqueue(ctx, items, n.now()); err != nil {
		n.Log.ErrorContext(ctx, "could not queue notifications", slog.Any("error", err))
	}
}

func (n *Notifier) person(ctx context.Context, id string) models.Person {
	authors, err := n.Users.Authors(ctx, []string{id})
	if a, ok := authors[id]; err == nil && ok {
		return models.Person{ID: a.ID, FirstName: a.FirstName, Username: a.Username}
	}
	return models.Person{ID: id}
}

// Joined tells the organizer someone joined (o is the updated outing).
func (n *Notifier) Joined(ctx context.Context, o models.Outing, userID string) {
	n.enqueue(ctx, []store.Notification{n.message(o.UserID, "joined", o, joinedText(o, n.person(ctx, userID)))})
}

// Left tells the organizer someone dropped out.
func (n *Notifier) Left(ctx context.Context, o models.Outing, userID string) {
	n.enqueue(ctx, []store.Notification{n.message(o.UserID, "left", o, leftText(o, n.person(ctx, userID)))})
}

// Changed tells participants when the start, meeting point or route
// changed.
func (n *Notifier) Changed(ctx context.Context, before, after models.Outing, participantIDs []string) {
	var changes []string
	if !before.StartsAt.Equal(after.StartsAt.Time) {
		changes = append(changes, "🕕 Сбор: <b>"+when(after.StartsAt.Time)+"</b> (было "+when(before.StartsAt.Time)+")")
	}
	if before.MeetingPoint != after.MeetingPoint {
		changes = append(changes, "📍 Место сбора: <b>"+esc(after.MeetingPoint)+"</b>")
	}
	if trailName(before) != trailName(after) {
		changes = append(changes, "🏔 Маршрут: <b>"+esc(orDash(trailName(after)))+"</b>")
	}
	if len(changes) == 0 {
		return
	}
	text := changedText(after, changes)
	items := make([]store.Notification, 0, len(participantIDs))
	for _, id := range participantIDs {
		items = append(items, n.message(id, "changed", after, text))
	}
	n.enqueue(ctx, items)
}

// Cancelled tells participants the outing is off.
func (n *Notifier) Cancelled(ctx context.Context, o models.Outing, participantIDs []string) {
	text := cancelledText(o)
	items := make([]store.Notification, 0, len(participantIDs))
	for _, id := range participantIDs {
		items = append(items, n.message(id, "cancelled", o, text))
	}
	n.enqueue(ctx, items)
}

func trailName(o models.Outing) string {
	if o.Trail == nil {
		return ""
	}
	return o.Trail.Name
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// Run schedules reminders and follow-ups and delivers the queue until ctx
// ends.
func (n *Notifier) Run(ctx context.Context) {
	interval := n.Interval
	if interval == 0 {
		interval = 20 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		n.Tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Tick is one worker pass.
func (n *Notifier) Tick(ctx context.Context) {
	if err := n.schedule(ctx); err != nil {
		n.Log.ErrorContext(ctx, "scheduling notifications failed", slog.Any("error", err))
	}
	if err := n.deliver(ctx); err != nil {
		n.Log.ErrorContext(ctx, "delivering notifications failed", slog.Any("error", err))
	}
}

func (n *Notifier) schedule(ctx context.Context) error {
	now := n.now()
	due, err := n.Store.OutingsDueForReminder(ctx, now)
	if err != nil {
		return err
	}
	for _, id := range due {
		o, participants, err := n.loadOuting(ctx, id)
		if err != nil {
			return err
		}
		toOrganizer := n.message(o.UserID, "reminder", o, reminderText(o, true))
		if o.Taken > 0 {
			toOrganizer.ButtonText, toOrganizer.ButtonData = checkInButton(o)
		}
		items := []store.Notification{once(toOrganizer, o.RemindAt)}
		for _, p := range participants {
			items = append(items, once(n.message(p, "reminder", o, reminderText(o, false)), o.RemindAt))
		}
		if err := n.Store.Enqueue(ctx, items, n.now()); err != nil {
			return err
		}
		if err := n.Store.MarkReminderSent(ctx, id); err != nil {
			return err
		}
	}

	due, err = n.Store.OutingsDueForFollowUp(ctx, now, now.Add(-followUpStale))
	if err != nil {
		return err
	}
	for _, id := range due {
		o, participants, err := n.loadOuting(ctx, id)
		if err != nil {
			return err
		}
		// Participants rate the outing (and, on a catalog route, then tell
		// how it is) in the bot; the organizer reports on the route.
		items := make([]store.Notification, 0, len(participants)+1)
		if o.Trail != nil {
			text, data := botButton("📝 Рассказать, как маршрут", "c:s:"+o.ID)
			items = append(items, once(store.Notification{
				UserID: o.UserID, Kind: "followup", OutingID: &o.ID, Text: followUpText(o), ButtonText: text, ButtonData: data,
			}, o.FollowUpAt))
		}
		text, data := botButton("⭐ Оценить выход", "r:s:"+o.ID)
		for _, userID := range participants {
			items = append(items, once(store.Notification{
				UserID: userID, Kind: "followup", OutingID: &o.ID, Text: rateText(o), ButtonText: text, ButtonData: data,
			}, o.FollowUpAt))
		}
		if err := n.Store.Enqueue(ctx, items, n.now()); err != nil {
			return err
		}
		if err := n.Store.MarkFollowUpSent(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// once sets a dedup key so a scheduled message is queued at most once per
// due time (a moved outing is reminded again).
func once(item store.Notification, due *time.Time) store.Notification {
	var at int64
	if due != nil {
		at = due.Unix()
	}
	key := fmt.Sprintf("%s:%s:%s:%d", item.Kind, *item.OutingID, item.UserID, at)
	item.DedupKey = &key
	return item
}

func (n *Notifier) loadOuting(ctx context.Context, id string) (models.Outing, []string, error) {
	o, err := n.Store.Outing(ctx, id)
	if err != nil {
		return o, nil, err
	}
	participants, err := n.Store.ParticipantIDs(ctx, id)
	return o, participants, err
}

func (n *Notifier) deliver(ctx context.Context) error {
	items, err := n.Store.DueNotifications(ctx, n.now(), batchSize)
	if err != nil || len(items) == 0 {
		return err
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.UserID)
	}
	chats, err := n.Users.TelegramChats(ctx, ids)
	if err != nil {
		return err // users service down: try the whole batch again later
	}
	for _, item := range items {
		if err := n.deliverOne(ctx, item, chats); err != nil {
			return err
		}
	}
	return nil
}

func (n *Notifier) deliverOne(ctx context.Context, item store.Notification, chats map[string]int64) error {
	chat, ok := chats[item.UserID]
	if !ok {
		reason := "no linked Telegram account"
		return n.Store.MarkNotification(ctx, item.ID, "skipped", item.Attempts, &reason, n.now())
	}
	var button *telegram.Button
	switch {
	case item.ButtonText != nil && item.ButtonData != nil:
		button = &telegram.Button{Text: *item.ButtonText, Data: *item.ButtonData}
	case item.ButtonText != nil && item.ButtonURL != nil:
		// Queued before buttons pressed back to the bot.
		button = &telegram.Button{Text: *item.ButtonText, URL: *item.ButtonURL}
	}
	err := n.Sender.SendMessage(ctx, chat, item.Text, button)
	var rateLimited *telegram.RateLimitedError
	switch {
	case err == nil:
		return n.Store.MarkNotification(ctx, item.ID, "sent", item.Attempts+1, nil, n.now())
	case errors.Is(err, telegram.ErrUndeliverable):
		reason := err.Error()
		return n.Store.MarkNotification(ctx, item.ID, "skipped", item.Attempts+1, &reason, n.now())
	case errors.As(err, &rateLimited):
		return n.Store.MarkNotification(ctx, item.ID, "pending", item.Attempts, nil, n.now().Add(rateLimited.After))
	default:
		reason := err.Error()
		attempts := item.Attempts + 1
		if attempts >= maxAttempts {
			n.Log.WarnContext(ctx, "giving up on a notification", slog.String("id", item.ID), slog.String("error", reason))
			return n.Store.MarkNotification(ctx, item.ID, "failed", attempts, &reason, n.now())
		}
		backoff := time.Minute << (attempts - 1)
		return n.Store.MarkNotification(ctx, item.ID, "pending", attempts, &reason, n.now().Add(backoff))
	}
}
