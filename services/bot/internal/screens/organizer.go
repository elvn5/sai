package screens

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
)

// The organizer's room: callback data "g:<action>[:<arg>…]".
const (
	OrgHome = keyboards.OrganizerPrefix + "home"
	// "g:r:<outing>" the roster; "g:rn:<outing>" the same from a
	// notification, as a message of its own.
	OrgRoster       = "r"
	OrgRosterNotice = "rn"
	// "g:a:<outing>:<participant id prefix>" marks one arrived or not;
	// "g:all:<outing>" marks everyone.
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
	// week after it (as the trails service takes them).
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

// CanCheckIn: the organizer can mark arrivals of an open outing from a day
// before its start until a week after.
func CanCheckIn(o api.Outing, now time.Time) bool {
	return o.Status == "open" && !now.Before(o.StartsAt.Add(-checkInBefore)) && !now.After(o.StartsAt.Add(checkInAfter))
}

// OrganizedOutings are the outings the user runs that still need them:
// open, not more than a week since the start; soonest first.
func OrganizedOutings(list []api.Outing, now time.Time) []api.Outing {
	var out []api.Outing
	for _, o := range list {
		if o.IsOrganizer && o.Status == "open" && now.Before(o.StartsAt.Add(checkInAfter)) {
			out = append(out, o)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartsAt.Before(out[j].StartsAt) })
	return out
}

func arrivedCount(o api.Outing) int {
	n := 0
	for _, p := range o.Roster {
		if p.ArrivedAt != nil {
			n++
		}
	}
	return n
}

func publishedLabel(t api.Tour) string {
	if t.IsPublished {
		return "🟢 опубликован"
	}
	return "📝 черновик"
}

// OrganizerHome is the organizer's room: the outings they run and their
// tours.
func OrganizerHome(outings []api.Outing, tours []api.Tour, now time.Time) Screen {
	var b strings.Builder
	b.WriteString("🧑‍✈️ <b>Кабинет организатора</b>\nУчастники, отметка на сборе и ваши туры.")
	var buttons [][]telego.InlineKeyboardButton
	if len(outings) == 0 && len(tours) == 0 {
		b.WriteString("\n\nВы пока ничего не организуете. Соберите выход — здесь появятся его участники, " +
			"отметка прибывших и рассылка.")
	}
	n := 0
	if len(outings) > 0 {
		b.WriteString("\n\n🥾 <b>Ваши выходы</b>")
	}
	for _, o := range outings {
		n++
		meta := []string{fmt.Sprintf("👥 %d/%d", o.Taken, o.Seats)}
		if CanCheckIn(o, now) && o.Taken > 0 {
			meta = append(meta, fmt.Sprintf("✅ на месте %d из %d", arrivedCount(o), o.Taken))
		}
		b.WriteString("\n\n" + outingLine(n, o, meta...))
		buttons = append(buttons, []telego.InlineKeyboardButton{
			keyboards.Button(fmt.Sprintf("%d · %s", n, clip(o.Title, 40)), OrgData(OrgRoster, o.ID)),
		})
	}
	if len(tours) > 0 {
		b.WriteString("\n\n🧭 <b>Ваши туры</b>")
	}
	for i, t := range tours {
		n++
		meta := []string{publishedLabel(t)}
		if r := dateRange(t.StartDate, t.EndDate); r != "" {
			meta = append([]string{r}, meta...)
		}
		fmt.Fprintf(&b, "\n\n<b>%d. %s</b>\n%s", n, esc(t.Title), strings.Join(meta, " · "))
		buttons = append(buttons, []telego.InlineKeyboardButton{
			keyboards.Button(fmt.Sprintf("%d · %s", n, clip(t.Title, 40)), OrgData(OrgTour, strconv.Itoa(i))),
		})
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{
		keyboards.Button("➕ Собрать выход", NewOutingData), keyboards.Button("⬅️ Мои выходы", ListData(FromMine)),
	})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: noPreview()}
}

// personLink names a participant, linking to their Telegram profile when
// their chat is known.
func personLink(p api.Person, n int, chats map[string]int64) string {
	name := p.Name()
	if name == "" {
		name = fmt.Sprintf("Участник %d", n)
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

func buttonName(p api.Person, n int) string {
	if name := p.Name(); name != "" {
		return clip(name, 20)
	}
	return fmt.Sprintf("Участник %d", n)
}

// Roster is an outing's participants for its organizer: who they are
// (linked to their Telegram when chats knows it), who has arrived, and a
// message to all of them.
func Roster(o api.Outing, chats map[string]int64, links Links, now time.Time) Screen {
	checkIn := CanCheckIn(o, now)
	var b strings.Builder
	fmt.Fprintf(&b, "📋 <b>Участники · %s</b>\n%s · 📍 %s", esc(o.Title), When(o.StartsAt), esc(o.MeetingPoint))
	fmt.Fprintf(&b, "\n👥 %d из %d мест", len(o.Roster), o.Seats)
	if checkIn && len(o.Roster) > 0 {
		fmt.Fprintf(&b, " · ✅ на месте %d из %d", arrivedCount(o), len(o.Roster))
	}

	if len(o.Roster) == 0 {
		b.WriteString("\n\nПока никто не записался. Позовите друзей — участники появятся здесь.")
	} else {
		b.WriteString("\n")
	}
	var marks []telego.InlineKeyboardButton
	for i, p := range o.Roster {
		mark := "•"
		if checkIn {
			mark = "⬜"
			if p.ArrivedAt != nil {
				mark = "✅"
			}
		}
		fmt.Fprintf(&b, "\n%d. %s %s · с %d %s", i+1, mark, personLink(p.Person, i+1, chats),
			p.JoinedAt.In(bishkek).Day(), months[p.JoinedAt.In(bishkek).Month()-1])
		if checkIn {
			marks = append(marks, keyboards.Button(mark+" "+buttonName(p.Person, i+1), OrgData(OrgArrival, o.ID, ParticipantKey(p.ID))))
		}
	}

	switch {
	case o.Status == "cancelled":
		b.WriteString("\n\n❌ Выход отменён")
	case checkIn && len(o.Roster) > 0:
		b.WriteString("\n\nНажмите на участника, когда он придёт на сбор.")
	case len(o.Roster) > 0 && now.Before(o.StartsAt):
		b.WriteString("\n\nОтмечать прибывших можно будет за сутки до сбора.")
	}

	var buttons [][]telego.InlineKeyboardButton
	for i := 0; i < len(marks); i += 2 {
		buttons = append(buttons, marks[i:min(i+2, len(marks))])
	}
	if checkIn && arrivedCount(o) < len(o.Roster) {
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("✅ Все на месте", OrgData(OrgArrivedAll, o.ID))})
	}
	if o.Status == "open" && len(o.Roster) > 0 {
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("📣 Написать участникам", OrgData(OrgWrite, o.ID))})
	}
	if isUpcoming(o, now) && o.Taken < o.Seats {
		if share := links.Share("📤 Позвать друзей", OutingDeepLink(o.ID), "Пойдём в горы: «"+o.Title+"», "+When(o.StartsAt)); share != nil {
			buttons = append(buttons, []telego.InlineKeyboardButton{*share})
		}
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{
		keyboards.Button("🥾 Карточка выхода", OutingData(ActView, FromMine, o.ID)), keyboards.Button("⬅️ Кабинет", OrgHome),
	})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: noPreview()}
}

func people(n int) string {
	return fmt.Sprintf("%d %s", n, plural(n, "человек", "человека", "человек"))
}

// BroadcastPrompt asks the organizer for a message to the participants.
func BroadcastPrompt(o api.Outing) Screen {
	text := fmt.Sprintf("📣 <b>Сообщение участникам</b>\n«%s» · %s\n\n"+
		"Напишите сообщение — бот перешлёт его всем, кто записался. Например: "+
		"<i>«Сбор переносится на 06:30, ждём у южного входа»</i>.", esc(o.Title), people(len(o.Participants)))
	return Screen{Text: text, Preview: noPreview(), Markup: rows([]telego.InlineKeyboardButton{
		keyboards.Button("Отмена", OrgData(OrgWriteAbort, o.ID)),
	})}
}

// BroadcastPreview shows the message before it goes out.
func BroadcastPreview(o api.Outing, message string) Screen {
	text := fmt.Sprintf("Отправить участникам «%s» (%s)?\n\n<blockquote>%s</blockquote>\n\n"+
		"<i>Чтобы исправить, просто напишите сообщение заново.</i>", esc(o.Title), people(len(o.Participants)), esc(message))
	return Screen{Text: text, Preview: noPreview(), Markup: rows([]telego.InlineKeyboardButton{
		keyboards.Button("📣 Отправить", OrgData(OrgSend, o.ID)), keyboards.Button("Отмена", OrgData(OrgWriteAbort, o.ID)),
	})}
}

// BroadcastMessage is the organizer's message as a participant gets it.
func BroadcastMessage(o api.Outing, message string) Screen {
	var b strings.Builder
	b.WriteString("📣 <b>Сообщение от организатора</b>")
	if name := o.Organizer.Name(); name != "" {
		b.WriteString(" · " + esc(name))
	}
	fmt.Fprintf(&b, "\nВыход «%s», %s\n\n%s", esc(o.Title), When(o.StartsAt), esc(message))
	return Screen{Text: b.String(), Preview: noPreview(), Markup: rows([]telego.InlineKeyboardButton{
		keyboards.Button("Открыть выход", OutingData(ActView, FromNotice, o.ID)),
	})}
}

// BroadcastSent tells the organizer how many got the message.
func BroadcastSent(o api.Outing, sent, total int) Screen {
	text := fmt.Sprintf("✅ Сообщение отправлено: %d из %d.", sent, total)
	if sent < total {
		missed := total - sent
		text += fmt.Sprintf("\n\nНе дошло до %d %s — у них не подключён Telegram или бот заблокирован. "+
			"Свяжитесь с ними сами.", missed, plural(missed, "человека", "человек", "человек"))
	}
	return Screen{Text: text, Preview: noPreview(), Markup: rows([]telego.InlineKeyboardButton{
		keyboards.Button("⬅️ К участникам", OrgData(OrgRoster, o.ID)),
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
func OrganizerTour(t api.Tour, index int, stats *api.StatCounts, bookings []api.Order, links Links) Screen {
	var b strings.Builder
	fmt.Fprintf(&b, "🧭 <b>%s</b>", esc(t.Title))
	var when []string
	if r := dateRange(t.StartDate, t.EndDate); r != "" {
		when = append(when, r)
	}
	if t.DurationDays != nil && *t.DurationDays > 0 {
		when = append(when, days(*t.DurationDays))
	}
	if len(when) > 0 {
		b.WriteString("\n📅 " + strings.Join(when, " · "))
	}
	if t.Location != nil && *t.Location != "" {
		b.WriteString("\n📍 " + esc(*t.Location))
	}
	var terms []string
	if t.Price != nil {
		terms = append(terms, "💳 "+money(*t.Price)+" сом")
	}
	if t.Capacity != nil {
		terms = append(terms, fmt.Sprintf("👥 мест: %d", *t.Capacity))
	}
	if len(terms) > 0 {
		b.WriteString("\n" + strings.Join(terms, " · "))
	}
	if t.IsPublished {
		b.WriteString("\n🟢 Опубликован")
	} else {
		b.WriteString("\n📝 Не опубликован — на сайте его не видно")
	}
	if t.ReviewCount > 0 {
		fmt.Fprintf(&b, "\n⭐ %.1f · %d %s", t.RatingAverage, t.ReviewCount, plural(t.ReviewCount, "отзыв", "отзыва", "отзывов"))
	}
	if stats != nil {
		fmt.Fprintf(&b, "\n\n📊 <b>За 30 дней</b>: 👁 %d %s · 📞 %d %s", stats.Views,
			plural(stats.Views, "просмотр", "просмотра", "просмотров"), stats.Contacts,
			plural(stats.Contacts, "обращение", "обращения", "обращений"))
	}
	if paid, n := paidBookings(bookings); paid > 0 {
		fmt.Fprintf(&b, "\n🎫 Онлайн-заявок: %d · %s", paid, people(n))
	}

	idx := strconv.Itoa(index)
	var buttons [][]telego.InlineKeyboardButton
	if len(t.Program) > 0 {
		buttons = append(buttons, []telego.InlineKeyboardButton{
			keyboards.Button(fmt.Sprintf("📖 Программа · %s", days(len(t.Program))), OrgData(OrgProgram, idx, "0")),
		})
	}
	if len(bookings) > 0 {
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("🎫 Заявки", OrgData(OrgBookings, idx))})
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("⬅️ Кабинет", OrgHome)})
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: coverPreview(links.Picture(t.CoverImageURL))}
}

// OrganizerProgram is one day of a tour's program, paged.
func OrganizerProgram(t api.Tour, index, day int) Screen {
	n := len(t.Program)
	idx := strconv.Itoa(index)
	back := []telego.InlineKeyboardButton{keyboards.Button("⬅️ К туру", OrgData(OrgTour, idx))}
	if n == 0 {
		return Screen{Text: fmt.Sprintf("📖 <b>Программа · %s</b>\n\nПрограмма пока не заполнена.", esc(t.Title)),
			Markup: rows(back), Preview: noPreview()}
	}
	day = min(max(day, 0), n-1)
	d := t.Program[day]
	var b strings.Builder
	fmt.Fprintf(&b, "📖 <b>Программа · %s</b>\n\n<b>День %d", esc(t.Title), day+1)
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
func OrganizerBookings(t api.Tour, index int, bookings []api.Order) Screen {
	var b strings.Builder
	fmt.Fprintf(&b, "🎫 <b>Заявки · %s</b>", esc(t.Title))
	if len(bookings) == 0 {
		b.WriteString("\n\nОнлайн-заявок пока нет.")
	}
	sorted := append([]api.Order(nil), bookings...)
	sort.SliceStable(sorted, func(i, j int) bool { return deref(sorted[i].ItemDate) < deref(sorted[j].ItemDate) })
	date := "\x00"
	for i, o := range sorted {
		if i == bookingsShown {
			fmt.Fprintf(&b, "\n\n…и ещё %d — все заявки в кабинете на сайте.", len(sorted)-bookingsShown)
			break
		}
		if d := deref(o.ItemDate); d != date {
			date = d
			label := dateRange(d, d)
			if label == "" {
				label = "Без даты"
			}
			b.WriteString("\n\n<b>" + label + "</b>")
		}
		line := fmt.Sprintf("\n%d. %s · %s · %s · %s сом", i+1, esc(o.ContactName), people(o.Quantity),
			esc(o.ContactPhone), money(o.Amount))
		if o.Status == "refunded" {
			line += " · ↩️ возврат"
		}
		b.WriteString(line)
		if o.Comment != nil && *o.Comment != "" {
			b.WriteString("\n<i>" + esc(clip(*o.Comment, 200)) + "</i>")
		}
	}
	return Screen{Text: b.String(), Preview: noPreview(), Markup: rows([]telego.InlineKeyboardButton{
		keyboards.Button("⬅️ К туру", OrgData(OrgTour, strconv.Itoa(index))),
	})}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
