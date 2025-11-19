# TODO - Flap 3.0 Roadmap

**Status:** Very close to release
**Target:** Major improvements to lambda handling and compiler robustness  
**Current:** All tests passing! 3 critical issues fixed, 1 enhancement remaining

---

## Known Issues (Must Fix for 3.0)

### 1. ✅ pop() Function - FIXED
**Status:** Working correctly  
**Tests:** TestPopMethod, TestPopFunction, TestPopEmptyList all pass

### 2. ✅ Deeply Nested Loops (5+ levels) - FIXED
**Status:** Working correctly  
**Location:** register_tracker.go AllocIntCalleeSaved()  
**Fix:** Prevented caller-saved register fallback in AllocIntCalleeSaved. Now returns empty string when all callee-saved registers exhausted, forcing stack-based counters.
**Tests:** TestDeeplyNestedLoops passes

### 3. ✅ += Operator for List Append - WORKING
**Status:** Already implemented and working  
**Feature:** "result += 42" appends to list, "x += 5" adds to number
**Implementation:** Parser transforms compound assignments into binary expressions
**Tests:** Updated TestQuickSort and TestInsertionSort to use += syntax

### 4. Multiple Return Values  
**Status:** Feature not implemented  
**Issue:** Functions cannot return multiple values  
**Workaround:** Return a list/map with multiple values  
**Action:** See MULTIPLE_RETURNS_IMPLEMENTATION.md for design
**Priority:** Medium - workaround exists, but multiple returns would be cleaner

---

## Core Type System Redesign

### Universal Value Format with Type Tags

**Goal:** Add type byte prefix to all values for better runtime type checking and dispatch

**New Format:**
```
[type_byte][length_u64][key0_u64][val0_f64][key1_u64][val1_f64]...[0x00_terminator]
```

**Type Byte Values:**
- 0x01: Flap number (single value, length=1)
- 0x02: Flap string (ordered map of char codes)
- 0x03: Flap list (ordered map with sequential keys)
- 0x04: Flap map (unordered key-value pairs)
- 0x05: Flap address (ENet address)
- 0x10: C string (null-terminated)
- 0x11: C pointer (raw memory address)
- 0x12: C int8
- 0x13: C int16
- 0x14: C int32
- 0x15: C int64
- 0x16: C uint8
- 0x17: C uint16
- 0x18: C uint32
- 0x19: C uint64
- 0x1A: C float32
- 0x1B: C float64

**Actions:**
1. Update LANGUAGESPEC.md with new type system specification
2. Update grammar to reflect type-tagged values
3. Modify lexer to recognize type contexts
4. Update parser to generate type information
5. Rewrite literal compilation (numbers, strings, lists, maps)
6. Update all runtime helpers (_flap_string_concat, _flap_list_update, etc.)
7. Add type checking operations (typeof, is_string, is_number, etc.)
8. Update all operators to check/preserve type tags
9. Rewrite value loading/storing in codegen
10. Update C FFI to handle type conversions properly

**Benefits:**
- Fixes match+string edge case automatically
- Enables proper type introspection at runtime
- Better error messages
- Safer C FFI with explicit type conversions
- Foundation for future optimizations

**Breaking Changes:**
- All binary formats change
- Existing compiled programs incompatible
- Runtime helpers need complete rewrite

---

## Bug Fixes from 1.2.0

### Match Expressions with String Literals

**Current Issue:** Match expressions returning string literals produce garbage values

**Example:**
```flap
result := 0 {
    0 -> "zero"  // Returns garbage
    ~> "other"
}
```

**Root Cause:** String pointers in xmm0 not preserved across match clause jumps

**Actions:**
1. Debug `compileMatchClauseResult()` in codegen.go
2. Trace xmm0 value through jump instructions
3. Add explicit xmm0 preservation if needed
4. Verify fix doesn't break existing match behavior
5. Add test case for string literals in match

**Alternative:** Type system redesign fixes this automatically

---

## Language Features

### Implicit Match Blocks in Function Bodies

**Current State:** LANGUAGESPEC.md claims all function bodies `{ ... }` are match expressions, but parser doesn't support it

**Documentation says:**
```flap
factorial := n => {
    n == 0 -> 1
    ~> n * factorial(n - 1)
}
```

**Current reality:** This syntax doesn't parse

**Actions:**
1. Decide: Keep current behavior or implement documented behavior?
2. If implement: Modify parser to treat `=>` followed by `{` as implicit match
3. Update BlockExpr parsing to accept match syntax
4. Test extensively (this is a major syntax change)
5. Update all examples and documentation

**Alternative:** Update LANGUAGESPEC.md to match current implementation

---

## Performance Optimizations

### Tail Call Optimization (TCO)

**Current:** Partially implemented for recursive lambdas

**Actions:**
1. Extend TCO to regular function calls
2. Detect tail position more accurately
3. Add tail call optimization for mutual recursion
4. Benchmark improvement on recursive workloads

### Register Allocation

**Current:** Uses rbx, r12-r15 for some operations

**Actions:**
1. Implement full register allocator
2. Track register liveness
3. Minimize memory operations
4. Use more registers for temporaries
5. Benchmark improvement

### Constant Folding

**Current:** No compile-time evaluation

**Actions:**
1. Detect constant expressions at compile time
2. Evaluate and inline results
3. Eliminate dead code
4. Propagate constants through operations

---

## Tooling

### Debugger Support

**Actions:**
1. Generate DWARF debug information
2. Map machine code to source lines
3. Support gdb/lldb integration
4. Add variable inspection
5. Support breakpoints

### Better Error Messages

**Actions:**
1. Add column numbers (currently only line numbers)
2. Show context lines with error position
3. Suggest fixes for common errors
4. Add "did you mean?" for typos
5. Improve type mismatch messages

### Package Manager

**Actions:**
1. Design package format
2. Implement dependency resolution
3. Create package registry
4. Add versioning support
5. Build/install automation

---

## Platform Support

### Windows Native Support

**Current:** Linux and macOS only

**Actions:**
1. Add PE/COFF binary format support
2. Implement Windows syscalls
3. Handle Windows calling conventions
4. Test on Windows platforms

### WASM Target

**Actions:**
1. Add WebAssembly code generation backend
2. Implement WASM binary format
3. Map Flap operations to WASM instructions
4. Test in browser and Node.js

---

## Documentation

### Language Reference

**Actions:**
1. Complete API documentation for all built-ins
2. Add more examples for each feature
3. Create tutorial series
4. Document best practices
5. Add performance guide

### Compiler Internals

**Actions:**
1. Document code generation strategy
2. Explain register allocation
3. Describe binary format
4. Add architecture diagrams
5. Create contributor guide

---

## Standard Library

### Core Libraries

**Actions:**
1. String manipulation functions
2. List/map utilities
3. Math functions (beyond basic arithmetic)
4. File I/O operations
5. JSON parsing/generation
6. HTTP client/server
7. Regular expressions

---

## Testing

### Extended Test Suite

**Actions:**
1. Add fuzz testing
2. Property-based tests
3. Stress tests for memory management
4. Concurrency tests with ENet
5. Cross-platform compatibility tests

---

## Migration Guide

### Flap 1.2.0 → 1.3.0

**Actions:**
1. Document all breaking changes
2. Create migration tool/script
3. Provide compatibility layer (if possible)
4. Update all examples
5. Create migration timeline

---

## Release Criteria

**For Flap 1.3.0 Release:**
- [ ] Type system redesign complete
- [ ] All 1.2.0 tests passing with new system
- [ ] Match+string bug fixed
- [ ] Documentation updated
- [ ] Migration guide ready
- [ ] At least 2 platforms supported
- [ ] Performance parity or better than 1.2.0

---

## Non-Goals (Out of Scope for 1.3.0)

- Garbage collection (still manual memory management)
- Object-oriented features (Flap is functional)
- Generic types (universal map is the generic type)
- Traditional class hierarchies
- Async/await (use ENet instead)

---

**Note:** This is a living document. Items will be prioritized and scheduled as development progresses.

## Current Status (2025-11-19)

### ✅ ALL TESTS PASSING - Ready for 3.0 Release

### Completed Features
- ✅ pop() function fully implemented and tested
- ✅ Multiple return values working correctly (a, b = [1, 2])
- ✅ Nested loops (5+ levels) fully working with register/stack allocation
- ✅ += operator for lists (result += 42) and numbers (count += 1)
- ✅ Array indexing (xs[1] returns element at index 1)
- ✅ Register tracking to prevent clobbering
- ✅ Closures (nested function definitions)
- ✅ All 149 tests passing

### Known Limitations (Not Blocking 3.0)
- ⚠️ Local variables in lambda bodies not yet supported
  - Workaround: Use lambda parameters or expression-only bodies
  - Example: `f = x => x + 1` works, but `f = x => { y = x + 1; y }` doesn't
  - This limits complex function implementations but doesn't break core functionality

### Next Steps for Future Releases
1. Add support for local variables in lambda bodies (major feature)
2. Implement type system redesign with type tags
3. Add debugger support (DWARF info)
4. Windows/WASM platform support
