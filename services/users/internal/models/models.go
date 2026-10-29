// Package models defines the users service's entities and the JSON shapes
// the NestJS API returned for them.
package models

import (
	"time"

	"github.com/elvn5/sai/services/pkg/jsontime"
)

// User is an account row. Secrets (password and verification hashes) are
// loaded only by the queries that need them.
type User struct {
	ID                        string
	TelegramID                *string // bigint, as text — TypeORM returned it as a string
	Email                     *string
	PasswordHash              *string
	Username                  *string
	FirstName                 *string
	LastName                  *string
	Phone                     *string
	AvatarURL                 *string
	IsVerified                bool
	VerificationCodeHash      *string
	VerificationCodeExpiresAt *time.Time
	VerificationCodeSentAt    *time.Time
	VerificationAttempts      int
	PartnerStatus             string
	PartnerRequestedAt        *time.Time
	PartnerKind               string // "organizer" or "driver"
	// Sessions issued before this are refused (the password was reset).
	SessionsValidAfter *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// PublicUser is what the site sees about the signed-in user.
type PublicUser struct {
	ID            string  `json:"id"`
	TelegramID    *string `json:"telegramId"`
	Email         *string `json:"email"`
	Username      *string `json:"username"`
	FirstName     *string `json:"firstName"`
	LastName      *string `json:"lastName"`
	Phone         *string `json:"phone"`
	AvatarURL     *string `json:"avatarUrl"`
	PartnerStatus string  `json:"partnerStatus"`
	PartnerKind   string  `json:"partnerKind"`
}

func (u User) Public() PublicUser {
	return PublicUser{
		ID: u.ID, TelegramID: u.TelegramID, Email: u.Email, Username: u.Username, FirstName: u.FirstName,
		LastName: u.LastName, Phone: u.Phone, AvatarURL: u.AvatarURL, PartnerStatus: u.PartnerStatus,
		PartnerKind: u.PartnerKind,
	}
}

// AdminUser is the full row as the admin panel received it from TypeORM:
// every column except the password and verification-code hashes.
type AdminUser struct {
	ID                        string         `json:"id"`
	TelegramID                *string        `json:"telegramId"`
	Email                     *string        `json:"email"`
	Username                  *string        `json:"username"`
	FirstName                 *string        `json:"firstName"`
	LastName                  *string        `json:"lastName"`
	Phone                     *string        `json:"phone"`
	AvatarURL                 *string        `json:"avatarUrl"`
	IsVerified                bool           `json:"isVerified"`
	VerificationCodeExpiresAt *jsontime.Time `json:"verificationCodeExpiresAt"`
	VerificationCodeSentAt    *jsontime.Time `json:"verificationCodeSentAt"`
	VerificationAttempts      int            `json:"verificationAttempts"`
	PartnerStatus             string         `json:"partnerStatus"`
	PartnerRequestedAt        *jsontime.Time `json:"partnerRequestedAt"`
	PartnerKind               string         `json:"partnerKind"`
	CreatedAt                 jsontime.Time  `json:"createdAt"`
	UpdatedAt                 jsontime.Time  `json:"updatedAt"`
}

func (u User) Admin() AdminUser {
	return AdminUser{
		ID: u.ID, TelegramID: u.TelegramID, Email: u.Email, Username: u.Username, FirstName: u.FirstName,
		LastName: u.LastName, Phone: u.Phone, AvatarURL: u.AvatarURL, IsVerified: u.IsVerified,
		VerificationCodeExpiresAt: jsontime.Ptr(u.VerificationCodeExpiresAt),
		VerificationCodeSentAt:    jsontime.Ptr(u.VerificationCodeSentAt),
		VerificationAttempts:      u.VerificationAttempts, PartnerStatus: u.PartnerStatus,
		PartnerRequestedAt: jsontime.Ptr(u.PartnerRequestedAt), PartnerKind: u.PartnerKind,
		CreatedAt: jsontime.New(u.CreatedAt), UpdatedAt: jsontime.New(u.UpdatedAt),
	}
}

func AdminUsers(users []User) []AdminUser {
	out := make([]AdminUser, len(users))
	for i, u := range users {
		out[i] = u.Admin()
	}
	return out
}

type SupportInfo struct {
	ID              string        `json:"id"`
	Message         string        `json:"message"`
	Phone           *string       `json:"phone"`
	TelegramContact *string       `json:"telegramContact"`
	Email           *string       `json:"email"`
	WorkingHours    *string       `json:"workingHours"`
	UpdatedAt       jsontime.Time `json:"updatedAt"`
}

// SavedAdminUser is AdminUser as TypeORM returned it from save() (the
// partner approve/reject endpoints): the hidden secret columns appear as
// null. They are always null here — the real hashes are never sent.
type SavedAdminUser struct {
	AdminUser
	PasswordHash         *string `json:"passwordHash"`
	VerificationCodeHash *string `json:"verificationCodeHash"`
}

func (u User) SavedAdmin() SavedAdminUser {
	return SavedAdminUser{AdminUser: u.Admin()}
}

// Achievement rarities, lowest first, with the points each is worth in the
// hall of fame.
var AchievementRarities = map[string]int{"common": 10, "rare": 25, "epic": 50, "legendary": 100}

// Achievement is a catalog entry. Title and Description map a locale to
// its text.
type Achievement struct {
	ID          string            `json:"id"`
	Title       map[string]string `json:"title"`
	Description map[string]string `json:"description"`
	Icon        string            `json:"icon"`
	Rarity      string            `json:"rarity"`
	Points      int               `json:"points"`
	IsActive    bool              `json:"isActive"`
	SortOrder   int               `json:"sortOrder"`
	CreatedAt   jsontime.Time     `json:"createdAt"`
	UpdatedAt   jsontime.Time     `json:"updatedAt"`
}

// Person is the public part of a profile (never email or phone).
type Person struct {
	ID        string  `json:"id"`
	FirstName *string `json:"firstName"`
	Username  *string `json:"username"`
	AvatarURL *string `json:"avatarUrl"`
}

// Award is an achievement a partner handed to a user.
type Award struct {
	ID          string        `json:"id"`
	Achievement Achievement   `json:"achievement"`
	User        Person        `json:"user"`
	AwardedBy   *Person       `json:"awardedBy"`
	Note        *string       `json:"note"`
	AwardedAt   jsontime.Time `json:"awardedAt"`
	IsNew       bool          `json:"isNew"`
}

// HallEntry is one traveler on the hall-of-fame leaderboard.
type HallEntry struct {
	Rank         int           `json:"rank"`
	User         Person        `json:"user"`
	Points       int           `json:"points"`
	Count        int           `json:"count"`
	Achievements []Achievement `json:"achievements"`
}
