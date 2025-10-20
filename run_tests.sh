#!/bin/bash
# Simple test runner for Flapc

passed=0
failed=0
skipped=0

for flap_file in programs/*.flap; do
    base=$(basename "$flap_file" .flap)
    result_file="programs/${base}.result"

    # Skip if no result file exists
    if [ ! -f "$result_file" ]; then
        continue
    fi

    # Compile
    if ! ./flapc "$flap_file" > /dev/null 2>&1; then
        echo "FAIL: $base (compilation failed)"
        ((failed++))
        continue
    fi

    # Run and capture output
    actual_output=$(./"$base" 2>&1)
    exit_code=$?

    # Read expected output
    expected_output=$(cat "$result_file")

    # Compare
    if [ "$actual_output" == "$expected_output" ]; then
        echo "PASS: $base"
        ((passed++))
    else
        echo "FAIL: $base"
        echo "  Expected: $expected_output"
        echo "  Got: $actual_output"
        ((failed++))
    fi

    # Clean up binary
    rm -f "$base"
done

echo ""
echo "=========================================="
echo "Test Results: $passed passed, $failed failed, $skipped skipped"
echo "=========================================="

if [ $failed -eq 0 ]; then
    exit 0
else
    exit 1
fi
