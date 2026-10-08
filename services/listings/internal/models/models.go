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
	ID                 string         `json:"id"`
	Locale             string         `json:"locale"`
	Slug               string         `json:"slug"`
	Title              string         `json:"title"`
	Excerpt            *string        `json:"excerpt"`
	Content            string         `json:"content"`
	Price              *int           `json:"price"`
	DurationDays       *int           `json:"durationDays"`
	Location           *string        `json:"location"`
	StartDate          string         `json:"startDate"` // YYYY-MM-DD
	EndDate            string         `json:"endDate"`
	Destination        *string        `json:"destination"`
	CancellationPolicy *string        `json:"cancellationPolicy"`
	CoverImageURL      *string        `json:"coverImageUrl"`
	Phone              *string        `json:"phone"`
	Whatsapp           *string        `json:"whatsapp"`
	Telegram           *string        `json:"telegram"`
	RatingAverage      float32        `json:"ratingAverage"`
	ReviewCount        int            `json:"reviewCount"`
	IsPublished        bool           `json:"isPublished"`
	PublishedAt        *jsontime.Time `json:"publishedAt"`
	UserID             *string        `json:"userId"`
	CreatedAt          jsontime.Time  `json:"createdAt"`
	UpdatedAt          jsontime.Time  `json:"updatedAt"`
}

// Review is the public shape: the author's profile, never their contacts.
type Review struct {
	ID        string             `json:"id"`
	Rating    int                `json:"rating"`
	Text      string             `json:"text"`
	Photos    []string           `json:"photos"`
	CreatedAt jsontime.Time      `json:"createdAt"`
	Author    usersclient.Author `json:"author"`
	UserID    string             `json:"-"`
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
