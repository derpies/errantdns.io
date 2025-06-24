#!/bin/bash

# Test Helper Functions
# Specialized helper functions for DNS testing

# Query DNS and return result
query_dns() {
    local domain="$1"
    local record_type="$2"
    local use_tcp="${3:-false}"
    
    local tcp_flag=""
    if [ "$use_tcp" = "true" ]; then
        tcp_flag="+tcp"
    else
        tcp_flag="+notcp"
    fi
    
    dig @"$DNS_SERVER" -p "$DNS_PORT" +short $tcp_flag +time="$DNS_TIMEOUT" "$domain" "$record_type" 2>/dev/null
}

# Query DNS with full output
query_dns_full() {
    local domain="$1"
    local record_type="$2"
    
    dig @"$DNS_SERVER" -p "$DNS_PORT" +time="$DNS_TIMEOUT" "$domain" "$record_type" 2>/dev/null
}

# Test UDP vs TCP consistency
test_protocol_consistency() {
    local domain="$1"
    local record_type="$2"
    
    local udp_result tcp_result
    udp_result=$(query_dns "$domain" "$record_type" false)
    tcp_result=$(query_dns "$domain" "$record_type" true)
    
    echo "  UDP Result: $udp_result"
    echo "  TCP Result: $tcp_result"
    
    [ "$udp_result" = "$tcp_result" ] && [ -n "$udp_result" ]
}

# Collect multiple DNS responses over time
collect_responses_over_time() {
    local domain="$1"
    local record_type="$2"
    local count="$3"
    local delay="$4"
    
    local results=()
    for ((i=1; i<=count; i++)); do
        local result
        result=$(query_dns "$domain" "$record_type")
        if [ -n "$result" ]; then
            results+=("$result")
            echo "    Query $i: $result ($(date +%H:%M:%S))" >&2
        fi
        
        if [ $i -lt $count ]; then
            echo "    Waiting ${delay} seconds..." >&2
            sleep "$delay"
        fi
    done
    
    # Return all results (not just unique ones)
    printf '%s\n' "${results[@]}"
}

# Get unique results from responses
get_unique_responses() {
    local results=("$@")
    printf '%s\n' "${results[@]}" | sort -u
}

# Collect burst responses (rapid queries)
collect_burst_responses() {
    local domain="$1"
    local record_type="$2"
    local count="$3"
    
    local results=()
    for ((i=1; i<=count; i++)); do
        local result
        result=$(query_dns "$domain" "$record_type" | head -1)
        if [ -n "$result" ]; then
            results+=("$result")
        fi
    done
    
    printf '%s\n' "${results[@]}"
}

# Check if all elements in array are the same
all_same() {
    local -a arr=("$@")
    local first="${arr[0]}"
    
    for element in "${arr[@]}"; do
        if [ "$element" != "$first" ]; then
            return 1
        fi
    done
    
    return 0
}

# Get unique elements from array
get_unique() {
    local -a arr=("$@")
    printf '%s\n' "${arr[@]}" | sort -u
}

# Check if IP is in expected range
ip_in_range() {
    local ip="$1"
    shift
    local -a valid_ips=("$@")
    
    for valid_ip in "${valid_ips[@]}"; do
        if [ "$ip" = "$valid_ip" ]; then
            return 0
        fi
    done
    
    return 1
}

# Validate all IPs are in expected range
validate_ip_range() {
    local -a test_ips=()
    local -a valid_ips=()
    
    # Parse arguments - first part is test IPs, second part (after --) is valid IPs
    local parsing_valid=false
    for arg in "$@"; do
        if [ "$arg" = "--" ]; then
            parsing_valid=true
            continue
        fi
        
        if [ "$parsing_valid" = true ]; then
            valid_ips+=("$arg")
        else
            test_ips+=("$arg")
        fi
    done
    
    # Check if we have any test IPs
    if [ ${#test_ips[@]} -eq 0 ]; then
        echo "  Debug: No test IPs provided" >&2
        return 1
    fi
    
    # Check if we have any valid IPs
    if [ ${#valid_ips[@]} -eq 0 ]; then
        echo "  Debug: No valid IPs provided" >&2
        return 1
    fi
    
    echo "  Debug: Validating ${#test_ips[@]} test IPs against ${#valid_ips[@]} valid IPs" >&2
    echo "  Debug: Test IPs: ${test_ips[*]}" >&2
    echo "  Debug: Valid range: ${valid_ips[*]}" >&2
    
    # Validate each test IP
    for ip in "${test_ips[@]}"; do
        if ! ip_in_range "$ip" "${valid_ips[@]}"; then
            echo "  Debug: IP '$ip' not in valid range" >&2
            return 1
        fi
    done
    
    echo "  Debug: All IPs validated successfully" >&2
    return 0
}
