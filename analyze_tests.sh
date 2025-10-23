#!/bin/bash
for test in test_cache test_closure test_factorial_float test_factorial_simple test_lambda_match test_recursion_no_match test_simple_recursion test_two_param_recursion; do
    echo "=== $test ==="
    if [ -f "$test.flap" ]; then
        timeout 1s ./flapc "$test.flap" "$test" 2>&1 | head -5
    else
        echo "File not found"
    fi
    echo
done
