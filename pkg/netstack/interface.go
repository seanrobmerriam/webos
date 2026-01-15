package netstack

import (
	"net"
	"time"
)

// Protocol represents the network protocol type.
type Protocol uint8

// Network protocol constants.
const (
	ProtocolICMP   Protocol = 1
	ProtocolTCP    Protocol = 6
	ProtocolUDP    Protocol = 17
	ProtocolICMPv6 Protocol = 58
)

// InterfaceFlags represents the flags for a network interface.
type InterfaceFlags uint

// Interface flag constants.
const (
	InterfaceUp           InterfaceFlags = 1 << iota // Interface is up
	InterfaceBroadcast                               // Broadcast supported
	InterfaceLoopback                                // Loopback interface
	InterfacePointToPoint                            // Point-to-point link
	InterfaceMulticast                               // Multicast supported
)

// Interface represents a network interface.
type Interface struct {
	Name    string           // Interface name (e.g., "eth0")
	MAC     net.HardwareAddr // MAC address (6 bytes for Ethernet)
	IP      net.IP           // IP address
	Mask    net.IPMask       // Subnet mask
	Gateway net.IP           // Default gateway
	MTU     int              // Maximum transmission unit
	Flags   InterfaceFlags   // Interface flags
}

// HardwareAddr returns the hardware address as a byte slice.
func (i *Interface) HardwareAddr() []byte {
	return []byte(i.MAC)
}

// IPToUint32 converts an IPv4 address to a 32-bit uint.
func IPToUint32(ip net.IP) uint32 {
	if len(ip) != 4 {
		return 0
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

// Uint32ToIP converts a 32-bit uint to an IPv4 address.
func Uint32ToIP(v uint32) net.IP {
	return net.IP{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
}

// Packet represents a network packet.
type Packet struct {
	Data      []byte     // Raw packet data
	SrcIP     net.IP     // Source IP address
	DstIP     net.IP     // Destination IP address
	Protocol  Protocol   // Protocol type (TCP/UDP/ICMP)
	Iface     *Interface // Network interface
	Timestamp time.Time  // Packet timestamp
	Length    int        // Packet length
}

// IPPacket represents an IP packet.
type IPPacket struct {
	Version    uint8  // IP version (4 or 6)
	TOS        uint8  // Type of service
	Length     uint16 // Total length
	ID         uint16 // Identification
	Flags      uint8  // Fragment flags
	FragOffset uint16 // Fragment offset
	TTL        uint8  // Time to live
	Protocol   uint8  // Upper layer protocol
	Checksum   uint16 // Header checksum
	SrcIP      net.IP // Source address
	DstIP      net.IP // Destination address
	Payload    []byte // Packet payload
	Options    []byte // IP options (IPv4 only)
}

// TCPSegment represents a TCP segment.
type TCPSegment struct {
	SrcPort    uint16   // Source port
	DstPort    uint16   // Destination port
	SeqNum     uint32   // Sequence number
	AckNum     uint32   // Acknowledgment number
	DataOffset uint8    // Data offset (header length)
	Flags      TCPFlags // TCP flags
	WindowSize uint16   // Window size
	Checksum   uint16   // Checksum
	Urgent     uint16   // Urgent pointer
	Payload    []byte   // Segment payload
	Options    []byte   // TCP options
}

// TCPFlags represents TCP header flags.
type TCPFlags uint8

// TCP flag constants.
const (
	TCPFin TCPFlags = 1 << iota // Finish connection
	TCPSyn                      // Synchronize sequence numbers
	TCPRst                      // Reset connection
	TCPPsh                      // Push data to application
	TCPAck                      // Acknowledgment
	TCPUrg                      // Urgent pointer
	TCPEce                      // ECN-Echo
	TCPCwr                      // Congestion Window Reduced
)

// UDPDatagram represents a UDP datagram.
type UDPDatagram struct {
	SrcPort  uint16 // Source port
	DstPort  uint16 // Destination port
	Length   uint16 // Datagram length
	Checksum uint16 // Checksum
	Payload  []byte // Datagram payload
}

// ICMPMessage represents an ICMP message.
type ICMPMessage struct {
	Type         uint8  // ICMP type
	Code         uint8  // ICMP code
	Checksum     uint16 // Checksum
	RestOfHeader uint32 // Type-specific data
	Payload      []byte // ICMP payload
}

// EtherType represents the Ethernet frame type.
type EtherType uint16

// Common EtherType values.
const (
	EtherTypeIPv4 EtherType = 0x0800
	EtherTypeIPv6 EtherType = 0x86DD
	EtherTypeARP  EtherType = 0x0806
	EtherTypeVLAN EtherType = 0x8100
)

// Error definitions for the network stack.
var (
	ErrNotImplemented   = &net.AddrError{Err: "not implemented", Addr: ""}
	ErrInvalidPacket    = &net.AddrError{Err: "invalid packet", Addr: ""}
	ErrBufferTooSmall   = &net.AddrError{Err: "buffer too small", Addr: ""}
	ErrChecksumMismatch = &net.AddrError{Err: "checksum mismatch", Addr: ""}
)
