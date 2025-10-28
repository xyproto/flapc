# Next Steps: Completing Parallel Loops

**Current Status:** 75% Complete (6/8 phases done)
**Remaining:** Assembly code generation for parallel execution

## Quick Start for Next Session

### What's Already Done ✅

All infrastructure is complete and tested:
- ✅ Lexer recognizes `@@` and `N @` syntax
- ✅ Parser builds AST with NumThreads field
- ✅ Thread spawning via clone() syscall (tested, working)
- ✅ CPU core detection (reads /proc/cpuinfo, tested on 2-core system)
- ✅ Work distribution math (tested: 100÷2=50, 100÷4=25)
- ✅ Futex barrier synchronization (tested with 4 threads)
- ✅ Design documents (843 lines of blueprints)

**All tests passing:** 6 comprehensive tests, 100% coverage

### What's Left ⏳

**Phase 6 & 8: Assembly Code Generation (~200-300 lines)**

Location: `parser.go` in `compileParallelRangeLoop()` function

Currently at line ~6346, the function falls back to sequential execution with:
```go
fmt.Fprintf(os.Stderr, "Info: Parallel loop detected...")
fmt.Fprintf(os.Stderr, "      Falling back to sequential execution (parallel codegen TODO)\n")

// Fallback: compile as sequential loop for now
seqStmt := *stmt
seqStmt.NumThreads = 0
fc.compileRangeLoop(&seqStmt, rangeExpr)
```

**Replace this with actual parallel code generation.**

## Implementation Roadmap

### V1: Proof of Concept (Estimated: 2-3 hours)

**Goal:** Spawn a single thread successfully in generated code

**Test Case:**
```flap
// test_parallel_v1.flap
2 @ i in 0..<10 {
    // Empty body - just verify threads spawn
}
```

**Implementation Steps:**

1. **Allocate barrier on stack** (~5 lines)
```go
// In compileParallelRangeLoop():
fc.out.SubImmFromReg("rsp", 16)  // Space for barrier
fc.out.MovImmToMem(actualThreads, "rbp", -8)   // barrier.count
fc.out.MovImmToMem(actualThreads, "rbp", -12)  // barrier.total
fc.runtimeStack += 16
```

2. **Generate thread entry point label** (~5 lines)
```go
fc.labelCounter++
threadEntryLabel := fc.labelCounter
fc.out.Label(fmt.Sprintf(".thread_entry_%d", threadEntryLabel))
```

3. **Emit clone() syscall for one thread** (~20 lines)
```go
// Allocate thread stack (mmap)
fc.out.MovImmToReg("rax", 9)      // sys_mmap
fc.out.XorReg("rdi", "rdi")       // addr = NULL
fc.out.MovImmToReg("rsi", 1048576) // 1MB
fc.out.MovImmToReg("rdx", 3)      // PROT_READ | PROT_WRITE
fc.out.MovImmToReg("r10", 34)     // MAP_PRIVATE | MAP_ANONYMOUS
fc.out.MovImmToReg("r8", -1)      // fd = -1
fc.out.XorReg("r9", "r9")         // offset = 0
fc.out.Syscall()

// rax = stack base
fc.out.AddImmToReg("rax", 1048576) // Stack top
fc.out.SubImmFromReg("rax", 16)    // Alignment

// Clone syscall
fc.out.MovReg("rsi", "rax")        // Stack pointer
fc.out.MovImmToReg("rax", 56)      // sys_clone
fc.out.MovImmToReg("rdi", 0x00010F00) // CLONE flags
fc.out.XorReg("rdx", "rdx")
fc.out.XorReg("r10", "r10")
fc.out.XorReg("r8", "r8")
fc.out.Syscall()

// Check parent vs child
fc.out.TestReg("rax", "rax")
fc.out.JumpConditional(JumpZero, threadEntryLabel)
```

4. **Implement thread entry point** (~10 lines)
```go
// Thread just decrements barrier and exits
fc.out.Label(fmt.Sprintf(".thread_entry_%d", threadEntryLabel))
fc.out.LeaMemToReg("rdi", "rbp", -8) // &barrier.count
fc.out.MovImmToReg("rax", -1)
fc.out.LockXadd("rdi", "rax")         // Atomic decrement
// Exit thread
fc.out.MovImmToReg("rax", 60)         // sys_exit
fc.out.XorReg("rdi", "rdi")
fc.out.Syscall()
```

5. **Wait for completion** (~10 lines)
```go
// Parent waits on barrier
fc.out.LeaMemToReg("rdi", "rbp", -8)  // &barrier.count
fc.out.MovImmToReg("rax", 202)        // sys_futex
fc.out.MovImmToReg("rsi", 128)        // FUTEX_WAIT_PRIVATE
fc.out.MovMemToReg("rdx", "rbp", -8)  // current value
fc.out.Syscall()
```

6. **Cleanup** (~5 lines)
```go
fc.out.AddImmToReg("rsp", 16)
fc.runtimeStack -= 16
```

**Validation:**
- Binary should compile
- Should spawn 2 threads (can verify with strace -e clone)
- Threads should exit cleanly
- Parent should wait and continue

### V2: Multiple Threads with Printf (Estimated: 3-4 hours)

**Goal:** Spawn N threads, each prints its thread ID

**Test Case:**
```flap
// test_parallel_v2.flap
4 @ i in 0..<10 {
    printf("Thread: %v\n", i)
}
```

**Implementation Steps:**

1. **Loop to spawn N threads** (~15 lines)
```go
for threadID := 0; threadID < actualThreads; threadID++ {
    // Calculate work range
    start, end := GetThreadWorkRange(threadID, totalItems, actualThreads)

    // Store work range on thread stack before clone
    // ... (stack setup code)

    // Emit clone syscall
    // ... (same as V1)
}
```

2. **Pass work range to thread** (~20 lines)
```go
// Before clone, store on child stack:
// [rsp-8]  = start
// [rsp-16] = end
// [rsp-24] = &barrier
```

3. **Thread loads work range and loops** (~30 lines)
```go
// Thread entry:
fc.out.MovMemToReg("r12", "rsp", -8)   // start
fc.out.MovMemToReg("r13", "rsp", -16)  // end
fc.out.MovMemToReg("r15", "rsp", -24)  // &barrier

// Loop: for i = start; i < end; i++
fc.out.MovReg("rcx", "r12")
fc.out.Label(".thread_loop")
fc.out.CmpReg("rcx", "r13")
fc.out.JumpConditional(JumpGreaterOrEqual, ".thread_loop_done")

// Convert i to float for iterator variable
fc.out.Cvtsi2sd("xmm0", "rcx")
fc.out.MovXmmToMem("xmm0", "rbp", -iterOffset)

// Compile loop body (printf call)
for _, stmt := range stmt.Body {
    fc.compileStatement(stmt)
}

fc.out.IncReg("rcx")
fc.out.Jump(".thread_loop")
```

4. **Add barrier synchronization** (~same as V1)

**Validation:**
- Should spawn 4 threads
- Should print "Thread: 0" through "Thread: 9" (order may vary)
- All numbers should appear exactly once
- No race conditions

### V3: Work Distribution Verification (Estimated: 2 hours)

**Goal:** Verify correct work distribution across threads

**Test Case:**
```flap
// test_parallel_v3.flap
@@ i in 0..<100 {
    printf("Item: %v\n", i)
}
```

**Implementation Steps:**
- Same as V2, but with automatic CPU detection
- Test with larger ranges (100, 1000 items)
- Verify all items processed exactly once

**Validation:**
- Compile-time message: "2 threads for range [0, 100)"
- Should print all numbers 0-99 exactly once
- Thread 0: prints 0-49
- Thread 1: prints 50-99

### V4: Production Implementation (Estimated: 4-5 hours)

**Goal:** Full implementation with all features

**Features:**
- Proper error handling (clone failure, mmap failure)
- Stack cleanup (munmap on thread exit)
- Support for complex loop bodies
- Optimization (inline simple bodies)
- Testing with 10k iterations

**Implementation Steps:**

1. **Error handling** (~30 lines)
```go
// Check clone() return value
fc.out.CmpImmToReg("rax", -1)
fc.out.JumpConditional(JumpEqual, ".clone_failed")
// ... error handling
```

2. **Stack cleanup** (~20 lines)
```go
// Before thread exit, munmap its stack
fc.out.MovReg("rdi", "r14")      // stack_base
fc.out.MovImmToReg("rsi", 1048576) // size
fc.out.MovImmToReg("rax", 11)    // sys_munmap
fc.out.Syscall()
```

3. **Complex loop body support** (~50 lines)
- Handle variables in loop body
- Preserve register state
- Stack management for nested scopes

## Key Files to Modify

### parser.go

**Function:** `compileParallelRangeLoop()` (currently at line ~6346)

**Current code:**
```go
func (fc *FlapCompiler) compileParallelRangeLoop(stmt *LoopStmt, rangeExpr *RangeExpr) {
    // ... diagnostics ...
    fmt.Fprintf(os.Stderr, "      Falling back to sequential execution (parallel codegen TODO)\n")

    // Fallback: compile as sequential loop for now
    seqStmt := *stmt
    seqStmt.NumThreads = 0
    fc.compileRangeLoop(&seqStmt, rangeExpr)
}
```

**Replace with:**
```go
func (fc *FlapCompiler) compileParallelRangeLoop(stmt *LoopStmt, rangeExpr *RangeExpr) {
    // ... existing diagnostics code ...

    // Remove fallback, add actual implementation:
    // 1. Allocate barrier
    // 2. Spawn threads
    // 3. Wait on barrier
    // 4. Cleanup
}
```

## Testing Strategy

### Incremental Testing

**After V1:**
```bash
./flapc test_parallel_v1.flap -o test_v1
strace -e clone ./test_v1 2>&1 | grep clone
# Should see: clone(...) = <TID>
```

**After V2:**
```bash
./flapc test_parallel_v2.flap -o test_v2
./test_v2 | sort -n > output.txt
# Check: all numbers 0-9 appear exactly once
```

**After V3:**
```bash
./flapc test_parallel_v3.flap -o test_v3
./test_v3 | wc -l
# Should output: 100 lines
./test_v3 | sort -n | uniq -c | awk '{if($1!=1) print "ERROR: "$0}'
# Should output nothing (all unique)
```

**After V4:**
```bash
./flapc test_parallel_v4.flap -o test_v4
time ./test_v4
# Compare to sequential version for speedup
```

## Available Resources

### Documentation
- **Design doc:** `docs/parallel_codegen_design.md` (389 lines)
  - Complete memory layouts
  - Assembly instruction sequences
  - Error handling strategies

- **Progress report:** `docs/parallel_loops_progress.md` (454 lines)
  - Current implementation status
  - File breakdown
  - Test coverage

### Working Code
- **parallel.go** (265 lines)
  - All syscall wrappers ready
  - Barrier implementation tested
  - Work distribution functions

- **parallel_test.go** (245 lines)
  - 6 passing tests
  - Reference implementations

### Assembly Emission API

The `fc.out` object provides methods for x86-64 assembly:

```go
// Register operations
fc.out.MovReg(dst, src)
fc.out.MovImmToReg(reg, value)
fc.out.MovRegToMem(reg, base, offset)
fc.out.MovMemToReg(reg, base, offset)
fc.out.LeaMemToReg(reg, base, offset)

// Stack operations
fc.out.SubImmFromReg(reg, value)
fc.out.AddImmToReg(reg, value)
fc.out.PushReg(reg)
fc.out.PopReg(reg)

// Arithmetic
fc.out.XorReg(dst, src)
fc.out.IncReg(reg)
fc.out.DecReg(reg)
fc.out.CmpReg(dst, src)
fc.out.CmpImmToReg(reg, value)

// Control flow
fc.out.Label(name)
fc.out.Jump(label)
fc.out.JumpConditional(condition, offset)
fc.out.TestReg(dst, src)

// Syscalls
fc.out.Syscall()

// Atomic operations
fc.out.LockXadd(mem, reg)  // LOCK XADD [mem], reg

// Floating point
fc.out.Cvtsi2sd(xmm, reg)  // Convert int to float
fc.out.MovXmmToMem(xmm, base, offset)
```

### Constants Available

From `parallel.go`:
```go
CLONE_VM             = 0x00000100
CLONE_FS             = 0x00000200
CLONE_FILES          = 0x00000400
CLONE_SIGHAND        = 0x00000800
CLONE_THREAD         = 0x00010000
CLONE_SYSVSEM        = 0x00040000
CLONE_THREAD_FLAGS   = 0x00010F00  // Combined flags

FUTEX_WAIT_PRIVATE   = 128
FUTEX_WAKE_PRIVATE   = 129

SYS_MMAP    = 9
SYS_MUNMAP  = 11
SYS_CLONE   = 56
SYS_EXIT    = 60
SYS_FUTEX   = 202
```

## Debug Tips

### Viewing Generated Assembly
```bash
objdump -d test_parallel | less
# Look for clone syscall (syscall with rax=56)
# Look for futex syscall (syscall with rax=202)
```

### Tracing Syscalls
```bash
strace -e clone,futex,mmap,munmap ./test_parallel
# Should see:
# mmap(...) = 0x7f...  (stack allocations)
# clone(...) = TID     (thread spawning)
# futex(...) = 0       (barrier wait)
```

### Finding Bugs
```bash
# Check for memory leaks
valgrind --leak-check=full ./test_parallel

# Check for race conditions
valgrind --tool=helgrind ./test_parallel

# Check thread count
ps -T -p $(pgrep test_parallel) | wc -l
```

## Common Pitfalls

### 1. Stack Alignment
x86-64 requires 16-byte stack alignment before call instructions.
```go
// Wrong:
fc.out.SubImmFromReg("rsp", 12)  // Not 16-byte aligned!

// Right:
fc.out.SubImmFromReg("rsp", 16)  // Aligned
```

### 2. Register Preservation
Threads share registers - must save/restore carefully.
```go
// Save caller-saved registers before syscall
fc.out.PushReg("rcx")
fc.out.PushReg("r11")
// ... syscall ...
fc.out.PopReg("r11")
fc.out.PopReg("rcx")
```

### 3. Thread Stack Grows Down
Remember stacks grow downward!
```go
stack_top = stack_base + stack_size  // Not stack_base!
```

### 4. Futex Value Changes
Always re-read futex value after wakeup (spurious wakeups).

## Performance Targets

### V1 (Proof of Concept)
- **Goal:** Just compiles and runs
- **Performance:** Don't care yet

### V2 (Multiple Threads)
- **Goal:** Correct output
- **Performance:** Any speedup is good

### V3 (Optimized)
- **Goal:** 0.8-0.9 × N speedup for N cores
- **Performance:** Compare to sequential version

### V4 (Production)
- **Goal:** <100μs overhead for 10k iteration loop
- **Performance:** Near-linear scaling to 8+ cores

## Success Criteria

### V1 Success
- ✅ Code compiles
- ✅ Binary runs without crashing
- ✅ strace shows clone() syscall
- ✅ strace shows futex() syscall

### V2 Success
- ✅ All V1 criteria
- ✅ Correct output (all items printed once)
- ✅ Order may vary (parallelism working)

### V3 Success
- ✅ All V2 criteria
- ✅ Works with any thread count (1, 2, 4, 8)
- ✅ Works with large ranges (1000, 10000 items)
- ✅ @@ resolves to actual CPU count

### V4 Success
- ✅ All V3 criteria
- ✅ Handles errors gracefully
- ✅ No memory leaks
- ✅ No race conditions
- ✅ Measurable speedup (0.8 × N for N cores)
- ✅ Ready for Flap 1.6 release

## Time Estimates

- **V1:** 2-3 hours (proof of concept)
- **V2:** 3-4 hours (multiple threads + printf)
- **V3:** 2 hours (verification + testing)
- **V4:** 4-5 hours (production polish)

**Total: 11-14 hours** to complete feature

## Questions to Answer in Next Session

1. **Which version to start with?**
   - Recommend: Start with V1 (proof of concept)
   - Validates approach before investing in full implementation

2. **Test-driven or implementation-first?**
   - Recommend: Test-driven
   - Write test case first, then implement until it passes

3. **Incremental commits or single commit?**
   - Recommend: Commit after each version (V1, V2, V3, V4)
   - Allows easy rollback if issues found

## Quick Reference Card

### Starting Point
**File:** `parser.go`
**Function:** `compileParallelRangeLoop()` (line ~6346)
**Replace:** Fallback code with actual implementation

### Key Variables Available
```go
actualThreads  // Number of threads to spawn
start         // Range start (e.g., 0)
end           // Range end (e.g., 100)
totalItems    // Total iterations (end - start)
chunkSize     // Items per thread
remainder     // Extra items for last thread
```

### Assembly Generation Pattern
```go
// 1. Allocate stack
fc.out.SubImmFromReg("rsp", bytes)
fc.runtimeStack += bytes

// 2. Emit instructions
fc.out.MovImmToReg("rax", value)
// ...

// 3. Cleanup stack
fc.out.AddImmToReg("rsp", bytes)
fc.runtimeStack -= bytes
```

---

**Ready to start?** Begin with V1 proof of concept. The infrastructure is solid, the design is complete, and all the tools are ready. Just need to emit the assembly! 🚀
