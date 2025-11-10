# TODO - Language Feature Implementation

This file contains only actionable items related to features defined in LANGUAGE.md. Items are ordered by priority for completing the language specification.

## Critical - Fix Parallel Loops (`@@` syntax)

**Status:** Broken - 2 of 28 parallel tests failing
**LANGUAGE.md Reference:** Section "Parallel Loops"

**Issue:** Calling `malloc()` from within pthread-created threads causes SIGSEGV.

**Failing Test Cases:**
1. `parallel_no_atomic` - f-strings call malloc internally
2. `parallel_malloc_access` - direct malloc from thread

**Root Cause:**
- Stack alignment issue before `call malloc` (x86-64 ABI requires 16-byte alignment)
- Thread-local storage (TLS) not properly initialized for glibc malloc
- Missing thread setup for glibc malloc arena

**Action Items:**
- [ ] Ensure 16-byte stack alignment before all `call` instructions in thread functions
- [ ] Initialize TLS properly for pthread threads
- [ ] Test with alternative malloc implementations (jemalloc, tcmalloc)
- [ ] Once fixed, re-enable f-strings in parallel loops

**Workaround:** Pre-allocate all memory in parent thread before parallel loop.

**Test Command:**
```bash
go test -v -run "TestFlapPrograms/parallel"
```

---

## High Priority - Complete Result Type

**Status:** Partially implemented
**LANGUAGE.md Reference:** Section "Error Handling - Result Type"

**Currently Working:**
- ✅ `safe_divide()` - returns NaN on division by zero
- ✅ `safe_sqrt()` - returns NaN for negative input
- ✅ `safe_ln()` - returns NaN for non-positive input

**Not Working:**
- ⚠️ `safe_divide_result()` - crashes on map access
- ❌ `safe_sqrt_result()` - not implemented
- ❌ `safe_ln_result()` - not implemented
- ❌ Result helper methods (`then`, `map`, `unwrap_or`)

**Action Items:**
- [ ] Fix `safe_divide_result()` pointer-to-float64 conversion
- [ ] Verify arena-allocated map format matches Flap map implementation
- [ ] Implement `safe_sqrt_result()` using same pattern
- [ ] Implement `safe_ln_result()` using same pattern
- [ ] Add Result chaining builtins: `result.then()`, `result.map()`, `result.unwrap_or()`
- [ ] Document Result usage patterns and examples in LANGUAGE.md

---

## High Priority - Fix Atomic Operations in Parallel Loops

**Status:** Not working inside `@@` loops
**LANGUAGE.md Reference:** Sections "Parallel Loops" + "Atomic Operations"

**Problem:** Atomic operations use fixed registers that may be clobbered by parallel loop infrastructure.

**Action Items:**
- [ ] Implement context-aware register allocation for parallel sections
- [ ] Reserve registers for atomic operations in parallel contexts
- [ ] Test: atomic increment inside `@@` loop
- [ ] Document register usage constraints

---

## Medium Priority - Implement Channels (CSP)

**Status:** Not implemented
**LANGUAGE.md Reference:** Section "Channels" (if present)

**Note:** Check if channels are actually specified in LANGUAGE.md. If not, this should be added to LANGUAGE.md first.

**Action Items:**
- [ ] Define channel syntax and semantics in LANGUAGE.md
- [ ] Design channel API: `chan_create(capacity)`, `chan_send()`, `chan_recv()`
- [ ] Implement buffered channels with ring buffer
- [ ] Implement blocking send/recv with futex
- [ ] Add `select` statement for multiple channels
- [ ] Add test cases for producer-consumer patterns

---

## Medium Priority - Implement `spawn` with Result Waiting

**Status:** Not implemented
**LANGUAGE.md Reference:** Section "Concurrency - spawn" (if present)

**Dependencies:** Requires channels implementation

**Note:** Check if spawn result waiting is specified in LANGUAGE.md. If not, add specification first.

**Action Items:**
- [ ] Define spawn result semantics in LANGUAGE.md
- [ ] Update spawn to return channel
- [ ] Child process writes result to channel on exit
- [ ] Parent blocks on channel recv to get result
- [ ] Add fork/join pattern examples

---

## Low Priority - Implement `???` Pseudo-Random Syntax

**Status:** Not implemented
**LANGUAGE.md Reference:** Section "Operators - Random Number Generation" (if present)

**Specification:**
- Returns float64 in range [0.0, 1.0)
- Use `SEED` env var if set, else UNIX timestamp
- Use Linux `getrandom()` syscall for quality
- Initialize RNG state at program start

**Action Items:**
- [ ] Add `???` token to lexer
- [ ] Add UnaryExpr case for `???` in parser
- [ ] Implement RNG state in runtime (xoshiro256**)
- [ ] Add `getrandom()` syscall wrapper
- [ ] Generate call to `_flap_random()` runtime function
- [ ] Test: verify distribution, reproducibility with SEED
- [ ] Document behavior and seeding in LANGUAGE.md

---

## Low Priority - Improve CStruct Ergonomics

**Status:** Partially implemented
**LANGUAGE.md Reference:** Section "C Interop - CStruct"

**Current Syntax:**
```flap
write_f32(ptr, Vec3_x_OFFSET as int32, 1.0)
```

**Desired Syntax:**
```flap
set(ptr, Vec3.x, 1.0)
```

**Action Items:**
- [ ] Add `alloc_struct(Type)` builtin - returns typed pointer
- [ ] Add `set(ptr, Type.field, value)` builtin - type-aware write
- [ ] Add `get(ptr, Type.field)` builtin - type-aware read
- [ ] Update CStruct compiler to track field types
- [ ] Generate field accessor metadata
- [ ] Update LANGUAGE.md with new syntax examples

---

## Completed ✅

Features from LANGUAGE.md that are fully working:

- ✅ Basic parallel loops (`@@` syntax) - 26/28 tests passing
- ✅ Parallel map operator (`||`)
- ✅ NaN/Inf propagation (`is_nan`, `is_finite`, `is_inf`)
- ✅ Safe arithmetic with NaN (`safe_divide`, `safe_sqrt`, `safe_ln`)
- ✅ Atomic operations (outside parallel loops)
- ✅ Tail-call optimization
- ✅ Arena memory management
- ✅ C FFI (functions and data)
- ✅ Lambda expressions
- ✅ For loops, while loops, if/else
- ✅ Maps (unified type system)
- ✅ Spawn (basic process spawning)

---

## Notes

**Adding New Language Features:**

When proposing a new language feature:
1. **First** add specification to LANGUAGE.md with syntax, semantics, examples
2. **Then** add implementation task to this TODO.md
3. **Finally** implement and test

This ensures language design is intentional, not accidental.

**Out of Scope:**

The following are implementation details, not language features:
- Compiler optimizations (strength reduction, register allocation, etc.)
- Error message quality
- Code organization and refactoring
- Performance tuning
- Build system improvements

These belong in implementation notes or commit messages, not in language TODO.
