/*
Package auth provides authentication and session management for the webos project.

This package implements secure authentication mechanisms including:

# Password Hashing

The package provides bcrypt-style password hashing using crypto/subtle for
constant-time comparisons. Passwords are hashed with a configurable cost factor
(default 10) which provides a good balance between security and performance.

Example:

	hash, err := auth.HashPassword("user-password")
	if err != nil {
		// handle error
	}
	// Store hash in database

	err = auth.CheckPassword("user-password", storedHash)
	// Returns nil if password matches

# Session Management

Sessions track authenticated users with secure, randomly-generated tokens.
Each session has an expiration time and can be renewed or invalidated.

Example:

	session := auth.NewSession("user-id", 24*time.Hour)
	token := session.Token // 256-bit secure random token

# Token Generation

Secure random token generation using crypto/rand for cryptographic security.
Tokens are suitable for session IDs, API keys, and authentication tokens.

Example:

	token, err := auth.GenerateToken(32) // 32 bytes = 256 bits

# Multi-Factor Authentication (MFA)

TOTP (Time-based One-Time Password) support for two-factor authentication.
Uses HMAC-SHA1 as per RFC 6238.

Example:

	secret, err := auth.NewTOTPSecret()
	// Share secret with user (QR code, manual entry)

	code, err := auth.GenerateTOTP(secret)
	// Verify: auth.ValidateTOTP(secret, code)

# Thread Safety

All authentication operations are thread-safe using sync.Map for session
storage and appropriate locking where needed.
*/
package auth
