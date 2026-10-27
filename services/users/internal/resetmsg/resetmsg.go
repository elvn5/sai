// Package resetmsg writes the password reset message, for email and for
// the Telegram chat, in the recipient's language. Texts are written in
// Russian and translated with the catalogs in locales/ (keyed by the
// Russian text), as in the bot.
package resetmsg

import (
	"embed"
	"html"
	"strings"

	"github.com/elvn5/sai/services/pkg/telegram"
)

//go:embed locales/*.json
var locales embed.FS

var texts = telegram.MustLoadCatalog(locales, "locales")

// Texts lists every text, for the catalog test.
var Texts = []string{
	subject, title, intro, emailIntro, button, validity, ignore,
}

const (
	subject    = "Сброс пароля Ermekten"
	title      = "Сброс пароля"
	intro      = "Кто-то запросил новый пароль для вашего аккаунта Ermekten. Если это вы, нажмите кнопку ниже."
	emailIntro = "Чтобы задать новый пароль для аккаунта Ermekten, откройте ссылку:"
	button     = "Задать новый пароль"
	// Matches handlers.PasswordResetTTL.
	validity = "Ссылка действует 30 минут и сработает один раз."
	ignore   = "Если вы не запрашивали сброс, просто проигнорируйте это сообщение — пароль останется прежним."
)

// Link is the site's page for setting the new password, in l's language
// (Russian unprefixed). The token goes in the fragment, which browsers
// never send to the server or in a Referer.
func Link(siteURL string, l telegram.Lang, token string) string {
	prefix := ""
	if l != telegram.RU {
		prefix = "/" + string(l)
	}
	return strings.TrimRight(siteURL, "/") + prefix + "/reset-password#" + token
}

// Email is the subject and HTML body of the reset email.
func Email(l telegram.Lang, link string) (subjectText, body string) {
	href := html.EscapeString(link)
	body = "<p>" + html.EscapeString(texts.T(l, emailIntro)) + "</p>" +
		`<p><a href="` + href + `">` + html.EscapeString(texts.T(l, button)) + "</a></p>" +
		"<p>" + html.EscapeString(texts.T(l, validity)) + "</p>" +
		"<p>" + html.EscapeString(texts.T(l, ignore)) + "</p>"
	return texts.T(l, subject), body
}

// Telegram is the chat message (Telegram HTML) and its button opening link.
func Telegram(l telegram.Lang, link string) (string, telegram.Button) {
	text := "🔑 <b>" + texts.T(l, title) + "</b>\n\n" + texts.T(l, intro) + "\n\n" +
		texts.T(l, validity) + "\n" + texts.T(l, ignore)
	return text, telegram.Button{Text: texts.T(l, button), URL: link}
}
