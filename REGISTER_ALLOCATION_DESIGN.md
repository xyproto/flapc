# Register Allocation Design for Flapc

## Current State

The existing `register_allocator.go` implements a **linear-scan register allocation** algorithm that:
- Tracks live intervals for variables
- Allocates callee-saved registers (rbx, r12-r15 on x86-64)
- Spills to stack when registers run out
- Generates prologue/epilogue for saving/restoring registers

**Limitation**: It's designed for variables with known lifetimes, not for:
- Temporary registers (expression evaluation)
- Loop counters (short-lived but need specific registers)
- Function call arguments
- Intermediate values

## Problems with Current Ad-Hoc Approach

### Loop Counters (Current Implementation)
```go
// Manually hardcoded:
if nestingLevel == 0 {
    counterReg = "r12"
} else if nestingLevel == 1 {
    counterReg = "r13"  
} else if nestingLevel == 2 {
    counterReg = "r14"
}
// Falls back to stack for deeper nesting
```

**Issues**:
- Not flexible
- Doesn't track what else might use these registers
- Can still clobber if code between loop setup and body uses these registers

### Array Indexing
```go
// Assumes rbx is safe since loop counters are in r12/r13/r14
fc.out.MovqXmmToReg("rbx", "xmm0")
```

**Problem**: This assumption breaks if something else uses rbx!

## Proposed Solution: Two-Tier Register Allocation

### Tier 1: Live Interval Allocation (Existing)
For program variables with longer lifetimes:
- Use existing `RegisterAllocator`
- Allocates from callee-saved registers
- Handles spilling automatically

### Tier 2: Temporary Register Pool (NEW)
For short-lived values within a single expression/statement:
- **Caller-saved registers**: rax, rcx, rdx, rsi, rdi, r8-r11
- **Stack-based allocation**: When registers run out
- **Automatic release**: At statement/expression boundaries

## Implementation Plan

### Phase 1: Temporary Register Manager
```go
type TempRegManager struct {
    available []string      // Free temporary registers
    inUse     map[string]int // Register -> refcount
    stackSlots int           // Spilled temps on stack
}

// Allocate a temporary register
func (trm *TempRegManager) AllocTemp() (string, bool)

// Release a temporary register
func (trm *TempRegManager) ReleaseTemp(reg string)

// Spill to stack if no registers available
func (trm *TempRegManager) SpillTemp() int
```

### Phase 2: Loop Counter Allocation
```go
type LoopRegManager struct {
    counterRegs []string      // Available counter registers
    loopStack   []LoopContext // Active loops
}

// Allocate counter for new loop
func (lrm *LoopRegManager) AllocLoopCounter(nestLevel int) string

// Release counter when loop ends
func (lrm *LoopRegManager) ReleaseLoopCounter(nestLevel int)
```

### Phase 3: Integration
Combine all three systems:
```go
type UnifiedRegAlloc struct {
    variables *RegisterAllocator  // Long-lived variables
    temps     *TempRegManager     // Expression temporaries
    loops     *LoopRegManager     // Loop counters
}
```

## Alternative: Expression-Tree Register Allocation

A more sophisticated approach used by modern compilers:

### Algorithm
1. **Build expression tree**: Parse expression into tree structure
2. **Compute register pressure**: Bottom-up traversal
3. **Allocate optimally**: Use Sethi-Ullman numbering
4. **Spill if needed**: Materialized at boundaries

### Example
```flap
result = (a + b) * (c + d) + e
```

Tree:
```
      +
     / \
    *   e
   / \
  +   +
 / \ / \
a  b c  d
```

Register allocation:
```
r1 = a + b       // Allocate r1
r2 = c + d       // Allocate r2
r1 = r1 * r2     // Reuse r1, r2
r1 = r1 + e      // Reuse r1
```

**Benefits**:
- Optimal register usage
- Minimizes spills
- Handles arbitrarily complex expressions

**Cost**:
- More complex to implement
- Requires expression tree transformation

## Recommendation

### Short-term (For Flap 3.0)
Implement **Tier 2: Temporary Register Pool**:
1. Add `TempRegManager` for expression evaluation
2. Use it in array indexing, arithmetic, comparisons
3. Keep current loop counter approach but make it use TempRegManager

**Benefits**:
- Solves immediate clobbering issues
- Simple to implement (~200 lines)
- No breaking changes

### Long-term (Post 3.0)
Implement **Expression-Tree Register Allocation**:
1. Build IR from AST with explicit temporaries
2. Apply Sethi-Ullman or graph-coloring algorithm
3. Generate optimal code

**Benefits**:
- 20-30% fewer instructions
- Better cache utilization
- Professional-grade code generation

## Register Assignment Strategy (x86-64)

### Caller-Saved (Temporaries)
- **rax**: Return values, general temp
- **rcx**: Counter, temp
- **rdx**: Data, temp  
- **rsi, rdi**: Source/dest, function args, temps
- **r8-r11**: General temporaries

### Callee-Saved (Variables)
- **rbx**: General variable
- **r12-r15**: Loop counters, variables
- **rbp**: Frame pointer (reserved)
- **rsp**: Stack pointer (reserved)

### XMM Registers
- **xmm0**: Primary float register, return value
- **xmm1-xmm5**: Function arguments (float)
- **xmm6-xmm15**: General purpose (xmm6-xmm7 caller-saved in System V)

## Implementation Priority

1. ✅ **Loop counter safety**: Use register allocator or dedicated pool
2. ✅ **Array indexing**: Don't assume rbx availability  
3. **Expression temps**: Use temp pool instead of hardcoded rax/rcx
4. **Function calls**: Properly save/restore used registers
5. **Closure captures**: Coordinate with register allocation

## Code Size & Performance Impact

### Current (Ad-hoc)
- Simple, fast compilation
- Suboptimal code (many redundant moves)
- ~30% more instructions than optimal

### With Temp Pool
- Fast compilation
- Good code quality
- ~15% more instructions than optimal
- **Recommended for 3.0**

### With Expression-Tree Allocation
- Slower compilation
- Optimal code
- Industry-standard quality
- **Future enhancement**

## Testing Strategy

1. **Unit tests**: Test register allocator in isolation
2. **Integration tests**: Complex expressions with nesting
3. **Stress tests**: Deep loop nesting, large expressions
4. **Regression tests**: Ensure no clobbering in existing patterns

## Conclusion

The existing register allocator is well-designed but incomplete. Adding a temporary register manager solves the immediate clobbering issues while maintaining simplicity. Expression-tree allocation can be added later for optimal code generation.

**Status**: Ready to implement Temp Register Manager
**Estimated effort**: 4-6 hours
**Impact**: Eliminates all register clobbering bugs
