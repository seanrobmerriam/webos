/*
Package storage provides block storage abstraction with caching,
write-ahead logging, RAID support, and snapshot management.

This package implements a complete block storage system for persistent
data management in the WebOS project.
*/
package storage

import (
	"container/list"
	"errors"
	"fmt"
	"sync"
	"time"
)

// CachePolicy defines the caching algorithm to use.
type CachePolicy string

const (
	// CachePolicyLRU evicts the least recently used blocks.
	CachePolicyLRU CachePolicy = "LRU"
	// CachePolicyLFU evicts the least frequently used blocks.
	CachePolicyLFU CachePolicy = "LFU"
	// CachePolicyFIFO evicts the oldest blocks first.
	CachePolicyFIFO CachePolicy = "FIFO"
)

// Common cache errors
var (
	ErrCacheMiss    = errors.New("cache miss")
	ErrCacheClosed  = errors.New("cache is closed")
	ErrInvalidBlock = errors.New("invalid block number")
)

// BlockCache provides caching for block device reads and writes.
type BlockCache struct {
	device    BlockDevice
	policy    CachePolicy
	maxSize   int // Maximum number of blocks in cache
	cache     map[uint64]*list.Element
	lruList   *list.List     // For LRU/FIFO
	lfuFreq   map[uint64]int // Frequency counts for LFU
	dirty     map[uint64]bool
	hitCount  uint64
	missCount uint64
	closed    bool
	mu        sync.RWMutex
	cond      *sync.Cond // For blocking cache operations
}

// CacheEntry represents a cached block.
type CacheEntry struct {
	BlockNumber uint64
	Data        []byte
	Dirty       bool
	LastAccess  time.Time
	AccessCount int
}

// NewBlockCache creates a new block cache.
func NewBlockCache(device BlockDevice, policy CachePolicy, maxSize int) *BlockCache {
	c := &BlockCache{
		device:    device,
		policy:    policy,
		maxSize:   maxSize,
		cache:     make(map[uint64]*list.Element),
		lruList:   list.New(),
		lfuFreq:   make(map[uint64]int),
		dirty:     make(map[uint64]bool),
		hitCount:  0,
		missCount: 0,
	}
	c.cond = sync.NewCond(&c.mu)
	return c
}

// Read retrieves a block from cache or underlying device.
func (c *BlockCache) Read(block uint64, data []byte) error {
	if block >= c.device.BlockCount() {
		return ErrInvalidBlock
	}
	if len(data) != c.device.BlockSize() {
		return fmt.Errorf("data length %d != block size %d", len(data), c.device.BlockSize())
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrCacheClosed
	}

	// Check if block is in cache
	if elem, ok := c.cache[block]; ok {
		entry := elem.Value.(*CacheEntry)
		copy(data, entry.Data)
		c.hitCount++

		// Update access metadata
		entry.LastAccess = time.Now()
		entry.AccessCount++
		c.lfuFreq[block]++

		// Move to front for LRU
		if c.policy == CachePolicyLRU {
			c.lruList.MoveToFront(elem)
		}

		return nil
	}

	// Cache miss - read from device
	c.missCount++
	if err := c.device.Read(block, data); err != nil {
		return err
	}

	// Cache the data
	c.cacheBlock(block, data, false)

	return nil
}

// Write writes a block to cache (marking it dirty).
func (c *BlockCache) Write(block uint64, data []byte) error {
	if block >= c.device.BlockCount() {
		return ErrInvalidBlock
	}
	if len(data) != c.device.BlockSize() {
		return ErrBlockTooLarge
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrCacheClosed
	}

	// Update or add to cache
	if elem, ok := c.cache[block]; ok {
		entry := elem.Value.(*CacheEntry)
		copy(entry.Data, data)
		entry.Dirty = true
		entry.LastAccess = time.Now()
		entry.AccessCount++
		c.dirty[block] = true
		c.lfuFreq[block]++

		if c.policy == CachePolicyLRU {
			c.lruList.MoveToFront(elem)
		}
	} else {
		c.cacheBlock(block, data, true)
	}

	return nil
}

// cacheBlock adds a block to the cache, evicting if necessary.
func (c *BlockCache) cacheBlock(block uint64, data []byte, dirty bool) {
	// Check if cache is full
	if len(c.cache) >= c.maxSize {
		c.evict()
		// If still full after eviction, force eviction of oldest
		if len(c.cache) >= c.maxSize {
			c.evictOldest()
		}
	}

	entry := &CacheEntry{
		BlockNumber: block,
		Data:        make([]byte, len(data)),
		Dirty:       dirty,
		LastAccess:  time.Now(),
		AccessCount: 1,
	}
	copy(entry.Data, data)

	elem := c.lruList.PushFront(entry)
	c.cache[block] = elem
	c.lfuFreq[block] = 1
	if dirty {
		c.dirty[block] = true
	}
}

// evict removes blocks according to the cache policy.
func (c *BlockCache) evict() {
	if len(c.cache) < c.maxSize {
		return
	}

	switch c.policy {
	case CachePolicyLRU, CachePolicyFIFO:
		// Remove from back of list
		c.evictOldest()
	case CachePolicyLFU:
		// Remove least frequently used
		c.evictLFU()
	}
}

// evictOldest removes the oldest entry (back of LRU list).
func (c *BlockCache) evictOldest() {
	if c.lruList.Len() == 0 {
		return
	}

	elem := c.lruList.Back()
	if elem == nil {
		return
	}

	entry := elem.Value.(*CacheEntry)
	delete(c.cache, entry.BlockNumber)
	delete(c.lfuFreq, entry.BlockNumber)
	delete(c.dirty, entry.BlockNumber)
	c.lruList.Remove(elem)
}

// evictLFU removes the least frequently used entry.
func (c *BlockCache) evictLFU() {
	var minFreq int = -1
	var evictBlock uint64

	for block, freq := range c.lfuFreq {
		if minFreq == -1 || freq < minFreq {
			minFreq = freq
			evictBlock = block
		}
	}

	if elem, ok := c.cache[evictBlock]; ok {
		delete(c.cache, evictBlock)
		delete(c.lfuFreq, evictBlock)
		delete(c.dirty, evictBlock)
		c.lruList.Remove(elem)
	}
}

// Flush writes all dirty blocks to the underlying device.
func (c *BlockCache) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrCacheClosed
	}

	for block, isDirty := range c.dirty {
		if !isDirty {
			continue
		}

		if elem, ok := c.cache[block]; ok {
			entry := elem.Value.(*CacheEntry)
			if err := c.device.Write(block, entry.Data); err != nil {
				return fmt.Errorf("failed to flush block %d: %w", block, err)
			}
			entry.Dirty = false
		}
		c.dirty[block] = false
	}

	return c.device.Flush()
}

// Invalidate removes a block from the cache.
func (c *BlockCache) Invalidate(block uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrCacheClosed
	}

	return c.invalidateLocked(block)
}

// invalidateLocked removes a block from the cache (caller must hold lock).
func (c *BlockCache) invalidateLocked(block uint64) error {
	elem, ok := c.cache[block]
	if !ok {
		return nil
	}

	entry := elem.Value.(*CacheEntry)
	if entry.Dirty {
		// Write dirty data before invalidating
		if err := c.device.Write(block, entry.Data); err != nil {
			return err
		}
	}

	delete(c.cache, block)
	delete(c.lfuFreq, block)
	delete(c.dirty, block)
	c.lruList.Remove(elem)

	return nil
}

// InvalidateAll removes all blocks from the cache.
func (c *BlockCache) InvalidateAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrCacheClosed
	}

	// Flush all dirty blocks first
	for block, isDirty := range c.dirty {
		if isDirty {
			if elem, ok := c.cache[block]; ok {
				entry := elem.Value.(*CacheEntry)
				if err := c.device.Write(block, entry.Data); err != nil {
					return err
				}
			}
		}
	}

	c.cache = make(map[uint64]*list.Element)
	c.lruList = list.New()
	c.lfuFreq = make(map[uint64]int)
	c.dirty = make(map[uint64]bool)

	return nil
}

// Close closes the cache and flushes dirty blocks.
func (c *BlockCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	// Flush all dirty blocks
	for block, isDirty := range c.dirty {
		if isDirty {
			if elem, ok := c.cache[block]; ok {
				entry := elem.Value.(*CacheEntry)
				c.device.Write(block, entry.Data)
			}
		}
	}

	c.cond.Broadcast()
	return nil
}

// Stats returns cache statistics.
func (c *BlockCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hitCount + c.missCount
	var hitRate float64
	if total > 0 {
		hitRate = float64(c.hitCount) / float64(total) * 100
	}

	return CacheStats{
		Entries:     len(c.cache),
		MaxSize:     c.maxSize,
		DirtyBlocks: len(c.dirty),
		HitCount:    c.hitCount,
		MissCount:   c.missCount,
		HitRate:     hitRate,
	}
}

// CacheStats contains cache statistics.
type CacheStats struct {
	Entries     int
	MaxSize     int
	DirtyBlocks int
	HitCount    uint64
	MissCount   uint64
	HitRate     float64
}

// BlockSize returns the underlying device's block size.
func (c *BlockCache) BlockSize() int {
	return c.device.BlockSize()
}

// BlockCount returns the underlying device's block count.
func (c *BlockCache) BlockCount() uint64 {
	return c.device.BlockCount()
}

// Peek returns a block from cache without updating access metadata.
func (c *BlockCache) Peek(block uint64) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrCacheClosed
	}

	elem, ok := c.cache[block]
	if !ok {
		return nil, ErrCacheMiss
	}

	data := make([]byte, c.device.BlockSize())
	copy(data, elem.Value.(*CacheEntry).Data)
	return data, nil
}

// WriteBack delays writing dirty blocks until Flush is called.
func (c *BlockCache) WriteBack(block uint64, data []byte) error {
	return c.Write(block, data)
}

// WriteThrough writes through to the underlying device immediately.
func (c *BlockCache) WriteThrough(block uint64, data []byte) error {
	if err := c.Write(block, data); err != nil {
		return err
	}
	return c.flushBlock(block)
}

// flushBlock writes a single block to the underlying device.
func (c *BlockCache) flushBlock(block uint64) error {
	if elem, ok := c.cache[block]; ok {
		entry := elem.Value.(*CacheEntry)
		if entry.Dirty {
			if err := c.device.Write(block, entry.Data); err != nil {
				return err
			}
			entry.Dirty = false
			delete(c.dirty, block)
		}
	}
	return nil
}

// Prefetch reads blocks into cache ahead of time.
func (c *BlockCache) Prefetch(startBlock uint64, count int) error {
	if count <= 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrCacheClosed
	}

	blockSize := c.device.BlockSize()
	for i := 0; i < count; i++ {
		block := startBlock + uint64(i)
		if block >= c.device.BlockCount() {
			break
		}

		// Skip if already cached
		if _, ok := c.cache[block]; ok {
			continue
		}

		// Read from device
		data := make([]byte, blockSize)
		if err := c.device.Read(block, data); err != nil {
			continue // Skip failed reads
		}

		// Cache the data
		c.cacheBlock(block, data, false)
	}

	return nil
}

// SetMaxSize changes the maximum cache size.
func (c *BlockCache) SetMaxSize(maxSize int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.maxSize = maxSize

	// Evict if necessary
	for len(c.cache) > c.maxSize {
		c.evict()
	}
}

// IsDirty returns whether a block is dirty.
func (c *BlockCache) IsDirty(block uint64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.dirty[block]
}

// DirtyBlocks returns all dirty block numbers.
func (c *BlockCache) DirtyBlocks() []uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	blocks := make([]uint64, 0, len(c.dirty))
	for block := range c.dirty {
		blocks = append(blocks, block)
	}
	return blocks
}

// CacheStatsDetailed returns detailed cache statistics.
type CacheStatsDetailed struct {
	CacheStats
	Policy       CachePolicy
	DeviceType   string
	BlockSize    int
	TotalBlocks  uint64
	CachedBlocks int
	MemoryUsage  int64 // Approximate bytes used
}

// GetDetailedStats returns detailed statistics.
func (c *BlockCache) GetDetailedStats() CacheStatsDetailed {
	stats := c.Stats()

	c.mu.RLock()
	defer c.mu.RUnlock()

	memoryUsage := int64(len(c.cache) * c.device.BlockSize())

	return CacheStatsDetailed{
		CacheStats:   stats,
		Policy:       c.policy,
		DeviceType:   GetInfo(c.device).Type,
		BlockSize:    c.device.BlockSize(),
		TotalBlocks:  c.device.BlockCount(),
		CachedBlocks: len(c.cache),
		MemoryUsage:  memoryUsage,
	}
}

// WriteBuffer provides buffered write support.
type WriteBuffer struct {
	cache     *BlockCache
	buffers   map[uint64][]byte
	mu        sync.Mutex
	maxBlocks int
}

// NewWriteBuffer creates a new write buffer.
func NewWriteBuffer(cache *BlockCache, maxBlocks int) *WriteBuffer {
	return &WriteBuffer{
		cache:     cache,
		buffers:   make(map[uint64][]byte),
		maxBlocks: maxBlocks,
	}
}

// Write buffers a write operation.
func (wb *WriteBuffer) Write(block uint64, data []byte) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	if len(wb.buffers) >= wb.maxBlocks {
		if err := wb.flush(); err != nil {
			return err
		}
	}

	existing, ok := wb.buffers[block]
	if ok {
		copy(existing, data)
	} else {
		buf := make([]byte, len(data))
		copy(buf, data)
		wb.buffers[block] = buf
	}

	return nil
}

// Flush writes all buffered data to the cache.
func (wb *WriteBuffer) Flush() error {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	return wb.flush()
}

// flush is the internal flush (caller holds lock).
func (wb *WriteBuffer) flush() error {
	for block, data := range wb.buffers {
		if err := wb.cache.Write(block, data); err != nil {
			return err
		}
	}
	wb.buffers = make(map[uint64][]byte)
	return nil
}

// Close flushes and closes the buffer.
func (wb *WriteBuffer) Close() error {
	return wb.Flush()
}
