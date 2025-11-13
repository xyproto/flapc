# TODO - Bug Fixes

**Test Status:** 122/176 passing (69.3%)  
**Goal:** 95%+ pass rate for Flap 2.0 release

**Recent Progress:** Fixed list literal allocation - all list operations now work except cons operator

---

## Critical Bugs

### 1. Lambda Block Bodies Not Working
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

**Files:** `codegen.go` (map update logic)

---

### 3. List Cons Operator Crashes
**Failing Tests:** 1 test (list_cons)

**Problem:**
```flap
result := 1 :: [2, 3]  // Segfaults
```

**Investigation:**
- Check `_flap_list_cons` runtime implementation
- Verify cons operator codegen in `case "::"`
- Cons should create new list: [1, 2, 3]

**Files:** `codegen.go` (lines ~3914-3934)

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
