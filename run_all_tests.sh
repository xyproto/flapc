#!/bin/bash
passed=0
failed=0
errors=""

for test in programs/*_test.flap; do
    name=$(basename "$test" .flap)
    if ./flapc "$test" > /dev/null 2>&1; then
        if ./"$name" > /dev/null 2>&1; then
            ((passed++))
        else
            ((failed++))
            errors="${errors}RUNTIME FAIL: $name\n"
        fi
    else
        ((failed++))
        errors="${errors}COMPILE FAIL: $name\n"
    fi
done

echo "Test Results: $passed passed, $failed failed"
if [ $failed -gt 0 ]; then
    echo -e "\nFailed tests:"
    echo -e "$errors"
fi
