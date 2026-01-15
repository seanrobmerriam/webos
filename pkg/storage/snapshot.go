/*
Package storage provides block storage abstraction with caching,
write-ahead logging, RAID support, and snapshot management.
*/
package storage

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Snapshot errors
var (
	ErrSnapshotNotFound  = errors.New("snapshot not found")
	ErrSnapshotCorrupted = errors.New("snapshot is corrupted")
	ErrSnapshotInUse     = errors.New("snapshot is currently in use")
	ErrInvalidSnapshotID = errors.New("invalid snapshot ID")
	ErrSnapshotLimit     = errors.New("maximum snapshot limit reached")
)

// Snapshot represents a point-in-time copy of the storage.
type Snapshot struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	CreatedAt   time.Time              `json:"createdAt"`
	Description string                 `json:"description"`
	DeviceInfo  DeviceInfo             `json:"deviceInfo"`
	Metadata    map[string]interface{} `json:"metadata"`
	Size        int64                  `json:"size"`
	Blocks      uint64                 `json:"blocks"`
	Checksum    uint32                 `json:"checksum"`
}

// SnapshotManager manages storage snapshots.
type SnapshotManager struct {
	device       BlockDevice
	snapshotDir  string
	snapshots    map[string]*Snapshot
	activeSnap   map[string]*SnapshotFile
	maxSnapshots int
	mu           sync.RWMutex
}

// SnapshotFile represents an open snapshot for reading or writing.
type SnapshotFile struct {
	snapshot *Snapshot
	file     *os.File
	offset   int64
	readonly bool
}

// NewSnapshotManager creates a new snapshot manager.
func NewSnapshotManager(device BlockDevice, snapshotDir string, maxSnapshots int) (*SnapshotManager, error) {
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	mgr := &SnapshotManager{
		device:       device,
		snapshotDir:  snapshotDir,
		snapshots:    make(map[string]*Snapshot),
		activeSnap:   make(map[string]*SnapshotFile),
		maxSnapshots: maxSnapshots,
	}

	// Load existing snapshots
	if err := mgr.loadSnapshots(); err != nil {
		return nil, fmt.Errorf("failed to load snapshots: %w", err)
	}

	return mgr, nil
}

// Create creates a new snapshot.
func (m *SnapshotManager) Create(name, description string) (*Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check snapshot limit
	if len(m.snapshots) >= m.maxSnapshots {
		return nil, ErrSnapshotLimit
	}

	// Generate snapshot ID
	id := generateSnapshotID()
	now := time.Now()

	snapshot := &Snapshot{
		ID:          id,
		Name:        name,
		CreatedAt:   now,
		Description: description,
		DeviceInfo:  GetInfo(m.device),
		Metadata:    make(map[string]interface{}),
		Blocks:      m.device.BlockCount(),
	}

	// Create snapshot file
	snapPath := m.getSnapshotPath(id)
	file, err := os.OpenFile(snapPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot file: %w", err)
	}

	// Write header
	header := m.createSnapshotHeader(snapshot)
	if _, err := file.Write(header); err != nil {
		file.Close()
		os.Remove(snapPath)
		return nil, fmt.Errorf("failed to write snapshot header: %w", err)
	}

	// Copy data blocks
	blockSize := m.device.BlockSize()
	data := make([]byte, blockSize)
	totalSize := int64(len(header))

	for block := uint64(0); block < m.device.BlockCount(); block++ {
		if err := m.device.Read(block, data); err != nil {
			file.Close()
			os.Remove(snapPath)
			return nil, fmt.Errorf("failed to read block %d: %w", block, err)
		}

		if _, err := file.Write(data); err != nil {
			file.Close()
			os.Remove(snapPath)
			return nil, fmt.Errorf("failed to write block %d: %w", block, err)
		}

		totalSize += int64(blockSize)
	}

	// Update snapshot info
	snapshot.Size = totalSize
	snapshot.Checksum = CalculateChecksum(header)

	// Write updated header
	file.Seek(0, os.SEEK_SET)
	header = m.createSnapshotHeader(snapshot)
	if _, err := file.Write(header); err != nil {
		file.Close()
		os.Remove(snapPath)
		return nil, fmt.Errorf("failed to update snapshot header: %w", err)
	}

	file.Close()

	// Save snapshot metadata
	if err := m.saveSnapshotMetadata(snapshot); err != nil {
		return nil, err
	}

	m.snapshots[id] = snapshot
	return snapshot, nil
}

// Open opens a snapshot for reading.
func (m *SnapshotManager) Open(id string) (*SnapshotFile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	snapshot, ok := m.snapshots[id]
	if !ok {
		return nil, ErrSnapshotNotFound
	}

	// Check if already open
	if _, ok := m.activeSnap[id]; ok {
		return nil, ErrSnapshotInUse
	}

	snapPath := m.getSnapshotPath(id)
	file, err := os.Open(snapPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open snapshot: %w", err)
	}

	sf := &SnapshotFile{
		snapshot: snapshot,
		file:     file,
		offset:   0,
		readonly: true,
	}

	m.activeSnap[id] = sf
	return sf, nil
}

// Restore restores a snapshot to the device.
func (m *SnapshotManager) Restore(id string) error {
	sf, err := m.Open(id)
	if err != nil {
		return err
	}
	defer m.Close(sf)

	// Skip header
	headerSize := m.getSnapshotHeaderSize()
	_, err = sf.file.Seek(int64(headerSize), os.SEEK_SET)
	if err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	// Read and write blocks
	blockSize := m.device.BlockSize()
	data := make([]byte, blockSize)

	for block := uint64(0); block < sf.snapshot.Blocks; block++ {
		n, err := io.ReadFull(sf.file, data)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read block %d: %w", block, err)
		}
		if n < blockSize {
			// Pad with zeros
			for i := n; i < blockSize; i++ {
				data[i] = 0
			}
		}

		if err := m.device.Write(block, data); err != nil {
			return fmt.Errorf("failed to write block %d: %w", block, err)
		}
	}

	return m.device.Flush()
}

// Delete deletes a snapshot.
func (m *SnapshotManager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.snapshots[id]
	if !ok {
		return ErrSnapshotNotFound
	}

	// Check if in use
	if _, ok := m.activeSnap[id]; ok {
		return ErrSnapshotInUse
	}

	// Remove files
	snapPath := m.getSnapshotPath(id)
	metaPath := m.getMetadataPath(id)

	os.Remove(snapPath)
	os.Remove(metaPath)

	delete(m.snapshots, id)
	return nil
}

// List returns all snapshots.
func (m *SnapshotManager) List() []*Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshots := make([]*Snapshot, 0, len(m.snapshots))
	for _, snap := range m.snapshots {
		snapshots = append(snapshots, snap)
	}

	return snapshots
}

// Get returns a specific snapshot.
func (m *SnapshotManager) Get(id string) (*Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snap, ok := m.snapshots[id]
	if !ok {
		return nil, ErrSnapshotNotFound
	}

	return snap, nil
}

// Close closes a snapshot file.
func (m *SnapshotManager) Close(sf *SnapshotFile) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.activeSnap, sf.snapshot.ID)
	return sf.file.Close()
}

// Read reads from a snapshot file.
func (sf *SnapshotFile) Read(data []byte) (int, error) {
	return sf.file.Read(data)
}

// Seek seeks in a snapshot file.
func (sf *SnapshotFile) Seek(offset int64, whence int) (int64, error) {
	return sf.file.Seek(offset, whence)
}

// Stat returns snapshot info.
func (sf *SnapshotFile) Stat() (os.FileInfo, error) {
	return &snapshotFileInfo{sf.snapshot}, nil
}

type snapshotFileInfo struct {
	*Snapshot
}

func (i *snapshotFileInfo) Name() string       { return i.ID }
func (i *snapshotFileInfo) Size() int64        { return i.Snapshot.Size }
func (i *snapshotFileInfo) Mode() os.FileMode  { return 0600 }
func (i *snapshotFileInfo) ModTime() time.Time { return i.Snapshot.CreatedAt }
func (i *snapshotFileInfo) IsDir() bool        { return false }
func (i *snapshotFileInfo) Sys() interface{}   { return nil }

// Snapshot returns the snapshot metadata.
func (sf *SnapshotFile) Snapshot() *Snapshot {
	return sf.snapshot
}

// Read reads a block from a snapshot.
func (m *SnapshotManager) ReadBlock(sf *SnapshotFile, block uint64, data []byte) error {
	headerSize := m.getSnapshotHeaderSize()
	offset := int64(headerSize) + int64(block)*int64(m.device.BlockSize())

	_, err := sf.file.Seek(offset, os.SEEK_SET)
	if err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	_, err = io.ReadFull(sf.file, data)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read: %w", err)
	}

	return nil
}

// getSnapshotPath returns the file path for a snapshot.
func (m *SnapshotManager) getSnapshotPath(id string) string {
	return filepath.Join(m.snapshotDir, id+".snap")
}

// getMetadataPath returns the metadata file path for a snapshot.
func (m *SnapshotManager) getMetadataPath(id string) string {
	return filepath.Join(m.snapshotDir, id+".json")
}

// getSnapshotHeaderSize returns the size of the snapshot header.
func (m *SnapshotManager) getSnapshotHeaderSize() int {
	return 256 // Fixed header size
}

// createSnapshotHeader creates the header bytes for a snapshot.
func (m *SnapshotManager) createSnapshotHeader(snapshot *Snapshot) []byte {
	header := make([]byte, 256)

	// Write magic number
	copy(header[0:8], "SNAPSHOT")

	// Write version
	binary.BigEndian.PutUint16(header[8:10], 1)

	// Write block size
	binary.BigEndian.PutUint32(header[10:14], uint32(m.device.BlockSize()))

	// Write block count
	binary.BigEndian.PutUint64(header[14:22], snapshot.Blocks)

	// Write created timestamp
	binary.BigEndian.PutUint64(header[22:30], uint64(snapshot.CreatedAt.UnixNano()))

	// Write ID length and ID
	header[30] = byte(len(snapshot.ID))
	copy(header[31:31+len(snapshot.ID)], []byte(snapshot.ID))

	// Write name length and name
	offset := 31 + len(snapshot.ID)
	header[offset] = byte(len(snapshot.Name))
	copy(header[offset+1:offset+1+len(snapshot.Name)], []byte(snapshot.Name))

	// Write description length and description
	offset += 1 + len(snapshot.Name)
	header[offset] = byte(len(snapshot.Description))
	copy(header[offset+1:offset+1+len(snapshot.Description)], []byte(snapshot.Description))

	// Write device type
	offset += 1 + len(snapshot.Description)
	copy(header[offset:offset+16], snapshot.DeviceInfo.Type)

	return header
}

// parseSnapshotHeader parses a snapshot header.
func (m *SnapshotManager) parseSnapshotHeader(data []byte) (*Snapshot, error) {
	if len(data) < 256 {
		return nil, ErrSnapshotCorrupted
	}

	// Verify magic number
	if string(data[0:8]) != "SNAPSHOT" {
		return nil, ErrSnapshotCorrupted
	}

	snapshot := &Snapshot{}

	snapshot.Blocks = binary.BigEndian.Uint64(data[14:22])
	snapshot.CreatedAt = time.Unix(0, int64(binary.BigEndian.Uint64(data[22:30])))

	// Read ID
	idLen := int(data[30])
	snapshot.ID = string(data[31 : 31+idLen])

	// Read name
	offset := 31 + idLen
	nameLen := int(data[offset])
	snapshot.Name = string(data[offset+1 : offset+1+nameLen])

	// Read description
	offset += 1 + nameLen
	descLen := int(data[offset])
	snapshot.Description = string(data[offset+1 : offset+1+descLen])

	return snapshot, nil
}

// loadSnapshots loads existing snapshots from disk.
func (m *SnapshotManager) loadSnapshots() error {
	entries, err := os.ReadDir(m.snapshotDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			id := entry.Name()[:len(entry.Name())-5]
			metaPath := filepath.Join(m.snapshotDir, entry.Name())

			data, err := os.ReadFile(metaPath)
			if err != nil {
				continue
			}

			var snapshot Snapshot
			if err := json.Unmarshal(data, &snapshot); err != nil {
				continue
			}

			m.snapshots[id] = &snapshot
		}
	}

	return nil
}

// saveSnapshotMetadata saves snapshot metadata to disk.
func (m *SnapshotManager) saveSnapshotMetadata(snapshot *Snapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metaPath := m.getMetadataPath(snapshot.ID)
	return os.WriteFile(metaPath, data, 0600)
}

// generateSnapshotID generates a unique snapshot ID.
func generateSnapshotID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

// SnapshotDiff represents the differences between two snapshots.
type SnapshotDiff struct {
	AddedBlocks    []uint64 `json:"addedBlocks"`
	RemovedBlocks  []uint64 `json:"removedBlocks"`
	ModifiedBlocks []uint64 `json:"modifiedBlocks"`
	BlockSize      int      `json:"blockSize"`
}

// Diff calculates the differences between two snapshots.
func (m *SnapshotManager) Diff(from, to string) (*SnapshotDiff, error) {
	fromSnap, err := m.Get(from)
	if err != nil {
		return nil, err
	}

	toSnap, err := m.Get(to)
	if err != nil {
		return nil, err
	}

	diff := &SnapshotDiff{
		BlockSize: m.device.BlockSize(),
	}

	// Compare blocks
	blockData1 := make([]byte, m.device.BlockSize())
	blockData2 := make([]byte, m.device.BlockSize())

	sf1, err := m.Open(from)
	if err != nil {
		return nil, err
	}
	defer m.Close(sf1)

	sf2, err := m.Open(to)
	if err != nil {
		return nil, err
	}
	defer m.Close(sf2)

	headerSize := m.getSnapshotHeaderSize()
	maxBlocks := fromSnap.Blocks
	if toSnap.Blocks > maxBlocks {
		maxBlocks = toSnap.Blocks
	}

	for block := uint64(0); block < maxBlocks; block++ {
		offset := int64(headerSize) + int64(block)*int64(m.device.BlockSize())

		if block < fromSnap.Blocks {
			sf1.file.Seek(offset, os.SEEK_SET)
			io.ReadFull(sf1.file, blockData1)
		}

		if block < toSnap.Blocks {
			sf2.file.Seek(offset, os.SEEK_SET)
			io.ReadFull(sf2.file, blockData2)
		}

		switch {
		case block >= fromSnap.Blocks && block < toSnap.Blocks:
			diff.AddedBlocks = append(diff.AddedBlocks, block)
		case block < fromSnap.Blocks && block >= toSnap.Blocks:
			diff.RemovedBlocks = append(diff.RemovedBlocks, block)
		case block < fromSnap.Blocks && block < toSnap.Blocks && !bytes.Equal(blockData1, blockData2):
			diff.ModifiedBlocks = append(diff.ModifiedBlocks, block)
		}
	}

	return diff, nil
}

// IncrementalSnapshot creates an incremental snapshot.
type IncrementalSnapshot struct {
	ParentID string        `json:"parentID"`
	Snapshot *Snapshot     `json:"snapshot"`
	Diff     *SnapshotDiff `json:"diff"`
}

// CreateIncremental creates an incremental snapshot.
func (m *SnapshotManager) CreateIncremental(parentID, name, description string) (*IncrementalSnapshot, error) {
	_, err := m.Get(parentID)
	if err != nil {
		return nil, err
	}

	// Create new snapshot
	snapshot, err := m.Create(name, description)
	if err != nil {
		return nil, err
	}

	// Calculate diff with parent
	diff, err := m.Diff(parentID, snapshot.ID)
	if err != nil {
		m.Delete(snapshot.ID)
		return nil, err
	}

	return &IncrementalSnapshot{
		ParentID: parentID,
		Snapshot: snapshot,
		Diff:     diff,
	}, nil
}
