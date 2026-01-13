// Package protocol implements the WebOS binary communication protocol.
//
// This package provides the core messaging protocol for communication
// between the browser client and Go backend services. The protocol
// uses a compact binary format with the following structure:
//
//   - Magic Bytes: 4 bytes ("WEBS")
//   - Version: 1 byte
//   - Opcode: 1 byte (message type)
//   - Timestamp: 8 bytes (Unix nanoseconds)
//   - Payload Length: 4 bytes
//   - Payload: N bytes
//
// # Usage
//
// Creating and encoding a message:
//
//	msg := protocol.NewMessage(protocol.OpcodeDisplay, []byte("data"))
//	encoded, err := msg.Encode()
//
// Decoding a message:
//
//	var msg protocol.Message
//	err := msg.Decode(data)
//	if err != nil {
//	    // handle error
//	}
//
// # Architecture
//
// The protocol is designed for:
//   - Low-latency communication
//   - Efficient binary encoding
//   - Cross-platform compatibility (Go and JavaScript)
//
// # Protocol Constants
//
//   - MagicBytes: [4]byte{'W', 'E', 'B', 'S'}
//   - ProtocolVersion: 1
//   - HeaderSize: 18 bytes
//   - MaxPayloadSize: 16 MB
//
// # Examples
//
// See cmd/protocol-demo/main.go for complete examples.
package protocol
