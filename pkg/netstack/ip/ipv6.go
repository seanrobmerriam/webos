package ipv4

import (
	"encoding/binary"
	"fmt"
	network "net"
)

// IPv6 constants.
const (
	IPv6HeaderLength = 40 // Fixed header length for IPv6
	IPv6Version      = 6
)

// IPv6 extension header types.
const (
	IPv6ExtHeaderHopByHop     uint8 = 0
	IPv6ExtHeaderDestination  uint8 = 60
	IPv6ExtHeaderRouting      uint8 = 43
	IPv6ExtHeaderFragment     uint8 = 44
	IPv6ExtHeaderAuth         uint8 = 51
	IPv6ExtHeaderESP          uint8 = 50
	IPv6ExtHeaderNoNextHeader uint8 = 59
)

// IPv6Header represents an IPv6 header.
type IPv6Header struct {
	Version      uint8  // IP version (6)
	TrafficClass uint8  // Traffic class
	FlowLabel    uint32 // Flow label
	PayloadLen   uint16 // Payload length
	NextHeader   uint8  // Next header (extension header or upper layer)
	HopLimit     uint8  // Hop limit (TTL)
	SrcIP        network.IP
	DstIP        network.IP
}

// Payload returns the packet payload (data after the header).
func (h *IPv6Header) Payload(data []byte) []byte {
	if IPv6HeaderLength > len(data) {
		return nil
	}
	return data[IPv6HeaderLength:]
}

// ParseIPv6Header parses an IPv6 header from raw bytes.
func ParseIPv6Header(data []byte) (*IPv6Header, error) {
	if len(data) < IPv6HeaderLength {
		return nil, fmt.Errorf("IPv6 header too short: %d bytes", len(data))
	}

	h := &IPv6Header{
		Version:      data[0] >> 4,
		TrafficClass: (data[0]&0x0F)<<4 | data[1]>>4,
		FlowLabel:    uint32(data[1]&0x0F)<<16 | uint32(data[2])<<8 | uint32(data[3]),
		PayloadLen:   binary.BigEndian.Uint16(data[4:6]),
		NextHeader:   data[6],
		HopLimit:     data[7],
	}

	h.SrcIP = make(network.IP, 16)
	h.DstIP = make(network.IP, 16)
	copy(h.SrcIP, data[8:16])
	copy(h.DstIP, data[16:24])

	return h, nil
}

// Serialize serializes the IPv6 header to bytes.
func (h *IPv6Header) Serialize() []byte {
	buf := make([]byte, IPv6HeaderLength)

	buf[0] = (h.Version << 4) | (h.TrafficClass >> 4)
	buf[1] = (h.TrafficClass&0x0F)<<4 | byte(h.FlowLabel>>16&0x0F)
	buf[2] = byte(h.FlowLabel >> 8)
	buf[3] = byte(h.FlowLabel)
	binary.BigEndian.PutUint16(buf[4:6], h.PayloadLen)
	buf[6] = h.NextHeader
	buf[7] = h.HopLimit
	copy(buf[8:16], h.SrcIP)
	copy(buf[16:24], h.DstIP)

	return buf
}

// CalcChecksum calculates the pseudo-header checksum for upper layer protocols.
func (h *IPv6Header) CalcChecksum(upperProtocol uint8, upperLen int) uint16 {
	// IPv6 pseudo-header for checksum calculation
	sum := uint32(0)

	// Source address (as two 64-bit words)
	for i := 0; i < 16; i += 2 {
		sum += uint32(h.SrcIP[i])<<8 | uint32(h.SrcIP[i+1])
	}

	// Destination address (as two 64-bit words)
	for i := 0; i < 16; i += 2 {
		sum += uint32(h.DstIP[i])<<8 | uint32(h.DstIP[i+1])
	}

	// Upper layer packet length
	sum += uint32(upperLen >> 16)
	sum += uint32(upperLen & 0xFFFF)

	// Zero padding and next header
	sum += uint32(0)
	sum += uint32(upperProtocol)

	for sum > 0xFFFF {
		sum = (sum >> 16) + (sum & 0xFFFF)
	}

	return ^uint16(sum)
}

// Datagram represents a complete IPv6 datagram.
type IPv6Datagram struct {
	Header  *IPv6Header
	Payload []byte
}

// ParseIPv6Datagram parses an IPv6 datagram from raw bytes.
func ParseIPv6Datagram(data []byte) (*IPv6Datagram, error) {
	header, err := ParseIPv6Header(data)
	if err != nil {
		return nil, err
	}

	payload := header.Payload(data)
	if payload == nil {
		return nil, fmt.Errorf("invalid payload")
	}

	return &IPv6Datagram{
		Header:  header,
		Payload: payload,
	}, nil
}

// Serialize serializes the datagram to bytes.
func (d *IPv6Datagram) Serialize() []byte {
	// Update payload length
	d.Header.PayloadLen = uint16(len(d.Payload))

	// Build the full packet
	packet := d.Header.Serialize()
	packet = append(packet, d.Payload...)

	return packet
}

// NewIPv6Datagram creates a new IPv6 datagram.
func NewIPv6Datagram(srcIP, dstIP network.IP, nextHeader uint8, payload []byte) *IPv6Datagram {
	h := &IPv6Header{
		Version:      6,
		TrafficClass: 0,
		FlowLabel:    0,
		PayloadLen:   uint16(len(payload)),
		NextHeader:   nextHeader,
		HopLimit:     64,
		SrcIP:        srcIP,
		DstIP:        dstIP,
	}

	return &IPv6Datagram{
		Header:  h,
		Payload: payload,
	}
}

// IPv6FragmentHeader represents the IPv6 fragment extension header.
type IPv6FragmentHeader struct {
	NextHeader  uint8
	Reserved    uint8
	FragmentOff uint16 // Fragment offset (in 8-byte units)
	MoreFrag    uint8  // More fragments flag
	Ident       uint32 // Identification
}

// ParseIPv6FragmentHeader parses a fragment header.
func ParseIPv6FragmentHeader(data []byte) (*IPv6FragmentHeader, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("IPv6 fragment header too short")
	}

	return &IPv6FragmentHeader{
		NextHeader:  data[0],
		Reserved:    data[1],
		FragmentOff: binary.BigEndian.Uint16(data[2:4]) & 0xFFF8,
		MoreFrag:    data[3],
		Ident:       binary.BigEndian.Uint32(data[4:8]),
	}, nil
}

// Serialize serializes the IPv6 fragment header.
func (h *IPv6FragmentHeader) Serialize() []byte {
	buf := make([]byte, 8)

	buf[0] = h.NextHeader
	buf[1] = h.Reserved
	binary.BigEndian.PutUint16(buf[2:4], h.FragmentOff)
	buf[3] = h.MoreFrag
	binary.BigEndian.PutUint32(buf[4:8], h.Ident)

	return buf
}

// IsFragment returns true if more fragments follow.
func (h *IPv6FragmentHeader) IsFragment() bool {
	return h.MoreFrag != 0
}

// IPv6HopByHopHeader represents the IPv6 hop-by-hop options extension header.
type IPv6HopByHopHeader struct {
	NextHeader uint8
	HdrLen     uint8
	Options    []byte
}

// ParseIPv6HopByHopHeader parses a hop-by-hop options header.
func ParseIPv6HopByHopHeader(data []byte) (*IPv6HopByHopHeader, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("IPv6 hop-by-hop header too short")
	}

	hdrLen := int(data[1]+1) * 8
	if len(data) < hdrLen {
		return nil, fmt.Errorf("IPv6 hop-by-hop options too short")
	}

	return &IPv6HopByHopHeader{
		NextHeader: data[0],
		HdrLen:     data[1],
		Options:    data[2:hdrLen],
	}, nil
}
