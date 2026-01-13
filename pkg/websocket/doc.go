// Package websocket provides a RFC 6455 compliant WebSocket server implementation.
// This package includes frame parsing/generation, HTTP upgrade handshake,
// connection lifecycle management, and session tracking.
//
// The package integrates with the webos/pkg/protocol package for message
// encoding/decoding, enabling real-time bidirectional communication between
// browser clients and the Go backend.
//
// # Usage
//
// To create a WebSocket server:
//
//	s := websocket.NewServer(":8080", websocket.WithMaxConnections(100))
//	if err := s.Start(); err != nil {
//	    log.Fatal(err)
//	}
package websocket

/*
   WebSocket Frame Format (RFC 6455):

   0                   1                   2                   3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
  +-+-+-+-+-------+-+-------------+-------------------------------+
  |F|R|R|R| opcode|R| Payload len |    Extended payload length    |
  |I|S|S|S|  (4)  |S|     (7)     |             (16/64)           |
  |N|V|V|V|       |V|             |   (if payload len==126/127)   |
  | |1|2|3|       |4|             |                               |
  +-+-+-+-+-------+-+-------------+-------------------------------+
  |     Extended payload length continued, if payload len == 127  |
  +---------------------------------------------------------------+
  |                               | Masking-key, if MASK set to 1 |
  +-------------------------------+-------------------------------+
  | Masking-key (continued)       |          Payload Data         |
  +-------------------------------+-------------------------------+
  |                     Payload Data continued ...                |
  +---------------------------------------------------------------+
  |                     Payload Data continued ...                |
  +---------------------------------------------------------------+
*/
