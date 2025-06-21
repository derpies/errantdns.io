// MX Record Validation
//
// Validates DNS MX records according to RFC 1035/2181 standards:
// - Must contain valid domain name (FQDN)
// - Cannot be empty (except special null MX case)
// - Cannot point to IP addresses (use A/AAAA for mail server)
// - Cannot point to CNAME targets (RFC 2181 - requires resolution to verify)
// - Supports null MX record "." (RFC 7505 - no mail accepted)
// - Priority field validated separately in models layer
//
// Examples:
//   mail.example.com      (valid mail server)
//   mx1.provider.net.     (valid FQDN)
//   .                     (valid null MX)
//   192.168.1.10          (invalid - IP address)
//   alias.example.com     (potentially invalid if CNAME exists)
//
// Note: CNAME conflict detection requires DNS resolution during validation

package models

import (
	"fmt"
	"net"
)

func (r *DNSRecord) validateMXTarget() error {
	// MX records must contain a valid domain name
	if r.Target == "" {
		return fmt.Errorf("MX record target cannot be empty")
	}

	// MX target must be a valid domain name (FQDN)
	if err := r.validateDomainName(); err != nil {
		return fmt.Errorf("MX record target is not a valid domain name: %s", r.Target)
	}

	// MX cannot point to an IP address
	if net.ParseIP(r.Target) != nil {
		return fmt.Errorf("MX record target cannot be an IP address: %s", r.Target)
	}

	// MX cannot be empty after normalization
	normalized := NormalizeDomainName(r.Target)
	if normalized == "" {
		return fmt.Errorf("MX record target cannot be empty after normalization: %s", r.Target)
	}

	// MX cannot point to a CNAME (RFC 2181 section 10.3)
	// Note: This would require DNS resolution to fully validate
	// We can only do basic format checking here

	// Special case: MX can point to "." (null MX - RFC 7505)
	if r.Target == "." {
		return nil // Valid null MX record
	}

	return nil
}
