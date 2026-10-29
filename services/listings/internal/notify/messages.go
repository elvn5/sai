package notify

import (
	"embed"
	"strings"
	"time"

	"github.com/elvn5/sai/services/listings/internal/models"
	"github.com/elvn5/sai/services/pkg/telegram"
	"github.com/elvn5/sai/services/pkg/usersclient"
)

// Messages are written in Russian and translated with the catalogs in
// locales/ (keyed by the Russian text, worded as the bot words it) into
// the recipient's language.

//go:embed locales/*.json
var locales embed.FS

var texts = telegram.MustLoadCatalog(locales, "locales")

// when formats a time in Bishkek, e.g. "сб, 17 окт., 06:00".
func when(l telegram.Lang, t time.Time) string { return l.When(t.In(models.Bishkek)) }

func esc(s string) string { return telegram.EscapeHTML(s) }

func personName(l telegram.Lang, p usersclient.Author) string {
	switch {
	case p.FirstName != nil && *p.FirstName != "":
		return *p.FirstName
	case p.Username != nil && *p.Username != "":
		return "@" + *p.Username
	default:
		return texts.T(l, "Кто-то")
	}
}

func orEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// places is the head count after a sign-up or a drop-out.
func places(l telegram.Lang, t models.Tour) string {
	if t.Capacity == nil {
		return texts.F(l, "Записались: %d.", t.Taken)
	}
	return texts.F(l, "Занято мест: %d из %d.", t.Taken, *t.Capacity)
}

func joinedText(l telegram.Lang, t models.Tour, who usersclient.Author) string {
	return texts.F(l, "🙋 <b>%s</b> записывается на тур «%s» (%s).", esc(personName(l, who)), esc(t.Title), when(l, t.StartsAt())) +
		"\n" + places(l, t)
}

func leftText(l telegram.Lang, t models.Tour, who usersclient.Author) string {
	return texts.F(l, "👋 <b>%s</b> отменяет запись на тур «%s» (%s).", esc(personName(l, who)), esc(t.Title), when(l, t.StartsAt())) +
		"\n" + places(l, t)
}

// changedText lists what changed between before and after; "" when
// nothing participants need to know did.
func changedText(l telegram.Lang, before, after models.Tour) string {
	var changes []string
	if !before.StartsAt().Equal(after.StartsAt()) {
		changes = append(changes, texts.F(l, "🕕 Сбор: <b>%s</b> (было %s)", when(l, after.StartsAt()), when(l, before.StartsAt())))
	}
	if orEmpty(before.MeetingPoint) != orEmpty(after.MeetingPoint) && orEmpty(after.MeetingPoint) != "" {
		changes = append(changes, texts.F(l, "📍 Место сбора: <b>%s</b>", esc(*after.MeetingPoint)))
	}
	if len(changes) == 0 {
		return ""
	}
	return texts.F(l, "✏️ Организатор изменил тур «%s»:\n%s", esc(after.Title), strings.Join(changes, "\n"))
}

// messageText tells about a chat message, quoting its start.
func messageText(l telegram.Lang, from usersclient.Author, body string) string {
	if runes := []rune(body); len(runes) > 500 {
		body = string(runes[:500]) + "…"
	}
	return texts.F(l, "💬 Новое сообщение от <b>%s</b>:\n\n%s", esc(personName(l, from)), esc(body))
}

// promotedText tells someone on the waitlist a place freed up for them.
func promotedText(l telegram.Lang, t models.Tour) string {
	return texts.F(l, "🎉 Освободилось место на туре «%s» (%s) — вы в группе.", esc(t.Title), when(l, t.StartsAt()))
}

func cancelledText(l telegram.Lang, t models.Tour) string {
	return texts.F(l, "❌ Тур «%s» (%s) отменён организатором.", esc(t.Title), when(l, t.StartsAt()))
}

// contact is how participants reach the organizer: Telegram, else phone.
func contact(t models.Tour) string {
	switch {
	case t.Telegram != nil && *t.Telegram != "":
		return "@" + *t.Telegram
	case t.Phone != nil && *t.Phone != "":
		return *t.Phone
	case t.Whatsapp != nil && *t.Whatsapp != "":
		return *t.Whatsapp
	}
	return ""
}

func reminderText(l telegram.Lang, t models.Tour, forOrganizer bool) string {
	var b strings.Builder
	b.WriteString(texts.F(l, "⏰ Скоро тур «%s»\n\n🕕 Сбор: %s", esc(t.Title), when(l, t.StartsAt())))
	if p := orEmpty(t.MeetingPoint); p != "" {
		b.WriteString("\n📍 " + esc(p))
	}
	if forOrganizer {
		b.WriteString("\n" + texts.F(l, "👥 Участников: %d", t.Taken))
	} else if c := contact(t); c != "" {
		b.WriteString("\n" + texts.F(l, "💬 Организатор: %s", esc(c)))
	}
	b.WriteString("\n\n" + texts.T(l, "Проверьте прогноз погоды и снаряжение перед выездом."))
	return b.String()
}

// rateText asks a participant how the tour went.
func rateText(l telegram.Lang, t models.Tour) string {
	return texts.F(l, "🏔 Как прошёл тур «%s»?\n\nОцените его — так другие поймут, с кем ехать в горы.", esc(t.Title))
}

func transportRequestedText(l telegram.Lang, t models.Tour, from usersclient.Author, startsAt time.Time) string {
	return texts.F(l, "🚐 <b>%s</b> просит транспорт для тура «%s» — подача %s.",
		esc(personName(l, from)), esc(t.Title), when(l, startsAt))
}

func transportAnsweredText(l telegram.Lang, t models.Tour, driver, status string) string {
	switch status {
	case "accepted":
		return texts.F(l, "✅ Водитель <b>%s</b> принял заявку на транспорт для тура «%s».", esc(driver), esc(t.Title))
	case "declined":
		return texts.F(l, "❌ Водитель <b>%s</b> отклонил заявку на транспорт для тура «%s».", esc(driver), esc(t.Title))
	}
	return texts.F(l, "⚠️ Водитель <b>%s</b> отменил поездку для тура «%s».", esc(driver), esc(t.Title))
}

func transportCancelledText(l telegram.Lang, t models.Tour) string {
	return texts.F(l, "❌ Организатор отменил заявку на транспорт для тура «%s».", esc(t.Title))
}
