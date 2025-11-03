# Complex Technical Issues - Flapc Compiler

This document tracks complex technical issues that require significant architectural changes, design decisions, or extensive refactoring. For straightforward technical debt that can be fixed without major changes, see DEBT.md. For platform-specific complex issues (ARM64, RISC-V64, macOS), see PLATFORMS.md.

---

## 1. ARCHITECTURAL ISSUES (2 items)

**Note:** Platform-specific architectural issues moved to PLATFORMS.md

### 1.1 Monolithic Parser File
**Priority:** Medium
**Complexity:** High
**Risk:** High (breaking changes)

**Current State:**
- parser.go: 16,927 lines, 170+ functions
- Combines lexing, parsing, AST construction, AND x86_64 code generation
- Hard to navigate, review, and maintain
- Makes cross-platform development difficult

**Issues:**
- Violates single responsibility principle
- x86_64 codegen mixed with parser logic
- ARM64/RISC-V implementations scattered across separate files
- Hard to find relevant code
- Merge conflicts common
- Testing difficult

**Proposed Solution:**

**Option A: Split by Responsibility (Recommended)**
```
parser/
  ├── lexer.go             (already separate)
  ├── parser_core.go       (AST construction, ~3,000 lines)
  ├── parser_expr.go       (expression parsing, ~4,000 lines)
  ├── parser_stmt.go       (statement parsing, ~3,000 lines)
  ├── ast.go               (already separate)
  ├── codegen_interface.go (CodeGenerator interface)
  ├── codegen_x86_64.go    (x86_64 implementation, ~6,000 lines)
  ├── codegen_arm64.go     (already separate)
  └── codegen_riscv64.go   (already separate)
```

**Challenges:**
- Breaking existing code
- Need to move 16K+ lines carefully
- Test suite must pass at every step
- Imports will change throughout codebase

**Approach:**
1. **Phase 1:** Extract code generation to separate functions (keep in parser.go)
2. **Phase 2:** Move codegen functions to new files (symlink initially)
3. **Phase 3:** Update imports and remove symlinks
4. **Phase 4:** Refactor parser into logical modules
5. **Phase 5:** Update documentation and examples

**Decision Needed:**
- Which splitting strategy to use?
- When to schedule this refactoring?
- How to maintain backward compatibility?

### 1.2 Atomic Operations in Parallel Loops
**Priority:** Medium (WORKAROUND IMPLEMENTED)
**Complexity:** High
**Risk:** Medium
**Status:** ✅ Compile-time detection implemented (v1.7.4)

**Current State:**
- Atomic operations crash when used inside `@@` parallel loops (root cause: register conflicts)
- Works fine in sequential code
- Crashes on both x86_64 and ARM64
- **NOW**: Compiler detects this pattern and provides helpful error message with workarounds

**Issue:**
```flap
ptr := malloc(8)
atomic_store(ptr, 0)

@@ i in 0..<10 {
    atomic_add(ptr, i)  // Crashes here
}
```

**Symptoms:**
- Segfault (exit code 139)
- Crashes immediately upon loop entry
- No useful error message

**Analysis Needed:**
1. **Why does parallel loop codegen conflict with atomics?**
   - Are atomic builtin calls incompatible with thread spawning?
   - Register clobbering issue?
   - Stack corruption?

2. **How does x86_64 parallel loop work?**
   - Review thread spawning mechanism
   - Check register preservation
   - Verify stack frame setup

3. **What's different about atomic operations?**
   - Use inline assembly (xchg, lock prefix)
   - Require specific registers
   - May have alignment requirements

**Solution Implemented (v1.7.4):**

The compiler now detects atomic operations in parallel loops at compile time and emits a clear error message with workarounds:

```
Error: Atomic operations inside parallel loops (@@ or N @) are not currently supported
       This causes a segfault due to register conflicts in the current implementation.

Workarounds:
  1. Use a sequential loop (@ instead of @@) with atomic operations
  2. Use manual thread spawning with spawn expressions
  3. Restructure code to avoid atomic ops inside parallel loop
```

**Implementation Details:**
- Added `hasAtomicOperations()` function that recursively scans loop body AST
- Added `hasAtomicInExpr()` to check expressions for atomic calls
- Detection occurs before code generation in `compileParallelRangeLoop()`
- Prevents silent crashes and guides users to working alternatives

**Future Work (v2.0+):**
1. Debug parallel loop + atomic interaction
2. Fix register allocation for atomics in parallel context
3. Ensure thread-local state doesn't conflict
4. Test thoroughly

**Challenges for Full Fix:**
- Complex interaction of features
- May require redesign of parallel loop codegen
- Hard to test (race conditions)
- May not be fixable without major refactoring

---

## 2. PERFORMANCE & OPTIMIZATION (2 items)

**Note:** Platform-specific issues moved to PLATFORMS.md

### 2.1 Register Allocator Implementation
**Priority:** Low
**Complexity:** Very High
**Risk:** High

**Current State:**
- Ad-hoc register usage
- No register pressure tracking
- Spills to stack frequently
- Documented in REGISTER_ALLOCATOR.md

**Impact:**
- Generated code not optimal
- More stack usage than necessary
- Slower performance
- Larger code size

**Proposed Solution:**
Implement proper register allocator with:
1. Live range analysis
2. Register pressure tracking
3. Smart spilling strategy
4. Register coalescing
5. Callee-saved register usage

**Current Architecture:**
```go
type RegisterAllocator struct {
    registers     map[string]bool     // Available registers
    allocated     map[string]Variable // Current allocations
    spillCount    int                 // Stack spill counter
    liveIntervals map[Variable]Interval
}
```

**Approach:**
1. **Phase 1:** Implement linear scan algorithm
2. **Phase 2:** Add register coalescing
3. **Phase 3:** Optimize for architecture-specific constraints
4. **Phase 4:** Profile and tune

**Challenges:**
- Complex algorithm
- Architecture-specific constraints
- Need to preserve correctness
- Performance sensitive
- Hard to test thoroughly

**References:**
- REGISTER_ALLOCATOR.md has detailed plan
- Linear scan paper (Poletto & Sarkar)
- LLVM register allocator source

**Decision Needed:**
- Priority for v2.0?
- Worth the complexity?
- Performance gain measurable?

### 2.2 Missing Optimizations
**Priority:** Low
**Complexity:** High
**Risk:** Medium

**Current State:**
From TODO.md, these optimizations are planned but not implemented:
- Auto-vectorization
- Escape analysis
- Common Subexpression Elimination (CSE)
- Strength reduction
- Loop-invariant code motion (LICM)

**Impact:**
- Generated code could be significantly faster
- Competitive with optimizing compilers
- Better resource usage

**Proposed Solution:**

Implement optimizations in priority order:

**Phase 1: CSE **
- Detect repeated expressions
- Compute once, reuse result
- Significant wins for math-heavy code

**Phase 2: Strength Reduction **
- Replace expensive operations with cheaper ones
- x * 2 → x << 1
- x / power-of-2 → x >> n
- Easy wins, low risk

**Phase 3: Loop-Invariant Code Motion **
- Move calculations out of loops
- Significant performance gain
- Requires dataflow analysis

**Phase 4: Escape Analysis **
- Determine if allocations can be on stack
- Reduce heap pressure
- Improve performance
- Complex analysis

**Phase 5: Auto-Vectorization **
- Detect SIMD-friendly loops
- Generate vector code automatically
- Huge performance gains possible
- Very complex

**Challenges:**
- Each optimization is complex
- Need to preserve correctness
- Interactions between optimizations
- Need good test coverage
- Performance gains vary

---

## 3. LANGUAGE FEATURES (2 items)

### 3.1 Disabled Language Features
**Priority:** Low
**Complexity:** High
**Risk:** High

**Current State:**
From parser.go, these features are explicitly disabled:
- Blocks-as-arguments (parser.go:3871, 3962, 4082)
- Struct literal syntax (parser.go:4041)

**Why Disabled:**
- Conflicts with match expressions
- Parsing ambiguity
- Design not finalized

**Impact:**
- Language less expressive
- Some patterns impossible
- Users may expect these features

**Examples:**

**Blocks-as-arguments:**
```flap
// Would like to write:
map(list) { x => x * 2 }

// Currently must write:
map(list, x => x * 2)
```

**Struct literals:**
```flap
// Would like to write:
player := {name: "Alice", health: 100}

// Currently must write:
player := map()
player["name"] <- "Alice"
player["health"] <- 100
```

**Analysis Needed:**
1. What is the parsing ambiguity?
2. Can syntax be disambiguated?
3. Is feature worth the complexity?
4. Would users actually use it?

**Proposed Solution:**

**Option A: Redesign Syntax**
- Find non-ambiguous syntax
- Update parser
- Enable features
- Add tests

**Option B: Use Different Syntax**
```flap
// Blocks-as-arguments with `do`:
map(list) do { x => x * 2 }

// Struct literals with `new`:
player := new {name: "Alice", health: 100}
```

**Option C: Keep Disabled**
- Document why disabled
- Provide alternatives
- Revisit in v2.0 with fresh perspective

**Challenges:**
- Parsing ambiguity hard to solve
- May require grammar changes
- Risk breaking existing code
- Users may have different syntax preferences

### 3.2 Pipe-Based Result Waiting
**Priority:** Low
**Complexity:** Medium
**Risk:** Low

**Current State:**
- parser.go:6419: `// TODO: Implement pipe-based result waiting`
- Spawn expression doesn't support result retrieval
- Process spawned but can't get return value

**Impact:**
- spawn expression less useful
- Cannot implement fork/join patterns
- Limits concurrent programming

**Example Need:**
```flap
// Spawn computation
result_pipe := spawn heavy_computation(data)

// Do other work...
do_other_stuff()

// Wait for result
value := wait(result_pipe)
```

**Proposed Solution:**
1. Implement pipe creation for spawn
2. Add wait() builtin to block on pipe
3. Support timeouts
4. Handle errors

**Challenges:**
- Need OS pipe implementation
- Cross-platform (Unix vs Windows)
- Error handling complex
- Blocking vs non-blocking

---

## Summary

### By Priority
- **High:** 0 items (all platform-specific items moved to PLATFORMS.md)
- **Medium:** 2 items (parser refactoring, atomic ops)
- **Low:** 5 items (register allocator, optimizations, features)

### By Complexity
- **Very High:** 2 items (parser refactor, register allocator)
- **High:** 3 items (parser refactor, atomics, disabled features)
- **Medium:** 2 items (pipe-based waiting, platform code)

### By Risk
- **High:** 3 items (parser refactor, atomics, disabled features)
- **Medium:** 2 items (atomics, optimizations)
- **Low:** 2 items (pipe waiting, features)

### Recommendations

**For v1.7.4:**
1. Document atomic operations limitation (accept as-is)
2. Keep disabled features disabled (revisit in v2.0)
3. Defer all complex refactoring to v2.0

**For v2.0:**
1. Begin parser refactoring (MEDIUM PRIORITY)
2. Implement priority optimizations (CSE, strength reduction)
3. Consider fixing atomic operations in parallel loops

**For v3.0+:**
1. Implement register allocator
2. Complete all optimizations
3. Fix or remove disabled features
4. Implement pipe-based result waiting

---

**Note:** These are complex issues requiring significant design, implementation, and testing effort. Unlike DEBT.md items, these cannot be quickly fixed and need careful planning. But! We will be bold in the face of complexity and use techniques from computer science, "How to Solve It?" by Polya and from the software engineering profession!
