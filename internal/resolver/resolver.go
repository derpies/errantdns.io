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

// NewResolver creates a new DNS resolver instance
func NewResolver(storage storage.Storage, config *Config) *Resolver {
	return &Resolver{
		storage: storage,
	}
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
