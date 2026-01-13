package websocket

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
	"webos/pkg/protocol"
)

// Pool manages WebSocket connections.
type Pool struct {
	// connections stores all active connections.
	connections sync.Map
	// maxConns is the maximum number of connections.
	maxConns int32
	// maxConnsPerIP is the max connections per IP.
	maxConnsPerIP int32
	// connCount tracks total connections.
	connCount int64
	// acceptedCount tracks total accepted connections.
	acceptedCount int64
	// closedCount tracks total closed connections.
	closedCount int64
	// failedCount tracks total failed handshakes.
	failedCount int64
	// ipCounts tracks connections per IP.
	ipCounts map[string]int32
	ipMu     sync.RWMutex
	// onConnect is called when a connection is added.
	onConnect func(*Connection)
	// onDisconnect is called when a connection is removed.
	onDisconnect func(*Connection)
	// cleanupInterval is how often to run cleanup.
	cleanupInterval time.Duration
	// connTimeout is the connection timeout.
	connTimeout time.Duration
	// stopChan stops the cleanup goroutine.
	stopChan chan struct{}
}

// PoolConfig holds pool configuration.
type PoolConfig struct {
	// MaxConnections is the maximum number of connections.
	MaxConnections int
	// MaxConnectionsPerIP is the max connections per IP.
	MaxConnectionsPerIP int
	// CleanupInterval is how often to run cleanup.
	CleanupInterval time.Duration
	// ConnectionTimeout is the timeout for idle connections.
	ConnectionTimeout time.Duration
}

// DefaultPoolConfig returns the default pool configuration.
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxConnections:      10000,
		MaxConnectionsPerIP: 100,
		CleanupInterval:     5 * time.Minute,
		ConnectionTimeout:   30 * time.Minute,
	}
}

// NewPool creates a new connection pool.
func NewPool(config *PoolConfig) *Pool {
	if config == nil {
		config = DefaultPoolConfig()
	}

	return &Pool{
		maxConns:        int32(config.MaxConnections),
		maxConnsPerIP:   int32(config.MaxConnectionsPerIP),
		ipCounts:        make(map[string]int32),
		cleanupInterval: config.CleanupInterval,
		connTimeout:     config.ConnectionTimeout,
		stopChan:        make(chan struct{}),
	}
}

// OnConnect sets the connection callback.
func (p *Pool) OnConnect(fn func(*Connection)) {
	p.onConnect = fn
}

// OnDisconnect sets the disconnection callback.
func (p *Pool) OnDisconnect(fn func(*Connection)) {
	p.onDisconnect = fn
}

// Add adds a connection to the pool.
func (p *Pool) Add(conn *Connection) error {
	if conn == nil {
		return nil
	}

	// Check max connections
	currentCount := atomic.LoadInt64(&p.connCount)
	if currentCount >= int64(p.maxConns) {
		atomic.AddInt64(&p.failedCount, 1)
		return ErrConnectionLimit
	}

	// Check max connections per IP
	ip := p.getIP(conn)
	p.ipMu.Lock()
	count := p.ipCounts[ip]
	if count >= p.maxConnsPerIP {
		p.ipMu.Unlock()
		atomic.AddInt64(&p.failedCount, 1)
		return ErrConnectionLimit
	}
	p.ipCounts[ip]++
	p.ipMu.Unlock()

	// Add to pool
	p.connections.Store(conn.ID, conn)
	atomic.AddInt64(&p.connCount, 1)
	atomic.AddInt64(&p.acceptedCount, 1)

	// Call onConnect callback
	if p.onConnect != nil {
		p.onConnect(conn)
	}

	return nil
}

// Remove removes a connection from the pool.
func (p *Pool) Remove(conn *Connection) {
	if conn == nil {
		return
	}

	p.connections.Delete(conn.ID)

	// Update IP count
	ip := p.getIP(conn)
	p.ipMu.Lock()
	if count, ok := p.ipCounts[ip]; ok && count > 0 {
		p.ipCounts[ip] = count - 1
	}
	p.ipMu.Unlock()

	atomic.AddInt64(&p.connCount, -1)
	atomic.AddInt64(&p.closedCount, 1)

	// Call onDisconnect callback
	if p.onDisconnect != nil {
		p.onDisconnect(conn)
	}
}

// Get gets a connection by ID.
func (p *Pool) Get(id string) (*Connection, bool) {
	conn, ok := p.connections.Load(id)
	if !ok {
		return nil, false
	}
	return conn.(*Connection), true
}

// GetByAddr gets a connection by remote address.
func (p *Pool) GetByAddr(addr net.Addr) *Connection {
	var found *Connection

	p.connections.Range(func(key, value interface{}) bool {
		conn := value.(*Connection)
		if conn.Conn != nil && conn.Conn.RemoteAddr().String() == addr.String() {
			found = conn
			return false
		}
		return true
	})

	return found
}

// Count returns the number of active connections.
func (p *Pool) Count() int {
	return int(atomic.LoadInt64(&p.connCount))
}

// AcceptedCount returns the total number of accepted connections.
func (p *Pool) AcceptedCount() int64 {
	return atomic.LoadInt64(&p.acceptedCount)
}

// ClosedCount returns the total number of closed connections.
func (p *Pool) ClosedCount() int64 {
	return atomic.LoadInt64(&p.closedCount)
}

// FailedCount returns the total number of failed connections.
func (p *Pool) FailedCount() int64 {
	return atomic.LoadInt64(&p.failedCount)
}

// All returns all active connections.
func (p *Pool) All() []*Connection {
	conns := make([]*Connection, 0)
	p.connections.Range(func(key, value interface{}) bool {
		conns = append(conns, value.(*Connection))
		return true
	})
	return conns
}

// Broadcast sends a message to all connections.
func (p *Pool) Broadcast(fn func(*Connection) error) {
	p.connections.Range(func(key, value interface{}) bool {
		conn := value.(*Connection)
		if err := fn(conn); err != nil {
			// Log error but continue broadcasting
		}
		return true
	})
}

// BroadcastMessage sends a protocol message to all connections.
func (p *Pool) BroadcastMessage(msg *protocol.Message) {
	p.Broadcast(func(conn *Connection) error {
		if conn.Conn == nil {
			return nil
		}
		return conn.WriteMessage(msg)
	})
}

// CloseAll closes all connections in the pool.
func (p *Pool) CloseAll() {
	p.connections.Range(func(key, value interface{}) bool {
		conn := value.(*Connection)
		if conn.Conn != nil {
			conn.Close()
		}
		return true
	})
}

// getIP extracts the IP address from a connection.
func (p *Pool) getIP(conn *Connection) string {
	if conn == nil || conn.Conn == nil {
		return "unknown"
	}

	addr := conn.Conn.RemoteAddr()
	if tcpAddr, ok := addr.(*net.TCPAddr); ok {
		return tcpAddr.IP.String()
	}
	return addr.String()
}

// StartCleanup starts the periodic cleanup goroutine.
func (p *Pool) StartCleanup() {
	go func() {
		ticker := time.NewTicker(p.cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				p.Cleanup()
			case <-p.stopChan:
				return
			}
		}
	}()
}

// StopCleanup stops the periodic cleanup goroutine.
func (p *Pool) StopCleanup() {
	close(p.stopChan)
}

// Cleanup removes stale connections.
func (p *Pool) Cleanup() {
	now := time.Now()
	p.connections.Range(func(key, value interface{}) bool {
		conn := value.(*Connection)

		// Check if connection is still connected
		if !conn.IsConnected() {
			p.Remove(conn)
			return true
		}

		// Check timeout
		if p.connTimeout > 0 && now.Sub(conn.CreatedAt) > p.connTimeout {
			if conn.Conn != nil {
				conn.Close()
			}
			p.Remove(conn)
		}

		return true
	})
}

// Statistics returns pool statistics.
type PoolStats struct {
	ActiveConnections int64
	TotalAccepted     int64
	TotalClosed       int64
	TotalFailed       int64
	ConnectionsPerIP  map[string]int32
}

// Stats returns current pool statistics.
func (p *Pool) Stats() PoolStats {
	p.ipMu.RLock()
	counts := make(map[string]int32, len(p.ipCounts))
	for k, v := range p.ipCounts {
		counts[k] = v
	}
	p.ipMu.RUnlock()

	return PoolStats{
		ActiveConnections: atomic.LoadInt64(&p.connCount),
		TotalAccepted:     atomic.LoadInt64(&p.acceptedCount),
		TotalClosed:       atomic.LoadInt64(&p.closedCount),
		TotalFailed:       atomic.LoadInt64(&p.failedCount),
		ConnectionsPerIP:  counts,
	}
}
