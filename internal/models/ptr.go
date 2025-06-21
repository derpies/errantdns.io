// PTR Record Name Validation
//
// Validates PTR record name field according to RFC 1035/3596 standards:
// - IPv4: Must be in format "octet.octet.octet.octet.in-addr.arpa"
// - IPv6: Must be in format "digit.digit...digit.ip6.arpa"
// - Each IPv4 octet must be 0-255
// - Each IPv6 digit must be valid hex (0-9, a-f, A-F)
// - IPv6 can have 1-32 hex digits (partial reverse zones allowed)
// - Case insensitive for hex digits and domain suffixes
//
// Examples:
//   1.0.168.192.in-addr.arpa          (valid IPv4: 192.168.0.1)
//   1.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa  (valid IPv6)
//   10.168.192.in-addr.arpa           (valid partial IPv4 zone)
//   example.com                       (invalid - not reverse format)
//   256.0.168.192.in-addr.arpa        (invalid - octet > 255)

// PTR Record Validation
//
// Validates DNS PTR records according to RFC 1035 standards:
// - Must contain valid domain name (FQDN)
// - Cannot be empty
// - Cannot point to IP addresses
// - Used for reverse DNS lookups (IP → hostname)
// - Typically found in .in-addr.arpa (IPv4) and .ip6.arpa (IPv6) zones
// - Target should resolve back to original IP (but not enforced here)
//
// Examples:
//   mail.example.com      (valid hostname)
//   server01.local.       (valid FQDN)
//   192.168.1.10          (invalid - IP address)
//   ""                    (invalid - empty)
//
// Note: PTR name format (1.0.168.192.in-addr.arpa) validated elsewhere
// This only validates the target hostname the PTR points to

package models

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func (r *DNSRecord) validatePTRRecord() error {
	// PTR records must contain a valid domain name
	if r.Target == "" {
		return fmt.Errorf("PTR record target cannot be empty")
	}

	// PTR target must be a valid domain name (FQDN)
	if err := r.validateDomainName(); err != nil {
		return fmt.Errorf("PTR record target is not a valid domain name: %s", r.Target)
	}

	// PTR cannot point to an IP address
	if net.ParseIP(r.Target) != nil {
		return fmt.Errorf("PTR record target cannot be an IP address: %s", r.Target)
	}

	// PTR cannot be empty after normalization
	normalized := NormalizeDomainName(r.Target)
	if normalized == "" {
		return fmt.Errorf("PTR record target cannot be empty after normalization: %s", r.Target)
	}

	return nil
}

func (r *DNSRecord) validatePTRName() error {
	// PTR records must have a name in reverse DNS format
	if r.Name == "" {
		return fmt.Errorf("PTR record name cannot be empty")
	}

	// Normalize the name
	normalized := NormalizeDomainName(r.Name)

	// Check for IPv4 reverse DNS format (x.x.x.x.in-addr.arpa)
	if strings.HasSuffix(normalized, ".in-addr.arpa") {
		// Remove the suffix to get the IP part
		ipPart := strings.TrimSuffix(normalized, ".in-addr.arpa")

		// Split into octets
		octets := strings.Split(ipPart, ".")
		if len(octets) != 4 {
			return fmt.Errorf("PTR record name invalid IPv4 format: %s (expected 4 octets)", r.Name)
		}

		// Validate each octet (0-255)
		for _, octet := range octets {
			val, err := strconv.Atoi(octet)
			if err != nil {
				return fmt.Errorf("PTR record name invalid octet '%s': %s", octet, r.Name)
			}
			if val < 0 || val > 255 {
				return fmt.Errorf("PTR record name octet out of range (%d): %s", val, r.Name)
			}
		}

		return nil
	}

	// Check for IPv6 reverse DNS format (x.x.x...x.ip6.arpa)
	if strings.HasSuffix(normalized, ".ip6.arpa") {
		// Remove the suffix to get the hex part
		hexPart := strings.TrimSuffix(normalized, ".ip6.arpa")

		// Split into hex digits
		hexDigits := strings.Split(hexPart, ".")

		// IPv6 can have up to 32 hex digits (128 bits / 4 bits per hex digit)
		if len(hexDigits) > 32 {
			return fmt.Errorf("PTR record name too many hex digits for IPv6: %s", r.Name)
		}

		// Validate each hex digit
		for _, digit := range hexDigits {
			if len(digit) != 1 {
				return fmt.Errorf("PTR record name invalid hex digit length '%s': %s", digit, r.Name)
			}

			if !((digit[0] >= '0' && digit[0] <= '9') ||
				(digit[0] >= 'a' && digit[0] <= 'f') ||
				(digit[0] >= 'A' && digit[0] <= 'F')) {
				return fmt.Errorf("PTR record name invalid hex digit '%s': %s", digit, r.Name)
			}
		}

		return nil
	}

	// PTR record name must be in reverse DNS format
	return fmt.Errorf("PTR record name must end with .in-addr.arpa (IPv4) or .ip6.arpa (IPv6): %s", r.Name)
}
