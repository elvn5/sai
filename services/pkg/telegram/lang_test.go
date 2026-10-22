package telegram_test

import (
	"testing"
	"testing/fstest"
	"time"

	"github.com/elvn5/sai/services/pkg/telegram"
)

func TestParseLang(t *testing.T) {
	for code, want := range map[string]telegram.Lang{"ru": telegram.RU, "ky": telegram.KY, "en": telegram.EN, "de": telegram.DE, "": telegram.RU, "fr": telegram.RU} {
		if got := telegram.ParseLang(code); got != want {
			t.Errorf("%q: %q", code, got)
		}
	}
}

func TestCatalog(t *testing.T) {
	fsys := fstest.MapFS{
		"locales/ky.json": {Data: []byte(`{"Привет, %s": "Салам, %s"}`)},
		"locales/en.json": {Data: []byte(`{"Привет, %s": "Hi, %s", "Пусто": ""}`)},
		"locales/de.json": {Data: []byte(`{}`)},
	}
	c, err := telegram.LoadCatalog(fsys, "locales")
	if err != nil {
		t.Fatal(err)
	}
	for l, want := range map[telegram.Lang]string{telegram.RU: "Привет, Айбек", telegram.KY: "Салам, Айбек", telegram.EN: "Hi, Айбек", telegram.DE: "Привет, Айбек"} {
		if got := c.F(l, "Привет, %s", "Айбек"); got != want {
			t.Errorf("%s: %q", l, got)
		}
	}
	if got := c.T(telegram.EN, "Пусто"); got != "Пусто" {
		t.Errorf("empty translation: %q", got)
	}
	if _, err := telegram.LoadCatalog(fstest.MapFS{}, "locales"); err == nil {
		t.Error("missing catalogs loaded")
	}
}

func TestWhen(t *testing.T) {
	at := time.Date(2026, 10, 24, 6, 0, 0, 0, time.UTC)
	for l, want := range map[telegram.Lang]string{
		telegram.RU: "сб, 24 окт., 06:00",
		telegram.KY: "иш, 24-окт., 06:00",
		telegram.EN: "Sat, 24 Oct, 06:00",
		telegram.DE: "Sa., 24. Okt., 06:00",
	} {
		if got := l.When(at); got != want {
			t.Errorf("%s: %q", l, got)
		}
	}
}
