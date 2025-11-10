# TODO - Actionable Implementation Tasks

This file lists concrete implementation tasks for features defined in LANGUAGE.md.

**Status indicators:**
- 🔴 **CRITICAL** - Blocking or broken functionality
- 🟡 **HIGH** - Important for language completeness
- 🟢 **MEDIUM** - Nice to have, enhances usability
- 🔵 **LOW** - Future enhancements

---

## ✅ RESOLVED - Parallel Malloc (Was Critical)

**Reference:** LANGUAGE.md Section "Parallel Loops"

**Status:** FIXED in commit 8593240

### What Was Fixed
`malloc()` calls from pthread threads were causing SIGSEGV due to incorrect stack alignment.

### The Solution
Changed thread function stack allocation from `sub rsp, 64` to `sub rsp, 56` to ensure RSP is misaligned by 8 bytes before call instructions, as required by x86-64 ABI.

### Verification
- ✅ `parallel_no_atomic` - f-strings now work in parallel loops
- ✅ `parallel_malloc_access` - direct malloc works from threads
- ✅ All parallel tests passing

**Details:** See commit 8593240 and LEARNINGS.md section "Stack Alignment Requirements"

---

## 🟡 HIGH - Implement List Operators (::, ^, _)

**Reference:** LANGUAGE.md Section "List Operations" 

### Cons Operator `::`
- [ ] Add `::` token to lexer
- [ ] Add cons expression to AST
- [ ] Implement cons codegen (create new list, prepend item)
- [ ] Test with `1 :: 2 :: 3 :: []`

### Head Operator `^`
- [ ] Add `^` as unary prefix operator in parser
- [ ] Generate code to extract first element
- [ ] Return NaN for empty list
- [ ] Test: `^[1, 2, 3]` returns `1.0`

### Tail Operator `_`
- [ ] Add `_` as unary prefix operator in parser
- [ ] Generate code to return list minus first element
- [ ] Return `[]` for empty list
- [ ] Test: `_[1, 2, 3]` returns `[2, 3]`

---

## 🟡 HIGH - Implement Reduce Pipe `|||`

**Reference:** LANGUAGE.md Section "Reduce Pipe" 

- [ ] Add `ReduceExpr` to AST (parallel to PipeExpr, ParallelExpr)
- [ ] Parse `|||` operator (lower precedence than `||`)
- [ ] Generate reduce/fold codegen with accumulator
- [ ] Use first element as initial accumulator value
- [ ] Test: `[1, 2, 3, 4, 5] ||| (acc, x) => acc + x` returns `15.0`

---

## 🟡 HIGH - Complete Result Type Built-ins

**Reference:** LANGUAGE.md Section "Result Type Operations"

### is_error() - ✅ Already implemented
- [x] Already working in compiler

### error_code() - Extract 4-letter error code
- [ ] Implement `error_code(value)` builtin
- [ ] Extract encoded error string from invalid pointer bits
- [ ] Return as Flap string (e.g., `"dv0 "`, `"nan "`)
- [ ] Test with division by zero

### unwrap_or() - Get value or default
- [ ] Implement `unwrap_or(value, default)` builtin
- [ ] Check if value is error using is_error
- [ ] Return value if success, default if error
- [ ] Test: `unwrap_or(10/0, 0.0)` returns `0.0`

---

## 🟡 HIGH - Implement Inclusive Range `..`

**Reference:** LANGUAGE.md Section "Range Loop"

- [ ] Add `..` token to lexer (distinct from `..<`)
- [ ] Parse inclusive range in parser
- [ ] Add `inclusive` flag to RangeExpr AST node
- [ ] Generate loop bounds: `start` to `end` (inclusive)
- [ ] Test: `@ i in 1..5` iterates 1, 2, 3, 4, 5

---

## 🟢 MEDIUM - Implement Random Operator `???`

**Reference:** LANGUAGE.md Section "Random Operator"

- [ ] Add `???` token to lexer
- [ ] Add random expression to AST
- [ ] Implement xoshiro256** PRNG state in runtime
- [ ] Add `getrandom()` syscall wrapper for Linux
- [ ] Initialize RNG from `SEED` env var or system entropy
- [ ] Generate code to call `_flap_random()` runtime function
- [ ] Make thread-safe for parallel code
- [ ] Test: verify `???` returns values in [0.0, 1.0)
- [ ] Test: reproducibility with `SEED=12345`

---

## 🟢 MEDIUM - Fix Atomic Operations in Parallel Loops

**Reference:** LANGUAGE.md Sections "Parallel Loops" + "Atomic Operations"

### Problem
Atomic operations fail inside `@@` loops due to register clobbering

### Tasks
- [ ] Implement context-aware register allocation for parallel sections
- [ ] Reserve registers for atomic operations in parallel contexts
- [ ] Update codegen to avoid clobbering atomic operation registers
- [ ] Test: `atomic_add()` inside `@@` loop
- [ ] Document register constraints in LEARNINGS.md

---

## 🔵 LOW - Improve CStruct Ergonomics

**Reference:** LANGUAGE.md Section "CStruct"

### Current syntax
```flap
ptr[Vec3.x.offset] <- 1.0 as float64
```

### Proposed improvement
```flap
set(ptr, Vec3.x, 1.0)
get(ptr, Vec3.x)
```

### Tasks
- [ ] Add `set(ptr, Type.field, value)` builtin
- [ ] Add `get(ptr, Type.field)` builtin
- [ ] Track field types in CStruct compiler
- [ ] Generate type-aware read/write code
- [ ] Update LANGUAGE.md examples

---

## ✅ COMPLETED

Features from LANGUAGE.md that are fully implemented:

- Sequential pipe `|`
- Parallel map `||`
- Parallel loops `@@` (all tests passing - malloc fixed!)
- Arena allocation with `alloc()`
- Move operator `!`
- Exclusive range `..<`
- Length operator `#`
- Result type with `is_error()`
- NaN/Inf propagation
- Atomic operations (outside parallel loops)
- Tail-call optimization
- C FFI
- Lambda expressions
- Maps and lists
- Spawn (basic)
- Power operator `**`

---

## Adding New Features

**Process:**
1. Specify in LANGUAGE.md with syntax, semantics, examples
2. Add actionable task to this TODO.md
3. Implement and test
4. Add passing tests to testprograms/
5. Move to Completed section in TODO.md

**Out of Scope for this file:**
- Compiler optimizations
- Error message improvements
- Code refactoring
- Performance tuning
- Build system changes

These are implementation details, not language features.
