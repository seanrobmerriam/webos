package websocket

import (
	"encoding/binary"
	"io"
)

// FrameReader provides methods for reading WebSocket frames from an io.Reader.
type FrameReader struct {
	r io.Reader
}

// NewFrameReader creates a new FrameReader for reading from r.
func NewFrameReader(r io.Reader) *FrameReader {
	return &FrameReader{r: r}
}

// ReadFrame reads and parses a WebSocket frame from the underlying reader.
// Returns nil if end of stream is reached.
func (fr *FrameReader) ReadFrame() (*Frame, error) {
	// Read the first 2 bytes of the frame header
	header := make([]byte, 2)
	n, err := fr.r.Read(header)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, io.EOF
	}

	frame := &Frame{}

	// Parse first byte: FIN, RSV1-3, Opcode
	frame.Fin = (header[0] & 0x80) != 0
	frame.RSV1 = (header[0] & 0x40) != 0
	frame.RSV2 = (header[0] & 0x20) != 0
	frame.RSV3 = (header[0] & 0x10) != 0
	frame.Opcode = Opcode(header[0] & 0x0F)

	// Parse second byte: MASK, Payload length
	frame.Masked = (header[1] & 0x80) != 0
	payloadLen := uint64(header[1] & 0x7F)

	// Handle extended payload length
	if payloadLen == 126 {
		extLen := make([]byte, 2)
		if _, err := io.ReadFull(fr.r, extLen); err != nil {
			return nil, err
		}
		payloadLen = uint64(binary.BigEndian.Uint16(extLen))
	} else if payloadLen == 127 {
		extLen := make([]byte, 8)
		if _, err := io.ReadFull(fr.r, extLen); err != nil {
			return nil, err
		}
		payloadLen = binary.BigEndian.Uint64(extLen)
	}

	// Check for overflow
	if payloadLen > MaxFramePayloadSize {
		return nil, &FrameError{Err: ErrFrameTooLarge, Opcode: frame.Opcode}
	}

	// Read masking key if present (client-to-server frames must be masked)
	if frame.Masked {
		if _, err := io.ReadFull(fr.r, frame.Mask[:]); err != nil {
			return nil, err
		}
	}

	// Read payload
	if payloadLen > 0 {
		frame.Payload = make([]byte, payloadLen)
		if _, err := io.ReadFull(fr.r, frame.Payload); err != nil {
			return nil, err
		}

		// Unmask payload if necessary
		if frame.Masked {
			frame.unmask()
		}
	} else {
		frame.Payload = nil
	}

	// Validate frame
	if err := frame.Validate(); err != nil {
		return nil, err
	}

	return frame, nil
}

// unmask applies the masking key to the payload.
func (f *Frame) unmask() {
	for i := range f.Payload {
		f.Payload[i] ^= f.Mask[i%4]
	}
}

// ReadMessage reads a complete message, handling fragmentation.
// Returns the frame with the first opcode and the concatenated payload.
func (fr *FrameReader) ReadMessage() (*Frame, error) {
	firstFrame, err := fr.ReadFrame()
	if err != nil {
		return nil, err
	}

	// If this is not a fragmented message, return as-is
	if firstFrame.Fin {
		return firstFrame, nil
	}

	// Handle fragmented message - collect all fragments
	var fragments []*Frame
	fragments = append(fragments, firstFrame)

	// Read continuation frames
	for {
		frame, err := fr.ReadFrame()
		if err != nil {
			return nil, err
		}

		// Continuation frames must have opcode 0
		if frame.Opcode != OpcodeContinuation {
			return nil, &FrameError{Err: ErrInvalidOpcode, Opcode: frame.Opcode}
		}

		fragments = append(fragments, frame)

		if frame.Fin {
			break
		}
	}

	// Concatenate all payloads
	totalLen := 0
	for _, f := range fragments {
		totalLen += len(f.Payload)
	}

	combined := make([]byte, totalLen)
	pos := 0
	for _, f := range fragments {
		copy(combined[pos:], f.Payload)
		pos += len(f.Payload)
	}

	// Return a frame with the first opcode and combined payload
	return &Frame{
		Fin:     true,
		Opcode:  firstFrame.Opcode,
		Payload: combined,
	}, nil
}
