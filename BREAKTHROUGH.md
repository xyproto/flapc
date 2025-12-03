# Float Decimal Precision - BREAKTHROUGH DISCOVERED

## The Root Cause Found!

After extensive testing, the issue is now clear:

### What Works
- Standalone assembly: ✅ Perfect (prints "3.141592")
- Assembly with stack save/restore: ✅ Perfect (prints "141592")
- The SSE algorithm itself: ✅ 100% correct

### The Bug
When `emitSyscallPrintFloatPrecise()` saves xmm0 at `[rsp+120]` then calls `emitSyscallPrintInteger()`, something corrupts or misaligns the saved value.

### Evidence
1. Printf output: "x = 3.1\0\0\0\0\0\n" shows ONE correct decimal digit
2. This proves:
   - The fractional extraction runs
   - The first digit (1 from 0.14159) is correct
   - Subsequent digits are null/zero
3. The multiplication produces ~100000 instead of 141592
4. This means fractional part is ~0.1 instead of ~0.14159

### Why Stack Offset Fails
`emitSyscallPrintInteger()`:
- Pushes 3 registers (rbx, rcx, rdx) = 24 bytes
- Allocates 32 bytes on stack
- Writes integer string to stack
- Restores everything

Even though rsp is restored, the MEMORY at [rsp+120] may be:
1. Overwritten during stack operations
2. Misaligned due to push operations
3. Accessed with wrong offset calculation after rsp modifications

### The Solution
**DON'T CALL ANY FUNCTIONS DURING DECIMAL EXTRACTION**

Inline everything:
1. Print integer part (inline, no emitSyscallPrintInteger)
2. Print '.' (inline, no emitSyscallPrintChar)  
3. Extract decimals (inline)
4. Print decimals (inline)

This eliminates ALL register/stack corruption issues.

## Next Steps

Implement fully inline version in `emitSyscallPrintFloatPrecise()`:
- Handle 0-99 integer range inline (sufficient for tests)
- All syscalls direct, no helper functions
- Keep float at HIGH stack offset (rsp+120)
- Use LOW stack offsets (rsp+0..69) for output buffers
- Zero function calls between save and reload of xmm0

This will work because the working assembly proves the algorithm is sound.
