package h2

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Frame types as defined in RFC 7540.
const (
	FrameData         = 0x0
	FrameHeaders      = 0x1
	FramePriority     = 0x2
	FrameRSTStream    = 0x3
	FrameSettings     = 0x4
	FramePushPromise  = 0x5
	FramePing         = 0x6
	FrameGoAway       = 0x7
	FrameWindowUpdate = 0x8
	FrameContinuation = 0x9
)

// Frame flags.
const (
	FlagDataEndStream         = 0x1
	FlagDataPadded            = 0x8
	FlagHeadersEndStream      = 0x1
	FlagHeadersEndHeaders     = 0x4
	FlagHeadersPadded         = 0x8
	FlagHeadersPriority       = 0x20
	FlagSettingsAck           = 0x1
	FlagPingAck               = 0x1
	FlagPushPromiseEndHeaders = 0x4
	FlagPushPromisePadded     = 0x8
)

// Settings parameters.
const (
	SettingsHeaderTableSize      = 0x1
	SettingsEnablePush           = 0x2
	SettingsMaxConcurrentStreams = 0x3
	SettingsInitialWindowSize    = 0x4
	SettingsMaxFrameSize         = 0x5
	SettingsMaxHeaderListSize    = 0x6
)

// Default values.
const (
	DefaultMaxFrameSize = 16384
	MaxMaxFrameSize     = 16777216
	InitialWindowSize   = 65535
	MaxControlFrameSize = 16384
	MinMaxFrameSize     = 16384
	ConnectionPreface   = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
)

// Error codes.
const (
	ErrCodeNo               = 0x0
	ErrCodeProtocolError    = 0x1
	ErrCodeInternalError    = 0x2
	ErrCodeFlowControlError = 0x3
	ErrCodeSettingsTimeout  = 0x4
	ErrCodeStreamClosed     = 0x5
	ErrCodeFrameSizeError   = 0x6
	ErrCodeRefusedStream    = 0x7
	ErrCodeCancel           = 0x8
	ErrCodeCompressionError = 0x9
	ErrCodeConnectError     = 0xa
)

// FrameHeader is the 9-byte frame header.
type FrameHeader struct {
	Length   uint32
	Type     uint8
	Flags    uint8
	StreamID uint32
}

// ReadFrameHeader reads a frame header from the reader.
func ReadFrameHeader(r io.Reader) (FrameHeader, error) {
	var h FrameHeader
	data := make([]byte, 9)
	if _, err := io.ReadFull(r, data); err != nil {
		return h, err
	}
	h.Length = uint32(data[0])<<16 | uint32(data[1])<<8 | uint32(data[2])
	h.Type = data[3]
	h.Flags = data[4]
	h.StreamID = binary.BigEndian.Uint32(data[5:9]) & 0x7fffffff
	return h, nil
}

// WriteFrameHeader writes a frame header to the writer.
func WriteFrameHeader(w io.Writer, h FrameHeader) error {
	data := make([]byte, 9)
	data[0] = byte(h.Length >> 16)
	data[1] = byte(h.Length >> 8)
	data[2] = byte(h.Length)
	data[3] = h.Type
	data[4] = h.Flags
	binary.BigEndian.PutUint32(data[5:9], h.StreamID&0x7fffffff)
	_, err := w.Write(data)
	return err
}

// String returns a string representation.
func (h FrameHeader) String() string {
	return fmt.Sprintf("FrameHeader{len=%d, type=%d, flags=0x%02x, stream=%d}",
		h.Length, h.Type, h.Flags, h.StreamID)
}

// DataFrame represents a DATA frame.
type DataFrame struct {
	frameHeader FrameHeader
	Data        []byte
	Padding     []byte
}

// FrameHeader returns the frame header.
func (f *DataFrame) FrameHeader() FrameHeader {
	return f.frameHeader
}

// ReadDataFrame reads a DATA frame.
func ReadDataFrame(h FrameHeader, r io.Reader) (*DataFrame, error) {
	length := int(h.Length)
	padded := h.Flags&FlagDataPadded != 0
	df := &DataFrame{frameHeader: h}
	if padded {
		length--
		pad := make([]byte, 1)
		if _, err := io.ReadFull(r, pad); err != nil {
			return nil, err
		}
		df.Padding = make([]byte, pad[0])
		length -= int(pad[0])
	}
	df.Data = make([]byte, length)
	if _, err := io.ReadFull(r, df.Data); err != nil {
		return nil, err
	}
	return df, nil
}

// Write writes the DATA frame.
func (f *DataFrame) Write(w io.Writer) error {
	length := len(f.Data) + len(f.Padding)
	if len(f.Padding) > 0 {
		length++
		f.frameHeader.Flags |= FlagDataPadded
	}
	f.frameHeader.Length = uint32(length)
	if err := WriteFrameHeader(w, f.frameHeader); err != nil {
		return err
	}
	if len(f.Padding) > 0 {
		if _, err := w.Write([]byte{byte(len(f.Padding))}); err != nil {
			return err
		}
	}
	if _, err := w.Write(f.Data); err != nil {
		return err
	}
	if _, err := w.Write(f.Padding); err != nil {
		return err
	}
	return nil
}

// HeadersFrame represents a HEADERS frame.
type HeadersFrame struct {
	frameHeader   FrameHeader
	BlockFragment []byte
	Padding       []byte
	Priority      *PrioritySpec
}

// FrameHeader returns the frame header.
func (f *HeadersFrame) FrameHeader() FrameHeader {
	return f.frameHeader
}

// ReadHeadersFrame reads a HEADERS frame.
func ReadHeadersFrame(h FrameHeader, r io.Reader) (*HeadersFrame, error) {
	length := int(h.Length)
	hf := &HeadersFrame{frameHeader: h}
	if h.Flags&FlagHeadersPadded != 0 {
		length--
		pad := make([]byte, 1)
		if _, err := io.ReadFull(r, pad); err != nil {
			return nil, err
		}
		hf.Padding = make([]byte, pad[0])
		length -= int(pad[0])
	}
	if h.Flags&FlagHeadersPriority != 0 {
		length -= 5
		ps := make([]byte, 5)
		if _, err := io.ReadFull(r, ps); err != nil {
			return nil, err
		}
		hf.Priority = &PrioritySpec{
			StreamDep: binary.BigEndian.Uint32(ps[:4]) & 0x7fffffff,
			Weight:    uint8(ps[4]) + 1,
		}
	}
	hf.BlockFragment = make([]byte, length)
	if _, err := io.ReadFull(r, hf.BlockFragment); err != nil {
		return nil, err
	}
	return hf, nil
}

// Write writes the HEADERS frame.
func (f *HeadersFrame) Write(w io.Writer) error {
	length := len(f.BlockFragment) + len(f.Padding)
	if len(f.Padding) > 0 {
		length++
		f.frameHeader.Flags |= FlagHeadersPadded
	}
	if f.Priority != nil {
		length += 5
		f.frameHeader.Flags |= FlagHeadersPriority
	}
	f.frameHeader.Length = uint32(length)
	if err := WriteFrameHeader(w, f.frameHeader); err != nil {
		return err
	}
	if len(f.Padding) > 0 {
		if _, err := w.Write([]byte{byte(len(f.Padding))}); err != nil {
			return err
		}
	}
	if f.Priority != nil {
		data := make([]byte, 5)
		binary.BigEndian.PutUint32(data[:4], f.Priority.StreamDep&0x7fffffff)
		data[4] = f.Priority.Weight - 1
		if _, err := w.Write(data); err != nil {
			return err
		}
	}
	if _, err := w.Write(f.BlockFragment); err != nil {
		return err
	}
	if _, err := w.Write(f.Padding); err != nil {
		return err
	}
	return nil
}

// PrioritySpec specifies stream priority.
type PrioritySpec struct {
	StreamDep uint32
	Weight    uint8
}

// SettingsFrame represents a SETTINGS frame.
type SettingsFrame struct {
	frameHeader FrameHeader
	Params      []Setting
	Ack         bool
}

// Setting represents a single setting parameter.
type Setting struct {
	Identifier uint16
	Value      uint32
}

// FrameHeader returns the frame header.
func (f *SettingsFrame) FrameHeader() FrameHeader {
	return f.frameHeader
}

// ReadSettingsFrame reads a SETTINGS frame.
func ReadSettingsFrame(h FrameHeader, r io.Reader) (*SettingsFrame, error) {
	sf := &SettingsFrame{
		frameHeader: h,
		Ack:         h.Flags&FlagSettingsAck != 0,
	}
	if sf.Ack {
		return sf, nil
	}
	count := h.Length / 6
	sf.Params = make([]Setting, count)
	data := make([]byte, count*6)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	for i := uint32(0); i < count; i++ {
		off := i * 6
		sf.Params[i] = Setting{
			Identifier: binary.BigEndian.Uint16(data[off:]),
			Value:      binary.BigEndian.Uint32(data[off+2:]),
		}
	}
	return sf, nil
}

// Write writes the SETTINGS frame.
func (f *SettingsFrame) Write(w io.Writer) error {
	length := len(f.Params) * 6
	f.frameHeader.Length = uint32(length)
	if f.Ack {
		f.frameHeader.Flags |= FlagSettingsAck
	}
	if err := WriteFrameHeader(w, f.frameHeader); err != nil {
		return err
	}
	data := make([]byte, len(f.Params)*6)
	for i, p := range f.Params {
		off := i * 6
		binary.BigEndian.PutUint16(data[off:], p.Identifier)
		binary.BigEndian.PutUint32(data[off+2:], p.Value)
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	return nil
}

// RSTStreamFrame represents an RST_STREAM frame.
type RSTStreamFrame struct {
	frameHeader FrameHeader
	ErrorCode   uint32
}

// FrameHeader returns the frame header.
func (f *RSTStreamFrame) FrameHeader() FrameHeader {
	return f.frameHeader
}

// ReadRSTStreamFrame reads an RST_STREAM frame.
func ReadRSTStreamFrame(h FrameHeader, r io.Reader) (*RSTStreamFrame, error) {
	data := make([]byte, 4)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return &RSTStreamFrame{
		frameHeader: h,
		ErrorCode:   binary.BigEndian.Uint32(data),
	}, nil
}

// Write writes the RST_STREAM frame.
func (f *RSTStreamFrame) Write(w io.Writer) error {
	f.frameHeader.Length = 4
	if err := WriteFrameHeader(w, f.frameHeader); err != nil {
		return err
	}
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, f.ErrorCode)
	_, err := w.Write(data)
	return err
}

// PingFrame represents a PING frame.
type PingFrame struct {
	frameHeader FrameHeader
	Data        [8]byte
	Ack         bool
}

// FrameHeader returns the frame header.
func (f *PingFrame) FrameHeader() FrameHeader {
	return f.frameHeader
}

// ReadPingFrame reads a PING frame.
func ReadPingFrame(h FrameHeader, r io.Reader) (*PingFrame, error) {
	pf := &PingFrame{
		frameHeader: h,
		Ack:         h.Flags&FlagPingAck != 0,
	}
	if !pf.Ack {
		if _, err := io.ReadFull(r, pf.Data[:]); err != nil {
			return nil, err
		}
	}
	return pf, nil
}

// Write writes the PING frame.
func (f *PingFrame) Write(w io.Writer) error {
	if f.Ack {
		f.frameHeader.Flags |= FlagPingAck
	}
	if err := WriteFrameHeader(w, f.frameHeader); err != nil {
		return err
	}
	if !f.Ack {
		if _, err := w.Write(f.Data[:]); err != nil {
			return err
		}
	}
	return nil
}

// GoAwayFrame represents a GOAWAY frame.
type GoAwayFrame struct {
	frameHeader  FrameHeader
	LastStreamID uint32
	ErrorCode    uint32
	DebugData    []byte
}

// FrameHeader returns the frame header.
func (f *GoAwayFrame) FrameHeader() FrameHeader {
	return f.frameHeader
}

// ReadGoAwayFrame reads a GOAWAY frame.
func ReadGoAwayFrame(h FrameHeader, r io.Reader) (*GoAwayFrame, error) {
	data := make([]byte, 8)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	gf := &GoAwayFrame{
		frameHeader:  h,
		LastStreamID: binary.BigEndian.Uint32(data[:4]) & 0x7fffffff,
		ErrorCode:    binary.BigEndian.Uint32(data[4:8]),
	}
	debugLen := int(h.Length) - 8
	if debugLen > 0 {
		gf.DebugData = make([]byte, debugLen)
		if _, err := io.ReadFull(r, gf.DebugData); err != nil {
			return nil, err
		}
	}
	return gf, nil
}

// Write writes the GOAWAY frame.
func (f *GoAwayFrame) Write(w io.Writer) error {
	length := 8 + len(f.DebugData)
	f.frameHeader.Length = uint32(length)
	if err := WriteFrameHeader(w, f.frameHeader); err != nil {
		return err
	}
	data := make([]byte, 8)
	binary.BigEndian.PutUint32(data[:4], f.LastStreamID&0x7fffffff)
	binary.BigEndian.PutUint32(data[4:8], f.ErrorCode)
	if _, err := w.Write(data); err != nil {
		return err
	}
	if _, err := w.Write(f.DebugData); err != nil {
		return err
	}
	return nil
}

// WindowUpdateFrame represents a WINDOW_UPDATE frame.
type WindowUpdateFrame struct {
	frameHeader FrameHeader
	Increment   uint32
}

// FrameHeader returns the frame header.
func (f *WindowUpdateFrame) FrameHeader() FrameHeader {
	return f.frameHeader
}

// ReadWindowUpdateFrame reads a WINDOW_UPDATE frame.
func ReadWindowUpdateFrame(h FrameHeader, r io.Reader) (*WindowUpdateFrame, error) {
	data := make([]byte, 4)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return &WindowUpdateFrame{
		frameHeader: h,
		Increment:   binary.BigEndian.Uint32(data) & 0x7fffffff,
	}, nil
}

// Write writes the WINDOW_UPDATE frame.
func (f *WindowUpdateFrame) Write(w io.Writer) error {
	f.frameHeader.Length = 4
	if err := WriteFrameHeader(w, f.frameHeader); err != nil {
		return err
	}
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, f.Increment&0x7fffffff)
	_, err := w.Write(data)
	return err
}

// ContinuationFrame represents a CONTINUATION frame.
type ContinuationFrame struct {
	frameHeader   FrameHeader
	BlockFragment []byte
}

// FrameHeader returns the frame header.
func (f *ContinuationFrame) FrameHeader() FrameHeader {
	return f.frameHeader
}

// ReadContinuationFrame reads a CONTINUATION frame.
func ReadContinuationFrame(h FrameHeader, r io.Reader) (*ContinuationFrame, error) {
	cf := &ContinuationFrame{frameHeader: h}
	cf.BlockFragment = make([]byte, h.Length)
	if _, err := io.ReadFull(r, cf.BlockFragment); err != nil {
		return nil, err
	}
	return cf, nil
}

// Write writes the CONTINUATION frame.
func (f *ContinuationFrame) Write(w io.Writer) error {
	f.frameHeader.Length = uint32(len(f.BlockFragment))
	if err := WriteFrameHeader(w, f.frameHeader); err != nil {
		return err
	}
	_, err := w.Write(f.BlockFragment)
	return err
}

// PushPromiseFrame represents a PUSH_PROMISE frame.
type PushPromiseFrame struct {
	frameHeader    FrameHeader
	BlockFragment  []byte
	Padding        []byte
	PromisedStream uint32
}

// FrameHeader returns the frame header.
func (f *PushPromiseFrame) FrameHeader() FrameHeader {
	return f.frameHeader
}

// ReadPushPromiseFrame reads a PUSH_PROMISE frame.
func ReadPushPromiseFrame(h FrameHeader, r io.Reader) (*PushPromiseFrame, error) {
	length := int(h.Length)
	ppf := &PushPromiseFrame{frameHeader: h}
	if h.Flags&FlagPushPromisePadded != 0 {
		length--
		pad := make([]byte, 1)
		if _, err := io.ReadFull(r, pad); err != nil {
			return nil, err
		}
		ppf.Padding = make([]byte, pad[0])
		length -= int(pad[0])
	}
	length -= 4
	ps := make([]byte, 4)
	if _, err := io.ReadFull(r, ps); err != nil {
		return nil, err
	}
	ppf.PromisedStream = binary.BigEndian.Uint32(ps) & 0x7fffffff
	ppf.BlockFragment = make([]byte, length)
	if _, err := io.ReadFull(r, ppf.BlockFragment); err != nil {
		return nil, err
	}
	return ppf, nil
}

// Write writes the PUSH_PROMISE frame.
func (f *PushPromiseFrame) Write(w io.Writer) error {
	length := 4 + len(f.BlockFragment) + len(f.Padding)
	if len(f.Padding) > 0 {
		length++
		f.frameHeader.Flags |= FlagPushPromisePadded
	}
	f.frameHeader.Length = uint32(length)
	if err := WriteFrameHeader(w, f.frameHeader); err != nil {
		return err
	}
	if len(f.Padding) > 0 {
		if _, err := w.Write([]byte{byte(len(f.Padding))}); err != nil {
			return err
		}
	}
	ps := make([]byte, 4)
	binary.BigEndian.PutUint32(ps, f.PromisedStream&0x7fffffff)
	if _, err := w.Write(ps); err != nil {
		return err
	}
	if _, err := w.Write(f.BlockFragment); err != nil {
		return err
	}
	if _, err := w.Write(f.Padding); err != nil {
		return err
	}
	return nil
}
