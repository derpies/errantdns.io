#!/bin/bash

# Simple Redis integration test for ErrantDNS
# Tests that Redis caching is working correctly

set -e

echo "=== ErrantDNS Redis Integration Test ==="
echo

# Configuration
DNS_SERVER="127.0.0.1"
DNS_PORT="10001"
REDIS_HOST="localhost"
REDIS_PORT="6379"

# Check if Redis is running
echo "Checking Redis connectivity..."
if ! redis-cli -h $REDIS_HOST -p $REDIS_PORT ping > /dev/null 2>&1; then
    echo "ERROR: Redis is not running at $REDIS_HOST:$REDIS_PORT"
    echo "Please start Redis server: redis-server"
    exit 1
fi
echo "✓ Redis is running"

# Check if DNS server is running with Redis enabled
echo "Checking if DNS server is running..."
if ! dig @$DNS_SERVER -p $DNS_PORT +short test.internal A > /dev/null 2>&1; then
    echo "ERROR: DNS server is not running or not responding"
    echo "Start the server with: REDIS_ENABLED=true ./launch.sh"
    exit 1
fi
echo "✓ DNS server is responding"

# Test DNS queries and check Redis for cached entries
echo ""
echo "Testing DNS queries and Redis caching..."

# Clear any existing test cache entries
redis-cli -h $REDIS_HOST -p $REDIS_PORT --scan --pattern "errantdns:*" | xargs -r redis-cli -h $REDIS_HOST -p $REDIS_PORT del > /dev/null 2>&1

# Query DNS records to populate cache
echo "Querying test.internal A record..."
RESULT1=$(dig @$DNS_SERVER -p $DNS_PORT +short test.internal A)
echo "Result: $RESULT1"

echo "Querying www.test.internal A record..."
RESULT2=$(dig @$DNS_SERVER -p $DNS_PORT +short www.test.internal A)
echo "Result: $RESULT2"

# Check if Redis has cached entries
echo ""
echo "Checking Redis for cached entries..."
CACHE_KEYS=$(redis-cli -h $REDIS_HOST -p $REDIS_PORT --scan --pattern "errantdns:*" | wc -l)

if [ "$CACHE_KEYS" -gt 0 ]; then
    echo "✓ Found $CACHE_KEYS cached entries in Redis"
    echo "Cache keys:"
    redis-cli -h $REDIS_HOST -p $REDIS_PORT --scan --pattern "errantdns:*" | while read key; do
        echo "  - $key"
    done
else
    echo "⚠ No cached entries found in Redis"
    echo "This might indicate Redis caching is not working"
fi

# Test cache performance by timing queries
echo ""
echo "Testing cache performance..."
echo "First query (should hit database):"
time dig @$DNS_SERVER -p $DNS_PORT +short round-robin.internal A > /dev/null

echo "Second query (should hit cache):"
time dig @$DNS_SERVER -p $DNS_PORT +short round-robin.internal A > /dev/null

echo ""
echo "Test completed. Check the timing differences above."
echo "Cached queries should be significantly faster."

# Clean up test cache entries
redis-cli -h $REDIS_HOST -p $REDIS_PORT --scan --pattern "errantdns:*" | xargs -r redis-cli -h $REDIS_HOST -p $REDIS_PORT del > /dev/null 2>&1
echo "✓ Test cache entries cleaned up"
