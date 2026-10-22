package handlers

import (
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/elvn5/sai/services/bot/internal/analytics"
	"github.com/elvn5/sai/services/bot/internal/i18n"
)

// Tracker takes the bot's analytics events (analytics.Client: queued, sent
// in the background).
type Tracker interface {
	Track(e analytics.Event)
}

// The events the bot reports; each carries the chat's language ("lang").
const (
	eventStart        = "bot_start"     // deep_link: verify, link, outing or none; outing_id
	eventMenu         = "menu_select"   // section
	eventJoinOuting   = "join_outing"   // outing_id
	eventLeaveOuting  = "leave_outing"  // outing_id
	eventCreateOuting = "create_outing" // outing_id
	eventLanguage     = "language_change"
	eventViewTour     = "view_tour"    // tour_id, source: list or alert
	eventViewArticle  = "view_article" // post_id
)

// track reports what a user did, without waiting.
func (h *Handlers) track(from *telego.User, l i18n.Lang, name string, params map[string]any) {
	if h.Analytics == nil || from == nil {
		return
	}
	h.Analytics.Track(analytics.Event{ChatID: from.ID, Name: name, Lang: string(l), Params: params})
}

// menu reports a main menu section opened (a button or its command), then
// shows it.
func (h *Handlers) menu(section string, next th.MessageHandler) th.MessageHandler {
	return func(ctx *th.Context, msg telego.Message) error {
		if h.Analytics != nil && msg.From != nil {
			h.track(msg.From, h.lang(ctx, msg.From), eventMenu, map[string]any{"section": section})
		}
		return next(ctx, msg)
	}
}

// startParams describes a /start deep link without its secrets (the
// verification and linking tokens).
func startParams(payload string) map[string]any {
	for _, kind := range []struct{ prefix, name string }{
		{verifyPrefix, "verify"}, {linkPrefix, "link"},
	} {
		if strings.HasPrefix(payload, kind.prefix) {
			return map[string]any{"deep_link": kind.name}
		}
	}
	if id, ok := strings.CutPrefix(payload, outingPrefix); ok && id != "" {
		return map[string]any{"deep_link": "outing", "outing_id": id}
	}
	if payload != "" {
		return map[string]any{"deep_link": "other"}
	}
	return map[string]any{"deep_link": "none"}
}
