package models

import "time"

const (
	// ReminderLead: the reminder goes out this long before the start (an
	// early-morning start gets it the evening before).
	ReminderLead = 12 * time.Hour
	// DefaultMeetingTime stands in for a tour without a meeting time.
	DefaultMeetingTime = "09:00"
	// The last day of a tour is taken to end at this hour.
	endHour = 18
	// FollowUpDelay: the "how did it go?" message, after the end.
	FollowUpDelay = 2 * time.Hour
)

// Bishkek is the time zone tours' dates and meeting times are in.
var Bishkek = time.FixedZone("Asia/Bishkek", 6*60*60)

// StartsAt is when the group meets: startDate at the meeting time in
// Bishkek (zero for a malformed date).
func (t Tour) StartsAt() time.Time {
	day, err := time.ParseInLocation(time.DateOnly, t.StartDate, Bishkek)
	if err != nil {
		return time.Time{}
	}
	at := DefaultMeetingTime
	if t.MeetingTime != nil && *t.MeetingTime != "" {
		at = *t.MeetingTime
	}
	clock, err := time.Parse("15:04", at)
	if err != nil {
		return day
	}
	return day.Add(time.Duration(clock.Hour())*time.Hour + time.Duration(clock.Minute())*time.Minute)
}

// EndsAt is the evening of the last day.
func (t Tour) EndsAt() time.Time {
	day, err := time.ParseInLocation(time.DateOnly, t.EndDate, Bishkek)
	if err != nil || day.Before(t.StartsAt()) {
		return t.StartsAt().Add(10 * time.Hour)
	}
	return day.Add(endHour * time.Hour)
}

// Schedule sets when the tour's reminder and follow-up (a request to rate
// it) are due, as of now. A tour set up (or moved) inside the reminder
// window gets no reminder.
func (t *Tour) Schedule(now time.Time) {
	t.RemindAt, t.FollowUpAt = nil, nil
	if remind := t.StartsAt().Add(-ReminderLead); remind.After(now) {
		t.RemindAt = &remind
	}
	followUp := t.EndsAt().Add(FollowUpDelay)
	t.FollowUpAt = &followUp
}
