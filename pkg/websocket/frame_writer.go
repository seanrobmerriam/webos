package websocket

import (
	"encoding/binary"
	"io"
	"math/rand"
)

// FrameWriter provides methods for writing WebSocket frames to an io.Writer.
type FrameWriter struct {
	w io.Writer
}

// NewFrameWriter creates a new FrameWriter for writing to w.
func NewFrameWriter(w io.Writer) *FrameWriter {
	return &FrameWriter{w: w}
}

// WriteFrame writes a WebSocket frame to the underlying writer.
func (fw *FrameWriter) WriteFrame(frame *Frame) error {
	// Validate frame before writing
	if err := frame.Validate(); err != nil {
		return err
	}

	// Calculate frame size
	headerSize := 2
	if len(frame.Payload) > 65535 {
		headerSize += 8
	} else if len(frame.Payload) > 125 {
		headerSize += 2
	}

	// Generate masking key for server-to-client frames
	var mask [4]byte
	if frame.Masked {
		// Note: Server-to-client frames may be masked per RFC 6455
		// Some implementations prefer unmasked frames for simplicity
		// We'll generate a random mask as per spec
		rand.Read(mask[:])
	}

	frameSize := headerSize
	if frame.Masked {
		frameSize += 4
	}
	frameSize += len(frame.Payload)

	// Build frame
	buf := make([]byte, frameSize)
	pos := 0

	// First byte: FIN, RSV1-3, Opcode
	buf[pos] = 0x00
	if frame.Fin {
		buf[pos] |= 0x80
	}
	if frame.RSV1 {
		buf[pos] |= 0x40
	}
	if frame.RSV2 {
		buf[pos] |= 0x20
	}
	if frame.RSV3 {
		buf[pos] |= 0x10
	}
	buf[pos] |= byte(frame.Opcode & 0x0F)
	pos++

	// Second byte: MASK, Payload length
	buf[pos] = 0x00
	payloadLen := len(frame.Payload)
	if frame.Masked {
		buf[pos] |= 0x80
	}

	if payloadLen <= 125 {
		buf[pos] |= byte(payloadLen)
		pos++
	} else if payloadLen <= 65535 {
		buf[pos] |= 126
		pos++
		binary.BigEndian.PutUint16(buf[pos:pos+2], uint16(payloadLen))
		pos += 2
	} else {
		buf[pos] |= 127
		pos++
		binary.BigEndian.PutUint64(buf[pos:pos+8], uint64(payloadLen))
		pos += 8
	}

	// Write mask if present
	if frame.Masked {
		copy(buf[pos:pos+4], mask[:])
		pos += 4
	}

	// Copy and mask payload
	if payloadLen > 0 {
		copy(buf[pos:], frame.Payload)
		if frame.Masked {
			for i := 0; i < payloadLen; i++ {
				buf[pos+i] ^= mask[i%4]
			}
		}
	}

	// Write to underlying writer
	_, err := fw.w.Write(buf)
	return err
}

// WriteMessage writes a complete message, handling fragmentation if necessary.
func (fw *FrameWriter) WriteMessage(opcode Opcode, payload []byte, maxFragmentSize int) error {
	if len(payload) <= maxFragmentSize {
		// Write as a single frame
		frame := &Frame{
			Fin:     true,
			Opcode:  opcode,
			Payload: payload,
		}
		return fw.WriteFrame(frame)
	}

	// Fragment the message
	pos := 0
	remaining := len(payload)
	firstFrame := true

	for remaining > 0 {
		chunkSize := maxFragmentSize
		if remaining < chunkSize {
			chunkSize = remaining
		}

		var frameOpcode Opcode
		if firstFrame {
			frameOpcode = opcode
			firstFrame = false
		} else {
			frameOpcode = OpcodeContinuation
		}

		frame := &Frame{
			Fin:     chunkSize == remaining,
			Opcode:  frameOpcode,
			Payload: payload[pos : pos+chunkSize],
		}

		if err := fw.WriteFrame(frame); err != nil {
			return err
		}

		pos += chunkSize
		remaining -= chunkSize
	}

	return nil
}

// WriteText writes a text frame.
func (fw *FrameWriter) WriteText(data []byte) error {
	return fw.WriteFrame(&Frame{
		Fin:     true,
		Opcode:  OpcodeText,
		Payload: data,
	})
}

// WriteBinary writes a binary frame.
func (fw *FrameWriter) WriteBinary(data []byte) error {
	return fw.WriteFrame(&Frame{
		Fin:     true,
		Opcode:  OpcodeBinary,
		Payload: data,
	})
}

// WriteClose writes a close frame with the given code and reason.
func (fw *FrameWriter) WriteClose(code uint16, reason string) error {
	payload := make([]byte, 2+len(reason))
	binary.BigEndian.PutUint16(payload[:2], code)
	copy(payload[2:], reason)

	return fw.WriteFrame(&Frame{
		Fin:     true,
		Opcode:  OpcodeClose,
		Payload: payload,
	})
}

// WritePing writes a ping frame with optional payload.
func (fw *FrameWriter) WritePing(payload []byte) error {
	if len(payload) > MaxControlPayloadSize {
		return &FrameError{Err: ErrControlFrameTooLong, Opcode: OpcodePing}
	}

	return fw.WriteFrame(&Frame{
		Fin:     true,
		Opcode:  OpcodePing,
		Payload: payload,
	})
}

// WritePong writes a pong frame with optional payload.
func (fw *FrameWriter) WritePong(payload []byte) error {
	if len(payload) > MaxControlPayloadSize {
		return &FrameError{Err: ErrControlFrameTooLong, Opcode: OpcodePong}
	}

	return fw.WriteFrame(&Frame{
		Fin:     true,
		Opcode:  OpcodePong,
		Payload: payload,
	})
}
