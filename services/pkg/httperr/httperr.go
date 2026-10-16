// Package httperr writes error responses in the same JSON shape NestJS
// produced, which the frontends parse.
package httperr

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Body mirrors NestJS's HttpException response:
// {"message": "...", "error": "Not Found", "statusCode": 404}.
type Body struct {
	Message    string `json:"message"`
	Error      string `json:"error,omitempty"`
	StatusCode int    `json:"statusCode"`
}

// NestJS's names where they differ from Go's http.StatusText.
var nestStatusText = map[int]string{
	http.StatusRequestEntityTooLarge: "Payload Too Large",
}

// New builds the error body for a status code; Error is the status text
// NestJS would use.
func New(status int, message string) Body {
	text, ok := nestStatusText[status]
	if !ok {
		text = http.StatusText(status)
	}
	return Body{Message: message, Error: text, StatusCode: status}
}

// Abort stops the handler chain and writes a NestJS-shaped error.
func Abort(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, New(status, message))
}

// InternalServerError matches NestJS's response for unhandled exceptions,
// which omits the "error" field.
func InternalServerError() Body {
	return Body{Message: "Internal server error", StatusCode: http.StatusInternalServerError}
}

// RouteNotFound answers an unmatched route the way Express/NestJS does:
// {"message":"Cannot GET /path","error":"Not Found","statusCode":404}.
func RouteNotFound(c *gin.Context) {
	Abort(c, http.StatusNotFound, "Cannot "+c.Request.Method+" "+c.Request.URL.Path)
}

// Plain mirrors a NestJS `new HttpException(message, status)` — its body
// has no "error" field and lists statusCode first.
type Plain struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
}

func AbortPlain(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, Plain{StatusCode: status, Message: message})
}
