// Package config holds the gateway's settings.
package config

import "time"

type Config struct {
	Port string `env:"PORT" envDefault:"8080"`

	// The NestJS API that still serves every endpoint not yet moved to a
	// Go service.
	LegacyAPIURL string `env:"LEGACY_API_URL,required"`

	// Go services. Each is optional: while unset, its endpoints keep going
	// to the legacy API, so unsetting one is the rollback for its stage.
	MediaURL   string `env:"MEDIA_URL"`
	ContentURL string `env:"CONTENT_URL"`

	// Browser origins allowed by CORS — the same variables the NestJS API
	// reads, with the same defaults.
	AdminOrigin  string `env:"ADMIN_ORIGIN" envDefault:"http://localhost:3000"`
	ClientOrigin string `env:"CLIENT_ORIGIN" envDefault:"http://localhost:3002"`

	// How long to wait for an upstream to start responding. Generous because
	// image uploads are converted to WebP before the API replies.
	UpstreamTimeout time.Duration `env:"UPSTREAM_TIMEOUT" envDefault:"60s"`

	// Proxies (e.g. nginx) whose X-Forwarded-For is trusted for client IPs
	// in logs. Empty means use the direct peer address.
	TrustedProxies []string `env:"TRUSTED_PROXIES" envSeparator:","`
}

func (c Config) AllowedOrigins() []string {
	return []string{c.AdminOrigin, c.ClientOrigin}
}
