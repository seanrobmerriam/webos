package auth

import (
	"errors"
	"sync"
	"time"
)

// Session-related errors
var (
	// ErrSessionNotFound is returned when a session is not found.
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionExpired is returned when a session has expired.
	ErrSessionExpired = errors.New("session expired")
	// ErrSessionInvalid is returned when a session token is invalid.
	ErrSessionInvalid = errors.New("invalid session token")
)

// Session represents an authenticated user session.
type Session struct {
	// Token is the unique session identifier.
	Token string
	// UserID is the identifier of the authenticated user.
	UserID string
	// CreatedAt is the time the session was created.
	CreatedAt time.Time
	// ExpiresAt is the time the session expires.
	ExpiresAt time.Time
	// LastActivity is the time of the last activity.
	LastActivity time.Time
	// IPAddress is the client IP address (optional).
	IPAddress string
	// UserAgent is the client user agent (optional).
	UserAgent string
	// Data is arbitrary session data.
	Data map[string]interface{}
	// Valid indicates if the session is still valid.
	Valid bool
}

// SessionManager manages user sessions.
type SessionManager struct {
	sessions sync.Map
	// DefaultTimeout is the default session timeout duration.
	DefaultTimeout time.Duration
	// CleanupInterval is how often to clean up expired sessions.
	CleanupInterval time.Duration
	// stopCleanup is the channel to stop the cleanup goroutine.
	stopCleanup chan struct{}
	mu          sync.RWMutex
}

// NewSessionManager creates a new SessionManager with default settings.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		DefaultTimeout:  24 * time.Hour,
		CleanupInterval: time.Hour,
		stopCleanup:     make(chan struct{}),
	}
}

// NewSession creates a new session for the specified user.
func (sm *SessionManager) NewSession(userID string) (*Session, error) {
	return sm.NewSessionWithTimeout(userID, sm.DefaultTimeout)
}

// NewSessionWithTimeout creates a new session with a custom timeout.
func (sm *SessionManager) NewSessionWithTimeout(userID string, timeout time.Duration) (*Session, error) {
	token, err := GenerateSessionID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	session := &Session{
		Token:        token,
		UserID:       userID,
		CreatedAt:    now,
		ExpiresAt:    now.Add(timeout),
		LastActivity: now,
		Valid:        true,
		Data:         make(map[string]interface{}),
	}

	sm.sessions.Store(token, session)

	// Start cleanup if not already running
	sm.startCleanupOnce()

	return session, nil
}

// GetSession retrieves a session by its token.
func (sm *SessionManager) GetSession(token string) (*Session, error) {
	session, ok := sm.sessions.Load(token)
	if !ok {
		return nil, ErrSessionNotFound
	}

	s := session.(*Session)
	if !s.Valid {
		return nil, ErrSessionInvalid
	}
	if time.Now().After(s.ExpiresAt) {
		sm.sessions.Delete(token)
		return nil, ErrSessionExpired
	}

	return s, nil
}

// ValidateSession validates a session token and returns the session if valid.
func (sm *SessionManager) ValidateSession(token string) (*Session, error) {
	return sm.GetSession(token)
}

// InvalidateSession removes a session.
func (sm *SessionManager) InvalidateSession(token string) error {
	if _, ok := sm.sessions.Load(token); !ok {
		return ErrSessionNotFound
	}
	sm.sessions.Delete(token)
	return nil
}

// InvalidateAllForUser removes all sessions for a specific user.
func (sm *SessionManager) InvalidateAllForUser(userID string) {
	sm.sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)
		if session.UserID == userID {
			sm.sessions.Delete(key)
		}
		return true
	})
}

// RefreshSession updates the session's last activity time and extends expiration.
func (sm *SessionManager) RefreshSession(token string) error {
	session, err := sm.GetSession(token)
	if err != nil {
		return err
	}

	session.LastActivity = time.Now()
	session.ExpiresAt = time.Now().Add(sm.DefaultTimeout)

	return nil
}

// SetSessionData sets arbitrary data in the session.
func (sm *SessionManager) SetSessionData(token string, key string, value interface{}) error {
	session, err := sm.GetSession(token)
	if err != nil {
		return err
	}
	session.Data[key] = value
	return nil
}

// GetSessionData retrieves data from the session.
func (sm *SessionManager) GetSessionData(token string, key string) (interface{}, bool) {
	session, err := sm.GetSession(token)
	if err != nil {
		return nil, false
	}
	value, ok := session.Data[key]
	return value, ok
}

// SessionCount returns the number of active sessions.
func (sm *SessionManager) SessionCount() int {
	count := 0
	sm.sessions.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// SessionsForUser returns all sessions for a specific user.
func (sm *SessionManager) SessionsForUser(userID string) []*Session {
	sessions := make([]*Session, 0)
	sm.sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)
		if session.UserID == userID {
			sessions = append(sessions, session)
		}
		return true
	})
	return sessions
}

// CleanupExpiredSessions removes all expired sessions.
func (sm *SessionManager) CleanupExpiredSessions() {
	now := time.Now()
	sm.sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)
		if now.After(session.ExpiresAt) || !session.Valid {
			sm.sessions.Delete(key)
		}
		return true
	})
}

// StartCleanup starts the periodic cleanup goroutine.
func (sm *SessionManager) StartCleanup() {
	go sm.cleanupLoop()
}

// StopCleanup stops the cleanup goroutine.
func (sm *SessionManager) StopCleanup() {
	select {
	case sm.stopCleanup <- struct{}{}:
	default:
	}
}

func (sm *SessionManager) cleanupLoop() {
	ticker := time.NewTicker(sm.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.CleanupExpiredSessions()
		case <-sm.stopCleanup:
			return
		}
	}
}

var initOnce sync.Once

// startCleanupOnce starts the cleanup goroutine exactly once.
func (sm *SessionManager) startCleanupOnce() {
	initOnce.Do(func() {
		go sm.cleanupLoop()
	})
}

// SessionInfo returns a safe summary of session information for logging.
func (sm *SessionManager) SessionInfo(token string) map[string]interface{} {
	session, err := sm.GetSession(token)
	if err != nil {
		return map[string]interface{}{
			"valid": false,
		}
	}

	return map[string]interface{}{
		"user_id":       session.UserID,
		"created_at":    session.CreatedAt,
		"expires_at":    session.ExpiresAt,
		"last_activity": session.LastActivity,
		"valid":         session.Valid,
		"data_keys":     len(session.Data),
	}
}
