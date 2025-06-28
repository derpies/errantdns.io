// CAA Record Validation
//
// Validates DNS CAA records according to RFC 8659 standards:
// - Flag must be 0 (non-critical) or 128 (critical)
// - Tag must be one of: issue, issuewild, iodef
// - Value format depends on tag type:
//   - issue/issuewild: CA domain name or ";" (deny all)
//   - iodef: mailto: URL or https: URL
//
// - Tag names are case-insensitive but stored lowercase
// - Value cannot be empty
//
// Examples:
// Flag: 0, Tag: "issue", Value: "letsencrypt.org" (valid)
// Flag: 0, Tag: "iodef", Value: "mailto:admin@example.com" (valid)
// Flag: 128, Tag: "issue", Value: ";" (valid - deny all)
// Flag: 0, Tag: "invalid", Value: "test" (invalid tag)
// Flag: 255, Tag: "issue", Value: "ca.com" (invalid flag)
package models

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

func (r *DNSRecord) validateCAARecord() error {
	// CAA records use Priority field as Flag (0 or 128)
	flag := r.Priority
	if flag != 0 && flag != 128 {
		return fmt.Errorf("CAA record flag must be 0 (non-critical) or 128 (critical), got: %d", flag)
	}

	// Tag cannot be empty and must be valid
	if r.Tag == "" {
		return fmt.Errorf("CAA record tag cannot be empty")
	}

	// Normalize tag to lowercase for validation
	tag := strings.ToLower(strings.TrimSpace(r.Tag))

	// Valid CAA tags according to RFC 8659
	validTags := map[string]bool{
		"issue":     true,
		"issuewild": true,
		"iodef":     true,
	}

	if !validTags[tag] {
		return fmt.Errorf("CAA record tag must be 'issue', 'issuewild', or 'iodef', got: %s", r.Tag)
	}

	// Value cannot be empty
	if r.Target == "" {
		return fmt.Errorf("CAA record value cannot be empty")
	}

	// Validate value based on tag type
	switch tag {
	case "issue", "issuewild":
		return r.validateCAAIssueValue()
	case "iodef":
		return r.validateCAAIodefValue()
	}

	return nil
}

// validateCAAIssueValue validates issue/issuewild CAA record values
func (r *DNSRecord) validateCAAIssueValue() error {
	value := strings.TrimSpace(r.Target)

	// ";" means "no CA is authorized" - this is valid
	if value == ";" {
		return nil
	}

	// Empty value after trimming is not allowed
	if value == "" {
		return fmt.Errorf("CAA issue/issuewild value cannot be empty (use ';' to deny all)")
	}

	// Basic domain name validation for CA domain
	// Should be a valid domain name (CA's domain)
	if err := r.validateDomainNameOther(value); err != nil {
		return fmt.Errorf("CAA issue/issuewild value must be valid CA domain name: %w", err)
	}

	// Additional validation: should not contain protocol or path
	if strings.Contains(value, "://") {
		return fmt.Errorf("CAA issue/issuewild value should be domain name only, not URL: %s", value)
	}

	// Should not contain spaces
	if strings.Contains(value, " ") {
		return fmt.Errorf("CAA issue/issuewild value cannot contain spaces: %s", value)
	}

	return nil
}

// validateCAAIodefValue validates iodef CAA record values
func (r *DNSRecord) validateCAAIodefValue() error {
	value := strings.TrimSpace(r.Target)

	if value == "" {
		return fmt.Errorf("CAA iodef value cannot be empty")
	}

	// Must be either mailto: or https: URL
	if strings.HasPrefix(value, "mailto:") {
		return r.validateCAAMailto(value)
	} else if strings.HasPrefix(value, "https://") {
		return r.validateCAAHttps(value)
	} else {
		return fmt.Errorf("CAA iodef value must start with 'mailto:' or 'https://', got: %s", value)
	}
}

// validateCAAMailto validates mailto: URLs in CAA iodef records
func (r *DNSRecord) validateCAAMailto(mailto string) error {
	// Parse the mailto URL
	parsedURL, err := url.Parse(mailto)
	if err != nil {
		return fmt.Errorf("CAA iodef mailto URL is invalid: %w", err)
	}

	if parsedURL.Scheme != "mailto" {
		return fmt.Errorf("CAA iodef URL scheme must be mailto, got: %s", parsedURL.Scheme)
	}

	// Extract email address (everything after mailto:)
	email := parsedURL.Opaque
	if email == "" {
		return fmt.Errorf("CAA iodef mailto URL missing email address")
	}

	// Basic email validation
	if err := r.validateEmailAddress(email); err != nil {
		return fmt.Errorf("CAA iodef mailto contains invalid email: %w", err)
	}

	return nil
}

// validateCAAHttps validates https: URLs in CAA iodef records
func (r *DNSRecord) validateCAAHttps(httpsURL string) error {
	// Parse the HTTPS URL
	parsedURL, err := url.Parse(httpsURL)
	if err != nil {
		return fmt.Errorf("CAA iodef HTTPS URL is invalid: %w", err)
	}

	if parsedURL.Scheme != "https" {
		return fmt.Errorf("CAA iodef URL scheme must be https, got: %s", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("CAA iodef HTTPS URL missing hostname")
	}

	// Basic hostname validation
	if err := r.validateDomainNameOther(parsedURL.Host); err != nil {
		return fmt.Errorf("CAA iodef HTTPS URL has invalid hostname: %w", err)
	}

	return nil
}

// validateEmailAddress performs basic email validation
func (r *DNSRecord) validateEmailAddress(email string) error {
	if email == "" {
		return fmt.Errorf("email address cannot be empty")
	}

	// Basic email regex - not RFC 5322 compliant but good enough for CAA
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format: %s", email)
	}

	return nil
}
