# Flap v1.7.4 Final Release Plan

**Target:** Frozen language specification and production-ready compiler
**Timeline:** 5 days
**Goal:** Last version with language changes. Only portability and security fixes after this.

---

## 10-Step Action Plan

### Step 1: Fix Critical Bugs ✅ PRIORITY 1
**Task:** Fix cstruct_arena_test.flap - values wrong (10.0 instead of 10.5)

**Why:** Data corruption bug in write_f64 offset calculation

**Action:**
- Debug Vec3_SIZEOF and field offset computation
- Verify all cstruct field offset calculations
- Test with various field types (float32, float64, int32, etc)

**Verification:**
```bash
./flapc testprograms/cstruct_arena_test.flap
./testprograms/cstruct_arena_test
# Should print correct values (10.5, not 10.0)
```

---

### Step 2: Fix Race Conditions in Tests ✅ PRIORITY 1
**Task:** Fix parallel test suite race conditions

**Why:** test_string_different and test_not fail when run in parallel

**Action:**
- Identify shared state (likely global string/map pools)
- Isolate per-goroutine state or add proper synchronization
- Review all global variables for thread safety

**Verification:**
```bash
for i in {1..10}; do go test || exit 1; done
# All 10 runs should pass
```

---

### Step 3: Complete Arena Allocator Runtime ✅ PRIORITY 1
**Task:** Implement arena block runtime (parser already done)

**Why:** Essential for per-frame game allocations - the main use case

**Action:**
- Add runtime functions to flap_runtime.c:
  - `_flap_arena_init(size)` - Create arena (4KB default)
  - `_flap_arena_alloc(arena_ptr, size)` - Bump-pointer allocation
  - `_flap_arena_free(arena_ptr)` - Free entire arena
- Modify codegen in parser.go:
  - Generate arena_init() at block entry
  - Replace alloc() with arena_alloc() inside arena blocks
  - Generate arena_free() at block exit (even on early return)
- Support nested arenas (arena stack in TLS)

**Test Program:**
```flap
// Per-frame allocation pattern
main ==> {
    @ frame in 0..<1000 {
        arena {
            entities := alloc(1000 * 64)  // Allocate entity buffer
            // ... use entities ...
        }  // Instant cleanup, zero fragmentation
    }
    println("1000 frames completed")
}
```

**Verification:**
```bash
valgrind --leak-check=full ./arena_test
# Should show 0 memory leaks
```

---

### Step 4: Constant Folding Optimization 📊 QUALITY
**Task:** Implement compile-time constant folding

**Why:** Basic optimization expected in production compiler

**Action:**
- Add constant folding pass in parser during expression compilation
- Handle arithmetic: `2 + 3` → `5`, `10 * 0` → `0`
- Handle logical: `true and false` → `false`
- Handle comparisons: `5 > 3` → `true`

**Test Cases:**
```flap
x := 2 + 3           // Should compile to: x := 5
y := 10 * 0          // Should compile to: y := 0
z := true and false  // Should compile to: z := false
```

**Verification:**
- Check generated assembly has no redundant computations
- Disassemble output and verify immediate values used

---

### Step 5: Dead Code Elimination 📊 QUALITY
**Task:** Remove unreachable code after ret/jumps

**Why:** Cleaner assembly, smaller binaries

**Action:**
- Add reachability analysis pass after parsing
- Mark statements as reachable/unreachable
- Skip code generation for unreachable statements
- Warn user about unreachable code (optional)

**Test Case:**
```flap
f := x => {
    ret 42
    println("This should not be compiled")
    y := 100
}
```

**Verification:**
- Assembly should contain no code for unreachable statements
- Binary size should decrease for programs with dead code

---

### Step 6: Improve Error Messages 💬 USABILITY
**Task:** Better error messages for common mistakes

**Why:** Last chance to improve developer experience before freeze

**Action:**
- Undefined variable: suggest similar names using Levenshtein distance
  - Error: `undefined variable 'lenth'` → `Did you mean 'length'?`
- Type mismatch in C FFI: show expected vs got
  - Error: `Expected ptr, got float64`
- Missing imports: suggest "use" statement
  - Error: `Undefined 'SDL_Init'` → `Did you forget: use "sdl3"`
- Line/column numbers for all errors

**Verification:**
- Trigger each error type intentionally
- Review error messages for clarity and helpfulness

---

### Step 7: Test Edge Cases 🧪 ROBUSTNESS
**Task:** Test all parallel loop edge cases

**Why:** Parallel loops are a key feature, must be bulletproof

**Test Cases:**
```flap
// Empty range
@@ i in 0..<0 { println("Should not execute") }

// Single iteration
@@ i in 0..<1 { println("One thread") }

// Large range
@@ i in 0..<10000000 { /* stress test */ }

// Thread spawn failure (simulate by ulimit -u)
@@ i in 0..<1000 { /* many threads */ }
```

**Action:**
- Add test programs for each edge case
- Handle errors gracefully (no segfaults)
- Document when to use `@@` vs `@` in LANGUAGE.md

**Verification:**
- All edge cases pass or fail with clear error messages
- No crashes, hangs, or undefined behavior

---

### Step 8: Documentation Completeness 📚 ESSENTIAL
**Task:** Ensure LANGUAGE.md covers ALL features with examples

**Why:** This is the spec - must be 100% complete for frozen release

**Action:**
1. Verify every keyword has examples:
   - Review lexer token list vs LANGUAGE.md
   - Ensure all operators documented
   - Check all builtin functions listed

2. Add "Common Patterns" section:
   - Defer cleanup pattern
   - Arena per-frame allocation
   - Parallel loop work distribution
   - C FFI usage patterns
   - Error handling patterns

3. Add "Gotchas and Pitfalls" section:
   - Map is float64-valued (not typed)
   - Immutable by default (use `<-` for mutation)
   - Arena blocks require proper nesting
   - Parallel loops share no state by default

4. Add "Performance Guide":
   - When to use parallel loops
   - Memory allocation best practices
   - Cache-friendly data layouts

**Verification:**
- Read through entire LANGUAGE.md as if learning the language
- Can you write non-trivial programs from docs alone?

---

### Step 9: Create Comprehensive Test Suite 🧪 VERIFICATION
**Task:** Ensure test coverage for all language features

**Why:** These tests will verify future ports work correctly

**Action:**
1. Feature coverage tests:
   - defer_test.flap ✅ (already exists)
   - fstring_test.flap ✅ (already exists)
   - arena_test.flap (create)
   - parallel_test.flap (create)
   - atomic_test.flap (expand)
   - cstruct_test.flap ✅ (already exists)

2. Negative tests (should fail to compile):
   - undefined_variable_test.flap
   - type_mismatch_test.flap
   - syntax_error_test.flap

3. Performance regression tests:
   - Baseline timings for common operations
   - Detect if changes cause slowdowns

**Verification:**
```bash
go test -cover
# Target: >85% test coverage
```

---

## Summary

### Priority Breakdown:
- **Must Fix (Steps 1-3):** Critical bugs and core feature
- **Quality (Steps 4-5):** Expected optimizations
- **Polish (Steps 6-7):** UX and robustness
- **Verification (Step 8):** Comprehensive testing
- **Documentation (Step 9):** Complete specification

### Recommended Schedule:
- **Day 1:** Steps 1-2 (fix critical bugs) ✅
- **Day 2:** Step 3 (arena runtime) ✅
- **Day 3:** Steps 4-5 (optimizations) 📊
- **Day 4:** Steps 6-7 (usability + edge cases) 💬
- **Day 5:** Steps 8-10 (docs + tests + release) 🎉



---

## Items Explicitly EXCLUDED

These are NOT blocking the v1.7.4 freeze:

### Not Critical for Language Freeze:
- ❌ Register allocator integration - Infrastructure exists, runtime optimization can be added later without language changes, might be added already
- ❌ Multiple return values - Would require language design changes

### Can Be Added Post-Freeze (Backward Compatible):
- Performance optimizations (register allocation, inlining, etc)
- Platform ports (FreeBSD, RISC-V, ARM64, etc)
- Build system improvements

## Success Criteria for v1.7.4

The release is ready when:

1. ✅ All Step 1-3 bugs fixed (critical)
2. ✅ Steps 4-5 optimizations implemented (quality)
3. ✅ Step 6-7 UX improvements done (usability)
4. ✅ Step 8 documentation complete (essential)
5. ✅ Step 9 test coverage >85% (verification)
6. ✅ Clean `go test && go build` on fresh clone
7. ✅ All 344+ test programs pass
8. ✅ Step 10 release tagged and documented (finalization)
9. ✅ LANGUAGE.md marked as frozen specification
10. ✅ README clearly states language is frozen

**After v1.7.4: Only portability fixes, security patches, and performance optimizations. No language changes.**
