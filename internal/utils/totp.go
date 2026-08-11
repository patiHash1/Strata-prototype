package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// ValidateTOTP verifies a 6-digit TOTP code against the given base32-encoded secret.
// It uses a ±1 step window (30-second steps) to account for clock drift.
// The algorithm follows RFC 6238 with SHA-1, 6-digit codes, and 30-second periods.
func ValidateTOTP(code string, secret string) bool {
	// Normalize: strip spaces and dashes that some authenticator apps insert.
	secret = strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(secret, " ", ""), "-", ""))

	// Base32-decode the secret (no padding variant is common for TOTP).
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		// Try with standard padding as a fallback.
		secretBytes, err = base32.StdEncoding.DecodeString(secret)
		if err != nil {
			return false
		}
	}

	now := time.Now().Unix()
	timeStep := int64(30)

	// Check current step and ±1 step for clock-drift tolerance.
	for _, offset := range []int64{0, -1, 1} {
		counter := (now / timeStep) + offset
		if counter < 0 {
			continue
		}
		expected := generateHOTP(secretBytes, uint64(counter))
		if hmac.Equal([]byte(code), []byte(expected)) {
			return true
		}
	}
	return false
}

// generateHOTP generates a 6-digit HOTP code per RFC 4226.
func generateHOTP(secret []byte, counter uint64) string {
	// Convert counter to big-endian 8-byte buffer.
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	// HMAC-SHA1
	mac := hmac.New(sha1.New, secret)
	mac.Write(buf)
	sum := mac.Sum(nil)

	// Dynamic truncation (RFC 4226 §5.4).
	offset := sum[len(sum)-1] & 0x0F
	code := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7FFFFFFF

	return fmt.Sprintf("%06d", code%1_000_000)
}

// GenerateTOTPSecret creates a new base32-encoded TOTP secret (20 bytes / 160 bits).
// The returned string is suitable for provisioning in authenticator apps.
func GenerateTOTPSecret() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate TOTP secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), nil
}
