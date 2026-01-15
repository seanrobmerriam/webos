package ipv4_test

import (
	"bytes"
	"encoding/binary"
	network "net"
	"testing"

	ipv4 "webos/pkg/netstack/ip"
)

func TestParseHeader(t *testing.T) {
	// Create a valid IPv4 header
	data := []byte{
		0x45,       // Version and IHL
		0x00,       // TOS
		0x00, 0x2a, // Length (42)
		0x12, 0x34, // ID
		0x40, 0x00, // Flags and Fragment Offset
		0x40,       // TTL
		0x06,       // Protocol (TCP)
		0xb1, 0xb2, // Checksum
		0xc0, 0xa8, 0x01, 0x64, // Source IP (192.168.1.100)
		0xc0, 0xa8, 0x01, 0x01, // Dest IP (192.168.1.1)
	}

	h, err := ipv4.ParseHeader(data)
	if err != nil {
		t.Fatalf("ParseHeader failed: %v", err)
	}

	if h.Version != 4 {
		t.Errorf("Version = %d, want 4", h.Version)
	}
	if h.IHL != 5 {
		t.Errorf("IHL = %d, want 5", h.IHL)
	}
	if h.TOS != 0 {
		t.Errorf("TOS = %d, want 0", h.TOS)
	}
	if h.Length != 42 {
		t.Errorf("Length = %d, want 42", h.Length)
	}
	if h.ID != 0x1234 {
		t.Errorf("ID = 0x%04x, want 0x1234", h.ID)
	}
	if h.TTL != 64 {
		t.Errorf("TTL = %d, want 64", h.TTL)
	}
	if h.Protocol != 6 {
		t.Errorf("Protocol = %d, want 6 (TCP)", h.Protocol)
	}
	if !h.SrcIP.Equal(network.IP{192, 168, 1, 100}) {
		t.Errorf("SrcIP = %v, want 192.168.1.100", h.SrcIP)
	}
	if !h.DstIP.Equal(network.IP{192, 168, 1, 1}) {
		t.Errorf("DstIP = %v, want 192.168.1.1", h.DstIP)
	}
}

func TestSerializeHeader(t *testing.T) {
	h := &ipv4.Header{
		Version:    4,
		IHL:        5,
		TOS:        0,
		Length:     20, // Header only, no payload
		ID:         0x1234,
		Flags:      0x2,
		FragOffset: 0,
		TTL:        64,
		Protocol:   6,
		Checksum:   0,
		SrcIP:      network.IP{192, 168, 1, 100},
		DstIP:      network.IP{192, 168, 1, 1},
	}

	// Calculate checksum
	h.Checksum = h.CalcChecksum()

	serialized := h.Serialize()
	if len(serialized) != 20 {
		t.Errorf("Serialized length = %d, want 20", len(serialized))
	}

	if serialized[0] != 0x45 {
		t.Errorf("First byte = 0x%02x, want 0x45", serialized[0])
	}

	// Verify checksum
	checksum := binary.BigEndian.Uint16(serialized[10:12])
	if checksum != h.Checksum {
		t.Errorf("Checksum = 0x%04x, want 0x%04x", checksum, h.Checksum)
	}
}

func TestParseDatagram(t *testing.T) {
	// Create a complete datagram
	header := []byte{
		0x45,       // Version and IHL
		0x00,       // TOS
		0x00, 0x2a, // Length (42)
		0x12, 0x34, // ID
		0x40, 0x00, // Flags and Fragment Offset
		0x40,       // TTL
		0x06,       // Protocol (TCP)
		0x00, 0x00, // Checksum (will be filled)
		0xc0, 0xa8, 0x01, 0x64, // Source IP
		0xc0, 0xa8, 0x01, 0x01, // Dest IP
	}
	payload := []byte("Hello, World!")

	// Calculate and set checksum
	h, _ := ipv4.ParseHeader(header)
	header[10], header[11] = byte(h.CalcChecksum()>>8), byte(h.CalcChecksum())

	datagram := append(header, payload...)

	d, err := ipv4.ParseDatagram(datagram)
	if err != nil {
		t.Fatalf("ParseDatagram failed: %v", err)
	}

	if !bytes.Equal(d.Payload, payload) {
		t.Errorf("Payload = %q, want %q", d.Payload, payload)
	}
}

func TestNewDatagram(t *testing.T) {
	srcIP := network.IP{192, 168, 1, 100}
	dstIP := network.IP{192, 168, 1, 1}
	payload := []byte("Test payload")

	d := ipv4.NewDatagram(srcIP, dstIP, ipv4.ProtocolTCP, payload)

	if d.Header.Version != 4 {
		t.Errorf("Version = %d, want 4", d.Header.Version)
	}
	if d.Header.IHL != 5 {
		t.Errorf("IHL = %d, want 5", d.Header.IHL)
	}
	if !bytes.Equal(d.Payload, payload) {
		t.Errorf("Payload = %v, want %v", d.Payload, payload)
	}
}

func TestFragment(t *testing.T) {
	srcIP := network.IP{192, 168, 1, 100}
	dstIP := network.IP{192, 168, 1, 1}
	payload := make([]byte, 4000) // Large payload

	d := ipv4.NewDatagram(srcIP, dstIP, ipv4.ProtocolUDP, payload)

	fragments, err := ipv4.Fragment(d, 1500)
	if err != nil {
		t.Fatalf("Fragment failed: %v", err)
	}

	if len(fragments) < 2 {
		t.Errorf("Expected multiple fragments, got %d", len(fragments))
	}

	// Verify total payload
	totalPayload := 0
	for _, frag := range fragments {
		totalPayload += len(frag.Payload)
	}
	if totalPayload != len(payload) {
		t.Errorf("Total payload = %d, want %d", totalPayload, len(payload))
	}
}

func TestReassemble(t *testing.T) {
	srcIP := network.IP{192, 168, 1, 100}
	dstIP := network.IP{192, 168, 1, 1}
	originalPayload := []byte("This is a test payload that will be fragmented and reassembled")

	d := ipv4.NewDatagram(srcIP, dstIP, ipv4.ProtocolTCP, originalPayload)

	fragments, err := ipv4.Fragment(d, 100)
	if err != nil {
		t.Fatalf("Fragment failed: %v", err)
	}

	reassembled, err := ipv4.Reassemble(fragments)
	if err != nil {
		t.Fatalf("Reassemble failed: %v", err)
	}

	if !bytes.Equal(reassembled.Payload, originalPayload) {
		t.Errorf("Reassembled payload = %q, want %q", reassembled.Payload, originalPayload)
	}
}

func TestIsFragment(t *testing.T) {
	h := &ipv4.Header{
		Flags:      0,
		FragOffset: 0,
	}
	if h.IsFragment() {
		t.Error("Non-fragment reported as fragment")
	}

	h.FragOffset = 100
	if !h.IsFragment() {
		t.Error("Fragment not detected")
	}

	h.FragOffset = 0
	h.Flags = 0x1
	if !h.IsFragment() {
		t.Error("More-fragments flag not detected")
	}
}
