package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

func TestMessageEncodeDecode(t *testing.T) {
	tests := []struct {
		name    string
		opcode  Opcode
		payload []byte
	}{
		{
			name:    "display message",
			opcode:  OpcodeDisplay,
			payload: []byte("display data"),
		},
		{
			name:    "input message",
			opcode:  OpcodeInput,
			payload: []byte(`{"type":"keydown","key":"a"}`),
		},
		{
			name:    "auth message",
			opcode:  OpcodeAuth,
			payload: []byte(`{"token":"abc123"}`),
		},
		{
			name:    "empty payload",
			opcode:  OpcodePing,
			payload: nil,
		},
		{
			name:    "filesystem message",
			opcode:  OpcodeFileSystem,
			payload: []byte("file read request"),
		},
		{
			name:    "network message",
			opcode:  OpcodeNetwork,
			payload: []byte("http request"),
		},
		{
			name:    "process message",
			opcode:  OpcodeProcess,
			payload: []byte("spawn process"),
		},
		{
			name:    "connect message",
			opcode:  OpcodeConnect,
			payload: []byte("client connect"),
		},
		{
			name:    "disconnect message",
			opcode:  OpcodeDisconnect,
			payload: []byte("client disconnect"),
		},
		{
			name:    "pong message",
			opcode:  OpcodePong,
			payload: nil,
		},
		{
			name:    "error message",
			opcode:  OpcodeError,
			payload: []byte("error details"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMessage(tt.opcode, tt.payload)
			encoded, err := msg.Encode()
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			if len(encoded) < HeaderSize {
				t.Error("Encoded message too short")
			}

			if !bytes.Equal(encoded[0:4], MagicBytes[:]) {
				t.Error("Magic bytes mismatch")
			}

			if encoded[4] != ProtocolVersion {
				t.Error("Version mismatch")
			}

			if Opcode(encoded[5]) != tt.opcode {
				t.Error("Opcode mismatch")
			}

			var decoded Message
			if err := decoded.Decode(encoded); err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			if decoded.Opcode != tt.opcode {
				t.Errorf("Opcode mismatch: got %v, want %v", decoded.Opcode, tt.opcode)
			}

			if !bytes.Equal(decoded.Payload, tt.payload) {
				t.Errorf("Payload mismatch: got %v, want %v", decoded.Payload, tt.payload)
			}
		})
	}
}

func TestMessageEncodeDecodeLargePayload(t *testing.T) {
	payload := make([]byte, 1024*1024)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	msg := NewMessage(OpcodeDisplay, payload)
	encoded, err := msg.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	expectedSize := HeaderSize + len(payload)
	if len(encoded) != expectedSize {
		t.Errorf("Encoded size mismatch: got %d, want %d", len(encoded), expectedSize)
	}

	var decoded Message
	if err := decoded.Decode(encoded); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if !bytes.Equal(decoded.Payload, payload) {
		t.Error("Large payload mismatch")
	}
}

func TestMessageErrors(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr error
	}{
		{
			name:    "buffer too small",
			data:    []byte{0x57, 0x45, 0x42, 0x53, 1, 1},
			wantErr: ErrBufferTooSmall,
		},
		{
			name:    "invalid magic",
			data:    make([]byte, 20),
			wantErr: ErrInvalidMagic,
		},
		{
			name:    "invalid version",
			data:    append(append([]byte("WEBS"), 2), make([]byte, 16)...),
			wantErr: ErrInvalidVersion,
		},
		{
			name:    "invalid opcode",
			data:    createMessage(Opcode(100), nil),
			wantErr: ErrInvalidOpcode,
		},
		{
			name:    "payload too large",
			data:    createLargePayloadMessage(17 * 1024 * 1024),
			wantErr: ErrPayloadTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg Message
			err := msg.Decode(tt.data)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestOpcodeString(t *testing.T) {
	tests := []struct {
		opcode Opcode
		want   string
	}{
		{OpcodeDisplay, "DISPLAY"},
		{OpcodeInput, "INPUT"},
		{OpcodeFileSystem, "FILESYSTEM"},
		{OpcodeNetwork, "NETWORK"},
		{OpcodeProcess, "PROCESS"},
		{OpcodeAuth, "AUTH"},
		{OpcodeConnect, "CONNECT"},
		{OpcodeDisconnect, "DISCONNECT"},
		{OpcodePing, "PING"},
		{OpcodePong, "PONG"},
		{OpcodeError, "ERROR"},
		{Opcode(0), "UNKNOWN"},
		{Opcode(100), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.opcode.String(); got != tt.want {
			t.Errorf("Opcode.String() = %v, want %v", got, tt.want)
		}
	}
}

func TestOpcodeIsValid(t *testing.T) {
	tests := []struct {
		opcode Opcode
		valid  bool
	}{
		{OpcodeInvalid, false},
		{OpcodeDisplay, true},
		{OpcodeInput, true},
		{OpcodeFileSystem, true},
		{OpcodeNetwork, true},
		{OpcodeProcess, true},
		{OpcodeAuth, true},
		{OpcodeConnect, true},
		{OpcodeDisconnect, true},
		{OpcodePing, true},
		{OpcodePong, true},
		{OpcodeError, true},
		{Opcode(12), false},
		{Opcode(100), false},
	}

	for _, tt := range tests {
		if got := tt.opcode.IsValid(); got != tt.valid {
			t.Errorf("Opcode.IsValid() = %v, want %v for opcode %d", got, tt.valid, tt.opcode)
		}
	}
}

func TestNewMessage(t *testing.T) {
	before := time.Now().UnixNano()
	msg := NewMessage(OpcodeDisplay, []byte("test"))
	after := time.Now().UnixNano()

	if msg.Opcode != OpcodeDisplay {
		t.Errorf("Opcode mismatch: got %v, want %v", msg.Opcode, OpcodeDisplay)
	}

	if msg.Timestamp < before || msg.Timestamp > after {
		t.Error("Timestamp out of expected range")
	}

	if !bytes.Equal(msg.Payload, []byte("test")) {
		t.Errorf("Payload mismatch: got %v, want %v", msg.Payload, []byte("test"))
	}
}

func TestMaxPayloadSize(t *testing.T) {
	maxPayload := make([]byte, MaxPayloadSize)
	msg := NewMessage(OpcodeDisplay, maxPayload)

	_, err := msg.Encode()
	if err != nil {
		t.Errorf("Max payload encoding failed: %v", err)
	}
}

func TestPayloadTooLarge(t *testing.T) {
	tooLargePayload := make([]byte, MaxPayloadSize+1)
	msg := NewMessage(OpcodeDisplay, tooLargePayload)

	_, err := msg.Encode()
	if err != ErrPayloadTooLarge {
		t.Errorf("Expected ErrPayloadTooLarge, got %v", err)
	}
}

func TestEmptyPayload(t *testing.T) {
	msg := NewMessage(OpcodePing, nil)
	encoded, err := msg.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(encoded) != HeaderSize {
		t.Errorf("Expected header size %d, got %d", HeaderSize, len(encoded))
	}

	var decoded Message
	if err := decoded.Decode(encoded); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded.Payload != nil {
		t.Errorf("Expected nil payload, got %v", decoded.Payload)
	}
}

func TestMagicBytes(t *testing.T) {
	msg := NewMessage(OpcodeDisplay, []byte("test"))
	encoded, _ := msg.Encode()

	expected := [4]byte{0x57, 0x45, 0x42, 0x53}
	if !bytes.Equal(encoded[0:4], expected[:]) {
		t.Errorf("Magic bytes mismatch: got %v, want %v", encoded[0:4], expected)
	}
}

func TestTimestampEncoding(t *testing.T) {
	testTimestamp := int64(1234567890123456789)
	msg := &Message{
		Opcode:    OpcodeDisplay,
		Timestamp: testTimestamp,
		Payload:   []byte("test"),
	}

	encoded, err := msg.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	var decoded Message
	if err := decoded.Decode(encoded); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded.Timestamp != testTimestamp {
		t.Errorf("Timestamp mismatch: got %d, want %d", decoded.Timestamp, testTimestamp)
	}
}

func TestPayloadLength(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
	}{
		{"empty", nil},
		{"small", []byte("a")},
		{"medium", make([]byte, 100)},
		{"large", make([]byte, 1024)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMessage(OpcodeDisplay, tt.payload)
			encoded, err := msg.Encode()
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			// Extract payload length from header
			expectedLen := uint32(0)
			if tt.payload != nil {
				expectedLen = uint32(len(tt.payload))
			}
			encodedLen := uint32(encoded[14])<<24 | uint32(encoded[15])<<16 | uint32(encoded[16])<<8 | uint32(encoded[17])

			if encodedLen != expectedLen {
				t.Errorf("Payload length mismatch: got %d, want %d", encodedLen, expectedLen)
			}
		})
	}
}

// Benchmark tests
func BenchmarkMessageEncode(b *testing.B) {
	msg := NewMessage(OpcodeDisplay, []byte("test payload"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg.Encode()
	}
}

func BenchmarkMessageDecode(b *testing.B) {
	msg := NewMessage(OpcodeDisplay, []byte("test payload"))
	encoded, _ := msg.Encode()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m Message
		m.Decode(encoded)
	}
}

func BenchmarkCodecRead(b *testing.B) {
	buf := make([]byte, 1024)
	codec := NewCodec(buf)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		codec.Reset()
		codec.WriteUint32(12345)
		codec.WriteUint64(9876543210)
		codec.WriteBytes([]byte("test"))
	}
}

func BenchmarkCodecWrite(b *testing.B) {
	buf := make([]byte, 1024)
	codec := NewCodec(buf)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		codec.Reset()
		codec.WriteUint32(12345)
		codec.WriteUint64(9876543210)
		codec.WriteBytes([]byte("test"))
	}
}

// Helper function to create a test message with specific opcode
func createMessage(opcode Opcode, payload []byte) []byte {
	buf := make([]byte, HeaderSize+len(payload))
	copy(buf[0:4], MagicBytes[:])
	buf[4] = ProtocolVersion
	buf[5] = byte(opcode)
	if len(payload) > 0 {
		copy(buf[HeaderSize:], payload)
	}
	return buf
}

// Helper function to create a message with large payload
func createLargePayloadMessage(size int) []byte {
	buf := make([]byte, HeaderSize+size)
	copy(buf[0:4], MagicBytes[:])
	buf[4] = ProtocolVersion
	buf[5] = byte(OpcodeDisplay)
	binary.BigEndian.PutUint32(buf[14:18], uint32(size))
	return buf
}
