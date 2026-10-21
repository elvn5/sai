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

func role(o api.Outing) string {
	switch {
	case o.Status == "cancelled":
		return "❌ отменён"
	case o.IsOrganizer:
		return "⭐ вы организатор"
	case o.IsJoined:
		return "вы идёте"
	default:
		return ""
	}
}

func outingLine(n int, o api.Outing, meta ...string) string {
	parts := append([]string{When(o.StartsAt)}, meta...)
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

func summaryLine(sum *api.OutingsSummary) string {
	if sum == nil || sum.Outings == 0 {
		return ""
	}
	parts := []string{fmt.Sprintf("%d %s", sum.Outings, plural(sum.Outings, "выход", "выхода", "выходов"))}
	if km := math.Round(sum.DistanceKm); km > 0 {
		parts = append(parts, money(int(km))+" км")
	}
	if sum.ElevationGain > 0 {
		parts = append(parts, "↗ "+money(sum.ElevationGain)+" м")
	}
	return "За всё время: " + strings.Join(parts, " · ")
}

// MyOutings lists the user's upcoming (tab FromMine) or past (FromPast)
// outings; sum adds the profile's numbers when known.
func MyOutings(list []api.Outing, sum *api.OutingsSummary, tab string, now time.Time) Screen {
	upcoming, past := SplitOutings(list, now)
	shown, from := upcoming, FromMine
	if tab == FromPast {
		shown, from = past, FromPast
	}

	var b strings.Builder
	b.WriteString("🥾 <b>Мои выходы</b>")
	if line := summaryLine(sum); line != "" {
		b.WriteString("\n" + line)
	}
	if from == FromMine {
		b.WriteString("\n\n<b>Предстоящие</b>")
	} else {
		b.WriteString("\n\n<b>Прошедшие</b>")
	}

	var buttons [][]telego.InlineKeyboardButton
	switch {
	case len(shown) == 0 && from == FromMine:
		b.WriteString("\n\nВы пока не записаны ни на один выход.\nЗагляните в ближайшие — может, найдётся компания на выходные.")
	case len(shown) == 0:
		b.WriteString("\n\nПрошедших выходов пока нет.")
	}
	for i, o := range shown {
		if i == myOutingsShown {
			fmt.Fprintf(&b, "\n\n…и ещё %d.", len(shown)-myOutingsShown)
			break
		}
		meta := []string{}
		if o.Status == "open" {
			meta = append(meta, fmt.Sprintf("👥 %d/%d", o.Taken, o.Seats))
		}
		meta = append(meta, role(o))
		b.WriteString("\n\n" + outingLine(i+1, o, meta...))
		buttons = append(buttons, outingButton(i+1, o, from))
	}

	buttons = append(buttons,
		keyboards.Tabs(ListData(from), [2]string{"Предстоящие", ListData(FromMine)}, [2]string{"Прошедшие", ListData(FromPast)}),
		[]telego.InlineKeyboardButton{keyboards.Button("🔎 Найти выход", FindData(0)), keyboards.Button("➕ Собрать выход", NewOutingData)},
	)
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: noPreview()}
}

// FindOutings is a page of all upcoming outings.
func FindOutings(list []api.Outing, page int) Screen {
	pages := max(1, (len(list)+findPageSize-1)/findPageSize)
	page = min(max(page, 0), pages-1)

	var b strings.Builder
	b.WriteString("🔎 <b>Ближайшие выходы</b>")
	var buttons [][]telego.InlineKeyboardButton
	if len(list) == 0 {
		b.WriteString("\n\nБлижайших выходов пока нет. Загляните позже — или соберите свою компанию.")
	}
	start := page * findPageSize
	for i := start; i < min(start+findPageSize, len(list)); i++ {
		o := list[i]
		seats := "мест нет"
		if free := o.Seats - o.Taken; free > 0 {
			seats = fmt.Sprintf("👥 свободно %d из %d", free, o.Seats)
		}
		place := ""
		if o.Trail != nil {
			place = "🏔 " + esc(o.Trail.Name)
		}
		b.WriteString("\n\n" + outingLine(i+1, o, seats, place, role(o)))
		buttons = append(buttons, outingButton(i+1, o, FromFind))
	}
	if pages > 1 {
		buttons = append(buttons, keyboards.Pager(page, pages, FindData))
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{
		keyboards.Button("➕ Собрать выход", NewOutingData), keyboards.Button("⬅️ Мои выходы", ListData(FromMine)),
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
func outingDetails(o api.Outing) string {
	var b strings.Builder
	fmt.Fprintf(&b, "🥾 <b>%s</b>", esc(o.Title))
	var sub []string
	if o.Trail != nil {
		sub = append(sub, "🏔 "+esc(o.Trail.Name))
	}
	if level, ok := levels[o.Level]; ok {
		sub = append(sub, level)
	}
	if len(sub) > 0 {
		b.WriteString("\n" + strings.Join(sub, " · "))
	}

	b.WriteString("\n\n📅 " + When(o.StartsAt))
	if o.ReturnsAt != nil {
		if sameDay(o.StartsAt, *o.ReturnsAt) {
			b.WriteString(" → " + o.ReturnsAt.In(bishkek).Format("15:04"))
		} else {
			b.WriteString(" → " + When(*o.ReturnsAt))
		}
	}
	b.WriteString("\n📍 Сбор: " + esc(o.MeetingPoint))
	if t, ok := transports[o.Transport]; ok {
		b.WriteString("\n🚗 " + t)
	}
	if o.CostPerPerson != nil {
		if *o.CostPerPerson == 0 {
			b.WriteString("\n💳 Бесплатно")
		} else {
			b.WriteString("\n💳 " + money(*o.CostPerPerson) + " сом с человека")
		}
	}
	if o.ID == "" {
		// Not published yet: nobody could join.
		fmt.Fprintf(&b, "\n👥 Мест: %d", o.Seats)
	} else {
		fmt.Fprintf(&b, "\n👥 %s %d из %d мест", seatsBar(o.Taken, o.Seats), o.Taken, o.Seats)
	}
	if name := o.Organizer.Name(); name != "" {
		b.WriteString("\n🧑 Организатор: " + esc(name))
	}
	if o.Contact != nil && *o.Contact != "" {
		b.WriteString("\n💬 Связь: " + esc(*o.Contact))
	}
	if o.Description != nil && *o.Description != "" {
		b.WriteString("\n\n<i>" + esc(clip(*o.Description, 1500)) + "</i>")
	}
	return b.String()
}

// participantNames lists who joined: "Аня, @bob и ещё 3".
func participantNames(people []api.Person) string {
	var names []string
	others := 0
	for _, p := range people {
		if name := p.Name(); name != "" && len(names) < namesShown {
			names = append(names, esc(name))
		} else {
			others++
		}
	}
	text := strings.Join(names, ", ")
	switch {
	case others > 0 && len(names) > 0:
		text += fmt.Sprintf(" и ещё %d", others)
	case others > 0:
		text = fmt.Sprintf("%d %s", others, plural(others, "человек", "человека", "человек"))
	}
	return text
}

// OutingCard is one outing with what the user can do about it.
func OutingCard(o api.Outing, from string, links Links, now time.Time) Screen {
	var b strings.Builder
	b.WriteString(outingDetails(o))
	if (o.IsOrganizer || o.IsJoined) && len(o.Participants) > 0 {
		b.WriteString("\n\n👥 Идут: " + participantNames(o.Participants))
	}

	upcoming := isUpcoming(o, now)
	full := o.Taken >= o.Seats
	switch {
	case o.Status == "cancelled":
		b.WriteString("\n\n❌ Выход отменён организатором")
	case !upcoming:
		b.WriteString("\n\n🏁 Выход уже прошёл")
	case o.IsOrganizer:
		b.WriteString("\n\n⭐ Вы организатор")
	case o.IsJoined:
		b.WriteString("\n\n✅ Вы идёте")
	case full:
		b.WriteString("\n\nМест больше нет")
	default:
		b.WriteString("\n\nЕсть свободные места — присоединяйтесь!")
	}

	var buttons [][]telego.InlineKeyboardButton
	switch {
	case upcoming && o.IsOrganizer && o.StartsAt.After(now):
		buttons = append(buttons, []telego.InlineKeyboardButton{
			keyboards.Button("✏️ Изменить", OutingData(ActEdit, from, o.ID)),
			keyboards.Button("❌ Отменить выход", OutingData(ActCancelAsk, from, o.ID)),
		})
	case upcoming && o.IsJoined:
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("🙅 Не смогу пойти", OutingData(ActAsk, from, o.ID))})
	case upcoming && !o.IsOrganizer && !full:
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("✋ Пойду", OutingData(ActJoin, from, o.ID))})
	}
	if upcoming && !full {
		if share := links.Share("📤 Позвать друзей", OutingDeepLink(o.ID), "Пойдём в горы: «"+o.Title+"», "+When(o.StartsAt)); share != nil {
			buttons = append(buttons, []telego.InlineKeyboardButton{*share})
		}
	}
	if CanReport(o, now) {
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("📝 Рассказать, как маршрут", ConditionStart(o.ID))})
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{backButton(from)})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: noPreview()}
}

// OutingDeepLink is the /start payload opening an outing's card.
func OutingDeepLink(id string) string { return "outing_" + id }

// CanReport: someone who went on a finished outing along a catalog route
// can say what the route looked like.
func CanReport(o api.Outing, now time.Time) bool {
	return o.Trail != nil && o.Trail.Slug != "" && o.Status == "open" && (o.IsOrganizer || o.IsJoined) &&
		o.StartsAt.Before(now) && now.Sub(o.StartsAt) < conditionWindow
}

func backButton(from string) telego.InlineKeyboardButton {
	switch from {
	case FromFind:
		return keyboards.Button("⬅️ К ближайшим выходам", FindData(0))
	case FromPast:
		return keyboards.Button("⬅️ К моим выходам", ListData(FromPast))
	default:
		return keyboards.Button("⬅️ К моим выходам", ListData(FromMine))
	}
}

// CancelConfirm asks the organizer before calling an outing off.
func CancelConfirm(o api.Outing, from string) Screen {
	text := fmt.Sprintf("Отменить выход «%s»?", esc(o.Title))
	if o.Taken > 0 {
		text += fmt.Sprintf("\n\nУчастников: %d — бот сообщит им об отмене.", o.Taken)
	}
	text += "\n\nВернуть выход после отмены не получится."
	return Screen{Text: text, Preview: noPreview(), Markup: rows([]telego.InlineKeyboardButton{
		keyboards.Button("Да, отменить", OutingData(ActCancel, from, o.ID)),
		keyboards.Button("Нет, оставить", OutingData(ActView, from, o.ID)),
	})}
}

// LeaveConfirm asks before dropping out of an outing.
func LeaveConfirm(o api.Outing, from string) Screen {
	text := fmt.Sprintf("Точно не пойдёте на «%s»?\n\nМесто освободится, организатор получит уведомление.", esc(o.Title))
	return Screen{Text: text, Preview: noPreview(), Markup: rows([]telego.InlineKeyboardButton{
		keyboards.Button("Да, не пойду", OutingData(ActLeave, from, o.ID)),
		keyboards.Button("Нет, иду", OutingData(ActView, from, o.ID)),
	})}
}
