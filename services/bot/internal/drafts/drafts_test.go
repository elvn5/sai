package drafts_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/elvn5/sai/services/bot/internal/api"
	"github.com/elvn5/sai/services/bot/internal/drafts"
)

// 10:00 in Bishkek, Tuesday 20 October 2026.
var now = time.Date(2026, 10, 20, 4, 0, 0, 0, time.UTC)

func TestNewOutingStepByStep(t *testing.T) {
	d := drafts.New("aibek", now)
	if d.Step != drafts.StepTrail || d.Seats != 3 || d.Transport != "car" || d.Level != "beginner" || d.Month != "2026-10" {
		t.Fatalf("new: %+v", d)
	}
	d.TrailID, d.TrailName = "t1", "Пик Учитель"
	d.Next()
	for _, answer := range []struct {
		step drafts.Step
		text string
	}{
		{drafts.StepTitle, "  Пик Учитель за день "},
		{drafts.StepDate, "24.10"},
		{drafts.StepTime, "6.30"},
		{drafts.StepReturn, "19:00"},
		{drafts.StepMeeting, "ТЦ «Вефа»"},
	} {
		if d.Step != answer.step {
			t.Fatalf("at %s, want %s", d.Step, answer.step)
		}
		if problem := d.Text(answer.text, now); problem != "" {
			t.Fatalf("%s %q: %s", answer.step, answer.text, problem)
		}
	}
	if d.Step != drafts.StepTransport {
		t.Fatalf("after the meeting point: %s", d.Step)
	}
	d.Next() // transport kept
	if d.SetSeats(6) != "" || d.Step != drafts.StepCost {
		t.Fatalf("seats: %+v", d)
	}
	if d.Text("1 500 сом", now) != "" || *d.Cost != 1500 {
		t.Fatalf("cost: %v", d.Cost)
	}
	d.Next() // level
	if d.Text("Темп спокойный", now) != "" || d.Step != drafts.StepContact {
		t.Fatalf("description: %+v", d)
	}
	if d.Text("@aibek", now) != "" || d.Step != drafts.StepReview || !d.Reviewing {
		t.Fatalf("contact: %+v", d)
	}
	if d.Missing() != "" {
		t.Fatalf("missing %s", d.Missing())
	}

	raw, _ := json.Marshal(d.Input())
	want := `{"trailId":"t1","title":"Пик Учитель за день","startsAt":"2026-10-24T06:30:00+06:00","returnsAt":"2026-10-24T19:00:00+06:00",` +
		`"meetingPoint":"ТЦ «Вефа»","transport":"car","seats":6,"costPerPerson":1500,"level":"beginner",` +
		`"description":"Темп спокойный","contact":"@aibek"}`
	if string(raw) != want {
		t.Errorf("input:\n%s\nwant:\n%s", raw, want)
	}

	// Changing a field from the review comes back to it.
	d.Go(drafts.StepSeats)
	if d.SetSeats(8) != "" || d.Step != drafts.StepReview {
		t.Errorf("change from the review: %s", d.Step)
	}
	d.Go(drafts.StepTime)
	d.Back()
	if d.Step != drafts.StepFields {
		t.Errorf("back from a field: %s", d.Step)
	}
}

func TestDatesAndTimes(t *testing.T) {
	d := drafts.New("", now)
	d.Step = drafts.StepDate
	for text, problem := range map[string]string{
		"31.02":      "Не понял дату. Напишите, например, 24.10",
		"завтра":     "Не понял дату. Напишите, например, 24.10",
		"19.10.2026": "Этот день уже прошёл",
		"01.06.2027": "Выход можно запланировать на полгода вперёд",
	} {
		if got := d.Text(text, now); got != problem {
			t.Errorf("date %q: %q", text, got)
		}
	}
	// A day without a year that has passed is next year's.
	if date, ok := drafts.ParseDate("5/1", now); !ok || date != "2027-01-05" {
		t.Errorf("5/1 = %s", date)
	}

	// Today, an hour already gone: the time is asked again.
	if d.SetDate("2026-10-20", now) != "" || d.Step != drafts.StepTime {
		t.Fatalf("today: %s", d.Step)
	}
	if got := d.SetTime("09:00", now); got != "Это время уже прошло — выберите другое" || d.Time != "" {
		t.Errorf("past hour: %q %q", got, d.Time)
	}
	if d.SetTime("18:00", now) != "" || d.Step != drafts.StepReturn {
		t.Fatalf("evening: %s", d.Step)
	}
	// Back before the start: the next day.
	if d.Text("02:00", now) != "" || d.Returns != "2026-10-21T02:00" {
		t.Errorf("overnight: %s", d.Returns)
	}
	if got := d.SetReturn("2026-10-20T17:00"); got != "Возвращение должно быть позже сбора" {
		t.Errorf("return before start: %q", got)
	}

	// Moving the day moves a same-day return along.
	d.Returns = "2026-10-20T22:00"
	d.Reviewing = true
	if d.SetDate("2026-10-25", now) != "" || d.Returns != "2026-10-25T22:00" || d.Step != drafts.StepReview {
		t.Errorf("moved: %+v", d)
	}
	// A start after the return drops the return.
	d.Returns = "2026-10-25T19:00"
	if d.SetTime("20:00", now) != "" || d.Returns != "" {
		t.Errorf("stale return kept: %s", d.Returns)
	}

	for text, want := range map[string]string{"6": "06:00", "0630": "06:30", "7-15": "07:15", "23:59": "23:59"} {
		if got, ok := drafts.ParseClock(text); !ok || got != want {
			t.Errorf("clock %q = %q", text, got)
		}
	}
	for _, text := range []string{"24:00", "6:60", "утром", ""} {
		if _, ok := drafts.ParseClock(text); ok {
			t.Errorf("clock %q accepted", text)
		}
	}
}

func TestLimits(t *testing.T) {
	d := drafts.New("", now)
	d.Step = drafts.StepTitle
	if got := d.Text("   ", now); got != "Название не может быть пустым" {
		t.Errorf("empty title: %q", got)
	}
	long := make([]rune, 161)
	for i := range long {
		long[i] = 'а'
	}
	if got := d.Text(string(long), now); got != "Название — не длиннее 160 символов (сейчас 161)" {
		t.Errorf("long title: %q", got)
	}
	d.Step = drafts.StepSeats
	for text, want := range map[string]string{"0": "Можно от 1 до 50 человек", "51": "Можно от 1 до 50 человек", "много": "Напишите число, например 6"} {
		if got := d.Text(text, now); got != want {
			t.Errorf("seats %q: %q", text, got)
		}
	}
	d.Taken = 4
	if got := d.SetSeats(3); got != "Уже записались 4 — мест не может быть меньше" {
		t.Errorf("below taken: %q", got)
	}
	d.Step = drafts.StepCost
	if d.Text("Бесплатно", now) != "" || d.Cost == nil || *d.Cost != 0 {
		t.Errorf("free: %v", d.Cost)
	}
	d.Step = drafts.StepCost
	if got := d.Text("дорого", now); got != "Напишите сумму в сомах, например 500" {
		t.Errorf("cost: %q", got)
	}
	d.Step = drafts.StepTransport
	if got := d.Text("машина", now); got != "Выберите вариант кнопкой выше" {
		t.Errorf("buttons-only step: %q", got)
	}
}

func TestEditOuting(t *testing.T) {
	starts := time.Date(2026, 11, 1, 1, 30, 0, 0, time.UTC) // 07:30 in Bishkek
	returns := starts.Add(12 * time.Hour)
	o := api.Outing{ID: "o2", Title: "Барскоон", StartsAt: starts, ReturnsAt: &returns, MeetingPoint: "Вефа", Transport: "bus",
		Seats: 12, Taken: 4, Level: "intermediate", Trail: &api.TrailRef{ID: "t9", Name: "Барскоон"}, Contact: ptr("@me")}
	d := drafts.FromOuting(o, "aibek")
	if d.OutingID != "o2" || d.Step != drafts.StepReview || !d.Reviewing || d.Date != "2026-11-01" || d.Time != "07:30" ||
		d.Returns != "2026-11-01T19:30" || d.TrailID != "t9" || d.Contact != "@me" || d.Taken != 4 {
		t.Fatalf("from outing: %+v", d)
	}
	in := d.Input()
	if in.StartsAt != "2026-11-01T07:30:00+06:00" || in.Description != nil || *in.TrailID != "t9" {
		t.Errorf("input: %+v", in)
	}
	preview := d.Outing()
	if !preview.StartsAt.Equal(starts) || !preview.IsOrganizer || preview.Taken != 4 {
		t.Errorf("preview: %+v", preview)
	}
}

func TestStore(t *testing.T) {
	s := drafts.NewStore()
	clock := now
	s.Now = func() time.Time { return clock }
	s.Put(1, drafts.Draft{Title: "A"})
	if d, ok := s.Get(1); !ok || d.Title != "A" {
		t.Fatalf("get: %+v %v", d, ok)
	}
	clock = clock.Add(3 * time.Hour)
	if _, ok := s.Get(1); ok {
		t.Error("abandoned draft kept")
	}
	s.Put(2, drafts.Draft{})
	s.Drop(2)
	if _, ok := s.Get(2); ok {
		t.Error("dropped draft kept")
	}
}

func ptr(s string) *string { return &s }
