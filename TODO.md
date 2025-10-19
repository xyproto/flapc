# Flap Compiler TODO

## ✅ Recently Completed

### 1. ✅ FIXED: macOS Mach-O Dynamic Linking (commit 0870c56)
- ✅ Indirect symbol table 8-byte alignment (required by dyld)
- ✅ Separate import-only string table for chained fixups
- ✅ Correct page_start offset to GOT within __DATA segment
- ✅ Updated chained fixups size calculation for 4 segments
- ✅ TestMachOExecutable now passes

### 2. ✅ FIXED: String Escape Sequences (commit 863f5a9)
- ✅ Strings now properly interpret `\n`, `\t`, `\r`, `\\`, `\"`
- ✅ Added call to processEscapeSequences() in lexer

### 3. ✅ FIXED: GOT Alignment (commit 7425394)
- ✅ GOT now properly 8-byte aligned after variable-sized rodata
- ✅ Added padding calculation between rodata and GOT
- ✅ Fixes SIGBUS crashes when string sizes change

### 4. ✅ IMPLEMENTED: Printf with Numeric Arguments (commit 78cefb8)
- ✅ Variadic function calling with stack-based argument passing
- ✅ Follows ARM64 calling convention (args on stack, not registers)
- ✅ Works for %g, %f, etc. format specifiers

### 5. ✅ IMPLEMENTED: Printf with String Arguments (commit 4b17dcc)
- ✅ Type detection for string vs numeric arguments
- ✅ String arguments passed as pointers (in x registers)
- ✅ Numeric arguments passed as floats (on stack)
- ✅ Mixed arguments work: `printf("String: %s, Number: %g\n", "test", 42)`

### 6. ✅ FIXED: Multiple Dynamic Function Calls (commit 3e52809)
- ✅ Fixed chained fixups "next" field in GOT entries
- ✅ dyld now processes all GOT entries, not just the first
- ✅ Programs can call multiple dynamic functions: `printf(); exit(0)`

### 7. ✅ IMPLEMENTED: Println Numeric Conversion (commit de8b0cf)
- ✅ println() now handles all numeric types and expressions
- ✅ Delegates to printf("%g\n", value) for non-string arguments
- ✅ String literals still use efficient syscall path
- ✅ Fixed issue where println(num) would print "?"

### 8. ✅ FIXED: ARM64 Variable Storage and Binary Expressions (commit 63a266d)
- ✅ Fixed SIGBUS crashes when using variables in binary expressions
- ✅ Increased stack frame from 32 to 272 bytes (16 for saved regs + 256 for locals)
- ✅ Changed variable storage from negative to positive offsets from x29
- ✅ Fixed BinaryExpr to maintain 16-byte stack alignment
- ✅ `a = 10; b = 20; c = a + b; println(c)` now outputs "30" correctly

**Result**: ARM64 dynamic linking fully functional! Printf works with all args! Multiple function calls work! Variables and binary expressions work!

---

## 🚨 Current Issues

(None currently - all known ARM64 blockers resolved!)

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
- ARM64: Core features working, multiple dynamic calls working!
- Mach-O: 10/10 tests pass ✅

**Blockers**:
1. ~~macOS dynamic linking~~ - ✅ **FIXED!**
2. ~~Printf string arguments (%s)~~ - ✅ **FIXED!**
3. ~~Multiple dynamic function calls~~ - ✅ **FIXED!**
4. ~~ARM64 println() numeric arguments~~ - ✅ **FIXED!**
5. ~~ARM64 binary expressions with variables~~ - ✅ **FIXED!**
6. RISC-V backend incomplete - Medium priority
7. Missing ARM64 expression types - Low priority

**Recent Wins (Today's Session)**:
- ✅ **macOS ARM64 dynamic linking FIXED!** - No more SIGKILL!
- ✅ **String escape sequences working** - `\n`, `\t`, etc. now work
- ✅ **GOT alignment fixed** - Handles variable-sized rodata correctly
- ✅ **Printf with numeric arguments** - Variadic functions working!
- ✅ **Printf with string arguments** - Mixed string/numeric args working!
- ✅ **Multiple dynamic function calls FIXED!** - Chained fixups work!
- ✅ **Println numeric conversion FIXED!** - No more "?" output!
- ✅ **ARM64 variable storage FIXED!** - Binary expressions with variables work!
- ✅ Self-signing implementation (no external codesign needed)
- ✅ TestMachOExecutable passes
- ✅ ARM64 binaries execute successfully with dynamic linking
- ✅ Can now call `printf()` then `exit()` without crashes!
- ✅ `println(42)` now outputs "42" instead of "?"!
- ✅ `a = 10; b = 20; c = a + b; println(c)` outputs "30" correctly!
