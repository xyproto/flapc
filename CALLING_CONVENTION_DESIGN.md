# Calling Convention Design for Flapc

## Current Issues

### Problem: Lambda Stack Corruption
When lambdas call external functions (especially printf), the stack frame gets corrupted.

**Root Cause**: The current lambda calling convention doesn't properly maintain stack alignment and frame integrity when making nested function calls.

### Current Lambda Convention (BROKEN)
```
Lambda entry:
  push rbp
  mov rbp, rsp
  push rbx              # Save callee-saved
  sub rsp, N*16         # Allocate params + locals
  # Parameters in xmm0, xmm1, xmm2...
  # Store params to [rbp-16], [rbp-32], etc.
  # Environment pointer in r15
```

**Issue**: When printf is called:
1. System V ABI requires 16-byte stack alignment before `call`
2. Parameters go in rdi, rsi, rdx, rcx, r8, r9 (integers) and xmm0-xmm7 (floats)
3. The call instruction pushes return address (8 bytes)
4. If rsp isn't properly aligned, everything breaks
5. rbp might get clobbered by external functions

## Recommended Solutions

### Option 1: Red Zone Usage (BEST for x86-64 Linux)
The System V ABI defines a 128-byte "red zone" below rsp that leaf functions can use without adjusting rsp.

**Implementation**:
```
Lambda entry:
  push rbp
  mov rbp, rsp
  # DON'T push rbx yet
  # DON'T adjust rsp for locals yet
  
  # Store params in red zone temporarily:
  mov [rsp-8], xmm0     # param 0
  mov [rsp-16], xmm1    # param 1
  mov [rsp-24], xmm2    # param 2
  
  # Now properly allocate stack (maintaining 16-byte alignment):
  sub rsp, aligned(N*16 + 8)  # +8 for rbx, round to 16
  push rbx              # Now aligned
  
  # Copy params from red zone to proper locations:
  mov xmm0, [rbp-8]
  mov [rbp-16], xmm0
  # etc.
```

**Pros**: 
- Uses CPU-provided feature
- Fast (no extra stack ops)
- Guaranteed to work with external calls

**Cons**:
- Only available on x86-64 Linux/Unix
- Macros would need separate handling

### Option 2: Fixed Frame Layout (RECOMMENDED - Works Everywhere)
Use a predictable, System V compliant frame layout.

**Implementation**:
```
Lambda prologue:
  push rbp
  mov rbp, rsp
  
  # Calculate total stack needed:
  # - Params: N * 16 bytes
  # - Locals: M * 16 bytes  
  # - Saved registers: 8 bytes (rbx)
  # - Alignment padding
  # Total = align_to_16(N*16 + M*16 + 8)
  
  sub rsp, TOTAL        # Allocate all at once
  mov [rbp-8], rbx      # Save rbx at known location
  
  # Store params at predictable offsets:
  mov [rbp-24], xmm0    # param 0 at rbp-24
  mov [rbp-40], xmm1    # param 1 at rbp-40
  # Each param at rbp-(16 + param_index*16 + 8)
  
Lambda epilogue:
  mov rbx, [rbp-8]      # Restore rbx
  mov rsp, rbp          # Deallocate
  pop rbp
  ret
```

**Frame layout**:
```
[rbp+0]   = saved rbp (pushed by caller)
[rbp-8]   = saved rbx
[rbp-16]  = alignment padding (if needed)
[rbp-24]  = param 0
[rbp-40]  = param 1
[rbp-56]  = param 2
[rbp-72]  = local 0
[rbp-88]  = local 1
...
[rsp]     = current stack top (16-byte aligned)
```

**Function calls from lambda**:
```
Before call:
  # Save any live values
  # Load arguments into proper registers (rdi, rsi, rdx, xmm0, etc.)
  # Ensure rsp is 16-byte aligned (it should be from prologue)
  call function
  # Result in rax/xmm0
```

**Pros**:
- Works on all platforms
- System V ABI compliant
- Simple and predictable
- External functions can't corrupt our frame

**Cons**:
- Slightly more stack usage
- Fixed overhead per lambda

### Option 3: Separate Shadow Space (Windows-style)
Allocate shadow space for function calls.

**Implementation**:
```
Lambda prologue:
  push rbp
  mov rbp, rsp
  sub rsp, 32           # Shadow space for calls
  sub rsp, N*16         # Params + locals
  push rbx
  
Function call:
  # Arguments already in registers
  # Shadow space available at [rsp]
  call function
```

**Pros**:
- Compatible with Windows x64 calling convention
- Explicit separation of concerns

**Cons**:
- Wastes 32 bytes per lambda
- Not idiomatic for Linux

## Recommended Implementation: Option 2

### Why Option 2 is Best:
1. **Platform independent** - Works on Linux, macOS, Windows
2. **ABI compliant** - Follows System V / Windows x64 conventions
3. **Debuggable** - Clear frame layout for debuggers
4. **Safe** - External functions can't corrupt our data
5. **Simple** - Easy to implement and maintain

### Implementation Steps:

#### Step 1: Fix Lambda Prologue
```go
// In generateLambdaFunctions():

// Calculate stack frame size
paramCount := len(lambda.Params)
capturedCount := len(lambda.CapturedVars)
localCount := 0  // Will be determined during collection phase

// Stack layout:
// [rbp-8] = saved rbx
// [rbp-16] = alignment padding if needed
// [rbp-24], [rbp-40], ... = parameters (16 bytes each)
// After params = captured vars
// After captured = locals

frameSize := 8 + paramCount*16 + capturedCount*16
// Round up to 16-byte alignment
frameSize = (frameSize + 15) & ^15

// Prologue
fc.out.PushReg("rbp")
fc.out.MovRegToReg("rbp", "rsp")
fc.out.SubImmFromReg("rsp", int64(frameSize))
fc.out.MovRegToMem("rbx", "rbp", -8)  // Save rbx

// Store parameters at fixed offsets
baseOffset := 24  // First param at rbp-24
for i, paramName := range lambda.Params {
    offset := baseOffset + i*16
    fc.variables[paramName] = offset
    fc.out.MovXmmToMem(xmmRegs[i], "rbp", -offset)
}

// Store captured vars
captureBase := baseOffset + paramCount*16
for i, capturedVar := range lambda.CapturedVars {
    offset := captureBase + i*16
    fc.variables[capturedVar] = offset
    fc.out.MovMemToXmm("xmm15", "r15", i*8)
    fc.out.MovXmmToMem("xmm15", "rbp", -offset)
}
```

#### Step 2: Fix Epilogue
```go
// Epilogue - already mostly correct
fc.out.MovMemToReg("rbx", "rbp", -8)  // Restore rbx
fc.out.MovRegToReg("rsp", "rbp")       // Deallocate
fc.out.PopReg("rbp")
fc.out.Ret()
```

#### Step 3: Ensure Function Call Alignment
```go
// Before any external function call in compileCall():
if fc.currentLambda != nil {
    // We're in a lambda - rsp should already be aligned from prologue
    // Just verify (optional debug check)
    // The prologue ensures rsp is 16-byte aligned
}
// Proceed with call as normal
```

### Key Insight: Don't Modify rsp in Lambda Body

The crucial fix is:
- **Allocate entire frame in prologue**
- **Never modify rsp during lambda execution**  
- **All locals/temporaries use rbp-relative addressing**
- **rsp remains 16-byte aligned for external calls**

### Current Bug Analysis

Looking at our current code:
```go
// Line 6180: We do this for each parameter:
fc.out.SubImmFromReg("rsp", 16)  // ❌ WRONG! Breaks alignment
fc.out.MovXmmToMem(xmmRegs[i], "rbp", -paramOffset)
```

**Problem**: Each parameter allocation does `sub rsp, 16`, which:
1. Moves rsp unpredictably
2. Breaks 16-byte alignment after first param
3. When printf is called, rsp is misaligned → CRASH

**Solution**: Don't modify rsp for each parameter. Do one allocation:
```go
// Allocate entire frame at once
totalSize := paramCount*16 + capturedCount*16 + 8
totalSize = (totalSize + 15) & ^15  // Round to 16
fc.out.SubImmFromReg("rsp", int64(totalSize))

// Now store params without modifying rsp
for i := range lambda.Params {
    fc.out.MovXmmToMem(xmmRegs[i], "rbp", -(24 + i*16))
}
```

## Testing Strategy

1. **Simple lambda**: `f = x => x + 1`
2. **Lambda with call**: `f = x => printf("%v\n", x)`
3. **Recursive lambda**: `fact = n => n == 0 { 1 } : { n * fact(n-1) }`
4. **Lambda with multiple params**: `add = (x, y) => x + y`
5. **Lambda with external calls**: `process = n => { printf("%v\n", n); n + 1 }`

All should work after the fix.

## Expected Impact

- ✅ Fixes printf in lambdas
- ✅ Fixes recursive lambdas with match blocks
- ✅ Fixes two-parameter lambdas  
- ✅ Makes stack traces debuggable
- ✅ ABI compliant for all external calls
- ✅ No performance loss (might be faster due to fewer rsp modifications)

## Implementation Priority

**CRITICAL**: This fixes the lambda match block segfaults and enables proper function composition.

Estimated time: 2-3 hours
Risk: Low (simplifies existing code)
Impact: HIGH (enables entire class of programs)
