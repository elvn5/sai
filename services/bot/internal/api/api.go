// Package api is the bot's view of the public API (the gateway): tours
// and signing up for them, favorites, tour alerts and the blog. Requests that need a user go
// with a short-lived session of that user, signed with the shared
// JWT_SECRET, so the services treat them exactly like the site's.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/elvn5/sai/services/pkg/auth"
)

const (
	callTimeout = 10 * time.Second
	// Lifetime of the sessions the bot signs: one request.
	sessionTTL = "5m"
)

// Error is a failed API call: its HTTP status, the error body's message
// and, for a rejected form, the first error of each field.
type Error struct {
	Status  int
	Message string
	Fields  map[string]string
}

func (e *Error) Error() string { return fmt.Sprintf("api: %d %s", e.Status, e.Message) }

// StatusOf is the HTTP status of a failed call, 0 for other errors.
func StatusOf(err error) int {
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr.Status
	}
	return 0
}

// MessageOf is the API's error message, "" for other errors.
func MessageOf(err error) string {
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr.Message
	}
	return ""
}

// FieldsOf is a rejected form's errors by field, nil for other errors.
func FieldsOf(err error) map[string]string {
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr.Fields
	}
	return nil
}

type Person struct {
	ID        string  `json:"id"`
	FirstName *string `json:"firstName"`
	Username  *string `json:"username"`
}

// Name is how the bot calls a person: first name, else @username.
func (p Person) Name() string {
	switch {
	case p.FirstName != nil && *p.FirstName != "":
		return *p.FirstName
	case p.Username != nil && *p.Username != "":
		return "@" + *p.Username
	default:
		return ""
	}
}

// RatingMine is the user's own 1–5 rating of a tour they went on.
type RatingMine struct {
	Rating int     `json:"rating"`
	Text   *string `json:"text"`
}

// RosterEntry is a participant as the organizer sees them.
type RosterEntry struct {
	Person
	JoinedAt  time.Time  `json:"joinedAt"`
	ArrivedAt *time.Time `json:"arrivedAt"`
}

type Tour struct {
	ID            string  `json:"id"`
	Locale        string  `json:"locale"`
	Slug          string  `json:"slug"`
	Title         string  `json:"title"`
	Excerpt       *string `json:"excerpt"`
	Price         *int    `json:"price"`
	DurationDays  *int    `json:"durationDays"`
	Location      *string `json:"location"`
	StartDate     string  `json:"startDate"`
	EndDate       string  `json:"endDate"`
	CoverImageURL *string `json:"coverImageUrl"`
	RatingAverage float32 `json:"ratingAverage"`
	ReviewCount   int     `json:"reviewCount"`

	// On one tour's details and the organizer's own list.
	Content  string       `json:"content"` // HTML
	Program  []ProgramDay `json:"program"`
	Capacity *int         `json:"capacity"`
	Phone    *string      `json:"phone"`
	Whatsapp *string      `json:"whatsapp"`
	Telegram *string      `json:"telegram"`
	// On the organizer's own list.
	IsPublished bool `json:"isPublished"`

	// Signing up: where and when (HH:MM, Bishkek, on StartDate) the group
	// meets, and the travelers signed up.
	MeetingPoint *string `json:"meetingPoint"`
	MeetingTime  *string `json:"meetingTime"`
	Taken        int     `json:"taken"`
	Organizer    *Person `json:"organizer"`
	// As the user sees it (one tour by id, the user's own lists).
	IsJoined    bool `json:"isJoined"`
	IsOrganizer bool `json:"isOrganizer"`
	// The user's place on one tour: "waitlist" while waiting for one.
	MyStatus string `json:"myStatus"`
	// The organizer's: who signed up and who came, and how many wait for a
	// place.
	Roster   []RosterEntry `json:"roster"`
	Waitlist int           `json:"waitlist"`
	// A participant's: whether they can rate it now, and their rating.
	CanRate  bool        `json:"canRate"`
	MyRating *RatingMine `json:"myRating"`
}

// bishkek is the time zone tours' dates and meeting times are in.
var bishkek = time.FixedZone("Asia/Bishkek", 6*60*60)

// defaultMeetingTime stands in for a tour without one, as listings does.
const defaultMeetingTime = "09:00"

// StartsAt is when the group meets: the start date at the meeting time in
// Bishkek (zero for a malformed date).
func (t Tour) StartsAt() time.Time {
	at := defaultMeetingTime
	if t.MeetingTime != nil && *t.MeetingTime != "" {
		at = *t.MeetingTime
	}
	starts, err := time.ParseInLocation(time.DateOnly+" 15:04", t.StartDate+" "+at, bishkek)
	if err != nil {
		day, _ := time.ParseInLocation(time.DateOnly, t.StartDate, bishkek)
		return day
	}
	return starts
}

// Full: every place is taken (a tour without a limit never is).
func (t Tour) Full() bool { return t.Capacity != nil && t.Taken >= *t.Capacity }

// Waiting reports whether the user is on the tour's waitlist.
func (t Tour) Waiting() bool { return t.MyStatus == "waitlist" }

// StatCounts are a listing's visits and contact-button clicks.
type StatCounts struct {
	Views    int `json:"views"`
	Contacts int `json:"contacts"`
}

// ListingStats are the organizer's listings' numbers over the last days.
type ListingStats struct {
	Totals   StatCounts `json:"totals"`
	Listings []struct {
		Listing struct {
			TargetType string `json:"targetType"`
			ID         string `json:"id"`
		} `json:"listing"`
		StatCounts
	} `json:"listings"`
}

// Of is one listing's numbers (zero when it had none).
func (s ListingStats) Of(targetType, id string) StatCounts {
	for _, l := range s.Listings {
		if l.Listing.TargetType == targetType && l.Listing.ID == id {
			return l.StatCounts
		}
	}
	return StatCounts{}
}

// Order is a paid booking of the organizer's listing.
type Order struct {
	ID           string     `json:"id"`
	Kind         string     `json:"kind"`
	ItemID       string     `json:"itemId"`
	ItemDate     *string    `json:"itemDate"`
	Quantity     int        `json:"quantity"`
	Amount       int        `json:"amount"`
	ContactName  string     `json:"contactName"`
	ContactPhone string     `json:"contactPhone"`
	Comment      *string    `json:"comment"`
	Status       string     `json:"status"`
	PaidAt       *time.Time `json:"paidAt"`
}

// ProgramDay is a day of a tour's program (plain text).
type ProgramDay struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type BlogPost struct {
	ID            string     `json:"id"`
	Locale        string     `json:"locale"`
	Slug          string     `json:"slug"`
	Title         string     `json:"title"`
	Excerpt       *string    `json:"excerpt"`
	CoverImageURL *string    `json:"coverImageUrl"`
	PublishedAt   *time.Time `json:"publishedAt"`
	// On one post only.
	Content string `json:"content"` // HTML
}

type Client struct {
	base   string
	signer *auth.Signer
	http   *http.Client
	// Now is overridable in tests.
	Now func() time.Time
}

// New returns a client of the gateway at baseURL, acting as users with
// sessions signed with jwtSecret.
func New(baseURL, jwtSecret string) (*Client, error) {
	signer, err := auth.NewSigner(jwtSecret, sessionTTL)
	if err != nil {
		return nil, fmt.Errorf("session signer: %w", err)
	}
	return &Client{
		base: strings.TrimRight(baseURL, "/"), signer: signer,
		http: &http.Client{Timeout: callTimeout}, Now: time.Now,
	}, nil
}

// do calls the API, as userID when it isn't "", and decodes the JSON
// answer into out (when not nil).
func (c *Client) do(ctx context.Context, method, path, userID string, body, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if userID != "" {
		token, err := c.signer.UserToken(userID)
		if err != nil {
			return fmt.Errorf("sign session: %w", err)
		}
		// A request cookie: the attributes of a response cookie don't apply.
		req.Header.Set("Cookie", auth.UserCookie+"="+token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode >= http.StatusBadRequest {
		var e struct {
			Message     any                 `json:"message"`
			FieldErrors map[string][]string `json:"fieldErrors"`
		}
		_ = json.NewDecoder(res.Body).Decode(&e)
		msg, _ := e.Message.(string)
		apiErr := &Error{Status: res.StatusCode, Message: msg}
		for field, errs := range e.FieldErrors {
			if len(errs) > 0 {
				if apiErr.Fields == nil {
					apiErr.Fields = map[string]string{}
				}
				apiErr.Fields[field] = errs[0]
			}
		}
		return apiErr
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return fmt.Errorf("%s %s: decode: %w", method, path, err)
	}
	return nil
}

// JoinedTours lists the tours the user signed up for, latest start first.
func (c *Client) JoinedTours(ctx context.Context, userID string) ([]Tour, error) {
	var out []Tour
	return out, c.do(ctx, http.MethodGet, "/tours/joined", userID, nil, &out)
}

// TourByID is one tour as the user sees it: their place, and for its
// organizer the roster.
func (c *Client) TourByID(ctx context.Context, userID, id string) (Tour, error) {
	var out Tour
	return out, c.do(ctx, http.MethodGet, "/tours/by-id/"+url.PathEscape(id), userID, nil, &out)
}

// JoinTour signs the user up for a tour.
func (c *Client) JoinTour(ctx context.Context, userID, id string) (Tour, error) {
	var out Tour
	return out, c.do(ctx, http.MethodPost, "/tours/"+url.PathEscape(id)+"/join", userID, nil, &out)
}

// LeaveTour gives up the user's place on a tour.
func (c *Client) LeaveTour(ctx context.Context, userID, id string) (Tour, error) {
	var out Tour
	return out, c.do(ctx, http.MethodDelete, "/tours/"+url.PathEscape(id)+"/join", userID, nil, &out)
}

// SetArrival marks a participant of a tour the user runs as arrived at the
// meeting point, or not.
func (c *Client) SetArrival(ctx context.Context, userID, tourID, participantID string, arrived bool) (Tour, error) {
	method := http.MethodPut
	if !arrived {
		method = http.MethodDelete
	}
	var out Tour
	path := "/tours/" + url.PathEscape(tourID) + "/participants/" + url.PathEscape(participantID) + "/arrival"
	return out, c.do(ctx, method, path, userID, nil, &out)
}

// RateTour rates a tour the user went on (1–5, an optional comment): their
// review of it; rating again changes it.
func (c *Client) RateTour(ctx context.Context, userID, tourID string, rating int, text *string) (Tour, error) {
	var out Tour
	body := map[string]any{"rating": rating, "text": text}
	return out, c.do(ctx, http.MethodPut, "/tours/"+url.PathEscape(tourID)+"/rating", userID, body, &out)
}

// MyTours lists the tours the user runs, published or not, newest first.
func (c *Client) MyTours(ctx context.Context, userID string) ([]Tour, error) {
	var out []Tour
	return out, c.do(ctx, http.MethodGet, "/tours/mine", userID, nil, &out)
}

// ListingStats are the user's listings' visits over the last 30 days
// (partners only: 403 for others).
func (c *Client) ListingStats(ctx context.Context, userID string) (ListingStats, error) {
	var out ListingStats
	return out, c.do(ctx, http.MethodGet, "/listing-stats/mine?days=30", userID, nil, &out)
}

// ReceivedOrders lists paid bookings of the user's listings, latest first
// (partners only).
func (c *Client) ReceivedOrders(ctx context.Context, userID string) ([]Order, error) {
	var out []Order
	return out, c.do(ctx, http.MethodGet, "/orders/received", userID, nil, &out)
}

// UpcomingTours lists published tours in a language departing from today
// on.
func (c *Client) UpcomingTours(ctx context.Context, locale string) ([]Tour, error) {
	var out []Tour
	q := url.Values{"locale": {locale}, "from": {c.Now().Format(time.DateOnly)}}
	return out, c.do(ctx, http.MethodGet, "/tours?"+q.Encode(), "", nil, &out)
}

// Tour is one published tour with its description and program.
func (c *Client) Tour(ctx context.Context, locale, slug string) (Tour, error) {
	var out Tour
	return out, c.do(ctx, http.MethodGet, "/tours/"+url.PathEscape(locale)+"/"+url.PathEscape(slug), "", nil, &out)
}

func (c *Client) FavoriteTours(ctx context.Context, userID string) ([]Tour, error) {
	var out struct {
		Tours []Tour `json:"tours"`
	}
	return out.Tours, c.do(ctx, http.MethodGet, "/favorites/mine", userID, nil, &out)
}

// SetFavoriteTour saves or forgets a tour (both idempotent).
func (c *Client) SetFavoriteTour(ctx context.Context, userID, tourID string, saved bool) error {
	method := http.MethodPut
	if !saved {
		method = http.MethodDelete
	}
	return c.do(ctx, method, "/favorites/tour/"+url.PathEscape(tourID), userID, nil, nil)
}

// TourAlerts lists the destinations the user follows; "" is any.
func (c *Client) TourAlerts(ctx context.Context, userID string) ([]string, error) {
	var out struct {
		Destinations []string `json:"destinations"`
	}
	return out.Destinations, c.do(ctx, http.MethodGet, "/tour-alerts/mine", userID, nil, &out)
}

// SetAllToursAlert follows (or stops following) every new tour.
func (c *Client) SetAllToursAlert(ctx context.Context, userID string, on bool) error {
	method := http.MethodPut
	if !on {
		method = http.MethodDelete
	}
	return c.do(ctx, method, "/tour-alerts", userID, map[string]string{"destination": ""}, nil)
}

// BlogPosts lists published posts in a language, newest first.
func (c *Client) BlogPosts(ctx context.Context, locale string) ([]BlogPost, error) {
	var out []BlogPost
	return out, c.do(ctx, http.MethodGet, "/blog?locale="+url.QueryEscape(locale), "", nil, &out)
}

// BlogPost is one published post with its text.
func (c *Client) BlogPost(ctx context.Context, locale, slug string) (BlogPost, error) {
	var out BlogPost
	return out, c.do(ctx, http.MethodGet, "/blog/"+url.PathEscape(locale)+"/"+url.PathEscape(slug), "", nil, &out)
}
