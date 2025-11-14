# TODO - Bug Fixes

**Test Status:** 118/130 passing (91%)  
**Goal:** 95%+ pass rate for Flap 2.0 release

**Recent Progress:** 
- ✅ FIXED: List/map update bug - inline traversal and mutation for linked lists
- ✅ FIXED: println crash bug - added null terminators to format strings
- Fixed cons operator type recognition (:: now returns "list" type)
- Cons-built lists work correctly with indexing and printf

---

## Critical Bugs

### 1. ✅ FIXED: println Crashes After Cons Operations
**Status:** RESOLVED - println now works correctly

**Fix Applied:**
- Added `\x00` null terminators to all println format strings
- Printf was reading garbage without null terminators
- Added return statements to prevent case fall-through
- Added fflush to PLT functions list

**Commit:** e9f2151

---

### 2. ✅ FIXED: List Update Returns Wrong Value
**Status:** RESOLVED - list updates now work correctly

**Fix Applied:**
- Lists are linked lists (cons cells: [head][tail]), not arrays
- Removed broken `_flap_list_update` that assumed array structure
- Emit inline code that walks to index-th cons cell and mutates head
- Maps still use array-style in-place update

**Commit:** 8535877

---

### 3. Lambda Block Bodies Not Working
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

## Medium Priority

### 4. ENet Integration
**Failing Tests:** 2 tests (ENet compilation/codegen)

May be test environment issue (missing libenet). Verify:
- ENet library available
- Linking works
- Test expectations are correct

**Files:** `enet_test.go`, `enet_codegen.go`

---

### 5. Compilation Error Tests
**Failing Tests:** 1 test (lambda_bad_syntax)

Test expects compilation to fail but it succeeds. Review test expectations against LANGUAGE.md spec.

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
- Cons operator now uses arena allocation (fixed!)
- Default arena (arena0) initialized at program start
- Lists built with cons use arena0 automatically
- See MEMORY.md for full details

**Debugging Approach:**
1. Create minimal failing test case
2. Compile with `./flapc -o test test.flap`
3. Use printf instead of println to isolate issue
4. Inspect machine code with `objdump -d test`
5. Run with timeout to catch segfaults
6. Use gdb if needed: `gdb ./test`

**Priority:** Fix println/cons interaction first (blocks 1 test), then lambda blocks (blocks 2 tests), then map update (blocks 1 test).

---

**Next:** After fixing these bugs, run full test suite and aim for 95%+ pass rate.
