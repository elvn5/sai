package routes_test

import (
	"bytes"
	"image"
	"image/color"
	"image/color/palette"
	"image/gif"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/gen2brain/webp"

	"github.com/elvn5/sai/services/media/internal/config"
)

func webpWidth(t *testing.T, data []byte) int {
	t.Helper()
	img, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("not WebP: %v", err)
	}
	return img.Bounds().Dx()
}

func get(e env, path string) *httptest.ResponseRecorder {
	return e.do(httptest.NewRequest(http.MethodGet, path, nil), false)
}

func TestImageThumbnailFromUploads(t *testing.T) {
	e := setup(t, config.Config{})
	if err := os.WriteFile(filepath.Join(e.dir, "a.png"), pngBytes(1000, 500), 0o600); err != nil {
		t.Fatal(err)
	}

	rec := get(e, "/images/384/uploads/a.png")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if got := rec.Header().Get("Content-Type"); got != "image/webp" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if w := webpWidth(t, rec.Body.Bytes()); w != 384 {
		t.Fatalf("width = %d, want 384", w)
	}

	// Served from the cache once the original is gone.
	if err := os.Remove(filepath.Join(e.dir, "a.png")); err != nil {
		t.Fatal(err)
	}
	if rec := get(e, "/images/384/uploads/a.png"); rec.Code != http.StatusOK || webpWidth(t, rec.Body.Bytes()) != 384 {
		t.Fatalf("cached: status = %d", rec.Code)
	}
	// The cache lives under .thumbs, which /uploads doesn't serve.
	if rec := get(e, "/uploads/.thumbs"); rec.Code != http.StatusNotFound {
		t.Fatalf("/uploads/.thumbs status = %d", rec.Code)
	}
}

func TestImageThumbnailRejects(t *testing.T) {
	e := setup(t, config.Config{})
	_ = os.WriteFile(filepath.Join(e.dir, "a.png"), pngBytes(10, 10), 0o600)
	_ = os.WriteFile(filepath.Join(e.dir, "doc.pdf"), []byte("%PDF"), 0o600)

	for _, path := range []string{
		"/images/500/uploads/a.png", // not a thumbnail width
		"/images/abc/uploads/a.png", // not a number
		"/images/384/uploads/missing.png",
		"/images/384/uploads/doc.pdf", // not an image
		"/images/384/uploads/.hidden.png",
		"/images/384/uploads/..%2Fa.png",
		"/images/384/elsewhere/a.png",
		"/images/384/r2/gallery/a.png", // R2 not configured
	} {
		if rec := get(e, path); rec.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", path, rec.Code)
		}
	}
}

func TestImageThumbnailFromR2(t *testing.T) {
	var hits atomic.Int32
	bucket := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path != "/gallery/abc-1.png" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(pngBytes(2000, 1000))
	}))
	defer bucket.Close()
	e := setup(t, config.Config{R2PublicURL: bucket.URL + "/"})

	rec := get(e, "/images/640/r2/gallery/abc-1.png")
	if rec.Code != http.StatusOK || webpWidth(t, rec.Body.Bytes()) != 640 {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if rec := get(e, "/images/640/r2/gallery/abc-1.png"); rec.Code != http.StatusOK {
		t.Fatalf("cached: status = %d", rec.Code)
	}
	if n := hits.Load(); n != 1 {
		t.Fatalf("bucket hits = %d, want 1 (second request from the cache)", n)
	}

	if rec := get(e, "/images/640/r2/gallery/other.png"); rec.Code != http.StatusNotFound {
		t.Fatalf("missing object: status = %d", rec.Code)
	}
	if rec := get(e, "/images/640/r2/Gallery/../x.png"); rec.Code != http.StatusNotFound {
		t.Fatalf("bad key: status = %d", rec.Code)
	}
}

func TestImageThumbnailRedirectsAnimated(t *testing.T) {
	e := setup(t, config.Config{})
	g := &gif.GIF{}
	for _, c := range []color.Color{color.White, color.Black} {
		frame := image.NewPaletted(image.Rect(0, 0, 20, 10), palette.Plan9)
		for i := range frame.Pix {
			frame.Pix[i] = uint8(frame.Palette.Index(c))
		}
		g.Image = append(g.Image, frame)
		g.Delay = append(g.Delay, 10)
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(e.dir, "anim.gif"), buf.Bytes(), 0o600)

	for range 2 { // made, then from the cache
		rec := get(e, "/images/256/uploads/anim.gif")
		if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/uploads/anim.gif" {
			t.Fatalf("status = %d, Location = %q", rec.Code, rec.Header().Get("Location"))
		}
	}
}
