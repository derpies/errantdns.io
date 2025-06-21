#!/bin/bash

# Basic DNS Tests
# Tests for fundamental DNS record types and basic functionality

run_basic_tests() {
    print_section "BASIC TESTS"
    
    # A Record Tests
    run_test "Basic A Record" "test.internal" "A" "10.0.0.10" "Main test domain"
    run_test "WWW A Record" "www.test.internal" "A" "10.0.0.10" "Should match main domain"
    run_test "Mail A Record" "mail.test.internal" "A" "10.0.0.20" "Mail server"
    run_test "API A Record" "api.test.internal" "A" "10.0.0.30" "API server"
    
    # AAAA Record Tests (IPv6)
    run_test "IPv6 A Record" "test.internal" "AAAA" "fd00::1" "IPv6 for main domain"
    run_test "IPv6 WWW Record" "www.test.internal" "AAAA" "fd00::1" "IPv6 for www"
    
    # Protocol consistency test
    run_custom_test "UDP vs TCP Consistency" "test_basic_protocol_consistency" "Both protocols should return same results"
}

test_basic_protocol_consistency() {
    test_protocol_consistency "test.internal" "A"
}
