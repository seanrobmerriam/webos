# WebOS

A custom binary protocol implementation for web-based operating system communication, written in Go.

## Overview

WebOS is a lightweight, efficient binary communication protocol designed for real-time communication between browser clients and Go servers. It features a simple message-based structure with operation codes (opcodes) to identify message types, making it ideal for building responsive web applications and services.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      WebOS Architecture                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐   │
│  │   Client    │◄───►│   Server    │◄───►│   Storage   │   │
│  │  (Browser)  │     │   (Go)      │     │   Layer     │   │
│  └─────────────┘     └─────────────┘     └─────────────┘   │
│         │                   │                   │           │
│         └───────────────────┴───────────────────┘           │
│                         │                                   │
│              ┌──────────▼──────────┐                       │
│              │  WebOS Protocol     │                       │
│              │  (Binary Messages)  │                       │
│              └─────────────────────┘                       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Project Structure

```
webos/
├── cmd/
│   └── protocol-demo/          # Demo application
│       └── main.go
├── pkg/
│   └── protocol/               # Core protocol package
│       ├── doc.go              # Package documentation
│       ├── opcode.go           # Opcode definitions
│       ├── message.go          # Message encoding/decoding
│       └── message_test.go     # Protocol tests
└── go.mod                      # Go module definition
```

## Protocol Format

The WebOS protocol uses a compact binary message format optimized for performance and ease of parsing.

### Message Structure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Message Header (10 bytes)                       │
├──────────┬──────────┬──────────┬───────────────────────────────────────────┤
│  Magic   │ Version  │  Opcode  │              Payload Length                │
│ (4 byte) │ (1 byte) │ (1 byte) │                (4 bytes)                   │
└──────────┴──────────┴──────────┴───────────────────────────────────────────┘
         │              │              │
         │              │              │
         │              │              └─► Big-endian uint32 (0 to 16MB)
         │              └─► Protocol version (currently 1)
         └─► "WEMS" (0x57 0x45 0x4D 0x53)
```

### Field Details

| Field | Size | Description |
|-------|------|-------------|
| Magic | 4 bytes | Protocol magic bytes: `0x57454D53` ("WEMS") |
| Version | 1 byte | Protocol version (currently `1`) |
| Opcode | 1 byte | Message type identifier |
| Length | 4 bytes | Payload length in bytes (big-endian uint32) |
| Payload | Variable | Message data (max 16 MB) |

### Protocol Constants

```go
const (
    MagicBytes     = [4]byte{0x57, 0x45, 0x4D, 0x53}  // "WEMS"
    ProtocolVersion = 1
    HeaderSize      = 10 bytes
    MaxPayloadSize  = 16 * 1024 * 1024  // 16 MB
    DefaultBufferSize = 256 bytes
)
```

### Example Usage

```go
package main

import (
    "log"
    "github.com/yourusername/webos/pkg/protocol"
)

func main() {
    // Create a text message
    msg := &protocol.Message{
        Opcode:  protocol.OpText,
        Payload: []byte("Hello, WebOS!"),
    }

    // Encode the message
    data, err := msg.Encode()
    if err != nil {
        log.Fatal(err)
    }

    // Decode the message
    var decoded protocol.Message
    if err := decoded.Decode(data); err != nil {
        log.Fatal(err)
    }

    // Use the decoded message
    println(string(decoded.Payload))
}
```

## Opcode Reference

| Opcode | Value | Name | Description |
|--------|-------|------|-------------|
| `OpConnect` | 1 | CONNECT | Client connection initiation |
| `OpDisconnect` | 2 | DISCONNECT | Client disconnection |
| `OpText` | 3 | TEXT | Text-based messages (UTF-8) |
| `OpBinary` | 4 | BINARY | Raw binary data transfer |
| `OpPing` | 5 | PING | Heartbeat check |
| `OpPong` | 6 | PONG | Heartbeat response |
| `OpError` | 7 | ERROR | Error message |
| `OpAck` | 8 | ACK | Delivery acknowledgment |
| `OpSubscribe` | 9 | SUBSCRIBE | Channel/topic subscription |
| `OpUnsubscribe` | 10 | UNSUBSCRIBE | Channel/topic unsubscription |
| `OpPublish` | 11 | PUBLISH | Publish to channel/topic |
| `OpRequest` | 12 | REQUEST | Request-response pattern |
| `OpResponse` | 13 | RESPONSE | Request response |
| `OpAuth` | 14 | AUTH | Authentication credentials |
| `OpAuthResult` | 15 | AUTH_RESULT | Authentication result |
| `OpSync` | 16 | SYNC | State synchronization |
| `OpHeartbeat` | 17 | HEARTBEAT | Periodic heartbeat |

## Installation

### Prerequisites

- Go 1.21 or higher

### Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/webos.git
   cd webos
   ```

2. Navigate to the project directory:
   ```bash
   cd webos
   ```

3. Verify the module:
   ```bash
   go mod tidy
   ```

## Testing

Run the protocol tests to verify the implementation:

```bash
cd webos
go test ./pkg/protocol/ -v
```

### Test Coverage

The test suite includes:

- **Message encoding/decoding** - Verifies correct serialization and deserialization
- **Invalid magic detection** - Ensures malformed messages are rejected
- **Version validation** - Checks protocol version enforcement
- **Truncated message handling** - Tests incomplete message detection
- **Payload size limits** - Validates max payload size enforcement
- **Buffer management** - Tests buffer allocation and overflow protection
- **Opcode string conversion** - Verifies opcode string representation
- **Benchmark tests** - Measures encode/decode performance

### Run Benchmarks

```bash
go test ./pkg/protocol/ -bench=.
```

Example output:
```
BenchmarkEncode-8    1000000    1200 ns/op
BenchmarkDecode-8    1500000     800 ns/op
```

## Running the Demo

The demo program demonstrates the protocol's encoding/decoding capabilities:

```bash
go run ./cmd/protocol-demo/
```

### Demo Features

1. **Message Encoding** - Shows how to create and encode different message types
2. **Message Decoding** - Demonstrates decoding messages from binary data
3. **Error Handling** - Tests invalid magic bytes and truncated messages
4. **Opcode Display** - Lists all available opcodes
5. **Determinism Testing** - Verifies encoding/decoding consistency

### Demo Output

```
WebOS Protocol Demo
===================

1. Creating and encoding messages...
   Connect message encoded: 58 bytes
   Text message encoded: 24 bytes
   Binary message encoded: 266 bytes

2. Decoding messages...
   Decoded: Message{Opcode: CONNECT, PayloadSize: 48}
   Payload: {"clientId": "web-client-123", "version": "1.0"}
   Decoded: Message{Opcode: TEXT, PayloadSize: 14}
   Payload: Hello, WebOS!

3. Testing error handling...
   Invalid magic error: invalid magic bytes
   Truncated message error: truncated message

4. Available opcodes:
    1: CONNECT
    2: DISCONNECT
   ...

5. Testing encode/decode determinism...
   ✓ Encoding is deterministic (same input = same output)
   ✓ Decoding is deterministic

Demo completed successfully!
```

## Current Phase Status

### Phase 1.1.1 - Protocol Foundation ✅ Completed

- [x] Define binary message format specification
- [x] Implement message encoding/decoding logic
- [x] Define protocol opcodes (17 total)
- [x] Create protocol package documentation
- [x] Write comprehensive unit tests
- [x] Build demo application

## Next Steps

### Phase 1.1.2 - Connection Management (Planned)

- [ ] Implement connection handler
- [ ] Add client connection lifecycle management
- [ ] Create connection state tracking
- [ ] Implement graceful disconnect handling

### Phase 1.1.3 - Message Router (Planned)

- [ ] Build message routing system
- [ ] Implement opcode-based message handlers
- [ ] Add message queuing for high throughput
- [ ] Create error response handling

### Phase 1.1.4 - Client Implementation (Planned)

- [ ] Develop JavaScript client library
- [ ] Implement WebSocket-like API
- [ ] Add automatic reconnection
- [ ] Create connection state management

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Built with Go for high performance and reliability
- Designed for seamless browser-to-server communication
- Inspired by WebSocket and other real-time protocols
