#!/bin/bash

# Negative DNS Tests
# Tests that should fail or return NXDOMAIN

run_negative_tests() {
    print_section "NEGATIVE TESTS"
    
    run_negative_test "Non-existent Domain" "does-not-exist.internal" "A" "Should return NXDOMAIN"
    run_negative_test "Non-existent Subdomain" "missing.test.internal" "A" "Should return NXDOMAIN"
    run_negative_test "Wrong Record Type" "test.internal" "PTR" "PTR not configured for this domain"
}
