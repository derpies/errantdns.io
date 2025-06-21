// AAAA Record Validation
//
// Validates DNS AAAA records according to RFC 3596 standards:
// - Must contain valid IPv6 address (colon-separated hexadecimal)
// - Cannot be empty
// - Rejects IPv4 addresses (use A instead)
// - Rejects unspecified address (::)
// - Supports compressed notation (::1, 2001:db8::1)
// - Allows link-local, ULA, and other special-use addresses
//
// Examples:
//   2001:db8::1           (valid global)
//   ::1                   (valid loopback)
//   fe80::1               (valid link-local)
//   192.168.1.1           (invalid - IPv4)
//   ::                    (invalid - unspecified)

package models

import (
	"fmt"
	"net"
)

func (r *DNSRecord) validateAAAARecord() error {
	// AAAA records must contain a valid IPv6 address
	if r.Target == "" {
		return fmt.Errorf("AAAA record target cannot be empty")
	}

	// Parse as IP address
	ip := net.ParseIP(r.Target)
	if ip == nil {
		return fmt.Errorf("AAAA record target is not a valid IP address: %s", r.Target)
	}

	// Ensure it's specifically IPv6 (not IPv4)
	if ip.To4() != nil {
		return fmt.Errorf("AAAA record target must be IPv6 address, got IPv4: %s", r.Target)
	}

	// Additional IPv6 validation
	ipv6 := ip.To16()
	if ipv6 == nil {
		return fmt.Errorf("AAAA record target is not a valid IPv6 address: %s", r.Target)
	}

	// Check for IPv6 unspecified address (::)
	if ip.Equal(net.IPv6unspecified) {
		return fmt.Errorf("AAAA record target cannot be unspecified address (::): %s", r.Target)
	}

	// Optionally validate against reserved ranges if needed
	// Note: Many IPv6 special-use addresses are still valid in DNS
	// Examples: ::1 (loopback), fe80::/10 (link-local), fc00::/7 (ULA)

	return nil
}
