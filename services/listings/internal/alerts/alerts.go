// Package alerts tells travelers about new tours: a Telegram message, sent
// as the site's bot, to whoever follows the tour's destination (or any).
// Best effort: a failed message is logged, not retried. Each follower gets
// it in the language their chat reads the bot in.
package alerts

import (
	"context"
	"embed"
	"log/slog"
	"strings"
	"time"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/listings/internal/store"
	"github.com/elvn5/sai/services/pkg/telegram"
	"github.com/elvn5/sai/services/pkg/usersclient"
)

// Texts are written in Russian and translated with the catalogs in
// locales/ (keyed by the Russian text).
//
//go:embed locales/*.json
var locales embed.FS

var texts = telegram.MustLoadCatalog(locales, "locales")

// Pause between messages: well under Telegram's per-bot limit.
const sendGap = 50 * time.Millisecond

// Sender delivers one message (pkg/telegram).
type Sender interface {
	SendMessage(ctx context.Context, chatID int64, text string, button *telegram.Button) error
}

// Chats resolves users' Telegram chats and their languages (the users
// service).
type Chats interface {
	TelegramRecipients(ctx context.Context, ids []string) (map[string]usersclient.TelegramChat, error)
}

type Notifier struct {
	Store  *store.Store
	Chats  Chats
	Sender Sender
	Log    *slog.Logger
}

// Linked reports whether a user has a Telegram chat to write to.
func (n *Notifier) Linked(ctx context.Context, id string) (bool, error) {
	chats, err := n.Chats.TelegramRecipients(ctx, []string{id})
	if err != nil {
		return false, err
	}
	_, ok := chats[id]
	return ok, nil
}

// TourPublished writes to the tour's followers, except its own author. Run
// it in the background: it takes a while for many followers.
func (n *Notifier) TourPublished(ctx context.Context, tour models.Tour) {
	ids, err := n.Store.TourAlertSubscribers(ctx, tour.Destination)
	if err != nil {
		n.Log.ErrorContext(ctx, "tour alert subscribers", slog.Any("error", err))
		return
	}
	followers := make([]string, 0, len(ids))
	for _, id := range ids {
		if tour.UserID == nil || id != *tour.UserID {
			followers = append(followers, id)
		}
	}
	if len(followers) == 0 {
		return
	}
	chats, err := n.Chats.TelegramRecipients(ctx, followers)
	if err != nil {
		n.Log.ErrorContext(ctx, "tour alert chats", slog.Any("error", err))
		return
	}
	for _, chat := range chats {
		text, button := message(chat.Lang, tour)
		if err := n.Sender.SendMessage(ctx, chat.ID, text, button); err != nil {
			n.Log.WarnContext(ctx, "tour alert not sent", slog.Int64("chat", chat.ID), slog.Any("error", err))
		}
		time.Sleep(sendGap)
	}
}

func message(l telegram.Lang, tour models.Tour) (string, *telegram.Button) {
	lines := []string{texts.T(l, "🏔 <b>Новый тур на Ermekten</b>"), telegram.EscapeHTML(tour.Title),
		"📅 " + tour.StartDate + " — " + tour.EndDate}
	if tour.Price != nil {
		lines = append(lines, texts.F(l, "💳 %d сом за человека", *tour.Price))
	}
	// Opens the tour in the bot (services/bot answers this callback data).
	return strings.Join(lines, "\n"), &telegram.Button{Text: texts.T(l, "Открыть тур"), Data: "t:open:" + tour.ID}
}
