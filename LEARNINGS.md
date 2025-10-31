# Flapc Development Learnings

Hard-earned lessons, design decisions, and insights from building a production compiler.

## Core Design Decisions

### 1. Unified Type System: `map[uint64]float64`

**Decision:** Everything is a sparse hash map internally.

**Why:**
- **Simplicity**: One runtime representation for numbers, strings, lists, objects
- **Performance**: SIMD-optimized lookups (AVX-512/SSE2) make maps fast
- **Flexibility**: Easy to add new "types" - they're just different usage patterns

**Tradeoffs:**
- Memory overhead: Small values (integers, booleans) still need map storage
- Cache locality: Not as good as packed arrays for numeric data
- Learning curve: Counterintuitive for programmers expecting traditional types

**Lesson:** Simplicity wins. The unified type system eliminated hundreds of edge cases in codegen and makes the language much easier to implement.

### 2. Direct Machine Code Generation

**Decision:** Lexer → Parser → x86-64 → ELF. No IR, no LLVM.

**Why:**
- **Compilation speed**: ~1ms for typical programs (vs seconds with LLVM)
- **Simplicity**: Fewer layers to debug
- **Learning**: Forces deep understanding of x86-64, ELF, calling conventions

**Tradeoffs:**
- Limited optimization: Can't do global analysis without IR
- Platform porting: Need separate codegen for each CPU (x86/ARM/RISC-V)
- Debugging: Harder to debug than IR-based compilers

**Lesson:** For a small language, direct codegen is viable and fast. But we're hitting the point where a proper register allocator requires something IR-like.

### 3. Immutable by Default (`=` vs `:=`)

**Decision:** Variables assigned with `=` cannot be reassigned. Use `:=` for mutable.

**Why:**
- **Safety**: Prevents accidental mutations
- **Reasoning**: Easier to understand code flow
- **Optimization**: Compiler knows values don't change

**Tradeoffs:**
- Confusion: Newcomers expect `=` to be mutable
- Verbosity: Need `:=` for counters, accumulators, etc.

**Lesson:** Users initially complain, then appreciate it. The key is clear error messages: "variable 'x' is immutable, use ':=' if you need to reassign".

### 4. Tail-Call Optimization as First-Class Feature

**Decision:** Automatic TCO with explicit `->` syntax for clarity.

**Why:**
- **Enables functional style**: Recursive algorithms without stack overflow
- **Game loops**: `@ { ... }` compiles to jmp instruction (zero overhead)
- **Clarity**: `->` makes tail calls explicit in source

**Tradeoffs:**
- Teaching burden: Programmers must understand TCO
- Debugging: Stack traces don't show tail-called functions

**Lesson:** TCO is essential for systems programming without GC. The `->` syntax makes it visible and explicit.

### 5. Type Names: Full Forms Only

**Decision:** `int32`, `uint64`, `float32` - never `i32`, `u64`, `f32`.

**Why:**
- **Readability**: Clear what each type means
- **Consistency**: Matches C99 stdint.h conventions
- **Professionalism**: Looks more mature than abbreviated forms

**Lesson:** Spent time refactoring from abbreviated to full names. Should have started with full names from day one. Users universally prefer the full forms.

## Implementation Lessons

### 1. Parser: Don't Be Clever

**Mistake:** Early parser tried to be too clever with operator precedence and expression parsing.

**Fix:** Straightforward recursive descent with explicit precedence levels.

**Lesson:** Parser clarity > parser cleverness. When debugging at 2am, you want obvious code.

### 2. ELF Generation: Trust But Verify

**Mistake:** Initially trusted my ELF generation was correct.

**Problem:** Subtle bugs in section alignment, relocation types, PLT/GOT setup.

**Fix:** Compare against GCC output with `objdump -d`, `readelf -a`, manual hex dumps.

**Lesson:** ELF spec is precise but easy to misunderstand. Always validate against reference implementation (GCC/Clang).

### 3. Calling Conventions Are Hard

**Mistake:** Assumed System V ABI was straightforward.

**Reality:**
- Caller-saved vs callee-saved registers matter
- Stack alignment (16-byte boundary before `call`)
- Red zone (-128 bytes below RSP not to be touched)
- Varargs require `al` to specify float count

**Lesson:** Read the ABI document 10 times. Test with simple C programs. Compare assembly output.

### 4. PLT/GOT for Dynamic Linking

**Problem:** Initially generated direct calls to libc, which failed.

**Solution:** Proper PLT (Procedure Linkage Table) and GOT (Global Offset Table) setup.

**Key insight:** First call goes through PLT stub → dynamic linker → resolves symbol → updates GOT. Subsequent calls: PLT → GOT (cached) → function.

**Lesson:** Dynamic linking is complex but necessary for C FFI. The PLT/GOT indirection is worth it for seamless library integration.

### 5. Parallel Loops: Futex Over Pthread

**Decision:** Use raw `futex()` syscall for barrier synchronization instead of pthread.

**Why:**
- **Control**: Know exactly what's happening
- **Performance**: No pthread overhead
- **Learning**: Deep understanding of synchronization primitives

**Tradeoffs:**
- Portability: Linux-specific (need pthread fallback for other OSes)
- Complexity: Easy to get wrong (memory ordering, spurious wakeups)

**Lesson:** For performance-critical primitives, going low-level is worth it. But have fallbacks for portability.

## Testing Insights

### 1. Test Everything, Test Early

**Approach:**
- 344+ test programs covering all features
- Integration tests that compile and run actual Flap code
- `.result` files for expected output comparison

**Lesson:** Test-driven development for compilers is incredibly effective. When refactoring, tests catch regressions immediately.

### 2. The `-short` Flag

**Problem:** Full test suite (6s) too slow for rapid iteration.

**Solution:** `-short` flag runs 9 essential tests in 0.3s (~20x faster).

**Lesson:** Fast feedback loop > comprehensive testing during development. Save full suite for CI.

### 3. Wildcard Matching in Test Output

**Problem:** Pointer addresses, timestamps change between runs.

**Solution:** Use `*` wildcard in `.result` files to match any number.

```
Allocated at pointer: *    // Matches any address
Time elapsed: * ms         // Matches any duration
```

**Lesson:** Test output comparison needs flexibility for non-deterministic values.

## Performance Lessons

### 1. Register Allocation Matters

**Current state:** Ad-hoc register usage leads to many unnecessary `mov` instructions.

**Impact:** ~30-40% more instructions than necessary in tight loops.

**Next step:** Implement linear-scan register allocator (Priority 1).

**Lesson:** Premature optimization is evil, but late optimization is expensive. Should have done register allocation earlier.

### 2. Compilation Speed vs Runtime Speed

**Tradeoff:** Fast compilation (1ms) means less time for optimization.

**Reality:** For game development, compile time matters more than 5% runtime difference.

**Lesson:** Know your audience. Game developers iterate rapidly - compilation speed wins.

### 3. String Operations Are Slow

**Problem:** Everything-is-a-map means strings are sparse maps, not packed arrays.

**Impact:** String operations ~10x slower than C.

**Potential fix:** Special-case strings as dense arrays when possible.

**Lesson:** Unified type system has costs. Profile before committing to a representation.

## Language Design Insights

### 1. Syntax: Less Is More

**Initial design:** Many operators (`|||`, `or!`, `@++`, `@1++`, etc.)

**Reality:** Most operators never used in practice.

**Lesson:** Start minimal. Add features only when users request them. Removing features is much harder than adding.

### 2. Error Messages Matter

**Bad:** `error: expected '{' at line 42`

**Good:** `error: expected '{' after 'arena' keyword at arena_test.flap:42`

**Lesson:** Error messages are user interface. Include:
- What went wrong
- Where it went wrong (file:line)
- What was expected
- Context (surrounding tokens)

### 3. Examples > Documentation

**Observation:** Users learn from `testprograms/*.flap` more than from docs.

**Lesson:** Provide many small, focused examples. Each should demonstrate one feature clearly.

### 4. C FFI: Automatic > Manual

**Initial approach:** Manual function signatures for C calls.

**Better approach:** Read DWARF debug info from libraries automatically.

**Lesson:** C FFI should "just work" - no boilerplate. Compiler should infer types from debug info.

## Architectural Decisions

### 1. Single-Pass Compilation

**Decision:** Parse and emit code in one pass.

**Why:**
- Simple implementation
- Fast compilation
- Low memory usage

**Tradeoffs:**
- No forward references
- Limited optimization
- Harder to implement features like mutual recursion

**Lesson:** Single-pass works well for small-to-medium programs. For large projects, might need multiple passes.

### 2. No Garbage Collector

**Decision:** Manual memory management with arena allocators.

**Why:**
- Predictable performance (no GC pauses)
- Suitable for games/real-time applications
- Simpler runtime

**Tradeoffs:**
- User must think about memory
- Potential for leaks/use-after-free
- Arena discipline required

**Lesson:** For systems programming, manual control > GC convenience. Arenas make it tolerable.

### 3. Static Linking by Default

**Decision:** Generate statically-linked ELF by default.

**Why:**
- Zero dependencies at runtime
- Predictable deployment
- Fast startup (no dynamic linking)

**Tradeoffs:**
- Larger binaries
- Can't share code between processes
- Must recompile to update libraries

**Lesson:** For game development, static linking wins. Single-file deployment is worth the binary size.

## Debugging Experiences

### 1. GDB/LLDB Are Essential

**Approach:** Generate minimal DWARF info for source-line mapping.

**Impact:** Can set breakpoints by line number, see source in debugger.

**Next:** Full DWARF support (variables, types, stack unwinding).

**Lesson:** Debug info is not optional. Without it, debugging is painful.

### 2. Printf Debugging Still Works

**Reality:** Often faster than setting up debugger.

**Tip:** Add `-v` flag to compiler to show generated assembly.

**Lesson:** Modern tools are great, but sometimes `println(x)` is fastest.

### 3. Valgrind for Memory Errors

**Use case:** Detect leaks in arena implementation.

**Command:** `valgrind --leak-check=full ./program`

**Lesson:** Run valgrind on all new features. Find leaks early.

## Future Direction

### What Worked Well

1. **Direct codegen** - Fast compilation, simple implementation
2. **C FFI** - Seamless SDL3/OpenGL integration
3. **Parallel loops** - Simple syntax (`@@`) for powerful feature
4. **Test-driven development** - Caught countless bugs early
5. **Immutable by default** - Users appreciate after initial learning curve

### What Needs Work

1. **Register allocator** - Ad-hoc usage is hurting performance
2. **DWARF debug info** - Need full support for variables/types
3. **Optimization passes** - Currently zero optimization beyond TCO
4. **Documentation** - Need more tutorials and guides
5. **Error messages** - Can be much better with more context

### What To Avoid

1. **Feature creep** - Resist adding every requested feature
2. **Clever tricks** - Straightforward code > clever code
3. **Premature abstraction** - Solve concrete problems first
4. **Complex type system** - Unified representation is a strength
5. **Breaking changes** - Stability matters for real users

## Conclusion

Building Flapc taught me:

1. **Simple designs scale** - The unified type system eliminated complexity
2. **Testing is essential** - 344 tests give confidence to refactor
3. **Performance comes later** - Get correctness first, optimize second
4. **Users want speed** - Compilation time matters more than optimizations
5. **Documentation is hard** - Examples teach better than prose

The key insight: **Focus on one thing and do it well.** Flap's goal is fast-compiling, C-FFI-capable systems programming for games. Every feature should serve that goal.

---

*"The best code is no code. The second best is simple code that works."*
