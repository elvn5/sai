package i18n

import (
	"fmt"
	"strings"
	"time"
)

// names are a language's words for dates.
type names struct {
	// Short weekdays, Sunday first ("сб").
	weekdays [7]string
	// Weekdays in full, Sunday first ("суббота").
	weekdaysFull [7]string
	// Abbreviated months ("окт.").
	months [12]string
	// Months in a date ("24 октября"): Russian genitive.
	monthsFull [12]string
	// Months on their own ("Октябрь 2026").
	monthsAlone [12]string
}

var dateNames = map[Lang]names{
	RU: {
		weekdays:     [7]string{"вс", "пн", "вт", "ср", "чт", "пт", "сб"},
		weekdaysFull: [7]string{"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"},
		months:       [12]string{"янв.", "фев.", "мар.", "апр.", "мая", "июн.", "июл.", "авг.", "сен.", "окт.", "нояб.", "дек."},
		monthsFull: [12]string{
			"января", "февраля", "марта", "апреля", "мая", "июня",
			"июля", "августа", "сентября", "октября", "ноября", "декабря",
		},
		monthsAlone: [12]string{
			"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
			"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
		},
	},
	KY: {
		weekdays:     [7]string{"жк", "дш", "шш", "шр", "бш", "жм", "иш"},
		weekdaysFull: [7]string{"жекшемби", "дүйшөмбү", "шейшемби", "шаршемби", "бейшемби", "жума", "ишемби"},
		months:       [12]string{"янв.", "фев.", "мар.", "апр.", "май", "июн.", "июл.", "авг.", "сен.", "окт.", "нояб.", "дек."},
		monthsFull: [12]string{
			"январь", "февраль", "март", "апрель", "май", "июнь",
			"июль", "август", "сентябрь", "октябрь", "ноябрь", "декабрь",
		},
		monthsAlone: [12]string{
			"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
			"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
		},
	},
	EN: {
		weekdays:     [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
		weekdaysFull: [7]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
		months:       [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
		monthsFull: [12]string{
			"January", "February", "March", "April", "May", "June",
			"July", "August", "September", "October", "November", "December",
		},
		monthsAlone: [12]string{
			"January", "February", "March", "April", "May", "June",
			"July", "August", "September", "October", "November", "December",
		},
	},
	DE: {
		weekdays:     [7]string{"So.", "Mo.", "Di.", "Mi.", "Do.", "Fr.", "Sa."},
		weekdaysFull: [7]string{"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"},
		months:       [12]string{"Jan.", "Feb.", "März", "Apr.", "Mai", "Juni", "Juli", "Aug.", "Sept.", "Okt.", "Nov.", "Dez."},
		monthsFull: [12]string{
			"Januar", "Februar", "März", "April", "Mai", "Juni",
			"Juli", "August", "September", "Oktober", "November", "Dezember",
		},
		monthsAlone: [12]string{
			"Januar", "Februar", "März", "April", "Mai", "Juni",
			"Juli", "August", "September", "Oktober", "November", "Dezember",
		},
	},
}

func (l Lang) names() names {
	if n, ok := dateNames[l]; ok {
		return n
	}
	return dateNames[RU]
}

// dayMonth joins a day and a month as the language does: "24 окт.",
// "24-окт.", "24 Oct", "24. Okt.".
func (l Lang) dayMonth(day int, month string) string {
	switch l {
	case KY:
		return fmt.Sprintf("%d-%s", day, month)
	case DE:
		return fmt.Sprintf("%d. %s", day, month)
	default:
		return fmt.Sprintf("%d %s", day, month)
	}
}

// When is a moment as wall clock time, e.g. "сб, 24 окт., 06:00",
// "Sat, 24 Oct, 06:00".
func (l Lang) When(t time.Time) string {
	n := l.names()
	return n.weekdays[t.Weekday()] + ", " + l.dayMonth(t.Day(), n.months[t.Month()-1]) + ", " + t.Format("15:04")
}

// DayMonth is a day of a month, abbreviated: "24 окт.".
func (l Lang) DayMonth(t time.Time) string { return l.dayMonth(t.Day(), l.names().months[t.Month()-1]) }

// Date is a day with its year: "18 окт. 2026".
func (l Lang) Date(t time.Time) string { return fmt.Sprintf("%s %d", l.DayMonth(t), t.Year()) }

// LongDate is a day in full: "суббота, 24 октября".
func (l Lang) LongDate(t time.Time) string {
	n := l.names()
	return n.weekdaysFull[t.Weekday()] + ", " + l.dayMonth(t.Day(), n.monthsFull[t.Month()-1])
}

// MonthYear names a month of a year: "Октябрь 2026".
func (l Lang) MonthYear(t time.Time) string {
	return fmt.Sprintf("%s %d", l.names().monthsAlone[t.Month()-1], t.Year())
}

// Weekdays are the short weekday names, Monday first, for a calendar.
func (l Lang) Weekdays() []string {
	w := l.names().weekdays
	out := make([]string, 0, 7)
	for i := 1; i <= 7; i++ {
		out = append(out, upperFirst(w[i%7]))
	}
	return out
}

// DateRange is the days from start to end: "12 — 14 июня", "30 июня —
// 2 июля"; one day when end is zero or the same.
func (l Lang) DateRange(start, end time.Time) string {
	full := l.names().monthsFull
	sameMonth := start.Month() == end.Month() && start.Year() == end.Year()
	switch {
	case end.IsZero() || start.Equal(end):
		return l.dayMonth(start.Day(), full[start.Month()-1])
	case sameMonth && l == DE:
		return fmt.Sprintf("%d. — %s", start.Day(), l.dayMonth(end.Day(), full[end.Month()-1]))
	case sameMonth:
		return fmt.Sprintf("%d — %s", start.Day(), l.dayMonth(end.Day(), full[end.Month()-1]))
	default:
		return l.dayMonth(start.Day(), full[start.Month()-1]) + " — " + l.dayMonth(end.Day(), full[end.Month()-1])
	}
}

func upperFirst(s string) string {
	for i := range s {
		if i > 0 {
			return strings.ToUpper(s[:i]) + s[i:]
		}
	}
	return strings.ToUpper(s)
}
