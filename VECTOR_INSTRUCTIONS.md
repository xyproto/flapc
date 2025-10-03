# Modern Vector Instructions for Flap

## Why SIMD Matters for Flap

Flap's `map[uint64]float64` foundation and functional programming patterns are **perfectly suited** for SIMD vectorization. Operations that traditionally process one value at a time can process 8 values simultaneously on modern CPUs.

## Critical Instructions (Highest Impact)

### 1. **VGATHER** - Sparse Map Access
**Files**: `vgather.go`

**Why Essential**: THE killer feature for `map[uint64]float64` workloads with integer indices.
```flap
// Instead of 8 serial lookups:
results = [map[indices[0]], map[indices[1]], ..., map[indices[7]]]

// Single VGATHER loads all 8 values in parallel
// 8× speedup for non-contiguous map access
```

**Use Cases**:
- Indirect map lookups: `map_values[indices]`
- Sparse data access in pattern matching
- Graph traversal with random access
- Hash table probing

**Architecture**: AVX-512, SVE2, RVV all support indexed loads.

---

### 2. **VCMPPD** - Vectorized Filtering
**Files**: `vcmppd.go`

**Why Essential**: List comprehensions and filters are core Flap operations.
```flap
// Filter with predicate
[x for x in values if x > threshold]

// Vectorized: compare 8 values at once, produce mask
// Then compress/select based on mask
```

**Use Cases**:
- List comprehensions: `[x in xs]{x > min}`
- Pattern matching guards: `n <= 1 -> 1`
- Error handling: `value > 0 or! "invalid"`
- Conditional pipeline stages

**Produces**: Mask register (k1-k7 on x86, p0-p15 on ARM64, v0 on RISC-V)

---

### 3. **VFMADD** - Precise Numerical Operations
**Files**: `vfmadd.go`

**Why Essential**: Single rounding = better precision for float64.
```flap
// Dot product: sum(a[i] * b[i])
// Matrix multiply accumulate
// Polynomial evaluation: a*x² + b*x + c
```

**Use Cases**:
- Dot products and matrix operations
- Running totals with multiplication
- Physics calculations (position += velocity * dt)
- Financial calculations requiring precision

**Performance**: 2× operations (multiply + add) in single instruction.

---

## High-Impact Instructions

### 4. **VMOVUPD** - Bulk Memory I/O
**Files**: `vmovupd.go`

**Why Important**: Loading/storing 8 float64 values (64 bytes) at once.
```flap
// Load array chunk for processing
values = load_vector(array, offset)  // 8 values
// Process...
store_vector(results, array, offset)  // 8 values
```

**Use Cases**:
- Pipeline input/output
- Batch loading from maps
- Array initialization
- Streaming operations

**Note**: "Unaligned" = works with any memory address (not just 64-byte aligned).

---

### 5. **VADDPD/VMULPD/VSUBPD/VDIVPD** - Basic Arithmetic
**Files**: `vaddpd.go`, `vmulpd.go`, `vsubpd.go`, `vdivpd.go`

**Why Important**: Foundation of all numerical pipelines.
```flap
// Map operations vectorize automatically
values |> map(x -> x * scale)        // VMULPD
values |> map(x -> x + offset)       // VADDPD
values |> map(x -> x / total)        // VDIVPD
deltas |> map((a, b) -> a - b)       // VSUBPD
```

**Use Cases**:
- Scaling/normalization
- Element-wise operations
- Parallel transforms
- Reduction operations (partial)

---

## Performance Expectations

### Speedup Factors (vs scalar code)

| Operation | AVX-512 | AVX2 | SSE2 | SVE2 | RVV |
|-----------|---------|------|------|------|-----|
| Arithmetic | 8× | 4× | 2× | 4-8× | 4-16× |
| Comparisons | 8× | 4× | 2× | 4-8× | 4-16× |
| Gather | 4-6× | 2-3× | N/A | 3-5× | 3-6× |
| FMA | 16× ops/sec | 8× | 4× | 8-16× | 8-32× |

**Note**: Actual speedup depends on memory bandwidth, cache behavior, and branch patterns.

---

## Optimization Strategy for Flap Compiler

### Phase 1: Detect Vectorizable Patterns
```flap
// These patterns → automatic vectorization
[f(x) for x in array]           // Map
[x for x in array if pred(x)]   // Filter
sum([x * y for x, y in zip(a, b)])  // Reduce with FMA
```

### Phase 2: Code Generation
1. **Check vector length**: Process 8 elements (AVX-512) or 4 (AVX2) at a time
2. **Generate vector loop**: Use SIMD instructions for bulk
3. **Handle remainder**: Scalar code for last N % 8 elements
4. **Emit GATHER**: For sparse/indirect access patterns

### Phase 3: Predication/Masking
```flap
// Use mask registers for conditionals
// Instead of branches, use:
VCMPPD k1, zmm0, zmm1, GT      // k1 = (zmm0 > zmm1)
VMULPD zmm2{k1}, zmm3, zmm4    // Masked multiply (only where k1=1)
```

---

## Architecture-Specific Notes

### x86-64 (Intel/AMD)
- **Prefer AVX-512** when available (Skylake-X, Ice Lake, Zen 4+)
- **Fallback to AVX2** (universal since 2013)
- **Baseline SSE2** (all x86-64 CPUs)
- **Mask registers (k0-k7)** enable efficient predication

### ARM64
- **SVE2** on Neoverse V1+, Apple M-series, Ampere
- **NEON** universal baseline (all ARM64)
- **Scalable vectors**: Same code works at different vector lengths
- **Predicate registers (p0-p15)** for fine-grained control

### RISC-V
- **RVV 1.0** in newer cores (SiFive, Alibaba T-Head)
- **VLEN-agnostic**: Code portable across vector lengths
- **Simple ISA**: Clean implementation
- **Future-proof**: Designed for ML and data processing

---

## Example: Vectorized List Comprehension

```flap
// Flap source
result = [x * 2.0 for x in values if x > 0.0]

// Generated code (conceptual)
for i in 0..len(values) step 8:
    zmm0 = VMOVUPD [values + i*8]     // Load 8 values
    zmm1 = VBROADCASTSD 0.0           // Broadcast 0.0
    k1 = VCMPPD zmm0, zmm1, GT        // Compare: x > 0.0
    zmm2 = VBROADCASTSD 2.0           // Broadcast 2.0
    zmm3 = VMULPD{k1} zmm0, zmm2      // Masked multiply
    VMOVUPD [result + j*8], zmm3      // Store (with compression)
```

**Result**: 8× faster than scalar loop (in ideal conditions).

---

## Next Steps

1. **Implement scatter**: `vscatter.go` for writing sparse results
2. **Add broadcasts**: `vbroadcast.go` for replicating scalars
3. **Implement blends**: `vblend.go` for conditional select
4. **Add reductions**: Horizontal sum/min/max for fold operations
5. **Permutations**: Shuffles for data reorganization

---

## Conclusion

Vector instructions transform Flap's performance on numerical workloads:
- **8× parallelism**: Process 8 float64 values simultaneously
- **Perfect fit**: Functional patterns map directly to SIMD
- **Cross-platform**: Single source → AVX-512/SVE2/RVV
- **Future-proof**: Modern CPUs only getting wider vectors

The combination of **VGATHER** (sparse access), **VCMPPD** (filtering), and **VFMADD** (precision) makes Flap competitive with hand-optimized C code for numerical computing.
