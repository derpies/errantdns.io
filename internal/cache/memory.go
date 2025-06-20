// internal/cache/memory.go
package cache

import (
	"sync"
	"time"

	"errantdns.io/internal/models"
)

// Cache interface defines the contract for DNS record caching
type Cache interface {
	// Basic operations
	Get(key string) ([]*models.DNSRecord, bool)
	Set(key string, records []*models.DNSRecord, ttl time.Duration)
	Delete(key string)
	Clear()

	// Management
	Size() int
	Stats() Stats
	Close() error
}

// Stats represents cache performance statistics
type Stats struct {
	Hits        int64     `json:"hits"`
	Misses      int64     `json:"misses"`
	Entries     int       `json:"entries"`
	Evictions   int64     `json:"evictions"`
	LastCleanup time.Time `json:"last_cleanup"`
	HitRate     float64   `json:"hit_rate"`
}

// calculateHitRate computes the cache hit rate as a percentage
func (s *Stats) calculateHitRate() {
	total := s.Hits + s.Misses
	if total == 0 {
		s.HitRate = 0.0
	} else {
		s.HitRate = float64(s.Hits) / float64(total) * 100.0
	}
}

type cacheEntry struct {
	records    []*models.DNSRecord // <- Should be records (plural)
	expiresAt  time.Time
	lastAccess time.Time
}

// isExpired checks if the cache entry has expired
func (e *cacheEntry) isExpired() bool {
	return time.Now().After(e.expiresAt)
}

// MemoryCache implements an in-memory LRU cache with TTL support
type MemoryCache struct {
	mu          sync.RWMutex
	data        map[string]*cacheEntry
	accessOrder []string
	maxEntries  int
	stats       Stats

	// Background cleanup
	cleanupInterval time.Duration
	cleanupTicker   *time.Ticker
	cleanupStop     chan struct{}
	cleanupDone     chan struct{}
}

// Config holds configuration for the memory cache
type Config struct {
	MaxEntries      int
	CleanupInterval time.Duration
}

// DefaultConfig returns a cache config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		MaxEntries:      10000,
		CleanupInterval: 60 * time.Second,
	}
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(config *Config) *MemoryCache {
	if config == nil {
		config = DefaultConfig()
	}

	cache := &MemoryCache{
		data:            make(map[string]*cacheEntry),
		accessOrder:     make([]string, 0, config.MaxEntries),
		maxEntries:      config.MaxEntries,
		cleanupInterval: config.CleanupInterval,
		cleanupStop:     make(chan struct{}),
		cleanupDone:     make(chan struct{}),
	}

	// Start background cleanup if interval is set
	if config.CleanupInterval > 0 {
		cache.startCleanup()
	}

	return cache
}

// Get retrieves records from the cache
func (c *MemoryCache) Get(key string) ([]*models.DNSRecord, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.data[key]
	if !exists {
		c.stats.Misses++
		return nil, false
	}

	// Check if expired
	if entry.isExpired() {
		c.deleteUnlocked(key)
		c.stats.Misses++
		return nil, false
	}

	// Update access time and move to front for LRU
	entry.lastAccess = time.Now()
	c.moveToFrontUnlocked(key)
	c.stats.Hits++

	return entry.records, true
}

// Set stores records in the cache with TTL
func (c *MemoryCache) Set(key string, records []*models.DNSRecord, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// If key already exists, update it
	if _, exists := c.data[key]; exists {
		c.data[key] = &cacheEntry{
			records:    records,
			expiresAt:  now.Add(ttl),
			lastAccess: now,
		}
		c.moveToFrontUnlocked(key)
		return
	}

	// Check if we need to evict entries
	for len(c.data) >= c.maxEntries {
		c.evictLRUUnlocked()
	}

	// Add new entry
	c.data[key] = &cacheEntry{
		records:    records,
		expiresAt:  now.Add(ttl),
		lastAccess: now,
	}

	// Add to front of access order
	c.accessOrder = append([]string{key}, c.accessOrder...)
}

// Delete removes a record from the cache
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deleteUnlocked(key)
}

// Clear removes all entries from the cache
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]*cacheEntry)
	c.accessOrder = c.accessOrder[:0]
	c.stats.Entries = 0
}

// Size returns the current number of entries in the cache
func (c *MemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// Stats returns current cache statistics
func (c *MemoryCache) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.stats
	stats.Entries = len(c.data)
	stats.calculateHitRate()
	return stats
}

// Close stops the background cleanup and releases resources
func (c *MemoryCache) Close() error {
	if c.cleanupTicker != nil {
		close(c.cleanupStop)
		<-c.cleanupDone // Wait for cleanup goroutine to finish
	}
	return nil
}

// startCleanup begins the background cleanup goroutine
func (c *MemoryCache) startCleanup() {
	c.cleanupTicker = time.NewTicker(c.cleanupInterval)
	go c.cleanupLoop()
}

// cleanupLoop runs the periodic cleanup of expired entries
func (c *MemoryCache) cleanupLoop() {
	defer close(c.cleanupDone)

	for {
		select {
		case <-c.cleanupTicker.C:
			c.cleanupExpired()
		case <-c.cleanupStop:
			c.cleanupTicker.Stop()
			return
		}
	}
}

// cleanupExpired removes expired entries from the cache
func (c *MemoryCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	// Find expired keys
	for key, entry := range c.data {
		if now.After(entry.expiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// Remove expired keys
	for _, key := range expiredKeys {
		c.deleteUnlocked(key)
	}

	c.stats.LastCleanup = now
}

// moveToFrontUnlocked moves a key to the front of the access order (most recently used)
// Must be called with mutex locked
func (c *MemoryCache) moveToFrontUnlocked(key string) {
	// Find and remove the key from its current position
	for i, k := range c.accessOrder {
		if k == key {
			// Remove from current position
			c.accessOrder = append(c.accessOrder[:i], c.accessOrder[i+1:]...)
			break
		}
	}

	// Add to front
	c.accessOrder = append([]string{key}, c.accessOrder...)
}

// evictLRUUnlocked removes the least recently used entry
// Must be called with mutex locked
func (c *MemoryCache) evictLRUUnlocked() {
	if len(c.accessOrder) == 0 {
		return
	}

	// Remove the last entry (least recently used)
	lruKey := c.accessOrder[len(c.accessOrder)-1]
	c.deleteUnlocked(lruKey)
	c.stats.Evictions++
}

// deleteUnlocked removes an entry from the cache
// Must be called with mutex locked
func (c *MemoryCache) deleteUnlocked(key string) {
	delete(c.data, key)

	// Remove from access order
	for i, k := range c.accessOrder {
		if k == key {
			c.accessOrder = append(c.accessOrder[:i], c.accessOrder[i+1:]...)
			break
		}
	}
}
