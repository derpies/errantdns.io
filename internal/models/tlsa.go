// TLSA Record Validation
//
// Validates DNS TLSA records according to RFC 6698 standards:
// - Certificate Usage field: 0-3 (CA constraint, Service certificate constraint, Trust anchor assertion, Domain-issued certificate)
// - Selector field: 0-1 (Full certificate, SubjectPublicKeyInfo)
// - Matching Type field: 0-2 (Exact match, SHA-256 hash, SHA-512 hash)
// - Certificate Association Data: Valid hexadecimal string
// - Name format: Must be _port._protocol.domain (e.g., _443._tcp.example.com)
// - Target format: "usage selector matchtype certdata"
//
// Examples:
//   "_443._tcp.example.com" → "3 1 1 1234567890ABCDEF..." (valid)
//   "_25._tcp.mail.example.com" → "2 0 1 ABCDEF1234567890..." (valid)
//   "_443._udp.example.com" → "3 1 1 1234..." (valid - DTLS)
//   "example.com" → "3 1 1 1234..." (invalid - wrong name format)
//   "_443._tcp.example.com" → "4 1 1 1234..." (invalid - usage out of range)

package models

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func (r *DNSRecord) validateTLSARecord() error {
	// TLSA name must be in _port._protocol.domain format
	if err := r.validateTLSAName(); err != nil {
		return err
	}

	// TLSA target must contain certificate usage, selector, matching type, and cert data
	if err := r.validateTLSATarget(); err != nil {
		return err
	}

	return nil
}

// validateTLSAName validates the TLSA record name format
func (r *DNSRecord) validateTLSAName() error {
	if r.Name == "" {
		return fmt.Errorf("TLSA record name cannot be empty")
	}

	// Normalize the name
	normalized := NormalizeDomainName(r.Name)

	// TLSA records must be in format _port._protocol.domain
	// Example: _443._tcp.example.com
	parts := strings.Split(normalized, ".")
	if len(parts) < 3 {
		return fmt.Errorf("TLSA record name must have at least 3 labels: _port._protocol.domain")
	}

	// First label must be port starting with underscore
	portLabel := parts[0]
	if !strings.HasPrefix(portLabel, "_") {
		return fmt.Errorf("TLSA port label must start with underscore: %s", portLabel)
	}
	if len(portLabel) < 2 {
		return fmt.Errorf("TLSA port label too short: %s", portLabel)
	}

	// Validate port number (after underscore)
	portStr := portLabel[1:]
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return fmt.Errorf("TLSA port number invalid: %s", portStr)
	}
	if port == 0 || port > 65535 {
		return fmt.Errorf("TLSA port number out of range (1-65535): %d", port)
	}

	// Second label must be protocol starting with underscore
	protocolLabel := parts[1]
	if !strings.HasPrefix(protocolLabel, "_") {
		return fmt.Errorf("TLSA protocol label must start with underscore: %s", protocolLabel)
	}

	// Validate protocol (common protocols for TLSA)
	protocol := strings.ToLower(protocolLabel)
	validProtocols := map[string]bool{
		"_tcp":  true, // Most common - HTTPS, SMTP, etc.
		"_udp":  true, // DTLS, QUIC
		"_sctp": true, // Less common but valid
	}

	if !validProtocols[protocol] {
		return fmt.Errorf("TLSA protocol must be _tcp, _udp, or _sctp, got: %s", protocolLabel)
	}

	// Remaining labels form the domain name - validate using standard domain validation
	domainLabels := parts[2:]
	domainName := strings.Join(domainLabels, ".")

	if err := r.validateDomainNameOther(domainName); err != nil {
		return fmt.Errorf("TLSA domain portion invalid: %s", domainName)
	}

	return nil
}

// validateTLSATarget validates the TLSA record target format and values
func (r *DNSRecord) validateTLSATarget() error {
	if r.Target == "" {
		return fmt.Errorf("TLSA record target cannot be empty")
	}

	fields := strings.Fields(r.Target)
	if len(fields) != 4 {
		return fmt.Errorf("TLSA record target must have 4 fields (usage selector matchtype certdata), got %d", len(fields))
	}

	usageStr := fields[0]
	selectorStr := fields[1]
	matchTypeStr := fields[2]
	certData := fields[3]

	// Validate Certificate Usage (0-3)
	usage, err := strconv.ParseUint(usageStr, 10, 8)
	if err != nil {
		return fmt.Errorf("TLSA certificate usage invalid: %s is not a valid integer", usageStr)
	}
	if usage > 3 {
		return fmt.Errorf("TLSA certificate usage out of range (0-3): %d", usage)
	}

	// Validate Selector (0-1)
	selector, err := strconv.ParseUint(selectorStr, 10, 8)
	if err != nil {
		return fmt.Errorf("TLSA selector invalid: %s is not a valid integer", selectorStr)
	}
	if selector > 1 {
		return fmt.Errorf("TLSA selector out of range (0-1): %d", selector)
	}

	// Validate Matching Type (0-2)
	matchType, err := strconv.ParseUint(matchTypeStr, 10, 8)
	if err != nil {
		return fmt.Errorf("TLSA matching type invalid: %s is not a valid integer", matchTypeStr)
	}
	if matchType > 2 {
		return fmt.Errorf("TLSA matching type out of range (0-2): %d", matchType)
	}

	// Validate Certificate Association Data
	if err := r.validateTLSACertData(certData, int(matchType)); err != nil {
		return err
	}

	return nil
}

// validateTLSACertData validates the certificate association data based on matching type
func (r *DNSRecord) validateTLSACertData(certData string, matchType int) error {
	if certData == "" {
		return fmt.Errorf("TLSA certificate data cannot be empty")
	}

	// Must be valid hexadecimal
	hexRegex := regexp.MustCompile("^[0-9A-Fa-f]+$")
	if !hexRegex.MatchString(certData) {
		return fmt.Errorf("TLSA certificate data must be hexadecimal: %s", certData)
	}

	// Must be even length (each byte requires 2 hex characters)
	if len(certData)%2 != 0 {
		return fmt.Errorf("TLSA certificate data must have even length (complete bytes): %d characters", len(certData))
	}

	// Validate expected length based on matching type
	expectedLengths := map[int][]int{
		0: {},    // Exact match - any length valid
		1: {64},  // SHA-256 hash - 32 bytes = 64 hex chars
		2: {128}, // SHA-512 hash - 64 bytes = 128 hex chars
	}

	if lengths, exists := expectedLengths[matchType]; exists && len(lengths) > 0 {
		validLength := false
		for _, expectedLen := range lengths {
			if len(certData) == expectedLen {
				validLength = true
				break
			}
		}
		if !validLength {
			return fmt.Errorf("TLSA certificate data length invalid for matching type %d: got %d characters, expected %v",
				matchType, len(certData), expectedLengths[matchType])
		}
	}

	// Minimum reasonable length check (at least 2 bytes = 4 hex chars)
	if len(certData) < 4 {
		return fmt.Errorf("TLSA certificate data too short: %d characters (minimum 4)", len(certData))
	}

	// Maximum reasonable length check (avoid DoS with huge cert data)
	if len(certData) > 8192 { // 4KB maximum
		return fmt.Errorf("TLSA certificate data too long: %d characters (maximum 8192)", len(certData))
	}

	return nil
}
