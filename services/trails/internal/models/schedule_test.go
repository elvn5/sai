package models_test

import (
	"testing"
	"time"

	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/trails/internal/models"
)

func TestSchedule(t *testing.T) {
	now := time.Date(2026, 10, 12, 6, 0, 0, 0, time.UTC)
	start := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
	trail := &models.TrailRef{ID: "t"}

	o := models.Outing{StartsAt: jsontime.New(start), Trail: trail}
	o.Schedule(now)
	if !o.RemindAt.Equal(start.Add(-12*time.Hour)) || !o.FollowUpAt.Equal(start.Add(12*time.Hour)) {
		t.Errorf("default: remind %v, follow-up %v", o.RemindAt, o.FollowUpAt)
	}

	back := jsontime.New(start.Add(36 * time.Hour))
	o.ReturnsAt = &back
	o.Schedule(now)
	if !o.FollowUpAt.Equal(back.Add(2 * time.Hour)) {
		t.Errorf("with return: %v", o.FollowUpAt)
	}

	// Set up 5 hours before the start: no reminder.
	o.Schedule(start.Add(-5 * time.Hour))
	if o.RemindAt != nil {
		t.Errorf("last-minute outing reminded at %v", o.RemindAt)
	}

	// No catalog route: no follow-up.
	o.Trail = nil
	o.Schedule(now)
	if o.FollowUpAt != nil {
		t.Errorf("follow-up without a route: %v", o.FollowUpAt)
	}
}
