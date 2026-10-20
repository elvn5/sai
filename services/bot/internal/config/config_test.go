package config_test

import (
	"strings"
	"testing"

	"github.com/elvn5/sai/services/bot/internal/config"
)

func TestSite(t *testing.T) {
	c := config.Config{MiniAppURL: "https://app.ermekten.kg"}
	if c.Site() != "https://app.ermekten.kg" {
		t.Errorf("fallback: %q", c.Site())
	}
	c.SiteURL = "https://ermekten.kg"
	if c.Site() != "https://ermekten.kg" {
		t.Errorf("site: %q", c.Site())
	}
}

func TestDSN(t *testing.T) {
	c := config.Config{PostgresHost: "postgres", PostgresPort: "5432", PostgresUser: "u", PostgresPassword: "p'w", Database: "bot"}
	if dsn := c.DSN(); !strings.Contains(dsn, "host='postgres'") || !strings.Contains(dsn, "dbname='bot'") ||
		!strings.Contains(dsn, `password='p\'w'`) {
		t.Errorf("DSN: %s", dsn)
	}
}
