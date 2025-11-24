# TODO for Flap 2.0.0 Release
**Last Updated:** 2025-11-24  
**Sorted from foundational to higher-level**

## PHILOSOPHY: Bottom-Up Excellence

We're working **bottom-up** from the most foundational issues to higher-level concerns:
1. Make GRAMMAR.md and LANGUAGESPEC.md excellent and ready for 2.0.0
2. Ensure lexer and parser match the grammar exactly
3. Fix the three most problematic bugs: tail/_, printf, variadic arguments
4. Then proceed to higher-level features and documentation

## CRITICAL: Documentation Foundation (FIRST)

### 1. Review and Finalize GRAMMAR.md ⭐ PRIORITY #1
**Status:** Mostly complete, needs final review  
**Estimated:** 30-45 minutes

**Checks:**
- [x] Variadic function syntax documented (line 300-309, 363-362)
- [x] Lambda syntax clear (line 296-362 with comprehensive explanation)
- [x] Match block forms (value vs guard) documented (line 60-104)
- [x] All operators documented with precedence (line 560-716)
- [x] Memory management philosophy (arena over malloc) stated (line 518-557)
- [x] Register allocator/tracker mentioned (line 1626-1628, 1632-1645)
- [ ] Final consistency check for arrow operators (-> vs =>)
- [ ] Verify all examples use correct syntax
- [ ] Ensure no contradictions or ambiguities

**Critical Sections to Review:**
- Lambda syntax rules (lines 312-362) - Is this clear enough?
- Block disambiguation (lines 40-124) - Any edge cases missing?
- Variadic syntax (lines 300-309) - Complete coverage?
- Assignment operators (lines 607-655) - All cases covered?

### 2. Review and Finalize LANGUAGESPEC.md ⭐ PRIORITY #2
**Status:** Excellent content, needs consistency check with GRAMMAR.md  
**Estimated:** 30-45 minutes

**Checks:**
- [x] Variadic functions documented (lines 702-767)
- [x] Error handling with or! documented (lines 1917-2241)
- [x] defer statement documented (lines 1632-1976)
- [x] All built-in functions listed (lines 1810-1872)
- [x] Arena allocator usage clear (lines 1591-1636)
- [ ] Verify all code examples use correct syntax (especially -> vs =>)
- [ ] Ensure no contradictions with GRAMMAR.md
- [ ] Check match block examples use correct forms
- [ ] Verify lambda examples follow grammar rules

**Critical Consistency Checks:**
- Arrow operators: Ensure -> for lambdas, => for match arms throughout
- Match syntax: Ensure guard form uses `|` at line start consistently
- Assignment: Ensure functions defined with `=` not `:=` consistently
- Builtin philosophy: Ensure minimal builtin principle stated clearly

## CRITICAL: The Three Most Problematic Bugs

### 3. Fix Tail (_) Operator ⭐ PRIORITY #3
**Status:** DEFERRED - Produces garbage values  
**Priority:** HIGH  
**Estimated:** 3-4 hours

**Problem:** The tail operator doesn't correctly copy and re-index elements.

**What works:** Head (^) operator  
**What doesn't:** Tail (_) operator for lists and maps

**Action Plan:**
1. Review current tail implementation in codegen (search for "tail" or "_" operator)
2. Understand how head (^) works correctly
3. Fix element copying algorithm for tail
4. Fix index re-mapping (reduce all indices by 1)
5. Add comprehensive tests: `_[1,2,3]` → `[2,3]`, `_[5]` → `[]`, `_[]` → `[]`
6. Test with maps as well
7. Commit and push

### 4. Fix printf in Complex Contexts ⭐ PRIORITY #4
**Status:** Works in simple cases, fails with recursion + loops  
**Priority:** HIGH  
**Estimated:** 2-3 hours

**Known Issues:**
- printf in recursive function + loop causes problems (factorial example)
- May be related to register allocation or stack frame management

**Action Plan:**
1. Create minimal test case that reproduces the issue
2. Debug with --verbose flag
3. Check register allocator state during printf calls
4. Verify stack alignment (16-byte) before C FFI calls
5. Check if xmm registers properly saved/restored
6. Add tests for printf in various contexts (loops, recursion, nested calls)
7. Commit and push

### 5. Variadic Functions - Complete Implementation ⭐ PRIORITY #5
**Status:** Infrastructure complete, list building has segfault bug  
**Priority:** CRITICAL  
**Estimated:** 4-6 hours

- [x] Grammar and lexer support for `...` syntax
- [x] Parser handles variadic parameters
- [x] Function signature tracking 
- [x] r14 register convention for arg count
- [x] Call site passes arg count in r14
- [ ] **FIX: Variadic list construction segfault**
  - Issue: Stack manipulation during function entry causes crash
  - Solution: Save xmm args first, build list after frame stable
  - Must use arena allocation, not malloc
- [ ] Implement spread operator `func(list...)` at call sites
- [ ] Comprehensive tests (0 args, 1 arg, many args, mixed)

**Action Plan:**
1. Study current implementation in codegen.go (search for "variadic")
2. Identify exact location of segfault (likely in generateLambdaFunctions)
3. Fix list construction to use arena allocator properly
4. Save all xmm registers to temp space immediately on entry
5. Build list from saved values after frame is stable
6. Test with simple sum function: `sum = (first, rest...) -> first + #rest`
7. Add spread operator support at call sites
8. Create variadic stdlib functions (printf, eprintf, exitf) once working
9. Comprehensive tests
10. Commit and push



## HIGHER-LEVEL: Standard Library & Core Functions

### 6. Core I/O Functions - Verify All Work
**Priority:** MEDIUM (after bugs fixed)  
**Estimated:** 1-2 hours

**Check each:**
- [x] println() - Works
- [x] print() - Works  
- [x] printf() - Works in simple cases, issues in complex contexts (fixing in #4)
- [x] eprintln() - Works
- [x] eprint() - Works
- [x] eprintf() - Works  
- [x] exitln() - Works
- [x] exitf() - Works (fixed to use syscall on stderr)
- [ ] Test all functions after printf fix
- [ ] Document limitations if any remain

### 7. Standard Library (stdlib.flap)
**Status:** Reference docs exist, not auto-included yet  
**Priority:** MEDIUM (after variadic functions work)  
**Estimated:** 4-5 hours

**Needed (requires variadic functions working first):**
- [ ] Complete variadic printf implementation in Flap
- [ ] Complete variadic eprintf implementation in Flap  
- [ ] Complete variadic exitf implementation in Flap
- [ ] Implement auto-inclusion when stdlib functions are used
- [ ] Add common string utilities (length, concat, substring)
- [ ] Add common list utilities (map, filter, reduce, sum)
- [ ] Document all stdlib functions

**Philosophy:** Keep builtins MINIMAL. Use:
1. Operators (^, _, #, etc.) for common operations
2. C FFI (c.malloc, c.sin) for system functions
3. stdlib.flap for convenience functions in Flap code

## DEFERRED: Windows, Optimizations, Advanced Features

### 8. Windows + SDL3 Support  
**Status:** Basic PE works, SDL3 fails under Wine  
**Priority:** DEFERRED (not blocking 2.0.0)  
**Estimated:** 8-10 hours (requires actual Windows testing)

**Current State:**
- ✅ PE generation works
- ✅ Basic programs run under Wine
- ✅ printf works with Windows calling convention
- ✅ C FFI works (malloc, math functions, etc.)
- ✅ exitf() fixed for Windows
- ❌ SDL3 window creation fails under Wine (Wine limitation, not flapc bug)

**Deferred because:**
- Wine's DirectX/DXGI support incomplete
- Need actual Windows machine for proper testing  
- Core compiler functionality works on Windows
- Can revisit after 2.0.0 release with Windows testing

### 9. Performance Optimizations (Post-Release)
**Priority:** DEFERRED  
**Estimated:** 8-12 hours

- [ ] Whole program optimization
- [ ] More aggressive constant folding
- [ ] Dead code elimination improvements
- [ ] Better register allocation (reduce spills)
- [ ] Reduce binary size (strip unused strings)
- [ ] SIMD optimization improvements

### 10. Advanced Language Features (Post-Release)
**Priority:** DEFERRED  
**Estimated:** 16-20 hours

- [ ] Pattern destructuring in match clauses
- [ ] Full tail call optimization for mutual recursion
- [ ] Import from git repositories
- [ ] Module system
- [ ] Type conversions return Result type (`42 as uint32 or! {...}`)
- [ ] Local variables in lambda bodies (`f = x -> { y := x + 1; y }`)

## INFRASTRUCTURE: Testing & CI

### 11. Testing Infrastructure
**Priority:** MEDIUM (after core bugs fixed)  
**Estimated:** 3-4 hours

- [x] SDL3 headless tests (using dummy video driver)
- [x] Existing tests pass
- [ ] GitHub Actions CI setup
- [ ] Test coverage for tail operator (after #3)
- [ ] Test coverage for printf in complex contexts (after #4)
- [ ] Test coverage for variadic functions (after #5)
- [ ] Benchmark suite
- [ ] Memory leak detection tests

### 12. Final Documentation Polish
**Priority:** HIGH (before release)  
**Estimated:** 2-3 hours

- [ ] README.md updated for 2.0.0
- [ ] CHANGELOG.md created with all changes since 1.5.0
- [ ] INSTALL.md updated
- [ ] All examples verified to work
- [ ] Tutorial reviewed
- [ ] API reference reviewed
- [ ] Best practices guide reviewed

---

## COMPLETED ITEMS (Foundation for 2.0.0)

✅ Windows x64 C FFI with proper calling convention  
✅ PE import table generation  
✅ exitf() implementation fixed  
✅ Variadic function infrastructure (grammar, lexer, parser, r14 convention)  
✅ head() and tail() removed as builtins (use ^ and _ operators)  
✅ Minimal builtin philosophy enforced  
✅ Arena allocator documented as preferred over malloc  
✅ Register allocator and register tracker implemented  
✅ Error handling with or! operator  
✅ defer statement for resource cleanup  
✅ Result type with NaN error encoding  
✅ GRAMMAR.md excellent and comprehensive  
✅ LANGUAGESPEC.md excellent and comprehensive  
✅ SDL3 examples with defer and railway-oriented error handling  

---

## 2.0.0 RELEASE CHECKLIST (Bottom-Up Order)

**Phase 1: Documentation Foundation (FIRST)**
- [ ] 1. Final review of GRAMMAR.md for consistency
- [ ] 2. Final review of LANGUAGESPEC.md for consistency
- [ ] 3. Ensure no contradictions between docs

**Phase 2: Fix The Three Most Problematic Bugs**
- [ ] 3. Fix tail (_) operator completely
- [ ] 4. Fix printf in complex contexts (recursion + loops)
- [ ] 5. Fix variadic function list construction segfault
- [ ] 6. Implement spread operator for variadic calls

**Phase 3: Testing & Validation**
- [ ] 7. All existing tests pass
- [ ] 8. Test tail operator thoroughly
- [ ] 9. Test printf in complex contexts
- [ ] 10. Test variadic functions (0 args, many args, spread)

**Phase 4: Polish & Release**
- [ ] 11. Test all core I/O functions
- [ ] 12. Create CHANGELOG.md
- [ ] 13. Update README.md for 2.0.0
- [ ] 14. Update version to 2.0.0
- [ ] 15. Git tag and release

**Nice to Have (Can defer to 2.0.1):**
- [ ] Complete stdlib.flap with auto-inclusion
- [ ] GitHub Actions CI
- [ ] Windows + SDL3 full support (needs actual Windows machine)

**Can Wait (Post-Release):**
- [ ] Performance optimizations
- [ ] Advanced language features  
- [ ] Module system

---

## EXECUTION PLAN (Next Steps)

**Now (Session Goal):**
1. Review GRAMMAR.md thoroughly (30 min)
2. Review LANGUAGESPEC.md thoroughly (30 min)
3. Fix tail operator (3-4 hours)
4. Commit and push progress

**Next Session:**
1. Fix printf in complex contexts (2-3 hours)
2. Fix variadic functions (4-6 hours)
3. Run all tests
4. Commit and push

**Final Session:**
1. Create CHANGELOG.md
2. Update documentation
3. Tag release
4. Celebrate! 🎉

