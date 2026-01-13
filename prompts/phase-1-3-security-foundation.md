# PHASE 1.3: Security Foundation

**Phase Context**: Phase 1 builds the communication foundation. This sub-phase implements OpenBSD-inspired security model with pledge/unveil capability system.

**Sub-Phase Objective**: Implement capability-based security, authentication system, session management, and authorization framework.

**Prerequisites**: 
- Phase 1.1 (Protocol) must be complete
- Phase 1.2 (WebSocket) recommended but not required

**Integration Point**: Security will be integrated into all subsequent components, enforcing capability restrictions at each layer.

---

## IMPLEMENTATION REQUIREMENTS

### Overview

You are implementing OpenBSD-inspired security model:
- **Pledge System**: Capability-based restrictions on what operations a component can perform
- **Unveil System**: Filesystem visibility restrictions
- **Authentication**: Password hashing, session tokens, MFA support
- **Authorization**: Capability-based access control

---

### Directory Structure

```
webos/
├── pkg/
│   ├── security/
│   │   ├── doc.go              # Package documentation
│   │   ├── pledge.go           # Capability promises
│   │   ├── unveil.go           # Filesystem restrictions
│   │   ├── capabilities.go     # Capability management
│   │   └── security_test.go    # Tests
│   ├── auth/
│   │   ├── doc.go              # Package documentation
│   │   ├── auth.go             # Authentication core
│   │   ├── password.go         # Password hashing
│   │   ├── session.go          # Session management
│   │   ├── token.go            # Token generation
│   │   ├── mfa.go              # Multi-factor auth
│   │   └── auth_test.go        # Tests
│   └── authz/
│       ├── doc.go              # Package documentation
│       ├── policy.go           # Policy engine
│       ├── access.go           # Access control
│       └── authz_test.go       # Tests
└── cmd/
    └── security-demo/
        └── main.go             # Demonstration program
```

---

### Core Types

```go
package security

// Promise represents capability permissions (inspired by OpenBSD pledge)
type Promise uint64

const (
    PromiseStdio Promise = 1 << iota
    PromiseRpath      // Read path access
    PromiseWpath      // Write path access
    PromiseInet       // Internet access
    PromiseUnix       // Unix domain sockets
    PromiseFork       // Process forking
    PromiseExec       // Program execution
    PromiseSignal     // Signal handling
    PromiseTimer      // Timer access
    PromiseAudio      // Audio access
    PromiseVideo      // Video access
    PromiseSocket     // Generic socket access
    PromiseResolve    // DNS resolution
)

// UnveilPath represents a filesystem path with permissions
type UnveilPath struct {
    Path        string
    Permissions string // "r", "w", "x", "rw"
}

// Capability represents a security capability
type Capability struct {
    Promises    Promise
    UnveilPaths []UnveilPath
    Timeout     time.Duration
}

// SecurityManager manages security policies
type SecurityManager struct {
    activeCaps sync.Map // map[component]*Capability
    policies   map[string]Promise
}
```

---

### Implementation Steps

1. **Pledge System**: Implement Promise type, capability checking, promise enforcement
2. **Unveil System**: Implement filesystem path restrictions, permission checking
3. **Authentication**: Implement bcrypt password hashing, session tokens, login/logout
4. **Session Management**: Implement session creation, expiration, secure cookies
5. **Authorization**: Implement policy engine, capability-based access control
6. **Audit Logging**: Implement security event logging

---

### Testing Requirements

- Password hashing verification
- Session token uniqueness
- Capability enforcement
- Unauthorized access rejection
- Audit log completeness

---

### Next Sub-Phase

**PHASE 1.4**: Basic HTTP Server & Routing

---

## Deliverables

- `pkg/security/` - Pledge/unveil implementation
- `pkg/auth/` - Authentication system
- `pkg/authz/` - Authorization framework
- `cmd/security-demo/main.go` - Demo program
- Comprehensive tests with 85%+ coverage
