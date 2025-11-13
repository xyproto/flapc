# TODO - Bug Fixes

**Test Status:** 118/147 passing (80.3%)  
**Goal:** 95%+ pass rate for Flap 2.0 release

**Language Spec Status:** ✅ FIXED - Mutability semantics clarified in LANGUAGE.md

---

## Critical Bugs

### 1. List/Map Mutation Implementation
**Failing Tests:** 14 tests (list_update_*, list_cons, map_update)  
**Root Cause:** `:=` collections stored in .rodata instead of writable memory

**Language Design (NOW CLARIFIED):**
- `=` creates immutable variable with immutable value (stored in .rodata)
- `:=` creates mutable variable with mutable value (must be in writable memory)

**Implementation Fix Needed:**
```flap
nums := [1, 2, 3]     // Must allocate in writable memory (heap/arena)
nums[0] <- 99         // Should work (currently segfaults)

readonly = [1, 2, 3]  // Can stay in .rodata
readonly[0] <- 99     // Should error (currently does)
```

**Solution:**
1. When generating code for `:=` with list/map literal:
   - Allocate writable memory (malloc or arena)
   - Copy literal data from .rodata to writable memory
   - Return pointer to writable copy
2. Keep `=` behavior (direct .rodata reference)
3. Update codegen.go ListExpr and MapExpr cases

**Files:** `codegen.go` (lines ~3860)

---

### 2. Lambda Block Bodies Not Working
**Failing Tests:** 2 tests (lambda_with_block, lambda_match)

**Problem:**
```flap
f := x => {
    y := x + 1
    y * 2
}  // Fails
```
Single expressions work: `f := x => x + 1`

**Investigation:**
- Check parser handling of `LambdaExpr` with `BlockExpr` body
- Verify codegen for lambda closures with multiple statements
- Compare with function definition codegen (which works)

**Files:** `parser.go` (parseLambda), `codegen.go` (case *LambdaExpr)

---

### 3. Map Update Returns Wrong Value
**Failing Tests:** 1 test (map_update)

**Problem:**
```flap
m := {a: 10}
m[a] <- 20
println(m[a])  // Prints 0 instead of 20
```

**Investigation:**
- Check `__flap_map_update` implementation
- Verify map update codegen generates correct machine code
- May be related to same root cause as list mutation

**Files:** `codegen.go` (map update logic)

---

## Medium Priority

### 4. Parallel Loop Edge Cases
**Failing Tests:** 1 test (TestParallelSimpleCompiles)

Basic parallel loops work, but some edge case fails. Investigate specific test failure.

**Files:** `parallel_programs_test.go`

---

### 5. ENet Integration
**Failing Tests:** 2 tests (ENet compilation/codegen)

May be test environment issue (missing libenet). Verify:
- ENet library available
- Linking works
- Test expectations are correct

**Files:** `enet_test.go`, `enet_codegen.go`

---

### 6. Compilation Error Tests
**Failing Tests:** Various error-checking tests

Some tests expect compilation to fail but it succeeds. Review test expectations against LANGUAGE.md spec.

**Files:** `compiler_test.go`

---

## Quick Commands

```bash
# Run all tests
go test

# Run specific test
go test -v -run="TestName"

# Debug segfault
./flapc test.flap test && timeout 2 ./test || echo "Crashed: $?"

# Check machine code
objdump -d ./test | less
```

---

## Implementation Notes

**Memory Strategy:**
- Short-term: Use malloc for list/map literals
- Long-term: Debug arena allocator, switch to arena-based allocation
- Arena infrastructure exists in `flap_runtime.go`

**Debugging Approach:**
1. Create minimal failing test case
2. Compile with `./flapc -o test test.flap`
3. Inspect machine code with `objdump -d test`
4. Run with timeout to catch segfaults
5. Use gdb if needed: `gdb ./test`

**Priority:** Fix list mutation first (blocks 14 tests), then lambda blocks (blocks 2 tests).

---

**Next:** After fixing these bugs, run full test suite and aim for 95%+ pass rate.
