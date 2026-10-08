// Package config holds the listings service's settings.
package config

import "github.com/elvn5/sai/services/pkg/database"

type Config struct {
	Port string `env:"PORT" envDefault:"8083"`

	// Shared with the NestJS API so existing session cookies stay valid.
	JWTSecret string `env:"JWT_SECRET,required"`

	PostgresHost     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	PostgresPort     string `env:"POSTGRES_PORT" envDefault:"5432"`
	PostgresUser     string `env:"POSTGRES_USER,required"`
	PostgresPassword string `env:"POSTGRES_PASSWORD,required"`
	Database         string `env:"LISTINGS_DB_NAME" envDefault:"listings"`

	// The NestJS API's database. When set, hotels, local guides, tours,
	// reviews and listing posts are copied from it once, on first start.
	LegacyDatabase string `env:"LEGACY_DB_NAME"`

	// users service: session checks, partner status, review/post authors.
	UsersGRPCAddr string `env:"USERS_GRPC_ADDR,required"`
}

func (c Config) DSN() string {
	return database.DSN(c.PostgresHost, c.PostgresPort, c.PostgresUser, c.PostgresPassword, c.Database)
}

func (c Config) LegacyDSN() string {
	return database.DSN(c.PostgresHost, c.PostgresPort, c.PostgresUser, c.PostgresPassword, c.LegacyDatabase)
}
