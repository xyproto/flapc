# Platform-Specific Issues - Flapc Compiler

**Last Updated:** 2025-11-03
**Total Items:** 28 platform-specific issues
**Primary Platform:** x86_64 Linux (Production Ready)
**Secondary Platforms:** ARM64 (Beta), RISC-V64 (Experimental)

This document tracks all platform-specific technical debt and architectural issues. For general technical debt, see DEBT.md. For complex architectural issues not related to platforms, see COMPLEX.md.

---

## Platform Status Overview

### x86_64 Linux ✅ Production Ready
- **Status:** Fully functional, production-ready
- **Test Pass Rate:** 95.5% (147/154 tests)
- **Features:** All language features working
- **Performance:** Excellent (8,000-10,000 LOC/sec compilation)
- **Binary Size:** ~13KB for simple programs
- **Known Issues:** None critical

### ARM64 (macOS/Linux) ⚠️ Beta
- **Status:** Functional for basic programs (78% working)
- **Test Pass Rate:** 78% (15/19 tested programs)
- **Main Issues:**
  - Parallel map operator crashes (segfault)
  - Stack size limitation on macOS
  - Incomplete instruction set (20 unimplemented functions)
  - C import not implemented
- **Working:** Loops, arithmetic, lambdas (non-recursive), alloc
- **Not Working:** Recursive lambdas (macOS), parallel map, unsafe ops

### RISC-V64 🚧 Experimental
- **Status:** Minimal implementation (~30% complete)
- **Test Pass Rate:** Unknown (not systematically tested)
- **Main Issues:**
  - String literals don't work (PC-relative addressing missing)
  - Incomplete instruction set (18 unimplemented functions)
  - Most features stubbed out
- **Working:** Basic arithmetic, simple loops
- **Not Working:** Strings, floating-point, SIMD, most features

---

## 1. ARM64 ISSUES (15 items)

### 1.1 ARM64 Incomplete Instruction Set ⚠️ HIGH PRIORITY
**Effort:** 4-6 weeks
**Complexity:** High
**Risk:** Medium
**Status:** 20 unimplemented functions

**Missing Instructions:**

**Memory Operations (Critical):**
```go
// arm64_backend.go
Line 97:  MovMemToReg not implemented
Line 101: MovRegToMem not implemented
Line 468: LeaSymbolToReg not implemented
```
**Impact:** Cannot load/store from memory addresses, cannot get symbol addresses

**Floating-Point Operations (Critical):**
```go
Lines 494-534: All XMM/SSE operations not implemented:
- MovXmmToMem / MovMemToXmm
- MovRegToXmm / MovXmmToReg
- Cvtsi2sd / Cvttsd2si (conversions)
- AddpdXmm / SubpdXmm / MulpdXmm / DivpdXmm (arithmetic)
- Ucomisd (comparisons)
```
**Impact:** Floating-point operations unavailable, numeric code doesn't work

**Other Operations:**
```go
Line 268: XorRegWithImm not implemented
Line 327: PushReg not implemented (ARM64 doesn't have PUSH/POP)
Line 331: PopReg not implemented
```
**Impact:** Some bitwise ops unavailable, stack operations different

**Implementation Plan:**

**Phase 1: Memory Operations (1 week)**
- [ ] Implement LDR/STR for MovMemToReg/MovRegToMem
- [ ] Implement ADR/ADRP for LeaSymbolToReg
- [ ] Test with memory access patterns
- [ ] Verify symbol addressing works

**Phase 2: Floating-Point (2 weeks)**
- [ ] Implement FMOV for float moves (MovXmm ops)
- [ ] Implement FCVT for conversions (Cvtsi2sd, Cvttsd2si)
- [ ] Implement FADD/FSUB/FMUL/FDIV for arithmetic
- [ ] Implement FCMP for comparisons (Ucomisd)
- [ ] Test with floating-point intensive code

**Phase 3: Remaining Operations (1 week)**
- [ ] Implement EOR for XorRegWithImm
- [ ] Document PUSH/POP unavailability (use STR/LDR instead)
- [ ] Add tests for all new instructions

**Phase 4: Validation (1 week)**
- [ ] Run all test programs on ARM64
- [ ] Fix discovered issues
- [ ] Update ARM64_STATUS.md
- [ ] Benchmark performance

### 1.2 ARM64 Parallel Map Operator Crash 🔥 HIGH PRIORITY
**Effort:** 2-3 weeks
**Complexity:** Very High
**Risk:** High
**Status:** Segfaults at arm64_codegen.go:1444

**Symptoms:**
```flap
numbers := [1, 2, 3]
doubled := numbers || x => x * 2  // Segfaults on ARM64
```

**Current State:**
- x86_64 implementation works perfectly
- ARM64 crashes in compileParallelExpr
- Test skipped: compiler_test.go:227
- Blocks production use on Apple Silicon

**Analysis Needed:**
1. Compare x86_64 vs ARM64 parallel codegen side-by-side
2. Debug crash with GDB/LLDB on ARM64
3. Check thread spawning code for ARM64-specific issues
4. Verify closure environment handling
5. Check stack alignment requirements

**Suspected Issues:**
- Incorrect stack frame setup for threads
- Register corruption in thread context
- Closure environment not properly passed to threads
- Alignment issues (ARM64 requires 16-byte stack alignment)
- Race condition in parallel execution setup

**Implementation Plan:**

**Phase 1: Diagnosis (3-5 days)**
- [ ] Set up ARM64 debugging environment (real hardware or VM)
- [ ] Create minimal failing test case
- [ ] Use LLDB to find exact crash location
- [ ] Examine register state at crash
- [ ] Compare with working x86_64 assembly output

**Phase 2: Fix (1-2 weeks)**
- [ ] Implement fix based on diagnosis
- [ ] Test with simple parallel map first
- [ ] Gradually increase complexity
- [ ] Verify barrier synchronization works
- [ ] Test closure capture works correctly

**Phase 3: Validation (2-3 days)**
- [ ] Enable TestParallelSimpleCompiles on ARM64
- [ ] Run all parallel tests 100 times
- [ ] Test on real ARM64 hardware (Mac or Linux)
- [ ] Benchmark performance vs x86_64
- [ ] Update documentation

### 1.3 ARM64 C Import Not Implemented ⚠️ MEDIUM PRIORITY
**Effort:** 1-2 weeks
**Complexity:** Medium
**Risk:** Medium
**Status:** integration_test.go:107

**Current State:**
- C FFI unavailable on ARM64
- Tests skip C import features
- SDL3, OpenGL integration impossible
- Limits practical use of ARM64 backend

**Impact:**
- Cannot use C libraries on ARM64
- Game development impossible
- System programming limited
- Major feature gap vs x86_64

**Implementation Plan:**

**Phase 1: Basic C Import (1 week)**
- [ ] Implement C function signature parsing for ARM64
- [ ] Implement ARM64 calling convention (AAPCS64)
- [ ] Handle argument passing (x0-x7, d0-d7)
- [ ] Handle return values
- [ ] Test with simple C functions (malloc, free, printf)

**Phase 2: Advanced Features (1 week)**
- [ ] Implement structure passing by value
- [ ] Handle variable argument functions (va_list)
- [ ] Implement callback support
- [ ] Test with SDL3, other complex libraries
- [ ] Update tests to run on ARM64

### 1.4 ARM64 macOS Stack Size Limitation ⚠️ LOW PRIORITY (OS ISSUE)
**Effort:** Unknown (may be impossible to fix)
**Complexity:** Very High
**Risk:** Very High
**Status:** macOS dyld limitation

**Current State:**
- LC_MAIN specifies 8MB stack in Mach-O
- macOS dyld provides only ~5.6KB stack
- Recursive lambdas overflow stack immediately
- Documented in macho_test.go:436, TODO.md:50

**Impact:**
- Deep recursion fails on macOS ARM64
- Tail-call optimization becomes critical
- Some algorithms impractical
- Intel macOS may have same issue

**Root Cause:**
- macOS dyld doesn't honor stacksize field
- Apple bug or intentional security limitation
- No documented workaround from Apple

**Possible Solutions:**

**Option A: Accept as OS Limitation (RECOMMENDED)**
- Document clearly in LANGUAGE.md
- Provide iterative alternatives to recursion
- Emphasize tail-call optimization (which works)
- Note that Linux ARM64 doesn't have this issue

**Option B: Custom Loader (NOT RECOMMENDED)**
- Write custom dyld replacement
- Extremely complex, security issues
- Apple may block in future updates
- High maintenance burden

**Option C: Runtime Stack Switching (COMPLEX)**
- Detect stack overflow at runtime
- Switch to heap-allocated stack
- Very complex, performance impact

**Recommendation:** Accept as documented limitation
- Not a compiler bug
- Workarounds too risky/complex
- Tail recursion works fine
- Linux ARM64 unaffected

**Action Items:**
- [x] Document in macho_test.go
- [ ] Document in LANGUAGE.md limitations section
- [ ] Add to README.md known issues
- [ ] Provide examples of tail-recursive patterns

### 1.5 ARM64 Additional Instruction TODOs 📝 LOW PRIORITY
**Effort:** 2-3 weeks
**Complexity:** Medium
**Status:** arm64_instructions.go:434-439

**Missing Instruction Categories:**
```go
// TODO: Add more floating-point instructions (FADD, FSUB, FMUL, FDIV, FCVT, etc.)
// TODO: Add SIMD/NEON instructions
// TODO: Add load/store pair instructions (STP, LDP)
// TODO: Add more arithmetic instructions (MUL, UDIV, SDIV, etc.)
// TODO: Add logical instructions (AND, OR, EOR, etc.)
// TODO: Add shift instructions (LSL, LSR, ASR, ROR)
```

**Implementation Plan:**
- [ ] Prioritize based on feature usage
- [ ] Implement in order: arithmetic → logical → shifts → SIMD
- [ ] Test each category thoroughly
- [ ] Update instruction reference documentation

### 1.6 ARM64 Platform-Specific Test Skips 📋 MEDIUM PRIORITY
**Effort:** 2-3 hours
**Complexity:** Low
**Files:** Multiple test files

**Currently Skipped:**
- integration_test.go:107 - C import tests
- compiler_test.go:227 - Parallel map tests
- macho_test.go - 11 tests (macOS-only)

**Action Items:**
- [ ] Track which tests are skipped and why
- [ ] Re-enable as features are implemented
- [ ] Add ARM64-specific test variants where needed
- [ ] Update test documentation

---

## 2. RISC-V64 ISSUES (10 items)

### 2.1 RISC-V64 String Literal Loading 🔥 HIGH PRIORITY
**Effort:** 2-4 hours
**Complexity:** Medium
**Risk:** Low
**Status:** riscv64_codegen.go:88

**Current Code:**
```go
case *StringExpr:
    label := fmt.Sprintf("str_%d", len(rcg.eb.consts))
    rcg.eb.Define(label, e.Value+"\x00")
    return rcg.out.LoadImm("a0", 0) // TODO: Load actual address
```

**Problem:** Returns 0 instead of string address
**Impact:** String operations completely broken on RISC-V64

**Solution:** Implement PC-relative addressing with AUIPC + ADDI

**Implementation Plan:**
```go
// 1. Get label offset (will be filled by relocation)
labelOffset := 0 // Placeholder, patched later

// 2. AUIPC rd, imm20 - Add upper immediate to PC
// Loads upper 20 bits of PC-relative offset
auipc := encodeUType(0x17, getReg("a0"), labelOffset >> 12)
rcg.out.encodeInstr(auipc)

// 3. ADDI rd, rs1, imm12 - Add immediate
// Adds lower 12 bits
addi := encodeIType(0x13, 0x0, getReg("a0"), getReg("a0"), labelOffset & 0xFFF)
rcg.out.encodeInstr(addi)

// 4. Record relocation for patching
rcg.eb.AddRelocation(labelName, currentPosition, RelocTypeRISCVPCRel)
```

**Action Items:**
- [ ] Implement AUIPC instruction encoding (U-type)
- [ ] Implement proper relocation for PC-relative addresses
- [ ] Test with string literals
- [ ] Test with multiple strings
- [ ] Verify rodata section addressing

### 2.2 RISC-V64 PC-Relative Load for Rodata 🔥 HIGH PRIORITY
**Effort:** 3-4 hours
**Complexity:** Medium
**Risk:** Low
**Status:** riscv64_codegen.go:158

**Current Code:**
```go
// Load string address into a1
// TODO: Implement PC-relative load for rodata symbols
if err := rcg.out.LoadImm("a1", 0); err != nil {
    return err
}
```

**Problem:** Cannot load addresses from rodata section
**Impact:** Constants, floating-point values, strings all broken

**Solution:** Same as 2.1 - AUIPC + ADDI pattern

**Implementation Plan:**
- [ ] Create helper function: LoadSymbolAddress(reg, symbol)
- [ ] Use AUIPC + ADDI pattern
- [ ] Handle relocations correctly
- [ ] Test with floating-point constants
- [ ] Test with string constants

### 2.3 RISC-V64 Incomplete Instruction Set ⚠️ HIGH PRIORITY
**Effort:** 6-8 weeks
**Complexity:** High
**Risk:** Medium
**Status:** 18 unimplemented functions

**Missing Instructions:**

**Memory Operations:**
```go
// riscv64_backend.go
Line 100: MovMemToReg not implemented
Line 104: MovRegToMem not implemented
Line 499: LeaSymbolToReg not implemented
```

**Stack Operations:**
```go
Line 324: PushReg not implemented
Line 328: PopReg not implemented
```

**Floating-Point Operations (All):**
```go
Lines 526-566: All XMM/SSE operations not implemented
- MovXmmToMem / MovMemToXmm
- MovRegToXmm / MovXmmToReg
- Cvtsi2sd / Cvttsd2si
- AddpdXmm / SubpdXmm / MulpdXmm / DivpdXmm
- Ucomisd
```

**Implementation Plan:**

**Phase 1: Memory Operations (1 week)**
- [ ] Implement LD/SD for MovMemToReg/MovRegToMem
- [ ] Implement LA (load address) pseudo-instruction
- [ ] Use AUIPC + ADDI for LeaSymbolToReg
- [ ] Test memory operations thoroughly

**Phase 2: Floating-Point (2-3 weeks)**
- [ ] Implement FLD/FSD for float loads/stores
- [ ] Implement FADD.D/FSUB.D/FMUL.D/FDIV.D
- [ ] Implement FCVT.* for conversions
- [ ] Implement FEQ/FLT/FLE for comparisons
- [ ] Test floating-point arithmetic

**Phase 3: Extensions (1-2 weeks)**
- [ ] Implement Multiply/Divide (M extension)
- [ ] Implement basic atomic ops (A extension)
- [ ] Implement compressed instructions (C extension - optional)
- [ ] Test all extensions

**Phase 4: Validation (1 week)**
- [ ] Run all test programs
- [ ] Fix discovered issues
- [ ] Update documentation
- [ ] Benchmark performance

### 2.4 RISC-V64 Additional Instruction TODOs 📝 MEDIUM PRIORITY
**Effort:** 3-4 weeks
**Complexity:** Medium
**Status:** riscv64_instructions.go:380-385

**Missing Instruction Categories:**
```go
// TODO: Add floating-point instructions (FADD.D, FSUB.D, FMUL.D, FDIV.D, FCVT, etc.)
// TODO: Add multiply/divide instructions (MUL, MULH, DIV, REM, etc.)
// TODO: Add logical instructions (AND, OR, XOR, etc.)
// TODO: Add shift instructions (SLL, SRL, SRA, etc.)
// TODO: Add atomic instructions (LR, SC, AMO*, etc.)
// TODO: Add CSR instructions
```

**Priority Order:**
1. Multiply/Divide (M extension) - Common operations
2. Logical instructions (AND, OR, XOR)
3. Shift instructions (SLL, SRL, SRA)
4. Floating-point (already covered in 2.3)
5. Atomic instructions (A extension)
6. CSR instructions (system programming)

**Implementation Plan:**
- [ ] Implement M extension first (1 week)
- [ ] Implement logical ops (3-4 days)
- [ ] Implement shift ops (3-4 days)
- [ ] Implement A extension (1 week)
- [ ] CSR ops (optional, 3-4 days)

### 2.5 RISC-V64 Test Coverage 📋 LOW PRIORITY
**Effort:** 1-2 weeks
**Complexity:** Low
**Status:** No systematic testing

**Current State:**
- No dedicated RISC-V64 tests
- Unknown test pass rate
- No validation of generated code
- No performance benchmarks

**Action Items:**
- [ ] Create testprograms/riscv64/ directory
- [ ] Add basic functionality tests
- [ ] Set up RISC-V64 test environment (QEMU)
- [ ] Run all test programs and track results
- [ ] Create RISCV64_STATUS.md

---

## 3. PLATFORM ABSTRACTIONS (3 items)

### 3.1 Platform-Specific Code Duplication 📝 LOW PRIORITY
**Effort:** 1-2 weeks
**Complexity:** Medium
**Risk:** Low

**Duplicated Files:**
- parallel_unix.go vs parallel_other.go
- filewatcher_unix.go vs filewatcher_other.go
- hotreload_unix.go vs hotreload_other.go
- parallel_test_unix.go vs parallel_test_other.go

**Issues:**
- Changes must be made to multiple files
- Easy to miss platform-specific bugs
- Testing harder (need multiple OSes)
- Code maintenance overhead

**Proposed Solution:**
```go
platform/
  ├── interface.go      // Common interface
  ├── unix.go          // Unix implementation
  ├── windows.go       // Windows implementation (future)
  └── darwin.go        // macOS-specific (if needed)
```

**Action Items:**
- [ ] Extract common interfaces
- [ ] Refactor Unix-specific code
- [ ] Add build tags consistently
- [ ] Test on multiple platforms
- [ ] Document platform requirements

### 3.2 Dynamic Linking Platform Issues 📝 MEDIUM PRIORITY
**Effort:** 3-4 weeks
**Complexity:** High
**Risk:** Medium

**Current Issues:**
- dynamic_test.go:279 - ldd test skipped
- elf_test.go:444 - WriteCompleteDynamicELF incomplete
- dynamic_test.go:87 - No symbol section warning

**Platform Differences:**
- Linux: ELF with PLT/GOT
- macOS: Mach-O with dyld
- FreeBSD: ELF with different conventions

**Action Items:**
- [ ] Complete ELF dynamic linking for Linux
- [ ] Test Mach-O dynamic linking on macOS
- [ ] Add FreeBSD support
- [ ] Enable skipped tests
- [ ] Document platform differences

### 3.3 Non-Unix Platform Support 📝 LOW PRIORITY (FUTURE)
**Effort:** 8-12 weeks
**Complexity:** Very High
**Risk:** High

**Current State:**
- Unix-only features (futex, fork, etc.)
- Windows completely unsupported
- Would require significant work

**Windows Support Would Require:**
- [ ] Windows PE/COFF binary format
- [ ] Windows calling convention
- [ ] Windows threading (CreateThread)
- [ ] Windows system calls
- [ ] Windows file watching
- [ ] Windows-specific tests

**Recommendation:** Defer to v3.0+
- Focus on Linux/macOS/FreeBSD first
- Windows is very different
- Large effort for potentially small user base

---

## 4. CROSS-PLATFORM TESTING (2 items)

### 4.1 Multi-Platform CI/CD 📋 MEDIUM PRIORITY
**Effort:** 1-2 weeks
**Complexity:** Medium
**Risk:** Low

**Current State:**
- CI runs on Linux x86_64 only
- No ARM64 testing in CI
- No RISC-V64 testing
- No macOS testing

**Proposed CI Matrix:**
```yaml
matrix:
  os: [ubuntu-latest, macos-latest]
  arch: [amd64, arm64]
  go-version: [1.21, 1.22, 1.23]
```

**Action Items:**
- [ ] Add macOS to CI matrix
- [ ] Add ARM64 runners (GitHub or self-hosted)
- [ ] Add RISC-V64 emulation (QEMU)
- [ ] Track test results per platform
- [ ] Report platform-specific failures

### 4.2 Platform-Specific Performance Benchmarks 📋 LOW PRIORITY
**Effort:** 1-2 weeks
**Complexity:** Low
**Risk:** Low

**Current State:**
- No performance tracking
- No platform comparisons
- Unknown if optimizations work across platforms

**Action Items:**
- [ ] Create benchmark suite
- [ ] Run on x86_64, ARM64, RISC-V64
- [ ] Compare compilation speed
- [ ] Compare generated code performance
- [ ] Track regressions per platform
- [ ] Document performance characteristics

---

## Priority Summary

### IMMEDIATE (This Week)
1. **RISC-V64 String Literals** (2-4 hours) - Blocks basic functionality
2. **RISC-V64 PC-Relative** (3-4 hours) - Blocks constants
3. **Document ARM64 Limitations** (1 hour) - User clarity

### HIGH PRIORITY (Next Month)
1. **ARM64 Instruction Set** (4 weeks) - Complete core functionality
2. **ARM64 Parallel Map** (2-3 weeks) - Major feature
3. **ARM64 C Import** (1-2 weeks) - FFI support
4. **RISC-V64 Instruction Set** (6-8 weeks) - Complete core functionality

### MEDIUM PRIORITY (Next Quarter)
1. **Dynamic Linking** (3-4 weeks) - Library support
2. **Multi-Platform CI** (1-2 weeks) - Testing infrastructure
3. **Platform Code Refactoring** (1-2 weeks) - Maintainability

### LOW PRIORITY (Future)
1. **Additional Instructions** (4-6 weeks) - Nice-to-have
2. **Performance Benchmarks** (1-2 weeks) - Optimization guidance
3. **Windows Support** (8-12 weeks) - New platform
4. **Accept macOS Stack Limitation** (documentation only)

---

## Platform Support Goals

### v1.7.4 (Current)
- ✅ x86_64 Linux: Production ready
- ⚠️ ARM64: Beta (document limitations)
- 🚧 RISC-V64: Experimental (document as incomplete)

### v2.0 (Q3 2025)
- ✅ x86_64 Linux: Stable
- ✅ ARM64 Linux/macOS: Production ready (all features)
- ⚠️ RISC-V64: Beta (basic features working)

### v3.0 (Q4 2025+)
- ✅ x86_64 Linux: Stable
- ✅ ARM64: Stable
- ✅ RISC-V64: Production ready
- ⚠️ Windows x86_64: Beta (if demand exists)

---

## Progress Tracking

### ARM64 Completion
- [ ] Instruction Set: 0/20 functions (0%)
- [ ] Parallel Map Fix: Not started
- [ ] C Import: Not started
- [ ] Test Coverage: 78% (15/19 tested)
- **Overall: ~65% complete**

### RISC-V64 Completion
- [ ] String Literals: Not started (CRITICAL)
- [ ] PC-Relative: Not started (CRITICAL)
- [ ] Instruction Set: 0/18 functions (0%)
- [ ] Test Coverage: Unknown
- **Overall: ~30% complete**

### Cross-Platform Support
- [x] x86_64 Linux: 100%
- [ ] x86_64 macOS: ~95% (stack issue)
- [ ] ARM64 Linux: ~70%
- [ ] ARM64 macOS: ~65% (stack + parallel)
- [ ] RISC-V64 Linux: ~30%

---

**Note:** This document consolidates all platform-specific issues from DEBT.md and COMPLEX.md. General technical debt remains in DEBT.md, complex architectural issues in COMPLEX.md.
