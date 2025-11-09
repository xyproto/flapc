# TODO - Actionable Issues & Improvements

Prioritized from foundational to enhancement. Focus on fixing bugs and completing core features.

## Critical Bugs 🔴

### 1. Fix parallel loop threading SEGFAULT and iteration bugs
**Priority:** CRITICAL
**Status:** BROKEN (threads crash immediately)
**Location:** `codegen.go:2166-2450`
**Session Context:** See PROMPT.md for full details

**Three Related Issues:**

**Issue 1A: Thread SEGFAULT (NEW - introduced in latest refactor)**
- Threads created via pthread_create crash with SIGSEGV immediately
- Crash occurs in or before thread entry function `_parallel_thread_entry`
- No output from threads before crash
- Test: `@@ i in 0..<2 { println(f"i={i}") }` → SEGFAULT

**Issue 1B: Loop iteration bug (PRE-EXISTING)**
- Even when not crashing, loops execute only ONE iteration per thread
- Thread 0 with range [0, 2) prints only i=0 (should print i=0, i=1)
- Thread 1 with range [2, 4) prints only i=2 (should print i=2, i=3)
- This bug existed BEFORE the refactoring (confirmed in commit 59149d3)

**Issue 1C: Barrier synchronization hang (PRE-EXISTING)**
- Program doesn't exit after loop completion
- Futex-based barrier may have race condition
- All threads wait indefinitely instead of being woken

**Recent Refactoring (Current State):**
- Changed from `clone()` syscall to `pthread_create()` for portability
- Moved loop control from hardcoded registers (r12-r14) to rbp-relative stack slots
- Proper 16-byte stack alignment (64 bytes total)
- Fixed bug: use rbx instead of rdi after saving argument pointer
- Stack layout documented in PROMPT.md

**Root Cause Analysis Needed:**
1. Where exactly does SEGFAULT occur? (use gdb/objdump)
2. Is function pointer (`_parallel_thread_entry`) being passed correctly?
3. Is rbp-relative addressing working when rsp changes during function calls?
4. Why does loop only execute once? (counter increment? condition check? jump?)
5. Is barrier futex logic correct?

**Action Items:**
- [ ] **PRIORITY 1:** Debug SEGFAULT with gdb - find exact crash instruction
- [ ] Verify `LeaSymbolToReg("rdx", "_parallel_thread_entry")` generates correct function pointer
- [ ] Test with empty loop body to isolate crash location
- [ ] Examine generated assembly with objdump
- [ ] **PRIORITY 2:** Once threads run, debug loop iteration issue
- [ ] Check counter increment code (lines 2356-2359)
- [ ] Verify loop condition and jump logic (lines 2305-2364)
- [ ] **PRIORITY 3:** Fix barrier synchronization hang
- [ ] Review futex wake/wait logic (lines 2373-2427)
- [ ] Test barrier with simple counter increment

**Alternative Approaches:**
- Option A: Debug current pthread + rbp-relative implementation
- Option B: Revert to clone() syscall, fix register clobbering differently
- Option C: Use register allocator (as user originally requested)
- Option D: Hybrid - keep pthread, simplify stack layout

**Test Command:**
```bash
go build -o flapc *.go
./flapc testprograms/parallel_no_atomic.flap -o /tmp/test 2>&1 | grep -v DEBUG
timeout 2 /tmp/test
# Expected: Print all thread iterations, clean exit
# Actual: SEGFAULT
```

---

### 2. Fix `safe_divide_result()` map allocation crash
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

### 3. Refactor printf as robust runtime function, make println use it
**Priority:** LOW (infrastructure exists, needs implementation decision)
**Status:** Framework created, helper functions ready

**Problem:**
- Current printf/println implementations are complex inline codegen
- Prone to register clobbering and edge cases
- Difficult to maintain and debug
- Would benefit from single robust implementation

**Solution Realized:**
- Format specifiers encode type information (%v=float, %d=int, %s=string, %x=hex)
- Runtime printf CAN work by interpreting arguments based on format string
- All arguments passed as float64 in xmm0-xmm7, format string provides interpretation

**Infrastructure Created:**
- ✅ printf.go: Framework for runtime printf in x86-64, ARM64, RISC-V
- ✅ printf_helper.go: Helper functions for jump patching and float-to-string conversion
- ✅ EmitFloatToStringRuntime: Runtime function that converts float64 to ASCII string
- ✅ PrintfCodeGen: Wrapper with utilities for printf code generation

**Current Status:**
- Existing inline codegen works and all tests pass
- Runtime framework exists but is not integrated into compilation
- Helper functions compile successfully

**Action Items (Deferred):**
- [ ] Complete x86-64 runtime printf implementation (format string parsing, argument handling)
- [ ] Integrate _flap_printf into compilation process (emit at startup like flap_arena_create)
- [ ] Refactor println() to call _flap_printf("%v\n", arg)
- [ ] Implement ARM64 and RISC-V printf (currently stubs)
- [ ] Test with all value types and edge cases
- [ ] Measure performance vs current inline approach

**Note:** This is a large refactoring. Current code works. Prioritize bug fixes and critical features first.

---

### 4. Fix atomic operations in parallel loops
**Priority:** MEDIUM (blocked by #1)
**Status:** Not working inside `@@` loops
**Reason:** Register allocation conflicts

**Problem:**
- Atomic ops use fixed registers (r12)
- Parallel loops may clobber these registers
- Requires parallel-aware register allocation

**Action Items:**
- [ ] First fix parallel loop threading issues (#1)
- [ ] Implement context-aware register allocation for parallel sections
- [ ] Reserve registers for atomic operations in parallel contexts
- [ ] Add test cases: atomic increment inside `@@` loop
- [ ] Document register usage constraints

---

## Core Features to Complete 🟡

### 5. Complete Result type implementation
**Priority:** HIGH
**Dependencies:** Fix #2

**Status:** Partially implemented
- ✅ `safe_divide()`, `safe_sqrt()`, `safe_ln()` - NaN propagation
- ⚠️ `safe_divide_result()` - crashes (see #2)
- ❌ `safe_sqrt_result()` - not implemented
- ❌ `safe_ln_result()` - not implemented
- ❌ Result helper methods (`then`, `map`, `unwrap_or`)

**Action Items:**
- [ ] Fix `safe_divide_result()` crash (#2)
- [ ] Implement `safe_sqrt_result()` using same pattern
- [ ] Implement `safe_ln_result()` using same pattern
- [ ] Add Result helper builtins for chaining
- [ ] Document Result usage patterns in LANGUAGE.md

---

### 6. Implement channels for CSP-style concurrency
**Priority:** HIGH (blocked by #1)
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

### 7. Implement spawn with channel-based result waiting
**Priority:** MEDIUM (blocked by #6)
**Status:** Not started
**Dependencies:** #6 (channels)
**Design:** See SPAWN_DESIGN.md

**Action Items:**
- [ ] Update spawn to return channel
- [ ] Child process writes result to channel on exit
- [ ] Parent blocks on channel recv to get result
- [ ] Add fork/join pattern examples
- [ ] Test with parallel computation tasks

---

## Error Handling & Diagnostics 🟢

### 8. Improve compile-time error messages
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

### 9. Add `???` syntax for pseudo-random numbers
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

### 10. Improve CStruct ergonomics with typed accessors
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

### 11. Re-enable strength reduction for integer contexts
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

### 12. Register allocator Phase 2/3 (local variables)
**Priority:** LOW
**Status:** Deferred pending profiling data

**Note:** Register allocator exists (register_allocator.go) but not fully integrated

**Action Items:**
- [ ] Profile real-world Flap programs for bottlenecks
- [ ] Identify hot paths that would benefit from register allocation
- [ ] Implement local variable register mapping
- [ ] Add register spilling when needed
- [ ] Measure performance gains vs complexity cost
- [ ] Consider using for parallel loop control variables (Issue #1)

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
- ✅ Switch parallel loops from clone() to pthread_create() (in progress - has bugs)
- ✅ Refactor parallel loop control to use stack slots instead of hardcoded registers (in progress - has bugs)

---

**Workflow:** Start from #1 (critical bugs), then move down. Test thoroughly at each step. Commit after each completed item.

**Current Focus:** Issue #1 is CRITICAL and blocking other parallel programming features. All effort should focus on debugging and fixing the thread SEGFAULT, then the iteration bug, then the barrier hang.

---

**Be bold in the face of complexity!** These challenges seem daunting, but with techniques from computer science, "How to Solve It?" by Polya, and decades of compiler expertise, each one is tractable. Break problems into smaller pieces, solve incrementally, test thoroughly. The journey of a thousand commits begins with a single keystroke. Stay focused on capabilities and robustness, and the Flapc compiler will become a masterpiece of systems programming.
