// Package models defines the content entities. Field order and JSON names
// match the NestJS (TypeORM) responses.
package models

import "github.com/elvn5/sai/services/pkg/jsontime"

var (
	Locales      = []string{"ru", "ky", "en", "de"}
	Difficulties = []string{"easy", "moderate", "hard"}
	Seasons      = []string{"summer", "winter", "all_year"}
	SeoPages     = []string{"home", "blog", "gallery", "guides", "tours"}
)

type BlogPost struct {
	ID             string         `json:"id"`
	Locale         string         `json:"locale"`
	Slug           string         `json:"slug"`
	Title          string         `json:"title"`
	Excerpt        *string        `json:"excerpt"`
	Content        string         `json:"content"`
	CoverImageURL  *string        `json:"coverImageUrl"`
	SeoTitle       *string        `json:"seoTitle"`
	SeoDescription *string        `json:"seoDescription"`
	IsPublished    bool           `json:"isPublished"`
	PublishedAt    *jsontime.Time `json:"publishedAt"`
	// The partner who wrote it; nil for posts written in the admin panel.
	UserID    *string       `json:"userId"`
	CreatedAt jsontime.Time `json:"createdAt"`
	UpdatedAt jsontime.Time `json:"updatedAt"`
}

type Attachment struct {
	URL  string `json:"url" binding:"min=1,max=2000"`
	Name string `json:"name" binding:"min=1,max=255"`
	Size int64  `json:"size" binding:"gte=0"`
}

type Guide struct {
	ID                 string         `json:"id"`
	Locale             string         `json:"locale"`
	Slug               string         `json:"slug"`
	Title              string         `json:"title"`
	Excerpt            *string        `json:"excerpt"`
	Content            string         `json:"content"`
	Difficulty         string         `json:"difficulty"`
	Season             string         `json:"season"`
	EquipmentChecklist []string       `json:"equipmentChecklist"`
	CoverImageURL      *string        `json:"coverImageUrl"`
	Attachments        []Attachment   `json:"attachments"`
	SeoTitle           *string        `json:"seoTitle"`
	SeoDescription     *string        `json:"seoDescription"`
	IsPublished        bool           `json:"isPublished"`
	PublishedAt        *jsontime.Time `json:"publishedAt"`
	UserID             *string        `json:"userId"`
	CreatedAt          jsontime.Time  `json:"createdAt"`
	UpdatedAt          jsontime.Time  `json:"updatedAt"`
}

type GalleryItem struct {
	ID          string        `json:"id"`
	ImageURL    string        `json:"imageUrl"`
	Location    *string       `json:"location"`
	Caption     *string       `json:"caption"`
	AltText     *string       `json:"altText"`
	SortOrder   int           `json:"sortOrder"`
	IsPublished bool          `json:"isPublished"`
	UserID      *string       `json:"userId"`
	CreatedAt   jsontime.Time `json:"createdAt"`
	UpdatedAt   jsontime.Time `json:"updatedAt"`
}

type PageSeo struct {
	ID          string        `json:"id"`
	Page        string        `json:"page"`
	Locale      string        `json:"locale"`
	Title       *string       `json:"title"`
	Description *string       `json:"description"`
	OgImageURL  *string       `json:"ogImageUrl"`
	UpdatedAt   jsontime.Time `json:"updatedAt"`
}
