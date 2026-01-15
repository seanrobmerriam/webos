package ipc

import (
	"errors"
	"sync"
	"sync/atomic"
)

// Shared memory errors.
var (
	ErrSegNotFound     = errors.New("shared memory segment not found")
	ErrSegExists       = errors.New("shared memory segment already exists")
	ErrInvalidSize     = errors.New("invalid segment size")
	ErrInvalidOffset   = errors.New("invalid offset")
	ErrSegFull         = errors.New("segment is full")
	ErrNotAttached     = errors.New("not attached to segment")
	ErrAlreadyAttached = errors.New("already attached to segment")
)

// SharedMemorySegment represents a shared memory region.
type SharedMemorySegment struct {
	// ID is the segment identifier.
	ID string
	// data is the shared memory data.
	data []byte
	// size is the segment size.
	size int
	// mu protects the segment.
	mu sync.Mutex
	// readers tracks attached readers.
	readers int32
	// writers tracks attached writers.
	writers int32
	// creatorID is the PID of the creator.
	creatorID int
}

// NewSharedMemorySegment creates a new shared memory segment.
func NewSharedMemorySegment(id string, size int, creatorID int) (*SharedMemorySegment, error) {
	if size <= 0 {
		return nil, ErrInvalidSize
	}

	return &SharedMemorySegment{
		ID:        id,
		data:      make([]byte, size),
		size:      size,
		creatorID: creatorID,
		readers:   0,
		writers:   0,
	}, nil
}

// Read reads from the shared memory.
func (s *SharedMemorySegment) Read(offset int, p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if offset < 0 || offset >= s.size {
		return 0, ErrInvalidOffset
	}

	n = len(p)
	if offset+n > s.size {
		n = s.size - offset
	}

	copy(p, s.data[offset:offset+n])
	return n, nil
}

// Write writes to the shared memory.
func (s *SharedMemorySegment) Write(offset int, p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if offset < 0 || offset >= s.size {
		return 0, ErrInvalidOffset
	}

	n = len(p)
	if offset+n > s.size {
		n = s.size - offset
	}

	copy(s.data[offset:], p[:n])
	return n, nil
}

// AttachReader attaches a reader to the segment.
func (s *SharedMemorySegment) AttachReader() {
	atomic.AddInt32(&s.readers, 1)
}

// DetachReader detaches a reader from the segment.
func (s *SharedMemorySegment) DetachReader() {
	atomic.AddInt32(&s.readers, -1)
}

// AttachWriter attaches a writer to the segment.
func (s *SharedMemorySegment) AttachWriter() {
	atomic.AddInt32(&s.writers, 1)
}

// DetachWriter detaches a writer from the segment.
func (s *SharedMemorySegment) DetachWriter() {
	atomic.AddInt32(&s.writers, -1)
}

// ReaderCount returns the number of attached readers.
func (s *SharedMemorySegment) ReaderCount() int {
	return int(atomic.LoadInt32(&s.readers))
}

// WriterCount returns the number of attached writers.
func (s *SharedMemorySegment) WriterCount() int {
	return int(atomic.LoadInt32(&s.writers))
}

// Size returns the segment size.
func (s *SharedMemorySegment) Size() int {
	return s.size
}

// CreatorID returns the creator's PID.
func (s *SharedMemorySegment) CreatorID() int {
	return s.creatorID
}

// SharedMemoryRegistry manages shared memory segments.
type SharedMemoryRegistry struct {
	// segments holds all shared memory segments.
	segments map[string]*SharedMemorySegment
	// mu protects the registry.
	mu sync.RWMutex
}

// NewSharedMemoryRegistry creates a new registry.
func NewSharedMemoryRegistry() *SharedMemoryRegistry {
	return &SharedMemoryRegistry{
		segments: make(map[string]*SharedMemorySegment),
	}
}

// Create creates a new shared memory segment.
func (r *SharedMemoryRegistry) Create(id string, size int, creatorID int) (*SharedMemorySegment, error) {
	if size <= 0 {
		return nil, ErrInvalidSize
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.segments[id]; exists {
		return nil, ErrSegExists
	}

	seg, err := NewSharedMemorySegment(id, size, creatorID)
	if err != nil {
		return nil, err
	}

	r.segments[id] = seg
	return seg, nil
}

// Get retrieves a shared memory segment.
func (r *SharedMemoryRegistry) Get(id string) (*SharedMemorySegment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seg, exists := r.segments[id]
	if !exists {
		return nil, ErrSegNotFound
	}

	return seg, nil
}

// Remove removes a shared memory segment.
func (r *SharedMemoryRegistry) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	seg, exists := r.segments[id]
	if !exists {
		return ErrSegNotFound
	}

	// Check if anyone is still attached
	if seg.ReaderCount() > 0 || seg.WriterCount() > 0 {
		// In a real system, you might want to prevent removal
		// For now, we allow it
	}

	delete(r.segments, id)
	return nil
}

// Exists checks if a segment exists.
func (r *SharedMemoryRegistry) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.segments[id]
	return exists
}

// AttachState tracks attachment state for processes.
type AttachState struct {
	segmentID string
	isWriter  bool
}

// SharedMemoryManager provides high-level shared memory management.
type SharedMemoryManager struct {
	registry *SharedMemoryRegistry
	// attachments tracks which processes are attached to which segments.
	attachments map[int]map[string]*AttachState
	// mu protects attachments.
	attachMu sync.RWMutex
}

// NewSharedMemoryManager creates a new manager.
func NewSharedMemoryManager() *SharedMemoryManager {
	return &SharedMemoryManager{
		registry:    NewSharedMemoryRegistry(),
		attachments: make(map[int]map[string]*AttachState),
	}
}

// CreateSegment creates a new shared memory segment.
func (m *SharedMemoryManager) CreateSegment(pid int, id string, size int) (*SharedMemorySegment, error) {
	return m.registry.Create(id, size, pid)
}

// Attach attaches a process to a shared memory segment.
func (m *SharedMemoryManager) Attach(pid int, id string, isWriter bool) (*SharedMemorySegment, error) {
	seg, err := m.registry.Get(id)
	if err != nil {
		return nil, err
	}

	m.attachMu.Lock()
	defer m.attachMu.Unlock()

	// Check if already attached
	if attachments, exists := m.attachments[pid]; exists {
		if _, attached := attachments[id]; attached {
			return nil, ErrAlreadyAttached
		}
	}

	// Track attachment
	if _, exists := m.attachments[pid]; !exists {
		m.attachments[pid] = make(map[string]*AttachState)
	}
	m.attachments[pid][id] = &AttachState{
		segmentID: id,
		isWriter:  isWriter,
	}

	// Update segment
	if isWriter {
		seg.AttachWriter()
	} else {
		seg.AttachReader()
	}

	return seg, nil
}

// Detach detaches a process from a shared memory segment.
func (m *SharedMemoryManager) Detach(pid int, id string) error {
	seg, err := m.registry.Get(id)
	if err != nil {
		return err
	}

	m.attachMu.Lock()
	defer m.attachMu.Unlock()

	attachments, exists := m.attachments[pid]
	if !exists {
		return ErrNotAttached
	}

	state, attached := attachments[id]
	if !attached {
		return ErrNotAttached
	}

	// Update segment
	if state.isWriter {
		seg.DetachWriter()
	} else {
		seg.DetachReader()
	}

	// Remove attachment
	delete(attachments, id)
	if len(attachments) == 0 {
		delete(m.attachments, pid)
	}

	return nil
}

// GetAttachments returns all segments a process is attached to.
func (m *SharedMemoryManager) GetAttachments(pid int) []*AttachState {
	m.attachMu.RLock()
	defer m.attachMu.RUnlock()

	attachments, exists := m.attachments[pid]
	if !exists {
		return nil
	}

	states := make([]*AttachState, 0, len(attachments))
	for _, state := range attachments {
		states = append(states, state)
	}

	return states
}

// IsAttached checks if a process is attached to a segment.
func (m *SharedMemoryManager) IsAttached(pid int, id string) bool {
	m.attachMu.RLock()
	defer m.attachMu.RUnlock()

	attachments, exists := m.attachments[pid]
	if !exists {
		return false
	}

	_, attached := attachments[id]
	return attached
}

// ReadAt reads from a shared memory segment.
func (m *SharedMemoryManager) ReadAt(pid int, id string, offset int, p []byte) (n int, err error) {
	seg, err := m.registry.Get(id)
	if err != nil {
		return 0, err
	}

	m.attachMu.RLock()
	attached := m.isProcessAttachedLocked(pid, id)
	m.attachMu.RUnlock()

	if !attached {
		return 0, ErrNotAttached
	}

	return seg.Read(offset, p)
}

// WriteAt writes to a shared memory segment.
func (m *SharedMemoryManager) WriteAt(pid int, id string, offset int, p []byte) (n int, err error) {
	seg, err := m.registry.Get(id)
	if err != nil {
		return 0, err
	}

	m.attachMu.RLock()
	state := m.getAttachStateLocked(pid, id)
	m.attachMu.RUnlock()

	if state == nil {
		return 0, ErrNotAttached
	}

	if !state.isWriter {
		return 0, ErrNotAttached
	}

	return seg.Write(offset, p)
}

// isProcessAttachedLocked checks attachment without acquiring lock.
func (m *SharedMemoryManager) isProcessAttachedLocked(pid int, id string) bool {
	attachments, exists := m.attachments[pid]
	if !exists {
		return false
	}
	_, attached := attachments[id]
	return attached
}

// getAttachStateLocked returns attach state without acquiring lock.
func (m *SharedMemoryManager) getAttachStateLocked(pid int, id string) *AttachState {
	attachments, exists := m.attachments[pid]
	if !exists {
		return nil
	}
	return attachments[id]
}

// SharedMemoryBuffer provides a ring-buffer backed by shared memory.
type SharedMemoryBuffer struct {
	segment *SharedMemorySegment
	// head is the read position.
	head int
	// tail is the write position.
	tail int
	// mu protects head and tail.
	mu sync.Mutex
	// mask for wrap-around.
	mask int
}

// NewSharedMemoryBuffer creates a new shared memory buffer.
func NewSharedMemoryBuffer(segment *SharedMemorySegment) *SharedMemoryBuffer {
	// Find power of 2 for size
	size := segment.Size() - 8 // Reserve 4 bytes each for head and tail
	var mask int
	for mask = 1; mask < size; mask <<= 1 {
	}
	mask >>= 1

	return &SharedMemoryBuffer{
		segment: segment,
		mask:    mask,
	}
}

// Write writes data to the buffer.
func (b *SharedMemoryBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	avail := b.available()
	if avail < len(p) {
		return 0, ErrSegFull
	}

	for _, c := range p {
		b.segment.data[b.tail] = c
		b.tail = (b.tail + 1) & b.mask
	}

	return len(p), nil
}

// Read reads data from the buffer.
func (b *SharedMemoryBuffer) Read(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	count := 0
	for count < len(p) && b.head != b.tail {
		p[count] = b.segment.data[b.head]
		b.head = (b.head + 1) & b.mask
		count++
	}

	return count, nil
}

// available returns available space in the buffer.
func (b *SharedMemoryBuffer) available() int {
	// This is simplified; real implementation would need proper sync
	return b.mask - ((b.tail - b.head) & b.mask)
}

// Used returns the number of bytes used in the buffer.
func (b *SharedMemoryBuffer) Used() int {
	return (b.tail - b.head) & b.mask
}

// Capacity returns the buffer capacity.
func (b *SharedMemoryBuffer) Capacity() int {
	return b.mask
}
