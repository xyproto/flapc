# Flapc

[![Go CI](https://github.com/xyproto/flapc/actions/workflows/ci.yml/badge.svg)](https://github.com/xyproto/flapc/actions/workflows/ci.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/xyproto/flapc.svg)](https://pkg.go.dev/github.com/xyproto/flapc) [![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/flapc)](https://goreportcard.com/report/github.com/xyproto/flapc)

**Version 1.4.0** - C Library FFI Release

Flapc is the compiler for Flap, a functional programming language that compiles directly to native machine code with zero runtime overhead.

## Philosophy

Flap embraces **radical simplicity through uniformity**. Instead of juggling types, memory models, and runtime complexity, everything is `map[uint64]float64`:

- **One Foundation**: Numbers are `{0: 42.0}`, strings are `{0: 72.0, 1: 101.0, ...}`, lists are `{0: 1.0, 1: 2.0, ...}`. One type, infinite flexibility.
- **Zero Runtime**: No GC pauses, no interpreter overhead, no hidden allocations. Direct compilation to native machine code with predictable performance.
- **Performance by Default**: SIMD operations (SSE2/AVX-512) with automatic CPU detection. Hardware FPU instructions. Every abstraction compiles to tight assembly.
- **C Interoperability**: Call any C library function with simple `import sdl2 as sdl` syntax. No bindings, no wrappers, just works.
- **Railway-Oriented Errors**: The `or!` operator creates clean error handling pipelines without the noise of `if err != nil` or exception hierarchies.
- **Suckless Ethos**: Simple, clear, maintainable. The compiler is ~12,000 lines of readable Go code. No macros, no templates, no DSLs.

**For game developers**: Direct SDL2/SDL3/Raylib access, hex/binary literals for colors and flags, compile-time constants with zero overhead, unsafe blocks for hardware access, predictable performance.

**For systems programmers**: Direct register manipulation via unsafe blocks, System V ABI compatibility, no runtime dependencies, tiny executables, full control.

Flap proves that **simplicity scales**. One type system, one calling convention, one memory model — yet expressive enough for real-world software.

## What's New in v1.4.0

### C Library FFI (Complete)

Call C library functions directly with automatic dynamic linking:

```flap
import sdl2 as sdl

result := sdl.SDL_Init(0x00000020)  // SDL_INIT_VIDEO
window := sdl.SDL_CreateWindow("Game", 100, 100, 800, 600, 0)
// ... game code ...
sdl.SDL_Quit()
```

**Features:**
- Auto-detection of C vs Flap imports (no "/" = C library)
- Namespace-based function calls
- Automatic PLT/GOT dynamic linking
- System V AMD64 ABI calling convention
- Works with SDL2, SDL3, Raylib, OpenGL, SQLite, and any C library

**Current Limitations:**
- Max 6 arguments per call
- Arguments converted to integers
- Return values assumed to be integers
- String/struct/pointer support coming in v1.5.0

### Enhanced Unsafe Language

Direct register manipulation for systems programming:

```flap
result := unsafe {
    rax <- 100
    rbx <- 50
    rax <- rax + rbx   // Arithmetic operations
    rcx <- [rax]       // Memory loads
    rax                // Return value
} { /* arm64 */ } { /* riscv64 */ }
```

**Supported Operations (v1.4.0):**
- Register loads: `rax <- 42`, `rax <- rbx`
- Stack operations: `stack <- rax`, `rax <- stack`
- Arithmetic: `rax <- rax + rbx`, `rax <- rax - 10`
- Memory loads: `rax <- [rbx]`, `rax <- [rbx + 16]`

**Coming in v1.5.0:** Multiply, divide, bitwise ops, shifts, memory stores, syscall

## Main Features

- **Direct to machine code** - `.flap` source compiles directly to native executables (ELF/Mach-O)
- **Multi-architecture** - Supports x86_64, aarch64, and riscv64
- **C Library FFI** - Call any C function with simple import syntax
- **SIMD by default** - Automatic SSE2/AVX-512 CPU detection and optimization
- **Unsafe blocks** - Direct register access for systems programming
- **Constants** - Compile-time constant folding with uppercase identifiers
- **Hex/Binary literals** - `0xFF` and `0b11110000` for colors and flags
- **Hash map foundation** - `map[uint64]float64` is the core data type
- **No nil** - Simplified memory model
- **Few keywords** - Minimal syntax for maximum expressiveness

## Test Status

**x86-64 Linux:** 200/201 tests passing (99.5%) ✅
- All core language features working
- C FFI working (getpid, SDL tests)
- SIMD-optimized map operations
- 1 expected failure: `unsafe_arithmetic_test` (multiply/divide not yet implemented)

**ARM64 macOS:** Compiler hang issue (deferred) 🔧
- Core features implemented
- Binary execution blocked by platform-specific issue
- Full support planned for v1.5.0+

**RISC-V 64-bit:** Partial (instruction encoders ready) ⚠️

## Quick Start

### Building the Compiler

```bash
go build
```

### Compiling a Flap Program

```bash
# Basic compilation
./flapc hello.flap
./hello

# Verbose mode (see assembly)
./flapc -v program.flap

# Specify output file
./flapc -o myapp program.flap
```

### Example: Hello World

```flap
println("Hello, World!")
exit(0)
```

### Example: C Library FFI

```flap
import c as libc

pid := libc.getpid()
println("Process ID:")
println(pid)
exit(0)
```

### Example: Unsafe Block

```flap
result := unsafe {
    rax <- 42
    rbx <- 100
    rax <- rax + rbx
    rax  // Returns 142
} { /* arm64 */ } { /* riscv64 */ }

println(result)
exit(0)
```

## Language Features

### Core Language
- **Comments**: `//` for single-line
- **Variables**: Immutable (`=`) and mutable (`:=`)
- **Constants**: Uppercase identifiers (`PI`, `MAX_SIZE`)
- **Numbers**: All values are `float64`
- **Literals**: Decimal, hex (`0xFF`), binary (`0b1010`)
- **Operators**: `+`, `-`, `*`, `/`, `%`, `<`, `>`, `==`, `!=`, etc.
- **Length**: `#list` returns length

### Control Flow
- **Match expressions**: `x > 0 { println("positive") ~> println("non-positive") }`
- **Loops**: `@+ i in range(10) { println(i) }`
- **Loop control**: `ret @label` (break), `@label` (continue)
- **Loop vars**: `@first`, `@last`, `@counter`, `@i`

### Data Structures
- **Strings**: `"Hello"` stored as `map[uint64]float64`
- **Lists**: `[1, 2, 3]` with unified indexing
- **Maps**: `{1: 100, 2: 200}` native type
- **Indexing**: SIMD-optimized for all containers

### Functions
- **Lambdas**: `(x) -> x * 2` or `x -> x + 1`
- **First-class**: Store and pass functions
- **Tail calls**: `me()` for recursion
- **Builtins**: `println()`, `printf()`, `exit()`, `str()`, `num()`

### Module System
- **Git imports**: `import "github.com/user/pkg" as name`
- **C imports**: `import sdl2 as sdl` (auto-detected)
- **Versions**: `import "pkg@v1.0.0" as name`
- **Wildcards**: `import "pkg" as *`
- **Private**: Functions starting with `_` not exported

### Unsafe Language
- **Register operations**: x86-64, ARM64, RISC-V
- **Arithmetic**: `rax <- rax + rbx`
- **Memory**: `rax <- [rbx + 8]`
- **Stack**: `stack <- rax`, `rax <- stack`
- **Three architectures**: Conditional compilation

### Memory Management (Syntax Only - Runtime in v1.5.0)
- **Arena blocks**: `arena { buffer := alloc(1024) }`
- **Defer**: `defer cleanup()` (LIFO execution)
- **Manual alloc**: `ptr := alloc(size)` (currently calls malloc)

## Supported Platforms

- **Operating Systems**: Linux (ELF), macOS (Mach-O)
- **Architectures**:
  - ✅ **x86-64**: Full support, 99.5% tests passing
  - 🔧 **ARM64**: Core features complete, binary hang blocks testing
  - ⚠️ **RISC-V**: Instruction encoders ready, codegen in progress

## Documentation

- **[LANGUAGE.md](LANGUAGE.md)** - Complete language specification
- **[TODO.md](TODO.md)** - Roadmap and implementation status
- **PACKAGE_SYSTEM.md** - Module system and dependencies

## Example Programs

See `programs/` directory:
- `hello.flap` - Basic Hello World
- `c_getpid_test.flap` - C FFI with standard library
- `c_ffi_test.flap` - SDL2 example
- `unsafe_test.flap` - Register manipulation
- `arithmetic_test.flap` - Math operations
- `list_test.flap` - List operations
- `lambda_comprehensive.flap` - Function examples

## Architecture

### Compilation Pipeline
1. **Lexer** - Tokenization with keyword recognition
2. **Parser** - Recursive descent, produces AST
3. **Code Generator** - Direct x86-64/ARM64/RISC-V emission
4. **ELF/Mach-O Builder** - Complete binary generation
5. **Two-pass** - Address resolution and PLT patching

### Key Technical Details
- **Calling Convention**: System V AMD64 ABI
- **Stack Alignment**: 16-byte before CALL
- **Float Operations**: SSE2 scalar double-precision
- **SIMD Indexing**: Automatic SSE2/AVX-512 selection
- **C FFI**: PLT/GOT dynamic linking
- **Binary Format**: ELF64 (Linux), Mach-O (macOS)

## Performance Features

### SIMD-Optimized Map Indexing

Automatic runtime CPU detection selects optimal SIMD path:

- **AVX-512**: 8 keys/iteration (if CPU supports)
- **SSE2**: 2 keys/iteration (fallback for all x86-64)
- **Scalar**: 1 key/iteration (baseline)

No recompilation needed - every binary includes all paths!

### Zero-Runtime Overhead
- No garbage collector
- No interpreter
- No hidden allocations
- Direct syscalls (no libc dependency for I/O)
- Inline constant folding

## Current Limitations

### v1.4.0 Known Issues
- **ARM64**: Compiler binary hang blocks all ARM64 development
- **Unsafe ops**: Multiply, divide, bitwise, shifts not implemented
- **C FFI**: Limited to 6 integer arguments, no strings/structs/pointers
- **Memory**: Arena/defer syntax parsed but runtime not implemented

### Planned for v1.5.0
- Complete unsafe operations (multiply, divide, bitwise, shifts, syscall)
- Enhanced C FFI (strings, pointers, floats, >6 args)
- Arena allocator runtime
- Defer statement code generation

See [TODO.md](TODO.md) for complete roadmap.

## Contributing

This is an educational/experimental project exploring direct machine code generation from a high-level functional language. Feel free to explore, learn, and experiment.

## License

BSD-3-Clause

## Acknowledgments

Built with insights from:
- System V AMD64 ABI specification
- Intel Software Developer Manuals
- ELF and Mach-O format specifications
- The suckless philosophy of simplicity and clarity

---

**Flap**: Simple. Fast. Direct.
