#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT="$SCRIPT_DIR"
PROGRAM_DIR="$REPO_ROOT/programs"
BUILD_ROOT="$REPO_ROOT/build/test_programs"
mkdir -p "$BUILD_ROOT"

# Expected compile results: "success" or "failure"
declare -A compile_expectation
compile_expectation[const]="failure"
compile_expectation[hash_length_test]="failure"

# Expected substrings in compiler output when compilation fails.
declare -A compile_failure_patterns
compile_failure_patterns[const]="cannot reassign const variable"
compile_failure_patterns[hash_length_test]="panic: runtime error"

# Expected stdout patterns for successfully compiled programs.
declare -A expected_stdout
expected_stdout[add]=$'42'
expected_stdout[all_arithmetic]=$'13\n7\n30\n3'
expected_stdout[arithmetic_test]=$'8'
expected_stdout[comparison_test]=$'10 < 20: true\n20 > 10: true\n10 == 10: true\n10 != 20: true\n10 <= 10: true\n20 >= 10: true'
expected_stdout[const_test]=$'0'
expected_stdout[divide]=$'42'
expected_stdout[first]='__EMPTY__'
expected_stdout[hello]=$'Hello, World!'
expected_stdout[iftest]=$'x is less than y'
expected_stdout[iftest2]=$'yes'
expected_stdout[iftest3]=$'before if\nyes\nafter if'
expected_stdout[iftest4]=$'x is NOT less than y'
expected_stdout[lambda_comprehensive]=$'25\n14\n7\n42\n17\n26'
expected_stdout[lambda_direct_test]=$'0'
expected_stdout[lambda_loop]=$'0\n2\n4'
expected_stdout[lambda_multi_arg_test]=$'10'
expected_stdout[lambda_multiple_test]=$'22'
expected_stdout[lambda_parse_test]=$'Testing lambda parsing'
expected_stdout[lambda_parse_test2]=$'This should not execute'
expected_stdout[lambda_store_only]=$'Stored lambda'
expected_stdout[lambda_store_test]=$'10'
expected_stdout[len_empty]=$'0'
expected_stdout[len_simple]=$'5'
expected_stdout[len_test]=$'5\n0'
expected_stdout[list_index_test]=$'10\n20\n50'
expected_stdout[list_iter_test]=$'10\n20\n30\n40\n50'
expected_stdout[list_simple]=$'10'
expected_stdout[list_test]=$'List created'
expected_stdout[list_test2]=$'Multiple lists created'
expected_stdout[loop_mult]=$'0\n2\n4\n6\n8'
expected_stdout[loop_test]=$'0\n1\n2\n3\n4'
expected_stdout[loop_test2]=$'0\n1\n2\n3\n4\n5\n6\n7\n8\n9'
expected_stdout[loop_with_arithmetic]=$'10'
expected_stdout[manual_map]=$'2\n4\n6'
expected_stdout[mixed]=$'75'
expected_stdout[multiply]=$'42'
expected_stdout[mutable]=$'30'
expected_stdout[nested_loop]=$'0\n0\n0\n1\n0\n2\n1\n0\n1\n1\n1\n2\n2\n0\n2\n1\n2\n2'
expected_stdout[parallel_empty]=$'0'
expected_stdout[parallel_map_test]=$'0\n4\n6\n8\n10'
expected_stdout[parallel_noop]=$'42'
expected_stdout[parallel_parse_test]=$'Done'
expected_stdout[parallel_simple]=$'0'
expected_stdout[parallel_simple_test]=$'0'
expected_stdout[parallel_single]=$'0'
expected_stdout[parallel_test]=$'0\n4\n6\n8\n10'
expected_stdout[precedence]=$'14'
expected_stdout[subtract]=$'42'
expected_stdout[test_lambda_ptr]=$'10'
expected_stdout[test_list_only]=$'1'

# Expected exit codes for successful program runs (default to 0 when unspecified).
declare -A expected_exit_code
expected_exit_code[first]=0

failures=()

run_grep_check() {
    local pattern_file="$1"
    local file_to_check="$2"
    local program_name="$3"

    if [[ "$pattern_file" == "__EMPTY__" ]]; then
        if [[ -s "$file_to_check" ]]; then
            failures+=("${program_name}: expected no output but got output")
        fi
        return
    fi

    local found_all=0
    while IFS= read -r expected_line; do
        # Skip completely empty lines in the expectation (none currently used).
        if [[ -z "$expected_line" ]]; then
            continue
        fi
        if ! grep -Fxq "$expected_line" "$file_to_check"; then
            failures+=("${program_name}: missing expected line: $expected_line")
            found_all=1
        fi
    done <<<"$pattern_file"

    if [[ $found_all -eq 0 ]]; then
        # Ensure there are no unexpected extra lines by comparing counts.
        local expected_count
        expected_count=$(printf '%s\n' "$pattern_file" | sed '/^$/d' | wc -l)
        local actual_count
        actual_count=$(grep -c '' "$file_to_check")
        if [[ "$expected_count" -ne "$actual_count" ]]; then
            failures+=("${program_name}: expected $expected_count lines but found $actual_count")
        fi
    fi
}

for src in "$PROGRAM_DIR"/*.flap; do
    base=$(basename "$src" .flap)
    compile_log="$BUILD_ROOT/${base}.compile.log"
    executable="$BUILD_ROOT/${base}"
    out_file="$BUILD_ROOT/${base}.out"

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

    expected_output="${expected_stdout[$base]:-__MISSING__}"
    if [[ "$expected_output" == "__MISSING__" ]]; then
        failures+=("${base}: no expected output defined")
        continue
    fi

    run_grep_check "$expected_output" "$out_file" "$base"
done

if [[ ${#failures[@]} -gt 0 ]]; then
    printf 'Test failures:\n'
    printf ' - %s\n' "${failures[@]}"
    exit 1
fi

printf 'All Flap programs compiled and produced expected output.\n'
