package pty

import (
	"bytes"
	"errors"
	"io"
	"sync"
)

// ErrNotMaster is returned when operations requiring master side are called on slave.
var ErrNotMaster = errors.New("operation requires master side")

// ErrNotSlave is returned when operations requiring slave side are called on master.
var ErrNotSlave = errors.New("operation requires slave side")

// PTY represents a pseudo-terminal pair with master and slave sides.
type PTY struct {
	master *Pipe
	slave  *Pipe
	term   *Terminal
	mu     sync.RWMutex
	closed bool
}

// Pipe represents one side of a PTY.
type Pipe struct {
	pty       *PTY
	isMaster  bool
	readBuf   bytes.Buffer
	writeBuf  bytes.Buffer
	readCond  *sync.Cond
	writeCond *sync.Cond
}

// NewPTY creates a new pseudo-terminal pair.
func NewPTY(cols, rows int) (*PTY, error) {
	term, err := NewTerminal(cols, rows)
	if err != nil {
		return nil, err
	}

	pty := &PTY{
		term: term,
		master: &Pipe{
			isMaster:  true,
			readCond:  sync.NewCond(&sync.Mutex{}),
			writeCond: sync.NewCond(&sync.Mutex{}),
		},
		slave: &Pipe{
			isMaster:  false,
			readCond:  sync.NewCond(&sync.Mutex{}),
			writeCond: sync.NewCond(&sync.Mutex{}),
		},
	}

	pty.master.pty = pty
	pty.slave.pty = pty

	return pty, nil
}

// Master returns the master side of the PTY.
func (p *PTY) Master() *Pipe {
	return p.master
}

// Slave returns the slave side of the PTY.
func (p *PTY) Slave() *Pipe {
	return p.slave
}

// Terminal returns the underlying terminal.
func (p *PTY) Terminal() *Terminal {
	return p.term
}

// Close closes the PTY.
func (p *PTY) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true

	p.master.readCond.Broadcast()
	p.master.writeCond.Broadcast()
	p.slave.readCond.Broadcast()
	p.slave.writeCond.Broadcast()

	return nil
}

// IsClosed returns whether the PTY is closed.
func (p *PTY) IsClosed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.closed
}

// Read reads data from the pipe.
func (p *Pipe) Read(data []byte) (int, error) {
	p.readCond.L.Lock()
	defer p.readCond.L.Unlock()

	for {
		p.pty.mu.RLock()
		closed := p.pty.closed
		p.pty.mu.RUnlock()

		if closed {
			return 0, io.EOF
		}

		if p.readBuf.Len() > 0 {
			return p.readBuf.Read(data)
		}

		// Wait for data
		p.readCond.Wait()
	}
}

// Write writes data to the pipe.
func (p *Pipe) Write(data []byte) (int, error) {
	p.writeCond.L.Lock()
	defer p.writeCond.L.Unlock()

	for {
		p.pty.mu.RLock()
		closed := p.pty.closed
		p.pty.mu.RUnlock()

		if closed {
			return 0, io.EOF
		}

		// For master: writes go to slave (terminal input)
		// For slave: writes go to master (terminal output)
		if p.isMaster {
			// Master writes to terminal (simulating process output)
			p.pty.term.Write(data)
			p.pty.slave.readCond.Broadcast()
		} else {
			// Slave writes are stored for master to read
			n, _ := p.writeBuf.Write(data)
			p.pty.master.readCond.Broadcast()
			return n, nil
		}

		return len(data), nil
	}
}

// ReadFromTerminal reads pending output from the terminal.
func (p *Pipe) ReadFromTerminal(data []byte) (int, error) {
	p.readCond.L.Lock()
	defer p.readCond.L.Unlock()

	p.pty.mu.RLock()
	output := p.pty.term.ReadAll()
	p.pty.mu.RUnlock()

	if len(output) == 0 {
		return 0, nil
	}

	n := copy(data, output)
	return n, nil
}

// WriteToTerminal writes data to be processed by the terminal.
func (p *Pipe) WriteToTerminal(data []byte) (int, error) {
	if !p.isMaster {
		return 0, ErrNotMaster
	}

	p.pty.mu.Lock()
	p.pty.term.Write(data)
	p.pty.mu.Unlock()

	return len(data), nil
}

// ParseANSI parses ANSI escape sequences from data.
func (p *Pipe) ParseANSI(data []byte) {
	if !p.isMaster {
		return
	}

	p.pty.mu.Lock()
	parser := NewParser(p.pty.term)
	parser.Parse(data)
	p.pty.mu.Unlock()
}

// GetTerminal returns the underlying terminal.
func (p *Pipe) GetTerminal() *Terminal {
	return p.pty.term
}

// Size returns the current terminal size.
func (p *Pipe) Size() (cols, rows int, err error) {
	p.pty.mu.RLock()
	defer p.pty.mu.RUnlock()
	return p.pty.term.Cols, p.pty.term.Rows, nil
}

// Resize changes the terminal size.
func (p *Pipe) Resize(cols, rows int) error {
	return p.pty.term.Resize(cols, rows)
}

// Winsize represents the terminal size.
type Winsize struct {
	Rows uint16
	Cols uint16
	X    uint16
	Y    uint16
}

// GetWinsize returns the terminal size as Winsize.
func (p *Pipe) GetWinsize() Winsize {
	p.pty.mu.RLock()
	defer p.pty.mu.RUnlock()
	return Winsize{
		Rows: uint16(p.pty.term.Rows),
		Cols: uint16(p.pty.term.Cols),
	}
}

// SetWinsize sets the terminal size.
func (p *Pipe) SetWinsize(ws Winsize) error {
	return p.pty.term.Resize(int(ws.Cols), int(ws.Rows))
}

// DrainOutput drains all pending output from the terminal.
func (p *Pipe) DrainOutput() []byte {
	p.readCond.L.Lock()
	defer p.readCond.L.Unlock()

	p.pty.mu.RLock()
	output := p.pty.term.ReadAll()
	p.pty.mu.RUnlock()

	return output
}
