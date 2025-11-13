# TODO - Bugs to Fix for Flap 2.0

**Status:** 96.5% tests passing (253/262)  
**Goal:** Fix remaining 9 test failures before release

---

## 🐛 Bugs (From Test Failures)

### 1. Shift operators use wrong syntax in tests
**Files:** `arithmetic_comprehensive_test.go`  
**Issue:** Tests use old syntax `<b` and `>b` instead of new `<<b` and `>>b`  
**Fix:** Update test code to use correct syntax from LANGUAGE.md  
**Priority:** HIGH (trivial fix)

```flap
# Current (wrong):
x := 5 <b 2    # Should be <<b
x := 20 >b 2   # Should be >>b

# Correct:
x := 5 <<b 2
x := 20 >>b 2
```

---

### 2. List update operations cause segfault
**Files:** `list_update_test.go`, `list_programs_test.go`  
**Tests failing:**
- TestListUpdateBasic
- TestListUpdateMinimal  
- TestListUpdateSingleElement
- TestListPrograms/list_update

**Issue:** Updating list elements causes core dump  
**Example:**
```flap
nums := [1, 2, 3]
nums[0] <- 99    # CRASH
```

**Priority:** CRITICAL  
**Likely cause:** List update codegen not implemented or incorrect memory handling  
**Related:** Cons operator `::` implementation may help fix this

---

### 3. Lambda with block syntax fails
**Files:** `lambda_programs_test.go`  
**Tests failing:**
- TestLambdaPrograms/lambda_with_block
- TestLambdaPrograms/lambda_match

**Issue:** Lambda bodies with curly braces `{}` don't compile correctly  
**Example:**
```flap
f := x => {
    y := x + 1
    y * 2
}
```

**Priority:** HIGH  
**Note:** Single-expression lambdas work fine: `x => x + 1`

---

### 4. Tail operator `_` not implemented
**Files:** `list_programs_test.go`  
**Test:** TestTailOperator

**Issue:** Tail operator for lists not implemented  
**Example:**
```flap
rest := _[1, 2, 3]    # Should return [2, 3]
```

**Priority:** MEDIUM  
**Status:** Feature specified in LANGUAGE.md but not implemented  
**Related:** Head `^` operator is implemented

---

### 5. Parallel compilation test fails
**Files:** `parallel_programs_test.go`  
**Test:** TestParallelSimpleCompiles

**Issue:** Unknown - needs investigation  
**Priority:** MEDIUM  
**Note:** Parallel execution tests pass, only compilation test fails

---

### 6. Compilation error test fails
**Files:** `compiler_test.go`  
**Test:** TestCompilationErrors/lambda_bad_syntax

**Issue:** Expected compilation error not triggered  
**Priority:** LOW (test expectations may be wrong)

---

### 7. ENet tests fail
**Files:** `enet_test.go`  
**Tests:**
- TestENetCompilation/enet_simple
- TestENetCodeGeneration/simple_test.flap

**Issue:** ENet library integration broken  
**Priority:** LOW (external library, not core feature)  
**Note:** May be test environment issue

---

### 8. Flap programs test fails
**Files:** `basic_programs_test.go` or similar  
**Test:** TestFlapPrograms

**Issue:** Needs investigation - generic test name  
**Priority:** MEDIUM

---

## ✅ Everything Else Works!

All other features are complete and tested:
- Lexer (119 tokens)
- Parser (100% grammar coverage)
- AST (52 node types)
- Semantic analysis
- Optimizer
- Codegen (x86_64)
- Arithmetic, logical, comparison operators
- Bitwise operators (except test syntax issues)
- Pattern matching
- Loops (sequential and parallel)
- C FFI
- CStruct
- Arena allocation
- Defer, spawn, unsafe blocks
- Random operator `???`

---

## 🎯 Fix Priority

1. **Update shift operator tests** (5 minutes)
2. **Fix list update segfault** (CRITICAL - needs investigation)
3. **Fix lambda block syntax** (HIGH)
4. **Implement tail operator** (MEDIUM)
5. **Investigate other failures** (MEDIUM-LOW)

**Estimated effort to 100% tests passing:** 4-8 hours

---

## 📝 Notes

- The compiler architecture is sound (lexer/parser/AST/optimizer/codegen all complete)
- Most bugs are in codegen for specific edge cases
- List operations need attention (update, cons, tail)
- Lambda block bodies need parser/codegen fix

**After these bugs are fixed, Flap 2.0 is ready for release!**

---

**Last Updated:** 2025-11-13  
**Test Results:** 253/262 passing (96.5%)  
**Next Milestone:** 100% tests passing
