package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/httperr"
)

// Content types for what /uploads holds, as Express's `send` reports them
// (Go's built-in table differs, e.g. it serves .xml as text/xml).
var contentTypes = map[string]string{
	".webp": "image/webp",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".pdf":  "application/pdf",
	".gpx":  "application/gpx+xml",
	".kml":  "application/vnd.google-earth.kml+xml",
	".zip":  "application/zip",
	".xml":  "application/xml",
}

// Files serves GET/HEAD /uploads/<name> like NestJS's useStaticAssets
// (express.static): same Cache-Control, weak ETag and Content-Type, Range
// and conditional requests, dotfiles hidden, and "Cannot GET ..." 404s.
type Files struct {
	Dir string
}

func (h *Files) Serve(c *gin.Context) {
	name := strings.TrimPrefix(c.Param("filepath"), "/")
	if name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		httperr.RouteNotFound(c)
		return
	}

	f, err := os.Open(filepath.Join(h.Dir, name)) //nolint:gosec // name is a single, non-dot path segment
	if err != nil {
		httperr.RouteNotFound(c)
		return
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		httperr.RouteNotFound(c)
		return
	}

	header := c.Writer.Header()
	header.Set("Cache-Control", "public, max-age=0")
	// Same formula as the `etag` npm package for file stats.
	header.Set("ETag", fmt.Sprintf(`W/"%x-%x"`, info.Size(), info.ModTime().UnixMilli()))
	contentType, ok := contentTypes[strings.ToLower(path.Ext(name))]
	if !ok {
		contentType = "application/octet-stream"
	}
	header.Set("Content-Type", contentType)

	http.ServeContent(c.Writer, c.Request, name, info.ModTime(), f)
}
