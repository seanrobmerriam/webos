package ipv4

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	network "net"
)

// IPv4 header length in bytes (without options).
const HeaderLength = 20

// Protocol numbers.
const (
	ProtocolICMP uint8 = 1
	ProtocolTCP  uint8 = 6
	ProtocolUDP  uint8 = 17
)

// Header represents an IPv4 header.
type Header struct {
	Version    uint8  // IP version (4)
	IHL        uint8  // Internet Header Length (number of 32-bit words)
	TOS        uint8  // Type of Service
	Length     uint16 // Total length of the datagram
	ID         uint16 // Identification
	Flags      uint8  // Fragment flags
	FragOffset uint16 // Fragment offset
	TTL        uint8  // Time to Live
	Protocol   uint8  // Upper layer protocol
	Checksum   uint16 // Header checksum
	SrcIP      network.IP
	DstIP      network.IP
	Options    []byte // IP options (if IHL > 5)
}

// ParseHeader parses an IPv4 header from raw bytes.
func ParseHeader(data []byte) (*Header, error) {
	if len(data) < HeaderLength {
		return nil, fmt.Errorf("IPv4 header too short: %d bytes", len(data))
	}

	h := &Header{
		Version:    data[0] >> 4,
		IHL:        data[0] & 0x0F,
		TOS:        data[1],
		Length:     binary.BigEndian.Uint16(data[2:4]),
		ID:         binary.BigEndian.Uint16(data[4:6]),
		Flags:      data[6] >> 5,
		FragOffset: binary.BigEndian.Uint16(data[6:8]) & 0x1FFF,
		TTL:        data[8],
		Protocol:   data[9],
		Checksum:   binary.BigEndian.Uint16(data[10:12]),
		SrcIP:      network.IP{data[12], data[13], data[14], data[15]},
		DstIP:      network.IP{data[16], data[17], data[18], data[19]},
	}

	// Parse options if present
	if h.IHL > 5 {
		optLen := int(h.IHL-5) * 4
		if len(data) < HeaderLength+optLen {
			return nil, fmt.Errorf("IPv4 options too short")
		}
		h.Options = data[HeaderLength : HeaderLength+optLen]
	}

	return h, nil
}

// Serialize serializes the IPv4 header to bytes.
func (h *Header) Serialize() []byte {
	buf := make([]byte, h.Length)

	buf[0] = (h.Version << 4) | (h.IHL & 0x0F)
	buf[1] = h.TOS
	binary.BigEndian.PutUint16(buf[2:4], h.Length)
	binary.BigEndian.PutUint16(buf[4:6], h.ID)
	frag := (uint16(h.Flags) << 5) | (h.FragOffset & 0x1FFF)
	binary.BigEndian.PutUint16(buf[6:8], frag)
	buf[8] = h.TTL
	buf[9] = h.Protocol
	binary.BigEndian.PutUint16(buf[10:12], h.Checksum)
	copy(buf[12:16], []byte(h.SrcIP))
	copy(buf[16:20], []byte(h.DstIP))

	if len(h.Options) > 0 {
		copy(buf[20:], h.Options)
	}

	return buf
}

// CalcChecksum calculates the IPv4 header checksum.
func (h *Header) CalcChecksum() uint16 {
	sum := uint32(0)
	buf := h.Serialize()

	// Zero out checksum field for calculation
	buf[10] = 0
	buf[11] = 0

	for i := 0; i < len(buf); i += 2 {
		if i+1 < len(buf) {
			sum += uint32(buf[i])<<8 | uint32(buf[i+1])
		} else {
			sum += uint32(buf[i]) << 8
		}
	}

	for sum > 0xFFFF {
		sum = (sum >> 16) + (sum & 0xFFFF)
	}

	return ^uint16(sum)
}

// Payload returns the packet payload (data after the header).
func (h *Header) Payload(data []byte) []byte {
	headerLen := int(h.IHL) * 4
	if headerLen > len(data) {
		return nil
	}
	return data[headerLen:]
}

// IsFragment returns true if the packet is a fragment.
func (h *Header) IsFragment() bool {
	return h.Flags&0x1 != 0 || h.FragOffset != 0
}

// IsFirstFragment returns true if this is the first fragment.
func (h *Header) IsFirstFragment() bool {
	return h.FragOffset == 0
}

// IsLastFragment returns true if this is the last fragment.
func (h *Header) IsLastFragment() bool {
	return h.Flags&0x1 == 0
}

// Datagram represents a complete IPv4 datagram.
type Datagram struct {
	Header  *Header
	Payload []byte
}

// ParseDatagram parses an IPv4 datagram from raw bytes.
func ParseDatagram(data []byte) (*Datagram, error) {
	header, err := ParseHeader(data)
	if err != nil {
		return nil, err
	}

	payload := header.Payload(data)
	if payload == nil {
		return nil, fmt.Errorf("invalid payload")
	}

	return &Datagram{
		Header:  header,
		Payload: payload,
	}, nil
}

// Serialize serializes the datagram to bytes.
func (d *Datagram) Serialize() []byte {
	// Update header length and checksum
	headerLenBytes := int(d.Header.IHL) * 4
	d.Header.Length = uint16(headerLenBytes + len(d.Payload))
	d.Header.Checksum = d.Header.CalcChecksum()

	// Build the full packet
	packet := d.Header.Serialize()
	copy(packet[len(d.Header.Serialize()):], d.Payload)

	return packet
}

// NewDatagram creates a new IPv4 datagram.
func NewDatagram(srcIP, dstIP network.IP, protocol uint8, payload []byte) *Datagram {
	ihl := 5 // No options by default
	headerLen := ihl * 4

	h := &Header{
		Version:    4,
		IHL:        uint8(ihl),
		TOS:        0,
		Length:     uint16(headerLen + len(payload)),
		ID:         0,
		Flags:      0,
		FragOffset: 0,
		TTL:        64,
		Protocol:   protocol,
		SrcIP:      srcIP,
		DstIP:      dstIP,
	}

	return &Datagram{
		Header:  h,
		Payload: payload,
	}
}

// Fragmentation constants.
const (
	MaxDatagramSize   = 65535
	MinFragmentSize   = 8 // Minimum fragment data size (must be multiple of 8)
	DefaultMTU        = 1500
	FragmentOffsetMul = 8 // Fragment offset is in 8-byte units
)

// Fragment fragments a datagram into smaller pieces.
func Fragment(d *Datagram, mtu int) ([]*Datagram, error) {
	headerLen := int(d.Header.IHL) * 4
	maxPayload := mtu - headerLen

	// Ensure max payload is a multiple of 8 (except for last fragment)
	maxPayload = (maxPayload / FragmentOffsetMul) * FragmentOffsetMul
	if maxPayload < MinFragmentSize {
		return nil, fmt.Errorf("MTU too small for fragmentation: %d", mtu)
	}

	totalPayload := len(d.Payload)
	var fragments []*Datagram

	offset := 0
	fragmentID := d.Header.ID
	if fragmentID == 0 {
		fragmentID = uint16(rand.Uint32())
	}

	for offset < totalPayload {
		chunkSize := maxPayload
		isLast := false

		if offset+chunkSize >= totalPayload {
			chunkSize = totalPayload - offset
			isLast = true
		}

		var flags uint8
		if !isLast {
			flags = 0x1 // More fragments flag
		}

		fragPayload := make([]byte, chunkSize)
		copy(fragPayload, d.Payload[offset:offset+chunkSize])

		fragHdr := &Header{
			Version:    4,
			IHL:        5,
			TOS:        d.Header.TOS,
			Length:     uint16(headerLen + chunkSize),
			ID:         fragmentID,
			Flags:      flags,
			FragOffset: uint16(offset / FragmentOffsetMul),
			TTL:        d.Header.TTL,
			Protocol:   d.Header.Protocol,
			SrcIP:      d.Header.SrcIP,
			DstIP:      d.Header.DstIP,
		}

		fragments = append(fragments, &Datagram{
			Header:  fragHdr,
			Payload: fragPayload,
		})

		offset += chunkSize
	}

	return fragments, nil
}

// Reassemble reassembles fragmented datagrams into a complete datagram.
func Reassemble(fragments []*Datagram) (*Datagram, error) {
	if len(fragments) == 0 {
		return nil, fmt.Errorf("no fragments provided")
	}

	first := fragments[0]
	last := fragments[len(fragments)-1]

	// Check that this is indeed the last fragment
	if last.Header.Flags&0x1 != 0 {
		return nil, fmt.Errorf("incomplete fragment set: last fragment has more-fragments flag set")
	}

	// Verify all fragments have same ID, src, dst
	id := first.Header.ID
	srcIP := first.Header.SrcIP
	dstIP := first.Header.DstIP

	for _, frag := range fragments {
		if frag.Header.ID != id {
			return nil, fmt.Errorf("fragment ID mismatch")
		}
		if !frag.Header.SrcIP.Equal(srcIP) {
			return nil, fmt.Errorf("source IP mismatch")
		}
		if !frag.Header.DstIP.Equal(dstIP) {
			return nil, fmt.Errorf("destination IP mismatch")
		}
	}

	// Calculate total payload size
	totalPayload := 0
	for _, frag := range fragments {
		totalPayload += len(frag.Payload)
	}

	// Verify fragment offsets and order
	expectedOffset := 0
	for _, frag := range fragments {
		if int(frag.Header.FragOffset)*FragmentOffsetMul != expectedOffset {
			return nil, fmt.Errorf("fragment offset gap or overlap at offset %d", expectedOffset)
		}
		expectedOffset += len(frag.Payload)
	}

	// Reassemble payload
	payload := make([]byte, totalPayload)
	offset := 0
	for _, frag := range fragments {
		copy(payload[offset:], frag.Payload)
		offset += len(frag.Payload)
	}

	// Use first fragment's header as template
	reassembled := &Datagram{
		Header: &Header{
			Version:    first.Header.Version,
			IHL:        first.Header.IHL,
			TOS:        first.Header.TOS,
			Length:     uint16(int(first.Header.IHL)*4 + totalPayload),
			ID:         first.Header.ID,
			Flags:      0,
			FragOffset: 0,
			TTL:        first.Header.TTL,
			Protocol:   first.Header.Protocol,
			SrcIP:      first.Header.SrcIP,
			DstIP:      first.Header.DstIP,
		},
		Payload: payload,
	}

	return reassembled, nil
}
