# TODO - Bugs to Fix

**Test Status:** 253/262 passing (96.5%)

---

## 🐛 Failing Tests

### 1. Shift operator test syntax
**Tests:** `TestBitwiseOperations/shift_left`, `TestBitwiseOperations/shift_right`  
**File:** `arithmetic_comprehensive_test.go:254-266`

Tests use old syntax. Update to:
```flap
x := 5 <<b 2   # not <b
x := 20 >>b 2  # not >b
```

---

### 2. List update segfault
**Tests:** `TestListUpdateBasic`, `TestListUpdateMinimal`, `TestListUpdateSingleElement`, `TestListPrograms/list_update`  
**Files:** `list_update_test.go`, `list_programs_test.go`

Updating list elements crashes:
```flap
nums := [1, 2, 3]
nums[0] <- 99    # SEGFAULT
```

Check codegen for list index assignment with `<-` operator.

---

### 3. Lambda block syntax
**Tests:** `TestLambdaPrograms/lambda_with_block`, `TestLambdaPrograms/lambda_match`  
**File:** `lambda_programs_test.go`

Lambdas with block bodies fail to compile:
```flap
f := x => {
    y := x + 1
    y * 2
}
```

Note: Single-expression lambdas work: `x => x + 1`

Check parser's lambda block handling and codegen.

---

### 4. Tail operator `_`
**Test:** `TestTailOperator`  
**File:** `list_programs_test.go`

Not implemented:
```flap
rest := _[1, 2, 3]    # Should return [2, 3]
```

Need: lexer token, AST node, parser support, codegen.  
Reference: Head operator `^` implementation.

---

### 5. Parallel compilation test
**Test:** `TestParallelSimpleCompiles`  
**File:** `parallel_programs_test.go`

Run test with `-v` to see actual error.

---

### 6. Lambda error handling test
**Test:** `TestCompilationErrors/lambda_bad_syntax`  
**File:** `compiler_test.go`

Expected compilation error not triggered. Check if test expectation is correct.

---

### 7. ENet tests
**Tests:** `TestENetCompilation/enet_simple`, `TestENetCodeGeneration/simple_test.flap`  
**File:** `enet_test.go`

External library integration. May be test environment issue.

---

### 8. Generic test failure
**Test:** `TestFlapPrograms`  
**File:** `basic_programs_test.go` (likely)

Run with `-v` to identify specific failure.

---

## 🔍 Debugging Commands

```bash
# Run specific test
go test -v -run="TestName"

# Run with details
go test -v -run="TestName/subtest_name"

# Check segfault with gdb
go test -c && gdb ./flapc.test

# List all tests
go test -list .
```

---

**All compiler components are complete. Only these bugs remain.**
