package screens

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
)

// Where a tour's card was opened from, for its back button.
const (
	FromMine  = "u" // my upcoming tours
	FromPast  = "p" // my past tours
	FromTours = "t" // the tours section
	// A notification's button: the card comes as a new message.
	FromNotice = "n"
)

// "My tours" callback actions: "o:<action>:<from>:<tour id>", plus
// "o:list:<tab>".
const (
	ActList  = "list"
	ActView  = "view"
	ActJoin  = "join"
	ActAsk   = "ask"
	ActLeave = "leave"
)

const (
	myToursShown = 8
	// A tour still counts as upcoming this long after the meeting time.
	startGrace = 3 * time.Hour
	// Participants shown by name on a card; the rest are counted.
	namesShown = 15
)

func TripData(action, from, id string) string {
	return keyboards.MyToursPrefix + action + ":" + from + ":" + id
}

func ListData(tab string) string { return keyboards.MyToursPrefix + ActList + ":" + tab }

// IsUpcoming: the group hasn't met yet (with a few hours' grace).
func IsUpcoming(t api.Tour, now time.Time) bool { return t.StartsAt().After(now.Add(-startGrace)) }

// SplitTours separates upcoming tours (soonest first) from past ones
// (latest first).
func SplitTours(list []api.Tour, now time.Time) (upcoming, past []api.Tour) {
	for _, t := range list {
		if IsUpcoming(t, now) {
			upcoming = append(upcoming, t)
		} else {
			past = append(past, t)
		}
	}
	sort.SliceStable(upcoming, func(i, j int) bool { return upcoming[i].StartsAt().Before(upcoming[j].StartsAt()) })
	sort.SliceStable(past, func(i, j int) bool { return past[i].StartsAt().After(past[j].StartsAt()) })
	return upcoming, past
}

// placesLine is "7 из 10 мест", or "записались: 7" without a limit.
func placesLine(l i18n.Lang, t api.Tour) string {
	if t.Capacity == nil {
		return l.F("записались: %d", t.Taken)
	}
	return l.F("%d из %d мест", t.Taken, *t.Capacity)
}

func tripLine(l i18n.Lang, n int, t api.Tour, meta ...string) string {
	parts := append([]string{When(l, t.StartsAt())}, meta...)
	var kept []string
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return fmt.Sprintf("<b>%d. %s</b>\n%s", n, esc(t.Title), strings.Join(kept, " · "))
}

func tripButton(n int, t api.Tour, from string) []telego.InlineKeyboardButton {
	return []telego.InlineKeyboardButton{keyboards.Button(fmt.Sprintf("%d · %s", n, clip(t.Title, 40)), TripData(ActView, from, t.ID))}
}

// MyTours lists the tours the user signed up for: upcoming (tab FromMine)
// or past (FromPast).
func MyTours(l i18n.Lang, list []api.Tour, tab string, now time.Time) Screen {
	upcoming, past := SplitTours(list, now)
	shown, from := upcoming, FromMine
	if tab == FromPast {
		shown, from = past, FromPast
	}

	var b strings.Builder
	b.WriteString(l.T("🎒 <b>Мои туры</b>"))
	if len(past) > 0 {
		b.WriteString("\n" + l.F("За всё время: %d %s", len(past), l.N(len(past), "тур", "тура", "туров")))
	}
	if from == FromMine {
		b.WriteString("\n\n<b>" + l.T("Предстоящие") + "</b>")
	} else {
		b.WriteString("\n\n<b>" + l.T("Прошедшие") + "</b>")
	}

	var buttons [][]telego.InlineKeyboardButton
	switch {
	case len(shown) == 0 && from == FromMine:
		b.WriteString("\n\n" + l.T("Вы пока не записаны ни на один тур.\nЗагляните в ближайшие — может, найдётся компания на выходные."))
	case len(shown) == 0:
		b.WriteString("\n\n" + l.T("Прошедших туров пока нет."))
	}
	for i, t := range shown {
		if i == myToursShown {
			b.WriteString("\n\n" + l.F("…и ещё %d.", len(shown)-myToursShown))
			break
		}
		b.WriteString("\n\n" + tripLine(l, i+1, t, "👥 "+placesLine(l, t)))
		buttons = append(buttons, tripButton(i+1, t, from))
	}

	buttons = append(buttons,
		keyboards.Tabs(ListData(from), [2]string{l.T("Предстоящие"), ListData(FromMine)}, [2]string{l.T("Прошедшие"), ListData(FromPast)}),
		[]telego.InlineKeyboardButton{keyboards.Button(l.T("🧭 Все туры"), TourData(TabUpcoming, 0))},
		[]telego.InlineKeyboardButton{keyboards.Button(l.T("🧑‍✈️ Кабинет организатора"), OrgHome)},
	)
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: noPreview()}
}

// seatsBar draws taken of capacity as ▰▱, ten marks at most.
func seatsBar(taken, capacity int) string {
	if capacity <= 0 {
		return ""
	}
	marks := min(capacity, 10)
	filled := int(math.Round(float64(taken) / float64(capacity) * float64(marks)))
	if taken > 0 && filled == 0 {
		filled = 1
	}
	filled = min(filled, marks)
	return strings.Repeat("▰", filled) + strings.Repeat("▱", marks-filled)
}

// participantNames lists who signed up: "Аня, @bob и ещё 3".
func participantNames(l i18n.Lang, roster []api.RosterEntry) string {
	var names []string
	others := 0
	for _, p := range roster {
		if name := p.Name(); name != "" && len(names) < namesShown {
			names = append(names, esc(name))
		} else {
			others++
		}
	}
	text := strings.Join(names, ", ")
	switch {
	case others > 0 && len(names) > 0:
		text += " " + l.F("и ещё %d", others)
	case others > 0:
		text = people(l, others)
	}
	return text
}

// contacts are a tour's organizer's contacts, one per line.
func contacts(l i18n.Lang, t api.Tour) []string {
	var out []string
	if t.Telegram != nil && *t.Telegram != "" {
		out = append(out, "Telegram: @"+esc(strings.TrimPrefix(*t.Telegram, "@")))
	}
	if t.Whatsapp != nil && *t.Whatsapp != "" {
		out = append(out, "WhatsApp: "+esc(*t.Whatsapp))
	}
	if t.Phone != nil && *t.Phone != "" {
		out = append(out, l.F("Телефон: %s", esc(*t.Phone)))
	}
	return out
}

// tripDetails is what a sign-up needs: when and where the group meets,
// the price, places and the organizer.
func tripDetails(l i18n.Lang, t api.Tour) string {
	var b strings.Builder
	fmt.Fprintf(&b, "🧭 <b>%s</b>", esc(t.Title))
	if r := dateRange(l, t.StartDate, t.EndDate); r != "" {
		b.WriteString("\n📅 " + r)
	}
	b.WriteString("\n\n🕕 " + l.F("Сбор: %s", When(l, t.StartsAt())))
	if t.MeetingPoint != nil && *t.MeetingPoint != "" {
		b.WriteString("\n📍 " + esc(*t.MeetingPoint))
	}
	if t.Price != nil {
		b.WriteString("\n💳 " + l.F("%s сом с человека", money(*t.Price)))
	}
	if t.Capacity != nil {
		b.WriteString("\n👥 " + seatsBar(t.Taken, *t.Capacity) + " " + placesLine(l, t))
	} else {
		b.WriteString("\n👥 " + placesLine(l, t))
	}
	if t.Organizer != nil && t.Organizer.Name() != "" {
		b.WriteString("\n🧑 " + l.F("Организатор: %s", esc(t.Organizer.Name())))
	}
	return b.String()
}

// TourCard is one tour with the user's place on it and what they can do.
func TourCard(l i18n.Lang, t api.Tour, from string, links Links, now time.Time) Screen {
	var b strings.Builder
	b.WriteString(tripDetails(l, t))
	if t.IsJoined || t.IsOrganizer {
		if c := contacts(l, t); len(c) > 0 && !t.IsOrganizer {
			b.WriteString("\n\n📞 <b>" + l.T("Связь с организатором") + "</b>\n" + strings.Join(c, "\n"))
		}
		if len(t.Roster) > 0 {
			b.WriteString("\n\n👥 " + l.F("Едут: %s", participantNames(l, t.Roster)))
		}
	}

	upcoming := IsUpcoming(t, now)
	switch {
	case !upcoming:
		b.WriteString("\n\n" + l.T("🏁 Тур уже прошёл"))
	case t.IsOrganizer:
		b.WriteString("\n\n" + l.T("⭐ Вы организатор"))
	case t.IsJoined:
		b.WriteString("\n\n" + l.T("✅ Вы записаны"))
	case t.Full():
		b.WriteString("\n\n" + l.T("Мест больше нет"))
	default:
		b.WriteString("\n\n" + l.T("Есть свободные места — записывайтесь!"))
	}

	var buttons [][]telego.InlineKeyboardButton
	if t.IsOrganizer && now.Before(t.StartsAt().Add(checkInAfter)) {
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("📋 Участники и отметка"), OrgData(OrgRoster, t.ID))})
	}
	switch {
	case upcoming && t.IsJoined:
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("🙅 Отменить запись"), TripData(ActAsk, from, t.ID))})
	case upcoming && !t.IsOrganizer && !t.Full():
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("✋ Записаться"), TripData(ActJoin, from, t.ID))})
	}
	if t.CanRate || t.MyRating != nil {
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("⭐ Оценить тур"), RateData(RateAsk, t.ID))})
	}
	if upcoming && !t.Full() {
		if share := inviteFriends(l, t, links); share != nil {
			buttons = append(buttons, []telego.InlineKeyboardButton{*share})
		}
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{backButton(l, from)})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: noPreview()}
}

// inviteFriends shares a tour with Telegram's share dialog.
func inviteFriends(l i18n.Lang, t api.Tour, links Links) *telego.InlineKeyboardButton {
	return links.Share(l.T("📤 Позвать друзей"), TourDeepLink(t.ID), l.F("Поехали в тур: «%s», %s", t.Title, When(l, t.StartsAt())))
}

// TourDeepLink is the /start payload opening a tour's card.
func TourDeepLink(id string) string { return "tour_" + id }

func backButton(l i18n.Lang, from string) telego.InlineKeyboardButton {
	switch from {
	case FromTours:
		return keyboards.Button(l.T("⬅️ К турам"), TourData(TabUpcoming, 0))
	case FromPast:
		return keyboards.Button(l.T("⬅️ К моим турам"), ListData(FromPast))
	default:
		return keyboards.Button(l.T("⬅️ К моим турам"), ListData(FromMine))
	}
}

// LeaveConfirm asks before giving up a place on a tour.
func LeaveConfirm(l i18n.Lang, t api.Tour, from string) Screen {
	text := l.F("Точно отменить запись на «%s»?\n\nМесто освободится, организатор получит уведомление.", esc(t.Title))
	return Screen{Text: text, Preview: noPreview(), Markup: rows([]telego.InlineKeyboardButton{
		keyboards.Button(l.T("Да, отменить"), TripData(ActLeave, from, t.ID)),
		keyboards.Button(l.T("Нет, еду"), TripData(ActView, from, t.ID)),
	})}
}
