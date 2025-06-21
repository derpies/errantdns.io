#!/bin/bash

# DNS Record Type Tests
# Tests for various DNS record types (CNAME, MX, TXT, NS, etc.)

run_record_tests() {
    print_section "RECORD TYPE TESTS"
    
    # CNAME Record Tests
    run_test "FTP CNAME" "ftp.test.internal" "CNAME" "www.test.internal" "Should point to www"
    run_test "Blog CNAME" "blog.test.internal" "CNAME" "www.test.internal" "Should point to www"
    
    # MX Record Tests
    run_test "MX Records" "test.internal" "MX" "mail.test.internal" "Should show priority 10 first"
    
    # TXT Record Tests
    run_test "SPF Record" "test.internal" "TXT" "spf1" "Should contain SPF record"
    run_test "DMARC Record" "_dmarc.test.internal" "TXT" "DMARC1" "Should contain DMARC policy"
    
    # NS Record Tests
    run_test "NS Records" "test.internal" "NS" "ns" "Should show nameservers"
    
    # TTL Tests
    run_test "Short TTL" "short-ttl.internal" "A" "10.0.1.20" "30 second TTL"
    run_test "Long TTL" "long-ttl.internal" "A" "10.0.1.30" "3600 second TTL"
    run_test "Cache Test" "dns-cache-test.internal" "A" "10.0.2.10" "Cache behavior test"
    
    # Priority Tests
    run_test "Priority Test" "priority-test.internal" "A" "10.0.2.2" "Should return priority 10 records (not priority 20)"
}
