# Flap Compiler TODO

## Current Version: 1.5.0

**Status**: Enhanced C FFI with automatic type inference and header constant extraction
**Test Suite**: 206/206 tests passing (100%) - all compile-only tests pass in CI
**Target Architecture**: x86-64 Linux (primary), ARM64, RISC-V

## Recent Additions (v1.5.0)

### Completed Features ✓

#### C Header Constant Extraction
- Automatically extracts `#define` constants from C library headers
- Supports hex, decimal, binary, bitwise operations, and constant references
- Successfully extracts 450+ constants from SDL3
- Constants accessible via dot notation: `sdl.SDL_INIT_VIDEO`

#### Simple Function-like Macro Support
- Parses and expands simple function-like macros
- Handles token-pasting macros like `SDL_UINT64_C(c)` → `c##ULL`
- Enables use of macro-defined constants in FFI

#### Automatic Type Inference for C FFI
- Strings automatically cast to `cstr` (C char*)
- Numbers automatically cast to `int` by default
- Explicit casts still supported and override inference
- Example: `sdl.SDL_CreateWindow(title, width, height, 0)` - no casts needed!

#### Enhanced Type Casting
- Support for all C types: `int`, `uint32`, `cstr`, `pointer`, etc.
- Casts work in both C FFI calls and unsafe blocks
- Parser validates cast types at compile time

#### CI-Ready Testing
- All TestFlapPrograms tests pass in CI
- SDL3-dependent tests marked as compile-only
- 100% test pass rate for continuous integration

## Version 1.6.0 - Goals

Focus on completing the core language features and improving C FFI usability.

### Priority 1: Extended Unsafe Operations

**Status**: ✓ Complete - all major operations implemented

The Unsafe Language is complete for low-level systems programming.

#### Arithmetic Operations ✓
- [x] Add/Subtract: `rax <- rax + rbx` (Complete)
- [x] Multiply: `rax <- rax * rbx` (Complete - `unsafe_multiply_test.flap`)
- [x] Divide: `rax <- rax / rbx` (Complete - `unsafe_divide_test.flap`)

#### Bitwise Operations ✓
- [x] AND: `rax <- rax & rbx` (Complete - `unsafe_arithmetic_test.flap`)
- [x] OR: `rax <- rax | rbx` (Complete - `unsafe_arithmetic_test.flap`)
- [x] XOR: `rax <- rax ^ rbx` (Complete)
- [x] NOT: `rax <- ~b rbx` (Complete - `unsafe_arithmetic_test.flap`)

#### Shift Operations ✓
- [x] Shift left: `rax <- rax << 4` (Complete - `unsafe_shift_test.flap`)
- [x] Shift right: `rax <- rax >> 2` (Complete - `unsafe_shift_test.flap`)
- [x] Variable shifts with register (Complete)

#### Memory Operations ✓
- [x] Memory loads: `rax <- [rbx]` (Complete - `unsafe_stack_test.flap`)
- [x] Memory stores: `[rax] <- rbx` (Complete - `unsafe_memory_store_test.flap`)
- [x] Type casts in unsafe: `rax <- 42 as uint8` (Complete)

#### System Calls ✓
- [x] Syscall instruction: `syscall` (Complete - `unsafe_syscall_test.flap`)
  - x86-64: `syscall` (0x0F 0x05)
  - ARM64: `svc #0`
  - RISC-V: `ecall`

#### Future Enhancements
- [ ] Sized loads: `rax <- u8 [rbx]`, `rax <- u16 [rbx]`
- [ ] Signed loads: `rax <- i8 [rbx]` with sign extension
- [ ] Sized stores: `u8 [rax] <- rbx`

### Priority 2: Enhanced C FFI

**Status**: ✓ Complete - full SDL3/RayLib support

SDL3 and RayLib5 programs can now be easily written in Flap!

#### String Arguments ✓
- [x] Pass Flap strings as C char pointers (Complete)
  - Automatic conversion with `as cstr` or type inference
  - Calls `flap_string_to_cstr` runtime helper
  - Example: `sdl.SDL_CreateWindow("My Window", 800, 600, 0)`

#### Automatic Type Inference ✓
- [x] Strings automatically cast to `cstr` (Complete)
- [x] Numbers automatically cast to `int` (Complete)
- [x] Explicit casts override defaults (Complete)

#### Pointer Support ✓
- [x] Allow Flap values as C pointers (Complete)
  - Cast numbers to pointers: `ptr as pointer`
  - Pass to C functions expecting pointers

#### C Header Constants ✓
- [x] Automatically extract constants from headers (Complete)
  - 450+ constants from SDL3
  - Supports simple function-like macros
  - Access via dot notation: `sdl.SDL_INIT_VIDEO`

#### Future Enhancements
- [ ] Float arguments (use xmm0-xmm7 registers)
- [ ] Float return values
- [ ] More than 6 arguments (stack-based passing)
- [ ] Custom library search paths

### Priority 3: Memory Management Runtime

**Status**: Syntax parsed, runtime not implemented

Implement the arena allocator and defer statement runtime.

#### Arena Allocator
- [ ] Create `arena_runtime.c` (or implement in assembly)
  - `flap_arena_create(size)` - initial malloc()
  - `flap_arena_alloc(size)` - bump pointer with realloc() growth
  - `flap_arena_destroy(arena)` - free()
  - Global arena stack for nesting

- [ ] Generate arena block code
  - Call `flap_arena_create()` on block entry
  - Replace `alloc()` calls with `flap_arena_alloc()`
  - Call `flap_arena_destroy()` on block exit
  - Test: Nested arenas, multiple allocations

#### Defer Statements
- [ ] Track deferred expressions per scope
  - Add `deferredExprs` stack to FlapCompiler
  - Push expressions when encountering `defer`

- [ ] Emit cleanup code in LIFO order
  - At end of scope: emit deferred expressions in reverse
  - Before `return`: emit all pending defers
  - Test: Multiple defers, early returns

### Priority 4: Language Features

**Status**: Various

Complete remaining language features for v1.5.0.

#### Slice Expressions (ARM64 blocked)
- [ ] Implement `list[1:5]` and `string[0:10]`
  - Already works on x86-64
  - Port to ARM64 when platform issue resolved

#### Pipe Operator (ARM64 blocked)
- [ ] Implement `data | transform | filter`
  - Already works on x86-64
  - Port to ARM64 when platform issue resolved

#### Jump Expressions (ARM64 blocked)
- [ ] Implement `break` and `continue` in loops
  - Already works on x86-64
  - Port to ARM64 when platform issue resolved

### Priority 5: Testing and Quality

**Current**: 200/201 tests passing

- [ ] Reach 100% test pass rate
  - Fix remaining failing test: `unsafe_arithmetic_test` (needs multiply/divide)

- [ ] Add comprehensive C FFI tests
  - Test with real libraries (if available)
  - Test error cases (missing library, wrong types)

- [ ] Add arena allocator tests
  - Stress test with many allocations
  - Test growth behavior
  - Test nested arenas

- [ ] Add defer statement tests
  - Test LIFO ordering
  - Test with early returns
  - Test with panics (future)

### Priority 6: Documentation

- [x] Update LANGUAGE.md to v1.4.0 ✓
- [x] Document C library imports ✓
- [x] Document unsafe arithmetic and memory operations ✓
- [ ] Create C FFI tutorial with practical examples
  - SDL3 or RayLib game window
  - SQLite database access
  - File I/O with POSIX API
- [ ] Create unsafe block tutorial for systems programming
  - Custom memory allocator
  - Bit manipulation
  - Hardware register access

## Deferred to v1.6.0+

### Advanced Features
- [ ] Generics/parametric polymorphism
- [ ] Compile-time metaprogramming
- [ ] SIMD intrinsics (beyond automatic vectorization)
- [ ] Garbage collection option (alternative to arena/manual)

### Platform Support
- [ ] Resolve ARM64 macOS binary hang issue
  - Currently blocks all ARM64 development
  - May require debugging dyld or signing issues
- [ ] Windows support (PE/COFF format)
- [ ] FreeBSD support

### Optimizations
- [ ] Register allocation improvements
- [ ] Dead code elimination
- [ ] Constant propagation across functions
- [ ] Inline small functions automatically

### Standard Library
- [ ] Collections (hash map, tree, queue, stack)
- [ ] String manipulation (split, join, replace, regex)
- [ ] File I/O library (high-level wrappers)
- [ ] Network programming (sockets, HTTP)
- [ ] JSON parsing and serialization
- [ ] Date/time library

## Notes

- **Target Architecture**: Focus on x86-64 Linux until ARM64 hang is resolved
- **Philosophy**: Implement features completely before moving to next
- **Testing**: Every feature must have test coverage
- **Documentation**: Update LANGUAGE.md before marking feature complete
