package auth

import (
	"errors"
	"sync"
	"time"
)

// Auth-related errors
var (
	// ErrInvalidCredentials is returned when username/password is invalid.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUserNotFound is returned when the user is not found.
	ErrUserNotFound = errors.New("user not found")
	// ErrUserAlreadyExists is returned when trying to create a user that exists.
	ErrUserAlreadyExists = errors.New("user already exists")
	// ErrAccountLocked is returned when the account is locked due to too many attempts.
	ErrAccountLocked = errors.New("account is locked")
	// ErrMFARequired is returned when MFA verification is required.
	ErrMFARequired = errors.New("MFA verification required")
	// ErrMFAInvalid is returned when MFA code is invalid.
	ErrMFAInvalid = errors.New("invalid MFA code")
	// ErrSessionRequired is returned when a session is required but not provided.
	ErrSessionRequired = errors.New("authentication required")
)

// MaxLoginAttempts is the maximum number of failed login attempts before lockout.
const MaxLoginAttempts = 5

// DefaultLockoutDuration is the default lockout duration after too many attempts.
const DefaultLockoutDuration = 15 * time.Minute

// User represents a user account in the authentication system.
type User struct {
	// ID is the unique user identifier.
	ID string
	// Username is the login username.
	Username string
	// Email is the user's email address.
	Email string
	// PasswordHash is the hashed password.
	PasswordHash string
	// MFASecret is the TOTP secret if MFA is enabled.
	MFASecret string
	// MFAEnabled indicates if MFA is enabled for this user.
	MFAEnabled bool
	// CreatedAt is when the account was created.
	CreatedAt time.Time
	// LastLogin is the last successful login time.
	LastLogin time.Time
	// LoginAttempts is the number of consecutive failed login attempts.
	LoginAttempts int
	// LockedUntil is when the lockout expires (zero if not locked).
	LockedUntil time.Time
	// Active indicates if the account is active.
	Active bool
}

// Authenticator provides authentication services.
type Authenticator struct {
	users       sync.Map
	sessions    *SessionManager
	maxAttempts int
	lockout     time.Duration
}

// NewAuthenticator creates a new Authenticator.
func NewAuthenticator() *Authenticator {
	return &Authenticator{
		sessions:    NewSessionManager(),
		maxAttempts: MaxLoginAttempts,
		lockout:     DefaultLockoutDuration,
	}
}

// RegisterUser registers a new user. Returns an error if the username already exists.
func (a *Authenticator) RegisterUser(id, username, email, password string) (*User, error) {
	// Check if user already exists
	if _, exists := a.users.Load(username); exists {
		return nil, ErrUserAlreadyExists
	}

	// Hash the password
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:            id,
		Username:      username,
		Email:         email,
		PasswordHash:  hash,
		MFAEnabled:    false,
		CreatedAt:     time.Now(),
		LoginAttempts: 0,
		Active:        true,
	}

	a.users.Store(username, user)
	return user, nil
}

// Authenticate verifies user credentials and returns a session on success.
// If MFA is enabled, returns ErrMFARequired after password verification.
func (a *Authenticator) Authenticate(username, password string) (*Session, error) {
	user, ok := a.users.Load(username)
	if !ok {
		// Use constant-time comparison to prevent username enumeration
		_, _ = HashPassword(password)
		return nil, ErrInvalidCredentials
	}

	u := user.(*User)

	// Check if account is active
	if !u.Active {
		return nil, ErrInvalidCredentials
	}

	// Check if account is locked
	if !u.LockedUntil.IsZero() && time.Now().Before(u.LockedUntil) {
		return nil, ErrAccountLocked
	}

	// Verify password
	if err := CheckPassword(password, u.PasswordHash); err != nil {
		// Increment failed login attempts
		u.LoginAttempts++
		if u.LoginAttempts >= a.maxAttempts {
			u.LockedUntil = time.Now().Add(a.lockout)
		}
		return nil, ErrInvalidCredentials
	}

	// Reset login attempts on success
	u.LoginAttempts = 0
	u.LockedUntil = time.Time{}
	u.LastLogin = time.Now()

	// Check if MFA is required
	if u.MFAEnabled {
		return nil, ErrMFARequired
	}

	// Create session
	return a.sessions.NewSession(u.ID)
}

// VerifyMFA verifies the MFA code for a user after password authentication.
func (a *Authenticator) VerifyMFA(username, code string) (*Session, error) {
	user, ok := a.users.Load(username)
	if !ok {
		return nil, ErrUserNotFound
	}

	u := user.(*User)

	if !u.MFAEnabled {
		return nil, ErrMFAInvalid
	}

	if u.MFASecret == "" {
		return nil, ErrMFAInvalid
	}

	secret := &TOTPSecret{Secret: u.MFASecret}
	if err := ValidateTOTP(secret, code); err != nil {
		return nil, ErrMFAInvalid
	}

	// Create session after successful MFA
	return a.sessions.NewSession(u.ID)
}

// GetUser retrieves a user by username.
func (a *Authenticator) GetUser(username string) (*User, bool) {
	user, ok := a.users.Load(username)
	if !ok {
		return nil, false
	}
	return user.(*User), true
}

// GetUserByID retrieves a user by ID.
func (a *Authenticator) GetUserByID(id string) *User {
	var found *User
	a.users.Range(func(key, value interface{}) bool {
		user := value.(*User)
		if user.ID == id {
			found = user
			return false
		}
		return true
	})
	return found
}

// UpdatePassword updates a user's password.
func (a *Authenticator) UpdatePassword(username, newPassword string) error {
	user, ok := a.users.Load(username)
	if !ok {
		return ErrUserNotFound
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	u := user.(*User)
	u.PasswordHash = hash
	return nil
}

// EnableMFA enables MFA for a user and returns the secret.
func (a *Authenticator) EnableMFA(username, issuer string) (*TOTPSecret, error) {
	user, ok := a.users.Load(username)
	if !ok {
		return nil, ErrUserNotFound
	}

	u := user.(*User)

	secret, err := NewTOTPSecret(issuer, u.Email)
	if err != nil {
		return nil, err
	}

	u.MFASecret = secret.Secret
	u.MFAEnabled = true

	return secret, nil
}

// ConfirmMFA confirms MFA setup with a valid code.
func (a *Authenticator) ConfirmMFA(username, code string) error {
	user, ok := a.users.Load(username)
	if !ok {
		return ErrUserNotFound
	}

	u := user.(*User)

	if !u.MFAEnabled || u.MFASecret == "" {
		return ErrUserNotFound
	}

	secret := &TOTPSecret{Secret: u.MFASecret}
	return ValidateTOTP(secret, code)
}

// DisableMFA disables MFA for a user.
func (a *Authenticator) DisableMFA(username, code string) error {
	user, ok := a.users.Load(username)
	if !ok {
		return ErrUserNotFound
	}

	u := user.(*User)

	if !u.MFAEnabled {
		return nil
	}

	// Verify code before disabling
	secret := &TOTPSecret{Secret: u.MFASecret}
	if err := ValidateTOTP(secret, code); err != nil {
		return err
	}

	u.MFAEnabled = false
	u.MFASecret = ""
	return nil
}

// ValidateSession validates a session token and returns the session.
func (a *Authenticator) ValidateSession(token string) (*Session, error) {
	return a.sessions.GetSession(token)
}

// InvalidateSession removes a session.
func (a *Authenticator) InvalidateSession(token string) error {
	return a.sessions.InvalidateSession(token)
}

// InvalidateAllSessions removes all sessions for a user.
func (a *Authenticator) InvalidateAllSessions(userID string) {
	a.sessions.InvalidateAllForUser(userID)
}

// LockAccount locks a user account.
func (a *Authenticator) LockAccount(username string, duration time.Duration) error {
	user, ok := a.users.Load(username)
	if !ok {
		return ErrUserNotFound
	}

	u := user.(*User)
	u.LockedUntil = time.Now().Add(duration)
	return nil
}

// UnlockAccount unlocks a user account.
func (a *Authenticator) UnlockAccount(username string) error {
	user, ok := a.users.Load(username)
	if !ok {
		return ErrUserNotFound
	}

	u := user.(*User)
	u.LockedUntil = time.Time{}
	u.LoginAttempts = 0
	return nil
}

// IsAccountLocked checks if an account is currently locked.
func (a *Authenticator) IsAccountLocked(username string) bool {
	user, ok := a.users.Load(username)
	if !ok {
		return false
	}

	u := user.(*User)
	return !u.LockedUntil.IsZero() && time.Now().Before(u.LockedUntil)
}

// DeactivateAccount deactivates a user account.
func (a *Authenticator) DeactivateAccount(username string) error {
	user, ok := a.users.Load(username)
	if !ok {
		return ErrUserNotFound
	}

	u := user.(*User)
	u.Active = false
	a.InvalidateAllSessions(u.ID)
	return nil
}

// ReactivateAccount reactivates a user account.
func (a *Authenticator) ReactivateAccount(username string) error {
	user, ok := a.users.Load(username)
	if !ok {
		return ErrUserNotFound
	}

	u := user.(*User)
	u.Active = true
	return nil
}

// UserCount returns the number of registered users.
func (a *Authenticator) UserCount() int {
	count := 0
	a.users.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// SessionManager returns the session manager for external use.
func (a *Authenticator) SessionManager() *SessionManager {
	return a.sessions
}
