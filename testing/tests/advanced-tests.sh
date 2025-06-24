#!/bin/bash

# Advanced DNS Tests
# Tests for complex functionality like round-robin, load balancing, etc.

run_advanced_tests() {
    print_section "ADVANCED TESTS"
    
    # Round-robin Tests
    run_custom_test "Round-Robin Burst Test" "test_round_robin_burst" "Same IP for rapid queries within time window"
    run_custom_test "Round-Robin Time Rotation" "test_round_robin_rotation" "Different IPs as time boundaries are crossed"
    run_custom_test "Round-Robin Pool Verification" "test_round_robin_pool" "Verify all expected IPs are in rotation"
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
