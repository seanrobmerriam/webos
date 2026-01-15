package dns

import (
	"sync"
	"time"
)

// CacheEntry represents a cached DNS resource record
type CacheEntry struct {
	Answers    []ResourceRecord
	Expiration time.Time
	Created    time.Time
}

// Valid returns true if the cache entry is still valid
func (e *CacheEntry) Valid() bool {
	return time.Now().Before(e.Expiration)
}

// Cache provides TTL-based DNS caching
type Cache struct {
	entries    map[string]CacheEntry
	mu         sync.RWMutex
	maxTTL     time.Duration
	defaultTTL time.Duration
}

// NewCache creates a new DNS cache
func NewCache() *Cache {
	return NewCacheWithTTL(DefaultCacheTTL)
}

// NewCacheWithTTL creates a new DNS cache with a custom maximum TTL
func NewCacheWithTTL(maxTTL time.Duration) *Cache {
	return &Cache{
		entries:    make(map[string]CacheEntry),
		maxTTL:     maxTTL,
		defaultTTL: maxTTL / 2,
	}
}

// SetMaxTTL sets the maximum TTL for cache entries
func (c *Cache) SetMaxTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxTTL = ttl
}

// Get retrieves a cache entry for the given name and type
func (c *Cache) Get(name string, qtype RecordType) ([]ResourceRecord, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := cacheKey(name, qtype)
	entry, ok := c.entries[key]
	if !ok || !entry.Valid() {
		return nil, false
	}

	// Filter out expired records
	validAnswers := make([]ResourceRecord, 0, len(entry.Answers))
	for _, rr := range entry.Answers {
		if time.Now().Before(rr.Expiration) {
			validAnswers = append(validAnswers, rr)
		}
	}

	if len(validAnswers) == 0 {
		return nil, false
	}

	return validAnswers, true
}

// Set stores a DNS response in the cache
func (c *Cache) Set(name string, qtype RecordType, answers []ResourceRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(answers) == 0 {
		return
	}

	// Calculate the minimum expiration time among all answers
	minExp := time.Now().Add(c.maxTTL)
	for _, rr := range answers {
		if rr.Expiration.Before(minExp) {
			minExp = rr.Expiration
		}
	}

	// Cap at max TTL
	if minExp.Sub(time.Now()) > c.maxTTL {
		minExp = time.Now().Add(c.maxTTL)
	}

	key := cacheKey(name, qtype)
	c.entries[key] = CacheEntry{
		Answers:    answers,
		Expiration: minExp,
		Created:    time.Now(),
	}
}

// Remove removes a cache entry
func (c *Cache) Remove(name string, qtype RecordType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, cacheKey(name, qtype))
}

// Clear removes all cache entries
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]CacheEntry)
}

// Len returns the number of cache entries
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := 0
	for _, entry := range c.entries {
		if entry.Valid() {
			count++
		}
	}
	return count
}

// Cleanup removes expired entries from the cache
func (c *Cache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if !entry.Valid() {
			delete(c.entries, key)
		}
		// Also check individual answer expirations
		if now.After(entry.Expiration) {
			delete(c.entries, key)
		}
	}
}

// Stats returns cache statistics
func (c *Cache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := CacheStats{
		Entries: make(map[string]CacheEntryStats),
	}

	for key, entry := range c.entries {
		if entry.Valid() {
			stats.Total++
			stats.Entries[key] = CacheEntryStats{
				AnswerCount: len(entry.Answers),
				TTL:         time.Until(entry.Expiration),
				Created:     entry.Created,
			}
			for _, rr := range entry.Answers {
				switch rr.Type {
				case RecordTypeA:
					stats.A++
				case RecordTypeAAAA:
					stats.AAAA++
				case RecordTypeCNAME:
					stats.CNAME++
				case RecordTypeMX:
					stats.MX++
				case RecordTypeTXT:
					stats.TXT++
				case RecordTypeNS:
					stats.NS++
				}
			}
		}
	}

	return stats
}

// CacheStats contains statistics about the DNS cache
type CacheStats struct {
	Total   int
	A       int
	AAAA    int
	CNAME   int
	MX      int
	TXT     int
	NS      int
	Entries map[string]CacheEntryStats
}

// CacheEntryStats contains statistics about a single cache entry
type CacheEntryStats struct {
	AnswerCount int
	TTL         time.Duration
	Created     time.Time
}

// cacheKey generates a cache key for a name and type
func cacheKey(name string, qtype RecordType) string {
	return name + "|" + qtype.String()
}
