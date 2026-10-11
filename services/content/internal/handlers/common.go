// Package handlers holds the content service's HTTP handlers.
package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/elvn5/sai/services/content/internal/store"
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/middleware"
	"github.com/elvn5/sai/services/pkg/validation"
)

const slugTakenMessage = "Slug is already used for this locale"

func init() {
	validation.RegisterPattern("slug", regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`),
		"Slug must be lowercase, alphanumeric, hyphen-separated")
}

// fail answers a store error: not found → 404 with notFoundMessage, slug
// conflict → 409, anything else → NestJS's generic 500.
func fail(c *gin.Context, log *slog.Logger, err error, notFoundMessage string) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		httperr.Abort(c, http.StatusNotFound, notFoundMessage)
	case errors.Is(err, store.ErrSlugTaken):
		httperr.Abort(c, http.StatusConflict, slugTakenMessage)
	default:
		log.ErrorContext(c.Request.Context(), "request failed", slog.String("path", c.Request.URL.Path), slog.Any("error", err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, httperr.InternalServerError())
	}
}

// validID reports whether a path id can exist. NestJS passed malformed ids
// to Postgres and answered 500; here they're simply not found.
func validID(id string) bool {
	return uuid.Validate(id) == nil
}

// optionalEnumQuery reads an optional enum query parameter. A missing
// value means "no filter" (nil); an unknown one matches nothing.
func optionalEnumQuery(c *gin.Context, name string, allowed []string) (value *string, ok bool) {
	raw, present := c.GetQuery(name)
	if !present || raw == "" {
		return nil, true
	}
	if !slices.Contains(allowed, raw) {
		return nil, false
	}
	return &raw, true
}

func isAdmin(c *gin.Context) bool {
	return c.GetBool(middleware.ContextIsAdmin)
}

func currentUserID(c *gin.Context) string {
	return c.GetString(middleware.ContextUserID)
}

// jsParseInt mimics JavaScript's parseInt(raw, 10): leading whitespace and
// sign, then as many digits as there are ("12abc" → 12); ok is false
// where parseInt returns NaN.
func jsParseInt(raw string) (n int, ok bool) {
	s := strings.TrimLeftFunc(raw, unicode.IsSpace)
	sign := 1
	if s != "" && (s[0] == '+' || s[0] == '-') {
		if s[0] == '-' {
			sign = -1
		}
		s = s[1:]
	}
	digits := 0
	for digits < len(s) && s[digits] >= '0' && s[digits] <= '9' {
		if n < 1<<31 {
			n = n*10 + int(s[digits]-'0')
		}
		digits++
	}
	if digits == 0 {
		return 0, false
	}
	return sign * n, true
}

func applyField(dst **string, f validation.Field[string]) {
	if f.Set {
		*dst = f.Ptr()
	}
}
