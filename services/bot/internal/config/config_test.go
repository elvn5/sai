package config_test

import (
	"strings"
	"testing"

	"github.com/elvn5/sai/services/bot/internal/config"
)

func TestDSN(t *testing.T) {
	c := config.Config{PostgresHost: "postgres", PostgresPort: "5432", PostgresUser: "u", PostgresPassword: "p'w", Database: "bot"}
	if dsn := c.DSN(); !strings.Contains(dsn, "host='postgres'") || !strings.Contains(dsn, "dbname='bot'") ||
		!strings.Contains(dsn, `password='p\'w'`) {
		t.Errorf("DSN: %s", dsn)
	}
}
