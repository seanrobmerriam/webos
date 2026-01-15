package udp

import (
	"encoding/binary"
	"fmt"
	network "net"
)

// Header represents a UDP header.
type Header struct {
	SrcPort  uint16 // Source port
	DstPort  uint16 // Destination port
	Length   uint16 // Length of the datagram
	Checksum uint16 // Checksum
}

// Payload returns the datagram payload (data after the header).
func (h *Header) Payload(data []byte) []byte {
	if 8 > len(data) {
		return nil
	}
	return data[8:]
}

// ParseHeader parses a UDP header from raw bytes.
func ParseHeader(data []byte) (*Header, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("UDP header too short: %d bytes", len(data))
	}

	return &Header{
		SrcPort:  binary.BigEndian.Uint16(data[0:2]),
		DstPort:  binary.BigEndian.Uint16(data[2:4]),
		Length:   binary.BigEndian.Uint16(data[4:6]),
		Checksum: binary.BigEndian.Uint16(data[6:8]),
	}, nil
}

// Serialize serializes the UDP header to bytes.
func (h *Header) Serialize() []byte {
	buf := make([]byte, 8)

	binary.BigEndian.PutUint16(buf[0:2], h.SrcPort)
	binary.BigEndian.PutUint16(buf[2:4], h.DstPort)
	binary.BigEndian.PutUint16(buf[4:6], h.Length)
	binary.BigEndian.PutUint16(buf[6:8], h.Checksum)

	return buf
}

// CalcChecksum calculates the UDP checksum using pseudo-header.
func (h *Header) CalcChecksum(srcIP, dstIP network.IP, payload []byte) uint16 {
	// UDP pseudo-header
	sum := uint32(0)

	// Source IP
	for i := 0; i < 16; i += 2 {
		sum += uint32(srcIP[i])<<8 | uint32(srcIP[i+1])
	}

	// Destination IP
	for i := 0; i < 16; i += 2 {
		sum += uint32(dstIP[i])<<8 | uint32(dstIP[i+1])
	}

	// Protocol (17 for UDP) and UDP length
	sum += 17
	sum += uint32(len(h.Serialize()) + len(payload))

	// Sum header and payload
	data := append(h.Serialize(), payload...)
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

	// Checksum of zero means no checksum was computed
	if sum == 0 {
		return 0
	}

	return ^uint16(sum)
}

// Datagram represents a complete UDP datagram.
type Datagram struct {
	Header  *Header
	SrcIP   network.IP
	DstIP   network.IP
	Payload []byte
}

// ParseDatagram parses a UDP datagram from raw bytes.
func ParseDatagram(data []byte, srcIP, dstIP network.IP) (*Datagram, error) {
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
		SrcIP:   srcIP,
		DstIP:   dstIP,
		Payload: payload,
	}, nil
}

// Serialize serializes the datagram to bytes.
func (d *Datagram) Serialize() []byte {
	// Update length
	d.Header.Length = uint16(8 + len(d.Payload))

	// Update checksum
	d.Header.Checksum = d.Header.CalcChecksum(d.SrcIP, d.DstIP, d.Payload)

	// Build full datagram
	datagram := d.Header.Serialize()
	datagram = append(datagram, d.Payload...)

	return datagram
}

// NewDatagram creates a new UDP datagram.
func NewDatagram(srcPort, dstPort uint16, srcIP, dstIP network.IP, payload []byte) *Datagram {
	return &Datagram{
		Header: &Header{
			SrcPort: srcPort,
			DstPort: dstPort,
			Length:  uint16(8 + len(payload)),
		},
		SrcIP:   srcIP,
		DstIP:   dstIP,
		Payload: payload,
	}
}

// Socket represents a UDP socket.
type Socket struct {
	Port    uint16
	Addr    network.IP
	Payload chan *Datagram
}

// NewSocket creates a new UDP socket.
func NewSocket(port uint16, addr network.IP) *Socket {
	return &Socket{
		Port:    port,
		Addr:    addr,
		Payload: make(chan *Datagram, 100),
	}
}

// Send sends a datagram.
func (s *Socket) Send(d *Datagram) error {
	select {
	case s.Payload <- d:
		return nil
	default:
		return fmt.Errorf("socket buffer full")
	}
}

// Receive receives a datagram.
func (s *Socket) Receive() (*Datagram, error) {
	select {
	case d := <-s.Payload:
		return d, nil
	default:
		return nil, fmt.Errorf("no data available")
	}
}
