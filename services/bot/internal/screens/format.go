// Package screens renders the bot's sections — outings, tours, the blog,
// notification settings — as HTML messages with inline keyboards. A
// section is one message: moving around inside it edits that message.
// Layouts follow the bot's design sketches.
package screens

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mymmrac/telego"
)

// Screen is one message: HTML text, buttons and an optional link preview
// (the cover picture of a tour or post, shown above the text).
type Screen struct {
	Text    string
	Markup  *telego.InlineKeyboardMarkup
	Preview *telego.LinkPreviewOptions
}

// Links builds absolute URLs for messages.
type Links struct {
	// The public site, no trailing slash.
	Site string
	// The gateway's public address, for /uploads/… pictures.
	PublicAPI string
}

func (l Links) site(path string) string { return l.Site + path }

// Picture is an absolute URL of a cover Telegram can fetch, "" if none.
// Uploads go through the media service's thumbnails (1280 px wide).
func (l Links) Picture(src *string) string {
	if src == nil || *src == "" {
		return ""
	}
	s := *src
	switch {
	case strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://"):
		return s
	case strings.HasPrefix(s, "/uploads/") && l.PublicAPI != "":
		return l.PublicAPI + "/images/1280" + s
	default:
		return ""
	}
}

// coverPreview shows a picture above the text, or no preview at all.
func coverPreview(picture string) *telego.LinkPreviewOptions {
	if picture == "" {
		return &telego.LinkPreviewOptions{IsDisabled: true}
	}
	return &telego.LinkPreviewOptions{URL: picture, PreferLargeMedia: true, ShowAboveText: true}
}

func noPreview() *telego.LinkPreviewOptions { return &telego.LinkPreviewOptions{IsDisabled: true} }

var (
	bishkek  = time.FixedZone("Asia/Bishkek", 6*60*60)
	weekdays = []string{"вс", "пн", "вт", "ср", "чт", "пт", "сб"}
	months   = []string{"янв.", "фев.", "мар.", "апр.", "мая", "июн.", "июл.", "авг.", "сен.", "окт.", "нояб.", "дек."}
	// Months in "12 — 14 июня" (genitive, unabbreviated).
	monthsFull = []string{
		"января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря",
	}
)

// When formats a time in Bishkek, e.g. "сб, 24 окт., 06:00" (as the
// outing notifications do).
func When(t time.Time) string {
	t = t.In(bishkek)
	return fmt.Sprintf("%s, %d %s, %s", weekdays[t.Weekday()], t.Day(), months[t.Month()-1], t.Format("15:04"))
}

func sameDay(a, b time.Time) bool {
	a, b = a.In(bishkek), b.In(bishkek)
	return a.YearDay() == b.YearDay() && a.Year() == b.Year()
}

// day formats a date as "18 окт. 2026".
func day(t time.Time) string {
	t = t.In(bishkek)
	return fmt.Sprintf("%d %s %d", t.Day(), months[t.Month()-1], t.Year())
}

// dateRange formats YYYY-MM-DD dates: "12 — 14 июня", "30 июня — 2 июля".
func dateRange(start, end string) string {
	s, err1 := time.Parse(time.DateOnly, start)
	e, err2 := time.Parse(time.DateOnly, end)
	switch {
	case err1 != nil:
		return ""
	case err2 != nil || s.Equal(e):
		return fmt.Sprintf("%d %s", s.Day(), monthsFull[s.Month()-1])
	case s.Month() == e.Month() && s.Year() == e.Year():
		return fmt.Sprintf("%d — %d %s", s.Day(), e.Day(), monthsFull[e.Month()-1])
	default:
		return fmt.Sprintf("%d %s — %d %s", s.Day(), monthsFull[s.Month()-1], e.Day(), monthsFull[e.Month()-1])
	}
}

func esc(s string) string { return html.EscapeString(s) }

// money formats som with thin groups: "18 000".
func money(n int) string {
	s := strconv.Itoa(n)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// clip shortens text to n characters at a word boundary, with "…".
func clip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	cut := string([]rune(s)[:n])
	if i := strings.LastIndex(cut, " "); i > n/2 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,.;:—-") + "…"
}

// days is "1 день", "3 дня", "7 дней".
func days(n int) string { return strconv.Itoa(n) + " " + plural(n, "день", "дня", "дней") }

// plural picks the form for n: one (1 выход), few (2 выхода), many (5 выходов).
func plural(n int, one, few, many string) string {
	switch {
	case n%10 == 1 && n%100 != 11:
		return one
	case n%10 >= 2 && n%10 <= 4 && (n%100 < 10 || n%100 >= 20):
		return few
	default:
		return many
	}
}

func rows(r ...[]telego.InlineKeyboardButton) *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{InlineKeyboard: r}
}
