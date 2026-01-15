# WebOS - Web-Based Operating System

A web-based operating system written in Go, featuring a custom binary protocol, WebSocket communication, security foundation, and a JavaScript client.

## 🚀 Quick Start

```bash
# Start the web server
go run ./cmd/webos-server/main.go

# Open browser to http://localhost:8080
```

Or for development testing:

```bash
# Start the client demo server
go run ./cmd/client-demo/main.go

# Open browser to http://localhost:8080
```

## 📁 Project Structure

```
webos/
├── cmd/                          # Command-line applications
│   ├── protocol-demo/            # Protocol demonstration
│   ├── websocket-demo/           # WebSocket demonstration
│   ├── security-demo/            # Security demonstration
│   ├── webos-server/             # Main production server
│   └── client-demo/              # Client testing server
│
├── pkg/                          # Go packages
│   ├── protocol/                 # Binary protocol (88.8% coverage)
│   │   ├── doc.go                # Package documentation
│   │   ├── const.go              # Protocol constants
│   │   ├── opcode.go             # Opcode definitions
│   │   ├── message.go            # Message encoding/decoding
│   │   ├── codec.go              # Binary codec utilities
│   │   └── *_test.go             # Comprehensive tests
│   │
│   ├── websocket/                # WebSocket server (RFC 6455)
│   │   ├── doc.go                # Package documentation
│   │   ├── frame.go              # Frame types and constants
│   │   ├── frame_reader.go       # Frame parsing
│   │   ├── frame_writer.go       # Frame generation
│   │   ├── handshake.go          # HTTP upgrade handshake
│   │   ├── connection.go         # Connection handling
│   │   ├── session.go            # Session management
│   │   ├── pool.go               # Connection pooling
│   │   └── server.go             # WebSocket server
│   │
│   ├── security/                 # OpenBSD-inspired security (97.1% coverage)
│   │   ├── doc.go                # Package documentation
│   │   ├── pledge.go             # Capability promises
│   │   ├── unveil.go             # Filesystem restrictions
│   │   └── capabilities.go       # Security manager
│   │
│   ├── auth/                     # Authentication (86.4% coverage)
│   │   ├── doc.go                # Package documentation
│   │   ├── auth.go               # Authentication core
│   │   ├── password.go           # Password hashing (PBKDF2)
│   │   ├── session.go            # Session management
│   │   ├── token.go              # Token generation
│   │   └── mfa.go                # MFA (TOTP/HOTP)
│   │
│   ├── authz/                    # Authorization (92.8% coverage)
│   │   ├── doc.go                # Package documentation
│   │   ├── policy.go             # Policy engine
│   │   └── access.go             # Access control
│   │
│   ├── router/                   # HTTP router (88.8% coverage)
│   │   ├── doc.go                # Package documentation
│   │   ├── router.go             # Router implementation
│   │   ├── middleware.go         # Middleware chain
│   │   └── params.go             # URL parameters
│   │
│   └── server/                   # HTTP server
│       ├── doc.go                # Package documentation
│       ├── server.go             # Server implementation
│       ├── static.go             # Static file serving
│       └── tls.go                # TLS configuration
│
├── static/                       # Static files served to clients
│   ├── index.html                # Main HTML page
│   ├── js/
│   │   ├── doc.js                # JavaScript documentation
│   │   ├── protocol.js           # Protocol client
│   │   ├── connection.js         # WebSocket connection
│   │   ├── display.js            # Canvas rendering
│   │   ├── input.js              # Event capture
│   │   ├── shell.js              # UI shell
│   │   ├── state.js              # State management
│   │   └── client.js             # Client entry point
│   └── css/
│       └── style.css             # Client styling
│
└── docs/
    └── PROTOCOL_SPEC.md          # Protocol specification
```

## 🏗️ Architecture

### Communication Layer

```
┌─────────────────────────────────────────────────────────┐
│                     Browser Client                       │
│  ┌─────────┐ ┌──────────┐ ┌─────────┐ ┌──────────────┐ │
│  │Protocol │ │Connection│ │ Display │ │   Shell/UI   │ │
│  └────┬────┘ └────┬─────┘ └────┬────┘ └──────┬───────┘ │
│       │           │            │              │         │
│       └───────────┴─────┬──────┴──────────────┘         │
│                         │                                 │
└─────────────────────────┼─────────────────────────────────┘
                          │ WebSocket
                          ▼
┌─────────────────────────────────────────────────────────┐
│                   WebOS Backend (Go)                     │
│  ┌──────────┐ ┌───────────┐ ┌──────────┐ ┌───────────┐ │
│  │WebSocket │ │  Security │ │   HTTP   │ │  Router   │ │
│  │  Server  │ │  (AuthZ)  │ │  Server  │ │           │ │
│  └────┬─────┘ └─────┬─────┘ └────┬─────┘ └─────┬─────┘ │
│       │             │            │              │        │
│       └─────────────┴─────┬──────┴──────────────┘        │
│                           │                                 │
│              ┌────────────┴────────────┐                   │
│              │      Protocol Layer      │                   │
│              │   (Binary Encoding)      │                   │
│              └──────────────────────────┘                   │
└─────────────────────────────────────────────────────────┘
```

### Protocol Format

All messages use a binary format:

```
┌──────────┬─────────┬────────┬────────────┬──────────┬──────────┐
│  Magic   │ Version │ Opcode │ Timestamp  │  Length  │ Payload  │
│ (4 byte) │ (1 byte)│(1 byte)│  (8 byte)  │ (4 byte) │   N      │
└──────────┴─────────┴────────┴────────────┴──────────┴──────────┘
```

**Magic Bytes:** `WEBS` (0x57, 0x45, 0x42, 0x53)  
**Version:** 1  
**Opcodes:** Display, Input, FileSystem, Network, Process, Auth, Connect, Disconnect, Ping, Pong, Error

### Security Model

OpenBSD-inspired pledge/unveil system:

```go
// Capabilities (promises)
PromiseStdio      // Standard I/O
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
```

## 📦 Packages

### pkg/protocol
Binary protocol for client-server communication.

**Coverage:** 88.8%

```go
msg := protocol.NewMessage(protocol.OpcodeDisplay, []byte("data"))
encoded, err := msg.Encode()
```

### pkg/websocket
RFC 6455 compliant WebSocket server.

**Features:**
- Frame parsing/generation
- HTTP upgrade handshake
- Connection lifecycle management
- Session tracking
- Connection pooling

### pkg/security
OpenBSD-inspired security model.

**Coverage:** 97.1%

```go
capability := &security.Capability{
    Promises: security.PromiseRpath | security.PromiseInet,
    UnveilPaths: []security.UnveilPath{
        {Path: "/tmp", Permissions: "rw"},
    },
}
```

### pkg/auth
Authentication system.

**Coverage:** 86.4%

**Features:**
- PBKDF2 password hashing
- Session management
- Secure tokens (256-bit)
- TOTP/HOTP MFA

### pkg/authz
Authorization framework.

**Coverage:** 92.8%

```go
policy := authz.NewPolicy()
policy.AddRule("admin", "/*", "/*", authz.EffectAllow)
```

### pkg/router
Custom HTTP router.

**Coverage:** 88.8%

```go
router := router.New()
router.Get("/users/:id", handler)
router.Post("/api/v1/*", apiHandler)
```

### pkg/server
Production HTTP server.

**Features:**
- TLS 1.3 with strong cipher suites
- HTTP/2 support
- Static file serving
- Graceful shutdown

## 🧪 Testing

Run all tests:

```bash
# Run all package tests
go test ./... -v

# Run with race detection
go test ./... -race

# Check coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

Individual package tests:

```bash
go test ./pkg/protocol/ -v -cover
go test ./pkg/websocket/ -v -cover
go test ./pkg/security/ -v -cover
go test ./pkg/auth/ -v -cover
go test ./pkg/authz/ -v -cover
go test ./pkg/router/ -v -cover
go test ./pkg/server/ -v -cover
```

## 📊 Coverage Summary

| Package | Coverage |
|---------|----------|
| pkg/protocol | 88.8% |
| pkg/security | 97.1% |
| pkg/auth | 86.4% |
| pkg/authz | 92.8% |
| pkg/router | 88.8% |
| pkg/server | 55.6% |
| pkg/websocket | 45.5% |

## 🚀 Running the Server

### Development

```bash
# Run client demo server (serves static files only)
go run ./cmd/client-demo/main.go
# Visit http://localhost:8080
```

### Production

```bash
# Run full webos server
go run ./cmd/webos-server/main.go
# Visit http://localhost:8080
```

### Demos

```bash
# Protocol demo
go run ./cmd/protocol-demo/main.go

# WebSocket demo
go run ./cmd/websocket-demo/main.go

# Security demo
go run ./cmd/security-demo/main.go
```

## 🔧 Configuration

The server can be configured via environment variables:

```bash
WEBOS_PORT=8080           # Server port (default: 8080)
WEBOS_TLS_ENABLED=true    # Enable TLS (default: false)
WEBOS_CERT_FILE=cert.pem  # TLS certificate
WEBOS_KEY_FILE=key.pem    # TLS key
WEBOS_STATIC_DIR=./static # Static files directory
```

## 📝 API Reference

### Protocol Opcodes

| Opcode | Value | Description |
|--------|-------|-------------|
| OpcodeDisplay | 1 | Display rendering instructions |
| OpcodeInput | 2 | Keyboard/mouse input events |
| OpcodeFileSystem | 3 | File system operations |
| OpcodeNetwork | 4 | Network operations |
| OpcodeProcess | 5 | Process management |
| OpcodeAuth | 6 | Authentication |
| OpcodeConnect | 7 | Connection initiation |
| OpcodeDisconnect | 8 | Connection termination |
| OpcodePing | 9 | Keep-alive ping |
| OpcodePong | 10 | Keep-alive pong |
| OpcodeError | 11 | Error responses |

### HTTP Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | / | Main HTML page |
| GET | /health | Health check |
| GET | /ready | Readiness check |
| GET | /metrics | Metrics endpoint |
| WS | /ws | WebSocket upgrade |

## 🤝 Contributing

1. Read the AGENTS.md guidelines
2. Follow the phase prompts in prompts/
3. Write tests first (TDD approach)
4. Maintain 85%+ test coverage
5. Document all public APIs
6. No external dependencies

## 📄 License

This project is part of the WebOS development effort.

## 🔗 Related Documentation

- [Protocol Specification](docs/PROTOCOL_SPEC.md)
- [Phase Prompts](prompts/)
- [DEVPLAN.md](DEVPLAN.md)
- [PROJECT.md](PROJECT.md)
