// Package config holds the bot's settings.
package config

import "github.com/elvn5/sai/services/pkg/database"

type Config struct {
	BotToken string `env:"BOT_TOKEN,required"`
	// Overrides the Bot API server (a local Bot API server, or tests).
	TelegramAPIURL string `env:"TELEGRAM_API_URL"`

	// users service: accounts, Telegram linking, support contacts.
	UsersGRPCAddr string `env:"USERS_GRPC_ADDR,required"`

	// URL of the Ermekten Mini App (the public site). When set, the bot
	// exposes it via the chat menu button and the "Открыть Ermekten"
	// keyboard button — the only ways into it: everything else is done in
	// the chat.
	MiniAppURL string `env:"MINI_APP_URL"`

	// The gateway the bot calls on a user's behalf (outings, tours,
	// favorites, the blog), e.g. http://gateway:8080 on the Docker network,
	// with user sessions it signs with JWT_SECRET. Without both the
	// outings, tours and blog sections are off.
	APIURL    string `env:"API_URL"`
	JWTSecret string `env:"JWT_SECRET"`
	// The gateway's public address, for cover pictures Telegram fetches
	// itself (/uploads/… paths).
	PublicAPIURL string `env:"PUBLIC_API_URL"`

	// Analytics: events go to the Firebase project's Google Analytics 4
	// property through the Measurement Protocol — its web data stream
	// (FIREBASE_MEASUREMENT_ID, "G-…", the site's) with the stream's
	// Measurement Protocol API secret. Off without both. ANALYTICS_SALT
	// keys the hash that stands for a chat id there.
	AnalyticsMeasurementID string `env:"FIREBASE_MEASUREMENT_ID"`
	AnalyticsAPISecret     string `env:"GA_API_SECRET"`
	AnalyticsSalt          string `env:"ANALYTICS_SALT"`

	// The bot's own database: who reads the blog in Telegram and which
	// posts went out.
	PostgresHost     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	PostgresPort     string `env:"POSTGRES_PORT" envDefault:"5432"`
	PostgresUser     string `env:"POSTGRES_USER,required"`
	PostgresPassword string `env:"POSTGRES_PASSWORD,required"`
	Database         string `env:"BOT_DB_NAME" envDefault:"bot"`
}

func (c Config) DSN() string {
	return database.DSN(c.PostgresHost, c.PostgresPort, c.PostgresUser, c.PostgresPassword, c.Database)
}
