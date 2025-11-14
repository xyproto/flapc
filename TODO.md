# TODO - Bug Fixes

**Test Status:** 119/147 passing (81%)  
**Goal:** 95%+ pass rate for Flap 2.0 release

**Recent Progress:** Created MEMORY.md documenting memory management philosophy

---

## Critical Bugs

### 1. Remove Malloc from Cons Operator (HIGH PRIORITY)
**Failing Tests:** 1 test (list_cons)
**Issue:** #1 - Memory management philosophy violation

**Problem:**
```flap
result := 1 :: [2, 3]  // Segfaults due to malloc in _flap_list_cons
```

**Root Cause:**
The cons operator (`::`) currently calls malloc directly, which violates the Flap memory philosophy. According to MEMORY.md and LANGUAGE.md:
- Cons should use arena allocation
- Malloc should ONLY be used for: (1) user c.malloc(), (2) arena metadata, (3) arena growth
- The current implementation crashes in parallel contexts and with stack alignment issues

**Solution:**
1. Create a global default arena for implicit allocations
2. Rewrite `_flap_list_cons` to use arena allocation instead of malloc
3. If inside explicit arena block, use that arena; otherwise use default arena

**Files:** 
- `codegen.go` (line 7697-7700: remove malloc, use arena allocation)
- `flap_runtime.go` (implement global default arena)
- `MEMORY.md` (documents the philosophy - already done)

---

### 2. Map Update Returns Wrong Value
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
- May be related to key hashing or lookup logic

**Files:** `codegen.go` (map update logic)

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
