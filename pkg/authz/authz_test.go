package authz

import (
	"testing"
	"time"
)

// TestPolicyValidation tests policy validation
func TestPolicyValidation(t *testing.T) {
	t.Run("valid policy", func(t *testing.T) {
		policy := NewPolicy("test-policy", EffectAllow)
		err := policy.Validate()
		if err != nil {
			t.Errorf("Validate returned error: %v", err)
		}
	})

	t.Run("policy without name", func(t *testing.T) {
		policy := NewPolicy("", EffectAllow)
		err := policy.Validate()
		if err == nil {
			t.Error("Expected error for empty name")
		}
	})

	t.Run("policy with invalid effect", func(t *testing.T) {
		policy := NewPolicy("test-policy", "invalid")
		err := policy.Validate()
		if err == nil {
			t.Error("Expected error for invalid effect")
		}
	})
}

// TestPolicyMatching tests policy pattern matching
func TestPolicyMatching(t *testing.T) {
	policy := &Policy{
		Name:      "test",
		Effect:    EffectAllow,
		Subjects:  []string{"user:*", "admin:*"},
		Resources: []string{"file:/home/*", "api:/users/*"},
		Actions:   []string{"read", "write"},
	}

	t.Run("matching subject", func(t *testing.T) {
		if !policy.MatchesSubject("user:123") {
			t.Error("Expected user:123 to match pattern user:*")
		}
		if !policy.MatchesSubject("admin:456") {
			t.Error("Expected admin:456 to match pattern admin:*")
		}
	})

	t.Run("non-matching subject", func(t *testing.T) {
		if policy.MatchesSubject("guest:789") {
			t.Error("Expected guest:789 not to match")
		}
	})

	t.Run("matching resource", func(t *testing.T) {
		if !policy.MatchesResource("file:/home/user/doc.txt") {
			t.Error("Expected resource to match pattern")
		}
		if !policy.MatchesResource("api:/users/123") {
			t.Error("Expected API resource to match")
		}
	})

	t.Run("non-matching resource", func(t *testing.T) {
		if policy.MatchesResource("file:/etc/passwd") {
			t.Error("Expected /etc/passwd not to match /home/*")
		}
	})

	t.Run("matching action", func(t *testing.T) {
		if !policy.MatchesAction("read") {
			t.Error("Expected read action to match")
		}
		if !policy.MatchesAction("write") {
			t.Error("Expected write action to match")
		}
	})

	t.Run("non-matching action", func(t *testing.T) {
		if policy.MatchesAction("delete") {
			t.Error("Expected delete action not to match")
		}
	})
}

// TestRole tests role functionality
func TestRole(t *testing.T) {
	role := NewRole("admin")
	role.Permissions = []string{"users:*", "system:*"}

	t.Run("has permission", func(t *testing.T) {
		if !role.HasPermission("users:read") {
			t.Error("Expected role to have users:read permission")
		}
		if !role.HasPermission("system:config") {
			t.Error("Expected role to have system:config permission")
		}
	})

	t.Run("missing permission", func(t *testing.T) {
		if role.HasPermission("files:write") {
			t.Error("Expected role to not have files:write permission")
		}
	})
}

// TestPolicyStore tests the policy store
func TestPolicyStore(t *testing.T) {
	store := NewPolicyStore()

	t.Run("add policy", func(t *testing.T) {
		policy := NewPolicy("test-policy", EffectAllow)
		err := store.AddPolicy(policy)
		if err != nil {
			t.Errorf("AddPolicy returned error: %v", err)
		}
	})

	t.Run("add duplicate policy", func(t *testing.T) {
		policy := NewPolicy("test-policy", EffectAllow)
		err := store.AddPolicy(policy)
		if err == nil {
			t.Error("Expected error for duplicate policy")
		}
	})

	t.Run("get policy", func(t *testing.T) {
		policy, ok := store.GetPolicy("test-policy")
		if !ok {
			t.Error("Expected policy to be found")
		}
		if policy.Name != "test-policy" {
			t.Errorf("Policy name = %s, want test-policy", policy.Name)
		}
	})

	t.Run("get non-existent policy", func(t *testing.T) {
		_, ok := store.GetPolicy("non-existent")
		if ok {
			t.Error("Expected policy not to be found")
		}
	})

	t.Run("remove policy", func(t *testing.T) {
		err := store.RemovePolicy("test-policy")
		if err != nil {
			t.Errorf("RemovePolicy returned error: %v", err)
		}

		_, ok := store.GetPolicy("test-policy")
		if ok {
			t.Error("Expected policy to be removed")
		}
	})

	t.Run("list policies", func(t *testing.T) {
		store.AddPolicy(NewPolicy("policy1", EffectAllow))
		store.AddPolicy(NewPolicy("policy2", EffectAllow))

		policies := store.ListPolicies()
		if len(policies) < 2 {
			t.Errorf("Expected at least 2 policies, got %d", len(policies))
		}
	})
}

// TestAuthorizer tests the main authorizer
func TestAuthorizer(t *testing.T) {
	auth := NewAuthorizer()

	// Add policies
	auth.AddPolicy(&Policy{
		Name:      "read-files",
		Effect:    EffectAllow,
		Subjects:  []string{"user:*"},
		Resources: []string{"file:/home/*"},
		Actions:   []string{"read"},
		Priority:  10,
		Enabled:   true,
	})

	auth.AddPolicy(&Policy{
		Name:      "write-files",
		Effect:    EffectAllow,
		Subjects:  []string{"user:*"},
		Resources: []string{"file:/home/*"},
		Actions:   []string{"write"},
		Priority:  10,
		Enabled:   true,
	})

	auth.AddPolicy(&Policy{
		Name:      "deny-system",
		Effect:    EffectDeny,
		Subjects:  []string{"user:*"},
		Resources: []string{"system:*"},
		Actions:   []string{"*"},
		Priority:  100, // Higher priority
		Enabled:   true,
	})

	t.Run("allow read access", func(t *testing.T) {
		decision, err := auth.CheckAccess("user:123", "file:/home/user/doc.txt", "read")
		if err != nil {
			t.Errorf("CheckAccess returned error: %v", err)
		}
		if decision.Decision != DecisionAllowed {
			t.Errorf("Decision = %s, want %s", decision.Decision, DecisionAllowed)
		}
	})

	t.Run("allow write access", func(t *testing.T) {
		decision, err := auth.CheckAccess("user:123", "file:/home/user/doc.txt", "write")
		if err != nil {
			t.Errorf("CheckAccess returned error: %v", err)
		}
		if decision.Decision != DecisionAllowed {
			t.Errorf("Decision = %s, want %s", decision.Decision, DecisionAllowed)
		}
	})

	t.Run("deny system access", func(t *testing.T) {
		decision, err := auth.CheckAccess("user:123", "system:/etc/config", "read")
		if err != ErrAccessDenied {
			t.Errorf("Expected ErrAccessDenied, got %v", err)
		}
		if decision.Decision != DecisionDenied {
			t.Errorf("Decision = %s, want %s", decision.Decision, DecisionDenied)
		}
	})

	t.Run("deny by default", func(t *testing.T) {
		_, err := auth.CheckAccess("user:123", "file:/etc/passwd", "read")
		if err != ErrAccessDenied {
			t.Errorf("Expected ErrAccessDenied, got %v", err)
		}
	})

	t.Run("IsAllowed helper", func(t *testing.T) {
		if !auth.IsAllowed("user:123", "file:/home/user/doc.txt", "read") {
			t.Error("Expected access to be allowed")
		}
		if auth.IsAllowed("user:123", "system:/etc/passwd", "read") {
			t.Error("Expected access to be denied")
		}
	})
}

// TestRoleAssignment tests role assignment
func TestRoleAssignment(t *testing.T) {
	auth := NewAuthorizer()

	// Add a role
	auth.AddRole(&Role{
		Name:        "file-admin",
		Description: "Can manage files",
		Permissions: []string{"file:*"},
	})

	// Assign role to user
	err := auth.AssignRole("user123", "file-admin")
	if err != nil {
		t.Errorf("AssignRole returned error: %v", err)
	}

	// Check permission through role
	if !auth.HasPermission("user123", "file:/home/user/doc.txt") {
		t.Error("Expected user to have file permission through role")
	}

	// Revoke role
	err = auth.RevokeRole("user123", "file-admin")
	if err != nil {
		t.Errorf("RevokeRole returned error: %v", err)
	}

	if auth.HasPermission("user123", "file:/home/user/doc.txt") {
		t.Error("Expected user to not have permission after role revocation")
	}
}

// TestDefaultDecision tests setting default decision
func TestDefaultDecision(t *testing.T) {
	auth := NewAuthorizer()

	// Set default to allow
	auth.SetDefaultDecision(DecisionAllowed)
	decision, _ := auth.CheckAccess("user:123", "unknown:resource", "read")
	if decision.Decision != DecisionAllowed {
		t.Errorf("Decision = %s, want %s", decision.Decision, DecisionAllowed)
	}

	// Set default back to deny
	auth.SetDefaultDecision(DecisionDenied)
	decision, _ = auth.CheckAccess("user:123", "unknown:resource", "read")
	if decision.Decision != DecisionDenied {
		t.Errorf("Decision = %s, want %s", decision.Decision, DecisionDenied)
	}
}

// TestAuditLog tests audit logging
func TestAuditLog(t *testing.T) {
	auth := NewAuthorizerWithAudit()

	// Make some access requests
	auth.CheckAccess("user:123", "file:/home/user/doc.txt", "read")
	auth.CheckAccess("user:123", "system:/etc/passwd", "read")

	log := auth.GetAuditLog()
	if len(log) != 2 {
		t.Errorf("Audit log length = %d, want 2", len(log))
	}

	// Clear log
	auth.ClearAuditLog()
	log = auth.GetAuditLog()
	if len(log) != 0 {
		t.Errorf("Audit log length after clear = %d, want 0", len(log))
	}
}

// TestAccessRequest tests access request creation
func TestAccessRequest(t *testing.T) {
	req := NewAccessRequest("user:123", "file:/home/doc.txt", "read")
	if req.Subject != "user:123" {
		t.Errorf("Subject = %s, want user:123", req.Subject)
	}
	if req.Resource != "file:/home/doc.txt" {
		t.Errorf("Resource = %s, want file:/home/doc.txt", req.Resource)
	}
	if req.Action != "read" {
		t.Errorf("Action = %s, want read", req.Action)
	}
	if req.Context == nil {
		t.Error("Expected context to be initialized")
	}
}

// TestPatternMatching tests wildcard pattern matching
func TestPatternMatching(t *testing.T) {
	tests := []struct {
		value    string
		pattern  string
		expected bool
	}{
		{"user:123", "user:*", true},
		{"user:123", "user:123", true},
		{"user:123", "admin:*", false},
		{"file:/home/user/doc.txt", "file:/home/*", true},
		{"file:/home/user/doc.txt", "file:/home/user/*", true},
		{"file:/etc/passwd", "file:/home/*", false},
		{"read", "*", true},
		{"write", "*", true},
	}

	for _, tt := range tests {
		result := matchPattern(tt.value, tt.pattern)
		if result != tt.expected {
			t.Errorf("matchPattern(%s, %s) = %v, want %v",
				tt.value, tt.pattern, result, tt.expected)
		}
	}
}

// TestPolicyPriority tests policy priority ordering
func TestPolicyPriority(t *testing.T) {
	auth := NewAuthorizer()

	// Add lower priority allow policy
	auth.AddPolicy(&Policy{
		Name:      "allow-all",
		Effect:    EffectAllow,
		Subjects:  []string{"user:*"},
		Resources: []string{"*"},
		Actions:   []string{"*"},
		Priority:  1,
		Enabled:   true,
	})

	// Add higher priority deny policy
	auth.AddPolicy(&Policy{
		Name:      "deny-system",
		Effect:    EffectDeny,
		Subjects:  []string{"user:*"},
		Resources: []string{"system:*"},
		Actions:   []string{"*"},
		Priority:  100,
		Enabled:   true,
	})

	// System access should be denied despite allow-all
	decision, _ := auth.CheckAccess("user:123", "system:/config", "read")
	if decision.Decision != DecisionDenied {
		t.Errorf("Decision = %s, want %s (deny should take precedence)", decision.Decision, DecisionDenied)
	}
}

// TestDisabledPolicy tests that disabled policies are not evaluated
func TestDisabledPolicy(t *testing.T) {
	auth := NewAuthorizer()

	auth.AddPolicy(&Policy{
		Name:      "allow-access",
		Effect:    EffectAllow,
		Subjects:  []string{"user:*"},
		Resources: []string{"file:/home/*"},
		Actions:   []string{"read"},
		Priority:  10,
		Enabled:   false, // Disabled
	})

	// Access should be denied since policy is disabled
	decision, err := auth.CheckAccess("user:123", "file:/home/user/doc.txt", "read")
	if err != ErrAccessDenied {
		t.Errorf("Expected ErrAccessDenied, got %v", err)
	}
	if decision.Decision != DecisionDenied {
		t.Errorf("Decision = %s, want %s", decision.Decision, DecisionDenied)
	}
}

// TestGetUserPermissions tests getting all permissions for a user
func TestGetUserPermissions(t *testing.T) {
	auth := NewAuthorizer()

	// Create roles
	auth.AddRole(&Role{
		Name:        "reader",
		Permissions: []string{"files:read", "docs:read"},
	})
	auth.AddRole(&Role{
		Name:        "writer",
		Permissions: []string{"files:write"},
	})

	// Assign roles to user
	auth.AssignRole("user123", "reader")
	auth.AssignRole("user123", "writer")

	perms := auth.GetUserPermissions("user123")
	if len(perms) < 3 {
		t.Errorf("Expected at least 3 permissions, got %d", len(perms))
	}
}

// TestPolicyStoreRole tests role operations in policy store
func TestPolicyStoreRole(t *testing.T) {
	store := NewPolicyStore()

	t.Run("add role", func(t *testing.T) {
		role := NewRole("test-role")
		err := store.AddRole(role)
		if err != nil {
			t.Errorf("AddRole returned error: %v", err)
		}
	})

	t.Run("get role", func(t *testing.T) {
		role, ok := store.GetRole("test-role")
		if !ok {
			t.Error("Expected role to be found")
		}
		if role.Name != "test-role" {
			t.Errorf("Role name = %s, want test-role", role.Name)
		}
	})

	t.Run("list roles", func(t *testing.T) {
		roles := store.ListRoles()
		if len(roles) < 1 {
			t.Errorf("Expected at least 1 role, got %d", len(roles))
		}
	})
}

// TestAccessDecision tests access decision creation
func TestAccessDecision(t *testing.T) {
	request := NewAccessRequest("user:123", "file:/doc.txt", "read")
	decision := &AccessDecision{
		Decision:  DecisionAllowed,
		Reason:    "Test policy",
		Policy:    "test-policy",
		Request:   request,
		Timestamp: time.Now(),
	}

	if decision.Decision != DecisionAllowed {
		t.Errorf("Decision = %s, want %s", decision.Decision, DecisionAllowed)
	}
	if decision.Policy != "test-policy" {
		t.Errorf("Policy = %s, want test-policy", decision.Policy)
	}
	if decision.Request != request {
		t.Error("Request mismatch")
	}
}

// TestRoleInheritance tests role inheritance
func TestRoleInheritance(t *testing.T) {
	store := NewPolicyStore()

	// Add parent role
	parentRole := NewRole("parent")
	parentRole.Permissions = []string{"admin:*"}
	store.AddRole(parentRole)

	// Add child role with inheritance
	childRole := NewRole("child")
	childRole.InheritFrom = []string{"parent"}
	childRole.Permissions = []string{"user:*"}
	store.AddRole(childRole)

	// Assign child role to user
	childRole.Users = append(childRole.Users, "user123")

	// Get permissions should include inherited permissions
	perms := store.GetAllPermissionsForUser("user123")
	found := false
	for _, p := range perms {
		if p == "admin:*" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected inherited permission admin:* to be included")
	}
}

// TestRoleNotFound tests role operations with non-existent role
func TestRoleNotFound(t *testing.T) {
	auth := NewAuthorizer()

	// Assign non-existent role
	err := auth.AssignRole("user123", "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent role")
	}

	// Revoke non-existent role
	err = auth.RevokeRole("user123", "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent role")
	}
}

// TestGetRoleNotFound tests getting non-existent role
func TestGetRoleNotFound(t *testing.T) {
	store := NewPolicyStore()

	role, ok := store.GetRole("nonexistent")
	if ok {
		t.Error("Expected role not to be found")
	}
	if role != nil {
		t.Error("Expected nil role")
	}
}

// TestRemoveRoleNotFound tests removing non-existent role
func TestRemoveRoleNotFound(t *testing.T) {
	store := NewPolicyStore()

	err := store.RemoveRole("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent role")
	}
}

// TestAddInvalidPolicy tests adding invalid policy
func TestAddInvalidPolicy(t *testing.T) {
	store := NewPolicyStore()

	// Invalid policy (empty name)
	err := store.AddPolicy(NewPolicy("", EffectAllow))
	if err == nil {
		t.Error("Expected error for invalid policy")
	}
}

// TestHasPermissionNoMatch tests HasPermission with non-matching permission
func TestHasPermissionNoMatch(t *testing.T) {
	role := NewRole("test")
	role.Permissions = []string{"read:*"}

	if role.HasPermission("write:file") {
		t.Error("Expected permission not to match")
	}
}

// TestListPoliciesEmpty tests listing policies from empty store
func TestListPoliciesEmpty(t *testing.T) {
	store := NewPolicyStore()

	policies := store.ListPolicies()
	if len(policies) != 0 {
		t.Errorf("Expected 0 policies, got %d", len(policies))
	}
}

// TestListRolesEmpty tests listing roles from empty store
func TestListRolesEmpty(t *testing.T) {
	store := NewPolicyStore()

	roles := store.ListRoles()
	if len(roles) != 0 {
		t.Errorf("Expected 0 roles, got %d", len(roles))
	}
}

// TestRemovePolicyNotFound tests removing non-existent policy
func TestRemovePolicyNotFound(t *testing.T) {
	store := NewPolicyStore()

	err := store.RemovePolicy("nonexistent")
	if err != ErrPolicyNotFound {
		t.Errorf("Expected ErrPolicyNotFound, got %v", err)
	}
}

// TestAdvancedPatternMatching tests advanced wildcard patterns
func TestAdvancedPatternMatching(t *testing.T) {
	tests := []struct {
		value    string
		pattern  string
		expected bool
	}{
		{"api:/users/123/profile", "api:/users/*", true},
		{"api:/users/123/profile", "api:/users/*/profile", true},
		{"api:/users/123/profile", "api:/admins/*/profile", false},
		{"resource:db:primary", "resource:*:primary", true},
		{"resource:db:secondary", "resource:*:primary", false},
	}

	for _, tt := range tests {
		result := matchPattern(tt.value, tt.pattern)
		if result != tt.expected {
			t.Errorf("matchPattern(%s, %s) = %v, want %v",
				tt.value, tt.pattern, result, tt.expected)
		}
	}
}

// TestQuestionMarkPattern tests ? wildcard pattern
func TestQuestionMarkPattern(t *testing.T) {
	if !matchPattern("user1", "user?") {
		t.Error("Expected user1 to match user?")
	}
	if !matchPattern("user9", "user?") {
		t.Error("Expected user9 to match user?")
	}
	if matchPattern("user123", "user?") {
		t.Error("Expected user123 not to match user?")
	}
}

// TestEmptyPattern tests empty pattern handling
func TestEmptyPattern(t *testing.T) {
	if !matchPattern("", "") {
		t.Error("Expected empty to match empty")
	}
	if matchPattern("value", "") {
		t.Error("Expected value not to match empty pattern")
	}
}

// TestRoleWithDescription tests role with description
func TestRoleWithDescription(t *testing.T) {
	role := NewRole("admin")
	role.Description = "Administrator role with full access"

	if role.Description != "Administrator role with full access" {
		t.Errorf("Description = %s, want expected value", role.Description)
	}
}

// TestPolicyWithDescription tests policy with description
func TestPolicyWithDescription(t *testing.T) {
	policy := NewPolicy("test-policy", EffectAllow)
	policy.Description = "Test policy for unit tests"

	if policy.Description != "Test policy for unit tests" {
		t.Errorf("Description = %s, want expected value", policy.Description)
	}
}

// TestPolicyConditions tests policy with conditions
func TestPolicyConditions(t *testing.T) {
	policy := NewPolicy("test-policy", EffectAllow)
	policy.Conditions = map[string]interface{}{
		"ipRange":    "192.168.1.*",
		"requireMFA": true,
	}

	if policy.Conditions["ipRange"] != "192.168.1.*" {
		t.Error("Expected ipRange condition to be set")
	}
}

// TestGetAllPermissionsForUserEmpty tests getting permissions for user with no roles
func TestGetAllPermissionsForUserEmpty(t *testing.T) {
	store := NewPolicyStore()

	perms := store.GetAllPermissionsForUser("nonexistent")
	if len(perms) != 0 {
		t.Errorf("Expected 0 permissions, got %d", len(perms))
	}
}

// TestCheckAccessWithContext tests access check with context
func TestCheckAccessWithContext(t *testing.T) {
	auth := NewAuthorizer()

	auth.AddPolicy(&Policy{
		Name:      "context-policy",
		Effect:    EffectAllow,
		Subjects:  []string{"user:*"},
		Resources: []string{"resource:*"},
		Actions:   []string{"action:*"},
		Priority:  10,
		Enabled:   true,
	})

	context := map[string]interface{}{
		"ipAddress": "192.168.1.100",
	}

	decision, err := auth.CheckAccessWithContext("user:123", "resource:test", "action:run", context)
	if err != nil {
		t.Errorf("CheckAccessWithContext returned error: %v", err)
	}
	if decision.Decision != DecisionAllowed {
		t.Errorf("Decision = %s, want %s", decision.Decision, DecisionAllowed)
	}
}

// TestAssignDuplicateRole tests assigning same role twice
func TestAssignDuplicateRole(t *testing.T) {
	auth := NewAuthorizer()

	auth.AddRole(NewRole("test-role"))

	err := auth.AssignRole("user123", "test-role")
	if err != nil {
		t.Errorf("AssignRole returned error: %v", err)
	}

	// Second assignment should not fail (idempotent)
	err = auth.AssignRole("user123", "test-role")
	if err != nil {
		t.Errorf("AssignRole returned error on duplicate: %v", err)
	}
}

// TestPolicyListOrdering tests that policies are returned in list order
func TestPolicyListOrdering(t *testing.T) {
	store := NewPolicyStore()

	store.AddPolicy(NewPolicy("policy1", EffectAllow))
	store.AddPolicy(NewPolicy("policy2", EffectDeny))
	store.AddPolicy(NewPolicy("policy3", EffectAllow))

	policies := store.ListPolicies()
	if len(policies) != 3 {
		t.Errorf("Expected 3 policies, got %d", len(policies))
	}
}

// TestRoleListOrdering tests that roles are returned in list order
func TestRoleListOrdering(t *testing.T) {
	store := NewPolicyStore()

	store.AddRole(NewRole("role1"))
	store.AddRole(NewRole("role2"))
	store.AddRole(NewRole("role3"))

	roles := store.ListRoles()
	if len(roles) != 3 {
		t.Errorf("Expected 3 roles, got %d", len(roles))
	}
}

// TestAuthorizerGetPolicy tests getting policy from authorizer
func TestAuthorizerGetPolicy(t *testing.T) {
	auth := NewAuthorizer()

	auth.AddPolicy(NewPolicy("test-policy", EffectAllow))

	policy, ok := auth.GetPolicy("test-policy")
	if !ok {
		t.Error("Expected policy to be found")
	}
	if policy.Name != "test-policy" {
		t.Errorf("Policy name = %s, want test-policy", policy.Name)
	}
}

// TestAuthorizerRemovePolicy tests removing policy from authorizer
func TestAuthorizerRemovePolicy(t *testing.T) {
	auth := NewAuthorizer()

	auth.AddPolicy(NewPolicy("test-policy", EffectAllow))

	err := auth.RemovePolicy("test-policy")
	if err != nil {
		t.Errorf("RemovePolicy returned error: %v", err)
	}

	_, ok := auth.GetPolicy("test-policy")
	if ok {
		t.Error("Expected policy to be removed")
	}
}

// TestAuthorizerListPolicies tests listing policies from authorizer
func TestAuthorizerListPolicies(t *testing.T) {
	auth := NewAuthorizer()

	auth.AddPolicy(NewPolicy("policy1", EffectAllow))
	auth.AddPolicy(NewPolicy("policy2", EffectDeny))

	policies := auth.ListPolicies()
	if len(policies) != 2 {
		t.Errorf("Expected 2 policies, got %d", len(policies))
	}
}

// TestAuthorizerGetRole tests getting role from authorizer
func TestAuthorizerGetRole(t *testing.T) {
	auth := NewAuthorizer()

	auth.AddRole(NewRole("test-role"))

	role, ok := auth.GetRole("test-role")
	if !ok {
		t.Error("Expected role to be found")
	}
	if role.Name != "test-role" {
		t.Errorf("Role name = %s, want test-role", role.Name)
	}
}

// TestAuthorizerRemoveRole tests removing role from authorizer
func TestAuthorizerRemoveRole(t *testing.T) {
	auth := NewAuthorizer()

	auth.AddRole(NewRole("test-role"))

	err := auth.RemoveRole("test-role")
	if err != nil {
		t.Errorf("RemoveRole returned error: %v", err)
	}

	_, ok := auth.GetRole("test-role")
	if ok {
		t.Error("Expected role to be removed")
	}
}

// TestAuthorizerListRoles tests listing roles from authorizer
func TestAuthorizerListRoles(t *testing.T) {
	auth := NewAuthorizer()

	auth.AddRole(NewRole("role1"))
	auth.AddRole(NewRole("role2"))

	roles := auth.ListRoles()
	if len(roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(roles))
	}
}
