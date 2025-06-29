// internal/storage/source.go
package storage

import "errantdns.io/internal/models"

// CacheSource indicates where a DNS record was retrieved from
type CacheSource string

const (
	SourceDatabase CacheSource = "DB" // Retrieved from database (L3)
	SourceRedis    CacheSource = "L2" // Retrieved from Redis cache (L2)
	SourceMemory   CacheSource = "L1" // Retrieved from memory cache (L1)
)

// String returns a human-readable representation of the cache source
func (cs CacheSource) String() string {
	return string(cs)
}

// LookupResult represents a DNS lookup result with source information
type LookupResult struct {
	Record *models.DNSRecord
	Source CacheSource
}

// LookupGroupResult represents a group lookup result with source information
type LookupGroupResult struct {
	Records []*models.DNSRecord
	Source  CacheSource
}
