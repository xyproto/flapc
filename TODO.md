# Flap Compiler - Implementation Status and TODO

## ✅ Implemented Features

### Core Language
- [x] Comments: `//` for single-line comments
- [x] Variables: Mutable (`:=`) and immutable (`=`) assignment
- [x] Data Types: Float64 foundation (all values are float64)
- [x] Arithmetic: `+`, `-`, `*`, `/` (scalar double-precision)
- [x] Comparisons: `<`, `<=`, `>`, `>=`, `==`, `!=`
- [x] Length Operator: `#list` returns the length of a list

### Control Flow
- [x] Conditionals: `if`/`else`/`end` blocks with comparison operators
- [x] Loops: `@ identifier in range(n) { }` syntax
- [x] Builtin Functions: `range(n)`, `println()`, `exit()`, `len()`

### Data Structures
- [x] Lists: Literal syntax `[1, 2, 3]`, stored in .rodata with length prefix
- [x] List Indexing: Access elements with `list[index]`
- [x] List Iteration: Loop over elements with `@ item in list { }`
- [x] List Length: Get list length with `len(list)` or `#list`, returns 0.0 for empty lists
- [x] Empty Lists: `[]` evaluates to 0 (null pointer)

### Functions & Lambdas
- [x] Lambda Expressions: `(x) -> x * 2` or `(x, y) -> x + y`
- [x] First-Class Functions: Store lambdas in variables
- [x] Function Pointers: Functions represented as addresses (reinterpreted as float64)
- [x] Closures: Lambdas capture no external state (stateless)
- [x] Calling Convention: Up to 6 parameters in xmm0-xmm5, result in xmm0

### Code Generation
- [x] Architectures: x86-64 (primary), ARM64 (partial), RISC-V (partial)
- [x] Instructions:
  - Scalar FP: ADDSD, SUBSD, MULSD, DIVSD, CVTSI2SD, CVTTSD2SI
  - Comparisons: UCOMISD with conditional jumps
  - Stack operations: Proper 16-byte alignment
  - Memory: MOVSD for XMM register loads/stores

### Binary Generation
- [x] Format: ELF64 with dynamic linking
- [x] Sections: .text, .rodata, .data, .bss, .dynamic, .got, .plt
- [x] Relocations: PC-relative for data, PLT for external functions
- [x] ABI: Proper x86-64 calling convention with stack alignment

## 🚧 In Progress

### SIMD Features (Core Language Feature)
- [ ] Pipe operator `|` for sequential operations
- [ ] Parallel operator `||` for SIMD operations (parsing done, runtime needs work)
- [ ] Concurrent gather operator `|||` for concurrent result gathering
- [ ] Gather/scatter `@[]` for sparse access
- [ ] Mask type for predication
- [ ] Reductions `||>` (sum, max, min, etc.) - note: sum should not be a keyword
- [ ] Fused multiply-add `*+`

## 📋 TODO

### High Priority

1. **Remove sum as keyword** - Make it definable in the language instead
2. **Implement pipe operators**:
   - `|` for piping/sequential operations
   - `||` for parallelization (SIMD)
   - `|||` for gathering results concurrently
3. **Hash map datastructure**: Implement map[uint64]float64 as the fundamental datastructure
4. **Parallel execution groundwork**: Implement required mnemonics and machine code emission for parallel operations
5. **Fix parallel operator crash**: The `||` operator crashes at runtime (all components work in isolation)

### Advanced Features

- [ ] Pattern matching `=~` and `~`
- [ ] Objects `@{ }` with methods
- [ ] Self-reference `me`
- [ ] Error handling `or!` operator
- [ ] List methods (append, etc.)
- [ ] Map literals `{key: value}`
- [ ] Map operations with uint64 keys
- [ ] Map/filter as builtin functions

### Language Constructs

- [ ] Function definitions
- [ ] Recursion support
- [ ] String operations
- [ ] Type annotations (mask, float64)

### SIMD/Vector Instructions

#### Critical Instructions (Highest Impact)
- [ ] **VGATHER** - Sparse map access (THE killer feature for map[uint64]float64)
- [ ] **VCMPPD** - Vectorized filtering (essential for list comprehensions)
- [ ] **VFMADD** - Precise numerical operations (fused multiply-add)

#### High-Impact Instructions
- [ ] **VMOVUPD** - Bulk memory I/O (load/store 8 float64 values)
- [ ] **VADDPD/VMULPD/VSUBPD/VDIVPD** - Basic vectorized arithmetic

#### Additional Vector Instructions
- [ ] **VSCATTER** - Sparse writes to maps
- [ ] **VBROADCAST** - Replicate scalars across vector
- [ ] **VBLEND** - Conditional select
- [ ] Reductions: Horizontal sum/min/max
- [ ] Permutations: Shuffles for data reorganization

### Architecture Support

#### x86-64
- [ ] AVX-512 support (8-wide float64 operations)
- [ ] AVX2 fallback (4-wide operations)
- [ ] SSE2 baseline (2-wide operations)
- [ ] Mask registers (k0-k7) for predication

#### ARM64
- [ ] SVE2 support (scalable vectors)
- [ ] NEON baseline (universal on ARM64)
- [ ] Predicate registers (p0-p15)

#### RISC-V
- [ ] RVV 1.0 support (vector extension)
- [ ] VLEN-agnostic code generation

## 📊 Test Coverage

### Passing Tests
- ✅ Arithmetic: 10 + 3 = 13, 10 - 3 = 7, 10 * 3 = 30, 10 / 3 = 3
- ✅ Comparisons: All 6 operators (<, <=, >, >=, ==, !=)
- ✅ Loops: Simple (range(5)), nested (3x3), with arithmetic
- ✅ Conditionals: if/else branching
- ✅ Variables: Assignment and reassignment
- ✅ Lists: Literals [1, 2, 3], indexing list[0], multiple lists, empty lists
- ✅ List Length: `len(list)` and `#list` for both empty and non-empty lists
- ✅ List Iteration: `@ item in list { println(item) }`
- ✅ Loop variables: Using iterator in expressions (i * 2)
- ✅ Lambda Expressions: `(x) -> x * 2`, storage, calling, multi-argument
- ✅ First-Class Functions: Multiple lambdas, passing results between calls

### Tests Needed
- [ ] Hash map operations (map[uint64]float64)
- [ ] Pipe operator `|` for sequential operations
- [ ] Parallel operator `||` for SIMD map (currently crashes)
- [ ] Concurrent gather `|||`
- [ ] Gather/scatter `@[]` operations
- [ ] SIMD arithmetic (VADDPD, VMULPD, etc.)
- [ ] User-defined functions (once implemented)
- [ ] Pattern matching (once implemented)

## 🐛 Known Issues

1. **Parallel Operator Crash**: The `||` operator for SIMD map operations is parsed correctly but crashes at runtime. All individual components (lambdas, lists, function calls) work correctly in isolation. Need to debug the parallel execution runtime.

2. **sum is a keyword**: Currently `sum` is a keyword for reductions, but it should be definable in the language instead.

## 📈 Next Steps (Prioritized)

1. **Consolidate documentation** ✅ (DONE)
2. **Remove sum as keyword** - Allow defining sum in Flap
3. **Implement pipe operators**:
   - `|` for piping
   - `||` for parallelization
   - `|||` for concurrent gathering
4. **Hash map foundation**: Implement map[uint64]float64 as the core datastructure
5. **Parallel execution**: Implement machine code emission for parallel operations
6. **Add tests**: Create test programs for all new parallel features
7. **VGATHER/VSCATTER**: Implement critical vector instructions for map operations
8. **User-Defined Functions**: Named function definitions
9. **Closures**: Lambda capture of outer scope variables
10. **Pattern Matching**: Core language feature `=~`

## 🏗️ Implementation Strategy

### Phase 1: Core Parallel Infrastructure (Current Focus)
1. Remove sum as keyword
2. Implement pipe operators (|, ||, |||)
3. Implement hash map datastructure (map[uint64]float64)
4. Add parallel execution machine code generation
5. Create comprehensive tests

### Phase 2: SIMD Instruction Set
1. VGATHER/VSCATTER for sparse access
2. VCMPPD for vectorized comparisons
3. VFMADD for fused multiply-add
4. Basic vector arithmetic (VADDPD, VMULPD, etc.)
5. VMOVUPD for bulk memory operations

### Phase 3: Advanced Features
1. User-defined functions
2. Pattern matching
3. Objects and methods
4. Error handling (or! operator)
5. String operations

### Phase 4: Optimization
1. Multi-architecture SIMD (AVX-512, AVX2, SSE2, SVE2, RVV)
2. Auto-vectorization of suitable patterns
3. Mask-based predication
4. Horizontal reductions
5. Performance tuning

## Performance Goals

### Target Speedups (vs scalar code)

| Operation | AVX-512 | AVX2 | SSE2 | SVE2 | RVV |
|-----------|---------|------|------|------|-----|
| Arithmetic | 8× | 4× | 2× | 4-8× | 4-16× |
| Comparisons | 8× | 4× | 2× | 4-8× | 4-16× |
| Gather | 4-6× | 2-3× | N/A | 3-5× | 3-6× |
| FMA | 16× ops/sec | 8× | 4× | 8-16× | 8-32× |

## Contributing

Contributions are welcome! Priority areas:
1. SIMD instruction implementation
2. Hash map datastructure
3. Test coverage
4. Documentation improvements
5. Bug fixes (especially parallel operator crash)

See the implementation strategy above for the current roadmap.
