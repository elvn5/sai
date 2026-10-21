package models

import "time"

const (
	// ReminderLead: the reminder goes out this long before the start (an
	// early-morning start gets it the evening before).
	ReminderLead = 12 * time.Hour
	// AssumedLength stands in for a missing return time.
	AssumedLength = 10 * time.Hour
	// FollowUpDelay: the "how did it go?" message, after the return.
	FollowUpDelay = 2 * time.Hour
)

// Schedule sets when the outing's reminder and follow-up (a request to
// rate it, and to report on its route) are due, as of now. An outing set
// up (or moved) inside the reminder window gets no reminder.
func (o *Outing) Schedule(now time.Time) {
	o.RemindAt, o.FollowUpAt = nil, nil
	if remind := o.StartsAt.Add(-ReminderLead); remind.After(now) {
		o.RemindAt = &remind
	}
	followUp := o.EndsAt().Add(FollowUpDelay)
	o.FollowUpAt = &followUp
}

// EndsAt is the return, or the start plus a usual day out.
func (o Outing) EndsAt() time.Time {
	if o.ReturnsAt != nil {
		return o.ReturnsAt.Time
	}
	return o.StartsAt.Add(AssumedLength)
}
