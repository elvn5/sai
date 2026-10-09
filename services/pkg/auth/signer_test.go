package auth_test

import (
	"testing"
	"time"

	"github.com/elvn5/sai/services/pkg/auth"
)

func TestParseExpiresIn(t *testing.T) {
	cases := map[string]time.Duration{
		"12h": 12 * time.Hour, "7d": 7 * 24 * time.Hour, "30m": 30 * time.Minute, "90s": 90 * time.Second,
		"1.5h": 90 * time.Minute, "2 days": 48 * time.Hour, "60000": time.Minute, "500ms": 500 * time.Millisecond,
	}
	for in, want := range cases {
		if got, err := auth.ParseExpiresIn(in); err != nil || got != want {
			t.Errorf("%q = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "soon", "-1h"} {
		if _, err := auth.ParseExpiresIn(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestSignedTokensVerify(t *testing.T) {
	s, err := auth.NewSigner(secret, "12h")
	if err != nil {
		t.Fatal(err)
	}
	v := auth.NewVerifier(secret)
	user, _ := s.UserToken("u-1")
	if id, err := v.UserID(user); err != nil || id != "u-1" {
		t.Errorf("user token: %q %v", id, err)
	}
	admin, _ := s.AdminToken()
	if err := v.Admin(admin); err != nil {
		t.Errorf("admin token: %v", err)
	}
	if err := v.Admin(user); err == nil {
		t.Error("user token accepted as admin")
	}
}
