// Package models defines the trails service's entities and their JSON.
package models

import (
	"time"

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

type Trail struct {
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
	UpcomingOutings int               `json:"upcomingOutings"`
	CreatedAt       jsontime.Time     `json:"createdAt"`
	UpdatedAt       jsontime.Time     `json:"updatedAt"`
}

// TrailRef is how reports and outings point at their route.
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

// OutingsSummary is a person's record: outings been on, and the distance
// and climb of their routes.
type OutingsSummary struct {
	Outings       int     `json:"outings"`
	DistanceKm    float64 `json:"distanceKm"`
	ElevationGain int     `json:"elevationGain"`
}

// RatingMine is the visitor's own rating of an outing.
type RatingMine struct {
	Rating int     `json:"rating"`
	Text   *string `json:"text"`
}

// OutingReview is a participant's rating of an outing, as the organizer's
// public page shows it.
type OutingReview struct {
	Rating    int           `json:"rating"`
	Text      *string       `json:"text"`
	CreatedAt jsontime.Time `json:"createdAt"`
	Author    Person        `json:"author"`
	Outing    struct {
		ID       string        `json:"id"`
		Title    string        `json:"title"`
		StartsAt jsontime.Time `json:"startsAt"`
	} `json:"outing"`
	UserID string `json:"-"`
}

// RatingSummary is an average of 1–5 ratings and how many there are.
type RatingSummary struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}

// OrganizerOutings is what an organizer's public page shows about their
// outings.
type OrganizerOutings struct {
	// Outings held (open, started) and the people who joined them.
	Hosted       int            `json:"hosted"`
	Participants int            `json:"participants"`
	Rating       RatingSummary  `json:"rating"`
	Upcoming     []Outing       `json:"upcoming"`
	Reviews      []OutingReview `json:"reviews"`
}

// RosterEntry is a participant as the organizer sees them.
type RosterEntry struct {
	Person
	JoinedAt  jsontime.Time  `json:"joinedAt"`
	ArrivedAt *jsontime.Time `json:"arrivedAt"`
}

type Outing struct {
	ID            string         `json:"id"`
	Trail         *TrailRef      `json:"trail"`
	Title         string         `json:"title"`
	StartsAt      jsontime.Time  `json:"startsAt"`
	ReturnsAt     *jsontime.Time `json:"returnsAt"`
	MeetingPoint  string         `json:"meetingPoint"`
	Transport     string         `json:"transport"`
	Seats         int            `json:"seats"`
	Taken         int            `json:"taken"`
	CostPerPerson *int           `json:"costPerPerson"`
	Level         string         `json:"level"`
	Description   *string        `json:"description"`
	// Contact is shown to the organizer and participants only.
	Contact      *string  `json:"contact"`
	Status       string   `json:"status"`
	Organizer    Person   `json:"organizer"`
	Participants []Person `json:"participants,omitempty"`
	// Who joined and who turned up: for the organizer, on one outing's
	// details only.
	Roster      []RosterEntry `json:"roster,omitempty"`
	IsJoined    bool          `json:"isJoined"`
	IsOrganizer bool          `json:"isOrganizer"`
	// For a participant, on one outing's details: whether they can rate it
	// now, and their rating if given.
	CanRate   bool          `json:"canRate,omitempty"`
	MyRating  *RatingMine   `json:"myRating,omitempty"`
	CreatedAt jsontime.Time `json:"createdAt"`
	UserID    string        `json:"-"`
	// When the reminder and follow-up notifications are due (see Schedule).
	RemindAt   *time.Time `json:"-"`
	FollowUpAt *time.Time `json:"-"`
}
