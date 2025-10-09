# Flap Compiler TODO

## Recently Completed (2025-10-09)
- [x] Strings as map[uint64]float64 (index → char code) ✓
- [x] String indexing: `s[1]` returns correct character code ✓
- [x] Compile-time string concatenation: `"Hello, " + "World!"` ✓
- [x] Unified SIMD indexing for strings/lists/maps ✓
- [x] Runtime CPU detection (AVX-512) ✓
- [x] SSE2 SIMD optimization (2 keys/iteration) ✓
- [x] AVX-512 SIMD optimization (8 keys/iteration) ✓

## High Priority - String Support
- [x] `println(string_variable)` support ✓
  - Implemented CString conversion: `map[uint64]float64` → C string at runtime
  - CString format: `[length_byte][char0]...[null]` (length stored before string)
  - Runtime conversion via inline assembly
- [x] Handle string variables in `println()` ✓
- [x] String concatenation works at compile-time ✓
- [ ] Runtime string concatenation for expressions (not just literals)
- [ ] Optimize CString conversion (currently O(n²), make it O(n))

## Polymorphic Operators
- [ ] Implement `+` for lists: `[1, 2] + [3, 4]` → `[1, 2, 3, 4]`
- [ ] Implement `+` for maps: `{1: 10} + {2: 20}` → `{1: 10, 2: 20}` (union)
- [ ] Implement `++` operator:
  - Numbers: increment by 1.0
  - Lists/maps: append single value
- [ ] Implement `--` operator:
  - Numbers: decrement by 1.0
  - Lists/maps: pop last element
- [ ] Implement `-` for intersection:
  - Strings: remove characters
  - Lists: remove elements
  - Maps: remove keys

## SIMD & Performance
- [ ] ARM64 NEON SIMD for map lookups (2-4 keys/iteration)
- [ ] RISC-V Vector extension support (scalable)
- [ ] Perfect hashing for compile-time constant maps
- [ ] Binary search for maps with 32+ sorted keys
- [ ] AVX2 path (4 keys/iteration) - optional middle tier

## Language Features (Planned)
- [ ] Regular expressions: `text =~ /pattern/`
- [ ] `in` operator improvements
- [ ] `#` length operator for strings
- [ ] Error handling: `or!` operator
- [ ] `me` self-reference for recursion
- [ ] Pattern matching in match expressions
- [ ] Guard expressions
- [ ] Object definitions with `@{}`
- [ ] Method definitions
- [ ] SIMD blocks: `@simd { }`
- [ ] Parallel operators: `||`
- [ ] Gather/scatter: `@[]`
- [ ] Reductions: `||> sum`
- [ ] FMA: `*+` operator

## Known Limitations
- String variables can't be printed (only string literals)
- String concatenation only works at compile-time
- No runtime polymorphic operators yet
- No garbage collection (using .rodata for all constants)
- Linux/ELF only (no Windows/macOS)

## Testing
- [x] All 90+ tests passing ✓
- [ ] Add more string operation tests
- [ ] Add operator overloading tests
- [ ] Add performance benchmarks

## Documentation
- [x] README.md updated ✓
- [x] LANGUAGE.md updated ✓
- [x] TODO.md created ✓
- [ ] Add ARCHITECTURE.md (compilation pipeline details)
- [ ] Add CONTRIBUTING.md
