// Package middleware holds Gin middleware shared by all services.
package middleware

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
)

const corsAllowedMethods = "GET,HEAD,PUT,PATCH,POST,DELETE"

// CORS reproduces NestJS's app.enableCors({ origin: [...], credentials: true })
// (the `cors` npm package) exactly, so browsers see the same headers after
// the move to Go:
//   - an allowed Origin is echoed back; any other origin simply gets no
//     Access-Control-Allow-Origin header (the request is not rejected);
//   - Access-Control-Allow-Credentials: true is always sent;
//   - preflight requests are answered here with 204, reflecting the
//     requested headers, and never reach the handlers. (Unlike Node, Go
//     omits Content-Length on 204 responses, as RFC 9110 requires.)
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.Writer.Header()
		origin := c.GetHeader("Origin")
		if origin != "" && slices.Contains(allowedOrigins, origin) {
			header.Set("Access-Control-Allow-Origin", origin)
		}
		header.Add("Vary", "Origin")
		header.Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method != http.MethodOptions {
			c.Next()
			return
		}

		header.Set("Access-Control-Allow-Methods", corsAllowedMethods)
		if requested := c.GetHeader("Access-Control-Request-Headers"); requested != "" {
			header.Set("Access-Control-Allow-Headers", requested)
			header.Set("Vary", "Origin, Access-Control-Request-Headers")
		}
		c.AbortWithStatus(http.StatusNoContent)
	}
}
