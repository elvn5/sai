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
	"github.com/elvn5/sai/services/pkg/httperr"
	"github.com/elvn5/sai/services/pkg/middleware"
)

// Setup builds the gateway router: each service's path prefixes are
// proxied to it; anything else is a NestJS-style 404.
func Setup(cfg config.Config, log *slog.Logger) (*gin.Engine, error) {
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

	services := []struct {
		env      string
		url      string
		prefixes []string
	}{
		{"MEDIA_URL", cfg.MediaURL, []string{"/uploads", "/storage", "/images"}},
		{"CONTENT_URL", cfg.ContentURL, []string{"/blog", "/guides", "/gallery", "/page-seo"}},
		{"LISTINGS_URL", cfg.ListingsURL, []string{"/hotels", "/local-guides", "/tours", "/reviews", "/listing-posts", "/listing-stats", "/favorites", "/tour-alerts", "/crm", "/chats"}},
		{"USERS_URL", cfg.UsersURL, []string{"/auth", "/users", "/partners", "/support", "/achievements", "/hall-of-fame"}},
		{"TRAILS_URL", cfg.TrailsURL, []string{"/trails", "/conditions"}},
		{"PAYMENTS_URL", cfg.PaymentsURL, []string{"/orders", "/payouts", "/payments"}},
	}
	for _, svc := range services {
		target, err := url.Parse(svc.url)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", svc.env, err)
		}
		handler := proxy.New(target, cfg.UpstreamTimeout, log)
		for _, prefix := range svc.prefixes {
			r.Any(prefix, handler)
			r.Any(prefix+"/*path", handler)
		}
	}

	r.NoRoute(httperr.RouteNotFound)

	return r, nil
}
