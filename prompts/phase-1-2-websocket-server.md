# PHASE 1.2: WebSocket Server & Connection Management

**Phase Context**: Phase 1 builds the communication foundation between browser and backend. This sub-phase builds on Phase 1.1 (Protocol) to implement WebSocket communication.

**Sub-Phase Objective**: Implement a custom WebSocket server using Go's `net` package (RFC 6455 compliant), handle client connections, implement session management, and create connection pooling.

**Prerequisites**: 
- Phase 1.1 (Custom Protocol) must be complete
- `pkg/protocol` package must exist with Message types and encode/decode functions
- Understanding of RFC 6455 WebSocket protocol

**Integration Point**: This will be used by the HTTP server to upgrade connections to WebSocket and handle real-time protocol communication.

---

## IMPLEMENTATION REQUIREMENTS

### Overview

You are implementing a custom WebSocket server from scratch using only Go's standard library. This includes the HTTP handshake, frame parsing/generation, connection lifecycle management, and session tracking.

**Key Constraints**:
- ✅ Use ONLY Go standard library (no external dependencies like gorilla/websocket)
- ✅ Must be RFC 6455 compliant
- ✅ Must integrate with `pkg/protocol` for message encoding/decoding
- ✅ Must handle concurrent connections safely
- ✅ Must include comprehensive tests

---

### Directory Structure

Create the following structure:

```
webos/
├── pkg/
│   ├── websocket/
│   │   ├── doc.go              # Package documentation
│   │   ├── frame.go            # Frame types and constants
│   │   ├── frame_reader.go     # Frame reading logic
│   │   ├── frame_writer.go     # Frame writing logic
│   │   ├── handshake.go        # HTTP upgrade handshake
│   │   ├── connection.go       # Connection handling
│   │   ├── session.go          # Session management
│   │   ├── pool.go             # Connection pooling
│   │   └── websocket_test.go   # Comprehensive tests
│   └── protocol/
│       └── (existing from Phase 1.1)
└── cmd/
    └── websocket-demo/
        └── main.go             # Demonstration program
```

---

### Core Types and Interfaces

```go
package websocket

import (
    "net"
    "sync"
    "time"
    
    "webos/pkg/protocol"
)

// Opcode represents WebSocket frame opcodes per RFC 6455
type Opcode uint8

const (
    OpcodeContinuation Opcode = 0x0
    OpcodeText         Opcode = 0x1
    OpcodeBinary       Opcode = 0x2
    OpcodeClose        Opcode = 0x8
    OpcodePing         Opcode = 0x9
    OpcodePong         Opcode = 0xA
)

// Frame represents a WebSocket frame
type Frame struct {
    Fin     bool     // Final fragment flag
    RSV1    bool     // Reserved bit 1
    RSV2    bool     // Reserved bit 2
    RSV3    bool     // Reserved bit 3
    Opcode  Opcode   // Frame opcode
    Masked  bool     // Mask flag
    Mask    [4]byte  // Masking key
    Payload []byte   // Frame payload
}

// Connection represents a WebSocket connection
type Connection struct {
    ID        string
    Conn      net.Conn
    Session   *Session
    CreatedAt time.Time
    LastPing  time.Time
    mu        sync.Mutex
}

// Session represents a user session
type Session struct {
    ID        string
    UserID    string
    Connected time.Time
    ExpiresAt time.Time
    Data      map[string]interface{}
}

// ConnectionManager manages all connections
type ConnectionManager struct {
    connections sync.Map // map[string]*Connection
    sessions    sync.Map // map[string]*Session
    maxConns    int
    maxSessions int
    timeout     time.Duration
    pingInterval time.Duration
}

// Server represents the WebSocket server
type Server struct {
    addr         string
    manager      *ConnectionManager
    upgrader     *Upgrader
    handler      MessageHandler
}

// MessageHandler handles incoming messages
type MessageHandler func(conn *Connection, msg *protocol.Message) error

// Errors
var (
    ErrInvalidFrame        = errors.New("invalid frame")
    ErrInvalidOpcode       = errors.New("invalid opcode")
    ErrFrameTooLarge       = errors.New("frame too large")
    ErrControlFrameTooLong = errors.New("control frame payload too long")
    ErrFragmentedControl   = errors.New("control frames cannot be fragmented")
    ErrConnectionClosed    = errors.New("connection closed")
    ErrConnectionLimit     = errors.New("connection limit reached")
    ErrSessionLimit        = errors.New("session limit reached")
)
```

---

### Implementation Steps

#### STEP 1: Frame Types and Constants

**Purpose**: Define frame structure and all RFC 6455 constants

**Implementation**:

Create `pkg/websocket/frame.go` with frame types, opcodes, validation logic, and error definitions.

**Requirements**:
- Define all opcodes per RFC 6455
- Create Frame struct with all required fields
- Implement validation logic
- Define error types

**Validation**:
```bash
go build ./pkg/websocket/
go vet ./pkg/websocket/
```

---

#### STEP 2: Frame Reading

**Purpose**: Implement RFC 6455 compliant frame parsing

**Implementation**:

Create `pkg/websocket/frame_reader.go` with frame reading logic including:
- Variable length payload decoding (7, 7+16, 7+64 bit lengths)
- Mask handling (client-to-server frames must be masked)
- Frame validation

**Requirements**:
- Handle all frame types (text, binary, close, ping, pong)
- Support fragmented messages
- Validate masking
- Handle frame length correctly

**Validation**:
```bash
go test ./pkg/websocket/ -run TestFrameRead -v
```

---

#### STEP 3: Frame Writing

**Purpose**: Implement RFC 6455 compliant frame generation

**Implementation**:

Create `pkg/websocket/frame_writer.go` with frame writing logic including:
- Variable length payload encoding
- Mask generation (for server-to-client frames)
- Frame header construction

**Requirements**:
- Generate valid frames for all opcodes
- Handle masking correctly
- Support large payloads

**Validation**:
```bash
go test ./pkg/websocket/ -run TestFrameWrite -v
```

---

#### STEP 4: HTTP Upgrade Handshake

**Purpose**: Implement RFC 6455 WebSocket handshake

**Implementation**:

Create `pkg/websocket/handshake.go` with:
- HTTP request parsing
- Sec-WebSocket-Key validation
- Sec-WebSocket-Accept generation
- Response header construction

**Requirements**:
- Validate handshake headers
- Support both HTTP/1.0 and HTTP/1.1
- Handle invalid handshakes gracefully
- Support subprotocol negotiation

**Validation**:
```bash
go test ./pkg/websocket/ -run TestHandshake -v
```

---

#### STEP 5: Connection and Session Management

**Purpose**: Implement connection handling and session lifecycle

**Implementation**:

Create `pkg/websocket/connection.go` and `session.go` with:
- Connection read/write loops
- Heartbeat mechanism
- Session creation and expiration
- Connection cleanup

**Requirements**:
- Handle concurrent connections safely
- Implement ping/pong for keepalive
- Session timeout handling
- Graceful shutdown

**Validation**:
```bash
go test ./pkg/websocket/ -run TestConnection -v
go test ./pkg/websocket/ -race
```

---

#### STEP 6: Connection Pool

**Purpose**: Implement connection pooling for resource management

**Implementation**:

Create `pkg/websocket/pool.go` with:
- Connection tracking
- Rate limiting
- Connection limits per IP
- Pool statistics

**Requirements**:
- Track active connections
- Enforce connection limits
- Provide pool statistics
- Thread-safe operations

**Validation**:
```bash
go test ./pkg/websocket/ -run TestPool -v
```

---

#### STEP 7: Server Implementation

**Purpose**: Implement the main WebSocket server

**Implementation**:

Create `pkg/websocket/server.go` with:
- TCP listener setup
- Connection accepting
- Protocol upgrade handling
- Message routing

**Requirements**:
- Accept incoming connections
- Perform protocol upgrade
- Route messages to handler
- Handle errors gracefully

**Validation**:
```bash
go test ./pkg/websocket/ -run TestServer -v
```

---

### Testing Requirements

**Test Coverage**: Minimum 85%

**Required Test Cases**:

1. **Frame Operations**:
   - Frame encoding/decoding round-trip
   - Fragmented message handling
   - Control frame validation

2. **Handshake**:
   - Valid handshake request
   - Invalid key rejection
   - Subprotocol negotiation

3. **Connection Management**:
   - Multiple concurrent connections
   - Connection timeout
   - Graceful disconnect

4. **Error Cases**:
   - Invalid frame data
   - Mask violations
   - Protocol violations

5. **Concurrency**:
   - Race condition testing
   - Connection pool thread safety

**Test Implementation**:
```go
func TestFrameRoundTrip(t *testing.T) {
    // Test frame encode/decode consistency
}

func TestHandshakeValid(t *testing.T) {
    // Test valid WebSocket handshake
}

func TestConnectionConcurrency(t *testing.T) {
    // Test concurrent connection handling
}
```

---

### Demonstration Program

Create `cmd/websocket-demo/main.go` that demonstrates:

1. **Server Startup**: Start WebSocket server on a port
2. **Client Connection**: Simulate client connections
3. **Message Exchange**: Send/receive protocol messages
4. **Connection Pool**: Show pool statistics
5. **Session Management**: Demonstrate session lifecycle

**Expected Output**:
```
WebSocket Server Demo
=====================

1. Starting server on :8080...
   Server started successfully

2. Simulating client connections...
   Client 1 connected: session-abc123
   Client 2 connected: session-def456
   Pool size: 2 connections

3. Sending messages...
   Client 1: Message sent
   Client 2: Message received

4. Testing disconnect...
   Client 1 disconnected
   Pool size: 1 connection

5. Session statistics...
   Active sessions: 1
   Total connections: 2
   Failed handshakes: 0

Demo completed successfully!
```

---

### Validation Checklist

- [ ] All code compiles without warnings
- [ ] `go vet ./pkg/websocket/...` passes
- [ ] Test coverage ≥85%
- [ ] No race conditions: `go test ./pkg/websocket/ -race`
- [ ] Demo program runs successfully
- [ ] RFC 6455 compliant handshake
- [ ] Connection pooling working
- [ ] Session management functional

---

### Next Sub-Phase

After completing this sub-phase, proceed to:
**PHASE 1.3**: Security Foundation

**What it will build on**:
- WebSocket connection handling
- Session management
- Message framing
