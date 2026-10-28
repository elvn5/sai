package screens

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
)

// The organizer's room: callback data "g:<action>[:<arg>…]".
const (
	OrgHome = keyboards.OrganizerPrefix + "home"
	// "g:r:<tour>" the roster; "g:rn:<tour>" the same from a notification,
	// as a message of its own.
	OrgRoster       = "r"
	OrgRosterNotice = "rn"
	// "g:a:<tour>:<participant id prefix>" marks one arrived or not;
	// "g:all:<tour>" marks everyone.
	OrgArrival    = "a"
	OrgArrivedAll = "all"
	// A message to the participants: write it, send it, drop it.
	OrgWrite      = "m"
	OrgSend       = "ms"
	OrgWriteAbort = "mx"
	// "g:t:<index>" one of the organizer's tours, "g:tp:<index>:<day>" its
	// program, "g:tb:<index>" its bookings.
	OrgTour     = "t"
	OrgProgram  = "tp"
	OrgBookings = "tb"
)

const (
	// The organizer marks arrivals from a day before the start until a
	// week after it (as the listings service takes them).
	checkInBefore = 24 * time.Hour
	checkInAfter  = 7 * 24 * time.Hour
	// A participant's button carries this much of their id: callback data
	// is 64 bytes at most.
	idPrefixLen = 8
	// What a message to the participants can hold.
	BroadcastMaxLen = 2000
	bookingsShown   = 20
)

func OrgData(action string, args ...string) string {
	return keyboards.OrganizerPrefix + strings.Join(append([]string{action}, args...), ":")
}

// ParticipantKey is the part of a participant's id their button carries.
func ParticipantKey(id string) string { return id[:min(len(id), idPrefixLen)] }

// CanCheckIn: the organizer can mark arrivals from a day before the
// meeting until a week after.
func CanCheckIn(t api.Tour, now time.Time) bool {
	starts := t.StartsAt()
	return !now.Before(starts.Add(-checkInBefore)) && !now.After(starts.Add(checkInAfter))
}

// ActiveTours are the organizer's tours that still need them (not more
// than a week since the start), soonest first; the rest keep their order.
func ActiveTours(list []api.Tour, now time.Time) (active, rest []api.Tour) {
	for _, t := range list {
		if now.Before(t.StartsAt().Add(checkInAfter)) {
			active = append(active, t)
		} else {
			rest = append(rest, t)
		}
	}
	sort.SliceStable(active, func(i, j int) bool { return active[i].StartsAt().Before(active[j].StartsAt()) })
	return active, rest
}

func arrivedCount(t api.Tour) int {
	n := 0
	for _, p := range t.Roster {
		if p.ArrivedAt != nil {
			n++
		}
	}
	return n
}

func publishedLabel(l i18n.Lang, t api.Tour) string {
	if t.IsPublished {
		return l.T("🟢 опубликован")
	}
	return l.T("📝 черновик")
}

// OrganizerHome is the organizer's room: the tours they run (tours[i]
// opens as "g:t:<i>"), with who signed up and, around the start, who came.
func OrganizerHome(l i18n.Lang, tours []api.Tour, now time.Time) Screen {
	var b strings.Builder
	b.WriteString(l.T("🧑‍✈️ <b>Кабинет организатора</b>\nУчастники, отметка на сборе и ваши туры."))
	var buttons [][]telego.InlineKeyboardButton
	if len(tours) == 0 {
		b.WriteString("\n\n" + l.T("У вас пока нет туров. Добавьте тур на сайте — здесь появятся его участники, отметка прибывших и рассылка."))
	} else {
		b.WriteString("\n\n" + l.T("🧭 <b>Ваши туры</b>"))
	}
	for i, t := range tours {
		meta := []string{}
		if r := dateRange(l, t.StartDate, t.EndDate); r != "" {
			meta = append(meta, r)
		}
		meta = append(meta, "👥 "+placesLine(l, t))
		if CanCheckIn(t, now) && len(t.Roster) > 0 {
			meta = append(meta, l.F("✅ на месте %d из %d", arrivedCount(t), len(t.Roster)))
		}
		if !t.IsPublished {
			meta = append(meta, publishedLabel(l, t))
		}
		fmt.Fprintf(&b, "\n\n<b>%d. %s</b>\n%s", i+1, esc(t.Title), strings.Join(meta, " · "))
		buttons = append(buttons, []telego.InlineKeyboardButton{
			keyboards.Button(fmt.Sprintf("%d · %s", i+1, clip(t.Title, 40)), OrgData(OrgTour, strconv.Itoa(i))),
		})
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("⬅️ Мои туры"), ListData(FromMine))})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: noPreview()}
}

// personLink names a participant, linking to their Telegram profile when
// their chat is known.
func personLink(l i18n.Lang, p api.Person, n int, chats map[string]int64) string {
	name := p.Name()
	if name == "" {
		name = l.F("Участник %d", n)
	}
	text := esc(name)
	if chat, ok := chats[p.ID]; ok {
		text = fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, chat, text)
	}
	if p.Username != nil && *p.Username != "" && name != "@"+*p.Username {
		text += " · @" + esc(*p.Username)
	}
	return text
}

func buttonName(l i18n.Lang, p api.Person, n int) string {
	if name := p.Name(); name != "" {
		return clip(name, 20)
	}
	return l.F("Участник %d", n)
}

// Roster is a tour's participants for its organizer: who they are
// (linked to their Telegram when chats knows it), who has arrived, and a
// message to all of them.
func Roster(l i18n.Lang, t api.Tour, chats map[string]int64, links Links, now time.Time) Screen {
	checkIn := CanCheckIn(t, now)
	var b strings.Builder
	b.WriteString(l.F("📋 <b>Участники · %s</b>", esc(t.Title)) + "\n" + When(l, t.StartsAt()))
	if t.MeetingPoint != nil && *t.MeetingPoint != "" {
		b.WriteString(" · 📍 " + esc(*t.MeetingPoint))
	}
	b.WriteString("\n👥 " + placesLine(l, t))
	if checkIn && len(t.Roster) > 0 {
		b.WriteString(" · " + l.F("✅ на месте %d из %d", arrivedCount(t), len(t.Roster)))
	}
	if t.Waitlist > 0 {
		b.WriteString("\n⏳ " + l.F("В листе ожидания: %d", t.Waitlist))
	}

	if len(t.Roster) == 0 {
		b.WriteString("\n\n" + l.T("Пока никто не записался. Позовите друзей — участники появятся здесь."))
	} else {
		b.WriteString("\n")
	}
	var marks []telego.InlineKeyboardButton
	for i, p := range t.Roster {
		mark := "•"
		if checkIn {
			mark = "⬜"
			if p.ArrivedAt != nil {
				mark = "✅"
			}
		}
		fmt.Fprintf(&b, "\n%d. %s %s · %s", i+1, mark, personLink(l, p.Person, i+1, chats),
			l.F("с %s", l.DayMonth(p.JoinedAt.In(bishkek))))
		if checkIn {
			marks = append(marks, keyboards.Button(mark+" "+buttonName(l, p.Person, i+1), OrgData(OrgArrival, t.ID, ParticipantKey(p.ID))))
		}
	}

	switch {
	case checkIn && len(t.Roster) > 0:
		b.WriteString("\n\n" + l.T("Нажмите на участника, когда он придёт на сбор."))
	case len(t.Roster) > 0 && now.Before(t.StartsAt()):
		b.WriteString("\n\n" + l.T("Отмечать прибывших можно будет за сутки до сбора."))
	}

	var buttons [][]telego.InlineKeyboardButton
	for i := 0; i < len(marks); i += 2 {
		buttons = append(buttons, marks[i:min(i+2, len(marks))])
	}
	if checkIn && arrivedCount(t) < len(t.Roster) {
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("✅ Все на месте"), OrgData(OrgArrivedAll, t.ID))})
	}
	if len(t.Roster) > 0 {
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("📣 Написать участникам"), OrgData(OrgWrite, t.ID))})
	}
	if IsUpcoming(t, now) && !t.Full() {
		if share := inviteFriends(l, t, links); share != nil {
			buttons = append(buttons, []telego.InlineKeyboardButton{*share})
		}
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{
		keyboards.Button(l.T("🧭 Карточка тура"), TripData(ActView, FromMine, t.ID)), keyboards.Button(l.T("⬅️ Кабинет"), OrgHome),
	})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: noPreview()}
}

func people(l i18n.Lang, n int) string {
	return fmt.Sprintf("%d %s", n, l.N(n, "человек", "человека", "человек"))
}

// BroadcastPrompt asks the organizer for a message to the participants.
func BroadcastPrompt(l i18n.Lang, t api.Tour) Screen {
	text := l.T("📣 <b>Сообщение участникам</b>") + "\n«" + esc(t.Title) + "» · " + people(l, len(t.Roster)) + "\n\n" +
		l.T("Напишите сообщение — бот перешлёт его всем, кто записался. Например: <i>«Сбор переносится на 06:30, ждём у южного входа»</i>.")
	return Screen{Text: text, Preview: noPreview(), Markup: rows([]telego.InlineKeyboardButton{
		keyboards.Button(l.T("Отмена"), OrgData(OrgWriteAbort, t.ID)),
	})}
}

// BroadcastPreview shows the message before it goes out.
func BroadcastPreview(l i18n.Lang, t api.Tour, message string) Screen {
	text := l.F("Отправить участникам «%s» (%s)?", esc(t.Title), people(l, len(t.Roster))) +
		"\n\n<blockquote>" + esc(message) + "</blockquote>\n\n" + l.T("<i>Чтобы исправить, просто напишите сообщение заново.</i>")
	return Screen{Text: text, Preview: noPreview(), Markup: rows([]telego.InlineKeyboardButton{
		keyboards.Button(l.T("📣 Отправить"), OrgData(OrgSend, t.ID)), keyboards.Button(l.T("Отмена"), OrgData(OrgWriteAbort, t.ID)),
	})}
}

// BroadcastMessage is the organizer's message as a participant gets it.
func BroadcastMessage(l i18n.Lang, t api.Tour, message string) Screen {
	var b strings.Builder
	b.WriteString(l.T("📣 <b>Сообщение от организатора</b>"))
	if t.Organizer != nil && t.Organizer.Name() != "" {
		b.WriteString(" · " + esc(t.Organizer.Name()))
	}
	b.WriteString("\n" + l.F("Тур «%s», %s", esc(t.Title), When(l, t.StartsAt())) + "\n\n" + esc(message))
	return Screen{Text: b.String(), Preview: noPreview(), Markup: rows([]telego.InlineKeyboardButton{
		keyboards.Button(l.T("Открыть тур"), TripData(ActView, FromNotice, t.ID)),
	})}
}

// BroadcastSent tells the organizer how many got the message.
func BroadcastSent(l i18n.Lang, t api.Tour, sent, total int) Screen {
	text := l.F("✅ Сообщение отправлено: %d из %d.", sent, total)
	if sent < total {
		missed := total - sent
		text += "\n\n" + l.F("Не дошло до %d %s — у них не подключён Telegram или бот заблокирован. Свяжитесь с ними сами.",
			missed, l.N(missed, "человека", "человек", "человек"))
	}
	return Screen{Text: text, Preview: noPreview(), Markup: rows([]telego.InlineKeyboardButton{
		keyboards.Button(l.T("⬅️ К участникам"), OrgData(OrgRoster, t.ID)),
	})}
}

// TourBookings are the paid bookings of a tour.
func TourBookings(orders []api.Order, tourID string) []api.Order {
	var out []api.Order
	for _, o := range orders {
		if o.Kind == "tour" && o.ItemID == tourID {
			out = append(out, o)
		}
	}
	return out
}

// paidBookings counts the bookings still paid (not refunded) and the
// people they bring.
func paidBookings(orders []api.Order) (bookings, people int) {
	for _, o := range orders {
		if o.Status == "paid" {
			bookings++
			people += o.Quantity
		}
	}
	return bookings, people
}

// OrganizerTour is one of the organizer's tours: dates, state, last
// month's numbers and bookings; stats is nil when unknown.
func OrganizerTour(l i18n.Lang, t api.Tour, index int, stats *api.StatCounts, bookings []api.Order, links Links) Screen {
	var b strings.Builder
	fmt.Fprintf(&b, "🧭 <b>%s</b>", esc(t.Title))
	var when []string
	if r := dateRange(l, t.StartDate, t.EndDate); r != "" {
		when = append(when, r)
	}
	if t.DurationDays != nil && *t.DurationDays > 0 {
		when = append(when, days(l, *t.DurationDays))
	}
	if len(when) > 0 {
		b.WriteString("\n📅 " + strings.Join(when, " · "))
	}
	if t.Location != nil && *t.Location != "" {
		b.WriteString("\n📍 " + esc(*t.Location))
	}
	var terms []string
	if t.Price != nil {
		terms = append(terms, "💳 "+som(l, *t.Price))
	}
	terms = append(terms, "👥 "+placesLine(l, t))
	if len(terms) > 0 {
		b.WriteString("\n" + strings.Join(terms, " · "))
	}
	if t.IsPublished {
		b.WriteString("\n" + l.T("🟢 Опубликован"))
	} else {
		b.WriteString("\n" + l.T("📝 Не опубликован — на сайте его не видно"))
	}
	b.WriteString(reviews(l, t))
	if stats != nil {
		fmt.Fprintf(&b, "\n\n📊 %s: 👁 %d %s · 📞 %d %s", l.T("<b>За 30 дней</b>"), stats.Views,
			l.N(stats.Views, "просмотр", "просмотра", "просмотров"), stats.Contacts,
			l.N(stats.Contacts, "обращение", "обращения", "обращений"))
	}
	if paid, n := paidBookings(bookings); paid > 0 {
		b.WriteString("\n🎫 " + l.F("Онлайн-заявок: %d", paid) + " · " + people(l, n))
	}

	idx := strconv.Itoa(index)
	var buttons [][]telego.InlineKeyboardButton
	buttons = append(buttons, []telego.InlineKeyboardButton{
		keyboards.Button(l.F("📋 Участники · %d", t.Taken), OrgData(OrgRoster, t.ID)),
	})
	if len(t.Program) > 0 {
		buttons = append(buttons, []telego.InlineKeyboardButton{
			keyboards.Button("📖 "+l.T("Программа")+" · "+days(l, len(t.Program)), OrgData(OrgProgram, idx, "0")),
		})
	}
	if len(bookings) > 0 {
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("🎫 Заявки"), OrgData(OrgBookings, idx))})
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("⬅️ Кабинет"), OrgHome)})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: coverPreview(links.Picture(t.CoverImageURL))}
}

// OrganizerProgram is one day of a tour's program, paged.
func OrganizerProgram(l i18n.Lang, t api.Tour, index, day int) Screen {
	n := len(t.Program)
	idx := strconv.Itoa(index)
	back := []telego.InlineKeyboardButton{keyboards.Button(l.T("⬅️ К туру"), OrgData(OrgTour, idx))}
	if n == 0 {
		return Screen{Text: l.F("📖 <b>Программа · %s</b>", esc(t.Title)) + "\n\n" + l.T("Программа пока не заполнена."),
			Markup: rows(back), Preview: noPreview()}
	}
	day = min(max(day, 0), n-1)
	d := t.Program[day]
	var b strings.Builder
	b.WriteString(l.F("📖 <b>Программа · %s</b>", esc(t.Title)) + "\n\n<b>" + l.F("День %d", day+1))
	if d.Title != "" {
		b.WriteString(". " + esc(d.Title))
	}
	b.WriteString("</b>")
	if text := strings.TrimSpace(d.Text); text != "" {
		b.WriteString("\n" + esc(clipLines(text, 3500)))
	}
	var buttons [][]telego.InlineKeyboardButton
	if n > 1 {
		buttons = append(buttons, keyboards.Pager(day, n, func(page int) string { return OrgData(OrgProgram, idx, strconv.Itoa(page)) }))
	}
	buttons = append(buttons, back)
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: noPreview()}
}

// clipLines shortens text to n characters, keeping its line breaks.
func clipLines(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimRight(string(r[:n]), " \n") + "…"
}

// OrganizerBookings lists a tour's paid bookings with the contacts left.
func OrganizerBookings(l i18n.Lang, t api.Tour, index int, bookings []api.Order) Screen {
	var b strings.Builder
	b.WriteString(l.F("🎫 <b>Заявки · %s</b>", esc(t.Title)))
	if len(bookings) == 0 {
		b.WriteString("\n\n" + l.T("Онлайн-заявок пока нет."))
	}
	sorted := append([]api.Order(nil), bookings...)
	sort.SliceStable(sorted, func(i, j int) bool { return deref(sorted[i].ItemDate) < deref(sorted[j].ItemDate) })
	date := "\x00"
	for i, o := range sorted {
		if i == bookingsShown {
			b.WriteString("\n\n" + l.F("…и ещё %d — все заявки в кабинете на сайте.", len(sorted)-bookingsShown))
			break
		}
		if d := deref(o.ItemDate); d != date {
			date = d
			label := dateRange(l, d, d)
			if label == "" {
				label = l.T("Без даты")
			}
			b.WriteString("\n\n<b>" + label + "</b>")
		}
		line := fmt.Sprintf("\n%d. %s · %s · %s · %s", i+1, esc(o.ContactName), people(l, o.Quantity),
			esc(o.ContactPhone), som(l, o.Amount))
		if o.Status == "refunded" {
			line += " · " + l.T("↩️ возврат")
		}
		b.WriteString(line)
		if o.Comment != nil && *o.Comment != "" {
			b.WriteString("\n<i>" + esc(clip(*o.Comment, 200)) + "</i>")
		}
	}
	return Screen{Text: b.String(), Preview: noPreview(), Markup: rows([]telego.InlineKeyboardButton{
		keyboards.Button(l.T("⬅️ К туру"), OrgData(OrgTour, strconv.Itoa(index))),
	})}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
