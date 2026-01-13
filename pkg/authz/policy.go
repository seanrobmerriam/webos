package authz

import (
	"errors"
	"strings"
	"sync"
	"time"
)

// Policy effect constants
const (
	// EffectAllow indicates the policy allows access.
	EffectAllow = "allow"
	// EffectDeny indicates the policy denies access.
	EffectDeny = "deny"
)

// Common policy errors
var (
	// ErrPolicyNotFound is returned when a policy is not found.
	ErrPolicyNotFound = errors.New("policy not found")
	// ErrPolicyExists is returned when a policy already exists.
	ErrPolicyExists = errors.New("policy already exists")
	// ErrInvalidEffect is returned when the policy effect is invalid.
	ErrInvalidEffect = errors.New("invalid policy effect")
)

// Policy represents an authorization policy.
type Policy struct {
	// Name is the unique name of the policy.
	Name string
	// Description is a human-readable description.
	Description string
	// Effect is either "allow" or "deny".
	Effect string
	// Subjects is a list of subject patterns (e.g., "user:*", "role:admin").
	Subjects []string
	// Resources is a list of resource patterns (e.g., "file:/home/*", "api:/users/*").
	Resources []string
	// Actions is a list of action patterns (e.g., "read", "write", "*").
	Actions []string
	// Conditions is an optional map of conditions that must be met.
	Conditions map[string]interface{}
	// Priority is used for policy ordering (higher = evaluated first).
	Priority int
	// Enabled indicates if the policy is currently active.
	Enabled bool
	// CreatedAt is when the policy was created.
	CreatedAt time.Time
	// UpdatedAt is when the policy was last updated.
	UpdatedAt time.Time
}

// NewPolicy creates a new policy with default settings.
func NewPolicy(name, effect string) *Policy {
	return &Policy{
		Name:       name,
		Effect:     effect,
		Subjects:   []string{},
		Resources:  []string{},
		Actions:    []string{},
		Conditions: make(map[string]interface{}),
		Priority:   0,
		Enabled:    true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// Validate validates the policy configuration.
func (p *Policy) Validate() error {
	if p.Name == "" {
		return errors.New("policy name is required")
	}
	if p.Effect != EffectAllow && p.Effect != EffectDeny {
		return ErrInvalidEffect
	}
	return nil
}

// MatchesSubject checks if the given subject matches any of the policy's subject patterns.
func (p *Policy) MatchesSubject(subject string) bool {
	return matchesAny(subject, p.Subjects)
}

// MatchesResource checks if the given resource matches any of the policy's resource patterns.
func (p *Policy) MatchesResource(resource string) bool {
	return matchesAny(resource, p.Resources)
}

// MatchesAction checks if the given action matches any of the policy's action patterns.
func (p *Policy) MatchesAction(action string) bool {
	return matchesAny(action, p.Actions)
}

// Role represents a role with a set of permissions.
type Role struct {
	// Name is the unique name of the role.
	Name string
	// Description is a human-readable description.
	Description string
	// Permissions is a list of permission strings.
	Permissions []string
	// InheritFrom is a list of roles to inherit permissions from.
	InheritFrom []string
	// Users is a list of users who have this role.
	Users []string
	// CreatedAt is when the role was created.
	CreatedAt time.Time
}

// NewRole creates a new role.
func NewRole(name string) *Role {
	return &Role{
		Name:        name,
		Permissions: []string{},
		InheritFrom: []string{},
		Users:       []string{},
		CreatedAt:   time.Now(),
	}
}

// HasPermission checks if the role has the given permission.
func (r *Role) HasPermission(permission string) bool {
	for _, p := range r.Permissions {
		if matchPattern(permission, p) {
			return true
		}
	}
	return false
}

// PolicyStore manages policies.
type PolicyStore struct {
	policies map[string]*Policy
	roles    map[string]*Role
	mu       sync.RWMutex
}

// NewPolicyStore creates a new PolicyStore.
func NewPolicyStore() *PolicyStore {
	return &PolicyStore{
		policies: make(map[string]*Policy),
		roles:    make(map[string]*Role),
	}
}

// AddPolicy adds a policy to the store.
func (s *PolicyStore) AddPolicy(policy *Policy) error {
	if err := policy.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.policies[policy.Name]; exists {
		return ErrPolicyExists
	}

	s.policies[policy.Name] = policy
	return nil
}

// GetPolicy retrieves a policy by name.
func (s *PolicyStore) GetPolicy(name string) (*Policy, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.policies[name]
	return p, ok
}

// RemovePolicy removes a policy from the store.
func (s *PolicyStore) RemovePolicy(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.policies[name]; !exists {
		return ErrPolicyNotFound
	}

	delete(s.policies, name)
	return nil
}

// ListPolicies returns all policies.
func (s *PolicyStore) ListPolicies() []*Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policies := make([]*Policy, 0, len(s.policies))
	for _, p := range s.policies {
		policies = append(policies, p)
	}
	return policies
}

// AddRole adds a role to the store.
func (s *PolicyStore) AddRole(role *Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.roles[role.Name]; exists {
		return errors.New("role already exists")
	}

	s.roles[role.Name] = role
	return nil
}

// GetRole retrieves a role by name.
func (s *PolicyStore) GetRole(name string) (*Role, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.roles[name]
	return r, ok
}

// RemoveRole removes a role from the store.
func (s *PolicyStore) RemoveRole(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.roles[name]; !exists {
		return errors.New("role not found")
	}

	delete(s.roles, name)
	return nil
}

// ListRoles returns all roles.
func (s *PolicyStore) ListRoles() []*Role {
	s.mu.RLock()
	defer s.mu.RUnlock()

	roles := make([]*Role, 0, len(s.roles))
	for _, r := range s.roles {
		roles = append(roles, r)
	}
	return roles
}

// GetAllPermissionsForUser returns all permissions for a user (including role permissions).
func (s *PolicyStore) GetAllPermissionsForUser(userID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	perms := make(map[string]bool)

	// Collect permissions from all roles the user has
	for _, role := range s.roles {
		for _, user := range role.Users {
			if user == userID {
				for _, perm := range role.Permissions {
					perms[perm] = true
				}
			}
		}
	}

	// Also check role inheritance
	for _, role := range s.roles {
		for _, user := range role.Users {
			if user == userID {
				for _, inheritName := range role.InheritFrom {
					if parentRole, ok := s.roles[inheritName]; ok {
						for _, perm := range parentRole.Permissions {
							perms[perm] = true
						}
					}
				}
			}
		}
	}

	result := make([]string, 0, len(perms))
	for p := range perms {
		result = append(result, p)
	}
	return result
}

// matchesAny checks if the given value matches any of the patterns.
func matchesAny(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchPattern(value, pattern) {
			return true
		}
	}
	return false
}

// matchPattern checks if a value matches a pattern.
// Supports wildcards: * matches anything, ? matches single character.
func matchPattern(value, pattern string) bool {
	if pattern == "" {
		return value == ""
	}

	// Exact match
	if value == pattern {
		return true
	}

	// Wildcard patterns - use simple glob-style matching
	// * matches zero or more characters
	// ? matches exactly one character

	// Handle leading *
	if strings.HasPrefix(pattern, "*") {
		pattern = pattern[1:]
		for i := 0; i <= len(value); i++ {
			if i < len(value) && value[i] == '*' {
				continue
			}
			if matchSimple(value[i:], pattern) {
				return true
			}
		}
		return false
	}

	// Handle trailing *
	if strings.HasSuffix(pattern, "*") {
		pattern = strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(value, pattern)
	}

	// Handle * in the middle
	parts := strings.Split(pattern, "*")
	if len(parts) > 1 {
		return matchWithWildcards(value, parts)
	}

	// Handle ? wildcards
	if strings.Contains(pattern, "?") {
		return matchQuestionMarks(value, pattern)
	}

	return false
}

// matchSimple matches without wildcards (except leading * which was handled)
func matchSimple(s, pattern string) bool {
	if pattern == "" {
		return s == ""
	}
	if len(s) < len(pattern) {
		return false
	}
	if len(s) == len(pattern) {
		return s == pattern
	}
	// Check suffix match
	return strings.HasSuffix(s, pattern)
}

// matchWithWildcards matches a value against pattern parts separated by *
func matchWithWildcards(value string, parts []string) bool {
	idx := 0
	for i, part := range parts {
		if part == "" {
			continue // Skip empty parts (from leading/trailing *)
		}
		if i == 0 {
			// First part - must match prefix
			if !strings.HasPrefix(value[idx:], part) {
				return false
			}
			idx += len(part)
		} else if i == len(parts)-1 {
			// Last part - must match suffix
			return strings.HasSuffix(value, part)
		} else {
			// Middle part - find the part in value
			nextIdx := strings.Index(value[idx:], part)
			if nextIdx == -1 {
				return false
			}
			idx += nextIdx + len(part)
		}
	}
	return true
}

// matchQuestionMarks matches patterns with ? wildcards
func matchQuestionMarks(value, pattern string) bool {
	if len(value) != len(pattern) {
		return false
	}
	for i := 0; i < len(pattern); i++ {
		if pattern[i] != '?' && pattern[i] != value[i] {
			return false
		}
	}
	return true
}
