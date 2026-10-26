package screens

import (
	"strconv"
	"strings"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
)

// Rating callbacks: "r:s:<tour>" asks for stars, "r:v:<tour>:<1-5>" gives
// them, "r:w:<tour>" starts the comment.
const (
	RateAsk   = "s"
	RateVote  = "v"
	RateWrite = "w"
	// What a comment can hold (as the listings service takes it).
	ReviewMaxLen = 1000
)

func RateData(action, tourID string, stars ...int) string {
	data := keyboards.RatingPrefix + action + ":" + tourID
	for _, n := range stars {
		data += ":" + strconv.Itoa(n)
	}
	return data
}

// StarLine draws a rating as ★★★★☆.
func StarLine(n int) string {
	n = min(max(n, 0), 5)
	return strings.Repeat("★", n) + strings.Repeat("☆", 5-n)
}

// RateTour asks a participant for 1–5 stars.
func RateTour(l i18n.Lang, o api.Tour) Screen {
	var b strings.Builder
	b.WriteString(l.F("⭐ <b>Как прошёл тур «%s»?</b>", esc(o.Title)) + "\n" + When(l, o.StartsAt()))
	if o.Organizer != nil && o.Organizer.Name() != "" {
		b.WriteString(" · " + l.F("организатор %s", esc(o.Organizer.Name())))
	}
	b.WriteString("\n\n" + l.T("Оцените от 1 до 5. Оценка станет вашим отзывом о туре — так другие поймут, с кем ехать в горы."))
	if o.MyRating != nil {
		b.WriteString("\n\n" + l.F("Ваша оценка сейчас: %s", StarLine(o.MyRating.Rating)))
	}
	stars := make([]telego.InlineKeyboardButton, 0, 5)
	for n := 1; n <= 5; n++ {
		stars = append(stars, keyboards.Button(strconv.Itoa(n)+" ★", RateData(RateVote, o.ID, n)))
	}
	return Screen{Text: b.String(), Markup: rows(stars), Preview: noPreview()}
}

// RateThanks thanks for a rating and offers a comment.
func RateThanks(l i18n.Lang, o api.Tour) Screen {
	rating := 0
	hasText := false
	if o.MyRating != nil {
		rating = o.MyRating.Rating
		hasText = o.MyRating.Text != nil && *o.MyRating.Text != ""
	}
	text := l.F("Спасибо! Ваша оценка тура «%s»: %s", esc(o.Title), StarLine(rating))
	if hasText {
		text += "\n\n<i>" + esc(clip(*o.MyRating.Text, 500)) + "</i>"
	} else {
		text += "\n\n" + l.T("Добавьте пару слов — что понравилось, что можно улучшить. Отзыв увидят на странице тура и организатора.")
	}
	write := l.T("✍️ Написать отзыв")
	if hasText {
		write = l.T("✍️ Изменить отзыв")
	}
	buttons := [][]telego.InlineKeyboardButton{{keyboards.Button(write, RateData(RateWrite, o.ID))}}
	buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("Изменить оценку"), RateData(RateAsk, o.ID))})
	return Screen{Text: text, Markup: rows(buttons...), Preview: noPreview()}
}

// ReviewPrompt asks for the comment.
func ReviewPrompt(l i18n.Lang, o api.Tour) Screen {
	return Screen{
		Text:    l.F("Напишите отзыв о туре «%s» одним сообщением.", esc(o.Title)),
		Markup:  rows([]telego.InlineKeyboardButton{keyboards.Button(l.T("Отмена"), RateData(RateVote, o.ID, 0))}),
		Preview: noPreview(),
	}
}
