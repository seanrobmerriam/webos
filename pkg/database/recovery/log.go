// Package recovery provides WAL (Write-Ahead Log) for database recovery.
package recovery

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// WAL errors.
var (
	ErrWALClosed       = errors.New("WAL is closed")
	ErrWALCorrupted    = errors.New("WAL file is corrupted")
	ErrInvalidLogEntry = errors.New("invalid log entry")
	ErrLogFull         = errors.New("WAL log is full")
)

// LogOperation represents the type of operation in the log.
type LogOperation uint8

const (
	// OpBegin marks the beginning of a transaction.
	OpBegin LogOperation = iota
	// OpInsert represents an insert operation.
	OpInsert
	// OpUpdate represents an update operation.
	OpUpdate
	// OpDelete represents a delete operation.
	OpDelete
	// OpCommit marks the transaction commit.
	OpCommit
	// OpRollback marks the transaction rollback.
	OpRollback
	// OpCheckpoint marks a checkpoint.
	OpCheckpoint
)

// String returns a string representation of the operation.
func (o LogOperation) String() string {
	switch o {
	case OpBegin:
		return "BEGIN"
	case OpInsert:
		return "INSERT"
	case OpUpdate:
		return "UPDATE"
	case OpDelete:
		return "DELETE"
	case OpCommit:
		return "COMMIT"
	case OpRollback:
		return "ROLLBACK"
	case OpCheckpoint:
		return "CHECKPOINT"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a single entry in the WAL.
type LogEntry struct {
	TxID        uint64       // Transaction ID
	Operation   LogOperation // Type of operation
	TableName   string       // Table name
	RowID       uint64       // Row ID (for DML operations)
	BeforeImage []byte       // Row data before modification (for UPDATE/DELETE)
	AfterImage  []byte       // Row data after modification (for INSERT/UPDATE)
	Timestamp   int64        // Entry timestamp
	LSN         uint64       // Log Sequence Number
}

// WAL represents the Write-Ahead Log.
type WAL struct {
	path        string        // Path to WAL file
	file        *os.File      // WAL file handle
	mu          sync.RWMutex  // Mutex for synchronization
	closed      bool          // Whether WAL is closed
	lsn         uint64        // Current Log Sequence Number
	buffer      *bytes.Buffer // Write buffer
	maxSize     int64         // Maximum WAL size
	currentSize int64         // Current WAL size
}

// NewWAL creates a new Write-Ahead Log.
func NewWAL(path string, maxSize int64) (*WAL, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create WAL directory: %w", err)
	}

	wal := &WAL{
		path:        path,
		buffer:      new(bytes.Buffer),
		maxSize:     maxSize,
		currentSize: 0,
	}

	// Open or create WAL file
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open WAL file: %w", err)
	}
	wal.file = f

	// Get current file size
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("stat WAL file: %w", err)
	}
	wal.currentSize = info.Size()

	return wal, nil
}

// Write writes a log entry to the WAL.
func (w *WAL) Write(entry *LogEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrWALClosed
	}

	// Set LSN and timestamp
	entry.LSN = w.incrementLSN()
	entry.Timestamp = time.Now().Unix()

	// Serialize the entry
	data, err := entry.Serialize()
	if err != nil {
		return fmt.Errorf("serialize log entry: %w", err)
	}

	// Check if we need to rotate
	if w.currentSize+int64(len(data)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return fmt.Errorf("rotate WAL: %w", err)
		}
	}

	// Write entry length and data
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))

	if _, err := w.file.Write(lenBuf); err != nil {
		return fmt.Errorf("write entry length: %w", err)
	}
	if _, err := w.file.Write(data); err != nil {
		return fmt.Errorf("write entry data: %w", err)
	}

	w.currentSize += int64(len(lenBuf)) + int64(len(data))

	// Flush to disk
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("sync WAL: %w", err)
	}

	return nil
}

// Read reads all log entries from the WAL.
func (w *WAL) Read() ([]*LogEntry, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.closed {
		return nil, ErrWALClosed
	}

	// Seek to beginning
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek WAL: %w", err)
	}

	var entries []*LogEntry
	for {
		// Read entry length
		lenBuf := make([]byte, 4)
		n, err := w.file.Read(lenBuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read entry length: %w", err)
		}
		if n != 4 {
			return nil, ErrWALCorrupted
		}

		length := binary.BigEndian.Uint32(lenBuf)

		// Read entry data
		data := make([]byte, length)
		n, err = w.file.Read(data)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("read entry data: %w", err)
		}
		if uint32(n) != length {
			return nil, ErrWALCorrupted
		}

		// Deserialize entry
		entry := &LogEntry{}
		if err := entry.Deserialize(data); err != nil {
			return nil, fmt.Errorf("deserialize entry: %w", err)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// GetLsn returns the current LSN.
func (w *WAL) GetLsn() uint64 {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.lsn
}

// Close closes the WAL.
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	w.closed = true

	// Flush any pending data
	if w.buffer.Len() > 0 {
		if err := w.flush(); err != nil {
			return fmt.Errorf("flush buffer: %w", err)
		}
	}

	// Close file
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("close WAL file: %w", err)
	}

	return nil
}

// Truncate truncates the WAL up to the given LSN (used after checkpoint).
func (w *WAL) Truncate(lsn uint64) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrWALClosed
	}

	entries, err := w.Read()
	if err != nil {
		return fmt.Errorf("read WAL: %w", err)
	}

	// Reopen file in truncate mode
	w.file.Close()
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("reopen WAL file: %w", err)
	}
	w.file = f
	w.currentSize = 0
	w.lsn = 0

	// Write entries with LSN >= given LSN
	for _, entry := range entries {
		if entry.LSN >= lsn {
			data, err := entry.Serialize()
			if err != nil {
				return fmt.Errorf("serialize entry: %w", err)
			}

			lenBuf := make([]byte, 4)
			binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))

			if _, err := w.file.Write(lenBuf); err != nil {
				return fmt.Errorf("write entry length: %w", err)
			}
			if _, err := w.file.Write(data); err != nil {
				return fmt.Errorf("write entry data: %w", err)
			}

			w.currentSize += int64(len(lenBuf)) + int64(len(data))
			w.lsn = entry.LSN
		}
	}

	return nil
}

// rotate rotates the WAL file.
func (w *WAL) rotate() error {
	// Get current time for backup filename
	timestamp := time.Now().Format("20060102150405")
	backupPath := fmt.Sprintf("%s.%s", w.path, timestamp)

	// Close current file
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("close WAL file: %w", err)
	}

	// Rename current file to backup
	if err := os.Rename(w.path, backupPath); err != nil {
		return fmt.Errorf("rename WAL file: %w", err)
	}

	// Create new WAL file
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("create new WAL file: %w", err)
	}
	w.file = f
	w.currentSize = 0
	w.lsn = 0

	return nil
}

// flush flushes the write buffer to disk.
func (w *WAL) flush() error {
	if w.buffer.Len() == 0 {
		return nil
	}

	if _, err := w.file.Write(w.buffer.Bytes()); err != nil {
		return fmt.Errorf("write buffer: %w", err)
	}
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("sync WAL: %w", err)
	}

	w.buffer.Reset()
	return nil
}

// incrementLSN increments and returns the current LSN.
func (w *WAL) incrementLSN() uint64 {
	w.lsn++
	return w.lsn
}

// Serialize serializes the log entry to bytes.
func (e *LogEntry) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(e); err != nil {
		return nil, fmt.Errorf("encode log entry: %w", err)
	}

	return buf.Bytes(), nil
}

// Deserialize deserializes the log entry from bytes.
func (e *LogEntry) Deserialize(data []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(data))

	if err := dec.Decode(e); err != nil {
		return fmt.Errorf("decode log entry: %w", err)
	}

	return nil
}
