package handlers

import (
	"context"
	"errors"
	"slices"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
	"github.com/elvn5/sai/services/bot/internal/users"
)

// phonePayload is the /start deep link the site's «Добавить телефон»
// opens.
const phonePayload = "phone"

// The phone number is confirmed by Telegram: the user shares their own
// contact, never types it. A number that belongs to another account (say,
// one registered on the site) means it's the same person: the users
// service merges this Telegram account into it.

// askPhone (the profile's phone button) offers to share the number.
func (h *Handlers) askPhone(ctx *th.Context, q telego.CallbackQuery) error {
	l := h.lang(ctx, &q.From)
	if err := h.answer(ctx, q, ""); err != nil {
		return err
	}
	return h.sharePhone(ctx, l, chatOf(q))
}

func (h *Handlers) sharePhone(ctx *th.Context, l i18n.Lang, chatID int64) error {
	return h.reply(ctx, chatID, l.T("Нажмите «📱 Поделиться номером» внизу: Telegram отправит ваш номер, и он станет подтверждённым. "+
		"Если с этим номером уже есть аккаунт Ermekten (например, с сайта), мы объединим его с вашим Telegram — туры, избранное и достижения будут в одном аккаунте."),
		keyboards.ShareContact(l))
}

// contact takes the number the user shared.
func (h *Handlers) contact(ctx *th.Context, msg telego.Message) error {
	l := h.lang(ctx, msg.From)
	c := msg.Contact
	if msg.From == nil || c == nil {
		return nil
	}
	if c.UserID != msg.From.ID {
		return h.reply(ctx, msg.Chat.ID, l.T("Это не ваш номер. Нажмите «📱 Поделиться номером», чтобы отправить свой."), keyboards.ShareContact(l))
	}
	// The account exists before its number is confirmed.
	if _, err := h.Users.FindOrCreate(ctx, profileOf(msg.From)); err != nil {
		return err
	}
	account, merged, err := h.Users.ConfirmPhone(ctx, msg.From.ID, c.PhoneNumber)
	if errors.Is(err, users.ErrPhoneTaken) {
		return h.reply(ctx, msg.Chat.ID, l.T("Этот номер уже есть в другом аккаунте Ermekten, и объединить их сами мы не можем. Напишите в поддержку — поможем."), h.mainMenu(l))
	}
	if err != nil {
		return err
	}
	if merged {
		return h.reply(ctx, msg.Chat.ID, l.F("✅ Номер подтверждён. С ним уже был аккаунт Ermekten%s — теперь это один аккаунт: туры, избранное и достижения вместе, войти можно и через Telegram, и по номеру.",
			withName(account)), h.mainMenu(l))
	}
	return h.reply(ctx, msg.Chat.ID, l.T("✅ Номер подтверждён."), h.mainMenu(l))
}

// shareCancel is «Не сейчас» under the share button.
func (h *Handlers) shareCancel(ctx *th.Context, msg telego.Message) error {
	l := h.lang(ctx, msg.From)
	return h.reply(ctx, msg.Chat.ID, l.T("Хорошо. Добавить номер можно в любой момент в профиле."), h.mainMenu(l))
}

func isContact(_ context.Context, update telego.Update) bool {
	return update.Message != nil && update.Message.Contact != nil
}

func isShareCancel(_ context.Context, update telego.Update) bool {
	return update.Message != nil && slices.Contains(keyboards.Texts(keyboards.ShareCancel), update.Message.Text)
}
