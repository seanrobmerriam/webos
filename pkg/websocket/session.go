package websocket

import (
	"errors"
	"sync"
	"time"
)

// Session represents a user session associated with a WebSocket connection.
type Session struct {
	// ID is the unique session identifier.
	ID string
	// UserID is the user identifier.
	UserID string
	// Connected is the time when the session was connected.
	Connected time.Time
	// ExpiresAt is the session expiration time.
	ExpiresAt time.Time
	// Data is arbitrary session data.
	Data map[string]interface{}
	// conn is the associated connection.
	conn *Connection
	// mu protects the session.
	mu sync.Mutex
}

// Session errors.
var (
	ErrSessionNotFound  = errors.New("session not found")
	ErrSessionExpired   = errors.New("session expired")
	ErrInvalidSessionID = errors.New("invalid session ID")
)

// SessionConfig holds session configuration.
type SessionConfig struct {
	// Duration is the session duration.
	Duration time.Duration
	// ExtendOnActivity extends the session when there's activity.
	ExtendOnActivity bool
	// MaxDataSize is the maximum size of session data.
	MaxDataSize int
}

// DefaultSessionConfig returns the default session configuration.
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		Duration:         24 * time.Hour,
		ExtendOnActivity: true,
		MaxDataSize:      1024 * 1024, // 1MB
	}
}

// NewSession creates a new session.
func NewSession(id, userID string, config *SessionConfig) *Session {
	if config == nil {
		config = DefaultSessionConfig()
	}

	return &Session{
		ID:        id,
		UserID:    userID,
		Connected: time.Now(),
		ExpiresAt: time.Now().Add(config.Duration),
		Data:      make(map[string]interface{}),
	}
}

// Set sets a session value.
func (s *Session) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data[key] = value
}

// Get gets a session value.
func (s *Session) Get(key string) (interface{}, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.Data[key]
	return v, ok
}

// Delete deletes a session value.
func (s *Session) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Data, key)
}

// Clear clears all session data.
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data = make(map[string]interface{})
}

// IsExpired checks if the session is expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Extend extends the session expiration.
func (s *Session) Extend(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ExpiresAt = time.Now().Add(d)
}

// TTL returns the time until the session expires.
func (s *Session) TTL() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Until(s.ExpiresAt)
}

// Connection returns the associated connection.
func (s *Session) Connection() *Connection {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn
}

// SetConnection sets the associated connection.
func (s *Session) SetConnection(conn *Connection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conn = conn
}

// SessionManager manages sessions.
type SessionManager struct {
	// sessions stores active sessions.
	sessions map[string]*Session
	// config is the session configuration.
	config *SessionConfig
	// mu protects the sessions map.
	mu sync.RWMutex
	// onCreate is called when a session is created.
	onCreate func(*Session)
	// onExpire is called when a session expires.
	onExpire func(*Session)
	// onDestroy is called when a session is destroyed.
	onDestroy func(*Session)
	// cleanupInterval is how often to run cleanup.
	cleanupInterval time.Duration
	// stopChan stops the cleanup goroutine.
	stopChan chan struct{}
}

// NewSessionManager creates a new session manager.
func NewSessionManager(config *SessionConfig) *SessionManager {
	if config == nil {
		config = DefaultSessionConfig()
	}

	return &SessionManager{
		sessions:        make(map[string]*Session),
		config:          config,
		cleanupInterval: 5 * time.Minute,
		stopChan:        make(chan struct{}),
	}
}

// OnCreate sets the session created callback.
func (sm *SessionManager) OnCreate(fn func(*Session)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onCreate = fn
}

// OnExpire sets the session expired callback.
func (sm *SessionManager) OnExpire(fn func(*Session)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onExpire = fn
}

// OnDestroy sets the session destroyed callback.
func (sm *SessionManager) OnDestroy(fn func(*Session)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onDestroy = fn
}

// Create creates a new session.
func (sm *SessionManager) Create(id, userID string) (*Session, error) {
	sm.mu.Lock()

	// Check for duplicate
	if _, exists := sm.sessions[id]; exists {
		sm.mu.Unlock()
		return nil, ErrInvalidSessionID
	}

	session := NewSession(id, userID, sm.config)
	sm.sessions[id] = session

	sm.mu.Unlock()

	// Call onCreate callback
	if sm.onCreate != nil {
		sm.onCreate(session)
	}

	return session, nil
}

// Get gets a session by ID.
func (sm *SessionManager) Get(id string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[id]
	if !exists {
		return nil, ErrSessionNotFound
	}

	// Check expiration
	if session.IsExpired() {
		return nil, ErrSessionExpired
	}

	return session, nil
}

// GetOrCreate gets an existing session or creates a new one.
func (sm *SessionManager) GetOrCreate(id, userID string) (*Session, error) {
	sm.mu.Lock()

	session, exists := sm.sessions[id]
	if exists {
		if !session.IsExpired() {
			sm.mu.Unlock()
			if sm.config.ExtendOnActivity {
				session.Extend(sm.config.Duration)
			}
			return session, nil
		}
		// Session expired, remove it
		delete(sm.sessions, id)
	}

	session = NewSession(id, userID, sm.config)
	sm.sessions[id] = session

	sm.mu.Unlock()

	// Call onCreate callback
	if sm.onCreate != nil {
		sm.onCreate(session)
	}

	return session, nil
}

// Destroy destroys a session.
func (sm *SessionManager) Destroy(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[id]
	if !exists {
		return ErrSessionNotFound
	}

	delete(sm.sessions, id)

	// Call onDestroy callback
	if sm.onDestroy != nil {
		sm.onDestroy(session)
	}

	return nil
}

// DestroyAll destroys all sessions.
func (sm *SessionManager) DestroyAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for id, session := range sm.sessions {
		delete(sm.sessions, id)
		if sm.onDestroy != nil {
			sm.onDestroy(session)
		}
	}
}

// Count returns the number of active sessions.
func (sm *SessionManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// All returns all active sessions.
func (sm *SessionManager) All() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		if !session.IsExpired() {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// Cleanup removes expired sessions.
func (sm *SessionManager) Cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for id, session := range sm.sessions {
		if now.After(session.ExpiresAt) {
			delete(sm.sessions, id)
			if sm.onExpire != nil {
				sm.onExpire(session)
			}
		}
	}
}

// StartCleanup starts the periodic cleanup goroutine.
func (sm *SessionManager) StartCleanup() {
	go func() {
		ticker := time.NewTicker(sm.cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				sm.Cleanup()
			case <-sm.stopChan:
				return
			}
		}
	}()
}

// StopCleanup stops the periodic cleanup goroutine.
func (sm *SessionManager) StopCleanup() {
	close(sm.stopChan)
}
