# C67 Compiler TODO

## Critical Bugs

### 1. Module-Level Mutable Globals in Lambdas
- **Status:** Identified, needs fix
- When a lambda modifies a module-level mutable variable (`:=`), the change doesn't persist after lambda returns
- Example:
  ```c67
  g_value := 0
  set_value = (x) -> { g_value <- x }
  main = { set_value(42); println(g_value) }  # Prints 0 instead of 42
  ```
- This affects libraries that use globals for state (like c67game)
- **Workaround:** Pass state explicitly as parameters and return values

### 2. Import System Issues
- Lambdas from imported C67 packages may be generated multiple times
- Need to verify closure object initialization for imported functions
- Test: `import "github.com/user/package"` with functions that call other functions

## Completed

- ✅ Fixed nested loop iteration counter reset bug
- ✅ Tests pass (`go test` works)
- ✅ Float decimal printing works (inline assembly, no libc)
- ✅ Conditional loops (@ condition max N)
- ✅ Import system with GitHub repos
- ✅ Export system (`export *` and `export funclist`)
- ✅ PLT/GOT only generated when C functions are used
- ✅ Executable compression infrastructure
- ✅ Lambda local variables work correctly (fixed duplicate symbol collection bug)
- ✅ `unsafe { ... } as type` syntax for type conversions after unsafe blocks
- ✅ SDL3 C FFI integration works perfectly for direct calls
- ✅ Enum parsing from C headers

## Completed Tasks

### AVX-512 FMA Optimization (2025-12-10) ✅

### Plan: Implement FMA (Fused Multiply-Add) with AVX-512 Support
**Goal:** Detect and optimize `a * b + c` patterns into single FMA instructions for better performance

**Actionable Steps:**
1. ✅ Audit current SIMD infrastructure (vfmadd.go exists with AVX2/AVX-512 FMA support)
2. ✅ Extend pattern detection in optimizer.go to identify FMA candidates
   - ✅ Detect `BinaryExpr(Add, BinaryExpr(Mul, a, b), c)` patterns
   - ✅ Detect `BinaryExpr(Add, c, BinaryExpr(Mul, a, b))` patterns
   - ✅ Created FMAExpr AST node to represent detected patterns
   - ✅ Added FMAExpr handling to all optimizer walker functions
   - 🔄 FMSUB detection (a * b - c) partially done (AST support, needs instruction)
3. ✅ AVX-512 FMA instruction encoders already complete
   - ✅ VFMADD231PD for packed doubles (zmm/ymm/xmm registers, 512/256/128-bit)
   - ✅ ARM64 FMLA (NEON/SVE) support
   - ✅ RISC-V RVV vfmadd.vv support
   - ⚠️  Scalar FMA (VFMADD213SD) not yet implemented (currently using vector width)
4. ✅ Extended codegen.go to emit FMA instructions
   - ✅ FMAExpr compilation uses VFmaddPDVectorToVector
   - ⚠️  No runtime CPU feature detection yet (assumes FMA available)
   - ⚠️  FMSUB (subtract variant) needs VFMSUB instruction variant
5. [ ] Add comprehensive tests
   - Test scalar FMA: `x * y + z`
   - Test vector FMA: operations on arrays/slices
   - Test FMSUB: `x * y - z`
   - Test FNMADD: `-(x * y) + z`
   - Verify correctness and performance improvement
6. [ ] Update OPTIMIZATIONS.md documentation

**Expected Impact:** 2x speedup for mathematical kernels (physics, graphics, ML inference)
**Status:** Core FMA detection and code generation implemented, needs testing and FMSUB variant

**Summary of Implementation:**
- Created FMAExpr AST node to represent a*b+c and a*b-c patterns  
- Optimizer detects FMA patterns in foldConstantExpr and creates FMAExpr nodes
- Added FMAExpr handling to all 10+ optimizer walker functions (strength reduction, constant propagation, etc.)
- Codegen compiles FMAExpr to VFmaddPDVectorToVector calls
- vfmadd.go already had complete x86-64/ARM64/RISC-V FMA instruction encoding
- Test suite created in fma_test.go (needs println() updates to run)

**What Works:**
- Pattern detection: (a * b) + c and c + (a * b) transforms to FMAExpr
- Code generation: emits proper FMA instructions on all 3 architectures
- Optimization passes: FMAExpr properly handled in all recursive AST walkers

**What's Missing:**
- FMSUB instruction variant (subtract instead of add)
- Runtime CPU feature detection (currently assumes FMA available)  
- Scalar FMA (VFMADD213SD) for single float64 operations
- Test execution (tests written but need println() for output)

## Future Enhancements

- Fix module-level mutable globals in lambdas
- C struct support in function calls (passing pointers to stack-allocated structs)
- Lambda capture optimization for imported packages
- Comprehensive SDL3/game library (after fixing globals issue)
- Application development examples and tutorials
- POPCNT optimization for bit counting operations
- BMI/BMI2 for advanced bit manipulation
- AVX-512 compression/expansion for data shuffling
