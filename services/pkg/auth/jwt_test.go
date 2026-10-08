package auth_test

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/elvn5/sai/services/pkg/auth"
)

const secret = "test-secret-at-least-16-chars"

func sign(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return token
}

// A token exactly as NestJS's JwtService signs a user session.
func TestUserIDAcceptsNestUserToken(t *testing.T) {
	token := sign(t, jwt.SigningMethodHS256, []byte(secret), jwt.MapClaims{
		"sub": "11111111-1111-1111-1111-111111111111", "role": "user",
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
	})
	id, err := auth.NewVerifier(secret).UserID(token)
	if err != nil || id != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("UserID = %q, %v", id, err)
	}
}

func TestAdminAcceptsNestAdminToken(t *testing.T) {
	token := sign(t, jwt.SigningMethodHS256, []byte(secret), jwt.MapClaims{"role": "admin", "iat": time.Now().Unix()})
	if err := auth.NewVerifier(secret).Admin(token); err != nil {
		t.Fatalf("Admin: %v", err)
	}
}

func TestRejectsInvalidTokens(t *testing.T) {
	v := auth.NewVerifier(secret)
	cases := map[string]string{
		"expired": sign(t, jwt.SigningMethodHS256, []byte(secret), jwt.MapClaims{
			"sub": "u", "role": "user", "exp": time.Now().Add(-time.Minute).Unix(),
		}),
		"wrong secret":        sign(t, jwt.SigningMethodHS256, []byte("another-secret-value"), jwt.MapClaims{"sub": "u", "role": "user"}),
		"admin token as user": sign(t, jwt.SigningMethodHS256, []byte(secret), jwt.MapClaims{"role": "admin"}),
		"missing subject":     sign(t, jwt.SigningMethodHS256, []byte(secret), jwt.MapClaims{"role": "user"}),
		"alg none":            sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, jwt.MapClaims{"sub": "u", "role": "user"}),
		"garbage":             "not-a-jwt",
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := v.UserID(token); !errors.Is(err, auth.ErrInvalidToken) {
				t.Fatalf("UserID error = %v, want ErrInvalidToken", err)
			}
		})
	}
	userToken := sign(t, jwt.SigningMethodHS256, []byte(secret), jwt.MapClaims{"sub": "u", "role": "user"})
	if err := v.Admin(userToken); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("Admin accepted a user token: %v", err)
	}
}
