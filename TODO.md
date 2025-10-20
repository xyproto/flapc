# Flap Compiler TODO

**Status**: 206/206 tests passing (100%)
**Target Architecture**: x86-64 Linux (primary), ARM64, RISC-V

## Actionable Tasks

Sorted by usefulness for game development and demoscene production, then by completeness needs.

### Game Development & Demoscene Priority

- [ ] **Float arguments in C FFI** - Use xmm0-xmm7 registers for passing floats to C functions (needed for graphics APIs)
- [ ] **Float return values in C FFI** - Retrieve float results from C functions (needed for math libraries)
- [ ] **Arena allocator runtime** - Implement fast memory allocation for game objects and demo effects
  - Create `arena_runtime.c` with `flap_arena_create()`, `flap_arena_alloc()`, `flap_arena_destroy()`
  - Generate arena block code (call create on entry, destroy on exit)
  - Support nested arenas for hierarchical allocation
- [ ] **Defer statements runtime** - Implement cleanup code for resource management
  - Track deferred expressions per scope in FlapCompiler
  - Emit deferred code in LIFO order at scope exit and before returns
- [ ] **Sized memory loads in unsafe** - `rax <- u8 [rbx]`, `rax <- u16 [rbx]` for packed data structures
- [ ] **Sized memory stores in unsafe** - `u8 [rax] <- rbx` for writing packed formats
- [ ] **Signed loads in unsafe** - `rax <- i8 [rbx]` with sign extension for audio/image processing
- [ ] **More than 6 C function arguments** - Stack-based argument passing for complex APIs
- [ ] **SIMD intrinsics** - Vector operations for audio DSP, graphics effects, particle systems
- [ ] **Custom library search paths** - Allow loading libraries from non-standard locations

### Language Completeness Priority

- [ ] **Slice expressions** - Implement `list[1:5]` and `string[0:10]` (works on x86-64, needs ARM64 port)
- [ ] **Pipe operator** - Implement `data | transform | filter` (works on x86-64, needs ARM64 port)
- [ ] **Jump expressions** - Implement `break` and `continue` in loops (works on x86-64, needs ARM64 port)
- [ ] **Generics/parametric polymorphism** - Type-safe generic data structures and functions
- [ ] **Compile-time metaprogramming** - Code generation at compile time
- [ ] **Register allocation improvements** - Better register usage for performance
- [ ] **Dead code elimination** - Remove unused code from output
- [ ] **Constant propagation across functions** - Optimize constants through call boundaries
- [ ] **Inline small functions automatically** - Performance optimization

### Platform Support

- [ ] **Resolve ARM64 macOS binary hang issue** - Debug dyld or signing problems blocking ARM64 development
- [ ] **Windows support** - PE/COFF format output for Windows executables
- [ ] **FreeBSD support** - BSD platform compatibility

### Standard Library

- [ ] **Collections** - Hash map, tree, queue, stack for game state management
- [ ] **String manipulation** - split, join, replace, regex for text processing
- [ ] **File I/O library** - High-level wrappers for asset loading
- [ ] **Network programming** - Sockets, HTTP for multiplayer and web integration
- [ ] **JSON parsing and serialization** - Configuration and data exchange
- [ ] **Date/time library** - Timing and scheduling utilities

### Testing & Documentation

- [ ] **C FFI tutorial** - Practical examples with SDL3/RayLib game window, SQLite, POSIX file I/O
- [ ] **Unsafe block tutorial** - Systems programming examples: custom allocator, bit manipulation, hardware registers
- [ ] **Arena allocator tests** - Stress tests, growth behavior, nested arenas
- [ ] **Defer statement tests** - LIFO ordering, early returns, panic handling

### Notes

- **Philosophy**: Implement features completely before moving to next
- **Testing**: Every feature must have test coverage
- **Documentation**: Update LANGUAGE.md before marking feature complete
- **ARM64**: Several features work on x86-64 but need ARM64 ports (blocked until hang issue resolved)
