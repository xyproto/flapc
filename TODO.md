# C67 Compiler TODO

## Critical Bugs

### 1. SDL3 Function Signatures
- SDL_RenderFillRect expects (renderer, SDL_FRect*) but is being called with (renderer, x, y, w, h)
- Need to support creating C structs on the stack and passing pointers
- Alternatively, find SDL functions that accept individual parameters

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

## Future Enhancements

- C struct support in function calls (passing pointers to stack-allocated structs)
- Lambda capture optimization for imported packages
- More comprehensive SDL3 wrapper library
- Application development examples and tutorials
