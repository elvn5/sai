package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
	"github.com/elvn5/sai/services/bot/internal/screens"
)

var bishkek = time.FixedZone("Asia/Bishkek", 6*60*60)

// conditionPress walks a condition report: "c:s:<outing>",
// "c:t:<outing>:<state>", "c:d:<outing>:<state>:<traction>" sends it.
func (h *Handlers) conditionPress(ctx *th.Context, q telego.CallbackQuery) error {
	parts := strings.Split(strings.TrimPrefix(q.Data, keyboards.ConditionPrefix), ":")
	if len(parts) < 2 {
		return h.answer(ctx, q, "")
	}
	answers := make([]int, 0, 2)
	for _, p := range parts[2:] {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return h.answer(ctx, q, "")
		}
		answers = append(answers, n)
	}
	if len(answers) > 0 && answers[0] >= len(screens.TrailStates) || len(answers) > 1 && answers[1] >= len(screens.Tractions) {
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
	if !screens.CanReport(o, h.now()) {
		return h.answer(ctx, q, "Сводку можно оставить в течение 30 дней после выхода")
	}

	switch {
	case parts[0] == screens.CondState:
		if err := h.answer(ctx, q, ""); err != nil {
			return err
		}
		return h.show(ctx, q, screens.ConditionState(o))
	case parts[0] == screens.CondTraction && len(answers) == 1:
		if err := h.answer(ctx, q, ""); err != nil {
			return err
		}
		return h.show(ctx, q, screens.ConditionTraction(o, answers[0]))
	case parts[0] == screens.CondDone && len(answers) == 2:
		// The day of the outing (it may have gone on overnight: never
		// later than today).
		day := o.StartsAt.In(bishkek).Format(time.DateOnly)
		if today := h.now().In(bishkek).Format(time.DateOnly); day > today {
			day = today
		}
		err := h.Content.ReportCondition(ctx, userID, o.Trail.Slug, api.ConditionInput{
			ObservedOn: day, TrailState: screens.TrailStates[answers[0]].Value, Traction: screens.Tractions[answers[1]].Value,
		})
		if api.StatusOf(err) == http.StatusBadRequest {
			return h.answer(ctx, q, "Сводку можно оставить в течение 30 дней после выхода")
		}
		if err != nil {
			return h.failedPress(ctx, q, err)
		}
		if err := h.answer(ctx, q, "🙏 Спасибо за сводку!"); err != nil {
			return err
		}
		return h.show(ctx, q, screens.ConditionThanks(o, answers[0], answers[1]))
	}
	return h.answer(ctx, q, "")
}
