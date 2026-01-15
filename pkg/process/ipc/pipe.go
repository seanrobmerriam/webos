package ipc

import (
	"bytes"
	"errors"
	"sync"
)

// Pipe errors.
var (
	ErrPipeClosed  = errors.New("pipe is closed")
	ErrWouldBlock  = errors.New("operation would block")
	ErrBrokenPipe  = errors.New("pipe is broken")
	ErrInvalidPipe = errors.New("invalid pipe")
)

// Pipe represents an anonymous pipe for inter-process communication.
type Pipe struct {
	// readChan is the channel for reading data.
	readChan chan []byte
	// writeChan is the channel for writing data.
	writeChan chan []byte
	// closeChan signals pipe closure.
	closeChan chan struct{}
	// closed is true if the pipe is closed.
	closed bool
	// mu protects the closed flag.
	mu sync.Mutex
	// buffer holds unread data.
	buffer *bytes.Buffer
	// writeBuffer holds data waiting to be read.
	writeBuffer *bytes.Buffer
}

// NewPipe creates a new anonymous pipe.
func NewPipe() *Pipe {
	return &Pipe{
		readChan:    make(chan []byte, 64),
		writeChan:   make(chan []byte, 64),
		closeChan:   make(chan struct{}),
		closed:      false,
		buffer:      bytes.NewBuffer(nil),
		writeBuffer: bytes.NewBuffer(nil),
	}
}

// Read reads data from the pipe.
func (p *Pipe) Read(b []byte) (n int, err error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return 0, ErrPipeClosed
	}
	p.mu.Unlock()

	// Try to read from buffer first
	p.mu.Lock()
	if p.buffer.Len() > 0 {
		n, _ = p.buffer.Read(b)
		p.mu.Unlock()
		return n, nil
	}
	p.mu.Unlock()

	// Wait for data or close
	select {
	case data := <-p.readChan:
		if data == nil {
			return 0, ErrBrokenPipe
		}
		p.mu.Lock()
		if len(data) <= len(b) {
			copy(b, data)
			p.mu.Unlock()
			return len(data), nil
		}
		// Buffer excess data
		copy(b, data[:len(b)])
		p.mu.Lock()
		p.buffer.Write(data[len(b):])
		p.mu.Unlock()
		return len(b), nil

	case <-p.closeChan:
		return 0, ErrPipeClosed
	}
}

// Write writes data to the pipe.
func (p *Pipe) Write(b []byte) (n int, err error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return 0, ErrPipeClosed
	}
	p.mu.Unlock()

	// Make a copy of the data
	data := make([]byte, len(b))
	copy(data, b)

	select {
	case p.readChan <- data:
		return len(b), nil
	case <-p.closeChan:
		return 0, ErrBrokenPipe
	}
}

// Close closes the pipe for writing.
func (p *Pipe) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	close(p.closeChan)

	// Drain the read channel
	select {
	case <-p.readChan:
	default:
	}

	return nil
}

// Closed returns true if the pipe is closed.
func (p *Pipe) Closed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closed
}

// PipePair represents a pair of connected pipes (read/write ends).
type PipePair struct {
	// Read is the read end of the pipe.
	Read *Pipe
	// Write is the write end of the pipe.
	Write *Pipe
}

// NewPipePair creates a new pipe pair.
func NewPipePair() *PipePair {
	readPipe := NewPipe()
	writePipe := NewPipe()

	// Connect the pipes
	go func() {
		for {
			select {
			case data := <-writePipe.readChan:
				if data == nil {
					readPipe.Close()
					return
				}
				select {
				case readPipe.readChan <- data:
				case <-readPipe.closeChan:
					return
				case <-writePipe.closeChan:
					return
				}
			case <-readPipe.closeChan:
				return
			case <-writePipe.closeChan:
				return
			}
		}
	}()

	return &PipePair{
		Read:  readPipe,
		Write: writePipe,
	}
}

// BufferedPipe is a pipe with internal buffering.
type BufferedPipe struct {
	pipe    *Pipe
	mu      sync.Mutex
	readBuf *bytes.Buffer
}

// NewBufferedPipe creates a new buffered pipe.
func NewBufferedPipe() *BufferedPipe {
	return &BufferedPipe{
		pipe:    NewPipe(),
		readBuf: bytes.NewBuffer(nil),
	}
}

// Read reads data from the buffered pipe.
func (b *BufferedPipe) Read(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.readBuf.Len() == 0 {
		buf := make([]byte, 1024)
		n, err := b.pipe.Read(buf)
		if n > 0 {
			b.readBuf.Write(buf[:n])
		}
		if err != nil {
			return 0, err
		}
	}

	return b.readBuf.Read(p)
}

// Write writes data to the pipe.
func (b *BufferedPipe) Write(p []byte) (n int, err error) {
	return b.pipe.Write(p)
}

// Close closes the pipe.
func (b *BufferedPipe) Close() error {
	return b.pipe.Close()
}

// BytesAvailable returns the number of bytes available for reading.
func (b *BufferedPipe) BytesAvailable() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.readBuf.Len()
}

// PipeReader wraps a pipe for reading with additional functionality.
type PipeReader struct {
	pipe *Pipe
	mu   sync.Mutex
}

// NewPipeReader creates a new pipe reader.
func NewPipeReader(pipe *Pipe) *PipeReader {
	return &PipeReader{pipe: pipe}
}

// Read reads data from the pipe.
func (r *PipeReader) Read(b []byte) (n int, err error) {
	return r.pipe.Read(b)
}

// ReadByte reads a single byte.
func (r *PipeReader) ReadByte() (c byte, err error) {
	buf := make([]byte, 1)
	n, err := r.Read(buf)
	if n > 0 {
		return buf[0], nil
	}
	return 0, err
}

// ReadBytes reads until the first occurrence of delim.
func (r *PipeReader) ReadBytes(delim byte) ([]byte, error) {
	data := make([]byte, 0, 1024)
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == delim {
			return data, nil
		}
		data = append(data, b)
	}
}

// ReadString reads until the first occurrence of delim.
func (r *PipeReader) ReadString(delim byte) (string, error) {
	b, err := r.ReadBytes(delim)
	return string(b), err
}

// PipeWriter wraps a pipe for writing with additional functionality.
type PipeWriter struct {
	pipe *Pipe
	mu   sync.Mutex
}

// NewPipeWriter creates a new pipe writer.
func NewPipeWriter(pipe *Pipe) *PipeWriter {
	return &PipeWriter{pipe: pipe}
}

// Write writes data to the pipe.
func (w *PipeWriter) Write(b []byte) (n int, err error) {
	return w.pipe.Write(b)
}

// WriteByte writes a single byte.
func (w *PipeWriter) WriteByte(c byte) error {
	buf := []byte{c}
	_, err := w.pipe.Write(buf)
	return err
}

// WriteString writes a string to the pipe.
func (w *PipeWriter) WriteString(s string) (n int, err error) {
	return w.pipe.Write([]byte(s))
}
