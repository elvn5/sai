// Package jsontime serializes timestamps exactly like JavaScript's
// Date.prototype.toJSON — UTC with millisecond precision, e.g.
// "2026-10-08T08:15:26.064Z" — the format the NestJS API returned.
package jsontime

import (
	"encoding/json"
	"time"
)

const layout = "2006-01-02T15:04:05.000Z"

type Time struct{ time.Time }

func New(t time.Time) Time { return Time{t} }

func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.UTC().Format(layout))
}

// Ptr converts a nullable timestamp.
func Ptr(t *time.Time) *Time {
	if t == nil {
		return nil
	}
	return &Time{*t}
}
