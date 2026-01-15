package h2

import (
	"container/heap"
	"fmt"
	"sync"
)

// StreamState represents the state of an HTTP/2 stream.
type StreamState int

const (
	StreamIdle StreamState = iota
	StreamOpen
	StreamHalfClosedLocal
	StreamHalfClosedRemote
	StreamClosed
)

func (s StreamState) String() string {
	switch s {
	case StreamIdle:
		return "idle"
	case StreamOpen:
		return "open"
	case StreamHalfClosedLocal:
		return "half-closed (local)"
	case StreamHalfClosedRemote:
		return "half-closed (remote)"
	case StreamClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// stream represents an HTTP/2 stream.
type stream struct {
	id         uint32
	state      StreamState
	weight     uint8
	dependency uint32
	conn       *serverConn
	sendWindow int32
	recvWindow int32
	data       []byte
	headers    []byte
	err        error
}

// newStream creates a new stream.
func newStream(id uint32, conn *serverConn) *stream {
	return &stream{
		id:         id,
		state:      StreamIdle,
		weight:     16,
		conn:       conn,
		sendWindow: InitialWindowSize,
		recvWindow: InitialWindowSize,
	}
}

// serverConn represents an HTTP/2 server connection.
type serverConn struct {
	streams    map[uint32]*stream
	nextStream uint32
	settings   Settings
	recvWindow int32
	sendWindow int32
	mu         sync.Mutex
	cond       *sync.Cond
	closed     bool
}

// Settings holds HTTP/2 settings.
type Settings struct {
	HeaderTableSize      uint32
	EnablePush           uint32
	MaxConcurrentStreams uint32
	InitialWindowSize    uint32
	MaxFrameSize         uint32
	MaxHeaderListSize    uint32
}

// DefaultSettings returns the default HTTP/2 settings.
func DefaultSettings() Settings {
	return Settings{
		EnablePush:           1,
		MaxConcurrentStreams: 100,
		InitialWindowSize:    InitialWindowSize,
		MaxFrameSize:         DefaultMaxFrameSize,
	}
}

// newServerConn creates a new server connection.
func newServerConn() *serverConn {
	sc := &serverConn{
		streams:    make(map[uint32]*stream),
		nextStream: 1,
		settings:   DefaultSettings(),
		recvWindow: InitialWindowSize,
		sendWindow: InitialWindowSize,
	}
	sc.cond = sync.NewCond(&sc.mu)
	return sc
}

// newStream creates a new stream on the connection.
func (sc *serverConn) newStream(id uint32) *stream {
	s := newStream(id, sc)
	sc.streams[id] = s
	return s
}

// getStream returns the stream with the given ID.
func (sc *serverConn) getStream(id uint32) *stream {
	return sc.streams[id]
}

// removeStream removes a stream from the connection.
func (sc *serverConn) removeStream(id uint32) {
	delete(sc.streams, id)
}

// maxStreamID returns the maximum stream ID.
func (sc *serverConn) maxStreamID() uint32 {
	return sc.nextStream - 1
}

// clientConn represents an HTTP/2 client connection.
type clientConn struct {
	streams    map[uint32]*stream
	nextStream uint32
	settings   Settings
	recvWindow int32
	sendWindow int32
	mu         sync.Mutex
}

// newClientConn creates a new client connection.
func newClientConn() *clientConn {
	return &clientConn{
		streams:    make(map[uint32]*stream),
		nextStream: 1,
		settings:   DefaultSettings(),
		recvWindow: InitialWindowSize,
		sendWindow: InitialWindowSize,
	}
}

// newStream creates a new stream on the connection.
func (cc *clientConn) newStream() uint32 {
	id := cc.nextStream
	cc.nextStream += 2
	return id
}

// StreamPriority represents stream priority.
type StreamPriority struct {
	StreamDep uint32
	Weight    uint8
}

// PriorityQueue implements a priority queue for stream scheduling.
type PriorityQueue []*stream

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	if pq[i].dependency != pq[j].dependency {
		return pq[i].dependency < pq[j].dependency
	}
	return pq[i].weight > pq[j].weight
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*stream))
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

// errStreamClosed is returned when operating on a closed stream.
type errStreamClosed struct {
	id uint32
}

func (e *errStreamClosed) Error() string {
	return fmt.Sprintf("stream %d closed", e.id)
}

// errStreamIdle is returned when operating on an idle stream.
type errStreamIdle struct {
	id uint32
}

func (e *errStreamIdle) Error() string {
	return fmt.Sprintf("stream %d is idle", e.id)
}

// validateStreamID checks if a stream ID is valid.
func validateStreamID(id uint32, isClient bool) bool {
	if id == 0 {
		return false
	}
	if isClient {
		return id%2 == 1
	}
	return id%2 == 0
}

// maxValidStreamID returns the maximum valid stream ID.
func maxValidStreamID(isClient bool) uint32 {
	if isClient {
		return 0x7ffffffe
	}
	return 0x7fffffff
}

// Ensure heap interface is implemented
var _ heap.Interface = (*PriorityQueue)(nil)
