# Proposal: Heap-Allocated Shadow Stack for Recursive Lambdas on macOS

## Problem

macOS dyld only provides ~5.6KB of stack space regardless of the LC_MAIN stacksize field. This causes crashes in recursive lambdas before user code even runs.

## Solution: Hybrid Stack Approach

Use the native stack for non-recursive calls, but allocate a separate heap-based "shadow stack" for recursive lambda invocations.

## Implementation Strategy

### Phase 1: Shadow Stack Allocation (Startup)

At program initialization, allocate a large shadow stack:

```assembly
; In _start or early in _main:
; Allocate 8MB shadow stack using mmap
mov x0, #0              ; addr = NULL (let kernel choose)
mov x1, #0x800000       ; length = 8MB
mov x2, #3              ; prot = PROT_READ | PROT_WRITE
mov x3, #0x1002         ; flags = MAP_PRIVATE | MAP_ANON
mov x4, #-1             ; fd = -1
mov x5, #0              ; offset = 0
mov x16, #197           ; syscall number for mmap
svc #0

; Store shadow stack base in a global
adrp x1, shadow_stack_base
str x0, [x1, :lo12:shadow_stack_base]

; Initialize shadow stack pointer to top
add x0, x0, #0x800000
adrp x1, shadow_stack_ptr
str x0, [x1, :lo12:shadow_stack_ptr]
```

### Phase 2: Detect Recursive Lambda Calls

At compile time, detect which lambdas are self-recursive:

```go
type ARM64LambdaFunc struct {
    Name         string
    Params       []string
    Body         Expression
    BodyStart    int
    FuncStart    int
    VarName      string
    IsRecursive  bool  // NEW: Mark recursive lambdas
}
```

### Phase 3: Modified Call Convention for Recursive Lambdas

For recursive lambda calls, use shadow stack instead of native stack:

```assembly
; Before recursive call:
; 1. Save return address to shadow stack
adrp x10, shadow_stack_ptr
ldr x10, [x10, :lo12:shadow_stack_ptr]
sub x10, x10, #16          ; Allocate shadow stack frame
adrp x11, shadow_stack_ptr
str x10, [x11, :lo12:shadow_stack_ptr]

; 2. Save return address
adr x11, .return_point
str x11, [x10, #0]         ; Save return PC

; 3. Save frame pointer
str x29, [x10, #8]         ; Save old FP

; 4. Set up new frame pointer pointing to shadow stack
mov x29, x10

; 5. Call recursive function
bl recursive_lambda

.return_point:
; On return, restore shadow stack pointer
adrp x10, shadow_stack_ptr
ldr x10, [x10, :lo12:shadow_stack_ptr]
add x10, x10, #16          ; Pop shadow stack frame
adrp x11, shadow_stack_ptr
str x10, [x11, :lo12:shadow_stack_ptr]
```

### Phase 4: Modified Function Prologue for Recursive Lambdas

Recursive lambdas check if they're being called recursively:

```assembly
recursive_lambda:
    ; Check if x29 points into shadow stack range
    adrp x10, shadow_stack_base
    ldr x10, [x10, :lo12:shadow_stack_base]
    cmp x29, x10
    b.lt .use_native_stack      ; FP < shadow base = native stack

    adrp x11, shadow_stack_base
    ldr x11, [x11, :lo12:shadow_stack_base]
    add x11, x11, #0x800000     ; shadow top
    cmp x29, x11
    b.ge .use_native_stack      ; FP >= shadow top = native stack

.use_shadow_stack:
    ; FP is in shadow stack range - we're in recursion
    ; Allocate locals on shadow stack
    sub x10, x29, #local_size
    ; ... function body with shadow stack ...
    ret                          ; Return address is on shadow stack

.use_native_stack:
    ; First call - use native stack
    stp x29, x30, [sp, #-16]!
    mov x29, sp
    ; ... standard prologue ...
```

## Alternative: Simpler Approach for Initial Implementation

For a quicker fix, we could use a simpler approach:

### Arena-Based Recursion

Treat recursive calls like arena allocations:

```go
// At lambda definition
if isRecursive {
    // Allocate recursion state structure on heap
    recursionState := malloc(sizeof(RecursionFrame) * maxDepth)
    currentDepth := 0
}

// On each recursive call
if currentDepth >= maxDepth {
    panic("Max recursion depth exceeded")
}

frames[currentDepth] = {args, locals}
currentDepth++
result := recursiveCall(...)
currentDepth--
return result
```

This is simpler but requires:
- Explicit max depth specification
- Heap allocation for each lambda
- Runtime depth checking

## Recommended Approach: Shadow Stack

**Advantages:**
1. ✅ Works with unlimited recursion (8MB = ~1M call frames)
2. ✅ No modification to calling code
3. ✅ Near-native performance
4. ✅ Transparent to user
5. ✅ Can be platform-specific (macOS only)

**Implementation Complexity:** Medium
- Add shadow stack globals
- Modify recursive lambda prologue/epilogue
- Add stack range checking
- ~200 lines of ARM64 assembly generation code

**Performance Impact:** Minimal
- One extra check per recursive call
- Shadow stack access is still just memory access
- No heap allocation per call

## Code Changes Required

### 1. arm64_codegen.go

```go
// Add shadow stack support
func (acg *ARM64CodeGen) generateShadowStackInit() {
    // Generate mmap syscall for shadow stack
    // Store base and current pointer in globals
}

func (acg *ARM64CodeGen) compileSelfRecursiveCall(call *CallExpr) error {
    // Modified to use shadow stack for storage
    // Add return address to shadow stack
    // Jump to function with shadow FP
}

func (acg *ARM64CodeGen) generateRecursiveLambdaPrologue(lambda *ARM64LambdaFunc) {
    // Check if FP is in shadow stack range
    // Branch to appropriate prologue
}
```

### 2. parser.go

```go
// Add shadow stack globals
fc.eb.DefineGlobal("shadow_stack_base", 8)   // uint64
fc.eb.DefineGlobal("shadow_stack_ptr", 8)    // uint64

// Call shadow stack init early in main
fc.generateShadowStackInit()
```

### 3. Build Tags

```go
//go:build darwin && arm64

// Only include shadow stack for macOS ARM64
```

## Testing Plan

1. **Unit Tests:**
   - Test shadow stack allocation
   - Test stack range checking
   - Test frame save/restore

2. **Integration Tests:**
   - Simple recursive factorial
   - Deep recursion (10,000+ levels)
   - Mutual recursion (A calls B calls A)
   - Mixed recursive/non-recursive calls

3. **Performance Tests:**
   - Compare with x86_64 native stack
   - Measure overhead of stack check

## Timeline Estimate

- **Phase 1:** Shadow stack allocation and globals - 2 hours
- **Phase 2:** Detect recursive lambdas - 1 hour
- **Phase 3:** Modified call convention - 4 hours
- **Phase 4:** Function prologue/epilogue - 3 hours
- **Testing:** 2 hours

**Total:** ~12 hours (1.5 days)

## Risks and Mitigation

**Risk:** Shadow stack overflow
**Mitigation:** Add guard page at end, check depth counter

**Risk:** Performance degradation
**Mitigation:** Only use for recursive lambdas, native stack for everything else

**Risk:** Debugging difficulty
**Mitigation:** Add shadow stack dump function for errors

## Conclusion

The shadow stack approach is the most robust solution for macOS recursive lambda support. It works around the macOS stack limitation while maintaining good performance and transparency to users.

**Recommendation:** Implement shadow stack for macOS ARM64 only. Keep native stack for x86_64/Linux.
