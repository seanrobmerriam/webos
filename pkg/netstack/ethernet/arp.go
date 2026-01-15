package ethernet

import (
	"encoding/binary"
	"fmt"
	network "net"
	"time"

	"webos/pkg/netstack"
)

// ARP operation types.
const (
	ARPOperationRequest uint16 = 1
	ARPOperationReply   uint16 = 2
)

// ARPPacketSize is the size of an ARP packet in bytes.
const ARPPacketSize = 28

// ARPPacket represents an ARP packet for Ethernet/IP networks.
type ARPPacket struct {
	HardwareType uint16
	ProtocolType uint16
	HardwareSize uint8
	ProtocolSize uint8
	Operation    uint16
	SenderMAC    network.HardwareAddr
	SenderIP     network.IP
	TargetMAC    network.HardwareAddr
	TargetIP     network.IP
}

// ParseARPPacket parses an ARP packet from raw bytes.
func ParseARPPacket(data []byte) (*ARPPacket, error) {
	if len(data) < ARPPacketSize {
		return nil, fmt.Errorf("ARP packet too short: %d bytes", len(data))
	}

	p := &ARPPacket{
		HardwareType: binary.BigEndian.Uint16(data[0:2]),
		ProtocolType: binary.BigEndian.Uint16(data[2:4]),
		HardwareSize: data[4],
		ProtocolSize: data[5],
		Operation:    binary.BigEndian.Uint16(data[6:8]),
		SenderMAC:    network.HardwareAddr{data[8], data[9], data[10], data[11], data[12], data[13]},
		TargetMAC:    network.HardwareAddr{data[18], data[19], data[20], data[21], data[22], data[23]},
	}
	p.SenderIP = network.IP{data[14], data[15], data[16], data[17]}
	p.TargetIP = network.IP{data[24], data[25], data[26], data[27]}

	return p, nil
}

// Serialize converts the ARP packet to raw bytes.
func (p *ARPPacket) Serialize() []byte {
	buf := make([]byte, ARPPacketSize)
	binary.BigEndian.PutUint16(buf[0:2], p.HardwareType)
	binary.BigEndian.PutUint16(buf[2:4], p.ProtocolType)
	buf[4] = p.HardwareSize
	buf[5] = p.ProtocolSize
	binary.BigEndian.PutUint16(buf[6:8], p.Operation)
	copy(buf[8:14], p.SenderMAC)
	copy(buf[14:18], []byte(p.SenderIP))
	copy(buf[18:24], p.TargetMAC)
	copy(buf[24:28], []byte(p.TargetIP))
	return buf
}

// NewARPRequest creates an ARP request packet.
func NewARPRequest(senderMAC network.HardwareAddr, senderIP network.IP, targetIP network.IP) *ARPPacket {
	zeroMAC := network.HardwareAddr{0, 0, 0, 0, 0, 0}
	return &ARPPacket{
		HardwareType: 1,
		ProtocolType: uint16(netstack.EtherTypeIPv4),
		HardwareSize: 6,
		ProtocolSize: 4,
		Operation:    ARPOperationRequest,
		SenderMAC:    senderMAC,
		SenderIP:     senderIP,
		TargetMAC:    zeroMAC,
		TargetIP:     targetIP,
	}
}

// NewARPReply creates an ARP reply packet.
func NewARPReply(senderMAC network.HardwareAddr, sndrIP network.IP, targetMAC network.HardwareAddr, tgtIP network.IP) *ARPPacket {
	return &ARPPacket{
		HardwareType: 1,
		ProtocolType: uint16(netstack.EtherTypeIPv4),
		HardwareSize: 6,
		ProtocolSize: 4,
		Operation:    ARPOperationReply,
		SenderMAC:    senderMAC,
		SenderIP:     sndrIP,
		TargetMAC:    targetMAC,
		TargetIP:     tgtIP,
	}
}

// IsValid returns true if the ARP packet has valid fields.
func (p *ARPPacket) IsValid() bool {
	return p.HardwareType == 1 &&
		p.ProtocolType == uint16(netstack.EtherTypeIPv4) &&
		p.HardwareSize == 6 &&
		p.ProtocolSize == 4
}

// ARPTable maintains a cache of IP-to-MAC mappings.
type ARPTable struct {
	entries map[string]*ARPEntry
}

// ARPEntry represents a single entry in the ARP cache.
type ARPEntry struct {
	MAC     network.HardwareAddr
	IP      network.IP
	Created time.Time
	Updated time.Time
	State   ARPState
}

// ARPState represents the state of an ARP entry.
type ARPState int

const (
	ARPStateIncomplete ARPState = iota
	ARPStateReachable
	ARPStateStale
	ARPStateFailed
)

// NewARPTable creates a new ARP table.
func NewARPTable() *ARPTable {
	return &ARPTable{entries: make(map[string]*ARPEntry)}
}

// Lookup returns the MAC address for the given IP.
func (t *ARPTable) Lookup(ip network.IP) (network.HardwareAddr, error) {
	if entry, ok := t.entries[ip.String()]; ok {
		return entry.MAC, nil
	}
	return nil, fmt.Errorf("no ARP entry for %s", ip)
}

// Set adds or updates an ARP entry.
func (t *ARPTable) Set(ip network.IP, mac network.HardwareAddr) {
	now := time.Now()
	key := ip.String()
	if entry, ok := t.entries[key]; ok {
		entry.MAC = mac
		entry.Updated = now
		entry.State = ARPStateReachable
	} else {
		t.entries[key] = &ARPEntry{
			MAC:     mac,
			IP:      ip,
			Created: now,
			Updated: now,
			State:   ARPStateReachable,
		}
	}
}

// Remove deletes an ARP entry.
func (t *ARPTable) Remove(ip network.IP) {
	delete(t.entries, ip.String())
}
