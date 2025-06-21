// TXT Record Validation
//
// Validates DNS TXT records according to RFC 1035/1123 standards:
// - Max 255 octets per individual string, 65535 total length
// - Supports quoted ("string") and unquoted (string) formats
// - Handles backslash escaping (\") within quoted strings
// - Space/tab separates multiple strings outside quotes
// - Requires valid UTF-8 encoding
// - Empty records allowed
//
// Examples:
//   "v=spf1 include:_spf.google.com ~all"     (single quoted string)
//   key=value "quoted string" other=data      (mixed quoted/unquoted)
//   ""                                        (empty, valid)

package models

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func (r *DNSRecord) validateTXTRecord() error {
	// TXT records can be empty
	if r.Target == "" {
		return nil
	}

	// Check total length - RFC 1035 allows up to 255 octets per string
	// But implementations often allow longer total records
	if len(r.Target) > 65535 {
		return fmt.Errorf("TXT record too long: %d characters (max 65535)", len(r.Target))
	}

	// Parse quoted strings - TXT records are stored as quoted strings
	// Examples: "v=spf1 include:_spf.google.com ~all"
	//          "key=value" "another=string"
	//          "This is a single string"

	var myStrings []string
	var current strings.Builder
	var inQuotes bool
	var escaped bool

	for _, r := range r.Target {
		if escaped {
			// Previous character was backslash, include this character literally
			current.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' {
			// Next character should be escaped
			escaped = true
			current.WriteRune(r)
			continue
		}

		if r == '"' {
			if inQuotes {
				// End of quoted string
				str := current.String()
				if len(str) > 255 {
					return fmt.Errorf("TXT string too long: %d characters (max 255 per string)", len(str))
				}
				myStrings = append(myStrings, str)
				current.Reset()
				inQuotes = false
			} else {
				// Start of quoted string
				inQuotes = true
			}
			continue
		}

		if !inQuotes && (r == ' ' || r == '\t') {
			// Whitespace outside quotes - separator between strings
			if current.Len() > 0 {
				// We have an unquoted string
				str := current.String()
				if len(str) > 255 {
					return fmt.Errorf("TXT string too long: %d characters (max 255 per string)", len(str))
				}
				myStrings = append(myStrings, str)
				current.Reset()
			}
			continue
		}

		// Regular character
		current.WriteRune(r)
	}

	// Handle final string
	if inQuotes {
		return fmt.Errorf("TXT record has unclosed quoted string")
	}

	if current.Len() > 0 {
		str := current.String()
		if len(str) > 255 {
			return fmt.Errorf("TXT string too long: %d characters (max 255 per string)", len(str))
		}
		myStrings = append(myStrings, str)
	}

	// If no strings were parsed, treat entire target as single unquoted string
	if len(myStrings) == 0 && r.Target != "" {
		if len(r.Target) > 255 {
			return fmt.Errorf("TXT string too long: %d characters (max 255 per string)", len(r.Target))
		}
	}

	// Validate character encoding - should be UTF-8
	if !utf8.ValidString(r.Target) {
		return fmt.Errorf("TXT record contains invalid UTF-8 characters")
	}

	return nil
}
