# Flap Compiler TODO

## Current Version: 1.4.0

**Status**: C Library FFI complete and working
**Test Suite**: 200/201 tests passing (99.5%)
**Target Architecture**: x86-64 Linux (primary)

## Version 1.5.0 - Goals

Focus on completing the core language features and improving C FFI usability.

### Priority 1: Extended Unsafe Operations

**Status**: Partially implemented (add/sub only)

Make the Unsafe Language complete for low-level systems programming.

#### Arithmetic Operations
- [ ] Multiply: `rax <- rax * rbx`, `rax <- rax * 10`
  - File: Create `imul.go` with `ImulRegWithReg` and `ImulRegWithImm`
  - Parser: Extend `compileRegisterOp()` to handle `*` operator
  - Test: Create `programs/unsafe_multiply_test.flap`

- [ ] Divide: `rax <- rax / rbx`, `rdx:rax <- rax / rbx` (with remainder)
  - File: Create `div.go` with `DivRegByReg`
  - Note: Division produces quotient in rax, remainder in rdx
  - Test: Create `programs/unsafe_divide_test.flap`

#### Bitwise Operations
- [ ] AND: `rax <- rax & rbx`, `rax <- rax & 0xFF`
  - File: Create `and.go` with `AndRegWithReg` and `AndImmToReg`
  - Test: Bitmask operations

- [ ] OR: `rax <- rax | rbx`, `rax <- rax | 0x80`
  - File: Create `or.go` with `OrRegWithReg` and `OrImmToReg`
  - Test: Flag setting

- [ ] XOR: `rax <- rax ^ rbx`, `rax <- rax ^ 0xFFFFFFFF`
  - File: Create `xor.go` with `XorRegWithReg` and `XorImmToReg`
  - Test: Bit flipping, zeroing with `rax <- rax ^ rax`

- [ ] NOT: `rax <- ~rbx` (already implemented, verify)
  - Test: Bitwise negation

#### Shift Operations
- [ ] Shift left: `rax <- rax << 4`, `rax <- rax << cl`
  - File: Create `shift.go` with `ShlRegByImm` and `ShlRegByCL`
  - Test: Bit manipulation, multiplication by powers of 2

- [ ] Shift right (logical): `rax <- rax >> 2`
  - Function: `ShrRegByImm` and `ShrRegByCL`
  - Test: Division by powers of 2

- [ ] Shift right (arithmetic): `rax <- rax >>> 1` (sign-extending)
  - Function: `SarRegByImm` and `SarRegByCL`
  - Test: Signed division

#### Memory Operations
- [ ] Sized loads: `rax <- u8 [rbx]`, `rax <- u16 [rbx]`, `rax <- u32 [rbx]`
  - Extend `compileMemoryLoad()` to handle size prefixes
  - Zero-extend smaller values into 64-bit register
  - Test: Loading bytes, words, dwords

- [ ] Signed loads: `rax <- i8 [rbx]`, `rax <- i16 [rbx]`, `rax <- i32 [rbx]`
  - Use MOVSX instruction for sign extension
  - Test: Loading signed values

- [ ] Memory stores: `[rax] <- rbx`, `[rax + 8] <- 42`
  - Implement `compileMemoryStore()` (currently stubbed)
  - Support sized stores: `u8 [rax] <- rbx`, `u32 [rax] <- 100`
  - Test: Writing to memory

#### System Calls
- [ ] Syscall instruction: `syscall`
  - File: Create `syscall.go` with architecture-specific implementations
  - x86-64: `syscall` (0x0F 0x05)
  - ARM64: `svc #0`
  - RISC-V: `ecall`
  - Example: System calls for file I/O, process control
  - Test: Create `programs/unsafe_syscall_test.flap` (write to stdout)

### Priority 2: Enhanced C FFI

**Status**: Basic functionality complete

Improve C FFI to handle more real-world use cases.

#### String Arguments
- [ ] Pass Flap strings as C char pointers
  - Convert map[uint64]float64 string to null-terminated char array
  - Allocate temporary buffer (or use arena)
  - Example: `sdl.SDL_CreateWindow("My Window", ...)`

#### Pointer Support
- [ ] Allow Flap values as C pointers (treat as uint64)
  - Cast numbers to pointers: `ptr := 0x1000 as pointer`
  - Pass to C functions expecting pointers

#### Float Arguments and Returns
- [ ] Detect float arguments (use xmm0-xmm7 registers)
  - Extend ABI marshaling to handle floating-point
- [ ] Handle float return values
  - Convert xmm0 to float64 instead of rax to float64

#### More Than 6 Arguments
- [ ] Implement stack-based argument passing
  - Push args 7+ onto stack before call
  - Maintain 16-byte stack alignment

#### pkg-config Integration
- [ ] Use pkg-config to discover library paths
  - Run `pkg-config --libs <library>` during compilation
  - Extract library name from `-l` flag
  - Support custom library search paths

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
  - SDL2 game window
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
