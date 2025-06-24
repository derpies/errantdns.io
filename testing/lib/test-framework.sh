#!/bin/bash

# Test Framework Core Module
# Provides test execution, reporting, and utility functions

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Global test counters
TESTS_RUN=0
TESTS_PASSED=0

# Configuration variables (set by init_test_framework)
DNS_SERVER=""
DNS_PORT=""
DNS_TIMEOUT=""

# Initialize the test framework
init_test_framework() {
    DNS_SERVER="$1"
    DNS_PORT="$2"
    DNS_TIMEOUT="$3"
}

# Print colored header
print_header() {
    local title="$1"
    echo "======================================"
    echo "$title"
    echo "======================================"
}

# Print section header
print_section() {
    local section="$1"
    echo -e "${YELLOW}=== $section ===${NC}"
}

# Core test execution function
run_test() {
    local test_name="$1"
    local domain="$2"
    local record_type="$3"
    local expected="$4"
    local description="${5:-}"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    echo -e "${BLUE}[TEST $TESTS_RUN]${NC} $test_name"
    echo "  Query: $domain $record_type"
    echo "  Expected: $expected"
    
    # Run dig command with timeout
    local result
    result=$(dig @"$DNS_SERVER" -p "$DNS_PORT" +short +time="$DNS_TIMEOUT" "$domain" "$record_type" 2>/dev/null)
    local exit_code=$?
    
    if [ $exit_code -ne 0 ]; then
        print_test_failure "DNS query failed (timeout or error)" ""
    elif [ -z "$result" ]; then
        if [ "$expected" = "NXDOMAIN" ]; then
            print_test_success "Got expected NXDOMAIN"
        else
            print_test_failure "No result returned" "(empty)"
        fi
    else
        # Check if result matches any of the expected values
        if check_expected_match "$result" "$expected"; then
            print_test_success "Got expected result"
        else
            print_test_failure "Result doesn't match expected" "$result"
        fi
        echo "  Result: $result"
    fi
    
    if [ -n "$description" ]; then
        echo "  Note: $description"
    fi
    echo
}

# Run negative test (expecting NXDOMAIN)
run_negative_test() {
    local test_name="$1"
    local domain="$2"
    local record_type="$3"
    local description="${4:-}"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    echo -e "${BLUE}[TEST $TESTS_RUN]${NC} $test_name"
    echo "  Query: $domain $record_type"
    echo "  Expected: NXDOMAIN or no result"
    
    # Run dig command with full output to check NXDOMAIN
    local result
    result=$(dig @"$DNS_SERVER" -p "$DNS_PORT" +time="$DNS_TIMEOUT" "$domain" "$record_type" 2>/dev/null | grep "status:")
    local short_result
    short_result=$(dig @"$DNS_SERVER" -p "$DNS_PORT" +short +time="$DNS_TIMEOUT" "$domain" "$record_type" 2>/dev/null)
    
    if echo "$result" | grep -q "NXDOMAIN" || [ -z "$short_result" ]; then
        print_test_success "Got expected NXDOMAIN/no result"
    else
        print_test_failure "Expected NXDOMAIN but got result" "$short_result"
    fi
    
    if [ -n "$description" ]; then
        echo "  Note: $description"
    fi
    echo
}

# Custom test function for complex scenarios
run_custom_test() {
    local test_name="$1"
    local test_function="$2"
    local description="${3:-}"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    echo -e "${BLUE}[TEST $TESTS_RUN]${NC} $test_name"
    
    if [ -n "$description" ]; then
        echo "  Description: $description"
    fi
    
    # Execute the custom test function
    if "$test_function"; then
        print_test_success "Custom test passed"
    else
        print_test_failure "Custom test failed" ""
    fi
    
    echo
}

# Print test success
print_test_success() {
    local message="$1"
    echo -e "  ${GREEN}✓ PASSED${NC} - $message"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

# Print test failure
print_test_failure() {
    local message="$1"
    local result="$2"
    echo -e "  ${RED}✗ FAILED${NC} - $message"
    if [ -n "$result" ]; then
        echo "  Result: $result"
    fi
}

# Check server connectivity
check_server_connectivity() {
    echo -e "${YELLOW}[SETUP]${NC} Testing server connectivity..."
    
    local ping_result
    if ping_result=$(dig @"$DNS_SERVER" -p "$DNS_PORT" +short +time=2 test.internal A 2>/dev/null); then
        echo -e "${GREEN}✓ Server is responding${NC}"
        echo
        return 0
    else
        echo -e "${RED}✗ Server is not responding - check if DNS server is running on port $DNS_PORT${NC}"
        echo
        return 1
    fi
}

# Check if result matches any of the expected values
check_expected_match() {
    local result="$1"
    local expected="$2"
    
    # Handle special cases
    if [ "$expected" = "ANY" ]; then
        return 0
    fi
    
    if [ "$expected" = "NXDOMAIN" ]; then
        return 1  # This should be handled separately
    fi
    
    # Split expected values by comma and check each one
    local IFS=','
    local expected_array=($expected)
    
    for expected_value in "${expected_array[@]}"; do
        # Trim whitespace from expected value
        expected_value=$(echo "$expected_value" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        
        # Check if the result contains this expected value
        if echo "$result" | grep -q "$expected_value"; then
            return 0
        fi
    done
    
    return 1
}

# Print final test summary
print_summary() {
    print_header "TEST SUMMARY"
    echo "Tests Run: $TESTS_RUN"
    echo "Tests Passed: $TESTS_PASSED"
    echo "Tests Failed: $((TESTS_RUN - TESTS_PASSED))"

    if [ $TESTS_PASSED -eq $TESTS_RUN ]; then
        echo -e "${GREEN}✓ ALL TESTS PASSED!${NC}"
        return 0
    else
        echo -e "${RED}✗ SOME TESTS FAILED${NC}"
        echo "Success Rate: $(( (TESTS_PASSED * 100) / TESTS_RUN ))%"
        return 1
    fi
}