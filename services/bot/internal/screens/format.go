// Package screens renders the bot's sections — my tours, tours, the blog,
// notification settings — as HTML messages with inline keyboards. A
// section is one message: moving around inside it edits that message.
// Layouts follow the bot's design sketches.
package screens

import (
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
)

// Screen is one message: HTML text, buttons and an optional link preview
// (the cover picture of a tour or post, shown above the text).
type Screen struct {
	Text    string
	Markup  *telego.InlineKeyboardMarkup
	Preview *telego.LinkPreviewOptions
}

// Links builds absolute URLs for messages. Everything is done inside the
// bot: links only show pictures and share the bot's own deep links.
type Links struct {
	// The gateway's public address, for /uploads/… pictures.
	PublicAPI string
	// The bot's @username, for t.me deep links ("" leaves them out).
	Bot string
}

// Share is a button opening Telegram's "share to a chat" dialog with a
// deep link into the bot (start payload), nil without the bot's name.
func (l Links) Share(text, payload, message string) *telego.InlineKeyboardButton {
	if l.Bot == "" {
		return nil
	}
	link := "https://t.me/" + l.Bot + "?start=" + payload
	b := keyboards.Link(text, "https://t.me/share/url?url="+url.QueryEscape(link)+"&text="+url.QueryEscape(message))
	return &b
}

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

var bishkek = time.FixedZone("Asia/Bishkek", 6*60*60)

// When formats a time in Bishkek, e.g. "сб, 24 окт., 06:00" (as the
// tour notifications do).
func When(l i18n.Lang, t time.Time) string { return l.When(t.In(bishkek)) }

func sameDay(a, b time.Time) bool {
	a, b = a.In(bishkek), b.In(bishkek)
	return a.YearDay() == b.YearDay() && a.Year() == b.Year()
}

// day formats a date as "18 окт. 2026".
func day(l i18n.Lang, t time.Time) string { return l.Date(t.In(bishkek)) }

// dateRange formats YYYY-MM-DD dates: "12 — 14 июня", "30 июня — 2 июля".
func dateRange(l i18n.Lang, start, end string) string {
	s, err := time.Parse(time.DateOnly, start)
	if err != nil {
		return ""
	}
	e, err := time.Parse(time.DateOnly, end)
	if err != nil {
		e = time.Time{}
	}
	return l.DateRange(s, e)
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
func days(l i18n.Lang, n int) string {
	return strconv.Itoa(n) + " " + l.N(n, "день", "дня", "дней")
}

// som is an amount in som: "18 000 сом".
func som(l i18n.Lang, n int) string { return money(n) + " " + l.T("сом") }

func rows(r ...[]telego.InlineKeyboardButton) *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{InlineKeyboard: r}
}
