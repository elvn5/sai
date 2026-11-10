package handlers_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/cabinet"
	"github.com/elvn5/sai/services/bot/internal/handlers"
	"github.com/elvn5/sai/services/bot/internal/screens"
)

type cabinetAPI struct{}

func (cabinetAPI) CabinetRequest(_ context.Context, _ string, method, path string, _ any, out any) error {
	answer := `[]`
	if path == "/auth/user/me" {
		answer = `{"user":{"partnerStatus":"approved","partnerKind":"organizer"}}`
	}
	if strings.HasPrefix(path, "/crm/clients") {
		answer = `{"items":[],"total":0}`
	}
	if out != nil {
		return json.Unmarshal([]byte(answer), out)
	}
	return nil
}
func cabinetButton(t *testing.T, c call, label string) string {
	t.Helper()
	var m telego.InlineKeyboardMarkup
	if err := json.Unmarshal([]byte(c.markup()), &m); err != nil {
		t.Fatal(err)
	}
	for _, row := range m.InlineKeyboard {
		for _, b := range row {
			if b.Text == label || strings.TrimSuffix(b.Text, " *") == label {
				return b.CallbackData
			}
		}
	}
	t.Fatalf("missing %s: %s", label, c.markup())
	return ""
}
func TestCabinetRoutesAndProfileEditDoNotStealEachOthersInput(t *testing.T) {
	h, _, _ := newSections(t, func(h *handlers.Handlers) { h.Cabinet = cabinet.New(cabinetAPI{}, screens.Links{}) })
	h.message("/partner")
	home := h.expect("sendMessage")[0]
	if !strings.Contains(home.text(), "Туры и партнёры") {
		t.Fatal(home.text())
	}
	h.press(cabinetButton(t, home, "Клиенты"))
	list := h.expect("answerCallbackQuery", "editMessageText")[1]
	h.press(cabinetButton(t, list, "➕ Добавить"))
	form := h.expect("answerCallbackQuery", "editMessageText")[1]
	h.press(cabinetButton(t, form, "Имя"))
	h.expect("answerCallbackQuery", "editMessageText")
	// A profile edit must cancel pending CRM input, rather than overwrite a client.
	h.press("profile_edit_name")
	h.expect("answerCallbackQuery", "sendMessage")
	h.message("New profile name")
	reply := h.expect("sendMessage")[0]
	if !strings.Contains(reply.text(), "Имя обновлено") {
		t.Fatal(reply.text())
	}
	calls := strings.Join(h.users.calls(), "\n")
	if !strings.Contains(calls, "name u1 New profile name") {
		t.Fatal(calls)
	}
	// Forwarding a cabinet button to a group must never disclose account data.
	h.pressOn("k:home", &telego.Message{MessageID: 9, Chat: telego.Chat{ID: -100, Type: "group"}})
	h.expect("answerCallbackQuery")
	h.quiet()
	h.message("/cancel")
	if msg := h.expect("sendMessage")[0]; !strings.Contains(msg.text(), "отменено") {
		t.Fatal(msg.text())
	}
}
