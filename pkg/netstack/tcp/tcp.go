package tcp

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	network "net"
	"sync"
	"time"
)

// TCP header length in bytes (without options).
const HeaderLength = 20

// TCP flags.
const (
	FlagFIN uint8 = 1 << iota
	FlagSYN
	FlagRST
	FlagPSH
	FlagACK
	FlagURG
)

// TCP connection states.
const (
	StateClosed uint8 = iota
	StateListen
	StateSynSent
	StateSynReceived
	StateEstablished
	StateFinWait1
	StateFinWait2
	StateClosing
	StateTimeWait
	StateCloseWait
	StateLastAck
)

// TCP socket options.
const (
	OptionMSS         uint8 = 2
	OptionWindowScale uint8 = 3
	OptionSACKPerm    uint8 = 4
	OptionTimestamp   uint8 = 8
)

// Default values.
const (
	DefaultMSS          = 1460
	DefaultWindowSize   = 65535
	MaxWindowScale      = 14
	InitialWindowScale  = 0
	DefaultRTOTimeout   = 1000 * time.Millisecond
	MaxRTOTimeout       = 60 * time.Second
	MinRTOTimeout       = 200 * time.Millisecond
	MaxRetries          = 12
	InitialSSThresh     = 65535
	InitialCWND         = 10 * DefaultMSS
	CongestionWindowMax = 4 * DefaultMSS
)

// Header represents a TCP header.
type Header struct {
	SrcPort    uint16 // Source port
	DstPort    uint16 // Destination port
	SeqNum     uint32 // Sequence number
	AckNum     uint32 // Acknowledgment number
	DataOffset uint8  // Data offset (number of 32-bit words)
	Flags      uint8  // Control flags
	Window     uint16 // Window size
	Checksum   uint16 // Checksum
	Urgent     uint16 // Urgent pointer
	Options    []byte // TCP options
}

// GetPayload returns the segment payload (data after the header).
func (h *Header) GetPayload(data []byte) []byte {
	offset := int(h.DataOffset) * 4
	if offset > len(data) {
		return nil
	}
	return data[offset:]
}

// ParseHeader parses a TCP header from raw bytes.
func ParseHeader(data []byte) (*Header, error) {
	if len(data) < HeaderLength {
		return nil, fmt.Errorf("TCP header too short: %d bytes", len(data))
	}

	h := &Header{
		SrcPort:    binary.BigEndian.Uint16(data[0:2]),
		DstPort:    binary.BigEndian.Uint16(data[2:4]),
		SeqNum:     binary.BigEndian.Uint32(data[4:8]),
		AckNum:     binary.BigEndian.Uint32(data[8:12]),
		DataOffset: data[12] >> 4,
		Flags:      data[13],
		Window:     binary.BigEndian.Uint16(data[14:16]),
		Checksum:   binary.BigEndian.Uint16(data[16:18]),
		Urgent:     binary.BigEndian.Uint16(data[18:20]),
	}

	// Parse options
	optLen := int(h.DataOffset)*4 - HeaderLength
	if optLen > 0 {
		if len(data) < HeaderLength+optLen {
			return nil, fmt.Errorf("TCP options too short")
		}
		h.Options = data[HeaderLength : HeaderLength+optLen]
	}

	return h, nil
}

// Serialize serializes the TCP header to bytes.
func (h *Header) Serialize() []byte {
	offset := int(h.DataOffset) * 4
	buf := make([]byte, offset)

	binary.BigEndian.PutUint16(buf[0:2], h.SrcPort)
	binary.BigEndian.PutUint16(buf[2:4], h.DstPort)
	binary.BigEndian.PutUint32(buf[4:8], h.SeqNum)
	binary.BigEndian.PutUint32(buf[8:12], h.AckNum)
	buf[12] = h.DataOffset << 4
	buf[13] = h.Flags
	binary.BigEndian.PutUint16(buf[14:16], h.Window)
	binary.BigEndian.PutUint16(buf[16:18], h.Checksum)
	binary.BigEndian.PutUint16(buf[18:20], h.Urgent)

	if len(h.Options) > 0 {
		copy(buf[20:], h.Options)
	}

	return buf
}

// CalcChecksum calculates the TCP checksum using pseudo-header.
func (h *Header) CalcChecksum(srcIP, dstIP network.IP, payload []byte) uint16 {
	sum := calcPseudoHeaderChecksum(srcIP, dstIP, 6, uint16(len(h.Serialize())+len(payload)))

	// Sum header and payload
	data := append(h.Serialize(), payload...)
	for i := 0; i < len(data); i += 2 {
		if i+1 < len(data) {
			sum += uint32(data[i])<<8 | uint32(data[i+1])
		} else {
			sum += uint32(data[i]) << 8
		}
	}

	for sum > 0xFFFF {
		sum = (sum >> 16) + (sum & 0xFFFF)
	}

	return ^uint16(sum)
}

func calcPseudoHeaderChecksum(srcIP, dstIP network.IP, protocol uint8, length uint16) uint32 {
	sum := uint32(0)

	// Source IP
	for i := 0; i < 16; i += 2 {
		sum += uint32(srcIP[i])<<8 | uint32(srcIP[i+1])
	}

	// Destination IP
	for i := 0; i < 16; i += 2 {
		sum += uint32(dstIP[i])<<8 | uint32(dstIP[i+1])
	}

	sum += uint32(protocol)
	sum += uint32(length)

	return sum
}

// Segment represents a complete TCP segment.
type Segment struct {
	Header  *Header
	SrcIP   network.IP
	DstIP   network.IP
	Payload []byte
}

// ParseSegment parses a TCP segment from raw bytes.
func ParseSegment(data []byte, srcIP, dstIP network.IP) (*Segment, error) {
	header, err := ParseHeader(data)
	if err != nil {
		return nil, err
	}

	payload := header.GetPayload(data)

	return &Segment{
		Header:  header,
		SrcIP:   srcIP,
		DstIP:   dstIP,
		Payload: payload,
	}, nil
}

// Serialize serializes the segment to bytes.
func (s *Segment) Serialize() []byte {
	// Update checksum
	s.Header.Checksum = s.Header.CalcChecksum(s.SrcIP, s.DstIP, s.Payload)

	// Build full segment
	segment := s.Header.Serialize()
	if len(s.Payload) > 0 {
		segment = append(segment, s.Payload...)
	}

	return segment
}

// NewSegment creates a new TCP segment.
func NewSegment(srcPort, dstPort uint16, srcIP, dstIP network.IP, flags uint8, seq, ack uint32, payload []byte) *Segment {
	h := &Header{
		SrcPort:    srcPort,
		DstPort:    dstPort,
		SeqNum:     seq,
		AckNum:     ack,
		DataOffset: 5, // 20 bytes = 5 * 4
		Flags:      flags,
		Window:     DefaultWindowSize,
		Urgent:     0,
	}

	return &Segment{
		Header:  h,
		SrcIP:   srcIP,
		DstIP:   dstIP,
		Payload: payload,
	}
}

// Connection represents a TCP connection.
type Connection struct {
	ID         ConnectionID
	State      uint8
	LocalAddr  network.Addr
	RemoteAddr network.Addr

	// Sequence numbers
	ISS    uint32 // Initial send sequence number
	IRS    uint32 // Initial receive sequence number
	SND    uint32 // Send next
	SNDUNA uint32 // Send unacknowledged
	SNDWL1 uint32 // Last window update seq
	SNDWL2 uint32 // Last window update ack
	RCV    uint32 // Receive next

	// Flow control
	SNDWND uint16 // Send window
	RCVWND uint16 // Receive window

	// Congestion control
	SSThresh  uint32        // Slow start threshold
	CWND      uint32        // Congestion window
	SRT       time.Duration // Smoothed RTT
	RTTSample time.Duration // RTT sample
	RTO       time.Duration // Retransmission timeout
	Retries   int

	// Timestamps
	TSRecent  uint32
	TSLastAck uint32

	// Reliability
	retransmitQueue map[uint32]*Segment
	sendLock        sync.Mutex

	// Callbacks
	OnReceive func(*Segment)
	OnClose   func()
}

// ConnectionID identifies a TCP connection.
type ConnectionID struct {
	SrcIP   network.IP
	SrcPort uint16
	DstIP   network.IP
	DstPort uint16
}

// String returns a string representation of the connection ID.
func (c *ConnectionID) String() string {
	return fmt.Sprintf("%s:%d -> %s:%d", c.SrcIP, c.SrcPort, c.DstIP, c.DstPort)
}

// NewConnection creates a new TCP connection.
func NewConnection(id ConnectionID, localAddr, remoteAddr network.Addr) *Connection {
	return &Connection{
		ID:              id,
		State:           StateClosed,
		LocalAddr:       localAddr,
		RemoteAddr:      remoteAddr,
		ISS:             rand.Uint32(),
		SSThresh:        InitialSSThresh,
		CWND:            InitialCWND,
		RTO:             DefaultRTOTimeout,
		SNDWND:          DefaultWindowSize,
		RCVWND:          DefaultWindowSize,
		retransmitQueue: make(map[uint32]*Segment),
	}
}

// IsState checks if the connection is in the given state.
func (c *Connection) IsState(state uint8) bool {
	return c.State == state
}

// IsEstablished returns true if the connection is established.
func (c *Connection) IsEstablished() bool {
	return c.State == StateEstablished
}

// Send sends a segment.
func (c *Connection) Send(seg *Segment) error {
	c.sendLock.Lock()
	defer c.sendLock.Unlock()

	c.SND = c.SND + uint32(len(seg.Payload))
	c.retransmitQueue[c.SND] = seg

	return nil
}

// Acknowledge acknowledges a sequence number.
func (c *Connection) Acknowledge(ack uint32) {
	c.sendLock.Lock()
	defer c.sendLock.Unlock()

	if seqLess(c.SNDUNA, ack) && seqLessOrEqual(ack, c.SND) {
		// Remove acknowledged segments from retransmit queue
		for seq := c.SNDUNA; seqLess(seq, ack); seq++ {
			delete(c.retransmitQueue, seq)
		}
		c.SNDUNA = ack
	}
}

// UpdateRTT updates the RTT measurement.
func (c *Connection) UpdateRTT(sample time.Duration) {
	if c.SRT == 0 {
		c.SRT = sample
	} else {
		c.SRT = (7*c.SRT + sample) / 8
	}
	c.RTO = c.SRT*3/2 + 200*time.Millisecond
	if c.RTO > MaxRTOTimeout {
		c.RTO = MaxRTOTimeout
	}
}

// seqLess returns true if a < b (modulo 2^32).
func seqLess(a, b uint32) bool {
	return int32(a-b) < 0
}

// seqLessOrEqual returns true if a <= b (modulo 2^32).
func seqLessOrEqual(a, b uint32) bool {
	return int32(a-b) <= 0
}
