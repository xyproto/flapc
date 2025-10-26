# Flapc

[![Go CI](https://github.com/xyproto/flapc/actions/workflows/ci.yml/badge.svg)](https://github.com/xyproto/flapc/actions/workflows/ci.yml) [![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/flapc)](https://goreportcard.com/report/github.com/xyproto/flapc)

Compiler for Flap, a functional language targeting game development. Generates native x86-64 machine code directly—no LLVM, no GCC, no runtime.

**Current status:** x86-64 Arch Linux stable (435+ tests passing, all optimizations working).

## What's Interesting

### Direct Code Generation
Lexer → Parser → x86-64 instructions → ELF binary. No intermediate representations. Compilation is instant (~1ms for small programs).

### Unified Type System
Everything is `map[uint64]float64`:
```flap
42              // {0: 42.0}
"Hi"            // {0: 72.0, 1: 105.0}  (UTF-8 codes)
[1, 2, 3]       // {0: 1.0, 1: 2.0, 2: 3.0}
{x: 10, y: 20}  // {x: 10.0, y: 20.0}
```

No type tags, no polymorphism, no generics. Just IEEE 754 doubles and SIMD-optimized map indexing (AVX-512/SSE2).

### Functional + Imperative
```flap
// Immutable by default (use = for constants)
fib = n => n < 2 {
    -> n
    ~> fib(n-1) max inf + fib(n-2) max inf
}

// Mutable when needed (use := for variables)
sum := 0.0
@ i in 0..<100 { sum <- sum + i }

// Infinite loops (game loops)
@ {
    update()
    render()
}
```

### Arena Memory
```flap
arena {
    buffer := alloc(1024)
    // ... use buffer ...
}  // freed here
```

Perfect for frame-local allocations in games.

### Unsafe Blocks
```flap
result := unsafe {
    rax <- 42
    rbx <- 100
    rax <- rax + rbx
    rax  // returns 142
}
```

Direct register access when you need it. Returns register values as expressions.

### C FFI
```flap
import sdl3 as sdl
import c as libc

pid := libc.getpid()
window := sdl.SDL_CreateWindow("Game", 0, 0, 800, 600, 0)
```

PLT/GOT dynamic linking, System V ABI. Calls C functions directly.

**Current limitation:** Integer args/return only (6 max). Float/pointer/struct support in progress for SDL3/RayLib5.

## Installation

### From Source
```bash
git clone https://github.com/xyproto/flapc
cd flapc
go build
sudo install -Dm755 flapc /usr/bin/flapc
```

### Dependencies
- Runtime: None (static binary generation)
- Build: Go 1.21+ (for compiler only)

## Usage

```bash
# Compile and run
flapc hello.flap
./hello

# Verbose output (see assembly)
flapc -v program.flap

# Specify output
flapc -o game program.flap
```

## Examples

**Hello World**
```flap
println("Hello, World!")
```

**Factorial**
```flap
factorial = n => n == 0 {
    -> 1
    ~> n * factorial(n - 1) max inf
}
println(factorial(10))
```

**Game Loop**
```flap
import sdl3 as sdl

running := 1.0
@ {
    running == 0.0 { ret }
    event := sdl.SDL_PollEvent()
    // ... handle events, update, render ...
}
```

See `programs/` for more examples.

## Language Features

### Syntax
- Comments: `// comment`
- Variables: `=` (immutable), `:=` (mutable)
- Update: `a <- new_value`
- Constants: Uppercase identifiers
- Loops: `@ i in list { ... }`, `@ { ... }` (infinite)
- Match: `x > 0 { positive() ~> negative() }`
- Lambdas: `x => x * 2` or `(x, y) => x + y`
- Recursion: Use function name for tail calls

### Data Types
All values are `map[uint64]float64`, but syntax sugar makes them feel typed:
- Numbers: `42`, `3.14`, `0xFF`, `0b1010`
- Strings: `"text"` (UTF-8)
- Lists: `[1, 2, 3]`
- Maps: `{key: value}`

### Control Flow
- Match expressions (no `if`)
- Auto-labeled loops (`@1`, `@2` for nesting)
- Loop shortcuts: `@-` (break prev), `@=` (continue)
- Loop vars: `@first`, `@last`, `@counter`, `@i`

### Functions
- First-class lambdas
- Tail-call optimization (automatic)
- Heap-allocated closures
- Built-ins: `println`, `printf`, `exit`, `str`, `num`, `alloc`, `free`

### Memory
- Manual: `ptr := alloc(size)` / `free(ptr)`
- Arena: `arena { ... }` (auto-freed on scope exit, not yet implemented)
- Defer: `defer cleanup()` (LIFO, not yet implemented)

### FFI
- C libraries: `import sdl3 as sdl`
- Direct PLT/GOT calls
- System V ABI
- **Limitation:** Integer args/return only (6 max), no float/pointer/struct yet

### Unsafe
- Register operations: `rax <- 42`, `rax <- [rbx]`, `[rax] <- value`
- Arithmetic: `rax * rbx`, `rax / rbx` (multiply/divide)
- Bitwise: `rax << rbx`, `rax >> rbx` (shifts)
- Return values: Last expression in unsafe block
- Multi-arch: `unsafe { /* x64 */ } { /* arm64 */ } { /* riscv */ }`

## Technical Details

### Compilation
1. Lexer: Tokenization
2. Parser: Hand-written recursive descent → AST
3. Code generator: Direct x86-64 emission
4. Binary builder: ELF generation with PLT/GOT
5. Two-pass: Address resolution and patching

No IR, no optimizer, no linker. ~12K lines of Go.

### ABI
- Calling convention: System V AMD64
- Registers: rdi, rsi, rdx, rcx, r8, r9 (args), rax (return)
- Float ops: SSE2 scalar (xmm0-xmm15)
- Stack: 16-byte alignment before calls

### SIMD
Map indexing uses runtime CPUID detection:
- AVX-512: 8 keys/iteration
- SSE2: 2 keys/iteration
- Scalar: 1 key/iteration (fallback)

All three implementations compiled into every binary.

### Binary
- Format: ELF64
- Dynamic linking: libc (malloc/free)
- Syscalls: write, read, open, close, lseek (direct)
- Size: Small (no runtime, no GC)

## Platform Support

Arch Linux x86-64 only. Other platforms (Windows, macOS, ARM64, RISC-V) planned but not yet implemented.

## Known Issues

- Inner lambdas can't capture outer lambda variables (segfaults)
- FFI limited to integer args/return (float/pointer crash)
- Arena syntax parsed but runtime not implemented (no auto-free)

See [TODO.md](TODO.md) for roadmap.

## Documentation

- **[README.md](README.md)** - This file
- **[LANGUAGE.md](LANGUAGE.md)** - Language specification and grammar
- **[TODO.md](TODO.md)** - Development roadmap
- **[LEARNINGS.md](LEARNINGS.md)** - Compiler implementation notes

No other documentation exists.

## Recent Improvements

**Optimizer Enhancements:**
- Dead Code Elimination now handles all expression types
- Constant propagation respects lambda parameter shadowing
- Loop unrolling supports loop state expressions (@i, @i1, @i2)
- Recursion depth limiting prevents compiler hangs
- All 435+ tests passing with optimizations enabled

## Goals

Short-term:
1. Implement hot code reload (Phase 2: File watching)
2. Implement closure capture for nested lambdas
3. Enhance FFI (float/pointer args for SDL3/RayLib5)
4. Unsafe block improvements (multiply/divide, return values)

Long-term:
- Game development with SDL3/RayLib5
- Multiplatform support (Windows, macOS, ARM64, RISC-V)
- Steam publishing and Steamworks integration

## License

BSD-3-Clause. See [LICENSE](LICENSE).

Safe for:
- Commercial use
- Arch/Debian packaging
- Closed-source projects
- Distribution and modification

No copyleft, no patent clauses.

## Development

**About vibecoding:** This compiler was developed with AI assistance (Claude). All code is BSD-licensed, tested (100% on x86-64), and auditable. Standard compiler architecture: lexer → parser → codegen.

**Contributing:** Bug reports welcome. Minimal test cases preferred. See `programs/` for examples.

## References

- System V AMD64 ABI
- ELF-64 Specification
- Intel 64 Manual (x86-64 instructions)

---

**Version:** 1.1.1
