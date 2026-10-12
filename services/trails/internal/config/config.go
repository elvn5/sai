// Package config holds the trails service's settings.
package config

import "github.com/elvn5/sai/services/pkg/database"

type Config struct {
	Port string `env:"PORT" envDefault:"8085"`

	JWTSecret string `env:"JWT_SECRET,required"`

	PostgresHost     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	PostgresPort     string `env:"POSTGRES_PORT" envDefault:"5432"`
	PostgresUser     string `env:"POSTGRES_USER,required"`
	PostgresPassword string `env:"POSTGRES_PASSWORD,required"`
	Database         string `env:"TRAILS_DB_NAME" envDefault:"trails"`

	// users service: session checks, partner status, public profiles.
	UsersGRPCAddr string `env:"USERS_GRPC_ADDR,required"`

	// Telegram notifications about outings, sent as the site's bot. Off
	// without a token.
	BotToken       string `env:"BOT_TOKEN"`
	TelegramAPIURL string `env:"TELEGRAM_API_URL"`
	// The public site, for links in notifications (no trailing slash).
	SiteURL string `env:"SITE_URL" envDefault:"http://localhost:3011"`
}

func (c Config) DSN() string {
	return database.DSN(c.PostgresHost, c.PostgresPort, c.PostgresUser, c.PostgresPassword, c.Database)
}
