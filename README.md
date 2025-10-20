# Flapc

[![Go CI](https://github.com/xyproto/flapc/actions/workflows/ci.yml/badge.svg)](https://github.com/xyproto/flapc/actions/workflows/ci.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/xyproto/flapc.svg)](https://pkg.go.dev/github.com/xyproto/flapc) [![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/flapc)](https://goreportcard.com/report/github.com/xyproto/flapc)

**Version 1.4.0**

Compiler for Flap, a functional programming language that compiles to native x86-64, ARM64, and RISC-V machine code.

## Technical Overview

Flap uses `map[uint64]float64` as its unified type representation:

- Numbers: `{0: 42.0}`
- Strings: `{0: 72.0, 1: 101.0, ...}` (index → character code)
- Lists: `{0: 1.0, 1: 2.0, ...}` (index → element value)
- Maps: Direct representation

The compiler generates native machine code directly without intermediate representations or runtime systems. All values use IEEE 754 double-precision floating point. SIMD optimizations (SSE2/AVX-512) are automatically selected at runtime via CPUID detection.

## v1.4.0 Release Notes

### C Library FFI

Direct C library function calls via PLT/GOT dynamic linking:

```flap
import sdl2 as sdl

result := sdl.SDL_Init(0x00000020)
window := sdl.SDL_CreateWindow("Game", 100, 100, 800, 600, 0)
sdl.SDL_Quit()
```

**Implementation:**
- Import syntax: `import <library> as <namespace>` (identifier without "/" = C library)
- ELF DT_NEEDED entries generated automatically
- System V AMD64 ABI calling convention
- Arguments: float64 → int64 conversion (Cvttsd2si)
- Return values: int64 → float64 conversion (Cvtsi2sd)
- PLT call patching via `trackFunctionCall()` and `patchPLTCalls()`

**Limitations:**
- Maximum 6 arguments (rdi, rsi, rdx, rcx, r8, r9 registers)
- Integer arguments only (no float, pointer, or struct support)
- Integer return values only

### Unsafe Blocks

Architecture-specific register manipulation (x86-64, ARM64, RISC-V):

```flap
result := unsafe {
    rax <- 100
    rbx <- 50
    rax <- rax + rbx
    rcx <- [rax]
    rax
} { /* arm64 */ } { /* riscv64 */ }
```

**Implemented operations:**
- Immediate loads: `rax <- 42`
- Register moves: `rax <- rbx`
- Stack push/pop: `stack <- rax`, `rax <- stack`
- Addition: `rax <- rax + rbx`, `rax <- rax + 10`
- Subtraction: `rax <- rax - rbx`, `rax <- rax - 10`
- Memory loads: `rax <- [rbx]`, `rax <- [rbx + 16]` (64-bit only)

**Not yet implemented:** multiply, divide, bitwise operations, shifts, sized loads/stores, memory stores, syscall instruction

## Implementation

- **Code generation**: Direct x86-64/ARM64/RISC-V machine code emission (no LLVM, no GCC)
- **Binary format**: ELF64 (Linux), Mach-O (macOS)
- **Type system**: Single type (`map[uint64]float64`) with IEEE 754 double-precision values
- **SIMD**: Runtime CPU detection (CPUID) selects SSE2 or AVX-512 paths for map indexing
- **FFI**: PLT/GOT dynamic linking for C library calls
- **Unsafe blocks**: Three-architecture register manipulation (x86-64, ARM64, RISC-V)
- **Constants**: Uppercase identifiers with compile-time substitution
- **Literals**: Decimal, hexadecimal (`0xFF`), binary (`0b1010`)
- **Calling convention**: System V AMD64 ABI
- **Stack alignment**: 16-byte before CALL instructions
- **Compiler**: ~12,000 lines of Go code

## Test Status

**x86-64 Linux:** 200/201 tests passing (99.5%)
- Expected failure: `unsafe_arithmetic_test` (multiply/divide not implemented)

**ARM64 macOS:** Binary hang on execution
- Compiler builds successfully
- Generated binaries hang before entering main()
- Issue appears to be macOS-specific (dyld or code signing related)

**RISC-V 64-bit:** Instruction encoders implemented, code generation incomplete

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
- **Loops**: `@ i in range(10) { println(i) }`
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

## Performance Characteristics

### SIMD Map Indexing

Runtime CPUID detection selects implementation:

- **AVX-512** (if supported): 8 keys per iteration using VGATHERQPD
- **SSE2** (baseline): 2 keys per iteration using UNPCKLPD + CMPEQPD
- **Scalar** (fallback): 1 key per iteration

Each compiled binary includes all three implementations.

### Runtime Characteristics
- No garbage collector
- No bytecode interpreter
- No JIT compilation
- Direct Linux syscalls for I/O (write, read, open, close, lseek)
- Compile-time constant folding

## Known Issues (v1.4.0)

- **ARM64**: Binary execution hang (macOS-specific)
- **Unsafe operations**: Missing multiply, divide, bitwise (AND/OR/XOR), shifts, memory stores, syscall
- **C FFI**: Limited to 6 integer arguments; no support for strings, structs, pointers, or floats
- **Memory management**: Arena/defer syntax parsed but code generation not implemented

See [TODO.md](TODO.md) for v1.5.0 development priorities.

## License

BSD-3-Clause

## References

- System V AMD64 ABI specification
- Intel 64 and IA-32 Architectures Software Developer's Manual
- ELF-64 Object File Format specification
- Mach-O executable format specification
