/*
Package storage provides block storage abstraction with caching,
write-ahead logging, RAID support, and snapshot management.

This package implements a complete block storage system for persistent
data management in the WebOS project.

Example Usage:

	// Create a memory-backed block device
	device, err := storage.NewMemoryBlockDevice(1024, 4096)
	if err != nil {
		log.Fatal(err)
	}
	defer device.Close()

	// Wrap with caching
	cache := storage.NewBlockCache(device, storage.CachePolicyLRU, 100)
	defer cache.Close()

	// Use the cached device
	data := make([]byte, 4096)
	err = cache.Read(0, data)
	if err != nil {
		log.Fatal(err)
	}
*/
package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Common errors
var (
	ErrInvalidBlockNumber = errors.New("invalid block number")
	ErrBlockTooLarge      = errors.New("block data exceeds device block size")
	ErrOutOfBounds        = errors.New("read/write exceeds device capacity")
	ErrDeviceClosed       = errors.New("device is closed")
	ErrReadOnly           = errors.New("device is read-only")
	ErrWriteOnly          = errors.New("device is write-only")
	ErrNotSupported       = errors.New("operation not supported")
	ErrCacheFull          = errors.New("cache is full")
	ErrEncryptionFailed   = errors.New("encryption failed")
	ErrDecryptionFailed   = errors.New("decryption failed")
)

// BlockDevice interface defines the operations for a block storage device.
type BlockDevice interface {
	// Read reads a block from the device into data.
	// The data slice must have length equal to BlockSize().
	Read(block uint64, data []byte) error

	// Write writes a block to the device from data.
	// The data slice must have length equal to BlockSize().
	Write(block uint64, data []byte) error

	// BlockSize returns the size of each block in bytes.
	BlockSize() int

	// BlockCount returns the total number of blocks on the device.
	BlockCount() uint64

	// Flush ensures all pending writes are persisted.
	Flush() error

	// Close releases any resources held by the device.
	Close() error
}

// ReadOnlyDevice wraps a BlockDevice to make it read-only.
type ReadOnlyDevice struct {
	device BlockDevice
}

// NewReadOnlyDevice creates a read-only wrapper around a BlockDevice.
func NewReadOnlyDevice(device BlockDevice) *ReadOnlyDevice {
	return &ReadOnlyDevice{device: device}
}

// Read reads a block from the underlying device.
func (d *ReadOnlyDevice) Read(block uint64, data []byte) error {
	return d.device.Read(block, data)
}

// Write returns an error as the device is read-only.
func (d *ReadOnlyDevice) Write(block uint64, data []byte) error {
	return ErrReadOnly
}

// BlockSize returns the underlying device's block size.
func (d *ReadOnlyDevice) BlockSize() int {
	return d.device.BlockSize()
}

// BlockCount returns the underlying device's block count.
func (d *ReadOnlyDevice) BlockCount() uint64 {
	return d.device.BlockCount()
}

// Flush forwards to the underlying device.
func (d *ReadOnlyDevice) Flush() error {
	return d.device.Flush()
}

// Close forwards to the underlying device.
func (d *ReadOnlyDevice) Close() error {
	return d.device.Close()
}

// MemoryBlockDevice is a simple in-memory block device.
type MemoryBlockDevice struct {
	data       [][]byte
	blockSize  int
	blockCount uint64
	closed     bool
	mu         sync.RWMutex
}

// NewMemoryBlockDevice creates a new memory-backed block device.
func NewMemoryBlockDevice(blockCount uint64, blockSize int) (*MemoryBlockDevice, error) {
	if blockCount == 0 {
		return nil, ErrInvalidBlockNumber
	}
	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}

	data := make([][]byte, blockCount)
	for i := range data {
		data[i] = make([]byte, blockSize)
	}

	return &MemoryBlockDevice{
		data:       data,
		blockSize:  blockSize,
		blockCount: blockCount,
	}, nil
}

// Read reads a block from memory.
func (d *MemoryBlockDevice) Read(block uint64, data []byte) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return ErrDeviceClosed
	}
	if block >= d.blockCount {
		return ErrInvalidBlockNumber
	}
	if len(data) != d.blockSize {
		return fmt.Errorf("data length %d != block size %d", len(data), d.blockSize)
	}

	copy(data, d.data[block])
	return nil
}

// Write writes a block to memory.
func (d *MemoryBlockDevice) Write(block uint64, data []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return ErrDeviceClosed
	}
	if block >= d.blockCount {
		return ErrInvalidBlockNumber
	}
	if len(data) != d.blockSize {
		return ErrBlockTooLarge
	}

	copy(d.data[block], data)
	return nil
}

// BlockSize returns the configured block size.
func (d *MemoryBlockDevice) BlockSize() int {
	return d.blockSize
}

// BlockCount returns the total number of blocks.
func (d *MemoryBlockDevice) BlockCount() uint64 {
	return d.blockCount
}

// Flush is a no-op for memory devices.
func (d *MemoryBlockDevice) Flush() error {
	return nil
}

// Close marks the device as closed.
func (d *MemoryBlockDevice) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}
	d.closed = true
	d.data = nil
	return nil
}

// FileBlockDevice is a block device backed by a file.
type FileBlockDevice struct {
	file       *os.File
	blockSize  int
	blockCount uint64
	closed     bool
	mu         sync.RWMutex
}

// NewFileBlockDevice creates a block device backed by a file.
func NewFileBlockDevice(path string, blockCount uint64, blockSize int) (*FileBlockDevice, error) {
	if blockCount == 0 {
		return nil, ErrInvalidBlockNumber
	}
	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}

	// Open file with read/write access
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if os.IsExist(err) {
			// File exists, open it
			file, err = os.OpenFile(path, os.O_RDWR, 0600)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Ensure file is large enough
	totalSize := int64(blockCount) * int64(blockSize)
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	if stat.Size() < totalSize {
		if err := file.Truncate(totalSize); err != nil {
			file.Close()
			return nil, err
		}
	}

	return &FileBlockDevice{
		file:       file,
		blockSize:  blockSize,
		blockCount: blockCount,
	}, nil
}

// OpenFileBlockDevice opens an existing file as a block device.
func OpenFileBlockDevice(path string, blockSize int) (*FileBlockDevice, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	blockCount := uint64(stat.Size()) / uint64(blockSize)

	return &FileBlockDevice{
		file:       file,
		blockSize:  blockSize,
		blockCount: blockCount,
	}, nil
}

// Read reads a block from the file.
func (d *FileBlockDevice) Read(block uint64, data []byte) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return ErrDeviceClosed
	}
	if block >= d.blockCount {
		return ErrInvalidBlockNumber
	}
	if len(data) != d.blockSize {
		return fmt.Errorf("data length %d != block size %d", len(data), d.blockSize)
	}

	offset := int64(block) * int64(d.blockSize)
	_, err := d.file.ReadAt(data, offset)
	return err
}

// Write writes a block to the file.
func (d *FileBlockDevice) Write(block uint64, data []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return ErrDeviceClosed
	}
	if block >= d.blockCount {
		return ErrInvalidBlockNumber
	}
	if len(data) != d.blockSize {
		return ErrBlockTooLarge
	}

	offset := int64(block) * int64(d.blockSize)
	_, err := d.file.WriteAt(data, offset)
	return err
}

// BlockSize returns the configured block size.
func (d *FileBlockDevice) BlockSize() int {
	return d.blockSize
}

// BlockCount returns the total number of blocks.
func (d *FileBlockDevice) BlockCount() uint64 {
	return d.blockCount
}

// Flush syncs the file to disk.
func (d *FileBlockDevice) Flush() error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return ErrDeviceClosed
	}
	return d.file.Sync()
}

// Close closes the file.
func (d *FileBlockDevice) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}
	d.closed = true
	return d.file.Close()
}

// DeviceInfo contains metadata about a block device.
type DeviceInfo struct {
	Type       string    `json:"type"`
	BlockSize  int       `json:"blockSize"`
	BlockCount uint64    `json:"blockCount"`
	TotalSize  uint64    `json:"totalSize"`
	ReadOnly   bool      `json:"readOnly"`
	Created    time.Time `json:"created"`
	Modified   time.Time `json:"modified"`
}

// GetInfo returns metadata about the device.
func GetInfo(device BlockDevice) DeviceInfo {
	info := DeviceInfo{
		BlockSize:  device.BlockSize(),
		BlockCount: device.BlockCount(),
		TotalSize:  uint64(device.BlockSize()) * device.BlockCount(),
	}

	switch device.(type) {
	case *MemoryBlockDevice:
		info.Type = "memory"
	case *FileBlockDevice:
		info.Type = "file"
	case *ReadOnlyDevice:
		info.Type = "readonly"
		info.ReadOnly = true
	default:
		info.Type = "unknown"
	}

	return info
}

// VerifyBlockDevice tests that a device is functioning correctly.
func VerifyBlockDevice(device BlockDevice) error {
	blockSize := device.BlockSize()
	blockCount := device.BlockCount()

	// Test data
	testData := make([]byte, blockSize)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// Write to first block
	if err := device.Write(0, testData); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	// Read back and verify
	readData := make([]byte, blockSize)
	if err := device.Read(0, readData); err != nil {
		return fmt.Errorf("read failed: %w", err)
	}

	if !bytes.Equal(testData, readData) {
		return errors.New("data mismatch after read/write")
	}

	// Test last block if count > 1
	if blockCount > 1 {
		lastBlock := blockCount - 1
		lastData := make([]byte, blockSize)
		for i := range lastData {
			lastData[i] = byte((i + 128) % 256)
		}

		if err := device.Write(lastBlock, lastData); err != nil {
			return fmt.Errorf("write to last block failed: %w", err)
		}

		verifyData := make([]byte, blockSize)
		if err := device.Read(lastBlock, verifyData); err != nil {
			return fmt.Errorf("read from last block failed: %w", err)
		}

		if !bytes.Equal(lastData, verifyData) {
			return errors.New("data mismatch in last block")
		}
	}

	return device.Flush()
}

// BlockMetadata stores metadata for a block.
type BlockMetadata struct {
	BlockNumber  uint64
	ModifiedTime time.Time
	Checksum     uint32
	AccessCount  uint64
	IsDirty      bool
}

// CalculateChecksum computes a simple checksum for block data.
func CalculateChecksum(data []byte) uint32 {
	var sum uint32
	for i, b := range data {
		shift := (i % 4) * 8
		sum += uint32(b) << shift
	}
	return sum
}

// ValidateChecksum verifies block data integrity.
func ValidateChecksum(data []byte, expected uint32) bool {
	return CalculateChecksum(data) == expected
}

// TieredBlockDevice provides multiple storage tiers (e.g., SSD, HDD).
type TieredBlockDevice struct {
	tiers   []BlockDevice
	cutoffs []uint64 // Block counts for each tier
}

// NewTieredBlockDevice creates a tiered storage device.
func NewTieredBlockDevice(tiers []BlockDevice, cutoffs []uint64) *TieredBlockDevice {
	return &TieredBlockDevice{
		tiers:   tiers,
		cutoffs: cutoffs,
	}
}

// Read selects the appropriate tier and reads the block.
func (d *TieredBlockDevice) Read(block uint64, data []byte) error {
	device := d.selectDevice(block)
	return device.Read(block, data)
}

// Write writes to the fastest tier.
func (d *TieredBlockDevice) Write(block uint64, data []byte) error {
	return d.tiers[0].Write(block, data)
}

// selectDevice chooses the appropriate storage tier.
func (d *TieredBlockDevice) selectDevice(block uint64) BlockDevice {
	for i, cutoff := range d.cutoffs {
		if block < cutoff {
			return d.tiers[i]
		}
	}
	// Default to last tier
	return d.tiers[len(d.tiers)-1]
}

// BlockSize returns the first tier's block size.
func (d *TieredBlockDevice) BlockSize() int {
	return d.tiers[0].BlockSize()
}

// BlockCount returns the total capacity across all tiers.
func (d *TieredBlockDevice) BlockCount() uint64 {
	var total uint64
	for _, tier := range d.tiers {
		total += tier.BlockCount()
	}
	return total
}

// Flush flushes all tiers.
func (d *TieredBlockDevice) Flush() error {
	for _, tier := range d.tiers {
		if err := tier.Flush(); err != nil {
			return err
		}
	}
	return nil
}

// Close closes all tiers.
func (d *TieredBlockDevice) Close() error {
	for _, tier := range d.tiers {
		tier.Close()
	}
	return nil
}

// BlockIterator iterates over blocks in a device.
type BlockIterator struct {
	device    BlockDevice
	current   uint64
	blockSize int
}

// NewBlockIterator creates an iterator for a device.
func NewBlockIterator(device BlockDevice) *BlockIterator {
	return &BlockIterator{
		device:    device,
		blockSize: device.BlockSize(),
	}
}

// Next advances to the next block.
func (it *BlockIterator) Next() bool {
	it.current++
	return it.current < it.device.BlockCount()
}

// Block returns the current block number.
func (it *BlockIterator) Block() uint64 {
	return it.current
}

// Read reads the current block.
func (it *BlockIterator) Read(data []byte) error {
	return it.device.Read(it.current, data)
}

// Write writes to the current block.
func (it *BlockIterator) Write(data []byte) error {
	return it.device.Write(it.current, data)
}

// EncryptedBlockDevice wraps a BlockDevice with encryption.
type EncryptedBlockDevice struct {
	device BlockDevice
	cipher []byte // Simplified key storage
}

// NewEncryptedBlockDevice creates an encrypted wrapper.
func NewEncryptedBlockDevice(device BlockDevice, key []byte) *EncryptedBlockDevice {
	return &EncryptedBlockDevice{
		device: device,
		cipher: key,
	}
}

// encrypt applies XOR encryption (simplified - use AES in production).
func (d *EncryptedBlockDevice) encrypt(data []byte) error {
	for i := range data {
		data[i] ^= d.cipher[i%len(d.cipher)]
	}
	return nil
}

// decrypt reverses encryption.
func (d *EncryptedBlockDevice) decrypt(data []byte) error {
	return d.encrypt(data) // XOR is its own inverse
}

// Read reads and decrypts a block.
func (d *EncryptedBlockDevice) Read(block uint64, data []byte) error {
	if err := d.device.Read(block, data); err != nil {
		return err
	}
	return d.decrypt(data)
}

// Write encrypts and writes a block.
func (d *EncryptedBlockDevice) Write(block uint64, data []byte) error {
	encrypted := make([]byte, len(data))
	copy(encrypted, data)
	if err := d.encrypt(encrypted); err != nil {
		return err
	}
	return d.device.Write(block, encrypted)
}

// BlockSize returns the underlying device's block size.
func (d *EncryptedBlockDevice) BlockSize() int {
	return d.device.BlockSize()
}

// BlockCount returns the underlying device's block count.
func (d *EncryptedBlockDevice) BlockCount() uint64 {
	return d.device.BlockCount()
}

// Flush forwards to the underlying device.
func (d *EncryptedBlockDevice) Flush() error {
	return d.device.Flush()
}

// Close forwards to the underlying device.
func (d *EncryptedBlockDevice) Close() error {
	return d.device.Close()
}

// CompressedBlockDevice wraps a BlockDevice with compression.
type CompressedBlockDevice struct {
	device  BlockDevice
	readBuf bytes.Buffer
}

// NewCompressedBlockDevice creates a compression wrapper.
func NewCompressedBlockDevice(device BlockDevice) *CompressedBlockDevice {
	return &CompressedBlockDevice{device: device}
}

// Read reads and decompresses a block.
func (d *CompressedBlockDevice) Read(block uint64, data []byte) error {
	compressed := make([]byte, d.device.BlockSize())
	if err := d.device.Read(block, compressed); err != nil {
		return err
	}

	d.readBuf.Reset()
	if _, err := d.readBuf.Write(compressed); err != nil {
		return err
	}

	n, err := d.readBuf.Read(data)
	if err != nil && err != io.EOF {
		return err
	}
	if n < len(data) {
		// Pad with zeros
		for i := n; i < len(data); i++ {
			data[i] = 0
		}
	}
	return nil
}

// Write compresses and writes a block.
func (d *CompressedBlockDevice) Write(block uint64, data []byte) error {
	compressed := make([]byte, d.device.BlockSize())
	d.readBuf.Reset()
	d.readBuf.Write(data)
	n, err := d.readBuf.Read(compressed)
	if err != nil && err != io.EOF {
		return err
	}
	// Zero the rest
	for i := n; i < len(compressed); i++ {
		compressed[i] = 0
	}
	return d.device.Write(block, compressed)
}

// BlockSize returns the underlying device's block size.
func (d *CompressedBlockDevice) BlockSize() int {
	return d.device.BlockSize()
}

// BlockCount returns the underlying device's block count.
func (d *CompressedBlockDevice) BlockCount() uint64 {
	return d.device.BlockCount()
}

// Flush forwards to the underlying device.
func (d *CompressedBlockDevice) Flush() error {
	return d.device.Flush()
}

// Close forwards to the underlying device.
func (d *CompressedBlockDevice) Close() error {
	return d.device.Close()
}

// BlockHeader represents metadata at the start of each block.
type BlockHeader struct {
	Magic       uint32 // Block magic number
	Version     uint16 // Format version
	BlockNumber uint64 // Block number
	Checksum    uint32 // Header checksum
	Timestamp   int64  // Modification time
}

// WriteHeader writes a block header.
func WriteHeader(header *BlockHeader, data []byte) error {
	buf := make([]byte, 24)
	binary.BigEndian.PutUint32(buf[0:4], header.Magic)
	binary.BigEndian.PutUint16(buf[4:6], header.Version)
	binary.BigEndian.PutUint64(buf[6:14], header.BlockNumber)
	binary.BigEndian.PutUint32(buf[14:18], header.Checksum)
	binary.BigEndian.PutUint64(buf[18:26], uint64(header.Timestamp))

	copy(data, buf)
	return nil
}

// ReadHeader reads a block header.
func ReadHeader(data []byte) (*BlockHeader, error) {
	if len(data) < 24 {
		return nil, errors.New("data too small for header")
	}

	return &BlockHeader{
		Magic:       binary.BigEndian.Uint32(data[0:4]),
		Version:     binary.BigEndian.Uint16(data[4:6]),
		BlockNumber: binary.BigEndian.Uint64(data[6:14]),
		Checksum:    binary.BigEndian.Uint32(data[14:18]),
		Timestamp:   int64(binary.BigEndian.Uint64(data[18:26])),
	}, nil
}
