# Session Continuation: Fix Parallel Loop Threading Issues

## Context

You are working on the **flapc compiler** - a compiler for the Flap programming language that generates x86-64 ELF binaries. The compiler has parallel loop functionality using the `@@` syntax (e.g., `@@ i in 0..<4 { ... }`).

## Current Status

**What Works:**
- Parallel loops compile and execute
- Threads are created successfully
- Basic infrastructure (barrier synchronization, pthread integration) is in place

**Critical Issues:**
1. **SEGFAULT in thread execution** - Threads crash immediately after being created by pthread_create
2. **Loop iteration bug** - Even when not crashing, loops only execute ONE iteration per thread instead of all assigned iterations
3. **Barrier synchronization hang** - Program doesn't exit cleanly after loop completion

## Recent Work (This Commit)

I refactored the parallel loop implementation to fix register clobbering issues:

### What Changed:
- **Before:** Used `clone()` syscall with hardcoded registers (r12=start, r13=end, r14=counter)
- **After:** Switched to `pthread_create()` with stack-based loop control using rbp-relative addressing

### Refactoring Details:
```
Thread function stack layout (rbp-relative):
  [rbp-8]:  saved rbx
  [rbp-16]: start value
  [rbp-24]: end value
  [rbp-32]: loop counter
  [rbp-40]: barrier_ptr
  [rbp-48]: parent_rbp (for accessing parent variables)
  [rbp-56]: iterator value (float64)

Stack allocation: 64 bytes for proper 16-byte alignment
```

### Bug Fixed During Refactoring:
- Line 2287-2298 in codegen.go: Was using `rdi` to read argument structure after saving it to `rbx` - fixed to use `rbx`

### Current Crash:
- Threads created successfully via pthread_create
- SIGSEGV occurs immediately in thread entry function
- Crash location: Unknown (needs debugging)

## Key Files

**codegen.go** (lines 2166-2450):
- `compileParallelRangeLoop()` - Main parallel loop compilation
- Line 2259: Thread entry function `_parallel_thread_entry`
- Lines 2261-2300: Thread prologue and parameter extraction
- Lines 2302-2360: Loop body compilation
- Lines 2373-2427: Barrier synchronization

**Test Files:**
- `testprograms/parallel_no_atomic.flap` - Main test case
- Expected behavior: Print "Thread iteration: 0", "Thread iteration: 1", "Thread iteration: 2", "Thread iteration: 3"
- Actual behavior: Crashes with SIGSEGV before any output

## Architecture Notes

**x86-64 ABI Requirements:**
- Stack must be 16-byte aligned before `call` instructions
- After `push rbp` (8) + `push rbx` (8) = 16 bytes, we're aligned
- After `sub rsp, 64`, we're still aligned (64 is multiple of 16)
- Callee-saved registers: rbx, rbp, r12-r15 (must be preserved)
- Caller-saved registers: rax, rcx, rdx, rsi, rdi, r8-r11 (can be clobbered)

**pthread_create signature:**
```c
int pthread_create(pthread_t *thread, const pthread_attr_t *attr,
                   void *(*start_routine)(void*), void *arg);
// Parameters: rdi=thread, rsi=attr, rdx=start_routine, rcx=arg
```

**Thread argument structure (malloc'd, passed via rcx to pthread_create):**
```
Offset 0:  start (int64)
Offset 8:  end (int64)
Offset 16: barrier_ptr (int64)
Offset 24: parent_rbp (int64)
```

## Debugging Strategy

### Priority 1: Fix the SEGFAULT

**Likely causes:**
1. **LeaSymbolToReg issue** - Function pointer for `_parallel_thread_entry` may not be resolved correctly
2. **Stack alignment problem** - Despite calculations, alignment might be off
3. **NULL pointer dereference** - Accessing invalid memory during parameter extraction
4. **Calling convention mismatch** - pthread_create might not be calling the function correctly

**Debugging steps:**
1. Use `objdump -d` to examine the generated thread entry function assembly
2. Use `gdb` to see the exact crash instruction
3. Add minimal debug output at thread entry (e.g., write to stderr)
4. Verify `LeaSymbolToReg("rdx", "_parallel_thread_entry")` generates correct code
5. Test with empty loop body to isolate crash location

### Priority 2: Fix Loop Iteration Bug

**Problem:** Loops execute only 1 iteration per thread instead of all assigned iterations.

**This bug existed BEFORE the refactoring** (confirmed in commit 59149d3).

**Example:**
- Thread 0 assigned range [0, 2) should print i=0, i=1
- Actually prints only i=0
- Thread 1 assigned range [2, 4) should print i=2, i=3
- Actually prints only i=2

**Likely causes:**
1. Loop counter increment not working (line 2356-2359)
2. Loop condition check incorrect (lines 2305-2312)
3. Jump back to loop start not working (lines 2361-2364)
4. Counter value being clobbered during loop body execution

**Key code to review:**
```go
// codegen.go lines 2305-2364
// Loop structure:
1. Load counter and end from stack
2. Compare counter >= end, exit if true
3. Execute loop body
4. Increment counter
5. Jump back to step 1
```

### Priority 3: Fix Barrier Synchronization Hang

**Current behavior:** Program hangs after loop execution (if it gets that far).

**Barrier implementation** (lines 2373-2427):
- Uses futex syscalls for synchronization
- Last thread to finish wakes all waiting threads
- May have race condition or incorrect futex usage

## How to Approach This

### Option A: Debug Current Implementation (Recommended)
1. Find and fix the SEGFAULT bug
2. Once threads run, fix the loop iteration bug
3. Fix barrier synchronization
4. Run tests to verify

### Option B: Revert and Take Different Approach
1. Git revert to commit 59149d3 (working baseline that prints one iteration per thread)
2. Fix the original issue differently:
   - Keep using `clone()` syscall
   - Fix register clobbering by saving/restoring r12-r14 around loop body compilation
   - Or use register allocator to allocate non-conflicting registers

### Option C: Hybrid Approach
1. Keep pthread_create() (better than clone)
2. Simplify stack layout
3. Use fewer variables
4. Avoid rbp-relative addressing complexity

## Test Commands

```bash
# Build compiler
go build -o flapc *.go

# Test simple parallel loop
./flapc /tmp/test_stack_check.flap -o /tmp/test 2>&1 | grep -v "^DEBUG"
timeout 2 /tmp/test

# Test with gdb
echo 'run' | gdb -batch -x /dev/stdin /tmp/test 2>&1 | grep -A 10 "Program received"

# Examine generated assembly
objdump -d /tmp/test | awk '/_parallel_thread_entry/,/ret.*$/'

# Run full test suite
go test -v -run "TestFlapPrograms/(parallel_no_atomic|parallel_atomic_minimal)$" -count=1
```

## Success Criteria

1. **No crashes** - Threads execute without SIGSEGV
2. **All iterations execute** - Each thread prints all its assigned iterations
3. **Clean exit** - Program completes and exits without hanging
4. **Tests pass** - parallel_no_atomic.flap test passes

## Questions to Answer

1. Where exactly does the SEGFAULT occur? (instruction address, register state)
2. Is the function pointer being passed correctly to pthread_create?
3. Is rbp-relative addressing working as expected when rsp changes?
4. Why does the loop only execute once per thread?
5. Is the barrier synchronization logic correct?

## Additional Context

- **Register allocator exists** (register_allocator.go) but wasn't used
- Original user request: "use the register allocator"
- Simpler stack-based approach may be more practical for this specific case
- All background bash jobs from previous session are still running (may have useful output)

## File Locations

- Main implementation: `/home/alexander/clones/flapc/codegen.go`
- Test program: `/home/alexander/clones/flapc/testprograms/parallel_no_atomic.flap`
- Working directory: `/home/alexander/clones/flapc`
- Git status: On branch main, 3 commits ahead of origin

Good luck! Focus on getting threads to execute without crashing first, then tackle the iteration bug.
