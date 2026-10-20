package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/elvn5/sai/services/bot/internal/api"
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
	UpcomingTours(ctx context.Context) ([]api.Tour, error)
	FavoriteTours(ctx context.Context, userID string) ([]api.Tour, error)
	SetFavoriteTour(ctx context.Context, userID, tourID string, saved bool) error
	TourAlerts(ctx context.Context, userID string) ([]string, error)
	SetAllToursAlert(ctx context.Context, userID string, on bool) error
	BlogPosts(ctx context.Context) ([]api.BlogPost, error)
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
func (h *Handlers) failed(ctx *th.Context, chatID int64, err error) error {
	h.Log.ErrorContext(ctx, "section failed", slog.Any("error", err))
	return h.reply(ctx, chatID, loadFailed, nil)
}

// failedPress logs an error and answers the button press with it.
func (h *Handlers) failedPress(ctx *th.Context, q telego.CallbackQuery, err error) error {
	h.Log.ErrorContext(ctx, "section failed", slog.Any("error", err))
	return h.answer(ctx, q, loadFailed)
}

func (h *Handlers) noop(ctx *th.Context, q telego.CallbackQuery) error { return h.answer(ctx, q, "") }

// --- Outings ---

func (h *Handlers) myOutings(ctx context.Context, userID, tab string) (screens.Screen, error) {
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
	return screens.MyOutings(list, sum, tab, h.now()), nil
}

func (h *Handlers) outingsMenu(ctx *th.Context, msg telego.Message) error {
	if msg.From == nil {
		return nil
	}
	userID, err := h.userOf(ctx, msg.From)
	if err != nil {
		return err
	}
	s, err := h.myOutings(ctx, userID, screens.FromMine)
	if err != nil {
		return h.failed(ctx, msg.Chat.ID, err)
	}
	return h.send(ctx, msg.Chat.ID, s)
}

// outingCard sends an outing's card (a deep link from the site).
func (h *Handlers) outingCard(ctx *th.Context, chatID int64, from *telego.User, id string) error {
	userID, err := h.userOf(ctx, from)
	if err != nil {
		return err
	}
	o, err := h.Content.Outing(ctx, userID, id)
	if api.StatusOf(err) == http.StatusNotFound || api.StatusOf(err) == http.StatusBadRequest {
		return h.reply(ctx, chatID, "Такого выхода нет — возможно, его удалили.", nil)
	}
	if err != nil {
		return h.failed(ctx, chatID, err)
	}
	return h.send(ctx, chatID, screens.OutingCard(o, screens.FromMine, h.Links, h.now()))
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
	userID, err := h.userOf(ctx, &q.From)
	if err != nil {
		return err
	}

	switch {
	case len(parts) == 2 && parts[0] == screens.ActList:
		s, err := h.myOutings(ctx, userID, parts[1])
		if err != nil {
			return h.failedPress(ctx, q, err)
		}
		if err := h.answer(ctx, q, ""); err != nil {
			return err
		}
		return h.show(ctx, q, s)

	case len(parts) == 2 && parts[0] == screens.ActFind:
		page, _ := strconv.Atoi(parts[1])
		list, err := h.Content.UpcomingOutings(ctx, userID)
		if err != nil {
			return h.failedPress(ctx, q, err)
		}
		h.markJoined(ctx, userID, list)
		if err := h.answer(ctx, q, ""); err != nil {
			return err
		}
		return h.show(ctx, q, screens.FindOutings(list, page, h.Links))

	case len(parts) == 3:
		return h.outingAction(ctx, q, userID, parts[0], parts[1], parts[2])
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

func (h *Handlers) outingAction(ctx *th.Context, q telego.CallbackQuery, userID, action, from, id string) error {
	toast := ""
	switch action {
	case screens.ActJoin:
		err := h.Content.JoinOuting(ctx, userID, id)
		if msg, known := joinErrors[api.MessageOf(err)]; known {
			toast = msg
		} else if err != nil && api.StatusOf(err) != http.StatusNotFound {
			return h.failedPress(ctx, q, err)
		} else if err == nil {
			toast = "✅ Вы записаны! Организатор получит уведомление."
		}
	case screens.ActLeave:
		err := h.Content.LeaveOuting(ctx, userID, id)
		if err != nil && api.StatusOf(err) != http.StatusNotFound {
			return h.failedPress(ctx, q, err)
		}
		if err == nil {
			toast = "Вы больше не идёте на этот выход"
		}
	}

	o, err := h.Content.Outing(ctx, userID, id)
	if api.StatusOf(err) == http.StatusNotFound || api.StatusOf(err) == http.StatusBadRequest {
		if err := h.answer(ctx, q, "Этого выхода больше нет"); err != nil {
			return err
		}
		s, err := h.myOutings(ctx, userID, screens.FromMine)
		if err != nil {
			return h.failed(ctx, chatOf(q), err)
		}
		return h.show(ctx, q, s)
	}
	if err != nil {
		return h.failedPress(ctx, q, err)
	}
	if err := h.answer(ctx, q, toast); err != nil {
		return err
	}
	if action == screens.ActAsk && o.IsJoined {
		return h.show(ctx, q, screens.LeaveConfirm(o, from))
	}
	return h.show(ctx, q, screens.OutingCard(o, from, h.Links, h.now()))
}

// --- Tours ---

func (h *Handlers) toursView(ctx context.Context, userID, tab string, index int) (screens.ToursView, error) {
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
	v.Tours, err = h.Content.UpcomingTours(ctx)
	return v, err
}

func (h *Handlers) toursMenu(ctx *th.Context, msg telego.Message) error {
	if msg.From == nil {
		return nil
	}
	userID, err := h.userOf(ctx, msg.From)
	if err != nil {
		return err
	}
	v, err := h.toursView(ctx, userID, "", 0)
	if err != nil {
		return h.failed(ctx, msg.Chat.ID, err)
	}
	return h.send(ctx, msg.Chat.ID, screens.Tours(v, h.Links))
}

func (h *Handlers) tourPress(ctx *th.Context, q telego.CallbackQuery) error {
	parts := strings.Split(strings.TrimPrefix(q.Data, keyboards.ToursPrefix), ":")
	userID, err := h.userOf(ctx, &q.From)
	if err != nil {
		return err
	}
	// "<tab>:<index>", or "<action>:<tab>:<index>[:<id>]".
	tab := parts[0]
	if len(parts) >= 3 {
		tab = parts[1]
	}
	if len(parts) < 2 || (tab != screens.TabFavorites && tab != screens.TabUpcoming) {
		return h.answer(ctx, q, "")
	}
	toast := ""
	if len(parts) >= 3 {
		switch action := parts[0]; action {
		case screens.ActSave, screens.ActDrop:
			if len(parts) != 4 {
				return h.answer(ctx, q, "")
			}
			saved := action == screens.ActSave
			if err := h.Content.SetFavoriteTour(ctx, userID, parts[3], saved); err != nil && api.StatusOf(err) != http.StatusNotFound {
				return h.failedPress(ctx, q, err)
			}
			toast = "❤️ Добавлено в избранное"
			if !saved {
				toast = "Убрано из избранного"
			}
		case screens.ActAlert:
			v, err := h.toursView(ctx, userID, parts[1], 0)
			if err != nil {
				return h.failedPress(ctx, q, err)
			}
			if err := h.Content.SetAllToursAlert(ctx, userID, !v.AlertOn); err != nil {
				return h.failedPress(ctx, q, err)
			}
			toast = "🔔 Сообщим о новых турах"
			if v.AlertOn {
				toast = "Больше не сообщаем о новых турах"
			}
		}
	}
	index, _ := strconv.Atoi(parts[len(parts)-1])
	if len(parts) >= 3 {
		index, _ = strconv.Atoi(parts[2])
	}
	v, err := h.toursView(ctx, userID, tab, index)
	if err != nil {
		return h.failedPress(ctx, q, err)
	}
	if err := h.answer(ctx, q, toast); err != nil {
		return err
	}
	return h.show(ctx, q, screens.Tours(v, h.Links))
}

// --- Blog ---

func (h *Handlers) blogScreen(ctx context.Context, chatID int64, index int) (screens.Screen, error) {
	posts, err := h.Content.BlogPosts(ctx)
	if err != nil {
		return screens.Screen{}, err
	}
	subscribed, err := h.Subscribers.IsBlogSubscriber(ctx, chatID)
	if err != nil {
		return screens.Screen{}, err
	}
	return screens.Blog(posts, index, subscribed, h.Links), nil
}

func (h *Handlers) blogMenu(ctx *th.Context, msg telego.Message) error {
	s, err := h.blogScreen(ctx, msg.Chat.ID, 0)
	if err != nil {
		return h.failed(ctx, msg.Chat.ID, err)
	}
	return h.send(ctx, msg.Chat.ID, s)
}

func (h *Handlers) blogPress(ctx *th.Context, q telego.CallbackQuery) error {
	parts := strings.Split(strings.TrimPrefix(q.Data, keyboards.BlogPrefix), ":")
	chatID := chatOf(q)
	toast := ""
	if len(parts) == 2 {
		on := parts[0] == screens.ActSubscribe
		if err := h.Subscribers.SetBlogSubscriber(ctx, chatID, on); err != nil {
			return h.failedPress(ctx, q, err)
		}
		toast = "🔔 Пришлём новые статьи"
		if !on {
			toast = "Больше не присылаем статьи"
		}
		parts = parts[1:]
	}
	index, _ := strconv.Atoi(parts[0])
	s, err := h.blogScreen(ctx, chatID, index)
	if err != nil {
		return h.failedPress(ctx, q, err)
	}
	if err := h.answer(ctx, q, toast); err != nil {
		return err
	}
	return h.show(ctx, q, s)
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
	userID, err := h.userOf(ctx, msg.From)
	if err != nil {
		return err
	}
	s, err := h.notifySettings(ctx, userID, msg.Chat.ID)
	if err != nil {
		return h.failed(ctx, msg.Chat.ID, err)
	}
	return h.send(ctx, msg.Chat.ID, screens.Notifications(s))
}

func (h *Handlers) notifyPress(ctx *th.Context, q telego.CallbackQuery) error {
	chatID := chatOf(q)
	if q.Data == screens.NotifyBlogOff {
		if err := h.Subscribers.SetBlogSubscriber(ctx, chatID, false); err != nil {
			return h.failedPress(ctx, q, err)
		}
		return h.answer(ctx, q, "Больше не присылаем статьи. Включить снова — в «🔔 Уведомления».")
	}

	userID, err := h.userOf(ctx, &q.From)
	if err != nil {
		return err
	}
	s, err := h.notifySettings(ctx, userID, chatID)
	if err != nil {
		return h.failedPress(ctx, q, err)
	}
	var toast string
	switch q.Data {
	case screens.NotifyTours:
		err = h.Content.SetAllToursAlert(ctx, userID, !s.ToursAll)
		s.ToursAll = !s.ToursAll
		toast = map[bool]string{true: "Новые туры: включено", false: "Новые туры: выключено"}[s.ToursAll]
	case screens.NotifyBlog:
		err = h.Subscribers.SetBlogSubscriber(ctx, chatID, !s.Blog)
		s.Blog = !s.Blog
		toast = map[bool]string{true: "Статьи блога: включено", false: "Статьи блога: выключено"}[s.Blog]
	}
	if err != nil {
		return h.failedPress(ctx, q, err)
	}
	if err := h.answer(ctx, q, toast); err != nil {
		return err
	}
	return h.show(ctx, q, screens.Notifications(s))
}
