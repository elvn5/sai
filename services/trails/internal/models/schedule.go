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

// Schedule sets when the outing's reminder and follow-up are due, as of
// now. An outing set up (or moved) inside the reminder window gets no
// reminder; only outings on a catalog route get a follow-up.
func (o *Outing) Schedule(now time.Time) {
	o.RemindAt, o.FollowUpAt = nil, nil
	if remind := o.StartsAt.Add(-ReminderLead); remind.After(now) {
		o.RemindAt = &remind
	}
	if o.Trail != nil {
		end := o.StartsAt.Add(AssumedLength)
		if o.ReturnsAt != nil {
			end = o.ReturnsAt.Time
		}
		followUp := end.Add(FollowUpDelay)
		o.FollowUpAt = &followUp
	}
}
