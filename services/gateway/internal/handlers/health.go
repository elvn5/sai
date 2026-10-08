// Package handlers holds the gateway's own endpoints.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health reports that the gateway process is up. It deliberately doesn't
// check upstreams, so a slow backend can't get the gateway restarted.
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
