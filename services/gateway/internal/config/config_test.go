package config_test

import (
	"slices"
	"testing"

	"github.com/elvn5/sai/services/gateway/internal/config"
)

func TestAllowedOrigins(t *testing.T) {
	cfg := config.Config{
		AdminOrigin:  "https://admin.ermekten.com",
		ClientOrigin: "https://ermekten.com",
		ExtraOrigins: []string{" https://next.test.ermekten.com", "", "https://next.admin.test.ermekten.com"},
	}
	want := []string{
		"https://admin.ermekten.com", "https://ermekten.com",
		"https://next.test.ermekten.com", "https://next.admin.test.ermekten.com",
	}
	if got := cfg.AllowedOrigins(); !slices.Equal(got, want) {
		t.Errorf("origins: %v", got)
	}
}
