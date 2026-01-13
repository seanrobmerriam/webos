package security

import (
	"errors"
	"sync"
	"time"
)

// Common security manager errors
var (
	// ErrComponentAlreadyRegistered is returned when registering a component that already exists.
	ErrComponentAlreadyRegistered = errors.New("component already registered")
	// ErrComponentNotFound is returned when a component is not found.
	ErrComponentNotFound = errors.New("component not found")
	// ErrPolicyNotFound is returned when a policy is not found.
	ErrPolicyNotFound = errors.New("policy not found")
)

// Capability represents the security capabilities granted to a component.
// It combines promised capabilities with filesystem path restrictions.
type Capability struct {
	// Promises specifies the operations the component is allowed to perform.
	Promises Promise
	// UnveilPaths specifies the filesystem paths the component can access.
	UnveilPaths []UnveilPath
	// Timeout specifies the maximum duration for which the capability is valid.
	Timeout time.Duration
}

// SecurityManager manages security policies and capabilities for components.
// It is the central authority for enforcing capability-based security.
type SecurityManager struct {
	// activeCaps stores the capabilities for each registered component.
	activeCaps sync.Map
	// policies stores named policies that can be applied to components.
	policies map[string]Promise
	// unveilRegistry manages filesystem path restrictions.
	unveilRegistry *UnveilRegistry
	mu             sync.RWMutex
}

// NewSecurityManager creates a new SecurityManager with empty policies and
// no registered components.
func NewSecurityManager() *SecurityManager {
	return &SecurityManager{
		policies:       make(map[string]Promise),
		unveilRegistry: NewUnveilRegistry(),
	}
}

// RegisterComponent registers a component with the specified capability.
// Returns ErrComponentAlreadyRegistered if the component is already registered.
func (sm *SecurityManager) RegisterComponent(component string, cap *Capability) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.activeCaps.Load(component); exists {
		return ErrComponentAlreadyRegistered
	}

	sm.activeCaps.Store(component, cap)

	// Register unveil paths
	for _, up := range cap.UnveilPaths {
		sm.unveilRegistry.AddPath(component, up)
	}

	return nil
}

// GetCapability returns the capability for the specified component.
// Returns false if the component is not registered.
func (sm *SecurityManager) GetCapability(component string) (*Capability, bool) {
	cap, ok := sm.activeCaps.Load(component)
	if !ok {
		return nil, false
	}
	return cap.(*Capability), true
}

// RevokeCapability removes the capability for the specified component.
// Returns ErrComponentNotFound if the component is not registered.
func (sm *SecurityManager) RevokeCapability(component string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.activeCaps.Load(component); !exists {
		return ErrComponentNotFound
	}

	sm.activeCaps.Delete(component)
	sm.unveilRegistry.RemovePaths(component)

	return nil
}

// CheckPermission returns true if the component has the specified capability.
func (sm *SecurityManager) CheckPermission(component string, promise Promise) bool {
	cap, ok := sm.GetCapability(component)
	if !ok {
		return false
	}
	return cap.Promises.HasCapability(promise)
}

// RegisterPolicy registers a named policy with the specified promises.
func (sm *SecurityManager) RegisterPolicy(name string, promises Promise) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.policies[name] = promises
	return nil
}

// GetPolicy returns the promises for the specified policy.
// Returns false if the policy is not found.
func (sm *SecurityManager) GetPolicy(name string) (Promise, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	promises, ok := sm.policies[name]
	return promises, ok
}

// ApplyPolicy applies a named policy to the specified component.
// The component must already be registered.
// Returns ErrPolicyNotFound if the policy is not found.
func (sm *SecurityManager) ApplyPolicy(component, policyName string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	promises, ok := sm.policies[policyName]
	if !ok {
		return ErrPolicyNotFound
	}

	cap, exists := sm.activeCaps.Load(component)
	if !exists {
		return ErrComponentNotFound
	}

	// Update the capability's promises
	c := cap.(*Capability)
	c.Promises = promises

	return nil
}

// AddUnveilPath adds a filesystem path restriction for the specified component.
func (sm *SecurityManager) AddUnveilPath(component, path, permissions string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	up, err := NewUnveilPath(path, permissions)
	if err != nil {
		return err
	}

	cap, exists := sm.activeCaps.Load(component)
	if !exists {
		return ErrComponentNotFound
	}

	c := cap.(*Capability)
	c.UnveilPaths = append(c.UnveilPaths, *up)
	sm.unveilRegistry.AddPath(component, *up)

	return nil
}

// GetUnveilPaths returns the filesystem paths the component can access.
func (sm *SecurityManager) GetUnveilPaths(component string) []UnveilPath {
	return sm.unveilRegistry.GetPaths(component)
}

// CanAccessPath checks if the component can access the specified path
// with the requested permissions.
func (sm *SecurityManager) CanAccessPath(component, path string, wantRead, wantWrite, wantExecute bool) bool {
	return sm.unveilRegistry.CanAccessPath(component, path, wantRead, wantWrite, wantExecute)
}

// UpdateTimeout updates the timeout for the specified component.
func (sm *SecurityManager) UpdateTimeout(component string, timeout time.Duration) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	cap, exists := sm.activeCaps.Load(component)
	if !exists {
		return ErrComponentNotFound
	}

	c := cap.(*Capability)
	c.Timeout = timeout
	return nil
}

// ListComponents returns a list of all registered component names.
func (sm *SecurityManager) ListComponents() []string {
	components := make([]string, 0)

	sm.activeCaps.Range(func(key, value interface{}) bool {
		components = append(components, key.(string))
		return true
	})

	return components
}

// ListPolicies returns a list of all registered policy names.
func (sm *SecurityManager) ListPolicies() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	policies := make([]string, 0, len(sm.policies))
	for name := range sm.policies {
		policies = append(policies, name)
	}
	return policies
}
