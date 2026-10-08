// Package httperr writes error responses in the same JSON shape NestJS
// produces, so the Nuxt apps keep parsing errors identically while
// endpoints move from the NestJS API to Go services.
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

// New builds the error body for a status code; Error is the standard
// status text, as NestJS sets it.
func New(status int, message string) Body {
	return Body{Message: message, Error: http.StatusText(status), StatusCode: status}
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
