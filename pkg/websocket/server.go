package websocket

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"webos/pkg/protocol"
)

// Server errors.
var (
	ErrServerNotStarted = errors.New("server not started")
	ErrServerClosed     = errors.New("server closed")
)

// MessageHandler handles incoming messages.
type MessageHandler func(conn *Connection, msg *protocol.Message) error

// Server represents a WebSocket server.
type Server struct {
	// addr is the listen address.
	addr string
	// listener is the TCP listener.
	listener net.Listener
	// pool is the connection pool.
	pool *Pool
	// upgrader is the HTTP upgrader.
	upgrader *Upgrader
	// handler is the message handler.
	handler MessageHandler
	// sessionManager manages sessions.
	sessionManager *SessionManager
	// wg waits for server goroutines.
	wg sync.WaitGroup
	// running indicates if the server is running.
	running bool
	// stopChan stops the server.
	stopChan chan struct{}
	// onAccept is called when a connection is accepted.
	onAccept func(*Connection)
	// onUpgrade is called when a connection is upgraded.
	onUpgrade func(*Connection)
	// readTimeout is the read timeout.
	readTimeout time.Duration
	// writeTimeout is the write timeout.
	writeTimeout time.Duration
	// pingInterval is the ping interval.
	pingInterval time.Duration
}

// ServerConfig holds server configuration.
type ServerConfig struct {
	// Addr is the listen address.
	Addr string
	// PoolConfig is the connection pool configuration.
	PoolConfig *PoolConfig
	// SessionConfig is the session configuration.
	SessionConfig *SessionConfig
	// ReadTimeout is the read timeout.
	ReadTimeout time.Duration
	// WriteTimeout is the write timeout.
	WriteTimeout time.Duration
	// PingInterval is the ping interval.
	PingInterval time.Duration
}

// DefaultServerConfig returns the default server configuration.
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Addr:          ":8080",
		PoolConfig:    DefaultPoolConfig(),
		SessionConfig: DefaultSessionConfig(),
		ReadTimeout:   30 * time.Second,
		WriteTimeout:  10 * time.Second,
		PingInterval:  25 * time.Second,
	}
}

// NewServer creates a new WebSocket server.
func NewServer(config *ServerConfig) *Server {
	if config == nil {
		config = DefaultServerConfig()
	}

	return &Server{
		addr:           config.Addr,
		pool:           NewPool(config.PoolConfig),
		upgrader:       NewUpgrader(),
		sessionManager: NewSessionManager(config.SessionConfig),
		stopChan:       make(chan struct{}),
		readTimeout:    config.ReadTimeout,
		writeTimeout:   config.WriteTimeout,
		pingInterval:   config.PingInterval,
	}
}

// SetHandler sets the message handler.
func (s *Server) SetHandler(handler MessageHandler) {
	s.handler = handler
}

// OnAccept sets the accept callback.
func (s *Server) OnAccept(fn func(*Connection)) {
	s.onAccept = fn
}

// OnUpgrade sets the upgrade callback.
func (s *Server) OnUpgrade(fn func(*Connection)) {
	s.onUpgrade = fn
}

// Pool returns the connection pool.
func (s *Server) Pool() *Pool {
	return s.pool
}

// SessionManager returns the session manager.
func (s *Server) SessionManager() *SessionManager {
	return s.sessionManager
}

// Start starts the WebSocket server.
func (s *Server) Start() error {
	if s.running {
		return ErrServerClosed
	}

	// Create listener
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.addr, err)
	}

	s.listener = listener
	s.running = true

	// Start cleanup goroutines
	s.pool.StartCleanup()
	s.sessionManager.StartCleanup()

	// Accept connections
	s.wg.Add(1)
	go s.acceptLoop()

	return nil
}

// Stop stops the WebSocket server.
func (s *Server) Stop() error {
	if !s.running {
		return ErrServerNotStarted
	}

	s.running = false
	close(s.stopChan)

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Close all connections
	s.pool.CloseAll()

	// Wait for goroutines
	s.wg.Wait()

	// Stop cleanup
	s.pool.StopCleanup()
	s.sessionManager.StopCleanup()

	return nil
}

// Addr returns the server address.
func (s *Server) Addr() net.Addr {
	if s.listener != nil {
		return s.listener.Addr()
	}
	return nil
}

// generateConnID generates a unique connection ID.
func generateConnID() (string, error) {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

// acceptLoop accepts incoming connections.
func (s *Server) acceptLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopChan:
			return
		default:
		}

		// Accept connection
		conn, err := s.listener.Accept()
		if err != nil {
			if s.running {
				// Log error but continue
				continue
			}
			return
		}

		// Handle connection
		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection handles a new connection.
func (s *Server) handleConnection(rawConn net.Conn) {
	defer s.wg.Done()

	// Generate connection ID
	connID, err := generateConnID()
	if err != nil {
		rawConn.Close()
		return
	}

	// Create connection
	wsConn := NewConnection(connID, rawConn)
	wsConn.SetReadTimeout(s.readTimeout)
	wsConn.SetWriteTimeout(s.writeTimeout)

	// Call onAccept callback
	if s.onAccept != nil {
		s.onAccept(wsConn)
	}

	// Read HTTP upgrade request
	req, err := wsConn.ReadHTTPRequest()
	if err != nil {
		rawConn.Close()
		atomic.AddInt64(&s.pool.failedCount, 1)
		return
	}

	// Validate upgrade request
	if err := s.upgrader.validateRequest(req); err != nil {
		// Write error response
		http.Error(&responseWriter{rawConn}, err.Error(), http.StatusBadRequest)
		rawConn.Close()
		atomic.AddInt64(&s.pool.failedCount, 1)
		return
	}

	// Get the underlying connection from hijacker
	hijacker, ok := rawConn.(http.Hijacker)
	if !ok {
		rawConn.Close()
		return
	}

	connNet, _, err := hijacker.Hijack()
	if err != nil {
		rawConn.Close()
		atomic.AddInt64(&s.pool.failedCount, 1)
		return
	}

	// Update connection
	wsConn.Conn = connNet

	// Generate accept key
	secKey := req.Header.Get("Sec-WebSocket-Key")
	acceptKey := generateAcceptKey(secKey)

	// Build response
	resp := buildUpgradeResponse(acceptKey, s.upgrader.Subprotocols)

	// Write response
	if _, err := connNet.Write([]byte(resp)); err != nil {
		connNet.Close()
		return
	}

	// Add to pool
	if err := s.pool.Add(wsConn); err != nil {
		connNet.Close()
		return
	}

	// Call onUpgrade callback
	if s.onUpgrade != nil {
		s.onUpgrade(wsConn)
	}

	// Set up message handler
	if s.handler != nil {
		wsConn.OnMessage(s.handler)
	}

	// Start read loop
	wsConn.StartReadLoop()

	// Start heartbeat
	wsConn.StartHeartbeat()

	// Handle close
	wsConn.OnClose(func(c *Connection) {
		s.pool.Remove(c)
	})
}

// responseWriter is a minimal http.ResponseWriter implementation.
type responseWriter struct {
	net.Conn
}

func (rw *responseWriter) Header() http.Header    { return http.Header{} }
func (rw *responseWriter) WriteHeader(status int) {}
func (rw *responseWriter) Write(data []byte) (int, error) {
	return rw.Conn.Write(data)
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Generate connection ID
	connID, err := generateConnID()
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Upgrade connection
	conn, err := s.upgrader.Upgrade(w, r, connID)
	if err != nil {
		// Upgrade already wrote the error response
		return
	}

	// Configure connection
	conn.CreatedAt = time.Now()
	conn.SetReadTimeout(s.readTimeout)
	conn.SetWriteTimeout(s.writeTimeout)

	// Add to pool
	if err := s.pool.Add(conn); err != nil {
		conn.Close()
		return
	}

	// Call onUpgrade callback
	if s.onUpgrade != nil {
		s.onUpgrade(conn)
	}

	// Set up message handler
	if s.handler != nil {
		conn.OnMessage(s.handler)
	}

	// Start read loop
	conn.StartReadLoop()

	// Start heartbeat
	conn.StartHeartbeat()

	// Handle close
	conn.OnClose(func(c *Connection) {
		s.pool.Remove(c)
	})
}

// ListenAndServe starts the server and waits for shutdown.
func (s *Server) ListenAndServe() error {
	if err := s.Start(); err != nil {
		return err
	}

	// Wait for shutdown signal
	<-s.stopChan

	return s.Stop()
}
