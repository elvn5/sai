// Package config holds the content service's settings.
package config

import "github.com/elvn5/sai/services/pkg/database"

type Config struct {
	Port string `env:"PORT" envDefault:"8082"`

	// Shared with the NestJS API so existing session cookies stay valid.
	JWTSecret string `env:"JWT_SECRET,required"`

	// PostgreSQL server; the service owns its own database on it.
	PostgresHost     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	PostgresPort     string `env:"POSTGRES_PORT" envDefault:"5432"`
	PostgresUser     string `env:"POSTGRES_USER,required"`
	PostgresPassword string `env:"POSTGRES_PASSWORD,required"`
	Database         string `env:"CONTENT_DB_NAME" envDefault:"content"`

	// The NestJS API's database. When set, blog posts, guides, gallery items
	// and page SEO are copied from it once, on the service's first start.
	LegacyDatabase string `env:"LEGACY_DB_NAME"`

	// media service's internal gRPC address, for deleting gallery photos
	// from R2. Without it, deleted photos stay in the bucket.
	MediaGRPCAddr string `env:"MEDIA_GRPC_ADDR"`

	// users service's gRPC address, to confirm a session's account still
	// exists. Without it only the session token is checked.
	UsersGRPCAddr string `env:"USERS_GRPC_ADDR"`
}

func (c Config) DSN() string {
	return database.DSN(c.PostgresHost, c.PostgresPort, c.PostgresUser, c.PostgresPassword, c.Database)
}

func (c Config) LegacyDSN() string {
	return database.DSN(c.PostgresHost, c.PostgresPort, c.PostgresUser, c.PostgresPassword, c.LegacyDatabase)
}
