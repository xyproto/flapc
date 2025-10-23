# Flap Compiler Test Results

## Summary

**Total tests:** 67  
**Passing:** 60 (89.6%)  
**Failing:** 7 (10.4%)  

## Test Results by Category

### ✅ Passing Tests (60)

All core functionality tests pass, including:
- Nested loops (3+ levels) with printf
- Float and integer printf formats
- Pattern matching and guards  
- Lambda expressions and currying
- Arithmetic and recursion with match expressions
- String operations
- Loop variable access across nesting levels

### ❌ Failing Tests (7)

#### 1. test_cache.flap - Map Assignment Not Implemented
**Issue:** Map literal syntax `[:]` is not recognized (should use `{}`), and map index assignment doesn't properly update values.  
**Root cause:** Map assignment (`m[key] = value`) requires cache implementation.  
**Status:** Feature not yet implemented.

#### 2. test_closure.flap - Closures Not Implemented  
**Issue:** Error "undefined variable 'y' at line 0"  
**Root cause:** Inner lambdas cannot capture variables from outer lambda scopes.  
**Status:** True closures not yet implemented.

#### 3. test_factorial_simple.flap - Recursive Lambda Symbol Resolution
**Issue:** Runtime error "undefined symbol: factorial"  
**Root cause:** Named lambdas that call themselves recursively don't generate proper symbols for linking.  
**Test code:**
```flap
factorial := (n) => n <= 1 {
    -> 1
    ~> n * factorial(n - 1)  // Symbol 'factorial' not found at link time
}
```
**Status:** Compiler bug in recursive lambda code generation.

#### 4. test_lambda_match.flap - Same as #3
Recursive lambda symbol resolution issue.

#### 5. test_recursion_no_match.flap - Infinite Recursion (Test Bug)
**Issue:** Runs infinitely into negative numbers.  
**Root cause:** Test has no base case to stop recursion.  
**Test code:**
```flap
fact := (n, acc) => {
  printf("fact(%ld, %ld)\n", n as number, acc as number)
  me(n - 1, acc * n)  // No stopping condition!
}
```
**Status:** Test is buggy, not a compiler issue.

#### 6. test_simple_recursion.flap - Same as #5
Infinite recursion due to missing base case in test.

#### 7. test_two_param_recursion.flap - Same as #5  
Infinite recursion due to missing base case in test.

## Major Fixes Completed

### Printf Stack Alignment Fix
**Commit:** 4a74601

Fixed segmentation faults when calling printf with float arguments inside nested loops.

**Problem:**  
- Loops allocate 24 bytes per level on stack
- x86-64 ABI requires 16-byte alignment before function calls
- Nested loops caused misalignment → crashes in printf

**Solution:**  
Added dynamic stack alignment before printf calls:
```go
fc.out.MovRegToReg("r10", "rsp")
fc.out.AndRegWithImm("r10", 0xF) // r10 = misalignment amount
fc.out.SubRegFromReg("rsp", "r10") // Align stack
// Call printf
fc.out.AddRegToReg("rsp", "r10") // Restore stack
```

**Tests fixed:** 13 tests that were crashing now pass (26% improvement)

### Integer Printf Format Support
**Commit:** 4a74601

Added support for integer format specifiers in printf:
- `%d`, `%i`, `%u` - integer formats
- `%ld`, `%li`, `%lu` - long integer formats  
- `%x`, `%X`, `%o` - hex and octal formats

Values are automatically converted from float64 to integers when using these formats.

## Recommendations

1. **test_cache:** Implement map index assignment or document as unsupported
2. **test_closure:** Implement closures or document as future feature
3. **test_factorial_simple, test_lambda_match:** Fix recursive lambda symbol generation
4. **test_*recursion_no_match:** Add base cases to tests or mark as infinite recursion examples
