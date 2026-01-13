package protocol

import (
	"encoding/binary"
	"io"
)

// Codec provides additional utilities for working with the protocol.
type Codec struct {
	// buf is the internal buffer for reading/writing.
	buf []byte
	// pos is the current position in the buffer.
	pos int
}

// NewCodec creates a new Codec with the given buffer.
func NewCodec(buf []byte) *Codec {
	return &Codec{
		buf: buf,
		pos: 0,
	}
}

// Reset resets the codec position to the beginning.
func (c *Codec) Reset() {
	c.pos = 0
}

// Remaining returns the number of bytes remaining in the buffer.
func (c *Codec) Remaining() int {
	return len(c.buf) - c.pos
}

// ReadByte reads a single byte from the buffer.
func (c *Codec) ReadByte() (byte, error) {
	if c.pos >= len(c.buf) {
		return 0, io.EOF
	}
	b := c.buf[c.pos]
	c.pos++
	return b, nil
}

// WriteByte writes a single byte to the buffer.
func (c *Codec) WriteByte(b byte) error {
	if c.pos >= len(c.buf) {
		return io.EOF
	}
	c.buf[c.pos] = b
	c.pos++
	return nil
}

// ReadUint16 reads a 16-bit big-endian unsigned integer.
func (c *Codec) ReadUint16() (uint16, error) {
	if c.pos+2 > len(c.buf) {
		return 0, io.EOF
	}
	v := binary.BigEndian.Uint16(c.buf[c.pos : c.pos+2])
	c.pos += 2
	return v, nil
}

// WriteUint16 writes a 16-bit big-endian unsigned integer.
func (c *Codec) WriteUint16(v uint16) error {
	if c.pos+2 > len(c.buf) {
		return io.EOF
	}
	binary.BigEndian.PutUint16(c.buf[c.pos:c.pos+2], v)
	c.pos += 2
	return nil
}

// ReadUint32 reads a 32-bit big-endian unsigned integer.
func (c *Codec) ReadUint32() (uint32, error) {
	if c.pos+4 > len(c.buf) {
		return 0, io.EOF
	}
	v := binary.BigEndian.Uint32(c.buf[c.pos : c.pos+4])
	c.pos += 4
	return v, nil
}

// WriteUint32 writes a 32-bit big-endian unsigned integer.
func (c *Codec) WriteUint32(v uint32) error {
	if c.pos+4 > len(c.buf) {
		return io.EOF
	}
	binary.BigEndian.PutUint32(c.buf[c.pos:c.pos+4], v)
	c.pos += 4
	return nil
}

// ReadUint64 reads a 64-bit big-endian unsigned integer.
func (c *Codec) ReadUint64() (uint64, error) {
	if c.pos+8 > len(c.buf) {
		return 0, io.EOF
	}
	v := binary.BigEndian.Uint64(c.buf[c.pos : c.pos+8])
	c.pos += 8
	return v, nil
}

// WriteUint64 writes a 64-bit big-endian unsigned integer.
func (c *Codec) WriteUint64(v uint64) error {
	if c.pos+8 > len(c.buf) {
		return io.EOF
	}
	binary.BigEndian.PutUint64(c.buf[c.pos:c.pos+8], v)
	c.pos += 8
	return nil
}

// ReadBytes reads exactly n bytes from the buffer.
func (c *Codec) ReadBytes(n int) ([]byte, error) {
	if c.pos+n > len(c.buf) {
		return nil, io.EOF
	}
	b := make([]byte, n)
	copy(b, c.buf[c.pos:c.pos+n])
	c.pos += n
	return b, nil
}

// WriteBytes writes bytes to the buffer.
func (c *Codec) WriteBytes(b []byte) error {
	if c.pos+len(b) > len(c.buf) {
		return io.EOF
	}
	copy(c.buf[c.pos:], b)
	c.pos += len(b)
	return nil
}

// ReadString reads a null-terminated string.
func (c *Codec) ReadString() (string, error) {
	start := c.pos
	for c.pos < len(c.buf) && c.buf[c.pos] != 0 {
		c.pos++
	}
	if c.pos >= len(c.buf) {
		return "", io.EOF
	}
	c.pos++ // skip null terminator
	return string(c.buf[start : c.pos-1]), nil
}

// WriteString writes a null-terminated string.
func (c *Codec) WriteString(s string) error {
	if c.pos+len(s)+1 > len(c.buf) {
		return io.EOF
	}
	copy(c.buf[c.pos:], s)
	c.buf[c.pos+len(s)] = 0
	c.pos += len(s) + 1
	return nil
}
