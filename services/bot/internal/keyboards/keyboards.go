// Package keyboards builds the bot's reply and inline keyboards. Button
// texts and callback data are part of the bot's behavior: keep them stable.
package keyboards

import (
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

// Main menu buttons (matched by text when pressed).
const (
	Profile = "👤 Профиль"
	Support = "🆘 Поддержка"
	MiniApp = "🏔 Открыть Ermekten"
)

// Callback data of inline buttons.
const (
	EditName       = "profile_edit_name"
	EditPhone      = "profile_edit_phone"
	EditCancel     = "profile_edit_cancel"
	LinkMovePrefix = "link_move:"
	LinkCancel     = "link_cancel"
)

// MainMenu is the persistent menu, with a Mini App button when its URL is set.
func MainMenu(miniAppURL string) *telego.ReplyKeyboardMarkup {
	rows := [][]telego.KeyboardButton{tu.KeyboardRow(tu.KeyboardButton(Profile), tu.KeyboardButton(Support))}
	if miniAppURL != "" {
		rows = append(rows, tu.KeyboardRow(tu.KeyboardButton(MiniApp).WithWebApp(tu.WebAppInfo(miniAppURL))))
	}
	return tu.Keyboard(rows...).WithResizeKeyboard()
}

func ProfileEdit() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(tu.InlineKeyboardButton("✏️ Изменить имя").WithCallbackData(EditName)),
		tu.InlineKeyboardRow(tu.InlineKeyboardButton("✏️ Изменить телефон").WithCallbackData(EditPhone)),
	)
}

func CancelEdit() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(tu.InlineKeyboardRow(tu.InlineKeyboardButton("Отмена").WithCallbackData(EditCancel)))
}

// LinkMove asks whether to move Telegram from its bot-made account to the
// site account behind token.
func LinkMove(token string) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(tu.InlineKeyboardRow(
		tu.InlineKeyboardButton("Привязать").WithCallbackData(LinkMovePrefix+token),
		tu.InlineKeyboardButton("Отмена").WithCallbackData(LinkCancel),
	))
}
