// Package websocket provides frame types and constants for RFC 6455 compliant
// WebSocket frame handling.
package websocket

import (
	"errors"
)

// Opcode represents WebSocket frame opcodes per RFC 6455.
type Opcode uint8

// Frame opcodes as defined in RFC 6455 Section 5.2.
const (
	// OpcodeContinuation indicates a continuation frame.
	OpcodeContinuation Opcode = 0x0
	// OpcodeText indicates a text frame.
	OpcodeText Opcode = 0x1
	// OpcodeBinary indicates a binary frame.
	OpcodeBinary Opcode = 0x2
	// OpcodeClose indicates a close frame.
	OpcodeClose Opcode = 0x8
	// OpcodePing indicates a ping frame.
	OpcodePing Opcode = 0x9
	// OpcodePong indicates a pong frame.
	OpcodePong Opcode = 0xA
)

// IsValid checks if the opcode is a valid WebSocket opcode.
func (o Opcode) IsValid() bool {
	switch o {
	case OpcodeContinuation, OpcodeText, OpcodeBinary,
		OpcodeClose, OpcodePing, OpcodePong:
		return true
	default:
		return false
	}
}

// IsControl checks if the opcode is a control frame opcode.
func (o Opcode) IsControl() bool {
	switch o {
	case OpcodeClose, OpcodePing, OpcodePong:
		return true
	default:
		return false
	}
}

// IsData checks if the opcode is a data frame opcode.
func (o Opcode) IsData() bool {
	switch o {
	case OpcodeText, OpcodeBinary, OpcodeContinuation:
		return true
	default:
		return false
	}
}

// String returns the string representation of the opcode.
func (o Opcode) String() string {
	switch o {
	case OpcodeContinuation:
		return "CONTINUATION"
	case OpcodeText:
		return "TEXT"
	case OpcodeBinary:
		return "BINARY"
	case OpcodeClose:
		return "CLOSE"
	case OpcodePing:
		return "PING"
	case OpcodePong:
		return "PONG"
	default:
		return "UNKNOWN"
	}
}

// Frame represents a WebSocket frame as defined in RFC 6455 Section 5.2.
type Frame struct {
	// Fin indicates whether this is the final fragment of a message.
	Fin bool
	// RSV1, RSV2, RSV3 are reserved bits for extensions.
	RSV1 bool
	RSV2 bool
	RSV3 bool
	// Opcode identifies the type of frame.
	Opcode Opcode
	// Masked indicates whether the payload is masked.
	Masked bool
	// Mask is the masking key (4 bytes).
	Mask [4]byte
	// Payload contains the frame's payload data.
	Payload []byte
}

// Frame errors as defined in RFC 6455 Section 7.4.1.
var (
	// ErrInvalidFrame is returned when a frame is malformed.
	ErrInvalidFrame = errors.New("invalid frame")
	// ErrInvalidOpcode is returned when an unknown opcode is encountered.
	ErrInvalidOpcode = errors.New("invalid opcode")
	// ErrFrameTooLarge is returned when a frame exceeds the maximum size.
	ErrFrameTooLarge = errors.New("frame too large")
	// ErrControlFrameTooLong is returned when a control frame is too long.
	ErrControlFrameTooLong = errors.New("control frame payload too long")
	// ErrFragmentedControl is returned when a control frame is fragmented.
	ErrFragmentedControl = errors.New("control frames cannot be fragmented")
	// ErrConnectionClosed is returned when the connection is closed.
	ErrConnectionClosed = errors.New("connection closed")
	// ErrConnectionLimit is returned when the connection limit is reached.
	ErrConnectionLimit = errors.New("connection limit reached")
	// ErrSessionLimit is returned when the session limit is reached.
	ErrSessionLimit = errors.New("session limit reached")
	// ErrInvalidMask is returned when a frame has an invalid mask.
	ErrInvalidMask = errors.New("invalid frame mask")
	// ErrReservedBitsSet is returned when reserved bits are set improperly.
	ErrReservedBitsSet = errors.New("reserved bits set without extension")
)

// Frame size limits per RFC 6455.
const (
	// MaxControlPayloadSize is the maximum payload size for control frames (125 bytes).
	MaxControlPayloadSize = 125
	// MaxFramePayloadSize is the maximum payload size for data frames.
	// We use a conservative default of 16MB to match the protocol package.
	MaxFramePayloadSize = 16 * 1024 * 1024
	// MaxFrameSize is the maximum total frame size including header.
	MaxFrameSize = MaxFramePayloadSize + 14
)

// Validate validates the frame according to RFC 6455 rules.
func (f *Frame) Validate() error {
	// Check opcode validity
	if !f.Opcode.IsValid() {
		return &FrameError{Err: ErrInvalidOpcode, Opcode: f.Opcode}
	}

	// Control frames cannot be fragmented
	if f.Opcode.IsControl() && !f.Fin {
		return &FrameError{Err: ErrFragmentedControl, Opcode: f.Opcode}
	}

	// Control frames have a maximum payload of 125 bytes
	if f.Opcode.IsControl() && len(f.Payload) > MaxControlPayloadSize {
		return &FrameError{Err: ErrControlFrameTooLong, Opcode: f.Opcode}
	}

	// Check payload size
	if len(f.Payload) > MaxFramePayloadSize {
		return &FrameError{Err: ErrFrameTooLarge, Opcode: f.Opcode}
	}

	// Reserved bits should not be set unless using extensions
	if f.RSV1 || f.RSV2 || f.RSV3 {
		return &FrameError{Err: ErrReservedBitsSet, Opcode: f.Opcode}
	}

	return nil
}

// FrameError represents a frame validation error.
type FrameError struct {
	Err    error
	Opcode Opcode
}

func (e *FrameError) Error() string {
	return e.Err.Error()
}

func (e *FrameError) Unwrap() error {
	return e.Err
}

// NewFrame creates a new frame with the given parameters.
func NewFrame(opcode Opcode, payload []byte, fin bool) *Frame {
	return &Frame{
		Fin:     fin,
		Opcode:  opcode,
		Payload: payload,
	}
}
