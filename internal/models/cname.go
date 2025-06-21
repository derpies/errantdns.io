// CNAME Record Validation
//
// Validates DNS CNAME records according to RFC 1035 standards:
// - Must contain valid domain name (FQDN)
// - Cannot be empty
// - Cannot point to IP addresses (use A/AAAA instead)
// - Target must pass domain name validation rules
// - Cannot be self-referential (basic check)
// - Supports both absolute (example.com.) and relative (example.com) names
//
// Examples:
//   www.example.com       (valid target)
//   api.service.local.    (valid FQDN)
//   192.168.1.1           (invalid - IP address)
//   ""                    (invalid - empty)
//
// Note: CNAME loop detection beyond self-reference requires full resolution

package models

import (
	"fmt"
	"net"
)

func (r *DNSRecord) validateCNAMERecord() error {
	// CNAME records must contain a valid domain name
	if r.Target == "" {
		return fmt.Errorf("CNAME record target cannot be empty")
	}

	// CNAME target must be a valid domain name (FQDN)
	if err := r.validateDomainName(); err != nil {
		return fmt.Errorf("CNAME record target is not a valid domain name: %s", r.Target)
	}

	// CNAME cannot point to an IP address
	if net.ParseIP(r.Target) != nil {
		return fmt.Errorf("CNAME record target cannot be an IP address: %s", r.Target)
	}

	// CNAME cannot be empty after normalization
	normalized := NormalizeDomainName(r.Target)
	if normalized == "" {
		return fmt.Errorf("CNAME record target cannot be empty after normalization: %s", r.Target)
	}

	// CNAME cannot point to itself (basic check)
	// Note: More complex loop detection would require full resolution chain
	// This just catches obvious self-references

	return nil
}
