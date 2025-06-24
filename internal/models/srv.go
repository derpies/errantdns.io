// SRV Record Validation
//
// Validates DNS SRV records according to RFC 2782 standards:
// - Target must be valid domain name or "." (no service available)
// - Service format: "_service._protocol.domain" (underscores required)
// - Protocol must be "_tcp" or "_udp"
// - Priority: 0-65535 (lower values = higher priority)
// - Weight: 0-65535 (for load balancing among same priority)
// - Port: 1-65535 (0 invalid for SRV)
// - Target format in database: "priority weight port target"
//
// Examples:
//   "_http._tcp.example.com" → "10 60 80 web1.example.com"     (valid)
//   "_sip._udp.example.com"  → "0 5 5060 sip.example.com"     (valid)
//   "_service._tcp.test.com" → "10 0 443 ."                   (valid - no service)
//   "http._tcp.example.com"  → (invalid - missing underscore)
//   "_http._sctp.example.com"→ (invalid - unsupported protocol)
//
// Note: Name format validation separate from target validation

package models

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func (r *DNSRecord) validateSRVTarget() error {
	// SRV records must contain priority, weight, port, and target
	if r.Target == "" {
		return fmt.Errorf("SRV record target cannot be empty")
	}

	fields := strings.Fields(r.Target)
	if len(fields) != 4 {
		return fmt.Errorf("SRV record target must have 4 fields (priority weight port target), got %d", len(fields))
	}

	priorityStr := fields[0]
	weightStr := fields[1]
	portStr := fields[2]
	targetHost := fields[3]

	// Validate priority (0-65535)
	priority, err := strconv.ParseUint(priorityStr, 10, 16)
	if err != nil {
		return fmt.Errorf("SRV priority invalid: %s is not a valid 16-bit unsigned integer", priorityStr)
	}
	_ = priority // Valid priority

	// Validate weight (0-65535)
	weight, err := strconv.ParseUint(weightStr, 10, 16)
	if err != nil {
		return fmt.Errorf("SRV weight invalid: %s is not a valid 16-bit unsigned integer", weightStr)
	}
	_ = weight // Valid weight

	// Validate port (1-65535, 0 is invalid for SRV)
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return fmt.Errorf("SRV port invalid: %s is not a valid 16-bit unsigned integer", portStr)
	}
	if port == 0 {
		return fmt.Errorf("SRV port cannot be 0")
	}

	// Validate target host
	if targetHost == "." {
		// Special case: "." means no service available (RFC 2782)
		return nil
	}

	// Target must be a valid domain name
	if err := r.validateDomainName(); err != nil {
		return fmt.Errorf("SRV target host is not a valid domain name: %s", targetHost)
	}

	// SRV target cannot be an IP address
	if net.ParseIP(targetHost) != nil {
		return fmt.Errorf("SRV target cannot be an IP address: %s", targetHost)
	}

	return nil
}

func (r *DNSRecord) validateSRVName() error {
	// SRV records must have name in format "_service._protocol.domain"
	if r.Name == "" {
		return fmt.Errorf("SRV record name cannot be empty")
	}

	// Normalize the name
	normalized := NormalizeDomainName(r.Name)

	// Split into labels
	labels := strings.Split(normalized, ".")
	if len(labels) < 3 {
		return fmt.Errorf("SRV record name must have at least 3 labels: _service._protocol.domain")
	}

	// First label must be service name starting with underscore
	serviceLabel := labels[0]
	if !strings.HasPrefix(serviceLabel, "_") {
		return fmt.Errorf("SRV service label must start with underscore: %s", serviceLabel)
	}
	if len(serviceLabel) < 2 {
		return fmt.Errorf("SRV service label too short: %s", serviceLabel)
	}

	// Validate service name characters (after underscore)
	serviceName := serviceLabel[1:]
	for i, r := range serviceName {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-') {
			return fmt.Errorf("SRV service name invalid character '%c' at position %d: %s", r, i, serviceLabel)
		}
	}

	// Second label must be protocol starting with underscore
	protocolLabel := labels[1]
	if !strings.HasPrefix(protocolLabel, "_") {
		return fmt.Errorf("SRV protocol label must start with underscore: %s", protocolLabel)
	}

	// Validate protocol (must be _tcp or _udp)
	protocol := strings.ToLower(protocolLabel)
	if protocol != "_tcp" && protocol != "_udp" {
		return fmt.Errorf("SRV protocol must be _tcp or _udp, got: %s", protocolLabel)
	}

	// Remaining labels form the domain name - validate using standard domain validation
	domainLabels := labels[2:]
	domainName := strings.Join(domainLabels, ".")

	if err := r.validateDomainName(); err != nil {
		return fmt.Errorf("SRV domain portion invalid: %s", domainName)
	}

	return nil
}

func (r *DNSRecord) validateSRVRecord() error {
	if err := r.validateSRVTarget(); err != nil {
		return fmt.Errorf("invalid SRV record: %s: %w", r.Target, err)
	}
	if err := r.validateSRVName(); err != nil {
		return fmt.Errorf("invalid SRV record: %s: %w", r.Name, err)
	}
	return nil
}
