# TODO - Bug Fixes

**Test Status:** 119/147 passing (81%)  
**Goal:** 95%+ pass rate for Flap 2.0 release

**Recent Progress:** 
- Fixed cons operator type recognition (:: now returns "list" type)
- Cons-built lists work correctly with indexing and printf

---

## Critical Bugs

### 1. println Crashes After Cons Operations (HIGH PRIORITY)
**Failing Tests:** 1 test (list_cons with println)
**Issue:** #1 - println segfaults when used after cons operations

**Problem:**
```flap
lst := 1 :: 2 :: []
println(lst[0])  // Crashes with segfault
printf("%f\n", lst[0])  // Works fine
```

**Root Cause:**
- Cons operations use arena allocation (calling flap_arena_alloc)
- Arena allocator may call realloc which is a C function
- println implementation has inline string formatting code
- Crash occurs specifically with println, not printf
- Issue appears to be register/stack corruption in println inline code
- Warning: "No PLT entry or label for fflush" suggests fflush being called incorrectly

**Workaround:** Use printf instead of println after cons operations

**Solution Steps:**
1. Investigate why fflush is being called when printf wasn't used
2. Check if println's inline code properly saves/restores registers
3. Verify stack alignment in println after cons operations
4. Consider: should println use printf internally instead of inline code?

**Files:** 
- `codegen.go` (line 9961-10100: println implementation)
- Need to debug register usage and stack state

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
