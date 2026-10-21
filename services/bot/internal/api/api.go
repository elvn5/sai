// Package api is the bot's view of the public API (the gateway): outings,
// tours, favorites, tour alerts and the blog. Requests that need a user go
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

type TrailRef struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
	Area string `json:"area"`
}

type Outing struct {
	ID            string     `json:"id"`
	Trail         *TrailRef  `json:"trail"`
	Title         string     `json:"title"`
	StartsAt      time.Time  `json:"startsAt"`
	ReturnsAt     *time.Time `json:"returnsAt"`
	MeetingPoint  string     `json:"meetingPoint"`
	Transport     string     `json:"transport"`
	Seats         int        `json:"seats"`
	Taken         int        `json:"taken"`
	CostPerPerson *int       `json:"costPerPerson"`
	Level         string     `json:"level"`
	Description   *string    `json:"description"`
	Contact       *string    `json:"contact"`
	Status        string     `json:"status"`
	Organizer     Person     `json:"organizer"`
	IsJoined      bool       `json:"isJoined"`
	IsOrganizer   bool       `json:"isOrganizer"`
	// Who joined: on one outing's details only.
	Participants []Person `json:"participants"`
	// The organizer's view of them: when each joined and arrived.
	Roster []RosterEntry `json:"roster"`
	// A participant's: whether they can rate it now, and their rating.
	CanRate  bool        `json:"canRate"`
	MyRating *RatingMine `json:"myRating"`
}

// RatingMine is the user's own 1–5 rating of an outing.
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

// OutingInput creates an outing or replaces an outing's fields. Times are
// RFC 3339; nil clears an optional field.
type OutingInput struct {
	TrailID       *string `json:"trailId"`
	Title         string  `json:"title"`
	StartsAt      string  `json:"startsAt"`
	ReturnsAt     *string `json:"returnsAt"`
	MeetingPoint  string  `json:"meetingPoint"`
	Transport     string  `json:"transport"`
	Seats         int     `json:"seats"`
	CostPerPerson *int    `json:"costPerPerson"`
	Level         string  `json:"level"`
	Description   *string `json:"description"`
	Contact       *string `json:"contact"`
}

// ConditionInput is a trail condition report.
type ConditionInput struct {
	ObservedOn string `json:"observedOn"` // YYYY-MM-DD
	TrailState string `json:"trailState"`
	Traction   string `json:"traction"`
}

type OutingsSummary struct {
	Outings       int     `json:"outings"`
	DistanceKm    float64 `json:"distanceKm"`
	ElevationGain int     `json:"elevationGain"`
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
}

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

// MyOutings lists what the user organizes or joined, latest start first.
// The list says only who organizes, so the rest are the ones joined.
func (c *Client) MyOutings(ctx context.Context, userID string) ([]Outing, error) {
	var out []Outing
	if err := c.do(ctx, http.MethodGet, "/outings/mine", userID, nil, &out); err != nil {
		return nil, err
	}
	for i := range out {
		out[i].IsJoined = !out[i].IsOrganizer
	}
	return out, nil
}

func (c *Client) OutingsSummary(ctx context.Context, userID string) (OutingsSummary, error) {
	var out OutingsSummary
	return out, c.do(ctx, http.MethodGet, "/outings/mine/summary", userID, nil, &out)
}

// UpcomingOutings lists open outings, soonest first.
func (c *Client) UpcomingOutings(ctx context.Context, userID string) ([]Outing, error) {
	var out []Outing
	return out, c.do(ctx, http.MethodGet, "/outings?limit=50", userID, nil, &out)
}

// Outing is one outing as the user sees it (the contact once joined).
func (c *Client) Outing(ctx context.Context, userID, id string) (Outing, error) {
	var out Outing
	return out, c.do(ctx, http.MethodGet, "/outings/"+url.PathEscape(id), userID, nil, &out)
}

// CreateOuting organizes an outing.
func (c *Client) CreateOuting(ctx context.Context, userID string, in OutingInput) (Outing, error) {
	var out Outing
	return out, c.do(ctx, http.MethodPost, "/outings", userID, in, &out)
}

// UpdateOuting changes an outing the user organizes.
func (c *Client) UpdateOuting(ctx context.Context, userID, id string, in OutingInput) (Outing, error) {
	var out Outing
	return out, c.do(ctx, http.MethodPut, "/outings/"+url.PathEscape(id), userID, in, &out)
}

// CancelOuting calls off an outing the user organizes.
func (c *Client) CancelOuting(ctx context.Context, userID, id string) error {
	return c.do(ctx, http.MethodPost, "/outings/"+url.PathEscape(id)+"/cancel", userID, nil, nil)
}

// SetArrival marks a participant of an outing the user organizes as
// arrived at the meeting point, or not.
func (c *Client) SetArrival(ctx context.Context, userID, outingID, participantID string, arrived bool) (Outing, error) {
	method := http.MethodPut
	if !arrived {
		method = http.MethodDelete
	}
	var out Outing
	path := "/outings/" + url.PathEscape(outingID) + "/participants/" + url.PathEscape(participantID) + "/arrival"
	return out, c.do(ctx, method, path, userID, nil, &out)
}

// RateOuting rates an outing the user went on (1–5, an optional comment);
// rating again changes it.
func (c *Client) RateOuting(ctx context.Context, userID, outingID string, rating int, text *string) (Outing, error) {
	var out Outing
	body := map[string]any{"rating": rating, "text": text}
	return out, c.do(ctx, http.MethodPut, "/outings/"+url.PathEscape(outingID)+"/rating", userID, body, &out)
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

// Trails lists published routes, most popular first; search narrows them
// by name.
func (c *Client) Trails(ctx context.Context, search string) ([]TrailRef, error) {
	q := url.Values{"sort": {"popular"}}
	if search != "" {
		q.Set("search", search)
	}
	var out []TrailRef
	return out, c.do(ctx, http.MethodGet, "/trails?"+q.Encode(), "", nil, &out)
}

// ReportCondition posts what a route looked like.
func (c *Client) ReportCondition(ctx context.Context, userID, trail string, in ConditionInput) error {
	return c.do(ctx, http.MethodPost, "/trails/"+url.PathEscape(trail)+"/conditions", userID, in, nil)
}

func (c *Client) JoinOuting(ctx context.Context, userID, id string) error {
	return c.do(ctx, http.MethodPost, "/outings/"+url.PathEscape(id)+"/join", userID, nil, nil)
}

func (c *Client) LeaveOuting(ctx context.Context, userID, id string) error {
	return c.do(ctx, http.MethodDelete, "/outings/"+url.PathEscape(id)+"/join", userID, nil, nil)
}

// UpcomingTours lists published Russian tours departing from today on.
func (c *Client) UpcomingTours(ctx context.Context) ([]Tour, error) {
	var out []Tour
	q := url.Values{"locale": {"ru"}, "from": {c.Now().Format(time.DateOnly)}}
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

// BlogPosts lists published Russian posts, newest first.
func (c *Client) BlogPosts(ctx context.Context) ([]BlogPost, error) {
	var out []BlogPost
	return out, c.do(ctx, http.MethodGet, "/blog?locale=ru", "", nil, &out)
}

// BlogPost is one published post with its text.
func (c *Client) BlogPost(ctx context.Context, locale, slug string) (BlogPost, error) {
	var out BlogPost
	return out, c.do(ctx, http.MethodGet, "/blog/"+url.PathEscape(locale)+"/"+url.PathEscape(slug), "", nil, &out)
}
