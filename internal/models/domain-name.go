// Domain Name Validation and ETLD Processing with Public Suffix List
//
// Validates DNS domain names according to RFC 1035/1123/3696 standards and extracts
// ETLD (Effective Top-Level Domain) information using Mozilla's Public Suffix List:
//
// Validation Rules:
// - Total length: 1-253 characters
// - Label length: 1-63 characters each
// - Valid characters: a-z, A-Z, 0-9, hyphens (not at label start/end)
// - TLD requirements: minimum 2 chars, must start with letter, not all-numeric
// - Wildcard labels: "*" allowed, partial wildcards rejected
//
// Public Suffix List Integration:
// - Uses golang.org/x/net/publicsuffix for authoritative ETLD detection
// - Handles complex suffixes: "co.uk", "github.io", "s3.amazonaws.com"
// - Supports both ICANN and private suffixes
// - Accurately identifies registrable domain boundaries
//
// ETLD Processing:
// - ETLD: Public suffix from PSL ("com", "co.uk", "github.io")
// - ApexDomain: Registrable domain ("example.com", "user.github.io")
// - SubdomainLabels: Labels before apex domain for wildcard processing
// - WildcardMask: uint64 bitmask (bit N set = label N is wildcard)
//
// Sets DNSRecord Fields:
// - ETLD, ApexDomain, SubdomainLabels, IsWildcard, WildcardMask
//
// Examples:
//   "api.service.example.com"     → ETLD:"com", Apex:"example.com", Subs:["api","service"]
//   "*.v1.user.github.io"         → ETLD:"github.io", Apex:"user.github.io", Subs:["*","v1"], Mask:1
//   "api.*.prod.example.co.uk"    → ETLD:"co.uk", Apex:"example.co.uk", Subs:["api","*","prod"], Mask:2

package models

import (
	"fmt"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// validateDomainName validates the domain name and extracts ETLD/apex information
func (r *DNSRecord) validateDomainName() error {
	domain := r.Name

	if len(domain) == 0 || len(domain) > 253 {
		return fmt.Errorf("domain name length invalid: %d characters (must be 1-253)", len(domain))
	}

	// Handle absolute vs relative names
	domain = strings.TrimSuffix(domain, ".")

	// Empty after removing dot is invalid
	if len(domain) == 0 {
		return fmt.Errorf("domain name cannot be empty")
	}

	// Extract ETLD using Public Suffix List and set DNSRecord fields
	if err := r.extractAndSetETLDInfo(domain); err != nil {
		return fmt.Errorf("ETLD extraction failed: %w", err)
	}

	// Split into labels for validation
	labels := strings.Split(domain, ".")
	if len(labels) == 0 {
		return fmt.Errorf("domain name must contain at least one label")
	}

	// Validate each label
	for i, label := range labels {
		if err := r.validateLabel(label); err != nil {
			return fmt.Errorf("invalid label '%s': %w", label, err)
		}

		// Additional TLD validation for last label (if multiple labels exist)
		if len(labels) > 1 && i == len(labels)-1 {
			if err := r.validateTLD(label); err != nil {
				return fmt.Errorf("invalid TLD '%s': %w", label, err)
			}
		}
	}

	// Detect and process wildcards
	if err := r.detectAndSetWildcards(); err != nil {
		return fmt.Errorf("wildcard processing failed: %w", err)
	}

	return nil
}

// TODO: This is a copy of validateDomainName;  this could probably be made more efficient by combining the two.
// validateDomainName validates the domain name and extracts ETLD/apex information
func (r *DNSRecord) validateDomainNameOther(domain string) error {

	if len(domain) == 0 || len(domain) > 253 {
		return fmt.Errorf("domain name length invalid: %d characters (must be 1-253)", len(domain))
	}

	// Handle absolute vs relative names
	domain = strings.TrimSuffix(domain, ".")

	// Empty after removing dot is invalid
	if len(domain) == 0 {
		return fmt.Errorf("domain name cannot be empty")
	}

	// Extract ETLD using Public Suffix List and set DNSRecord fields
	if err := r.extractAndSetETLDInfo(domain); err != nil {
		return fmt.Errorf("ETLD extraction failed: %w", err)
	}

	// Split into labels for validation
	labels := strings.Split(domain, ".")
	if len(labels) == 0 {
		return fmt.Errorf("domain name must contain at least one label")
	}

	// Validate each label
	for i, label := range labels {
		if err := r.validateLabel(label); err != nil {
			return fmt.Errorf("invalid label '%s': %w", label, err)
		}

		// Additional TLD validation for last label (if multiple labels exist)
		if len(labels) > 1 && i == len(labels)-1 {
			if err := r.validateTLD(label); err != nil {
				return fmt.Errorf("invalid TLD '%s': %w", label, err)
			}
		}
	}

	// Detect and process wildcards
	if err := r.detectAndSetWildcards(); err != nil {
		return fmt.Errorf("wildcard processing failed: %w", err)
	}

	return nil
}

// extractAndSetETLDInfo extracts ETLD using Public Suffix List and sets DNSRecord fields
func (r *DNSRecord) extractAndSetETLDInfo(domain string) error {
	// Get the effective TLD + 1 (the registrable domain)
	etldPlusOne, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return fmt.Errorf("failed to determine ETLD+1 for %s: %w", domain, err)
	}

	// Get just the effective TLD
	etld, icann := publicsuffix.PublicSuffix(domain)
	if etld == "" {
		return fmt.Errorf("failed to determine public suffix for %s", domain)
	}

	// Handle private suffixes (like github.io, s3.amazonaws.com)
	// For DNS purposes, we treat them the same as ICANN domains
	_ = icann // We don't need different logic for private vs ICANN for now

	r.ETLD = etld              // "co.uk", "com", "github.io"
	r.ApexDomain = etldPlusOne // "example.co.uk", "example.com", "user.github.io"

	// Extract subdomain labels (everything before the apex domain)
	r.SubdomainLabels = r.extractSubdomainLabels(domain, etldPlusOne)

	return nil
}

// extractSubdomainLabels gets the labels between the full domain and apex domain
func (r *DNSRecord) extractSubdomainLabels(fullDomain, apexDomain string) []string {
	if fullDomain == apexDomain {
		return []string{} // No subdomains
	}

	// Remove apex domain from end to get subdomain portion
	if strings.HasSuffix(fullDomain, "."+apexDomain) {
		subdomainPortion := strings.TrimSuffix(fullDomain, "."+apexDomain)
		if subdomainPortion == "" {
			return []string{}
		}
		return strings.Split(subdomainPortion, ".")
	}

	// Handle case where fullDomain doesn't have leading dot
	if fullDomain != apexDomain && strings.HasSuffix(fullDomain, apexDomain) {
		prefix := strings.TrimSuffix(fullDomain, apexDomain)
		prefix = strings.TrimSuffix(prefix, ".") // Remove trailing dot if present
		if prefix == "" {
			return []string{}
		}
		return strings.Split(prefix, ".")
	}

	// Fallback - parse labels and subtract apex labels
	labels := strings.Split(fullDomain, ".")
	apexLabels := strings.Split(apexDomain, ".")

	if len(labels) > len(apexLabels) {
		return labels[:len(labels)-len(apexLabels)]
	}

	return []string{}
}

// detectAndSetWildcards detects wildcard patterns and sets wildcard fields
func (r *DNSRecord) detectAndSetWildcards() error {
	// Check if any subdomain labels contain wildcards
	hasWildcard := false
	var wildcardMask uint64

	for i, label := range r.SubdomainLabels {
		if label == "*" {
			hasWildcard = true
			// Set bit for this position (position 0 is leftmost subdomain)
			wildcardMask |= (1 << uint(i))
		} else if strings.Contains(label, "*") {
			// Partial wildcards not supported in this implementation
			return fmt.Errorf("partial wildcard labels not supported: %s", label)
		}
	}

	r.IsWildcard = hasWildcard
	r.WildcardMask = wildcardMask

	return nil
}

// validateLabel validates individual DNS label
func (r *DNSRecord) validateLabel(label string) error {
	// Skip wildcard validation - wildcards are handled separately
	if label == "*" {
		return nil
	}

	if len(label) == 0 || len(label) > 63 {
		return fmt.Errorf("label length invalid: %d characters (must be 1-63)", len(label))
	}

	// Cannot start or end with hyphen
	if label[0] == '-' || label[len(label)-1] == '-' {
		return fmt.Errorf("label cannot start or end with hyphen")
	}

	// Validate characters
	for i, r := range label {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-') {
			return fmt.Errorf("invalid character '%c' at position %d", r, i)
		}
	}

	return nil
}

// validateTLD validates top-level domain requirements
func (r *DNSRecord) validateTLD(tld string) error {
	if len(tld) < 2 {
		return fmt.Errorf("TLD too short: %d characters (minimum 2)", len(tld))
	}

	// TLD cannot be all numeric (RFC 3696)
	allNumeric := true
	for _, r := range tld {
		if r < '0' || r > '9' {
			allNumeric = false
			break
		}
	}
	if allNumeric {
		return fmt.Errorf("TLD cannot be all numeric")
	}

	// TLD must start with a letter
	first := tld[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z')) {
		return fmt.Errorf("TLD must start with a letter")
	}

	return nil
}
