#!/bin/bash

# Test script for all zlog examples
# This script runs tests for each example to ensure they compile and work correctly

set -e  # Exit on first error

echo "Testing all zlog examples..."
echo "==========================="

# Array of example directories
examples=(
    "custom-fields"
    "custom-signals"
    "custom-sink"
    "event-pipeline"
    "resilient-sinks"
    "standard-logging"
)

# Track results
passed=0
failed=0

# Test each example
for example in "${examples[@]}"; do
    echo ""
    echo "Testing $example..."
    echo "-------------------"
    
    cd "$example"
    
    # Run tests
    if go test -v -short ./...; then
        echo "✓ $example tests passed"
        ((passed++))
    else
        echo "✗ $example tests failed"
        ((failed++))
    fi
    
    cd ..
done

# Summary
echo ""
echo "==========================="
echo "Test Summary:"
echo "  Passed: $passed"
echo "  Failed: $failed"
echo ""

if [ $failed -eq 0 ]; then
    echo "All tests passed! ✓"
    exit 0
else
    echo "Some tests failed! ✗"
    exit 1
fi