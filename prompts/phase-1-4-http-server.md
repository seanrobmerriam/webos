# PHASE 1.4: Basic HTTP Server & Routing

**Phase Context**: Phase 1 builds the communication foundation. This sub-phase implements the HTTP server layer that serves the client and handles API requests.

**Sub-Phase Objective**: Implement HTTP/1.1 and HTTP/2 server, static file serving, custom routing engine, and TLS configuration.

**Prerequisites**: 
- Phase 1.3 (Security) recommended

**Integration Point**: HTTP server will serve the JavaScript client and provide WebSocket upgrade endpoint.

---

## IMPLEMENTATION REQUIREMENTS

### Overview

You are implementing a production-ready HTTP server using Go's `net/http` with custom routing, middleware support, and TLS.

---

### Directory Structure

```
webos/
├── cmd/
│   └── webos-server/
│       └── main.go             # Main server entry point
├── pkg/
│   ├── router/
│   │   ├── doc.go              # Package documentation
│   │   ├── router.go           # Router implementation
│   │   ├── middleware.go       # Middleware chain
│   │   ├── params.go           # URL parameter extraction
│   │   └── router_test.go      # Tests
│   └── server/
│       ├── doc.go              # Package documentation
│       ├── server.go           # HTTP server
│       ├── static.go           # Static file serving
│       ├── tls.go              # TLS configuration
│       └── server_test.go      # Tests
└── static/
    └── (client files served here)
```

---

### Core Types

```go
package router

// Route represents an HTTP route
type Route struct {
    Method      string
    Pattern     string
    Handler     http.Handler
    Middleware  []Middleware
    Params      []string
}

// Router is a custom HTTP router
type Router struct {
    routes      map[string][]Route
    middleware  []Middleware
    notFound    http.Handler
    methodNotAllowed http.Handler
}

// Middleware is HTTP middleware function
type Middleware func(http.Handler) http.Handler

// Params holds URL parameter values
type Params map[string]string

// ContextKey is type for context keys
type ContextKey string
```

---

### Implementation Steps

1. **Router Implementation**: Pattern matching, parameter extraction, method dispatch
2. **Middleware Chain**: Request/response processing chain
3. **Static File Serving**: MIME types, caching headers, range requests
4. **TLS Configuration**: Certificate handling, strong cipher suites
5. **HTTP/2 Support**: ALPN negotiation, stream handling
6. **Request Logging**: Access logs, metrics collection

---

### Testing Requirements

- Route matching correctness
- Parameter extraction
- Middleware execution order
- Static file serving
- TLS handshake

---

### Next Sub-Phase

**PHASE 1.5**: Client Foundation (JavaScript)

---

## Deliverables

- `cmd/webos-server/main.go` - Main server
- `pkg/router/` - Custom router
- `pkg/server/` - HTTP server implementation
- Static file serving
- TLS configuration
- 85%+ test coverage
