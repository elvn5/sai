package screens

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/api"
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
func tourFacts(t api.Tour) string {
	var b strings.Builder
	var place []string
	if t.Location != nil && *t.Location != "" {
		place = append(place, "📍 "+esc(*t.Location))
	}
	if t.DurationDays != nil && *t.DurationDays > 0 {
		place = append(place, days(*t.DurationDays))
	}
	if len(place) > 0 {
		b.WriteString("\n" + strings.Join(place, " · "))
	}
	if r := dateRange(t.StartDate, t.EndDate); r != "" {
		b.WriteString("\n📅 " + r)
	}
	if t.Price != nil {
		b.WriteString("\n💳 " + money(*t.Price) + " сом с человека")
	}
	if t.ReviewCount > 0 {
		fmt.Fprintf(&b, "\n⭐ %.1f · %d %s", t.RatingAverage, t.ReviewCount, plural(t.ReviewCount, "отзыв", "отзыва", "отзывов"))
	}
	return b.String()
}

// Tours shows one tour of a tab, with its cover above the text.
func Tours(v ToursView, links Links) Screen {
	n := len(v.Tours)
	index := 0
	if n > 0 {
		index = min(max(v.Index, 0), n-1)
	}
	tabName := "Избранное"
	if v.Tab == TabUpcoming {
		tabName = "Ближайшие"
	}

	var b strings.Builder
	b.WriteString("🧭 <b>Туры</b> · " + tabName)
	var buttons [][]telego.InlineKeyboardButton
	picture := ""
	switch {
	case n == 0 && v.Tab == TabFavorites:
		b.WriteString("\n\n❤️ В избранном пока пусто.\n\nОтмечайте туры сердечком — здесь, в «Ближайших», или на сайте, чтобы вернуться к ним позже.")
	case n == 0:
		b.WriteString("\n\nБлижайших туров пока нет. Включите уведомления — сообщим, как только появятся.")
	default:
		t := v.Tours[index]
		picture = links.Picture(t.CoverImageURL)
		fmt.Fprintf(&b, " %d из %d\n\n<b>%s</b>", index+1, n, esc(t.Title))
		b.WriteString(tourFacts(t))
		if t.Excerpt != nil && *t.Excerpt != "" {
			b.WriteString("\n\n<i>" + esc(clip(*t.Excerpt, 300)) + "</i>")
		}

		if n > 1 {
			buttons = append(buttons, keyboards.Pager(index, n, func(i int) string { return TourData(v.Tab, i) }))
		}
		save := keyboards.Button("❤️ В избранное", tourAction(ActSave, v.Tab, index, t.ID))
		if v.Favorites[t.ID] {
			save = keyboards.Button("💔 Из избранного", tourAction(ActDrop, v.Tab, index, t.ID))
		}
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("📖 Подробнее", MoreData(v.Tab, index, 0)), save})
	}

	buttons = append(buttons, keyboards.Tabs(TourData(v.Tab, 0),
		[2]string{"❤️ Избранное", TourData(TabFavorites, 0)},
		[2]string{"🆕 Ближайшие", TourData(TabUpcoming, 0)},
	))
	alert := "🔔 Сообщать о новых турах"
	if v.AlertOn {
		alert = "🔕 Не сообщать о новых турах"
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(alert, tourAction(ActAlert, v.Tab, index, ""))})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: coverPreview(picture)}
}

// tourParagraphs are a tour's description, program and contacts.
func tourParagraphs(t api.Tour) []string {
	var out []string
	if t.Capacity != nil && *t.Capacity > 0 {
		out = append(out, fmt.Sprintf("👥 В группе до %d %s", *t.Capacity, plural(*t.Capacity, "человека", "человек", "человек")))
	}
	out = append(out, Paragraphs(t.Content)...)
	if len(out) == 0 && t.Excerpt != nil {
		out = append(out, PlainParagraphs(*t.Excerpt)...)
	}
	if len(t.Program) > 0 {
		out = append(out, "🗓 <b>Программа</b>")
		for i, day := range t.Program {
			title := fmt.Sprintf("День %d", i+1)
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
		contacts = append(contacts, "Телефон: "+esc(*t.Phone))
	}
	if len(contacts) > 0 {
		out = append(out, "📞 <b>Записаться и спросить</b>\n"+strings.Join(contacts, "\n"))
	}
	return out
}

// TourMore is a page of a tour's description, read in the chat; the
// cover and the facts open the first page.
func TourMore(t api.Tour, tab string, index, page int, favorite bool, links Links) Screen {
	pages := Pages(tourParagraphs(t), pageLimit)
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
		b.WriteString(tourFacts(t))
	}
	if len(pages) > 0 {
		b.WriteString("\n\n" + pages[page])
	}

	var buttons [][]telego.InlineKeyboardButton
	if n > 1 {
		var row []telego.InlineKeyboardButton
		if page > 0 {
			row = append(row, keyboards.Button("◀️ Назад", MoreData(tab, index, page-1)))
		}
		if page < n-1 {
			row = append(row, keyboards.Button("Дальше ▶️", MoreData(tab, index, page+1)))
		}
		buttons = append(buttons, row)
	}
	save := keyboards.Button("❤️ В избранное", tourAction(ActSave, tab, index, t.ID)+":"+ActMore)
	if favorite {
		save = keyboards.Button("💔 Из избранного", tourAction(ActDrop, tab, index, t.ID)+":"+ActMore)
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{save, keyboards.Button("⬅️ К турам", TourData(tab, index))})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: coverPreview(picture)}
}
