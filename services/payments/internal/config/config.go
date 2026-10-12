// Package config holds the payments service's settings.
package config

import "github.com/elvn5/sai/services/pkg/database"

type Config struct {
	Port string `env:"PORT" envDefault:"8086"`

	JWTSecret string `env:"JWT_SECRET,required"`

	PostgresHost     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	PostgresPort     string `env:"POSTGRES_PORT" envDefault:"5432"`
	PostgresUser     string `env:"POSTGRES_USER,required"`
	PostgresPassword string `env:"POSTGRES_PASSWORD,required"`
	Database         string `env:"PAYMENTS_DB_NAME" envDefault:"payments"`

	UsersGRPCAddr    string `env:"USERS_GRPC_ADDR,required"`
	ListingsGRPCAddr string `env:"LISTINGS_GRPC_ADDR,required"`

	// "bakai" (Bakai Bank Open Banking) or "fake" (a test page instead of
	// the bank, for development and staging without bank access).
	Provider string `env:"PAYMENTS_PROVIDER" envDefault:"fake"`
	// Bakai Open Banking API (https://openbanking-api.bakai.kg/swagger).
	BakaiAPIURL   string `env:"BAKAI_API_URL" envDefault:"https://openbanking-api.bakai.kg"`
	BakaiLogin    string `env:"BAKAI_LOGIN"`
	BakaiPassword string `env:"BAKAI_PASSWORD"`
	// The qrType of payment links for GetStateCustomQr (from the bank).
	BakaiQRType string `env:"BAKAI_QR_TYPE"`

	// The platform's commission on bookings, in percent.
	FeePercent float64 `env:"PLATFORM_FEE_PERCENT" envDefault:"0"`
	// How long a payment link stays valid.
	PaymentTTLMinutes int `env:"PAYMENT_TTL_MINUTES" envDefault:"30"`

	// The public site, for the bank's return URL and links in messages.
	SiteURL string `env:"SITE_URL" envDefault:"http://localhost:3011"`
	// Telegram messages about paid bookings; off without a token.
	BotToken string `env:"BOT_TOKEN"`
}

func (c Config) DSN() string {
	return database.DSN(c.PostgresHost, c.PostgresPort, c.PostgresUser, c.PostgresPassword, c.Database)
}
