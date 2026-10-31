// Package handlers holds the bot's update handlers: account linking and
// the profile (texts, buttons and callback data as in the former NestJS
// bot: the site links into the bot with t.me deep links), the language
// picker, and the my tours, tours, blog and notification sections
// (sections.go). Every answer is in the chat's language (lang).
package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
	"github.com/elvn5/sai/services/bot/internal/screens"
	"github.com/elvn5/sai/services/bot/internal/session"
	"github.com/elvn5/sai/services/bot/internal/users"
)

// Deep link payloads of /start.
const (
	verifyPrefix = "verify_"
	linkPrefix   = "link_"
	tourPrefix   = "tour_"
)

// Accounts is the users service, as the handlers use it.
type Accounts interface {
	FindOrCreate(ctx context.Context, p users.Profile) (users.Account, error)
	ConfirmVerification(ctx context.Context, token string) (*users.Account, error)
	Link(ctx context.Context, token string, telegramID int64, username string, move bool) (users.LinkResult, error)
	UpdateFirstName(ctx context.Context, userID, firstName string) error
	UpdatePhone(ctx context.Context, userID, phone string) error
	Support(ctx context.Context) (users.Support, error)
}

// Languages keeps the language of each chat (store.Store).
type Languages interface {
	Language(ctx context.Context, chatID int64, fallback string) (string, error)
	KnownLanguage(ctx context.Context, chatID int64) (string, error)
	SetLanguage(ctx context.Context, chatID int64, lang string) error
}

type Handlers struct {
	Users      Accounts
	Sessions   *session.Store
	MiniAppURL string
	Log        *slog.Logger
	// The chats' languages; without it the bot speaks the language of
	// each user's Telegram app.
	Languages Languages

	// The my tours, tours, blog and notification sections; off unless
	// both are set.
	Content     Content
	Subscribers Subscribers
	// The organizer's room (arrivals, tours); off without the sections.
	Organizer Organizing
	Chats     Chats
	// Participants rating tours; off without the sections.
	Ratings Rating
	// The profile as on the site (form, password, achievements,
	// partnership); off without the sections.
	Profiles Profiles
	Links    screens.Links
	// Now is overridable in tests.
	Now func() time.Time
	// Where events go (analytics.Client); nil: not reported.
	Analytics Tracker
}

// Register routes updates to the handlers. The first matching route wins,
// so free text (a value being edited) comes last.
func (h *Handlers) Register(bh *th.BotHandler) {
	bh.HandleMessage(h.start, th.CommandEqual("start"))
	profile := h.profile
	if h.profiles() {
		profile = h.profileSection
		bh.HandleCallbackQuery(h.profilePress, th.CallbackDataPrefix(keyboards.ProfilePrefix))
	}
	bh.HandleMessage(h.menu("profile", profile), menuButton(keyboards.Profile, "profile"))
	bh.HandleMessage(h.menu("support", h.support), menuButton(keyboards.Support, "support"))
	bh.HandleMessage(h.menu("language", h.languageMenu), th.CommandEqual("language"))
	bh.HandleCallbackQuery(h.languagePress, th.CallbackDataPrefix(keyboards.LanguagePrefix))
	if h.sections() {
		bh.HandleMessage(h.menu("my_tours", h.myToursMenu), menuButton(keyboards.MyTours, "mytours"))
		bh.HandleMessage(h.menu("tours", h.toursMenu), menuButton(keyboards.Tours, "tours"))
		bh.HandleMessage(h.menu("blog", h.blogMenu), menuButton(keyboards.Blog, "blog"))
		bh.HandleMessage(h.menu("notifications", h.notifyMenu), menuButton(keyboards.Notifications, "notifications"))
		bh.HandleCallbackQuery(h.myToursPress, th.CallbackDataPrefix(keyboards.MyToursPrefix))
		bh.HandleCallbackQuery(h.tourPress, th.CallbackDataPrefix(keyboards.ToursPrefix))
		bh.HandleCallbackQuery(h.blogPress, th.CallbackDataPrefix(keyboards.BlogPrefix))
		bh.HandleCallbackQuery(h.notifyPress, th.CallbackDataPrefix(keyboards.NotifyPrefix))
		bh.HandleCallbackQuery(h.stalePress, th.CallbackDataPrefix(keyboards.DraftPrefix))
		bh.HandleCallbackQuery(h.stalePress, th.CallbackDataPrefix(keyboards.ConditionPrefix))
	}
	if h.rating() {
		bh.HandleCallbackQuery(h.ratingPress, th.CallbackDataPrefix(keyboards.RatingPrefix))
	}
	if h.organizing() {
		bh.HandleMessage(h.menu("organizer", h.organizerMenu), th.CommandEqual("organizer"))
		bh.HandleCallbackQuery(h.organizerPress, th.CallbackDataPrefix(keyboards.OrganizerPrefix))
	}
	bh.HandleCallbackQuery(h.noop, th.CallbackDataEqual(keyboards.Noop))
	bh.HandleCallbackQuery(h.linkMove, th.CallbackDataPrefix(keyboards.LinkMovePrefix))
	bh.HandleCallbackQuery(h.linkCancel, th.CallbackDataEqual(keyboards.LinkCancel))
	bh.HandleCallbackQuery(h.editField(session.FieldName, i18n.M("Введите новое имя:")), th.CallbackDataEqual(keyboards.EditName))
	bh.HandleCallbackQuery(h.editField(session.FieldPhone, i18n.M("Введите новый номер телефона:")), th.CallbackDataEqual(keyboards.EditPhone))
	bh.HandleCallbackQuery(h.editCancel, th.CallbackDataEqual(keyboards.EditCancel))
	bh.HandleMessage(h.text, th.AnyMessageWithText())
}

// menuButton matches a main menu button, in any language, or its command.
func menuButton(button, command string) th.Predicate {
	texts := keyboards.Texts(button)
	return th.Or(th.CommandEqual(command), func(_ context.Context, update telego.Update) bool {
		return update.Message != nil && slices.Contains(texts, update.Message.Text)
	})
}

// lang is the language to answer a user in: the one kept for their chat,
// else their Telegram app's (kept from then on).
func (h *Handlers) lang(ctx context.Context, from *telego.User) i18n.Lang {
	if from == nil {
		return i18n.RU
	}
	detected := i18n.Parse(from.LanguageCode)
	if h.Languages == nil {
		return detected
	}
	kept, err := h.Languages.Language(ctx, from.ID, string(detected))
	if err != nil {
		h.Log.WarnContext(ctx, "chat language", slog.Any("error", err))
		return detected
	}
	if l := i18n.Lang(kept); l.Valid() {
		return l
	}
	return detected
}

// chatLang is the language of someone else's chat (a participant getting
// the organizer's message): Russian when the bot doesn't know it yet.
func (h *Handlers) chatLang(ctx context.Context, chatID int64) i18n.Lang {
	if h.Languages == nil {
		return i18n.RU
	}
	kept, err := h.Languages.KnownLanguage(ctx, chatID)
	if err != nil {
		h.Log.WarnContext(ctx, "chat language", slog.Any("error", err))
	}
	if l := i18n.Lang(kept); l.Valid() {
		return l
	}
	return i18n.RU
}

func (h *Handlers) reply(ctx *th.Context, chatID int64, text string, markup telego.ReplyMarkup) error {
	msg := tu.Message(tu.ID(chatID), text)
	if markup != nil {
		msg = msg.WithReplyMarkup(markup)
	}
	if _, err := ctx.Bot().SendMessage(ctx, msg); err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
}

func (h *Handlers) mainMenu(l i18n.Lang) telego.ReplyMarkup {
	return keyboards.MainMenu(l, h.MiniAppURL, h.sections())
}

func profileOf(u *telego.User) users.Profile {
	return users.Profile{TelegramID: u.ID, Username: u.Username, FirstName: u.FirstName, LastName: u.LastName}
}

// withName appends ", <name>" when the account has one.
func withName(a users.Account) string {
	if a.FirstName != nil && *a.FirstName != "" {
		return ", " + *a.FirstName
	}
	return ""
}

func (h *Handlers) start(ctx *th.Context, msg telego.Message) error {
	from := msg.From
	if from == nil {
		return nil
	}
	l := h.lang(ctx, from)
	_, _, payload := tu.ParseCommandPayload(msg.Text)
	h.track(from, l, eventStart, startParams(payload))
	if token, ok := strings.CutPrefix(payload, verifyPrefix); ok {
		return h.verify(ctx, l, msg.Chat.ID, token)
	}
	// Before FindOrCreate: that would give this Telegram user an account of
	// its own, which is exactly what linking avoids.
	if token, ok := strings.CutPrefix(payload, linkPrefix); ok {
		return h.link(ctx, l, msg.Chat.ID, from, token, false)
	}

	account, err := h.Users.FindOrCreate(ctx, profileOf(from))
	if err != nil {
		return err
	}
	if h.sections() {
		if id, ok := strings.CutPrefix(payload, tourPrefix); ok && id != "" {
			return h.tourCard(ctx, l, msg.Chat.ID, from, id)
		}
	}
	name := from.FirstName
	if name == "" {
		name = l.T("гость")
	}
	if !h.sections() {
		return h.reply(ctx, msg.Chat.ID, l.F("Здравствуйте, %s! Я помогу с гайдами и маршрутами по Кыргызстану и отвечу на вопросы. Выберите раздел в меню ниже.", name),
			h.mainMenu(l))
	}
	greeting := tu.Message(tu.ID(msg.Chat.ID), h.welcome(ctx, l, account.ID, name)).
		WithParseMode(telego.ModeHTML).WithReplyMarkup(h.mainMenu(l))
	if _, err := ctx.Bot().SendMessage(ctx, greeting); err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
}

// welcome greets the user with the main menu and their nearest tour.
func (h *Handlers) welcome(ctx context.Context, l i18n.Lang, userID, name string) string {
	var next *api.Tour
	if list, err := h.Content.JoinedTours(ctx, userID); err == nil {
		if upcoming, _ := screens.SplitTours(list, h.now()); len(upcoming) > 0 {
			next = &upcoming[0]
		}
	} else {
		h.Log.WarnContext(ctx, "nearest tour", slog.Any("error", err))
	}
	return screens.Welcome(l, name, next)
}

func (h *Handlers) verify(ctx *th.Context, l i18n.Lang, chatID int64, token string) error {
	account, err := h.Users.ConfirmVerification(ctx, token)
	if err != nil {
		return err
	}
	if account == nil {
		return h.reply(ctx, chatID, l.T("❌ Ссылка недействительна или истекла. Запросите новую ссылку на сайте Ermekten."), nil)
	}
	return h.reply(ctx, chatID, l.F("✅ Аккаунт Ermekten подтверждён%s! Возвращайтесь на сайт — вход выполнится автоматически.",
		withName(*account)), nil)
}

// link links this Telegram user to the site account behind the deep link.
func (h *Handlers) link(ctx *th.Context, l i18n.Lang, chatID int64, from *telego.User, token string, move bool) error {
	res, err := h.Users.Link(ctx, token, from.ID, from.Username, move)
	if err != nil {
		return err
	}
	switch res.Status {
	case users.Linked:
		return h.reply(ctx, chatID, l.F("✅ Telegram привязан к вашему аккаунту Ermekten%s! Теперь сюда будут приходить уведомления о турах, а войти на сайт можно через Telegram.",
			withName(res.Account)), h.mainMenu(l))
	case users.LinkInvalid:
		return h.reply(ctx, chatID,
			l.T("❌ Ссылка недействительна или истекла. Нажмите «Привязать Telegram» в профиле на сайте ещё раз."), nil)
	case users.LinkTaken:
		return h.reply(ctx, chatID, l.T("Этот Telegram уже привязан к другому аккаунту Ermekten (с почтой или телефоном). "+
			"Войдите на сайт через Telegram, чтобы пользоваться тем аккаунтом, или отвяжите Telegram в его профиле."), nil)
	default: // LinkNeedsMove
		return h.reply(ctx, chatID, l.T("У этого Telegram уже есть свой аккаунт Ermekten, созданный через бота. "+
			"Привязать Telegram к аккаунту с сайта? Войти в старый Telegram-аккаунт после этого будет нельзя."),
			keyboards.LinkMove(l, token))
	}
}

// chatOf is where to answer a button press: the chat of the message with
// the button (the user's private chat if Telegram no longer has it).
func chatOf(q telego.CallbackQuery) int64 {
	if q.Message != nil {
		return q.Message.GetChat().ID
	}
	return q.From.ID
}

// answer acknowledges a button press, with an optional toast.
func (h *Handlers) answer(ctx *th.Context, q telego.CallbackQuery, text string) error {
	params := tu.CallbackQuery(q.ID)
	if text != "" {
		params = params.WithText(text)
	}
	if err := ctx.Bot().AnswerCallbackQuery(ctx, params); err != nil {
		return fmt.Errorf("answer callback query: %w", err)
	}
	return nil
}

// dropButtons removes the inline keyboard from the pressed message, best
// effort (it may be too old to edit).
func (h *Handlers) dropButtons(ctx *th.Context, q telego.CallbackQuery) {
	if q.Message == nil {
		return
	}
	_, err := ctx.Bot().EditMessageReplyMarkup(ctx, &telego.EditMessageReplyMarkupParams{
		ChatID:    tu.ID(q.Message.GetChat().ID),
		MessageID: q.Message.GetMessageID(),
	})
	if err != nil {
		h.Log.DebugContext(ctx, "could not remove buttons", slog.Any("error", err))
	}
}

func (h *Handlers) linkMove(ctx *th.Context, q telego.CallbackQuery) error {
	token := strings.TrimPrefix(q.Data, keyboards.LinkMovePrefix)
	if err := h.answer(ctx, q, ""); err != nil {
		return err
	}
	h.dropButtons(ctx, q)
	return h.link(ctx, h.lang(ctx, &q.From), chatOf(q), &q.From, token, true)
}

func (h *Handlers) linkCancel(ctx *th.Context, q telego.CallbackQuery) error {
	l := h.lang(ctx, &q.From)
	if err := h.answer(ctx, q, l.T("Отменено")); err != nil {
		return err
	}
	h.dropButtons(ctx, q)
	return h.reply(ctx, chatOf(q), l.T("Хорошо, Telegram не привязан. Ссылка на сайте действует 15 минут."), nil)
}

func (h *Handlers) profile(ctx *th.Context, msg telego.Message) error {
	if msg.From == nil {
		h.Log.WarnContext(ctx, "message without a sender")
		return nil
	}
	l := h.lang(ctx, msg.From)
	user, err := h.Users.FindOrCreate(ctx, profileOf(msg.From))
	if err != nil {
		return err
	}
	firstName, username, phone := l.T("не указано"), l.T("не указан"), l.T("не указан")
	if user.FirstName != nil {
		firstName = *user.FirstName
	}
	if user.Username != nil && *user.Username != "" {
		username = "@" + *user.Username
	}
	if user.Phone != nil {
		phone = *user.Phone
	}
	return h.reply(ctx, msg.Chat.ID, l.F("Ваш профиль:\n\nИмя: %s\nUsername: %s\nТелефон: %s", firstName, username, phone),
		keyboards.ProfileEdit(l))
}

func (h *Handlers) support(ctx *th.Context, msg telego.Message) error {
	l := h.lang(ctx, msg.From)
	s, err := h.Users.Support(ctx)
	if err != nil {
		return err
	}
	lines := []string{s.Message}
	for _, field := range []struct {
		label string
		value *string
	}{
		{l.T("Телефон"), s.Phone},
		{"Telegram", s.TelegramContact},
		{"Email", s.Email},
		{l.T("Часы работы"), s.WorkingHours},
	} {
		if field.value != nil && *field.value != "" {
			lines = append(lines, field.label+": "+*field.value)
		}
	}
	return h.reply(ctx, msg.Chat.ID, strings.Join(lines, "\n"), nil)
}

// editField starts editing a profile field: the next text is its value.
func (h *Handlers) editField(field session.Field, prompt i18n.Msg) th.CallbackQueryHandler {
	return func(ctx *th.Context, q telego.CallbackQuery) error {
		chatID := chatOf(q)
		l := h.lang(ctx, &q.From)
		h.Sessions.Start(chatID, field)
		if err := h.answer(ctx, q, ""); err != nil {
			return err
		}
		return h.reply(ctx, chatID, l.Say(prompt), keyboards.CancelEdit(l))
	}
}

func (h *Handlers) editCancel(ctx *th.Context, q telego.CallbackQuery) error {
	chatID := chatOf(q)
	l := h.lang(ctx, &q.From)
	h.Sessions.Cancel(chatID)
	if err := h.answer(ctx, q, l.T("Отменено")); err != nil {
		return err
	}
	return h.reply(ctx, chatID, l.T("Редактирование отменено."), h.mainMenu(l))
}

// text saves the value of a profile field being edited, takes an
// organizer's message to the participants, a participant's review, an
// answer of the participant's form or a password; other text is ignored.
func (h *Handlers) text(ctx *th.Context, msg telego.Message) error {
	l := h.lang(ctx, msg.From)
	field, arg, ok := h.Sessions.ConsumeWith(msg.Chat.ID)
	if ok && h.organizing() && (field == session.FieldBroadcast || field == session.FieldBroadcastReady) {
		tourID, _, _ := strings.Cut(arg, "\n")
		return h.broadcastText(ctx, l, msg, tourID)
	}
	if ok && h.rating() && field == session.FieldReview {
		return h.reviewText(ctx, l, msg, arg)
	}
	if ok && h.profiles() && field == session.FieldForm {
		return h.formText(ctx, l, msg, arg)
	}
	if ok && h.profiles() && field == session.FieldPassword {
		return h.passwordText(ctx, l, msg, arg)
	}
	if ok && field != session.FieldName && field != session.FieldPhone {
		ok = false
	}
	if !ok {
		return nil
	}
	if msg.From == nil {
		h.Log.WarnContext(ctx, "message without a sender")
		return nil
	}
	user, err := h.Users.FindOrCreate(ctx, profileOf(msg.From))
	if err != nil {
		return err
	}
	value := strings.TrimSpace(msg.Text)
	if value == "" {
		return h.reply(ctx, msg.Chat.ID, l.T("Пустое значение не сохранено."), nil)
	}

	if field == session.FieldName {
		if err := h.Users.UpdateFirstName(ctx, user.ID, value); err != nil {
			return err
		}
		return h.reply(ctx, msg.Chat.ID, l.T("Имя обновлено."), h.mainMenu(l))
	}
	err = h.Users.UpdatePhone(ctx, user.ID, value)
	if errors.Is(err, users.ErrPhoneTaken) {
		return h.reply(ctx, msg.Chat.ID, l.T("Этот номер уже привязан к другому аккаунту."), h.mainMenu(l))
	}
	if err != nil {
		return err
	}
	return h.reply(ctx, msg.Chat.ID, l.T("Телефон обновлен."), h.mainMenu(l))
}
