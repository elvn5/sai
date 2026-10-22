package handlers

import (
	"log/slog"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
)

// languagePrompt asks for a language, in all of them: whoever opens the
// picker may not read the current one.
const languagePrompt = "🌐 Выберите язык · Тилди тандаңыз · Choose a language · Sprache wählen"

func (h *Handlers) languageMenu(ctx *th.Context, msg telego.Message) error {
	return h.reply(ctx, msg.Chat.ID, languagePrompt, keyboards.Languages(h.lang(ctx, msg.From)))
}

// languagePress shows the picker ("l:menu") or switches the chat to a
// language ("l:<lang>"): the picker says so in it, and the main menu comes
// again with the buttons in that language.
func (h *Handlers) languagePress(ctx *th.Context, q telego.CallbackQuery) error {
	if q.Data == keyboards.LanguageMenu {
		if err := h.answer(ctx, q, ""); err != nil {
			return err
		}
		return h.reply(ctx, chatOf(q), languagePrompt, keyboards.Languages(h.lang(ctx, &q.From)))
	}
	l := i18n.Lang(strings.TrimPrefix(q.Data, keyboards.LanguagePrefix))
	if !l.Valid() {
		return h.answer(ctx, q, "")
	}
	if h.Languages != nil {
		if err := h.Languages.SetLanguage(ctx, q.From.ID, string(l)); err != nil {
			return h.failedPress(ctx, h.lang(ctx, &q.From), q, err)
		}
	}
	if err := h.answer(ctx, q, l.Name()); err != nil {
		return err
	}
	h.dropButtons(ctx, q)
	h.localMiniApp(ctx, chatOf(q), l)
	return h.reply(ctx, chatOf(q), l.T("✅ Теперь я говорю по-русски. Сменить язык — /language или в «👤 Профиль»."), h.mainMenu(l))
}

// localMiniApp points the chat's menu button at the site in the chat's
// language, best effort.
func (h *Handlers) localMiniApp(ctx *th.Context, chatID int64, l i18n.Lang) {
	if h.MiniAppURL == "" {
		return
	}
	err := ctx.Bot().SetChatMenuButton(ctx, &telego.SetChatMenuButtonParams{
		ChatID: chatID,
		MenuButton: &telego.MenuButtonWebApp{
			Type: telego.ButtonTypeWebApp, Text: "Ermekten",
			WebApp: telego.WebAppInfo{URL: keyboards.SiteURL(h.MiniAppURL, l)},
		},
	})
	if err != nil {
		h.Log.DebugContext(ctx, "could not set the chat's menu button", slog.Any("error", err))
	}
}
