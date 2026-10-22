package screens

import (
	"strings"
	"testing"
	"time"

	"github.com/elvn5/sai/services/bot/internal/i18n"
)

func TestDateRange(t *testing.T) {
	for _, tc := range []struct{ start, end, want string }{
		{"2027-06-12", "2027-06-14", "12 — 14 июня"},
		{"2027-06-30", "2027-07-02", "30 июня — 2 июля"},
		{"2027-12-30", "2028-01-03", "30 декабря — 3 января"},
		{"2027-06-12", "2027-06-12", "12 июня"},
		{"2027-06-12", "", "12 июня"},
		{"", "2027-06-12", ""},
	} {
		if got := dateRange(i18n.RU, tc.start, tc.end); got != tc.want {
			t.Errorf("dateRange(%q, %q) = %q, want %q", tc.start, tc.end, got, tc.want)
		}
	}
}

func TestWhenAndDay(t *testing.T) {
	// 23:30 UTC is already the next day in Bishkek.
	ts := time.Date(2026, 12, 31, 23, 30, 0, 0, time.UTC)
	if got := When(i18n.RU, ts); got != "пт, 1 янв., 05:30" {
		t.Errorf("When = %q", got)
	}
	if got := day(i18n.RU, ts); got != "1 янв. 2027" {
		t.Errorf("day = %q", got)
	}
	if !sameDay(ts, ts.Add(10*time.Hour)) || sameDay(ts, ts.Add(20*time.Hour)) {
		t.Error("sameDay")
	}
}

func TestDays(t *testing.T) {
	for _, tc := range []struct {
		l    i18n.Lang
		n    int
		want string
	}{
		{i18n.RU, 1, "1 день"}, {i18n.RU, 3, "3 дня"}, {i18n.RU, 7, "7 дней"},
		{i18n.EN, 1, "1 day"}, {i18n.EN, 3, "3 days"}, {i18n.DE, 2, "2 Tage"}, {i18n.KY, 5, "5 күн"},
	} {
		if got := days(tc.l, tc.n); got != tc.want {
			t.Errorf("days(%s, %d) = %q, want %q", tc.l, tc.n, got, tc.want)
		}
	}
}

func TestMoney(t *testing.T) {
	for n, want := range map[int]string{0: "0", 500: "500", 5200: "5 200", 18000: "18 000", 1234567: "1 234 567"} {
		if got := money(n); got != want {
			t.Errorf("money(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestClip(t *testing.T) {
	if got := clip("  short\n text ", 20); got != "short text" {
		t.Errorf("short: %q", got)
	}
	if got := clip("Перевал Калмак-Ашуу, табуны на джайлоо", 22); got != "Перевал Калмак-Ашуу…" {
		t.Errorf("at a word: %q", got)
	}
	// One long word: cut mid-word rather than lose everything.
	if got := clip(strings.Repeat("а", 30), 10); got != strings.Repeat("а", 10)+"…" {
		t.Errorf("long word: %q", got)
	}
}

func TestSeatsBar(t *testing.T) {
	for _, tc := range []struct {
		taken, seats int
		want         string
	}{
		{7, 10, "▰▰▰▰▰▰▰▱▱▱"},
		{0, 4, "▱▱▱▱"},
		{1, 50, "▰▱▱▱▱▱▱▱▱▱"}, // someone is in: never an empty bar
		{12, 10, "▰▰▰▰▰▰▰▰▰▰"},
		{0, 0, ""},
	} {
		if got := seatsBar(tc.taken, tc.seats); got != tc.want {
			t.Errorf("seatsBar(%d, %d) = %q, want %q", tc.taken, tc.seats, got, tc.want)
		}
	}
}
