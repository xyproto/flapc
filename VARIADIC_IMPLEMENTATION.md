# Variadic Functions Implementation

## Status: Infrastructure Complete - Argument Collection TODO

### Completed:
1. ✅ Updated GRAMMAR.md with variadic parameter syntax
2. ✅ Updated LANGUAGESPEC.md with variadic function documentation
3. ✅ Added TOKEN_ELLIPSIS to lexer
4. ✅ Added lexer support for `...` token
5. ✅ Updated AST LambdaExpr to include VariadicParam field
6. ✅ Updated parser to handle variadic parameters in lambda expressions
7. ✅ Added VariadicParam to LambdaFunc codegen structure
8. ✅ Implemented variadic argument collection in lambda generation (function def side)
9. ✅ Compiler builds successfully

### Remaining Work:
1. ⚠️  Codegen: Complete variadic list construction (PARTIALLY WORKING)
   - ✅ Function receives args in xmm registers
   - ✅ r14 register used to pass variadic arg count  
   - ✅ Call sites pass arg count in r14 (both direct and stored function calls)
   - ✅ Track which functions are variadic via functionSignatures map
   - ✅ Function entry works and can be called
   - ✅ Empty list stub works (returns count=0)
   - ⚠️  Complex list construction with actual args causes segfault
   - Issue: Stack manipulation during function entry causes crashes
   - Need: Build list in pre-allocated frame space OR use arena properly
   - Current workaround: Using empty list stub for now

2. ✅ Type tracking for variadic functions (DONE)
   - ✅ Store VariadicParam info for named functions via functionSignatures
   - ✅ Check at call sites if function is variadic
   - ✅ Pass r14 register with variadic count

3. ❌ Codegen: Implement spread operator for function calls
   - Allow `func(values...)` to unpack list into variadic args
   - Support mixing: `func(1, 2, values..., 3, 4)`

4. ❌ Update built-in functions to use variadic syntax
   - printf, eprintf, exitf, etc.

5. ❌ Write tests for variadic functions

## Syntax Examples:

```flap
// Define variadic function
sum = (first, rest...) -> {
    total := first
    @ item in rest {
        total <- total + item
    }
    total
}

// Call with individual args
result = sum(1, 2, 3, 4, 5)  // 15

// Call with spread
values = [2, 3, 4, 5]
result = sum(1, values...)  // 15

// Printf with variadic args
printf = (fmt, args...) -> {
    c.printf(fmt, args...)  // Spread args to C function
}

printf("Hello %s, you are %d years old\n", "Alice", 30)
```

## Implementation Notes:

### Calling Convention:
When calling a variadic function `f(a, b, rest...)` with `f(1, 2, 3, 4, 5)`:
1. Fixed params (a=1, b=2) go in registers/stack as normal
2. Remaining args (3, 4, 5) are packed into a list
3. List is passed as an additional parameter

### Code Generation Strategy:
1. **Function definition:** Check if LambdaExpr.VariadicParam is non-empty
2. **Argument collection:** Allocate list for remaining arguments
3. **Function body:** Treat variadic param as a regular list variable
4. **Spread operator:** When calling with `...`, unpack list elements

### C FFI Integration:
For C variadic functions like printf:
```flap
printf = (fmt, args...) -> {
    // Need to unpack args list and pass to C printf
    // This requires special handling in codegen
    c.printf(fmt, args...)
}
```

## Next Steps to Complete Full Implementation:

1. **Complete variadic argument collection:**
   - Save xmm register arguments immediately on function entry (before any operations)
   - Allocate list from arena (NOT malloc or stack manipulation)
   - Copy saved arguments to list with proper key-value pairs
   - Tested approach that crashed: direct memory operations during prologue
   - Recommended: Save args first, then build list after frame is stable

2. **Implement spread operator (`...`) for call sites:**
   - Parse `func(values...)` syntax
   - Unpack list elements into individual arguments
   - Handle mixed: `func(1, 2, list..., 3)

`

3. **Create standard library (stdlib.flap):**
   - Implement printf, eprintf, exitf as variadic Flap functions
   - Use c.printf internally with proper argument unpacking
   - Auto-include stdlib when these functions are used

4. **Write comprehensive tests:**
   - Test variadic with 0, 1, 2, many arguments
   - Test with fixed + variadic parameters
   - Test spread operator
   - Test nested variadic calls

## Implementation Notes:

The segfault during list construction was caused by:
- Stack manipulation (SubImmFromReg on rsp) after frame allocation
- Using registers that contained arguments before saving them
- Complex operations during function entry before stack stabilization

The working approach should be:
1. Function prologue allocates full frame once
2. Save xmm argument registers to temp space immediately
3. Set up regular parameters from saved values
4. Build variadic list from saved values (using LEA from rbp offsets)
5. No stack pointer modification after prologue
