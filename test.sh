#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT="$SCRIPT_DIR"
PROGRAM_DIR="$REPO_ROOT/programs"
BUILD_ROOT="$REPO_ROOT/build"
mkdir -p "$BUILD_ROOT"

# Expected compile results: "success" or "failure"
declare -A compile_expectation
compile_expectation[const]="failure"
compile_expectation[hash_length_test]="failure"

# Skip experimental/demo programs that don't have test expectations yet
declare -A skip_programs
skip_programs[bool_test]=1
skip_programs[format_test]=1
skip_programs[in_demo]=1
skip_programs[in_simple]=1
skip_programs[in_test]=1
skip_programs[map_test]=1
skip_programs[new_features]=1
skip_programs[printf_demo]=1
skip_programs[printf_test]=1
skip_programs[showcase]=1
skip_programs[simple_printf]=1
skip_programs[test_escape]=1
skip_programs[test_g]=1
skip_programs[test_v]=1

# Expected substrings in compiler output when compilation fails.
declare -A compile_failure_patterns
compile_failure_patterns[const]="cannot reassign immutable variable"
compile_failure_patterns[hash_length_test]="panic: runtime error"

# Expected output is now stored in .result files in programs/ directory

# Expected exit codes for successful program runs (default to 0 when unspecified).
declare -A expected_exit_code
expected_exit_code[first]=0

failures=()

run_grep_check() {
    local expected_file="$1"
    local actual_file="$2"
    local program_name="$3"

    # Check if expected output file is empty
    if [[ ! -s "$expected_file" ]]; then
        # Expected empty output
        if [[ -s "$actual_file" ]]; then
            failures+=("${program_name}: expected no output but got output")
        fi
        return
    fi

    # Compare expected and actual output line by line
    local found_all=0
    while IFS= read -r expected_line; do
        if ! grep -Fxq "$expected_line" "$actual_file"; then
            failures+=("${program_name}: missing expected line: $expected_line")
            found_all=1
        fi
    done < "$expected_file"

    if [[ $found_all -eq 0 ]]; then
        # Ensure there are no unexpected extra lines by comparing counts.
        local expected_count
        expected_count=$(grep -c '' "$expected_file" 2>/dev/null || echo 0)
        local actual_count
        actual_count=$(grep -c '' "$actual_file" 2>/dev/null || echo 0)
        if [[ "$expected_count" -ne "$actual_count" ]]; then
            failures+=("${program_name}: expected $expected_count lines but found $actual_count")
        fi
    fi
}

for src in "$PROGRAM_DIR"/*.flap; do
    base=$(basename "$src" .flap)

    # Skip experimental/demo programs
    if [[ "${skip_programs[$base]:-0}" == "1" ]]; then
        continue
    fi

    compile_log="$BUILD_ROOT/${base}.compile.log"
    executable="$BUILD_ROOT/${base}"
    out_file="$BUILD_ROOT/${base}.out"
    expected_file="$PROGRAM_DIR/${base}.result"

    # Compile
    echo "Compiling $src"
    if ./flapc -o "$executable" "$src" >"$compile_log" 2>&1; then
        if [[ "${compile_expectation[$base]:-success}" == "failure" ]]; then
            failures+=("${base}: expected compilation to fail but it succeeded")
            continue
        fi
    else
        if [[ "${compile_expectation[$base]:-success}" == "success" ]]; then
            failures+=("${base}: compilation failed unexpectedly. See $compile_log")
            continue
        fi
        pattern="${compile_failure_patterns[$base]:-}"
        if [[ -n "$pattern" ]] && ! grep -Fq "$pattern" "$compile_log"; then
            failures+=("${base}: compilation failed but missing expected pattern '$pattern'")
        fi
        # No runtime step when compilation is expected to fail.
        continue
    fi

    # Run program
    echo "Testing $executable"
    if "$executable" >"$out_file" 2>&1; then
        status=0
    else
        status=$?
    fi

    expected_status="${expected_exit_code[$base]:-0}"
    if [[ "$status" -ne "$expected_status" ]]; then
        failures+=("${base}: expected exit status $expected_status but got $status")
    fi

    # Check if .result file exists
    if [[ ! -f "$expected_file" ]]; then
        failures+=("${base}: no .result file found at $expected_file")
        continue
    fi

    run_grep_check "$expected_file" "$out_file" "$base"
done

if [[ ${#failures[@]} -gt 0 ]]; then
    printf 'Test failures:\n'
    printf ' - %s\n' "${failures[@]}"
    exit 1
fi

printf 'All Flap programs compiled and produced expected output.\n'
