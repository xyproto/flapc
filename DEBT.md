# Technical Debt - Flapc Compiler

**Last Updated:** 2025-11-03
**Total Items:** 47 fixable items
**Estimated Effort:** 8-12 weeks

This document tracks technical debt that can be addressed without major architectural changes. For complex design issues requiring significant refactoring or architectural decisions, see COMPLEX.md.

---

## 1. CLEANUP & CODE QUALITY (15 items)

### 1.1 Remove Excessive Debug Output
**Priority:** Medium
**Effort:** 2-3 hours
**Files:** parser.go (73 statements), arm64_codegen.go (7), macho.go (11), main.go (7)

**Issue:** Excessive DEBUG fprintf statements clutter the codebase
**Action:**
- [ ] Replace fprintf(stderr, "DEBUG:...") with proper logging package
- [ ] Add log levels (ERROR, WARN, INFO, DEBUG, TRACE)
- [ ] Make debug output controllable via flags, not just env vars
- [ ] Remove or consolidate redundant debug statements

**Files to update:**
```
parser.go:768, 1201, 1216, 3385, 3394, 3401, 3406, 3466, 5032, 5123,
5354, 5356, 5361, 5365, 5392, 5396, 5401, 5575, 5857, 5877, 5893,
5897, 5981, 6073, 6098, 6204, 6326, 6330, 6336, 6346, 6351, 6358, 6493
arm64_codegen.go:108, 113, 1296, 2811, 2816
macho.go: Multiple debug lines
main.go: Multiple debug lines
```

### 1.2 Clean Up Placeholder Comments
**Priority:** Low
**Effort:** 1-2 hours

**Action:**
- [ ] elf_complete.go:190,196 - Document why relocations are temporary
- [ ] libdef.go:252-261 - Document PLT stub placeholder purpose
- [ ] Remove or document "TEMPORARY" comments with proper explanation

### 1.3 Standardize Error Messages
**Priority:** Medium
**Effort:** 3-4 hours

**Issue:** compilerError() messages are inconsistent
**Action:**
- [ ] Create error message style guide
- [ ] Audit all compilerError() calls for clarity
- [ ] Add suggestions for common mistakes
- [ ] Include file:line information consistently

### 1.4 Remove Dead Code
**Priority:** Low
**Effort:** 2-3 hours

**Action:**
- [ ] Search for commented-out code that's no longer needed
- [ ] Remove obsolete functions/variables
- [ ] Clean up unused imports

### 1.5 Fix TODO Comments (10 addressable)
**Priority:** Medium
**Effort:** Varies (see below)

**Addressable TODOs:**
- [ ] main.go:1265 - Document hot reload limitation (15 min)
- [ ] elf_sections.go:41,251 - Document DT_DEBUG usage (10 min)
- [ ] parser.go:3871,3962,4041,4082 - Document disabled features (30 min)
- [ ] parser.go:6419 - Document pipe-based waiting (10 min)

---

## 2. TEST IMPROVEMENTS (8 items)

### 2.1 Add Missing Test Result Files
**Priority:** High
**Effort:** 1 hour

**Issue:** Some tests missing .result files
**Action:**
- [ ] Review all testprograms/*.flap
- [ ] Create .result files for any missing ones
- [ ] Verify result files match expected output format

### 2.2 Improve Test Skipping Logic
**Priority:** Low
**Effort:** 2 hours

**Issue:** Test skipping duplicated across files
**Action:**
- [ ] Create test helper: skipIfNotArch(), skipIfNotOS()
- [ ] Consolidate platform checks
- [ ] Document why tests are skipped

**Files:**
- integration_test.go:107 - ARM64 C import skip
- compiler_test.go:227 - ARM64 parallel map skip
- dynamic_test.go:279 - ldd test skip
- macho_test.go - Multiple macOS-only skips

### 2.3 Add Negative Test Suite
**Priority:** Medium
**Effort:** 4-6 hours

**Issue:** Limited testing of error conditions
**Action:**
- [ ] Create testprograms/negative/ directory
- [ ] Add tests for compilation errors
- [ ] Add tests for runtime errors
- [ ] Verify error messages are helpful

**Test cases:**
- Undefined variable detection
- Type mismatch errors
- Invalid syntax variations
- Immutable update attempts
- Use-after-move detection

### 2.4 Test Edge Cases
**Priority:** Medium
**Effort:** 3-4 hours

**Action:**
- [ ] Empty string tests
- [ ] Maximum integer tests
- [ ] Zero-length list tests
- [ ] Deeply nested expressions
- [ ] Very long identifiers

### 2.5 Add Wildcard Support to Test Runner
**Priority:** Low
**Effort:** 2-3 hours

**Issue:** Tests with dynamic values (PIDs, pointers) marked as failures
**Action:**
- [ ] Implement wildcard matching in integration_test.go
- [ ] Update .result files to use * for dynamic values
- [ ] Document wildcard syntax

**Affected tests:**
- alloc_simple_test.result
- c_getpid_test.result
- c_simple_test.result
- cstruct_helpers_test.result

### 2.6 Performance Benchmarks
**Priority:** Low
**Effort:** 4-6 hours

**Action:**
- [ ] Add benchmark tests for compilation speed
- [ ] Add benchmark tests for generated code performance
- [ ] Compare against reference implementations (C, Go)
- [ ] Track performance regressions

### 2.7 Increase Test Parallelization
**Priority:** Low
**Effort:** 2 hours

**Action:**
- [ ] Review which tests can run in parallel
- [ ] Mark tests with t.Parallel() where safe
- [ ] Reduce test suite runtime

### 2.8 Add Fuzzing Tests
**Priority:** Low
**Effort:** 1-2 days

**Action:**
- [ ] Add go-fuzz integration
- [ ] Fuzz lexer input
- [ ] Fuzz parser input
- [ ] Find crash/hang conditions

---

## 3. DOCUMENTATION (12 items)

### 3.1 Add Architecture Support Matrix
**Priority:** High
**Effort:** 2 hours

**Action:**
- [ ] Create ARCHITECTURE.md
- [ ] Document what features work on each platform
- [ ] Document known limitations per architecture
- [ ] Add testing status per platform

**Matrix to include:**
- Feature availability (x86_64, ARM64, RISC-V64)
- Test pass rates per platform
- Known bugs per platform

### 3.2 Improve Code Comments
**Priority:** Medium
**Effort:** Ongoing

**Action:**
- [ ] Add package-level documentation
- [ ] Document complex algorithms
- [ ] Explain non-obvious design choices
- [ ] Add examples in comments

**Priority files:**
- parser.go (16,927 lines, minimal comments)
- arm64_codegen.go (4,201 lines)
- elf_complete.go (complex relocation logic)

### 3.3 Create Debugging Guide
**Priority:** Medium
**Effort:** 3-4 hours

**Action:**
- [ ] Create DEBUGGING.md
- [ ] Document environment variables (DEBUG_FLAP, etc.)
- [ ] Explain debug output format
- [ ] Add troubleshooting section
- [ ] Document common errors and solutions

### 3.4 Document Known Limitations
**Priority:** High
**Effort:** 1 hour

**Action:**
- [ ] Update LANGUAGE.md with limitations section
- [ ] Document atomic operations in parallel loops limitation
- [ ] Document ARM64 parallel map operator issue
- [ ] Document macOS stack size limitation

### 3.5 Add Migration Guide
**Priority:** Low
**Effort:** 2-3 hours

**Action:**
- [ ] Create MIGRATION.md for cross-platform development
- [ ] Document platform-specific workarounds
- [ ] Provide examples of portable code
- [ ] List platform-specific features to avoid

### 3.6 Improve Function Documentation
**Priority:** Medium
**Effort:** 4-6 hours

**Action:**
- [ ] Add godoc comments to exported functions
- [ ] Document parameters and return values
- [ ] Add usage examples
- [ ] Document side effects

**Priority packages:**
- Main compiler API
- CodeGenerator interface
- AST node types

### 3.7 Create Performance Guide
**Priority:** Low
**Effort:** 2-3 hours

**Action:**
- [ ] Document optimization flags
- [ ] Explain WPO timeout setting
- [ ] List performance best practices
- [ ] Show before/after examples

### 3.8 Document Build System
**Priority:** Low
**Effort:** 1 hour

**Action:**
- [ ] Document Go version requirements
- [ ] List system dependencies (SDL3, etc.)
- [ ] Explain CI/CD setup
- [ ] Document release process

### 3.9 Add Contributing Guide
**Priority:** Low
**Effort:** 2 hours

**Action:**
- [ ] Create CONTRIBUTING.md
- [ ] Document code style
- [ ] Explain PR process
- [ ] List testing requirements

### 3.10 Document Test Programs
**Priority:** Low
**Effort:** 2-3 hours

**Action:**
- [ ] Add testprograms/README.md
- [ ] Categorize tests by feature
- [ ] Document test naming conventions
- [ ] Explain .result file format

### 3.11 Create Examples Directory
**Priority:** Low
**Effort:** 4-6 hours

**Action:**
- [ ] Create examples/ directory
- [ ] Add "Hello World" example
- [ ] Add SDL3 game example
- [ ] Add C FFI example
- [ ] Add parallel processing example

### 3.12 Document Internal Architecture
**Priority:** Medium
**Effort:** 4-6 hours

**Action:**
- [ ] Create INTERNALS.md
- [ ] Document compilation pipeline
- [ ] Explain code generation strategy
- [ ] Document register allocation
- [ ] Explain optimization passes

---

## 4. SMALL FIXES (12 items)

### 4.1 Fix String Literal Address Loading (RISC-V64)
**Priority:** High
**Effort:** 2-4 hours

**File:** riscv64_codegen.go:88
**Issue:** `return rcg.out.LoadImm("a0", 0) // TODO: Load actual address`
**Action:**
- [ ] Implement proper PC-relative addressing for RISC-V64
- [ ] Load string literal addresses correctly
- [ ] Add test to verify string operations work

### 4.2 Implement PC-Relative Load for Rodata (RISC-V64)
**Priority:** High
**Effort:** 3-4 hours

**File:** riscv64_codegen.go:158
**Issue:** `// TODO: Implement PC-relative load for rodata symbols`
**Action:**
- [ ] Implement AUIPC + ADDI for PC-relative loads
- [ ] Test with floating-point constants
- [ ] Verify rodata section is properly referenced

### 4.3 Standardize Placeholder Patching
**Priority:** Medium
**Effort:** 3-4 hours

**Issue:** Multiple files use different placeholder patching strategies
**Action:**
- [ ] Create unified placeholder system
- [ ] Document placeholder format
- [ ] Consolidate patching logic

**Files:**
- elf_complete.go:638-677
- arm64_backend.go:385
- riscv64_backend.go:397
- main.go:755-784

### 4.4 Fix Unused Variable Warnings
**Priority:** Low
**Effort:** 1 hour

**Action:**
- [ ] Run `go vet` and fix warnings
- [ ] Remove unused variables
- [ ] Use `_` for intentionally unused values

### 4.5 Improve Error Recovery
**Priority:** Medium
**Effort:** 4-6 hours

**Issue:** Parser panics on some errors instead of recovering
**Action:**
- [ ] Add error recovery in parser
- [ ] Report multiple errors per compilation
- [ ] Continue parsing after recoverable errors
- [ ] Improve error context information

### 4.6 Add Input Validation
**Priority:** Medium
**Effort:** 2-3 hours

**Action:**
- [ ] Validate compiler flags
- [ ] Check file existence before compilation
- [ ] Verify platform compatibility
- [ ] Add helpful error messages for invalid input

### 4.7 Improve Test Timeouts
**Priority:** Low
**Effort:** 1 hour

**Issue:** Some tests may hang indefinitely
**Action:**
- [ ] Add timeouts to all tests
- [ ] Use context.WithTimeout
- [ ] Make timeouts configurable

### 4.8 Fix Race Conditions in Tests
**Priority:** Medium
**Effort:** 2-3 hours

**Action:**
- [ ] Run `go test -race`
- [ ] Fix any detected race conditions
- [ ] Add race detector to CI

### 4.9 Improve Error Message for Undefined Functions
**Priority:** Medium
**Effort:** 3-4 hours

**Issue:** Undefined functions fail at link time, not compile time
**Action:**
- [ ] Add compile-time check for undefined functions
- [ ] Suggest similar function names (did you mean...?)
- [ ] Improve error message formatting

### 4.10 Add Version Information to Binaries
**Priority:** Low
**Effort:** 1-2 hours

**Action:**
- [ ] Embed compiler version in generated binaries
- [ ] Add build timestamp
- [ ] Include in --version output

### 4.11 Improve Binary Size
**Priority:** Low
**Effort:** 2-3 hours

**Action:**
- [ ] Remove unused sections
- [ ] Optimize ELF/Mach-O headers
- [ ] Strip debug info by default
- [ ] Add --strip flag

### 4.12 Add Compiler Flags
**Priority:** Low
**Effort:** 2-3 hours

**Action:**
- [ ] Add -O0, -O1, -O2, -O3 optimization levels
- [ ] Add -g for debug info
- [ ] Add -Wall for all warnings
- [ ] Document all flags

---

## Progress Tracking

### By Priority
- **High Priority:** 6 items (2-3 days)
- **Medium Priority:** 22 items (3-4 weeks)
- **Low Priority:** 19 items (2-3 weeks)

### By Category
- **Cleanup:** 15 items
- **Tests:** 8 items
- **Documentation:** 12 items
- **Fixes:** 12 items

### Completion Status
- [ ] Cleanup & Code Quality: 0/15 (0%)
- [ ] Test Improvements: 0/8 (0%)
- [ ] Documentation: 0/12 (0%)
- [ ] Small Fixes: 0/12 (0%)

**Overall Progress: 0/47 (0%)**

---

## Next Actions

### This Week
1. Fix RISC-V64 string literal loading (4.1)
2. Fix RISC-V64 PC-relative addressing (4.2)
3. Add Architecture Support Matrix (3.1)
4. Document Known Limitations (3.4)
5. Add Missing Test Result Files (2.1)

### Next Week
1. Remove excessive debug output (1.1)
2. Add negative test suite (2.3)
3. Improve debugging guide (3.3)
4. Fix undefined function errors (4.9)
5. Add input validation (4.6)

### This Month
1. Complete all High Priority items
2. Complete 50% of Medium Priority items
3. Improve test coverage
4. Update all documentation

---

**Note:** This technical debt is separate from COMPLEX.md items, which require architectural changes or design decisions.
