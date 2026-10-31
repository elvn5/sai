package handlers_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/handlers"
)

// fakeProfiles stands in for the gateway's profile endpoints.
type fakeProfiles struct {
	mu       sync.Mutex
	log      []string
	me       api.Me
	form     *api.ParticipantForm
	password string
	mine     api.MyAchievements
	leaders  []api.HallEntry
}

func (f *fakeProfiles) record(s string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log = append(f.log, s)
}

func (f *fakeProfiles) calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.log...)
}

func (f *fakeProfiles) Me(context.Context, string) (api.Me, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.me, nil
}

func (f *fakeProfiles) ParticipantForm(context.Context, string) (*api.ParticipantForm, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.form, nil
}

func (f *fakeProfiles) SaveParticipantForm(_ context.Context, userID string, form api.ParticipantForm) error {
	f.record("form " + userID + " " + form.EmergencyName + " " + form.Experience)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.form = &form
	return nil
}

func (f *fakeProfiles) ChangePassword(_ context.Context, userID string, current *string, password string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.me.HasPassword && (current == nil || *current != f.password) {
		return &api.Error{Status: 400, Message: "Current password is incorrect"}
	}
	f.log = append(f.log, "password "+userID+" "+password)
	f.password, f.me.HasPassword = password, true
	return nil
}

func (f *fakeProfiles) MyAchievements(context.Context, string) (api.MyAchievements, error) {
	return f.mine, nil
}

func (f *fakeProfiles) HallOfFame(context.Context) ([]api.HallEntry, error) { return f.leaders, nil }

func (f *fakeProfiles) ApplyAsPartner(_ context.Context, userID, kind string) (api.Me, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.me.PartnerStatus == api.PartnerPending || f.me.PartnerStatus == api.PartnerApproved {
		return api.Me{}, &api.Error{Status: 409, Message: "Partner request already submitted"}
	}
	f.log = append(f.log, "apply "+userID+" "+kind)
	f.me.PartnerStatus, f.me.PartnerKind = api.PartnerPending, kind
	return f.me, nil
}

func newProfile(t *testing.T) (*harness, *fakeProfiles) {
	t.Helper()
	profiles := &fakeProfiles{me: api.Me{ID: "u1", FirstName: str("Айгерим"), Username: str("aigerim"),
		Email: str("aigerim@example.kg"), PartnerStatus: api.PartnerNone, PartnerKind: api.KindOrganizer}}
	h, _, _ := newSections(t, func(hs *handlers.Handlers) { hs.Profiles = profiles })
	return h, profiles
}

func TestProfileSection(t *testing.T) {
	h, profiles := newProfile(t)
	profiles.mine = api.MyAchievements{Code: "AB12CD34", Points: 30, Awards: []api.Award{{Achievement: api.Achievement{Points: 30}}}}
	h.message("👤 Профиль")
	msg := h.expect("sendMessage")[0]
	want := "👤 <b>Айгерим</b> · @aigerim\n\n📞 Телефон: не указан\n✉️ Email: aigerim@example.kg\n🔑 Пароль: не задан\n\n" +
		"🎒 Туров пройдено: 0 · впереди: 1\n🏆 Достижений: 1 · очков: 30\n📝 Анкета участника: не заполнена\n🤝 Партнёрство: нет"
	if msg.text() != want {
		t.Errorf("profile:\n%s\n--- want:\n%s", msg.text(), want)
	}
	for _, data := range []string{"profile_edit_name", "profile_edit_phone", "p:form", "p:pw", "p:aw", "p:pa", "o:list:u", "t:f:0", "l:menu"} {
		if !strings.Contains(msg.markup(), `"callback_data":"`+data+`"`) {
			t.Errorf("no %s button: %s", data, msg.markup())
		}
	}

	// Back from a sub-screen edits the message into the profile again.
	h.press("p:home")
	if edit := h.expect("answerCallbackQuery", "editMessageText")[1]; edit.text() != want {
		t.Errorf("back: %q", edit.text())
	}
}

func TestProfileFormFlow(t *testing.T) {
	h, profiles := newProfile(t)
	h.press("p:form")
	if edit := h.expect("answerCallbackQuery", "editMessageText")[1]; !strings.Contains(edit.text(), "Анкета пока не заполнена.") ||
		!strings.Contains(edit.markup(), `"callback_data":"p:fe"`) {
		t.Errorf("empty form: %q %s", edit.text(), edit.markup())
	}

	h.press("p:fe")
	if edit := h.expect("answerCallbackQuery", "editMessageText")[1]; !strings.Contains(edit.text(), "1 из 5") ||
		strings.Contains(edit.markup(), "p:fk") {
		t.Errorf("step 1: %q %s", edit.text(), edit.markup())
	}
	h.message("  Мама ")
	if text := h.expect("sendMessage")[0].text(); !strings.Contains(text, "2 из 5") {
		t.Errorf("step 2: %q", text)
	}
	h.message("позвоните маме")
	if text := h.expect("sendMessage")[0].text(); !strings.HasPrefix(text, "Похоже, это не номер телефона.") || !strings.Contains(text, "2 из 5") {
		t.Errorf("bad phone: %q", text)
	}
	h.message("+996 555 000111")
	step3 := h.expect("sendMessage")[0]
	if !strings.Contains(step3.text(), "3 из 5") || !strings.Contains(step3.markup(), `"callback_data":"p:fx:hiking"`) {
		t.Errorf("step 3: %q %s", step3.text(), step3.markup())
	}
	h.message("немного")
	if text := h.expect("sendMessage")[0].text(); !strings.HasPrefix(text, "Выберите вариант кнопкой ниже.") {
		t.Errorf("typed level: %q", text)
	}
	h.press("p:fx:hiking")
	if edit := h.expect("answerCallbackQuery", "editMessageText")[1]; !strings.Contains(edit.text(), "4 из 5") {
		t.Errorf("step 4: %q", edit.text())
	}
	h.press("p:fs")
	if edit := h.expect("answerCallbackQuery", "editMessageText")[1]; !strings.Contains(edit.text(), "5 из 5") {
		t.Errorf("step 5: %q", edit.text())
	}
	h.message("Беру своего пса")
	saved := h.expect("sendMessage")[0]
	if !strings.HasPrefix(saved.text(), "✅ Анкета сохранена.\n\n📝 <b>Анкета участника</b>") ||
		!strings.Contains(saved.text(), "🆘 <b>Экстренный контакт:</b> Мама, +996 555 000111") ||
		!strings.Contains(saved.text(), "🧗 <b>Опыт в горах:</b> Однодневные походы") ||
		strings.Contains(saved.text(), "Здоровье") ||
		!strings.Contains(saved.text(), "Беру своего пса") {
		t.Errorf("saved: %q", saved.text())
	}
	if f := profiles.form; f == nil || f.Health != nil || f.Notes == nil || *f.Notes != "Беру своего пса" {
		t.Errorf("form: %+v", f)
	}

	// Editing keeps the answers given before, one button each.
	h.press("p:fe")
	if edit := h.expect("answerCallbackQuery", "editMessageText")[1]; !strings.Contains(edit.markup(), `"text":"➡️ Оставить: Мама"`) {
		t.Errorf("keep: %s", edit.markup())
	}
	h.press("p:fk")
	h.expect("answerCallbackQuery", "editMessageText")
	h.press("p:fk")
	if edit := h.expect("answerCallbackQuery", "editMessageText")[1]; !strings.Contains(edit.markup(), `"text":"✅ Однодневные походы"`) {
		t.Errorf("current level: %s", edit.markup())
	}
	h.press("p:fx:trekking")
	h.expect("answerCallbackQuery", "editMessageText")
	h.message("Аллергия на орехи")
	h.expect("sendMessage")
	h.press("p:fk")
	calls := h.expect("answerCallbackQuery", "editMessageText")
	if calls[0].params["text"] != "Анкета сохранена" || !strings.Contains(calls[1].text(), "Многодневные треки") ||
		!strings.Contains(calls[1].text(), "Аллергия на орехи") {
		t.Errorf("saved again: %v %q", calls[0].params, calls[1].text())
	}

	// A button of a form no longer being filled in only says so.
	h.press("p:fk")
	if answer := h.expect("answerCallbackQuery")[0]; answer.params["text"] != "Время вышло — начните заполнять анкету заново." {
		t.Errorf("stale: %v", answer.params)
	}
	h.quiet()

	// Cancelling leaves the form as it was.
	h.press("p:fe")
	h.expect("answerCallbackQuery", "editMessageText")
	h.press("p:fc")
	h.expect("answerCallbackQuery", "editMessageText")
	h.message("Не имя")
	h.quiet()
	if got := strings.Join(profiles.calls(), ","); got != "form u1 Мама hiking,form u1 Мама trekking" {
		t.Errorf("saved: %s", got)
	}
}

func TestProfilePasswordFlow(t *testing.T) {
	h, profiles := newProfile(t)
	profiles.me.HasPassword, profiles.password = true, "old-secret"
	h.press("p:pw")
	if edit := h.expect("answerCallbackQuery", "editMessageText")[1]; !strings.Contains(edit.text(), "Пароль задан") ||
		!strings.Contains(edit.markup(), `"text":"🔑 Сменить пароль","callback_data":"p:pc"`) &&
			!strings.Contains(edit.markup(), `"callback_data":"p:pc","text":"🔑 Сменить пароль"`) {
		t.Errorf("password: %q %s", edit.text(), edit.markup())
	}

	// type sends a password: the bot deletes it and asks the next step.
	typePassword := func(password, want string) {
		t.Helper()
		h.message(password)
		calls := h.expect("deleteMessage", "sendMessage")
		if calls[0].params["message_id"] != float64(h.nextID) {
			t.Errorf("deleted %v, want message %d", calls[0].params["message_id"], h.nextID)
		}
		if !strings.HasPrefix(calls[1].text(), want) {
			t.Errorf("after %q: %q, want %q…", password, calls[1].text(), want)
		}
	}

	h.press("p:pc")
	if edit := h.expect("answerCallbackQuery", "editMessageText")[1]; !strings.HasPrefix(edit.text(), "🔑 <b>Пароль</b>\n\nВведите текущий пароль.") {
		t.Errorf("current: %q", edit.text())
	}
	typePassword("wrong", "🔑 <b>Пароль</b>\n\nВведите новый пароль")
	typePassword("short", "Пароль слишком короткий — нужно не меньше 8 символов.")
	typePassword("new-secret", "🔑 <b>Пароль</b>\n\nПовторите новый пароль.")
	typePassword("other-secret", "Пароли не совпадают.")
	typePassword("new-secret", "🔑 <b>Пароль</b>\n\nПовторите новый пароль.")
	typePassword("new-secret", "❌ Неверный текущий пароль.")
	typePassword("old-secret", "🔑 <b>Пароль</b>\n\nВведите новый пароль")
	typePassword("new-secret", "🔑 <b>Пароль</b>\n\nПовторите новый пароль.")
	typePassword("new-secret", "✅ Пароль изменён.")
	if got := strings.Join(profiles.calls(), ","); got != "password u1 new-secret" {
		t.Errorf("changed: %s", got)
	}
	h.message("new-secret")
	h.quiet()

	// A Telegram-only account first needs a phone.
	profiles.me = api.Me{ID: "u1", PartnerStatus: api.PartnerNone}
	h.press("p:pw")
	if edit := h.expect("answerCallbackQuery", "editMessageText")[1]; !strings.Contains(edit.text(), "Вы входите через Telegram.") ||
		!strings.Contains(edit.markup(), "profile_edit_phone") || strings.Contains(edit.markup(), "p:pc") {
		t.Errorf("no login: %q %s", edit.text(), edit.markup())
	}
	h.press("p:pc")
	if edit := h.expect("answerCallbackQuery", "editMessageText")[1]; strings.Contains(edit.markup(), "p:px") {
		t.Errorf("started without a login: %s", edit.markup())
	}
}

func TestProfileAwardsAndPartner(t *testing.T) {
	h, profiles := newProfile(t)
	profiles.mine = api.MyAchievements{Code: "AB12CD34", Points: 80, Awards: []api.Award{
		{Achievement: api.Achievement{ID: "a1", Title: map[string]string{"ru": "Первая вершина"}, Points: 50}},
		{Achievement: api.Achievement{ID: "a2", Title: map[string]string{"ru": "Ранняя пташка"}, Points: 30}},
	}}
	profiles.leaders = []api.HallEntry{
		{Rank: 1, User: api.Person{ID: "u7", FirstName: str("Айбек")}, Points: 300},
		{Rank: 2, User: api.Person{ID: "u1", FirstName: str("Айгерим")}, Points: 80},
	}
	h.press("p:aw")
	want := "🏆 <b>Достижения</b>\n\n⭐ Очков: 80 · достижений: 2 · место: #2\n\n" +
		"🏅 Первая вершина · 50 очков\n🏅 Ранняя пташка · 30 очков\n\n" +
		"🎟 Ваш код путешественника: <code>AB12CD34</code>\n" +
		"Покажите код партнёру, чтобы получить достижение. Партнёры вручают их за пройденные маршруты, поездки и вклад в сообщество.\n\n" +
		"👑 <b>Лидеры</b>\n🥇 Айбек · 300\n🥈 Айгерим · 80"
	if edit := h.expect("answerCallbackQuery", "editMessageText")[1]; edit.text() != want {
		t.Errorf("awards:\n%s\n--- want:\n%s", edit.text(), want)
	}

	h.press("p:pa")
	if edit := h.expect("answerCallbackQuery", "editMessageText")[1]; !strings.Contains(edit.markup(), `"callback_data":"p:ap:driver"`) {
		t.Errorf("partner: %s", edit.markup())
	}
	h.press("p:ap:driver")
	calls := h.expect("answerCallbackQuery", "editMessageText")
	if calls[0].params["text"] != "Заявка отправлена" ||
		!strings.Contains(calls[1].text(), "⏳ Ваша заявка водителя на рассмотрении у администратора.") ||
		strings.Contains(calls[1].markup(), "p:ap:") {
		t.Errorf("applied: %v %q %s", calls[0].params, calls[1].text(), calls[1].markup())
	}
	// Pressing again (an old message) shows where it stands.
	h.press("p:ap:organizer")
	calls = h.expect("answerCallbackQuery", "editMessageText")
	if _, toast := calls[0].params["text"]; toast || strings.HasPrefix(calls[1].text(), "✅") {
		t.Errorf("again: %v %q", calls[0].params, calls[1].text())
	}
	if got := strings.Join(profiles.calls(), ","); got != "apply u1 driver" {
		t.Errorf("applications: %s", got)
	}
}
