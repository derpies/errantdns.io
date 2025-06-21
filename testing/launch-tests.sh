#!/bin/bash

# DNS Server Test Framework for ErrantDNS
# Usage: ./test-dns.sh [port] [test-suite]

set -euo pipefail

# Configuration
PORT=${1:-10001}
TEST_SUITE=${2:-all}
SERVER="127.0.0.1"
TIMEOUT=5

# Get script directory for relative imports
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source modules
source "$SCRIPT_DIR/lib/test-framework.sh"
source "$SCRIPT_DIR/lib/test-helpers.sh"
source "$SCRIPT_DIR/tests/basic-tests.sh"
source "$SCRIPT_DIR/tests/record-tests.sh"
source "$SCRIPT_DIR/tests/advanced-tests.sh"
source "$SCRIPT_DIR/tests/negative-tests.sh"

# Initialize test framework
init_test_framework "$SERVER" "$PORT" "$TIMEOUT"

# Main test execution
main() {
    print_header "ErrantDNS Server Test Suite"
    echo "Server: $SERVER:$PORT"
    echo "Timeout: ${TIMEOUT}s"
    echo "Test Suite: $TEST_SUITE"
    echo

    # Check server connectivity
    check_server_connectivity || exit 1

    # Run test suites based on argument
    case "$TEST_SUITE" in
        "all")
            run_basic_tests
            run_record_tests
            run_advanced_tests
            run_negative_tests
            ;;
        "basic")
            run_basic_tests
            ;;
        "records")
            run_record_tests
            ;;
        "advanced")
            run_advanced_tests
            ;;
        "negative")
            run_negative_tests
            ;;
        *)
            echo "Unknown test suite: $TEST_SUITE"
            echo "Available suites: all, basic, records, advanced, negative"
            exit 1
            ;;
    esac

    # Print final summary
    print_summary
}

# Run main function
main "$@"
