package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
	"github.com/elvn5/sai/services/bot/internal/screens"
	"github.com/elvn5/sai/services/bot/internal/session"
)

// Rating is the public API as rating an outing uses it (api.Client).
type Rating interface {
	RateOuting(ctx context.Context, userID, outingID string, rating int, text *string) (api.Outing, error)
}

// ratingErrors are the trails service's refusals to take a rating.
var ratingErrors = map[string]string{
	"Only participants can rate this outing":   "Оценить выход могут только его участники",
	"Only those who came can rate this outing": "Оценить могут те, кого организатор отметил на сборе",
	"The outing hasn't started yet":            "Оценить можно после выхода",
	"Rating is closed":                         "Оценить выход можно в течение месяца после него",
	"This outing was cancelled":                "Выход отменён — оценивать нечего",
}

func (h *Handlers) rating() bool { return h.sections() && h.Ratings != nil }

// ratingPress answers "r:s:<outing>" (ask for stars), "r:v:<outing>:<n>"
// (give them; 0 keeps the rating) and "r:w:<outing>" (write a comment).
func (h *Handlers) ratingPress(ctx *th.Context, q telego.CallbackQuery) error {
	parts := strings.Split(strings.TrimPrefix(q.Data, keyboards.RatingPrefix), ":")
	if len(parts) < 2 {
		return h.answer(ctx, q, "")
	}
	userID, err := h.userOf(ctx, &q.From)
	if err != nil {
		return err
	}
	o, err := h.Content.Outing(ctx, userID, parts[1])
	if api.StatusOf(err) == http.StatusNotFound || api.StatusOf(err) == http.StatusBadRequest {
		return h.answer(ctx, q, "Этого выхода больше нет")
	}
	if err != nil {
		return h.failedPress(ctx, q, err)
	}

	switch {
	case parts[0] == screens.RateAsk:
		if !o.CanRate && o.MyRating == nil {
			return h.answer(ctx, q, "Оценить выход могут его участники — после выхода")
		}
		if err := h.answer(ctx, q, ""); err != nil {
			return err
		}
		return h.show(ctx, q, screens.RateOuting(o))

	case parts[0] == screens.RateVote && len(parts) == 3:
		h.Sessions.Cancel(chatOf(q))
		stars, _ := strconv.Atoi(parts[2])
		toast := ""
		if stars >= 1 && stars <= 5 {
			var text *string
			if o.MyRating != nil {
				text = o.MyRating.Text
			}
			rated, err := h.Ratings.RateOuting(ctx, userID, o.ID, stars, text)
			if msg, known := ratingErrors[api.MessageOf(err)]; known {
				return h.answer(ctx, q, msg)
			}
			if err != nil {
				return h.failedPress(ctx, q, err)
			}
			o, toast = rated, "Спасибо за оценку!"
		}
		if o.MyRating == nil {
			if err := h.answer(ctx, q, ""); err != nil {
				return err
			}
			return h.show(ctx, q, screens.RateOuting(o))
		}
		if err := h.answer(ctx, q, toast); err != nil {
			return err
		}
		return h.show(ctx, q, screens.RateThanks(o, h.now()))

	case parts[0] == screens.RateWrite:
		if o.MyRating == nil {
			if err := h.answer(ctx, q, "Сначала поставьте оценку"); err != nil {
				return err
			}
			return h.show(ctx, q, screens.RateOuting(o))
		}
		h.Sessions.StartWith(chatOf(q), session.FieldReview, o.ID)
		if err := h.answer(ctx, q, ""); err != nil {
			return err
		}
		return h.show(ctx, q, screens.ReviewPrompt(o))
	}
	return h.answer(ctx, q, "")
}

// reviewText saves the typed comment with the participant's rating.
func (h *Handlers) reviewText(ctx *th.Context, msg telego.Message, outingID string) error {
	text := strings.TrimSpace(msg.Text)
	if msg.From == nil || text == "" {
		return nil
	}
	if utf8.RuneCountInString(text) > screens.ReviewMaxLen {
		h.Sessions.StartWith(msg.Chat.ID, session.FieldReview, outingID)
		return h.reply(ctx, msg.Chat.ID, "Слишком длинно: не больше "+strconv.Itoa(screens.ReviewMaxLen)+
			" символов. Сократите и отправьте ещё раз.", nil)
	}
	userID, err := h.userOf(ctx, msg.From)
	if err != nil {
		return err
	}
	o, err := h.Content.Outing(ctx, userID, outingID)
	if err != nil {
		return h.failed(ctx, msg.Chat.ID, err)
	}
	if o.MyRating == nil {
		return h.send(ctx, msg.Chat.ID, screens.RateOuting(o))
	}
	rated, err := h.Ratings.RateOuting(ctx, userID, o.ID, o.MyRating.Rating, &text)
	if refusal, known := ratingErrors[api.MessageOf(err)]; known {
		return h.reply(ctx, msg.Chat.ID, refusal, nil)
	}
	if err != nil {
		return h.failed(ctx, msg.Chat.ID, err)
	}
	return h.send(ctx, msg.Chat.ID, screens.RateThanks(rated, h.now()))
}
