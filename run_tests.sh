#!/bin/bash
passed=0
failed=0
for test in *.flap; do
    [[ -f "$test" ]] || continue
    name="${test%.flap}"
    echo "=== Testing $test ==="
    if timeout 2s ./flapc "$test" "$name" > /dev/null 2>&1; then
        if timeout 2s "./$name" > /dev/null 2>&1; then
            echo "PASS"
            ((passed++))
        else
            echo "FAIL (runtime error or timeout)"
            ((failed++))
        fi
    else
        echo "FAIL (compilation error or timeout)"
        ((failed++))
    fi
done
echo ""
echo "Results: $passed passed, $failed failed"
