package auth

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Signer issues session tokens exactly like the NestJS JwtModule did:
// HS256 with JWT_SECRET, claims {sub?, role, iat, exp}, exp = iat +
// JWT_EXPIRES_IN — for user and admin sessions alike.
type Signer struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewSigner parses expiresIn in the `ms` package format jsonwebtoken
// accepts ("12h", "7d", "30m", "90s", "1.5h"; a bare number means
// milliseconds).
func NewSigner(secret, expiresIn string) (*Signer, error) {
	ttl, err := ParseExpiresIn(expiresIn)
	if err != nil {
		return nil, err
	}
	return &Signer{secret: []byte(secret), ttl: ttl, now: time.Now}, nil
}

func (s *Signer) UserToken(userID string) (string, error) {
	return s.sign(jwt.MapClaims{"sub": userID, "role": RoleUser})
}

func (s *Signer) AdminToken() (string, error) {
	return s.sign(jwt.MapClaims{"role": RoleAdmin})
}

func (s *Signer) sign(claims jwt.MapClaims) (string, error) {
	now := s.now().Unix()
	claims["iat"] = now
	claims["exp"] = now + int64(s.ttl/time.Second)
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

var expiresInPattern = regexp.MustCompile(`(?i)^(-?(?:\d+)?\.?\d+) *(milliseconds?|msecs?|ms|seconds?|secs?|s|minutes?|mins?|m|hours?|hrs?|h|days?|d|weeks?|w|years?|yrs?|y)?$`)

var expiresInUnits = map[byte]time.Duration{
	's': time.Second, 'm': time.Minute, 'h': time.Hour, 'd': 24 * time.Hour,
	'w': 7 * 24 * time.Hour, 'y': time.Duration(365.25 * 24 * float64(time.Hour)),
}

// ParseExpiresIn mirrors the `ms` package's parse.
func ParseExpiresIn(raw string) (time.Duration, error) {
	m := expiresInPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if m == nil {
		return 0, fmt.Errorf("invalid JWT_EXPIRES_IN %q", raw)
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid JWT_EXPIRES_IN %q: %w", raw, err)
	}
	unit := strings.ToLower(m[2])
	scale := time.Millisecond
	if unit != "" && !strings.HasPrefix(unit, "ms") && !strings.HasPrefix(unit, "msec") && !strings.HasPrefix(unit, "milli") {
		scale = expiresInUnits[unit[0]]
	}
	d := time.Duration(math.Round(n * float64(scale)))
	if d <= 0 {
		return 0, fmt.Errorf("JWT_EXPIRES_IN %q must be positive", raw)
	}
	return d, nil
}
