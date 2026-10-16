package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/media/internal/imageconv"
	"github.com/elvn5/sai/services/pkg/httperr"
)

const (
	// Originals larger than this aren't resized (uploads are capped at 8 MB
	// on the API; direct R2 uploads aren't, so allow some room).
	maxSourceBytes = 25 << 20
	sourceTimeout  = 15 * time.Second
	// Thumbnails never change: a source name is a fresh UUID per upload.
	thumbnailCacheControl = "public, max-age=31536000, immutable"
)

// Keys the R2 presigner creates: "<folder>/<uuid><ext>".
var r2ImageKey = regexp.MustCompile(`^(?:[a-z0-9-]+/)*[a-z0-9-]+\.(?:jpe?g|png|webp|gif)$`)

var imageExts = map[string]bool{".webp": true, ".jpg": true, ".jpeg": true, ".png": true, ".gif": true}

var (
	errSourceNotFound = errors.New("source image not found")
	errSourceTooLarge = errors.New("source image too large")
)

// Images serves resized copies of uploaded images:
//
//	GET /images/<width>/uploads/<name>  — a file on the uploads volume
//	GET /images/<width>/r2/<key>        — an object in the R2 bucket
//
// <width> is one of imageconv.ThumbnailWidths. Thumbnails are WebP, made on
// the first request and kept in CacheDir; animated images redirect to the
// original.
type Images struct {
	UploadsDir string
	CacheDir   string
	// R2_PUBLIC_URL: R2 sources are read from here (no credentials needed).
	R2PublicURL string
	Client      *http.Client
	Log         *slog.Logger
	// Bounds concurrent resizes (CPU and memory); a buffered channel.
	Slots chan struct{}
}

// imageSource is a validated original: a cache key, a reader for its bytes
// and where the original itself lives.
type imageSource struct {
	id       string
	original string
	read     func(ctx context.Context) ([]byte, error)
}

func (h *Images) Serve(c *gin.Context) {
	width, err := strconv.Atoi(c.Param("width"))
	if err != nil || !imageconv.IsThumbnailWidth(width) {
		httperr.RouteNotFound(c)
		return
	}
	src, ok := h.source(strings.TrimPrefix(c.Param("source"), "/"))
	if !ok {
		httperr.RouteNotFound(c)
		return
	}

	sum := sha256.Sum256([]byte(src.id))
	base := filepath.Join(h.CacheDir, strconv.Itoa(width), hex.EncodeToString(sum[:]))
	if h.serveCached(c, base, src.original) {
		return
	}

	ctx := c.Request.Context()
	select {
	case h.Slots <- struct{}{}:
		defer func() { <-h.Slots }()
	case <-ctx.Done():
		return
	}
	// Another request may have made it while this one waited.
	if h.serveCached(c, base, src.original) {
		return
	}

	data, err := src.read(ctx)
	switch {
	case errors.Is(err, errSourceNotFound):
		httperr.RouteNotFound(c)
		return
	case err != nil:
		h.Log.WarnContext(ctx, "failed to read source image", slog.String("source", src.id), slog.Any("error", err))
		httperr.Abort(c, http.StatusBadGateway, "Could not read image")
		return
	}

	thumb, err := imageconv.Thumbnail(data, width)
	switch {
	case errors.Is(err, imageconv.ErrAnimated):
		h.store(ctx, base+".original", nil)
		c.Header("Cache-Control", thumbnailCacheControl)
		c.Redirect(http.StatusFound, src.original)
		return
	case err != nil:
		h.Log.WarnContext(ctx, "failed to make thumbnail", slog.String("source", src.id), slog.Any("error", err))
		httperr.Abort(c, http.StatusUnprocessableEntity, "Could not process image")
		return
	}
	h.store(ctx, base+".webp", thumb)

	c.Header("Cache-Control", thumbnailCacheControl)
	c.Data(http.StatusOK, "image/webp", thumb)
}

// serveCached answers from the cache: the thumbnail, or a redirect for an
// animated original.
func (h *Images) serveCached(c *gin.Context, base, original string) bool {
	if _, err := os.Stat(base + ".original"); err == nil {
		c.Header("Cache-Control", thumbnailCacheControl)
		c.Redirect(http.StatusFound, original)
		return true
	}
	if f, err := os.Open(base + ".webp"); err == nil { //nolint:gosec // base is built from a hash
		defer func() { _ = f.Close() }()
		if info, err := f.Stat(); err == nil {
			header := c.Writer.Header()
			header.Set("Cache-Control", thumbnailCacheControl)
			header.Set("Content-Type", "image/webp")
			http.ServeContent(c.Writer, c.Request, "", info.ModTime(), f)
			return true
		}
	}
	return false
}

// store writes a cache file atomically; a failure only costs a re-render.
func (h *Images) store(ctx context.Context, dst string, data []byte) {
	err := func() error {
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil { //nolint:gosec // public thumbnails
			return err
		}
		tmp, err := os.CreateTemp(filepath.Dir(dst), ".tmp-*")
		if err != nil {
			return err
		}
		_, writeErr := tmp.Write(data)
		closeErr := tmp.Close()
		if err := errors.Join(writeErr, closeErr); err != nil {
			_ = os.Remove(tmp.Name())
			return err
		}
		return os.Rename(tmp.Name(), dst)
	}()
	if err != nil {
		h.Log.WarnContext(ctx, "failed to cache thumbnail", slog.String("path", dst), slog.Any("error", err))
	}
}

func (h *Images) source(raw string) (imageSource, bool) {
	kind, rest, _ := strings.Cut(raw, "/")
	switch kind {
	case "uploads":
		if rest == "" || strings.ContainsAny(rest, `/\`) || strings.HasPrefix(rest, ".") ||
			!imageExts[strings.ToLower(path.Ext(rest))] {
			return imageSource{}, false
		}
		return imageSource{
			id:       "uploads/" + rest,
			original: "/uploads/" + rest,
			read: func(context.Context) ([]byte, error) {
				return readFileLimited(filepath.Join(h.UploadsDir, rest))
			},
		}, true
	case "r2":
		base := strings.TrimSuffix(h.R2PublicURL, "/")
		if base == "" || !r2ImageKey.MatchString(rest) {
			return imageSource{}, false
		}
		original := base + "/" + rest
		return imageSource{
			id:       "r2/" + rest,
			original: original,
			read:     func(ctx context.Context) ([]byte, error) { return h.fetch(ctx, original) },
		}, true
	}
	return imageSource{}, false
}

func readFileLimited(name string) ([]byte, error) {
	f, err := os.Open(name) //nolint:gosec // name is a single, non-dot path segment
	if errors.Is(err, os.ErrNotExist) {
		return nil, errSourceNotFound
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return readLimited(f)
}

func (h *Images) fetch(ctx context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, sourceTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden:
		return nil, errSourceNotFound
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return readLimited(resp.Body)
}

func readLimited(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxSourceBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxSourceBytes {
		return nil, errSourceTooLarge
	}
	return data, nil
}
