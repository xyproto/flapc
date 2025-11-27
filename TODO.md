# Flap TODO

## Core Features

### Parser ✅
- Track column positions for better error messages ✅
- Re-evaluate blocks-as-arguments syntax

### Optimizer
- Re-enable when type system is complete
- Add integer-only optimizations for `unsafe` blocks

### Code Generation
- Add explicit float32/float64 conversions where needed
- Replace malloc with arena allocation in string/map/list operations
- Optimize O(n²) algorithms

### Type System
- Complete type inference
- Ensure C types integrate with Flap's universal type
- Add runtime type checking (optional)

### Standard Library
- Expand minimal runtime
- Add common game utilities
- Document all builtins

### Code Quality ✅
- Fixed ARM64 type safety (uint8 shift overflow warnings) ✅
- All go vet warnings resolved (except intentional unsafe.Pointer uses) ✅
- Test coverage: 23.5% with 208+ test functions ✅
- Comprehensive error handling tests added ✅

## Known Issues

### printf Implementation
- Calculate proper offset in printf
- Improve float-to-string conversion
- Add ARM64/RISC-V assembly versions

### RISC-V Backend
- Load actual addresses for rodata symbols
- Implement PC-relative loads
- Add CSR instructions

### Test Fixes
- Fix superscript character printing
- Fix "bare match clause" error

## Future Enhancements

- Hot reload improvements (patch running process via IPC)
- WASM target
- WebGPU bindings
- More comprehensive test suite
- Performance profiling tools
- Interactive REPL
- Language server protocol support
- Package manager
