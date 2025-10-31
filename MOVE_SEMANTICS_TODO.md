# Move Semantics Implementation Status

## Completed
- ✅ Added TOKEN_BANG to lexer for `!` operator
- ✅ Added MoveExpr AST node
- ✅ Updated parser to handle `expr!` syntax in postfix position
- ✅ Added movedVars tracking fields to FlapCompiler
- ✅ Added use-after-move check in IdentExpr compilation
- ✅ Added MoveExpr compilation logic
- ✅ Added MoveExpr case to getExprType

## Issues Found
**Optimizer Interaction Bug**: The constant propagation/inlining optimizer is removing statements like `x := 42` before the collectSymbols phase runs. This causes "undefined variable" errors when trying to compile `x!` because x was never added to fc.variables.

### Example:
```flap
x := 42         // This statement gets optimized away
y := x! + 100   // Error: undefined variable 'x'
```

### Root Cause:
The optimization phase runs BEFORE collectSymbols, and it inlines the constant 42 directly into the expression. This removes the variable x from the AST entirely.

### Solution Options:
1. Disable constant inlining for moved variables
2. Run collectSymbols before optimization
3. Have optimizer preserve variable definitions even when inlined
4. Add move operator awareness to optimizer

## Next Steps
1. Fix optimizer to handle move semantics correctly
2. Add proper test cases once optimizer is fixed
3. Update LANGUAGE.md documentation
4. Commit the feature

