package screens

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
)

// Tours tabs.
const (
	TabFavorites = "f"
	TabUpcoming  = "n"
)

// Tours callback actions: "t:<tab>:<index>" shows a tour, "t:save:…" and
// "t:drop:…" (+ ":<tour id>") change favorites, "t:alert:…" toggles new
// tour alerts — each then shows the same place again. "t:more:<tab>:
// <index>:<page>" reads a tour's description; favorites changed there
// end with ":more" to stay on it.
const (
	ActSave  = "save"
	ActDrop  = "drop"
	ActAlert = "alert"
	ActMore  = "more"
	// "t:open:<tour id>": a new-tour alert's button (listings sends it).
	ActOpen = "open"
)

// MoreData opens a page of a tour's description.
func MoreData(tab string, index, page int) string {
	return keyboards.ToursPrefix + ActMore + ":" + tab + ":" + strconv.Itoa(index) + ":" + strconv.Itoa(page)
}

func TourData(tab string, index int) string {
	return keyboards.ToursPrefix + tab + ":" + strconv.Itoa(index)
}

func tourAction(action, tab string, index int, id string) string {
	data := keyboards.ToursPrefix + action + ":" + tab + ":" + strconv.Itoa(index)
	if id != "" {
		data += ":" + id
	}
	return data
}

// ToursView is what the tours screen shows.
type ToursView struct {
	Tab   string
	Index int
	Tours []api.Tour
	// Ids of the user's favorite tours.
	Favorites map[string]bool
	// Whether the user hears about every new tour.
	AlertOn bool
}

// tourFacts are a tour's place, length, dates, price and rating lines.
func tourFacts(l i18n.Lang, t api.Tour) string {
	var b strings.Builder
	var place []string
	if t.Location != nil && *t.Location != "" {
		place = append(place, "📍 "+esc(*t.Location))
	}
	if t.DurationDays != nil && *t.DurationDays > 0 {
		place = append(place, days(l, *t.DurationDays))
	}
	if len(place) > 0 {
		b.WriteString("\n" + strings.Join(place, " · "))
	}
	if r := dateRange(l, t.StartDate, t.EndDate); r != "" {
		b.WriteString("\n📅 " + r)
	}
	if t.Price != nil {
		b.WriteString("\n💳 " + l.F("%s сом с человека", money(*t.Price)))
	}
	b.WriteString(reviews(l, t))
	return b.String()
}

// reviews is a tour's rating line: "⭐ 4.8 · 12 отзывов".
func reviews(l i18n.Lang, t api.Tour) string {
	if t.ReviewCount == 0 {
		return ""
	}
	return fmt.Sprintf("\n⭐ %.1f · %d %s", t.RatingAverage, t.ReviewCount, l.N(t.ReviewCount, "отзыв", "отзыва", "отзывов"))
}

// Tours shows one tour of a tab, with its cover above the text.
func Tours(l i18n.Lang, v ToursView, links Links) Screen {
	n := len(v.Tours)
	index := 0
	if n > 0 {
		index = min(max(v.Index, 0), n-1)
	}
	tabName := l.T("Избранное")
	if v.Tab == TabUpcoming {
		tabName = l.T("Ближайшие")
	}

	var b strings.Builder
	b.WriteString(l.T("🧭 <b>Туры</b>") + " · " + tabName)
	var buttons [][]telego.InlineKeyboardButton
	picture := ""
	switch {
	case n == 0 && v.Tab == TabFavorites:
		b.WriteString("\n\n" + l.T("❤️ В избранном пока пусто.\n\nОтмечайте туры сердечком — здесь, в «Ближайших», или на сайте, чтобы вернуться к ним позже."))
	case n == 0:
		b.WriteString("\n\n" + l.T("Ближайших туров пока нет. Включите уведомления — сообщим, как только появятся."))
	default:
		t := v.Tours[index]
		picture = links.Picture(t.CoverImageURL)
		b.WriteString(" " + l.F("%d из %d", index+1, n) + "\n\n<b>" + esc(t.Title) + "</b>")
		b.WriteString(tourFacts(l, t))
		if t.Excerpt != nil && *t.Excerpt != "" {
			b.WriteString("\n\n<i>" + esc(clip(*t.Excerpt, 300)) + "</i>")
		}

		if n > 1 {
			buttons = append(buttons, keyboards.Pager(index, n, func(i int) string { return TourData(v.Tab, i) }))
		}
		save := keyboards.Button(l.T("❤️ В избранное"), tourAction(ActSave, v.Tab, index, t.ID))
		if v.Favorites[t.ID] {
			save = keyboards.Button(l.T("💔 Из избранного"), tourAction(ActDrop, v.Tab, index, t.ID))
		}
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("📖 Подробнее"), MoreData(v.Tab, index, 0)), save})
	}

	buttons = append(buttons, keyboards.Tabs(TourData(v.Tab, 0),
		[2]string{l.T("❤️ Избранное"), TourData(TabFavorites, 0)},
		[2]string{l.T("🆕 Ближайшие"), TourData(TabUpcoming, 0)},
	))
	alert := l.T("🔔 Сообщать о новых турах")
	if v.AlertOn {
		alert = l.T("🔕 Не сообщать о новых турах")
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(alert, tourAction(ActAlert, v.Tab, index, ""))})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: coverPreview(picture)}
}

// tourParagraphs are a tour's description, program and contacts.
func tourParagraphs(l i18n.Lang, t api.Tour) []string {
	var out []string
	if t.Capacity != nil && *t.Capacity > 0 {
		out = append(out, "👥 "+l.F("В группе до %d %s", *t.Capacity, l.N(*t.Capacity, "человека", "человек", "человек")))
	}
	out = append(out, Paragraphs(t.Content)...)
	if len(out) == 0 && t.Excerpt != nil {
		out = append(out, PlainParagraphs(*t.Excerpt)...)
	}
	if len(t.Program) > 0 {
		out = append(out, "🗓 <b>"+l.T("Программа")+"</b>")
		for i, day := range t.Program {
			title := l.F("День %d", i+1)
			if day.Title != "" {
				title += ". " + day.Title
			}
			text := "<b>" + esc(title) + "</b>"
			if body := strings.TrimSpace(day.Text); body != "" {
				text += "\n" + strings.Join(PlainParagraphs(body), "\n")
			}
			out = append(out, text)
		}
	}
	var contacts []string
	if t.Telegram != nil && *t.Telegram != "" {
		contacts = append(contacts, "Telegram: "+esc(*t.Telegram))
	}
	if t.Whatsapp != nil && *t.Whatsapp != "" {
		contacts = append(contacts, "WhatsApp: "+esc(*t.Whatsapp))
	}
	if t.Phone != nil && *t.Phone != "" {
		contacts = append(contacts, l.F("Телефон: %s", esc(*t.Phone)))
	}
	if len(contacts) > 0 {
		out = append(out, "📞 <b>"+l.T("Записаться и спросить")+"</b>\n"+strings.Join(contacts, "\n"))
	}
	return out
}

// TourMore is a page of a tour's description, read in the chat; the
// cover and the facts open the first page.
func TourMore(l i18n.Lang, t api.Tour, tab string, index, page int, favorite bool, links Links) Screen {
	pages := Pages(tourParagraphs(l, t), pageLimit)
	n := max(len(pages), 1)
	page = min(max(page, 0), n-1)

	var b strings.Builder
	b.WriteString("🧭 <b>" + esc(t.Title) + "</b>")
	if n > 1 {
		fmt.Fprintf(&b, " · %d/%d", page+1, n)
	}
	picture := ""
	if page == 0 {
		picture = links.Picture(t.CoverImageURL)
		b.WriteString(tourFacts(l, t))
	}
	if len(pages) > 0 {
		b.WriteString("\n\n" + pages[page])
	}

	var buttons [][]telego.InlineKeyboardButton
	if n > 1 {
		var row []telego.InlineKeyboardButton
		if page > 0 {
			row = append(row, keyboards.Button(l.T("◀️ Назад"), MoreData(tab, index, page-1)))
		}
		if page < n-1 {
			row = append(row, keyboards.Button(l.T("Дальше ▶️"), MoreData(tab, index, page+1)))
		}
		buttons = append(buttons, row)
	}
	save := keyboards.Button(l.T("❤️ В избранное"), tourAction(ActSave, tab, index, t.ID)+":"+ActMore)
	if favorite {
		save = keyboards.Button(l.T("💔 Из избранного"), tourAction(ActDrop, tab, index, t.ID)+":"+ActMore)
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{save, keyboards.Button(l.T("⬅️ К турам"), TourData(tab, index))})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: coverPreview(picture)}
}
