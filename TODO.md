# Flap Compiler TODO

## ✅ Recently Completed

### 1. ✅ FIXED: macOS Mach-O Dynamic Linking
**Status**: WORKING! (commit 0870c56)

**What was fixed**:
- ✅ Indirect symbol table 8-byte alignment (required by dyld)
- ✅ Separate import-only string table for chained fixups
- ✅ Correct page_start offset to GOT within __DATA segment
- ✅ Updated chained fixups size calculation for 4 segments
- ✅ TestMachOExecutable now passes
- ✅ Dynamic linking fully functional

**Result**: ARM64 binaries with `printf()`, `exit()`, etc. now execute successfully!

---

## 🚨 Current Issues

## 📋 Language Syntax Improvements

### 2. Simplify Loop Syntax
- [ ] Allow `@` instead of `@+` for loops (if parseable)
  - Current: `@+ i in range(10) { }`
  - Proposed: `@ i in range(10) { }`

### 3. Add Compound Assignment Operators
- [ ] Implement `+=`, `-=`, `*=`, `/=`
  - Current: `sum := sum + i`
  - Proposed: `sum += i`
  - Files to modify: `parser.go` (lexer + parser), `*_codegen.go` (all architectures)

### 4. Remove Requirement for exit(0)
- [ ] Make `exit(0)` implicit at end of programs
  - Programs should automatically return/exit if they reach the end
  - Requires: Check if last statement is already exit(), if not, emit exit(0)

### 5. Simplify Match Block Syntax
- [ ] Allow implicit `->` in match blocks when there's only one branch
  - Current: `x < y { -> println("yes") }`
  - Proposed: `x < y { println("yes") }`

### 6. Fix Lambda Assignment Syntax
- [ ] Require `=>` for lambda assignments, not `->`
  - Current: `double = (x) -> x * 2` (should fail)
  - Proposed: `double = x => x * 2` (correct)
  - Also allow dropping parentheses for single parameter

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

**Version**: 1.0.0 (x86-64 complete), 1.1.0-dev (ARM64 in progress)

**Test Results**:
- x86-64: 178/178 (100%) ✅
- ARM64: Testing in progress (dynamic linking now working!)
- Mach-O: 10/10 tests pass ✅ (TestMachOExecutable now passes!)

**Blockers**:
1. ~~macOS dynamic linking (SIGKILL)~~ - ✅ **FIXED!** (commit 0870c56)
2. RISC-V backend incomplete - Medium priority
3. Missing ARM64 expression types - Low priority (workarounds exist)

**Recent Wins**:
- ✅ **macOS ARM64 dynamic linking FIXED!** (commit 0870c56)
- ✅ Self-signing implementation complete (no codesign tool needed)
- ✅ Symbol naming fixed (single underscore)
- ✅ ARM64 binaries with printf(), exit() now execute successfully
