package i18n_test

import (
	"testing"
	"time"

	"github.com/elvn5/sai/services/bot/internal/i18n"
)

func TestParse(t *testing.T) {
	for code, want := range map[string]i18n.Lang{
		"ru": i18n.RU, "ky": i18n.KY, "en": i18n.EN, "en-US": i18n.EN, "de": i18n.DE, "DE_at": i18n.DE,
		"": i18n.RU, "uk": i18n.RU, "kk": i18n.RU, "fr": i18n.EN, "tr": i18n.EN,
	} {
		if got := i18n.Parse(code); got != want {
			t.Errorf("Parse(%q) = %s, want %s", code, got, want)
		}
	}
}

func TestPlural(t *testing.T) {
	for n, want := range map[int]string{
		0: "выходов", 1: "выход", 2: "выхода", 4: "выхода", 5: "выходов", 11: "выходов", 12: "выходов",
		14: "выходов", 21: "выход", 22: "выхода", 101: "выход", 111: "выходов", 112: "выходов",
	} {
		if got := i18n.RU.N(n, "выход", "выхода", "выходов"); got != want {
			t.Errorf("N(%d) = %q, want %q", n, got, want)
		}
	}
	for _, tc := range []struct {
		l    i18n.Lang
		n    int
		want string
	}{
		{i18n.EN, 1, "outing"}, {i18n.EN, 0, "outings"}, {i18n.EN, 21, "outings"},
		{i18n.DE, 1, "Ausflug"}, {i18n.DE, 5, "Ausflüge"}, {i18n.KY, 1, "чыгуу"}, {i18n.KY, 5, "чыгуу"},
	} {
		if got := tc.l.N(tc.n, "выход", "выхода", "выходов"); got != tc.want {
			t.Errorf("%s.N(%d) = %q, want %q", tc.l, tc.n, got, tc.want)
		}
	}
}

func TestSay(t *testing.T) {
	m := i18n.M("Можно от 1 до %d человек", 50)
	if got := m.String(); got != "Можно от 1 до 50 человек" {
		t.Errorf("String = %q", got)
	}
	if got := i18n.EN.Say(m); got != "From 1 to 50 people" {
		t.Errorf("EN = %q", got)
	}
	if i18n.EN.Say(i18n.Msg{}) != "" {
		t.Error("an empty message says something")
	}
	if got := i18n.EN.T("not in the catalog"); got != "not in the catalog" {
		t.Errorf("unknown text = %q", got)
	}
}

func TestDates(t *testing.T) {
	sat := time.Date(2026, 10, 24, 6, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	next := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		l                                          i18n.Lang
		when, date, long, month, sameMonth, months string
	}{
		{i18n.RU, "сб, 24 окт., 06:00", "24 окт. 2026", "суббота, 24 октября", "Октябрь 2026", "24 — 26 октября", "24 октября — 2 ноября"},
		{i18n.KY, "иш, 24-окт., 06:00", "24-окт. 2026", "ишемби, 24-октябрь", "Октябрь 2026", "24 — 26-октябрь", "24-октябрь — 2-ноябрь"},
		{i18n.EN, "Sat, 24 Oct, 06:00", "24 Oct 2026", "Saturday, 24 October", "October 2026", "24 — 26 October", "24 October — 2 November"},
		{i18n.DE, "Sa., 24. Okt., 06:00", "24. Okt. 2026", "Samstag, 24. Oktober", "Oktober 2026", "24. — 26. Oktober", "24. Oktober — 2. November"},
	} {
		got := []string{tc.l.When(sat), tc.l.Date(sat), tc.l.LongDate(sat), tc.l.MonthYear(sat), tc.l.DateRange(sat, end), tc.l.DateRange(sat, next)}
		want := []string{tc.when, tc.date, tc.long, tc.month, tc.sameMonth, tc.months}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%s: %q, want %q", tc.l, got[i], want[i])
			}
		}
	}
	if got := i18n.RU.Weekdays(); got[0] != "Пн" || got[6] != "Вс" {
		t.Errorf("Weekdays = %v", got)
	}
}
