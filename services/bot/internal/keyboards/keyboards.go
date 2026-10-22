// Package keyboards builds the bot's reply and inline keyboards. Button
// texts and callback data are part of the bot's behavior: keep them stable.
package keyboards

import (
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/elvn5/sai/services/bot/internal/i18n"
)

// Main menu buttons, in Russian: they show in the user's language and are
// matched by text in any language when pressed (see Texts).
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
	// The language picker: "l:menu" shows it, "l:<lang>" picks.
	LanguagePrefix = "l:"
	LanguageMenu   = LanguagePrefix + "menu"

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
	// An outing being put together: "d:<action>[:<arg>]".
	DraftPrefix = "d:"
	// A trail condition report: "c:<step>:<outing id>[:<answers>]".
	ConditionPrefix = "c:"
	// The organizer's room: "g:<action>[:<arg>…]".
	OrganizerPrefix = "g:"
	// Rating an outing after it: "r:<action>:<outing id>[:<stars>]".
	RatingPrefix = "r:"
)

// MainMenu is the persistent menu, with a Mini App button when its URL is
// set; sections reaches the outings, tours and blog buttons (they need the
// API).
func MainMenu(l i18n.Lang, miniAppURL string, sections bool) *telego.ReplyKeyboardMarkup {
	button := func(text string) telego.KeyboardButton { return tu.KeyboardButton(l.T(text)) }
	var rows [][]telego.KeyboardButton
	if sections {
		rows = append(rows,
			tu.KeyboardRow(button(Outings), button(Tours)),
			tu.KeyboardRow(button(Blog), button(Notifications)),
		)
	}
	rows = append(rows, tu.KeyboardRow(button(Profile), button(Support)))
	if miniAppURL != "" {
		rows = append(rows, tu.KeyboardRow(button(MiniApp).WithWebApp(tu.WebAppInfo(SiteURL(miniAppURL, l)))))
	}
	return tu.Keyboard(rows...).WithResizeKeyboard().WithInputFieldPlaceholder(l.T("Выберите раздел"))
}

// Texts are a main menu button in every language.
func Texts(button string) []string {
	out := make([]string, 0, len(i18n.Langs))
	for _, l := range i18n.Langs {
		out = append(out, l.T(button))
	}
	return out
}

// SiteURL is the site (the Mini App) in a language: Russian unprefixed,
// the others under /ky, /en, /de, as on the site.
func SiteURL(base string, l i18n.Lang) string {
	if l == i18n.RU || base == "" {
		return base
	}
	return strings.TrimRight(base, "/") + "/" + string(l)
}

func ProfileEdit(l i18n.Lang) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(tu.InlineKeyboardButton(l.T("✏️ Изменить имя")).WithCallbackData(EditName)),
		tu.InlineKeyboardRow(tu.InlineKeyboardButton(l.T("✏️ Изменить телефон")).WithCallbackData(EditPhone)),
		tu.InlineKeyboardRow(tu.InlineKeyboardButton("🌐 "+l.T("Язык")+" · "+l.Name()).WithCallbackData(LanguageMenu)),
	)
}

func CancelEdit(l i18n.Lang) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(tu.InlineKeyboardRow(tu.InlineKeyboardButton(l.T("Отмена")).WithCallbackData(EditCancel)))
}

// LinkMove asks whether to move Telegram from its bot-made account to the
// site account behind token.
func LinkMove(l i18n.Lang, token string) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(tu.InlineKeyboardRow(
		tu.InlineKeyboardButton(l.T("Привязать")).WithCallbackData(LinkMovePrefix+token),
		tu.InlineKeyboardButton(l.T("Отмена")).WithCallbackData(LinkCancel),
	))
}

// Languages offers the bot's languages, the current one checked.
func Languages(current i18n.Lang) *telego.InlineKeyboardMarkup {
	var rows [][]telego.InlineKeyboardButton
	for i := 0; i < len(i18n.Langs); i += 2 {
		var row []telego.InlineKeyboardButton
		for _, l := range i18n.Langs[i:min(i+2, len(i18n.Langs))] {
			label := l.Name()
			if l == current {
				label = "✅ " + label
			}
			row = append(row, Button(label, LanguagePrefix+string(l)))
		}
		rows = append(rows, row)
	}
	return tu.InlineKeyboard(rows...)
}

// Button is an inline button pressing back to the bot.
func Button(text, data string) telego.InlineKeyboardButton {
	return tu.InlineKeyboardButton(text).WithCallbackData(data)
}

// Link is an inline button opening a URL.
func Link(text, url string) telego.InlineKeyboardButton {
	return tu.InlineKeyboardButton(text).WithURL(url)
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
