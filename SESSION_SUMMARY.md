# Flapc Compiler Session Summary

## Major Fixes Implemented

### 1. Binary Operation Register Clobbering Bug
**Problem:** Binary operations were storing the left operand in xmm2 register, but when evaluating the right operand (which could contain function calls), xmm2 would be clobbered, corrupting the left operand value.

**Example:** `n * factorial(n - 1)` would fail because the recursive call clobbered xmm2.

**Solution:** Changed to stack-based storage for intermediate values:
```go
fc.compileExpression(e.Left)
fc.out.SubImmFromReg("rsp", 16)
fc.out.MovXmmToMem("xmm0", "rsp", 0)  // Save to stack
fc.compileExpression(e.Right)          // Safe - can clobber registers
fc.out.MovRegToReg("xmm1", "xmm0")
fc.out.MovMemToXmm("xmm0", "rsp", 0)  // Restore from stack
fc.out.AddImmToReg("rsp", 16)
```

**Files Changed:** `parser.go:5489-5501`

### 2. Critical Stack Alignment Bug
**Problem:** Range loops allocated 24 bytes (not a multiple of 16), violating x86-64 ABI's 16-byte stack alignment requirement. Additionally, printf had buggy alignment code that tried to restore rsp using r10 (a caller-saved register) after the printf call, when r10 had been clobbered.

**Symptoms:** Crashes (SIGBUS) when calling printf after nested loops.

**Solution:**
1. Changed range loop allocation from 24 to 32 bytes (16-byte aligned)
2. Removed buggy printf alignment code - no longer needed

**Files Changed:** `parser.go:4420-4429, 4507-4512, 10428-10441`

### 3. List Loop Stack Leak
**Problem:** List loops allocated 64 bytes of stack space but never freed it, causing stack leaks.

**Solution:** Added cleanup code to free 64 bytes at loop exit:
```go
fc.out.AddImmToReg("rsp", 64)
fc.stackOffset -= 64
delete(fc.variables, stmt.Iterator)
delete(fc.mutableVars, stmt.Iterator)
```

**Files Changed:** `parser.go:4663-4682`

### 4. Recursive Lambda Support
**Problem:** Lambdas could not call themselves by name, only using the `me` keyword.

**Solution:** Implemented lambda naming based on assignment context:
- Added `currentAssignName` field to track variable names during assignment
- Lambdas now use their assignment name instead of generated names
- Added `compileLambdaDirectCall` for recursive calls
- Lambda detection in `compileCall` routes to direct call mechanism

**Files Changed:** `parser.go:3477, 4244-4249, 6257-6267, 8991-9017, 10077-10090`

## Documentation Added

### LEARNINGS.md
Added comprehensive documentation on:
- Stack alignment requirements and debugging
- Register clobbering and the Stack-First Principle
- x86-64 calling convention register volatility
- Helper functions (`callMallocAligned`, `compileBinaryOpSafe`)
- Code patterns and red flags to watch for

## Test Results

### Before Fixes
- run_tests.sh: 67 passed, 9 failed
- Multiple crashes in nested loop tests
- Binary operations with recursion failed

### After Fixes
- run_tests.sh: 79 passed, 4 failed
- All nested loop tests pass
- Binary operations with recursion work correctly

### Remaining Failures
1. **test_cache.flap** - Compilation error (list literal elements must be constant)
2. **test_closure.flap** - Compilation error (closures not yet implemented)
3. **test_lambda_match.flap** - Runtime crash (needs investigation)
4. **test_map_simple.flap** - Runtime crash (needs investigation)

## Key Insights

### Stack-First Principle
**Always use stack-based storage for intermediate values across sub-expression evaluations.**

ALL XMM registers are caller-saved in x86-64 System V ABI, meaning they can be clobbered by any function call. The only safe approach is to:
1. Evaluate expression → result in xmm0
2. Save to stack if value must survive a sub-expression
3. Evaluate sub-expression (can clobber all registers)
4. Restore from stack

### Stack Alignment
All stack allocations MUST be multiples of 16 bytes to maintain x86-64 ABI alignment requirements. This is critical for:
- Function calls (especially variadic functions like printf)
- SIMD operations
- Avoiding SIGBUS crashes

### Helper Functions
Created reusable helpers to encapsulate correct patterns:
- `callMallocAligned(sizeReg, pushCount)` - Safe malloc calls with automatic alignment
- `compileBinaryOpSafe(left, right, operator)` - Safe binary operations with stack storage

## Commits in This Session
```
bc014cb Fix critical stack alignment bug in loops and printf
8aa9753 Fix list loop stack cleanup - add missing deallocation
e9c43b4 Add helper function and patterns for safe register management
346987b Document register clobbering and Stack-First Principle
aefbe74 Fix binary operation register clobbering bug
e7a3f94 Implement recursive lambda calls by name
```

## Future Work

### High Priority
1. Investigate and fix test_lambda_match and test_map_simple crashes
2. Implement closure support (variable capture from outer scopes)
3. Fix list literal constant requirement for test_cache

### Optimizations
- Consider using registers for simple loops without nesting or @iN usage
- Implement register allocation for non-nested code paths
- Add compile-time detection of safe register usage scenarios

### Code Quality
- Add automated tests for stack alignment
- Create linter to detect register clobbering patterns
- Add debug mode to verify stack alignment at runtime
