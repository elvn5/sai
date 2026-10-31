package api

import (
	"context"
	"net/http"
)

// Me is the user's account as the site's profile shows it.
type Me struct {
	ID            string  `json:"id"`
	Email         *string `json:"email"`
	Username      *string `json:"username"`
	FirstName     *string `json:"firstName"`
	Phone         *string `json:"phone"`
	PartnerStatus string  `json:"partnerStatus"`
	PartnerKind   string  `json:"partnerKind"`
	// Changing the password asks for the current one only then.
	HasPassword bool `json:"hasPassword"`
}

// Partner statuses of an account.
const (
	PartnerNone     = "none"
	PartnerPending  = "pending"
	PartnerApproved = "approved"
	PartnerRejected = "rejected"
)

// Partner kinds.
const (
	KindOrganizer = "organizer"
	KindDriver    = "driver"
)

// ParticipantForm is the participant's form kept in the profile: the
// organizers of the tours they sign up for see it.
type ParticipantForm struct {
	EmergencyName  string `json:"emergencyName"`
	EmergencyPhone string `json:"emergencyPhone"`
	// none, hiking, trekking or mountaineering.
	Experience string  `json:"experience"`
	Health     *string `json:"health"`
	Notes      *string `json:"notes"`
}

// Achievement is a medal of the hall of fame, its title per language.
type Achievement struct {
	ID     string            `json:"id"`
	Title  map[string]string `json:"title"`
	Points int               `json:"points"`
}

type Award struct {
	Achievement Achievement `json:"achievement"`
}

// MyAchievements is the user's medals and the code partners award them by.
type MyAchievements struct {
	Code   string  `json:"code"`
	Points int     `json:"points"`
	Awards []Award `json:"awards"`
}

// HallEntry is a traveler on the hall of fame's leaderboard.
type HallEntry struct {
	Rank   int    `json:"rank"`
	User   Person `json:"user"`
	Points int    `json:"points"`
}

func (c *Client) Me(ctx context.Context, userID string) (Me, error) {
	var out struct {
		User Me `json:"user"`
	}
	return out.User, c.do(ctx, http.MethodGet, "/auth/user/me", userID, nil, &out)
}

// ParticipantForm is the profile's form; nil when it isn't filled in.
func (c *Client) ParticipantForm(ctx context.Context, userID string) (*ParticipantForm, error) {
	var out struct {
		Form *ParticipantForm `json:"form"`
	}
	return out.Form, c.do(ctx, http.MethodGet, "/tours/participant-form", userID, nil, &out)
}

func (c *Client) SaveParticipantForm(ctx context.Context, userID string, form ParticipantForm) error {
	return c.do(ctx, http.MethodPut, "/tours/participant-form", userID, map[string]any{"form": form}, nil)
}

// ChangePassword sets a new password (current: nil for an account without
// one). It ends the account's sessions on the site.
func (c *Client) ChangePassword(ctx context.Context, userID string, current *string, password string) error {
	body := map[string]any{"password": password}
	if current != nil {
		body["currentPassword"] = *current
	}
	return c.do(ctx, http.MethodPost, "/auth/user/password/change", userID, body, nil)
}

func (c *Client) MyAchievements(ctx context.Context, userID string) (MyAchievements, error) {
	var out MyAchievements
	return out, c.do(ctx, http.MethodGet, "/achievements/me", userID, nil, &out)
}

// HallOfFame is the leaderboard, best first.
func (c *Client) HallOfFame(ctx context.Context) ([]HallEntry, error) {
	var out struct {
		Leaders []HallEntry `json:"leaders"`
	}
	return out.Leaders, c.do(ctx, http.MethodGet, "/hall-of-fame", "", nil, &out)
}

// ApplyAsPartner sends a partner application (kind: organizer or driver).
func (c *Client) ApplyAsPartner(ctx context.Context, userID, kind string) (Me, error) {
	var out struct {
		User Me `json:"user"`
	}
	return out.User, c.do(ctx, http.MethodPost, "/partners/apply", userID, map[string]string{"kind": kind}, &out)
}
