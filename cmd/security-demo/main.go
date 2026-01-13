// Package main provides a demonstration of the webOS security foundation.
//
// This demo showcases the OpenBSD-inspired security model including:
// - Capability-based security (pledge system)
// - Filesystem path restrictions (unveil system)
// - Authentication with password hashing and MFA support
// - Authorization with policy-based access control
package main

import (
	"fmt"
	"log"
	"time"

	"webos/pkg/auth"
	"webos/pkg/authz"
	"webos/pkg/security"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("  WebOS Security Foundation Demo")
	fmt.Println("========================================")
	fmt.Println()

	// Demo 1: Security Package (Pledge & Unveil)
	demoSecurityPackage()

	// Demo 2: Authentication Package
	demoAuthPackage()

	// Demo 3: Authorization Package
	demoAuthzPackage()

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  Demo Complete!")
	fmt.Println("========================================")
}

func demoSecurityPackage() {
	fmt.Println("--- Security Package Demo ---")
	fmt.Println()

	// Create a security manager
	sm := security.NewSecurityManager()

	// Create a capability for a component
	capability := &security.Capability{
		Promises: security.PromiseRpath | security.PromiseInet | security.PromiseSignal,
		Timeout:  30 * time.Second,
	}

	// Register a component with capabilities
	err := sm.RegisterComponent("web-browser", capability)
	if err != nil {
		log.Printf("Error registering component: %v", err)
		return
	}

	fmt.Printf("Registered component: %s\n", "web-browser")
	fmt.Printf("Initial promises: %s\n", capability.Promises.String())

	// Check if component has a specific capability
	hasInet := sm.CheckPermission("web-browser", security.PromiseInet)
	fmt.Printf("Has INET capability: %v\n", hasInet)

	// Add an unveil path (filesystem restriction)
	err = sm.AddUnveilPath("web-browser", "/home/user/Downloads", "rw")
	if err != nil {
		log.Printf("Error adding unveil path: %v", err)
		return
	}
	fmt.Printf("Added unveil path: /home/user/Downloads (permissions: rw)\n")

	// Can access check
	canRead := sm.CanAccessPath("web-browser", "/home/user/Downloads/file.txt", true, false, false)
	canWrite := sm.CanAccessPath("web-browser", "/home/user/Downloads/file.txt", false, true, false)
	canReadSystem := sm.CanAccessPath("web-browser", "/etc/passwd", true, false, false)
	fmt.Printf("Can read /home/user/Downloads: %v\n", canRead)
	fmt.Printf("Can write /home/user/Downloads: %v\n", canWrite)
	fmt.Printf("Can read /etc/passwd: %v (should be false due to unveil)\n", canReadSystem)

	// List all components
	components := sm.ListComponents()
	fmt.Printf("Total registered components: %d\n", len(components))

	// Register a named policy
	policyPromises := security.PromiseRpath | security.PromiseWpath
	err = sm.RegisterPolicy("file-access", policyPromises)
	if err != nil {
		log.Printf("Error registering policy: %v", err)
		return
	}
	fmt.Printf("Registered policy: file-access with promises: %s\n", policyPromises.String())

	// Apply policy to component
	err = sm.ApplyPolicy("web-browser", "file-access")
	if err != nil {
		log.Printf("Error applying policy: %v", err)
		return
	}
	fmt.Println("Applied policy 'file-access' to web-browser")

	// List policies
	policies := sm.ListPolicies()
	fmt.Printf("Registered policies: %v\n", policies)

	// Revoke capability
	err = sm.RevokeCapability("web-browser")
	if err != nil {
		log.Printf("Error revoking capability: %v", err)
		return
	}
	fmt.Println("Revoked capability from web-browser")

	fmt.Println()
}

func demoAuthPackage() {
	fmt.Println("--- Authentication Package Demo ---")
	fmt.Println()

	// Create an authenticator
	authenticator := auth.NewAuthenticator()

	// Register a new user
	fmt.Println("1. User Registration")
	user, err := authenticator.RegisterUser("user123", "alice", "alice@example.com", "SecurePassword123!")
	if err != nil {
		log.Printf("Error registering user: %v", err)
		return
	}
	fmt.Printf("   Registered user: %s (ID: %s)\n", user.Username, user.ID)

	// Authenticate with correct password
	fmt.Println("\n2. Authentication")
	session, err := authenticator.Authenticate("alice", "SecurePassword123!")
	if err != nil {
		log.Printf("Error authenticating: %v", err)
		return
	}
	fmt.Printf("   Session created: %s...\n", session.Token[:20])

	// Try wrong password
	_, err = authenticator.Authenticate("alice", "WrongPassword!")
	if err == auth.ErrInvalidCredentials {
		fmt.Println("   Invalid password rejected correctly")
	}

	// Enable MFA
	fmt.Println("\n3. Multi-Factor Authentication")
	secret, err := authenticator.EnableMFA("alice", "WebOS")
	if err != nil {
		log.Printf("Error enabling MFA: %v", err)
		return
	}
	fmt.Printf("   MFA enabled with secret: %s...\n", secret.Secret[:10])
	fmt.Printf("   TOTP URI: %s\n", secret.TOTPURI()[:50]+"...")

	// Authenticate with MFA enabled (should require MFA)
	_, err = authenticator.Authenticate("alice", "SecurePassword123!")
	if err == auth.ErrMFARequired {
		fmt.Println("   MFA required (as expected)")
	}

	// Verify MFA
	totpCode, _ := auth.GenerateTOTP(secret)
	mfaSession, err := authenticator.VerifyMFA("alice", totpCode)
	if err != nil {
		log.Printf("Error verifying MFA: %v", err)
		return
	}
	fmt.Printf("   MFA verified, session: %s...\n", mfaSession.Token[:20])

	// Password hashing demo
	fmt.Println("\n4. Password Hashing")
	password := "mySecretPassword"
	hash, _ := auth.HashPassword(password)
	fmt.Printf("   Password: %s\n", password)
	fmt.Printf("   Hash: %s...\n", hash[:40])

	// Verify password
	err = auth.CheckPassword(password, hash)
	if err == nil {
		fmt.Println("   Password verification: OK")
	}

	// Session management
	fmt.Println("\n5. Session Management")
	sessionCount := authenticator.SessionManager().SessionCount()
	fmt.Printf("   Active sessions: %d\n", sessionCount)

	// Invalidate session
	_ = authenticator.InvalidateSession(mfaSession.Token)
	fmt.Println("   Session invalidated")

	// Lock account demo
	fmt.Println("\n6. Account Locking")
	authenticator.LockAccount("alice", time.Hour)
	if authenticator.IsAccountLocked("alice") {
		fmt.Println("   Account is locked (as expected)")
	}
	authenticator.UnlockAccount("alice")
	if !authenticator.IsAccountLocked("alice") {
		fmt.Println("   Account is unlocked (as expected)")
	}

	fmt.Println()
}

func demoAuthzPackage() {
	fmt.Println("--- Authorization Package Demo ---")
	fmt.Println()

	// Create an authorizer
	authorizer := authz.NewAuthorizer()

	// Define policies
	fmt.Println("1. Policy Definition")

	// Allow read access to home files
	readPolicy := &authz.Policy{
		Name:      "read-home-files",
		Effect:    authz.EffectAllow,
		Subjects:  []string{"user:*", "role:admin"},
		Resources: []string{"file:/home/*"},
		Actions:   []string{"read"},
		Priority:  10,
		Enabled:   true,
	}
	authorizer.AddPolicy(readPolicy)
	fmt.Println("   Added policy: read-home-files")

	// Allow write access to home files
	writePolicy := &authz.Policy{
		Name:      "write-home-files",
		Effect:    authz.EffectAllow,
		Subjects:  []string{"user:*"},
		Resources: []string{"file:/home/*"},
		Actions:   []string{"write"},
		Priority:  10,
		Enabled:   true,
	}
	authorizer.AddPolicy(writePolicy)
	fmt.Println("   Added policy: write-home-files")

	// Deny access to system files
	denySystemPolicy := &authz.Policy{
		Name:      "deny-system",
		Effect:    authz.EffectDeny,
		Subjects:  []string{"user:*"},
		Resources: []string{"system:*"},
		Actions:   []string{"*"},
		Priority:  100, // Higher priority
		Enabled:   true,
	}
	authorizer.AddPolicy(denySystemPolicy)
	fmt.Println("   Added policy: deny-system (priority: 100)")

	// Define roles
	fmt.Println("\n2. Role Definition")

	adminRole := &authz.Role{
		Name:        "file-admin",
		Description: "Can manage files",
		Permissions: []string{"file:*", "admin:*"},
	}
	authorizer.AddRole(adminRole)
	fmt.Println("   Added role: file-admin")

	// Assign role to user
	err := authorizer.AssignRole("user123", "file-admin")
	if err != nil {
		log.Printf("Error assigning role: %v", err)
		return
	}
	fmt.Println("   Assigned file-admin role to user123")

	// Check access
	fmt.Println("\n3. Access Control Checks")

	// Allow case: read home file
	allowed := authorizer.IsAllowed("user:123", "file:/home/user/doc.txt", "read")
	fmt.Printf("   user:123 reads file:/home/user/doc.txt: %s\n", decisionString(allowed))

	// Allow case: write home file
	allowed = authorizer.IsAllowed("user:123", "file:/home/user/doc.txt", "write")
	fmt.Printf("   user:123 writes file:/home/user/doc.txt: %s\n", decisionString(allowed))

	// Deny case: access system file
	allowed = authorizer.IsAllowed("user:123", "system:/etc/config", "read")
	fmt.Printf("   user:123 reads system:/etc/config: %s (deny policy)\n", decisionString(allowed))

	// No matching policy case
	allowed = authorizer.IsAllowed("user:123", "unknown:resource", "action")
	fmt.Printf("   user:123 accesses unknown:resource: %s (default deny)\n", decisionString(allowed))

	// Check permissions through role
	fmt.Println("\n4. Role-Based Permissions")
	perms := authorizer.GetUserPermissions("user123")
	fmt.Printf("   Permissions for user123: %v\n", perms)

	// Revoke role
	err = authorizer.RevokeRole("user123", "file-admin")
	if err != nil {
		log.Printf("Error revoking role: %v", err)
		return
	}
	fmt.Println("   Revoked file-admin role from user123")

	// Check permissions after revocation
	perms = authorizer.GetUserPermissions("user123")
	fmt.Printf("   Permissions after revocation: %v\n", perms)

	// Audit log
	fmt.Println("\n5. Audit Logging")
	authorizer.CheckAccess("user:123", "file:/home/user/doc.txt", "read")
	authorizer.CheckAccess("user:123", "system:/etc/passwd", "read")

	log := authorizer.GetAuditLog()
	fmt.Printf("   Audit log entries: %d\n", len(log))

	// Clear log
	authorizer.ClearAuditLog()
	log = authorizer.GetAuditLog()
	fmt.Printf("   Audit log after clear: %d entries\n", len(log))

	// Set default decision
	fmt.Println("\n6. Default Decision")
	authorizer.SetDefaultDecision(authz.DecisionAllowed)
	allowed = authorizer.IsAllowed("user:123", "unknown:resource", "action")
	fmt.Printf("   With default=ALLOW: %s\n", decisionString(allowed))

	authorizer.SetDefaultDecision(authz.DecisionDenied)
	allowed = authorizer.IsAllowed("user:123", "unknown:resource", "action")
	fmt.Printf("   With default=DENY: %s\n", decisionString(allowed))

	fmt.Println()
}

func decisionString(allowed bool) string {
	if allowed {
		return "ALLOWED"
	}
	return "DENIED"
}
