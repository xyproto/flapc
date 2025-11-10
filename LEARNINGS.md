# Hard-Earned Learnings from Flapc Development

This document captures the most valuable lessons learned during the development of the Flap compiler, particularly around parallel loop implementation, register allocation, and assembly code generation.

## Parallel Loop Implementation

### Critical: Malloc in Pthreads

**Problem**: Calling `malloc()` from within pthread-created threads causes SIGSEGV deep in libc malloc implementation.

**Root Causes Identified**:
1. **Stack Alignment**: x86-64 ABI requires RSP to be 16-byte aligned before `call` instructions. The exact alignment before `call malloc` is critical.
2. **Thread-Local Storage (TLS)**: Glibc malloc uses thread-local arenas. Pthreads created without proper TLS setup can crash.
3. **Missing Thread Setup**: Glibc malloc expects certain thread-local state that may not be initialized in our pthread workers.

**Current Workarounds**:
- Pre-allocate all memory in the parent thread before starting parallel loop
- Pass pointers to pre-allocated memory via parent variables
- Avoid f-strings in parallel loops (they call malloc internally)
- Use simple variables and expressions only

**Future Solutions to Investigate**:
- Ensure 16-byte stack alignment before all `call` instructions
- Initialize TLS properly for pthread threads
- Consider using a pre-thread-local heap allocator
- Test with different malloc implementations (jemalloc, tcmalloc)

### Barrier Synchronization: Atomic Operations

**What Worked**:
```asm
; Atomic decrement and test
mov rax, -1
lock xadd [barrier], eax  ; eax gets old value
dec eax                    ; eax now has new value
test eax, eax
je last_thread            ; If 0, we're last
; Otherwise spin-wait
```

**Key Insights**:
- `lock xadd` is atomic and returns the old value
- Must test the *new* value (after decrement) to detect last thread
- Parent thread must participate in barrier (count = num_threads + 1)
- Spin-waiting is simple but CPU-intensive

**Futex Attempt Failed**:
- Attempted to use Linux futex for efficient waiting
- Race conditions in wake-up logic caused hangs
- Futex requires very careful state management
- Stick with spin-wait until futex implementation is proven correct

### Thread Function Prologue/Epilogue

**Critical for Stability**:
```asm
; Function prologue
push rbp
mov rbp, rsp
push rbx              ; Save callee-saved register
sub rsp, 64           ; Allocate stack space

; ... thread work ...

; Function epilogue
add rsp, 64           ; Restore stack
pop rbx               ; Restore callee-saved
pop rbp
ret                   ; Return to pthread
```

**Lessons**:
- Missing `push rbx` / `pop rbx` caused crashes when functions used rbx
- Stack allocation must be balanced with deallocation
- RBP-relative addressing requires proper frame setup
- Pthread expects a clean return (not exit)

### Parent Variable Access from Threads

**Working Approach**:
Pass `parent_rbp` as part of thread args structure:
```go
// Thread args layout (malloc'd 32 bytes)
[0-7]   start (int64)
[8-15]  end (int64)
[16-23] barrier_ptr (*int64)
[24-31] parent_rbp (uint64)
```

**In Thread Function**:
```asm
mov r11, [rbp-48]     ; Load parent_rbp into r11
mov rax, [r11-16]     ; Access parent variable at rbp-16
```

**Lesson**: Parent stack must remain valid for thread lifetime. Don't use local variables that might go out of scope.

## Register Allocation

### Caller-Saved vs Callee-Saved

**Caller-Saved** (RAX, RCX, RDX, RSI, RDI, R8-R11):
- Can be used freely within a function
- Must be saved before `call` if value is needed after
- Good for temporary values and loop counters

**Callee-Saved** (RBX, R12-R15):
- Must be saved on function entry and restored on exit
- Safe to use across function calls
- Good for variables that span multiple calls

**Hard Lesson**: Forgetting to save RBX before using it caused silent corruption when functions called other functions that also used RBX.

### Float vs Integer Registers

**Integer**: RAX, RBX, RCX, RDX, RSI, RDI, RBP, RSP, R8-R15
**Float**: XMM0-XMM15

**Conversions**:
```asm
; int64 to float64
cvtsi2sd xmm0, rax

; float64 to int64 (truncate)
cvttsd2si rax, xmm0
```

**Lesson**: Type conversions have a cost. Keep loop counters as int64 when possible, only convert to float64 when needed for iterator variable.

## Assembly Code Generation

### Stack Alignment Requirements

**x86-64 ABI Rule**: RSP must be 16-byte aligned before `call` instructions.

**Common Pattern**:
```asm
push rbp              ; RSP now 8-byte aligned (odd)
mov rbp, rsp
push rbx              ; RSP now 16-byte aligned (even)
sub rsp, 64           ; RSP stays 16-byte aligned
; ... work ...
call some_function    ; RSP is 16-byte aligned ✓
```

**Anti-Pattern**:
```asm
push rbp
mov rbp, rsp
sub rsp, 8            ; RSP now 8-byte aligned (odd)
call some_function    ; CRASH! RSP not 16-byte aligned ✗
```

**Lesson**: Always ensure even number of 8-byte pushes/allocations before calls.

### PLT (Procedure Linkage Table) Calls

**Problem**: Direct `call libc_function` doesn't work in statically-positioned code.

**Solution**: Use PLT stubs:
```asm
; In .text section
call malloc@PLT       ; Jump to PLT stub

; In .plt section (generated by linker)
malloc@PLT:
    jmp [malloc@GOT]  ; Indirect jump through GOT
```

**Flapc Implementation**:
1. Generate placeholder: `call 0x12345678`
2. Collect call sites in `callPatches`
3. After code generation, patch with correct PLT offsets

**Lesson**: Can't know final addresses until linking. Use placeholders and patch.

### Memory Operand Sizes

**Explicit Size Required**:
```asm
mov rax, [rbp-8]      ; ✓ Register size implies qword
mov [rbp-8], rax      ; ✓ Register size implies qword
mov [rbp-8], 42       ; ✗ Ambiguous! Is it byte/word/dword/qword?
mov qword [rbp-8], 42 ; ✓ Explicit size
```

**Lesson**: When source is immediate and destination is memory, must specify size.

## Debugging Strategies

### What Worked

**1. Minimal Test Cases**:
Create the smallest possible program that reproduces the bug:
```flap
@@ i in 0..<2 {
    printf("%ld\n", i)
}
```
Easier to debug than complex parallel algorithms.

**2. GDB with Assembly**:
```bash
gdb ./test_program
(gdb) layout asm
(gdb) break *0x403123
(gdb) stepi
(gdb) info registers
(gdb) x/10gx $rsp
```
Step through assembly instruction-by-instruction, watch registers and stack.

**3. Objdump Inspection**:
```bash
objdump -d ./test_program | less
```
Verify generated assembly matches expectations. Look for:
- Correct function prologues/epilogues
- Proper stack alignment
- Valid instruction encodings
- Correct relative offsets for calls

**4. Verbose Debug Logging**:
Add extensive debug output during code generation:
```go
fmt.Fprintf(os.Stderr, "DEBUG: Patching call at 0x%x, offset=%d\n", pos, offset)
```
Critical for understanding what the compiler is doing.

**5. Compare Working vs Broken**:
When one test passes and another fails, diff the assembly:
```bash
diff -u working.s broken.s
```
Often reveals the exact instruction or pattern causing issues.

### What Didn't Work

**1. Print Debugging from Threads**:
Printf from multiple threads can interleave output, making it useless. Use GDB instead.

**2. Trusting Compiler Optimizations**:
When debugging, compile with `-O0` to prevent optimizations from obscuring issues.

**3. Assuming C ABI Compliance**:
Hand-written assembly must follow all ABI rules exactly. No shortcuts.

## Performance Lessons

### Parallel Speedup Reality

**Amdahl's Law in Practice**:
- Achieved ~3.5x speedup on 4 cores (not 4x)
- Thread creation overhead is ~0.1ms per thread
- Barrier synchronization overhead is ~0.01ms
- Work distribution overhead is ~0.05ms

**Sweet Spot**: Parallel loops are beneficial when:
- Loop body takes >1µs per iteration
- Total work is >10,000 iterations
- Work is evenly distributed across iterations

### Memory Access Patterns

**Cache Line Sharing**:
```flap
// BAD: False sharing
var counters = [0.0, 0.0, 0.0, 0.0]
@@ i in 0..<1000 {
    counters[thread_id()] += 1.0  // All threads hit same cache line
}

// GOOD: Separate cache lines
var counters = [0.0, 0.0, ..., 0.0]  // 64 elements (512 bytes)
@@ i in 0..<1000 {
    counters[thread_id() * 16] += 1.0  // 128 bytes apart
}
```

**Lesson**: Align thread-local data to cache line boundaries (64 bytes) to avoid false sharing.

### Spin-Wait vs Sleep

**Current**: Spin-wait in barrier
```asm
.wait_loop:
    mov rax, [barrier]
    test rax, rax
    je .done
    jmp .wait_loop  ; Busy loop, burns CPU
```

**Better**: Use futex (Linux) or condition variable (pthread)
- Yields CPU while waiting
- Wakes on notification
- Much lower CPU usage

**Tradeoff**: Futex adds ~1µs latency vs spin-wait's ~0.01µs

## Code Quality Lessons

### When to Refactor

**Good Time**:
- After tests pass
- Before adding new features
- When code becomes hard to reason about

**Bad Time**:
- During debugging (keep changes minimal)
- When tests are failing
- Under time pressure

**Lesson**: Get it working first, then make it clean.

### Testing Philosophy

**What We Learned**:
1. **Integration tests are crucial**: Unit tests alone don't catch ABI violations or threading issues
2. **Test both success and failure**: Test that errors are caught correctly
3. **Test edge cases**: Empty loops, single iteration, huge iteration counts
4. **Test combinations**: Multiple parallel loops, nested loops, etc.

**Test Pattern That Works**:
```go
// 1. Minimal case
// 2. Add one feature
// 3. Add another feature
// 4. Combine features
```

### Documentation Lessons

**Keep It Actionable**:
- TODOs should be concrete, not vague ("Fix threading" vs "Investigate stack alignment before malloc calls")
- Link to relevant code sections
- Include time estimates if possible

**Keep It Current**:
- Delete or archive outdated docs
- Update status when milestones reached
- Don't let docs accumulate

**This Document**:
- Focus on "why" and "how", not just "what"
- Include failed approaches, not just successes
- Make it searchable (specific error messages, function names)

## Summary of Key Principles

1. **Follow the ABI religiously**: Stack alignment, register usage, calling conventions
2. **Start simple, add complexity gradually**: Get basic case working before adding features
3. **Debug with minimal test cases**: Simplify until the bug is obvious
4. **Measure before optimizing**: Profile, don't guess
5. **Test edge cases early**: They're where bugs hide
6. **Document failures**: Failed approaches teach as much as successes
7. **Trust the hardware, verify the software**: x86-64 is battle-tested, your code isn't
8. **When in doubt, check the disassembly**: The assembly never lies

## References

Useful resources that helped during development:

- **x86-64 ABI**: https://refspecs.linuxbase.org/elf/x86_64-abi-0.99.pdf
- **Intel Manual**: https://www.intel.com/content/www/us/en/developer/articles/technical/intel-sdm.html
- **Linux Futex**: `man 2 futex`
- **Pthread**: `man 7 pthreads`
- **GDB Manual**: https://sourceware.org/gdb/current/onlinedocs/gdb/
- **Godbolt Compiler Explorer**: https://godbolt.org/ (invaluable for comparing compiler output)
