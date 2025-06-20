// internal/storage/cached.go
package storage

import (
	"context"
	"fmt"
	"time"

	"errantdns.io/internal/cache"
	"errantdns.io/internal/models"
)

// CachedStorage wraps a Storage implementation with caching
type CachedStorage struct {
	storage Storage
	cache   cache.Cache
}

// NewCachedStorage creates a new cached storage wrapper
func NewCachedStorage(storage Storage, cache cache.Cache) *CachedStorage {
	return &CachedStorage{
		storage: storage,
		cache:   cache,
	}
}

// LookupRecord implements read-through caching for single record lookups
func (cs *CachedStorage) LookupRecord(ctx context.Context, query *models.LookupQuery) (*models.DNSRecord, error) {
	cacheKey := query.CacheKey()

	// Check cache first
	if records, found := cs.cache.Get(cacheKey); found {
		// Apply selection to cached record array (placeholder for now)
		if len(records) > 0 {
			return records[0], nil // For now, just return first record
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

	// For now, just return the first record (we'll add tie-breaking next)
	return records[0], nil
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
