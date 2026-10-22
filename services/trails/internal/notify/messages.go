package notify

import (
	"embed"
	"strings"
	"time"

	"github.com/elvn5/sai/services/pkg/telegram"
	"github.com/elvn5/sai/services/trails/internal/models"
)

// Messages are written in Russian and translated with the catalogs in
// locales/ (keyed by the Russian text, worded as the bot words it) into
// the recipient's language.

//go:embed locales/*.json
var locales embed.FS

var texts = telegram.MustLoadCatalog(locales, "locales")

var bishkek = time.FixedZone("Asia/Bishkek", 6*60*60)

// when formats a time in Bishkek, e.g. "сб, 17 окт., 06:00".
func when(l telegram.Lang, t time.Time) string { return l.When(t.In(bishkek)) }

func esc(s string) string { return telegram.EscapeHTML(s) }

func personName(l telegram.Lang, p models.Person) string {
	switch {
	case p.FirstName != nil && *p.FirstName != "":
		return *p.FirstName
	case p.Username != nil && *p.Username != "":
		return "@" + *p.Username
	default:
		return texts.T(l, "Кто-то")
	}
}

func joinedText(l telegram.Lang, o models.Outing, who models.Person) string {
	return texts.F(l, "🙋 <b>%s</b> присоединяется к выходу «%s» (%s).\nЗанято мест: %d из %d.",
		esc(personName(l, who)), esc(o.Title), when(l, o.StartsAt.Time), o.Taken, o.Seats)
}

func leftText(l telegram.Lang, o models.Outing, who models.Person) string {
	return texts.F(l, "👋 <b>%s</b> не сможет пойти на выход «%s» (%s).\nСвободно мест: %d из %d.",
		esc(personName(l, who)), esc(o.Title), when(l, o.StartsAt.Time), o.Seats-o.Taken, o.Seats)
}

// changedText lists what changed between before and after; "" when
// nothing participants need to know did.
func changedText(l telegram.Lang, before, after models.Outing) string {
	var changes []string
	if !before.StartsAt.Equal(after.StartsAt.Time) {
		changes = append(changes, texts.F(l, "🕕 Сбор: <b>%s</b> (было %s)", when(l, after.StartsAt.Time), when(l, before.StartsAt.Time)))
	}
	if before.MeetingPoint != after.MeetingPoint {
		changes = append(changes, texts.F(l, "📍 Место сбора: <b>%s</b>", esc(after.MeetingPoint)))
	}
	if trailName(before) != trailName(after) {
		changes = append(changes, texts.F(l, "🏔 Маршрут: <b>%s</b>", esc(orDash(trailName(after)))))
	}
	if len(changes) == 0 {
		return ""
	}
	return texts.F(l, "✏️ Организатор изменил выход «%s»:\n%s", esc(after.Title), strings.Join(changes, "\n"))
}

func cancelledText(l telegram.Lang, o models.Outing) string {
	return texts.F(l, "❌ Выход «%s» (%s) отменён организатором.", esc(o.Title), when(l, o.StartsAt.Time))
}

func reminderText(l telegram.Lang, o models.Outing, forOrganizer bool) string {
	var b strings.Builder
	b.WriteString(texts.F(l, "⏰ Скоро выход «%s»\n\n🕕 Сбор: %s\n📍 %s", esc(o.Title), when(l, o.StartsAt.Time), esc(o.MeetingPoint)))
	if o.Trail != nil {
		b.WriteString("\n" + texts.F(l, "🏔 Маршрут: %s", esc(o.Trail.Name)))
	}
	if forOrganizer {
		b.WriteString("\n" + texts.F(l, "👥 Участников: %d", o.Taken))
	} else if o.Contact != nil {
		b.WriteString("\n" + texts.F(l, "💬 Организатор: %s", esc(*o.Contact)))
	}
	b.WriteString("\n\n" + texts.T(l, "Проверьте прогноз и сводки о состоянии маршрута перед выходом."))
	return b.String()
}

// rateText asks a participant how the outing went.
func rateText(l telegram.Lang, o models.Outing) string {
	text := texts.F(l, "🏔 Как прошёл выход «%s»?\n\nОцените его — так другие поймут, с кем идти в горы.", esc(o.Title))
	if o.Trail != nil {
		text += " " + texts.F(l, "А после оценки можно рассказать, что сейчас на маршруте «%s».", esc(o.Trail.Name))
	}
	return text
}

func followUpText(l telegram.Lang, o models.Outing) string {
	return texts.F(l, "🏔 Как прошёл выход «%s»?\n\nРасскажите, что сейчас на маршруте «%s»: снег, лёд, состояние тропы. Это поможет тем, кто пойдёт следом.",
		esc(o.Title), esc(o.Trail.Name))
}
