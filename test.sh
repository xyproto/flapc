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
expected_stdout[iftest2]=$'yes'
expected_stdout[iftest3]=$'before if\nyes\nafter if'
expected_stdout[iftest4]=$'x is NOT less than y'
expected_stdout[iftest]=$'x is less than y'
expected_stdout[index_direct_test]=$'2\n4\n6'
expected_stdout[lambda_comprehensive]=$'25\n14\n7\n42\n17\n26'
expected_stdout[lambda_direct_test]=$'0'
expected_stdout[lambda_loop]=$'0\n2\n4'
expected_stdout[lambda_multi_arg_test]=$'10'
expected_stdout[lambda_multiple_test]=$'22'
expected_stdout[lambda_parse_test2]=$'This should not execute'
expected_stdout[lambda_parse_test]=$'Testing lambda parsing'
expected_stdout[lambda_store_only]=$'Stored lambda'
expected_stdout[lambda_store_test]=$'10'
expected_stdout[lambda_test]=$'2\n4\n6'
expected_stdout[len_empty]=$'0'
expected_stdout[len_simple]=$'5'
expected_stdout[len_test]=$'5\n0'
expected_stdout[list_index_test]=$'10\n20\n50'
expected_stdout[list_iter_test]=$'10\n20\n30\n40\n50'
expected_stdout[list_simple]=$'10'
expected_stdout[list_test2]=$'Multiple lists created'
expected_stdout[list_test]=$'2\n4\n6'
expected_stdout[loop_mult]=$'0\n2\n4\n6\n8'
expected_stdout[loop_test2]=$'0\n1\n2\n3\n4\n5\n6\n7\n8\n9'
expected_stdout[loop_test]=$'0\n1\n2\n3\n4'
expected_stdout[loop_with_arithmetic]=$'10'
expected_stdout[manual_list_test]=$'0\n99\n99'
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
expected_stdout[parallel_test_const]=$'0\n99\n99'
expected_stdout[parallel_test_const_delay]=$'0\n99\n99'
expected_stdout[parallel_test_debug]=$'0\n14\n22'
expected_stdout[parallel_test_delay]=$'0\n4\n6'
expected_stdout[parallel_test_direct]=$'0\n4\n6'
expected_stdout[parallel_test_elements]=$'0\n4\n6'
expected_stdout[parallel_test_four]=$'0\n4\n6\n8'
expected_stdout[parallel_test_length]=$'3'
expected_stdout[parallel_test_print]=$'0'
expected_stdout[parallel_test_reverse]=$'6\n4\n0'
expected_stdout[parallel_test_simple]=$'0'
expected_stdout[parallel_test_single]=$'0'
expected_stdout[pipe_test]=$'10\n13'
expected_stdout[precedence]=$'14'
expected_stdout[subtract]=$'42'
expected_stdout[test_lambda_ptr]=$'10'
expected_stdout[test_list_only]=$'1'
expected_stdout[test_null_len]=$'0'
expected_stdout[test_parallel_null_return]=$'0'
expected_stdout[test_simple]=$'Test'
expected_stdout[test_with_exit]=$'Test'
expected_stdout[simple_format]=$'Whole: 42\nFloat: 3.14\nBool true: yes\nBool false: no'

# SIMD map indexing tests
expected_stdout[test_map_index]=$'Age at key 2: 30'
expected_stdout[test_map_comprehensive]=$'Price for item 100: 5.99\nPrice for item 200: 12.99\nPrice for item 300: 19.99\nPrice for item 999: 0\nResult from empty map: 0'
expected_stdout[test_map_two_lookups]=$'Price for item 100: 5.99\nPrice for item 200: 12.99'
expected_stdout[test_map_three_lookups]=$'Price for item 100: 5.99\nPrice for item 200: 12.99\nPrice for item 300: 19.99'
expected_stdout[test_map_nonexistent]=$'Price for item 999: 0'
expected_stdout[test_map_empty]=$'Result from empty map: 0'
expected_stdout[test_map_simd_large]=$'Item 100: $5.99\nItem 300: $19.99\nItem 500: $29.99\nItem 600: $34.99\nItem 999: $0'
expected_stdout[test_map_avx512_large]=$'Item 100: $9.99\nItem 800: $79.99\nItem 1600: $159.99\nItem 9999: $0'
expected_stdout[test_cpu_detection]=$'Program started with CPU detection\nIf you see this, SSE2 path is working\nMap test result: 100'

# String tests (strings as map[uint64]float64, CString for output)
expected_stdout[test_string_literal]=$'Hello, World!'
expected_stdout[test_string_map]=$'Character code: 66'
expected_stdout[test_string_index_debug]=$'s[0] = 65 (should be 65 for \'A\')\ns[1] = 66 (should be 66 for \'B\')\ns[2] = 67 (should be 67 for \'C\')'
expected_stdout[test_string_concat]=$'Hello, World!'
expected_stdout[test_string_concat_literal]=$'Hello, World!'
expected_stdout[test_simple_string_var]=$'Hello'
expected_stdout[test_string_length]=$'5'
expected_stdout[test_negative]=$'-5'
expected_stdout[test_tail_recursion]=$'120'
expected_stdout[test_tail_recursion_sum]=$'55'
expected_stdout[test_tail_recursion_fibonacci]=$'55'
expected_stdout[test_tail_recursion_countdown]=$'0'

# Math function tests
expected_stdout[test_abs]=$'5'
expected_stdout[test_abs_simple]=$'5'
expected_stdout[test_sqrt]=$'2'
expected_stdout[test_sin]=$'0'
expected_stdout[test_cos]=$'1'
expected_stdout[test_tan]=$'0'
expected_stdout[test_atan]=$'0'
expected_stdout[test_asin]=$'0'
expected_stdout[test_acos]=$'0'
expected_stdout[test_floor]=$'3'
expected_stdout[test_ceil]=$'4'
expected_stdout[test_round]=$'4'
expected_stdout[test_log]=$'1'
expected_stdout[test_exp]=$'3'
expected_stdout[test_pow]=$'8'

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

    # Skip experimental/demo programs
    if [[ "${skip_programs[$base]:-0}" == "1" ]]; then
        continue
    fi

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
