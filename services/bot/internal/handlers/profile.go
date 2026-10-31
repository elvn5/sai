package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/i18n"
	"github.com/elvn5/sai/services/bot/internal/screens"
	"github.com/elvn5/sai/services/bot/internal/session"
)

// Profiles is the account as the site's profile has it (api.Client): the
// participant's form, the password, achievements and partnership.
type Profiles interface {
	Me(ctx context.Context, userID string) (api.Me, error)
	ParticipantForm(ctx context.Context, userID string) (*api.ParticipantForm, error)
	SaveParticipantForm(ctx context.Context, userID string, form api.ParticipantForm) error
	ChangePassword(ctx context.Context, userID string, current *string, password string) error
	MyAchievements(ctx context.Context, userID string) (api.MyAchievements, error)
	HallOfFame(ctx context.Context) ([]api.HallEntry, error)
	ApplyAsPartner(ctx context.Context, userID, kind string) (api.Me, error)
}

// The profile section is on with the API (and the other sections, which
// its buttons open); without it the profile is the account's fields.
func (h *Handlers) profiles() bool { return h.sections() && h.Profiles != nil }

// The limits of the form's answers, as the listings service checks them.
const (
	nameMax = 120
	textMax = 2000
	// bcrypt reads at most 72 bytes of a password.
	passwordMin, passwordMax = 8, 72
)

var phonePattern = regexp.MustCompile(`^\+?[0-9\s\-()]{5,32}$`)

// formDraft is the form being filled in, kept in the chat's session.
type formDraft struct {
	Step string              `json:"step"`
	Form api.ParticipantForm `json:"form"`
}

// passwordDraft is the password being changed, kept in the chat's session
// for a few minutes at most.
type passwordDraft struct {
	Step    string `json:"step"`
	Has     bool   `json:"has"`
	Current string `json:"current,omitempty"`
	New     string `json:"new,omitempty"`
}

func (h *Handlers) profileView(ctx context.Context, userID string) (screens.ProfileView, error) {
	me, err := h.Profiles.Me(ctx, userID)
	if err != nil {
		return screens.ProfileView{}, err
	}
	v := screens.ProfileView{Me: me, Organizing: h.organizing()}
	// The rest is best effort: what doesn't load is left out.
	if list, err := h.Content.JoinedTours(ctx, userID); err == nil {
		v.Joined, v.JoinedKnown = list, true
	} else {
		h.Log.WarnContext(ctx, "profile tours", slog.Any("error", err))
	}
	if mine, err := h.Profiles.MyAchievements(ctx, userID); err == nil {
		v.Awards = &mine
	} else {
		h.Log.WarnContext(ctx, "profile achievements", slog.Any("error", err))
	}
	if form, err := h.Profiles.ParticipantForm(ctx, userID); err == nil {
		v.Form, v.FormKnown = form, true
	} else {
		h.Log.WarnContext(ctx, "profile form", slog.Any("error", err))
	}
	return v, nil
}

// profileSection is "👤 Профиль" with the API: the section's first screen.
func (h *Handlers) profileSection(ctx *th.Context, msg telego.Message) error {
	l := h.lang(ctx, msg.From)
	userID, err := h.userOf(ctx, msg.From)
	if err != nil {
		return err
	}
	v, err := h.profileView(ctx, userID)
	if err != nil {
		return h.failed(ctx, l, msg.Chat.ID, err)
	}
	return h.send(ctx, msg.Chat.ID, screens.Profile(l, v, h.now()))
}

// profilePress answers the profile's buttons ("p:…").
func (h *Handlers) profilePress(ctx *th.Context, q telego.CallbackQuery) error {
	l := h.lang(ctx, &q.From)
	userID, err := h.userOf(ctx, &q.From)
	if err != nil {
		return err
	}
	chatID := chatOf(q)
	var s screens.Screen
	toast := ""
	switch data := q.Data; {
	case data == screens.ProfileHome:
		v, err := h.profileView(ctx, userID)
		if err != nil {
			return h.failedPress(ctx, l, q, err)
		}
		s = screens.Profile(l, v, h.now())

	case data == screens.ProfileForm || data == screens.FormCancel:
		if data == screens.FormCancel {
			h.Sessions.Cancel(chatID)
		}
		form, err := h.Profiles.ParticipantForm(ctx, userID)
		if err != nil {
			return h.failedPress(ctx, l, q, err)
		}
		s = screens.FormScreen(l, form, "")

	case data == screens.FormEdit:
		form, err := h.Profiles.ParticipantForm(ctx, userID)
		if err != nil {
			return h.failedPress(ctx, l, q, err)
		}
		draft := formDraft{Step: screens.StepName}
		if form != nil {
			draft.Form = *form
		}
		s = h.formStep(l, chatID, draft, "")

	case strings.HasPrefix(data, screens.FormLevel), data == screens.FormKeep, data == screens.FormSkip:
		field, arg, ok := h.Sessions.Peek(chatID)
		var draft formDraft
		if !ok || field != session.FieldForm || json.Unmarshal([]byte(arg), &draft) != nil {
			return h.answer(ctx, q, l.T("Время вышло — начните заполнять анкету заново."))
		}
		switch {
		case strings.HasPrefix(data, screens.FormLevel):
			level := strings.TrimPrefix(data, screens.FormLevel)
			if draft.Step != screens.StepExperience || !validLevel(level) {
				return h.answer(ctx, q, "")
			}
			draft.Form.Experience = level
		case data == screens.FormSkip:
			switch draft.Step {
			case screens.StepHealth:
				draft.Form.Health = nil
			case screens.StepNotes:
				draft.Form.Notes = nil
			default:
				return h.answer(ctx, q, "")
			}
		case formAnswer(draft) == "":
			// Nothing to keep.
			return h.answer(ctx, q, "")
		}
		s, toast, err = h.formNext(ctx, l, chatID, userID, draft)
		if err != nil {
			return h.failedPress(ctx, l, q, err)
		}

	case data == screens.ProfilePassword || data == screens.PasswordCancel:
		if data == screens.PasswordCancel {
			h.Sessions.Cancel(chatID)
		}
		me, err := h.Profiles.Me(ctx, userID)
		if err != nil {
			return h.failedPress(ctx, l, q, err)
		}
		s = screens.PasswordScreen(l, me, "")

	case data == screens.PasswordStart:
		me, err := h.Profiles.Me(ctx, userID)
		if err != nil {
			return h.failedPress(ctx, l, q, err)
		}
		if !me.HasPassword && !screens.HasLogin(me) {
			s = screens.PasswordScreen(l, me, "")
			break
		}
		draft := passwordDraft{Step: screens.StepNew, Has: me.HasPassword}
		if me.HasPassword {
			draft.Step = screens.StepCurrent
		}
		s = h.passwordStep(l, chatID, draft, "")

	case data == screens.ProfileAwards:
		mine, err := h.Profiles.MyAchievements(ctx, userID)
		if err != nil {
			return h.failedPress(ctx, l, q, err)
		}
		leaders, err := h.Profiles.HallOfFame(ctx)
		if err != nil {
			h.Log.WarnContext(ctx, "hall of fame", slog.Any("error", err))
		}
		s = screens.Awards(l, mine, leaders, userID)

	case data == screens.ProfilePartner:
		me, err := h.Profiles.Me(ctx, userID)
		if err != nil {
			return h.failedPress(ctx, l, q, err)
		}
		s = screens.Partner(l, me, h.organizing(), "")

	case strings.HasPrefix(data, screens.PartnerApply):
		kind := strings.TrimPrefix(data, screens.PartnerApply)
		if kind != api.KindOrganizer && kind != api.KindDriver {
			return h.answer(ctx, q, "")
		}
		me, err := h.Profiles.ApplyAsPartner(ctx, userID, kind)
		note := l.T("✅ Заявка отправлена.")
		if api.StatusOf(err) == http.StatusConflict {
			// Already sent (or approved): show where it stands.
			me, err = h.Profiles.Me(ctx, userID)
			note = ""
		}
		if err != nil {
			return h.failedPress(ctx, l, q, err)
		}
		if note != "" {
			toast = l.T("Заявка отправлена")
		}
		s = screens.Partner(l, me, h.organizing(), note)

	default:
		return h.answer(ctx, q, "")
	}
	if err := h.answer(ctx, q, toast); err != nil {
		return err
	}
	return h.show(ctx, q, s)
}

func validLevel(level string) bool {
	for _, l := range screens.Levels {
		if l == level {
			return true
		}
	}
	return false
}

// formAnswer is the draft's answer to its current step ("" for none).
func formAnswer(d formDraft) string {
	switch d.Step {
	case screens.StepName:
		return d.Form.EmergencyName
	case screens.StepPhone:
		return d.Form.EmergencyPhone
	case screens.StepExperience:
		return d.Form.Experience
	case screens.StepHealth:
		if d.Form.Health != nil {
			return *d.Form.Health
		}
	case screens.StepNotes:
		if d.Form.Notes != nil {
			return *d.Form.Notes
		}
	}
	return ""
}

// formStep keeps the draft in the chat's session and asks its step.
func (h *Handlers) formStep(l i18n.Lang, chatID int64, d formDraft, problem string) screens.Screen {
	raw, _ := json.Marshal(d)
	h.Sessions.StartWith(chatID, session.FieldForm, string(raw))
	return screens.FormStep(l, d.Step, formAnswer(d), problem)
}

var nextFormStep = map[string]string{
	screens.StepName:       screens.StepPhone,
	screens.StepPhone:      screens.StepExperience,
	screens.StepExperience: screens.StepHealth,
	screens.StepHealth:     screens.StepNotes,
}

// formNext moves the draft past its step: asks the next one, or saves the
// form after the last (with a toast for a button press).
func (h *Handlers) formNext(ctx context.Context, l i18n.Lang, chatID int64, userID string, d formDraft) (screens.Screen, string, error) {
	if next, ok := nextFormStep[d.Step]; ok {
		d.Step = next
		return h.formStep(l, chatID, d, ""), "", nil
	}
	h.Sessions.Cancel(chatID)
	err := h.Profiles.SaveParticipantForm(ctx, userID, d.Form)
	if fields := api.FieldsOf(err); fields != nil {
		// The service refused an answer (reported under "form"): ask it
		// again — the phone, or else the name.
		d.Step = screens.StepName
		problem := l.T("Проверьте имя экстренного контакта.")
		for _, message := range fields {
			if strings.Contains(strings.ToLower(message), "phone") {
				d.Step, problem = screens.StepPhone, l.T("Похоже, это не номер телефона. Напишите его цифрами, например +996 555 123 456.")
			}
		}
		return h.formStep(l, chatID, d, problem), "", nil
	}
	if err != nil {
		return screens.Screen{}, "", err
	}
	return screens.FormScreen(l, &d.Form, l.T("✅ Анкета сохранена.")), l.T("Анкета сохранена"), nil
}

// formText takes a typed answer of the form being filled in.
func (h *Handlers) formText(ctx *th.Context, l i18n.Lang, msg telego.Message, arg string) error {
	var d formDraft
	if json.Unmarshal([]byte(arg), &d) != nil || msg.From == nil {
		return nil
	}
	chatID := msg.Chat.ID
	value := strings.TrimSpace(msg.Text)
	problem := ""
	switch d.Step {
	case screens.StepName:
		switch n := utf8.RuneCountInString(value); {
		case n == 0:
			problem = l.T("Напишите имя.")
		case n > nameMax:
			problem = l.F("Слишком длинно — не больше %d символов.", nameMax)
		default:
			d.Form.EmergencyName = value
		}
	case screens.StepPhone:
		if !phonePattern.MatchString(value) {
			problem = l.T("Похоже, это не номер телефона. Напишите его цифрами, например +996 555 123 456.")
		} else {
			d.Form.EmergencyPhone = value
		}
	case screens.StepExperience:
		problem = l.T("Выберите вариант кнопкой ниже.")
	default:
		if utf8.RuneCountInString(value) > textMax {
			problem = l.F("Слишком длинно — не больше %d символов.", textMax)
			break
		}
		var text *string
		if value != "" {
			text = &value
		}
		if d.Step == screens.StepHealth {
			d.Form.Health = text
		} else {
			d.Form.Notes = text
		}
	}
	if problem != "" {
		return h.send(ctx, chatID, h.formStep(l, chatID, d, problem))
	}
	userID, err := h.userOf(ctx, msg.From)
	if err != nil {
		return err
	}
	s, _, err := h.formNext(ctx, l, chatID, userID, d)
	if err != nil {
		return h.failed(ctx, l, chatID, err)
	}
	return h.send(ctx, chatID, s)
}

// passwordStep keeps the draft in the chat's session and asks its step.
func (h *Handlers) passwordStep(l i18n.Lang, chatID int64, d passwordDraft, problem string) screens.Screen {
	raw, _ := json.Marshal(d)
	h.Sessions.StartWith(chatID, session.FieldPassword, string(raw))
	return screens.PasswordStep(l, d.Step, problem)
}

// deleteMessage removes a message from the chat, best effort (a typed
// password must not stay in the chat's history).
func (h *Handlers) deleteMessage(ctx *th.Context, chatID int64, messageID int) {
	if err := ctx.Bot().DeleteMessage(ctx, tu.Delete(tu.ID(chatID), messageID)); err != nil {
		h.Log.WarnContext(ctx, "could not delete a password message", slog.Any("error", err))
	}
}

// passwordText takes a typed password: the current one, the new one, or
// the new one again, and then changes it.
func (h *Handlers) passwordText(ctx *th.Context, l i18n.Lang, msg telego.Message, arg string) error {
	chatID := msg.Chat.ID
	h.deleteMessage(ctx, chatID, msg.MessageID)
	var d passwordDraft
	if json.Unmarshal([]byte(arg), &d) != nil || msg.From == nil {
		return nil
	}
	value := msg.Text
	switch d.Step {
	case screens.StepCurrent:
		if value == "" {
			return h.send(ctx, chatID, h.passwordStep(l, chatID, d, l.T("Введите текущий пароль")))
		}
		d.Current, d.Step = value, screens.StepNew
		return h.send(ctx, chatID, h.passwordStep(l, chatID, d, ""))
	case screens.StepNew:
		switch {
		case utf8.RuneCountInString(value) < passwordMin:
			return h.send(ctx, chatID, h.passwordStep(l, chatID, d, l.F("Пароль слишком короткий — нужно не меньше %d символов.", passwordMin)))
		case len(value) > passwordMax:
			return h.send(ctx, chatID, h.passwordStep(l, chatID, d, l.T("Пароль слишком длинный — выберите покороче.")))
		}
		d.New, d.Step = value, screens.StepConfirm
		return h.send(ctx, chatID, h.passwordStep(l, chatID, d, ""))
	}

	if value != d.New {
		d.New, d.Step = "", screens.StepNew
		return h.send(ctx, chatID, h.passwordStep(l, chatID, d, l.T("Пароли не совпадают. Введите новый пароль ещё раз.")))
	}
	userID, err := h.userOf(ctx, msg.From)
	if err != nil {
		return err
	}
	var current *string
	if d.Has {
		current = &d.Current
	}
	err = h.Profiles.ChangePassword(ctx, userID, current, d.New)
	switch {
	case api.StatusOf(err) == http.StatusBadRequest && api.FieldsOf(err) == nil:
		// The current password was wrong: start over.
		d = passwordDraft{Step: screens.StepCurrent, Has: true}
		return h.send(ctx, chatID, h.passwordStep(l, chatID, d, l.T("❌ Неверный текущий пароль.")))
	case api.StatusOf(err) == http.StatusBadRequest:
		d.New, d.Step = "", screens.StepNew
		return h.send(ctx, chatID, h.passwordStep(l, chatID, d, l.T("Этот пароль не подходит — выберите другой.")))
	case api.StatusOf(err) == http.StatusConflict:
		// No email or phone to sign in with.
		me, err := h.Profiles.Me(ctx, userID)
		if err != nil {
			return h.failed(ctx, l, chatID, err)
		}
		return h.send(ctx, chatID, screens.PasswordScreen(l, me, ""))
	case err != nil:
		return h.reply(ctx, chatID, l.T("Не удалось сменить пароль. Попробуйте ещё раз."), nil)
	}
	me, err := h.Profiles.Me(ctx, userID)
	if err != nil {
		me = api.Me{HasPassword: true}
	}
	return h.send(ctx, chatID, screens.PasswordScreen(l, me,
		l.T("✅ Пароль изменён. На других устройствах нужно будет войти заново.")))
}
