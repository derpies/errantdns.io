#!/bin/bash

# DNS Server Test Script for ErrantDNS
# Usage: ./test-dns.sh [port]

PORT=${1:-10001}
SERVER="127.0.0.1"
TIMEOUT=5

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
TESTS_RUN=0
TESTS_PASSED=0

# Helper function to run a DNS test
run_test() {
    local test_name="$1"
    local domain="$2"
    local record_type="$3"
    local expected="$4"
    local description="$5"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    echo -e "${BLUE}[TEST $TESTS_RUN]${NC} $test_name"
    echo "  Query: $domain $record_type"
    echo "  Expected: $expected"
    
    # Run dig command with timeout
    result=$(dig @$SERVER -p $PORT +short +time=$TIMEOUT $domain $record_type 2>/dev/null)
    exit_code=$?
    
    if [ $exit_code -ne 0 ]; then
        echo -e "  ${RED}✗ FAILED${NC} - DNS query failed (timeout or error)"
        echo "  Result: No response"
    elif [ -z "$result" ]; then
        if [ "$expected" = "NXDOMAIN" ]; then
            echo -e "  ${GREEN}✓ PASSED${NC} - Got expected NXDOMAIN"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            echo -e "  ${RED}✗ FAILED${NC} - No result returned"
            echo "  Result: (empty)"
        fi
    else
        # Check if result matches expected (basic string matching)
        if echo "$result" | grep -q "$expected" || [ "$expected" = "ANY" ]; then
            echo -e "  ${GREEN}✓ PASSED${NC} - Got expected result"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            echo -e "  ${RED}✗ FAILED${NC} - Result doesn't match expected"
        fi
        echo "  Result: $result"
    fi
    
    if [ -n "$description" ]; then
        echo "  Note: $description"
    fi
    echo
}

# Helper function for negative tests (expecting NXDOMAIN)
run_negative_test() {
    local test_name="$1"
    local domain="$2"
    local record_type="$3"
    local description="$4"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    echo -e "${BLUE}[TEST $TESTS_RUN]${NC} $test_name"
    echo "  Query: $domain $record_type"
    echo "  Expected: NXDOMAIN or no result"
    
    # Run dig command with full output to check NXDOMAIN
    result=$(dig @$SERVER -p $PORT +time=$TIMEOUT $domain $record_type 2>/dev/null | grep "status:")
    short_result=$(dig @$SERVER -p $PORT +short +time=$TIMEOUT $domain $record_type 2>/dev/null)
    
    if echo "$result" | grep -q "NXDOMAIN" || [ -z "$short_result" ]; then
        echo -e "  ${GREEN}✓ PASSED${NC} - Got expected NXDOMAIN/no result"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "  ${RED}✗ FAILED${NC} - Expected NXDOMAIN but got result"
        echo "  Result: $short_result"
    fi
    
    if [ -n "$description" ]; then
        echo "  Note: $description"
    fi
    echo
}

echo "======================================"
echo "ErrantDNS Server Test Suite"
echo "======================================"
echo "Server: $SERVER:$PORT"
echo "Timeout: ${TIMEOUT}s"
echo

# Test if server is reachable
echo -e "${YELLOW}[SETUP]${NC} Testing server connectivity..."
ping_result=$(dig @$SERVER -p $PORT +short +time=2 test.internal A 2>/dev/null)
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Server is responding${NC}"
else
    echo -e "${RED}✗ Server is not responding - check if DNS server is running on port $PORT${NC}"
    exit 1
fi
echo

# A Record Tests
echo -e "${YELLOW}=== A RECORD TESTS ===${NC}"
run_test "Basic A Record" "test.internal" "A" "10.0.0.10" "Main test domain"
run_test "WWW A Record" "www.test.internal" "A" "10.0.0.10" "Should match main domain"
run_test "Mail A Record" "mail.test.internal" "A" "10.0.0.20" "Mail server"
run_test "API A Record" "api.test.internal" "A" "10.0.0.30" "API server"

# AAAA Record Tests (IPv6)
echo -e "${YELLOW}=== AAAA RECORD TESTS ===${NC}"
run_test "IPv6 A Record" "test.internal" "AAAA" "fd00::1" "IPv6 for main domain"
run_test "IPv6 WWW Record" "www.test.internal" "AAAA" "fd00::1" "IPv6 for www"

# CNAME Record Tests
echo -e "${YELLOW}=== CNAME RECORD TESTS ===${NC}"
run_test "FTP CNAME" "ftp.test.internal" "CNAME" "www.test.internal" "Should point to www"
run_test "Blog CNAME" "blog.test.internal" "CNAME" "www.test.internal" "Should point to www"

# MX Record Tests
echo -e "${YELLOW}=== MX RECORD TESTS ===${NC}"
run_test "MX Records" "test.internal" "MX" "mail.test.internal" "Should show priority 10 first"

# TXT Record Tests
echo -e "${YELLOW}=== TXT RECORD TESTS ===${NC}"
run_test "SPF Record" "test.internal" "TXT" "spf1" "Should contain SPF record"
run_test "DMARC Record" "_dmarc.test.internal" "TXT" "DMARC1" "Should contain DMARC policy"

# NS Record Tests
echo -e "${YELLOW}=== NS RECORD TESTS ===${NC}"
run_test "NS Records" "test.internal" "NS" "ns1.test.internal" "Should show nameservers"

# TTL Tests
echo -e "${YELLOW}=== TTL TESTS ===${NC}"
run_test "Short TTL" "short-ttl.internal" "A" "10.0.1.20" "30 second TTL"
run_test "Long TTL" "long-ttl.internal" "A" "10.0.1.30" "3600 second TTL"
run_test "Cache Test" "dns-cache-test.internal" "A" "10.0.2.10" "Cache behavior test"

# Priority Tests
echo -e "${YELLOW}=== PRIORITY TESTS ===${NC}"
run_test "Priority Test" "priority-test.internal" "A" "10.0.2.20" "Should return highest priority (100)"

# Negative Tests (should fail)
echo -e "${YELLOW}=== NEGATIVE TESTS ===${NC}"
run_negative_test "Non-existent Domain" "does-not-exist.internal" "A" "Should return NXDOMAIN"
run_negative_test "Non-existent Subdomain" "missing.test.internal" "A" "Should return NXDOMAIN"
run_negative_test "Wrong Record Type" "test.internal" "PTR" "PTR not configured for this domain"

# Protocol Tests
echo -e "${YELLOW}=== PROTOCOL TESTS ===${NC}"
echo -e "${BLUE}[TEST $((TESTS_RUN + 1))]${NC} UDP vs TCP Test"
TESTS_RUN=$((TESTS_RUN + 1))
udp_result=$(dig @$SERVER -p $PORT +short +notcp +time=$TIMEOUT test.internal A 2>/dev/null)
tcp_result=$(dig @$SERVER -p $PORT +short +tcp +time=$TIMEOUT test.internal A 2>/dev/null)

if [ "$udp_result" = "$tcp_result" ] && [ -n "$udp_result" ]; then
    echo -e "  ${GREEN}✓ PASSED${NC} - UDP and TCP return same result"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "  ${RED}✗ FAILED${NC} - UDP and TCP results differ"
fi
echo "  UDP Result: $udp_result"
echo "  TCP Result: $tcp_result"
echo

# Summary
echo "======================================"
echo "TEST SUMMARY"
echo "======================================"
echo "Tests Run: $TESTS_RUN"
echo "Tests Passed: $TESTS_PASSED"
echo "Tests Failed: $((TESTS_RUN - TESTS_PASSED))"

if [ $TESTS_PASSED -eq $TESTS_RUN ]; then
    echo -e "${GREEN}✓ ALL TESTS PASSED!${NC}"
    exit 0
else
    echo -e "${RED}✗ SOME TESTS FAILED${NC}"
    echo "Success Rate: $(( (TESTS_PASSED * 100) / TESTS_RUN ))%"
    exit 1
fi
