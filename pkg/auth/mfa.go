package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"
)

// MFA-related errors
var (
	// ErrInvalidTOTP is returned when the TOTP code is invalid.
	ErrInvalidTOTP = errors.New("invalid TOTP code")
	// ErrInvalidSecret is returned when the TOTP secret is invalid.
	ErrInvalidSecret = errors.New("invalid TOTP secret")
	// ErrInvalidCodeLength is returned when the code length is invalid.
	ErrInvalidCodeLength = errors.New("invalid code length")
)

// DefaultTOTPCodeLength is the default length for TOTP codes.
const DefaultTOTPCodeLength = 6

// DefaultTOTPPeriod is the default time period for TOTP codes (30 seconds).
const DefaultTOTPPeriod = 30 * time.Second

// MinTOTPCodeLength is the minimum allowed code length.
const MinTOTPCodeLength = 6

// MaxTOTPCodeLength is the maximum allowed code length.
const MaxTOTPCodeLength = 8

// TOTPSecret represents a TOTP secret for two-factor authentication.
type TOTPSecret struct {
	// Secret is the base32-encoded secret key.
	Secret string
	// CreatedAt is when the secret was created.
	CreatedAt time.Time
	// Issuer is the service name (for QR code generation).
	Issuer string
	// Account is the account name (usually email).
	Account string
}

// NewTOTPSecret creates a new TOTP secret for the specified account.
func NewTOTPSecret(issuer, account string) (*TOTPSecret, error) {
	// Generate 20 bytes of random data for the secret
	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	// Use base32 encoding without padding
	secret := base32.StdEncoding.EncodeToString(secretBytes)
	secret = strings.TrimRight(secret, "=")

	return &TOTPSecret{
		Secret:    secret,
		CreatedAt: time.Now(),
		Issuer:    issuer,
		Account:   account,
	}, nil
}

// GenerateTOTP generates a TOTP code for the current time.
func GenerateTOTP(secret *TOTPSecret) (string, error) {
	return GenerateTOTPAt(secret, time.Now())
}

// GenerateTOTPAt generates a TOTP code at a specific time.
func GenerateTOTPAt(secret *TOTPSecret, t time.Time) (string, error) {
	return GenerateTOTPCode(secret.Secret, t, DefaultTOTPCodeLength)
}

// ValidateTOTP validates a TOTP code against the current time.
// Returns nil if valid, or an error if invalid.
func ValidateTOTP(secret *TOTPSecret, code string) error {
	return ValidateTOTPWithWindow(secret, code, 1) // Allow 1 step window
}

// ValidateTOTPWithWindow validates a TOTP code with a time window.
// window is the number of time steps to check before and after the current time.
func ValidateTOTPWithWindow(secret *TOTPSecret, code string, window int) error {
	if len(code) < MinTOTPCodeLength || len(code) > MaxTOTPCodeLength {
		return ErrInvalidCodeLength
	}

	now := time.Now()
	for i := -window; i <= window; i++ {
		offset := time.Duration(i) * DefaultTOTPPeriod
		expected, err := GenerateTOTPAt(secret, now.Add(offset))
		if err != nil {
			return err
		}
		if secureCompare(code, expected) {
			return nil
		}
	}

	return ErrInvalidTOTP
}

// GenerateTOTPCode generates a TOTP code using the given secret and time.
func GenerateTOTPCode(secret string, t time.Time, codeLength int) (string, error) {
	if codeLength < MinTOTPCodeLength || codeLength > MaxTOTPCodeLength {
		return "", ErrInvalidCodeLength
	}

	// Decode the base32 secret
	secretBytes, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		// Try without padding
		secret = strings.TrimRight(secret, "=")
		secretBytes, err = base32.StdEncoding.DecodeString(secret)
		if err != nil {
			return "", ErrInvalidSecret
		}
	}

	// Calculate the counter based on time
	// TOTP uses: counter = floor(unix_timestamp / period)
	counter := t.Unix() / int64(DefaultTOTPPeriod.Seconds())

	// Convert counter to 8-byte big-endian
	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, uint64(counter))

	// Calculate HMAC-SHA1
	mac := hmac.New(sha1.New, secretBytes)
	mac.Write(counterBytes)
	hash := mac.Sum(nil)

	// Dynamic truncation
	offset := hash[len(hash)-1] & 0x0f
	truncatedHash := hash[offset : offset+4]

	// Convert to integer and take modulo
	code := binary.BigEndian.Uint32(truncatedHash) & 0x7fffffff

	// Format to specified length
	mod := uint32(1)
	for i := 0; i < codeLength; i++ {
		mod *= 10
	}

	formatted := fmt.Sprintf("%0*d", codeLength, code%mod)

	return formatted, nil
}

// TOTPURI generates the otpauth:// URI for QR code generation.
func (s *TOTPSecret) TOTPURI() string {
	if s.Issuer == "" {
		return fmt.Sprintf("otpauth://totp/%s?secret=%s",
			s.Account, s.Secret)
	}
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
		s.Issuer, s.Account, s.Secret, s.Issuer)
}

// GetRemainingTime returns the seconds remaining until the code changes.
func (s *TOTPSecret) GetRemainingTime() int {
	now := time.Now().Unix()
	period := int64(DefaultTOTPPeriod.Seconds())
	remaining := int(period - (now % period))
	return remaining
}

// HOTPSecret represents a HOTP secret for counter-based authentication.
type HOTPSecret struct {
	// Secret is the base32-encoded secret key.
	Secret string
	// Counter is the current counter value.
	Counter uint64
	// CreatedAt is when the secret was created.
	CreatedAt time.Time
}

// NewHOTPSecret creates a new HOTP secret.
func NewHOTPSecret() (*HOTPSecret, error) {
	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	secret := base32.StdEncoding.EncodeToString(secretBytes)
	secret = strings.TrimRight(secret, "=")

	return &HOTPSecret{
		Secret:    secret,
		Counter:   0,
		CreatedAt: time.Now(),
	}, nil
}

// GenerateHOTP generates the next HOTP code and increments the counter.
func (s *HOTPSecret) GenerateHOTP(codeLength int) (string, error) {
	code, err := GenerateHOTPCode(s.Secret, s.Counter, codeLength)
	if err != nil {
		return "", err
	}
	s.Counter++
	return code, nil
}

// ValidateHOTP validates an HOTP code and advances the counter if valid.
func (s *HOTPSecret) ValidateHOTP(code string, codeLength int) error {
	if len(code) < MinTOTPCodeLength || len(code) > MaxTOTPCodeLength {
		return ErrInvalidCodeLength
	}

	// Check current and next few counters to handle synchronization
	// Also check previous counter (counter-1) since GenerateHOTP increments after use
	start := int64(s.Counter)
	if start > 0 {
		start-- // Also check previous counter
	}
	for i := int64(0); i <= 10; i++ {
		expected, err := GenerateHOTPCode(s.Secret, uint64(start+i), codeLength)
		if err != nil {
			return err
		}
		if secureCompare(code, expected) {
			// Valid code found, advance counter past this one
			s.Counter = uint64(start + i + 1)
			return nil
		}
	}

	return ErrInvalidTOTP
}

// GenerateHOTPCode generates an HOTP code using the given secret and counter.
func GenerateHOTPCode(secret string, counter uint64, codeLength int) (string, error) {
	if codeLength < MinTOTPCodeLength || codeLength > MaxTOTPCodeLength {
		return "", ErrInvalidCodeLength
	}

	secretBytes, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		secret = strings.TrimRight(secret, "=")
		secretBytes, err = base32.StdEncoding.DecodeString(secret)
		if err != nil {
			return "", ErrInvalidSecret
		}
	}

	// Convert counter to 8-byte big-endian
	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, counter)

	// Calculate HMAC-SHA1
	mac := hmac.New(sha1.New, secretBytes)
	mac.Write(counterBytes)
	hash := mac.Sum(nil)

	// Dynamic truncation
	offset := hash[len(hash)-1] & 0x0f
	truncatedHash := hash[offset : offset+4]

	// Convert to integer and take modulo
	code := binary.BigEndian.Uint32(truncatedHash) & 0x7fffffff

	// Format to specified length
	mod := uint32(1)
	for i := 0; i < codeLength; i++ {
		mod *= 10
	}

	formatted := fmt.Sprintf("%0*d", codeLength, code%mod)

	return formatted, nil
}

// secureCompare performs constant-time comparison to prevent timing attacks.
func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// GenerateBackupCodes generates a set of backup codes for account recovery.
func GenerateBackupCodes(count, length int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code, err := GenerateHexToken(length / 2)
		if err != nil {
			return nil, err
		}
		// Format as XXXX-XXXX-XXXX-XXXX
		var formatted strings.Builder
		for j, r := range code {
			formatted.WriteRune(r)
			if (j+1)%4 == 0 && j < len(code)-1 {
				formatted.WriteRune('-')
			}
		}
		codes[i] = formatted.String()
	}
	return codes, nil
}
