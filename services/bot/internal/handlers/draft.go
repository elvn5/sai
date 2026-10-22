package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/drafts"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/keyboards"
	"github.com/elvn5/sai/services/bot/internal/screens"
)

const (
	// Routes offered at the route step: the most popular, or the found.
	trailsOffered = 8
	// Steps with a required typed answer.
	requiredText = "Напишите ответ сообщением"
)

// draftErrors are the trails service's refusals by field: what to say and
// which step to go back to.
var draftErrors = map[string]struct {
	step drafts.Step
	text string
}{
	"trailId":       {drafts.StepTrail, "Маршрут не найден — выберите другой"},
	"title":         {drafts.StepTitle, "Проверьте название"},
	"startsAt":      {drafts.StepTime, "Сбор — не в прошлом и не дальше чем через полгода"},
	"returnsAt":     {drafts.StepReturn, "Возвращение должно быть позже сбора"},
	"meetingPoint":  {drafts.StepMeeting, "Проверьте место сбора"},
	"seats":         {drafts.StepSeats, "Мест не может быть меньше, чем уже записалось"},
	"costPerPerson": {drafts.StepCost, "Проверьте стоимость"},
	"description":   {drafts.StepDescription, "Описание слишком длинное"},
	"contact":       {drafts.StepContact, "Контакт слишком длинный"},
}

// organizerErrors are refusals to change an outing.
var organizerErrors = map[string]string{
	"This outing is closed":                     "Выход уже прошёл или отменён — изменить его нельзя",
	"Only the organizer can change this outing": "Изменять выход может только организатор",
}

// prepare loads what the draft's step offers: the popular routes.
func (h *Handlers) prepare(ctx *th.Context, d *drafts.Draft) {
	if d.Step != drafts.StepTrail || len(d.Trails) > 0 || d.Search != "" {
		return
	}
	trails, err := h.Content.Trails(ctx, "")
	if err != nil {
		h.Log.WarnContext(ctx, "trails for a draft", slog.Any("error", err))
		return
	}
	d.Trails = trails[:min(len(trails), trailsOffered)]
}

// showDraft shows the draft's step: in the pressed message, or — after a
// typed answer — as a new message under it (the old one loses its
// buttons, so only the latest step can be answered).
func (h *Handlers) showDraft(ctx *th.Context, l i18n.Lang, chatID int64, q *telego.CallbackQuery, d drafts.Draft) error {
	s := screens.DraftScreen(l, d, h.now())
	if q != nil && q.Message != nil && q.Message.IsAccessible() && q.Message.Message().Photo == nil {
		_, err := ctx.Bot().EditMessageText(ctx, &telego.EditMessageTextParams{
			ChatID: tu.ID(chatID), MessageID: q.Message.GetMessageID(),
			Text: s.Text, ParseMode: telego.ModeHTML, LinkPreviewOptions: s.Preview, ReplyMarkup: s.Markup,
		})
		if err == nil || strings.Contains(err.Error(), notModified) {
			d.MessageID = q.Message.GetMessageID()
			h.Drafts.Put(chatID, d)
			return nil
		}
		h.Log.DebugContext(ctx, "could not edit the draft, sending it anew", slog.Any("error", err))
	}
	if d.MessageID != 0 {
		h.dropButtonsOf(ctx, chatID, d.MessageID)
	}
	msg := tu.Message(tu.ID(chatID), s.Text).WithParseMode(telego.ModeHTML).
		WithLinkPreviewOptions(s.Preview).WithReplyMarkup(s.Markup)
	sent, err := ctx.Bot().SendMessage(ctx, msg)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	d.MessageID = sent.MessageID
	h.Drafts.Put(chatID, d)
	return nil
}

// dropButtonsOf removes a message's inline keyboard, best effort.
func (h *Handlers) dropButtonsOf(ctx *th.Context, chatID int64, messageID int) {
	_, err := ctx.Bot().EditMessageReplyMarkup(ctx, &telego.EditMessageReplyMarkupParams{ChatID: tu.ID(chatID), MessageID: messageID})
	if err != nil {
		h.Log.DebugContext(ctx, "could not remove buttons", slog.Any("error", err))
	}
}

// startDraft begins a new outing in the pressed message.
func (h *Handlers) startDraft(ctx *th.Context, l i18n.Lang, q telego.CallbackQuery) error {
	chatID := chatOf(q)
	h.Sessions.Cancel(chatID)
	if old, ok := h.Drafts.Get(chatID); ok && old.MessageID != 0 && (q.Message == nil || old.MessageID != q.Message.GetMessageID()) {
		h.dropButtonsOf(ctx, chatID, old.MessageID)
	}
	d := drafts.New(q.From.Username, h.now())
	h.prepare(ctx, &d)
	if err := h.answer(ctx, q, ""); err != nil {
		return err
	}
	return h.showDraft(ctx, l, chatID, &q, d)
}

// editDraft begins changing an outing the user organizes (the press is
// already answered).
func (h *Handlers) editDraft(ctx *th.Context, l i18n.Lang, q telego.CallbackQuery, o api.Outing) error {
	chatID := chatOf(q)
	defer h.Drafts.Lock(chatID)()
	h.Sessions.Cancel(chatID)
	if old, ok := h.Drafts.Get(chatID); ok && old.MessageID != 0 && (q.Message == nil || old.MessageID != q.Message.GetMessageID()) {
		h.dropButtonsOf(ctx, chatID, old.MessageID)
	}
	return h.showDraft(ctx, l, chatID, &q, drafts.FromOuting(o, q.From.Username))
}

// draftPress answers the draft's buttons.
func (h *Handlers) draftPress(ctx *th.Context, q telego.CallbackQuery) error {
	action, arg, _ := strings.Cut(strings.TrimPrefix(q.Data, keyboards.DraftPrefix), ":")
	chatID := chatOf(q)
	l := h.lang(ctx, &q.From)
	defer h.Drafts.Lock(chatID)()
	if action == "new" {
		return h.startDraft(ctx, l, q)
	}
	d, ok := h.Drafts.Get(chatID)
	if !ok || (q.Message != nil && d.MessageID != 0 && q.Message.GetMessageID() != d.MessageID) {
		h.dropButtons(ctx, q)
		return h.answer(ctx, q, l.T(screens.DraftGone))
	}
	now := h.now()
	toast := ""

	switch action {
	case screens.DraftBack:
		d.Back()
	case screens.DraftCancel:
		return h.cancelDraft(ctx, l, q, d)
	case screens.DraftReview:
		d.Step = drafts.StepReview
	case screens.DraftFields:
		d.Step = drafts.StepFields
	case screens.DraftSave:
		return h.saveDraft(ctx, l, q, d)
	case screens.DraftGo:
		d.Go(drafts.Step(arg))
	case screens.DraftSkip:
		if (d.Step == drafts.StepTitle && d.Title == "") || (d.Step == drafts.StepMeeting && d.MeetingPoint == "") {
			toast = l.T(requiredText)
		} else {
			d.Next()
		}
	case screens.DraftClear:
		switch d.Step {
		case drafts.StepDescription:
			d.Description = ""
		case drafts.StepContact:
			d.Contact = ""
		}
		d.Next()
	case screens.DraftTrail:
		if arg == screens.DraftNone {
			d.TrailID, d.TrailName = "", ""
			d.Next()
		} else if i, err := strconv.Atoi(arg); err == nil && i >= 0 && i < len(d.Trails) {
			d.TrailID, d.TrailName = d.Trails[i].ID, d.Trails[i].Name
			d.Next()
		}
	case screens.DraftTitle:
		if d.TrailName != "" {
			d.Title = d.TrailName
			d.Next()
		}
	case screens.DraftMonth:
		if _, err := time.Parse("2006-01", arg); err == nil {
			d.Month = arg
		}
	case screens.DraftDate:
		toast = l.Say(d.SetDate(arg, now))
	case screens.DraftTime:
		if len(arg) == 4 {
			toast = l.Say(d.SetTime(arg[:2]+":"+arg[2:], now))
		}
	case screens.DraftReturn:
		toast = l.Say(h.pickReturn(&d, arg))
	case screens.DraftMove:
		if slices.Contains(drafts.Transports, arg) {
			d.Transport = arg
			d.Next()
		}
	case screens.DraftSeats:
		if n, err := strconv.Atoi(arg); err == nil {
			toast = l.Say(d.SetSeats(n))
		}
	case screens.DraftCost:
		if arg == screens.DraftNone {
			d.Cost = nil
		} else if n, err := strconv.Atoi(arg); err == nil {
			d.Cost = &n
		}
		d.Next()
	case screens.DraftLevel:
		if slices.Contains(drafts.Levels, arg) {
			d.Level = arg
			d.Next()
		}
	case screens.DraftContact:
		if d.Username != "" {
			d.Contact = "@" + d.Username
			d.Next()
		}
	}

	h.prepare(ctx, &d)
	if err := h.answer(ctx, q, toast); err != nil {
		return err
	}
	return h.showDraft(ctx, l, chatID, &q, d)
}

// pickReturn applies a return button: "1900" on the start's day, "+1800"
// the day after, "none".
func (h *Handlers) pickReturn(d *drafts.Draft, arg string) i18n.Msg {
	if arg == screens.DraftNone {
		return d.SetReturn("")
	}
	date := d.Date
	if clock, ok := strings.CutPrefix(arg, "+"); ok {
		arg = clock
		if t, err := time.Parse(time.DateOnly, d.Date); err == nil {
			date = t.AddDate(0, 0, 1).Format(time.DateOnly)
		}
	}
	if len(arg) != 4 {
		return i18n.Msg{}
	}
	return d.SetReturn(date + "T" + arg[:2] + ":" + arg[2:])
}

// cancelDraft drops the draft: a new outing isn't created, an edited one
// shows its card again.
func (h *Handlers) cancelDraft(ctx *th.Context, l i18n.Lang, q telego.CallbackQuery, d drafts.Draft) error {
	chatID := chatOf(q)
	h.Drafts.Drop(chatID)
	if d.OutingID == "" {
		if err := h.answer(ctx, q, ""); err != nil {
			return err
		}
		return h.show(ctx, q, screens.DraftCancelled(l))
	}
	userID, err := h.userOf(ctx, &q.From)
	if err != nil {
		return err
	}
	o, err := h.Content.Outing(ctx, userID, d.OutingID)
	if err != nil {
		return h.failedPress(ctx, l, q, err)
	}
	if err := h.answer(ctx, q, l.T("Изменения отменены")); err != nil {
		return err
	}
	return h.show(ctx, q, screens.OutingCard(l, o, screens.FromMine, h.Links, h.now()))
}

// saveDraft publishes the outing (or saves the changes) and shows its card.
func (h *Handlers) saveDraft(ctx *th.Context, l i18n.Lang, q telego.CallbackQuery, d drafts.Draft) error {
	chatID := chatOf(q)
	if missing := d.Missing(); missing != "" {
		d.Go(missing)
		if err := h.answer(ctx, q, l.T("Осталось заполнить этот шаг")); err != nil {
			return err
		}
		return h.showDraft(ctx, l, chatID, &q, d)
	}
	userID, err := h.userOf(ctx, &q.From)
	if err != nil {
		return err
	}
	var o api.Outing
	toast := l.T("🎉 Выход опубликован!")
	if d.OutingID == "" {
		o, err = h.Content.CreateOuting(ctx, userID, d.Input())
	} else {
		o, err = h.Content.UpdateOuting(ctx, userID, d.OutingID, d.Input())
		toast = l.T("✅ Изменения сохранены")
	}
	if err != nil {
		return h.draftRefused(ctx, l, q, d, err)
	}
	h.Drafts.Drop(chatID)
	if d.OutingID == "" {
		h.track(&q.From, l, eventCreateOuting, map[string]any{"outing_id": o.ID})
	}
	if err := h.answer(ctx, q, toast); err != nil {
		return err
	}
	return h.show(ctx, q, screens.OutingCard(l, o, screens.FromMine, h.Links, h.now()))
}

// draftRefused explains why the API didn't take the draft and returns to
// the step to fix.
func (h *Handlers) draftRefused(ctx *th.Context, l i18n.Lang, q telego.CallbackQuery, d drafts.Draft, err error) error {
	status := api.StatusOf(err)
	if status != http.StatusBadRequest && status != http.StatusForbidden && status != http.StatusConflict {
		return h.failedPress(ctx, l, q, err)
	}
	if msg, ok := organizerErrors[api.MessageOf(err)]; ok {
		h.Drafts.Drop(chatOf(q))
		h.dropButtons(ctx, q)
		return h.answer(ctx, q, l.T(msg))
	}
	toast := l.T("Проверьте выход — что-то заполнено не так")
	for field, problem := range api.FieldsOf(err) {
		if known, ok := draftErrors[field]; ok {
			toast = l.T(known.text)
			d.Go(known.step)
			break
		}
		h.Log.WarnContext(ctx, "outing refused", slog.String("field", field), slog.String("error", problem))
	}
	if err := h.answer(ctx, q, toast); err != nil {
		return err
	}
	return h.showDraft(ctx, l, chatOf(q), &q, d)
}

// draftText takes a typed answer to the draft's step; reports whether the
// chat has a draft (otherwise the text isn't for it).
func (h *Handlers) draftText(ctx *th.Context, l i18n.Lang, msg telego.Message) (bool, error) {
	if h.Drafts == nil {
		return false, nil
	}
	defer h.Drafts.Lock(msg.Chat.ID)()
	d, ok := h.Drafts.Get(msg.Chat.ID)
	if !ok {
		return false, nil
	}
	text := strings.TrimSpace(msg.Text)
	switch d.Step {
	case drafts.StepTrail:
		trails, err := h.Content.Trails(ctx, text)
		if err != nil {
			return true, h.failed(ctx, l, msg.Chat.ID, err)
		}
		d.Search, d.Trails = text, trails[:min(len(trails), trailsOffered)]
	case drafts.StepReview, drafts.StepFields:
		return true, h.reply(ctx, msg.Chat.ID, l.T("Выберите, что сделать, кнопкой под выходом 👆"), nil)
	default:
		if problem := d.Text(text, h.now()); problem.Format != "" {
			return true, h.reply(ctx, msg.Chat.ID, l.Say(problem), nil)
		}
	}
	h.prepare(ctx, &d)
	return true, h.showDraft(ctx, l, msg.Chat.ID, nil, d)
}
