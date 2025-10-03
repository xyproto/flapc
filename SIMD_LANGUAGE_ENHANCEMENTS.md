# SIMD-Optimized Flap Language Enhancements

## Current Strengths (Already SIMD-Friendly!)

Flap's existing design already maps well to vector instructions:

```flap
// List comprehensions → automatic vectorization
[x * 2.0 for x in values]           // → VMULPD
[x for x in values if x > threshold] // → VCMPPD + compress

// Filter syntax → masked operations
entities{health > 0}                 // → VCMPPD + predication

// map[uint64]float64 → perfect for SIMD with integer indices
```

**But we can do better!**

---

## 1. Explicit Vectorization Operators

### Proposed: `||` for Explicit Parallelism

```flap
// Current (implicit vectorization)
result = [x * 2.0 for x in data]

// Enhanced (explicit SIMD guarantee)
result = data || map(x -> x * 2.0)
```

**Why**:
- Compiler **must** vectorize (or error if impossible)
- Clear to programmer which code is SIMD
- Maps directly to vector instructions

**Code Generation**:
```asm
; data || map(x -> x * 2.0)
VMOVUPD zmm0, [data]         ; Load 8 values
VBROADCASTSD zmm1, 2.0       ; Broadcast scalar
VMULPD zmm2, zmm0, zmm1      ; Multiply all 8
VMOVUPD [result], zmm2       ; Store 8 values
```

---

## 2. Gather/Scatter Syntax for Sparse Access

### Proposed: `@[]` for Indexed Access

```flap
// Current (serial access)
results = [map_data[indices[i]] for i in range(8)]

// Enhanced (parallel gather)
results = map_data@[indices]
```

**Why**:
- Single VGATHER instruction
- 4-8× faster than serial lookups
- Explicit sparse access pattern

**Code Generation**:
```asm
; map_data@[indices]
VMOVUPD zmm0, [indices]      ; Load 8 indices
VGATHERQPD zmm1{k1}, [map_data + zmm0*8]  ; Gather 8 values
```

### Scatter (Write Sparse)

```flap
// Scatter write
map_data@[indices] := values   // Store values at scattered indices
```

**Maps to**: `VSCATTERQPD [map_data + zmm0*8]{k1}, zmm1`

---

## 3. Width Hints for SIMD Control

### Proposed: `@8`, `@4` Annotations

```flap
// Process exactly 8 elements per iteration (AVX-512)
@8 result = data || map(x -> x * scale)

// Process 4 elements (AVX2 fallback)
@4 result = data || map(x -> x * scale)

// Auto-detect best width
@auto result = data || map(x -> x * scale)
```

**Why**:
- Control over vector register width
- Architecture-specific tuning
- Portable performance

**Alternative Syntax** (block-level):
```flap
@simd(width=8) {
    result1 = data1 || map(x -> x * 2.0)
    result2 = data2 || filter(x -> x > 0.0)
}
```

---

## 4. First-Class Mask/Predicate Support

### Proposed: `mask` Type and Operators

```flap
// Create mask from comparison
m: mask = values || (x -> x > threshold)

// Use mask for conditional operations
result = m ? (values || (x -> x * 2.0)) : values

// Combine masks
m1: mask = values || (x -> x > 0.0)
m2: mask = values || (x -> x < 100.0)
m3: mask = m1 and m2  // Element-wise AND
```

**Why**:
- Matches AVX-512 k registers, ARM64 predicates, RISC-V v0
- Branchless conditionals
- Efficient predication

**Code Generation**:
```asm
; m1 and m2 (mask AND)
KANDQ k3, k1, k2             ; Combine masks

; Masked operation
VMULPD zmm0{k3}, zmm1, zmm2  ; Only where k3=1
```

---

## 5. Fused Operations Syntax

### Proposed: `*+` for FMA

```flap
// Current (2 operations)
result = (a * b) + c

// Enhanced (single FMA)
result = a *+ b + c   // Fused multiply-add
```

**Alternative (Explicit)**:
```flap
result = fma(a, b, c)  // Built-in FMA function
```

**Why**:
- Single instruction = 2× throughput
- Better precision (one rounding)
- Perfect for dot products

**Common Pattern** (Dot Product):
```flap
// Dot product with FMA
dot_product = (a, b) -> {
    sum = 0.0
    @ i in range(len(a)) {
        sum := sum *+ a[i] + b[i]  // Accumulates with FMA
    }
    sum
}
```

---

## 6. Reduction Operations

### Proposed: Built-in Vector Reductions

```flap
// Horizontal sum (reduction)
total = values ||> sum    // Parallel sum of all elements

// Other reductions
max_val = values ||> max
min_val = values ||> min
product = values ||> product
any_true = masks ||> any
all_true = masks ||> all
```

**Why**:
- Maps to horizontal instructions
- Common pattern in parallel code
- Clearer than manual reduce

**Code Generation**:
```asm
; values ||> sum (AVX-512)
VMOVUPD zmm0, [values]
VEXTRACTF64X4 ymm1, zmm0, 1  ; Extract high 256 bits
VADDPD ymm0, ymm0, ymm1      ; Add high and low
; Continue tree reduction...
```

---

## 7. Batch/Chunk Processing

### Proposed: `@chunk(n)` Iterator

```flap
// Process in chunks of 8 (SIMD width)
@chunk(8) for chunk in data {
    // chunk is guaranteed to be 8 elements
    results += chunk || map(x -> x * 2.0)
}

// Auto-chunk based on SIMD width
@chunk(auto) for chunk in data {
    results += chunk || map(process)
}
```

**Why**:
- Explicit chunking for SIMD
- Handles remainder automatically
- Compiler knows chunk size at compile time

---

## 8. SIMD-Friendly Pattern Matching

### Enhanced: Parallel Comparisons

```flap
// Current (serial)
classify = (x) -> ~ x {
    < 0.0 -> "negative"
    0.0 -> "zero"
    > 0.0 -> "positive"
}

// Enhanced (vectorized classification)
classifications = values ||> classify_parallel(
    < 0.0 -> 0,
    0.0 -> 1,
    > 0.0 -> 2
)
```

**Code Generation**:
```asm
; Parallel classification
VXORPD zmm1, zmm1, zmm1        ; Zero
VCMPPD k1, zmm0, zmm1, LT      ; k1 = (x < 0)
VCMPPD k2, zmm0, zmm1, EQ      ; k2 = (x == 0)
VCMPPD k3, zmm0, zmm1, GT      ; k3 = (x > 0)
; Use masks to select results
```

---

## 9. Explicit Memory Layout Hints

### Proposed: `@aligned`, `@packed` Annotations

```flap
// Force 64-byte alignment (for AVX-512)
@aligned(64) big_array: [float64] = allocate(1000)

// Pack data for efficient access
@packed entity_positions = [@{x: float64, y: float64}]
```

**Why**:
- Aligned loads are faster
- Helps compiler generate better SIMD
- AoS → SoA transformations

---

## 10. Pipeline Optimization Markers

### Proposed: `~>` for SIMD Pipeline Stages

```flap
// Each stage must vectorize
result = data
    ~> filter(x -> x > 0.0)      // VCMPPD + compress
    ~> map(x -> x * 2.0)          // VMULPD
    ~> reduce(sum)                // Horizontal add
```

**Why**:
- Clear pipeline stages
- Each stage uses SIMD
- Compiler can fuse stages

---

## Complete Example: Before & After

### Before (Current Flap)

```flap
// Distance calculation for entities
update_distances = (entities, target) -> {
    results = []
    @ entity in entities {
        dx = entity.x - target.x
        dy = entity.y - target.y
        dist = sqrt(dx * dx + dy * dy)
        dist < 100.0 and results := results + [entity]
    }
    results
}
```

**Performance**: Scalar, one entity at a time

---

### After (Enhanced Flap)

```flap
// SIMD-optimized distance calculation
@simd(width=8)
update_distances = (entities, target) -> {
    // Gather positions (8 entities at once)
    xs = entities.positions@[0:8:x]  // Gather x coordinates
    ys = entities.positions@[0:8:y]  // Gather y coordinates

    // Parallel computation
    dxs = xs ||> (x -> x - target.x)       // VSUBPD
    dys = ys ||> (y -> y - target.y)       // VSUBPD

    // FMA for distance squared
    dist_sq = dxs *+ dxs + (dys *+ dys)    // VFMADD
    dists = dist_sq ||> sqrt                // VSQRTPD

    // Parallel comparison
    mask: mask = dists ||> (d -> d < 100.0) // VCMPPD

    // Compress results using mask
    mask ? entities[0:8] : []
}
```

**Performance**:
- 8× parallelism (AVX-512)
- 3 VGATHER loads
- 4 SIMD arithmetic ops
- 1 SIMD comparison
- 10-20× faster than scalar

---

## Recommended Changes Summary

### High Priority (Immediate Impact)

1. **`||` operator** - Explicit SIMD guarantee
2. **`@[]` syntax** - Gather/scatter for sparse access
3. **`mask` type** - First-class predication
4. **Built-in reductions** - `||> sum`, `||> max`, etc.

### Medium Priority (Nice to Have)

5. **`*+` operator** - Explicit FMA
6. **`@chunk(n)`** - Batch processing
7. **Width hints** - `@8`, `@4`, `@auto`

### Low Priority (Advanced)

8. **Memory layout** - `@aligned`, `@packed`
9. **Pipeline markers** - `~>` for stages
10. **Parallel patterns** - Vectorized pattern matching

---

## Backward Compatibility

**Option 1: Opt-in (Conservative)**
```flap
@simd_mode {
    // Enhanced syntax only inside this block
    result = data || map(x -> x * 2.0)
}
```

**Option 2: Dual Mode (Pragmatic)**
```flap
// Old syntax still works (implicit vectorization)
result = [x * 2.0 for x in data]

// New syntax gives guarantees (explicit vectorization)
result = data || map(x -> x * 2.0)
```

**Option 3: Gradual (Recommended)**
- Phase 1: Add `||` and `@[]` as optional syntactic sugar
- Phase 2: Add `mask` type and width hints
- Phase 3: Deprecate old patterns, migrate to SIMD-first

---

## Implementation Strategy

1. **Parser changes**: Recognize new operators (`||`, `@[]`, `*+`)
2. **Type system**: Add `mask` type, width annotations
3. **Codegen**: Emit SIMD instructions for new constructs
4. **Optimization**: Detect old patterns, auto-convert when safe
5. **Diagnostics**: Warn when vectorization fails

---

## Conclusion

With these enhancements, Flap becomes a **SIMD-first language** that:

✅ Makes vectorization **explicit and guaranteed**
✅ Exposes **hardware capabilities** directly to programmers
✅ Remains **clean and functional** (no ugly pragmas)
✅ Achieves **10-20× speedups** on numerical workloads
✅ Works across **x86-64, ARM64, RISC-V** transparently

The key insight: **SIMD isn't just an optimization—it's a first-class language feature.**
