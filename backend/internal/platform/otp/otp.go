// Package otp issues and verifies the one-time codes of phone auth (R-01).
// Codes are six random digits, stored only as HMAC-SHA256 hashes in Redis
// with a short TTL and consumed on the first verification attempt — they
// never appear in clear text in Redis, logs or errors (docs/seguridad/
// seguridad-privacidad.md). Delivery is delegated to a Sender so the local
// sandbox and Twilio (issue #6) share the exact same code path.
package otp

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// CodeLength is the number of digits of every OTP.
const CodeLength = 6

// Sender delivers a freshly generated code to a phone. Implementations must
// never log the code.
type Sender interface {
	Send(ctx context.Context, phone, code string) error
}

// Generate returns a cryptographically random six-digit code as a string
// (leading zeros preserved).
func Generate() (string, error) {
	const upper = 1000000 // 000000..999999
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("otp: generate: %w", err)
	}
	n := (int(buf[0])<<24 | int(buf[1])<<16 | int(buf[2])<<8 | int(buf[3])) % upper
	return fmt.Sprintf("%0*d", CodeLength, n), nil
}

// Hash derives the stored form of a code: HMAC-SHA256 keyed by the process
// secret. Same code + same secret → same hash; the Redis value is useless
// without the secret.
func Hash(secret, phone, code string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(phone))
	_, _ = mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify compares a candidate code against a stored hash in constant time.
func Verify(secret, phone, code, storedHash string) bool {
	return hmac.Equal([]byte(Hash(secret, phone, code)), []byte(storedHash))
}
