/*
Package storage provides block storage abstraction with caching,
write-ahead logging, RAID support, and snapshot management.
*/
package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// WAL errors
var (
	ErrWALClosed       = errors.New("WAL is closed")
	ErrWALCorrupted    = errors.New("WAL is corrupted")
	ErrWALNotFound     = errors.New("WAL record not found")
	ErrWALFull         = errors.New("WAL is full")
	ErrWALSeqMismatch  = errors.New("sequence number mismatch")
	ErrInvalidWALEntry = errors.New("invalid WAL entry")
)

// WALRecordType defines the type of WAL record.
type WALRecordType uint8

const (
	// WALRecordBlock indicates a block write record.
	WALRecordBlock WALRecordType = iota
	// WALRecordCommit indicates a transaction commit.
	WALRecordCommit
	// WALRecordCheckpoint indicates a checkpoint.
	WALRecordCheckpoint
	// WALRecordBegin indicates transaction begin.
	WALRecordBegin
	// WALRecordEnd indicates transaction end.
	WALRecordEnd
)

// WALRecord represents a write-ahead log entry.
type WALRecord struct {
	Sequence  uint64        // Unique sequence number
	Type      WALRecordType // Record type
	Block     uint64        // Block number (for block records)
	Data      []byte        // Block data
	Timestamp time.Time     // When record was written
	Checksum  uint32        // Record checksum
	CommitSeq uint64        // Commit sequence number (for commit records)
}

// WriteAheadLog provides durability through write-ahead logging.
type WriteAheadLog struct {
	file       *os.File
	path       string
	blockSize  int
	sequence   uint64
	commitSeq  uint64
	headerSize int
	closed     bool
	mu         sync.RWMutex
	cond       *sync.Cond
	// Recovery
	records []*WALRecord
	maxSeq  uint64
}

// NewWriteAheadLog creates a new WAL.
func NewWriteAheadLog(path string, blockSize int, maxSize int64) (*WriteAheadLog, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create WAL directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL: %w", err)
	}

	wal := &WriteAheadLog{
		file:       file,
		path:       path,
		blockSize:  blockSize,
		headerSize: 32, // Sequence(8) + Type(1) + Block(8) + DataLen(4) + Timestamp(8) + Checksum(4)
		sequence:   0,
		commitSeq:  0,
		cond:       sync.NewCond(&sync.Mutex{}),
	}

	// Get current file size to determine starting sequence
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	// Calculate initial sequence from file size
	wal.sequence = uint64(stat.Size()) / uint64(wal.headerSize+blockSize)
	wal.maxSeq = wal.sequence

	return wal, nil
}

// OpenWriteAheadLog opens an existing WAL for recovery.
func OpenWriteAheadLog(path string, blockSize int) (*WriteAheadLog, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL: %w", err)
	}

	wal := &WriteAheadLog{
		file:       file,
		path:       path,
		blockSize:  blockSize,
		headerSize: 32,
		sequence:   0,
		commitSeq:  0,
		cond:       sync.NewCond(&sync.Mutex{}),
	}

	return wal, nil
}

// Append adds a record to the WAL.
func (w *WriteAheadLog) Append(record *WALRecord) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrWALClosed
	}

	w.sequence++
	record.Sequence = w.sequence
	record.Timestamp = time.Now()
	record.Checksum = w.calculateChecksum(record)

	// Serialize record
	data, err := w.serializeRecord(record)
	if err != nil {
		return fmt.Errorf("failed to serialize record: %w", err)
	}

	// Write to file
	if _, err := w.file.Write(data); err != nil {
		return fmt.Errorf("failed to write record: %w", err)
	}

	// Keep in memory for recovery
	w.records = append(w.records, record)
	w.maxSeq = w.sequence

	w.cond.Broadcast()
	return nil
}

// WriteBlock writes a block to the WAL.
func (w *WriteAheadLog) WriteBlock(block uint64, data []byte) error {
	record := &WALRecord{
		Type:  WALRecordBlock,
		Block: block,
		Data:  data,
	}
	return w.Append(record)
}

// BeginTransaction starts a new transaction.
func (w *WriteAheadLog) BeginTransaction() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.sequence++
	seq := w.sequence

	record := &WALRecord{
		Sequence:  seq,
		Type:      WALRecordBegin,
		Timestamp: time.Now(),
	}
	w.records = append(w.records, record)

	return seq
}

// EndTransaction ends a transaction.
func (w *WriteAheadLog) EndTransaction(beginSeq uint64) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.sequence++
	record := &WALRecord{
		Sequence:  w.sequence,
		Type:      WALRecordEnd,
		Timestamp: time.Now(),
	}

	data, err := w.serializeRecord(record)
	if err != nil {
		return err
	}

	_, err = w.file.Write(data)
	return err
}

// Commit marks a transaction as committed.
func (w *WriteAheadLog) Commit(beginSeq uint64) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.commitSeq++
	record := &WALRecord{
		Sequence:  w.sequence + 1,
		Type:      WALRecordCommit,
		CommitSeq: w.commitSeq,
		Timestamp: time.Now(),
	}
	w.sequence++

	data, err := w.serializeRecord(record)
	if err != nil {
		return err
	}

	if _, err := w.file.Write(data); err != nil {
		return fmt.Errorf("failed to write commit: %w", err)
	}

	w.records = append(w.records, record)

	return nil
}

// Checkpoint marks a checkpoint in the WAL.
func (w *WriteAheadLog) Checkpoint() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.sequence++
	record := &WALRecord{
		Sequence:  w.sequence,
		Type:      WALRecordCheckpoint,
		Timestamp: time.Now(),
	}

	data, err := w.serializeRecord(record)
	if err != nil {
		return err
	}

	if _, err := w.file.Write(data); err != nil {
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}

	w.records = append(w.records, record)

	return nil
}

// ReadRecord reads a record by sequence number.
func (w *WriteAheadLog) ReadRecord(seq uint64) (*WALRecord, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Check in-memory records first
	for _, record := range w.records {
		if record.Sequence == seq {
			return record, nil
		}
	}

	return nil, ErrWALNotFound
}

// GetRecordsSince returns all records since a given sequence.
func (w *WriteAheadLog) GetRecordsSince(seq uint64) []*WALRecord {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var result []*WALRecord
	for _, record := range w.records {
		if record.Sequence > seq {
			result = append(result, record)
		}
	}
	return result
}

// GetUncommittedRecords returns records not yet committed.
func (w *WriteAheadLog) GetUncommittedRecords() []*WALRecord {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Find the last commit
	var lastCommit uint64
	for _, record := range w.records {
		if record.Type == WALRecordCommit && record.CommitSeq > lastCommit {
			lastCommit = record.CommitSeq
		}
	}

	var result []*WALRecord
	for _, record := range w.records {
		if record.Sequence > lastCommit && record.Type == WALRecordBlock {
			result = append(result, record)
		}
	}
	return result
}

// Flush ensures all records are persisted.
func (w *WriteAheadLog) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrWALClosed
	}

	return w.file.Sync()
}

// Truncate removes committed records up to a sequence.
func (w *WriteAheadLog) Truncate(upToSeq uint64) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrWALClosed
	}

	// Filter records
	var newRecords []*WALRecord
	for _, record := range w.records {
		if record.Sequence > upToSeq {
			newRecords = append(newRecords, record)
		}
	}
	w.records = newRecords

	// Truncate file (simplified - in production, you'd need to rebuild the file)
	return nil
}

// Close closes the WAL.
func (w *WriteAheadLog) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	w.closed = true
	w.cond.Broadcast()

	if err := w.file.Sync(); err != nil {
		return err
	}

	return w.file.Close()
}

// RecoveryInfo contains information for recovery.
type RecoveryInfo struct {
	LastSequence  uint64
	LastCommitSeq uint64
	CheckpointSeq uint64
	Uncommitted   []*WALRecord
	Corrupted     bool
}

// Recover performs recovery from the WAL.
func (w *WriteAheadLog) Recover() (*RecoveryInfo, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	info := &RecoveryInfo{
		LastSequence:  w.sequence,
		LastCommitSeq: w.commitSeq,
	}

	// Scan all records
	for _, record := range w.records {
		switch record.Type {
		case WALRecordCommit:
			if record.CommitSeq > info.LastCommitSeq {
				info.LastCommitSeq = record.CommitSeq
			}
		case WALRecordCheckpoint:
			info.CheckpointSeq = record.Sequence
		}
	}

	// Get uncommitted records
	info.Uncommitted = w.GetUncommittedRecords()

	return info, nil
}

// GetCommitSequence returns the current commit sequence.
func (w *WriteAheadLog) GetCommitSequence() uint64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.commitSeq
}

// GetSequence returns the current sequence number.
func (w *WriteAheadLog) GetSequence() uint64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.sequence
}

// serializeRecord converts a record to bytes.
func (w *WriteAheadLog) serializeRecord(record *WALRecord) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write header
	if err := binary.Write(buf, binary.BigEndian, record.Sequence); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, record.Type); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, record.Block); err != nil {
		return nil, err
	}
	dataLen := uint32(len(record.Data))
	if err := binary.Write(buf, binary.BigEndian, dataLen); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, record.Timestamp.UnixNano()); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, record.Checksum); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, record.CommitSeq); err != nil {
		return nil, err
	}

	// Write data
	if _, err := buf.Write(record.Data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// deserializeRecord converts bytes to a record.
func (w *WriteAheadLog) deserializeRecord(data []byte) (*WALRecord, error) {
	if len(data) < w.headerSize {
		return nil, ErrInvalidWALEntry
	}

	buf := bytes.NewReader(data)

	var sequence uint64
	if err := binary.Read(buf, binary.BigEndian, &sequence); err != nil {
		return nil, err
	}

	var recType WALRecordType
	if err := binary.Read(buf, binary.BigEndian, &recType); err != nil {
		return nil, err
	}

	var block uint64
	if err := binary.Read(buf, binary.BigEndian, &block); err != nil {
		return nil, err
	}

	var dataLen uint32
	if err := binary.Read(buf, binary.BigEndian, &dataLen); err != nil {
		return nil, err
	}

	var timestamp int64
	if err := binary.Read(buf, binary.BigEndian, &timestamp); err != nil {
		return nil, err
	}

	var checksum uint32
	if err := binary.Read(buf, binary.BigEndian, &checksum); err != nil {
		return nil, err
	}

	var commitSeq uint64
	if err := binary.Read(buf, binary.BigEndian, &commitSeq); err != nil {
		return nil, err
	}

	recordData := make([]byte, dataLen)
	if _, err := io.ReadFull(buf, recordData); err != nil {
		return nil, err
	}

	record := &WALRecord{
		Sequence:  sequence,
		Type:      recType,
		Block:     block,
		Data:      recordData,
		Timestamp: time.Unix(0, timestamp),
		Checksum:  checksum,
		CommitSeq: commitSeq,
	}

	// Verify checksum
	if record.Checksum != w.calculateChecksum(record) {
		return nil, ErrWALCorrupted
	}

	return record, nil
}

// calculateChecksum computes a checksum for a record.
func (w *WriteAheadLog) calculateChecksum(record *WALRecord) uint32 {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, record.Sequence)
	binary.Write(buf, binary.BigEndian, record.Type)
	binary.Write(buf, binary.BigEndian, record.Block)
	binary.Write(buf, binary.BigEndian, uint32(len(record.Data)))
	binary.Write(buf, binary.BigEndian, record.Timestamp.UnixNano())
	return CalculateChecksum(buf.Bytes())
}

// WALStats contains WAL statistics.
type WALStats struct {
	Sequence    uint64
	CommitSeq   uint64
	RecordCount int
	FileSize    int64
	Path        string
}

// Stats returns WAL statistics.
func (w *WriteAheadLog) Stats() (*WALStats, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	stat, err := os.Stat(w.path)
	if err != nil {
		return nil, err
	}

	return &WALStats{
		Sequence:    w.sequence,
		CommitSeq:   w.commitSeq,
		RecordCount: len(w.records),
		FileSize:    stat.Size(),
		Path:        w.path,
	}, nil
}

// WALManager manages multiple WALs.
type WALManager struct {
	walDir    string
	blockSize int
	active    *WriteAheadLog
	closed    bool
	mu        sync.Mutex
}

// NewWALManager creates a new WAL manager.
func NewWALManager(walDir string, blockSize int) (*WALManager, error) {
	if err := os.MkdirAll(walDir, 0755); err != nil {
		return nil, err
	}

	return &WALManager{
		walDir:    walDir,
		blockSize: blockSize,
	}, nil
}

// CreateWAL creates a new WAL.
func (m *WALManager) CreateWAL(name string) (*WriteAheadLog, error) {
	path := filepath.Join(m.walDir, name+".wal")
	return NewWriteAheadLog(path, m.blockSize, 0)
}

// Close closes all WALs.
func (m *WALManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true

	if m.active != nil {
		m.active.Close()
	}

	return nil
}
