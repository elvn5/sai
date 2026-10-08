package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/httperr"
)

// Recovery turns a panic into NestJS's generic 500 response and logs it.
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
		log.ErrorContext(c.Request.Context(), "panic recovered",
			slog.Any("panic", recovered),
			slog.String("path", c.Request.URL.Path),
		)
		c.AbortWithStatusJSON(http.StatusInternalServerError, httperr.InternalServerError())
	})
}
