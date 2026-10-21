// Package drafts keeps the outing a chat is putting together (or editing)
// step by step: the bot asks one thing at a time, in one message, and the
// answers — buttons or text — fill the draft until it is published.
package drafts

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/elvn5/sai/services/bot/internal/api"
)

// Step is what the draft asks next.
type Step string

const (
	StepTrail       Step = "trail"
	StepTitle       Step = "title"
	StepDate        Step = "date"
	StepTime        Step = "time"
	StepReturn      Step = "ret"
	StepMeeting     Step = "meet"
	StepTransport   Step = "transport"
	StepSeats       Step = "seats"
	StepCost        Step = "cost"
	StepLevel       Step = "level"
	StepDescription Step = "desc"
	StepContact     Step = "contact"
	// The whole outing to check before publishing.
	StepReview Step = "review"
	// Which field to change (from the review, or editing an outing).
	StepFields Step = "fields"
)

// Steps is the order a new outing is asked in.
var Steps = []Step{
	StepTrail, StepTitle, StepDate, StepTime, StepReturn, StepMeeting,
	StepTransport, StepSeats, StepCost, StepLevel, StepDescription, StepContact,
}

// Number is the step's place in Steps, from 1; 0 for the review.
func Number(s Step) int {
	for i, step := range Steps {
		if step == s {
			return i + 1
		}
	}
	return 0
}

// Draft is an outing being put together. Dates and times are Bishkek wall
// clock: Date "2026-10-24", Time "06:00", Returns "2026-10-24T19:00".
type Draft struct {
	// The outing being edited; "" for a new one.
	OutingID string
	// People already joined: seats can't go below.
	Taken int
	Step  Step
	// Set once every step was answered: a changed field returns to the
	// review instead of going on.
	Reviewing bool
	// The message showing the draft.
	MessageID int
	// The user's Telegram @username, offered as the contact.
	Username string

	TrailID, TrailName string
	Title              string
	Date, Time         string
	Returns            string
	MeetingPoint       string
	Transport          string
	Seats              int
	Cost               *int
	Level              string
	Description        string
	Contact            string

	// The calendar's month, "2026-10".
	Month string
	// The routes offered at the route step (buttons by index), and the
	// name searched for.
	Trails []api.TrailRef
	Search string
}

// ttl drops drafts a user walked away from.
const ttl = 2 * time.Hour

type entry struct {
	draft   Draft
	expires time.Time
}

// Store holds a draft per chat. It is safe for concurrent use and lives in
// memory, enough for a single bot instance.
type Store struct {
	mu     sync.Mutex
	drafts map[int64]entry
	// One answer at a time per chat: updates are handled in parallel, and
	// a quick second tap must see the first one's draft.
	chats map[int64]*sync.Mutex
	// Now is overridable in tests.
	Now func() time.Time
}

func NewStore() *Store {
	return &Store{drafts: map[int64]entry{}, chats: map[int64]*sync.Mutex{}, Now: time.Now}
}

// Lock holds the chat's draft until the returned unlock is called.
func (s *Store) Lock(chatID int64) (unlock func()) {
	s.mu.Lock()
	m, ok := s.chats[chatID]
	if !ok {
		m = &sync.Mutex{}
		s.chats[chatID] = m
	}
	s.mu.Unlock()
	m.Lock()
	return m.Unlock
}

// Get is the chat's draft, if it has one.
func (s *Store) Get(chatID int64) (Draft, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.drafts[chatID]
	if !ok || s.Now().After(e.expires) {
		delete(s.drafts, chatID)
		return Draft{}, false
	}
	return e.draft, true
}

// Put saves the chat's draft (and forgets abandoned ones).
func (s *Store) Put(chatID int64, d Draft) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.Now()
	for id, e := range s.drafts {
		if now.After(e.expires) {
			delete(s.drafts, id)
		}
	}
	s.drafts[chatID] = entry{draft: d, expires: now.Add(ttl)}
}

func (s *Store) Drop(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.drafts, chatID)
}

var bishkek = time.FixedZone("Asia/Bishkek", 6*60*60)

const (
	// How far ahead an outing can be planned (as the trails service).
	planAhead = 180 * 24 * time.Hour
	// Limits of the trails service.
	maxTitle, maxMeeting, maxDescription, maxContact = 160, 200, 2000, 120
	maxSeats, maxCost                                = 50, 1_000_000
)

// Levels and transports the trails service takes.
var (
	Levels     = []string{"beginner", "intermediate", "advanced"}
	Transports = []string{"car", "bus", "arranged"}
)

// FromOuting starts editing an outing: every field filled, the review
// first.
func FromOuting(o api.Outing, username string) Draft {
	starts := o.StartsAt.In(bishkek)
	d := Draft{
		OutingID: o.ID, Taken: o.Taken, Step: StepReview, Reviewing: true, Username: username,
		Title: o.Title, Date: starts.Format(time.DateOnly), Time: starts.Format("15:04"),
		MeetingPoint: o.MeetingPoint, Transport: o.Transport, Seats: o.Seats, Cost: o.CostPerPerson, Level: o.Level,
		Month: starts.Format("2006-01"),
	}
	if o.Trail != nil {
		d.TrailID, d.TrailName = o.Trail.ID, o.Trail.Name
	}
	if o.ReturnsAt != nil {
		d.Returns = o.ReturnsAt.In(bishkek).Format("2006-01-02T15:04")
	}
	if o.Description != nil {
		d.Description = *o.Description
	}
	if o.Contact != nil {
		d.Contact = *o.Contact
	}
	return d
}

// New starts a new outing: a group of three, by the organizer's car, for
// beginners — the site's defaults — with this month's calendar.
func New(username string, now time.Time) Draft {
	return Draft{
		Step: StepTrail, Username: username, Transport: "car", Seats: 3, Level: "beginner",
		Month: now.In(bishkek).Format("2006-01"),
	}
}

// Next moves on: to the following step, or back to the review once
// everything was answered.
func (d *Draft) Next() {
	if d.Reviewing {
		d.Step = StepReview
		return
	}
	n := Number(d.Step)
	if n == 0 || n >= len(Steps) {
		d.Step, d.Reviewing = StepReview, true
		return
	}
	d.Step = Steps[n]
}

// Back returns to the previous step (from a field being changed, to the
// list of fields).
func (d *Draft) Back() {
	if d.Reviewing {
		d.Step = StepFields
		return
	}
	if n := Number(d.Step); n > 1 {
		d.Step = Steps[n-2]
	}
}

// Go jumps to a step to change it.
func (d *Draft) Go(step Step) bool {
	if Number(step) == 0 {
		return false
	}
	d.Step = step
	if step == StepDate && d.Date != "" {
		d.Month = d.Date[:7]
	}
	return true
}

// Start is when the outing starts, if date and time are set.
func (d Draft) Start() (time.Time, bool) {
	t, err := time.ParseInLocation("2006-01-02 15:04", d.Date+" "+d.Time, bishkek)
	return t, err == nil && d.Date != "" && d.Time != ""
}

// Return is when the group is back, if set.
func (d Draft) Return() (time.Time, bool) {
	t, err := time.ParseInLocation("2006-01-02T15:04", d.Returns, bishkek)
	return t, err == nil
}

// Earliest and Latest are the days an outing can start on.
func Earliest(now time.Time) string { return now.In(bishkek).Format(time.DateOnly) }

func Latest(now time.Time) string { return now.Add(planAhead).In(bishkek).Format(time.DateOnly) }

// SetDate picks the day; a return on the old day moves along.
func (d *Draft) SetDate(date string, now time.Time) string {
	if _, err := time.Parse(time.DateOnly, date); err != nil {
		return "Не понял дату. Напишите, например, 24.10"
	}
	if date < Earliest(now) {
		return "Этот день уже прошёл"
	}
	if date > Latest(now) {
		return "Выход можно запланировать на полгода вперёд"
	}
	if d.Returns != "" && d.Date != "" && strings.HasPrefix(d.Returns, d.Date) {
		d.Returns = date + d.Returns[len(d.Date):]
	}
	d.Date, d.Month = date, date[:7]
	if start, ok := d.Start(); ok && !start.After(now) {
		// Today, but that hour is gone: ask the time again.
		d.Step = StepTime
		return ""
	}
	d.dropStaleReturn()
	d.Next()
	return ""
}

// SetTime picks the hour of the start ("06:00").
func (d *Draft) SetTime(clock string, now time.Time) string {
	if _, err := time.Parse("15:04", clock); err != nil {
		return "Не понял время. Напишите, например, 06:30"
	}
	old := d.Time
	d.Time = clock
	start, _ := d.Start()
	if !start.After(now) {
		d.Time = old
		return "Это время уже прошло — выберите другое"
	}
	d.dropStaleReturn()
	d.Next()
	return ""
}

// SetReturn sets when the group is back, "" for not saying.
func (d *Draft) SetReturn(returns string) string {
	if returns == "" {
		d.Returns = ""
		d.Next()
		return ""
	}
	t, err := time.ParseInLocation("2006-01-02T15:04", returns, bishkek)
	if err != nil {
		return "Не понял время. Напишите, например, 19:00 или 25.10 18:00"
	}
	if start, ok := d.Start(); ok && !t.After(start) {
		return "Возвращение должно быть позже сбора"
	}
	d.Returns = returns
	d.Next()
	return ""
}

// dropStaleReturn forgets a return that is no longer after the start.
func (d *Draft) dropStaleReturn() {
	start, ok1 := d.Start()
	back, ok2 := d.Return()
	if ok1 && ok2 && !back.After(start) {
		d.Returns = ""
	}
}

// SetSeats sets how many can join.
func (d *Draft) SetSeats(n int) string {
	switch {
	case n < 1 || n > maxSeats:
		return fmt.Sprintf("Можно от 1 до %d человек", maxSeats)
	case n < d.Taken:
		return fmt.Sprintf("Уже записались %d — мест не может быть меньше", d.Taken)
	}
	d.Seats = n
	d.Next()
	return ""
}

// Text applies a typed answer to the current step; the route step's text
// is a search, done by the caller.
func (d *Draft) Text(text string, now time.Time) string {
	text = strings.TrimSpace(text)
	switch d.Step {
	case StepTitle:
		return d.setText(&d.Title, text, maxTitle, "Название")
	case StepMeeting:
		return d.setText(&d.MeetingPoint, text, maxMeeting, "Место сбора")
	case StepDescription:
		return d.setText(&d.Description, text, maxDescription, "Описание")
	case StepContact:
		return d.setText(&d.Contact, text, maxContact, "Контакт")
	case StepDate:
		date, ok := ParseDate(text, now)
		if !ok {
			return "Не понял дату. Напишите, например, 24.10"
		}
		return d.SetDate(date, now)
	case StepTime:
		clock, ok := ParseClock(text)
		if !ok {
			return "Не понял время. Напишите, например, 06:30"
		}
		return d.SetTime(clock, now)
	case StepReturn:
		returns, ok := d.parseReturn(text, now)
		if !ok {
			return "Не понял время. Напишите, например, 19:00 или 25.10 18:00"
		}
		return d.SetReturn(returns)
	case StepSeats:
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(text, "чел")))
		if err != nil {
			return "Напишите число, например 6"
		}
		return d.SetSeats(n)
	case StepCost:
		n, ok := parseMoney(text)
		if !ok {
			return "Напишите сумму в сомах, например 500"
		}
		d.Cost = &n
		d.Next()
		return ""
	}
	return "Выберите вариант кнопкой выше"
}

func (d *Draft) setText(field *string, text string, limit int, name string) string {
	switch n := utf8.RuneCountInString(text); {
	case n == 0:
		return name + " не может быть пустым"
	case n > limit:
		return fmt.Sprintf("%s — не длиннее %d символов (сейчас %d)", name, limit, n)
	}
	*field = text
	d.Next()
	return ""
}

// ParseDate reads "24.10", "24.10.2026" or "24/10"; a day without a year
// is the next such day.
func ParseDate(text string, now time.Time) (string, bool) {
	text = strings.NewReplacer("/", ".", "-", ".", " ", "").Replace(strings.TrimSpace(text))
	parts := strings.Split(strings.TrimSuffix(text, "."), ".")
	if len(parts) < 2 || len(parts) > 3 {
		return "", false
	}
	day, err1 := strconv.Atoi(parts[0])
	month, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return "", false
	}
	today := now.In(bishkek)
	year := today.Year()
	if len(parts) == 3 {
		y, err := strconv.Atoi(parts[2])
		if err != nil {
			return "", false
		}
		if y < 100 {
			y += 2000
		}
		year = y
	}
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, bishkek)
	if t.Day() != day || int(t.Month()) != month {
		return "", false
	}
	if len(parts) == 2 && t.Format(time.DateOnly) < today.Format(time.DateOnly) {
		t = t.AddDate(1, 0, 0)
	}
	return t.Format(time.DateOnly), true
}

// ParseClock reads "6:30", "06.30", "0630" or "6".
func ParseClock(text string) (string, bool) {
	text = strings.NewReplacer(".", ":", "-", ":", " ", "", "ч", "").Replace(strings.TrimSpace(text))
	if !strings.Contains(text, ":") {
		switch len(text) {
		case 1, 2:
			text += ":00"
		case 3, 4:
			text = text[:len(text)-2] + ":" + text[len(text)-2:]
		}
	}
	h, m, ok := strings.Cut(text, ":")
	hour, err1 := strconv.Atoi(h)
	minute, err2 := strconv.Atoi(m)
	if !ok || err1 != nil || err2 != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return "", false
	}
	return fmt.Sprintf("%02d:%02d", hour, minute), true
}

// parseReturn reads "19:00" (the day of the start, or the next if that is
// before the start) or "25.10 18:00".
func (d Draft) parseReturn(text string, now time.Time) (string, bool) {
	fields := strings.Fields(text)
	switch len(fields) {
	case 1:
		clock, ok := ParseClock(fields[0])
		if !ok || d.Date == "" {
			return "", false
		}
		returns := d.Date + "T" + clock
		if start, ok := d.Start(); ok {
			if t, err := time.ParseInLocation("2006-01-02T15:04", returns, bishkek); err == nil && !t.After(start) {
				returns = t.AddDate(0, 0, 1).Format("2006-01-02T15:04")
			}
		}
		return returns, true
	case 2:
		date, ok1 := ParseDate(fields[0], now)
		clock, ok2 := ParseClock(fields[1])
		return date + "T" + clock, ok1 && ok2
	}
	return "", false
}

// parseMoney reads "500", "1 500 сом", "бесплатно".
func parseMoney(text string) (int, bool) {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "0" || strings.HasPrefix(text, "бесплат") {
		return 0, true
	}
	digits := strings.NewReplacer(" ", "", " ", "", "сом", "", "с", "", ".", "", ",", "").Replace(text)
	n, err := strconv.Atoi(digits)
	return n, err == nil && n >= 0 && n <= maxCost
}

// Missing is the first step without an answer, "" when the draft is
// complete.
func (d Draft) Missing() Step {
	switch {
	case d.Title == "":
		return StepTitle
	case d.Date == "":
		return StepDate
	case d.Time == "":
		return StepTime
	case d.MeetingPoint == "":
		return StepMeeting
	}
	return ""
}

// Input is the draft as the API takes it.
func (d Draft) Input() api.OutingInput {
	in := api.OutingInput{
		Title: d.Title, StartsAt: d.Date + "T" + d.Time + ":00+06:00", MeetingPoint: d.MeetingPoint,
		Transport: d.Transport, Seats: d.Seats, CostPerPerson: d.Cost, Level: d.Level,
	}
	if d.TrailID != "" {
		in.TrailID = &d.TrailID
	}
	if d.Returns != "" {
		r := d.Returns + ":00+06:00"
		in.ReturnsAt = &r
	}
	if d.Description != "" {
		in.Description = &d.Description
	}
	if d.Contact != "" {
		in.Contact = &d.Contact
	}
	return in
}

// Outing is the draft as an outing, for its preview.
func (d Draft) Outing() api.Outing {
	o := api.Outing{
		ID: d.OutingID, Title: d.Title, MeetingPoint: d.MeetingPoint, Transport: d.Transport,
		Seats: d.Seats, Taken: d.Taken, CostPerPerson: d.Cost, Level: d.Level, Status: "open", IsOrganizer: true,
	}
	if d.TrailID != "" {
		o.Trail = &api.TrailRef{ID: d.TrailID, Name: d.TrailName}
	}
	o.StartsAt, _ = d.Start()
	if t, ok := d.Return(); ok {
		o.ReturnsAt = &t
	}
	if d.Description != "" {
		o.Description = &d.Description
	}
	if d.Contact != "" {
		o.Contact = &d.Contact
	}
	return o
}
