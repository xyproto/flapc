# Error Handling Tests

This directory contains test files for the railway-oriented error handling system.

## Purpose

These files are intentionally designed to trigger compilation errors. They test:
- Error detection and reporting
- Error message quality
- Error recovery (multiple errors in one file)
- Source context in error messages

## Test Files

- `undefined_var.flap` - Tests undefined variable detection (semantic error)
- `syntax_error.flap` - Tests syntax error handling

## Running Tests

To test error handling manually:

```bash
./flapc tests/errors/undefined_var.flap -o /tmp/test 2>&1
```

Expected output should show:
- Clear error messages
- Source location (file:line:column)
- Helpful context and suggestions

## Future Tests

Planned test files:
- `type_mismatch.flap` - Type error detection
- `multiple_errors.flap` - Multiple errors in one file (tests error recovery)
- `immutable_update.flap` - Attempting to update immutable variable

## Success Criteria

- Errors provide file:line:column location
- Error messages include source context
- Multiple errors are reported (not just the first one)
- Suggestions are provided where applicable
