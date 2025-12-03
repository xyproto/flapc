# Flap TODO

## Current Status (December 2025)

### ✅ All Tests Passing
- **Go test**: PASS (0.748s)
- All 100+ tests pass reliably

## ✅ Core Features Working

### Compilation
- ✅ Direct to machine code (x86_64, ARM64, RISC-V)
- ✅ Linux x86_64: Fully working, no libc required for pure Flap code
- ✅ Windows x86_64: Fully working with PE format
- ✅ ELF and PE executable generation
- ✅ Mach-O support (basic)

### Memory Management
- ✅ Arena allocator with automatic cleanup
- ✅ Default arena at program start
- ✅ `arena { }` blocks with scope-based cleanup
- ✅ Used for all internal allocations

### Language Features
- ✅ Functions as first-class values
- ✅ Higher-order functions (map, filter, fold)
- ✅ Pattern matching with guards
- ✅ `defer` for LIFO cleanup
- ✅ `or!` error handling operator
- ✅ Loops: `@`, `@ in`, `@ condition`
- ✅ Match expressions with guards
- ✅ String interpolation
- ✅ List comprehensions

### I/O & Printing
- ✅ Float printing with full precision (inline assembly, no libc)
- ✅ SSE2-based decimal extraction
- ✅ Printf format specifiers (%.2f, %.6f, etc.)
- ✅ Direct syscalls for I/O on Linux
- ✅ Pure assembly number conversion

### C FFI
- ✅ Import C libraries (`import sdl3 as sdl`)
- ✅ Header parsing for constants and functions
- ✅ PLT/GOT dynamic linking
- ✅ Conditional libc linking (only when C FFI used)
- ✅ Windows DLL support
- ✅ SDL3 bindings working

## 🚧 Partial/Experimental Features

### Optimization
- 🚧 Tail call optimization (implemented but conservative)
- 🚧 General optimizer disabled (needs type system)

### Platform Support
- 🚧 ARM64 backend (code exists, needs testing)
- 🚧 RISC-V backend (code exists, needs testing)
- 🚧 macOS support (will require libc for syscalls)

### Advanced Features
- 🚧 Executable compression (LZ77 compressor working, decompressor stub has bugs)
- 🚧 Function composition `<>` operator (partial)
- 🚧 Automatic memoization (not implemented)
- 🚧 Parallel loops `@@` (basic support, needs testing)

## ❌ Not Yet Implemented

### Language Features
- ❌ Automatic memoization for pure functions
- ❌ SIMD operations
- ❌ Inline assembly blocks

### Tooling
- ❌ Hot reload (infrastructure exists)
- ❌ Interactive REPL
- ❌ Language server protocol
- ❌ Package manager
- ❌ Debugger integration

### Platform Support
- ❌ Windows ARM64
- ❌ macOS ARM64
- ❌ WASM target
- ❌ WebGPU bindings

## 🎯 Priority Work Items

### High Priority
1. Fix decompressor stub segfaults for compression feature
2. Test ARM64/RISC-V backends on real hardware
3. Implement float printing for ARM64/RISC-V
4. Complete type inference for optimizer

### Medium Priority
5. Add more SDL3 examples
6. Performance benchmarking suite
7. Improve error messages
8. Document all builtins

### Low Priority
9. REPL implementation
10. Hot reload improvements
11. Language server
12. Package ecosystem
