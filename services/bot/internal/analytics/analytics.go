// Package analytics sends the bot's events to Google Analytics 4 (the
// Firebase project's property) through the Measurement Protocol: Firebase
// has no Go SDK for Analytics. Events are queued and sent in the
// background, so a handler never waits for Google; when the queue is full
// an event is dropped.
//
// A chat is identified by a hash of its id (client_id and user_id), never
// the id itself.
package analytics

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	// DefaultEndpoint is GA4's Measurement Protocol.
	DefaultEndpoint = "https://www.google-analytics.com/mp/collect"
	queueSize       = 256
	sendTimeout     = 5 * time.Second
	// A Measurement Protocol request takes up to 25 events.
	batchSize = 25
)

// Config is where events go: the property's web data stream
// (MeasurementID, "G-…", the site's too) and its Measurement Protocol API
// secret. Not a Firebase app stream: those take only the app_instance_id
// of a real SDK installation.
type Config struct {
	MeasurementID string
	APISecret     string
	// Salt keys the hash of chat ids, so they can't be recovered from
	// Analytics by hashing every possible id.
	Salt string
	// Endpoint overrides DefaultEndpoint (tests).
	Endpoint string
}

// Enabled reports whether the config names a stream and its secret.
func (c Config) Enabled() bool {
	return c.APISecret != "" && c.MeasurementID != ""
}

// Event is one thing a chat did. Lang is the chat's language, sent with
// every event as the "lang" parameter and user property.
type Event struct {
	ChatID int64
	Name   string
	Lang   string
	Params map[string]any
}

// Client queues events and sends them from Run.
type Client struct {
	cfg    Config
	http   *http.Client
	log    *slog.Logger
	events chan Event
	done   chan struct{}
}

// New returns a client for cfg (nil when cfg is not Enabled: Track on a
// nil client does nothing).
func New(cfg Config, log *slog.Logger) *Client {
	if !cfg.Enabled() {
		return nil
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultEndpoint
	}
	return &Client{
		cfg: cfg, log: log, http: &http.Client{Timeout: sendTimeout},
		events: make(chan Event, queueSize), done: make(chan struct{}),
	}
}

// Track queues an event without waiting.
func (c *Client) Track(e Event) {
	if c == nil {
		return
	}
	select {
	case c.events <- e:
	default:
		c.log.Warn("analytics queue full, event dropped", slog.String("event", e.Name))
	}
}

// Run sends queued events until ctx ends, then sends what is left.
func (c *Client) Run(ctx context.Context) {
	defer close(c.done)
	for {
		select {
		case e := <-c.events:
			c.send(context.WithoutCancel(ctx), c.drain(e))
		case <-ctx.Done():
			for {
				select {
				case e := <-c.events:
					c.send(context.WithoutCancel(ctx), c.drain(e))
				default:
					return
				}
			}
		}
	}
}

// Done is closed once Run has sent the last events.
func (c *Client) Done() <-chan struct{} { return c.done }

// drain takes the events already waiting behind first, up to a batch.
func (c *Client) drain(first Event) []Event {
	batch := []Event{first}
	for len(batch) < batchSize {
		select {
		case e := <-c.events:
			batch = append(batch, e)
		default:
			return batch
		}
	}
	return batch
}

// send posts the events, one request per chat (each request is one user).
func (c *Client) send(ctx context.Context, events []Event) {
	var order []int64
	byChat := map[int64][]Event{}
	for _, e := range events {
		if _, seen := byChat[e.ChatID]; !seen {
			order = append(order, e.ChatID)
		}
		byChat[e.ChatID] = append(byChat[e.ChatID], e)
	}
	for _, chatID := range order {
		if err := c.post(ctx, chatID, byChat[chatID]); err != nil {
			c.log.WarnContext(ctx, "analytics", slog.Any("error", err))
		}
	}
}

type payload struct {
	ClientID       string                  `json:"client_id"`
	UserID         string                  `json:"user_id"`
	UserProperties map[string]propertyJSON `json:"user_properties,omitempty"`
	Events         []eventJSON             `json:"events"`
}

type propertyJSON struct {
	Value string `json:"value"`
}

type eventJSON struct {
	Name   string         `json:"name"`
	Params map[string]any `json:"params"`
}

func (c *Client) post(ctx context.Context, chatID int64, events []Event) error {
	id := c.ChatHash(chatID)
	p := payload{ClientID: id, UserID: id, Events: make([]eventJSON, 0, len(events))}
	for _, e := range events {
		params := map[string]any{
			// Counts the chat as an active user in the reports.
			"engagement_time_msec": 1,
		}
		for k, v := range e.Params {
			params[k] = v
		}
		if e.Lang != "" {
			params["lang"] = e.Lang
			p.UserProperties = map[string]propertyJSON{"lang": {Value: e.Lang}}
		}
		p.Events = append(p.Events, eventJSON{Name: e.Name, Params: params})
	}
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("encode events: %w", err)
	}

	query := url.Values{"measurement_id": {c.cfg.MeasurementID}, "api_secret": {c.cfg.APISecret}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint+"?"+query.Encode(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("analytics request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		// The URL holds the API secret: report the event names only.
		return fmt.Errorf("send %d events: %w", len(events), unwrapURL(err))
	}
	_ = res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("send %d events: status %d", len(events), res.StatusCode)
	}
	return nil
}

// unwrapURL drops the request URL (with the API secret) from err.
func unwrapURL(err error) error {
	var u *url.Error
	if errors.As(err, &u) {
		return u.Err
	}
	return err
}

// ChatHash is a chat's id in Analytics (client_id and user_id): 32 hex
// digits of an HMAC of the id.
func (c *Client) ChatHash(chatID int64) string {
	mac := hmac.New(sha256.New, []byte(c.cfg.Salt))
	mac.Write([]byte(strconv.FormatInt(chatID, 10)))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}
