// internal/models/dns.go
package models

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// DNSRecord represents a DNS record from storage
type DNSRecord struct {
	ID              int       `db:"id"`
	Name            string    `db:"name"`
	RecordType      string    `db:"record_type"`
	Target          string    `db:"target"`
	TTL             uint32    `db:"ttl"`
	Priority        int       `db:"priority"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
	ETLD            string    `db:"etld"`
	ApexDomain      string    `db:"apex_domain"`
	SubdomainLabels []string  `db:"subdomain_labels"`
	IsWildcard      bool      `db:"is_wildcard"`
	WildcardMask    uint64    `db:"wildcard_mask"` //bitstring
	Serial          uint32    `db:"serial"`
	Mbox            string    `db:"mbox"`
	Refresh         uint32    `db:"refresh"`
	Retry           uint32    `db:"retry"`
	Expire          uint32    `db:"expire"`
	Minttl          uint32    `db:"minttl"`
	Weight          uint32    `db:"weight"`
	Port            uint16    `db:"port"`
	Tag             string    `db:"tag"`
}

// RecordType represents supported DNS record types
type RecordType string

const (
	RecordTypeA     RecordType = "A"
	RecordTypeAAAA  RecordType = "AAAA"
	RecordTypeCNAME RecordType = "CNAME"
	RecordTypeTXT   RecordType = "TXT"
	RecordTypeMX    RecordType = "MX"
	RecordTypeNS    RecordType = "NS"
	RecordTypeSOA   RecordType = "SOA"
	RecordTypePTR   RecordType = "PTR"
	RecordTypeSRV   RecordType = "SRV"
	RecordTypeCAA   RecordType = "CAA"
	RecordTypeTLSA  RecordType = "TLSA"
)

// IsValid returns true if the record type is supported
func (rt RecordType) IsValid() bool {
	switch rt {
	case RecordTypeA, RecordTypeAAAA, RecordTypeCNAME, RecordTypeTXT, RecordTypeMX, RecordTypeNS, RecordTypeSOA, RecordTypePTR, RecordTypeSRV, RecordTypeCAA, RecordTypeTLSA:
		return true
	default:
		return false
	}
}

// String returns the string representation of the record type
func (rt RecordType) String() string {
	return string(rt)
}

// LookupQuery represents a DNS lookup request
type LookupQuery struct {
	Name string
	Type RecordType
}

// NewLookupQuery creates a normalized lookup query
func NewLookupQuery(name string, recordType string) *LookupQuery {
	return &LookupQuery{
		Name: NormalizeDomainName(name),
		Type: RecordType(strings.ToUpper(recordType)),
	}
}

// CacheKey returns a string key for caching this query
func (q *LookupQuery) CacheKey() string {
	return fmt.Sprintf("%s:%s", q.Name, q.Type)
}

// NormalizeDomainName normalizes a domain name for consistent storage/lookup
func NormalizeDomainName(name string) string {
	return strings.ToLower(strings.TrimSuffix(name, "."))
}

// Validate performs validation on a DNS record
func (r *DNSRecord) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	recordType := RecordType(r.RecordType)
	if !recordType.IsValid() {
		return fmt.Errorf("invalid record type: %s", r.RecordType)
	}

	if r.Target == "" {
		return fmt.Errorf("target cannot be empty")
	}

	// Type-specific validation
	switch recordType {
	case RecordTypeA:
		if err := r.validateARecord(); err != nil {
			return fmt.Errorf("invalid A record: %s: %w", r.Target, err)
		}
	case RecordTypeAAAA:
		if err := r.validateAAAARecord(); err != nil {
			return fmt.Errorf("invalid AAAA record: %s: %w", r.Target, err)
		}
	case RecordTypeCNAME:
		if err := r.validateCNAMERecord(); err != nil {
			return fmt.Errorf("invalid CNAME record: %s: %w", r.Target, err)
		}
	case RecordTypeMX:
		if err := r.validateMXTarget(); err != nil {
			return fmt.Errorf("invalid MX target domain: %s: %w", r.Target, err)
		}
	case RecordTypeTXT:
		// TXT records can contain any text, minimal validation
		if err := r.validateTXTRecord(); err != nil {
			return fmt.Errorf("TXT record too long: %d characters", len(r.Target))
		}
	case RecordTypeSOA:
		if err := r.validateSOARecord(); err != nil {
			return fmt.Errorf("invalid SOA target domain: %s: %w", r.Target, err)
		}
	case RecordTypePTR:
		if err := r.validatePTRRecord(); err != nil {
			return fmt.Errorf("invalid PTR record: %s: %w", r.Target, err)
		}
		if err := r.validatePTRName(); err != nil {
			return fmt.Errorf("invalid PTR name: %s: %w", r.Name, err)
		}
	case RecordTypeNS:
		if err := r.validateNSRecord(); err != nil {
			return fmt.Errorf("invalid NS record: %s: %w", r.Target, err)
		}
	case RecordTypeSRV:
		if err := r.validateSRVRecord(); err != nil {
			return fmt.Errorf("invalid SRV record: %s: %w", r.Target, err)
		}
	case RecordTypeCAA:
		if err := r.validateCAARecord(); err != nil {
			return fmt.Errorf("invalid CAA record: %s: %w", r.Target, err)
		}
	case RecordTypeTLSA:
		if err := r.validateTLSARecord(); err != nil {
			return fmt.Errorf("invalid TLSA record: %s: %w", r.Target, err)
		}
	}

	if r.TTL > 2147483647 {
		return fmt.Errorf("TTL too large: %d", r.TTL)
	}

	return nil
}

// Normalize ensures the DNS record has consistent formatting
func (r *DNSRecord) Normalize() {
	r.Name = NormalizeDomainName(r.Name)
	r.RecordType = strings.ToUpper(r.RecordType)

	// Normalize target based on record type
	recordType := RecordType(r.RecordType)
	switch recordType {
	case RecordTypeCNAME, RecordTypeNS, RecordTypeMX:
		// Ensure domain targets are normalized
		r.Target = NormalizeDomainName(r.Target)
	case RecordTypeA, RecordTypeAAAA:
		// IP addresses should be consistent format
		if ip := net.ParseIP(r.Target); ip != nil {
			r.Target = ip.String()
		}
	}
}

// RecordSet represents a collection of DNS records for the same name/type
type RecordSet struct {
	Name    string
	Type    RecordType
	Records []*DNSRecord
}

// NewRecordSet creates a new record set
func NewRecordSet(name string, recordType RecordType) *RecordSet {
	return &RecordSet{
		Name:    NormalizeDomainName(name),
		Type:    recordType,
		Records: make([]*DNSRecord, 0),
	}
}

// Add adds a record to the set if it matches the name/type
func (rs *RecordSet) Add(record *DNSRecord) error {
	if NormalizeDomainName(record.Name) != rs.Name {
		return fmt.Errorf("record name mismatch: expected %s, got %s", rs.Name, record.Name)
	}

	if RecordType(record.RecordType) != rs.Type {
		return fmt.Errorf("record type mismatch: expected %s, got %s", rs.Type, record.RecordType)
	}

	rs.Records = append(rs.Records, record)
	return nil
}

// IsEmpty returns true if the record set has no records
func (rs *RecordSet) IsEmpty() bool {
	return len(rs.Records) == 0
}

// HighestPriority returns the record with the highest priority (lowest number for MX)
func (rs *RecordSet) HighestPriority() *DNSRecord {
	if len(rs.Records) == 0 {
		return nil
	}

	highest := rs.Records[0]
	for _, record := range rs.Records[1:] {
		// For MX records, lower priority number = higher priority
		if rs.Type == RecordTypeMX {
			if record.Priority < highest.Priority {
				highest = record
			}
		} else {
			// For other records, higher priority number = higher priority
			if record.Priority > highest.Priority {
				highest = record
			}
		}
	}

	return highest
}
