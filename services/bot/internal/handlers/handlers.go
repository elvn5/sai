// Package handlers holds the bot's update handlers. Texts, buttons and
// callback data match the former NestJS bot one to one: the
// site links into the bot with t.me deep links.
package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/elvn5/sai/services/bot/internal/keyboards"
	"github.com/elvn5/sai/services/bot/internal/session"
	"github.com/elvn5/sai/services/bot/internal/users"
)

// Deep link payloads of /start.
const (
	verifyPrefix = "verify_"
	linkPrefix   = "link_"
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

type Handlers struct {
	Users      Accounts
	Sessions   *session.Store
	MiniAppURL string
	Log        *slog.Logger
}

// Register routes updates to the handlers. The first matching route wins,
// so free text (a value being edited) comes last.
func (h *Handlers) Register(bh *th.BotHandler) {
	bh.HandleMessage(h.start, th.CommandEqual("start"))
	bh.HandleMessage(h.profile, th.TextEqual(keyboards.Profile))
	bh.HandleMessage(h.support, th.TextEqual(keyboards.Support))
	bh.HandleCallbackQuery(h.linkMove, th.CallbackDataPrefix(keyboards.LinkMovePrefix))
	bh.HandleCallbackQuery(h.linkCancel, th.CallbackDataEqual(keyboards.LinkCancel))
	bh.HandleCallbackQuery(h.editField(session.FieldName, "Введите новое имя:"), th.CallbackDataEqual(keyboards.EditName))
	bh.HandleCallbackQuery(h.editField(session.FieldPhone, "Введите новый номер телефона:"), th.CallbackDataEqual(keyboards.EditPhone))
	bh.HandleCallbackQuery(h.editCancel, th.CallbackDataEqual(keyboards.EditCancel))
	bh.HandleMessage(h.text, th.AnyMessageWithText())
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

func (h *Handlers) mainMenu() telego.ReplyMarkup { return keyboards.MainMenu(h.MiniAppURL) }

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
	_, _, payload := tu.ParseCommandPayload(msg.Text)
	if token, ok := strings.CutPrefix(payload, verifyPrefix); ok {
		return h.verify(ctx, msg.Chat.ID, token)
	}
	// Before FindOrCreate: that would give this Telegram user an account of
	// its own, which is exactly what linking avoids.
	if token, ok := strings.CutPrefix(payload, linkPrefix); ok {
		return h.link(ctx, msg.Chat.ID, from, token, false)
	}

	if _, err := h.Users.FindOrCreate(ctx, profileOf(from)); err != nil {
		return err
	}
	name := from.FirstName
	if name == "" {
		name = "гость"
	}
	return h.reply(ctx, msg.Chat.ID, "Здравствуйте, "+name+
		"! Я помогу с гайдами и маршрутами по Кыргызстану и отвечу на вопросы. Выберите раздел в меню ниже.", h.mainMenu())
}

func (h *Handlers) verify(ctx *th.Context, chatID int64, token string) error {
	account, err := h.Users.ConfirmVerification(ctx, token)
	if err != nil {
		return err
	}
	if account == nil {
		return h.reply(ctx, chatID, "❌ Ссылка недействительна или истекла. Запросите новую ссылку на сайте Ermekten.", nil)
	}
	return h.reply(ctx, chatID, "✅ Аккаунт Ermekten подтверждён"+withName(*account)+
		"! Возвращайтесь на сайт — вход выполнится автоматически.", nil)
}

// link links this Telegram user to the site account behind the deep link.
func (h *Handlers) link(ctx *th.Context, chatID int64, from *telego.User, token string, move bool) error {
	res, err := h.Users.Link(ctx, token, from.ID, from.Username, move)
	if err != nil {
		return err
	}
	switch res.Status {
	case users.Linked:
		return h.reply(ctx, chatID, "✅ Telegram привязан к вашему аккаунту Ermekten"+withName(res.Account)+
			"! Теперь сюда будут приходить уведомления о выходах, а войти на сайт можно через Telegram.", h.mainMenu())
	case users.LinkInvalid:
		return h.reply(ctx, chatID,
			"❌ Ссылка недействительна или истекла. Нажмите «Привязать Telegram» в профиле на сайте ещё раз.", nil)
	case users.LinkTaken:
		return h.reply(ctx, chatID, "Этот Telegram уже привязан к другому аккаунту Ermekten (с почтой или телефоном). "+
			"Войдите на сайт через Telegram, чтобы пользоваться тем аккаунтом, или отвяжите Telegram в его профиле.", nil)
	default: // LinkNeedsMove
		return h.reply(ctx, chatID, "У этого Telegram уже есть свой аккаунт Ermekten, созданный через бота. "+
			"Привязать Telegram к аккаунту с сайта? Войти в старый Telegram-аккаунт после этого будет нельзя.",
			keyboards.LinkMove(token))
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
	return h.link(ctx, chatOf(q), &q.From, token, true)
}

func (h *Handlers) linkCancel(ctx *th.Context, q telego.CallbackQuery) error {
	if err := h.answer(ctx, q, "Отменено"); err != nil {
		return err
	}
	h.dropButtons(ctx, q)
	return h.reply(ctx, chatOf(q), "Хорошо, Telegram не привязан. Ссылка на сайте действует 15 минут.", nil)
}

func (h *Handlers) profile(ctx *th.Context, msg telego.Message) error {
	if msg.From == nil {
		h.Log.WarnContext(ctx, "message without a sender")
		return nil
	}
	user, err := h.Users.FindOrCreate(ctx, profileOf(msg.From))
	if err != nil {
		return err
	}
	firstName, username, phone := "не указано", "не указан", "не указан"
	if user.FirstName != nil {
		firstName = *user.FirstName
	}
	if user.Username != nil && *user.Username != "" {
		username = "@" + *user.Username
	}
	if user.Phone != nil {
		phone = *user.Phone
	}
	return h.reply(ctx, msg.Chat.ID, "Ваш профиль:\n\nИмя: "+firstName+"\nUsername: "+username+"\nТелефон: "+phone,
		keyboards.ProfileEdit())
}

func (h *Handlers) support(ctx *th.Context, msg telego.Message) error {
	s, err := h.Users.Support(ctx)
	if err != nil {
		return err
	}
	lines := []string{s.Message}
	for _, field := range []struct {
		label string
		value *string
	}{
		{"Телефон", s.Phone},
		{"Telegram", s.TelegramContact},
		{"Email", s.Email},
		{"Часы работы", s.WorkingHours},
	} {
		if field.value != nil && *field.value != "" {
			lines = append(lines, field.label+": "+*field.value)
		}
	}
	return h.reply(ctx, msg.Chat.ID, strings.Join(lines, "\n"), nil)
}

// editField starts editing a profile field: the next text is its value.
func (h *Handlers) editField(field session.Field, prompt string) th.CallbackQueryHandler {
	return func(ctx *th.Context, q telego.CallbackQuery) error {
		chatID := chatOf(q)
		h.Sessions.Start(chatID, field)
		if err := h.answer(ctx, q, ""); err != nil {
			return err
		}
		return h.reply(ctx, chatID, prompt, keyboards.CancelEdit())
	}
}

func (h *Handlers) editCancel(ctx *th.Context, q telego.CallbackQuery) error {
	chatID := chatOf(q)
	h.Sessions.Cancel(chatID)
	if err := h.answer(ctx, q, "Отменено"); err != nil {
		return err
	}
	return h.reply(ctx, chatID, "Редактирование отменено.", h.mainMenu())
}

// text saves the value of a profile field being edited; other text is
// ignored.
func (h *Handlers) text(ctx *th.Context, msg telego.Message) error {
	field, ok := h.Sessions.Consume(msg.Chat.ID)
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
		return h.reply(ctx, msg.Chat.ID, "Пустое значение не сохранено.", nil)
	}

	if field == session.FieldName {
		if err := h.Users.UpdateFirstName(ctx, user.ID, value); err != nil {
			return err
		}
		return h.reply(ctx, msg.Chat.ID, "Имя обновлено.", h.mainMenu())
	}
	err = h.Users.UpdatePhone(ctx, user.ID, value)
	if errors.Is(err, users.ErrPhoneTaken) {
		return h.reply(ctx, msg.Chat.ID, "Этот номер уже привязан к другому аккаунту.", h.mainMenu())
	}
	if err != nil {
		return err
	}
	return h.reply(ctx, msg.Chat.ID, "Телефон обновлен.", h.mainMenu())
}
