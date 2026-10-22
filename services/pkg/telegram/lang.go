package telegram

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"time"
)

// Lang is a language the bot speaks — the site's: Russian, Kyrgyz,
// English and German. Each chat reads the bot in one (services/bot keeps
// it; the users service hands it out with the chat id), and notifications
// are written in it.
type Lang string

const (
	RU Lang = "ru"
	KY Lang = "ky"
	EN Lang = "en"
	DE Lang = "de"
)

// Langs are the bot's languages, Russian (the source) first.
var Langs = []Lang{RU, KY, EN, DE}

// ParseLang is the language for a chat's language code: Russian when it
// is unknown or empty (the bot hasn't reported one yet).
func ParseLang(code string) Lang {
	switch l := Lang(code); l {
	case KY, EN, DE:
		return l
	default:
		return RU
	}
}

// Catalog translates texts written in Russian, keyed by the Russian text —
// the bot's scheme (services/bot/internal/i18n), so the wording can be
// shared.
type Catalog map[Lang]map[string]string

// LoadCatalog reads <dir>/<lang>.json for every language but Russian.
func LoadCatalog(fsys fs.FS, dir string) (Catalog, error) {
	c := Catalog{}
	for _, l := range Langs[1:] {
		data, err := fs.ReadFile(fsys, path.Join(dir, string(l)+".json"))
		if err != nil {
			return nil, err
		}
		texts := map[string]string{}
		if err := json.Unmarshal(data, &texts); err != nil {
			return nil, fmt.Errorf("%s/%s.json: %w", dir, l, err)
		}
		c[l] = texts
	}
	return c, nil
}

// MustLoadCatalog is LoadCatalog for embedded catalogs.
func MustLoadCatalog(fsys fs.FS, dir string) Catalog {
	c, err := LoadCatalog(fsys, dir)
	if err != nil {
		panic(err)
	}
	return c
}

// T is text in l; a text missing from the catalog stays in Russian.
func (c Catalog) T(l Lang, text string) string {
	if t, ok := c[l][text]; ok && t != "" {
		return t
	}
	return text
}

// F translates a format written in Russian and fills it in.
func (c Catalog) F(l Lang, format string, args ...any) string {
	return fmt.Sprintf(c.T(l, format), args...)
}

// Short weekdays (Sunday first) and abbreviated months, as the bot writes
// them.
var dateNames = map[Lang]struct{ weekdays, months []string }{
	RU: {
		[]string{"вс", "пн", "вт", "ср", "чт", "пт", "сб"},
		[]string{"янв.", "фев.", "мар.", "апр.", "мая", "июн.", "июл.", "авг.", "сен.", "окт.", "нояб.", "дек."},
	},
	KY: {
		[]string{"жк", "дш", "шш", "шр", "бш", "жм", "иш"},
		[]string{"янв.", "фев.", "мар.", "апр.", "май", "июн.", "июл.", "авг.", "сен.", "окт.", "нояб.", "дек."},
	},
	EN: {
		[]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
		[]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
	},
	DE: {
		[]string{"So.", "Mo.", "Di.", "Mi.", "Do.", "Fr.", "Sa."},
		[]string{"Jan.", "Feb.", "März", "Apr.", "Mai", "Juni", "Juli", "Aug.", "Sept.", "Okt.", "Nov.", "Dez."},
	},
}

// When is t as wall clock time in its location, the way the bot writes
// it: "сб, 24 окт., 06:00", "Sat, 24 Oct, 06:00", "Sa., 24. Okt., 06:00".
func (l Lang) When(t time.Time) string {
	n, ok := dateNames[l]
	if !ok {
		n = dateNames[RU]
	}
	month := n.months[t.Month()-1]
	var day string
	switch l {
	case KY:
		day = fmt.Sprintf("%d-%s", t.Day(), month)
	case DE:
		day = fmt.Sprintf("%d. %s", t.Day(), month)
	default:
		day = fmt.Sprintf("%d %s", t.Day(), month)
	}
	return n.weekdays[t.Weekday()] + ", " + day + ", " + t.Format("15:04")
}
