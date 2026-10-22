package screens

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
)

// Where an outing card was opened from, for its back button.
const (
	FromMine = "u" // my upcoming outings
	FromPast = "p" // my past outings
	FromFind = "f" // all upcoming outings
	// A notification's button: the card comes as a new message.
	FromNotice = "n"
)

// Outing callback actions: "o:<action>:<from>:<id>", plus "o:list:<tab>"
// and "o:find:<page>".
const (
	ActList  = "list"
	ActFind  = "find"
	ActView  = "view"
	ActJoin  = "join"
	ActAsk   = "ask"
	ActLeave = "leave"
	// The organizer's: change the outing, ask to cancel it, cancel it.
	ActEdit      = "edit"
	ActCancelAsk = "cask"
	ActCancel    = "cancel"
)

// NewOutingData starts putting an outing together.
const NewOutingData = keyboards.DraftPrefix + "new"

// conditionWindow is how long after an outing its route can be reported
// on (as the trails service takes reports).
const conditionWindow = 30 * 24 * time.Hour

const (
	myOutingsShown = 8
	findPageSize   = 4
	// An outing still counts as upcoming this long after its start, as on
	// the site.
	startGrace = 3 * time.Hour
)

func OutingData(action, from, id string) string {
	return keyboards.OutingsPrefix + action + ":" + from + ":" + id
}

func ListData(tab string) string { return keyboards.OutingsPrefix + ActList + ":" + tab }

func FindData(page int) string { return keyboards.OutingsPrefix + ActFind + ":" + strconv.Itoa(page) }

func isUpcoming(o api.Outing, now time.Time) bool {
	return o.Status == "open" && o.StartsAt.After(now.Add(-startGrace))
}

// SplitOutings separates upcoming outings (soonest first) from past and
// cancelled ones (latest first).
func SplitOutings(list []api.Outing, now time.Time) (upcoming, past []api.Outing) {
	for _, o := range list {
		if isUpcoming(o, now) {
			upcoming = append(upcoming, o)
		} else {
			past = append(past, o)
		}
	}
	sort.SliceStable(upcoming, func(i, j int) bool { return upcoming[i].StartsAt.Before(upcoming[j].StartsAt) })
	sort.SliceStable(past, func(i, j int) bool { return past[i].StartsAt.After(past[j].StartsAt) })
	return upcoming, past
}

func role(l i18n.Lang, o api.Outing) string {
	switch {
	case o.Status == "cancelled":
		return l.T("❌ отменён")
	case o.IsOrganizer:
		return l.T("⭐ вы организатор")
	case o.IsJoined:
		return l.T("вы идёте")
	default:
		return ""
	}
}

func outingLine(l i18n.Lang, n int, o api.Outing, meta ...string) string {
	parts := append([]string{When(l, o.StartsAt)}, meta...)
	var kept []string
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return fmt.Sprintf("<b>%d. %s</b>\n%s", n, esc(o.Title), strings.Join(kept, " · "))
}

func outingButton(n int, o api.Outing, from string) []telego.InlineKeyboardButton {
	return []telego.InlineKeyboardButton{keyboards.Button(fmt.Sprintf("%d · %s", n, clip(o.Title, 40)), OutingData(ActView, from, o.ID))}
}

func summaryLine(l i18n.Lang, sum *api.OutingsSummary) string {
	if sum == nil || sum.Outings == 0 {
		return ""
	}
	parts := []string{fmt.Sprintf("%d %s", sum.Outings, l.N(sum.Outings, "выход", "выхода", "выходов"))}
	if km := math.Round(sum.DistanceKm); km > 0 {
		parts = append(parts, money(int(km))+" "+l.T("км"))
	}
	if sum.ElevationGain > 0 {
		parts = append(parts, "↗ "+money(sum.ElevationGain)+" "+l.T("м"))
	}
	return l.F("За всё время: %s", strings.Join(parts, " · "))
}

// MyOutings lists the user's upcoming (tab FromMine) or past (FromPast)
// outings; sum adds the profile's numbers when known.
func MyOutings(l i18n.Lang, list []api.Outing, sum *api.OutingsSummary, tab string, now time.Time) Screen {
	upcoming, past := SplitOutings(list, now)
	shown, from := upcoming, FromMine
	if tab == FromPast {
		shown, from = past, FromPast
	}

	var b strings.Builder
	b.WriteString(l.T("🥾 <b>Мои выходы</b>"))
	if line := summaryLine(l, sum); line != "" {
		b.WriteString("\n" + line)
	}
	if from == FromMine {
		b.WriteString("\n\n<b>" + l.T("Предстоящие") + "</b>")
	} else {
		b.WriteString("\n\n<b>" + l.T("Прошедшие") + "</b>")
	}

	var buttons [][]telego.InlineKeyboardButton
	switch {
	case len(shown) == 0 && from == FromMine:
		b.WriteString("\n\n" + l.T("Вы пока не записаны ни на один выход.\nЗагляните в ближайшие — может, найдётся компания на выходные."))
	case len(shown) == 0:
		b.WriteString("\n\n" + l.T("Прошедших выходов пока нет."))
	}
	for i, o := range shown {
		if i == myOutingsShown {
			b.WriteString("\n\n" + l.F("…и ещё %d.", len(shown)-myOutingsShown))
			break
		}
		meta := []string{}
		if o.Status == "open" {
			meta = append(meta, fmt.Sprintf("👥 %d/%d", o.Taken, o.Seats))
		}
		meta = append(meta, role(l, o))
		b.WriteString("\n\n" + outingLine(l, i+1, o, meta...))
		buttons = append(buttons, outingButton(i+1, o, from))
	}

	buttons = append(buttons,
		keyboards.Tabs(ListData(from), [2]string{l.T("Предстоящие"), ListData(FromMine)}, [2]string{l.T("Прошедшие"), ListData(FromPast)}),
		[]telego.InlineKeyboardButton{keyboards.Button(l.T("🔎 Найти выход"), FindData(0)), keyboards.Button(l.T("➕ Собрать выход"), NewOutingData)},
		[]telego.InlineKeyboardButton{keyboards.Button(l.T("🧑‍✈️ Кабинет организатора"), OrgHome)},
	)
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: noPreview()}
}

// FindOutings is a page of all upcoming outings.
func FindOutings(l i18n.Lang, list []api.Outing, page int) Screen {
	pages := max(1, (len(list)+findPageSize-1)/findPageSize)
	page = min(max(page, 0), pages-1)

	var b strings.Builder
	b.WriteString(l.T("🔎 <b>Ближайшие выходы</b>"))
	var buttons [][]telego.InlineKeyboardButton
	if len(list) == 0 {
		b.WriteString("\n\n" + l.T("Ближайших выходов пока нет. Загляните позже — или соберите свою компанию."))
	}
	start := page * findPageSize
	for i := start; i < min(start+findPageSize, len(list)); i++ {
		o := list[i]
		seats := l.T("мест нет")
		if free := o.Seats - o.Taken; free > 0 {
			seats = l.F("👥 свободно %d из %d", free, o.Seats)
		}
		place := ""
		if o.Trail != nil {
			place = "🏔 " + esc(o.Trail.Name)
		}
		b.WriteString("\n\n" + outingLine(l, i+1, o, seats, place, role(l, o)))
		buttons = append(buttons, outingButton(i+1, o, FromFind))
	}
	if pages > 1 {
		buttons = append(buttons, keyboards.Pager(page, pages, FindData))
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{
		keyboards.Button(l.T("➕ Собрать выход"), NewOutingData), keyboards.Button(l.T("⬅️ Мои выходы"), ListData(FromMine)),
	})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: noPreview()}
}

var (
	levels = map[string]string{
		"beginner": "для новичков", "intermediate": "нужен опыт", "advanced": "для опытных",
	}
	transports = map[string]string{
		"car": "На машине организатора", "bus": "Общественным транспортом", "arranged": "Заказной транспорт",
	}
)

// seatsBar draws taken of seats as ▰▱, ten marks at most.
func seatsBar(taken, seats int) string {
	if seats <= 0 {
		return ""
	}
	marks := min(seats, 10)
	filled := int(math.Round(float64(taken) / float64(seats) * float64(marks)))
	if taken > 0 && filled == 0 {
		filled = 1
	}
	filled = min(filled, marks)
	return strings.Repeat("▰", filled) + strings.Repeat("▱", marks-filled)
}

// Levels and transports as the bot names them, in the order offered.
var (
	LevelNames = map[string]string{
		"beginner": "🌱 Для новичков", "intermediate": "🥾 Нужен опыт", "advanced": "🧗 Для опытных",
	}
	TransportNames = map[string]string{
		"car": "🚗 На машине организатора", "bus": "🚌 Общественным транспортом", "arranged": "🚐 Заказной транспорт",
	}
)

// Participants shown by name on a card; the rest are counted.
const namesShown = 15

// outingDetails is an outing's description: what, when, where, how much.
func outingDetails(l i18n.Lang, o api.Outing) string {
	var b strings.Builder
	fmt.Fprintf(&b, "🥾 <b>%s</b>", esc(o.Title))
	var sub []string
	if o.Trail != nil {
		sub = append(sub, "🏔 "+esc(o.Trail.Name))
	}
	if level, ok := levels[o.Level]; ok {
		sub = append(sub, l.T(level))
	}
	if len(sub) > 0 {
		b.WriteString("\n" + strings.Join(sub, " · "))
	}

	b.WriteString("\n\n📅 " + When(l, o.StartsAt))
	if o.ReturnsAt != nil {
		if sameDay(o.StartsAt, *o.ReturnsAt) {
			b.WriteString(" → " + o.ReturnsAt.In(bishkek).Format("15:04"))
		} else {
			b.WriteString(" → " + When(l, *o.ReturnsAt))
		}
	}
	b.WriteString("\n📍 " + l.F("Сбор: %s", esc(o.MeetingPoint)))
	if t, ok := transports[o.Transport]; ok {
		b.WriteString("\n🚗 " + l.T(t))
	}
	if o.CostPerPerson != nil {
		if *o.CostPerPerson == 0 {
			b.WriteString("\n💳 " + l.T("Бесплатно"))
		} else {
			b.WriteString("\n💳 " + l.F("%s сом с человека", money(*o.CostPerPerson)))
		}
	}
	if o.ID == "" {
		// Not published yet: nobody could join.
		b.WriteString("\n👥 " + l.F("Мест: %d", o.Seats))
	} else {
		b.WriteString("\n👥 " + seatsBar(o.Taken, o.Seats) + " " + l.F("%d из %d мест", o.Taken, o.Seats))
	}
	if name := o.Organizer.Name(); name != "" {
		b.WriteString("\n🧑 " + l.F("Организатор: %s", esc(name)))
	}
	if o.Contact != nil && *o.Contact != "" {
		b.WriteString("\n💬 " + l.F("Связь: %s", esc(*o.Contact)))
	}
	if o.Description != nil && *o.Description != "" {
		b.WriteString("\n\n<i>" + esc(clip(*o.Description, 1500)) + "</i>")
	}
	return b.String()
}

// participantNames lists who joined: "Аня, @bob и ещё 3".
func participantNames(l i18n.Lang, joined []api.Person) string {
	var names []string
	others := 0
	for _, p := range joined {
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

// OutingCard is one outing with what the user can do about it.
func OutingCard(l i18n.Lang, o api.Outing, from string, links Links, now time.Time) Screen {
	var b strings.Builder
	b.WriteString(outingDetails(l, o))
	if (o.IsOrganizer || o.IsJoined) && len(o.Participants) > 0 {
		b.WriteString("\n\n👥 " + l.F("Идут: %s", participantNames(l, o.Participants)))
	}

	upcoming := isUpcoming(o, now)
	full := o.Taken >= o.Seats
	switch {
	case o.Status == "cancelled":
		b.WriteString("\n\n" + l.T("❌ Выход отменён организатором"))
	case !upcoming:
		b.WriteString("\n\n" + l.T("🏁 Выход уже прошёл"))
	case o.IsOrganizer:
		b.WriteString("\n\n" + l.T("⭐ Вы организатор"))
	case o.IsJoined:
		b.WriteString("\n\n" + l.T("✅ Вы идёте"))
	case full:
		b.WriteString("\n\n" + l.T("Мест больше нет"))
	default:
		b.WriteString("\n\n" + l.T("Есть свободные места — присоединяйтесь!"))
	}

	var buttons [][]telego.InlineKeyboardButton
	switch {
	case upcoming && o.IsOrganizer && o.StartsAt.After(now):
		buttons = append(buttons, []telego.InlineKeyboardButton{
			keyboards.Button(l.T("✏️ Изменить"), OutingData(ActEdit, from, o.ID)),
			keyboards.Button(l.T("❌ Отменить выход"), OutingData(ActCancelAsk, from, o.ID)),
		})
	}
	if o.IsOrganizer && o.Status == "open" && now.Before(o.StartsAt.Add(checkInAfter)) {
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("📋 Участники и отметка"), OrgData(OrgRoster, o.ID))})
	}
	switch {
	case upcoming && o.IsJoined:
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("🙅 Не смогу пойти"), OutingData(ActAsk, from, o.ID))})
	case upcoming && !o.IsOrganizer && !full:
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("✋ Пойду"), OutingData(ActJoin, from, o.ID))})
	}
	if upcoming && !full {
		if share := inviteFriends(l, o, links); share != nil {
			buttons = append(buttons, []telego.InlineKeyboardButton{*share})
		}
	}
	if CanReport(o, now) {
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("📝 Рассказать, как маршрут"), ConditionStart(o.ID))})
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{backButton(l, from)})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: noPreview()}
}

// inviteFriends shares an outing with Telegram's share dialog.
func inviteFriends(l i18n.Lang, o api.Outing, links Links) *telego.InlineKeyboardButton {
	return links.Share(l.T("📤 Позвать друзей"), OutingDeepLink(o.ID), l.F("Пойдём в горы: «%s», %s", o.Title, When(l, o.StartsAt)))
}

// OutingDeepLink is the /start payload opening an outing's card.
func OutingDeepLink(id string) string { return "outing_" + id }

// CanReport: someone who went on a finished outing along a catalog route
// can say what the route looked like.
func CanReport(o api.Outing, now time.Time) bool {
	return o.Trail != nil && o.Trail.Slug != "" && o.Status == "open" && (o.IsOrganizer || o.IsJoined) &&
		o.StartsAt.Before(now) && now.Sub(o.StartsAt) < conditionWindow
}

func backButton(l i18n.Lang, from string) telego.InlineKeyboardButton {
	switch from {
	case FromFind:
		return keyboards.Button(l.T("⬅️ К ближайшим выходам"), FindData(0))
	case FromPast:
		return keyboards.Button(l.T("⬅️ К моим выходам"), ListData(FromPast))
	default:
		return keyboards.Button(l.T("⬅️ К моим выходам"), ListData(FromMine))
	}
}

// CancelConfirm asks the organizer before calling an outing off.
func CancelConfirm(l i18n.Lang, o api.Outing, from string) Screen {
	text := l.F("Отменить выход «%s»?", esc(o.Title))
	if o.Taken > 0 {
		text += "\n\n" + l.F("Участников: %d — бот сообщит им об отмене.", o.Taken)
	}
	text += "\n\n" + l.T("Вернуть выход после отмены не получится.")
	return Screen{Text: text, Preview: noPreview(), Markup: rows([]telego.InlineKeyboardButton{
		keyboards.Button(l.T("Да, отменить"), OutingData(ActCancel, from, o.ID)),
		keyboards.Button(l.T("Нет, оставить"), OutingData(ActView, from, o.ID)),
	})}
}

// LeaveConfirm asks before dropping out of an outing.
func LeaveConfirm(l i18n.Lang, o api.Outing, from string) Screen {
	text := l.F("Точно не пойдёте на «%s»?\n\nМесто освободится, организатор получит уведомление.", esc(o.Title))
	return Screen{Text: text, Preview: noPreview(), Markup: rows([]telego.InlineKeyboardButton{
		keyboards.Button(l.T("Да, не пойду"), OutingData(ActLeave, from, o.ID)),
		keyboards.Button(l.T("Нет, иду"), OutingData(ActView, from, o.ID)),
	})}
}
