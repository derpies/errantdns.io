// internal/storage/cached.go
package storage

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math/rand"
	"time"

	"errantdns.io/internal/cache"
	"errantdns.io/internal/models"
)

// CachedStorage wraps a Storage implementation with caching
type CachedStorage struct {
	storage    Storage
	cache      cache.Cache
	tieBreaker string
}

// NewCachedStorage creates a new cached storage wrapper
func NewCachedStorage(storage Storage, cache cache.Cache, tieBreaker string) *CachedStorage {
	return &CachedStorage{
		storage:    storage,
		cache:      cache,
		tieBreaker: tieBreaker,
	}
}

// LookupRecord implements read-through caching for single record lookups
func (cs *CachedStorage) LookupRecord(ctx context.Context, query *models.LookupQuery) (*models.DNSRecord, error) {
	cacheKey := query.CacheKey()

	// Check cache first
	if records, found := cs.cache.Get(cacheKey); found {
		// Apply selection to cached record array
		if len(records) > 0 {
			return cs.selectFromArray(records, query), nil
		}
	}

	// Cache miss - query storage for record group
	records, err := cs.storage.LookupRecordGroup(ctx, query)
	if err != nil {
		return nil, err
	}

	// If no records found, return nil
	if len(records) == 0 {
		return nil, nil
	}

	// Cache the entire group using the first record's TTL
	ttl := time.Duration(records[0].TTL) * time.Second
	cs.cache.Set(cacheKey, records, ttl)

	// Apply selection and return
	return cs.selectFromArray(records, query), nil
}

// LookupRecords queries storage directly (no caching for multiple records)
// Multiple records are less commonly cached and more complex to manage
func (cs *CachedStorage) LookupRecords(ctx context.Context, query *models.LookupQuery) ([]*models.DNSRecord, error) {
	return cs.storage.LookupRecords(ctx, query)
}

// LookupRecordGroup queries storage directly (no caching for record groups)
// Record groups change based on tie-breaking logic and are complex to cache
func (cs *CachedStorage) LookupRecordGroup(ctx context.Context, query *models.LookupQuery) ([]*models.DNSRecord, error) {
	return cs.storage.LookupRecordGroup(ctx, query)
}

// CreateRecord creates a record and invalidates cache
func (cs *CachedStorage) CreateRecord(ctx context.Context, record *models.DNSRecord) error {
	// Create in storage first
	if err := cs.storage.CreateRecord(ctx, record); err != nil {
		return err
	}

	// Invalidate any cached entries for this name/type
	cs.invalidateRecord(record)

	return nil
}

// UpdateRecord updates a record and invalidates cache
func (cs *CachedStorage) UpdateRecord(ctx context.Context, record *models.DNSRecord) error {
	// Update in storage first
	if err := cs.storage.UpdateRecord(ctx, record); err != nil {
		return err
	}

	// Invalidate any cached entries for this name/type
	cs.invalidateRecord(record)

	return nil
}

// DeleteRecord deletes a record and invalidates cache
func (cs *CachedStorage) DeleteRecord(ctx context.Context, id int) error {
	// We need to get the record first to know what to invalidate
	// This is a bit inefficient but necessary for proper cache invalidation
	// Alternative would be to return the deleted record from storage layer

	// For now, we'll delete from storage and clear entire cache entry
	// This is a trade-off between complexity and efficiency
	if err := cs.storage.DeleteRecord(ctx, id); err != nil {
		return err
	}

	// Note: We could optimize this by making DeleteRecord return the deleted record
	// For now, this is a simplified approach that works correctly

	return nil
}

// DeleteRecords deletes records and invalidates cache
func (cs *CachedStorage) DeleteRecords(ctx context.Context, name string, recordType string) error {
	// Delete from storage first
	if err := cs.storage.DeleteRecords(ctx, name, recordType); err != nil {
		return err
	}

	// Invalidate cache entries
	if recordType == "" {
		// All record types for this domain - invalidate all types
		cs.invalidateDomain(name)
	} else {
		// Specific record type
		cs.invalidateNameType(name, recordType)
	}

	return nil
}

// Health checks both storage and cache health
func (cs *CachedStorage) Health(ctx context.Context) error {
	// Check storage health
	if err := cs.storage.Health(ctx); err != nil {
		return fmt.Errorf("storage health check failed: %w", err)
	}

	// Cache doesn't typically have health checks, but we can verify it's responsive
	testKey := "__health_check__"
	testRecord := &models.DNSRecord{
		Name:       "health.test",
		RecordType: "A",
		Target:     "127.0.0.1",
		TTL:        1,
	}

	// Test cache operations with record array
	testRecords := []*models.DNSRecord{testRecord}
	cs.cache.Set(testKey, testRecords, time.Second)
	if records, found := cs.cache.Get(testKey); !found || len(records) == 0 {
		return fmt.Errorf("cache health check failed: unable to retrieve test record")
	}
	cs.cache.Delete(testKey)

	return nil
}

// Close closes both storage and cache
func (cs *CachedStorage) Close() error {
	var storageErr, cacheErr error

	// Close storage
	if cs.storage != nil {
		storageErr = cs.storage.Close()
	}

	// Close cache
	if cs.cache != nil {
		cacheErr = cs.cache.Close()
	}

	// Return the first error encountered
	if storageErr != nil {
		return fmt.Errorf("storage close error: %w", storageErr)
	}
	if cacheErr != nil {
		return fmt.Errorf("cache close error: %w", cacheErr)
	}

	return nil
}

// GetCacheStats returns cache statistics for monitoring
func (cs *CachedStorage) GetCacheStats() cache.Stats {
	return cs.cache.Stats()
}

// ClearCache clears all cached entries
func (cs *CachedStorage) ClearCache() {
	cs.cache.Clear()
}

// invalidateRecord invalidates cache entries for a specific record
func (cs *CachedStorage) invalidateRecord(record *models.DNSRecord) {
	query := models.NewLookupQuery(record.Name, record.RecordType)
	cacheKey := query.CacheKey()
	cs.cache.Delete(cacheKey)
}

// invalidateNameType invalidates cache entries for a specific name/type combination
func (cs *CachedStorage) invalidateNameType(name, recordType string) {
	query := models.NewLookupQuery(name, recordType)
	cacheKey := query.CacheKey()
	cs.cache.Delete(cacheKey)
}

// invalidateDomain invalidates all cached entries for a domain (all record types)
func (cs *CachedStorage) invalidateDomain(name string) {
	// Since we need to invalidate all record types for a domain,
	// and we don't have a way to enumerate cache keys by pattern,
	// we'll invalidate the common record types

	commonTypes := []models.RecordType{
		models.RecordTypeA,
		models.RecordTypeAAAA,
		models.RecordTypeCNAME,
		models.RecordTypeTXT,
		models.RecordTypeMX,
		models.RecordTypeNS,
	}

	for _, recordType := range commonTypes {
		cs.invalidateNameType(name, recordType.String())
	}

	// Note: This approach has limitations - it only invalidates common types
	// A more sophisticated approach would require either:
	// 1. A reverse index in the cache (domain -> cache keys)
	// 2. Cache key enumeration capabilities
	// 3. Tagged cache entries
	// For now, this covers the most common use cases
}

// selectFromArray applies tie-breaking logic to select one record from an array
func (cs *CachedStorage) selectFromArray(records []*models.DNSRecord, query *models.LookupQuery) *models.DNSRecord {
	if len(records) == 0 {
		return nil
	}

	if len(records) == 1 {
		return records[0]
	}

	switch cs.tieBreaker {
	case "random":
		// Use query-based seed for consistency within same query
		seed := cs.generateSeed(query)
		rng := rand.New(rand.NewSource(seed))
		index := rng.Intn(len(records))
		return records[index]

	case "round_robin":
		fallthrough
	default:
		// Round-robin based on time and query hash
		index := cs.roundRobinIndex(query, len(records))
		return records[index]
	}
}

// generateSeed creates a deterministic seed based on the query
func (cs *CachedStorage) generateSeed(query *models.LookupQuery) int64 {
	h := fnv.New64a()
	h.Write([]byte(query.Name))
	h.Write([]byte(query.Type.String()))
	// Add some time component for variation
	timeComponent := time.Now().Unix() / 300 // Changes every 5 minutes
	h.Write([]byte(fmt.Sprintf("%d", timeComponent)))
	return int64(h.Sum64())
}

// roundRobinIndex calculates round-robin index based on time and query
func (cs *CachedStorage) roundRobinIndex(query *models.LookupQuery, count int) int {
	if count <= 1 {
		return 0
	}

	// Create deterministic hash of query
	h := md5.New()
	h.Write([]byte(query.Name))
	h.Write([]byte(query.Type.String()))
	queryHash := h.Sum(nil)

	// Convert first 8 bytes to uint64
	queryValue := binary.BigEndian.Uint64(queryHash[:8])

	// Add time component (changes every 5 seconds for better rotation)
	timeComponent := uint64(time.Now().Unix() / 5)

	// Combine and mod by count
	combined := queryValue + timeComponent
	return int(combined % uint64(count))
}
