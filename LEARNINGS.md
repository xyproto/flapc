# Flapc Compiler Learnings

## Stack Alignment in x86-64

### The 16-byte Alignment Rule

The x86-64 System V ABI requires the stack pointer (rsp) to be aligned to 16 bytes **before** making any function call. This is critical when calling external functions like malloc, printf, etc.

### How to Calculate Stack Alignment

When a function is called, the CPU automatically pushes the return address (8 bytes). So at function entry, rsp is misaligned by 8 bytes.

Stack layout after various operations:
- After `call`: +8 bytes (misaligned - now at 8-byte boundary)
- After `push rbp`: +8 bytes (aligned - now at 16-byte boundary)
- After each `push`: +8 bytes per register

**Example calculation:**
```
call instruction         : +8  (total: 8,  misaligned)
push rbp (prologue)      : +8  (total: 16, aligned)
push r12                 : +8  (total: 24, misaligned)
push r13                 : +8  (total: 32, aligned)
push r14                 : +8  (total: 40, misaligned)
push r15                 : +8  (total: 48, aligned)
push rbx                 : +8  (total: 56, misaligned)
push rdi                 : +8  (total: 64, aligned)
```

Before calling malloc or any external function, count your stack usage. If it's misaligned (not a multiple of 16), subtract 8 more bytes from rsp.

### The Bug Pattern

In `flap_string_to_cstr` (parser.go line ~7520), we had:

```go
// BUGGY CODE (removed):
fc.out.SubImmFromReg("rsp", StackSlotSize)  // Sub 8
fc.out.MovXmmToMem("xmm0", "rsp", 0)
fc.out.MovMemToReg("r12", "rsp", 0)
fc.out.AddImmToReg("rsp", StackSlotSize)    // BUG: Added back too early!
```

At this point:
- call (8) + 6 pushes (48) = 56 bytes on stack
- 56 is not a multiple of 16 (misaligned!)
- The `sub rsp, 8` made it 64 bytes (aligned)
- But then we added it back before calling malloc
- malloc was called with misaligned stack → segfault or garbage data

**Fix:** Keep the stack aligned through the malloc call:

```go
// FIXED CODE:
fc.out.SubImmFromReg("rsp", StackSlotSize)  // Sub 8, now aligned
fc.out.MovXmmToMem("xmm0", "rsp", 0)
fc.out.MovMemToReg("r12", "rsp", 0)
// Keep rsp subtracted - restored later at line 7659
```

### General Principle

**Always verify stack alignment before calling external functions:**

1. Count bytes on stack: call(8) + pushes(8*N) + local_space
2. If total % 16 ≠ 0, subtract 8 more from rsp
3. Keep stack aligned until after the call returns
4. Restore rsp after the call completes

### Debugging Stack Alignment

If you see segfaults or garbage data from malloc/printf/etc:
1. Check stack alignment before the call
2. Use gdb: `info registers` and check rsp value
3. rsp & 0xF should equal 0 (bottom 4 bits zero)
4. Use ndisasm to verify generated assembly

### Impact

Incorrect stack alignment causes:
- Segmentation faults in external functions
- Garbage/corrupted return values
- Undefined behavior in SSE/AVX instructions (they require alignment)
- Intermittent bugs that are hard to reproduce

## Helper Function for Aligned malloc Calls

To make stack alignment easier and prevent bugs, we created a helper function:

```go
func (fc *FlapCompiler) callMallocAligned(sizeReg string, pushCount int)
```

**Parameters:**
- `sizeReg`: Register containing the allocation size (will be moved to rdi)
- `pushCount`: Number of registers pushed after the function prologue (not including `push rbp`)

**What it does:**
1. Calculates current stack usage: 16 + (8 * pushCount)
2. Checks if alignment is needed (total % 16 != 0)
3. Moves size to rdi (first argument for malloc)
4. Subtracts 8 from rsp if needed for alignment
5. Calls malloc
6. Restores rsp if it was adjusted
7. Returns allocated pointer in rax

**Usage example:**
```go
// Function with 5 register pushes after prologue
fc.out.PushReg("rbx")
fc.out.PushReg("r12")
fc.out.PushReg("r13")
fc.out.PushReg("r14")
fc.out.PushReg("r15")

// Allocate 512 bytes
fc.out.MovImmToReg("rax", "512")
fc.callMallocAligned("rax", 5) // 5 pushes
// Result is in rax
```

This replaces the manual alignment pattern:
```go
// OLD WAY (manual):
fc.out.SubImmFromReg("rsp", StackSlotSize)  // For alignment
fc.out.MovRegToReg("rdi", "rax")
fc.trackFunctionCall("malloc")
fc.eb.GenerateCallInstruction("malloc")
fc.out.AddImmToReg("rsp", StackSlotSize)  // Restore

// NEW WAY (helper):
fc.callMallocAligned("rax", pushCount)
```

The helper automatically handles alignment, making code clearer and preventing mistakes.
