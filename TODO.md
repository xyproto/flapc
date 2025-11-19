# TODO - Flap 3.0+

**Status:** 3.0 Release Ready - All tests passing (155 tests)
**Current Version:** 3.0.0

---

## ✅ Completed for 3.0

- ✅ All core language features working
- ✅ pop() function with multiple returns
- ✅ Deeply nested loops (5+ levels)
- ✅ += operator for lists and numbers
- ✅ Array indexing with SIMD optimization
- ✅ Register tracking to prevent clobbering
- ✅ Closures (lambda assignments)
- ✅ Multiple return values (a, b = [1, 2])
- ✅ Result type with NaN-boxing error encoding
- ✅ error() function for creating custom errors
- ✅ .error property for extracting error codes
- ✅ or! operator for error handling with defaults
- ✅ compileAndRun helper function in run.go
- ✅ 155 tests passing

---

## Future Enhancements (Post-3.0)

### Type System Enhancement
- Add type byte prefix to all values for runtime type checking
- Enable type introspection (typeof, is_string, etc.)
- Better error messages with type information

### Language Features
- Implicit match blocks in function bodies (if desired)
- More operator overloading
- Pattern destructuring in match clauses

### Performance Optimizations
- Full tail call optimization for mutual recursion
- Better constant folding and dead code elimination
- More aggressive register allocation
- SIMD optimizations for arithmetic operations

### Tooling
- Debugger support (DWARF debug info)
- Better error messages with column numbers and suggestions
- Package manager for dependencies
- Language server protocol (LSP) implementation

### Platform Support
- Windows native support (PE/COFF format)
- WebAssembly target
- Better ARM64 and RISC-V support

### Standard Library
- String manipulation functions
- File I/O operations
- JSON parsing/generation
- HTTP client/server
- Regular expressions
- Math library (beyond basic arithmetic)

### Testing
- Fuzz testing
- Property-based tests
- Stress tests for memory management
- Cross-platform compatibility tests

---

## Known Limitations

### Lambda Local Variables
- ⚠️ Local variables in lambda bodies not yet supported
- Workaround: Use lambda parameters or expression-only bodies
- Example: `f = x => x + 1` ✅ works
- Example: `f = x => { y = x + 1; y }` ❌ doesn't work
- Lambda assignments (closures) are allowed: `inner = y => x + y` ✅
- This is a deliberate design choice to simplify lambda frame management
- Full support would require complex stack frame analysis

---

## Architecture

- **Direct machine code generation**: AST → x86-64/ARM64/RISCV64 (no IR)
- **SIMD optimizations**: AVX-512/SSE2 for map operations
- **Register allocation**: Smart allocation with callee-saved registers
- **Arena allocator**: Scope-based memory management
- **Tail call optimization**: Automatic for recursive functions
