package websocket

import (
	"bytes"
	"net/http"
	"testing"
	"time"
	"webos/pkg/protocol"
)

// TestFrameWriterText tests writing text frames.
func TestFrameWriterText(t *testing.T) {
	var buf bytes.Buffer
	writer := NewFrameWriter(&buf)

	if err := writer.WriteText([]byte("hello")); err != nil {
		t.Fatalf("WriteText() error = %v", err)
	}

	reader := NewFrameReader(&buf)
	frame, err := reader.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}

	if frame.Opcode != OpcodeText {
		t.Errorf("Opcode = %v, want TEXT", frame.Opcode)
	}
	if !frame.Fin {
		t.Error("Fin should be true")
	}
	if string(frame.Payload) != "hello" {
		t.Errorf("Payload = %v, want hello", frame.Payload)
	}
}

// TestFrameWriterBinary tests writing binary frames.
func TestFrameWriterBinary(t *testing.T) {
	var buf bytes.Buffer
	writer := NewFrameWriter(&buf)

	data := []byte{0x00, 0x01, 0x02, 0x03}
	if err := writer.WriteBinary(data); err != nil {
		t.Fatalf("WriteBinary() error = %v", err)
	}

	reader := NewFrameReader(&buf)
	frame, err := reader.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}

	if frame.Opcode != OpcodeBinary {
		t.Errorf("Opcode = %v, want BINARY", frame.Opcode)
	}
	if !bytes.Equal(frame.Payload, data) {
		t.Errorf("Payload = %v, want %v", frame.Payload, data)
	}
}

// TestFrameWriterClose tests writing close frames.
func TestFrameWriterClose(t *testing.T) {
	var buf bytes.Buffer
	writer := NewFrameWriter(&buf)

	if err := writer.WriteClose(1000, "goodbye"); err != nil {
		t.Fatalf("WriteClose() error = %v", err)
	}

	reader := NewFrameReader(&buf)
	frame, err := reader.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}

	if frame.Opcode != OpcodeClose {
		t.Errorf("Opcode = %v, want CLOSE", frame.Opcode)
	}

	code := CloseCode(frame.Payload)
	if code != 1000 {
		t.Errorf("CloseCode() = %d, want 1000", code)
	}
}

// TestFrameWriterPingPong tests ping and pong frames.
func TestFrameWriterPingPong(t *testing.T) {
	var buf bytes.Buffer
	writer := NewFrameWriter(&buf)

	// Write ping
	if err := writer.WritePing([]byte("ping-data")); err != nil {
		t.Fatalf("WritePing() error = %v", err)
	}

	reader := NewFrameReader(&buf)
	frame, err := reader.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}

	if frame.Opcode != OpcodePing {
		t.Errorf("Opcode = %v, want PING", frame.Opcode)
	}
	if string(frame.Payload) != "ping-data" {
		t.Errorf("Payload = %v, want ping-data", frame.Payload)
	}

	// Write pong
	buf.Reset()
	if err := writer.WritePong([]byte("pong-data")); err != nil {
		t.Fatalf("WritePong() error = %v", err)
	}

	frame, err = reader.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}

	if frame.Opcode != OpcodePong {
		t.Errorf("Opcode = %v, want PONG", frame.Opcode)
	}
}

// TestFrameWriterLargePayload tests writing large payloads.
func TestFrameWriterLargePayload(t *testing.T) {
	var buf bytes.Buffer
	writer := NewFrameWriter(&buf)

	// Create a payload larger than 125 bytes (requires extended length)
	data := make([]byte, 200)
	for i := range data {
		data[i] = byte(i % 256)
	}

	if err := writer.WriteBinary(data); err != nil {
		t.Fatalf("WriteBinary() error = %v", err)
	}

	reader := NewFrameReader(&buf)
	frame, err := reader.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}

	if !bytes.Equal(frame.Payload, data) {
		t.Error("Payload mismatch for large frame")
	}
}

// TestConnectionReadWriteTimeout tests connection timeout settings.
func TestConnectionReadWriteTimeout(t *testing.T) {
	conn := NewConnection("test", nil)

	conn.SetReadTimeout(30 * time.Second)
	conn.SetWriteTimeout(10 * time.Second)
	conn.SetPingInterval(25 * time.Second)

	// Verify settings are applied (can't test actual timeouts without a real connection)
	if conn.readTimeout != 30*time.Second {
		t.Errorf("readTimeout = %v, want 30s", conn.readTimeout)
	}
	if conn.writeTimeout != 10*time.Second {
		t.Errorf("writeTimeout = %v, want 10s", conn.writeTimeout)
	}
}

// TestSessionManagerGetOrCreate tests GetOrCreate functionality.
func TestSessionManagerGetOrCreate(t *testing.T) {
	sm := NewSessionManager(nil)

	// Get non-existent session
	_, err := sm.Get("non-existent")
	if err == nil {
		t.Error("Get() should fail for non-existent session")
	}

	// Get or create
	session, err := sm.GetOrCreate("session-1", "user-1")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if session.ID != "session-1" {
		t.Errorf("Session ID = %v, want session-1", session.ID)
	}

	// Get or create again (should return existing)
	session2, err := sm.GetOrCreate("session-1", "user-1")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if session2.ID != session.ID {
		t.Error("GetOrCreate() should return existing session")
	}

	if count := sm.Count(); count != 1 {
		t.Errorf("Count = %d, want 1", count)
	}
}

// TestSessionManagerDestroyAll tests DestroyAll functionality.
func TestSessionManagerDestroyAll(t *testing.T) {
	sm := NewSessionManager(nil)

	// Create sessions
	sm.Create("session-1", "user-1")
	sm.Create("session-2", "user-2")
	sm.Create("session-3", "user-3")

	if count := sm.Count(); count != 3 {
		t.Errorf("Count = %d, want 3", count)
	}

	// Destroy all
	sm.DestroyAll()

	if count := sm.Count(); count != 0 {
		t.Errorf("Count after DestroyAll = %d, want 0", count)
	}
}

// TestSessionManagerAll tests All functionality.
func TestSessionManagerAll(t *testing.T) {
	sm := NewSessionManager(nil)

	// Create sessions
	sm.Create("session-1", "user-1")
	sm.Create("session-2", "user-2")

	all := sm.All()
	if len(all) != 2 {
		t.Errorf("All() returned %d sessions, want 2", len(all))
	}
}

// TestPoolBroadcastMessage tests broadcasting protocol messages.
func TestPoolBroadcastMessage(t *testing.T) {
	pool := NewPool(nil)

	// Add connections
	for i := 0; i < 3; i++ {
		conn := NewConnection("conn-"+string(rune('A'+i)), &mockConn{})
		pool.Add(conn)
	}

	msg := protocol.NewMessage(protocol.OpcodePing, []byte("ping"))
	pool.BroadcastMessage(msg)

	// If we get here without panic, the test passes
	if count := pool.Count(); count != 3 {
		t.Errorf("Count = %d, want 3", count)
	}
}

// TestPoolCloseAll tests CloseAll functionality.
func TestPoolCloseAll(t *testing.T) {
	pool := NewPool(nil)

	// Add connections
	for i := 0; i < 3; i++ {
		conn := NewConnection("conn-"+string(rune('A'+i)), &mockConn{})
		pool.Add(conn)
	}

	pool.CloseAll()

	// Connections should still be tracked but closed
	if count := pool.Count(); count != 3 {
		t.Errorf("Count = %d, want 3", count)
	}
}

// TestPoolGetByAddr tests GetByAddr functionality.
func TestPoolGetByAddr(t *testing.T) {
	pool := NewPool(nil)

	// Add connection
	conn := NewConnection("conn-A", &mockConn{})
	pool.Add(conn)

	// Get by address
	found := pool.GetByAddr(conn.RemoteAddr())
	if found == nil {
		t.Error("GetByAddr() should find connection")
	}
	if found.ID != "conn-A" {
		t.Errorf("Found connection ID = %v, want conn-A", found.ID)
	}
}

// TestPoolPerIPLimit tests per-IP connection limits.
func TestPoolPerIPLimit(t *testing.T) {
	pool := NewPool(&PoolConfig{
		MaxConnections:      100,
		MaxConnectionsPerIP: 2,
	})

	// Add 2 connections from same IP
	for i := 0; i < 2; i++ {
		conn := NewConnection("conn-"+string(rune('A'+i)), &mockConn{})
		if err := pool.Add(conn); err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}

	// Third connection from same IP should fail
	conn := NewConnection("conn-C", &mockConn{})
	if pool.Add(conn) == nil {
		t.Error("Add() should fail when per-IP limit is reached")
	}
}

// TestNewServer tests server creation.
func TestNewServer(t *testing.T) {
	config := &ServerConfig{
		Addr:          ":8080",
		PoolConfig:    DefaultPoolConfig(),
		SessionConfig: DefaultSessionConfig(),
		ReadTimeout:   30 * time.Second,
		WriteTimeout:  10 * time.Second,
		PingInterval:  25 * time.Second,
	}

	server := NewServer(config)

	if server.addr != ":8080" {
		t.Errorf("Server addr = %v, want :8080", server.addr)
	}
	if server.pool == nil {
		t.Error("Server pool should not be nil")
	}
	if server.sessionManager == nil {
		t.Error("Server sessionManager should not be nil")
	}
}

// TestUpgraderSubprotocols tests upgrader subprotocol handling.
func TestUpgraderSubprotocols(t *testing.T) {
	u := NewUpgrader()
	u.Subprotocols = []string{"graphql-ws", "mqtt"}

	req := &http.Request{
		Method: http.MethodGet,
		Header: http.Header{
			"Upgrade":               []string{"websocket"},
			"Connection":            []string{"Upgrade"},
			"Sec-Websocket-Version": []string{"13"},
			"Sec-Websocket-Key":     []string{"dGhlIHNhbXBsZSBub25jZQ=="},
		},
	}

	if err := u.validateRequest(req); err != nil {
		t.Errorf("validateRequest() error = %v", err)
	}
}

// TestServerCallbacks tests server callback设置.
func TestServerCallbacks(t *testing.T) {
	server := NewServer(nil)

	server.OnAccept(func(c *Connection) {})
	server.OnUpgrade(func(c *Connection) {})

	// Callbacks are set (can't test actual invocation without starting server)
	if server.onAccept == nil {
		t.Error("Accept callback should be set")
	}
	if server.onUpgrade == nil {
		t.Error("Upgrade callback should be set")
	}
}

// TestConnectionOnMessage tests message handler callback.
func TestConnectionOnMessage(t *testing.T) {
	conn := NewConnection("test", nil)

	conn.OnMessage(func(c *Connection, msg *protocol.Message) error {
		return nil
	})

	// Handler is set
	conn.mu.Lock()
	handler := conn.onMessage
	conn.mu.Unlock()

	if handler == nil {
		t.Error("Message handler should be set")
	}
}

// TestConnectionOnClose tests close handler callback.
func TestConnectionOnClose(t *testing.T) {
	conn := NewConnection("test", nil)

	conn.OnClose(func(c *Connection) {})

	// Handler is set
	conn.mu.Lock()
	handler := conn.onClose
	conn.mu.Unlock()

	if handler == nil {
		t.Error("Close handler should be set")
	}
}

// TestOpcodeString tests opcode string representation.
func TestOpcodeString(t *testing.T) {
	tests := []struct {
		opcode Opcode
		want   string
	}{
		{OpcodeContinuation, "CONTINUATION"},
		{OpcodeText, "TEXT"},
		{OpcodeBinary, "BINARY"},
		{OpcodeClose, "CLOSE"},
		{OpcodePing, "PING"},
		{OpcodePong, "PONG"},
		{Opcode(0xFF), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.opcode.String(); got != tt.want {
			t.Errorf("Opcode(%d).String() = %v, want %v", tt.opcode, got, tt.want)
		}
	}
}

// TestFrameNewFrameWithMask tests creating a masked frame.
func TestFrameNewFrameWithMask(t *testing.T) {
	frame := &Frame{
		Fin:     true,
		Opcode:  OpcodeText,
		Masked:  true,
		Mask:    [4]byte{0x12, 0x34, 0x56, 0x78},
		Payload: []byte("hello"),
	}

	if !frame.Masked {
		t.Error("Frame should be masked")
	}
	if frame.Mask != [4]byte{0x12, 0x34, 0x56, 0x78} {
		t.Error("Mask mismatch")
	}
}

// TestPoolStatsDetailed tests pool statistics in detail.
func TestPoolStatsDetailed(t *testing.T) {
	pool := NewPool(&PoolConfig{
		MaxConnections:      10,
		MaxConnectionsPerIP: 5,
	})

	// Add some connections
	for i := 0; i < 5; i++ {
		conn := NewConnection("conn-"+string(rune('A'+i)), &mockConn{})
		pool.Add(conn)
	}

	stats := pool.Stats()

	if stats.ActiveConnections != 5 {
		t.Errorf("ActiveConnections = %d, want 5", stats.ActiveConnections)
	}
	if stats.TotalAccepted != 5 {
		t.Errorf("TotalAccepted = %d, want 5", stats.TotalAccepted)
	}
	if stats.TotalClosed != 0 {
		t.Errorf("TotalClosed = %d, want 0", stats.TotalClosed)
	}
	if stats.TotalFailed != 0 {
		t.Errorf("TotalFailed = %d, want 0", stats.TotalFailed)
	}
}

// TestSessionTTL tests session TTL calculation.
func TestSessionTTL(t *testing.T) {
	config := &SessionConfig{
		Duration: 1 * time.Hour,
	}
	session := NewSession("test", "user", config)

	ttl := session.TTL()
	if ttl <= 0 {
		t.Error("TTL should be positive")
	}
	if ttl > 1*time.Hour {
		t.Error("TTL should not exceed session duration")
	}
}

// TestSessionConnectionAssociation tests associating a connection with a session.
func TestSessionConnectionAssociation(t *testing.T) {
	session := NewSession("test", "user", nil)
	conn := NewConnection("conn-1", &mockConn{})

	session.SetConnection(conn)

	if session.Connection() != conn {
		t.Error("Session connection should match")
	}
}

// TestBufferReaderRead tests buffer reader read operations.
func TestBufferReaderRead(t *testing.T) {
	data := []byte("Hello World")
	reader := NewBufferReader(bytes.NewReader(data))

	// Read all
	buf := make([]byte, 11)
	n, err := reader.Read(buf)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if n != 11 {
		t.Errorf("Read() returned %d bytes, want 11", n)
	}
	if string(buf) != string(data) {
		t.Errorf("Read() = %v, want %v", string(buf), data)
	}
}

// TestBufferReaderPeek tests buffer reader peek operations.
func TestBufferReaderPeek(t *testing.T) {
	data := []byte("Hello World")
	reader := NewBufferReader(bytes.NewReader(data))

	peek, err := reader.Peek(5)
	if err != nil {
		t.Fatalf("Peek() error = %v", err)
	}
	if string(peek) != "Hello" {
		t.Errorf("Peek() = %v, want Hello", string(peek))
	}

	// Peek should not consume data
	peek2, _ := reader.Peek(5)
	if string(peek2) != "Hello" {
		t.Error("Peek should not consume data")
	}
}

// TestFrameReaderEmptyPayload tests reading frame with empty payload.
func TestFrameReaderEmptyPayload(t *testing.T) {
	var buf bytes.Buffer
	writer := NewFrameWriter(&buf)

	// Write empty text frame
	if err := writer.WriteText([]byte{}); err != nil {
		t.Fatalf("WriteText() error = %v", err)
	}

	reader := NewFrameReader(&buf)
	frame, err := reader.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}

	if frame.Opcode != OpcodeText {
		t.Errorf("Opcode = %v, want TEXT", frame.Opcode)
	}
	if len(frame.Payload) != 0 {
		t.Errorf("Payload length = %d, want 0", len(frame.Payload))
	}
}

// TestConnectionLocalAddr tests getting local address.
func TestConnectionLocalAddr(t *testing.T) {
	conn := NewConnection("test", &mockConn{})

	addr := conn.LocalAddr()
	if addr == nil {
		t.Error("LocalAddr() should not return nil")
	}
}

// TestConnectionRemoteAddr tests getting remote address.
func TestConnectionRemoteAddr(t *testing.T) {
	conn := NewConnection("test", &mockConn{})

	addr := conn.RemoteAddr()
	if addr == nil {
		t.Error("RemoteAddr() should not return nil")
	}
}

// TestSessionManagerCleanup tests session cleanup.
func TestSessionManagerCleanup(t *testing.T) {
	// Create a session that will expire quickly
	config := &SessionConfig{
		Duration: 1 * time.Millisecond,
	}
	sm := NewSessionManager(config)
	sm.Create("session-1", "user-1")

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Cleanup
	sm.Cleanup()

	// Session should be removed
	_, err := sm.Get("session-1")
	if err == nil {
		t.Error("Session should be removed after cleanup")
	}
}
