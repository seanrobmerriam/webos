package ipc

import (
	"errors"
	"sync"
)

// Signal represents a signal number.
type Signal int

const (
	// SignalNil is a null signal (no operation).
	SignalNil Signal = 0
	// SignalInterrupt is sent when Ctrl+C is pressed.
	SignalInterrupt Signal = 2
	// SignalKill terminates the process immediately.
	SignalKill Signal = 9
	// SignalTerminate requests graceful termination.
	SignalTerminate Signal = 15
	// SignalChild is sent when a child process terminates.
	SignalChild Signal = 17
	// SignalStop stops the process.
	SignalStop Signal = 19
	// SignalContinue continues a stopped process.
	SignalContinue Signal = 18
)

// Signal errors.
var (
	ErrInvalidSignal = errors.New("invalid signal")
	ErrSignalBlocked = errors.New("signal is blocked")
	ErrNoHandler     = errors.New("no handler for signal")
)

// SignalInfo contains information about a signal.
type SignalInfo struct {
	// Signal is the signal number.
	Signal Signal
	// PID is the sender's PID.
	PID int
	// UID is the sender's user ID.
	UID int
	// Time is when the signal was sent.
	Time int64
}

// SignalManager manages signal handling for processes.
type SignalManager struct {
	// handlers maps signals to handler functions.
	handlers map[Signal][]SignalHandler
	// defaultHandler is called when no specific handler exists.
	defaultHandler SignalHandler
	// pendingSignals holds pending signals.
	pendingSignals map[int][]SignalInfo
	// mu protects the manager.
	mu sync.RWMutex
	// blockedSignals is the global blocked signal set.
	blockedSignals SignalSet
}

// SignalHandler is a function that handles signals.
type SignalHandler func(pid int, sig Signal)

// NewSignalManager creates a new signal manager.
func NewSignalManager() *SignalManager {
	return &SignalManager{
		handlers:       make(map[Signal][]SignalHandler),
		pendingSignals: make(map[int][]SignalInfo),
		blockedSignals: make(SignalSet),
	}
}

// SetDefaultHandler sets the default signal handler.
func (m *SignalManager) SetDefaultHandler(handler SignalHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultHandler = handler
}

// RegisterHandler registers a handler for a specific signal.
func (m *SignalManager) RegisterHandler(sig Signal, handler SignalHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[sig] = append(m.handlers[sig], handler)
}

// UnregisterHandler removes a handler for a signal.
func (m *SignalManager) UnregisterHandler(sig Signal, handler SignalHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	handlers := m.handlers[sig]
	for i, h := range handlers {
		if &h == &handler {
			m.handlers[sig] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// Send sends a signal to a process.
func (m *SignalManager) Send(pid int, sig Signal) error {
	if sig <= SignalNil || sig > SignalContinue {
		return ErrInvalidSignal
	}

	m.mu.RLock()
	blocked := m.blockedSignals[sig]
	m.mu.RUnlock()

	if blocked {
		return ErrSignalBlocked
	}

	// Queue the signal
	m.mu.Lock()
	m.pendingSignals[pid] = append(m.pendingSignals[pid], SignalInfo{
		Signal: sig,
		PID:    pid,
		Time:   0,
	})
	m.mu.Unlock()

	return nil
}

// Deliver delivers pending signals to a process.
func (m *SignalManager) Deliver(pid int) []SignalInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	signals := m.pendingSignals[pid]
	m.pendingSignals[pid] = nil

	return signals
}

// ProcessSignals processes all pending signals for a process.
func (m *SignalManager) ProcessSignals(pid int, handler SignalHandler) {
	signals := m.Deliver(pid)

	for _, info := range signals {
		handlers := m.handlers[info.Signal]
		if len(handlers) > 0 {
			for _, h := range handlers {
				h(pid, info.Signal)
			}
		} else if m.defaultHandler != nil {
			m.defaultHandler(pid, info.Signal)
		}
	}
}

// BlockSignal blocks a signal.
func (m *SignalManager) BlockSignal(sig Signal) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blockedSignals[sig] = true
}

// UnblockSignal unblocks a signal.
func (m *SignalManager) UnblockSignal(sig Signal) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.blockedSignals, sig)
}

// IsBlocked checks if a signal is blocked.
func (m *SignalManager) IsBlocked(sig Signal) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.blockedSignals[sig]
}

// PendingCount returns the number of pending signals for a process.
func (m *SignalManager) PendingCount(pid int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pendingSignals[pid])
}

// SignalAction represents a signal action.
type SignalAction string

const (
	// ActionTerminate terminates the process.
	ActionTerminate SignalAction = "terminate"
	// ActionIgnore ignores the signal.
	ActionIgnore SignalAction = "ignore"
	// ActionStop stops the process.
	ActionStop SignalAction = "stop"
	// ActionContinue continues the process.
	ActionContinue SignalAction = "continue"
	// ActionCustom calls a custom handler.
	ActionCustom SignalAction = "custom"
)

// SignalActionMap defines the default action for each signal.
var SignalActionMap = map[Signal]SignalAction{
	SignalInterrupt: ActionTerminate,
	SignalKill:      ActionTerminate,
	SignalTerminate: ActionTerminate,
	SignalChild:     ActionIgnore,
	SignalStop:      ActionStop,
	SignalContinue:  ActionContinue,
}

// SignalConfig holds signal handling configuration.
type SignalConfig struct {
	// Action is the action to take.
	Action SignalAction
	// Handler is the custom handler (for ActionCustom).
	Handler SignalHandler
	// Blocked is true if the signal should be blocked.
	Blocked bool
}

// SignalTable manages signal configurations for processes.
type SignalTable struct {
	// configs holds signal configurations.
	configs map[int]map[Signal]*SignalConfig
	// mu protects the table.
	mu sync.RWMutex
	// globalConfig is the default configuration.
	globalConfig map[Signal]*SignalConfig
}

// NewSignalTable creates a new signal table.
func NewSignalTable() *SignalTable {
	return &SignalTable{
		configs:      make(map[int]map[Signal]*SignalConfig),
		globalConfig: make(map[Signal]*SignalConfig),
	}
}

// SetConfig sets the configuration for a signal.
func (t *SignalTable) SetConfig(pid int, sig Signal, config *SignalConfig) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.configs[pid] == nil {
		t.configs[pid] = make(map[Signal]*SignalConfig)
	}

	t.configs[pid][sig] = config
}

// GetConfig gets the configuration for a signal.
func (t *SignalTable) GetConfig(pid int, sig Signal) (*SignalConfig, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Check process-specific config
	if configs, exists := t.configs[pid]; exists {
		if config, exists := configs[sig]; exists {
			return config, nil
		}
	}

	// Check global config
	if config, exists := t.globalConfig[sig]; exists {
		return config, nil
	}

	// Return default config
	return &SignalConfig{
		Action: SignalActionMap[sig],
	}, nil
}

// SetGlobalConfig sets the global default configuration for a signal.
func (t *SignalTable) SetGlobalConfig(sig Signal, config *SignalConfig) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.globalConfig[sig] = config
}

// RemoveConfig removes a signal configuration.
func (t *SignalTable) RemoveConfig(pid int, sig Signal) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if configs, exists := t.configs[pid]; exists {
		delete(configs, sig)
	}
}

// SignalSet represents a set of signals.
type SignalSet map[Signal]bool

// NewSignalSet creates a new signal set.
func NewSignalSet(signals ...Signal) SignalSet {
	ss := make(SignalSet)
	for _, s := range signals {
		ss[s] = true
	}
	return ss
}

// Add adds a signal to the set.
func (s SignalSet) Add(sig Signal) {
	s[sig] = true
}

// Remove removes a signal from the set.
func (s SignalSet) Remove(sig Signal) {
	delete(s, sig)
}

// Contains checks if a signal is in the set.
func (s SignalSet) Contains(sig Signal) bool {
	return s[sig]
}

// IsEmpty returns true if the set is empty.
func (s SignalSet) IsEmpty() bool {
	return len(s) == 0
}

// Union returns the union of two signal sets.
func (s SignalSet) Union(other SignalSet) SignalSet {
	result := make(SignalSet)
	for sig := range s {
		result[sig] = true
	}
	for sig := range other {
		result[sig] = true
	}
	return result
}

// Intersection returns the intersection of two signal sets.
func (s SignalSet) Intersection(other SignalSet) SignalSet {
	result := make(SignalSet)
	for sig := range s {
		if other[sig] {
			result[sig] = true
		}
	}
	return result
}

// Difference returns the difference of two signal sets.
func (s SignalSet) Difference(other SignalSet) SignalSet {
	result := make(SignalSet)
	for sig := range s {
		if !other[sig] {
			result[sig] = true
		}
	}
	return result
}
