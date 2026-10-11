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
	// ReminderAhead: the reminder goes out this long before the start
	// (an early-morning start gets it the evening before).
	ReminderAhead = 12 * time.Hour
	// AssumedLength stands in for a missing return time.
	AssumedLength = 10 * time.Hour
	// FollowUpAfter: the "how did it go?" message, after the return.
	FollowUpAfter = 2 * time.Hour

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
	Store   *store.Store
	Users   Users
	Sender  Sender
	SiteURL string
	Log     *slog.Logger
	Now     func() time.Time
	// Interval between worker passes.
	Interval time.Duration
}

func (n *Notifier) now() time.Time {
	if n.Now != nil {
		return n.Now()
	}
	return time.Now()
}

func (n *Notifier) outingURL(o models.Outing) string { return n.SiteURL + "/outings/" + o.ID }

func (n *Notifier) outingButton(o models.Outing) (*string, *string) {
	text, url := "Открыть выход", n.outingURL(o)
	return &text, &url
}

func (n *Notifier) message(userID, kind string, o models.Outing, text string) store.Notification {
	btnText, btnURL := n.outingButton(o)
	return store.Notification{UserID: userID, Kind: kind, OutingID: &o.ID, Text: text, ButtonText: btnText, ButtonURL: btnURL}
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
	due, err := n.Store.OutingsDueForReminder(ctx, now, ReminderAhead)
	if err != nil {
		return err
	}
	for _, id := range due {
		o, participants, err := n.loadOuting(ctx, id)
		if err != nil {
			return err
		}
		items := []store.Notification{n.once(n.message(o.UserID, "reminder", o, reminderText(o, true)))}
		for _, p := range participants {
			items = append(items, n.once(n.message(p, "reminder", o, reminderText(o, false))))
		}
		if err := n.Store.Enqueue(ctx, items, n.now()); err != nil {
			return err
		}
		if err := n.Store.MarkReminderSent(ctx, id); err != nil {
			return err
		}
	}

	due, err = n.Store.OutingsDueForFollowUp(ctx, now, AssumedLength, FollowUpAfter)
	if err != nil {
		return err
	}
	for _, id := range due {
		o, participants, err := n.loadOuting(ctx, id)
		if err != nil {
			return err
		}
		text, url := "Оставить сводку", n.SiteURL+"/trails/"+o.Trail.Slug
		items := make([]store.Notification, 0, len(participants)+1)
		for _, userID := range append([]string{o.UserID}, participants...) {
			items = append(items, n.once(store.Notification{
				UserID: userID, Kind: "followup", OutingID: &o.ID, Text: followUpText(o), ButtonText: &text, ButtonURL: &url,
			}))
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

// once sets a dedup key so a scheduled message is queued at most once.
func (n *Notifier) once(item store.Notification) store.Notification {
	key := fmt.Sprintf("%s:%s:%s", item.Kind, *item.OutingID, item.UserID)
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
	if item.ButtonText != nil && item.ButtonURL != nil {
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
