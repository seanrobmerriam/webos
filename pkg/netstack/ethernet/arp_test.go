package ethernet

import (
	"net"
	"testing"

	"webos/pkg/netstack"
)

func TestNewARPRequest(t *testing.T) {
	senderMAC := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	senderIP := net.ParseIP("192.168.1.100")
	targetIP := net.ParseIP("192.168.1.1")

	packet := NewARPRequest(senderMAC, senderIP, targetIP)

	if packet.HardwareType != 1 {
		t.Errorf("HardwareType = %d, want 1", packet.HardwareType)
	}
	if packet.ProtocolType != uint16(netstack.EtherTypeIPv4) {
		t.Errorf("ProtocolType = %d, want %d", packet.ProtocolType, netstack.EtherTypeIPv4)
	}
	if packet.Operation != ARPOperationRequest {
		t.Errorf("Operation = %d, want %d", packet.Operation, ARPOperationRequest)
	}
	if packet.SenderMAC.String() != senderMAC.String() {
		t.Errorf("SenderMAC = %s, want %s", packet.SenderMAC, senderMAC)
	}
}

func TestARPPacketSerialization(t *testing.T) {
	senderMAC := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	senderIP := net.ParseIP("192.168.1.100")
	targetIP := net.ParseIP("192.168.1.1")

	packet := NewARPRequest(senderMAC, senderIP, targetIP)
	serialized := packet.Serialize()

	if len(serialized) != ARPPacketSize {
		t.Errorf("Serialized length = %d, want %d", len(serialized), ARPPacketSize)
	}

	parsed, err := ParseARPPacket(serialized)
	if err != nil {
		t.Fatalf("ParseARPPacket failed: %v", err)
	}

	if parsed.Operation != packet.Operation {
		t.Errorf("Parsed Operation = %d, want %d", parsed.Operation, packet.Operation)
	}
}

func TestARPTable(t *testing.T) {
	table := NewARPTable()

	ip := net.ParseIP("192.168.1.100")
	mac := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	table.Set(ip, mac)

	result, err := table.Lookup(ip)
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if result.String() != mac.String() {
		t.Errorf("Lookup returned %s, want %s", result, mac)
	}

	table.Remove(ip)

	_, err = table.Lookup(ip)
	if err == nil {
		t.Error("Lookup should fail after Remove")
	}
}
