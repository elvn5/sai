// Package respond writes JSON bodies byte-for-byte like Express's
// res.json (JSON.stringify): no HTML escaping of <, > and &, and no
// trailing newline. Gin's c.JSON escapes HTML and c.PureJSON appends a
// newline.
package respond

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

func JSON(c *gin.Context, status int, v any) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		_ = c.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Data(status, "application/json; charset=utf-8", bytes.TrimSuffix(buf.Bytes(), []byte("\n")))
}

// Empty answers like NestJS does when a handler returns null/undefined:
// the status with no body and no Content-Type.
func Empty(c *gin.Context, status int) {
	c.Status(status)
	c.Writer.WriteHeaderNow()
}
