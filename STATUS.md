# Flapc Compiler Status Report

**Date:** 2025-11-11  
**Compiler Version:** 1.3.0  
**Test Pass Rate:** 102/118 (86%)

## Recent Changes

### Removed `is_error` Builtin

Following careful review of LANGUAGE.md (the source of truth), removed the `is_error()` builtin which was not part of the official Flap specification. Error handling in Flap is done through:

- **`or!` operator**: Returns value if success, default if error
  ```flap
  result := 10 / 0
  safe := result or! 42.0  // Returns 42.0 for error
  ```

- **`.error` property**: Extracts 4-letter error code from NaN-encoded Result
  ```flap
  code := result.error  // Returns "dv0 " for division by zero
  ```

### Parser Improvements

Added `blockContainsMatchArrows()` lookahead function to properly detect match blocks by scanning for `->` or `~>` arrows before deciding parsing strategy. This prevents incorrect treatment of match blocks as simple conditionals.

### Test Cleanup

Updated tests to remove references to `is_error()` and use proper `or!` operator for error handling as specified in LANGUAGE.md.

## Current Test Status

### Passing Tests (102)

✅ All arithmetic operations  
✅ All comparison operations  
✅ All logical operations (and, or, not, xor)  
✅ All bitwise operations  
✅ Variables and assignment  
✅ Loops (sequential and parallel)  
✅ Ranges (.. and ..<)  
✅ Lambda expressions  
✅ Sequential pipe (|)  
✅ Parallel pipe (||)  
✅ Move operator (!)  
✅ C FFI  
✅ CStruct definitions  
✅ Arena allocation  
✅ Unsafe blocks  
✅ Atomic operations  
✅ Defer statements  
✅ Spawn  
✅ Random operator (???)  
✅ String operations  
✅ Printf formatting  
✅ Register allocation  

### Failing Tests (16)

The following tests are failing and need investigation/fixes:

1. **TestParallelSimpleCompiles** - Parallel loop compilation issue
2. **TestCompilationErrors** - Expected compilation failure doesn't fail
3. **TestENetCompilation** - ENet library test (missing dependency)
4. **TestENetCodeGeneration** - ENet library test (missing files)
5. **TestFlapPrograms** - General program test failures
6. **TestLambdaPrograms** - Lambda-related test failures
7. **TestListPrograms** - List operation tests
8. **TestTailOperator** - Tail operator (_) tests
9. **TestLoopPrograms** - Loop control tests
10. **TestDivisionByZeroReturnsNaN** - Division by zero error handling
11. **TestOrBangWithError** - or! operator with errors
12. **TestOrBangChaining** - Chained or! operators
13. **TestErrorPropertySimple** - .error property access
14. **TestErrorPropertyLength** - .error string length
15. **TestMapOperations** - Map update operations (segfault)
16. **TestListOperationsComprehensive** - List cons operator

## Known Issues

### 1. Map/List Update Segfault

**Symptom:** Programs with `m[0] <- value` compile but segfault at runtime

**Example:**
```flap
m := [10, 20]
m[0] <- 99   // Compiles OK, segfaults at runtime
println(m[0])
```

**Root Cause:** The `__flap_map_update` implementation has a bug in the runtime code generation.

**Status:** Needs debugging

### 2. List Cons Operator (::)

**Status:** Parser/Lexer ready, NO CODEGEN

**Example:**
```flap
list1 := [2, 3]
list2 := 1 :: list1    // Should return [1, 2, 3]
```

**What's Needed:**
- ConsExpr type in ast.go
- case *ConsExpr in codegen.go
- Tests in list_programs_test.go

### 3. or! Operator Edge Cases

**Status:** Basic functionality works, edge cases fail

**Working:**
```flap
result := 10 / 2
safe := result or! 0.0  // Returns 5.0
```

**Failing:**
```flap
result := 10 / 0
safe := result or! 42.0  // Should return 42.0, currently fails
```

**Issue:** The NaN detection or the or! codegen has edge cases not handled correctly.

### 4. .error Property Access

**Status:** Parser transforms to `_error_code_extract()`, but implementation incomplete

**Current:** Always returns empty string  
**Needed:** Extract 4-letter error code from NaN mantissa bits

### 5. Tail Operator (_)

**Status:** No lexer token, no codegen

**Example:**
```flap
rest := _[1, 2, 3]  // Should return [2, 3]
```

### 6. Reduce Pipe (|||)

**Status:** Explicitly marked as "future feature" in LANGUAGE.md

**Example:**
```flap
sum := [1, 2, 3, 4, 5] ||| (acc, x) => acc + x  // Should return 15
```

## Priority Action Items

### HIGH Priority (Blocking Core Functionality)

1. **Fix Map/List Update Segfault**
   - Debug `__flap_map_update` runtime code
   - Ensure proper memory allocation and copying
   - Test with both lists and maps

2. **Fix or! Operator Edge Cases**
   - Debug NaN detection in or! codegen
   - Verify JumpNotParity opcode generation
   - Add comprehensive tests

3. **Implement .error Property**
   - Extract error code from NaN mantissa
   - Convert bits to 4-character string
   - Return empty string for non-errors

### MEDIUM Priority (Spec Compliance)

4. **Implement Cons Operator (::)**
   - Add ConsExpr AST node
   - Generate code to prepend element to list
   - Pure functional (returns new list)

5. **Implement Tail Operator (_)**
   - Add TOKEN_UNDERSCORE (if not unary)
   - Add TailExpr AST node
   - Generate code to return list[1:]

6. **Fix Test File Dependency Loading**
   - Sibling file loading causes variable conflicts
   - Need better isolation or scope management

### LOW Priority (Nice to Have)

7. **Implement Reduce Pipe (|||)**
   - Not blocking, marked as future feature
   - Would complete the pipe operator trilogy

8. **Fix Outdated Tests**
   - Some tests use old syntax
   - Some tests expect incorrect behavior
   - Need systematic review against LANGUAGE.md

## Architecture Notes

### Parser Strategy

The parser uses recursive descent with lookahead for disambiguating constructs:
- `blockContainsMatchArrows()` scans ahead to detect match blocks
- Creates temporary lexer/parser for lookahead without side effects
- Prevents incorrect classification of blocks

### Error Handling Philosophy

Flap uses **NaN-encoded Result types** instead of exceptions:
- Division by zero returns NaN with error code
- Errors are values, not control flow
- No `try/catch`, no `throw`
- Explicit handling with `or!` operator
- Optional error inspection with `.error` property

### Memory Model

Everything is `map[uint64]float64` internally:
- **Lists:** `[length][elem0][elem1]...` (dense, position-indexed)
- **Maps:** `[count][key0][val0][key1][val1]...` (sparse, key-value pairs)
- **Empty value `[]`:** Universal empty (empty map)

## Next Steps

1. Fix the map update segfault (highest priority)
2. Complete or! operator edge cases
3. Implement .error property fully
4. Add missing operators (cons, tail)
5. Systematic test review and updates
6. Aim for 95%+ test pass rate

## Notes for Continuation

- **Source of Truth:** LANGUAGE.md v2.0.0
- **Philosophy:** Follow Flap's principles (explicit, minimal, calculated)
- **Tests:** Many tests are outdated and need updating to match LANGUAGE.md
- **Debugging:** Use `-v` flag for verbose output, but be aware of /tmp file pollution
- **CI Status:** Currently failing due to test failures, will be green when fixes complete
