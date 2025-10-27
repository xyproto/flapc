# Flapc

[![Go CI](https://github.com/xyproto/flapc/actions/workflows/ci.yml/badge.svg)](https://github.com/xyproto/flapc/actions/workflows/ci.yml) [![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/flapc)](https://goreportcard.com/report/github.com/xyproto/flapc)

**Flap compiler** - Generates native x86-64 machine code directly. No LLVM, no GCC, no runtime.

**Built for:** Game development (SDL3/RayLib5 target)
**Platform:** x86-64 Linux (Arch/Debian tested)
**Status:** 435+ tests passing, full optimization pipeline working

## Key Features

**Direct machine code generation** - Lexer → Parser → x86-64 → ELF. No IR. Instant compilation (~1ms).

**Unified type system** - Everything is `map[uint64]float64`. Numbers, strings, lists, objects—all the same runtime representation. SIMD-optimized lookups (AVX-512/SSE2).

```flap
42              // {0: 42.0}
"Hi"            // {0: 72.0, 1: 105.0}
[1, 2, 3]       // {0: 1.0, 1: 2.0, 2: 3.0}
{x: 10, y: 20}  // {x: 10.0, y: 20.0}
```

**Tail-call optimization** - Automatic TCO. Immutable by default (`=`), mutable when needed (`:=`).

**Arena memory** - Scope-based allocation. Perfect for frame-local game buffers.

**C FFI** - Direct PLT/GOT calls to C libraries. Automatic type inference from DWARF debug info.

**Unsafe blocks** - Direct register access for performance-critical code.

```flap
// Tail recursion
fib = n => n < 2 { -> n ~> fib(n-1) + fib(n-2) }

// Arenas
arena { buffer := alloc(1024) /* ... */ }  // auto-freed

// C FFI
import sdl3 as sdl
window := sdl.SDL_CreateWindow("Game", 800, 600, 0)

// Unsafe
result := unsafe { rax <- 42; rax }
```

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

## Quick Start

```bash
# Install
go build && sudo install -Dm755 flapc /usr/bin/flapc

# Hello world
echo 'println("Hello, World!")' > hello.flap
flapc hello.flap && ./hello

# Game loop
cat > game.flap << 'EOF'
import sdl3 as sdl
@ { update(); render() }  // infinite loop
EOF
flapc game.flap -o game
```

See `testprograms/` for 50+ examples.

## Language Reference

**Syntax**
- Variables: `x = 42` (immutable), `x := 42` (mutable), `x <- 43` (update)
- Loops: `@ i in 0..10 { }`, `@ { }` (infinite)
- Match: `x > 0 { yes() ~> no() }` (if-else)
- Lambdas: `f = x => x * 2` or `(x, y) => x + y`

**Types** (all `map[uint64]float64` internally)
- Numbers: `42`, `3.14`, `0xFF`, `0b1010`
- Strings: `"text"`, Lists: `[1,2,3]`, Maps: `{x: 10}`

**Memory**
- Manual: `alloc(size)` / `free(ptr)`
- Arena: `arena { ... }` (scope-based)
- Defer: `defer cleanup()` (LIFO)

**C FFI**
- Import: `import sdl3 as sdl`
- Calls: `sdl.SDL_CreateWindow(...)`
- ABI: System V AMD64, PLT/GOT linking
- Signatures: Auto-discovered via DWARF/pkg-config

**Unsafe**
- Registers: `rax <- 42`, `rax <- [mem]`, `[mem] <- rax`
- Math: `rax * rbx`, `rax / rbx`, `rax << 2`
- Returns: Last expression value

See [LANGUAGE.md](LANGUAGE.md) for complete grammar.

## Technical Details

**Compilation:** Lexer → Parser → x86-64 → ELF. Two-pass for address resolution. No IR, no external linker. ~12K lines of Go.

**ABI:** System V AMD64. Args in `rdi, rsi, rdx, rcx, r8, r9`. Floats in `xmm0-7`. 16-byte stack alignment.

**SIMD:** Runtime CPUID detection. AVX-512 (8 keys/iter), SSE2 (2 keys/iter), scalar (1 key/iter). All three compiled into every binary.

**Binary:** ELF64. Dynamic link to libc. Direct syscalls for I/O. No GC, no runtime.

**Optimization:** Dead code elimination, constant propagation, loop unrolling, whole-program optimization (2s timeout default).

**Platform:** x86-64 Linux only. ARM64/RISC-V/Windows/macOS planned.

## Known Limitations

**FFI:** Type signatures require DWARF debug info or debug packages (`libc6-dbg`). Without them, use explicit casts.

**Arenas:** Nested arenas have a double-free bug. Use single-level arenas.

**Platform:** x86-64 Linux only. Other platforms in development.

See [TODO.md](TODO.md) for detailed roadmap.

## Documentation

- [LANGUAGE.md](LANGUAGE.md) - Complete language specification
- [TODO.md](TODO.md) - Development roadmap
- [LEARNINGS.md](LEARNINGS.md) - Implementation notes

## Roadmap

**In progress:**
- Hot code reload (Phases 1-4 complete, integration pending)
- C header parsing for automatic FFI signatures

**Planned:**
- Game development with SDL3/RayLib5
- Multiplatform (Windows/macOS/ARM64/RISC-V)
- Steamworks integration

## License

BSD-3-Clause - Commercial use, packaging, modification allowed. No copyleft.

## Contributing

**Bug reports:** Provide minimal test case. See `testprograms/` for examples.

**Development:** Compiler developed with AI assistance (Claude). All code BSD-licensed, auditable, tested.

---

**Version:** 1.1.1
**Refs:** System V ABI, ELF-64 spec, Intel x86-64 manual
