package ipc

import (
	"errors"
	"sync"
)

// Named pipe errors.
var (
	ErrPipeNotFound = errors.New("named pipe not found")
	ErrPipeExists   = errors.New("named pipe already exists")
	ErrInvalidName  = errors.New("invalid pipe name")
	ErrNotNamedPipe = errors.New("not a named pipe")
	ErrReaderExists = errors.New("reader already exists for this pipe")
	ErrWriterExists = errors.New("writer already exists for this pipe")
)

// NamedPipe represents a named pipe (FIFO) for IPC.
type NamedPipe struct {
	// Name is the pipe name.
	Name string
	// path is the file system path.
	path string
	// readers is the set of connected readers.
	readers map[*PipeReader]bool
	// writers is the set of connected writers.
	writers map[*PipeWriter]bool
	// mu protects the pipe state.
	mu sync.RWMutex
	// closed is true if the pipe is closed.
	closed bool
	// data holds queued data.
	data []byte
	// dataMu protects the data buffer.
	dataMu sync.Mutex
}

// NewNamedPipe creates a new named pipe.
func NewNamedPipe(name string) (*NamedPipe, error) {
	if name == "" {
		return nil, ErrInvalidName
	}

	return &NamedPipe{
		Name:    name,
		readers: make(map[*PipeReader]bool),
		writers: make(map[*PipeWriter]bool),
		closed:  false,
		data:    nil,
	}, nil
}

// ConnectReader connects a reader to the named pipe.
func (np *NamedPipe) ConnectReader() (*PipeReader, error) {
	np.mu.Lock()
	defer np.mu.Unlock()

	if np.closed {
		return nil, ErrPipeClosed
	}

	pipe := NewPipe()
	reader := NewPipeReader(pipe)
	np.readers[reader] = true

	return reader, nil
}

// ConnectWriter connects a writer to the named pipe.
func (np *NamedPipe) ConnectWriter() (*PipeWriter, error) {
	np.mu.Lock()
	defer np.mu.Unlock()

	if np.closed {
		return nil, ErrPipeClosed
	}

	pipe := NewPipe()
	writer := NewPipeWriter(pipe)
	np.writers[writer] = true

	return writer, nil
}

// Write writes data to all connected writers.
func (np *NamedPipe) Write(b []byte) (n int, err error) {
	np.mu.RLock()
	defer np.mu.RUnlock()

	if np.closed {
		return 0, ErrPipeClosed
	}

	// Queue the data
	np.dataMu.Lock()
	np.data = append(np.data, b...)
	np.dataMu.Unlock()

	// Broadcast to all writers
	for writer := range np.writers {
		writer.Write(b)
	}

	return len(b), nil
}

// Read reads data from the pipe.
func (np *NamedPipe) Read(b []byte) (n int, err error) {
	np.mu.RLock()
	defer np.mu.RUnlock()

	if np.closed {
		return 0, ErrPipeClosed
	}

	np.dataMu.Lock()
	if len(np.data) == 0 {
		np.dataMu.Unlock()
		return 0, nil
	}

	if len(np.data) <= len(b) {
		copy(b, np.data)
		np.data = nil
	} else {
		copy(b, np.data[:len(b)])
		np.data = np.data[len(b):]
	}
	np.dataMu.Unlock()

	return len(b), nil
}

// Close closes the named pipe.
func (np *NamedPipe) Close() error {
	np.mu.Lock()
	defer np.mu.Unlock()

	if np.closed {
		return nil
	}

	np.closed = true

	// Close all readers
	for reader := range np.readers {
		reader.pipe.Close()
	}

	// Close all writers
	for writer := range np.writers {
		writer.pipe.Close()
	}

	return nil
}

// Closed returns true if the pipe is closed.
func (np *NamedPipe) Closed() bool {
	np.mu.RLock()
	defer np.mu.RUnlock()
	return np.closed
}

// ReaderCount returns the number of connected readers.
func (np *NamedPipe) ReaderCount() int {
	np.mu.RLock()
	defer np.mu.RUnlock()
	return len(np.readers)
}

// WriterCount returns the number of connected writers.
func (np *NamedPipe) WriterCount() int {
	np.mu.RLock()
	defer np.mu.RUnlock()
	return len(np.writers)
}

// NamedPipeRegistry manages named pipes.
type NamedPipeRegistry struct {
	// pipes holds all named pipes.
	pipes map[string]*NamedPipe
	// mu protects the registry.
	mu sync.RWMutex
}

// NewNamedPipeRegistry creates a new registry.
func NewNamedPipeRegistry() *NamedPipeRegistry {
	return &NamedPipeRegistry{
		pipes: make(map[string]*NamedPipe),
	}
}

// Create creates a new named pipe.
func (r *NamedPipeRegistry) Create(name string) (*NamedPipe, error) {
	if name == "" {
		return nil, ErrInvalidName
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.pipes[name]; exists {
		return nil, ErrPipeExists
	}

	pipe, err := NewNamedPipe(name)
	if err != nil {
		return nil, err
	}

	r.pipes[name] = pipe
	return pipe, nil
}

// Get retrieves a named pipe.
func (r *NamedPipeRegistry) Get(name string) (*NamedPipe, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pipe, exists := r.pipes[name]
	if !exists {
		return nil, ErrPipeNotFound
	}

	return pipe, nil
}

// Remove removes a named pipe.
func (r *NamedPipeRegistry) Remove(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pipe, exists := r.pipes[name]
	if !exists {
		return ErrPipeNotFound
	}

	pipe.Close()
	delete(r.pipes, name)

	return nil
}

// Exists checks if a named pipe exists.
func (r *NamedPipeRegistry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.pipes[name]
	return exists
}

// List returns all named pipe names.
func (r *NamedPipeRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.pipes))
	for name := range r.pipes {
		names = append(names, name)
	}

	return names
}

// Count returns the number of named pipes.
func (r *NamedPipeRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.pipes)
}

// FIFO represents a first-in-first-out queue backed by a pipe.
type FIFO struct {
	// items holds the queued items.
	items [][]byte
	// mu protects the items.
	mu sync.Mutex
	// notEmpty signals when items are available.
	notEmpty chan struct{}
	// closed is true if the FIFO is closed.
	closed bool
}

// NewFIFO creates a new FIFO queue.
func NewFIFO() *FIFO {
	return &FIFO{
		items:    make([][]byte, 0),
		notEmpty: make(chan struct{}, 1),
		closed:   false,
	}
}

// Enqueue adds an item to the FIFO.
func (f *FIFO) Enqueue(item []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return
	}

	f.items = append(f.items, item)

	// Signal that an item is available
	select {
	case f.notEmpty <- struct{}{}:
	default:
	}
}

// Dequeue removes and returns an item from the FIFO.
func (f *FIFO) Dequeue() ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for len(f.items) == 0 && !f.closed {
		f.mu.Unlock()
		<-f.notEmpty
		f.mu.Lock()
	}

	if len(f.items) == 0 {
		return nil, ErrPipeClosed
	}

	item := f.items[0]
	f.items = f.items[1:]

	return item, nil
}

// DequeueNonBlocking removes and returns an item without blocking.
func (f *FIFO) DequeueNonBlocking() ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.items) == 0 {
		return nil, ErrWouldBlock
	}

	item := f.items[0]
	f.items = f.items[1:]

	return item, nil
}

// Len returns the number of items in the FIFO.
func (f *FIFO) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.items)
}

// Close closes the FIFO.
func (f *FIFO) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return nil
	}

	f.closed = true

	close(f.notEmpty)

	return nil
}

// Closed returns true if the FIFO is closed.
func (f *FIFO) Closed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}
