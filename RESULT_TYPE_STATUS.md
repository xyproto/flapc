# Result Type and Error Handling Status

**Date:** 2025-11-17  
**Version:** Flap 3.0 (in progress)

## Summary

This document tracks the implementation status of the Result type and error handling system for Flap 3.0.

## Completed

### Documentation
- ✅ **GRAMMAR.md** updated with:
  - Result type byte layout specification
  - Type byte definitions (0x01-0x05 for Flap types, 0xE0 for errors, 0x10-0x1B for C types)
  - Standard error codes (dv0, idx, key, typ, nil, mem, arg, io, net, prs, ovf, udf)
  - `.error` accessor specification (strips trailing spaces)
  - `or!` operator specification
  - Memory layout examples
  - Error propagation patterns

- ✅ **LANGUAGESPEC.md** updated with:
  - Complete Result type design
  - Standard error codes
  - `.error` accessor behavior
  - `or!` operator semantics
  - Error propagation patterns
  - Best practices and examples
  - Memory layout diagrams

### Lexer
- ✅ `TOKEN_OR_BANG` and `TOKEN_ORBANG` already implemented
- ✅ `or!` operator properly lexed

### Parser
- ✅ `or!` operator already parsed as BinaryExpr
- ✅ `.error` field access already parsed
- ✅ Error handling precedence integrated

### Code Generation
- ✅ `or!` operator works (returns default value on NaN/error)
- ✅ `.error` accessor works (returns empty string for non-errors)

### Tests
- ✅ All existing tests pass
- ✅ `or!` operator tests pass:
  - TestOrBangWithSuccess
  - TestOrBangWithError
  - TestOrBangChaining
- ✅ `.error` basic functionality tests pass
- ✅ Division by zero returns NaN (works with `or!`)

## In Progress / TODO

### 1. Error Encoding in Operations

**Status:** Not yet implemented

Division by zero and other error conditions need to encode error values according to the Result type specification:

```
Error value format:
[0xE0][error_code_4_bytes][0x00]

Example for "dv0 " (division by zero):
E0 64 76 30 20 00
```

**Files to modify:**
- `div.go` - Add error encoding for division by zero
- `codegen.go` - Add error encoding helpers
- Index operations - Add "idx " error for out of bounds
- Map operations - Add "key " error for missing keys

**Implementation steps:**
1. Create `encodeError(code string) []byte` helper
2. Modify division codegen to check for zero divisor
3. Return error-encoded result instead of NaN
4. Test with `.error` accessor

### 2. The `error()` Builtin Function

**Status:** Not yet implemented

Need to add a builtin function to create custom errors:

```flap
err = error("arg")   // Creates error with code "arg "
code = err.error     // Returns "arg"
```

**Implementation steps:**
1. Add `error` to builtin function list in parser
2. Add codegen for `error()` call
3. Generate error-encoded value from string argument
4. Add tests

### 3. Enhance `.error` Accessor

**Status:** Partially implemented

Currently `.error` returns empty string for all non-error values. Need to:

1. Check type byte (first byte of value)
2. If 0xE0: extract next 4 bytes as error code
3. Strip trailing spaces
4. Return error code string
5. Otherwise: return empty string

**Files to modify:**
- Field access codegen for `.error` property
- Add runtime helper to extract error code from Result value

### 4. Type Tracking for Results

**Status:** Design complete, implementation TODO

The compiler should track which expressions return Result types to:
- Warn about unchecked Results
- Optimize `or!` operator usage
- Better error messages

See TYPE_TRACKING.md for design.

**Implementation steps:**
1. Add `ResultType` flag to Expression nodes
2. Propagate Result type through AST
3. Mark operations that can fail (division, indexing, etc.)
4. Add warnings for unchecked Results

### 5. Additional Error Operations

**Status:** Design complete, implementation TODO

From documentation:

```flap
// Index errors
xs = [1, 2, 3]
z = xs[10]              // Should return error "idx "

// Key errors  
m = { x: 10 }
w = m.y                 // Should return error "key "

// Arithmetic errors
y = 2 ** 1000           // Should return error "ovf " (overflow)
```

### 6. Error Propagation Shorthand

**Status:** Design complete, not yet implemented

Consider adding `?` operator for auto-propagation:

```flap
// Manual propagation (current)
process = input => {
    step1 = validate(input)
    step1.error { != "" -> step1 }
    
    step2 = transform(step1)
    step2.error { != "" -> step2 }
    
    finalize(step2)
}

// Potential shorthand (future)
process = input => {
    step1 = validate(input)?
    step2 = transform(step1)?
    finalize(step2)
}
```

This is **not** in the current spec, just a potential future enhancement.

## Test Status

### Passing Tests
- All core Flap tests pass
- `or!` operator tests pass
- Basic `.error` accessor tests pass
- Division by zero with `or!` works

### Skipped Tests (TODO)
- `TestErrorPropertyBasic` - Needs division error encoding
- `TestErrorFunction` - Needs `error()` builtin
- `TestReducePipe` - Not related to Result types

## Memory Layout Reference

### Success Value (number 42)
```
Bytes: 01 01 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 40 45 00 00 00 00 00 00 00 00
       ↑  ↑----- length=1 ----↑  ↑------- key=0 -------↑  ↑------- value=42.0 ------↑  ↑ term
       type=01 (number)
```

### Error Value (division by zero)
```
Bytes: E0 64 76 30 20 00
       ↑  ↑----- "dv0 " -----↑  ↑ term
       type=0xE0 (error)
```

### Type Byte Mapping
```
0x01 - Flap Number
0x02 - Flap String
0x03 - Flap List
0x04 - Flap Map
0x05 - Flap Address
0xE0 - Error
0x10 - C int8
0x11 - C int16
0x12 - C int32
0x13 - C int64
0x14 - C uint8
0x15 - C uint16
0x16 - C uint32
0x17 - C uint64
0x18 - C float32
0x19 - C float64
0x1A - C pointer
0x1B - C string pointer
```

## Standard Error Codes

All error codes are 4 bytes, space-padded:

```
"dv0 " - Division by zero
"idx " - Index out of bounds
"key " - Key not found
"typ " - Type mismatch
"nil " - Null pointer
"mem " - Out of memory
"arg " - Invalid argument
"io  " - I/O error
"net " - Network error
"prs " - Parse error
"ovf " - Overflow
"udf " - Undefined
```

## Implementation Priority

1. **High Priority:**
   - Division by zero error encoding
   - `.error` accessor enhancement
   - `error()` builtin function

2. **Medium Priority:**
   - Index out of bounds errors
   - Key not found errors
   - Type tracking for Results

3. **Low Priority:**
   - Overflow detection
   - Additional error operations
   - Error propagation shorthand (`?` operator)

## References

- [GRAMMAR.md](GRAMMAR.md) - Complete grammar with Result type
- [LANGUAGESPEC.md](LANGUAGESPEC.md) - Language semantics
- [TYPE_TRACKING.md](TYPE_TRACKING.md) - Type system design
- [LIBERTIES.md](LIBERTIES.md) - Documentation accuracy guidelines

## Next Steps

1. Implement error encoding in `div.go`
2. Add `error()` builtin function
3. Enhance `.error` accessor to decode error bytes
4. Unskip and pass the Result type tests
5. Add comprehensive error handling examples to `example_test.go`

---

**Commits:**
- `d180d2a` - Add comprehensive error and Result type documentation
- `6b95a47` - Add Result type tests (skipped for now)
