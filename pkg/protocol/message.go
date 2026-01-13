package protocol

import (
	"encoding/binary"
	"errors"
	"time"
)

// Protocol errors.
var (
	// ErrInvalidMagic is returned when magic bytes don't match.
	ErrInvalidMagic = errors.New("invalid magic bytes")
	// ErrInvalidVersion is returned when protocol version doesn't match.
	ErrInvalidVersion = errors.New("invalid protocol version")
	// ErrInvalidOpcode is returned when opcode is invalid.
	ErrInvalidOpcode = errors.New("invalid opcode")
	// ErrPayloadTooLarge is returned when payload exceeds maximum size.
	ErrPayloadTooLarge = errors.New("payload exceeds maximum size")
	// ErrBufferTooSmall is returned when buffer is too small.
	ErrBufferTooSmall = errors.New("buffer too small")
	// ErrInvalidTimestamp is returned when timestamp is invalid.
	ErrInvalidTimestamp = errors.New("invalid timestamp")
)

// Message represents a protocol message with an opcode, timestamp, and payload.
type Message struct {
	// Opcode is the message type identifier.
	Opcode Opcode
	// Timestamp is the Unix timestamp in nanoseconds.
	Timestamp int64
	// Payload is the message data.
	Payload []byte
}

// NewMessage creates a new message with the given opcode and payload.
// The timestamp is set to the current time in nanoseconds.
func NewMessage(opcode Opcode, payload []byte) *Message {
	return &Message{
		Opcode:    opcode,
		Timestamp: time.Now().UnixNano(),
		Payload:   payload,
	}
}

// Encode encodes the message to binary format.
// Returns a byte slice containing the encoded message.
func (m *Message) Encode() ([]byte, error) {
	if len(m.Payload) > int(MaxPayloadSize) {
		return nil, ErrPayloadTooLarge
	}

	totalSize := HeaderSize + len(m.Payload)
	buf := make([]byte, totalSize)

	// Write magic bytes at offset 0
	copy(buf[0:4], MagicBytes[:])

	// Write version at offset 4
	buf[4] = ProtocolVersion

	// Write opcode at offset 5
	buf[5] = byte(m.Opcode)

	// Write timestamp (8 bytes, big-endian) at offset 6
	binary.BigEndian.PutUint64(buf[6:14], uint64(m.Timestamp))

	// Write payload length (4 bytes, big-endian) at offset 14
	binary.BigEndian.PutUint32(buf[14:18], uint32(len(m.Payload)))

	// Write payload at offset 18
	if len(m.Payload) > 0 {
		copy(buf[HeaderSize:], m.Payload)
	}

	return buf, nil
}

// Decode decodes a message from binary format.
// The input data must contain at least the header bytes.
func (m *Message) Decode(data []byte) error {
	if len(data) < HeaderSize {
		return ErrBufferTooSmall
	}

	// Validate magic bytes
	if string(data[0:4]) != string(MagicBytes[:]) {
		return ErrInvalidMagic
	}

	// Validate version
	if data[4] != ProtocolVersion {
		return ErrInvalidVersion
	}

	// Read opcode
	m.Opcode = Opcode(data[5])
	if !m.Opcode.IsValid() {
		return ErrInvalidOpcode
	}

	// Read timestamp
	m.Timestamp = int64(binary.BigEndian.Uint64(data[6:14]))
	if m.Timestamp < 0 {
		return ErrInvalidTimestamp
	}

	// Read payload length
	payloadLen := binary.BigEndian.Uint32(data[14:18])
	if payloadLen > MaxPayloadSize {
		return ErrPayloadTooLarge
	}

	// Validate buffer size
	if uint32(len(data)-HeaderSize) < payloadLen {
		return ErrBufferTooSmall
	}

	// Read payload
	if payloadLen > 0 {
		m.Payload = make([]byte, payloadLen)
		copy(m.Payload, data[HeaderSize:HeaderSize+int(payloadLen)])
	} else {
		m.Payload = nil
	}

	return nil
}
