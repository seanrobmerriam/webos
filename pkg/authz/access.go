package authz

import (
	"errors"
	"sync"
	"time"
)

// Decision constants
const (
	// DecisionAllowed indicates access is granted.
	DecisionAllowed = "allowed"
	// DecisionDenied indicates access is denied.
	DecisionDenied = "denied"
	// DecisionNoMatch indicates no policy matched the request.
	DecisionNoMatch = "no_match"
)

// Access-related errors
var (
	// ErrAccessDenied is returned when access is denied.
	ErrAccessDenied = errors.New("access denied")
	// ErrNoMatchingPolicy is returned when no policy matches the request.
	ErrNoMatchingPolicy = errors.New("no matching policy")
)

// AccessRequest represents an access control request.
type AccessRequest struct {
	// Subject is the entity requesting access (e.g., user ID, role).
	Subject string
	// Resource is the resource being accessed.
	Resource string
	// Action is the action being performed.
	Action string
	// Context is additional context for the request.
	Context map[string]interface{}
	// Timestamp is when the request was made.
	Timestamp time.Time
}

// NewAccessRequest creates a new access request.
func NewAccessRequest(subject, resource, action string) *AccessRequest {
	return &AccessRequest{
		Subject:   subject,
		Resource:  resource,
		Action:    action,
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// AccessDecision represents the result of an access check.
type AccessDecision struct {
	// Decision is the decision (allowed, denied, no_match).
	Decision string
	// Reason explains why the decision was made.
	Reason string
	// Policy is the name of the matching policy (if any).
	Policy string
	// Request is the original request.
	Request *AccessRequest
	// Timestamp is when the decision was made.
	Timestamp time.Time
}

// AuditLogEntry represents an audit log entry.
type AuditLogEntry struct {
	Request   *AccessRequest
	Decision  *AccessDecision
	IPAddress string
	UserAgent string
	Timestamp time.Time
}

// Authorizer performs access control checks.
type Authorizer struct {
	store    *PolicyStore
	auditLog []AuditLogEntry
	auditMu  sync.Mutex
	defaultD string // Default decision when no policy matches
	mu       sync.RWMutex
}

// NewAuthorizer creates a new Authorizer.
func NewAuthorizer() *Authorizer {
	return &Authorizer{
		store:    NewPolicyStore(),
		auditLog: make([]AuditLogEntry, 0),
		defaultD: DecisionDenied,
	}
}

// NewAuthorizerWithAudit creates a new Authorizer with audit logging enabled.
func NewAuthorizerWithAudit() *Authorizer {
	return &Authorizer{
		store:    NewPolicyStore(),
		auditLog: make([]AuditLogEntry, 0),
		defaultD: DecisionDenied,
	}
}

// CheckAccess checks if the subject can perform the action on the resource.
func (a *Authorizer) CheckAccess(subject, resource, action string) (*AccessDecision, error) {
	return a.CheckAccessWithContext(subject, resource, action, nil)
}

// CheckAccessWithContext checks access with additional context.
func (a *Authorizer) CheckAccessWithContext(subject, resource, action string, context map[string]interface{}) (*AccessDecision, error) {
	request := &AccessRequest{
		Subject:   subject,
		Resource:  resource,
		Action:    action,
		Context:   context,
		Timestamp: time.Now(),
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	// Get all enabled policies sorted by priority (higher first)
	policies := a.getEnabledPolicies()

	// Evaluate policies in priority order
	// DENY takes precedence over ALLOW
	var lastMatch *Policy

	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}

		if policy.MatchesSubject(subject) &&
			policy.MatchesResource(resource) &&
			policy.MatchesAction(action) {
			lastMatch = policy

			if policy.Effect == EffectDeny {
				decision := &AccessDecision{
					Decision:  DecisionDenied,
					Reason:    "Denied by policy",
					Policy:    policy.Name,
					Request:   request,
					Timestamp: time.Now(),
				}
				a.logAccess(request, decision)
				return decision, ErrAccessDenied
			}
		}
	}

	// If no DENY was encountered but an ALLOW matched
	if lastMatch != nil {
		decision := &AccessDecision{
			Decision:  DecisionAllowed,
			Reason:    "Allowed by policy",
			Policy:    lastMatch.Name,
			Request:   request,
			Timestamp: time.Now(),
		}
		a.logAccess(request, decision)
		return decision, nil
	}

	// No matching policy - use default decision
	decision := &AccessDecision{
		Decision:  a.defaultD,
		Reason:    "No matching policy, using default",
		Policy:    "",
		Request:   request,
		Timestamp: time.Now(),
	}
	a.logAccess(request, decision)

	if decision.Decision == DecisionDenied {
		return decision, ErrAccessDenied
	}

	return decision, nil
}

// IsAllowed is a convenience method that returns true if access is allowed.
func (a *Authorizer) IsAllowed(subject, resource, action string) bool {
	decision, _ := a.CheckAccess(subject, resource, action)
	return decision.Decision == DecisionAllowed
}

// AddPolicy adds a policy to the authorizer.
func (a *Authorizer) AddPolicy(policy *Policy) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.store.AddPolicy(policy)
}

// RemovePolicy removes a policy from the authorizer.
func (a *Authorizer) RemovePolicy(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.store.RemovePolicy(name)
}

// GetPolicy retrieves a policy by name.
func (a *Authorizer) GetPolicy(name string) (*Policy, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.store.GetPolicy(name)
}

// ListPolicies returns all policies.
func (a *Authorizer) ListPolicies() []*Policy {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.store.ListPolicies()
}

// AddRole adds a role to the authorizer.
func (a *Authorizer) AddRole(role *Role) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.store.AddRole(role)
}

// RemoveRole removes a role from the authorizer.
func (a *Authorizer) RemoveRole(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.store.RemoveRole(name)
}

// GetRole retrieves a role by name.
func (a *Authorizer) GetRole(name string) (*Role, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.store.GetRole(name)
}

// ListRoles returns all roles.
func (a *Authorizer) ListRoles() []*Role {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.store.ListRoles()
}

// AssignRole assigns a role to a user.
func (a *Authorizer) AssignRole(userID, roleName string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	role, ok := a.store.GetRole(roleName)
	if !ok {
		return errors.New("role not found")
	}

	// Check if user already has the role
	for _, u := range role.Users {
		if u == userID {
			return nil // Already assigned
		}
	}

	role.Users = append(role.Users, userID)
	return nil
}

// RevokeRole revokes a role from a user.
func (a *Authorizer) RevokeRole(userID, roleName string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	role, ok := a.store.GetRole(roleName)
	if !ok {
		return errors.New("role not found")
	}

	for i, u := range role.Users {
		if u == userID {
			role.Users = append(role.Users[:i], role.Users[i+1:]...)
			return nil
		}
	}

	return errors.New("user does not have this role")
}

// GetUserPermissions returns all permissions for a user.
func (a *Authorizer) GetUserPermissions(userID string) []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.store.GetAllPermissionsForUser(userID)
}

// HasPermission checks if a user has a specific permission.
func (a *Authorizer) HasPermission(userID, permission string) bool {
	perms := a.GetUserPermissions(userID)
	for _, p := range perms {
		if matchPattern(permission, p) {
			return true
		}
	}
	return false
}

// SetDefaultDecision sets the default decision when no policy matches.
func (a *Authorizer) SetDefaultDecision(decision string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if decision == DecisionAllowed || decision == DecisionDenied {
		a.defaultD = decision
	}
}

// GetAuditLog returns the audit log.
func (a *Authorizer) GetAuditLog() []AuditLogEntry {
	a.auditMu.Lock()
	defer a.auditMu.Unlock()

	log := make([]AuditLogEntry, len(a.auditLog))
	copy(log, a.auditLog)
	return log
}

// ClearAuditLog clears the audit log.
func (a *Authorizer) ClearAuditLog() {
	a.auditMu.Lock()
	defer a.auditMu.Unlock()

	a.auditLog = make([]AuditLogEntry, 0)
}

// getEnabledPolicies returns all enabled policies sorted by priority.
func (a *Authorizer) getEnabledPolicies() []*Policy {
	policies := a.store.ListPolicies()

	// Sort by priority (higher first)
	// Use stable sort to maintain order for same priority
	for i := 0; i < len(policies)-1; i++ {
		for j := i + 1; j < len(policies); j++ {
			if policies[j].Priority > policies[i].Priority {
				policies[i], policies[j] = policies[j], policies[i]
			}
		}
	}

	return policies
}

// logAccess logs an access decision.
func (a *Authorizer) logAccess(request *AccessRequest, decision *AccessDecision) {
	a.auditMu.Lock()
	defer a.auditMu.Unlock()

	a.auditLog = append(a.auditLog, AuditLogEntry{
		Request:   request,
		Decision:  decision,
		Timestamp: time.Now(),
	})

	// Keep log size manageable (max 1000 entries)
	if len(a.auditLog) > 1000 {
		a.auditLog = a.auditLog[len(a.auditLog)-1000:]
	}
}
