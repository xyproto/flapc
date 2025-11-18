# Lambda Frame Allocation - Complete Solution Required

## Problem

Lambdas with local variables crash because:

1. **Fixed frame allocation** in prologue: `sub rsp, FRAME_SIZE`
2. **Dynamic rsp modification** for local vars: `sub rsp, 16` per variable
3. **Mismatched offsets**: Variables use offsets assuming dynamic rsp changes

## Example Failure

```flap
f = x => {
    y = x + 1  // This crashes!
    y
}
```

### What Happens:
1. Prologue: `sub rsp, 4112` (fixed frame)
2. Variable `y` assignment: `sub rsp, 16` (goes beyond allocated frame!)
3. SEGFAULT: rsp now points to invalid/protected memory

## Root Cause

The compiler has two incompatible models:

**Top-level functions**: Dynamic stack growth
- Each variable: `sub rsp, 16`
- Offsets: relative to current rsp
- Works fine

**Lambdas**: Fixed pre-allocated frame  
- All space allocated upfront
- BUT code still does `sub rsp, 16`!
- Offsets calculated wrong

## Complete Solution (Required for 3.1)

### Phase 1: Scan Lambda Body for Locals

```go
func countLambdaLocals(body Expression) int {
    count := 0
    // Recursively scan BlockExpr, MatchExpr, etc.
    // Count all AssignStmt where !IsUpdate && !IsReuseMutable
    return count
}
```

### Phase 2: Pre-allocate All Space

```go
paramCount := len(lambda.Params)
capturedCount := len(lambda.CapturedVars)
localCount := countLambdaLocals(lambda.Body)

// Calculate offsets for each category
paramBase := 24
capturedBase := paramBase + paramCount*16
localBase := capturedBase + capturedCount*16

frameSize := localBase + localCount*16
frameSize = (frameSize + 15) & ^15  // Align

// Allocate once
fc.out.SubImmFromReg("rsp", frameSize)
```

### Phase 3: Track Lambda Variable Offsets

```go
type LambdaContext struct {
    params    map[string]int  // param -> offset
    captured  map[string]int  // captured -> offset
    locals    map[string]int  // local -> offset
    nextLocal int             // Next available local offset
}
```

### Phase 4: Modify Variable Assignment

```go
case *AssignStmt:
    if fc.currentLambda != nil {
        // In lambda: use pre-allocated offset
        if offset, exists := fc.lambdaContext.locals[s.Name]; exists {
            // Use existing
        } else {
            // Allocate from pre-calculated space
            offset = fc.lambdaContext.nextLocal
            fc.lambdaContext.locals[s.Name] = offset
            fc.lambdaContext.nextLocal += 16
        }
        // NO rsp modification!
    } else {
        // Top-level: dynamic allocation
        fc.out.SubImmFromReg("rsp", 16)
    }
```

## Workarounds (Until Fixed)

### 1. Use Function Parameters Only
```flap
// DON'T:
f = x => {
    y = x + 1  // Crashes
    y
}

// DO:
f = x => x + 1  // Works
```

### 2. Hoist Variables to Closure
```flap
// DON'T:
process = data => {
    result = data * 2  // Crashes
    result
}

// DO:
make_processor = data => {
    result = data * 2
    () => result  // Return lambda that captures result
}
process = make_processor(data)()
```

### 3. Use Match Instead of Variables
```flap
// Instead of:
f = x => {
    doubled = x * 2
    doubled + 1
}

// Use:
f = x => (x * 2) + 1  // Inline expression
```

## Testing Strategy

After fix, test:

```flap
// Test 1: Single local
f = x => {
    y = x + 1
    y
}

// Test 2: Multiple locals
f = x => {
    a = x + 1
    b = a * 2
    c = b - 3
    c
}

// Test 3: Locals in match
f = n => {
    | n == 0 -> {
        result = 0
        result
    }
    ~> {
        temp = n - 1
        f(temp) + n
    }
}

// Test 4: Nested blocks
f = x => {
    {
        y = x + 1
        {
            z = y * 2
            z
        }
    }
}
```

## Implementation Checklist

- [ ] Add `countLambdaLocals(Expression)` function
- [ ] Create `LambdaContext` struct
- [ ] Store context in `FlapCompiler.currentLambdaCtx`
- [ ] Calculate `localCount` in `generateLambdaFunctions`
- [ ] Update `frameSize` calculation
- [ ] Modify `AssignStmt` compilation to check `currentLambda`
- [ ] Modify `MultipleAssignStmt` similarly
- [ ] Test all 4 test cases above
- [ ] Update FLAP_3.0_RELEASE_STATUS.md

## Estimated Effort

- **Time**: 4-6 hours
- **Complexity**: Medium
- **Risk**: Low (well-understood problem)
- **Impact**: HIGH (enables all lambda patterns)

## Priority

**CRITICAL** for 3.1 release

This is the #1 blocker for full lambda functionality.

## Alternative: Hybrid Approach

If full solution is complex, consider:

1. **Detect local variables in lambda**
2. **If any found**: Reject at compile time with clear error
3. **Error message**: "Local variables in lambdas not yet supported. Use parameters or hoist to closure."

This makes the limitation explicit rather than silently crashing.

```go
if fc.currentLambda != nil && hasLocalVariables(lambda.Body) {
    compilerError("local variables in lambda bodies not yet supported")
}
```

**Pros**:
- Fast to implement (30 minutes)
- Clear error message
- No crashes

**Cons**:
- Still limits lambda functionality
- Users must work around

## Recommendation

Implement **Alternative (Hybrid Approach)** for 3.0, then proper fix for 3.1.

This gives:
- Safe 3.0 release (no crashes)
- Clear path forward
- Users know the limitation upfront
