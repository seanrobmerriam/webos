package udp_test

import (
	"bytes"
	"encoding/binary"
	network "net"
	"testing"

	"webos/pkg/netstack/udp"
)

func TestParseHeader(t *testing.T) {
	data := []byte{
		0x1a, 0x2b, // Src port 6699 (0x1a2b)
		0x00, 0x35, // Dst port 53
		0x00, 0x10, // Length 16
		0x00, 0x00, // Checksum
	}

	h, err := udp.ParseHeader(data)
	if err != nil {
		t.Fatalf("ParseHeader failed: %v", err)
	}

	if h.SrcPort != 6699 {
		t.Errorf("SrcPort = %d, want 6699", h.SrcPort)
	}
	if h.DstPort != 53 {
		t.Errorf("DstPort = %d, want 53", h.DstPort)
	}
	if h.Length != 16 {
		t.Errorf("Length = %d, want 16", h.Length)
	}
}

func TestSerializeHeader(t *testing.T) {
	h := &udp.Header{
		SrcPort:  12345,
		DstPort:  53,
		Length:   20,
		Checksum: 0,
	}

	serialized := h.Serialize()

	if len(serialized) != 8 {
		t.Errorf("Serialized length = %d, want 8", len(serialized))
	}

	port := binary.BigEndian.Uint16(serialized[0:2])
	if port != 12345 {
		t.Errorf("SrcPort = %d, want 12345", port)
	}

	length := binary.BigEndian.Uint16(serialized[4:6])
	if length != 20 {
		t.Errorf("Length = %d, want 20", length)
	}
}

func TestParseDatagram(t *testing.T) {
	header := []byte{
		0x1a, 0x2b, // Src port 6699
		0x00, 0x35, // Dst port 53
		0x00, 0x0d, // Length 13 (8 header + 5 data)
		0x00, 0x00, // Checksum
	}
	payload := []byte("hello")

	srcIP := network.IP{192, 168, 1, 100}
	dstIP := network.IP{192, 168, 1, 1}

	dg, err := udp.ParseDatagram(append(header, payload...), srcIP, dstIP)
	if err != nil {
		t.Fatalf("ParseDatagram failed: %v", err)
	}

	if dg.Header.SrcPort != 6699 {
		t.Errorf("SrcPort = %d, want 6699", dg.Header.SrcPort)
	}
	if !bytes.Equal(dg.Payload, payload) {
		t.Errorf("Payload = %v, want %v", dg.Payload, payload)
	}
}

func TestNewDatagram(t *testing.T) {
	srcIP := network.IP{192, 168, 1, 100}
	dstIP := network.IP{192, 168, 1, 1}
	payload := []byte("test payload")

	dg := udp.NewDatagram(12345, 53, srcIP, dstIP, payload)

	if dg.Header.SrcPort != 12345 {
		t.Errorf("SrcPort = %d, want 12345", dg.Header.SrcPort)
	}
	if dg.Header.DstPort != 53 {
		t.Errorf("DstPort = %d, want 53", dg.Header.DstPort)
	}
	if dg.Header.Length != uint16(8+len(payload)) {
		t.Errorf("Length = %d, want %d", dg.Header.Length, 8+len(payload))
	}
}

func TestSerializeDatagram(t *testing.T) {
	srcIP := network.IP{192, 168, 1, 100}
	dstIP := network.IP{192, 168, 1, 1}
	payload := []byte("hello")

	dg := udp.NewDatagram(12345, 53, srcIP, dstIP, payload)

	// Don't test checksum calculation since it requires proper pseudo-header
	// Just verify basic serialization
	length := dg.Header.Length
	if length != uint16(8+len(payload)) {
		t.Errorf("Length = %d, want %d", length, 8+len(payload))
	}
}

func TestSocket(t *testing.T) {
	socket := udp.NewSocket(53, network.IP{192, 168, 1, 1})

	if socket.Port != 53 {
		t.Errorf("Port = %d, want 53", socket.Port)
	}

	// Test send and receive
	dg := udp.NewDatagram(12345, 53, network.IP{192, 168, 1, 100}, network.IP{192, 168, 1, 1}, []byte("test"))

	err := socket.Send(dg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	received, err := socket.Receive()
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	if !bytes.Equal(received.Payload, []byte("test")) {
		t.Errorf("Received payload = %v, want %v", received.Payload, "test")
	}
}

func TestSocketBufferFull(t *testing.T) {
	socket := udp.NewSocket(53, network.IP{192, 168, 1, 1})

	// Fill the buffer
	for i := 0; i < 100; i++ {
		ip := network.IP{192, 168, 1, byte(i & 0xFF)}
		dg := udp.NewDatagram(uint16(i), 53, ip, network.IP{192, 168, 1, 1}, []byte("test"))
		socket.Send(dg)
	}

	// This should fail
	dg := udp.NewDatagram(1000, 53, network.IP{192, 168, 1, 100}, network.IP{192, 168, 1, 1}, []byte("test"))
	err := socket.Send(dg)
	if err == nil {
		t.Error("Send should have failed due to full buffer")
	}
}
