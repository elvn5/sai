package screens

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
)

// Rating callbacks: "r:s:<outing>" asks for stars, "r:v:<outing>:<1-5>"
// gives them, "r:w:<outing>" starts the comment.
const (
	RateAsk   = "s"
	RateVote  = "v"
	RateWrite = "w"
	// What a comment can hold (as the trails service takes it).
	ReviewMaxLen = 1000
)

func RateData(action, outingID string, stars ...int) string {
	data := keyboards.RatingPrefix + action + ":" + outingID
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

// RateOuting asks a participant for 1–5 stars.
func RateOuting(o api.Outing) Screen {
	var b strings.Builder
	fmt.Fprintf(&b, "⭐ <b>Как прошёл выход «%s»?</b>\n%s", esc(o.Title), When(o.StartsAt))
	if name := o.Organizer.Name(); name != "" {
		b.WriteString(" · организатор " + esc(name))
	}
	b.WriteString("\n\nОцените от 1 до 5. Оценка появится на странице организатора — так другие поймут, с кем идти в горы.")
	if o.MyRating != nil {
		b.WriteString("\n\nВаша оценка сейчас: " + StarLine(o.MyRating.Rating))
	}
	stars := make([]telego.InlineKeyboardButton, 0, 5)
	for n := 1; n <= 5; n++ {
		stars = append(stars, keyboards.Button(strconv.Itoa(n)+" ★", RateData(RateVote, o.ID, n)))
	}
	return Screen{Text: b.String(), Markup: rows(stars), Preview: noPreview()}
}

// RateThanks thanks for a rating and offers a comment and, on a catalog
// route, a condition report.
func RateThanks(o api.Outing, now time.Time) Screen {
	rating := 0
	hasText := false
	if o.MyRating != nil {
		rating = o.MyRating.Rating
		hasText = o.MyRating.Text != nil && *o.MyRating.Text != ""
	}
	text := fmt.Sprintf("Спасибо! Ваша оценка выхода «%s»: %s", esc(o.Title), StarLine(rating))
	if hasText {
		text += "\n\n<i>" + esc(clip(*o.MyRating.Text, 500)) + "</i>"
	} else {
		text += "\n\nДобавьте пару слов — что понравилось, что можно улучшить. Отзыв увидят на странице организатора."
	}
	write := "✍️ Написать отзыв"
	if hasText {
		write = "✍️ Изменить отзыв"
	}
	buttons := [][]telego.InlineKeyboardButton{{keyboards.Button(write, RateData(RateWrite, o.ID))}}
	if CanReport(o, now) {
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("📝 Рассказать, как маршрут", ConditionStart(o.ID))})
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button("Изменить оценку", RateData(RateAsk, o.ID))})
	return Screen{Text: text, Markup: rows(buttons...), Preview: noPreview()}
}

// ReviewPrompt asks for the comment.
func ReviewPrompt(o api.Outing) Screen {
	return Screen{
		Text:    fmt.Sprintf("Напишите отзыв о выходе «%s» одним сообщением.", esc(o.Title)),
		Markup:  rows([]telego.InlineKeyboardButton{keyboards.Button("Отмена", RateData(RateVote, o.ID, 0))}),
		Preview: noPreview(),
	}
}
