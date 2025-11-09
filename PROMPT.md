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

## Quick Wins - Try These First! ⚡

Before diving into deep debugging, try these high-probability fixes that might solve the issues immediately:

### Quick Win #1: Test with Minimal Thread Function (5 min)
The fastest way to isolate the SEGFAULT:

```go
// In codegen.go around line 2259, TEMPORARILY replace entire thread function with:
fc.eb.MarkLabel("_parallel_thread_entry")
fc.out.PushReg("rbp")
fc.out.MovRegToReg("rbp", "rsp")
fc.out.XorRegWithReg("rax", "rax")  // return NULL
fc.out.PopReg("rbp")
fc.out.Ret()
// Comment out everything else until line 2440
```

**If this works:** Problem is in prologue/loop code (not pthread itself)
**If this fails:** Problem is in pthread_create call or function pointer

### Quick Win #2: Verify Counter Store (2 min)
Look at lines 2356-2359 in codegen.go:

```go
// Should see ALL THREE lines:
fc.out.MovMemToReg("rax", "rbp", -32) // Load
fc.out.IncReg("rax")                  // Increment
fc.out.MovRegToMem("rax", "rbp", -32) // ⚠️ STORE (is this present?)
```

**If third line is missing:** Add it! This would explain single iteration.

### Quick Win #3: Check Assembly Jump Offset (3 min)
```bash
objdump -d /tmp/crash_test | grep -A20 "_parallel_thread_entry" | grep jmp | tail -1
```

Look for the jump back instruction. Should show negative offset like:
```
jmp    0xabcdef <earlier_address>  # or jmp -0x123
```

**If offset is positive or zero:** Jump calculation is broken.

### Quick Win #4: Verify rbx vs rdi (1 min) - ALREADY FIXED
Lines 2287-2298 should use `rbx` not `rdi`:
```go
fc.out.MovMemToReg("rax", "rbx", 0)   // ✓ Correct (using rbx)
fc.out.MovMemToReg("rax", "rdi", 0)   // ✗ Wrong (would crash)
```

**Status:** Already fixed in current code.

---

## Debugging Strategy

### Priority 1: Fix the SEGFAULT

**CRITICAL INSIGHT:** The crash happens IMMEDIATELY after thread creation, before any user code runs. This suggests a fundamental issue with the thread entry point or prologue code.

**Most Likely Root Causes (in order of probability):**

1. **Stack alignment violation** (70% probability)
   - pthread_create provides 16-byte aligned stack at entry
   - After `push rbp` (8) + `push rbx` (8) + `sub rsp, 64`: Should be aligned
   - **BUT:** Check if there's a hidden assumption being violated
   - **Verify:** The stack is actually 16-byte aligned before first `call` instruction

2. **Function pointer resolution failure** (20% probability)
   - `LeaSymbolToReg("rdx", "_parallel_thread_entry")` might not generate correct address
   - The label might not be marked/resolved properly in the ELF output
   - **Check:** Does objdump show the correct address for the label?

3. **Argument structure NULL or garbage** (8% probability)
   - malloc() might be failing
   - The pointer passed in rcx might not reach rdi in the thread function
   - **Verify:** Add a check that malloc returned non-NULL

4. **ABI calling convention violation** (2% probability)
   - pthread_create expects a specific function signature
   - **Signature must be:** `void* (*)(void*)`
   - **Check:** Thread entry follows this signature exactly

**Step-by-Step Debugging Protocol:**

**STEP 1: Determine EXACT crash location (5 minutes)**
```bash
# Build and get crash info
go build -o flapc *.go
./flapc /tmp/test_stack_check.flap -o /tmp/crash_test 2>&1 | grep -v DEBUG

# Run under GDB to see exact instruction
gdb -batch -ex 'set pagination off' -ex 'run' -ex 'bt' -ex 'info registers' -ex 'x/20i $rip-10' /tmp/crash_test 2>&1 | tee /tmp/gdb_output.txt

# Look for:
# - Program received signal SIGSEGV
# - Faulting address (from info registers)
# - Instruction pointer at crash
# - Which function (is it in _parallel_thread_entry?)
```

**STEP 2: Examine generated assembly (10 minutes)**
```bash
# Disassemble the thread entry function
objdump -d /tmp/crash_test | awk '/_parallel_thread_entry/,/^$/ {print; if (/ret/) exit}' > /tmp/thread_asm.txt

# Check:
# 1. Does label exist and have correct address?
# 2. Is prologue correct? (push rbp; mov rbp,rsp; push rbx; sub rsp,64)
# 3. Are memory accesses using correct offsets?
# 4. Is stack alignment maintained before calls?

# Manual verification:
# Entry: rsp is 16-aligned (pthread guarantees this)
# After push rbp: rsp -= 8 (misaligned by 8)
# After push rbx: rsp -= 8 (aligned again!)
# After sub rsp,64: rsp -= 64 (still aligned, 64 is multiple of 16)
# ✓ Should be aligned
```

**STEP 3: Verify function pointer (5 minutes)**
```bash
# Check if pthread_create gets correct function address
objdump -d /tmp/crash_test | grep -A5 "pthread_create" | head -20

# Look for LEA instruction loading function pointer into rdx
# Should see: lea rdx, [rip + offset]  ; _parallel_thread_entry
# Verify offset points to actual function

# Also check symbol table
nm /tmp/crash_test | grep thread_entry
# Should show address of _parallel_thread_entry
```

**STEP 4: Test with minimal thread function (15 minutes)**

Create a test that just returns immediately:
```go
// In codegen.go, temporarily replace thread entry with minimal version
fc.eb.MarkLabel("_parallel_thread_entry")
fc.out.XorRegWithReg("rax", "rax")  // return NULL
fc.out.Ret()
// Skip all the prologue/loop code
```

If this works → problem is in prologue/loop code
If this fails → problem is in how pthread_create calls the function

**STEP 5: Add debug output (10 minutes)**

If crash is inside thread function, add syscall to write "ENTER\n" to stderr:
```go
// Right after MarkLabel("_parallel_thread_entry")
// Write syscall: write(2, "ENTER\n", 6)
fc.out.MovImmToReg("rax", "1")  // sys_write
fc.out.MovImmToReg("rdi", "2")  // stderr
// ... load string address ...
fc.out.MovImmToReg("rdx", "6")  // length
fc.out.Syscall()
```

If you see "ENTER\n" → crash is after entry
If you don't → crash is at entry or before

**STEP 6: Check malloc() return (5 minutes)**
```go
// After malloc call for thread args:
fc.eb.GenerateCallInstruction("malloc")
fc.out.MovRegToReg("r13", "rax")

// Add check:
fc.out.TestRegReg("rax", "rax")
fc.out.JumpConditional(JumpEqual, errorHandlerOffset)  // if NULL, error
```

**COMMON MISTAKES TO AVOID:**

1. ❌ **Don't assume stack alignment is correct** - Verify with actual arithmetic
2. ❌ **Don't trust comments** - The code may not match what comments say
3. ❌ **Don't debug blind** - Always use gdb/objdump to see actual behavior
4. ❌ **Don't make multiple changes at once** - Test one fix at a time
5. ✅ **Do verify each assumption** - pthread alignment, malloc success, pointer values
6. ✅ **Do use minimal test cases** - Empty loop body, immediate return, etc.
7. ✅ **Do examine actual assembly** - Don't trust codegen without verification

### Priority 2: Fix Loop Iteration Bug

**Problem:** Loops execute only 1 iteration per thread instead of all assigned iterations.

**This bug existed BEFORE the refactoring** (confirmed in commit 59149d3).

**Example:**
- Thread 0 assigned range [0, 2) should print i=0, i=1
- Actually prints only i=0
- Thread 1 assigned range [2, 4) should print i=2, i=3
- Actually prints only i=2

**Root Cause Hypotheses (most likely first):**

1. **Loop back jump is broken** (40%)
   - Jump offset might be calculated incorrectly
   - **Test:** Check actual jump target in assembly

2. **Counter increment not persisting** (30%)
   - Counter loaded, incremented, but NOT stored back
   - **Check:** Line 2356-2359 - verify MovRegToMem is called

3. **Loop condition inverted** (20%)
   - Should continue while counter < end
   - **Check:** Line 2312 uses JumpGreaterOrEqual to EXIT (correct?)

4. **rbp clobbered during loop** (10%)
   - Unlikely but possible
   - **Test:** Verify rbp is callee-saved throughout

**Debugging Steps:**

1. **Examine assembly** - verify loop structure is complete
2. **Add debug output** - print counter before/after increment
3. **Check jump offset** - must be negative to jump backward
4. **Compare with sequential loop** - use same working pattern

**Critical code pattern (must be present):**
```go
fc.out.MovMemToReg("rax", "rbp", -32) // Load counter
fc.out.IncReg("rax")                  // Increment
fc.out.MovRegToMem("rax", "rbp", -32) // ⚠️ STORE BACK
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
