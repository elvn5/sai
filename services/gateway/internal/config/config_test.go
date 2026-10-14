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
		ExtraOrigins: []string{" http://localhost:3012", "", "http://localhost:3013"},
	}
	want := []string{
		"https://admin.ermekten.com", "https://ermekten.com",
		"http://localhost:3012", "http://localhost:3013",
	}
	if got := cfg.AllowedOrigins(); !slices.Equal(got, want) {
		t.Errorf("origins: %v", got)
	}
}
