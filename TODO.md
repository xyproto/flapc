# C67 Compiler TODO

## Critical Bugs

### 1. Module-Level Mutable Globals in Lambdas
- **Status:** Identified, needs fix
- When a lambda modifies a module-level mutable variable (`:=`), the change doesn't persist after lambda returns
- Example:
  ```c67
  g_value := 0
  set_value = (x) -> { g_value <- x }
  main = { set_value(42); println(g_value) }  # Prints 0 instead of 42
  ```
- This affects libraries that use globals for state (like c67game)
- **Workaround:** Pass state explicitly as parameters and return values

### 2. Import System Issues
- Lambdas from imported C67 packages may be generated multiple times
- Need to verify closure object initialization for imported functions
- Test: `import "github.com/user/package"` with functions that call other functions

## Completed

- ✅ Fixed nested loop iteration counter reset bug
- ✅ Tests pass (`go test` works)
- ✅ Float decimal printing works (inline assembly, no libc)
- ✅ Conditional loops (@ condition max N)
- ✅ Import system with GitHub repos
- ✅ Export system (`export *` and `export funclist`)
- ✅ PLT/GOT only generated when C functions are used
- ✅ Executable compression infrastructure
- ✅ Lambda local variables work correctly (fixed duplicate symbol collection bug)
- ✅ `unsafe { ... } as type` syntax for type conversions after unsafe blocks
- ✅ SDL3 C FFI integration works perfectly for direct calls
- ✅ Enum parsing from C headers

## Future Enhancements

- Fix module-level mutable globals in lambdas
- C struct support in function calls (passing pointers to stack-allocated structs)
- Lambda capture optimization for imported packages
- Comprehensive SDL3/game library (after fixing globals issue)
- Application development examples and tutorials
