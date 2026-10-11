package notify

import (
	"fmt"
	"strings"
	"time"

	"github.com/elvn5/sai/services/pkg/telegram"
	"github.com/elvn5/sai/services/trails/internal/models"
)

// Messages are in Russian, the bot's language.

var (
	bishkek  = time.FixedZone("Asia/Bishkek", 6*60*60)
	weekdays = []string{"вс", "пн", "вт", "ср", "чт", "пт", "сб"}
	months   = []string{"янв.", "фев.", "мар.", "апр.", "мая", "июн.", "июл.", "авг.", "сен.", "окт.", "нояб.", "дек."}
)

// when formats a time in Bishkek, e.g. "сб, 17 окт., 06:00".
func when(t time.Time) string {
	t = t.In(bishkek)
	return fmt.Sprintf("%s, %d %s, %s", weekdays[t.Weekday()], t.Day(), months[t.Month()-1], t.Format("15:04"))
}

func esc(s string) string { return telegram.EscapeHTML(s) }

func personName(p models.Person) string {
	switch {
	case p.FirstName != nil && *p.FirstName != "":
		return *p.FirstName
	case p.Username != nil && *p.Username != "":
		return "@" + *p.Username
	default:
		return "Кто-то"
	}
}

func outingLine(o models.Outing) string {
	return fmt.Sprintf("«%s» (%s)", esc(o.Title), when(o.StartsAt.Time))
}

func joinedText(o models.Outing, who models.Person) string {
	return fmt.Sprintf("🙋 <b>%s</b> присоединяется к выходу %s.\nЗанято мест: %d из %d.",
		esc(personName(who)), outingLine(o), o.Taken, o.Seats)
}

func leftText(o models.Outing, who models.Person) string {
	return fmt.Sprintf("👋 <b>%s</b> не сможет пойти на выход %s.\nСвободно мест: %d из %d.",
		esc(personName(who)), outingLine(o), o.Seats-o.Taken, o.Seats)
}

func changedText(o models.Outing, changes []string) string {
	return fmt.Sprintf("✏️ Организатор изменил выход «%s»:\n%s", esc(o.Title), strings.Join(changes, "\n"))
}

func cancelledText(o models.Outing) string {
	return fmt.Sprintf("❌ Выход %s отменён организатором.", outingLine(o))
}

func reminderText(o models.Outing, forOrganizer bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "⏰ Скоро выход «%s»\n\n🕕 Сбор: %s\n📍 %s", esc(o.Title), when(o.StartsAt.Time), esc(o.MeetingPoint))
	if o.Trail != nil {
		fmt.Fprintf(&b, "\n🏔 Маршрут: %s", esc(o.Trail.Name))
	}
	if forOrganizer {
		fmt.Fprintf(&b, "\n👥 Участников: %d", o.Taken)
	} else if o.Contact != nil {
		fmt.Fprintf(&b, "\n💬 Организатор: %s", esc(*o.Contact))
	}
	b.WriteString("\n\nПроверьте прогноз и сводки о состоянии маршрута перед выходом.")
	return b.String()
}

func followUpText(o models.Outing) string {
	return fmt.Sprintf("🏔 Как прошёл выход «%s»?\n\nРасскажите, что сейчас на маршруте «%s»: снег, лёд, состояние тропы. Это поможет тем, кто пойдёт следом.",
		esc(o.Title), esc(o.Trail.Name))
}
