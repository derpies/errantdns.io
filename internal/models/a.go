// A Record Validation
//
// Validates DNS A records according to RFC 1035 standards:
// - Must contain valid IPv4 address (dotted decimal notation)
// - Cannot be empty
// - Rejects IPv6 addresses (use AAAA instead)
// - Rejects Class E addresses (240.0.0.0/4)
// - Rejects addresses starting with 0.x.x.x
// - Allows private/internal addresses (10.x.x.x, 192.168.x.x, etc.)
//
// Examples:
//   192.168.1.10    (valid private)
//   8.8.8.8         (valid public)
//   ::1             (invalid - IPv6)
//   0.0.0.0         (invalid - zero network)

package models

import (
	"fmt"
	"net"
)

func (r *DNSRecord) validateARecord() error {
	// A records must contain a valid IPv4 address
	if r.Target == "" {
		return fmt.Errorf("A record target cannot be empty")
	}

	// Parse as IPv4 address
	ip := net.ParseIP(r.Target)
	if ip == nil {
		return fmt.Errorf("A record target is not a valid IP address: %s", r.Target)
	}

	// Ensure it's specifically IPv4 (not IPv6)
	if ip.To4() == nil {
		return fmt.Errorf("A record target must be IPv4 address, got IPv6: %s", r.Target)
	}

	// Validate against reserved/special use addresses if needed
	ipv4 := ip.To4()

	// Check for obviously invalid addresses
	if ipv4[0] == 0 {
		return fmt.Errorf("A record target cannot start with 0: %s", r.Target)
	}

	// Class E addresses (240.0.0.0/4) are reserved
	if ipv4[0] >= 240 {
		return fmt.Errorf("A record target cannot use Class E address space: %s", r.Target)
	}

	// Optionally warn about private/special addresses (but don't error)
	// 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 127.0.0.0/8, etc.
	// These are valid in DNS records even if not globally routable

	return nil
}
