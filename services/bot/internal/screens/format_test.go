package screens

import (
	"strings"
	"testing"
	"time"
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
		if got := dateRange(tc.start, tc.end); got != tc.want {
			t.Errorf("dateRange(%q, %q) = %q, want %q", tc.start, tc.end, got, tc.want)
		}
	}
}

func TestWhenAndDay(t *testing.T) {
	// 23:30 UTC is already the next day in Bishkek.
	ts := time.Date(2026, 12, 31, 23, 30, 0, 0, time.UTC)
	if got := When(ts); got != "пт, 1 янв., 05:30" {
		t.Errorf("When = %q", got)
	}
	if got := day(ts); got != "1 янв. 2027" {
		t.Errorf("day = %q", got)
	}
	if !sameDay(ts, ts.Add(10*time.Hour)) || sameDay(ts, ts.Add(20*time.Hour)) {
		t.Error("sameDay")
	}
}

func TestPlural(t *testing.T) {
	for n, want := range map[int]string{
		0: "выходов", 1: "выход", 2: "выхода", 4: "выхода", 5: "выходов", 11: "выходов", 12: "выходов",
		14: "выходов", 21: "выход", 22: "выхода", 101: "выход", 111: "выходов", 112: "выходов",
	} {
		if got := plural(n, "выход", "выхода", "выходов"); got != want {
			t.Errorf("plural(%d) = %q, want %q", n, got, want)
		}
	}
	if days(1) != "1 день" || days(3) != "3 дня" || days(7) != "7 дней" {
		t.Errorf("days: %s, %s, %s", days(1), days(3), days(7))
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
