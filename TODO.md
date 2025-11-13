# TODO - Remaining Issues

**Test Status:** 114/147 tests passing (77.6%)

---

## 🐛 Critical Issues

### 1. List mutation causes segfault (HIGH PRIORITY)
**Tests:** `TestListUpdateBasic`, `TestListUpdateMinimal`, `TestListUpdateSingleElement`, `TestListPrograms/list_update`, `TestListOperationsComprehensive/list_update`  
**Files:** `list_update_test.go`, `list_programs_test.go`, `string_map_test.go`

**Problem:** List literals are stored in .rodata (read-only memory), so updating them causes segfault:
```flap
nums := [1, 2, 3]
nums[0] <- 99    # SEGFAULT - trying to write to read-only memory
```

**Solution:** Implement arena-based allocation for list literals (started in commit 26f59f4 but reverted due to issues):
- Lists need to be allocated in writable memory (heap or arena)
- Use `flap_arena_alloc` or `malloc` for list creation
- Fix `initializeMetaArenaAndGlobalArena()` implementation
- Test arena allocation thoroughly before re-enabling

**Files to modify:** `codegen.go` (case *ListExpr, around line 3860)

---

### 2. List cons operator crashes
**Test:** `TestListOperationsComprehensive/list_cons`

Cons operator `::` also crashes, likely for same reason as list mutation (trying to modify read-only data).

**Solution:** Same as #1 - arena-based allocation.

---

### 3. Lambda block syntax not working
**Tests:** `TestLambdaPrograms/lambda_with_block`, `TestLambdaPrograms/lambda_match`  
**File:** `lambda_programs_test.go`

Lambdas with block bodies fail to compile:
```flap
f := x => {
    y := x + 1
    y * 2
}
```

Note: Single-expression lambdas work fine: `x => x + 1`

**Investigation needed:** Check parser's lambda block handling and codegen for block-based lambdas.

---

## 🔧 Medium Priority Issues

### 4. ENet tests failing
**Tests:** `TestENetCompilation/enet_simple`, `TestENetCodeGeneration/simple_test.flap`  
**File:** `enet_test.go`

External library integration issue. May be test environment setup problem.

---

### 5. Lambda error test not triggering
**Test:** `TestCompilationErrors/lambda_bad_syntax`  
**File:** `compiler_test.go`

Expected compilation error not triggered. Verify test expectation is correct per LANGUAGE.md spec.

---

### 6. Map update test failing
**Test:** `TestMapOperations/map_update`

Similar to list update - may be related to mutability issues.

---

## 📝 Implementation Notes

### Arena Allocator Status
The arena allocator infrastructure exists but has issues:
- `initializeMetaArenaAndGlobalArena()` - needs debugging
- `flap_arena_alloc` - runtime function exists
- Global arena creation - currently commented out due to crashes

**Next steps:**
1. Debug meta-arena initialization
2. Test arena alloc thoroughly with simple programs
3. Convert list literals to use arena allocation
4. Ensure proper cleanup on program exit

### Test Infrastructure
All tests use isolated temp directories to prevent cross-contamination.
Test helpers in `test_helpers.go` handle compilation and execution.

---

## 🔍 Debugging Commands

```bash
# Run all tests
go test

# Run specific test with details
go test -v -run="TestName/subtest"

# Check segfault with timeout
timeout 2 ./compiled_program

# Build and test manually
./flapc test.flap test && ./test

# Count passing tests
go test -v 2>&1 | grep -E "^---\s(PASS):" | wc -l
```

---

**Status:** Core compiler is complete and functional. Remaining issues are primarily around memory management for mutable data structures.
