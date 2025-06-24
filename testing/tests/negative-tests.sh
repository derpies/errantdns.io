#!/bin/bash

# Negative DNS Tests
# Tests that should fail or return NXDOMAIN

run_negative_tests() {
    print_section "NEGATIVE TESTS"
    
    # Basic negative tests
    run_negative_test "Non-existent Domain" "does-not-exist.internal" "A" "Should return NXDOMAIN"
    run_negative_test "Non-existent Subdomain" "missing.test.internal" "A" "Should return NXDOMAIN"
    run_negative_test "Wrong Record Type" "test.internal" "HINFO" "HINFO not configured for this domain"
    
    # SOA Negative Tests
    run_negative_test "Non-existent Zone SOA" "nonexistent.zone" "SOA" "Zone does not exist"
    run_negative_test "Invalid Zone SOA" "invalid.zone" "SOA" "Malformed domain name"
    
    # PTR Negative Tests  
    run_negative_test "Invalid PTR Format" "999.999.999.999.in-addr.arpa" "PTR" "Invalid IP address format"
    run_negative_test "Non-existent PTR" "1.1.1.1.in-addr.arpa" "PTR" "No PTR record for this IP"
    run_negative_test "Malformed PTR Query" "not-an-ip.in-addr.arpa" "PTR" "Malformed reverse lookup"
    run_negative_test "IPv6 PTR Non-existent" "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa" "PTR" "No IPv6 PTR record"
    
    # SRV Negative Tests
    run_negative_test "Non-existent Service" "_unknown._tcp.test.internal" "SRV" "Service not configured"
    run_negative_test "Wrong Protocol SRV" "_http._udp.test.internal" "SRV" "Protocol not supported for this service"
    run_negative_test "Malformed SRV Query" "_invalid-service._tcp.test.internal" "SRV" "Invalid service name format"
    run_negative_test "SRV on Non-existent Domain" "_http._tcp.nonexistent.zone" "SRV" "Domain does not exist"
}
