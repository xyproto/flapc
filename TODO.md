# TODO - Flap Compiler (Flapc)

- [ ] Add back the "import" feature, for being able to import directly from git repos with .flap source code files.
- [ ] Add an internal utility function for sorting a Flap type (map[uint64]float64) by key. This can be needed before calling the head or tail operators.
- [ ] Rename the tail operator from `_` to `¤` and then implement/fix it, and then enable the tests.
- [ ]

- Both versions create new lists but key renumbering logic has bugs
- Head operator `^list` works correctly
- Tests for tail() function are skipped
- Workaround: Use list slicing or manual iteration
- Complex fix needed: proper key renumbering in UNIVERSAL MAP format
- Marked for post-1.4.0 fixes (not critical for core functionality)

### Lambda Local Variables
- ⚠️ Local variables in lambda bodies not yet supported
- This is a deliberate design choice to simplify lambda frame management
- Workaround: Use lambda parameters or expression-only bodies
- Example: `f = x -> x + 1` ✅ works
- Example: `f = x -> { y := x + 1; y }` ❌ doesn't work yet
- Lambda assignments (closures) are allowed: `inner = y -> x + y` ✅

### Memory Management
- Currently using malloc for dynamic allocations
- Arena allocator infrastructure exists but not fully integrated
- TODOs in codegen.go mark locations that should use arena allocation
- All tests pass with current malloc-based approach

---

## Post-1.4.0 Enhancements

### Priority 1 - Core Improvements
- Fix tail operator `_list` to return correct results
- Complete arena allocator integration (replace malloc calls in codegen.go)
- Add local variable support in lambda bodies

### Priority 2 - Language Features
- Pattern destructuring in match clauses
- More operator overloading
- Full tail call optimization for mutual recursion

### Priority 3 - Type System
- Add type byte prefix to all values for runtime type checking
- Enable type introspection (typeof, is_string, etc.)
- Better error messages with type information

### Priority 4 - Performance
- Better constant folding and dead code elimination
- More aggressive register allocation
- SIMD optimizations for arithmetic operations

### Priority 5 - Tooling
- Debugger support (DWARF debug info)
- Better error messages with column numbers and suggestions
- Package manager for dependencies
- Language server protocol (LSP) implementation

### Priority 6 - Platform Support
- Windows native support (PE/COFF format)
- WebAssembly target
- Better ARM64 and RISC-V support

### Priority 7 - Standard Library
- String manipulation functions
- File I/O operations
- JSON parsing/generation
- HTTP client/server
- Regular expressions
- Math library (beyond basic arithmetic)

---

## Architecture

- **Direct machine code generation**: AST → x86-64/ARM64/RISCV64 (no IR)
- **SIMD optimizations**: AVX-512/SSE2 for map operations
- **Register allocation**: Smart allocation with callee-saved registers
- **Arena allocator**: Infrastructure in place, integration in progress
- **Tail call optimization**: Automatic for recursive functions
- **Result type**: NaN-boxing for efficient error handling
