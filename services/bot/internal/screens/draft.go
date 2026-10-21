package screens

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/drafts"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
)

// Draft callback actions, "d:<action>[:<arg>]".
const (
	DraftBack    = "back"
	DraftCancel  = "cancel"
	DraftReview  = "review"
	DraftFields  = "fields"
	DraftSave    = "save"
	DraftSkip    = "skip"    // keep the value and go on
	DraftClear   = "clear"   // drop an optional value and go on
	DraftGo      = "go"      // change a step: "d:go:<step>"
	DraftTrail   = "trail"   // "d:trail:<index>" or "d:trail:none"
	DraftTitle   = "title"   // "d:title:trail" names it after the route
	DraftMonth   = "month"   // "d:month:2026-11"
	DraftDate    = "date"    // "d:date:2026-10-24"
	DraftTime    = "time"    // "d:time:0600"
	DraftReturn  = "ret"     // "d:ret:1900", "d:ret:+1800" (next day), "d:ret:none"
	DraftMove    = "tr"      // "d:tr:car"
	DraftSeats   = "seats"   // "d:seats:6"
	DraftCost    = "cost"    // "d:cost:0", "d:cost:none"
	DraftLevel   = "lvl"     // "d:lvl:beginner"
	DraftContact = "contact" // "d:contact:me": the user's @username
	DraftNone    = "none"
)

func draftData(action string, arg ...string) string {
	return keyboards.DraftPrefix + strings.Join(append([]string{action}, arg...), ":")
}

var (
	monthsNominative = []string{
		"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
		"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
	}
	startTimes  = []string{"05:00", "05:30", "06:00", "06:30", "07:00", "07:30", "08:00", "08:30", "09:00", "10:00", "11:00", "12:00"}
	returnTimes = []string{"14:00", "16:00", "18:00", "19:00", "20:00", "22:00"}
	seatChoices = []int{2, 3, 4, 5, 6, 8, 10, 12, 15, 20, 25, 30}
	// The draft's fields, as the list of what to change shows them.
	draftFields = []struct {
		step  drafts.Step
		label string
	}{
		{drafts.StepTrail, "🏔 Маршрут"}, {drafts.StepTitle, "✏️ Название"},
		{drafts.StepDate, "📅 Дата"}, {drafts.StepTime, "🕕 Время сбора"},
		{drafts.StepReturn, "🏁 Возвращение"}, {drafts.StepMeeting, "📍 Место сбора"},
		{drafts.StepTransport, "🚗 Транспорт"}, {drafts.StepSeats, "👥 Места"},
		{drafts.StepCost, "💳 Стоимость"}, {drafts.StepLevel, "📊 Уровень"},
		{drafts.StepDescription, "📝 Описание"}, {drafts.StepContact, "💬 Контакт"},
	}
)

// longDate is "суббота, 24 октября".
func longDate(date string) string {
	t, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return ""
	}
	names := []string{"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"}
	return fmt.Sprintf("%s, %d %s", names[t.Weekday()], t.Day(), monthsFull[t.Month()-1])
}

// marked puts a check on the chosen option.
func marked(label string, chosen bool) string {
	if chosen {
		return "✅ " + label
	}
	return label
}

// DraftScreen is the draft's current step (or its review).
func DraftScreen(d drafts.Draft, now time.Time) Screen {
	switch d.Step {
	case drafts.StepReview:
		return draftReview(d)
	case drafts.StepFields:
		return draftFieldList(d)
	}

	var b strings.Builder
	switch {
	case d.OutingID != "":
		b.WriteString("✏️ <b>Изменение выхода</b>")
	case d.Reviewing:
		b.WriteString("➕ <b>Новый выход</b>")
	default:
		fmt.Fprintf(&b, "➕ <b>Новый выход</b> · шаг %d из %d", drafts.Number(d.Step), len(drafts.Steps))
	}
	b.WriteString("\n\n")
	var buttons [][]telego.InlineKeyboardButton
	current := func(value string) {
		if value != "" && d.Reviewing {
			b.WriteString("\n\nСейчас: <b>" + esc(clip(value, 200)) + "</b>")
		}
	}
	keep := func(value string) {
		if value != "" && !d.Reviewing {
			buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("➡️ Оставить «"+clip(value, 30)+"»", draftData(DraftSkip))})
		}
	}

	switch d.Step {
	case drafts.StepTrail:
		b.WriteString("🏔 <b>Куда идём?</b>\nВыберите маршрут из каталога или напишите его название — я поищу.")
		switch {
		case d.Search != "" && len(d.Trails) == 0:
			b.WriteString("\n\nПо «" + esc(d.Search) + "» ничего не нашлось. Попробуйте другое название — или идите без маршрута из каталога.")
		case d.Search != "":
			b.WriteString("\n\nНашлось по «" + esc(d.Search) + "»:")
		}
		current(d.TrailName)
		for i := 0; i < len(d.Trails); i += 2 {
			row := []telego.InlineKeyboardButton{keyboards.Button(marked(clip(d.Trails[i].Name, 28), d.Trails[i].ID == d.TrailID), draftData(DraftTrail, strconv.Itoa(i)))}
			if i+1 < len(d.Trails) {
				row = append(row, keyboards.Button(marked(clip(d.Trails[i+1].Name, 28), d.Trails[i+1].ID == d.TrailID), draftData(DraftTrail, strconv.Itoa(i+1))))
			}
			buttons = append(buttons, row)
		}
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(marked("Без маршрута из каталога", d.TrailID == "" && d.Reviewing), draftData(DraftTrail, DraftNone))})

	case drafts.StepTitle:
		b.WriteString("✏️ <b>Как назовём выход?</b>\nНапишите название, например «Пик Учитель за один день».")
		current(d.Title)
		if d.TrailName != "" && d.TrailName != d.Title {
			buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("«"+clip(d.TrailName, 40)+"»", draftData(DraftTitle, DraftTrail))})
		}
		keep(d.Title)

	case drafts.StepDate:
		b.WriteString("📅 <b>Когда выходим?</b>\nВыберите день в календаре или напишите дату, например 24.10.")
		current(longDate(d.Date))
		buttons = append(buttons, calendar(d, now)...)

	case drafts.StepTime:
		b.WriteString("🕕 <b>Во сколько сбор?</b>\n" + upperFirst(longDate(d.Date)) + ". Выберите время или напишите своё, например 06:45.")
		current(d.Time)
		buttons = append(buttons, grid(startTimes, 4, func(t string) telego.InlineKeyboardButton {
			return keyboards.Button(marked(t, t == d.Time), draftData(DraftTime, strings.ReplaceAll(t, ":", "")))
		})...)

	case drafts.StepReturn:
		b.WriteString("🏁 <b>Когда вернётесь?</b>\nНе обязательно, но так участникам проще планировать день. Выберите или напишите, например «19:00» или «25.10 18:00».")
		if back, ok := d.Return(); ok {
			current(When(back))
		}
		var options []string
		for _, t := range returnTimes {
			if t > d.Time {
				options = append(options, t)
			}
		}
		buttons = append(buttons, grid(options, 3, func(t string) telego.InlineKeyboardButton {
			return keyboards.Button(marked(t, d.Returns == d.Date+"T"+t), draftData(DraftReturn, strings.ReplaceAll(t, ":", "")))
		})...)
		buttons = append(buttons, []telego.InlineKeyboardButton{
			keyboards.Button("Завтра 12:00", draftData(DraftReturn, "+1200")),
			keyboards.Button("Завтра 18:00", draftData(DraftReturn, "+1800")),
		}, []telego.InlineKeyboardButton{keyboards.Button(marked("Не указывать", d.Returns == "" && d.Reviewing), draftData(DraftReturn, DraftNone))})

	case drafts.StepMeeting:
		b.WriteString("📍 <b>Где сбор?</b>\nНапишите место, например «ТЦ Вефа, парковка у входа».")
		current(d.MeetingPoint)
		keep(d.MeetingPoint)

	case drafts.StepTransport:
		b.WriteString("🚗 <b>Как добираемся?</b>")
		for _, v := range drafts.Transports {
			buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(marked(TransportNames[v], v == d.Transport), draftData(DraftMove, v))})
		}

	case drafts.StepSeats:
		b.WriteString("👥 <b>Сколько человек возьмёте?</b>\nНе считая вас. Выберите или напишите число до 50.")
		if d.Taken > 0 {
			fmt.Fprintf(&b, "\nУже записались: %d.", d.Taken)
		}
		labels := make([]string, len(seatChoices))
		for i, n := range seatChoices {
			labels[i] = strconv.Itoa(n)
		}
		buttons = append(buttons, grid(labels, 4, func(n string) telego.InlineKeyboardButton {
			return keyboards.Button(marked(n, n == strconv.Itoa(d.Seats)), draftData(DraftSeats, n))
		})...)

	case drafts.StepCost:
		b.WriteString("💳 <b>Сколько с человека?</b>\nНа бензин, транспорт, общее снаряжение. Напишите сумму в сомах или выберите.")
		switch {
		case d.Cost != nil && *d.Cost > 0:
			current(money(*d.Cost) + " сом")
		case d.Cost != nil:
			current("бесплатно")
		}
		buttons = append(buttons, []telego.InlineKeyboardButton{
			keyboards.Button(marked("Бесплатно", d.Cost != nil && *d.Cost == 0), draftData(DraftCost, "0")),
			keyboards.Button(marked("Не указывать", d.Cost == nil && d.Reviewing), draftData(DraftCost, DraftNone)),
		})

	case drafts.StepLevel:
		b.WriteString("📊 <b>Для кого выход?</b>")
		for _, v := range drafts.Levels {
			buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(marked(LevelNames[v], v == d.Level), draftData(DraftLevel, v))})
		}

	case drafts.StepDescription:
		b.WriteString("📝 <b>Пара слов о выходе</b>\nМаршрут, темп, что взять с собой. До 2000 символов.")
		current(d.Description)
		if d.Description == "" {
			buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("Пропустить", draftData(DraftSkip))})
		} else {
			keep(d.Description)
			buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("🗑 Без описания", draftData(DraftClear))})
		}

	case drafts.StepContact:
		b.WriteString("💬 <b>Как с вами связаться?</b>\nКонтакт увидят только записавшиеся. Напишите @username или телефон.")
		current(d.Contact)
		if d.Username != "" && d.Contact != "@"+d.Username {
			buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("@"+d.Username, draftData(DraftContact, "me"))})
		}
		if d.Contact == "" {
			buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("Пропустить", draftData(DraftSkip))})
		} else {
			keep(d.Contact)
			buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("🗑 Без контакта", draftData(DraftClear))})
		}
	}

	buttons = append(buttons, draftNav(d))
	return Screen{Text: b.String(), Markup: rows(buttons...), Preview: noPreview()}
}

// draftNav is the way back and out.
func draftNav(d drafts.Draft) []telego.InlineKeyboardButton {
	cancel := keyboards.Button("✖️ Отмена", draftData(DraftCancel))
	switch {
	case d.Reviewing:
		return []telego.InlineKeyboardButton{keyboards.Button("⬅️ Назад", draftData(DraftFields)), cancel}
	case drafts.Number(d.Step) > 1:
		return []telego.InlineKeyboardButton{keyboards.Button("⬅️ Назад", draftData(DraftBack)), cancel}
	default:
		return []telego.InlineKeyboardButton{cancel}
	}
}

// grid lays buttons out cols to a row.
func grid(items []string, cols int, button func(string) telego.InlineKeyboardButton) [][]telego.InlineKeyboardButton {
	var out [][]telego.InlineKeyboardButton
	for i := 0; i < len(items); i += cols {
		var row []telego.InlineKeyboardButton
		for _, item := range items[i:min(i+cols, len(items))] {
			row = append(row, button(item))
		}
		out = append(out, row)
	}
	return out
}

// calendar is the draft's month, Monday first; days an outing can't start
// on show as dots.
func calendar(d drafts.Draft, now time.Time) [][]telego.InlineKeyboardButton {
	month, err := time.Parse("2006-01", d.Month)
	if err != nil {
		month, _ = time.Parse("2006-01", now.In(bishkek).Format("2006-01"))
	}
	earliest, latest := drafts.Earliest(now), drafts.Latest(now)
	noop := func(text string) telego.InlineKeyboardButton { return keyboards.Button(text, keyboards.Noop) }

	prev, next := noop(" "), noop(" ")
	if p := month.AddDate(0, -1, 0); p.Format("2006-01") >= earliest[:7] {
		prev = keyboards.Button("◀️", draftData(DraftMonth, p.Format("2006-01")))
	}
	if n := month.AddDate(0, 1, 0); n.Format("2006-01") <= latest[:7] {
		next = keyboards.Button("▶️", draftData(DraftMonth, n.Format("2006-01")))
	}
	out := [][]telego.InlineKeyboardButton{
		{prev, noop(fmt.Sprintf("%s %d", monthsNominative[month.Month()-1], month.Year())), next},
	}
	var header []telego.InlineKeyboardButton
	for _, w := range []string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"} {
		header = append(header, noop(w))
	}
	out = append(out, header)

	offset := (int(month.Weekday()) + 6) % 7 // Monday first
	var row []telego.InlineKeyboardButton
	for i := 0; i < offset; i++ {
		row = append(row, noop(" "))
	}
	for day := month; day.Month() == month.Month(); day = day.AddDate(0, 0, 1) {
		key := day.Format(time.DateOnly)
		label := strconv.Itoa(day.Day())
		switch {
		case key < earliest || key > latest:
			row = append(row, noop("·"))
		case key == d.Date:
			row = append(row, keyboards.Button("["+label+"]", draftData(DraftDate, key)))
		default:
			row = append(row, keyboards.Button(label, draftData(DraftDate, key)))
		}
		if len(row) == 7 {
			// Weeks already gone aren't shown.
			if day.Format(time.DateOnly) >= earliest {
				out = append(out, row)
			}
			row = nil
		}
	}
	if len(row) > 0 {
		for len(row) < 7 {
			row = append(row, noop(" "))
		}
		out = append(out, row)
	}
	return out
}

func upperFirst(s string) string {
	for i := range s {
		if i > 0 {
			return strings.ToUpper(s[:i]) + s[i:]
		}
	}
	return strings.ToUpper(s)
}

// draftReview shows the whole outing before it is published (or saved).
func draftReview(d drafts.Draft) Screen {
	var b strings.Builder
	if d.OutingID != "" {
		b.WriteString("✏️ <b>Изменение выхода</b>\n\n")
	} else {
		b.WriteString("👀 <b>Проверьте выход</b>\n\n")
	}
	b.WriteString(outingDetails(d.Outing()))
	save := "✅ Опубликовать"
	if d.OutingID != "" {
		save = "✅ Сохранить"
		b.WriteString("\n\nЕсли поменяются время, место сбора или маршрут, бот сообщит участникам.")
	} else {
		b.WriteString("\n\nВсё верно? Выход появится в ближайших — в боте и на сайте, а бот будет сообщать вам о каждом, кто запишется.")
	}
	return Screen{Text: b.String(), Preview: noPreview(), Markup: rows(
		[]telego.InlineKeyboardButton{keyboards.Button(save, draftData(DraftSave))},
		[]telego.InlineKeyboardButton{keyboards.Button("✏️ Изменить", draftData(DraftFields)), keyboards.Button("✖️ Отмена", draftData(DraftCancel))},
	)}
}

// draftFieldList asks which field to change.
func draftFieldList(d drafts.Draft) Screen {
	text := "✏️ <b>Что изменить?</b>\n\n" + outingDetails(d.Outing())
	var buttons [][]telego.InlineKeyboardButton
	for i := 0; i < len(draftFields); i += 2 {
		buttons = append(buttons, []telego.InlineKeyboardButton{
			keyboards.Button(draftFields[i].label, draftData(DraftGo, string(draftFields[i].step))),
			keyboards.Button(draftFields[i+1].label, draftData(DraftGo, string(draftFields[i+1].step))),
		})
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("⬅️ К проверке", draftData(DraftReview))})
	return Screen{Text: text, Markup: rows(buttons...), Preview: noPreview()}
}

// DraftGone answers a press on a draft that no longer exists.
const DraftGone = "Черновик устарел — начните заново в «🥾 Мои выходы»"

// DraftCancelled replaces a new outing's draft once dropped.
func DraftCancelled() Screen {
	return Screen{
		Text:    "Выход не создан. Собрать новый можно в любой момент.",
		Preview: noPreview(),
		Markup: rows([]telego.InlineKeyboardButton{
			keyboards.Button("➕ Собрать выход", NewOutingData), keyboards.Button("🥾 Мои выходы", ListData(FromMine)),
		}),
	}
}
