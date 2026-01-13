package websocket

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"webos/pkg/protocol"
)

// Connection represents a WebSocket connection.
type Connection struct {
	// ID is the unique identifier for this connection.
	ID string
	// Conn is the underlying network connection.
	Conn net.Conn
	// Session is the associated session, if any.
	Session *Session
	// CreatedAt is the time when the connection was established.
	CreatedAt time.Time
	// LastPing is the time of the last ping.
	LastPing time.Time
	// mu protects concurrent access to the connection.
	mu sync.Mutex

	// reader is the frame reader.
	reader *FrameReader
	// writer is the frame writer.
	writer *FrameWriter
	// readTimeout is the read timeout duration.
	readTimeout time.Duration
	// writeTimeout is the write timeout duration.
	writeTimeout time.Duration
	// pingInterval is the interval between pings.
	pingInterval time.Duration
	// onClose is called when the connection is closed.
	onClose func(*Connection)
	// onMessage is called when a message is received.
	onMessage func(*Connection, *protocol.Message) error
}

// NewConnection creates a new WebSocket connection.
func NewConnection(id string, conn net.Conn) *Connection {
	return &Connection{
		ID:           id,
		Conn:         conn,
		CreatedAt:    time.Now(),
		LastPing:     time.Now(),
		reader:       NewFrameReader(conn),
		writer:       NewFrameWriter(conn),
		readTimeout:  30 * time.Second,
		writeTimeout: 10 * time.Second,
		pingInterval: 25 * time.Second,
	}
}

// SetReadTimeout sets the read timeout.
func (c *Connection) SetReadTimeout(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readTimeout = d
}

// SetWriteTimeout sets the write timeout.
func (c *Connection) SetWriteTimeout(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writeTimeout = d
}

// SetPingInterval sets the ping interval.
func (c *Connection) SetPingInterval(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pingInterval = d
}

// OnClose sets the callback for connection close events.
func (c *Connection) OnClose(fn func(*Connection)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onClose = fn
}

// OnMessage sets the callback for message events.
func (c *Connection) OnMessage(fn func(*Connection, *protocol.Message) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onMessage = fn
}

// ReadFrame reads a frame from the connection.
func (c *Connection) ReadFrame() (*Frame, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Set read deadline
	if c.readTimeout > 0 {
		if err := c.Conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
			return nil, err
		}
	}

	return c.reader.ReadFrame()
}

// WriteFrame writes a frame to the connection.
func (c *Connection) WriteFrame(frame *Frame) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Set write deadline
	if c.writeTimeout > 0 {
		if err := c.Conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return err
		}
	}

	return c.writer.WriteFrame(frame)
}

// WriteMessage writes a protocol message to the connection.
func (c *Connection) WriteMessage(msg *protocol.Message) error {
	// Encode the message
	data, err := msg.Encode()
	if err != nil {
		return err
	}

	// Determine opcode based on message type
	var opcode Opcode
	switch msg.Opcode {
	case protocol.OpcodePing:
		opcode = OpcodePing
	case protocol.OpcodePong:
		opcode = OpcodePong
	case protocol.OpcodeError:
		opcode = OpcodeClose
	default:
		opcode = OpcodeBinary
	}

	// Write as a binary frame
	return c.WriteFrame(NewFrame(opcode, data, true))
}

// WriteText writes a text message to the connection.
func (c *Connection) WriteText(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.writeTimeout > 0 {
		if err := c.Conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return err
		}
	}

	return c.writer.WriteText(data)
}

// WriteBinary writes binary data to the connection.
func (c *Connection) WriteBinary(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.writeTimeout > 0 {
		if err := c.Conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return err
		}
	}

	return c.writer.WriteBinary(data)
}

// WriteClose sends a close frame with the given code and reason.
func (c *Connection) WriteClose(code uint16, reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.writeTimeout > 0 {
		if err := c.Conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return err
		}
	}

	return c.writer.WriteClose(code, reason)
}

// WritePing sends a ping frame.
func (c *Connection) WritePing(payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.writeTimeout > 0 {
		if err := c.Conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return err
		}
	}

	return c.writer.WritePing(payload)
}

// WritePong sends a pong frame.
func (c *Connection) WritePong(payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.writeTimeout > 0 {
		if err := c.Conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return err
		}
	}

	return c.writer.WritePong(payload)
}

// Close closes the connection.
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Send close frame
	c.writer.WriteClose(1000, "connection closed")

	// Close the underlying connection
	err := c.Conn.Close()

	// Call onClose callback
	if c.onClose != nil {
		c.onClose(c)
	}

	return err
}

// CloseCode returns the close code from a close frame.
func CloseCode(payload []byte) uint16 {
	if len(payload) < 2 {
		return 0
	}
	return binary.BigEndian.Uint16(payload[:2])
}

// CloseReason returns the close reason from a close frame.
func CloseReason(payload []byte) string {
	if len(payload) <= 2 {
		return ""
	}
	return string(payload[2:])
}

// ReadMessage reads a complete protocol message from the connection.
func (c *Connection) ReadMessage() (*protocol.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Set read deadline
	if c.readTimeout > 0 {
		if err := c.Conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
			return nil, err
		}
	}

	// Read frame
	frame, err := c.reader.ReadFrame()
	if err != nil {
		return nil, err
	}

	// Handle control frames
	switch frame.Opcode {
	case OpcodeClose:
		return nil, ErrConnectionClosed
	case OpcodePing:
		c.writer.WritePong(frame.Payload)
		return nil, nil
	case OpcodePong:
		c.LastPing = time.Now()
		return nil, nil
	}

	// Parse protocol message
	msg := &protocol.Message{}
	if err := msg.Decode(frame.Payload); err != nil {
		return nil, err
	}

	return msg, nil
}

// StartReadLoop starts the read loop in a goroutine.
// It calls onMessage for each complete message received.
func (c *Connection) StartReadLoop() {
	go func() {
		for {
			msg, err := c.ReadMessage()
			if err != nil {
				if !errors.Is(err, io.EOF) && !errors.Is(err, ErrConnectionClosed) {
					// Log error
				}
				break
			}

			if msg == nil {
				continue
			}

			c.mu.Lock()
			if c.onMessage != nil {
				c.onMessage(c, msg)
			}
			c.mu.Unlock()
		}
	}()
}

// StartHeartbeat starts the heartbeat mechanism.
func (c *Connection) StartHeartbeat() {
	go func() {
		ticker := time.NewTicker(c.pingInterval)
		defer ticker.Stop()

		for range ticker.C {
			c.mu.Lock()
			if c.Conn == nil {
				c.mu.Unlock()
				return
			}

			// Check if connection is still alive
			if time.Since(c.LastPing) > c.pingInterval*2 {
				c.Conn.Close()
				c.mu.Unlock()
				return
			}

			// Send ping
			if err := c.writer.WritePing([]byte("heartbeat")); err != nil {
				c.Conn.Close()
				c.mu.Unlock()
				return
			}

			c.mu.Unlock()
		}
	}()
}

// LocalAddr returns the local network address.
func (c *Connection) LocalAddr() net.Addr {
	return c.Conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *Connection) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

// SetSession sets the session for this connection.
func (c *Connection) SetSession(s *Session) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Session = s
}

// GetSession returns the session for this connection.
func (c *Connection) GetSession() *Session {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Session
}

// IsConnected returns true if the connection is still connected.
func (c *Connection) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn != nil
}

// ReadHTTPRequest reads an HTTP upgrade request from the connection.
func (c *Connection) ReadHTTPRequest() (*http.Request, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Read data from connection
	reader := bufio.NewReader(c.Conn)

	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)

	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		return nil, ErrNotWebSocketRequest
	}

	method, path := parts[0], parts[1]

	// Read headers
	headers := make(http.Header)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
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

// BufferReader provides buffering for reading.
type BufferReader struct {
	reader *bufio.Reader
	buf    bytes.Buffer
}

// NewBufferReader creates a new BufferReader.
func NewBufferReader(r io.Reader) *BufferReader {
	return &BufferReader{
		reader: bufio.NewReaderSize(r, 4096),
	}
}

// Read reads data into p.
func (br *BufferReader) Read(p []byte) (n int, err error) {
	return br.reader.Read(p)
}

// ReadByte reads a single byte.
func (br *BufferReader) ReadByte() (byte, error) {
	return br.reader.ReadByte()
}

// ReadBytes reads until delim.
func (br *BufferReader) ReadBytes(delim byte) ([]byte, error) {
	return br.reader.ReadBytes(delim)
}

// ReadString reads until delim.
func (br *BufferReader) ReadString(delim byte) (string, error) {
	return br.reader.ReadString(delim)
}

// Peek returns the next n bytes.
func (br *BufferReader) Peek(n int) ([]byte, error) {
	return br.reader.Peek(n)
}
