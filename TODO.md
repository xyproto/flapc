# TODO - Bug Fixes

**Test Status:** 125/130 passing (96.2%) ✅
**New Goal:** 98%+ pass rate (128/130 tests needed - 3 more tests!)

**Recent Progress:** 
- ✅ 95% GOAL EXCEEDED: Now at 96.2%!
- ✅ FIXED: List iteration for map-based lists
- ✅ REDESIGNED: Lists now use universal map representation
- ✅ FIXED: List/map update bug
- ✅ FIXED: println crash bug
- ✅ FIXED: ENet tests - added example files
- ✅ FIXED: Lambda bad syntax test

---

## Remaining Issues (5 failing tests)

### String Variable Printing (3 tests)
**Failing Tests:** string_variable, string_concatenation, empty_string

**Issue:** println() doesn't handle string variables - prints "0.000000" instead of string content

**Solution:** Use write syscall to print character by character:
- Strings are maps: `[count][0][char0][1][char1]...`
- Loop through indices, load character codes
- Use `write(1, buffer, 1)` syscall for each char
- No PLT needed - pure syscalls!

**Files:** `codegen.go` (line 10091-10105)

---

### Lambda Block Bodies (2 tests)
**Failing Tests:** lambda_with_block, lambda_match

**Problem:**
```flap
f := x => {
    y := x + 1
    y * 2
}  // Not implemented
```

Single expressions work: `f := x => x + 1`

**Solution:** BlockExpr bodies need proper code generation in lambda compilation

**Files:** `codegen.go` (lambda compilation)

---

## Performance Wins from List Redesign
- O(1) indexing (was O(n) with linked lists)
- O(1) length (was O(n))
- O(1) updates
- Better cache locality
- 400+ lines of code removed

