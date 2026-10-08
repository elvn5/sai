// Package routes wires the gateway's middleware and routing table.
package routes

import (
	"fmt"
	"log/slog"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/gateway/internal/config"
	"github.com/elvn5/sai/services/gateway/internal/handlers"
	"github.com/elvn5/sai/services/gateway/internal/proxy"
	"github.com/elvn5/sai/services/pkg/middleware"
)

// Setup builds the gateway router.
//
// Endpoints move to Go services one domain at a time: each migrated domain
// gets its path prefixes registered here, pointing at its service. Anything
// not registered falls through to the legacy NestJS API.
func Setup(cfg config.Config, log *slog.Logger) (*gin.Engine, error) {
	legacyURL, err := url.Parse(cfg.LegacyAPIURL)
	if err != nil {
		return nil, fmt.Errorf("parse LEGACY_API_URL: %w", err)
	}

	r := gin.New()
	// Paths are forwarded verbatim; Gin must not redirect or rewrite them.
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, fmt.Errorf("set trusted proxies: %w", err)
	}

	r.Use(
		middleware.Recovery(log),
		middleware.RequestLogger(log),
		middleware.CORS(cfg.AllowedOrigins()),
	)

	r.GET("/healthz", handlers.Health)

	r.NoRoute(proxy.New(legacyURL, cfg.UpstreamTimeout, log))

	return r, nil
}
