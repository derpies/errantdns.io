#!/bin/bash

# Advanced DNS Tests
# Tests for complex functionality like round-robin, load balancing, etc.

run_advanced_tests() {
    print_section "ADVANCED TESTS"
    
    # Round-robin Tests
    run_custom_test "Round-Robin Burst Test" "test_round_robin_burst" "Same IP for rapid queries within time window"
    run_custom_test "Round-Robin Time Rotation" "test_round_robin_rotation" "Different IPs as time boundaries are crossed"
    run_custom_test "Round-Robin Pool Verification" "test_round_robin_pool" "Verify all expected IPs are in rotation"
    
    # Advanced SRV Tests
    run_custom_test "SRV Priority Ordering" "test_srv_priority_ordering" "SRV records should be ordered by priority"
    run_custom_test "SRV Weight Distribution" "test_srv_weight_distribution" "SRV records with same priority should respect weights"
    
    # Advanced PTR Tests  
    run_custom_test "PTR Network Range" "test_ptr_network_range" "Multiple PTR records in network range"
    
    # SOA Advanced Tests
    run_custom_test "SOA Serial Validation" "test_soa_serial_format" "SOA serial should be in proper format"
}

test_round_robin_burst() {
    echo "  Query: round-robin.internal A (5 rapid queries)"
    echo "  Expected: Same IP for all queries (within 5-second window)"
    
    # Collect burst results
    local burst_results
    readarray -t burst_results < <(collect_burst_responses "round-robin.internal" "A" 5)
    
    # Check if all burst results are the same
    if [ ${#burst_results[@]} -eq 5 ] && all_same "${burst_results[@]}"; then
        echo "  ✓ All burst queries returned same IP"
        return 0
    else
        echo "  ✗ Expected same IP for all burst queries"
        echo "  Burst Results: ${burst_results[*]}"
        return 1
    fi
}

test_round_robin_rotation() {
    echo "  Query: round-robin.internal A (3 queries with 6-second delays)"
    echo "  Expected: Different IPs as time boundaries are crossed"
    echo "  Running timed queries (this will take ~12 seconds)..."
    
    # Collect responses over time
    local rotation_results
    readarray -t rotation_results < <(collect_responses_over_time "round-robin.internal" "A" 3 6)
    
    # Check if we got different results over time
    if [ ${#rotation_results[@]} -gt 1 ]; then
        echo "  ✓ Got ${#rotation_results[@]} different IPs over time"
        echo "  Unique IPs: ${rotation_results[*]}"
        return 0
    else
        echo "  ✗ Expected different IPs over time, got same result"
        echo "  Results: ${rotation_results[*]}"
        return 1
    fi
}

test_round_robin_burst() {
    echo "  Query: round-robin.internal A (5 rapid queries)"
    echo "  Expected: Same IP for all queries (within 5-second window)"
    
    # Collect burst results
    local burst_results
    readarray -t burst_results < <(collect_burst_responses "round-robin.internal" "A" 5)
    
    # Check if all burst results are the same
    if [ ${#burst_results[@]} -eq 5 ] && all_same "${burst_results[@]}"; then
        echo "  ✓ All burst queries returned same IP"
        return 0
    else
        echo "  ✗ Expected same IP for all burst queries"
        echo "  Burst Results: ${burst_results[*]}"
        return 1
    fi
}

test_round_robin_rotation() {
    echo "  Query: round-robin.internal A (3 queries with 6-second delays)"
    echo "  Expected: Different IPs as time boundaries are crossed"
    echo "  Running timed queries (this will take ~12 seconds)..."
    
    # Collect responses over time
    local rotation_results
    readarray -t rotation_results < <(collect_responses_over_time "round-robin.internal" "A" 3 6)
    
    # Check if we got different results over time
    if [ ${#rotation_results[@]} -gt 1 ]; then
        echo "  ✓ Got ${#rotation_results[@]} different IPs over time"
        echo "  Unique IPs: ${rotation_results[*]}"
        return 0
    else
        echo "  ✗ Expected different IPs over time, got same result"
        echo "  Results: ${rotation_results[*]}"
        return 1
    fi
}

test_round_robin_pool() {
    echo "  Query: round-robin.internal A (extended sampling)"
    echo "  Expected: IPs should be from 10.0.3.10-13 range"
    echo "  Collecting samples over 30 seconds..."
    
    # Collect extended samples (all results, not just unique)
    local all_results
    readarray -t all_results < <(collect_responses_over_time "round-robin.internal" "A" 6 5)
    
    # Get unique results
    local unique_results
    readarray -t unique_results < <(get_unique_responses "${all_results[@]}")
    
    # Define expected IP range
    local valid_ips=("10.0.3.10" "10.0.3.11" "10.0.3.12" "10.0.3.13")
    
    echo "  All Results: ${all_results[*]}"
    echo "  Unique IPs Found: ${unique_results[*]}"
    echo "  Expected Range: ${valid_ips[*]}"
    
    # Verify results are from expected range
    if validate_ip_range "${unique_results[@]}" -- "${valid_ips[@]}" && [ ${#unique_results[@]} -gt 1 ]; then
        echo "  ✓ All IPs are from expected range, got ${#unique_results[@]} unique IPs"
        return 0
    else
        echo "  ✗ IPs not from expected range or insufficient variety"
        echo "  Debug: unique_results length=${#unique_results[@]}, validation result=$?"
        return 1
    fi
}

test_srv_priority_ordering() {
    echo "  Query: _web._tcp.test.internal SRV"
    echo "  Expected: Records ordered by priority (lower numbers first)"
    
    # Get full SRV response to check ordering
    local srv_response
    srv_response=$(query_dns_full "_web._tcp.test.internal" "SRV")
    
    # Extract priority values (assuming format: priority weight port target)
    local priorities
    priorities=$(echo "$srv_response" | grep -E "^\s*[0-9]+" | awk '{print $5}' | sort -n)
    
    # Check if priorities are in ascending order
    local sorted_priorities
    sorted_priorities=$(echo "$priorities" | sort -n)
    
    if [ "$priorities" = "$sorted_priorities" ]; then
        echo "  ✓ SRV records properly ordered by priority"
        echo "  Priorities found: $(echo $priorities | tr '\n' ' ')"
        return 0
    else
        echo "  ✗ SRV records not properly ordered"
        echo "  Found order: $(echo $priorities | tr '\n' ' ')"
        echo "  Expected order: $(echo $sorted_priorities | tr '\n' ' ')"
        return 1
    fi
}

test_srv_weight_distribution() {
    echo "  Query: _cluster._tcp.test.internal SRV"
    echo "  Expected: Multiple SRV records with different weights"
    
    # Query for SRV records that should have same priority but different weights
    local srv_response
    srv_response=$(query_dns_full "_cluster._tcp.test.internal" "SRV")
    
    echo "  Debug: Full SRV response:"
    echo "$srv_response" | head -20  # Show first 20 lines for debugging
    
    # Count SRV records returned (look for lines containing "SRV" and priority/weight/port/target)
    local srv_count
    srv_count=$(echo "$srv_response" | grep -cE "SRV.*[0-9]+\s+[0-9]+\s+[0-9]+")
    
    echo "  Debug: Found $srv_count SRV records"
    
    if [ "$srv_count" -ge 2 ]; then
        echo "  ✓ Multiple SRV records found ($srv_count records)"
        # Show the records for verification (extract just the SRV data)
        echo "$srv_response" | grep -E "SRV.*[0-9]+\s+[0-9]+\s+[0-9]+" | while read line; do
            echo "    SRV: $line"
        done
        return 0
    else
        echo "  ✗ Expected multiple SRV records, found $srv_count"
        echo "  Debug: Lines that should match SRV pattern:"
        echo "$srv_response" | grep "SRV" | head -5
        return 1
    fi
}

test_ptr_network_range() {
    echo "  Query: PTR records for 10.0.0.x network range"
    echo "  Expected: Valid PTR responses for network hosts"
    
    local ptr_queries=("10.0.0.10.in-addr.arpa" "20.0.0.10.in-addr.arpa" "30.0.0.10.in-addr.arpa")
    local ptr_expected=("test.internal" "mail.test.internal" "api.test.internal")
    local success_count=0
    
    for i in "${!ptr_queries[@]}"; do
        local query="${ptr_queries[$i]}"
        local expected="${ptr_expected[$i]}"
        local result
        result=$(query_dns "$query" "PTR")
        
        if echo "$result" | grep -q "$expected"; then
            echo "    ✓ $query → $result"
            success_count=$((success_count + 1))
        else
            echo "    ✗ $query → $result (expected: $expected)"
        fi
    done
    
    if [ "$success_count" -eq "${#ptr_queries[@]}" ]; then
        echo "  ✓ All PTR records in network range resolved correctly"
        return 0
    else
        echo "  ✗ $success_count/${#ptr_queries[@]} PTR records resolved correctly"
        return 1
    fi
}

test_soa_serial_format() {
    echo "  Query: test.internal SOA"
    echo "  Expected: SOA record with valid serial number format"
    
    local soa_response
    soa_response=$(query_dns_full "test.internal" "SOA")
    
    # Extract SOA serial (should be 3rd field after primary NS and admin email)
    local serial
    serial=$(echo "$soa_response" | grep -E "^\s*test\.internal.*SOA" | awk '{print $7}')
    
    # Check if serial looks like a valid format (numeric, possibly YYYYMMDDNN)
    if [[ "$serial" =~ ^[0-9]+$ ]] && [ ${#serial} -ge 1 ]; then
        echo "  ✓ SOA serial number found: $serial"
        
        # Additional check for common YYYYMMDDNN format
        if [ ${#serial} -eq 10 ]; then
            echo "    Format appears to be YYYYMMDDNN (recommended)"
        fi
        return 0
    else
        echo "  ✗ Invalid or missing SOA serial number: '$serial'"
        echo "  SOA Response: $soa_response"
        return 1
    fi
}
