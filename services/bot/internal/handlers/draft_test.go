package handlers_test

import (
	"strings"
	"testing"
	"time"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/api"
)

// pressAt presses a button under the bot's message id.
func (h *harness) pressAt(id int, data string) {
	h.pressOn(data, &telego.Message{MessageID: id, Chat: telego.Chat{ID: chatID, Type: "private"}})
}

// The whole new outing, in the chat: buttons edit the draft's message, a
// typed answer brings the next step as a new message (the fake Bot API
// numbers it 7) and takes the buttons off the old one.
func TestNewOutingInTheChat(t *testing.T) {
	h, content, _ := newSections(t)
	h.press("d:new")
	calls := h.expect("answerCallbackQuery", "editMessageText")
	if !strings.HasPrefix(calls[1].text(), "➕ <b>Новый выход</b> · шаг 1 из 12") || calls[1].params["message_id"] != float64(5) ||
		!strings.Contains(calls[1].markup(), `"callback_data":"d:trail:0","text":"Пик Учитель"`) {
		t.Fatalf("route step: %q %s", calls[1].text(), calls[1].markup())
	}

	// A typed route name is a search.
	h.message("ала")
	calls = h.expect("editMessageReplyMarkup", "sendMessage")
	if calls[0].params["message_id"] != float64(5) || !strings.Contains(calls[1].text(), "Нашлось по «ала»:") ||
		!strings.Contains(calls[1].markup(), `"callback_data":"d:trail:0","text":"Ала-Кёль"`) {
		t.Fatalf("search: %v %q %s", calls[0].params, calls[1].text(), calls[1].markup())
	}
	// The old message's buttons are stale now.
	h.press("d:back")
	if c := h.expect("editMessageReplyMarkup", "answerCallbackQuery"); !strings.HasPrefix(c[1].params["text"].(string), "Черновик устарел") {
		t.Errorf("stale press: %v", c[1].params)
	}

	h.pressAt(7, "d:trail:0")
	calls = h.expect("answerCallbackQuery", "editMessageText")
	if !strings.Contains(calls[1].text(), "Как назовём выход?") || !strings.Contains(calls[1].markup(), `"callback_data":"d:title:trail","text":"«Ала-Кёль»"`) {
		t.Fatalf("title step: %q %s", calls[1].text(), calls[1].markup())
	}
	h.message("Озеро Ала-Кёль на выходные")
	if c := h.expect("editMessageReplyMarkup", "sendMessage"); !strings.Contains(c[1].text(), "Когда выходим?") {
		t.Fatalf("date step: %q", c[1].text())
	}
	for _, step := range []struct{ data, next string }{
		{"d:date:2026-10-24", "Во сколько сбор?"},
		{"d:time:0600", "Когда вернётесь?"},
		{"d:ret:+1800", "Где сбор?"},
	} {
		h.pressAt(7, step.data)
		if c := h.expect("answerCallbackQuery", "editMessageText"); !strings.Contains(c[1].text(), step.next) {
			t.Fatalf("%s: %q", step.data, c[1].text())
		}
	}
	// A required text can't be skipped.
	h.pressAt(7, "d:skip")
	if c := h.expect("answerCallbackQuery", "editMessageText"); c[0].params["text"] != "Напишите ответ сообщением" {
		t.Errorf("skip a required step: %v", c[0].params)
	}
	h.message("ТЦ «Вефа»")
	h.expect("editMessageReplyMarkup", "sendMessage")
	// A wrong typed answer is explained; the step stays.
	h.message("на машине")
	if c := h.expect("sendMessage"); c[0].text() != "Выберите вариант кнопкой выше" {
		t.Errorf("text on a buttons step: %q", c[0].text())
	}
	for _, data := range []string{"d:tr:bus", "d:seats:6", "d:cost:0", "d:lvl:intermediate", "d:skip", "d:contact:me"} {
		h.pressAt(7, data)
		h.expect("answerCallbackQuery", "editMessageText")
	}
	h.pressAt(7, "d:fields")
	if c := h.expect("answerCallbackQuery", "editMessageText"); !strings.Contains(c[1].markup(), `"d:go:seats"`) {
		t.Errorf("fields: %s", c[1].markup())
	}
	h.pressAt(7, "d:review")
	review := h.expect("answerCallbackQuery", "editMessageText")[1]
	if !strings.HasPrefix(review.text(), "👀 <b>Проверьте выход</b>\n\n🥾 <b>Озеро Ала-Кёль на выходные</b>\n🏔 Ала-Кёль · нужен опыт\n\n"+
		"📅 сб, 24 окт., 06:00 → вс, 25 окт., 18:00\n📍 Сбор: ТЦ «Вефа»\n🚗 Общественным транспортом\n💳 Бесплатно\n👥 Мест: 6\n💬 Связь: @aigerim") {
		t.Errorf("review:\n%s", review.text())
	}

	// The API refuses the start (it passed meanwhile): back to that step.
	content.saveErr = &api.Error{Status: 400, Fields: map[string]string{"startsAt": "The start must be in the next 6 months"}}
	h.pressAt(7, "d:save")
	if c := h.expect("answerCallbackQuery", "editMessageText"); c[0].params["text"] != "Сбор — не в прошлом и не дальше чем через полгода" ||
		!strings.Contains(c[1].text(), "Во сколько сбор?") {
		t.Errorf("refused: %v %q", c[0].params, c[1].text())
	}
	content.saveErr = nil
	h.pressAt(7, "d:time:0630")
	h.expect("answerCallbackQuery", "editMessageText")
	h.pressAt(7, "d:save")
	calls = h.expect("answerCallbackQuery", "editMessageText")
	if calls[0].params["text"] != "🎉 Выход опубликован!" || !strings.HasSuffix(calls[1].text(), "⭐ Вы организатор") ||
		!strings.Contains(calls[1].markup(), `"o:edit:u:new"`) {
		t.Errorf("published: %v %q", calls[0].params, calls[1].text())
	}
	got := content.calls()
	want := `create u1 {"trailId":"tr2","title":"Озеро Ала-Кёль на выходные","startsAt":"2026-10-24T06:30:00+06:00",` +
		`"returnsAt":"2026-10-25T18:00:00+06:00","meetingPoint":"ТЦ «Вефа»","transport":"bus","seats":6,"costPerPerson":0,` +
		`"level":"intermediate","description":null,"contact":"@aigerim"}`
	if got[len(got)-1] != want {
		t.Errorf("created:\n%s\nwant:\n%s", got[len(got)-1], want)
	}
	// Published: the draft is gone.
	h.pressAt(7, "d:review")
	h.expect("editMessageReplyMarkup", "answerCallbackQuery")
	h.message("ещё текст")
	h.quiet()
}

func TestCancelNewOuting(t *testing.T) {
	h, content, _ := newSections(t)
	h.press("d:new")
	h.expect("answerCallbackQuery", "editMessageText")
	h.press("d:cancel")
	if c := h.expect("answerCallbackQuery", "editMessageText"); !strings.HasPrefix(c[1].text(), "Выход не создан.") ||
		!strings.Contains(c[1].markup(), `"d:new"`) {
		t.Errorf("cancelled: %q", c[1].text())
	}
	for _, call := range content.calls() {
		if strings.HasPrefix(call, "create") {
			t.Errorf("created anyway: %s", call)
		}
	}
}

func TestOrganizerEditsAndCancels(t *testing.T) {
	h, content, _ := newSections(t)
	content.outings["o3"] = api.Outing{ID: "o3", Title: "Кёль-Тор", StartsAt: now.Add(72 * time.Hour), Status: "open", Seats: 10, Taken: 4,
		MeetingPoint: "Вефа", Transport: "car", Level: "beginner", IsOrganizer: true,
		Participants: []api.Person{{FirstName: str("Аня")}}}
	h.press("o:view:u:o3")
	card := h.expect("answerCallbackQuery", "editMessageText")[1]
	if !strings.Contains(card.text(), "👥 Идут: Аня") || !strings.Contains(card.markup(), `"o:edit:u:o3"`) || !strings.Contains(card.markup(), `"o:cask:u:o3"`) {
		t.Fatalf("organizer card: %q %s", card.text(), card.markup())
	}

	h.press("o:edit:u:o3")
	if c := h.expect("answerCallbackQuery", "editMessageText"); !strings.HasPrefix(c[1].text(), "✏️ <b>Изменение выхода</b>") ||
		!strings.Contains(c[1].markup(), `"text":"✅ Сохранить"`) {
		t.Fatalf("edit: %q %s", c[1].text(), c[1].markup())
	}
	h.press("d:go:seats")
	h.expect("answerCallbackQuery", "editMessageText")
	h.press("d:seats:3")
	if c := h.expect("answerCallbackQuery", "editMessageText"); c[0].params["text"] != "Уже записались 4 — мест не может быть меньше" {
		t.Errorf("below taken: %v", c[0].params)
	}
	h.message("12")
	h.expect("editMessageReplyMarkup", "sendMessage")
	h.pressAt(7, "d:save")
	if c := h.expect("answerCallbackQuery", "editMessageText"); c[0].params["text"] != "✅ Изменения сохранены" {
		t.Errorf("saved: %v", c[0].params)
	}
	got := content.calls()
	if last := got[len(got)-1]; !strings.HasPrefix(last, `update u1 o3 {"trailId":null,"title":"Кёль-Тор","startsAt":"2026-10-23T10:00:00+06:00",`) ||
		!strings.Contains(last, `"seats":12`) {
		t.Errorf("update: %s", last)
	}

	// Closed meanwhile: said so, the draft dropped.
	h.press("o:edit:u:o3")
	h.expect("answerCallbackQuery", "editMessageText")
	content.saveErr = &api.Error{Status: 409, Message: "This outing is closed"}
	h.press("d:save")
	if c := h.expect("editMessageReplyMarkup", "answerCallbackQuery"); c[1].params["text"] != "Выход уже прошёл или отменён — изменить его нельзя" {
		t.Errorf("closed: %v", c[1].params)
	}
	content.saveErr = nil

	h.press("o:cask:u:o3")
	if c := h.expect("answerCallbackQuery", "editMessageText"); !strings.HasPrefix(c[1].text(), "Отменить выход «Кёль-Тор»?") {
		t.Errorf("confirm: %q", c[1].text())
	}
	h.press("o:cancel:u:o3")
	calls := h.expect("answerCallbackQuery", "editMessageText")
	if calls[0].params["text"] != "Выход отменён. Участники получат уведомление." || !strings.Contains(calls[1].text(), "❌ Выход отменён организатором") ||
		strings.Contains(calls[1].markup(), "o:edit") {
		t.Errorf("cancelled: %v %q", calls[0].params, calls[1].text())
	}

	// Not the organizer: no edit.
	h.press("o:edit:u:o1")
	if c := h.expect("answerCallbackQuery", "editMessageText"); !strings.HasPrefix(c[1].text(), "🥾 <b>Пик Учитель</b>") {
		t.Errorf("edit someone else's: %q", c[1].text())
	}
}

// A notification's button opens the card as a new message, under it.
func TestNotificationOpensCard(t *testing.T) {
	h, _, _ := newSections(t)
	h.press("o:view:n:o1")
	msg := h.expect("answerCallbackQuery", "sendMessage")[1]
	if !strings.HasPrefix(msg.text(), "🥾 <b>Пик Учитель</b>") || !strings.Contains(msg.markup(), `"o:ask:u:o1"`) {
		t.Errorf("card: %q %s", msg.text(), msg.markup())
	}
}

func TestReadInTheChat(t *testing.T) {
	h, content, _ := newSections(t)
	h.press("t:more:n:1:0")
	more := h.expect("answerCallbackQuery", "editMessageText")[1]
	if !strings.HasPrefix(more.text(), "🧭 <b>Ала-Кёль</b>") || !strings.Contains(more.text(), "Описание тура Ала-Кёль") ||
		!strings.Contains(more.markup(), `"t:save:n:1:t2:more"`) || !strings.Contains(more.markup(), `"t:n:1"`) {
		t.Errorf("tour: %q %s", more.text(), more.markup())
	}
	// A favorite from the description stays on it.
	h.press("t:save:n:1:t2:more")
	if c := h.expect("answerCallbackQuery", "editMessageText"); !strings.Contains(c[1].markup(), `"t:drop:n:1:t2:more"`) || !content.favorites["t2"] {
		t.Errorf("save from the description: %s", c[1].markup())
	}
	// A new-tour alert opens the tour under it.
	h.press("t:open:t1")
	if c := h.expect("answerCallbackQuery", "sendMessage"); !strings.HasPrefix(c[1].text(), "🧭 <b>Сон-Кёль</b>") {
		t.Errorf("alert: %q", c[1].text())
	}
	h.press("t:open:gone")
	if c := h.expect("answerCallbackQuery"); c[0].params["text"] != "Этого тура уже нет среди ближайших" {
		t.Errorf("gone tour: %v", c[0].params)
	}

	h.press("b:read:p1:0")
	if c := h.expect("answerCallbackQuery", "editMessageText"); !strings.HasPrefix(c[1].text(), "📰 <b>Чек-лист</b>\n\n<b>Рюкзак</b>\n\nЛитров на 30.") ||
		!strings.Contains(c[1].markup(), `"b:0"`) {
		t.Errorf("post: %q %s", c[1].text(), c[1].markup())
	}
	h.press("b:read:gone:0")
	if c := h.expect("answerCallbackQuery"); c[0].params["text"] != "Эта статья больше не опубликована" {
		t.Errorf("gone post: %v", c[0].params)
	}
	h.quiet()
}

func TestConditionReport(t *testing.T) {
	h, content, _ := newSections(t)
	content.outings["o4"] = api.Outing{ID: "o4", Title: "Ала-Кёль", StartsAt: now.Add(-48 * time.Hour), Status: "open", Seats: 8, Taken: 3,
		IsJoined: true, Trail: &api.TrailRef{Slug: "ala-kol", Name: "Ала-Кёль"}}
	h.press("c:s:o4")
	if c := h.expect("answerCallbackQuery", "editMessageText"); !strings.Contains(c[1].text(), "Какой была тропа?") {
		t.Errorf("state: %q", c[1].text())
	}
	h.press("c:t:o4:1")
	if c := h.expect("answerCallbackQuery", "editMessageText"); !strings.Contains(c[1].text(), "Тропа: 💧 Мокро, грязь") {
		t.Errorf("traction: %q", c[1].text())
	}
	h.press("c:d:o4:1:2")
	if c := h.expect("answerCallbackQuery", "editMessageText"); c[0].params["text"] != "🙏 Спасибо за сводку!" ||
		!strings.HasPrefix(c[1].text(), "🙏 <b>Спасибо!</b>") {
		t.Errorf("sent: %v %q", c[0].params, c[1].text())
	}
	if got := content.calls(); got[len(got)-1] != "condition u1 ala-kol 2026-10-18 wet crampons" {
		t.Errorf("report: %v", got)
	}

	// Not finished yet, or answers out of range: nothing sent.
	h.press("c:s:o1")
	if c := h.expect("answerCallbackQuery"); c[0].params["text"] != "Сводку можно оставить в течение 30 дней после выхода" {
		t.Errorf("upcoming: %v", c[0].params)
	}
	for _, data := range []string{"c:d:o4:9:0", "c:t:o4:x", "c:x"} {
		h.press(data)
		if c := h.expect("answerCallbackQuery"); c[0].params["text"] != nil {
			t.Errorf("%s: %v", data, c[0].params)
		}
	}
	h.quiet()
}

// Everything is done in the chat: the section buttons Telegram gets never
// open the Mini App or the site (the main menu's "Открыть Ermekten" does).
func TestSectionsStayInTheChat(t *testing.T) {
	h, content, _ := newSections(t)
	content.outings = map[string]api.Outing{}
	h.press("o:find:0")
	m := h.expect("answerCallbackQuery", "editMessageText")[1].markup()
	if !strings.Contains(m, `{"callback_data":"d:new","text":"➕ Собрать выход"}`) || strings.Contains(m, "web_app") || strings.Contains(m, `"url"`) {
		t.Errorf("create button: %s", m)
	}
	h.message("/start")
	if m := h.expect("sendMessage")[0].markup(); !strings.Contains(m, `{"text":"🏔 Открыть Ermekten","web_app":{"url":"`+miniApp+`"}}`) {
		t.Errorf("main menu: %s", m)
	}
}
