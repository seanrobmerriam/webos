package ipc

import (
	"encoding/binary"
	"errors"
	"sync"
	"time"
)

// Message queue errors.
var (
	ErrQueueFull      = errors.New("message queue is full")
	ErrQueueEmpty     = errors.New("message queue is empty")
	ErrInvalidMessage = errors.New("invalid message")
	ErrNotSupported   = errors.New("operation not supported")
)

// MessageType represents the type of a message.
type MessageType uint8

const (
	// MessageTypeData is a regular data message.
	MessageTypeData MessageType = iota
	// MessageTypeControl is a control message.
	MessageTypeControl
	// MessageTypePriority is a priority message.
	MessageTypePriority
	// MessageTypeDisconnect is a disconnect notification.
	MessageTypeDisconnect
)

// Priority levels for messages.
const (
	MessagePriorityLow    = 0
	MessagePriorityNormal = 5
	MessagePriorityHigh   = 10
)

// Message represents a message in the queue.
type Message struct {
	// Type is the message type.
	Type MessageType
	// Priority is the message priority.
	Priority int
	// Payload is the message data.
	Payload []byte
	// SenderID identifies the sender.
	SenderID int
	// Timestamp is when the message was created.
	Timestamp time.Time
}

// NewMessage creates a new message.
func NewMessage(msgType MessageType, payload []byte, priority int, senderID int) *Message {
	return &Message{
		Type:      msgType,
		Payload:   payload,
		Priority:  priority,
		SenderID:  senderID,
		Timestamp: time.Now(),
	}
}

// Size returns the size of the message in bytes.
func (m *Message) Size() int {
	return len(m.Payload) + 16 // Header overhead
}

// MessageQueue provides structured message passing between processes.
type MessageQueue struct {
	// messages holds the queued messages.
	messages []*Message
	// mu protects the queue.
	mu sync.Mutex
	// notEmpty signals when messages are available.
	notEmpty chan struct{}
	// maxSize is the maximum queue size in bytes.
	maxSize int
	// currentSize is the current queue size in bytes.
	currentSize int
	// maxMessages is the maximum number of messages.
	maxMessages int
	// currentMessages is the current message count.
	currentMessages int
	// closed is true if the queue is closed.
	closed bool
}

// NewMessageQueue creates a new message queue.
func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		messages:        make([]*Message, 0),
		notEmpty:        make(chan struct{}, 1),
		maxSize:         1024 * 1024, // 1 MB
		maxMessages:     1000,
		currentSize:     0,
		currentMessages: 0,
		closed:          false,
	}
}

// Configure sets queue configuration options.
func (q *MessageQueue) Configure(maxSize int, maxMessages int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.maxSize = maxSize
	q.maxMessages = maxMessages
}

// Send sends a message to the queue.
func (q *MessageQueue) Send(msg *Message) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return ErrQueueFull
	}

	// Check size limits
	msgSize := msg.Size()
	if q.maxSize > 0 && q.currentSize+msgSize > q.maxSize {
		return ErrQueueFull
	}
	if q.maxMessages > 0 && q.currentMessages >= q.maxMessages {
		return ErrQueueFull
	}

	// Add message (maintain priority order)
	inserted := false
	for i, m := range q.messages {
		if msg.Priority > m.Priority {
			// Insert at position i
			q.messages = append(q.messages[:i], append([]*Message{msg}, q.messages[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		q.messages = append(q.messages, msg)
	}

	q.currentSize += msgSize
	q.currentMessages++

	// Signal that a message is available
	select {
	case q.notEmpty <- struct{}{}:
	default:
	}

	return nil
}

// Receive receives a message from the queue.
func (q *MessageQueue) Receive() (*Message, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.messages) == 0 && !q.closed {
		q.mu.Unlock()
		<-q.notEmpty
		q.mu.Lock()
	}

	if len(q.messages) == 0 {
		return nil, ErrQueueEmpty
	}

	msg := q.messages[0]
	q.messages = q.messages[1:]
	q.currentSize -= msg.Size()
	q.currentMessages--

	return msg, nil
}

// ReceiveNonBlocking receives a message without blocking.
func (q *MessageQueue) ReceiveNonBlocking() (*Message, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.messages) == 0 {
		return nil, ErrQueueEmpty
	}

	msg := q.messages[0]
	q.messages = q.messages[1:]
	q.currentSize -= msg.Size()
	q.currentMessages--

	return msg, nil
}

// Peek returns the next message without removing it.
func (q *MessageQueue) Peek() (*Message, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.messages) == 0 {
		return nil, ErrQueueEmpty
	}

	return q.messages[0], nil
}

// Len returns the number of messages in the queue.
func (q *MessageQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.messages)
}

// Size returns the current size of queued data.
func (q *MessageQueue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.currentSize
}

// Capacity returns the remaining capacity.
func (q *MessageQueue) Capacity() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.maxSize == 0 {
		return 0
	}
	return q.maxSize - q.currentSize
}

// Close closes the message queue.
func (q *MessageQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true
	close(q.notEmpty)

	return nil
}

// Closed returns true if the queue is closed.
func (q *MessageQueue) Closed() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.closed
}

// MessageHeader represents a binary message header.
type MessageHeader struct {
	// Type is the message type.
	Type uint8
	// Priority is the message priority.
	Priority uint8
	// Length is the payload length.
	Length uint32
	// SenderID is the sender identifier.
	SenderID uint32
}

// Encode encodes the header to bytes.
func (h *MessageHeader) Encode() []byte {
	buf := make([]byte, 8)
	buf[0] = h.Type
	buf[1] = h.Priority
	binary.BigEndian.PutUint32(buf[2:6], h.Length)
	binary.BigEndian.PutUint32(buf[6:8], h.SenderID)
	return buf
}

// Decode decodes a header from bytes.
func (h *MessageHeader) Decode(buf []byte) error {
	if len(buf) < 8 {
		return ErrInvalidMessage
	}
	h.Type = buf[0]
	h.Priority = buf[1]
	h.Length = binary.BigEndian.Uint32(buf[2:6])
	h.SenderID = binary.BigEndian.Uint32(buf[6:8])
	return nil
}

// BinaryMessageQueue provides message queue with binary encoding.
type BinaryMessageQueue struct {
	*MessageQueue
	// mu protects binary operations.
	binMu sync.Mutex
}

// NewBinaryMessageQueue creates a new binary message queue.
func NewBinaryMessageQueue() *BinaryMessageQueue {
	return &BinaryMessageQueue{
		MessageQueue: NewMessageQueue(),
	}
}

// SendBinary sends a binary message.
func (q *BinaryMessageQueue) SendBinary(msgType MessageType, payload []byte, priority int, senderID int) error {
	header := &MessageHeader{
		Type:     uint8(msgType),
		Priority: uint8(priority),
		Length:   uint32(len(payload)),
		SenderID: uint32(senderID),
	}

	data := append(header.Encode(), payload...)
	msg := NewMessage(MessageTypeData, data, priority, senderID)

	return q.Send(msg)
}

// ReceiveBinary receives a binary message and returns the payload.
func (q *BinaryMessageQueue) ReceiveBinary() ([]byte, error) {
	msg, err := q.Receive()
	if err != nil {
		return nil, err
	}

	if len(msg.Payload) < 8 {
		return nil, ErrInvalidMessage
	}

	return msg.Payload[8:], nil
}

// TopicMessage supports pub/sub messaging pattern.
type TopicMessage struct {
	// Topic is the message topic.
	Topic string
	// Message is the wrapped message.
	*Message
}

// TopicQueue manages topic-based subscriptions.
type TopicQueue struct {
	// topics holds queues for each topic.
	topics map[string]*MessageQueue
	// defaultQueue is for unhandled topics.
	defaultQueue *MessageQueue
	// mu protects the registry.
	mu sync.RWMutex
}

// NewTopicQueue creates a new topic queue manager.
func NewTopicQueue() *TopicQueue {
	return &TopicQueue{
		topics:       make(map[string]*MessageQueue),
		defaultQueue: NewMessageQueue(),
	}
}

// Subscribe creates a subscription queue for a topic.
func (t *TopicQueue) Subscribe(topic string) *MessageQueue {
	t.mu.Lock()
	defer t.mu.Unlock()

	if q, exists := t.topics[topic]; exists {
		return q
	}

	q := NewMessageQueue()
	t.topics[topic] = q
	return q
}

// Publish publishes a message to a topic.
func (t *TopicQueue) Publish(topic string, msg *Message) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	q, exists := t.topics[topic]
	if !exists {
		// Use default queue for unhandled topics
		return t.defaultQueue.Send(msg)
	}

	return q.Send(msg)
}

// Unsubscribe removes a topic subscription.
func (t *TopicQueue) Unsubscribe(topic string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.topics, topic)
}

// Topics returns all active topics.
func (t *TopicQueue) Topics() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	topics := make([]string, 0, len(t.topics))
	for topic := range t.topics {
		topics = append(topics, topic)
	}
	return topics
}
