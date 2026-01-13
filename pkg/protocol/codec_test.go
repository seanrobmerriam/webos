package protocol

import (
	"bytes"
	"io"
	"testing"
)

func TestCodecBasic(t *testing.T) {
	// Test that codec works correctly with reset
	buf := make([]byte, 256)
	codec := NewCodec(buf)

	// Write then reset, then read
	err := codec.WriteUint16(0x1234)
	if err != nil {
		t.Fatalf("WriteUint16 failed: %v", err)
	}

	codec.Reset()
	val, err := codec.ReadUint16()
	if err != nil {
		t.Fatalf("ReadUint16 failed: %v", err)
	}
	if val != 0x1234 {
		t.Errorf("Uint16 mismatch: got 0x%04x, want 0x1234", val)
	}
}

func TestCodecWriteThenRead(t *testing.T) {
	buf := make([]byte, 256)
	codec := NewCodec(buf)

	// Write and read consecutively (position advances)
	codec.WriteUint32(0xDEADBEEF)
	val, err := codec.ReadUint32()
	if err != nil {
		t.Fatalf("ReadUint32 failed: %v", err)
	}
	// This reads from position 4, so it will be 0 (since we haven't written there)
	// The test is demonstrating that position advances
	if val != 0 {
		t.Errorf("Expected 0 (reading from position after write), got 0x%08x", val)
	}

	// Now test that we can read what we wrote by resetting
	codec.Reset()
	val2, err := codec.ReadUint32()
	if err != nil {
		t.Fatalf("ReadUint32 after reset failed: %v", err)
	}
	if val2 != 0xDEADBEEF {
		t.Errorf("Uint32 mismatch after reset: got 0x%08x, want 0xDEADBEEF", val2)
	}
}

func TestCodecMultipleWrites(t *testing.T) {
	buf := make([]byte, 256)
	codec := NewCodec(buf)

	codec.WriteByte(0x42)
	codec.WriteByte(0x43)
	codec.WriteByte(0x44)

	// Reset and read all
	codec.Reset()
	b1, _ := codec.ReadByte()
	b2, _ := codec.ReadByte()
	b3, _ := codec.ReadByte()

	if b1 != 0x42 || b2 != 0x43 || b3 != 0x44 {
		t.Errorf("Bytes mismatch: got 0x%02x 0x%02x 0x%02x, want 0x42 0x43 0x44", b1, b2, b3)
	}
}

func TestCodecUint64(t *testing.T) {
	buf := make([]byte, 256)
	codec := NewCodec(buf)

	testVal := uint64(0x123456789ABCDEF0)
	codec.WriteUint64(testVal)

	// Reset and read
	codec.Reset()
	val, err := codec.ReadUint64()
	if err != nil {
		t.Fatalf("ReadUint64 failed: %v", err)
	}
	if val != testVal {
		t.Errorf("Uint64 mismatch: got 0x%016x, want 0x%016x", val, testVal)
	}
}

func TestCodecBytes(t *testing.T) {
	buf := make([]byte, 256)
	codec := NewCodec(buf)

	testData := []byte("hello world")
	codec.WriteBytes(testData)

	// Reset and read
	codec.Reset()
	val, err := codec.ReadBytes(len(testData))
	if err != nil {
		t.Fatalf("ReadBytes failed: %v", err)
	}
	if !bytes.Equal(val, testData) {
		t.Errorf("Bytes mismatch: got %s, want %s", string(val), string(testData))
	}
}

func TestCodecString(t *testing.T) {
	buf := make([]byte, 256)
	codec := NewCodec(buf)

	testStr := "test string"
	codec.WriteString(testStr)

	// Reset and read
	codec.Reset()
	val, err := codec.ReadString()
	if err != nil {
		t.Fatalf("ReadString failed: %v", err)
	}
	if val != testStr {
		t.Errorf("String mismatch: got %s, want %s", val, testStr)
	}
}

func TestCodecBufferOverflow(t *testing.T) {
	smallBuf := make([]byte, 4)
	codec := NewCodec(smallBuf)

	// Write past buffer - this will fail because we can't write 5 bytes to 4-byte buffer
	err := codec.WriteBytes([]byte("hello"))
	if err != io.EOF {
		t.Errorf("Expected EOF for write overflow, got %v", err)
	}
}

func TestCodecRemaining(t *testing.T) {
	buf := make([]byte, 100)
	codec := NewCodec(buf)

	if codec.Remaining() != 100 {
		t.Errorf("Initial remaining mismatch: got %d, want 100", codec.Remaining())
	}

	codec.WriteByte(0x42)
	if codec.Remaining() != 99 {
		t.Errorf("Remaining after write mismatch: got %d, want 99", codec.Remaining())
	}
}

func TestCodecReset(t *testing.T) {
	buf := make([]byte, 256)
	codec := NewCodec(buf)

	codec.WriteByte(0x42)
	codec.WriteByte(0x43)
	codec.Reset()

	if codec.Remaining() != 256 {
		t.Errorf("Remaining after reset mismatch: got %d, want 256", codec.Remaining())
	}

	val, _ := codec.ReadByte()
	if val != 0x42 {
		t.Errorf("First byte after reset mismatch: got 0x%02x, want 0x42", val)
	}
}

func TestCodecEndianness(t *testing.T) {
	buf := make([]byte, 16)
	codec := NewCodec(buf)

	// Write a value
	codec.WriteUint16(0x1234)
	codec.Reset()

	// Check that first byte is 0x12 (big-endian)
	b, _ := codec.ReadByte()
	if b != 0x12 {
		t.Errorf("Big-endian check failed: got 0x%02x, want 0x12", b)
	}
}

func TestCodecConsecutiveOperations(t *testing.T) {
	buf := make([]byte, 256)
	codec := NewCodec(buf)

	// Write multiple values
	codec.WriteUint16(0x1234)
	codec.WriteUint32(0x56789ABC)
	codec.WriteUint64(0xDEF0123456789ABC)

	// Reset to read back in same order
	codec.Reset()

	v1, _ := codec.ReadUint16()
	v2, _ := codec.ReadUint32()
	v3, _ := codec.ReadUint64()

	if v1 != 0x1234 {
		t.Errorf("First value mismatch: got 0x%04x, want 0x1234", v1)
	}
	if v2 != 0x56789ABC {
		t.Errorf("Second value mismatch: got 0x%08x, want 0x56789ABC", v2)
	}
	if v3 != 0xDEF0123456789ABC {
		t.Errorf("Third value mismatch: got 0x%016x, want 0xDEF0123456789ABC", v3)
	}
}

func TestCodecPositionTracking(t *testing.T) {
	buf := make([]byte, 256)
	codec := NewCodec(buf)

	// Check initial position (via Remaining)
	if codec.Remaining() != 256 {
		t.Errorf("Initial remaining mismatch: got %d, want 256", codec.Remaining())
	}

	codec.WriteByte(0x42)
	if codec.Remaining() != 255 {
		t.Errorf("Remaining after 1 byte: got %d, want 255", codec.Remaining())
	}

	codec.WriteBytes([]byte("test"))
	if codec.Remaining() != 251 {
		t.Errorf("Remaining after 5 more bytes: got %d, want 251", codec.Remaining())
	}
}
