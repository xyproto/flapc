# Flap Compiler TODO

## Version 1.3.0 - RELEASED ✓

### Completed Features
- ✅ **Constants**: Uppercase identifiers (PI, MAX_HEALTH) with compile-time substitution
- ✅ **Hex/Binary Literals**: 0xFF and 0b11110000 syntax for colors and bitmasks
- ✅ **Unsafe Blocks**: Three-architecture blocks (x86_64, ARM64, RISC-V) with register operations
  - Register loads: `rax <- 42`, `rax <- rbx`
  - Stack operations: `stack <- rax`, `rax <- stack`
  - Arithmetic: `rax <- rax + rbx`, `rax <- rax - 10`
  - Memory loads (u64 only): `rax <- [rbx]`, `rax <- [rbx + 16]`
- ✅ **Memory Management Syntax**: `arena`, `defer`, `alloc()` keywords recognized and parsed
- ✅ **Documentation**: Comprehensive LANGUAGE.md updates with examples

### Deferred to v1.4.0
- Arena allocator runtime (bump pointer, growth, cleanup)
- Defer statement code generation
- alloc() runtime implementation
- Extended unsafe operations (multiply, bitwise, shifts, sized loads/stores, syscall)

## Version 1.4.0 - C Library FFI ✓ RELEASED

### C Library Import Syntax ✅ COMPLETE
- ✅ Auto-detect C imports (identifier without "/")
- ✅ Namespace-based function calls: `import sdl2 as sdl` → `sdl.SDL_Init(0)`
- ✅ PLT/GOT dynamic linking
- ✅ System V AMD64 ABI calling convention
- ✅ Support for up to 6 integer arguments
- ✅ Automatic ELF DT_NEEDED library dependencies
- ✅ Working with SDL2, SDL3, standard C library

### Deferred to v1.4.1
- [ ] pkg-config integration for library discovery
- [ ] String arguments (C char pointers)
- [ ] Struct and pointer support
- [ ] Float return values
- [ ] >6 argument support

## Version 1.4.1 - Memory Management Runtime

### Arena Allocator Implementation (Deferred from v1.3.0)

### Arena Allocator Implementation
- [ ] Implement arena runtime (using malloc/realloc/free)
  - [ ] Add `flap_arena_create(size)` - allocate initial arena
  - [ ] Add `flap_arena_alloc(size)` - bump pointer allocation with growth
  - [ ] Add `flap_arena_destroy(arena)` - free entire arena
  - [ ] Add global arena stack in .data section
- [ ] Implement arena runtime in ARM64 assembly
- [ ] Implement arena runtime in RISC-V assembly
- [ ] Add arena block code generation
  - [ ] Push new arena on entry
  - [ ] Destroy arena on exit
  - [ ] Handle nested arenas
- [ ] Test arena blocks with various allocation patterns

### Defer Statement Implementation
- [ ] Add defer tracking to FlapCompiler
  - [ ] Track deferred expressions per scope
  - [ ] Emit deferred code in LIFO order
- [ ] Implement defer code generation
  - [ ] Execute at end of scope
  - [ ] Execute before ret statements
  - [ ] Handle nested defer statements
- [ ] Test defer with various scenarios

### Unsafe Language Extensions (Missing Operations)
- [ ] Implement missing arithmetic operations
  - [ ] `ImulRegWithReg` - multiply register with register
  - [ ] `ImulRegWithImm` - multiply register with immediate
- [ ] Implement missing bitwise operations
  - [ ] `AndImmToReg` - AND immediate to register
  - [ ] `OrImmToReg` - OR immediate to register
  - [ ] `XorImmToReg` - XOR immediate to register
- [ ] Implement missing shift operations
  - [ ] `ShlReg` - shift left with immediate
  - [ ] `ShrReg` - shift right with immediate
  - [ ] `ShlRegByCL` - shift left by CL register
  - [ ] `ShrRegByCL` - shift right by CL register
- [ ] Implement memory operations
  - [ ] `MovMemToReg8/16/32` - load 8/16/32-bit values
  - [ ] `MovsxdMemToReg` - sign-extend 32->64
  - [ ] `MovImmToMem` - store immediate to memory
- [ ] Implement syscall instruction
  - [ ] x86_64: `syscall`
  - [ ] ARM64: `svc #0`
  - [ ] RISC-V: `ecall`
- [ ] Test all unsafe operations

### Documentation
- [ ] Update LANGUAGE.md to version 1.3.0
  - [ ] Document arena blocks
  - [ ] Document defer keyword
  - [ ] Document alloc() builtin
  - [ ] Update unsafe operations list
- [ ] Add examples for arena usage
- [ ] Add examples for defer usage

## Current Status

**x86-64 Linux**: 197/197 tests (100%) ✓
**ARM64 macOS**: Compilation hang issue (binary execution blocked)

## Deferred Issues

### Critical Issue - ARM64 Binary Hang
- **BLOCKING**: ARM64 flapc binary hangs immediately on execution
  - Hang occurs before main() debug prints execute
  - Process enters sleep state with minimal memory usage
  - Likely macOS-specific issue (permissions, signing, or dyld)
  - **Deferred until x86_64 features complete**

### Missing ARM64 Features (Blocked by hang)
- [ ] SliceExpr: List/string slicing
- [ ] PipeExpr: Pipe operator
- [ ] JumpExpr: Loop break/continue
- [ ] Float-to-string conversion for str()

### Low Priority
- [ ] RISC-V backend completion
- [ ] Standard library extensions (map, filter, reduce, etc.)
- [ ] Performance optimizations
