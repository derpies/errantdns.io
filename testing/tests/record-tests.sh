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
    
    # SOA Record Tests
    run_test "Zone SOA Record" "test.internal" "SOA" "ns1.test.internal" "Should return SOA with primary NS"
    run_test "Subdomain SOA Query" "www.test.internal" "SOA" "ns1.test.internal" "Should return parent zone SOA"
    run_test "Deep Subdomain SOA" "api.v1.test.internal" "SOA" "ns1.test.internal" "Should return zone SOA for deep subdomain"
    
    # PTR Record Tests (Reverse DNS)
    run_test "IPv4 PTR Record" "10.0.0.10.in-addr.arpa" "PTR" "test.internal" "Reverse lookup for test.internal"
    run_test "Mail Server PTR" "20.0.0.10.in-addr.arpa" "PTR" "mail.test.internal" "Reverse lookup for mail server"
    run_test "API Server PTR" "30.0.0.10.in-addr.arpa" "PTR" "api.test.internal" "Reverse lookup for API server"
    run_test "IPv6 PTR Record" "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.d.f.ip6.arpa" "PTR" "test.internal" "IPv6 reverse lookup"
    
    # SRV Record Tests (Service Discovery)
    run_test "HTTP Service" "_http._tcp.test.internal" "SRV" "web1.test.internal,web2.test.internal" "HTTP service discovery"
    run_test "HTTPS Service" "_https._tcp.test.internal" "SRV" "www.test.internal" "HTTPS service discovery"
    run_test "SMTP Service" "_smtp._tcp.test.internal" "SRV" "mail.test.internal" "SMTP service discovery"
    run_test "IMAP Service" "_imap._tcp.test.internal" "SRV" "mail.test.internal" "IMAP service discovery"
    run_test "SIP Service" "_sip._tcp.test.internal" "SRV" "sip.test.internal" "SIP service discovery"
    run_test "LDAP Service" "_ldap._tcp.test.internal" "SRV" "ldap.test.internal" "LDAP service discovery"
    run_test "Multiple SRV Priorities" "_web._tcp.test.internal" "SRV" "10" "Multiple SRV records with different priorities"
    
    # TTL Tests
    run_test "Short TTL" "short-ttl.internal" "A" "10.0.1.20" "30 second TTL"
    run_test "Long TTL" "long-ttl.internal" "A" "10.0.1.30" "3600 second TTL"
    run_test "Cache Test" "dns-cache-test.internal" "A" "10.0.2.10" "Cache behavior test"
    
    # Priority Tests
    run_test "Priority Test" "priority-test.internal" "A" "10.0.2.2" "Should return priority 10 records (not priority 20)"
}
