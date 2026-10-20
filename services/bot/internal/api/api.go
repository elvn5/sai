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

// Error is a failed API call: its HTTP status and the error body's message.
type Error struct {
	Status  int
	Message string
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
}

type BlogPost struct {
	ID            string     `json:"id"`
	Locale        string     `json:"locale"`
	Slug          string     `json:"slug"`
	Title         string     `json:"title"`
	Excerpt       *string    `json:"excerpt"`
	CoverImageURL *string    `json:"coverImageUrl"`
	PublishedAt   *time.Time `json:"publishedAt"`
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
			Message any `json:"message"`
		}
		_ = json.NewDecoder(res.Body).Decode(&e)
		msg, _ := e.Message.(string)
		return &Error{Status: res.StatusCode, Message: msg}
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
