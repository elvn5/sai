// Package config holds the gateway's settings.
package config

import "time"

type Config struct {
	Port string `env:"PORT" envDefault:"8080"`

	// The services behind the gateway.
	MediaURL    string `env:"MEDIA_URL,required"`
	ContentURL  string `env:"CONTENT_URL,required"`
	ListingsURL string `env:"LISTINGS_URL,required"`
	UsersURL    string `env:"USERS_URL,required"`
	TrailsURL   string `env:"TRAILS_URL,required"`

	// Browser origins allowed by CORS.
	AdminOrigin  string `env:"ADMIN_ORIGIN" envDefault:"http://localhost:3000"`
	ClientOrigin string `env:"CLIENT_ORIGIN" envDefault:"http://localhost:3002"`

	// How long to wait for an upstream to start responding. Generous because
	// image uploads are converted to WebP before media replies.
	UpstreamTimeout time.Duration `env:"UPSTREAM_TIMEOUT" envDefault:"60s"`

	// Proxies (e.g. nginx) whose X-Forwarded-For is trusted for client IPs
	// in logs. Empty means use the direct peer address.
	TrustedProxies []string `env:"TRUSTED_PROXIES" envSeparator:","`
}

func (c Config) AllowedOrigins() []string {
	return []string{c.AdminOrigin, c.ClientOrigin}
}
