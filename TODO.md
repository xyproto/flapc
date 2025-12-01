# Flap TODO

## Current Status (2025-12-01)

### Working Features ✅
- **Match expression return values FIXED!** ✅ 
  - `result := x { 5 => 42 ~> 99 }` now correctly returns 42/99
  - Supports both `=>` and `->` arrows in match clauses
  - Guardless matches still correctly wrapped in lambdas
- **Arena allocator FULLY IMPLEMENTED!** ✅
  - 100% libc-free memory management on Linux
  - Uses mmap/mremap/munmap syscalls directly (no malloc/realloc/free)
  - Dynamic arena growth with 1.3x scaling using mremap
  - Initial 1MB arena grows automatically as needed
  - Proper cleanup at program exit with munmap syscall
  - All 31 malloc calls replaced with arena allocator
  - Fixed: No more free() on arena memory (recursion now works!)
  - Platform-specific: syscalls on Linux, C functions on Windows/macOS
- **Number to string conversion PURE ASSEMBLY!** ✅
  - `_flap_itoa` implemented in pure x86_64 assembly
  - No sprintf dependency - completely libc-free
  - Handles positive, negative, and zero correctly
  - Used for all `x as string` conversions
- String to string casts: `s as string`
- **Windows SDL3 support WORKING!** ✅
  - No libc dependencies for core operations
  - Windows programs compile and run correctly  
  - SDL3 example works on Windows via Wine
- All core language features functional
- **All tests passing** (arithmetic, basic programs, lambdas working)

### Platform Support
- ✅ Linux x86_64: Fully working with mmap-based arenas
- ✅ Windows x86_64: Fully working (tested via Wine)
- 🚧 Linux ARM64: 95% complete (needs arena implementation)
- ❌ Linux RISC-V64: Not yet implemented
- ❌ Windows ARM64: Not yet implemented
- ❌ macOS ARM64: Not yet implemented (will need libc)

### Known Limitations
- `readln()` builtin removed (was using libc getline)
- Higher-order function tests have syntax errors (need fixing)
- Some C functions still used: pow for ** (can be replaced with pure assembly)
- macOS will need libc for syscalls (no direct syscall support)
- printf still uses libc (but can be replaced with write syscall + itoa)

## Core Features

### Parser ✅
- Track column positions for better error messages ✅
- Re-evaluate blocks-as-arguments syntax

### Optimizer
- Re-enable when type system is complete
- Add integer-only optimizations for `unsafe` blocks

### Code Generation
- Add explicit float32/float64 conversions where needed
- Replace malloc/realloc with mmap syscalls (Linux/BSD only, keep libc for macOS)
- Implement pure assembly float-to-string (avoid sprintf dependency)
- Increase default arena size or add auto-growth
- Optimize O(n²) algorithms

### Type System
- Complete type inference
- Ensure C types integrate with Flap's universal type
- Add runtime type checking (optional)

### Standard Library
- Expand minimal runtime
- Add common game utilities
- Document all builtins

### Code Quality ✅
- Fixed ARM64 type safety (uint8 shift overflow warnings) ✅
- All go vet warnings resolved (except intentional unsafe.Pointer uses) ✅
- Test coverage: 23.5% with 208+ test functions ✅
- Comprehensive error handling tests added ✅

## Known Issues

### printf Implementation
- Calculate proper offset in printf
- Improve float-to-string conversion
- Add ARM64/RISC-V assembly versions

### RISC-V Backend
- Load actual addresses for rodata symbols
- Implement PC-relative loads
- Add CSR instructions

### Test Fixes
- Fix superscript character printing
- Fix "bare match clause" error

## Future Enhancements

- Hot reload improvements (patch running process via IPC)
- WASM target
- WebGPU bindings
- More comprehensive test suite
- Performance profiling tools
- Interactive REPL
- Language server protocol support
- Package manager
# Flap Implementation Plan - Core Features

This document tracks the implementation of the 3 critical features for Flap's completion.

## Priority 1: Complete Functional Support ⏳

### Status: IN PROGRESS

### Components:

#### 1.1 Blocks as Arguments ❌
**Goal:** Allow passing lambdas and blocks as function parameters

**Current State:**
- Can store lambdas in variables: `f := x -> x * 2`
- Can compose function calls: `add_five(double(10))`
- Cannot pass lambdas as parameters: `apply(f, x)` fails

**What's Needed:**
- Allow function parameters to be called
- Runtime support for lambda-typed parameters
- Type checking for callable parameters

**Implementation Steps:**
1. Add lambda type detection in codegen
2. Generate indirect call code for parameter functions
3. Test with higher-order functions (map, filter, reduce)

#### 1.2 Function Composition (`<>` operator) ❌
**Goal:** `f <> g` creates `x -> f(g(x))`

**Current State:**
- Operator added to grammar ✅
- Parser support complete ✅
- AST node exists ✅
- Codegen stub exists ✅
- Full implementation blocked on closure capture

**What's Needed:**
- Generate wrapper functions that capture both lambdas
- Proper closure environment management
- Memory management for composed closures

**Implementation Approach:**
```
compose(f, g):
  1. Allocate environment {f_ptr, g_ptr}
  2. Generate wrapper: wrapper(x, env) -> f(g(x))
  3. Return closure {wrapper_ptr, env_ptr}
```

#### 1.3 Pipeline Operator (`|`) with Lambdas 🚧
**Goal:** `[1,2,3] | x -> x * 2` works properly

**Current State:**
- Basic pipeline exists
- Returns zeros instead of mapped values
- Needs proper lambda application over lists

**What's Needed:**
- Fix list iteration in pipeline
- Apply lambda to each element
- Collect results into new list

---

## Priority 2: Arena Allocator ✅

### Status: **COMPLETE!**

### Goal:
Replace malloc with arena allocation for all Flap data structures (strings, lists, maps).

### Current State: **IMPLEMENTED**
- ✅ All malloc/realloc/free replaced with arena allocator
- ✅ Uses mmap syscalls directly (no libc)
- ✅ 1MB initial arena with 1.3x growth
- ✅ Proper cleanup with munmap at exit
- ✅ Complete memory isolation per arena scope

### Implementation Plan:

#### 2.1 Arena Allocator Core
```
Arena structure:
- Base pointer
- Current pointer
- Size/capacity
- Parent arena (for nested scopes)
```

**Features:**
- Bump allocator (fast O(1) allocation)
- Frame-based cleanup (game loop arenas)
- Scope-based cleanup (function arenas)
- Zero-cost deallocation (just reset pointer)

#### 2.2 Integration Points

**String Operations:**
- String concatenation
- String slicing
- f-string interpolation

**List Operations:**
- List creation `[1, 2, 3]`
- List append `+=`
- List mapping `|`

**Map Operations:**
- Map literal `{x: 10, y: 20}`
- Map update/insertion
- Map growth/rehashing

#### 2.3 Memory Strategy

**Arena Types:**
1. **Global Arena** - Program lifetime (constants, globals)
2. **Frame Arena** - Per-game-frame (temp objects)
3. **Level Arena** - Per-game-level (level data)
4. **Function Arena** - Per-function-call (locals)

**API:**
```flap
arena.frame()     // Create frame-scoped arena
arena.global()    // Use global arena
arena.temp()      // Temporary scratch arena
```

#### 2.4 Implementation Steps

1. Create arena allocator in Go runtime helper
2. Replace all malloc calls with arena allocs
3. Add arena switching based on scope
4. Implement automatic cleanup at frame/function boundaries
5. Add `defer` support for manual cleanup
6. Benchmark vs malloc

**Estimated Impact:**
- 5-10x faster allocation
- Deterministic memory usage
- Zero fragmentation
- Perfect for game dev

---

## Priority 3: Re-enable Optimizer ❌

### Status: NOT STARTED (blocked on type system)

### Current State:
- Optimizer exists but disabled
- Comment says: "Re-enable when type system is complete"

### What's Needed Before Enabling:

#### 3.1 Type System Completion
- [ ] Complete type inference
- [ ] Track types through pipeline
- [ ] Handle C FFI types correctly
- [ ] Distinguish int vs float operations

#### 3.2 Optimizer Improvements Needed

**Current Optimizations to Fix:**
1. **Integer-only paths** - Use integer math when possible
2. **Constant folding** - Evaluate constants at compile time
3. **Dead code elimination** - Remove unreachable code
4. **Tail call optimization** - Convert recursion to loops
5. **Inline small functions** - Remove call overhead

**New Optimizations to Add:**
1. **SIMD vectorization** - Use AVX/NEON for lists
2. **Strength reduction** - Replace expensive ops with cheap ones
3. **Loop unrolling** - Unroll small fixed loops
4. **Register allocation** - Better register usage
5. **Peephole optimization** - Local instruction patterns

#### 3.3 Implementation Steps

1. Audit current optimizer code
2. Fix type-related assumptions
3. Add optimization flags (`-O0`, `-O1`, `-O2`, `-O3`)
4. Test each optimization in isolation
5. Create benchmark suite
6. Enable by default at `-O1`

**Safety:**
- Keep `-O0` (no optimization) as default initially
- Add `--unsafe-optimize` for aggressive opts
- Verify correctness with test suite

---

## Success Criteria

### Functional Support Complete When:
- [x] `<>` operator in grammar and parser
- [ ] Blocks can be passed as function arguments
- [ ] Higher-order functions work: `map`, `filter`, `compose`
- [ ] Pipeline with lambdas works correctly
- [ ] All compose_test.go tests pass

### Arena Allocator Complete When:
- [ ] All string/list/map ops use arenas
- [ ] Frame-based cleanup works
- [ ] `c.malloc` still available for C interop
- [ ] Benchmark shows >5x speedup
- [ ] Zero memory leaks in test suite

### Optimizer Complete When:
- [ ] Type inference tracks all expressions
- [ ] Can distinguish int/float operations
- [ ] All optimizations pass test suite
- [ ] Benchmark suite shows measurable gains
- [ ] Default build is optimized

---

## Development Approach

**Bottom-Up Priority:**
1. Get functional support working (enables everything else)
2. Implement arena allocator (performance foundation)
3. Re-enable optimizer (cherry on top)

**Testing Strategy:**
- Write tests first for each feature
- Use existing test infrastructure
- Add benchmarks for performance claims
- Test on all platforms (x86_64, ARM64, Windows)

**Commit Strategy:**
- Small, focused commits
- Keep tests passing at each commit
- Document breaking changes
- Update TODO.md as features complete

---

*Status: Functional and arena support in progress*
