// Package auth verifies the session JWTs issued by the NestJS API, so Go
// services accept existing sessions unchanged. Both sessions are HS256
// tokens signed with JWT_SECRET and stored in httpOnly cookies.
package auth

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// UserCookie holds a site visitor's session: {"sub": userID, "role": "user"}.
	UserCookie = "user_token"
	// AdminCookie holds the admin panel session: {"role": "admin"}.
	AdminCookie = "admin_token"

	RoleUser  = "user"
	RoleAdmin = "admin"
)

var ErrInvalidToken = errors.New("invalid or expired session")

// Claims is the payload NestJS's JwtService signs.
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// Verifier checks session tokens against the shared secret.
type Verifier struct {
	secret []byte
}

func NewVerifier(secret string) *Verifier {
	return &Verifier{secret: []byte(secret)}
}

// UserID validates a user session token and returns the user's id.
func (v *Verifier) UserID(token string) (string, error) {
	claims, err := v.parse(token, RoleUser)
	if err != nil {
		return "", err
	}
	if claims.Subject == "" {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}

// Admin validates an admin session token.
func (v *Verifier) Admin(token string) error {
	_, err := v.parse(token, RoleAdmin)
	return err
}

func (v *Verifier) parse(token, role string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) {
		return v.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	if claims.Role != role {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
