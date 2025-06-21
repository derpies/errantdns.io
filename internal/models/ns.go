// NS Record Validation
//
// Validates DNS NS records according to RFC 1035/1912 standards:
// - Must contain valid domain name (FQDN) of nameserver
// - Cannot be empty
// - Cannot point to IP addresses (use A/AAAA for nameserver IP)
// - Cannot point to CNAME targets (RFC 1912 - requires resolution to verify)
// - Used for DNS delegation and zone authority
// - Target nameserver should have A/AAAA records (glue records)
//
// Examples:
//   ns1.example.com       (valid nameserver)
//   dns.provider.net.     (valid FQDN)
//   192.168.1.53          (invalid - IP address)
//   ""                    (invalid - empty)
//   alias.example.com     (potentially invalid if CNAME exists)
//
// Note: CNAME conflict detection requires DNS resolution during validation
// Glue record validation (A/AAAA for NS target) handled separately

package models

import (
	"fmt"
	"net"
)

func (r *DNSRecord) validateNSRecord() error {
	// NS records must contain a valid domain name
	if r.Target == "" {
		return fmt.Errorf("NS record target cannot be empty")
	}

	// NS target must be a valid domain name (FQDN)
	if err := r.validateDomainName(); err != nil {
		return fmt.Errorf("NS record target is not a valid domain name: %s", r.Target)
	}

	// NS cannot point to an IP address
	if net.ParseIP(r.Target) != nil {
		return fmt.Errorf("NS record target cannot be an IP address: %s", r.Target)
	}

	// NS cannot be empty after normalization
	normalized := NormalizeDomainName(r.Target)
	if normalized == "" {
		return fmt.Errorf("NS record target cannot be empty after normalization: %s", r.Target)
	}

	// NS cannot point to a CNAME (RFC 1912 section 2.4)
	// Note: This would require DNS resolution to fully validate
	// We can only do basic format checking here

	return nil
}
