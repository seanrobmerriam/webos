package socket

import (
	"fmt"
	network "net"
	"sync"

	"webos/pkg/netstack/route"
	"webos/pkg/netstack/tcp"
	"webos/pkg/netstack/udp"
)

// Protocol represents the socket protocol.
type Protocol uint8

const (
	ProtocolTCP Protocol = 6
	ProtocolUDP Protocol = 17
)

// SocketType represents the socket type.
type SocketType uint8

const (
	SocketStream SocketType = iota
	SocketDgram
)

// Status represents the socket status.
type Status uint8

const (
	StatusUnconnected Status = iota
	StatusConnecting
	StatusConnected
	StatusListening
	StatusClosing
	StatusClosed
)

// Socket represents a network socket.
type Socket struct {
	ID       uint64
	Protocol Protocol
	Type     SocketType
	Status   Status

	// Address info
	localAddr  network.Addr
	remoteAddr network.Addr

	// TCP specific
	tcpConn *tcp.Connection

	// UDP specific
	udpSocket *udp.Socket

	// Routing
	rt *route.RouteTable

	// Synchronization
	mu       sync.RWMutex
	recvChan chan []byte
	sendChan chan []byte
	closed   bool
}

// NewTCPSocket creates a new TCP socket.
func NewTCPSocket(rt *route.RouteTable) *Socket {
	return &Socket{
		ID:       generateSocketID(),
		Protocol: ProtocolTCP,
		Type:     SocketStream,
		Status:   StatusUnconnected,
		rt:       rt,
		recvChan: make(chan []byte, 100),
		sendChan: make(chan []byte, 100),
	}
}

// NewUDPSocket creates a new UDP socket.
func NewUDPSocket(port uint16, localIP network.IP, rt *route.RouteTable) *Socket {
	return &Socket{
		ID:       generateSocketID(),
		Protocol: ProtocolUDP,
		Type:     SocketDgram,
		Status:   StatusConnected,
		localAddr: &network.UDPAddr{
			IP:   localIP,
			Port: int(port),
		},
		remoteAddr: nil,
		udpSocket:  udp.NewSocket(port, localIP),
		rt:         rt,
		recvChan:   make(chan []byte, 100),
		sendChan:   make(chan []byte, 100),
	}
}

// Connect establishes a TCP connection to the remote address.
func (s *Socket) Connect(addr network.Addr) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status != StatusUnconnected {
		return fmt.Errorf("socket already connected or connecting")
	}

	s.Status = StatusConnecting
	s.remoteAddr = addr

	// Get route to destination
	ip := addr.(*network.TCPAddr).IP
	r := s.rt.Lookup(ip)
	if r == nil {
		return fmt.Errorf("no route to host")
	}

	// Create TCP connection
	connID := tcp.ConnectionID{
		SrcIP:   s.localAddr.(*network.TCPAddr).IP,
		SrcPort: uint16(s.localAddr.(*network.TCPAddr).Port),
		DstIP:   ip,
		DstPort: uint16(addr.(*network.TCPAddr).Port),
	}

	s.tcpConn = tcp.NewConnection(connID, s.localAddr, addr)
	s.Status = StatusConnected

	return nil
}

// Listen puts the socket in listening mode.
func (s *Socket) Listen(backlog int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status != StatusUnconnected {
		return fmt.Errorf("socket must be unconnected to listen")
	}

	s.Status = StatusListening
	return nil
}

// Accept accepts a new connection.
func (s *Socket) Accept() (*Socket, error) {
	s.mu.RLock()
	if s.Status != StatusListening {
		s.mu.RUnlock()
		return nil, fmt.Errorf("socket not listening")
	}
	s.mu.RUnlock()

	// Simplified: return a new connected socket
	conn := NewTCPSocket(s.rt)
	conn.Status = StatusConnected
	return conn, nil
}

// Send sends data on the socket.
func (s *Socket) Send(data []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, fmt.Errorf("socket closed")
	}

	switch s.Protocol {
	case ProtocolTCP:
		if s.tcpConn == nil {
			return 0, fmt.Errorf("not connected")
		}
		// In a real implementation, this would queue the data
		return len(data), nil

	case ProtocolUDP:
		select {
		case s.sendChan <- data:
			return len(data), nil
		default:
			return 0, fmt.Errorf("send buffer full")
		}
	}

	return 0, fmt.Errorf("unsupported protocol")
}

// Recv receives data from the socket.
func (s *Socket) Recv(buf []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, fmt.Errorf("socket closed")
	}

	switch s.Protocol {
	case ProtocolTCP:
		if s.tcpConn == nil {
			return 0, fmt.Errorf("not connected")
		}
		// In a real implementation, this would read from the connection
		return 0, nil

	case ProtocolUDP:
		select {
		case data := <-s.recvChan:
			n := copy(buf, data)
			return n, nil
		default:
			return 0, fmt.Errorf("no data available")
		}
	}

	return 0, fmt.Errorf("unsupported protocol")
}

// Close closes the socket.
func (s *Socket) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("socket already closed")
	}

	s.closed = true
	s.Status = StatusClosed

	switch s.Protocol {
	case ProtocolTCP:
		if s.tcpConn != nil {
			s.tcpConn.State = tcp.StateClosed
		}
	case ProtocolUDP:
		if s.udpSocket != nil {
			close(s.recvChan)
			close(s.sendChan)
		}
	}

	return nil
}

// LocalAddr returns the local address.
func (s *Socket) LocalAddr() network.Addr {
	return s.localAddr
}

// RemoteAddr returns the remote address.
func (s *Socket) RemoteAddr() network.Addr {
	return s.remoteAddr
}

// SetLocalAddr sets the local address.
func (s *Socket) SetLocalAddr(addr network.Addr) {
	s.localAddr = addr
}

// Socket ID counter
var socketIDCounter uint64 = 0
var socketIDMu sync.Mutex

func generateSocketID() uint64 {
	socketIDMu.Lock()
	defer socketIDMu.Unlock()
	socketIDCounter++
	return socketIDCounter
}

// SocketManager manages a collection of sockets.
type SocketManager struct {
	mu      sync.RWMutex
	sockets map[uint64]*Socket
}

// NewSocketManager creates a new socket manager.
func NewSocketManager() *SocketManager {
	return &SocketManager{
		sockets: make(map[uint64]*Socket),
	}
}

// Add adds a socket to the manager.
func (sm *SocketManager) Add(s *Socket) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sockets[s.ID] = s
}

// Get retrieves a socket by ID.
func (sm *SocketManager) Get(id uint64) (*Socket, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.sockets[id]
	return s, ok
}

// Remove removes a socket from the manager.
func (sm *SocketManager) Remove(id uint64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sockets, id)
}

// List returns all socket IDs.
func (sm *SocketManager) List() []uint64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	ids := make([]uint64, 0, len(sm.sockets))
	for id := range sm.sockets {
		ids = append(ids, id)
	}
	return ids
}
