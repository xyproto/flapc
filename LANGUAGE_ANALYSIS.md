# Language Design Analysis for Flap 2.0

**Date:** 2025-11-13  
**Status:** Critical inconsistency found and fixed

## Executive Summary

A **critical inconsistency** was discovered in LANGUAGE.md regarding the semantics of the `:=` operator. The specification contradicted itself about whether `:=` creates mutable or immutable variables. This has been resolved.

## The Problem

### Contradictory Statements

**Location 1 (Line ~185, "What Makes Flap Unique #17"):**
```flap
### 17. **Immutable-by-Default**
Variables are immutable unless explicitly mutable:
x := 5        // Immutable (can't change)  ❌ WRONG
y <- 10       // Mutable (can change)
```

**Location 2 (Line ~805, "Variables and Assignment"):**
```flap
### Mutable Variables
y := 42       // Mutable  ✅ CORRECT
y <- 100      // OK, assignment
```

These directly contradict each other! The same operator `:=` was described as creating both immutable and mutable variables.

## The Core Issue

The language spec failed to distinguish between TWO orthogonal concepts:

1. **Variable Mutability** - Can you reassign the variable name?
2. **Value Mutability** - Can you mutate the contents of a collection?

## Current Compiler Behavior

```bash
# Test 1: Immutable variable
nums = [1, 2, 3]
nums[0] <- 99        # ✅ ERROR: "cannot modify immutable list"

# Test 2: Mutable variable  
nums := [1, 2, 3]
nums[0] <- 99        # ❌ SEGFAULT: tries to write to .rodata
```

The compiler **knows** the semantic difference but the implementation is incomplete. `:=` collections are still being placed in read-only memory.

## The Solution

### Clarified Semantics

| Operator | Variable | Value | Storage | Example |
|----------|----------|-------|---------|---------|
| `=` | Immutable | Immutable | `.rodata` | `x = [1,2,3]` |
| `:=` | Mutable | Mutable | Heap/Arena | `x := [1,2,3]` |

### `=` Means Immutable Everything
```flap
x = 42              // Immutable variable
x <- 100            // ERROR: can't reassign
x += 10             // ERROR: can't modify

list = [1, 2, 3]    // Immutable variable, immutable list
list <- [4, 5, 6]   // ERROR: can't reassign variable
list[0] <- 99       // ERROR: can't mutate immutable list
```

### `:=` Means Mutable Everything
```flap
y := 42             // Mutable variable
y <- 100            // ✅ OK: reassign variable
y += 10             // ✅ OK: modify variable

nums := [1, 2, 3]   // Mutable variable, mutable list
nums <- [4, 5, 6]   // ✅ OK: reassign variable to new list
nums[0] <- 99       // ✅ OK: mutate list contents (must allocate in writable memory!)
```

## Changes Made to LANGUAGE.md

### 1. Fixed Section #17 "Immutable-by-Default"
- **Before:** Contradictory examples
- **After:** Clear examples showing both `=` and `:=` with correct semantics
- Added explicit note: "The `:=` operator means both the variable AND its value are mutable"

### 2. Rewrote "Variables and Assignment" Section
- Split into clear subsections: "Immutable by Default" and "Mutable Variables and Values"
- Added comprehensive examples for both scalar and collection values
- Clarified ERROR cases explicitly

### 3. Enhanced "Update Operator" Section
- Added examples showing both variable updates and collection element updates
- Clearly marked what requires `:=` vs what works with `=`
- Added ERROR cases

### 4. Added "Mutability Semantics Clarified" Section
- Comparison table showing `=` vs `:=` semantics
- Examples for all combinations
- **Implementation note:** Explicitly states immutable collections go in `.rodata`, mutable ones in heap/arena

## Implementation Requirements

The language semantics are now clear. The implementation must match:

### What Works Today ✅
- `=` creates immutable bindings (correctly enforced)
- `=` with collections rejects mutation attempts (correctly enforced)
- `:=` allows variable reassignment (works)

### What Needs Fixing ❌
- **Critical:** `:=` with list/map literals must allocate in writable memory, not `.rodata`
- Currently causes segfault when trying to modify elements

### The Fix
In `codegen.go`, when generating code for `:=` with ListExpr or MapExpr:

```go
// Current (broken):
// - Literal data stored in .rodata
// - Return pointer to .rodata
// - Mutation attempts segfault

// Required:
if isMutable {
    // 1. Place literal template in .rodata
    // 2. Allocate writable memory (malloc or arena)
    // 3. Copy from .rodata to writable memory
    // 4. Return pointer to writable copy
} else {
    // Keep current behavior
    // Return direct pointer to .rodata
}
```

## Rejected Alternatives

### Option: Add `mut` Keyword
```flap
nums := mut [1, 2, 3]   // Explicit mutable list
```

**Rejected because:**
- Adds complexity without clear benefit
- `:=` already signals intent to mutate
- Violates Flap's minimalism principle
- User would need to think about two orthogonal choices

### Option: Separate Variable and Value Mutability
```flap
x := [1, 2, 3]      // Mutable variable, immutable value
x := mut [1, 2, 3]  // Mutable variable, mutable value
```

**Rejected because:**
- Too complex for the common case
- Most languages don't make this distinction
- `:=` clearly signals "I want to mutate things"
- Adds cognitive load

## Impact Assessment

### Tests Affected
- **14 failing tests** for list/map mutation (will pass once implementation fixed)
- **0 tests** need to be rewritten (semantics match existing tests)

### Breaking Changes
- **None** - This clarifies existing intent, doesn't change behavior
- The implementation was already trying to enforce these semantics

### Documentation Updated
- ✅ LANGUAGE.md sections 17, Variables and Assignment, Update Operator
- ✅ TODO.md updated to reflect language clarification
- ✅ This analysis document created

## Next Steps

1. **Fix implementation** in `codegen.go`:
   - Allocate `:=` collections in writable memory
   - Keep `=` collections in `.rodata`

2. **Verify tests pass**:
   - list_update_* tests (14 tests)
   - map_update tests

3. **Consider arena allocation**:
   - Short term: Use malloc for `:=` collections
   - Long term: Use arena allocator when stable

4. **Document memory model**:
   - Add section to LANGUAGE.md about memory layout
   - Explain .rodata vs heap/arena distinction

## Conclusion

The language specification had a critical ambiguity that has been resolved. The semantics are now clear and consistent:

- **`=` is immutable all the way down**
- **`:=` is mutable all the way down**

This is simple, intuitive, and matches user expectations from other systems programming languages.

The implementation now needs to catch up to match the clarified specification.

---

**Commit:** `99bcab1` - "Fix critical language spec inconsistency: clarify := mutability semantics"  
**Files Changed:** `LANGUAGE.md`, `TODO.md`  
**Lines Changed:** +107 insertions, -30 deletions
