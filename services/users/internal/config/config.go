// Package config holds the users service's settings.
package config

import "github.com/elvn5/sai/services/pkg/database"

type Config struct {
	// Internal gRPC API; not exposed publicly.
	GRPCPort string `env:"GRPC_PORT" envDefault:"9083"`

	PostgresHost     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	PostgresPort     string `env:"POSTGRES_PORT" envDefault:"5432"`
	PostgresUser     string `env:"POSTGRES_USER,required"`
	PostgresPassword string `env:"POSTGRES_PASSWORD,required"`
	// Until stage 4 moves accounts into the users service's own database,
	// users are read from the NestJS database (the bot still writes there).
	Database string `env:"USERS_DB_NAME,required"`
}

func (c Config) DSN() string {
	return database.DSN(c.PostgresHost, c.PostgresPort, c.PostgresUser, c.PostgresPassword, c.Database)
}
