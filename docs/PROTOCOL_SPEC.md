# WebOS Binary Protocol Specification

## Overview

This document defines the binary protocol for WebOS client-server communication. The protocol is designed for efficient, low-latency communication between a browser-based JavaScript client and a Go backend server.

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1 | 2024-01-13 | Initial specification |

## Message Format

All protocol messages follow a fixed 18-byte header structure:

```
+---+---+---+---+---+---+---+---+---+---+---+---+---+---+---+---+---+
| 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 |10 |11 |12 |13 |14 |15 |...|
+---+---+---+---+---+---+---+---+---+---+---+---+---+---+---+---+---+
|Magic Bytes    |Ver|Opc|        Timestamp (Unix nanoseconds)        |
+---------------+---+---+--------------------------------------------+
|Payload Length |              Payload Data                      |
+---------------+-----------------------------------------------+
```

### Header Structure (18 bytes total)

| Offset | Size | Field | Description |
|--------|------|-------|-------------|
| 0 | 4 | Magic Bytes | Protocol identifier: "WEBS" (0x57, 0x45, 0x42, 0x53) |
| 4 | 1 | Version | Protocol version (current: 1) |
| 5 | 1 | Opcode | Message type identifier (see Opcodes section) |
| 6 | 8 | Timestamp | Unix timestamp in nanoseconds (big-endian) |
| 14 | 4 | Payload Length | Length of payload in bytes (big-endian) |
| 18 | N | Payload | Variable-length message payload |

### Byte Order

All multi-byte fields use big-endian (network byte order) encoding:
- Timestamp: 8 bytes, big-endian uint64
- Payload Length: 4 bytes, big-endian uint32

### Maximum Payload Size

The maximum payload size is 16 MB (16,777,216 bytes). Messages exceeding this size will be rejected with an error.

## Opcodes

The following opcodes are defined for message type identification:

| Opcode | Value | Direction | Description |
|--------|-------|-----------|-------------|
| OpcodeInvalid | 0 | N/A | Invalid/unspecified opcode |
| OpcodeDisplay | 1 | Server→Client | Display rendering instructions |
| OpcodeInput | 2 | Client→Server | Keyboard/mouse input events |
| OpcodeFileSystem | 3 | Bidirectional | File system operations |
| OpcodeNetwork | 4 | Bidirectional | Network operations |
| OpcodeProcess | 5 | Bidirectional | Process management |
| OpcodeAuth | 6 | Bidirectional | Authentication messages |
| OpcodeConnect | 7 | Client→Server | Connection establishment |
| OpcodeDisconnect | 8 | Bidirectional | Connection termination |
| OpcodePing | 9 | Bidirectional | Keep-alive ping |
| OpcodePong | 10 | Bidirectional | Keep-alive pong response |
| OpcodeError | 11 | Bidirectional | Error messages |

### Opcode Details

#### OpcodeDisplay (1)
Server-to-client messages for rendering the display. Payload contains display commands in JSON format.

Example payload:
```json
{
  "type": "draw_rect",
  "x": 100,
  "y": 200,
  "width": 50,
  "height": 30,
  "color": "#FF0000"
}
```

#### OpcodeInput (2)
Client-to-server messages for user input events. Payload contains input data in JSON format.

Example payload:
```json
{
  "type": "keydown",
  "key": "a",
  "code": "KeyA",
  "shift": false,
  "ctrl": false
}
```

#### OpcodeFileSystem (3)
File system operation requests and responses. Uses a binary or JSON payload depending on operation type.

#### OpcodeNetwork (4)
Network operation messages for HTTP requests, WebSocket proxies, etc.

#### OpcodeProcess (5)
Process management messages for spawning and controlling processes.

#### OpcodeAuth (6)
Authentication handshake messages. Used during connection establishment.

#### OpcodeConnect (7)
Initial connection message sent by client to establish a session.

#### OpcodeDisconnect (8)
Connection termination message. Sent by either party before closing the connection.

#### OpcodePing (9)
Keep-alive ping message. Client or server can send to verify connection is alive.

#### OpcodePong (10)
Response to ping message. Contains the same timestamp as the ping.

#### OpcodeError (11)
Error message sent when an operation fails.

## Message Lifecycle

### Connection Establishment

1. Client opens WebSocket connection to server
2. Client sends OpcodeConnect message with client identification
3. Server responds with OpcodeAuth challenge (if authentication required)
4. Client sends OpcodeAuth response
5. Server sends OpcodeConnect response ( OpcodeDisplay or OpcodeError)

### Normal Communication

- Client sends OpcodeInput for user interactions
- Server sends OpcodeDisplay for rendering updates
- Either party can send OpcodePing; recipient responds with OpcodePong

### Connection Termination

1. Either party sends OpcodeDisconnect
2. Connection is closed after a short grace period

## Example Messages

### Example 1: Connect Message

```
Header:
  Magic: 57 45 42 53 ("WEBS")
  Version: 01
  Opcode: 07 (CONNECT)
  Timestamp: 0000000000000001 (1 nanosecond since epoch)
  Payload Length: 0000002A (42 bytes)

Payload: {"clientId":"web-client-123","version":"1.0","platform":"browser"}
```

### Example 2: Display Message

```
Header:
  Magic: 57 45 42 53 ("WEBS")
  Version: 01
  Opcode: 01 (DISPLAY)
  Timestamp: 0000000000000001
  Payload Length: 0000002B (43 bytes)

Payload: {"type":"text","x":10,"y":20,"text":"Hello, WebOS!","color":"#000000"}
```

### Example 3: Input Message

```
Header:
  Magic: 57 45 42 53 ("WEBS")
  Version: 01
  Opcode: 02 (INPUT)
  Timestamp: 0000000000000001
  Payload Length: 0000001A (26 bytes)

Payload: {"type":"keydown","key":"Enter","code":"Enter"}
```

## Error Handling

### Protocol Errors

| Error | Code | Description |
|-------|------|-------------|
| ErrInvalidMagic | -1 | Magic bytes do not match "WEBS" |
| ErrInvalidVersion | -2 | Protocol version mismatch |
| ErrInvalidOpcode | -3 | Unknown or unsupported opcode |
| ErrPayloadTooLarge | -4 | Payload exceeds 16MB limit |
| ErrBufferTooSmall | -5 | Message buffer too short |
| ErrInvalidTimestamp | -6 | Timestamp value invalid |

### Error Message Format

Error messages use OpcodeError with a JSON payload:

```json
{
  "code": -1,
  "message": "invalid magic bytes",
  "details": "expected 0x57454253, got 0x57454254"
}
```

## Security Considerations

1. **Magic Bytes Validation**: Always validate magic bytes before processing
2. **Payload Size Limits**: Enforce 16MB limit to prevent memory exhaustion
3. **Input Sanitization**: Validate and sanitize all payload data
4. **Authentication**: Use OpcodeAuth for session authentication
5. **Connection Limits**: Implement connection rate limiting

## Implementation Notes

### Go Implementation

- Use `encoding/binary` for big-endian encoding
- Allocate buffers with exact size to minimize memory usage
- Use `time.Now().UnixNano()` for timestamps

### JavaScript Implementation

- Use `DataView` for multi-byte integer handling
- Use `Uint8Array` for binary data
- Use `BigInt` for nanosecond timestamp handling

## References

- [RFC 7946 - WebSocket Protocol](https://tools.ietf.org/html/rfc7946)
- [Go encoding/binary package](https://pkg.go.dev/encoding/binary)
- [JavaScript Typed Arrays](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/TypedArray)
