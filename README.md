# Flap, the compiler

[![Go CI](https://github.com/xyproto/flapc/actions/workflows/ci.yml/badge.svg)](https://github.com/xyproto/flapc/actions/workflows/ci.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/xyproto/flapc.svg)](https://pkg.go.dev/github.com/xyproto/flapc) [![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/flapc)](https://goreportcard.com/report/github.com/xyproto/flapc)

`flapc` is an experiment in vibecoding a compiler.

## Overview

Flap is a functional programming language built on a `map[uint64]float64` foundation, designed for high-performance numerical computing with first-class SIMD support. The compiler generates native machine code directly, with no intermediate steps.

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
./flapc program.flap
./main

# Quiet mode
./flapc -q program.flap

# Specify output file
./flapc -o myprogram program.flap
```

### Running Tests

```bash
./test.sh
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
- **Loops**: `@ identifier in range(n) { }` syntax
- **Builtin Functions**: `range(n)`, `println()`, `exit()`, `len()`

### Data Structures
- **Lists**: Literal syntax `[1, 2, 3]`
- **List Indexing**: Access elements with `list[index]`
- **List Iteration**: Loop over elements with `@ item in list { }`
- **Empty Lists**: `[]` evaluates to 0 (null pointer)

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
