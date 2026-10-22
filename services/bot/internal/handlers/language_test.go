package handlers_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/mymmrac/telego"

	"github.com/elvn5/sai/services/bot/internal/handlers"
	"github.com/elvn5/sai/services/bot/internal/users"
)

// fakeLanguages is the bot's database of chat languages.
type fakeLanguages struct {
	mu    sync.Mutex
	langs map[int64]string
}

func (f *fakeLanguages) Language(_ context.Context, chatID int64, fallback string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if l, ok := f.langs[chatID]; ok {
		return l, nil
	}
	f.langs[chatID] = fallback
	return fallback, nil
}

func (f *fakeLanguages) KnownLanguage(_ context.Context, chatID int64) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.langs[chatID], nil
}

func (f *fakeLanguages) SetLanguage(_ context.Context, chatID int64, lang string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.langs[chatID] = lang
	return nil
}

func (f *fakeLanguages) of(chatID int64) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.langs[chatID]
}

func withLanguages(langs *fakeLanguages) func(*handlers.Handlers) {
	return func(h *handlers.Handlers) { h.Languages = langs }
}

func (h *harness) messageFrom(from telego.User, text string) {
	h.nextID++
	h.updates <- telego.Update{UpdateID: h.nextID, Message: &telego.Message{
		MessageID: h.nextID, Chat: telego.Chat{ID: from.ID, Type: "private"}, From: &from, Text: text,
	}}
}

// A user whose Telegram is in English is greeted in English, and keeps it.
func TestLanguageFromTelegram(t *testing.T) {
	langs := &fakeLanguages{langs: map[int64]string{}}
	h := newHarness(t, miniApp, withLanguages(langs))
	english := sender
	english.LanguageCode = "en-GB"
	h.messageFrom(english, "/start")
	msg := h.expect("sendMessage")[0]
	if !strings.HasPrefix(msg.text(), "Hello, Айгерим! I'll help with guides") {
		t.Errorf("greeting: %q", msg.text())
	}
	if m := msg.markup(); !strings.Contains(m, `{"text":"👤 Profile"}`) || !strings.Contains(m, `"url":"https://ermekten.com/en"`) {
		t.Errorf("main menu: %s", m)
	}
	if langs.of(chatID) != "en" {
		t.Errorf("kept language: %q", langs.of(chatID))
	}

	// The English button works, and the profile is in English.
	h.users.account = users.Account{ID: "u1", FirstName: str("Айгерим")}
	h.messageFrom(english, "👤 Profile")
	if text := h.expect("sendMessage")[0].text(); text != "Your profile:\n\nName: Айгерим\nUsername: not set\nPhone: not set" {
		t.Errorf("profile: %q", text)
	}
}

// The user picks German: the picker goes, the menu button opens the German
// site, and the main menu comes back in German.
func TestPickLanguage(t *testing.T) {
	langs := &fakeLanguages{langs: map[int64]string{chatID: "ru"}}
	h := newHarness(t, miniApp, withLanguages(langs))

	h.message("/language")
	if m := h.expect("sendMessage")[0].markup(); !strings.Contains(m, `"text":"✅ 🇷🇺 Русский"`) || !strings.Contains(m, `"callback_data":"l:de"`) {
		t.Errorf("picker: %s", m)
	}

	h.press("l:de")
	calls := h.expect("answerCallbackQuery", "editMessageReplyMarkup", "setChatMenuButton", "sendMessage")
	if calls[0].params["text"] != "🇩🇪 Deutsch" {
		t.Errorf("toast: %v", calls[0].params)
	}
	if button, _ := calls[2].params["menu_button"].(map[string]any); button == nil ||
		button["web_app"].(map[string]any)["url"] != "https://ermekten.com/de" {
		t.Errorf("menu button: %v", calls[2].params)
	}
	if text := calls[3].text(); !strings.HasPrefix(text, "✅ Ich spreche jetzt Deutsch.") ||
		!strings.Contains(calls[3].markup(), `{"text":"🆘 Support"}`) {
		t.Errorf("confirmation: %q %s", text, calls[3].markup())
	}
	if langs.of(chatID) != "de" {
		t.Errorf("kept language: %q", langs.of(chatID))
	}

	// A language the bot doesn't speak is ignored.
	h.press("l:fr")
	h.expect("answerCallbackQuery")
	if langs.of(chatID) != "de" {
		t.Errorf("language after l:fr: %q", langs.of(chatID))
	}
}

// The profile's language button opens the picker.
func TestLanguageFromProfile(t *testing.T) {
	h := newHarness(t, miniApp)
	h.press("l:menu")
	calls := h.expect("answerCallbackQuery", "sendMessage")
	if !strings.HasPrefix(calls[1].text(), "🌐") || !strings.Contains(calls[1].markup(), `"callback_data":"l:ky"`) {
		t.Errorf("picker: %q %s", calls[1].text(), calls[1].markup())
	}
}
