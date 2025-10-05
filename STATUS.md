# Flap Compiler - Implementation Status

## ✅ Implemented Features

### Core Language
- **Comments**: `//` for single-line comments
- **Variables**: Mutable (`:=`) and immutable (`=`) assignment
- **Data Types**: Float64 foundation (all values are float64)
- **Arithmetic**: `+`, `-`, `*`, `/` (scalar double-precision)
- **Comparisons**: `<`, `<=`, `>`, `>=`, `==`, `!=`
- **Length Operator**: `#list` returns the length of a list

### Control Flow
- **Conditionals**: `if`/`else`/`end` blocks with comparison operators
- **Loops**: `@ identifier in range(n) { }` syntax
- **Builtin Functions**: `range(n)`, `println()`, `exit()`, `len()`

### Data Structures
- **Lists**: Literal syntax `[1, 2, 3]`, stored in .rodata with length prefix
- **List Indexing**: Access elements with `list[index]`
- **List Iteration**: Loop over elements with `@ item in list { }`
- **List Length**: Get list length with `len(list)`, returns 0.0 for empty lists
- **Empty Lists**: `[]` evaluates to 0 (null pointer)

### Functions & Lambdas
- **Lambda Expressions**: `(x) -> x * 2` or `(x, y) -> x + y`
- **First-Class Functions**: Store lambdas in variables
- **Function Pointers**: Functions represented as addresses (reinterpreted as float64)
- **Closures**: Lambdas capture no external state (stateless)
- **Calling Convention**: Up to 6 parameters in xmm0-xmm5, result in xmm0

### Code Generation
- **Architectures**: x86-64 (primary), ARM64 (partial), RISC-V (partial)
- **Instructions**: 
  - Scalar FP: ADDSD, SUBSD, MULSD, DIVSD, CVTSI2SD, CVTTSD2SI
  - Comparisons: UCOMISD with conditional jumps
  - Stack operations: Proper 16-byte alignment
  - Memory: MOVSD for XMM register loads/stores
  
### Binary Generation
- **Format**: ELF64 with dynamic linking
- **Sections**: .text, .rodata, .data, .bss, .dynamic, .got, .plt
- **Relocations**: PC-relative for data, PLT for external functions
- **ABI**: Proper x86-64 calling convention with stack alignment

## 🚧 In Progress / Planned

### SIMD Features (Core Language Feature)
- [ ] Parallel operator `||` for SIMD operations (parsing done, codegen needs work)
- [x] Lambda expressions `(x) -> expression` ✅
- [ ] Gather/scatter `@[]` for sparse access
- [ ] Mask type for predication
- [ ] Reductions `||>` (sum, max, min, etc.)
- [ ] Fused multiply-add `*+`

### Advanced Features
- [ ] Pattern matching `=~` and `~`
- [ ] Objects `@{ }` with methods
- [ ] Self-reference `me`
- [ ] Error handling `or!` operator
- [x] List/array literals `[1, 2, 3]` ✅
- [x] List indexing `list[index]` ✅
- [x] List iteration `@ item in list { }` ✅
- [x] List length `len(list)` ✅
- [ ] List methods (append, etc.)
- [ ] Map literals `{key: value}`

### Language Constructs
- [ ] Function definitions
- [ ] Recursion support
- [ ] String operations
- [ ] Type annotations (mask, float64)

## 📊 Test Coverage

### Passing Tests
- ✅ Arithmetic: 10 + 3 = 13, 10 - 3 = 7, 10 * 3 = 30, 10 / 3 = 3
- ✅ Comparisons: All 6 operators (<, <=, >, >=, ==, !=)
- ✅ Loops: Simple (range(5)), nested (3x3), with arithmetic
- ✅ Conditionals: if/else branching
- ✅ Variables: Assignment and reassignment
- ✅ Lists: Literals [1, 2, 3], indexing list[0], multiple lists, empty lists
- ✅ List Length: `len(list)` for both empty and non-empty lists
- ✅ List Iteration: `@ item in list { println(item) }`
- ✅ Loop variables: Using iterator in expressions (i * 2)
- ✅ Lambda Expressions: `(x) -> x * 2`, storage, calling, multi-argument
- ✅ First-Class Functions: Multiple lambdas, passing results between calls

## 🐛 Known Issues
- **Parallel Operator Crash**: The `||` operator for SIMD map operations is parsed correctly but crashes at runtime. All individual components (lambdas, lists, function calls) work correctly in isolation.

## 🏗️ Architecture

### Compilation Pipeline
1. **Lexer**: Tokenization with keyword recognition
2. **Parser**: Recursive descent parser producing AST
3. **Code Generator**: Direct machine code emission
4. **ELF Builder**: Complete ELF64 file generation
5. **Two-pass**: Initial codegen → address resolution → final codegen

### Stack Frame Layout
```
[rbp + 0]     = saved rbp
[rbp - 8]     = alignment padding
[rbp - 24]    = first variable (16-byte aligned)
[rbp - 40]    = second variable (16-byte aligned)
...
```

### Calling Convention
- Float64 arguments/returns: xmm0
- Integer arguments: rdi, rsi, rdx, rcx, r8, r9
- Return address: rax (integers), xmm0 (floats)
- Stack: 16-byte aligned before CALL
- XMM registers: Used for all float64 operations

## 📈 Next Steps

1. **Fix Parallel Operator `||`**: Complete SIMD map operation (parsing done, codegen buggy)
2. **List with Lambda Integration**: `list || (x) -> x * 2` for element-wise operations
3. **User-Defined Functions**: Named function definitions with `name = (params) -> body`
4. **Closures**: Lambda capture of outer scope variables
5. **Pattern Matching**: Core language feature `=~`

