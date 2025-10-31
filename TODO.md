# Flapc TODO

Development roadmap and specific implementation tasks for Flap v1.4 and beyond.

## Priority 1: Register Allocator (Critical)

**Goal:** Replace ad-hoc register usage with proper linear-scan register allocation.

**Why:** Currently each expression uses registers inefficiently, leading to excessive moves and poor performance.

**Tasks:**
- [ ] Create `register_allocator.go` with live interval tracking
- [ ] Implement linear scan algorithm (simpler than graph coloring)
- [ ] Add liveness analysis pass (track variable lifetime)
- [ ] Modify codegen to query allocator for register assignments
- [ ] Emit function prologue/epilogue for callee-saved registers
- [ ] Test: Compare instruction count before/after (expect 30-40% reduction in loops)
- [ ] Benchmark: Ensure no performance regression

**Available Registers (x86-64):**
- Caller-saved (for temporaries): rax, rcx, rdx, rsi, rdi, r8, r9, r10, r11
- Callee-saved (for variables): rbx, r12, r13, r14, r15
- Float: xmm0-xmm15

## Priority 2: Arena Allocator Runtime

**Goal:** Complete the runtime implementation for arena blocks (parser already done).

**Why:** Parser recognizes `arena {...}` but doesn't generate proper runtime calls.

**Tasks:**
- [ ] Add arena runtime functions to runtime library:
  - `_flap_arena_init(size)` - Create arena (4KB default)
  - `_flap_arena_alloc(arena_ptr, size)` - Bump-pointer allocation
  - `_flap_arena_free(arena_ptr)` - Free entire arena
- [ ] Modify codegen to emit arena calls:
  - arena_init() at block entry
  - Replace alloc() with arena_alloc()
  - arena_free() at block exit (even on early return)
- [ ] Support nested arenas (arena stack in TLS)
- [ ] Test: Verify no memory leaks with valgrind
- [ ] Test: Per-frame allocations in game loop

**Example Usage:**
```flap
@ frame in 0..<1000 {
    arena {
        entities := alloc(1000 * entity_size)
        // ... frame work ...
    }  // Instant cleanup, zero fragmentation
}
```

## Priority 3: DWARF Debug Info

**Goal:** Generate .debug_line, .debug_info, .debug_frame sections for gdb/lldb support.

**Why:** Currently debugging requires reading assembly. Need source-level debugging.

**Tasks:**
- [ ] Create `dwarf.go` with DWARF v4 generation
- [ ] Generate .debug_line section:
  - Map assembly addresses to source lines
  - Enable breakpoints by line number
- [ ] Generate .debug_info section:
  - Variable names and locations
  - Function boundaries
  - Type information
- [ ] Generate .debug_frame section:
  - Stack unwinding info
  - Call stack traces
- [ ] Test with gdb: Set breakpoints, step through code, inspect variables
- [ ] Test with lldb: Same capabilities

## Priority 4: Polish Parallel Loops

**Goal:** Test edge cases and improve error handling for `@@` loops.

**Current Status:** Barrier synchronization and thread spawning work, but need more testing.

**Tasks:**
- [ ] Test edge cases:
  - Empty ranges: `@@ i in 0..<0`
  - Single iteration: `@@ i in 0..<1`
  - Very large ranges: `@@ i in 0..<1000000`
- [ ] Improve error messages:
  - When thread spawning fails
  - When barrier times out
  - Stack overflow in worker threads
- [ ] Performance tuning:
  - Benchmark overhead vs sequential
  - Tune work distribution (currently even split)
  - Cache-align barrier struct
- [ ] Add tests for atomic operations with parallel loops
- [ ] Document when to use `@@` vs `@`

## Priority 5: Optimization Passes

**Goal:** Improve generated code quality beyond register allocation.

**Tasks:**
- [ ] Constant folding: `2 + 3` → `5` at compile time
- [ ] Dead code elimination: Remove unreachable code after `ret`
- [ ] Common subexpression elimination: `x*x` used twice → compute once
- [ ] Strength reduction: `x * 8` → `x << 3`
- [ ] Loop invariant code motion: Move constant computations out of loops
- [ ] Inline small functions (1-3 instructions)

## Bug Fixes

### High Priority
- [ ] Fix cstruct_arena_test.flap - values are wrong (10.0 instead of 10.5)
  - Likely issue with write_f64 offset calculation
  - Check Vec3_SIZEOF and field offsets

### Medium Priority
- [ ] Race conditions in parallel test suite
  - Tests `test_string_different` and `test_not` fail when run in parallel
  - Need to identify shared state or codegen issue

### Low Priority
- [ ] ARM64 C import not yet implemented
  - Skip C FFI tests on ARM64/macOS
  - Need to implement DWARF debug info parsing for ARM64

## Feature Additions

### CStruct Enhancements
- [ ] Array fields: `data as [10]int32`
- [ ] Nested cstructs: `pos as Vec2` inside `Player`
- [ ] Pointer fields: `next as ptr`
- [ ] Function pointer fields: `callback as ptr`
- [ ] Dot notation sugar: `player.x` instead of manual offset calculations

### Language Features
- [ ] Defer statements: `defer file_close(f)`
- [ ] Multiple return values: `x, y := get_position()`
- [ ] String interpolation: `"x=${x}, y=${y}"`
- [ ] Range with step: `@ i in 0..<100 step 2`

## Testing

### Test Suite Improvements
- [x] Add `-short` flag support (0.3s vs 6s) - DONE
- [ ] Add benchmark suite for performance tracking
- [ ] Add fuzzing tests for parser
- [ ] Add integration tests with SDL3 window creation
- [ ] Add stress tests for parallel loops (1M+ iterations)

### Test Coverage
- [ ] Increase coverage to 80%+ (currently ~60%)
- [ ] Add tests for error conditions
- [ ] Add tests for all builtin functions
- [ ] Add tests for all atomic operations

## Documentation

### Code Documentation
- [x] README.md - User-facing overview - DONE
- [x] LANGUAGE.md - Complete language spec - DONE
- [ ] LEARNINGS.md - Design decisions and lessons learned - IN PROGRESS
- [ ] Add inline comments to complex codegen functions
- [ ] Document register allocation algorithm
- [ ] Document ELF generation process

### User Documentation
- [ ] Tutorial: "Writing your first Flap game"
- [ ] Guide: "Flap for C programmers"
- [ ] Guide: "Parallel programming in Flap"
- [ ] Reference: Complete builtin function list
- [ ] Examples: More real-world programs in testprograms/

## Infrastructure

### Build System
- [ ] Add Makefile for easier building
- [ ] Add install target: `make install`
- [ ] Add uninstall target: `make uninstall`
- [ ] Add test target: `make test`

### CI/CD
- [ ] Add ARM64 testing in GitHub Actions
- [ ] Add RISC-V testing (via QEMU)
- [ ] Add benchmark tracking over time
- [ ] Add test coverage reporting
- [ ] Add automatic release builds

### Platform Support
- [ ] Complete ARM64 support (macOS/Linux)
- [ ] Complete RISC-V support (Linux)
- [ ] Add FreeBSD support
- [ ] Test on more Linux distributions

## Performance

### Compiler Performance
- [ ] Profile compiler with pprof
- [ ] Optimize hot paths in lexer/parser
- [ ] Cache parsed files (incremental compilation)
- [ ] Parallel compilation of multiple files

### Runtime Performance
- [ ] Benchmark vs C equivalent programs
- [ ] Optimize string operations (currently slow)
- [ ] Optimize list operations (currently slow)
- [ ] SIMD optimization for map lookups (AVX-512)

## Long-term (v2.0+)

- [ ] Package manager (flap install/publish)
- [ ] LSP server for editor support
- [ ] Incremental compilation
- [ ] Hot code reloading
- [ ] Native struct types (not just cstruct)
- [ ] Generics/parametric polymorphism
- [ ] Effect system for IO/errors
- [ ] Async/await for I/O
- [ ] WebAssembly target

## Recently Completed

- [x] Type name standardization (int32, uint64, float32) - v1.3.0
- [x] CStruct constants generation (_SIZEOF, _OFFSET) - v1.3.0
- [x] Atomic operations (add, cas, load, store) - v1.3.0
- [x] Parallel loops with barrier synchronization (@@) - v1.3.0
- [x] Test suite optimization (-short flag) - v1.3.0
- [x] Updated documentation (README, LANGUAGE) - v1.3.0

---

**Note:** This TODO is living documentation. Tasks marked `[ ]` are pending, `[x]` are completed. Priority numbers indicate importance: 1=critical, 5=nice-to-have.
