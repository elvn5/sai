package routes_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/elvn5/sai/services/media/internal/config"
	"github.com/elvn5/sai/services/media/internal/routes"
	"github.com/elvn5/sai/services/media/internal/storage"
)

const secret = "test-secret-at-least-16-chars"

type env struct {
	router *gin.Engine
	dir    string
	cookie *http.Cookie
}

func setup(t *testing.T, cfg config.Config) env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg.JWTSecret = secret
	cfg.UploadsDir = t.TempDir()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "u1", "role": "user"}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return env{
		router: routes.Setup(cfg, storage.NewR2(cfg), nil, slog.New(slog.DiscardHandler)),
		dir:    cfg.UploadsDir,
		cookie: &http.Cookie{Name: "user_token", Value: token},
	}
}

func (e env) do(req *http.Request, authed bool) *httptest.ResponseRecorder {
	if authed {
		req.AddCookie(e.cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

type filePart struct {
	field, filename, contentType string
	data                         []byte
}

func multipartRequest(t *testing.T, path string, parts ...filePart) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("note", "ignored text field")
	for _, p := range parts {
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", `form-data; name="`+p.field+`"; filename="`+p.filename+`"`)
		h.Set("Content-Type", p.contentType)
		pw, err := w.CreatePart(h)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = pw.Write(p.data)
	}
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func pngBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func assertNestError(t *testing.T, rec *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != status || body["message"] != message || body["statusCode"] != float64(status) {
		t.Fatalf("got %d %s, want %d %q", rec.Code, rec.Body.String(), status, message)
	}
}

func TestImageUploadConvertsAndServes(t *testing.T) {
	e := setup(t, config.Config{})
	rec := e.do(multipartRequest(t, "/uploads/image", filePart{"file", "photo.png", "image/png", pngBytes(3000, 1500)}), true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d %s", rec.Code, rec.Body.String())
	}
	var res struct{ URL string }
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if !regexp.MustCompile(`^/uploads/[0-9a-f-]{36}\.webp$`).MatchString(res.URL) {
		t.Fatalf("url = %q", res.URL)
	}

	get := e.do(httptest.NewRequest(http.MethodGet, res.URL, nil), false)
	if get.Code != http.StatusOK || get.Header().Get("Content-Type") != "image/webp" {
		t.Fatalf("GET %s = %d %s", res.URL, get.Code, get.Header().Get("Content-Type"))
	}
	if get.Header().Get("Cache-Control") != "public, max-age=0" || !strings.HasPrefix(get.Header().Get("ETag"), `W/"`) {
		t.Errorf("cache headers = %v", get.Header())
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(get.Body.Bytes()))
	if err != nil || cfg.Width != 2000 || cfg.Height != 1000 {
		t.Errorf("served image %dx%d, err %v", cfg.Width, cfg.Height, err)
	}

	cond := httptest.NewRequest(http.MethodGet, res.URL, nil)
	cond.Header.Set("If-None-Match", get.Header().Get("ETag"))
	if rec := e.do(cond, false); rec.Code != http.StatusNotModified {
		t.Errorf("conditional GET = %d, want 304", rec.Code)
	}
}

func TestImageUploadErrors(t *testing.T) {
	e := setup(t, config.Config{})
	cases := []struct {
		name    string
		req     *http.Request
		authed  bool
		status  int
		message string
	}{
		{"anonymous", multipartRequest(t, "/uploads/image", filePart{"file", "a.png", "image/png", pngBytes(4, 4)}), false, 401, "Not authenticated"},
		{"wrong type", multipartRequest(t, "/uploads/image", filePart{"file", "a.svg", "image/svg+xml", []byte("<svg/>")}), true, 400, "Only JPEG, PNG, WEBP or GIF images are allowed"},
		{"no file", multipartRequest(t, "/uploads/image"), true, 400, "No file uploaded"},
		{"not multipart", httptest.NewRequest(http.MethodPost, "/uploads/image", strings.NewReader("{}")), true, 400, "No file uploaded"},
		{"wrong field", multipartRequest(t, "/uploads/image", filePart{"image", "a.png", "image/png", pngBytes(4, 4)}), true, 400, "Unexpected field"},
		{"two files", multipartRequest(t, "/uploads/image", filePart{"file", "a.png", "image/png", pngBytes(4, 4)}, filePart{"file", "b.png", "image/png", pngBytes(4, 4)}), true, 400, "Unexpected field"},
		{"too large", multipartRequest(t, "/uploads/image", filePart{"file", "a.png", "image/png", make([]byte, 8<<20+1)}), true, 413, "File too large"},
		{"corrupt image", multipartRequest(t, "/uploads/image", filePart{"file", "a.png", "image/png", []byte("not really a png")}), true, 400, "Could not process image file"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertNestError(t, e.do(tc.req, tc.authed), tc.status, tc.message)
		})
	}
	if entries, _ := os.ReadDir(e.dir); len(entries) != 0 {
		t.Errorf("failed uploads left %d file(s) behind", len(entries))
	}
}

func TestFileUpload(t *testing.T) {
	e := setup(t, config.Config{})
	gpx := []byte(`<?xml version="1.0"?><gpx></gpx>`)
	// .gpx is accepted by extension even with a generic content type.
	rec := e.do(multipartRequest(t, "/uploads/file", filePart{"file", "Маршрут Ала-Куль.GPX", "application/octet-stream", gpx}), true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d %s", rec.Code, rec.Body.String())
	}
	var res struct {
		URL  string
		Name string
		Size int64
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if !regexp.MustCompile(`^/uploads/[0-9a-f-]{36}\.gpx$`).MatchString(res.URL) || res.Name != "Маршрут Ала-Куль.GPX" || res.Size != int64(len(gpx)) {
		t.Fatalf("response = %+v", res)
	}
	stored, err := os.ReadFile(filepath.Join(e.dir, strings.TrimPrefix(res.URL, "/uploads/")))
	if err != nil || !bytes.Equal(stored, gpx) {
		t.Fatalf("stored file mismatch: %v", err)
	}

	assertNestError(t, e.do(multipartRequest(t, "/uploads/file", filePart{"file", "x.exe", "application/x-msdownload", []byte("MZ")}), true), 400, "File type is not allowed")
	assertNestError(t, e.do(multipartRequest(t, "/uploads/file", filePart{"file", "big.pdf", "application/pdf", make([]byte, 20<<20+1)}), true), 413, "File too large")

	if entries, _ := os.ReadDir(e.dir); len(entries) != 1 {
		t.Errorf("uploads dir has %d files, want only the successful one", len(entries))
	}
}

func TestStaticFileNotFoundAndTraversal(t *testing.T) {
	e := setup(t, config.Config{})
	_ = os.WriteFile(filepath.Join(e.dir, ".secret"), []byte("x"), 0o600)
	for _, path := range []string{"/uploads/missing.webp", "/uploads/.secret", "/uploads/image", "/uploads/..%2fetc%2fpasswd"} {
		rec := e.do(httptest.NewRequest(http.MethodGet, path, nil), false)
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"Cannot GET `) {
			t.Errorf("GET %s = %d %s", path, rec.Code, rec.Body.String())
		}
	}
	assertNestError(t, e.do(httptest.NewRequest(http.MethodPut, "/uploads/x.webp", nil), true), 404, "Cannot PUT /uploads/x.webp")
}

func TestStorageEndpoints(t *testing.T) {
	e := setup(t, config.Config{
		R2AccountID: "acc", R2AccessKeyID: "k", R2SecretAccessKey: "s",
		R2BucketName: "b", R2PublicURL: "https://cdn.example.com",
	})
	jsonReq := func(method, path, body string) *http.Request {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		return req
	}

	rec := e.do(jsonReq(http.MethodPost, "/storage/upload-url", `{"filename":"a.PNG","contentType":"image/png","folder":"gallery"}`), true)
	var up struct{ UploadURL, Key, PublicURL string }
	_ = json.Unmarshal(rec.Body.Bytes(), &up)
	if rec.Code != http.StatusCreated || !strings.HasPrefix(up.Key, "gallery/") || !strings.HasSuffix(up.Key, ".png") || up.PublicURL != "https://cdn.example.com/"+up.Key {
		t.Fatalf("upload-url = %d %s", rec.Code, rec.Body.String())
	}

	rec = e.do(jsonReq(http.MethodPost, "/storage/upload-url", `{"filename":"a.gif","contentType":"image/gif"}`), true)
	want := `{"formErrors":[],"fieldErrors":{"contentType":["Invalid enum value. Expected 'image/jpeg' | 'image/png' | 'image/webp', received 'image/gif'"]}}`
	if rec.Code != http.StatusBadRequest || rec.Body.String() != want {
		t.Fatalf("validation = %d %s", rec.Code, rec.Body.String())
	}

	assertNestError(t, e.do(jsonReq(http.MethodPost, "/storage/upload-url", `{}`), false), 401, "Not authenticated")
	assertNestError(t, e.do(jsonReq(http.MethodGet, "/storage/upload-url", ``), true), 404, "Cannot GET /storage/upload-url")
}

func TestStorageNotConfigured(t *testing.T) {
	e := setup(t, config.Config{})
	req := httptest.NewRequest(http.MethodDelete, "/storage/file", strings.NewReader(`{"key":"gallery/x.webp"}`))
	req.Header.Set("Content-Type", "application/json")
	assertNestError(t, e.do(req, true), 500, "Storage service is not configured")
}
