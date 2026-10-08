// Package handlers holds the media service's HTTP handlers.
package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/elvn5/sai/services/media/internal/imageconv"
	"github.com/elvn5/sai/services/media/internal/nodepath"
)

const (
	maxImageBytes = 8 << 20
	maxFileBytes  = 20 << 20
)

var allowedImageTypes = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/webp": true, "image/gif": true,
}

// Attachments for guides (maps, GPS tracks, ...); images use /uploads/image.
var allowedFileTypes = map[string]bool{
	"application/pdf":                      true,
	"application/gpx+xml":                  true,
	"application/vnd.google-earth.kml+xml": true,
	"application/zip":                      true,
	"application/xml":                      true,
	"text/xml":                             true,
}

// Uploads stores files on the local uploads volume, served back under
// /uploads/<name>.
type Uploads struct {
	Dir string
	Log *slog.Logger
}

type imageResponse struct {
	URL string `json:"url"`
}

type fileResponse struct {
	URL  string `json:"url"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// Image handles POST /uploads/image: any JPEG/PNG/WebP/GIF up to 8 MB is
// re-encoded to WebP before it's written.
func (h *Uploads) Image(c *gin.Context) {
	var data []byte
	_, _, err := receiveFile(c, maxImageBytes,
		func(part *multipart.Part) *uploadError {
			if !allowedImageTypes[part.Header.Get("Content-Type")] {
				return &uploadError{http.StatusBadRequest, "Only JPEG, PNG, WEBP or GIF images are allowed"}
			}
			return nil
		},
		func(_ *multipart.Part, body io.Reader) (int64, error) {
			var buf bytes.Buffer
			n, err := buf.ReadFrom(body)
			data = buf.Bytes()
			return n, err
		},
	)
	if err != nil {
		abortUpload(c, err)
		return
	}

	webp, err := imageconv.ToWebP(data)
	if err != nil {
		h.Log.WarnContext(c.Request.Context(), "image conversion failed", slog.Any("error", err))
		abortUpload(c, &uploadError{http.StatusBadRequest, "Could not process image file"})
		return
	}

	name := uuid.NewString() + ".webp"
	if err := os.WriteFile(filepath.Join(h.Dir, name), webp, 0o644); err != nil { //nolint:gosec // served publicly by design
		abortUpload(c, fmt.Errorf("write %s: %w", name, err))
		return
	}
	c.JSON(http.StatusCreated, imageResponse{URL: "/uploads/" + name})
}

// File handles POST /uploads/file: guide attachments up to 20 MB, stored
// as-is under a random name that keeps the original extension.
func (h *Uploads) File(c *gin.Context) {
	var name string
	part, size, err := receiveFile(c, maxFileBytes,
		func(part *multipart.Part) *uploadError {
			isGPX := strings.EqualFold(path.Ext(part.FileName()), ".gpx")
			if !allowedFileTypes[part.Header.Get("Content-Type")] && !isGPX {
				return &uploadError{http.StatusBadRequest, "File type is not allowed"}
			}
			return nil
		},
		func(part *multipart.Part, body io.Reader) (int64, error) {
			name = uuid.NewString() + strings.ToLower(nodepath.Extname(part.FileName()))
			return writeFile(filepath.Join(h.Dir, name), body)
		},
	)
	if err != nil {
		abortUpload(c, err)
		return
	}
	c.JSON(http.StatusCreated, fileResponse{URL: "/uploads/" + name, Name: part.FileName(), Size: size})
}

// writeFile streams body to dst, removing the partial file on failure.
func writeFile(dst string, body io.Reader) (int64, error) {
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644) //nolint:gosec // dst is a server-generated name
	if err != nil {
		return 0, fmt.Errorf("create %s: %w", dst, err)
	}
	n, copyErr := io.Copy(f, body)
	closeErr := f.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		_ = os.Remove(dst)
		return 0, err
	}
	return n, nil
}
