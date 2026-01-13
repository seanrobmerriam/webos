package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

// Token-related errors
var (
	// ErrInvalidTokenFormat is returned when the token format is invalid.
	ErrInvalidTokenFormat = errors.New("invalid token format")
)

// DefaultTokenLength is the default length for generated tokens in bytes.
const DefaultTokenLength = 32 // 256 bits

// GenerateToken generates a cryptographically secure random token of the
// specified length in bytes. The returned token is base64 URL-encoded without
// padding for safe transmission.
func GenerateToken(length int) (string, error) {
	if length <= 0 {
		length = DefaultTokenLength
	}

	tokenBytes := make([]byte, length)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}

	// Use URL-safe base64 encoding without padding
	return base64.RawURLEncoding.EncodeToString(tokenBytes), nil
}

// GenerateHexToken generates a cryptographically secure random token and
// returns it as a hex string.
func GenerateHexToken(length int) (string, error) {
	if length <= 0 {
		length = DefaultTokenLength
	}

	tokenBytes := make([]byte, length)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(tokenBytes), nil
}

// ValidateToken securely compares two tokens using constant-time comparison
// to prevent timing attacks.
func ValidateToken(token1, token2 string) bool {
	return subtle.ConstantTimeCompare([]byte(token1), []byte(token2)) == 1
}

// ParseToken validates and parses a token string. Returns an error if the
// token format is invalid (empty or contains invalid characters).
func ParseToken(token string) error {
	if len(token) == 0 {
		return ErrInvalidTokenFormat
	}

	// Check for invalid characters in base64 URL encoding
	for _, c := range token {
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '-' || c == '_' ||
			c == '=') {
			return ErrInvalidTokenFormat
		}
	}

	return nil
}

// GenerateSessionID generates a unique session ID.
func GenerateSessionID() (string, error) {
	return GenerateHexToken(32) // 64 hex characters
}

// GenerateAPIKey generates a secure API key with prefix for identification.
func GenerateAPIKey(prefix string) (string, error) {
	if prefix == "" {
		prefix = "webos"
	}
	key, err := GenerateToken(32)
	if err != nil {
		return "", err
	}
	return prefix + "_" + key, nil
}

// MaskToken masks a token for logging purposes, showing only the first
// and last few characters.
func MaskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// TokenLength returns the byte length of a base64 URL-encoded token.
func TokenLength(encodedToken string) int {
	// Remove padding for length calculation
	padded := strings.ReplaceAll(encodedToken, "_", "/")
	padding := strings.Count(padded, "=")
	return (len(padded) * 6 / 8) - padding
}
