package websocket

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Handshake errors.
var (
	ErrNotWebSocketRequest  = errors.New("not a WebSocket request")
	ErrMissingUpgrade       = errors.New("missing Upgrade header")
	ErrMissingSecKey        = errors.New("missing Sec-WebSocket-Key header")
	ErrInvalidSecKey        = errors.New("invalid Sec-WebSocket-Key header")
	ErrInvalidSecVersion    = errors.New("invalid Sec-WebSocket-Version")
	ErrMissingSecAccept     = errors.New("missing Sec-WebSocket-Accept header")
	ErrSecAcceptMismatch    = errors.New("Sec-WebSocket-Accept mismatch")
	ErrUnsupportedExtension = errors.New("unsupported extension")
	ErrSubprotocolMismatch  = errors.New("subprotocol mismatch")
)

// WebSocket GUID as defined in RFC 6455.
const webSocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Upgrader handles WebSocket upgrade requests.
type Upgrader struct {
	// Subprotocols is a list of supported subprotocols.
	Subprotocols []string
	// ReadBufferSize is the size of the read buffer.
	ReadBufferSize int
	// WriteBufferSize is the size of the write buffer.
	WriteBufferSize int
	// CheckOrigin returns true if the origin is allowed.
	CheckOrigin func(r *http.Request) bool
	// EnableCompression enables per-message compression.
	EnableCompression bool
}

// NewUpgrader creates a new Upgrader with default settings.
func NewUpgrader() *Upgrader {
	return &Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
}

// Upgrade upgrades an HTTP connection to a WebSocket connection.
// The returned Connection contains the upgraded net.Conn.
func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request, connID string) (*Connection, error) {
	// Validate request
	if err := u.validateRequest(r); err != nil {
		return nil, err
	}

	// Get the underlying connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("http.ResponseWriter does not support Hijacker")
	}

	conn, _, err := hijacker.Hijack()
	if err != nil {
		return nil, fmt.Errorf("failed to hijack connection: %w", err)
	}

	// Generate accept key
	secKey := r.Header.Get("Sec-WebSocket-Key")
	acceptKey := generateAcceptKey(secKey)

	// Build response headers
	respHeaders := buildUpgradeResponse(acceptKey, u.Subprotocols)

	// Write HTTP response
	if _, err := conn.Write([]byte(respHeaders)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to write upgrade response: %w", err)
	}

	// Create WebSocket connection
	wsConn := &Connection{
		ID:        connID,
		Conn:      conn,
		CreatedAt: time.Now(),
	}

	return wsConn, nil
}

// validateRequest validates the WebSocket upgrade request.
func (u *Upgrader) validateRequest(r *http.Request) error {
	// Check HTTP method
	if r.Method != http.MethodGet {
		return &HandshakeError{Err: ErrNotWebSocketRequest, Status: http.StatusMethodNotAllowed}
	}

	// Check Upgrade header
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return &HandshakeError{Err: ErrMissingUpgrade, Status: http.StatusBadRequest}
	}

	// Check Connection header
	if !strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") {
		return &HandshakeError{Err: ErrNotWebSocketRequest, Status: http.StatusBadRequest}
	}

	// Check Sec-WebSocket-Version
	version := r.Header.Get("Sec-WebSocket-Version")
	if version != "13" {
		return &HandshakeError{Err: ErrInvalidSecVersion, Status: http.StatusBadRequest}
	}

	// Check Sec-WebSocket-Key
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return &HandshakeError{Err: ErrMissingSecKey, Status: http.StatusBadRequest}
	}

	// Validate key format (base64 encoded, 16 bytes)
	if len(key) < 16 {
		return &HandshakeError{Err: ErrInvalidSecKey, Status: http.StatusBadRequest}
	}

	// Check origin if configured
	if u.CheckOrigin != nil && !u.CheckOrigin(r) {
		return &HandshakeError{Err: ErrNotWebSocketRequest, Status: http.StatusForbidden}
	}

	return nil
}

// HandshakeError represents a handshake error.
type HandshakeError struct {
	Err    error
	Status int
}

func (e *HandshakeError) Error() string {
	return e.Err.Error()
}

func (e *HandshakeError) Unwrap() error {
	return e.Err
}

// generateAcceptKey generates the Sec-WebSocket-Accept key per RFC 6455.
func generateAcceptKey(secKey string) string {
	// Concatenate key with GUID
	combined := secKey + webSocketGUID

	// SHA1 hash
	hash := sha1.Sum([]byte(combined))

	// Base64 encode
	return base64.StdEncoding.EncodeToString(hash[:])
}

// buildUpgradeResponse builds the WebSocket upgrade response.
func buildUpgradeResponse(acceptKey string, subprotocols []string) string {
	var sb strings.Builder

	sb.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	sb.WriteString("Upgrade: websocket\r\n")
	sb.WriteString("Connection: Upgrade\r\n")
	sb.WriteString(fmt.Sprintf("Sec-WebSocket-Accept: %s\r\n", acceptKey))

	if len(subprotocols) > 0 {
		sb.WriteString(fmt.Sprintf("Sec-WebSocket-Protocol: %s\r\n", strings.Join(subprotocols, ", ")))
	}

	sb.WriteString("\r\n")

	return sb.String()
}

// ReadHandshakeRequest reads and parses a WebSocket upgrade request from a reader.
func ReadHandshakeRequest(r io.Reader) (*http.Request, error) {
	br := bufio.NewReader(r)

	// Read request line
	line, err := br.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read request line: %w", err)
	}
	line = strings.TrimSpace(line)

	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		return nil, &HandshakeError{Err: ErrNotWebSocketRequest, Status: http.StatusBadRequest}
	}

	method, path := parts[0], parts[1]

	// Read headers
	headers := make(http.Header)
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read header: %w", err)
		}
		line = strings.TrimSpace(line)

		if line == "" {
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headers.Set(parts[0], strings.TrimSpace(parts[1]))
		}
	}

	u, _ := url.Parse(path)

	return &http.Request{
		Method: method,
		URL:    u,
		Header: headers,
	}, nil
}

// CreateHandshakeResponse creates an HTTP response for a WebSocket upgrade request.
func CreateHandshakeResponse(secKey string, subprotocols []string) (string, error) {
	// Validate key format
	if len(secKey) < 16 {
		return "", ErrInvalidSecKey
	}

	// Generate accept key
	acceptKey := generateAcceptKey(secKey)

	// Build response
	return buildUpgradeResponse(acceptKey, subprotocols), nil
}

// ReadHTTPRequest reads a complete HTTP request from the buffer.
func ReadHTTPRequest(data []byte) (*http.Request, error) {
	// Find the end of headers (double CRLF)
	idx := bytes.Index(data, []byte("\r\n\r\n"))
	if idx < 0 {
		return nil, errors.New("invalid HTTP request: missing end of headers")
	}

	headerData := data[:idx]

	// Parse request line
	lines := bytes.Split(headerData, []byte("\r\n"))
	if len(lines) < 1 {
		return nil, ErrNotWebSocketRequest
	}

	requestLine := string(lines[0])
	parts := strings.SplitN(requestLine, " ", 3)
	if len(parts) < 2 {
		return nil, &HandshakeError{Err: ErrNotWebSocketRequest, Status: http.StatusBadRequest}
	}

	// Parse headers
	headers := make(http.Header)
	for i := 1; i < len(lines); i++ {
		line := string(lines[i])
		idx := strings.Index(line, ":")
		if idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			headers.Set(key, value)
		}
	}

	u, _ := url.Parse(parts[1])

	return &http.Request{
		Method: parts[0],
		URL:    u,
		Header: headers,
	}, nil
}

// GenerateSecKey generates a valid Sec-WebSocket-Key.
func GenerateSecKey() (string, error) {
	// Generate 16 random bytes
	data := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

// VerifyAcceptKey verifies that the accept key is correct for the given request key.
func VerifyAcceptKey(requestKey, expectedAccept string) bool {
	return generateAcceptKey(requestKey) == expectedAccept
}

// GetRequestedSubprotocol returns the subprotocol requested by the client.
func GetRequestedSubprotocol(r *http.Request) string {
	return r.Header.Get("Sec-WebSocket-Protocol")
}
