// Package config loads a service's settings from the environment.
package config

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Load reads an optional .env file from the working directory into the
// process environment (without overriding variables that are already set)
// and parses the environment into T using `env` struct tags.
func Load[T any]() (T, error) {
	var cfg T
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return cfg, fmt.Errorf("load .env: %w", err)
	}
	if err := env.Parse(&cfg); err != nil {
		return cfg, fmt.Errorf("parse environment: %w", err)
	}
	return cfg, nil
}
