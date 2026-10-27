package resetmsg

import (
	"strings"
	"testing"

	"github.com/elvn5/sai/services/pkg/telegram"
)

// Every text has a translation in every language, and the catalogs hold
// nothing else.
func TestCatalogsAreComplete(t *testing.T) {
	for _, l := range telegram.Langs[1:] {
		for _, text := range Texts {
			if texts[l][text] == "" {
				t.Errorf("%s: no translation for %q", l, text)
			}
		}
		if len(texts[l]) != len(Texts) {
			t.Errorf("%s: %d entries, want %d", l, len(texts[l]), len(Texts))
		}
	}
}

func TestLink(t *testing.T) {
	if got := Link("https://ermekten.com/", telegram.RU, "abc"); got != "https://ermekten.com/reset-password#abc" {
		t.Errorf("ru: %s", got)
	}
	if got := Link("https://ermekten.com", telegram.EN, "abc"); got != "https://ermekten.com/en/reset-password#abc" {
		t.Errorf("en: %s", got)
	}
}

func TestMessagesCarryTheLink(t *testing.T) {
	link := Link("https://ermekten.com", telegram.DE, "tok")
	subject, body := Email(telegram.DE, link)
	if subject != "Ermekten-Passwort zurücksetzen" || !strings.Contains(body, `href="`+link+`"`) {
		t.Errorf("email: %s / %s", subject, body)
	}
	text, button := Telegram(telegram.KY, link)
	if !strings.Contains(text, "Сырсөздү калыбына келтирүү") || button.URL != link || button.Text != "Жаңы сырсөз коюу" {
		t.Errorf("telegram: %s / %+v", text, button)
	}
}
