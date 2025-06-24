#!/bin/bash

# DNS Server Stress Testing Suite
# Tests realistic traffic patterns with positive/negative cache scenarios

set -e

# Configuration
DNS_SERVER="127.0.0.1"
DNS_PORT="10001"
POSITIVE_FILE="stress.dat"
TEST_DURATION=60
RESULTS_DIR="dns_test_results_$(date +%Y%m%d_%H%M%S)"

# Create results directory
mkdir -p "$RESULTS_DIR"

echo "=== DNS Server Stress Testing Suite ==="
echo "Target: $DNS_SERVER:$DNS_PORT"
echo "Duration: ${TEST_DURATION}s per test"
echo "Results: $RESULTS_DIR/"
echo

# Check if positive records file exists
if [[ ! -f "$POSITIVE_FILE" ]]; then
    echo "ERROR: $POSITIVE_FILE not found!"
    echo "Create a file with your known records, format: domain.com A"
    exit 1
fi

# Function to generate random domains for cache misses
generate_negative_queries() {
    local count=$1
    local output=$2
    
    echo "# Generated negative cache queries" > "$output"
    
    # Random subdomains of non-existent domains
    for i in $(seq 1 $((count / 4))); do
        echo "nonexistent-$(openssl rand -hex 6).invalid A" >> "$output"
        echo "missing-$(openssl rand -hex 6).test AAAA" >> "$output"
        echo "fake-$(openssl rand -hex 6).example MX" >> "$output"
        echo "ddos-$(openssl rand -hex 6).attack TXT" >> "$output"
    done
}

# Function to create mixed query patterns
create_mixed_queries() {
    local positive_ratio=$1
    local total_queries=$2
    local output=$3
    
    local positive_count=$((total_queries * positive_ratio / 100))
    local negative_count=$((total_queries - positive_count))
    
    echo "Creating mixed query file: ${positive_ratio}% positive, $((100-positive_ratio))% negative"
    
    # Repeat positive queries to reach target count
    local positive_lines=$(wc -l < "$POSITIVE_FILE")
    local repeat_factor=$((positive_count / positive_lines + 1))
    
    > "$output"
    for i in $(seq 1 $repeat_factor); do
        cat "$POSITIVE_FILE" >> "$output"
    done
    head -n "$positive_count" "$output" > "${output}.tmp"
    
    # Add negative queries
    generate_negative_queries "$negative_count" "${output}.neg"
    cat "${output}.neg" >> "${output}.tmp"
    
    # Shuffle to simulate realistic traffic
    shuf "${output}.tmp" > "$output"
    rm -f "${output}.tmp" "${output}.neg"
    
    echo "Generated $(wc -l < "$output") queries for testing"
}

# Function to run a single test
run_test() {
    local test_name=$1
    local query_file=$2
    local qps_limit=$3
    
    echo "Running $test_name..."
    echo "  Query file: $query_file"
    echo "  QPS limit: ${qps_limit:-unlimited}"
    
    local cmd="dnsperf -s $DNS_SERVER -p $DNS_PORT -d $query_file -l $TEST_DURATION"
    if [[ -n "$qps_limit" ]]; then
        cmd="$cmd -Q $qps_limit"
    fi
    
    local output_file="$RESULTS_DIR/${test_name}_results.txt"
    
    echo "  Command: $cmd"
    $cmd > "$output_file" 2>&1
    
    # Extract key metrics
    local qps=$(grep "Queries per second:" "$output_file" | awk '{print $4}' || echo "N/A")
    local avg_latency=$(grep "Average Latency:" "$output_file" | awk '{print $3}' || echo "N/A")
    local timeouts=$(grep "Query timeout" "$output_file" | wc -l || echo "0")
    
    echo "  Results: ${qps} QPS, ${avg_latency} avg latency, ${timeouts} timeouts"
    echo "$test_name,$qps,$avg_latency,$timeouts" >> "$RESULTS_DIR/summary.csv"
    echo
}

# Initialize summary file
echo "Test,QPS,Avg_Latency,Timeouts" > "$RESULTS_DIR/summary.csv"

echo "=== Test 1: Baseline - 100% Positive Cache Hits ==="
run_test "positive_baseline" "$POSITIVE_FILE"

echo "=== Test 2: Realistic Mix - 95% Positive, 5% Negative ==="
create_mixed_queries 95 1000 "$RESULTS_DIR/mixed_95_5.txt"
run_test "mixed_95_5" "$RESULTS_DIR/mixed_95_5.txt"

echo "=== Test 3: Heavy Cache Miss - 80% Positive, 20% Negative ==="
create_mixed_queries 80 1000 "$RESULTS_DIR/mixed_80_20.txt"
run_test "mixed_80_20" "$RESULTS_DIR/mixed_80_20.txt"

echo "=== Test 4: DDoS Simulation - 50% Positive, 50% Random ==="
create_mixed_queries 50 2000 "$RESULTS_DIR/ddos_sim.txt"
run_test "ddos_simulation" "$RESULTS_DIR/ddos_sim.txt"

echo "=== Test 5: Pure Negative Cache (Worst Case) ==="
generate_negative_queries 500 "$RESULTS_DIR/negative_only.txt"
run_test "negative_only" "$RESULTS_DIR/negative_only.txt"

echo "=== Test 6: Rate Limited Tests (Sustained Load) ==="
echo "Testing sustained performance at different QPS levels..."

for qps in 1000 5000 10000 25000 50000; do
    echo "  Testing sustained $qps QPS..."
    run_test "sustained_${qps}qps" "$RESULTS_DIR/mixed_95_5.txt" "$qps"
done

echo "=== Performance Analysis ==="

# Generate performance report
cat > "$RESULTS_DIR/performance_report.txt" << 'EOF'
DNS Server Performance Test Report
==================================

ACCEPTABLE PERFORMANCE THRESHOLDS:
- Baseline (cached): >50,000 QPS
- Realistic mix (95/5): >25,000 QPS  
- Cache miss heavy (80/20): >10,000 QPS
- DDoS simulation: >5,000 QPS
- Pure negative: >2,000 QPS

DDOS CONSIDERATIONS:
- Random subdomain attacks (tested in negative scenarios)
- High query rate sustainability (tested with rate limits)
- Cache pollution resistance (mixed positive/negative)

PRODUCTION RECOMMENDATIONS:
- Deploy rate limiting at 75% of sustained capacity
- Monitor cache hit ratios (should stay >90%)
- Alert on negative query spikes (>10% of traffic)
- Implement query filtering for obviously malicious patterns

TEST RESULTS:
EOF

# Add results to report
echo >> "$RESULTS_DIR/performance_report.txt"
cat "$RESULTS_DIR/summary.csv" >> "$RESULTS_DIR/performance_report.txt"

echo
echo "=== Test Complete ==="
echo "Results saved to: $RESULTS_DIR/"
echo
echo "Key files:"
echo "  - summary.csv: All test results"
echo "  - performance_report.txt: Analysis and recommendations"
echo "  - *_results.txt: Detailed dnsperf output for each test"
echo
echo "QUICK ASSESSMENT:"
echo "Check sustained_*qps tests to find your maximum sustainable load."
echo "Compare mixed test results to determine cache dependency."
echo
cat "$RESULTS_DIR/summary.csv" | column -t -s ','