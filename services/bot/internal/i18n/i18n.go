// Package i18n speaks the bot's languages — the site's: Russian, Kyrgyz,
// English and German. Texts are written in Russian in the code and looked
// up in the catalogs (locales/<lang>.json, keyed by the Russian text): a
// text missing from a catalog shows in Russian, and catalog_test.go makes
// sure every Russian string in the code has all three translations.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

// Lang is a language the bot speaks.
type Lang string

const (
	RU Lang = "ru"
	KY Lang = "ky"
	EN Lang = "en"
	DE Lang = "de"
)

// Langs are the languages in the order the language picker offers them.
var Langs = []Lang{RU, KY, EN, DE}

// Translated are the languages with a catalog (Russian is the source).
var Translated = []Lang{KY, EN, DE}

//go:embed locales/*.json
var locales embed.FS

var catalogs = map[Lang]map[string]string{}

func init() {
	for _, l := range Translated {
		data, err := locales.ReadFile("locales/" + string(l) + ".json")
		if err != nil {
			panic(err)
		}
		catalog := map[string]string{}
		if err := json.Unmarshal(data, &catalog); err != nil {
			panic(fmt.Sprintf("i18n: locales/%s.json: %v", l, err))
		}
		catalogs[l] = catalog
	}
}

// Catalog is a language's translations, keyed by the Russian text (nil for
// Russian).
func Catalog(l Lang) map[string]string { return catalogs[l] }

// Parse is the language for a code ("ru", "en-US", Telegram's
// language_code): the bot's own languages as they are, Russian for the
// other languages of the region and when unknown, English for the rest.
func Parse(code string) Lang {
	code = strings.ToLower(code)
	if i := strings.IndexAny(code, "-_"); i >= 0 {
		code = code[:i]
	}
	switch code {
	case "ru", "ky", "en", "de":
		return Lang(code)
	case "", "uk", "be", "kk", "uz", "tg", "tk", "az", "hy", "ka":
		return RU
	default:
		return EN
	}
}

// Valid reports whether the bot speaks l.
func (l Lang) Valid() bool {
	switch l {
	case RU, KY, EN, DE:
		return true
	}
	return false
}

// Name is the language as its speakers call it.
func (l Lang) Name() string {
	switch l {
	case KY:
		return "🇰🇬 Кыргызча"
	case EN:
		return "🇬🇧 English"
	case DE:
		return "🇩🇪 Deutsch"
	default:
		return "🇷🇺 Русский"
	}
}

// T translates a text written in Russian.
func (l Lang) T(text string) string {
	if t, ok := catalogs[l][text]; ok && t != "" {
		return t
	}
	return text
}

// F translates a format written in Russian and fills it in.
func (l Lang) F(format string, args ...any) string { return fmt.Sprintf(l.T(format), args...) }

// N is the word for n things, given its Russian forms: one (1 день), few
// (2 дня) and many (5 дней). A catalog keeps the forms joined with "|":
// English and German one and other ("day|days"), Kyrgyz just one.
func (l Lang) N(n int, one, few, many string) string {
	if l == RU || l == "" {
		return ruPlural(n, one, few, many)
	}
	forms := strings.Split(l.T(one+"|"+few+"|"+many), "|")
	if len(forms) == 3 {
		return ruPlural(n, forms[0], forms[1], forms[2])
	}
	if len(forms) == 2 && n != 1 {
		return forms[1]
	}
	return forms[0]
}

func ruPlural(n int, one, few, many string) string {
	switch {
	case n%10 == 1 && n%100 != 11:
		return one
	case n%10 >= 2 && n%10 <= 4 && (n%100 < 10 || n%100 >= 20):
		return few
	default:
		return many
	}
}

// Msg is a text to be said in the reader's language later: packages that
// don't know the language (drafts) return it.
type Msg struct {
	Format string
	Args   []any
}

// M makes a Msg of a format written in Russian.
func M(format string, args ...any) Msg { return Msg{Format: format, Args: args} }

// Say is m in l.
func (l Lang) Say(m Msg) string {
	if m.Format == "" {
		return ""
	}
	if len(m.Args) == 0 {
		return l.T(m.Format)
	}
	return l.F(m.Format, m.Args...)
}

// String is m in Russian.
func (m Msg) String() string { return RU.Say(m) }
