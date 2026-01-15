package ipv4

import (
	"encoding/binary"
	"fmt"
)

// ICMP type and code constants.
const (
	// Types
	ICMPTypeEchoReply      uint8 = 0
	ICMPTypeDestUnreach    uint8 = 3
	ICMPTypeSourceQuench   uint8 = 4
	ICMPTypeRedirect       uint8 = 5
	ICMPTypeEcho           uint8 = 8
	ICMPTypeTimeExceeded   uint8 = 11
	ICMPTypeParamProblem   uint8 = 12
	ICMPTypeTimestamp      uint8 = 13
	ICMPTypeTimestampReply uint8 = 14
	ICMPTypeInfoRequest    uint8 = 15
	ICMPTypeInfoReply      uint8 = 16
)

// Codes for Destination Unreachable.
const (
	ICMPCodeNetUnreach     uint8 = 0
	ICMPCodeHostUnreach    uint8 = 1
	ICMPCodeProtoUnreach   uint8 = 2
	ICMPCodePortUnreach    uint8 = 3
	ICMPCodeFragNeeded     uint8 = 4
	ICMPCodeSrcRouteFailed uint8 = 5
)

// ICMPHeader represents an ICMP header.
type ICMPHeader struct {
	Type     uint8  // ICMP type
	Code     uint8  // ICMP code
	Checksum uint16 // Checksum
	ID       uint16 // Identifier (for echo requests/replies)
	Seq      uint16 // Sequence number (for echo requests/replies)
}

// Payload returns the ICMP payload (data after the header).
func (h *ICMPHeader) Payload(data []byte) []byte {
	headerLen := 8 // ICMP header is always 8 bytes
	if headerLen > len(data) {
		return nil
	}
	return data[headerLen:]
}

// ParseICMPHeader parses an ICMP header from raw bytes.
func ParseICMPHeader(data []byte) (*ICMPHeader, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("ICMP header too short: %d bytes", len(data))
	}

	return &ICMPHeader{
		Type:     data[0],
		Code:     data[1],
		Checksum: binary.BigEndian.Uint16(data[2:4]),
		ID:       binary.BigEndian.Uint16(data[4:6]),
		Seq:      binary.BigEndian.Uint16(data[6:8]),
	}, nil
}

// Serialize serializes the ICMP header to bytes.
func (h *ICMPHeader) Serialize() []byte {
	buf := make([]byte, 8)

	buf[0] = h.Type
	buf[1] = h.Code
	binary.BigEndian.PutUint16(buf[2:4], h.Checksum)
	binary.BigEndian.PutUint16(buf[4:6], h.ID)
	binary.BigEndian.PutUint16(buf[6:8], h.Seq)

	return buf
}

// CalcChecksum calculates the ICMP checksum.
func (h *ICMPHeader) CalcChecksum(data []byte) uint16 {
	sum := uint32(0)

	// Sum the header
	headerBytes := h.Serialize()
	for i := 0; i < len(headerBytes); i += 2 {
		if i+1 < len(headerBytes) {
			sum += uint32(headerBytes[i])<<8 | uint32(headerBytes[i+1])
		} else {
			sum += uint32(headerBytes[i]) << 8
		}
	}

	// Sum the data
	for i := 0; i < len(data); i += 2 {
		if i+1 < len(data) {
			sum += uint32(data[i])<<8 | uint32(data[i+1])
		} else {
			sum += uint32(data[i]) << 8
		}
	}

	for sum > 0xFFFF {
		sum = (sum >> 16) + (sum & 0xFFFF)
	}

	return ^uint16(sum)
}

// Message represents an ICMP message.
type Message struct {
	Header  *ICMPHeader
	Payload []byte
}

// ParseMessage parses an ICMP message from raw bytes.
func ParseMessage(data []byte) (*Message, error) {
	header, err := ParseICMPHeader(data)
	if err != nil {
		return nil, err
	}

	payload := header.Payload(data)

	return &Message{
		Header:  header,
		Payload: payload,
	}, nil
}

// Serialize serializes the ICMP message to bytes.
func (m *Message) Serialize() []byte {
	// Update checksum
	m.Header.Checksum = m.Header.CalcChecksum(m.Payload)

	// Build the full message
	msg := m.Header.Serialize()
	msg = append(msg, m.Payload...)

	return msg
}

// NewEchoRequest creates a new ICMP echo request (ping).
func NewEchoRequest(id, seq uint16, data []byte) *Message {
	return &Message{
		Header: &ICMPHeader{
			Type: ICMPTypeEcho,
			Code: 0,
			ID:   id,
			Seq:  seq,
		},
		Payload: data,
	}
}

// NewEchoReply creates a new ICMP echo reply.
func NewEchoReply(id, seq uint16, data []byte) *Message {
	return &Message{
		Header: &ICMPHeader{
			Type: ICMPTypeEchoReply,
			Code: 0,
			ID:   id,
			Seq:  seq,
		},
		Payload: data,
	}
}

// NewDestUnreach creates a destination unreachable message.
func NewDestUnreach(code uint8, origIPHdr []byte) *Message {
	return &Message{
		Header: &ICMPHeader{
			Type:     ICMPTypeDestUnreach,
			Code:     code,
			ID:       0,
			Seq:      0,
			Checksum: 0,
		},
		Payload: origIPHdr,
	}
}

// NewTimeExceeded creates a time exceeded message.
func NewTimeExceeded(origIPHdr []byte) *Message {
	return &Message{
		Header: &ICMPHeader{
			Type:     ICMPTypeTimeExceeded,
			Code:     0,
			ID:       0,
			Seq:      0,
			Checksum: 0,
		},
		Payload: origIPHdr,
	}
}

// IsEchoRequest returns true if the message is an echo request.
func (m *Message) IsEchoRequest() bool {
	return m.Header.Type == ICMPTypeEcho
}

// IsEchoReply returns true if the message is an echo reply.
func (m *Message) IsEchoReply() bool {
	return m.Header.Type == ICMPTypeEchoReply
}

// IsDestUnreach returns true if the message is a destination unreachable.
func (m *Message) IsDestUnreach() bool {
	return m.Header.Type == ICMPTypeDestUnreach
}

// IsTimeExceeded returns true if the message is a time exceeded.
func (m *Message) IsTimeExceeded() bool {
	return m.Header.Type == ICMPTypeTimeExceeded
}
