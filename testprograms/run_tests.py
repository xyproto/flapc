#!/usr/bin/env python3
import subprocess, os, sys, re

os.chdir('/home/alexander/clones/flapc/testprograms')
tests = [f.replace('.flap', '') for f in os.listdir('.') if f.endswith('.flap')]
passed, failed_compile, failed_run, failed_output = 0, 0, 0, 0

def matches_with_wildcards(actual, expected):
    """Check if actual matches expected, treating * as wildcard for any non-whitespace"""
    # Convert bytes to string for comparison
    try:
        actual_str = actual.decode('utf-8')
        expected_str = expected.decode('utf-8')
    except:
        # If decoding fails, do exact byte comparison
        return actual == expected

    # Normalize: strip final newline if present for comparison
    actual_str = actual_str.rstrip('\n')
    expected_str = expected_str.rstrip('\n')

    # Split into lines for line-by-line comparison
    actual_lines = actual_str.split('\n')
    expected_lines = expected_str.split('\n')

    if len(actual_lines) != len(expected_lines):
        return False

    for actual_line, expected_line in zip(actual_lines, expected_lines):
        # Strip trailing whitespace from both for comparison (common formatting difference)
        actual_line = actual_line.rstrip()
        expected_line = expected_line.rstrip()

        # Replace * with regex pattern for non-whitespace
        pattern = re.escape(expected_line).replace(r'\*', r'[^\s]+')
        if not re.fullmatch(pattern, actual_line):
            return False

    return True

for test in sorted(tests):
    if not os.path.exists(f'{test}.result'):
        continue

    # Compile
    ret = subprocess.run(['../flapc', f'{test}.flap', '-o', f'/tmp/{test}'],
                        capture_output=True)
    if ret.returncode != 0:
        failed_compile += 1
        print(f'FAIL_COMPILE: {test}')
        continue

    # Run
    try:
        ret = subprocess.run([f'/tmp/{test}'], capture_output=True, timeout=2)
    except subprocess.TimeoutExpired:
        failed_run += 1
        print(f'TIMEOUT: {test}')
        continue

    # Check output (with wildcard support)
    # For non-zero exit codes, check stderr first, then stdout
    with open(f'{test}.result', 'rb') as f:
        expected = f.read()

    if ret.returncode != 0:
        # Non-zero exit - check if expected output matches stderr (error messages)
        if matches_with_wildcards(ret.stderr, expected):
            passed += 1
            continue
        # Also try stdout for backwards compatibility
        if matches_with_wildcards(ret.stdout, expected):
            passed += 1
            continue
        # Neither matched
        failed_run += 1
        print(f'FAIL_RUN: {test} (exit {ret.returncode})')
        continue

    # Exit code 0 - check stdout
    if not matches_with_wildcards(ret.stdout, expected):
        failed_output += 1
        print(f'FAIL_OUTPUT: {test}')
        continue

    passed += 1

total = passed + failed_compile + failed_run + failed_output
print(f'\nSummary: {passed}/{total} passed ({100*passed//total}%)')
print(f'  {failed_compile} compile errors')
print(f'  {failed_run} runtime errors')
print(f'  {failed_output} output mismatches')
