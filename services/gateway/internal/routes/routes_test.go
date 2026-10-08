package routes_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/gateway/internal/config"
	"github.com/elvn5/sai/services/gateway/internal/routes"
)

const clientOrigin = "https://ermekten.com"

type seen struct {
	method, uri, host, cookie, forwardedFor, forwardedProto, contentType string
	body                                                                 []byte
}

// newGateway starts a fake legacy API that records what it received and
// replies like NestJS (including its own CORS headers and a session cookie).
func newGateway(t *testing.T) (*httptest.Server, *seen) {
	t.Helper()
	got := &seen{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		*got = seen{
			method: r.Method, uri: r.RequestURI, host: r.Host, cookie: r.Header.Get("Cookie"),
			forwardedFor: r.Header.Get("X-Forwarded-For"), forwardedProto: r.Header.Get("X-Forwarded-Proto"),
			contentType: r.Header.Get("Content-Type"), body: body,
		}
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Vary", "Origin, Accept-Encoding")
		http.SetCookie(w, &http.Cookie{Name: "user_token", Value: "abc", HttpOnly: true, Path: "/"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(upstream.Close)

	gin.SetMode(gin.TestMode)
	router, err := routes.Setup(config.Config{
		LegacyAPIURL:    upstream.URL,
		AdminOrigin:     "https://admin.ermekten.com",
		ClientOrigin:    clientOrigin,
		UpstreamTimeout: 5 * time.Second,
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	return serve(t, router), got
}

// serve runs the router on a real HTTP server: Gin's response writer relies
// on http.CloseNotifier, which httptest.ResponseRecorder doesn't implement.
func serve(t *testing.T, router *gin.Engine) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, srv *httptest.Server, req *http.Request) (*http.Response, string) {
	t.Helper()
	req.RequestURI = ""
	req.URL.Scheme = "http"
	req.URL.Host = srv.Listener.Addr().String()
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, _ := io.ReadAll(res.Body)
	return res, string(body)
}

func TestForwardsRequestUnchanged(t *testing.T) {
	srv, got := newGateway(t)

	req := httptest.NewRequest(http.MethodPut, "/hotels/abc-123?x=1&y=%D0%B0", bytes.NewBufferString(`{"name":"Kok-Jaiyk"}`))
	req.Host = "api.ermekten.com"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "user_token=tok; i18n_redirected=ru")
	req.Header.Set("Origin", clientOrigin)
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	req.Header.Set("X-Forwarded-Proto", "https")
	res, body := do(t, srv, req)

	if res.StatusCode != http.StatusCreated || body != `{"ok":true}` {
		t.Fatalf("response = %d %s", res.StatusCode, body)
	}
	if got.method != http.MethodPut || got.uri != "/hotels/abc-123?x=1&y=%D0%B0" {
		t.Errorf("upstream saw %s %s", got.method, got.uri)
	}
	if string(got.body) != `{"name":"Kok-Jaiyk"}` || got.contentType != "application/json" {
		t.Errorf("body/content-type not forwarded: %q %q", got.body, got.contentType)
	}
	if got.cookie != "user_token=tok; i18n_redirected=ru" {
		t.Errorf("cookie = %q", got.cookie)
	}
	if got.host != "api.ermekten.com" {
		t.Errorf("host = %q, want the public host", got.host)
	}
	if got.forwardedFor != "203.0.113.7, 127.0.0.1" {
		t.Errorf("X-Forwarded-For = %q, want nginx chain extended", got.forwardedFor)
	}
	if got.forwardedProto != "https" {
		t.Errorf("X-Forwarded-Proto = %q, want https kept from nginx", got.forwardedProto)
	}
	if setCookie := res.Header.Get("Set-Cookie"); setCookie != "user_token=abc; Path=/; HttpOnly" {
		t.Errorf("Set-Cookie = %q", setCookie)
	}
}

// The gateway owns CORS; the upstream's copies must not be duplicated.
func TestCORSHeadersNotDuplicated(t *testing.T) {
	srv, _ := newGateway(t)
	req := httptest.NewRequest(http.MethodGet, "/tours?locale=ru", nil)
	req.Header.Set("Origin", clientOrigin)
	res, _ := do(t, srv, req)

	if got := res.Header.Values("Access-Control-Allow-Origin"); len(got) != 1 || got[0] != clientOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q", got)
	}
	if got := res.Header.Values("Access-Control-Allow-Credentials"); len(got) != 1 {
		t.Errorf("Access-Control-Allow-Credentials = %q", got)
	}
	if got := res.Header.Values("Vary"); len(got) != 2 || got[0] != "Origin" || got[1] != "Accept-Encoding" {
		t.Errorf("Vary = %q, want [Origin Accept-Encoding]", got)
	}
}

func TestPreflightAnsweredByGateway(t *testing.T) {
	srv, got := newGateway(t)
	req := httptest.NewRequest(http.MethodOptions, "/reviews", nil)
	req.Header.Set("Origin", clientOrigin)
	req.Header.Set("Access-Control-Request-Method", "POST")
	res, _ := do(t, srv, req)

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", res.StatusCode)
	}
	if got.method != "" {
		t.Errorf("preflight reached the upstream (%s)", got.method)
	}
}

func TestMultipartUploadStreamsThrough(t *testing.T) {
	srv, got := newGateway(t)
	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)
	part, _ := form.CreateFormFile("file", "photo.png")
	_, _ = part.Write(bytes.Repeat([]byte{0x89}, 1<<20))
	_ = form.Close()

	req := httptest.NewRequest(http.MethodPost, "/uploads/image", &buf)
	req.Header.Set("Content-Type", form.FormDataContentType())
	res, _ := do(t, srv, req)

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d", res.StatusCode)
	}
	if got.contentType != form.FormDataContentType() || len(got.body) <= 1<<20 {
		t.Errorf("multipart body not forwarded intact: %q, %d bytes", got.contentType, len(got.body))
	}
}

func TestUpstreamDownReturnsNestStyle502(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, err := routes.Setup(config.Config{
		LegacyAPIURL:    "http://127.0.0.1:1",
		UpstreamTimeout: time.Second,
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	res, raw := do(t, serve(t, router), httptest.NewRequest(http.MethodGet, "/blog?locale=ru", nil))

	var body map[string]any
	_ = json.Unmarshal([]byte(raw), &body)
	if res.StatusCode != http.StatusBadGateway || body["statusCode"] != float64(502) || body["error"] != "Bad Gateway" {
		t.Fatalf("got %d %s", res.StatusCode, raw)
	}
}

func TestHealth(t *testing.T) {
	srv, got := newGateway(t)
	res, _ := do(t, srv, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if res.StatusCode != http.StatusOK || got.method != "" {
		t.Fatalf("health = %d, upstream hit = %q", res.StatusCode, got.method)
	}
}

func TestMediaRoutesGoToMediaService(t *testing.T) {
	hits := map[string]string{}
	upstream := func(name string) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits[r.Method+" "+r.URL.Path] = name
		}))
		t.Cleanup(srv.Close)
		return srv
	}
	legacy, media := upstream("legacy"), upstream("media")

	gin.SetMode(gin.TestMode)
	router, err := routes.Setup(config.Config{
		LegacyAPIURL: legacy.URL, MediaURL: media.URL, UpstreamTimeout: 5 * time.Second,
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	srv := serve(t, router)

	want := map[string]string{
		"POST /uploads/image":      "media",
		"GET /uploads/a.webp":      "media",
		"POST /storage/upload-url": "media",
		"DELETE /storage/file":     "media",
		"GET /hotels":              "legacy",
		"GET /gallery":             "legacy",
		"GET /uploadsx":            "legacy",
	}
	for route := range want {
		method, path, _ := strings.Cut(route, " ")
		do(t, srv, httptest.NewRequest(method, path, nil))
	}
	for route, name := range want {
		if hits[route] != name {
			t.Errorf("%s went to %q, want %q", route, hits[route], name)
		}
	}
}
