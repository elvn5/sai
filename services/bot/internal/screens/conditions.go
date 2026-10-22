package screens

import (
	"strconv"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
)

// A condition report after an outing, two taps: the trail's state, then
// what was needed on the feet. "c:s:<outing>" asks the state,
// "c:t:<outing>:<state>" the traction, "c:d:<outing>:<state>:<traction>"
// sends it (answers by index, to fit Telegram's 64 bytes).
const (
	CondState    = "s"
	CondTraction = "t"
	CondDone     = "d"
)

// Option is a value the API takes and how the bot says it.
type Option struct{ Value, Label string }

var (
	TrailStates = []Option{
		{"clear", "☀️ Сухо, тропа чистая"}, {"wet", "💧 Мокро, грязь"}, {"patchy_snow", "🌨 Местами снег"},
		{"snow", "❄️ Снег"}, {"deep_snow", "⛄ Глубокий снег"}, {"ice", "🧊 Лёд"},
	}
	Tractions = []Option{
		{"none", "👟 Хватило обычной обуви"}, {"microspikes", "🦶 Пригодились микрошипы"}, {"crampons", "⛓ Нужны кошки"},
	}
)

// ConditionStart asks about the route of a finished outing.
func ConditionStart(outingID string) string {
	return keyboards.ConditionPrefix + CondState + ":" + outingID
}

func conditionData(step, outingID string, answers ...int) string {
	data := keyboards.ConditionPrefix + step + ":" + outingID
	for _, a := range answers {
		data += ":" + strconv.Itoa(a)
	}
	return data
}

func trailName(o api.Outing) string {
	if o.Trail == nil {
		return ""
	}
	return o.Trail.Name
}

// ConditionState asks what the trail was like.
func ConditionState(l i18n.Lang, o api.Outing) Screen {
	text := l.F("📝 <b>Как маршрут «%s»?</b>", esc(trailName(o))) + "\n\n" +
		l.T("Расскажите в два нажатия — сводка появится на странице маршрута и поможет тем, кто пойдёт следом.") + "\n\n" +
		l.T("<b>Какой была тропа?</b>")
	var buttons [][]telego.InlineKeyboardButton
	for i := 0; i < len(TrailStates); i += 2 {
		row := []telego.InlineKeyboardButton{keyboards.Button(l.T(TrailStates[i].Label), conditionData(CondTraction, o.ID, i))}
		if i+1 < len(TrailStates) {
			row = append(row, keyboards.Button(l.T(TrailStates[i+1].Label), conditionData(CondTraction, o.ID, i+1)))
		}
		buttons = append(buttons, row)
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("⬅️ К выходу"), OutingData(ActView, FromPast, o.ID))})
	return Screen{Text: text, Markup: rows(buttons...), Preview: noPreview()}
}

// ConditionTraction asks what was needed on the feet.
func ConditionTraction(l i18n.Lang, o api.Outing, state int) Screen {
	text := l.F("📝 <b>Как маршрут «%s»?</b>", esc(trailName(o))) + "\n\n" + l.F("Тропа: %s", l.T(TrailStates[state].Label)) +
		"\n\n" + l.T("<b>Что понадобилось на ноги?</b>")
	var buttons [][]telego.InlineKeyboardButton
	for i, t := range Tractions {
		buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T(t.Label), conditionData(CondDone, o.ID, state, i))})
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{keyboards.Button(l.T("⬅️ Назад"), ConditionStart(o.ID))})
	return Screen{Text: text, Markup: rows(buttons...), Preview: noPreview()}
}

// ConditionThanks closes a sent report.
func ConditionThanks(l i18n.Lang, o api.Outing, state, traction int) Screen {
	text := l.T("🙏 <b>Спасибо!</b>") + "\n\n" + l.F("Сводка по маршруту «%s» опубликована:", esc(trailName(o))) + "\n" +
		l.T(TrailStates[state].Label) + "\n" + l.T(Tractions[traction].Label)
	return Screen{Text: text, Preview: noPreview(), Markup: rows(
		[]telego.InlineKeyboardButton{keyboards.Button(l.T("⬅️ К выходу"), OutingData(ActView, FromPast, o.ID))},
	)}
}
