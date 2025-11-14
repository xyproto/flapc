# String Printing Implementation Guide

## Current Status
**Problem:** `println(string_variable)` prints "0.000000" instead of string content

**Location:** `codegen.go` lines 10091-10105

## Solution: Use Write Syscall

### Algorithm
1. Get string pointer from xmm0
2. Read length from [string_ptr + 0]
3. For each index i from 0 to length-1:
   - Calculate offset: 16 + i*16 (skips count and keys)
   - Load character code as float64
   - Convert to integer
   - Use write(1, &char, 1) syscall
4. Print '\n' with write syscall

### Implementation Pseudocode
```go
// After fc.compileExpression(arg) which puts string pointer in xmm0:

// Convert xmm0 to integer pointer in rbx
MovXmmToMem(xmm0, stack)
MovMemToReg(rbx, stack)

// Get length
MovMemToXmm(xmm0, [rbx])
Cvttsd2si(r12, xmm0)  // r12 = length

// Allocate 1-byte buffer on stack
SubImmFromReg(rsp, 8)
MovRegToReg(r13, rsp)  // r13 = buffer address

// Loop: for rcx = 0; rcx < r12; rcx++
XorRegWithReg(rcx, rcx)
loop_start:
  CmpRegToReg(rcx, r12)
  JumpConditional(GreaterOrEqual, loop_end)
  
  // Get character: offset = 16 + rcx*16
  MovRegToReg(rax, rcx)
  ShlImmReg(rax, 4)         // rax = rcx * 16
  AddImmToReg(rax, 16)       // rax = 16 + rcx*16
  AddRegToReg(rax, rbx)      // rax = string_ptr + offset
  
  // Load and convert char
  MovMemToXmm(xmm0, [rax])
  Cvttsd2si(rdi, xmm0)
  MovRegToMem(rdi, [r13])
  
  // write(1, r13, 1)
  MovImmToReg(rax, 1)   // syscall number
  MovImmToReg(rdi, 1)   // stdout
  MovRegToReg(rsi, r13) // buffer
  MovImmToReg(rdx, 1)   // length
  Syscall()
  
  IncReg(rcx)
  JumpUnconditional(loop_start)

loop_end:
// Print newline
MovImmToReg(rax, 10)     // '\n'
MovRegToMem(rax, [r13])
MovImmToReg(rax, 1)      // write syscall
MovImmToReg(rdi, 1)      // stdout
MovRegToReg(rsi, r13)    // buffer
MovImmToReg(rdx, 1)      // length
Syscall()

// Cleanup
AddImmToReg(rsp, 8)
```

### Benefits
- No PLT dependencies
- Direct system calls
- Works for all string sizes including empty strings
- Minimal overhead

### Testing
```flap
// Test 1: Basic string
msg := "Test"
println(msg)  // Should output: Test

// Test 2: Empty string
empty := ""
println(empty)  // Should output: (newline)

// Test 3: String concatenation
result := "Hello" + " " + "World"
println(result)  // Should output: Hello World
```

## Expected Test Results
After implementation:
- TestStringOperations/string_variable ✅
- TestStringOperations/empty_string ✅  
- TestStringOperations/string_concatenation ✅
- TestBasicPrograms/fstring_basic ✅

Total: **129/130 passing (99.2%)**
