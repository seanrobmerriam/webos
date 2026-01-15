package ipv4_test

import (
	"bytes"
	"testing"

	ipv4 "webos/pkg/netstack/ip"
)

func TestParseICMPHeader(t *testing.T) {
	data := []byte{
		0x08,       // Type (Echo Request)
		0x00,       // Code
		0x00, 0x00, // Checksum
		0x12, 0x34, // ID
		0x00, 0x01, // Seq
		0x48, 0x65, 0x6c, // Payload start
	}

	h, err := ipv4.ParseICMPHeader(data)
	if err != nil {
		t.Fatalf("ParseICMPHeader failed: %v", err)
	}

	if h.Type != ipv4.ICMPTypeEcho {
		t.Errorf("Type = %d, want %d (Echo)", h.Type, ipv4.ICMPTypeEcho)
	}
	if h.Code != 0 {
		t.Errorf("Code = %d, want 0", h.Code)
	}
	if h.ID != 0x1234 {
		t.Errorf("ID = 0x%04x, want 0x1234", h.ID)
	}
	if h.Seq != 1 {
		t.Errorf("Seq = %d, want 1", h.Seq)
	}
}

func TestNewEchoRequest(t *testing.T) {
	msg := ipv4.NewEchoRequest(0x1234, 1, []byte("test data"))

	if msg.Header.Type != ipv4.ICMPTypeEcho {
		t.Errorf("Type = %d, want %d", msg.Header.Type, ipv4.ICMPTypeEcho)
	}
	if msg.Header.ID != 0x1234 {
		t.Errorf("ID = 0x%04x, want 0x1234", msg.Header.ID)
	}
	if msg.Header.Seq != 1 {
		t.Errorf("Seq = %d, want 1", msg.Header.Seq)
	}
	if !bytes.Equal(msg.Payload, []byte("test data")) {
		t.Errorf("Payload = %v, want %v", msg.Payload, "test data")
	}
}

func TestNewEchoReply(t *testing.T) {
	msg := ipv4.NewEchoReply(0x1234, 1, []byte("response"))

	if msg.Header.Type != ipv4.ICMPTypeEchoReply {
		t.Errorf("Type = %d, want %d", msg.Header.Type, ipv4.ICMPTypeEchoReply)
	}
}

func TestICMPChecksum(t *testing.T) {
	data := []byte("test data")
	msg := ipv4.NewEchoRequest(0x1234, 1, data)

	checksum := msg.Header.CalcChecksum(data)
	if checksum == 0 {
		t.Error("Checksum should not be zero")
	}
}

func TestICMPSerialize(t *testing.T) {
	msg := ipv4.NewEchoRequest(0x1234, 1, []byte("payload"))

	serialized := msg.Serialize()

	if len(serialized) != 8+len("payload") {
		t.Errorf("Serialized length = %d, want %d", len(serialized), 8+len("payload"))
	}

	if serialized[0] != ipv4.ICMPTypeEcho {
		t.Errorf("First byte = %d, want %d", serialized[0], ipv4.ICMPTypeEcho)
	}
}

func TestICMPMessageTypes(t *testing.T) {
	tests := []struct {
		msg    *ipv4.Message
		isReq  bool
		isRep  bool
		isDst  bool
		isTime bool
	}{
		{ipv4.NewEchoRequest(1, 1, nil), true, false, false, false},
		{ipv4.NewEchoReply(1, 1, nil), false, true, false, false},
		{ipv4.NewDestUnreach(ipv4.ICMPCodeNetUnreach, nil), false, false, true, false},
		{ipv4.NewTimeExceeded(nil), false, false, false, true},
	}

	for _, tt := range tests {
		if tt.msg.IsEchoRequest() != tt.isReq {
			t.Errorf("IsEchoRequest = %v, want %v", tt.msg.IsEchoRequest(), tt.isReq)
		}
		if tt.msg.IsEchoReply() != tt.isRep {
			t.Errorf("IsEchoReply = %v, want %v", tt.msg.IsEchoReply(), tt.isRep)
		}
		if tt.msg.IsDestUnreach() != tt.isDst {
			t.Errorf("IsDestUnreach = %v, want %v", tt.msg.IsDestUnreach(), tt.isDst)
		}
		if tt.msg.IsTimeExceeded() != tt.isTime {
			t.Errorf("IsTimeExceeded = %v, want %v", tt.msg.IsTimeExceeded(), tt.isTime)
		}
	}
}
