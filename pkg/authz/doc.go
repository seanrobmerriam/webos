/*
Package authz provides authorization and access control for the webos project.

This package implements a capability-based authorization system with:

# Policy Engine

Policies define what actions are allowed for different subjects (users, roles,
services) on different resources. Policies can be combined and evaluated
programmatically.

Example:

	policy := &authz.Policy{
		Name:        "file-read",
		Description: "Allow reading files in /home",
		Effect:      authz.EffectAllow,
		Subjects:    []string{"user:*"},
		Resources:   []string{"file:/home/*"},
		Actions:     []string{"read"},
	}

# Role-Based Access Control (RBAC)

Roles group permissions together for easier management. Users can have multiple
roles, and roles can inherit from other roles.

Example:

	adminRole := &authz.Role{
		Name:        "admin",
		Permissions: []string{"users:*", "system:*"},
	}
	userRole := &authz.Role{
		Name:        "user",
		Permissions: []string{"files:read", "files:write"},
	}

# Access Control

The Authorizer evaluates access requests against policies and returns a decision.
It supports both ALLOW and DENY effects, with explicit DENY taking precedence.

Example:

	auth := authz.NewAuthorizer()
	decision, err := auth.CheckAccess("user123", "file:/home/user/doc.txt", "read")
	if decision == authz.DecisionAllowed {
		// Access granted
	}

# Audit Logging

All access decisions are logged for security auditing. This provides an audit
trail of who accessed what and when.

Example:

	auth := authz.NewAuthorizerWithAudit()
	auth.LogAccess(request, decision, reason)

# Thread Safety

All Authorizer operations are thread-safe using sync.RWMutex.
*/
package authz
