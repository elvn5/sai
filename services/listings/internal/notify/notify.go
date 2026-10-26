// Package notify sends Telegram notifications about sign-ups for tours: to
// the organizer when people sign up or drop out, to participants when the
// tour moves or is cancelled, a reminder before it starts and a request to
// rate it after. Each is written in the recipient's language (the one
// their chat reads the bot in). Messages go through an outbox table and a
// background worker, so requests never wait on Telegram and failed sends
// are retried.
package notify

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/telegram"
	"github.com/elvn5/sai/services/pkg/usersclient"
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

// Users resolves profiles and Telegram chats with their languages (the
// users service).
type Users interface {
	Authors(ctx context.Context, ids []string) (map[string]usersclient.Author, error)
	TelegramRecipients(ctx context.Context, ids []string) (map[string]usersclient.TelegramChat, error)
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
func botButton(l telegram.Lang, text, data string) (*string, *string) {
	text = texts.T(l, text)
	return &text, &data
}

// tourButton opens the tour's card in the bot (the user's place on it),
// as a new message.
func tourButton(l telegram.Lang, t models.Tour) (*string, *string) {
	return botButton(l, "Открыть тур", "o:view:n:"+t.ID)
}

// checkInButton opens the organizer's roster in the bot, to mark who
// arrived.
func checkInButton(l telegram.Lang, t models.Tour) (*string, *string) {
	return botButton(l, "📋 Отметить прибывших", "g:rn:"+t.ID)
}

func (n *Notifier) message(l telegram.Lang, userID, kind string, t models.Tour, text string) store.Notification {
	btnText, btnData := tourButton(l, t)
	return store.Notification{UserID: userID, Kind: kind, TourID: &t.ID, Text: text, ButtonText: btnText, ButtonData: btnData}
}

// langs looks up the recipients' languages: Russian for a chat the bot
// hasn't reported one for, and for everyone when the users service is
// down (the message still goes out).
func (n *Notifier) langs(ctx context.Context, ids ...string) func(id string) telegram.Lang {
	chats, err := n.Users.TelegramRecipients(ctx, ids)
	if err != nil {
		n.Log.WarnContext(ctx, "recipients' languages", slog.Any("error", err))
	}
	return func(id string) telegram.Lang {
		if chat, ok := chats[id]; ok {
			return chat.Lang
		}
		return telegram.RU
	}
}

// enqueue queues notifications, logging rather than failing the request.
func (n *Notifier) enqueue(ctx context.Context, items []store.Notification) {
	if err := n.Store.Enqueue(ctx, items, n.now()); err != nil {
		n.Log.ErrorContext(ctx, "could not queue notifications", slog.Any("error", err))
	}
}

func (n *Notifier) person(ctx context.Context, id string) usersclient.Author {
	authors, err := n.Users.Authors(ctx, []string{id})
	if a, ok := authors[id]; err == nil && ok {
		return a
	}
	return usersclient.Author{ID: id}
}

func organizerOf(t models.Tour) string {
	if t.UserID == nil {
		return ""
	}
	return *t.UserID
}

// Joined tells the organizer someone signed up (t is the updated tour).
func (n *Notifier) Joined(ctx context.Context, t models.Tour, userID string) {
	owner := organizerOf(t)
	if owner == "" {
		return
	}
	l := n.langs(ctx, owner)(owner)
	n.enqueue(ctx, []store.Notification{n.message(l, owner, "joined", t, joinedText(l, t, n.person(ctx, userID)))})
}

// Left tells the organizer someone dropped out.
func (n *Notifier) Left(ctx context.Context, t models.Tour, userID string) {
	owner := organizerOf(t)
	if owner == "" {
		return
	}
	l := n.langs(ctx, owner)(owner)
	n.enqueue(ctx, []store.Notification{n.message(l, owner, "left", t, leftText(l, t, n.person(ctx, userID)))})
}

// Changed tells participants when the start or the meeting point changed.
func (n *Notifier) Changed(ctx context.Context, before, after models.Tour, participantIDs []string) {
	if changedText(telegram.RU, before, after) == "" || len(participantIDs) == 0 {
		return
	}
	lang := n.langs(ctx, participantIDs...)
	items := make([]store.Notification, 0, len(participantIDs))
	for _, id := range participantIDs {
		l := lang(id)
		items = append(items, n.message(l, id, "changed", after, changedText(l, before, after)))
	}
	n.enqueue(ctx, items)
}

// Cancelled tells participants the tour is off (it was deleted: the
// message has no button).
func (n *Notifier) Cancelled(ctx context.Context, t models.Tour, participantIDs []string) {
	if len(participantIDs) == 0 {
		return
	}
	lang := n.langs(ctx, participantIDs...)
	items := make([]store.Notification, 0, len(participantIDs))
	for _, id := range participantIDs {
		items = append(items, store.Notification{UserID: id, Kind: "cancelled", TourID: &t.ID, Text: cancelledText(lang(id), t)})
	}
	n.enqueue(ctx, items)
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
	due, err := n.Store.ToursDueForReminder(ctx, now)
	if err != nil {
		return err
	}
	for _, id := range due {
		t, participants, err := n.loadTour(ctx, id)
		if err != nil {
			return err
		}
		owner := organizerOf(t)
		lang := n.langs(ctx, append([]string{owner}, participants...)...)
		var items []store.Notification
		if owner != "" {
			l := lang(owner)
			toOrganizer := n.message(l, owner, "reminder", t, reminderText(l, t, true))
			toOrganizer.ButtonText, toOrganizer.ButtonData = checkInButton(l, t)
			items = append(items, once(toOrganizer, t.RemindAt))
		}
		for _, p := range participants {
			l := lang(p)
			items = append(items, once(n.message(l, p, "reminder", t, reminderText(l, t, false)), t.RemindAt))
		}
		if err := n.Store.Enqueue(ctx, items, n.now()); err != nil {
			return err
		}
		if err := n.Store.MarkReminderSent(ctx, id); err != nil {
			return err
		}
	}

	due, err = n.Store.ToursDueForFollowUp(ctx, now, now.Add(-followUpStale))
	if err != nil {
		return err
	}
	for _, id := range due {
		t, participants, err := n.loadTour(ctx, id)
		if err != nil {
			return err
		}
		// Participants rate the tour in the bot.
		lang := n.langs(ctx, participants...)
		items := make([]store.Notification, 0, len(participants))
		for _, userID := range participants {
			l := lang(userID)
			text, data := botButton(l, "⭐ Оценить тур", "r:s:"+t.ID)
			items = append(items, once(store.Notification{
				UserID: userID, Kind: "followup", TourID: &t.ID, Text: rateText(l, t), ButtonText: text, ButtonData: data,
			}, t.FollowUpAt))
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
// due time (a moved tour is reminded again).
func once(item store.Notification, due *time.Time) store.Notification {
	var at int64
	if due != nil {
		at = due.Unix()
	}
	key := fmt.Sprintf("%s:%s:%s:%d", item.Kind, *item.TourID, item.UserID, at)
	item.DedupKey = &key
	return item
}

func (n *Notifier) loadTour(ctx context.Context, id string) (models.Tour, []string, error) {
	t, err := n.Store.Tour(ctx, id)
	if err != nil {
		return t, nil, err
	}
	participants, err := n.Store.ParticipantIDs(ctx, id)
	return t, participants, err
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
	chats, err := n.Users.TelegramRecipients(ctx, ids)
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

func (n *Notifier) deliverOne(ctx context.Context, item store.Notification, chats map[string]usersclient.TelegramChat) error {
	chat, ok := chats[item.UserID]
	if !ok {
		reason := "no linked Telegram account"
		return n.Store.MarkNotification(ctx, item.ID, "skipped", item.Attempts, &reason, n.now())
	}
	var button *telegram.Button
	if item.ButtonText != nil && item.ButtonData != nil {
		button = &telegram.Button{Text: *item.ButtonText, Data: *item.ButtonData}
	}
	err := n.Sender.SendMessage(ctx, chat.ID, item.Text, button)
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
