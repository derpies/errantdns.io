// internal/resolver/resolver.go
package resolver

import (
	"context"
	"strings"

	"errantdns.io/internal/models"
	"errantdns.io/internal/storage"
)

// Resolver handles DNS-specific resolution logic
type Resolver struct {
	storage storage.Storage
}

// Config holds configuration for the DNS resolver
type Config struct {
	// Add resolver-specific configuration here in the future
	// For example: cache settings, recursion limits, etc.
}

// ResolverResult represents a DNS resolution result with source information
type ResolverResult struct {
	Record *models.DNSRecord
	Source storage.CacheSource
}

// ResolverGroupResult represents a group resolution result with source information
type ResolverGroupResult struct {
	Records []*models.DNSRecord
	Source  storage.CacheSource
}

// NewResolver creates a new DNS resolver instance
func NewResolver(storage storage.Storage, config *Config) *Resolver {
	return &Resolver{
		storage: storage,
	}
}

// ResolveWithSource performs DNS resolution with source tracking
func (r *Resolver) ResolveWithSource(ctx context.Context, query *models.LookupQuery) (*ResolverResult, error) {
	switch query.Type {
	case models.RecordTypeSOA:
		return r.resolveSOAWithSource(ctx, query)
	default:
		// Check if storage supports source tracking
		if sourceStorage, ok := r.storage.(interface {
			LookupRecordWithSource(context.Context, *models.LookupQuery) (*storage.LookupResult, error)
		}); ok {
			result, err := sourceStorage.LookupRecordWithSource(ctx, query)
			if err != nil {
				return nil, err
			}
			if result == nil {
				return nil, nil
			}
			return &ResolverResult{
				Record: result.Record,
				Source: result.Source,
			}, nil
		}

		// Fallback to regular lookup without source tracking
		record, err := r.storage.LookupRecord(ctx, query)
		if err != nil {
			return nil, err
		}
		return &ResolverResult{
			Record: record,
			Source: storage.SourceDatabase, // Assume database if no source tracking
		}, nil
	}
}

// ResolveAllWithSource returns all records with source tracking
func (r *Resolver) ResolveAllWithSource(ctx context.Context, query *models.LookupQuery) (*ResolverGroupResult, error) {
	switch query.Type {
	case models.RecordTypeSOA:
		result, err := r.resolveSOAWithSource(ctx, query)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		return &ResolverGroupResult{
			Records: []*models.DNSRecord{result.Record},
			Source:  result.Source,
		}, nil
	default:
		// Check if storage supports source tracking
		if sourceStorage, ok := r.storage.(interface {
			LookupRecordGroupWithSource(context.Context, *models.LookupQuery) (*storage.LookupGroupResult, error)
		}); ok {
			result, err := sourceStorage.LookupRecordGroupWithSource(ctx, query)
			if err != nil {
				return nil, err
			}
			if result == nil {
				return nil, nil
			}
			return &ResolverGroupResult{
				Records: result.Records,
				Source:  result.Source,
			}, nil
		}

		// Fallback to regular lookup without source tracking
		records, err := r.storage.LookupRecords(ctx, query)
		if err != nil {
			return nil, err
		}
		return &ResolverGroupResult{
			Records: records,
			Source:  storage.SourceDatabase,
		}, nil
	}
}

// resolveSOAWithSource implements SOA resolution with source tracking
func (r *Resolver) resolveSOAWithSource(ctx context.Context, query *models.LookupQuery) (*ResolverResult, error) {
	domains := r.generateDomainHierarchy(query.Name)

	for _, domain := range domains {
		soaQuery := &models.LookupQuery{
			Name: domain,
			Type: models.RecordTypeSOA,
		}

		// Check if storage supports source tracking
		if sourceStorage, ok := r.storage.(interface {
			LookupRecordWithSource(context.Context, *models.LookupQuery) (*storage.LookupResult, error)
		}); ok {
			result, err := sourceStorage.LookupRecordWithSource(ctx, soaQuery)
			if err != nil {
				return nil, err
			}
			if result != nil && result.Record != nil {
				resultRecord := *result.Record
				resultRecord.Name = query.Name
				return &ResolverResult{
					Record: &resultRecord,
					Source: result.Source,
				}, nil
			}
		} else {
			// Fallback to regular lookup
			record, err := r.storage.LookupRecord(ctx, soaQuery)
			if err != nil {
				return nil, err
			}
			if record != nil {
				resultRecord := *record
				resultRecord.Name = query.Name
				return &ResolverResult{
					Record: &resultRecord,
					Source: storage.SourceDatabase,
				}, nil
			}
		}
	}

	return nil, nil
}

// Resolve performs DNS resolution with DNS-specific logic
func (r *Resolver) Resolve(ctx context.Context, query *models.LookupQuery) (*models.DNSRecord, error) {
	switch query.Type {
	case models.RecordTypeSOA:
		return r.resolveSOA(ctx, query)
	default:
		// For all other record types, use direct storage lookup
		return r.storage.LookupRecord(ctx, query)
	}
}

// ResolveAll returns all records matching the query with DNS-specific logic
func (r *Resolver) ResolveAll(ctx context.Context, query *models.LookupQuery) ([]*models.DNSRecord, error) {
	switch query.Type {
	case models.RecordTypeSOA:
		// For SOA, we only want one record (the authoritative one)
		record, err := r.resolveSOA(ctx, query)
		if err != nil {
			return nil, err
		}
		if record == nil {
			return nil, nil
		}
		return []*models.DNSRecord{record}, nil
	default:
		// For other record types, return all matching records
		return r.storage.LookupRecords(ctx, query)
	}
}

// ResolveGroup returns the highest priority group of records
func (r *Resolver) ResolveGroup(ctx context.Context, query *models.LookupQuery) ([]*models.DNSRecord, error) {
	switch query.Type {
	case models.RecordTypeSOA:
		// For SOA, we only want one record
		record, err := r.resolveSOA(ctx, query)
		if err != nil {
			return nil, err
		}
		if record == nil {
			return nil, nil
		}
		return []*models.DNSRecord{record}, nil
	default:
		// For other record types, return the priority group
		return r.storage.LookupRecordGroup(ctx, query)
	}
}

// resolveSOA implements SOA resolution with domain hierarchy walking
func (r *Resolver) resolveSOA(ctx context.Context, query *models.LookupQuery) (*models.DNSRecord, error) {
	// Generate domain hierarchy from specific to general
	domains := r.generateDomainHierarchy(query.Name)

	// Try each domain in the hierarchy
	for _, domain := range domains {
		soaQuery := &models.LookupQuery{
			Name: domain,
			Type: models.RecordTypeSOA,
		}

		record, err := r.storage.LookupRecord(ctx, soaQuery)
		if err != nil {
			return nil, err
		}

		if record != nil {
			// Found SOA record, but update the name to match original query
			// This maintains the illusion that the SOA applies to the queried domain
			resultRecord := *record
			resultRecord.Name = query.Name
			return &resultRecord, nil
		}
	}

	return nil, nil // No SOA found in hierarchy
}

// generateDomainHierarchy creates a list of domains from specific to general
// Example: "www.test.internal" -> ["www.test.internal", "test.internal", "internal"]
func (r *Resolver) generateDomainHierarchy(domain string) []string {
	// Normalize domain name (remove trailing dot if present)
	domain = strings.TrimSuffix(domain, ".")
	domain = strings.ToLower(domain)

	var hierarchy []string
	parts := strings.Split(domain, ".")

	// Generate all possible parent domains
	for i := 0; i < len(parts); i++ {
		subdomain := strings.Join(parts[i:], ".")
		hierarchy = append(hierarchy, subdomain)
	}

	return hierarchy
}

// Future methods for wildcard resolution will go here:

// resolveWildcard will implement wildcard pattern matching
// func (r *Resolver) resolveWildcard(ctx context.Context, query *models.LookupQuery) (*models.DNSRecord, error) {
//     // Wildcard resolution logic will be implemented here
//     return nil, nil
// }

// Additional utility methods for DNS resolution can be added here
