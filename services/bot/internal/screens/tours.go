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
// tour alerts — each then shows the same place again.
const (
	ActSave  = "save"
	ActDrop  = "drop"
	ActAlert = "alert"
)

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

func tourPath(t api.Tour) string {
	path := "/tours/" + t.Slug
	if t.Locale != "" && t.Locale != "ru" {
		path = "/" + t.Locale + path
	}
	return path
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
		buttons = append(buttons, []telego.InlineKeyboardButton{links.Open("🌐 Подробнее", tourPath(t)), save})
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
