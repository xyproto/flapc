# Flapc

[![Go CI](https://github.com/xyproto/flapc/actions/workflows/ci.yml/badge.svg)](https://github.com/xyproto/flapc/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/xyproto/flapc.svg)](https://pkg.go.dev/github.com/xyproto/flapc)
[![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)
[![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/flapc)](https://goreportcard.com/report/github.com/xyproto/flapc)

Flapc is a tiny self-hosted experiment in compiling the Flap language straight to native ELF binaries with no intermediate representation. Every value in Flap is backed by the same `map[uint64]float64` layout, then lowered to SIMD-heavy machine code for x86-64, ARM64 and RISC-V 64-bit targets.【F:flap_runtime.go†L7-L135】【F:vaddpd.go†L8-L176】【F:arch.go†L5-L42】

## Highlights

* **Single runtime type** – integers, strings, lists and maps all share the same hash-map backing, enabling uniform code generation and simplified calling conventions.【F:flap_runtime.go†L7-L135】【F:hashmap.go†L8-L137】
* **Direct ELF emission** – the compiler writes program headers, sections, PLT/GOT tables and relocations itself for dynamically linked executables.【F:elf_complete.go†L10-L159】【F:parser.go†L1939-L2057】
* **Multi-architecture SIMD** – AVX-512, AVX2/SSE2, ARM SVE/NEON and RISC-V V instructions are all emitted from the same vector operations, guarded by runtime CPUID detection.【F:vaddpd.go†L8-L176】【F:parser.go†L1876-L1903】【F:parser.go†L2050-L2063】
* **Hash-map first** – the compiler ships a specialised hash map for `uint64 → float64`, complete with resizing, deletion and iteration support.【F:hashmap.go†L8-L194】
* **Pipelines and parallelism** – the language provides `|` pipes, `||` parallel list application and (reserved) `|||` concurrent gather operators for staged data processing.【F:parser.go†L584-L606】【F:parser.go†L1353-L1394】【F:parser.go†L3787-L3906】【F:parser.go†L5990-L6054】【F:parser.go†L6057-L6065】
* **Dependency-aware build** – unknown symbols trigger automatic cloning and parsing of external `.flap` libraries before compilation continues.【F:parser.go†L6183-L6256】

## Target platforms

* **Architectures:** x86_64, aarch64/arm64, riscv64.【F:arch.go†L5-L42】
* **Operating system:** Linux (ELF output).

## Building

```bash
make
```
This runs `go build` for all Go sources and produces the `flapc` binary.【F:Makefile†L15-L27】

```bash
make test
```
Runs the Go test suite for the compiler and runtime helpers.【F:Makefile†L20-L23】

## Usage

Compile a program:

```bash
./flapc hello.flap
./hello
```

Compile with explicit output filename or architecture:

```bash
./flapc -o app -m aarch64 hello.flap
```

Evaluate inline code without touching the filesystem:

```bash
./flapc -c 'println("hi")'
```

Useful flags:

* `-m`, `-machine` – choose target architecture (`x86_64`, `arm64/aarch64`, `riscv64`).
* `-o`, `-output` – select the output executable name.
* `-v`, `-verbose` – emit compilation progress and debug information.
* `-u`, `-update-deps` – refresh imported Flap libraries from their Git remotes.
* `-c` – compile and run inline source stored in a temporary file.【F:main.go†L605-L705】

## Language outline

### Data foundation

* Everything is a `map[uint64]float64` with a shared memory layout: a header followed by key/value pairs. Scalar values are stored as `{0: value}` maps, lists become sequential indices, and strings are maps of byte indices to character codes.【F:flap_runtime.go†L7-L161】
* The bundled hash map includes insertion, lookup, resizing and deletion, mirroring how the runtime treats user-level collections.【F:hashmap.go†L8-L194】

### Expressions and control flow

* Comments use `//`, and the lexer recognises identifiers, numbers, strings and a compact set of arithmetic and logical operators.【F:parser.go†L15-L104】
* Loops are labelled with `@` prefixes (`@1 i in range(10) { ... }`) with shortcuts such as `@+` to open the next label, `@=` to continue the current loop and `@-` to jump outward. A `for` keyword expands to the same mechanism.【F:parser.go†L1089-L1320】
* `match` expressions use `{ -> expr ~> default }` blocks to branch without introducing new keywords.【F:parser.go†L1085-L1086】【F:parser.go†L873-L1082】

### Functional tools

* Lambda expressions are first-class, parsed as `(x, y) -> body`, with list literals, map literals and index expressions built on top of them.【F:parser.go†L534-L582】
* The pipe operator `|` forwards values into lambdas or function calls in sequence.【F:parser.go†L1353-L1382】【F:parser.go†L5990-L6054】
* The parallel operator `||` maps a lambda across a list by compiling a dedicated loop that calls the captured function for each element.【F:parser.go†L1384-L1394】【F:parser.go†L3787-L3906】
* The concurrent gather operator `|||` is parsed but deliberately aborts at compile time until runtime support for concurrency lands.【F:parser.go†L1358-L1366】【F:parser.go†L6057-L6065】

### Hash maps, SIMD and math

* CPU feature probing stores an `AVX-512` flag at startup, letting map lookups fall back to AVX2/SSE2 when necessary.【F:parser.go†L1876-L1903】【F:parser.go†L2050-L2063】
* Vector operations are implemented once and specialised per architecture (e.g. AVX-512 `VADDPD`, ARM SVE `FADD`, RISC-V `vfadd`).【F:vaddpd.go†L8-L200】

## Code generation pipeline

1. Parse the main source file, augment it with any missing functions from discovered repositories, and build an AST.【F:parser.go†L6183-L6256】
2. Collect symbols, emit SIMD-aware machine code, and patch in runtime helpers and lambda bodies.【F:parser.go†L1876-L1934】【F:parser.go†L3787-L3906】
3. Assemble dynamic sections, PLT/GOT tables and relocation entries, then write a fully formed ELF binary for the chosen architecture.【F:elf_complete.go†L10-L159】【F:parser.go†L1939-L2057】

Dynamic library calls are described through explicit type signatures and dispatched with architecture-specific calling conventions, so compiled programs can call into shared objects without depending on an external linker.【F:dynlib.go†L3-L192】

## Roadmap

* Flesh out the `|||` concurrent gather operator with real threading support.【F:parser.go†L6057-L6065】
* Broaden architecture-specific vector instruction coverage beyond the current arithmetic focus.【F:vaddpd.go†L8-L200】

## License

BSD 3-Clause.
