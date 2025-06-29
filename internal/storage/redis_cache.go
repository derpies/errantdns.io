// internal/storage/redis_cache.go
package storage

import (
	"context"
	"fmt"
	"time"

	"errantdns.io/internal/cache"
	"errantdns.io/internal/models"
	"errantdns.io/internal/redis"
)

// RedisCacheStorage wraps existing cached storage with Redis as L2 cache
type RedisCacheStorage struct {
	storage     Storage
	memoryCache cache.Cache
	redisClient string
	keyPrefix   string
	tieBreaker  string
}

// CacheStats represents comprehensive cache statistics for three-tier caching
type CacheStats struct {
	L1Stats     cache.Stats `json:"l1_memory"`
	L2Stats     RedisStats  `json:"l2_redis"`
	TotalLayers int         `json:"total_layers"`
}

// RedisStats represents Redis-specific cache statistics
type RedisStats struct {
	Connected bool `json:"connected"`
	KeyCount  int  `json:"key_count"`
}

// NewRedisCacheStorage creates a new Redis-backed cache storage
func NewRedisCacheStorage(storage Storage, memoryCache cache.Cache, redisClientName, keyPrefix, tieBreaker string) *RedisCacheStorage {
	return &RedisCacheStorage{
		storage:     storage,
		memoryCache: memoryCache,
		redisClient: redisClientName,
		keyPrefix:   keyPrefix,
		tieBreaker:  tieBreaker,
	}
}

// GetCacheStats returns comprehensive cache statistics for both tiers
func (rcs *RedisCacheStorage) GetCacheStats() CacheStats {
	memStats := rcs.memoryCache.Stats()

	redisStats := RedisStats{
		Connected: redis.PingClient(rcs.redisClient) == nil,
		KeyCount:  rcs.getRedisKeyCount(),
	}

	return CacheStats{
		L1Stats:     memStats,
		L2Stats:     redisStats,
		TotalLayers: 2,
	}
}

// ClearCache clears both memory and Redis cache layers
func (rcs *RedisCacheStorage) ClearCache() {
	// Clear L1 (memory cache)
	rcs.memoryCache.Clear()

	// Clear L2 (Redis cache) - only our keys
	rcs.clearRedisCache()
}

// getRedisKeyCount counts keys with our prefix in Redis
func (rcs *RedisCacheStorage) getRedisKeyCount() int {
	pattern := rcs.keyPrefix + "*"
	keys, err := redis.ScanFrom(rcs.redisClient, pattern)
	if err != nil {
		return -1 // Indicate error
	}
	return len(keys)
}

// clearRedisCache removes all keys with our prefix from Redis
func (rcs *RedisCacheStorage) clearRedisCache() {
	pattern := rcs.keyPrefix + "*"
	keys, err := redis.ScanFrom(rcs.redisClient, pattern)
	if err != nil {
		return
	}

	if len(keys) > 0 {
		redis.DeleteOn(rcs.redisClient, keys...)
	}
}

// LookupRecordWithSource implements three-tier caching with source tracking
func (rcs *RedisCacheStorage) LookupRecordWithSource(ctx context.Context, query *models.LookupQuery) (*LookupResult, error) {
	cacheKey := rcs.getCacheKey(query)

	// L1: Check memory cache first
	if records, found := rcs.memoryCache.Get(cacheKey); found && len(records) > 0 {
		return &LookupResult{
			Record: rcs.selectFromArray(records, query),
			Source: SourceMemory,
		}, nil
	}

	// L2: Check Redis cache
	var records []*models.DNSRecord
	if err := redis.GetJSONFrom(rcs.redisClient, cacheKey, &records); err == nil && len(records) > 0 {
		// Cache hit in Redis - populate memory cache
		ttl := time.Duration(records[0].TTL/10) * time.Second
		rcs.memoryCache.Set(cacheKey, records, ttl)
		return &LookupResult{
			Record: rcs.selectFromArray(records, query),
			Source: SourceRedis,
		}, nil
	}

	// L3: Cache miss - query storage
	records, err := rcs.storage.LookupRecordGroup(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, nil
	}

	// Populate both cache layers
	l1TTL := time.Duration(records[0].TTL/10) * time.Second
	l2TTL := time.Duration(records[0].TTL/2) * time.Second

	rcs.memoryCache.Set(cacheKey, records, l1TTL)
	redis.SetJSONOn(rcs.redisClient, cacheKey, records)
	redis.ExpireOn(rcs.redisClient, cacheKey, int(l2TTL.Seconds()))

	return &LookupResult{
		Record: rcs.selectFromArray(records, query),
		Source: SourceDatabase,
	}, nil
}

// LookupRecordGroupWithSource implements three-tier caching with source tracking for groups
func (rcs *RedisCacheStorage) LookupRecordGroupWithSource(ctx context.Context, query *models.LookupQuery) (*LookupGroupResult, error) {
	cacheKey := rcs.getCacheKey(query)

	// L1: Check memory cache first
	if records, found := rcs.memoryCache.Get(cacheKey); found && len(records) > 0 {
		return &LookupGroupResult{
			Records: records,
			Source:  SourceMemory,
		}, nil
	}

	// L2: Check Redis cache
	var records []*models.DNSRecord
	if err := redis.GetJSONFrom(rcs.redisClient, cacheKey, &records); err == nil && len(records) > 0 {
		// Cache hit in Redis - populate memory cache
		ttl := time.Duration(records[0].TTL/10) * time.Second
		rcs.memoryCache.Set(cacheKey, records, ttl)
		return &LookupGroupResult{
			Records: records,
			Source:  SourceRedis,
		}, nil
	}

	// L3: Cache miss - query storage
	records, err := rcs.storage.LookupRecordGroup(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, nil
	}

	// Populate both cache layers
	l1TTL := time.Duration(records[0].TTL/10) * time.Second
	l2TTL := time.Duration(records[0].TTL/2) * time.Second

	rcs.memoryCache.Set(cacheKey, records, l1TTL)
	redis.SetJSONOn(rcs.redisClient, cacheKey, records)
	redis.ExpireOn(rcs.redisClient, cacheKey, int(l2TTL.Seconds()))

	return &LookupGroupResult{
		Records: records,
		Source:  SourceDatabase,
	}, nil
}

// LookupRecord implements three-tier caching: Memory -> Redis -> Storage
func (rcs *RedisCacheStorage) LookupRecord(ctx context.Context, query *models.LookupQuery) (*models.DNSRecord, error) {
	cacheKey := rcs.getCacheKey(query)

	// L1: Check memory cache first
	if records, found := rcs.memoryCache.Get(cacheKey); found && len(records) > 0 {
		return rcs.selectFromArray(records, query), nil
	}

	// L2: Check Redis cache
	var records []*models.DNSRecord
	if err := redis.GetJSONFrom(rcs.redisClient, cacheKey, &records); err == nil && len(records) > 0 {
		// Cache hit in Redis - populate memory cache
		ttl := time.Duration(records[0].TTL/10) * time.Second // 10% of record TTL for L1
		rcs.memoryCache.Set(cacheKey, records, ttl)
		return rcs.selectFromArray(records, query), nil
	}

	// L3: Cache miss - query storage
	records, err := rcs.storage.LookupRecordGroup(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, nil
	}

	// Populate both cache layers
	l1TTL := time.Duration(records[0].TTL/10) * time.Second // 10% for L1
	l2TTL := time.Duration(records[0].TTL/2) * time.Second  // 50% for L2

	rcs.memoryCache.Set(cacheKey, records, l1TTL)
	redis.SetJSONOn(rcs.redisClient, cacheKey, records) // Use JSON for complex objects
	redis.ExpireOn(rcs.redisClient, cacheKey, int(l2TTL.Seconds()))

	return rcs.selectFromArray(records, query), nil
}

// LookupRecords queries storage directly (no caching for multiple records)
func (rcs *RedisCacheStorage) LookupRecords(ctx context.Context, query *models.LookupQuery) ([]*models.DNSRecord, error) {
	return rcs.storage.LookupRecords(ctx, query)
}

// LookupRecordGroup queries with caching
func (rcs *RedisCacheStorage) LookupRecordGroup(ctx context.Context, query *models.LookupQuery) ([]*models.DNSRecord, error) {
	cacheKey := rcs.getCacheKey(query)

	// L1: Check memory cache first
	if records, found := rcs.memoryCache.Get(cacheKey); found && len(records) > 0 {
		return records, nil
	}

	// L2: Check Redis cache
	var records []*models.DNSRecord
	if err := redis.GetJSONFrom(rcs.redisClient, cacheKey, &records); err == nil && len(records) > 0 {
		// Cache hit in Redis - populate memory cache
		ttl := time.Duration(records[0].TTL/10) * time.Second
		rcs.memoryCache.Set(cacheKey, records, ttl)
		return records, nil
	}

	// L3: Cache miss - query storage
	records, err := rcs.storage.LookupRecordGroup(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, nil
	}

	// Populate both cache layers
	l1TTL := time.Duration(records[0].TTL/10) * time.Second
	l2TTL := time.Duration(records[0].TTL/2) * time.Second

	rcs.memoryCache.Set(cacheKey, records, l1TTL)
	redis.SetJSONOn(rcs.redisClient, cacheKey, records)
	redis.ExpireOn(rcs.redisClient, cacheKey, int(l2TTL.Seconds()))

	return records, nil
}

// CreateRecord creates a record and invalidates cache
func (rcs *RedisCacheStorage) CreateRecord(ctx context.Context, record *models.DNSRecord) error {
	if err := rcs.storage.CreateRecord(ctx, record); err != nil {
		return err
	}
	rcs.invalidateRecord(record)
	return nil
}

// UpdateRecord updates a record and invalidates cache
func (rcs *RedisCacheStorage) UpdateRecord(ctx context.Context, record *models.DNSRecord) error {
	if err := rcs.storage.UpdateRecord(ctx, record); err != nil {
		return err
	}
	rcs.invalidateRecord(record)
	return nil
}

// DeleteRecord deletes a record and invalidates cache
func (rcs *RedisCacheStorage) DeleteRecord(ctx context.Context, id int) error {
	return rcs.storage.DeleteRecord(ctx, id)
}

// DeleteRecords deletes records and invalidates cache
func (rcs *RedisCacheStorage) DeleteRecords(ctx context.Context, name string, recordType string) error {
	if err := rcs.storage.DeleteRecords(ctx, name, recordType); err != nil {
		return err
	}
	if recordType == "" {
		rcs.invalidateDomain(name)
	} else {
		rcs.invalidateNameType(name, recordType)
	}
	return nil
}

// Health checks storage, memory cache, and Redis
func (rcs *RedisCacheStorage) Health(ctx context.Context) error {
	if err := rcs.storage.Health(ctx); err != nil {
		return fmt.Errorf("storage health check failed: %w", err)
	}

	if err := redis.PingClient(rcs.redisClient); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	return nil
}

// Close closes storage and cache
func (rcs *RedisCacheStorage) Close() error {
	if rcs.storage != nil {
		return rcs.storage.Close()
	}
	return nil
}

// Helper methods
func (rcs *RedisCacheStorage) getCacheKey(query *models.LookupQuery) string {
	return rcs.keyPrefix + query.CacheKey()
}

func (rcs *RedisCacheStorage) invalidateRecord(record *models.DNSRecord) {
	query := models.NewLookupQuery(record.Name, record.RecordType)
	cacheKey := rcs.getCacheKey(query)
	rcs.memoryCache.Delete(cacheKey)
	redis.DeleteOn(rcs.redisClient, cacheKey)
}

func (rcs *RedisCacheStorage) invalidateNameType(name, recordType string) {
	query := models.NewLookupQuery(name, recordType)
	cacheKey := rcs.getCacheKey(query)
	rcs.memoryCache.Delete(cacheKey)
	redis.DeleteOn(rcs.redisClient, cacheKey)
}

func (rcs *RedisCacheStorage) invalidateDomain(name string) {
	commonTypes := []models.RecordType{
		models.RecordTypeA, models.RecordTypeAAAA, models.RecordTypeCNAME,
		models.RecordTypeTXT, models.RecordTypeMX, models.RecordTypeNS,
		models.RecordTypeSOA, models.RecordTypePTR, models.RecordTypeSRV, models.RecordTypeCAA,
	}

	for _, recordType := range commonTypes {
		rcs.invalidateNameType(name, recordType.String())
	}
}

func (rcs *RedisCacheStorage) selectFromArray(records []*models.DNSRecord, query *models.LookupQuery) *models.DNSRecord {
	if len(records) == 0 {
		return nil
	}
	if len(records) == 1 {
		return records[0]
	}

	// Simple round-robin for now
	// TODO: Use the same tie-breaking logic as the original cached storage
	return records[0]
}
