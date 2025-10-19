# Flap Compiler TODO

## ✅ Recent Progress (2025-10-19)

**x86-64 Linux: 186/188 tests passing (99%)**
- Fixed parser error handling (panic/recover instead of os.Exit)
- Standardized lambda syntax to `=>` across all test files
- Math functions working via x87 FPU hardware instructions
- Only 2 tests skipped: fstring_test (not implemented), match_unicode (compiler issue)

**ARM64 macOS: ~125/182 tests estimated (69%)**
- ✅ Fixed os.Exit(1) calls in parser.go (replaced all 66 with compilerError())
- ✅ Implemented ParallelExpr (|| operator) - unlocks 21 tests
- ✅ Implemented type tracking system (varTypes, getExprType)
- Previous session: 104/182 (57%) before os.Exit fix

**Latest Session Improvements:**
1. ✅ Replaced all 66 os.Exit(1) calls with compilerError() for proper test handling
2. ✅ Implemented parallel map operator (||) for ARM64
3. ✅ Added type tracking system for proper operator handling
4. ✅ Added helpful error messages for unimplemented concat operations
5. Note: Cannot verify test count due to execution environment issue

**Remaining x86-64 Issues:**
1. ✅ ~~Fix os.Exit calls in compiler code generation~~ - COMPLETE
2. Implement F-string interpolation (P1 priority)
3. Fix compiler race conditions for parallel test execution

## 🚨 Current High-Priority ARM64 Issues

Estimated ~57 tests still failing:

### Expression Types (~15 failures)
- ✅ ~~ParallelExpr (22 tests)~~ - IMPLEMENTED
- ✅ ~~Type tracking (getExprType)~~ - IMPLEMENTED (foundation for concat)
- **List/String concatenation** (8 tests) - Type detection done, needs runtime functions
- **SliceExpr** (2 tests) - List/string slicing `list[start:end:step]`
- **PipeExpr** (1 test) - Pipe operator `|`
- **JumpExpr** (1 test) - Loop break/continue

### Missing Functions (22 failures)
- **Math functions** (15 tests): Disabled due to compilation hanging issue
- **Recursion** (4 tests): `me` keyword for tail-recursive calls
- **Type conversion** (3 tests): `str` function

### Runtime Issues (~20 failures)
- **Lambda crashes**: Stack frame or calling convention issues

### Missing Operators (7 failures)
- **Bitwise**: xor, shl (<<), shr (>>), rol, ror
- **Logical**: and, or

## 📋 Language Syntax Improvements (Priority Order)

### P1: F-String Interpolation
- [ ] Add Python-style f-strings
  - `f"Hello, {name}! You are {age} years old"`
  - Compile-time string interpolation
  - File: `lexer.go`, `parser.go`

### P2: Lambda Assignment Syntax ✅ COMPLETE
- [x] Standardize lambda arrow to `=>`
  - Current: `double = (x) -> x * 2`
  - Proposed: `double = x => x * 2`
  - Also allow dropping parentheses for single parameter
  - File: `lexer.go`, `parser.go`
  - **Status**: Implemented and all test files updated

### P2: Fix O(n²) CString Conversion
- [ ] Optimize CString conversion from O(n²) to O(n)
  - Current implementation does linear search for each character
  - File: `parser.go:5774` (compileMapToCString)

---

## 🏗️ ARM64 Backend Completion

### 7. Implement Missing Expression Types
- [ ] **SliceExpr**: List/string slicing `list[start:end:step]`
  - Requires runtime functions for slice operations
  - File: `arm64_codegen.go`

- [ ] **UnaryExpr**: Head (^) and Tail (&) operators
  - `^list` - get first element
  - `&list` - get all but first element
  - File: `arm64_codegen.go`

### 8. Loop Enhancements
- [ ] Implement break/continue (`ret @N`, `@N`)
  - Requires jump label tracking per loop
  - File: `arm64_codegen.go`

- [ ] Support `@first`, `@last`, `@counter`, `@i` special variables
  - Stack-allocated iteration state
  - File: `arm64_codegen.go`

- [ ] Handle nested match blocks
  - Label generation for nested scopes
  - File: `arm64_codegen.go`

### 9. Function Enhancements
- [ ] **Recursive calls**: Proper stack frame management
  - Test: factorial, fibonacci functions

- [ ] **Tail call optimization**: Jump instead of call for tail position
  - Replace `BL` with `B` when function ends with call

---

## 🧪 Testing and Quality

### 10. Fix Integration Test Failures
- [ ] Debug printf test failures (currently failing due to SIGBUS)
- [ ] Run full test suite on ARM64 once dynamic linking works
- [ ] Verify output matches x86-64 version

### 11. Code Quality Improvements
- [ ] Fix O(n²) CString conversion in `parser.go:5737`
  - Current implementation is quadratic
  - Should be linear with proper buffer management

---

## 🏗️ RISC-V Support (Version 1.2.0)

### 12. RISC-V Backend Implementation
- [ ] Implement RISC-V register allocation (x0-x31, f0-f31)
- [ ] Implement RISC-V calling convention (a0-a7, fa0-fa7)
- [ ] Add floating-point instructions (FADD.D, FSUB.D, FMUL.D, FDIV.D)
  - File: `riscv64_instructions.go:380-385`
- [ ] Add multiply/divide instructions (MUL, MULH, DIV, REM)
- [ ] Add logical instructions (AND, OR, XOR)
- [ ] Add shift instructions (SLL, SRL, SRA)
- [ ] Add atomic instructions (LR, SC, AMO*)
- [ ] Add CSR instructions
- [ ] Fix PC-relative load for rodata symbols
  - File: `riscv64_codegen.go:153`
- [ ] Fix identifier loading
  - File: `riscv64_codegen.go:83`

---

## 📚 Standard Library

### 13. I/O Functions
- [ ] Fix `read_file(path)` - Files read successfully but `cstr_to_flap_string` has bug
  - File: `parser.go:4097`
- [ ] Implement `readln()` - Read line from stdin
  - File: `parser.go:4108`

### 14. String Functions
- [ ] Implement `num(string)` - Parse string to float64
- [ ] Implement `split(string, delimiter)` - Split into list
- [ ] Implement `join(list, delimiter)` - Join with delimiter
- [ ] Implement `upper/lower/trim(string)` - String manipulation

### 15. Collection Functions
- [ ] Implement `map(f, list)` - Apply function to elements
- [ ] Implement `filter(f, list)` - Filter by predicate
- [ ] Implement `reduce(f, list, init)` - Fold with binary function
- [ ] Implement `keys/values(map)` - Extract keys/values from maps
- [ ] Implement `sort(list)` - Sort in ascending order

---

## 🚀 ARM64 Additional Instructions

### 16. ARM64 Floating-Point Instructions
- [ ] Add more FP instructions (beyond current FADD, FSUB, FMUL, FDIV, FCMP)
  - File: `arm64_instructions.go:434`
- [ ] Implement SIMD/NEON instructions for vectorization
  - File: `arm64_instructions.go:435`

### 17. ARM64 Memory and Arithmetic
- [ ] Add load/store pair instructions (STP, LDP)
  - File: `arm64_instructions.go:436`
- [ ] Add multiply/divide (MUL, UDIV, SDIV)
  - File: `arm64_instructions.go:437`
- [ ] Add logical instructions (AND, OR, EOR)
  - File: `arm64_instructions.go:438`
- [ ] Add shift instructions (LSL, LSR, ASR, ROR)
  - File: `arm64_instructions.go:439`

### 18. ARM64 Printf Enhancements
- [ ] Add support for printf arguments beyond format string
  - File: `arm64_codegen.go:1489`
- [ ] Implement proper float-to-string conversion
  - File: `arm64_codegen.go:963`

---

## 🎯 Performance Optimizations

### 19. Map Operations
- [ ] Implement AVX-512 map lookup (8 keys/iteration)
- [ ] Add perfect hashing for compile-time constant maps
- [ ] Implement binary search for maps with 32+ keys

### 20. Compiler Optimizations
- [ ] Add constant folding (compile-time evaluation)
- [ ] Optimize CString conversion from O(n²) to O(n)
- [ ] Implement dead code elimination

---

## 📖 Documentation

### 21. Update Documentation
- [x] Update README.md with macOS signing status
- [ ] Add architecture comparison guide
- [ ] Document calling conventions for each architecture

---

## 🔧 Future Features (Version 2.0+)

### 22. Advanced Language Features
- [ ] Multiple lambda dispatch (overloading by arity)
- [ ] Pattern matching with destructuring
- [ ] Method call sugar: `obj.method(args)` syntax
- [ ] Regex matching: `=~` and `!~` operators
- [ ] Gather/scatter: `@[indices]` syntax
- [ ] SIMD annotations for explicit vectorization

### 23. Standard Library Packages
- [ ] HTTP package (basic HTTP client)
- [ ] JSON package (parsing and serialization)
- [ ] Testing package (assert functions and framework)
- [ ] Collections package (advanced data structures)

---

## 📊 Current Status

**Version**: 1.1.0 (ARM64 mostly complete, new assignment semantics)

**Test Results**:
- x86-64: 178/178 (100%) ✅
- ARM64: 104/182 passing (57%) ✅ 🎉
- Mach-O: 10/10 tests pass ✅

**ARM64 Test Failures by Category**:
- ParallelExpr not implemented (21 tests)
- Math functions runtime crash (13 tests) - dynamic linking issues
- String concatenation not implemented (8+ tests)
- Recursion `me` keyword (4 tests)
- Functions `str`, `call` (5 tests)
- SliceExpr (2 tests)
- Lambda/function-related failures (~25 tests)

**Major Recent Wins**:
- ✅ **Printf %v and %b formats** - Smart formatting and booleans! (+19 tests)
- ✅ **Printf argument order** - Fixed reverse evaluation! (+3 tests)
- ✅ **String interning** - String literals now reuse same address! (+2 tests)
- ✅ **IN operator** - Fixed LSL and FMOV encodings! (+4 tests)
- ✅ **Comparison operators** - Fixed all CSET encodings! (+3 tests)
- ✅ **List indexing** - Fixed stack alignment and LSL encoding!
- ✅ **Logical/bitwise operators** - and, or, xor, shl, shr, rol, ror working!
- ✅ **Three-operator assignment system** - Prevents variable shadowing bugs!
- ✅ **macOS ARM64 dynamic linking** - No more SIGKILL!
- ✅ **Printf with all argument types** - Variadic functions working!
- ✅ **Variable storage and binary expressions** - Core arithmetic works!
- ✅ **Compound assignments** - `+=`, `-=`, etc. fully functional
- ✅ **Loop syntax simplified** - `@` instead of `@+`
- ✅ **Auto exit(0)** - No need to write explicit exit calls

**Blockers**:
1. ~~macOS dynamic linking~~ - ✅ **FIXED!**
2. ~~Variable shadowing bugs~~ - ✅ **FIXED!**
3. Parallel expressions not implemented for ARM64 - Medium priority
4. Many lambda/list/string tests failing - Medium priority
5. RISC-V backend incomplete - Low priority

**Next Steps**:
1. Fix remaining ARM64 test failures (lambda, list, string operations)
2. Implement ParallelExpr for ARM64
3. Add f-string interpolation (P1)
4. Standardize lambda syntax to `=>` (P2)
5. Fix O(n²) CString conversion (P2)
