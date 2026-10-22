package analytics_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/elvn5/sai/services/bot/internal/analytics"
)

// request is one Measurement Protocol request the fake Google received.
type request struct {
	query url.Values
	body  map[string]any
}

func fakeGoogle(t *testing.T, status int) (string, <-chan request) {
	t.Helper()
	got := make(chan request, 16)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := map[string]any{}
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("body: %v", err)
		}
		got <- request{query: r.URL.Query(), body: body}
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, got
}

func next(t *testing.T, got <-chan request) request {
	t.Helper()
	select {
	case r := <-got:
		return r
	case <-time.After(2 * time.Second):
		t.Fatal("no request")
		return request{}
	}
}

func run(t *testing.T, c *analytics.Client) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	go c.Run(ctx)
	t.Cleanup(func() {
		cancel()
		<-c.Done()
	})
}

func TestDisabledWithoutConfig(t *testing.T) {
	for _, cfg := range []analytics.Config{
		{},
		{MeasurementID: "G-TEST"},
		{APISecret: "secret"},
	} {
		if c := analytics.New(cfg, slog.New(slog.DiscardHandler)); c != nil {
			t.Errorf("%+v: enabled", cfg)
		}
	}
	// A nil client takes events and does nothing.
	var c *analytics.Client
	c.Track(analytics.Event{ChatID: 1, Name: "bot_start"})
}

func TestSendsToMeasurementID(t *testing.T) {
	endpoint, got := fakeGoogle(t, http.StatusNoContent)
	c := analytics.New(analytics.Config{MeasurementID: "G-TEST", APISecret: "s3cret", Salt: "pepper", Endpoint: endpoint},
		slog.New(slog.DiscardHandler))
	run(t, c)

	c.Track(analytics.Event{ChatID: 42, Name: "join_outing", Lang: "ky", Params: map[string]any{"outing_id": "o1"}})
	r := next(t, got)
	if r.query.Get("measurement_id") != "G-TEST" || r.query.Get("api_secret") != "s3cret" {
		t.Errorf("query: %v", r.query)
	}
	id := c.ChatHash(42)
	if len(id) != 32 || strings.Contains(id, "42") || r.body["client_id"] != id || r.body["user_id"] != id {
		t.Errorf("ids: %v (hash %s)", r.body, id)
	}
	events := r.body["events"].([]any)
	event := events[0].(map[string]any)
	params := event["params"].(map[string]any)
	if event["name"] != "join_outing" || params["outing_id"] != "o1" || params["lang"] != "ky" || params["engagement_time_msec"] != float64(1) {
		t.Errorf("event: %v", event)
	}
	if props := r.body["user_properties"].(map[string]any); props["lang"].(map[string]any)["value"] != "ky" {
		t.Errorf("user properties: %v", props)
	}
}

// The hash depends on the salt and the chat, and stays the same.
func TestChatHash(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	a := analytics.New(analytics.Config{MeasurementID: "G-TEST", APISecret: "s", Salt: "a"}, log)
	again := analytics.New(analytics.Config{MeasurementID: "G-TEST", APISecret: "s", Salt: "a"}, log)
	b := analytics.New(analytics.Config{MeasurementID: "G-TEST", APISecret: "s", Salt: "b"}, log)
	if a.ChatHash(42) != again.ChatHash(42) || a.ChatHash(42) == a.ChatHash(43) || a.ChatHash(42) == b.ChatHash(42) {
		t.Error("hash")
	}
}

// Events waiting together go in one request per chat; Google's failures are
// only logged, and what is queued at shutdown still goes out.
func TestBatchesPerChatAndFlushes(t *testing.T) {
	endpoint, got := fakeGoogle(t, http.StatusBadRequest)
	c := analytics.New(analytics.Config{MeasurementID: "G-TEST", APISecret: "s", Endpoint: endpoint}, slog.New(slog.DiscardHandler))
	for i := range 3 {
		c.Track(analytics.Event{ChatID: 1, Name: "e" + strconv.Itoa(i)})
	}
	c.Track(analytics.Event{ChatID: 2, Name: "other"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.Run(ctx)

	first, second := next(t, got), next(t, got)
	if n := len(first.body["events"].([]any)); n != 3 || first.body["client_id"] != c.ChatHash(1) {
		t.Errorf("first: %d events %v", n, first.body)
	}
	if second.body["client_id"] != c.ChatHash(2) {
		t.Errorf("second: %v", second.body)
	}
}

// Track never blocks: a full queue drops the event.
func TestTrackNeverBlocks(t *testing.T) {
	c := analytics.New(analytics.Config{MeasurementID: "G-TEST", APISecret: "s", Endpoint: "http://127.0.0.1:0"}, slog.New(slog.DiscardHandler))
	done := make(chan struct{})
	go func() {
		for range 1000 {
			c.Track(analytics.Event{ChatID: 1, Name: "e"})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Track blocked")
	}
}
