package handlers

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
	"github.com/elvn5/sai/services/bot/internal/screens"
	"github.com/elvn5/sai/services/bot/internal/session"
)

// Rating is the public API as rating a tour uses it (api.Client).
type Rating interface {
	RateTour(ctx context.Context, userID, tourID string, rating int, text *string) (api.Tour, error)
}

// ratingErrors are the listings service's refusals to take a rating.
var ratingErrors = map[string]string{
	"Only participants can rate this tour":   "Оценить тур могут только его участники",
	"Only those who came can rate this tour": "Оценить могут те, кого организатор отметил на сборе",
	"The tour hasn't started yet":            "Оценить можно после тура",
	"Rating is closed":                       "Оценить тур можно в течение месяца после него",
}

func (h *Handlers) rating() bool { return h.sections() && h.Ratings != nil }

// ratingPress answers "r:s:<tour>" (ask for stars), "r:v:<tour>:<n>" (give
// them; 0 keeps the rating) and "r:w:<tour>" (write a comment).
func (h *Handlers) ratingPress(ctx *th.Context, q telego.CallbackQuery) error {
	parts := strings.Split(strings.TrimPrefix(q.Data, keyboards.RatingPrefix), ":")
	if len(parts) < 2 {
		return h.answer(ctx, q, "")
	}
	l := h.lang(ctx, &q.From)
	userID, err := h.userOf(ctx, &q.From)
	if err != nil {
		return err
	}
	o, err := h.Content.TourByID(ctx, userID, parts[1])
	if gone(err) {
		return h.answer(ctx, q, l.T("Этого тура больше нет"))
	}
	if err != nil {
		return h.failedPress(ctx, l, q, err)
	}

	switch {
	case parts[0] == screens.RateAsk:
		if !o.CanRate && o.MyRating == nil {
			return h.answer(ctx, q, l.T("Оценить тур могут его участники — после тура"))
		}
		if err := h.answer(ctx, q, ""); err != nil {
			return err
		}
		return h.show(ctx, q, screens.RateTour(l, o))

	case parts[0] == screens.RateVote && len(parts) == 3:
		h.Sessions.Cancel(chatOf(q))
		stars, _ := strconv.Atoi(parts[2])
		toast := ""
		if stars >= 1 && stars <= 5 {
			var text *string
			if o.MyRating != nil {
				text = o.MyRating.Text
			}
			rated, err := h.Ratings.RateTour(ctx, userID, o.ID, stars, text)
			if msg, known := ratingErrors[api.MessageOf(err)]; known {
				return h.answer(ctx, q, l.T(msg))
			}
			if err != nil {
				return h.failedPress(ctx, l, q, err)
			}
			o, toast = rated, l.T("Спасибо за оценку!")
		}
		if o.MyRating == nil {
			if err := h.answer(ctx, q, ""); err != nil {
				return err
			}
			return h.show(ctx, q, screens.RateTour(l, o))
		}
		if err := h.answer(ctx, q, toast); err != nil {
			return err
		}
		return h.show(ctx, q, screens.RateThanks(l, o))

	case parts[0] == screens.RateWrite:
		if o.MyRating == nil {
			if err := h.answer(ctx, q, l.T("Сначала поставьте оценку")); err != nil {
				return err
			}
			return h.show(ctx, q, screens.RateTour(l, o))
		}
		h.Sessions.StartWith(chatOf(q), session.FieldReview, o.ID)
		if err := h.answer(ctx, q, ""); err != nil {
			return err
		}
		return h.show(ctx, q, screens.ReviewPrompt(l, o))
	}
	return h.answer(ctx, q, "")
}

// reviewText saves the typed comment with the participant's rating.
func (h *Handlers) reviewText(ctx *th.Context, l i18n.Lang, msg telego.Message, tourID string) error {
	text := strings.TrimSpace(msg.Text)
	if msg.From == nil || text == "" {
		return nil
	}
	if utf8.RuneCountInString(text) > screens.ReviewMaxLen {
		h.Sessions.StartWith(msg.Chat.ID, session.FieldReview, tourID)
		return h.reply(ctx, msg.Chat.ID, l.F("Слишком длинно: не больше %d символов. Сократите и отправьте ещё раз.", screens.ReviewMaxLen), nil)
	}
	userID, err := h.userOf(ctx, msg.From)
	if err != nil {
		return err
	}
	o, err := h.Content.TourByID(ctx, userID, tourID)
	if err != nil {
		return h.failed(ctx, l, msg.Chat.ID, err)
	}
	if o.MyRating == nil {
		return h.send(ctx, msg.Chat.ID, screens.RateTour(l, o))
	}
	rated, err := h.Ratings.RateTour(ctx, userID, o.ID, o.MyRating.Rating, &text)
	if refusal, known := ratingErrors[api.MessageOf(err)]; known {
		return h.reply(ctx, msg.Chat.ID, l.T(refusal), nil)
	}
	if err != nil {
		return h.failed(ctx, l, msg.Chat.ID, err)
	}
	return h.send(ctx, msg.Chat.ID, screens.RateThanks(l, rated))
}
