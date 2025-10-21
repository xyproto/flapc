# Plans

## Recently Completed (2025-10-21)

- [x] **C FFI Implementation** - ✅ COMPLETE and production-ready
  - Float arguments/returns via xmm0-xmm7 registers (math.sqrt, math.pow tested)
  - Integer/bool arguments/returns with proper conversion (cvttsd2si/cvtsi2sd)
  - String literals as null-terminated C strings (printf, ncurses.printw tested)
  - Stack alignment for SIMD (System V AMD64 ABI compliance, movaps works)
  - Verified with: libc, libm, libncursesw, SDL3
  - Header parsing and constants extraction (227 ncurses constants, SDL3 constants)
  - pkg-config integration for automatic include path discovery

- [x] **Sized memory loads in unsafe** - ✅ COMPLETE
  - Syntax: `rax <- [rbx] as uint8` for zero-extension, `rax <- [rbx] as int8` for sign-extension
  - Supported types: uint8, int8, uint16, int16, uint32, int32, uint64, int64
  - Works with offsets: `rax <- [rbx + 8] as uint8`
  - All 6 sized types tested and verified (MOVZX for unsigned, MOVSX for signed)

- [x] **Sized memory stores in unsafe** - ✅ COMPLETE
  - Syntax: `[rax] <- value as uint8` for byte store, `[rax] <- value as uint16` for word store
  - Supported types: uint8, uint16, uint32, uint64 (int types work same as unsigned for stores)
  - Works with offsets: `[rax + 8] <- rbx as uint8`
  - All sized types tested and verified (MOV byte/word/dword [addr], reg)

- [x] **More than 6 C function arguments** - ✅ COMPLETE
  - Stack-based argument passing for complex C APIs (args 7+ go on stack)
  - Proper System V AMD64 ABI compliance (RSP % 16 == 8 before CALL)
  - Works for both integer and float arguments
  - Tested with 7, 8, and 10 argument functions

- [x] **Custom .so file imports** - ✅ COMPLETE (v1.6.0)
  - Import arbitrary .so files: `import "/tmp/libmylib.so" as mylib`
  - Automatic symbol extraction using `nm -D`
  - Automatic DT_NEEDED entries for dynamic linking
  - Works with >6 argument functions
  - Use with `LD_LIBRARY_PATH=/path/to/libs ./program`

## Next Actions
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
