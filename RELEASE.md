# Flapc 2.0.0 Release Plan

**Target Date:** December 2024  
**Status:** In Progress  
**Focus:** Production-ready compiler with complete feature set

## Release Goals

This release focuses on making Flapc production-ready with:
1. Complete variadic function support
2. Comprehensive standard library
3. Better error messages
4. Performance optimizations
5. Complete documentation
6. More examples

## Must-Have Features (Blocking Release)

### 1. Complete Variadic Functions ⚠️ IN PROGRESS
- [x] Grammar and lexer support
- [x] Parser implementation
- [x] Function signature tracking
- [x] r14 calling convention
- [x] xmm register saving on entry
- [ ] **Build list from saved xmm values**
- [ ] **Implement spread operator `func(list...)`**
- [ ] Comprehensive tests

**Status:** Infrastructure complete, needs argument collection
**Priority:** HIGH
**Estimated:** 3-4 hours

### 2. Standard Library (stdlib.flap)
- [x] Create stdlib.flap with common functions
- [x] String manipulation functions (documented)
- [x] List utility functions (documented)
- [x] Math utility functions (documented)
- [ ] Auto-include when functions are used (not yet implemented)
- [x] Documentation for all stdlib functions

**Status:** Reference documentation complete
**Priority:** MEDIUM (core functions are builtins)
**Note:** Most functions require local mutable variables in lambdas (not yet supported)

### 3. Error Messages & Diagnostics
- [ ] Better parser error messages
- [ ] Show code context in errors
- [ ] Suggest fixes for common mistakes
- [ ] Type mismatch diagnostics
- [ ] Undefined variable suggestions
- [ ] Line/column numbers for all errors

**Status:** Basic errors work, needs improvement
**Priority:** MEDIUM
**Estimated:** 3-4 hours

### 4. Documentation
- [ ] Complete language tutorial (in progress)
- [x] GRAMMAR.md (complete)
- [x] LANGUAGESPEC.md (complete)
- [x] Code examples (7 comprehensive examples added)
- [ ] API reference
- [ ] Best practices guide
- [ ] Performance tuning guide
- [ ] Windows development guide

**Status:** Foundation excellent, examples added
**Priority:** MEDIUM
**Estimated:** 2-3 hours remaining

## Nice-to-Have Features (Optional)

### 5. Performance Optimizations
- [ ] Whole program optimization
- [ ] More aggressive dead code elimination
- [ ] Constant propagation improvements
- [ ] Better register allocation
- [ ] Reduce binary size

**Status:** Good baseline, optimization opportunities exist
**Priority:** LOW
**Estimated:** 6-8 hours

### 6. Additional Language Features
- [ ] Pattern destructuring in match clauses
- [ ] Full tail call optimization
- [ ] Import from git repos
- [ ] Result type for conversions

**Status:** Core language complete
**Priority:** LOW
**Estimated:** 8-10 hours

### 7. Testing & CI
- [ ] GitHub Actions CI setup
- [ ] Automated cross-platform testing
- [ ] Benchmark suite
- [ ] Fuzzing tests
- [ ] Code coverage reporting

**Status:** Manual testing works well
**Priority:** LOW
**Estimated:** 4-6 hours

## Known Issues to Fix

### Critical
- [ ] Complete variadic argument collection (infrastructure solid, list building TODO)
- [ ] Fix map operation register corruption (squared = list | func corrupted)
- [ ] Fix recursive function + printf in loop issue (factorial example)

### Minor
- [ ] Local variables in lambda bodies (workaround exists)
- [ ] SDL3 on Wine/Wayland (Wine limitation)

## Testing Checklist

- [ ] All existing tests pass
- [ ] Variadic function tests
- [ ] Windows C FFI tests
- [ ] Cross-platform compilation tests
- [ ] Example programs work
- [ ] Performance benchmarks
- [ ] Memory leak checks

## Documentation Checklist

- [ ] README.md updated
- [ ] CHANGELOG.md created
- [ ] GRAMMAR.md complete
- [ ] LANGUAGESPEC.md reviewed
- [ ] Tutorial written
- [ ] Examples documented
- [ ] Windows guide complete

## Release Process

1. **Complete must-have features**
2. **Run full test suite**
3. **Update all documentation**
4. **Create CHANGELOG.md**
5. **Version bump to 2.0.0**
6. **Tag release in git**
7. **Build release binaries**
8. **Publish release notes**
9. **Announce release**

## Success Criteria

- ✅ All tests pass
- ✅ Variadic functions fully working
- ✅ Standard library available
- ✅ Cross-platform compilation works
- ✅ Documentation complete
- ✅ 10+ example programs
- ✅ No critical bugs
- ✅ Performance acceptable

## Timeline

### Week 1 (Current)
- [x] Variadic function infrastructure
- [x] Fix exitf()
- [x] Verify Windows C FFI
- [ ] Complete variadic argument collection (deferred - needs investigation)
- [x] Created stdlib.flap (reference documentation)
- [x] Added comprehensive examples (7 new examples)

### Week 2
- [ ] Complete stdlib.flap
- [ ] Improve error messages
- [ ] Write tutorial
- [ ] Create more examples

### Week 3
- [ ] Performance optimizations
- [ ] Complete documentation
- [ ] Testing and bug fixes
- [ ] Prepare release

### Week 4
- [ ] Final testing
- [ ] Release 2.0.0

## Current Progress: 75% Complete

**Completed:**
- Core language features ✅
- Cross-platform compilation ✅
- C FFI ✅
- Basic tests ✅
- Initial documentation ✅
- Variadic infrastructure ✅

**In Progress:**
- Bug fixes (map operation, recursion+printf) ⚠️
- Documentation expansion ⚠️

**Remaining:**
- Better errors
- Complete docs
- More examples
- Final testing

---

**Note:** This is a living document. Update as tasks complete.
**Last Updated:** 2025-11-24
