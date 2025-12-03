# Float Decimal Fix - 5 Step Plan

## Problem
The assembly in `asm/float_decimal.asm` works perfectly, but the compiler version fails.

## Root Cause Analysis
The working assembly does this:
```asm
movsd xmm0, [test_value]    ; Load once
cvttsd2si rax, xmm0         ; Extract integer
; ... print integer part ...
movsd xmm0, [test_value]    ; RELOAD from memory
cvttsd2si rax, xmm0         ; Extract integer again
cvtsi2sd xmm1, rax          ; Convert to double
subsd xmm0, xmm1            ; Get fraction
```

The compiler version does this:
```go
MovXmmToMem("xmm0", "rsp", 48)     // Save
MovMemToXmm("xmm0", "rsp", 48)     // Load
emitSyscallPrintInteger()           // CLOBBERS xmm0!
MovMemToXmm("xmm0", "rsp", 48)     // Reload (but xmm0 was modified)
```

**Key Issue**: `emitSyscallPrintInteger()` might be modifying xmm0 or stack memory.

## 5 Step Fix Plan

### Step 1: Verify the Problem
Create minimal test that isolates the float printing without calling other functions.

### Step 2: Eliminate Function Calls
Don't call `emitSyscallPrintInteger()` or `emitSyscallPrintChar()` that might clobber registers.
Emit ALL code inline without any function calls.

### Step 3: Use Separate Buffer Regions
Use stack offsets 0-15 for digits, 48-55 for saved float, 32-39 for integer part.
No overlap.

### Step 4: Copy Working Assembly Byte-for-Byte
Emit the exact same instruction sequence as the working assembly, just with
stack addresses instead of data section addresses.

### Step 5: Test and Verify
Test with multiple values (3.14, 10/3, 0.5) to ensure it works.

## Implementation Strategy

Create a completely self-contained function that:
1. Saves xmm0 to stack
2. Prints integer part inline (no function calls)
3. Prints '.' inline
4. Extracts decimals inline
5. Prints decimals inline
6. Everything inline, zero function calls except final syscall

This eliminates ALL register clobbering issues.
