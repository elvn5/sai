package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
	"github.com/elvn5/sai/services/bot/internal/screens"
)

// Content is the public API, as the outings, tours and blog sections use
// it (api.Client).
type Content interface {
	MyOutings(ctx context.Context, userID string) ([]api.Outing, error)
	OutingsSummary(ctx context.Context, userID string) (api.OutingsSummary, error)
	UpcomingOutings(ctx context.Context, userID string) ([]api.Outing, error)
	Outing(ctx context.Context, userID, id string) (api.Outing, error)
	JoinOuting(ctx context.Context, userID, id string) error
	LeaveOuting(ctx context.Context, userID, id string) error
	CreateOuting(ctx context.Context, userID string, in api.OutingInput) (api.Outing, error)
	UpdateOuting(ctx context.Context, userID, id string, in api.OutingInput) (api.Outing, error)
	CancelOuting(ctx context.Context, userID, id string) error
	Trails(ctx context.Context, search string) ([]api.TrailRef, error)
	ReportCondition(ctx context.Context, userID, trail string, in api.ConditionInput) error
	UpcomingTours(ctx context.Context, locale string) ([]api.Tour, error)
	Tour(ctx context.Context, locale, slug string) (api.Tour, error)
	FavoriteTours(ctx context.Context, userID string) ([]api.Tour, error)
	SetFavoriteTour(ctx context.Context, userID, tourID string, saved bool) error
	TourAlerts(ctx context.Context, userID string) ([]string, error)
	SetAllToursAlert(ctx context.Context, userID string, on bool) error
	BlogPosts(ctx context.Context, locale string) ([]api.BlogPost, error)
	BlogPost(ctx context.Context, locale, slug string) (api.BlogPost, error)
}

// Subscribers keeps who reads the blog in Telegram (store.Store).
type Subscribers interface {
	IsBlogSubscriber(ctx context.Context, chatID int64) (bool, error)
	SetBlogSubscriber(ctx context.Context, chatID int64, on bool) error
}

const (
	loadFailed = "😔 Не получилось загрузить данные. Попробуйте ещё раз чуть позже."
	// Telegram's answer to an edit that changes nothing.
	notModified = "message is not modified"
)

func (h *Handlers) sections() bool { return h.Content != nil && h.Subscribers != nil }

func (h *Handlers) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

// userOf is the account behind a Telegram user.
func (h *Handlers) userOf(ctx context.Context, from *telego.User) (string, error) {
	account, err := h.Users.FindOrCreate(ctx, profileOf(from))
	if err != nil {
		return "", err
	}
	return account.ID, nil
}

// send posts a screen as a new message.
func (h *Handlers) send(ctx *th.Context, chatID int64, s screens.Screen) error {
	msg := tu.Message(tu.ID(chatID), s.Text).WithParseMode(telego.ModeHTML)
	if s.Preview != nil {
		msg = msg.WithLinkPreviewOptions(s.Preview)
	}
	if s.Markup != nil {
		msg = msg.WithReplyMarkup(s.Markup)
	}
	if _, err := ctx.Bot().SendMessage(ctx, msg); err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
}

// show puts a screen in place of the pressed message — a section stays
// one message — or posts it anew when that message can't be edited (too
// old, or a photo).
func (h *Handlers) show(ctx *th.Context, q telego.CallbackQuery, s screens.Screen) error {
	if q.Message != nil && q.Message.IsAccessible() && q.Message.Message().Photo == nil {
		_, err := ctx.Bot().EditMessageText(ctx, &telego.EditMessageTextParams{
			ChatID: tu.ID(q.Message.GetChat().ID), MessageID: q.Message.GetMessageID(),
			Text: s.Text, ParseMode: telego.ModeHTML, LinkPreviewOptions: s.Preview, ReplyMarkup: s.Markup,
		})
		if err == nil || strings.Contains(err.Error(), notModified) {
			return nil
		}
		h.Log.DebugContext(ctx, "could not edit the screen, sending it anew", slog.Any("error", err))
	}
	return h.send(ctx, chatOf(q), s)
}

// failed logs an error and tells the user, instead of failing silently.
func (h *Handlers) failed(ctx *th.Context, l i18n.Lang, chatID int64, err error) error {
	h.Log.ErrorContext(ctx, "section failed", slog.Any("error", err))
	return h.reply(ctx, chatID, l.T(loadFailed), nil)
}

// failedPress logs an error and answers the button press with it.
func (h *Handlers) failedPress(ctx *th.Context, l i18n.Lang, q telego.CallbackQuery, err error) error {
	h.Log.ErrorContext(ctx, "section failed", slog.Any("error", err))
	return h.answer(ctx, q, l.T(loadFailed))
}

func (h *Handlers) noop(ctx *th.Context, q telego.CallbackQuery) error { return h.answer(ctx, q, "") }

// --- Outings ---

func (h *Handlers) myOutings(ctx context.Context, l i18n.Lang, userID, tab string) (screens.Screen, error) {
	list, err := h.Content.MyOutings(ctx, userID)
	if err != nil {
		return screens.Screen{}, err
	}
	var sum *api.OutingsSummary
	if s, err := h.Content.OutingsSummary(ctx, userID); err == nil {
		sum = &s
	} else {
		h.Log.WarnContext(ctx, "outings summary", slog.Any("error", err))
	}
	return screens.MyOutings(l, list, sum, tab, h.now()), nil
}

func (h *Handlers) outingsMenu(ctx *th.Context, msg telego.Message) error {
	if msg.From == nil {
		return nil
	}
	l := h.lang(ctx, msg.From)
	userID, err := h.userOf(ctx, msg.From)
	if err != nil {
		return err
	}
	s, err := h.myOutings(ctx, l, userID, screens.FromMine)
	if err != nil {
		return h.failed(ctx, l, msg.Chat.ID, err)
	}
	return h.send(ctx, msg.Chat.ID, s)
}

// outingCard sends an outing's card (a deep link from the site).
func (h *Handlers) outingCard(ctx *th.Context, l i18n.Lang, chatID int64, from *telego.User, id string) error {
	userID, err := h.userOf(ctx, from)
	if err != nil {
		return err
	}
	o, err := h.Content.Outing(ctx, userID, id)
	if api.StatusOf(err) == http.StatusNotFound || api.StatusOf(err) == http.StatusBadRequest {
		return h.reply(ctx, chatID, l.T("Такого выхода нет — возможно, его удалили."), nil)
	}
	if err != nil {
		return h.failed(ctx, l, chatID, err)
	}
	return h.send(ctx, chatID, screens.OutingCard(l, o, screens.FromMine, h.Links, h.now()))
}

// joinErrors are the trails service's refusals, as the user reads them.
var joinErrors = map[string]string{
	"No seats left":            "Мест больше нет 😔",
	"This outing is closed":    "Запись на этот выход закрыта",
	"You have already joined":  "Вы уже записаны на этот выход",
	"You organize this outing": "Это ваш выход — вы его организатор",
}

func (h *Handlers) outingPress(ctx *th.Context, q telego.CallbackQuery) error {
	parts := strings.Split(strings.TrimPrefix(q.Data, keyboards.OutingsPrefix), ":")
	l := h.lang(ctx, &q.From)
	userID, err := h.userOf(ctx, &q.From)
	if err != nil {
		return err
	}

	switch {
	case len(parts) == 2 && parts[0] == screens.ActList:
		s, err := h.myOutings(ctx, l, userID, parts[1])
		if err != nil {
			return h.failedPress(ctx, l, q, err)
		}
		if err := h.answer(ctx, q, ""); err != nil {
			return err
		}
		return h.show(ctx, q, s)

	case len(parts) == 2 && parts[0] == screens.ActFind:
		page, _ := strconv.Atoi(parts[1])
		list, err := h.Content.UpcomingOutings(ctx, userID)
		if err != nil {
			return h.failedPress(ctx, l, q, err)
		}
		h.markJoined(ctx, userID, list)
		if err := h.answer(ctx, q, ""); err != nil {
			return err
		}
		return h.show(ctx, q, screens.FindOutings(l, list, page))

	case len(parts) == 3:
		return h.outingAction(ctx, l, q, userID, parts[0], parts[1], parts[2])
	}
	return h.answer(ctx, q, "")
}

// markJoined marks the outings the user joined: the public list doesn't
// say (best effort).
func (h *Handlers) markJoined(ctx context.Context, userID string, list []api.Outing) {
	mine, err := h.Content.MyOutings(ctx, userID)
	if err != nil {
		h.Log.WarnContext(ctx, "my outings", slog.Any("error", err))
		return
	}
	joined := make(map[string]bool, len(mine))
	for _, o := range mine {
		joined[o.ID] = o.IsJoined
	}
	for i := range list {
		list[i].IsJoined = list[i].IsJoined || joined[list[i].ID]
	}
}

func (h *Handlers) outingAction(ctx *th.Context, l i18n.Lang, q telego.CallbackQuery, userID, action, from, id string) error {
	// A button under a notification opens the card as a message of its
	// own, keeping the notification.
	notice := from == screens.FromNotice
	if notice {
		from = screens.FromMine
	}
	toast := ""
	switch action {
	case screens.ActCancel:
		err := h.Content.CancelOuting(ctx, userID, id)
		if msg, known := organizerErrors[api.MessageOf(err)]; known {
			toast = l.T(msg)
		} else if err != nil && api.StatusOf(err) != http.StatusNotFound {
			return h.failedPress(ctx, l, q, err)
		} else if err == nil {
			toast = l.T("Выход отменён. Участники получат уведомление.")
		}
	case screens.ActJoin:
		err := h.Content.JoinOuting(ctx, userID, id)
		if msg, known := joinErrors[api.MessageOf(err)]; known {
			toast = l.T(msg)
		} else if err != nil && api.StatusOf(err) != http.StatusNotFound {
			return h.failedPress(ctx, l, q, err)
		} else if err == nil {
			toast = l.T("✅ Вы записаны! Организатор получит уведомление.")
			h.track(&q.From, l, eventJoinOuting, map[string]any{"outing_id": id})
		}
	case screens.ActLeave:
		err := h.Content.LeaveOuting(ctx, userID, id)
		if err != nil && api.StatusOf(err) != http.StatusNotFound {
			return h.failedPress(ctx, l, q, err)
		}
		if err == nil {
			toast = l.T("Вы больше не идёте на этот выход")
			h.track(&q.From, l, eventLeaveOuting, map[string]any{"outing_id": id})
		}
	}

	o, err := h.Content.Outing(ctx, userID, id)
	if api.StatusOf(err) == http.StatusNotFound || api.StatusOf(err) == http.StatusBadRequest {
		if err := h.answer(ctx, q, l.T("Этого выхода больше нет")); err != nil {
			return err
		}
		s, err := h.myOutings(ctx, l, userID, screens.FromMine)
		if err != nil {
			return h.failed(ctx, l, chatOf(q), err)
		}
		return h.show(ctx, q, s)
	}
	if err != nil {
		return h.failedPress(ctx, l, q, err)
	}
	if err := h.answer(ctx, q, toast); err != nil {
		return err
	}
	switch {
	case action == screens.ActAsk && o.IsJoined:
		return h.show(ctx, q, screens.LeaveConfirm(l, o, from))
	case action == screens.ActCancelAsk && o.IsOrganizer && o.Status == "open":
		return h.show(ctx, q, screens.CancelConfirm(l, o, from))
	case action == screens.ActEdit && o.IsOrganizer && o.Status == "open" && o.StartsAt.After(h.now()):
		return h.editDraft(ctx, l, q, o)
	case notice:
		return h.send(ctx, chatOf(q), screens.OutingCard(l, o, from, h.Links, h.now()))
	}
	return h.show(ctx, q, screens.OutingCard(l, o, from, h.Links, h.now()))
}

// --- Tours ---

func (h *Handlers) toursView(ctx context.Context, l i18n.Lang, userID, tab string, index int) (screens.ToursView, error) {
	favorites, err := h.Content.FavoriteTours(ctx, userID)
	if err != nil {
		return screens.ToursView{}, err
	}
	alerts, err := h.Content.TourAlerts(ctx, userID)
	if err != nil {
		return screens.ToursView{}, err
	}
	v := screens.ToursView{Tab: tab, Index: index, Favorites: map[string]bool{}}
	for _, t := range favorites {
		v.Favorites[t.ID] = true
	}
	for _, d := range alerts {
		if d == "" {
			v.AlertOn = true
		}
	}
	if tab == "" {
		v.Tab = screens.TabFavorites
		if len(favorites) == 0 {
			v.Tab = screens.TabUpcoming
		}
	}
	if v.Tab == screens.TabFavorites {
		v.Tours = favorites
		return v, nil
	}
	v.Tours, err = h.Content.UpcomingTours(ctx, string(l))
	return v, err
}

func (h *Handlers) toursMenu(ctx *th.Context, msg telego.Message) error {
	if msg.From == nil {
		return nil
	}
	l := h.lang(ctx, msg.From)
	userID, err := h.userOf(ctx, msg.From)
	if err != nil {
		return err
	}
	v, err := h.toursView(ctx, l, userID, "", 0)
	if err != nil {
		return h.failed(ctx, l, msg.Chat.ID, err)
	}
	return h.send(ctx, msg.Chat.ID, screens.Tours(l, v, h.Links))
}

func (h *Handlers) tourPress(ctx *th.Context, q telego.CallbackQuery) error {
	parts := strings.Split(strings.TrimPrefix(q.Data, keyboards.ToursPrefix), ":")
	l := h.lang(ctx, &q.From)
	userID, err := h.userOf(ctx, &q.From)
	if err != nil {
		return err
	}
	if len(parts) == 2 && parts[0] == screens.ActOpen {
		return h.openTour(ctx, l, q, userID, parts[1])
	}
	// "<tab>:<index>", or "<action>:<tab>:<index>[:<id>[:more]]", or
	// "more:<tab>:<index>:<page>".
	tab, action := parts[0], ""
	if len(parts) >= 3 {
		action, tab = parts[0], parts[1]
	}
	if len(parts) < 2 || (tab != screens.TabFavorites && tab != screens.TabUpcoming) {
		return h.answer(ctx, q, "")
	}
	index, _ := strconv.Atoi(parts[len(parts)-1])
	if action != "" {
		index, _ = strconv.Atoi(parts[2])
	}
	toast, more, page := "", false, 0
	switch action {
	case screens.ActMore:
		more = true
		if len(parts) == 4 {
			page, _ = strconv.Atoi(parts[3])
		}
	case screens.ActSave, screens.ActDrop:
		if len(parts) != 4 && len(parts) != 5 {
			return h.answer(ctx, q, "")
		}
		more = len(parts) == 5 && parts[4] == screens.ActMore
		saved := action == screens.ActSave
		if err := h.Content.SetFavoriteTour(ctx, userID, parts[3], saved); err != nil && api.StatusOf(err) != http.StatusNotFound {
			return h.failedPress(ctx, l, q, err)
		}
		toast = l.T("❤️ Добавлено в избранное")
		if !saved {
			toast = l.T("Убрано из избранного")
		}
	case screens.ActAlert:
		v, err := h.toursView(ctx, l, userID, tab, 0)
		if err != nil {
			return h.failedPress(ctx, l, q, err)
		}
		if err := h.Content.SetAllToursAlert(ctx, userID, !v.AlertOn); err != nil {
			return h.failedPress(ctx, l, q, err)
		}
		toast = l.T("🔔 Сообщим о новых турах")
		if v.AlertOn {
			toast = l.T("Больше не сообщаем о новых турах")
		}
	}
	v, err := h.toursView(ctx, l, userID, tab, index)
	if err != nil {
		return h.failedPress(ctx, l, q, err)
	}
	if more && index >= 0 && index < len(v.Tours) {
		t := v.Tours[index]
		full, err := h.Content.Tour(ctx, localeOf(t.Locale, l), t.Slug)
		if err != nil && api.StatusOf(err) != http.StatusNotFound {
			return h.failedPress(ctx, l, q, err)
		}
		if err == nil {
			if action == screens.ActMore && page == 0 {
				h.track(&q.From, l, eventViewTour, map[string]any{"tour_id": t.ID, "source": "list"})
			}
			if err := h.answer(ctx, q, toast); err != nil {
				return err
			}
			return h.show(ctx, q, screens.TourMore(l, full, tab, index, page, v.Favorites[full.ID], h.Links))
		}
		toast = l.T("Этот тур больше не опубликован")
	}
	if err := h.answer(ctx, q, toast); err != nil {
		return err
	}
	return h.show(ctx, q, screens.Tours(l, v, h.Links))
}

// openTour shows a new tour's description, from a new-tour alert, as a
// message of its own (the alert stays).
func (h *Handlers) openTour(ctx *th.Context, l i18n.Lang, q telego.CallbackQuery, userID, id string) error {
	v, err := h.toursView(ctx, l, userID, screens.TabUpcoming, 0)
	if err != nil {
		return h.failedPress(ctx, l, q, err)
	}
	index := slices.IndexFunc(v.Tours, func(t api.Tour) bool { return t.ID == id })
	if index < 0 {
		return h.answer(ctx, q, l.T("Этого тура уже нет среди ближайших"))
	}
	t := v.Tours[index]
	full, err := h.Content.Tour(ctx, localeOf(t.Locale, l), t.Slug)
	if api.StatusOf(err) == http.StatusNotFound {
		return h.answer(ctx, q, l.T("Этот тур больше не опубликован"))
	}
	if err != nil {
		return h.failedPress(ctx, l, q, err)
	}
	h.track(&q.From, l, eventViewTour, map[string]any{"tour_id": id, "source": "alert"})
	if err := h.answer(ctx, q, ""); err != nil {
		return err
	}
	return h.send(ctx, chatOf(q), screens.TourMore(l, full, screens.TabUpcoming, index, 0, v.Favorites[full.ID], h.Links))
}

// localeOf is the language a tour or post was written in, the reader's
// when it doesn't say.
func localeOf(locale string, l i18n.Lang) string {
	if locale == "" {
		return string(l)
	}
	return locale
}

// --- Blog ---

func (h *Handlers) blogScreen(ctx context.Context, l i18n.Lang, chatID int64, index int) (screens.Screen, error) {
	posts, err := h.Content.BlogPosts(ctx, string(l))
	if err != nil {
		return screens.Screen{}, err
	}
	subscribed, err := h.Subscribers.IsBlogSubscriber(ctx, chatID)
	if err != nil {
		return screens.Screen{}, err
	}
	return screens.Blog(l, posts, index, subscribed, h.Links), nil
}

func (h *Handlers) blogMenu(ctx *th.Context, msg telego.Message) error {
	l := h.lang(ctx, msg.From)
	s, err := h.blogScreen(ctx, l, msg.Chat.ID, 0)
	if err != nil {
		return h.failed(ctx, l, msg.Chat.ID, err)
	}
	return h.send(ctx, msg.Chat.ID, s)
}

func (h *Handlers) blogPress(ctx *th.Context, q telego.CallbackQuery) error {
	parts := strings.Split(strings.TrimPrefix(q.Data, keyboards.BlogPrefix), ":")
	chatID := chatOf(q)
	l := h.lang(ctx, &q.From)
	if len(parts) == 3 && parts[0] == screens.ActRead {
		page, _ := strconv.Atoi(parts[2])
		return h.readPost(ctx, l, q, parts[1], page)
	}
	toast := ""
	if len(parts) == 2 {
		on := parts[0] == screens.ActSubscribe
		if err := h.Subscribers.SetBlogSubscriber(ctx, chatID, on); err != nil {
			return h.failedPress(ctx, l, q, err)
		}
		toast = l.T("🔔 Пришлём новые статьи")
		if !on {
			toast = l.T("Больше не присылаем статьи")
		}
		parts = parts[1:]
	}
	index, _ := strconv.Atoi(parts[0])
	s, err := h.blogScreen(ctx, l, chatID, index)
	if err != nil {
		return h.failedPress(ctx, l, q, err)
	}
	if err := h.answer(ctx, q, toast); err != nil {
		return err
	}
	return h.show(ctx, q, s)
}

// readPost shows a page of a post (found among the published by id).
func (h *Handlers) readPost(ctx *th.Context, l i18n.Lang, q telego.CallbackQuery, id string, page int) error {
	posts, err := h.Content.BlogPosts(ctx, string(l))
	if err != nil {
		return h.failedPress(ctx, l, q, err)
	}
	index := slices.IndexFunc(posts, func(p api.BlogPost) bool { return p.ID == id })
	if index < 0 {
		return h.answer(ctx, q, l.T("Эта статья больше не опубликована"))
	}
	p := posts[index]
	full, err := h.Content.BlogPost(ctx, localeOf(p.Locale, l), p.Slug)
	if api.StatusOf(err) == http.StatusNotFound {
		return h.answer(ctx, q, l.T("Эта статья больше не опубликована"))
	}
	if err != nil {
		return h.failedPress(ctx, l, q, err)
	}
	if page == 0 {
		h.track(&q.From, l, eventViewArticle, map[string]any{"post_id": id})
	}
	if err := h.answer(ctx, q, ""); err != nil {
		return err
	}
	return h.show(ctx, q, screens.Article(l, full, index, page, h.Links))
}

// --- Notification settings ---

func (h *Handlers) notifySettings(ctx context.Context, userID string, chatID int64) (screens.NotifySettings, error) {
	var s screens.NotifySettings
	alerts, err := h.Content.TourAlerts(ctx, userID)
	if err != nil {
		return s, err
	}
	for _, d := range alerts {
		if d == "" {
			s.ToursAll = true
		} else {
			s.ToursDestinations++
		}
	}
	s.Blog, err = h.Subscribers.IsBlogSubscriber(ctx, chatID)
	return s, err
}

func (h *Handlers) notifyMenu(ctx *th.Context, msg telego.Message) error {
	if msg.From == nil {
		return nil
	}
	l := h.lang(ctx, msg.From)
	userID, err := h.userOf(ctx, msg.From)
	if err != nil {
		return err
	}
	s, err := h.notifySettings(ctx, userID, msg.Chat.ID)
	if err != nil {
		return h.failed(ctx, l, msg.Chat.ID, err)
	}
	return h.send(ctx, msg.Chat.ID, screens.Notifications(l, s))
}

func (h *Handlers) notifyPress(ctx *th.Context, q telego.CallbackQuery) error {
	chatID := chatOf(q)
	l := h.lang(ctx, &q.From)
	if q.Data == screens.NotifyBlogOff {
		if err := h.Subscribers.SetBlogSubscriber(ctx, chatID, false); err != nil {
			return h.failedPress(ctx, l, q, err)
		}
		return h.answer(ctx, q, l.T("Больше не присылаем статьи. Включить снова — в «🔔 Уведомления»."))
	}

	userID, err := h.userOf(ctx, &q.From)
	if err != nil {
		return err
	}
	s, err := h.notifySettings(ctx, userID, chatID)
	if err != nil {
		return h.failedPress(ctx, l, q, err)
	}
	var toast string
	switch q.Data {
	case screens.NotifyTours:
		err = h.Content.SetAllToursAlert(ctx, userID, !s.ToursAll)
		s.ToursAll = !s.ToursAll
		toast = map[bool]string{true: l.T("Новые туры: включено"), false: l.T("Новые туры: выключено")}[s.ToursAll]
	case screens.NotifyBlog:
		err = h.Subscribers.SetBlogSubscriber(ctx, chatID, !s.Blog)
		s.Blog = !s.Blog
		toast = map[bool]string{true: l.T("Статьи блога: включено"), false: l.T("Статьи блога: выключено")}[s.Blog]
	}
	if err != nil {
		return h.failedPress(ctx, l, q, err)
	}
	if err := h.answer(ctx, q, toast); err != nil {
		return err
	}
	return h.show(ctx, q, screens.Notifications(l, s))
}
