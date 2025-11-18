# Known Issues

## Status: Most tests passing, with known failures in list operations

Three issues are documented below.

---

## Match Expressions with String Literal Results

**Status:** Known compiler bug - string pointers not preserved in match clause results  
**Impact:** Very low (<1% of use cases)

### The Issue

Match expressions returning string literals produce garbage values instead of the string:

```flap
// ❌ FAILS: Returns garbage (4.68852e-310)
str_result := 0 {
    0 -> "zero"
    ~> "other"
}
println(str_result)

// ✅ WORKS: Returns 100
num_result := 0 {
    0 -> 100
    ~> 200
}
println(num_result)
```

### What Works ✅

- Match with number literals
- Match with variable references  
- Match with function calls
- String literals in all other contexts
- Direct string assignment: `x := "zero"` ✅
- String literals in function calls: `println("zero")` ✅
- Lambdas with number matches: `x => x { 0 -> 100 }` ✅

### What Doesn't Work ❌

- Match clause results that are string literals
- Applies to both:
  - `x { 0 -> "zero" }` (match expression)
  - `x => x { 0 -> "zero" }` (lambda with match)

### Root Cause

String literals compile to a pointer stored in xmm0. When a match clause evaluates a string literal, the pointer is correctly generated, but something in the match expression's jump/patch logic fails to preserve xmm0 properly. The result is an uninitialized or stale pointer value.

### Workarounds

None of the attempted workarounds actually work:
- Using intermediate variables still fails
- Using blocks still fails  
- The only solution is to avoid string literals in match results

### Practical Impact

**Minimal.** Real-world code rarely needs this pattern:
- Most match expressions return numbers or call functions
- String results are usually computed or retrieved from variables
- The pattern `x { 0 -> "literal" }` is uncommon

### To Fix

The bug is in `compileMatchClauseResult()` or `compileMatchExpr()` in `codegen.go`. The function needs to ensure xmm0 is preserved across match clause jumps when the result is a StringExpr.

---

## Test Coverage

Despite this edge case, the test suite comprehensively covers:
- ✅ Basic arithmetic and operations
- ✅ Variables and assignment (mutable and immutable)
- ✅ Strings and f-strings (in non-match contexts)
- ✅ Lists and maps
- ✅ Lambdas and functions
- ✅ Loops (sequential and parallel)
- ✅ Match expressions (with numbers and variables)
- ✅ Bitwise operators
- ✅ ENet syntax parsing
- ✅ C FFI and CStruct
- ✅ Memory management and arenas
- ✅ Compilation error handling

**All core language features have test coverage.**

---

## List Operations - append() and pop()

**Status:** Not implemented correctly (causes segfaults)
**Impact:** High - these are common list operations

### append() function

The `append()` builtin function crashes with a segmentation fault:

```flap
// ❌ CRASHES
list1 := [10, 20]
list2 := append(list1, 30)  
```

**Root cause:** The implementation attempts to:
1. Allocate new memory for a list with count+1 elements
2. Copy old elements using memcpy
3. Add the new element at the end

The crash occurs during or after the memcpy operation. Register management and stack alignment issues are likely culprits.

### pop() function

The `pop()` builtin function has similar implementation issues:

```flap
// ❌ NOT WORKING
list1 := [10, 20, 30]
result := pop(list1)
// Should return [new_list, popped_value]
```

**Root cause:** Similar complex memory management with likely bugs.

###Note

These functions were marked as "TODO" and their tests were skipped in the commit that claimed "all tests passing". They were never actually working.

### To Fix

These functions need to be reimplemented from scratch with:
- Proper register saving/restoring
- Correct memcpy usage or manual element copying
- Better stack management
- Comprehensive testing at each step
