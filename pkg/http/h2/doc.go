/*
Package h2 provides HTTP/2 framing layer implementation.

This package implements the HTTP/2 binary framing layer as specified in RFC 7540.
It includes:

  - Frame types: DATA, HEADERS, PRIORITY, RST_STREAM, SETTINGS, PUSH_PROMISE,
    PING, GOAWAY, WINDOW_UPDATE, CONTINUATION
  - Stream management
  - HPACK header compression

# HTTP/2 Overview

HTTP/2 is a binary protocol that uses frames for all communication.
The key concepts are:

  - Connections: TCP connections that carry multiple streams
  - Streams: Bidirectional sequences of frames between client and server
  - Frames: The smallest unit of communication (9 bytes header + payload)
  - Messages: Complete HTTP requests/responses (one or more frames)

# Usage

This package is typically used through the higher-level http package which
handles HTTP/2 transparently when the protocol is negotiated via ALPN.

# Frame Structure

All frames have a 9-byte header:

  - 3 bytes: Length (payload size, not including the 9 header bytes)
  - 1 byte: Type (DATA, HEADERS, SETTINGS, etc.)
  - 1 byte: Flags (type-specific flags)
  - 4 bytes: Stream Identifier (31 bits used, high bit is reserved)

Example frame header:

	00 00 0a 01 05 00 00 00 01
	|   |   | | |   |       |
	|   |   | | |   +------- Stream ID: 1
	|   |   | | +----------- Flags: END_HEADERS (0x04) | END_STREAM (0x01)
	|   |   | +------------- Type: HEADERS (0x01)
	|   |   +--------------- Length: 10 bytes
	+---------------------- Payload starts here
*/
package h2
