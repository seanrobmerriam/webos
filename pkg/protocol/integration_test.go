package protocol

import (
	"bytes"
	"testing"
)

// TestGoJavaScriptCompatibility tests that messages encoded in Go
// can be decoded by JavaScript and vice versa.
func TestGoJavaScriptCompatibility(t *testing.T) {
	t.Run("roundtrip display message", func(t *testing.T) {
		payload := []byte(`{"type":"text","content":"Hello WebOS!"}`)
		msg := NewMessage(OpcodeDisplay, payload)
		encoded, err := msg.Encode()
		if err != nil {
			t.Fatalf("Go encode failed: %v", err)
		}

		// Verify header structure
		if !bytes.Equal(encoded[0:4], []byte{0x57, 0x45, 0x42, 0x53}) {
			t.Error("Magic bytes mismatch")
		}
		if encoded[4] != 1 {
			t.Error("Version mismatch")
		}
		if encoded[5] != byte(OpcodeDisplay) {
			t.Error("Opcode mismatch")
		}

		// Verify payload length
		jsPayloadLen := int(encoded[14])<<24 | int(encoded[15])<<16 | int(encoded[16])<<8 | int(encoded[17])
		if jsPayloadLen != len(payload) {
			t.Errorf("Payload length mismatch: got %d, want %d", jsPayloadLen, len(payload))
		}

		// Decode in Go
		var decoded Message
		if err := decoded.Decode(encoded); err != nil {
			t.Fatalf("Go decode failed: %v", err)
		}

		if decoded.Opcode != OpcodeDisplay {
			t.Errorf("Opcode mismatch: got %v, want %v", decoded.Opcode, OpcodeDisplay)
		}

		if !bytes.Equal(decoded.Payload, payload) {
			t.Errorf("Payload mismatch: got %v, want %v", decoded.Payload, payload)
		}
	})

	t.Run("roundtrip input message", func(t *testing.T) {
		payload := []byte(`{"type":"keydown","key":"Enter","code":"Enter"}`)
		msg := NewMessage(OpcodeInput, payload)
		encoded, err := msg.Encode()
		if err != nil {
			t.Fatalf("Go encode failed: %v", err)
		}

		var decoded Message
		if err := decoded.Decode(encoded); err != nil {
			t.Fatalf("Go decode failed: %v", err)
		}

		if decoded.Opcode != OpcodeInput {
			t.Errorf("Opcode mismatch: got %v, want %v", decoded.Opcode, OpcodeInput)
		}

		if !bytes.Equal(decoded.Payload, payload) {
			t.Errorf("Payload mismatch: got %v, want %v", decoded.Payload, payload)
		}
	})

	t.Run("roundtrip connect message", func(t *testing.T) {
		payload := []byte(`{"clientId":"web-client-123","version":"1.0"}`)
		msg := NewMessage(OpcodeConnect, payload)
		encoded, err := msg.Encode()
		if err != nil {
			t.Fatalf("Go encode failed: %v", err)
		}

		var decoded Message
		if err := decoded.Decode(encoded); err != nil {
			t.Fatalf("Go decode failed: %v", err)
		}

		if decoded.Opcode != OpcodeConnect {
			t.Errorf("Opcode mismatch: got %v, want %v", decoded.Opcode, OpcodeConnect)
		}

		if !bytes.Equal(decoded.Payload, payload) {
			t.Errorf("Payload mismatch: got %v, want %v", decoded.Payload, payload)
		}
	})

	t.Run("roundtrip ping message", func(t *testing.T) {
		msg := NewMessage(OpcodePing, nil)
		encoded, err := msg.Encode()
		if err != nil {
			t.Fatalf("Go encode failed: %v", err)
		}

		var decoded Message
		if err := decoded.Decode(encoded); err != nil {
			t.Fatalf("Go decode failed: %v", err)
		}

		if decoded.Opcode != OpcodePing {
			t.Errorf("Opcode mismatch: got %v, want %v", decoded.Opcode, OpcodePing)
		}

		if decoded.Payload != nil {
			t.Errorf("Expected nil payload, got %v", decoded.Payload)
		}
	})

	t.Run("roundtrip binary payload", func(t *testing.T) {
		payload := make([]byte, 256)
		for i := range payload {
			payload[i] = byte(i)
		}

		msg := NewMessage(OpcodeFileSystem, payload)
		encoded, err := msg.Encode()
		if err != nil {
			t.Fatalf("Go encode failed: %v", err)
		}

		var decoded Message
		if err := decoded.Decode(encoded); err != nil {
			t.Fatalf("Go decode failed: %v", err)
		}

		if !bytes.Equal(decoded.Payload, payload) {
			t.Errorf("Binary payload mismatch")
		}
	})

	t.Run("timestamp precision", func(t *testing.T) {
		// Test that timestamp is stored as 8-byte big-endian
		msg := NewMessage(OpcodeDisplay, []byte("test"))

		encoded, err := msg.Encode()
		if err != nil {
			t.Fatalf("Encode failed: %v", err)
		}

		// Extract timestamp bytes (offsets 6-13)
		timestampBytes := encoded[6:14]
		if len(timestampBytes) != 8 {
			t.Errorf("Expected 8 timestamp bytes, got %d", len(timestampBytes))
		}
	})

	t.Run("all opcodes", func(t *testing.T) {
		opcodes := []Opcode{
			OpcodeDisplay, OpcodeInput, OpcodeFileSystem, OpcodeNetwork,
			OpcodeProcess, OpcodeAuth, OpcodeConnect, OpcodeDisconnect,
			OpcodePing, OpcodePong, OpcodeError,
		}

		for _, opcode := range opcodes {
			msg := NewMessage(opcode, []byte("test payload"))
			encoded, err := msg.Encode()
			if err != nil {
				t.Fatalf("Encode failed for opcode %v: %v", opcode, err)
			}

			var decoded Message
			if err := decoded.Decode(encoded); err != nil {
				t.Fatalf("Decode failed for opcode %v: %v", opcode, err)
			}

			if decoded.Opcode != opcode {
				t.Errorf("Opcode mismatch for %v: got %v, want %v", opcode, decoded.Opcode, opcode)
			}
		}
	})

	t.Run("empty payload roundtrip", func(t *testing.T) {
		msg := NewMessage(OpcodePong, nil)
		encoded, err := msg.Encode()
		if err != nil {
			t.Fatalf("Encode failed: %v", err)
		}

		var decoded Message
		if err := decoded.Decode(encoded); err != nil {
			t.Fatalf("Decode failed: %v", err)
		}

		if decoded.Payload != nil {
			t.Errorf("Expected nil payload, got %v", decoded.Payload)
		}
	})

	t.Run("large payload 1MB", func(t *testing.T) {
		payload := make([]byte, 1024*1024)
		for i := range payload {
			payload[i] = byte(i % 256)
		}

		msg := NewMessage(OpcodeFileSystem, payload)
		encoded, err := msg.Encode()
		if err != nil {
			t.Fatalf("Encode failed: %v", err)
		}

		if len(encoded) != HeaderSize+len(payload) {
			t.Errorf("Encoded size mismatch: got %d, want %d", len(encoded), HeaderSize+len(payload))
		}

		var decoded Message
		if err := decoded.Decode(encoded); err != nil {
			t.Fatalf("Decode failed: %v", err)
		}

		if !bytes.Equal(decoded.Payload, payload) {
			t.Error("Large payload mismatch")
		}
	})
}

// TestProtocolConstants verifies that all protocol constants are correct.
func TestProtocolConstants(t *testing.T) {
	if MagicBytes != [4]byte{0x57, 0x45, 0x42, 0x53} {
		t.Error("MagicBytes constant mismatch")
	}

	if ProtocolVersion != 1 {
		t.Errorf("ProtocolVersion mismatch: got %d, want 1", ProtocolVersion)
	}

	if HeaderSize != 18 {
		t.Errorf("HeaderSize mismatch: got %d, want 18", HeaderSize)
	}

	if MaxPayloadSize != 16*1024*1024 {
		t.Errorf("MaxPayloadSize mismatch: got %d, want %d", MaxPayloadSize, 16*1024*1024)
	}
}

// TestMessageEncodeDeterminism verifies that encoding the same message
// produces identical output.
func TestMessageEncodeDeterminism(t *testing.T) {
	payload := []byte("test payload")
	msg := NewMessage(OpcodeDisplay, payload)

	encoded1, err := msg.Encode()
	if err != nil {
		t.Fatalf("First encode failed: %v", err)
	}

	encoded2, err := msg.Encode()
	if err != nil {
		t.Fatalf("Second encode failed: %v", err)
	}

	if !bytes.Equal(encoded1, encoded2) {
		t.Error("Encoding is not deterministic")
	}
}

// TestMaxPayloadAtLimit tests encoding exactly at the maximum payload size.
func TestMaxPayloadAtLimit(t *testing.T) {
	payload := make([]byte, MaxPayloadSize)
	msg := NewMessage(OpcodeDisplay, payload)

	encoded, err := msg.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(encoded) != HeaderSize+int(MaxPayloadSize) {
		t.Errorf("Encoded size mismatch: got %d, want %d", len(encoded), HeaderSize+int(MaxPayloadSize))
	}

	var decoded Message
	if err := decoded.Decode(encoded); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if !bytes.Equal(decoded.Payload, payload) {
		t.Error("Max payload mismatch")
	}
}
