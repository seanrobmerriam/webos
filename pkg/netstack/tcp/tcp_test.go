package tcp_test

import (
	"bytes"
	"encoding/binary"
	network "net"
	"testing"

	"webos/pkg/netstack/tcp"
)

func TestParseHeader(t *testing.T) {
	data := []byte{
		0x1a, 0x2b, // Src port 6699 (0x1a2b)
		0x00, 0x50, // Dst port 80
		0x00, 0x00, 0x00, 0x01, // Seq num 1
		0x00, 0x00, 0x00, 0x00, // Ack num 0
		0x50,       // Data offset + reserved
		0x02,       // SYN flag
		0x04, 0x00, // Window 1024
		0xb1, 0xb2, // Checksum
		0x00, 0x00, // Urgent
		0x02, 0x04, 0x05, 0xb4, // MSS option
	}

	h, err := tcp.ParseHeader(data)
	if err != nil {
		t.Fatalf("ParseHeader failed: %v", err)
	}

	if h.SrcPort != 6699 {
		t.Errorf("SrcPort = %d, want 6699", h.SrcPort)
	}
	if h.DstPort != 80 {
		t.Errorf("DstPort = %d, want 80", h.DstPort)
	}
	if h.SeqNum != 1 {
		t.Errorf("SeqNum = %d, want 1", h.SeqNum)
	}
	if h.AckNum != 0 {
		t.Errorf("AckNum = %d, want 0", h.AckNum)
	}
	if h.DataOffset != 5 {
		t.Errorf("DataOffset = %d, want 5", h.DataOffset)
	}
	if h.Flags != tcp.FlagSYN {
		t.Errorf("Flags = 0x%02x, want 0x%02x (SYN)", h.Flags, tcp.FlagSYN)
	}
	if h.Window != 1024 {
		t.Errorf("Window = %d, want 1024", h.Window)
	}
}

func TestSerializeHeader(t *testing.T) {
	h := &tcp.Header{
		SrcPort:    12345,
		DstPort:    80,
		SeqNum:     1000,
		AckNum:     0,
		DataOffset: 5,
		Flags:      tcp.FlagSYN,
		Window:     65535,
		Checksum:   0,
		Urgent:     0,
	}

	serialized := h.Serialize()

	if len(serialized) != 20 {
		t.Errorf("Serialized length = %d, want 20", len(serialized))
	}

	port := binary.BigEndian.Uint16(serialized[0:2])
	if port != 12345 {
		t.Errorf("SrcPort = %d, want 12345", port)
	}

	flags := serialized[13]
	if flags != tcp.FlagSYN {
		t.Errorf("Flags = 0x%02x, want 0x%02x", flags, tcp.FlagSYN)
	}
}

func TestParseSegment(t *testing.T) {
	header := []byte{
		0x1a, 0x2b, // Src port 6699
		0x00, 0x50, // Dst port 80
		0x00, 0x00, 0x00, 0x01, // Seq num 1
		0x00, 0x00, 0x00, 0x00, // Ack num 0
		0x50,       // Data offset
		0x02,       // SYN flag
		0x04, 0x00, // Window 1024
		0x00, 0x00, // Checksum (will be filled)
		0x00, 0x00, // Urgent
	}
	payload := []byte("Hello")

	srcIP := network.IP{192, 168, 1, 100}
	dstIP := network.IP{192, 168, 1, 1}

	seg, err := tcp.ParseSegment(append(header, payload...), srcIP, dstIP)
	if err != nil {
		t.Fatalf("ParseSegment failed: %v", err)
	}

	if seg.Header.SrcPort != 6699 {
		t.Errorf("SrcPort = %d, want 6699", seg.Header.SrcPort)
	}
	if !bytes.Equal(seg.Payload, payload) {
		t.Errorf("Payload = %v, want %v", seg.Payload, payload)
	}
}

func TestNewSegment(t *testing.T) {
	srcIP := network.IP{192, 168, 1, 100}
	dstIP := network.IP{192, 168, 1, 1}
	payload := []byte("test")

	seg := tcp.NewSegment(12345, 80, srcIP, dstIP, tcp.FlagPSH|tcp.FlagACK, 100, 50, payload)

	if seg.Header.SrcPort != 12345 {
		t.Errorf("SrcPort = %d, want 12345", seg.Header.SrcPort)
	}
	if seg.Header.DstPort != 80 {
		t.Errorf("DstPort = %d, want 80", seg.Header.DstPort)
	}
	if seg.Header.SeqNum != 100 {
		t.Errorf("SeqNum = %d, want 100", seg.Header.SeqNum)
	}
	if seg.Header.AckNum != 50 {
		t.Errorf("AckNum = %d, want 50", seg.Header.AckNum)
	}
	if seg.Header.Flags != tcp.FlagPSH|tcp.FlagACK {
		t.Errorf("Flags = 0x%02x, want 0x%02x", seg.Header.Flags, tcp.FlagPSH|tcp.FlagACK)
	}
}

func TestNewConnection(t *testing.T) {
	id := tcp.ConnectionID{
		SrcIP:   network.IP{192, 168, 1, 100},
		SrcPort: 12345,
		DstIP:   network.IP{192, 168, 1, 1},
		DstPort: 80,
	}

	conn := tcp.NewConnection(id, nil, nil)

	if conn.State != tcp.StateClosed {
		t.Errorf("State = %d, want %d", conn.State, tcp.StateClosed)
	}
	if conn.ISS == 0 {
		t.Error("ISS should not be zero")
	}
	if conn.SSThresh != tcp.InitialSSThresh {
		t.Errorf("SSThresh = %d, want %d", conn.SSThresh, tcp.InitialSSThresh)
	}
}

func TestConnectionState(t *testing.T) {
	conn := tcp.NewConnection(tcp.ConnectionID{}, nil, nil)

	if !conn.IsState(tcp.StateClosed) {
		t.Error("Should be in Closed state")
	}
	if conn.IsEstablished() {
		t.Error("Should not be established")
	}

	conn.State = tcp.StateEstablished

	if !conn.IsEstablished() {
		t.Error("Should be established")
	}
}

func TestSeqLess(t *testing.T) {
	if !seqLess(100, 200) {
		t.Error("100 should be less than 200")
	}
	if seqLess(200, 100) {
		t.Error("200 should not be less than 100")
	}
	if !seqLess(0xffffffff-10, 10) {
		t.Error("wrap-around case failed")
	}
}

func seqLess(a, b uint32) bool {
	return int32(a-b) < 0
}

func TestConnectionAcknowledge(t *testing.T) {
	conn := tcp.NewConnection(tcp.ConnectionID{}, nil, nil)
	conn.SND = 1000
	conn.SNDUNA = 900

	conn.Acknowledge(950)

	if conn.SNDUNA != 950 {
		t.Errorf("SNDUNA = %d, want 950", conn.SNDUNA)
	}
}

func TestTCPFlags(t *testing.T) {
	if tcp.FlagFIN != 0x01 {
		t.Errorf("FlagFIN = 0x%02x, want 0x01", tcp.FlagFIN)
	}
	if tcp.FlagSYN != 0x02 {
		t.Errorf("FlagSYN = 0x%02x, want 0x02", tcp.FlagSYN)
	}
	if tcp.FlagRST != 0x04 {
		t.Errorf("FlagRST = 0x%02x, want 0x04", tcp.FlagRST)
	}
	if tcp.FlagPSH != 0x08 {
		t.Errorf("FlagPSH = 0x%02x, want 0x08", tcp.FlagPSH)
	}
	if tcp.FlagACK != 0x10 {
		t.Errorf("FlagACK = 0x%02x, want 0x10", tcp.FlagACK)
	}
	if tcp.FlagURG != 0x20 {
		t.Errorf("FlagURG = 0x%02x, want 0x20", tcp.FlagURG)
	}
}
