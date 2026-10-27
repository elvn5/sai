// Package models defines the trails service's entities and their JSON.
package models

import (
	"github.com/elvn5/sai/services/pkg/jsontime"
)

// Person is the public part of a profile (from the users service).
type Person struct {
	ID        string  `json:"id"`
	FirstName *string `json:"firstName"`
	Username  *string `json:"username"`
	AvatarURL *string `json:"avatarUrl"`
}

type Segment struct {
	From  string  `json:"from" binding:"min=1,max=100"`
	To    string  `json:"to" binding:"min=1,max=100"`
	Hours float64 `json:"hours" binding:"gt=0,lte=48"`
}

// TrackPoint is [latitude, longitude, elevation or null].
type TrackPoint [3]*float64

// TrackStats is computed from the track.
type TrackStats struct {
	DistanceKm    float64 `json:"distanceKm"`
	ElevationGain int     `json:"elevationGain"`
	MinElevation  *int    `json:"minElevation"`
	MaxElevation  *int    `json:"maxElevation"`
}

// ConditionSummary is the newest report, shown on route cards.
type ConditionSummary struct {
	ObservedOn string `json:"observedOn"`
	TrailState string `json:"trailState"`
	Traction   string `json:"traction"`
}

type TrailMarker struct {
	PointIndex int    `json:"pointIndex" binding:"gte=0,lte=4999"`
	Label      string `json:"label" binding:"required,max=160"`
}

type Trail struct {
	Markers         []TrailMarker     `json:"markers"`
	ChecklistURL    *string           `json:"checklistUrl"`
	ID              string            `json:"id"`
	Slug            string            `json:"slug"`
	Name            string            `json:"name"`
	Summary         *string           `json:"summary"`
	Description     string            `json:"description"`
	Area            string            `json:"area"`
	Kind            string            `json:"kind"`
	Grade           string            `json:"grade"`
	Seasons         []string          `json:"seasons"`
	Gear            []string          `json:"gear"`
	SummitElevation *int              `json:"summitElevation"`
	StartElevation  *int              `json:"startElevation"`
	ElevationGain   *int              `json:"elevationGain"`
	DistanceKm      *float64          `json:"distanceKm"`
	DurationHours   *float64          `json:"durationHours"`
	Days            int               `json:"days"`
	Segments        []Segment         `json:"segments"`
	GettingThere    *string           `json:"gettingThere"`
	StartLatitude   *float64          `json:"startLatitude"`
	StartLongitude  *float64          `json:"startLongitude"`
	Track           []TrackPoint      `json:"track,omitempty"`
	TrackStats      *TrackStats       `json:"trackStats"`
	GpxURL          *string           `json:"gpxUrl"`
	CoverImageURL   *string           `json:"coverImageUrl"`
	Photos          []string          `json:"photos"`
	IsPublished     bool              `json:"isPublished"`
	UserID          *string           `json:"userId"`
	LatestCondition *ConditionSummary `json:"latestCondition"`
	CreatedAt       jsontime.Time     `json:"createdAt"`
	UpdatedAt       jsontime.Time     `json:"updatedAt"`
}

// TrailRef is how reports point at their route.
type TrailRef struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
	Area string `json:"area"`
}

type ConditionReport struct {
	ID            string        `json:"id"`
	Trail         TrailRef      `json:"trail"`
	Author        Person        `json:"author"`
	ObservedOn    string        `json:"observedOn"`
	TrailState    string        `json:"trailState"`
	Traction      string        `json:"traction"`
	AvalancheRisk *string       `json:"avalancheRisk"`
	SnowLine      *int          `json:"snowLine"`
	Reached       *string       `json:"reached"`
	Text          *string       `json:"text"`
	Photos        []string      `json:"photos"`
	CreatedAt     jsontime.Time `json:"createdAt"`
	UserID        string        `json:"-"`
}
