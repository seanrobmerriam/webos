// Package ethernet provides Ethernet frame parsing and generation.
package ethernet

import (
	"encoding/binary"
	"fmt"
	"net"

	"webos/pkg/netstack"
)

// Ethernet header length in bytes.
const HeaderLength = 14

// Frame represents an Ethernet frame.
type Frame struct {
	DstMAC    net.HardwareAddr   // Destination MAC address (6 bytes)
	SrcMAC    net.HardwareAddr   // Source MAC address (6 bytes)
	EtherType netstack.EtherType // EtherType field
	Payload   []byte             // Frame payload (IP packet, ARP, etc.)
	FCS       uint32             // Frame Check Sequence (CRC-32)
}

// ParseFrame parses an Ethernet frame from raw bytes.
func ParseFrame(data []byte) (*Frame, error) {
	if len(data) < HeaderLength {
		return nil, fmt.Errorf("frame too short: %d bytes", len(data))
	}

	frame := &Frame{
		DstMAC:    net.HardwareAddr{data[0], data[1], data[2], data[3], data[4], data[5]},
		SrcMAC:    net.HardwareAddr{data[6], data[7], data[8], data[9], data[10], data[11]},
		EtherType: netstack.EtherType(binary.BigEndian.Uint16(data[12:14])),
	}

	frame.Payload = data[14:]

	if len(data) >= HeaderLength+4 {
		frame.FCS = binary.BigEndian.Uint32(data[len(data)-4:])
	}

	return frame, nil
}

// Serialize serializes the Ethernet frame to bytes.
func (f *Frame) Serialize() []byte {
	buf := make([]byte, HeaderLength+len(f.Payload))
	copy(buf[0:6], f.DstMAC)
	copy(buf[6:12], f.SrcMAC)
	binary.BigEndian.PutUint16(buf[12:14], uint16(f.EtherType))
	copy(buf[14:], f.Payload)
	return buf
}

// IsBroadcast checks if the destination MAC is broadcast.
func (f *Frame) IsBroadcast() bool {
	for _, b := range f.DstMAC {
		if b != 0xFF {
			return false
		}
	}
	return true
}

// IsMulticast checks if the destination MAC is multicast.
func (f *Frame) IsMulticast() bool {
	return f.DstMAC[0]&0x01 == 0x01
}

// IsUnicast checks if the frame is unicast.
func (f *Frame) IsUnicast() bool {
	return !f.IsBroadcast() && !f.IsMulticast()
}

// Checksum calculates the CRC-32 checksum of the frame.
func (f *Frame) Checksum() uint32 {
	return crc32Checksum(f.Serialize())
}

// crc32Checksum computes CRC-32 checksum.
func crc32Checksum(data []byte) uint32 {
	var crc uint32 = 0xFFFFFFFF
	for _, b := range data {
		crc ^= uint32(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xEDB88320
			} else {
				crc >>= 1
			}
		}
	}
	return ^crc
}

// NewFrame creates a new Ethernet frame.
func NewFrame(dstMAC, srcMAC net.HardwareAddr, etherType netstack.EtherType, payload []byte) *Frame {
	return &Frame{
		DstMAC:    dstMAC,
		SrcMAC:    srcMAC,
		EtherType: etherType,
		Payload:   payload,
	}
}

// BroadcastMAC returns the Ethernet broadcast MAC address.
func BroadcastMAC() net.HardwareAddr {
	return net.HardwareAddr{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
}

// ParseMAC parses a MAC address string.
func ParseMAC(s string) (net.HardwareAddr, error) {
	mac := make(net.HardwareAddr, 6)
	j := 0
	for _, c := range s {
		if c == ':' {
			continue
		}
		if j >= 6 {
			return nil, fmt.Errorf("MAC address too long")
		}
		mac[j] = byte(c)
		j++
	}
	if j != 6 {
		return nil, fmt.Errorf("invalid MAC address: %s", s)
	}
	return mac, nil
}
