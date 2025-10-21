# Plans

## Next actions

- [ ] **Float arguments in C FFI** - Use xmm0-xmm7 registers for passing floats to C functions (needed for graphics APIs)
- [ ] **Float return values in C FFI** - Retrieve float results from C functions (needed for math libraries)
- [ ] **Memoized recursion (cme) enhancements** - Add cache size limit and cleanup callback parameters:
  - `cme(arg, max_cache_size, cleanup_lambda)` where cleanup_lambda is called when cache is full
  - Currently `cme` only supports simple recursive calls without memoization
- [ ] **Arena allocator runtime** - Implement fast memory allocation for game objects and demo effects
  - Create `arena_runtime.c` with `flap_arena_create()`, `flap_arena_alloc()`, `flap_arena_destroy()`
  - Generate arena block code (call create on entry, destroy on exit)
  - Support nested arenas for hierarchical allocation
- [ ] **Defer statements runtime** - Implement cleanup code for resource management
  - Track deferred expressions per scope in FlapCompiler
  - Emit deferred code in LIFO order at scope exit and before returns
- [ ] **Sized memory loads in unsafe** - `rax <- [rbx] as uint8`, `rax <- [rbx] as uint8` for packed data structures
- [ ] **Sized memory stores in unsafe** - `[rax] <- rbx as uint8` for writing packed formats
- [ ] **Signed loads in unsafe** - `rax <- [rbx] as int8` with sign extension for audio/image processing
- [ ] **More than 6 C function arguments** - Stack-based argument passing for complex APIs
- [ ] **SIMD intrinsics** - Vector operations for audio DSP, graphics effects, particle systems
- [ ] **Register allocation improvements** - Better register usage for performance
- [ ] **Dead code elimination** - Remove unused code from output
- [ ] **Constant propagation across functions** - Optimize constants through call boundaries
- [ ] **Inline small functions automatically** - Performance optimization

## Specific for ARM64 and/or RISC-V 64 (wait a bit with these)

- [ ] **Resolve ARM64 macOS binary hang issue** - Debug dyld or signing problems blocking ARM64 development
- [ ] **Slice expressions** - Implement `list[1:5]` and `string[0:10]` (works on x86-64, needs ARM64 port)
- [ ] **Pipe operator** - Implement `data | transform | filter` (works on x86-64, needs ARM64 port)
- [ ] **Jump expressions** - Implement `break` and `continue` in loops (works on x86-64, needs ARM64 port)

## Future data structures and stdlib (wait a bit with these)

- [ ] **Collections** - Hash map, tree, queue, stack for game state management
- [ ] **String manipulation** - split, join, replace, regex for text processing
- [ ] **File I/O library** - High-level wrappers for asset loading
- [ ] **Network programming** - Sockets, HTTP for multiplayer and web integration
- [ ] **JSON parsing and serialization** - Configuration and data exchange
- [ ] **Date/time library** - Timing and scheduling utilities
- [ ] **Arena allocator tests** - Stress tests, growth behavior, nested arenas
- [ ] **Defer statement tests** - LIFO ordering, early returns, panic handling
