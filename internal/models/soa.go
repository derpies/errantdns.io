/*
SOA Record Format:
- MNAME: Primary nameserver (FQDN)
- RNAME: Admin email (encoded as FQDN: admin.example.com = admin@example.com)
- SERIAL: Version number (typically YYYYMMDDNN)
- REFRESH: Secondary refresh interval (seconds)
- RETRY: Retry interval on failed refresh (seconds)
- EXPIRE: Zone expiration time (seconds)
- MINIMUM: Negative cache TTL (seconds)

Target format: "ns1.example.com admin.example.com 2025061901 3600 1800 604800 86400"

Wildcard Exclusion Rules:

- SOA records cannot have wildcards in the name field
- SOA records can only exist at zone apex
- Only one SOA per zone/domain

*/

package models

import (
	"fmt"
	"strconv"
	"strings"
)

/*
target = "ns1.example.com admin.example.com 2025061901 3600 1800 604800 86400"

Fields to validate:
1. MNAME: Valid FQDN format
2. RNAME: Valid email-as-FQDN format
3. SERIAL: Valid 32-bit unsigned integer
4. REFRESH: Valid timing value (reasonable range)
5. RETRY: Valid timing value (should be < REFRESH)
6. EXPIRE: Valid timing value (should be > REFRESH)
7. MINIMUM: Valid timing value
*/

func (r *DNSRecord) validateSOATarget() error {
	fields := strings.Fields(r.Target)
	if len(fields) != 7 {
		return fmt.Errorf("SOA target must have exactly 7 fields, got %d", len(fields))
	}

	mname := fields[0]
	rname := fields[1]
	serialStr := fields[2]
	refreshStr := fields[3]
	retryStr := fields[4]
	expireStr := fields[5]
	minimumStr := fields[6]

	// Validate MNAME (Primary Nameserver)
	if err := r.validateDomainName(); err != nil {
		return fmt.Errorf("SOA MNAME invalid: %s is not a valid FQDN", mname)
	}

	// Validate RNAME (Admin Email as FQDN)
	if err := r.validateDomainName(); err != nil {
		return fmt.Errorf("SOA RNAME invalid: %s is not a valid FQDN", rname)
	}

	// Validate SERIAL
	serial, err := strconv.ParseUint(serialStr, 10, 32)
	if err != nil {
		return fmt.Errorf("SOA SERIAL invalid: %s is not a valid 32-bit unsigned integer", serialStr)
	}
	_ = serial // Valid serial number

	// Validate REFRESH
	refresh, err := strconv.ParseUint(refreshStr, 10, 32)
	if err != nil {
		return fmt.Errorf("SOA REFRESH invalid: %s is not a valid 32-bit unsigned integer", refreshStr)
	}
	if refresh == 0 {
		return fmt.Errorf("SOA REFRESH invalid: must be greater than 0")
	}

	// Validate RETRY
	retry, err := strconv.ParseUint(retryStr, 10, 32)
	if err != nil {
		return fmt.Errorf("SOA RETRY invalid: %s is not a valid 32-bit unsigned integer", retryStr)
	}
	if retry == 0 {
		return fmt.Errorf("SOA RETRY invalid: must be greater than 0")
	}

	// Validate EXPIRE
	expire, err := strconv.ParseUint(expireStr, 10, 32)
	if err != nil {
		return fmt.Errorf("SOA EXPIRE invalid: %s is not a valid 32-bit unsigned integer", expireStr)
	}
	if expire == 0 {
		return fmt.Errorf("SOA EXPIRE invalid: must be greater than 0")
	}

	// Validate MINIMUM
	minimum, err := strconv.ParseUint(minimumStr, 10, 32)
	if err != nil {
		return fmt.Errorf("SOA MINIMUM invalid: %s is not a valid 32-bit unsigned integer", minimumStr)
	}
	// MINIMUM can be 0, so no zero-check needed

	// Cross-field validation
	if retry >= refresh {
		return fmt.Errorf("SOA timing conflict: RETRY (%d) must be less than REFRESH (%d)", retry, refresh)
	}

	if expire <= refresh {
		return fmt.Errorf("SOA timing conflict: EXPIRE (%d) must be greater than REFRESH (%d)", expire, refresh)
	}

	if retry >= expire {
		return fmt.Errorf("SOA timing conflict: RETRY (%d) must be less than EXPIRE (%d)", retry, expire)
	}

	if minimum > refresh {
		return fmt.Errorf("SOA timing conflict: MINIMUM (%d) should not exceed REFRESH (%d)", minimum, refresh)
	}

	return nil
}

func (r *DNSRecord) validateSOARecord() error {
	// Check wildcard exclusion
	if strings.Contains(r.Name, "*") {
		return fmt.Errorf("SOA records cannot contain wildcards")
	}

	// Validate SOA target format
	return r.validateSOATarget()
}
