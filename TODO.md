# TODO - Actionable Issues & Improvements

Prioritized from foundational to enhancement. Focus on fixing bugs and completing core features.

## Critical Bugs 🔴

### 1. Fix `safe_divide_result()` map allocation crash
**Priority:** HIGH
**Status:** Broken (segfault on map access)
**Location:** `codegen.go:9889-9962`

**Problem:**
- `safe_divide_result()` creates arena-allocated maps but crashes when indexing
- Map format should be `[count:f64][key0:f64][val0:f64]...` (all float64)
- Pointer conversion via `Cvtsi2sd(xmm0, rbx)` may be incorrect
- Map indexing expects pointer in specific format

**Action Items:**
- [ ] Debug pointer-to-float64 conversion in `safe_divide_result`
- [ ] Verify map header format matches existing Flap map implementation
- [ ] Test with simple arena-allocated map before adding division logic
- [ ] Add comprehensive test cases for Result type access

**Workaround:** Use `safe_divide()` which returns Inf/NaN instead

---

### 2. Fix println not displaying float values from variables
**Priority:** MEDIUM
**Status:** Broken

**Problem:**
- `println(variable)` where variable is float shows nothing
- Works: `println(42.0)` (literal)
- Broken: `x := 42.0; println(x)` (shows empty)
- Affects debugging and user experience

**Action Items:**
- [ ] Investigate println codegen for variable vs literal handling
- [ ] Check if XMM register state is preserved
- [ ] Verify float-to-string conversion in runtime
- [ ] Add test cases for println with all variable types

---

### 3. Fix atomic operations in parallel loops
**Priority:** MEDIUM
**Status:** Not working inside `@@` loops
**Reason:** Register allocation conflicts

**Problem:**
- Atomic ops use fixed registers (r12)
- Parallel loops may clobber these registers
- Requires parallel-aware register allocation

**Action Items:**
- [ ] Implement context-aware register allocation for parallel sections
- [ ] Reserve registers for atomic operations in parallel contexts
- [ ] Add test cases: atomic increment inside `@@` loop
- [ ] Document register usage constraints

---

## Core Features to Complete 🟡

### 4. Complete Result type implementation
**Priority:** HIGH
**Dependencies:** Fix #1

**Status:** Partially implemented
- ✅ `safe_divide()`, `safe_sqrt()`, `safe_ln()` - NaN propagation
- ⚠️ `safe_divide_result()` - crashes (see #1)
- ❌ `safe_sqrt_result()` - not implemented
- ❌ `safe_ln_result()` - not implemented
- ❌ Result helper methods (`then`, `map`, `unwrap_or`)

**Action Items:**
- [ ] Fix `safe_divide_result()` crash (#1)
- [ ] Implement `safe_sqrt_result()` using same pattern
- [ ] Implement `safe_ln_result()` using same pattern
- [ ] Add Result helper builtins for chaining
- [ ] Document Result usage patterns in LANGUAGE.md

---

### 5. Implement channels for CSP-style concurrency
**Priority:** HIGH
**Status:** Not started
**Design:** See CHANNELS_AND_ENET_PLAN.md Part 1

**Required for:**
- Spawn with result waiting
- Inter-thread communication
- Actor patterns

**Action Items:**
- [ ] Design channel API: `chan_create(capacity)`, `chan_send()`, `chan_recv()`
- [ ] Implement buffered channels with ring buffer
- [ ] Implement blocking send/recv with futex
- [ ] Add select statement for multiple channels
- [ ] Add test cases for producer-consumer patterns

---

### 6. Implement spawn with channel-based result waiting
**Priority:** MEDIUM
**Status:** Not started
**Dependencies:** #5 (channels)
**Design:** See SPAWN_DESIGN.md

**Action Items:**
- [ ] Update spawn to return channel
- [ ] Child process writes result to channel on exit
- [ ] Parent blocks on channel recv to get result
- [ ] Add fork/join pattern examples
- [ ] Test with parallel computation tasks

---

## Error Handling & Diagnostics 🟢

### 7. Improve compile-time error messages
**Priority:** MEDIUM
**Status:** In progress

**Action Items:**
- [ ] Convert remaining `compilerError()` calls to use ErrorCollector
- [ ] Add column number tracking to all errors
- [ ] Add "did you mean?" suggestions for undefined variables
- [ ] Implement error recovery (continue after first error)
- [ ] Add negative test suite (intentionally wrong code)

---

## Language Enhancements 🔵

### 8. Add `???` syntax for pseudo-random numbers
**Priority:** LOW
**Status:** Not implemented

**Spec:**
- Returns float64 in range [0.0, 1.0)
- Use `SEED` env var if set, else UNIX timestamp
- Use Linux `getrandom()` syscall for quality
- Initialize RNG state at program start

**Action Items:**
- [ ] Add `???` token to lexer
- [ ] Add UnaryExpr case for `???` in parser
- [ ] Implement RNG state in runtime (xoshiro256**)
- [ ] Add `getrandom()` syscall wrapper
- [ ] Codegen: emit call to `_flap_random()` helper
- [ ] Test: verify distribution, reproducibility with SEED

---

### 9. Improve CStruct ergonomics with typed accessors
**Priority:** LOW
**Status:** Not implemented

**Problem:**
Current: `write_f32(ptr, Vec3_x_OFFSET as int32, 1.0)`
Desired: `set(ptr, Vec3.x, 1.0)`

**Action Items:**
- [ ] Add `alloc_struct(Type)` builtin - returns typed pointer
- [ ] Add `set(ptr, Type.field, value)` builtin - type-aware write
- [ ] Add `get(ptr, Type.field)` builtin - type-aware read
- [ ] Update CStruct compiler to track field types
- [ ] Generate field accessor metadata
- [ ] Update LANGUAGE.md with new syntax examples

---

## Optimizations 🟣

### 10. Re-enable strength reduction for integer contexts
**Priority:** LOW
**Status:** Disabled (broke float operations)

**Currently disabled:**
- `x * 2^n → x << n`
- `x / 2^n → x >> n`
- `x % 2^n → x & (2^n-1)`

**Action Items:**
- [ ] Add AST type annotations (float vs int context)
- [ ] Only apply strength reduction in `unsafe` blocks
- [ ] Add `@integer` or `@bitwise` annotations for opt-in
- [ ] Test both float and integer paths separately
- [ ] Measure performance impact

---

### 11. Register allocator Phase 2/3 (local variables)
**Priority:** LOW
**Status:** Deferred pending profiling data

**Action Items:**
- [ ] Profile real-world Flap programs for bottlenecks
- [ ] Identify hot paths that would benefit from register allocation
- [ ] Implement local variable register mapping
- [ ] Add register spilling when needed
- [ ] Measure performance gains vs complexity cost

---

## Completed ✅

- ✅ Refactor parser.go into separate files (parser, codegen, optimizer, utils)
- ✅ Implement register allocator Phase 1 (loop iterator optimization - 20-30% speedup)
- ✅ Fix atomic operations register conflicts (r11 → r12)
- ✅ Implement railway-oriented error handling (ErrorCollector)
- ✅ Fix undefined function errors at compile-time (not link-time)
- ✅ Fix lambda epilogue stack corruption
- ✅ Fix optimizer strength reduction breaking float operations
- ✅ Fix parallel map operator (`||`) segfaults
- ✅ Design hybrid error handling (NaN propagation + Result types)
- ✅ Implement NaN/Inf helper functions (`is_nan`, `is_finite`, `is_inf`)
- ✅ Implement safe arithmetic with NaN propagation (`safe_divide`, `safe_sqrt`, `safe_ln`)

---

**Workflow:** Start from #1 (critical bugs), then move down. Test thoroughly at each step. Commit after each completed item.

---

**Be bold in the face of complexity!** These challenges seem daunting, but with techniques from computer science, "How to Solve It?" by Polya, and decades of compiler expertise, each one is tractable. Break problems into smaller pieces, solve incrementally, test thoroughly. The journey of a thousand commits begins with a single keystroke. Stay focused on capabilities and robustness, and the Flapc compiler will become a masterpiece of systems programming.
