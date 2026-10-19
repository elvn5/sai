// Package models defines the listings entities. Field order and JSON names
// match the NestJS (TypeORM) responses.
package models

import (
	"github.com/elvn5/sai/services/pkg/jsontime"
	"github.com/elvn5/sai/services/pkg/usersclient"
)

type Hotel struct {
	ID                 string        `json:"id"`
	Slug               string        `json:"slug"`
	Name               string        `json:"name"`
	Type               string        `json:"type"`
	Description        string        `json:"description"`
	Destination        string        `json:"destination"`
	Address            *string       `json:"address"`
	Latitude           *float64      `json:"latitude"`
	Longitude          *float64      `json:"longitude"`
	PriceFrom          *int          `json:"priceFrom"`
	Stars              *int          `json:"stars"`
	Amenities          []string      `json:"amenities"`
	CancellationPolicy string        `json:"cancellationPolicy"`
	CheckInTime        *string       `json:"checkInTime"`
	CheckOutTime       *string       `json:"checkOutTime"`
	CoverImageURL      *string       `json:"coverImageUrl"`
	Photos             []string      `json:"photos"`
	Phone              *string       `json:"phone"`
	Whatsapp           *string       `json:"whatsapp"`
	Telegram           *string       `json:"telegram"`
	Website            *string       `json:"website"`
	RatingAverage      float32       `json:"ratingAverage"`
	ReviewCount        int           `json:"reviewCount"`
	IsPublished        bool          `json:"isPublished"`
	UserID             *string       `json:"userId"`
	CreatedAt          jsontime.Time `json:"createdAt"`
	UpdatedAt          jsontime.Time `json:"updatedAt"`
}

type LocalGuide struct {
	ID              string        `json:"id"`
	Slug            string        `json:"slug"`
	Name            string        `json:"name"`
	Headline        *string       `json:"headline"`
	Bio             string        `json:"bio"`
	AvatarURL       *string       `json:"avatarUrl"`
	CoverImageURL   *string       `json:"coverImageUrl"`
	Photos          []string      `json:"photos"`
	Languages       []string      `json:"languages"`
	Specializations []string      `json:"specializations"`
	Destinations    []string      `json:"destinations"`
	PricePerDay     *int          `json:"pricePerDay"`
	ExperienceYears *int          `json:"experienceYears"`
	HasVehicle      bool          `json:"hasVehicle"`
	Phone           *string       `json:"phone"`
	Whatsapp        *string       `json:"whatsapp"`
	Telegram        *string       `json:"telegram"`
	RatingAverage   float32       `json:"ratingAverage"`
	ReviewCount     int           `json:"reviewCount"`
	IsPublished     bool          `json:"isPublished"`
	UserID          *string       `json:"userId"`
	CreatedAt       jsontime.Time `json:"createdAt"`
	UpdatedAt       jsontime.Time `json:"updatedAt"`
}

type Tour struct {
	ID                 string  `json:"id"`
	Locale             string  `json:"locale"`
	Slug               string  `json:"slug"`
	Title              string  `json:"title"`
	Excerpt            *string `json:"excerpt"`
	Content            string  `json:"content"`
	Price              *int    `json:"price"`
	DurationDays       *int    `json:"durationDays"`
	Location           *string `json:"location"`
	StartDate          string  `json:"startDate"` // YYYY-MM-DD
	EndDate            string  `json:"endDate"`
	Destination        *string `json:"destination"`
	CancellationPolicy *string `json:"cancellationPolicy"`
	CoverImageURL      *string `json:"coverImageUrl"`
	// The gallery under the cover.
	Photos []string `json:"photos"`
	// The trip day by day.
	Program []ProgramDay `json:"program"`
	// Places per departure; nil: not limited.
	Capacity      *int           `json:"capacity"`
	Phone         *string        `json:"phone"`
	Whatsapp      *string        `json:"whatsapp"`
	Telegram      *string        `json:"telegram"`
	RatingAverage float32        `json:"ratingAverage"`
	ReviewCount   int            `json:"reviewCount"`
	IsPublished   bool           `json:"isPublished"`
	PublishedAt   *jsontime.Time `json:"publishedAt"`
	UserID        *string        `json:"userId"`
	// The partner who runs it, on public reads (nil on the owner's lists).
	Organizer *usersclient.Author `json:"organizer,omitempty"`
	CreatedAt jsontime.Time       `json:"createdAt"`
	UpdatedAt jsontime.Time       `json:"updatedAt"`
}

// ProgramDay is one day of a tour's program.
type ProgramDay struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// Review is the public shape: the author's profile, never their contacts.
type Review struct {
	ID        string             `json:"id"`
	Rating    int                `json:"rating"`
	Text      string             `json:"text"`
	Photos    []string           `json:"photos"`
	CreatedAt jsontime.Time      `json:"createdAt"`
	Author    usersclient.Author `json:"author"`
	// The listing owner's answer, if any.
	Reply  *ReviewReply `json:"reply"`
	UserID string       `json:"-"`
}

type ReviewReply struct {
	Text      string        `json:"text"`
	RepliedAt jsontime.Time `json:"repliedAt"`
}

// PartnerReview is a review of one of the signed-in partner's listings.
type PartnerReview struct {
	Review
	Listing ListingRef `json:"listing"`
}

type ReviewSummary struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
	// Reviews per star rating, keyed "1".."5".
	Distribution map[string]int `json:"distribution"`
}

type ReviewsPage struct {
	Items   []Review      `json:"items"`
	Total   int           `json:"total"`
	Summary ReviewSummary `json:"summary"`
}

type ListingPost struct {
	ID        string             `json:"id"`
	Text      string             `json:"text"`
	Photos    []string           `json:"photos"`
	CreatedAt jsontime.Time      `json:"createdAt"`
	Author    usersclient.Author `json:"author"`
	UserID    string             `json:"-"`
}

type Page[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

// ListingRef names one of a partner's listings.
type ListingRef struct {
	TargetType  string  `json:"targetType"`
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Slug        string  `json:"slug"`
	Locale      *string `json:"locale"` // tours only
	IsPublished bool    `json:"isPublished"`
}

// Listing stats events: a page view or a click on a contact button.
const (
	EventView     = "view"
	EventPhone    = "phone"
	EventWhatsapp = "whatsapp"
	EventTelegram = "telegram"
	EventWebsite  = "website"
)

var StatEvents = []string{EventView, EventPhone, EventWhatsapp, EventTelegram, EventWebsite}

type StatCounts struct {
	Views    int `json:"views"`
	Phone    int `json:"phone"`
	Whatsapp int `json:"whatsapp"`
	Telegram int `json:"telegram"`
	Website  int `json:"website"`
	// All contact clicks together.
	Contacts int `json:"contacts"`
}

type ListingStats struct {
	Listing ListingRef `json:"listing"`
	StatCounts
}

type DailyStats struct {
	Day      string `json:"day"` // YYYY-MM-DD
	Views    int    `json:"views"`
	Contacts int    `json:"contacts"`
}

// StatsReport is a partner's listings stats over [From, To], with the
// same-length period before it for comparison.
type StatsReport struct {
	From     string         `json:"from"`
	To       string         `json:"to"`
	Totals   StatCounts     `json:"totals"`
	Previous StatCounts     `json:"previous"`
	Daily    []DailyStats   `json:"daily"`
	Listings []ListingStats `json:"listings"`
}
