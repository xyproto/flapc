# Flapc

[![Go CI](https://github.com/xyproto/flapc/actions/workflows/ci.yml/badge.svg)](https://github.com/xyproto/flapc/actions/workflows/ci.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/xyproto/flapc.svg)](https://pkg.go.dev/github.com/xyproto/flapc) [![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/flapc)](https://goreportcard.com/report/github.com/xyproto/flapc)

`flapc` is an experiment in vibecoding a compiler.

## Overview

Flap is a functional programming language built on a `map[uint64]float64` foundation, designed for high-performance numerical computing with first-class SIMD support. The compiler generates native machine code directly, with no intermediate steps.

**Core Type System:**
- All data is either `float64` or `map[uint64]float64`
- Strings are `map[uint64]float64` (index → character code)
- Lists are `map[uint64]float64` (index → value)
- Maps are `map[uint64]float64` (key → value)
- Functions are `float64` (reinterpreted pointers)

## Main Features

- **Direct to machine code** - `.flap` source compiles directly to native executables
- **Multi-architecture** - Supports x86_64, aarch64, and riscv64
- **Modern instructions** - Uses SIMD/vector instructions whenever possible
- **Constant folding** - Compile-time optimization of constant expressions
- **Hash map foundation** - `map[uint64]float64` is the core data type
- **No nil** - Simplified memory model
- **Few keywords** - Minimal syntax for maximum expressiveness
- **Suckless philosophy** - Simple, clear, maintainable

## Supported Platforms

- **Operating Systems**: Linux only (ELF executables)
- **Architectures**: x86_64, ARM64 (aarch64), RISC-V 64-bit

## Quick Start

### Building the Compiler

```bash
make
```

### Compiling a Flap Program

```bash
# Basic compilation
./flapc hello.flap
./hello

# Quiet mode
./flapc -q hello.flap

# Specify output file
./flapc -o hello hello.flap
```

### Running Tests

```bash
make test
```

## Language Features (Current Implementation)

### Core Language
- **Comments**: `//` for single-line comments
- **Variables**: Mutable (`:=`) and immutable (`=`) assignment
- **Data Types**: Float64 foundation (all numeric values are float64)
- **Arithmetic**: `+`, `-`, `*`, `/` (scalar double-precision)
- **Comparisons**: `<`, `<=`, `>`, `>=`, `==`, `!=`
- **Length Operator**: `#list` returns the length of a list

### Control Flow
- **Match Expressions**: `condition { -> expr ~> expr }` syntax (default case optional)
- **Loops**: `@ identifier in range(n) { }` syntax, `break`, `continue`
- **Builtin Functions**: `range(n)`, `println()` (syscall-based), `printf()`, `exit()`

### Data Structures
- **Strings**: Stored as `map[uint64]float64` (index → char code)
  - `s := "Hello"` creates map `{0: 72.0, 1: 101.0, 2: 108.0, ...}`
  - `s[1]` returns `101.0` (char code for 'e')
  - **Syscall printing**: `println(string_var)` converts map to bytes using direct syscalls
    - No external dependencies (no libc printf)
    - Assembly-based number formatting for whole numbers
    - Efficient map-to-bytes conversion for string variables
- **Lists**: Literal syntax `[1, 2, 3]`, stored as `map[uint64]float64`
- **Maps**: Literal syntax `{1: 25, 2: 30}`, native `map[uint64]float64`
- **Indexing**: Unified SIMD-optimized indexing for all container types
- **Empty containers**: `[]`, `{}`, `""` evaluate to 0 (null pointer)

### Functions & Lambdas
- **Lambda Expressions**: `(x) -> x * 2` or `(x, y) -> x + y`
- **First-Class Functions**: Store lambdas in variables
- **Function Pointers**: Functions as float64-reinterpreted addresses

### Code Generation
- **Scalar FP**: ADDSD, SUBSD, MULSD, DIVSD, CVTSI2SD, CVTTSD2SI
- **Comparisons**: UCOMISD with conditional jumps
- **Stack operations**: Proper 16-byte alignment
- **Memory**: MOVSD for XMM register loads/stores

### Binary Format
- **Format**: ELF64 with dynamic linking
- **Sections**: .text, .rodata, .data, .bss, .dynamic, .got, .plt
- **Relocations**: PC-relative for data, PLT for external functions
- **ABI**: x86-64 calling convention with stack alignment

## Example Programs

See the `programs/` directory for examples:
- `hello.flap` - Basic output
- `arithmetic_test.flap` - Arithmetic operations
- `list_test.flap` - List operations
- `test_string_index_debug.flap` - String indexing (strings as maps)
- `test_map_*.flap` - SIMD-optimized map operations
- `lambda_comprehensive.flap` - Lambda expressions
- `hash_length_test.flap` - Length operator

## Documentation

- **LANGUAGE.md** - Complete language specification with EBNF grammar
- **TODO.md** - Implementation status, known issues, and roadmap

## Architecture

### Compilation Pipeline
1. **Lexer**: Tokenization with keyword recognition
2. **Parser**: Recursive descent parser producing AST
3. **Code Generator**: Direct machine code emission
4. **ELF Builder**: Complete ELF64 file generation
5. **Two-pass**: Initial codegen → address resolution → final codegen

### Stack Frame Layout
```
[rbp + 0]     = saved rbp
[rbp - 8]     = alignment padding
[rbp - 24]    = first variable (16-byte aligned)
[rbp - 40]    = second variable (16-byte aligned)
...
```

### Calling Convention
- Float64 arguments/returns: xmm0
- Integer arguments: rdi, rsi, rdx, rcx, r8, r9
- Return address: rax (integers), xmm0 (floats)
- Stack: 16-byte aligned before CALL
- XMM registers: Used for all float64 operations

## Contributing

This is an experimental educational project. Feel free to explore, learn, and experiment.

## License

BSD-3-Clause

## Status

This is a work in progress! See TODO.md for current status and roadmap.

---

# SIMD-Optimized Map Indexing

## Overview

Flap's map indexing uses **automatic runtime SIMD selection** for optimal performance on any CPU:

1. **AVX-512**: 8 keys/iteration (8× throughput) - *auto-enabled on supported CPUs*
2. **SSE2**: 2 keys/iteration (2× throughput) - *fallback for all x86-64 CPUs*
3. **Scalar**: 1 key/iteration (baseline)

Every executable includes **CPUID detection at startup** - no recompilation needed!

## Current Implementation: Runtime SIMD Selection

### Automatic CPU Detection

Every Flap program starts with a **CPUID check** that detects AVX-512 support:

```x86asm
; At program startup
mov eax, 7          ; CPUID leaf 7
xor ecx, ecx        ; subleaf 0
cpuid               ; Execute CPUID
bt  ebx, 16         ; Test bit 16 (AVX512F)
setc al             ; Set AL=1 if supported
mov [cpu_has_avx512], al  ; Store result
```

### Runtime Path Selection

Map lookups check the `cpu_has_avx512` flag and automatically select:
- **AVX-512**: 8 keys/iteration (8× throughput) - *if CPU supports it*
- **SSE2**: 2 keys/iteration (2× throughput) - *fallback for all x86-64 CPUs*
- **Scalar**: 1 key/iteration (baseline)

### Performance

- **SSE2**: 2× throughput compared to scalar (available on **all x86-64 CPUs**)
- **AVX-512**: 8× throughput compared to scalar (available on Xeon, high-end desktop)
- Zero overhead for small maps (1 key falls through to scalar)

### Instructions Used

```x86asm
unpcklpd xmm0, xmm1    ; Pack 2 keys into one register [key1 | key2]
cmpeqpd  xmm0, xmm3    ; Compare both with search key in parallel
movmskpd eax, xmm0     ; Extract 2-bit comparison mask
test     al, 1         ; Determine which key matched
```

### Algorithm

```
Map format: [count][key1][value1][key2][value2]...

if count >= 2:
    broadcast search_key to both lanes of xmm3
    while count >= 2:
        load keys at [rbx], [rbx+16] into xmm0
        compare both with search_key -> mask
        if mask != 0:
            return value at matched position
        advance rbx by 32 bytes (2 pairs)
        count -= 2

if count == 1:  # Handle remainder
    scalar comparison
```

### Performance Example

**Map with 6 keys:**
- SSE2: 3 iterations (process 2 keys each)
- Scalar: 6 iterations (process 1 key each)
- **Speedup: 2×**

## AVX-512 Path (Automatic)

### Why AVX-512?

- **8× throughput** compared to scalar
- **4× better than SSE2**
- Ideal for large maps (8+ keys)
- **Automatically enabled** when CPU supports it

### Requirements

1. **CPU Support**: Intel Xeon Scalable (Skylake-SP+), AMD Zen 4+, Intel Core 12th gen+
2. **Instruction Sets**: AVX512F, AVX512DQ
3. **Runtime Detection**: ✅ **Automatic** via CPUID at program startup

### Planned Instructions

```x86asm
vbroadcastsd zmm3, xmm2                ; Broadcast key to 8 lanes
vgatherqpd   zmm0{k1}, [rbx+zmm4*1]   ; Gather 8 keys in one instruction
vcmppd       k2{k1}, zmm0, zmm3, 0    ; Compare all 8 -> k-register mask
kmovb        eax, k2                   ; Extract mask to GPR
bsf          edx, eax                  ; Find first match
```

### How It Works

Every Flap executable includes runtime CPU detection:

1. **Program startup**: CPUID checks for AVX512F support
2. **Result stored**: `cpu_has_avx512` flag (1 byte in .data)
3. **Map lookups**: Check flag before entering AVX-512 path
4. **Fallback**: Use SSE2 if AVX-512 not available

**Benefit**: Write once, runs optimally everywhere - no recompilation needed!

## Performance Comparison

### Theoretical Throughput

| Map Size | Scalar | SSE2 | AVX-512 |
|----------|--------|------|---------|
| 1 key    | 1 iter | 1 iter | 1 iter |
| 2 keys   | 2 iter | 1 iter | 1 iter |
| 6 keys   | 6 iter | 3 iter | 1 iter |
| 16 keys  | 16 iter | 8 iter | 2 iter |
| 100 keys | 100 iter | 50 iter | 13 iter |

### Cache Efficiency

- **Sequential access pattern** through key-value pairs
- **Predictable branches** (loop condition)
- **Minimal register pressure** (xmm0-xmm4 for SSE2)

## Testing

All tests pass with SSE2 implementation:

```bash
# Small maps (2-3 keys)
./test_map_comprehensive

# Medium maps (6 keys)
./test_map_simd_large

# Large maps (16 keys)
./test_map_avx512_large
```

## Implementation Notes

### Why Gather for AVX-512?

Keys are interleaved with values in memory:
```
[key1][value1][key2][value2]...
  ^      +8      ^16     +24
```

VGATHERQPD loads keys at indices [0, 16, 32, 48, 64, 80, 96, 112] in a single instruction, avoiding manual unpacking.

### SSE2 vs AVX2

AVX2 could process 4 keys/iteration but:
- Requires explicit CPU detection (not baseline)
- Diminishing returns (2× vs SSE2 for ~10% real-world gain)
- SSE2 is universal on x86-64

For Flap's philosophy of "performance by default," SSE2 provides the best balance.

## Future Enhancements

1. **ARM64 NEON**: Use ARM's Advanced SIMD for 2-4 keys/iteration
2. **RISC-V Vector**: Use RVV for scalable vector processing
3. **CPUID Detection**: Safe runtime selection of best SIMD path
4. **Perfect Hashing**: For compile-time constant maps, generate perfect hash
5. **Binary Search**: For maps with 32+ sorted keys

---

**Status**:
- ✅ SSE2 optimization active and tested
- ✅ AVX-512 with automatic CPU detection
- ✅ Runtime SIMD selection (no recompilation needed)

**Tested on**: x86-64 without AVX-512 (falls back to SSE2 correctly)
