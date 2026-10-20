// Package keyboards builds the bot's reply and inline keyboards. Button
// texts and callback data are part of the bot's behavior: keep them stable.
package keyboards

import (
	"strconv"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

// Main menu buttons (matched by text when pressed).
const (
	Outings       = "🥾 Мои выходы"
	Tours         = "🧭 Туры"
	Blog          = "📰 Блог"
	Notifications = "🔔 Уведомления"
	Profile       = "👤 Профиль"
	Support       = "🆘 Поддержка"
	MiniApp       = "🏔 Открыть Ermekten"
)

// Callback data of inline buttons.
const (
	EditName       = "profile_edit_name"
	EditPhone      = "profile_edit_phone"
	EditCancel     = "profile_edit_cancel"
	LinkMovePrefix = "link_move:"
	LinkCancel     = "link_cancel"

	// Buttons that only show something (a page counter).
	Noop = "noop"

	// Outings: "o:<action>:<arg>…", see the handlers.
	OutingsPrefix = "o:"
	// Tours: "t:<tab>:<index>" and "t:fav:<tab>:<index>:<tour id>".
	ToursPrefix = "t:"
	// Blog: "b:<index>".
	BlogPrefix = "b:"
	// Notification settings: "n:tours", "n:blog", "n:blog_off" (from a post).
	NotifyPrefix = "n:"
)

// MainMenu is the persistent menu, with a Mini App button when its URL is
// set; sections reaches the outings, tours and blog buttons (they need the
// API).
func MainMenu(miniAppURL string, sections bool) *telego.ReplyKeyboardMarkup {
	var rows [][]telego.KeyboardButton
	if sections {
		rows = append(rows,
			tu.KeyboardRow(tu.KeyboardButton(Outings), tu.KeyboardButton(Tours)),
			tu.KeyboardRow(tu.KeyboardButton(Blog), tu.KeyboardButton(Notifications)),
		)
	}
	rows = append(rows, tu.KeyboardRow(tu.KeyboardButton(Profile), tu.KeyboardButton(Support)))
	if miniAppURL != "" {
		rows = append(rows, tu.KeyboardRow(tu.KeyboardButton(MiniApp).WithWebApp(tu.WebAppInfo(miniAppURL))))
	}
	return tu.Keyboard(rows...).WithResizeKeyboard().WithInputFieldPlaceholder("Выберите раздел")
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

// Button is an inline button pressing back to the bot.
func Button(text, data string) telego.InlineKeyboardButton {
	return tu.InlineKeyboardButton(text).WithCallbackData(data)
}

// Link is an inline button opening a URL.
func Link(text, url string) telego.InlineKeyboardButton {
	return tu.InlineKeyboardButton(text).WithURL(url)
}

// WebApp is an inline button opening a page of the Mini App inside
// Telegram (private chats only).
func WebApp(text, url string) telego.InlineKeyboardButton {
	return tu.InlineKeyboardButton(text).WithWebApp(tu.WebAppInfo(url))
}

// Pager is "◀️ i / n ▶️" for page index of n, wrapping around; data makes
// the callback data of a page.
func Pager(index, n int, data func(page int) string) []telego.InlineKeyboardButton {
	return tu.InlineKeyboardRow(
		Button("◀️", data((index-1+n)%n)),
		Button(strconv.Itoa(index+1)+" / "+strconv.Itoa(n), Noop),
		Button("▶️", data((index+1)%n)),
	)
}

// Tabs is a row of switches with the current one marked.
func Tabs(current string, tabs ...[2]string) []telego.InlineKeyboardButton {
	row := make([]telego.InlineKeyboardButton, 0, len(tabs))
	for _, tab := range tabs {
		label, data := tab[0], tab[1]
		if data == current {
			label = "• " + label + " •"
		}
		row = append(row, Button(label, data))
	}
	return row
}
