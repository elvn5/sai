// Package config holds the users service's settings.
package config

import "github.com/elvn5/sai/services/pkg/database"

type Config struct {
	Port string `env:"PORT" envDefault:"8084"`
	// Internal gRPC API for the other services and the bot; not public.
	GRPCPort string `env:"GRPC_PORT" envDefault:"9083"`

	JWTSecret    string `env:"JWT_SECRET,required"`
	JWTExpiresIn string `env:"JWT_EXPIRES_IN" envDefault:"12h"`
	// Session cookies get the Secure flag (the NestJS API set it when
	// NODE_ENV=production).
	CookieSecure bool `env:"COOKIE_SECURE" envDefault:"false"`

	// Plain-text password for the admin panel.
	AdminPassword string `env:"ADMIN_PASSWORD"`
	// Verifies Telegram Login Widget and Mini App signatures, and sends
	// password reset links to linked Telegram chats.
	BotToken string `env:"BOT_TOKEN"`
	// The public site, for the links in password reset messages.
	SiteURL string `env:"SITE_URL" envDefault:"http://localhost:3002"`

	// Email verification codes, sent through Resend.
	ResendAPIKey string `env:"RESEND_API_KEY"`
	EmailFrom    string `env:"EMAIL_FROM" envDefault:"Ermekten <onboarding@resend.dev>"`
	// Overrides https://api.resend.com (tests).
	ResendBaseURL string `env:"RESEND_BASE_URL" envDefault:"https://api.resend.com"`

	PostgresHost     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	PostgresPort     string `env:"POSTGRES_PORT" envDefault:"5432"`
	PostgresUser     string `env:"POSTGRES_USER,required"`
	PostgresPassword string `env:"POSTGRES_PASSWORD,required"`
	Database         string `env:"USERS_DB_NAME" envDefault:"users"`

	// The NestJS API's database. When set, accounts and support info are
	// copied from it once, on first start.
	LegacyDatabase string `env:"LEGACY_DB_NAME"`
}

func (c Config) DSN() string {
	return database.DSN(c.PostgresHost, c.PostgresPort, c.PostgresUser, c.PostgresPassword, c.Database)
}

func (c Config) LegacyDSN() string {
	return database.DSN(c.PostgresHost, c.PostgresPort, c.PostgresUser, c.PostgresPassword, c.LegacyDatabase)
}
