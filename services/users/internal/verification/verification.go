// Package verification generates and checks the codes that confirm a new
// email or phone registration (apps/api/src/credentials-auth/verification-code.ts).
package verification

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

const (
	CodeTTL        = 10 * time.Minute
	ResendCooldown = time.Minute
	MaxAttempts    = 5
)

// Code is the 6-digit code emailed to the user.
func Code() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// Token is the longer value carried in a t.me deep link for phone
// registrations; it's never typed, so it needn't be short.
func Token() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Hash is what's stored instead of the code itself.
func Hash(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func Matches(code, hash string) bool {
	return subtle.ConstantTimeCompare([]byte(Hash(code)), []byte(hash)) == 1
}
