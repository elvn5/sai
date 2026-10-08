package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/middleware"
)

const allowed = "https://ermekten.com"

func newCORSRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CORS([]string{allowed, "https://admin.ermekten.com"}))
	r.GET("/thing", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return r
}

func TestCORSAllowedOrigin(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/thing", nil)
	req.Header.Set("Origin", allowed)
	newCORSRouter().ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != allowed {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, allowed)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want true", got)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("handler did not run, body = %q", rec.Body.String())
	}
}

// The cors npm package doesn't block unknown origins — it just omits the
// allow-origin header and lets the browser enforce it.
func TestCORSDisallowedOriginStillServed(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/thing", nil)
	req.Header.Set("Origin", "https://evil.example")
	newCORSRouter().ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestCORSPreflight(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/anything/even/unrouted", nil)
	req.Header.Set("Origin", allowed)
	req.Header.Set("Access-Control-Request-Method", "PUT")
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	newCORSRouter().ServeHTTP(rec, req)

	checks := map[string]string{
		"Access-Control-Allow-Origin":      allowed,
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Allow-Methods":     "GET,HEAD,PUT,PATCH,POST,DELETE",
		"Access-Control-Allow-Headers":     "content-type",
		"Vary":                             "Origin, Access-Control-Request-Headers",
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	for name, want := range checks {
		if got := rec.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}
