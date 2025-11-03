# Complex Technical Issues - Flapc Compiler

**Last Updated:** 2025-11-03
**Total Items:** 11 complex issues
**Estimated Effort:** 16-24 weeks (4-6 months)

This document tracks complex technical issues that require significant architectural changes, design decisions, or extensive refactoring. For straightforward technical debt that can be fixed without major changes, see DEBT.md.

---

## 1. ARCHITECTURAL ISSUES (4 items)

### 1.1 Monolithic Parser File
**Priority:** Medium
**Complexity:** High
**Effort:** 2-3 weeks
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
  ├── lexer.go           (already separate)
  ├── parser_core.go     (AST construction, ~3,000 lines)
  ├── parser_expr.go     (expression parsing, ~4,000 lines)
  ├── parser_stmt.go     (statement parsing, ~3,000 lines)
  ├── ast.go             (already separate)
  └── codegen/
      ├── interface.go   (CodeGenerator interface)
      ├── x86_64.go      (x86_64 implementation, ~6,000 lines)
      ├── arm64.go       (already separate)
      └── riscv64.go     (already separate)
```

**Option B: Split by Compilation Phase**
```
compiler/
  ├── frontend/
  │   ├── lexer.go
  │   ├── parser.go      (just parsing, ~7,000 lines)
  │   └── ast.go
  ├── middleend/
  │   ├── optimizer.go   (already separate)
  │   └── analyzer.go    (semantic analysis)
  └── backend/
      ├── codegen.go     (interface)
      ├── x86_64.go
      ├── arm64.go
      └── riscv64.go
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

### 1.2 ARM64/RISC-V Incomplete Instruction Sets
**Priority:** High
**Complexity:** High
**Effort:** 4-6 weeks per architecture
**Risk:** Medium

**Current State:**
- ARM64: 20 unimplemented functions (MovMemToReg, XMM ops, etc.)
- RISC-V64: 18 unimplemented functions (same missing functions)
- Many operations fall back to compilerError()

**Impact:**
- Floating-point operations unavailable
- SIMD operations unavailable
- Memory operations limited
- Cannot run most test programs
- Not production-ready

**Missing Instructions:**

**ARM64:**
```go
// Memory operations
- MovMemToReg / MovRegToMem
- LeaSymbolToReg

// Floating-point
- MovXmmToMem / MovMemToXmm
- MovRegToXmm / MovXmmToReg
- Cvtsi2sd / Cvttsd2si
- AddpdXmm / SubpdXmm / MulpdXmm / DivpdXmm
- Ucomisd

// Advanced
- XorRegWithImm
- Push/Pop (use str/ldr instead)
```

**RISC-V64:**
```go
// Same as ARM64, plus:
- PC-relative addressing (critical)
- Floating-point extensions (F, D)
- Compressed instructions (C extension)
- Atomic operations (A extension)
- Multiply/Divide (M extension)
```

**Proposed Solution:**

**Phase 1: ARM64 Completion (4 weeks)**
1. Implement memory operations (1 week)
   - LDR/STR for MovMemToReg/MovRegToMem
   - ADR/ADRP for LeaSymbolToReg
2. Implement floating-point (2 weeks)
   - FMOV for float moves
   - FCVT for conversions
   - FADD/FSUB/FMUL/FDIV for arithmetic
   - FCMP for comparisons
3. Testing and validation (1 week)
   - Run all test programs
   - Fix discovered issues

**Phase 2: RISC-V64 Completion (6 weeks)**
1. Implement PC-relative addressing (1 week)
   - AUIPC + ADDI pattern
   - LA (load address) pseudo-instruction
2. Implement memory operations (1 week)
   - LD/SD instructions
   - LA for symbols
3. Implement floating-point (2 weeks)
   - FLD/FSD for loads/stores
   - FADD.D/FSUB.D/FMUL.D/FDIV.D
   - FCVT.* conversions
   - FEQ/FLT/FLE comparisons
4. Implement additional extensions (1 week)
   - Multiply/Divide (M extension)
   - Basic atomic ops (A extension)
5. Testing and validation (1 week)

**Challenges:**
- Different calling conventions per architecture
- Different register sets and constraints
- Need to understand each ISA deeply
- Testing on real hardware may be difficult

**Decision Needed:**
- Priority: ARM64 or RISC-V64 first?
- Allocate dedicated time for completion?
- Need access to ARM64/RISC-V hardware for testing?

### 1.3 ARM64 Parallel Map Operator Crash
**Priority:** High
**Complexity:** Very High
**Effort:** 2-3 weeks
**Risk:** High

**Current State:**
- Parallel map operator (`||`) crashes on ARM64 with segfault
- Location: arm64_codegen.go:1444 in compileParallelExpr
- x86_64 implementation works fine
- Tests skip this feature on ARM64

**Impact:**
- Major language feature unavailable on ARM64
- Limits usefulness of ARM64 backend
- Blocks production use on Apple Silicon

**Symptoms:**
```bash
numbers || lambda  # Segfaults on ARM64
```

**Analysis Needed:**
1. **Compare x86_64 vs ARM64 implementations:**
   - What does x86_64 do differently?
   - Are there ARM64-specific issues (alignment, register usage)?

2. **Debug the crash:**
   - Use GDB/LLDB to find exact crash location
   - Check stack state at crash
   - Verify register contents
   - Look for memory corruption

3. **Review parallel codegen:**
   - Thread spawning code
   - Work distribution
   - Barrier synchronization
   - Lambda closure handling

**Suspected Issues:**
- Incorrect stack frame setup for threads
- Register corruption in thread context
- Closure environment not properly passed
- Alignment issues with thread stacks
- Race condition in parallel execution

**Proposed Solution:**

**Phase 1: Diagnosis (3-5 days)**
- [ ] Set up ARM64 debugging environment
- [ ] Reproduce crash with minimal test case
- [ ] Use debugger to find exact crash point
- [ ] Compare working x86_64 codegen side-by-side
- [ ] Review ARM64 ABI for threading

**Phase 2: Fix (1-2 weeks)**
- [ ] Implement fix based on diagnosis
- [ ] Test with increasing complexity
- [ ] Ensure all parallel tests pass
- [ ] Benchmark performance vs x86_64

**Phase 3: Validation (2-3 days)**
- [ ] Enable all parallel tests on ARM64
- [ ] Run test suite 100 times (catch race conditions)
- [ ] Test on real ARM64 hardware
- [ ] Update documentation

**Challenges:**
- Complex multi-threaded code hard to debug
- May require deep ARM64 assembly knowledge
- Could be multiple interacting issues
- Race conditions hard to reproduce

**Decision Needed:**
- Allocate dedicated time for this issue?
- Need ARM64 hardware for testing?
- Consider disabling feature permanently?

### 1.4 Atomic Operations in Parallel Loops
**Priority:** Medium
**Complexity:** High
**Effort:** 2-3 weeks
**Risk:** Medium

**Current State:**
- Atomic operations crash when used inside `@@` parallel loops
- Works fine in sequential code
- Crashes on both x86_64 and ARM64
- Currently documented as "known limitation"

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

**Proposed Solution:**

**Option A: Fix the Root Cause**
1. Debug parallel loop + atomic interaction
2. Fix register allocation for atomics in parallel context
3. Ensure thread-local state doesn't conflict
4. Test thoroughly

**Option B: Workaround**
1. Detect atomic ops in parallel loops at compile time
2. Emit helpful error message
3. Suggest alternatives (manual threading)
4. Document limitation clearly

**Option C: Defer to v2.0**
1. Keep current limitation
2. Add better error message
3. Provide example of manual threading with atomics
4. Fix in v2.0 with better parallel codegen

**Challenges:**
- Complex interaction of features
- May require redesign of parallel loop codegen
- Hard to test (race conditions)
- May not be fixable without major refactoring

**Decision Needed:**
- Which option to pursue?
- Is this a blocker for v1.7.4?
- Worth the effort vs workaround?

---

## 2. PLATFORM-SPECIFIC ISSUES (3 items)

### 2.1 macOS Stack Size Limitation
**Priority:** Low
**Complexity:** Very High
**Effort:** Unknown (may be impossible)
**Risk:** Very High

**Current State:**
- LC_MAIN command specifies 8MB stack
- macOS dyld provides only ~5.6KB stack
- Recursive lambdas overflow stack immediately
- Issue documented in macho_test.go:436

**Impact:**
- Recursive functions don't work on macOS ARM64
- Major language feature unavailable
- Limits usefulness of macOS builds

**Root Cause:**
- macOS dyld doesn't honor stacksize field in LC_MAIN
- Apple bug or intentional limitation?
- No documented workaround

**Attempted Solutions:**
- ✗ Setting LC_MAIN stacksize field (doesn't work)
- ✗ Increasing stack in code (too late, already crashed)

**Possible Solutions:**

**Option A: Custom Loader (Very Complex)**
- Write custom dyld replacement
- Load and execute Mach-O ourselves
- Set up stack before jumping to _main
- **Challenges:**
  - Extremely complex
  - Security implications
  - Apple may block
  - Maintenance burden very high

**Option B: Inline Stack Growth (Complex)**
- Detect stack usage at runtime
- Switch to heap-allocated stack when needed
- Continue execution on new stack
- **Challenges:**
  - Performance impact
  - Complex implementation
  - May not work with all code
  - Trampoline code needed

**Option C: Alternative Entry Point**
- Use LC_UNIXTHREAD instead of LC_MAIN
- Deprecated but might give more control
- **Challenges:**
  - Apple deprecated this
  - May not work on newer macOS
  - Still might not honor stack size

**Option D: Document and Live With It**
- Clearly document limitation
- Provide iterative alternatives to recursion
- Suggest using tail recursion (TCO works)
- **Advantages:**
  - No complex implementation
  - Clear expectation setting
  - Focus on other priorities

**Recommendation:** Option D (Document limitation)
- Not a compiler bug, it's a macOS limitation
- Workarounds are too complex/risky
- Tail recursion works fine
- Most real programs don't need deep recursion

**Decision Needed:**
- Accept this as permanent limitation?
- Investigate further?
- Contact Apple about issue?

### 2.2 Dynamic Linking Incomplete
**Priority:** Medium
**Complexity:** High
**Effort:** 3-4 weeks
**Risk:** Medium

**Current State:**
- Dynamic ELF generation incomplete for some architectures
- ldd test skipped due to code generation issues
- Some tests skip dynamic linking tests
- PLT/GOT implementation may have bugs

**Issues:**
- elf_test.go:444 - WriteCompleteDynamicELF not fully implemented
- dynamic_test.go:279 - ldd test skipped
- dynamic_test.go:87 - No symbol section warning

**Impact:**
- Cannot fully link against shared libraries
- FFI may not work in all cases
- Dynamic executables may fail
- Limits library ecosystem

**Analysis Needed:**
1. What exactly is missing in WriteCompleteDynamicELF?
2. Why does ldd test fail?
3. Are there platform-specific issues?
4. Does static linking work as alternative?

**Proposed Solution:**
1. Complete dynamic section generation
2. Fix symbol table generation
3. Implement proper PLT/GOT for all platforms
4. Test with various shared libraries
5. Enable all dynamic linking tests

**Challenges:**
- ELF format complex
- Platform-specific details (Linux vs FreeBSD)
- Need to understand dynamic linker behavior
- Testing requires system libraries

**Decision Needed:**
- Priority for v1.7.4?
- Static linking sufficient for now?
- Need expert review of ELF code?

### 2.3 Platform-Specific Code Duplication
**Priority:** Low
**Complexity:** Medium
**Effort:** 1-2 weeks
**Risk:** Low

**Current State:**
- parallel_unix.go vs parallel_other.go
- filewatcher_unix.go vs filewatcher_other.go
- hotreload_unix.go vs hotreload_other.go
- Significant code duplication

**Issues:**
- Changes must be made to multiple files
- Easy to miss platform-specific bugs
- Testing harder (need multiple OSes)

**Proposed Solution:**

**Option A: Extract Common Code**
```go
parallel.go           // Common interface
parallel_unix.go      // Unix-specific
parallel_windows.go   // Windows-specific (future)
parallel_darwin.go    // macOS-specific
```

**Option B: Use Build Tags More Extensively**
```go
//go:build unix
// Unix implementation

//go:build !unix
// Fallback implementation
```

**Option C: Abstract Platform Layer**
```go
platform/
  ├── interface.go    // Common interface
  ├── unix.go         // Unix implementation
  └── other.go        // Fallback
```

**Challenges:**
- Need to test on multiple platforms
- May introduce abstraction overhead
- Build tags can be confusing

**Decision Needed:**
- Worth the refactoring effort?
- Which approach to use?
- When to schedule this work?

---

## 3. PERFORMANCE & OPTIMIZATION (2 items)

### 3.1 Register Allocator Implementation
**Priority:** Low
**Complexity:** Very High
**Effort:** 6-8 weeks
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
1. **Phase 1:** Implement linear scan algorithm (2 weeks)
2. **Phase 2:** Add register coalescing (2 weeks)
3. **Phase 3:** Optimize for architecture-specific constraints (2 weeks)
4. **Phase 4:** Profile and tune (2 weeks)

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

### 3.2 Missing Optimizations
**Priority:** Low
**Complexity:** High
**Effort:** 8-12 weeks (all optimizations)
**Risk:** Medium

**Current State:**
From TODO.md, these optimizations are planned but not implemented:
- Auto-vectorization
- Profile-Guided Optimization (PGO)
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

**Phase 1: CSE (2 weeks)**
- Detect repeated expressions
- Compute once, reuse result
- Significant wins for math-heavy code

**Phase 2: Strength Reduction (1 week)**
- Replace expensive operations with cheaper ones
- x * 2 → x << 1
- x / power-of-2 → x >> n
- Easy wins, low risk

**Phase 3: Loop-Invariant Code Motion (2 weeks)**
- Move calculations out of loops
- Significant performance gain
- Requires dataflow analysis

**Phase 4: Escape Analysis (3 weeks)**
- Determine if allocations can be on stack
- Reduce heap pressure
- Improve performance
- Complex analysis

**Phase 5: Auto-Vectorization (4 weeks)**
- Detect SIMD-friendly loops
- Generate vector code automatically
- Huge performance gains possible
- Very complex

**Phase 6: PGO (4 weeks)**
- Profile-guided optimization
- Use runtime data to guide optimization
- Requires two-pass compilation
- Complex infrastructure

**Challenges:**
- Each optimization is complex
- Need to preserve correctness
- Interactions between optimizations
- Need good test coverage
- Performance gains vary

**Decision Needed:**
- Which optimizations for v2.0?
- Priority order?
- Allocate dedicated time?

---

## 4. LANGUAGE FEATURES (2 items)

### 4.1 Disabled Language Features
**Priority:** Low
**Complexity:** High
**Effort:** 4-6 weeks
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

**Decision Needed:**
- Are these features important?
- Worth the parsing complexity?
- Keep disabled in v1.7.4?

### 4.2 Pipe-Based Result Waiting
**Priority:** Low
**Complexity:** Medium
**Effort:** 1-2 weeks
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

**Decision Needed:**
- Needed for v1.7.4?
- Design finalized?
- Part of v2.0 channels feature?

---

## Summary

### By Priority
- **High:** 4 items (ARM64/RISC-V completion, parallel map crash, atomics)
- **Medium:** 4 items (parser refactoring, dynamic linking, atomic ops)
- **Low:** 3 items (register allocator, optimizations, features)

### By Complexity
- **Very High:** 3 items (parser refactor, stack size, register allocator)
- **High:** 7 items (instruction sets, parallel crash, atomics, etc.)
- **Medium:** 1 item (pipe-based waiting)

### By Risk
- **Very High:** 1 item (macOS stack size)
- **High:** 5 items (parser refactor, instruction sets, parallel crash)
- **Medium:** 4 items (atomics, linking, platform code, optimizations)
- **Low:** 1 item (pipe waiting)

### Recommendations

**For v1.7.4:**
1. Document macOS stack limitation (accept as-is)
2. Document atomic operations limitation (accept as-is)
3. Document ARM64 parallel map crash (work in progress)
4. Keep disabled features disabled (revisit in v2.0)

**For v2.0:**
1. Complete ARM64 instruction set (HIGH PRIORITY)
2. Fix ARM64 parallel map operator (HIGH PRIORITY)
3. Begin parser refactoring (MEDIUM PRIORITY)
4. Implement priority optimizations (CSE, strength reduction)

**For v3.0+:**
1. Complete RISC-V64 instruction set
2. Implement register allocator
3. Complete all optimizations
4. Fix or remove disabled features

### Decision Points

Critical decisions needed before proceeding:
1. **Parser refactoring:** When and how?
2. **ARM64 completion:** Dedicated time allocation?
3. **macOS stack:** Accept limitation or investigate further?
4. **Atomic operations:** Fix or document as limitation?
5. **Optimization priority:** Which ones for v2.0?

---

**Note:** These are complex issues requiring significant design, implementation, and testing effort. Unlike DEBT.md items, these cannot be quickly fixed and need careful planning and dedicated time allocation.
