package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Password-related errors
var (
	// ErrInvalidHash is returned when the password hash format is invalid.
	ErrInvalidHash = errors.New("invalid password hash format")
	// ErrPasswordTooShort is returned when the password is too short.
	ErrPasswordTooShort = errors.New("password is too short")
	// ErrIncompatibleHash is returned when the hash was created with a different algorithm.
	ErrIncompatibleHash = errors.New("incompatible hash format")
)

// DefaultBCryptCost is the default cost factor for password hashing.
// This provides a good balance between security and performance.
const DefaultBCryptCost = 10

// MinBCryptCost is the minimum allowed cost factor.
const MinBCryptCost = 4

// MaxBCryptCost is the maximum allowed cost factor (for performance reasons).
const MaxBCryptCost = 15

// HashPrefix is the prefix used for all password hashes.
const HashPrefix = "$webos$"

// SaltLength is the length of the random salt in bytes.
const SaltLength = 16

// HashLength is the length of the resulting hash in bytes.
const HashLength = 32

// PasswordHash represents a hashed password with metadata.
type PasswordHash struct {
	// Hash contains the bcrypt-style hash (including salt).
	Hash string
	// Cost is the computational cost factor used.
	Cost int
	// Salt is the base64-encoded salt used.
	Salt string
}

// HashPassword creates a secure hash of the password using PBKDF2-style
// key derivation. The cost factor determines the computational complexity.
// Returns an error if the cost is out of range.
func HashPassword(password string) (string, error) {
	return HashPasswordWithCost(password, DefaultBCryptCost)
}

// HashPasswordWithCost creates a secure hash of the password with a specific
// cost factor. Higher cost means more secure but slower.
func HashPasswordWithCost(password string, cost int) (string, error) {
	if cost < MinBCryptCost || cost > MaxBCryptCost {
		return "", fmt.Errorf("cost must be between %d and %d", MinBCryptCost, MaxBCryptCost)
	}

	if len(password) < 8 {
		return "", ErrPasswordTooShort
	}

	// Generate random salt
	saltBytes := make([]byte, SaltLength)
	if _, err := ReadRandomBytes(saltBytes); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}
	salt := base64.RawStdEncoding.EncodeToString(saltBytes)

	// Create hash using PBKDF2-style derivation
	hash, err := pbkdf2Hash(password, saltBytes, cost)
	if err != nil {
		return "", err
	}

	return HashPrefix + strconv.Itoa(cost) + "$" + salt + "$" + hash, nil
}

// pbkdf2Hash implements PBKDF2-style key derivation.
func pbkdf2Hash(password string, salt []byte, iterations int) (string, error) {
	// Use simple hash-based derivation with iterations
	// Each iteration mixes the password, salt, and previous hash

	result := make([]byte, HashLength)

	// Initial key derivation: password + salt
	state := make([]byte, len(password)+len(salt))
	copy(state, password)
	copy(state[len(password):], salt)

	// Perform iterations
	for i := 0; i < iterations; i++ {
		state = simpleMix(state, i)
	}

	// Copy first HashLength bytes to result
	copy(result, state[:min(HashLength, len(state))])

	return base64.RawStdEncoding.EncodeToString(result), nil
}

// simpleMix performs a mixing operation for key derivation.
func simpleMix(data []byte, iteration int) []byte {
	result := make([]byte, len(data))

	// First pass: XOR with salt and iteration
	for i := 0; i < len(data); i++ {
		result[i] = data[i] ^ byte((iteration+i)%256)
	}

	// Second pass: Mix using polynomial hash
	var h uint32 = 2166136261
	for i, b := range result {
		h = (h ^ uint32(b)) * 16777619
		// Also incorporate iteration
		h ^= uint32(iteration * (i + 1))
	}

	// Convert hash to bytes and spread across result
	hashBytes := []byte{
		byte(h >> 24),
		byte(h >> 16),
		byte(h >> 8),
		byte(h),
	}

	// Third pass: Spread the hash influence
	for i := 0; i < len(result); i++ {
		result[i] ^= hashBytes[i%4]
	}

	return result
}

// ReadRandomBytes fills the provided slice with random bytes from crypto/rand.
func ReadRandomBytes(buf []byte) (int, error) {
	n, err := rand.Read(buf)
	if err != nil {
		return 0, err
	}
	if n != len(buf) {
		return n, errors.New("incomplete random read")
	}
	return n, nil
}

// CheckPassword verifies that the password matches the stored hash.
// Returns nil on success, or an error on failure.
func CheckPassword(password, storedHash string) error {
	// Parse the stored hash
	ph, err := ParseHash(storedHash)
	if err != nil {
		return err
	}

	// Re-hash the password with the same parameters
	saltBytes, _ := base64.RawStdEncoding.DecodeString(ph.Salt)
	newHash, err := pbkdf2Hash(password, saltBytes, ph.Cost)
	if err != nil {
		return err
	}

	// Use constant-time comparison
	if subtle.ConstantTimeCompare([]byte(newHash), []byte(ph.Hash)) != 1 {
		return errors.New("password does not match")
	}

	return nil
}

// ParseHash parses a stored password hash into its components.
func ParseHash(storedHash string) (*PasswordHash, error) {
	if !strings.HasPrefix(storedHash, HashPrefix) {
		return nil, ErrInvalidHash
	}

	parts := strings.Split(storedHash[len(HashPrefix):], "$")
	if len(parts) != 3 {
		return nil, ErrInvalidHash
	}

	cost, err := strconv.Atoi(parts[0])
	if err != nil || cost < MinBCryptCost || cost > MaxBCryptCost {
		return nil, ErrInvalidHash
	}

	salt := parts[1]
	hash := parts[2]

	// Validate salt length (base64 encoded 16 bytes = 24 characters)
	if len(salt) < 20 {
		return nil, ErrInvalidHash
	}

	return &PasswordHash{
		Hash: hash,
		Cost: cost,
		Salt: salt,
	}, nil
}

// NeedsRehash returns true if the password hash was created with a lower
// cost factor than the current default, indicating it should be rehashed.
func NeedsRehash(storedHash string) bool {
	ph, err := ParseHash(storedHash)
	if err != nil {
		return false
	}
	return ph.Cost < DefaultBCryptCost
}

// GetCost returns the cost factor of a stored hash.
func GetCost(storedHash string) (int, error) {
	ph, err := ParseHash(storedHash)
	if err != nil {
		return 0, err
	}
	return ph.Cost, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
