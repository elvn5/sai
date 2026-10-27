// Package auth verifies the session JWTs issued by the NestJS API, so Go
// services accept existing sessions unchanged. Both sessions are HS256
// tokens signed with JWT_SECRET and stored in httpOnly cookies.
package auth

import (
	"errors"
	"fmt"
	"time"

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

var (
	// ErrInvalidToken: bad signature, malformed, or expired.
	ErrInvalidToken = errors.New("invalid or expired session")
	// ErrInvalidSession: a genuine token that isn't a session of the
	// expected kind (wrong role, or no subject on a user token).
	ErrInvalidSession = errors.New("invalid session")
)

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
	userID, _, err := v.UserSession(token)
	return userID, err
}

// UserSession validates a user session token and returns the user's id
// and when the token was issued (zero without an iat claim), so a session
// from before a password reset can be refused.
func (v *Verifier) UserSession(token string) (userID string, issuedAt time.Time, err error) {
	claims, err := v.parse(token, RoleUser)
	if err != nil {
		return "", time.Time{}, err
	}
	if claims.Subject == "" {
		return "", time.Time{}, ErrInvalidSession
	}
	if claims.IssuedAt != nil {
		issuedAt = claims.IssuedAt.Time
	}
	return claims.Subject, issuedAt, nil
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
		return nil, ErrInvalidSession
	}
	return claims, nil
}
