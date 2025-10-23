# Flapc Bug Investigation Results

## Summary

Successfully fixed nested loop bug and identified root causes of other reported issues.

## Fixed Issues

### ✅ Nested Loops Bug - FIXED
**Status:** Resolved

**Root Cause:**  
Loop counters and limits were stored in callee-saved registers (r12, r13) which caused interference between nested loop levels.

**Solution:**  
Changed from register-based to stack-based storage:
- Each loop allocates dedicated stack space (16 bytes) for counter and limit
- Load values from stack to registers at loop start
- Store values back to stack after incrementing
- Each nested loop level has isolated stack slots

**Files Changed:**
- `parser.go`: Lines 4419-4516 (compileRangeLoop function)

**Tests Passing:**
- test_nested_var_bounds ✓
- test_simple_nested ✓
- test_nested_trace ✓  
- test_nested_loops_i ✓ (@i1/@i2 outer loop access)
- test_nested_with_printf ✓

## Investigated Issues (Not Fixed)

### 🔍 Lambda-Returning-Lambda (Closures)
**Status:** Requires major feature implementation

**Root Cause:**  
Inner lambdas cannot access outer lambda parameters because there's no closure capture mechanism. When generating lambda code, the compiler creates a new empty variable scope, losing access to outer parameters.

**Example that fails:**
```flap
makeAdder := (x) => {
    -> (y) => x + y  // ERROR: undefined variable 'x'
}
```

**Location:** parser.go:7168-7247 (generateLambdaFunctions)

**Fix Complexity:** High - requires implementing full closure capture with environment passing

### 🔍 Match Expressions with String Results
**Status:** Runtime crash identified, needs debugging

**Root Cause:**  
Match expressions work correctly with numeric results but crash with string results.

**Example that works:**
```flap
n { n == 1 -> 100 n == 2 -> 200 ~> 999 }  // ✓ Works
```

**Example that fails:**
```flap
n { n == 1 -> "one" n == 2 -> "two" ~> "other" }  // ✗ Crashes with "Error"
```

**Location:** parser.go:6365-6454 (compileMatchExpr), needs investigation of string handling

**Fix Complexity:** Medium - likely an issue with how strings are loaded/stored in match clauses

### 🔍 Recursive Lambdas
**Status:** Multiple issues identified

**Issue 1:** Can't call lambda by its own name
- Causes symbol lookup error at link time
- Workaround: Use `me` keyword for self-reference

**Issue 2:** Non-tail recursion with `me` incorrect
- `me` keyword triggers tail call optimization (jump instead of call)
- Doesn't work for non-tail-recursive patterns like `me(n-1) * n`
- Returns wrong results (e.g., factorial returns 1 instead of 120)

**Location:** parser.go:9643-9684 (compileTailCall)

**Fix Complexity:** Medium - need to detect whether `me` call is in tail position

## Test Files Created

During investigation, created these test files:
- test_closure.flap - Demonstrates closure bug
- test_match_numeric.flap - Match with numeric results (works)
- test_lambda_match.flap - Recursive lambda test
- test_simple_lambda_match.flap - Basic lambda match (works)
- test_lambda_match_guard.flap - Lambda with guards (works)
- test_match_inline.flap - Inline match with strings (fails)
- test_match_numbers.flap - Inline match with numbers (works)
- test_match_string_simple.flap - Simple string match (fails)

## Recommendations

1. **High Priority:** Fix match expressions with string results - blocks common use cases
2. **Medium Priority:** Implement closure capture - enables functional patterns
3. **Medium Priority:** Fix non-tail `me` recursion - or document tail-only restriction
4. **Low Priority:** Consider allowing lambdas to reference themselves by name (not just `me`)
