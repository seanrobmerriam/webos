package websocket

import (
	"bytes"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestOpcode tests opcode validation.
func TestOpcode(t *testing.T) {
	tests := []struct {
		name     string
		opcode   Opcode
		wantVal  bool
		wantCtrl bool
		wantData bool
	}{
		{"Continuation", OpcodeContinuation, true, false, true},
		{"Text", OpcodeText, true, false, true},
		{"Binary", OpcodeBinary, true, false, true},
		{"Close", OpcodeClose, true, true, false},
		{"Ping", OpcodePing, true, true, false},
		{"Pong", OpcodePong, true, true, false},
		{"Invalid", Opcode(0xFF), false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opcode.IsValid(); got != tt.wantVal {
				t.Errorf("Opcode.IsValid() = %v, want %v", got, tt.wantVal)
			}
			if got := tt.opcode.IsControl(); got != tt.wantCtrl {
				t.Errorf("Opcode.IsControl() = %v, want %v", got, tt.wantCtrl)
			}
			if got := tt.opcode.IsData(); got != tt.wantData {
				t.Errorf("Opcode.IsData() = %v, want %v", got, tt.wantData)
			}
		})
	}
}

// TestFrameValidation tests frame validation.
func TestFrameValidation(t *testing.T) {
	tests := []struct {
		name    string
		frame   *Frame
		wantErr bool
	}{
		{
			name: "Valid text frame",
			frame: &Frame{
				Fin:     true,
				Opcode:  OpcodeText,
				Payload: []byte("hello"),
			},
			wantErr: false,
		},
		{
			name: "Valid binary frame",
			frame: &Frame{
				Fin:     true,
				Opcode:  OpcodeBinary,
				Payload: []byte{0x00, 0x01, 0x02},
			},
			wantErr: false,
		},
		{
			name: "Invalid opcode",
			frame: &Frame{
				Fin:     true,
				Opcode:  Opcode(0xFF),
				Payload: []byte("test"),
			},
			wantErr: true,
		},
		{
			name: "Control frame fragmented",
			frame: &Frame{
				Fin:     false,
				Opcode:  OpcodeClose,
				Payload: []byte("close"),
			},
			wantErr: true,
		},
		{
			name: "Control frame too long",
			frame: &Frame{
				Fin:     true,
				Opcode:  OpcodePing,
				Payload: make([]byte, 126), // > 125
			},
			wantErr: true,
		},
		{
			name: "Valid close frame",
			frame: &Frame{
				Fin:     true,
				Opcode:  OpcodeClose,
				Payload: make([]byte, 2), // Just status code
			},
			wantErr: false,
		},
		{
			name: "Reserved bits set",
			frame: &Frame{
				Fin:     true,
				RSV1:    true,
				Opcode:  OpcodeText,
				Payload: []byte("test"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.frame.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Frame.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestFrameRoundTrip tests frame encoding and decoding round-trip.
func TestFrameRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		opcode  Opcode
		payload []byte
		fin     bool
	}{
		{"Text message", OpcodeText, []byte("Hello, World!"), true},
		{"Binary message", OpcodeBinary, []byte{0x00, 0x01, 0x02, 0x03}, true},
		{"Empty text", OpcodeText, []byte{}, true},
		{"Ping frame", OpcodePing, []byte("ping"), true},
		{"Pong frame", OpcodePong, []byte("pong"), true},
		{"Close frame", OpcodeClose, []byte{0x03, 0xE8}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create frame
			original := &Frame{
				Fin:     tt.fin,
				Opcode:  tt.opcode,
				Payload: tt.payload,
			}

			// Write to buffer
			var buf bytes.Buffer
			writer := NewFrameWriter(&buf)
			if err := writer.WriteFrame(original); err != nil {
				t.Fatalf("WriteFrame() error = %v", err)
			}

			// Read from buffer
			reader := NewFrameReader(&buf)
			decoded, err := reader.ReadFrame()
			if err != nil {
				t.Fatalf("ReadFrame() error = %v", err)
			}

			// Compare
			if decoded.Fin != original.Fin {
				t.Errorf("Fin = %v, want %v", decoded.Fin, original.Fin)
			}
			if decoded.Opcode != original.Opcode {
				t.Errorf("Opcode = %v, want %v", decoded.Opcode, original.Opcode)
			}
			if !bytes.Equal(decoded.Payload, original.Payload) {
				t.Errorf("Payload = %v, want %v", decoded.Payload, original.Payload)
			}
		})
	}
}

// TestHandshakeKeyGeneration tests Sec-WebSocket-Key generation and validation.
func TestHandshakeKeyGeneration(t *testing.T) {
	// Generate a key
	key, err := GenerateSecKey()
	if err != nil {
		t.Fatalf("GenerateSecKey() error = %v", err)
	}

	// Validate key format (base64, 16 bytes decoded)
	if len(key) < 16 {
		t.Errorf("Key length = %d, want >= 16", len(key))
	}
}

// TestAcceptKeyGeneration tests Sec-WebSocket-Accept key generation.
func TestAcceptKeyGeneration(t *testing.T) {
	requestKey := "dGhlIHNhbXBsZSBub25jZQ=="
	expectedAccept := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="

	acceptKey := generateAcceptKey(requestKey)
	if acceptKey != expectedAccept {
		t.Errorf("generateAcceptKey() = %v, want %v", acceptKey, expectedAccept)
	}
}

// TestVerifyAcceptKey tests accept key verification.
func TestVerifyAcceptKey(t *testing.T) {
	requestKey := "dGhlIHNhbXBsZSBub25jZQ=="
	expectedAccept := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="

	if !VerifyAcceptKey(requestKey, expectedAccept) {
		t.Error("VerifyAcceptKey() = false, want true")
	}

	if VerifyAcceptKey(requestKey, "wrong-key") {
		t.Error("VerifyAcceptKey() = true, want false")
	}
}

// TestCreateHandshakeResponse tests handshake response creation.
func TestCreateHandshakeResponse(t *testing.T) {
	secKey := "dGhlIHNhbXBsZSBub25jZQ=="
	expectedAccept := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="

	resp, err := CreateHandshakeResponse(secKey, nil)
	if err != nil {
		t.Fatalf("CreateHandshakeResponse() error = %v", err)
	}

	// Check that response contains expected headers
	if !strings.Contains(resp, "HTTP/1.1 101 Switching Protocols") {
		t.Error("Response missing status line")
	}
	if !strings.Contains(resp, "Upgrade: websocket") {
		t.Error("Response missing Upgrade header")
	}
	if !strings.Contains(resp, "Connection: Upgrade") {
		t.Error("Response missing Connection header")
	}
	if !strings.Contains(resp, "Sec-WebSocket-Accept: "+expectedAccept) {
		t.Error("Response missing or incorrect Accept header")
	}
}

// TestReadHTTPRequest tests HTTP request parsing.
func TestReadHTTPRequest(t *testing.T) {
	requestData := "GET /websocket HTTP/1.1\r\n" +
		"Host: localhost:8080\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"\r\n"

	req, err := ReadHTTPRequest([]byte(requestData))
	if err != nil {
		t.Fatalf("ReadHTTPRequest() error = %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Method = %v, want GET", req.Method)
	}
	if req.URL.Path != "/websocket" {
		t.Errorf("Path = %v, want /websocket", req.URL.Path)
	}
	if req.Header.Get("Upgrade") != "websocket" {
		t.Errorf("Upgrade = %v, want websocket", req.Header.Get("Upgrade"))
	}
	if req.Header.Get("Sec-WebSocket-Key") != "dGhlIHNhbXBsZSBub25jZQ==" {
		t.Errorf("Sec-WebSocket-Key = %v, want dGhlIHNhbXBsZSBub25jZQ==", req.Header.Get("Sec-WebSocket-Key"))
	}
}

// TestUpgraderValidation tests HTTP upgrade request validation.
func TestUpgraderValidation(t *testing.T) {
	u := NewUpgrader()

	tests := []struct {
		name    string
		request *http.Request
		wantErr bool
	}{
		{
			name: "Valid request",
			request: &http.Request{
				Method: http.MethodGet,
				Header: http.Header{
					"Upgrade":               []string{"websocket"},
					"Connection":            []string{"Upgrade"},
					"Sec-Websocket-Version": []string{"13"},
					"Sec-Websocket-Key":     []string{"dGhlIHNhbXBsZSBub25jZQ=="},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid method",
			request: &http.Request{
				Method: http.MethodPost,
				Header: http.Header{
					"Upgrade":               []string{"websocket"},
					"Connection":            []string{"Upgrade"},
					"Sec-Websocket-Version": []string{"13"},
					"Sec-Websocket-Key":     []string{"dGhlIHNhbXBsZSBub25jZQ=="},
				},
			},
			wantErr: true,
		},
		{
			name: "Missing upgrade header",
			request: &http.Request{
				Method: http.MethodGet,
				Header: http.Header{
					"Connection":            []string{"Upgrade"},
					"Sec-Websocket-Version": []string{"13"},
					"Sec-Websocket-Key":     []string{"dGhlIHNhbXBsZSBub25jZQ=="},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid version",
			request: &http.Request{
				Method: http.MethodGet,
				Header: http.Header{
					"Upgrade":               []string{"websocket"},
					"Connection":            []string{"Upgrade"},
					"Sec-Websocket-Version": []string{"8"},
					"Sec-Websocket-Key":     []string{"dGhlIHNhbXBsZSBub25jZQ=="},
				},
			},
			wantErr: true,
		},
		{
			name: "Missing key",
			request: &http.Request{
				Method: http.MethodGet,
				Header: http.Header{
					"Upgrade":               []string{"websocket"},
					"Connection":            []string{"Upgrade"},
					"Sec-WebSocket-Version": []string{"13"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := u.validateRequest(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSessionManager tests session management.
func TestSessionManager(t *testing.T) {
	sm := NewSessionManager(nil)

	// Create session
	session, err := sm.Create("session-1", "user-1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if session.ID != "session-1" {
		t.Errorf("Session ID = %v, want session-1", session.ID)
	}
	if session.UserID != "user-1" {
		t.Errorf("User ID = %v, want user-1", session.UserID)
	}

	// Get session
	got, err := sm.Get("session-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != session.ID {
		t.Errorf("Got session ID = %v, want %v", got.ID, session.ID)
	}

	// Count
	if count := sm.Count(); count != 1 {
		t.Errorf("Count = %d, want 1", count)
	}

	// Destroy session
	if err := sm.Destroy("session-1"); err != nil {
		t.Fatalf("Destroy() error = %v", err)
	}

	// Get should fail
	_, err = sm.Get("session-1")
	if err == nil {
		t.Error("Get() should fail after Destroy()")
	}
}

// TestSessionData tests session data operations.
func TestSessionData(t *testing.T) {
	session := NewSession("test", "user", nil)

	// Set data
	session.Set("key1", "value1")
	session.Set("key2", 123)

	// Get data
	val1, ok1 := session.Get("key1")
	if !ok1 {
		t.Error("Get() should return ok=true for existing key")
	}
	if val1 != "value1" {
		t.Errorf("Get() = %v, want value1", val1)
	}

	val2, ok2 := session.Get("key2")
	if !ok2 {
		t.Error("Get() should return ok=true for existing key")
	}
	if val2 != 123 {
		t.Errorf("Get() = %v, want 123", val2)
	}

	// Delete
	session.Delete("key1")
	_, ok := session.Get("key1")
	if ok {
		t.Error("Get() should return ok=false after Delete()")
	}

	// Clear
	session.Set("key3", "value3")
	session.Clear()
	_, ok = session.Get("key3")
	if ok {
		t.Error("Get() should return ok=false after Clear()")
	}
}

// TestSessionExpiration tests session expiration.
func TestSessionExpiration(t *testing.T) {
	config := &SessionConfig{
		Duration: 1 * time.Second,
	}
	session := NewSession("test", "user", config)

	// Should not be expired initially
	if session.IsExpired() {
		t.Error("Session should not be expired initially")
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Should be expired now
	if !session.IsExpired() {
		t.Error("Session should be expired after TTL")
	}
}

// TestPoolConnectionManagement tests connection pool operations.
func TestPoolConnectionManagement(t *testing.T) {
	pool := NewPool(&PoolConfig{
		MaxConnections:      10,
		MaxConnectionsPerIP: 5,
	})

	// Add connections
	for i := 0; i < 5; i++ {
		conn := NewConnection("conn-"+string(rune('A'+i)), nil)
		if err := pool.Add(conn); err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}

	if count := pool.Count(); count != 5 {
		t.Errorf("Count = %d, want 5", count)
	}

	// Get connection
	conn, ok := pool.Get("conn-A")
	if !ok {
		t.Error("Get() should return ok=true for existing connection")
	}
	if conn.ID != "conn-A" {
		t.Errorf("Got connection ID = %v, want conn-A", conn.ID)
	}

	// Remove connection
	pool.Remove(conn)
	if count := pool.Count(); count != 4 {
		t.Errorf("Count after remove = %d, want 4", count)
	}

	// Get should fail
	_, ok = pool.Get("conn-A")
	if ok {
		t.Error("Get() should return ok=false after Remove()")
	}
}

// TestPoolConnectionLimit tests connection limits.
func TestPoolConnectionLimit(t *testing.T) {
	pool := NewPool(&PoolConfig{
		MaxConnections:      3,
		MaxConnectionsPerIP: 5,
	})

	// Add max connections with non-nil connections
	for i := 0; i < 3; i++ {
		conn := NewConnection("conn-"+string(rune('A'+i)), &mockConn{})
		if err := pool.Add(conn); err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}

	// Try to add more - should fail
	conn := NewConnection("conn-D", &mockConn{})
	err := pool.Add(conn)
	if err == nil {
		t.Error("Add() should fail when connection limit is reached")
	}
	if err != ErrConnectionLimit {
		t.Errorf("Add() = %v, want ErrConnectionLimit", err)
	}
}

// mockConn is a mock network connection for testing.
type mockConn struct{}

func (m *mockConn) Read(b []byte) (n int, err error)  { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error) { return len(b), nil }
func (m *mockConn) Close() error                      { return nil }
func (m *mockConn) LocalAddr() net.Addr               { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080} }
func (m *mockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}
}
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

// TestPoolStats tests pool statistics.
func TestPoolStats(t *testing.T) {
	pool := NewPool(nil)

	// Add some connections with non-nil connections
	for i := 0; i < 3; i++ {
		conn := NewConnection("conn-"+string(rune('A'+i)), &mockConn{})
		pool.Add(conn)
	}

	stats := pool.Stats()
	if stats.ActiveConnections != 3 {
		t.Errorf("ActiveConnections = %d, want 3", stats.ActiveConnections)
	}
	if stats.TotalAccepted != 3 {
		t.Errorf("TotalAccepted = %d, want 3", stats.TotalAccepted)
	}
}

// TestNewFrame tests frame creation.
func TestNewFrame(t *testing.T) {
	frame := NewFrame(OpcodeText, []byte("hello"), true)

	if frame.Fin != true {
		t.Errorf("Fin = %v, want true", frame.Fin)
	}
	if frame.Opcode != OpcodeText {
		t.Errorf("Opcode = %v, want TEXT", frame.Opcode)
	}
	if string(frame.Payload) != "hello" {
		t.Errorf("Payload = %v, want hello", frame.Payload)
	}
}

// TestFrameError tests frame error handling.
func TestFrameError(t *testing.T) {
	err := &FrameError{
		Err:    ErrInvalidOpcode,
		Opcode: OpcodeText,
	}

	if err.Error() != "invalid opcode" {
		t.Errorf("Error() = %v, want 'invalid opcode'", err.Error())
	}
}

// TestBufferReader tests buffer reader operations.
func TestBufferReader(t *testing.T) {
	data := "Hello\nWorld\n"
	reader := NewBufferReader(strings.NewReader(data))

	// Read line
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("ReadString() error = %v", err)
	}
	if line != "Hello\n" {
		t.Errorf("ReadString() = %v, want Hello\\n", line)
	}

	// Peek
	peek, err := reader.Peek(5)
	if err != nil {
		t.Fatalf("Peek() error = %v", err)
	}
	if string(peek) != "World" {
		t.Errorf("Peek() = %v, want World", peek)
	}
}

// TestHandshakeError tests handshake error handling.
func TestHandshakeError(t *testing.T) {
	err := &HandshakeError{
		Err:    ErrNotWebSocketRequest,
		Status: http.StatusMethodNotAllowed,
	}

	if err.Error() != "not a WebSocket request" {
		t.Errorf("Error() = %v, want 'not a WebSocket request'", err.Error())
	}
	if err.Status != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", err.Status, http.StatusMethodNotAllowed)
	}
}

// TestCloseCodeAndReason tests close frame parsing.
func TestCloseCodeAndReason(t *testing.T) {
	// Test valid close frame
	payload := []byte{0x03, 0xE8, 'g', 'o', 'o', 'd', 'b', 'y', 'e'}
	code := CloseCode(payload)
	if code != 1000 {
		t.Errorf("CloseCode() = %d, want 1000", code)
	}

	reason := CloseReason(payload)
	if reason != "goodbye" {
		t.Errorf("CloseReason() = %v, want goodbye", reason)
	}

	// Test just code
	payload = []byte{0x03, 0xE8}
	code = CloseCode(payload)
	if code != 1000 {
		t.Errorf("CloseCode() = %d, want 1000", code)
	}

	reason = CloseReason(payload)
	if reason != "" {
		t.Errorf("CloseReason() = %v, want empty", reason)
	}
}

// TestGetRequestedSubprotocol tests subprotocol extraction.
func TestGetRequestedSubprotocol(t *testing.T) {
	req := &http.Request{
		Header: http.Header{
			"Sec-Websocket-Protocol": []string{"graphql-ws"},
		},
	}

	subprotocol := GetRequestedSubprotocol(req)
	if subprotocol != "graphql-ws" {
		t.Errorf("GetRequestedSubprotocol() = %v, want graphql-ws", subprotocol)
	}
}

// TestBroadcast tests pool broadcasting.
func TestBroadcast(t *testing.T) {
	pool := NewPool(nil)

	// Add connections
	for i := 0; i < 3; i++ {
		conn := NewConnection("conn-"+string(rune('A'+i)), nil)
		pool.Add(conn)
	}

	// Broadcast
	sent := 0
	pool.Broadcast(func(conn *Connection) error {
		sent++
		return nil
	})

	if sent != 3 {
		t.Errorf("Broadcast sent to %d connections, want 3", sent)
	}
}

// TestSessionExtend tests session extension.
func TestSessionExtend(t *testing.T) {
	config := &SessionConfig{
		Duration: 1 * time.Second,
	}
	session := NewSession("test", "user", config)

	initialTTL := session.TTL()

	// Extend
	session.Extend(5 * time.Second)

	newTTL := session.TTL()
	if newTTL <= initialTTL {
		t.Errorf("TTL should increase after Extend()")
	}
}
