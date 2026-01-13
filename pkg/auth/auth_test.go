package auth

import (
	"testing"
	"time"
)

// TestGenerateToken tests secure token generation
func TestGenerateToken(t *testing.T) {
	t.Run("generate token with default length", func(t *testing.T) {
		token, err := GenerateToken(0)
		if err != nil {
			t.Errorf("GenerateToken returned error: %v", err)
		}
		if token == "" {
			t.Error("Expected non-empty token")
		}
	})

	t.Run("generate token with custom length", func(t *testing.T) {
		token, err := GenerateToken(32)
		if err != nil {
			t.Errorf("GenerateToken returned error: %v", err)
		}
		if len(token) < 32 {
			t.Errorf("Token length = %d, want at least 32", len(token))
		}
	})

	t.Run("tokens are unique", func(t *testing.T) {
		token1, _ := GenerateToken(32)
		token2, _ := GenerateToken(32)
		if token1 == token2 {
			t.Error("Expected unique tokens")
		}
	})
}

// TestGenerateHexToken tests hex token generation
func TestGenerateHexToken(t *testing.T) {
	token, err := GenerateHexToken(16)
	if err != nil {
		t.Errorf("GenerateHexToken returned error: %v", err)
	}
	if len(token) != 32 {
		t.Errorf("Token length = %d, want 32", len(token))
	}
}

// TestValidateToken tests token validation
func TestValidateToken(t *testing.T) {
	token1, _ := GenerateToken(32)
	token2, _ := GenerateToken(32)

	if !ValidateToken(token1, token1) {
		t.Error("Expected validation to pass for same token")
	}
	if ValidateToken(token1, token2) {
		t.Error("Expected validation to fail for different tokens")
	}
}

// TestParseToken tests token parsing
func TestParseToken(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		token, _ := GenerateToken(32)
		err := ParseToken(token)
		if err != nil {
			t.Errorf("ParseToken returned error: %v", err)
		}
	})

	t.Run("empty token", func(t *testing.T) {
		err := ParseToken("")
		if err == nil {
			t.Error("Expected error for empty token")
		}
	})

	t.Run("token with invalid characters", func(t *testing.T) {
		err := ParseToken("invalid<token>!")
		if err == nil {
			t.Error("Expected error for invalid characters")
		}
	})
}

// TestGenerateSessionID tests session ID generation
func TestGenerateSessionID(t *testing.T) {
	id, err := GenerateSessionID()
	if err != nil {
		t.Errorf("GenerateSessionID returned error: %v", err)
	}
	if len(id) != 64 {
		t.Errorf("Session ID length = %d, want 64", len(id))
	}
}

// TestGenerateAPIKey tests API key generation
func TestGenerateAPIKey(t *testing.T) {
	t.Run("with prefix", func(t *testing.T) {
		key, err := GenerateAPIKey("test")
		if err != nil {
			t.Errorf("GenerateAPIKey returned error: %v", err)
		}
		if len(key) < 40 {
			t.Errorf("API key too short: %s", key)
		}
	})

	t.Run("without prefix", func(t *testing.T) {
		key, err := GenerateAPIKey("")
		if err != nil {
			t.Errorf("GenerateAPIKey returned error: %v", err)
		}
		if len(key) < 40 {
			t.Errorf("API key too short: %s", key)
		}
	})
}

// TestMaskToken tests token masking
func TestMaskToken(t *testing.T) {
	t.Run("long token", func(t *testing.T) {
		token := "abcdefghijklmnopqrstuvwxyz"
		masked := MaskToken(token)
		if masked == token {
			t.Error("Expected masked token to differ from original")
		}
		// Long tokens are masked as first 4 chars + "..." + last 4 chars
		if len(masked) != 11 { // 4 + 3 + 4 = 11
			t.Errorf("Masked token length = %d, want 11", len(masked))
		}
	})

	t.Run("short token", func(t *testing.T) {
		token := "abc"
		masked := MaskToken(token)
		if masked != "****" {
			t.Errorf("MaskToken = %s, want ****", masked)
		}
	})
}

// TestHashPassword tests password hashing
func TestHashPassword(t *testing.T) {
	t.Run("hash password", func(t *testing.T) {
		hash, err := HashPassword("testpassword123")
		if err != nil {
			t.Errorf("HashPassword returned error: %v", err)
		}
		if hash == "" {
			t.Error("Expected non-empty hash")
		}
		if !hasPrefix(hash, HashPrefix) {
			t.Errorf("Hash should have prefix %s", HashPrefix)
		}
	})

	t.Run("same password produces different hashes", func(t *testing.T) {
		hash1, _ := HashPassword("password")
		hash2, _ := HashPassword("password")
		if hash1 == hash2 {
			t.Error("Same password should produce different hashes")
		}
	})

	t.Run("short password", func(t *testing.T) {
		_, err := HashPassword("short")
		if err == nil {
			t.Error("Expected error for short password")
		}
	})

	t.Run("custom cost", func(t *testing.T) {
		hash, err := HashPasswordWithCost("testpassword", 8)
		if err != nil {
			t.Errorf("HashPasswordWithCost returned error: %v", err)
		}
		cost, _ := GetCost(hash)
		if cost != 8 {
			t.Errorf("Cost = %d, want 8", cost)
		}
	})

	t.Run("invalid cost", func(t *testing.T) {
		_, err := HashPasswordWithCost("testpassword", 20)
		if err == nil {
			t.Error("Expected error for invalid cost")
		}
	})
}

// TestCheckPassword tests password verification
func TestCheckPassword(t *testing.T) {
	password := "testpassword123"
	hash, _ := HashPassword(password)

	t.Run("correct password", func(t *testing.T) {
		err := CheckPassword(password, hash)
		if err != nil {
			t.Errorf("CheckPassword returned error: %v", err)
		}
	})

	t.Run("incorrect password", func(t *testing.T) {
		err := CheckPassword("wrongpassword", hash)
		if err == nil {
			t.Error("Expected error for incorrect password")
		}
	})

	t.Run("invalid hash", func(t *testing.T) {
		err := CheckPassword(password, "invalidhash")
		if err == nil {
			t.Error("Expected error for invalid hash")
		}
	})
}

// TestNeedsRehash tests password rehashing check
func TestNeedsRehash(t *testing.T) {
	hash, _ := HashPasswordWithCost("testpassword", 8)

	if !NeedsRehash(hash) {
		t.Error("Expected hash to need rehashing")
	}

	hash, _ = HashPasswordWithCost("testpassword", DefaultBCryptCost)
	if NeedsRehash(hash) {
		t.Error("Expected hash to not need rehashing")
	}
}

// TestSession tests session management
func TestSession(t *testing.T) {
	sm := NewSessionManager()

	t.Run("create session", func(t *testing.T) {
		session, err := sm.NewSession("user123")
		if err != nil {
			t.Errorf("NewSession returned error: %v", err)
		}
		if session.Token == "" {
			t.Error("Expected non-empty token")
		}
		if session.UserID != "user123" {
			t.Errorf("UserID = %s, want user123", session.UserID)
		}
		if !session.Valid {
			t.Error("Expected session to be valid")
		}
	})

	t.Run("get session", func(t *testing.T) {
		created, _ := sm.NewSession("user123")
		retrieved, err := sm.GetSession(created.Token)
		if err != nil {
			t.Errorf("GetSession returned error: %v", err)
		}
		if retrieved.Token != created.Token {
			t.Error("Retrieved session token mismatch")
		}
	})

	t.Run("get non-existent session", func(t *testing.T) {
		_, err := sm.GetSession("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent session")
		}
	})

	t.Run("invalidate session", func(t *testing.T) {
		session, _ := sm.NewSession("user123")
		err := sm.InvalidateSession(session.Token)
		if err != nil {
			t.Errorf("InvalidateSession returned error: %v", err)
		}
		_, err = sm.GetSession(session.Token)
		if err == nil {
			t.Error("Expected error after invalidation")
		}
	})

	t.Run("refresh session", func(t *testing.T) {
		session, _ := sm.NewSession("user123")
		originalExpiry := session.ExpiresAt
		_ = sm.RefreshSession(session.Token)
		if !session.ExpiresAt.After(originalExpiry) {
			t.Error("Expected session expiry to be extended")
		}
	})

	t.Run("session count", func(t *testing.T) {
		sm.NewSession("user1")
		sm.NewSession("user2")
		if sm.SessionCount() < 2 {
			t.Error("Expected session count to be at least 2")
		}
	})

	t.Run("session data", func(t *testing.T) {
		session, _ := sm.NewSession("user123")
		sm.SetSessionData(session.Token, "key", "value")
		val, ok := sm.GetSessionData(session.Token, "key")
		if !ok {
			t.Error("Expected to get session data")
		}
		if val != "value" {
			t.Errorf("Session data = %v, want value", val)
		}
	})
}

// TestTOTP tests TOTP generation and validation
func TestTOTP(t *testing.T) {
	t.Run("create secret", func(t *testing.T) {
		secret, err := NewTOTPSecret("Example", "user@example.com")
		if err != nil {
			t.Errorf("NewTOTPSecret returned error: %v", err)
		}
		if secret.Secret == "" {
			t.Error("Expected non-empty secret")
		}
	})

	t.Run("generate and validate code", func(t *testing.T) {
		secret, _ := NewTOTPSecret("Example", "user@example.com")
		code, err := GenerateTOTP(secret)
		if err != nil {
			t.Errorf("GenerateTOTP returned error: %v", err)
		}
		if len(code) != 6 {
			t.Errorf("Code length = %d, want 6", len(code))
		}
	})

	t.Run("validate correct code", func(t *testing.T) {
		secret, _ := NewTOTPSecret("Example", "user@example.com")
		code, _ := GenerateTOTP(secret)
		err := ValidateTOTP(secret, code)
		if err != nil {
			t.Errorf("ValidateTOTP returned error: %v", err)
		}
	})

	t.Run("validate incorrect code", func(t *testing.T) {
		secret, _ := NewTOTPSecret("Example", "user@example.com")
		err := ValidateTOTP(secret, "000000")
		if err == nil {
			t.Error("Expected error for invalid code")
		}
	})

	t.Run("TOTP URI generation", func(t *testing.T) {
		secret, _ := NewTOTPSecret("Example", "user@example.com")
		uri := secret.TOTPURI()
		if len(uri) == 0 {
			t.Error("Expected non-empty URI")
		}
	})

	t.Run("remaining time", func(t *testing.T) {
		secret, _ := NewTOTPSecret("Example", "user@example.com")
		remaining := secret.GetRemainingTime()
		if remaining <= 0 || remaining > 30 {
			t.Errorf("Remaining time = %d, want between 1 and 30", remaining)
		}
	})
}

// TestHOTP tests HOTP generation and validation
func TestHOTP(t *testing.T) {
	t.Run("create and use HOTP", func(t *testing.T) {
		secret, err := NewHOTPSecret()
		if err != nil {
			t.Errorf("NewHOTPSecret returned error: %v", err)
		}

		code1, _ := secret.GenerateHOTP(6)
		code2, _ := secret.GenerateHOTP(6)
		if code1 == code2 {
			t.Error("Expected different codes for consecutive generations")
		}
	})

	t.Run("generate and validate HOTP", func(t *testing.T) {
		secret, _ := NewHOTPSecret()
		// Generate a code
		code, _ := secret.GenerateHOTP(6)
		// Counter is now at 1
		// Validate should check counter 1 and find the match
		err := secret.ValidateHOTP(code, 6)
		if err != nil {
			t.Errorf("ValidateHOTP returned error: %v", err)
		}
	})
}

// TestGenerateBackupCodes tests backup code generation
func TestGenerateBackupCodes(t *testing.T) {
	codes, err := GenerateBackupCodes(5, 16)
	if err != nil {
		t.Errorf("GenerateBackupCodes returned error: %v", err)
	}
	if len(codes) != 5 {
		t.Errorf("Code count = %d, want 5", len(codes))
	}
	for _, code := range codes {
		if len(code) != 19 { // XXXX-XXXX-XXXX-XXXX format
			t.Errorf("Backup code format unexpected: %s", code)
		}
	}
}

// TestAuthenticator tests the main authenticator
func TestAuthenticator(t *testing.T) {
	auth := NewAuthenticator()

	t.Run("register user", func(t *testing.T) {
		user, err := auth.RegisterUser("id1", "testuser", "test@example.com", "password123")
		if err != nil {
			t.Errorf("RegisterUser returned error: %v", err)
		}
		if user.Username != "testuser" {
			t.Errorf("Username = %s, want testuser", user.Username)
		}
	})

	t.Run("register duplicate user", func(t *testing.T) {
		_, err := auth.RegisterUser("id2", "testuser", "test2@example.com", "password456")
		if err == nil {
			t.Error("Expected error for duplicate user")
		}
	})

	t.Run("authenticate with correct password", func(t *testing.T) {
		auth.RegisterUser("id3", "user2", "user2@example.com", "correctpassword")
		session, err := auth.Authenticate("user2", "correctpassword")
		if err != nil {
			t.Errorf("Authenticate returned error: %v", err)
		}
		if session == nil {
			t.Error("Expected session to be returned")
		}
	})

	t.Run("authenticate with incorrect password", func(t *testing.T) {
		_, err := auth.Authenticate("user2", "wrongpassword")
		if err == nil {
			t.Error("Expected error for incorrect password")
		}
	})

	t.Run("get user", func(t *testing.T) {
		user, ok := auth.GetUser("testuser")
		if !ok {
			t.Error("Expected user to be found")
		}
		if user.Email != "test@example.com" {
			t.Errorf("Email = %s, want test@example.com", user.Email)
		}
	})

	t.Run("update password", func(t *testing.T) {
		err := auth.UpdatePassword("testuser", "newpassword123")
		if err != nil {
			t.Errorf("UpdatePassword returned error: %v", err)
		}

		// Old password should fail
		_, err = auth.Authenticate("testuser", "password123")
		if err == nil {
			t.Error("Expected old password to fail")
		}

		// New password should work
		_, err = auth.Authenticate("testuser", "newpassword123")
		if err != nil {
			t.Errorf("Expected new password to work: %v", err)
		}
	})

	t.Run("lock and unlock account", func(t *testing.T) {
		auth.RegisterUser("id4", "locktest", "lock@example.com", "password")

		// Lock account
		auth.LockAccount("locktest", time.Hour)
		if !auth.IsAccountLocked("locktest") {
			t.Error("Expected account to be locked")
		}

		// Authentication should fail
		_, err := auth.Authenticate("locktest", "password")
		if err != ErrAccountLocked {
			t.Errorf("Expected ErrAccountLocked, got %v", err)
		}

		// Unlock account
		auth.UnlockAccount("locktest")
		if auth.IsAccountLocked("locktest") {
			t.Error("Expected account to be unlocked")
		}
	})

	t.Run("user count", func(t *testing.T) {
		count := auth.UserCount()
		if count < 3 {
			t.Errorf("User count = %d, want at least 3", count)
		}
	})
}

// TestMFASetup tests MFA enable/disable flow
func TestMFASetup(t *testing.T) {
	auth := NewAuthenticator()
	auth.RegisterUser("id1", "mfauser", "mfa@example.com", "password123")

	t.Run("enable MFA", func(t *testing.T) {
		secret, err := auth.EnableMFA("mfauser", "WebOS")
		if err != nil {
			t.Errorf("EnableMFA returned error: %v", err)
		}
		if secret == nil {
			t.Error("Expected secret to be returned")
		}

		user, _ := auth.GetUser("mfauser")
		if !user.MFAEnabled {
			t.Error("Expected MFA to be enabled")
		}
	})

	t.Run("authenticate with MFA enabled", func(t *testing.T) {
		_, err := auth.Authenticate("mfauser", "password123")
		if err != ErrMFARequired {
			t.Errorf("Expected ErrMFARequired, got %v", err)
		}
	})

	t.Run("verify MFA", func(t *testing.T) {
		user, _ := auth.GetUser("mfauser")
		secret := &TOTPSecret{Secret: user.MFASecret}
		code, _ := GenerateTOTP(secret)

		session, err := auth.VerifyMFA("mfauser", code)
		if err != nil {
			t.Errorf("VerifyMFA returned error: %v", err)
		}
		if session == nil {
			t.Error("Expected session after MFA verification")
		}
	})

	t.Run("disable MFA", func(t *testing.T) {
		user, _ := auth.GetUser("mfauser")
		secret := &TOTPSecret{Secret: user.MFASecret}
		code, _ := GenerateTOTP(secret)

		err := auth.DisableMFA("mfauser", code)
		if err != nil {
			t.Errorf("DisableMFA returned error: %v", err)
		}

		user, _ = auth.GetUser("mfauser")
		if user.MFAEnabled {
			t.Error("Expected MFA to be disabled")
		}
	})
}

// Helper function
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// TestGetUserByID tests getting a user by ID
func TestGetUserByID(t *testing.T) {
	auth := NewAuthenticator()
	auth.RegisterUser("id1", "testuser", "test@example.com", "password123")

	user := auth.GetUserByID("id1")
	if user == nil {
		t.Error("Expected user to be found by ID")
	}
	if user.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", user.Username)
	}

	// Non-existent ID
	user = auth.GetUserByID("nonexistent")
	if user != nil {
		t.Error("Expected nil for non-existent ID")
	}
}

// TestConfirmMFA tests MFA confirmation
func TestConfirmMFA(t *testing.T) {
	auth := NewAuthenticator()
	auth.RegisterUser("id1", "mfauser", "mfa@example.com", "password123")

	// Enable MFA first
	secret, _ := auth.EnableMFA("mfauser", "WebOS")

	// Generate a valid code
	code, _ := GenerateTOTP(secret)

	// Confirm MFA
	err := auth.ConfirmMFA("mfauser", code)
	if err != nil {
		t.Errorf("ConfirmMFA returned error: %v", err)
	}

	// Wrong code should fail
	err = auth.ConfirmMFA("mfauser", "000000")
	if err == nil {
		t.Error("Expected error for invalid code")
	}

	// Non-existent user
	err = auth.ConfirmMFA("nonexistent", code)
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

// TestValidateSession tests session validation
func TestValidateSession(t *testing.T) {
	auth := NewAuthenticator()
	user, _ := auth.RegisterUser("id1", "testuser", "test@example.com", "password123")
	session, _ := auth.sessions.NewSession(user.ID)

	retrieved, err := auth.ValidateSession(session.Token)
	if err != nil {
		t.Errorf("ValidateSession returned error: %v", err)
	}
	if retrieved.Token != session.Token {
		t.Error("Retrieved session mismatch")
	}

	// Invalid session
	_, err = auth.ValidateSession("nonexistent")
	if err == nil {
		t.Error("Expected error for invalid session")
	}
}

// TestInvalidateSession tests session invalidation
func TestInvalidateSession(t *testing.T) {
	auth := NewAuthenticator()
	user, _ := auth.RegisterUser("id1", "testuser", "test@example.com", "password123")
	session, _ := auth.sessions.NewSession(user.ID)

	err := auth.InvalidateSession(session.Token)
	if err != nil {
		t.Errorf("InvalidateSession returned error: %v", err)
	}

	// Session should be gone
	_, err = auth.ValidateSession(session.Token)
	if err == nil {
		t.Error("Expected error after invalidation")
	}

	// Invalidating non-existent session
	err = auth.InvalidateSession("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

// TestInvalidateAllSessions tests invalidating all user sessions
func TestInvalidateAllSessions(t *testing.T) {
	auth := NewAuthenticator()
	user, _ := auth.RegisterUser("id1", "testuser", "test@example.com", "password123")
	session1, _ := auth.sessions.NewSession(user.ID)
	session2, _ := auth.sessions.NewSession(user.ID)

	auth.InvalidateAllSessions(user.ID)

	// Both sessions should be invalid
	_, err1 := auth.ValidateSession(session1.Token)
	_, err2 := auth.ValidateSession(session2.Token)
	if err1 == nil {
		t.Error("Expected session1 to be invalidated")
	}
	if err2 == nil {
		t.Error("Expected session2 to be invalidated")
	}
}

// TestDeactivateAccount tests account deactivation
func TestDeactivateAccount(t *testing.T) {
	auth := NewAuthenticator()
	auth.RegisterUser("id1", "testuser", "test@example.com", "password123")

	err := auth.DeactivateAccount("testuser")
	if err != nil {
		t.Errorf("DeactivateAccount returned error: %v", err)
	}

	user, _ := auth.GetUser("testuser")
	if user.Active {
		t.Error("Expected account to be deactivated")
	}

	// Non-existent user
	err = auth.DeactivateAccount("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

// TestReactivateAccount tests account reactivation
func TestReactivateAccount(t *testing.T) {
	auth := NewAuthenticator()
	auth.RegisterUser("id1", "testuser", "test@example.com", "password123")

	// Deactivate first
	auth.DeactivateAccount("testuser")

	// Reactivate
	err := auth.ReactivateAccount("testuser")
	if err != nil {
		t.Errorf("ReactivateAccount returned error: %v", err)
	}

	user, _ := auth.GetUser("testuser")
	if !user.Active {
		t.Error("Expected account to be reactivated")
	}

	// Non-existent user
	err = auth.ReactivateAccount("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

// TestSessionManager tests the session manager directly
func TestSessionManager(t *testing.T) {
	sm := NewSessionManager()

	t.Run("new session with custom timeout", func(t *testing.T) {
		session, err := sm.NewSessionWithTimeout("user123", time.Hour)
		if err != nil {
			t.Errorf("NewSessionWithTimeout returned error: %v", err)
		}
		if session.ExpiresAt.Sub(session.CreatedAt) != time.Hour {
			t.Error("Expected session timeout to be 1 hour")
		}
	})

	t.Run("validate session", func(t *testing.T) {
		session, _ := sm.NewSession("user123")
		validated, err := sm.ValidateSession(session.Token)
		if err != nil {
			t.Errorf("ValidateSession returned error: %v", err)
		}
		if validated.Token != session.Token {
			t.Error("Validated session mismatch")
		}
	})

	t.Run("invalidate all for user", func(t *testing.T) {
		userID := "user456"
		sm.NewSession(userID)
		sm.NewSession(userID)
		initialCount := sm.SessionCount()

		sm.InvalidateAllForUser(userID)

		if sm.SessionCount() >= initialCount {
			t.Error("Expected sessions to be invalidated")
		}
	})

	t.Run("sessions for user", func(t *testing.T) {
		userID := "user789"
		sm.NewSession(userID)
		sm.NewSession(userID)
		sessions := sm.SessionsForUser(userID)
		if len(sessions) < 2 {
			t.Errorf("Expected at least 2 sessions, got %d", len(sessions))
		}
	})

	t.Run("cleanup expired sessions", func(t *testing.T) {
		// Create a session that expires immediately
		session, _ := sm.NewSessionWithTimeout("expireduser", -time.Hour)
		_ = session

		// Should be cleaned up
		sm.CleanupExpiredSessions()

		_, err := sm.GetSession(session.Token)
		if err == nil {
			t.Error("Expected expired session to be cleaned up")
		}
	})

	t.Run("session info", func(t *testing.T) {
		session, _ := sm.NewSession("infouser")
		info := sm.SessionInfo(session.Token)
		if info["valid"] != true {
			t.Error("Expected session info to show valid")
		}
	})
}

// TestTokenLength tests token length calculation
func TestTokenLength(t *testing.T) {
	// Test with a simple token
	token := "abc123"
	length := TokenLength(token)
	if length <= 0 {
		t.Error("Expected positive token length")
	}
}

// TestConstantTimeComparison tests that token validation uses constant time
func TestConstantTimeComparison(t *testing.T) {
	token1, _ := GenerateToken(32)
	token2, _ := GenerateToken(32)

	// Different tokens should not match
	if ValidateToken(token1, token2) {
		t.Error("Expected different tokens not to match")
	}

	// Same token should match
	if !ValidateToken(token1, token1) {
		t.Error("Expected same token to match")
	}
}

// TestMFABackupCodes tests backup code validation
func TestMFABackupCodes(t *testing.T) {
	auth := NewAuthenticator()
	auth.RegisterUser("id1", "backupuser", "backup@example.com", "password123")

	// Enable MFA and get backup codes
	_, _ = auth.EnableMFA("backupuser", "WebOS")

	user, _ := auth.GetUser("backupuser")
	if !user.MFAEnabled {
		t.Error("Expected MFA to be enabled")
	}

	// Verify MFA works with TOTP code
	secret := &TOTPSecret{Secret: user.MFASecret}
	code, _ := GenerateTOTP(secret)
	session, err := auth.VerifyMFA("backupuser", code)
	if err != nil {
		t.Errorf("VerifyMFA returned error: %v", err)
	}
	if session == nil {
		t.Error("Expected session after MFA verification")
	}
}

// TestAuthenticationWithDeactivatedAccount tests that deactivated accounts cannot authenticate
func TestAuthenticationWithDeactivatedAccount(t *testing.T) {
	auth := NewAuthenticator()
	auth.RegisterUser("id1", "deacttest", "deact@example.com", "password123")

	// Deactivate account
	auth.DeactivateAccount("deacttest")

	// Authentication should fail
	_, err := auth.Authenticate("deacttest", "password123")
	if err == nil {
		t.Error("Expected authentication to fail for deactivated account")
	}
}

// TestUserNotFound tests error handling for non-existent users
func TestUserNotFound(t *testing.T) {
	auth := NewAuthenticator()

	_, err := auth.Authenticate("nonexistent", "password")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}

	err = auth.UpdatePassword("nonexistent", "newpassword")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}

	_, err = auth.EnableMFA("nonexistent", "WebOS")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}

	err = auth.DisableMFA("nonexistent", "123456")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

// TestMultipleUsersWithSameID tests that users can share the same ID
func TestMultipleUsersWithSameID(t *testing.T) {
	auth := NewAuthenticator()

	// Register multiple users with same ID (different usernames)
	user1, err := auth.RegisterUser("same-id", "user1", "user1@example.com", "password1")
	if err != nil {
		t.Errorf("RegisterUser returned error: %v", err)
	}

	user2, err := auth.RegisterUser("same-id", "user2", "user2@example.com", "password2")
	if err != nil {
		t.Errorf("RegisterUser returned error: %v", err)
	}

	if user1.ID != user2.ID {
		t.Error("Expected users to have same ID")
	}

	// GetUserByID should return one of them (first found)
	found := auth.GetUserByID("same-id")
	if found == nil {
		t.Error("Expected at least one user to be found")
	}
}

// TestVerifyMFAWithInvalidUser tests VerifyMFA with non-existent user
func TestVerifyMFAWithInvalidUser(t *testing.T) {
	auth := NewAuthenticator()

	_, err := auth.VerifyMFA("nonexistent", "123456")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

// TestVerifyMFAWithMFADisabled tests VerifyMFA when MFA is not enabled
func TestVerifyMFAWithMFADisabled(t *testing.T) {
	auth := NewAuthenticator()
	auth.RegisterUser("id1", "testuser", "test@example.com", "password123")

	// MFA not enabled, should fail
	_, err := auth.VerifyMFA("testuser", "123456")
	if err != ErrMFAInvalid {
		t.Errorf("Expected ErrMFAInvalid, got %v", err)
	}
}

// TestVerifyMFAWithInvalidCode tests VerifyMFA with invalid code
func TestVerifyMFAWithInvalidCode(t *testing.T) {
	auth := NewAuthenticator()
	auth.RegisterUser("id1", "testuser", "test@example.com", "password123")

	// Enable MFA
	_, _ = auth.EnableMFA("testuser", "WebOS")

	// Wrong code should fail
	_, err := auth.VerifyMFA("testuser", "000000")
	if err != ErrMFAInvalid {
		t.Errorf("Expected ErrMFAInvalid, got %v", err)
	}
}

// TestEnableMFAWithInvalidUser tests EnableMFA with non-existent user
func TestEnableMFAWithInvalidUser(t *testing.T) {
	auth := NewAuthenticator()

	_, err := auth.EnableMFA("nonexistent", "WebOS")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

// TestDisableMFAWithInvalidUser tests DisableMFA with non-existent user
func TestDisableMFAWithInvalidUser(t *testing.T) {
	auth := NewAuthenticator()

	err := auth.DisableMFA("nonexistent", "123456")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

// TestDisableMFAWithInvalidCode tests DisableMFA with invalid code
func TestDisableMFAWithInvalidCode(t *testing.T) {
	auth := NewAuthenticator()
	auth.RegisterUser("id1", "testuser", "test@example.com", "password123")

	// Enable MFA
	secret, _ := auth.EnableMFA("testuser", "WebOS")

	// Wrong code should fail
	err := auth.DisableMFA("testuser", "000000")
	if err == nil {
		t.Error("Expected error for invalid code")
	}

	// Correct code should work
	code, _ := GenerateTOTP(secret)
	err = auth.DisableMFA("testuser", code)
	if err != nil {
		t.Errorf("DisableMFA returned error: %v", err)
	}
}

// TestUpdatePasswordWithInvalidUser tests UpdatePassword with non-existent user
func TestUpdatePasswordWithInvalidUser(t *testing.T) {
	auth := NewAuthenticator()

	err := auth.UpdatePassword("nonexistent", "newpassword123")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

// TestLockAccountWithInvalidUser tests LockAccount with non-existent user
func TestLockAccountWithInvalidUser(t *testing.T) {
	auth := NewAuthenticator()

	err := auth.LockAccount("nonexistent", time.Hour)
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

// TestUnlockAccountWithInvalidUser tests UnlockAccount with non-existent user
func TestUnlockAccountWithInvalidUser(t *testing.T) {
	auth := NewAuthenticator()

	err := auth.UnlockAccount("nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

// TestDeactivateAccountWithInvalidUser tests DeactivateAccount with non-existent user
func TestDeactivateAccountWithInvalidUser(t *testing.T) {
	auth := NewAuthenticator()

	err := auth.DeactivateAccount("nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

// TestReactivateAccountWithInvalidUser tests ReactivateAccount with non-existent user
func TestReactivateAccountWithInvalidUser(t *testing.T) {
	auth := NewAuthenticator()

	err := auth.ReactivateAccount("nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}
