# TODO - Bug Fixes

**Test Status:** 125/130 passing (96.2%) ✅  
**New Goal:** 100% pass rate (130/130 tests) 🎯
**Remaining:** 5 tests to fix (1 is skipped intermittently)

**Recent Progress:** 
- ✅ 95% GOAL EXCEEDED: Now at 96.2%!
- ✅ FIXED: List iteration for map-based lists
- ✅ REDESIGNED: Lists now use universal map representation
- ✅ FIXED: List/map update bug
- ✅ FIXED: println crash bug
- ✅ FIXED: ENet tests
- ✅ FIXED: Lambda bad syntax test

---

## Path to 100% (5 tests remaining)

### Priority 1: String Variable Printing (4 tests)
**Failing Tests:**
- TestStringOperations/string_variable
- TestStringOperations/string_concatenation  
- TestStringOperations/empty_string
- TestBasicPrograms/fstring_basic

**Issue:** `println(string_variable)` outputs "0.000000" instead of string content

**Solution:** Implement write syscall for character-by-character output
- Location: `codegen.go` lines 10091-10105
- Algorithm documented in STRING_PRINTING_SOLUTION.md
- Use direct syscalls (no PLT dependencies)

**Impact:** +4 tests → 129/130 (99.2%)

---

### Priority 2: Lambda Features (2 tests)
**Failing Tests:**
- TestLambdaPrograms/lambda_with_block
- TestLambdaPrograms/lambda_match

**Issues:**
1. Lambdas with block expressions not implemented
```flap
f := x => {
    y := x + 1
    y * 2
}  // Not supported - needs BlockExpr codegen
```

2. Lambdas with match expressions
```flap
classify := x => x {
    0 -> "zero"
    ~> x > 0 { -> "positive" ~> "negative" }
}
```

**Solution:** Implement BlockExpr and MatchExpr body compilation in lambda generation
- Single expression lambdas work: `f := x => x + 1`
- Need to handle multi-statement blocks and match expressions

**Impact:** +2 tests → 130/130 (100%) 🎉

---

## Implementation Order
1. String printing (quick win - algorithm ready)
2. Lambda blocks (more complex)

