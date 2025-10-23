# Nested Loop Implementation Design

## Problem Statement

Need to support arbitrary depth nested loops where each loop maintains:
- Loop counter (current iteration value)
- Loop limit (end value)
- Iterator variable (accessible in loop body as float64)

## Current Implementation Issues

### Original Approach (Push/Pop r12/r13)
```
if nested:
    push r12, r13
r12 = start
r13 = end
loop:
    compare r12, r13
    if r12 >= r13: goto end
    [body - may contain nested loop]
    inc r12
    goto loop
end:
if nested:
    pop r13, r12
```

**Problem:** Works for 2 levels but fails for 3+ levels because:
- Only saves the immediately outer loop's registers
- Doesn't form a proper stack of loop contexts

### Failed Approach (Stack-based with premature store)
- Attempted to store counter/limit to stack BEFORE allocating space
- Wrote to wrong memory location
- Caused infinite loops

## Design Requirements

1. **Arbitrary nesting depth**: Must support 3+ nested loops
2. **Isolation**: Each loop level must have independent counter/limit
3. **Performance**: Should be reasonably efficient (minimize memory ops)
4. **Iterator access**: Loop body must access current iterator value
5. **Register preservation**: Must work with function calls in loop body

## Proposed Solution: Pure Stack-Based Approach

### Memory Layout (per loop level)
```
[rbp - N*32]       Iterator value (float64, 8 bytes + 8 padding = 16 bytes)
[rbp - N*32 - 16]  Counter (int64, 8 bytes)
[rbp - N*32 - 24]  Limit (int64, 8 bytes)
```

Each loop allocates 32 bytes on stack for its state.

### Algorithm

```
LOOP_START:
    # Allocate stack space (32 bytes)
    stackOffset += 32
    counterOffset = stackOffset - 24
    limitOffset = stackOffset - 16
    iterOffset = stackOffset
    sub rsp, 32

    # Evaluate and store start value
    <compile start expression> -> xmm0
    cvttsd2si rax, xmm0
    mov [rbp - counterOffset], rax

    # Evaluate and store end value
    <compile end expression> -> xmm0
    cvttsd2si rax, xmm0
    if inclusive: inc rax
    mov [rbp - limitOffset], rax

LOOP_TOP:
    # Load counter and limit for comparison
    mov rax, [rbp - counterOffset]
    mov rcx, [rbp - limitOffset]
    cmp rax, rcx
    jge LOOP_END

    # Store iterator value for body access
    cvtsi2sd xmm0, rax
    movsd [rbp - iterOffset], xmm0

    # Compile body (may contain nested loops)
    <loop body>

CONTINUE_POINT:
    # Increment counter
    mov rax, [rbp - counterOffset]
    inc rax
    mov [rbp - counterOffset], rax
    jmp LOOP_TOP

LOOP_END:
    # Clean up stack
    add rsp, 32
    stackOffset -= 32
```

### Key Principles

1. **No register state between iterations**: Load from stack at loop start
2. **Stack space allocated FIRST**: Before any stores
3. **Independent state**: Each nesting level has 32-byte stack frame
4. **Consistent offsets**: Use rbp-relative addressing for all accesses

## Test Cases

### 1. Simple loop (baseline)
```flap
@ i in 0..<3 {
    println(i)
}
// Expected: 0, 1, 2
```

### 2. Two-level nesting
```flap
@ i in 0..<2 {
    @ j in 0..<3 {
        println(i * 10 + j)
    }
}
// Expected: 00, 01, 02, 10, 11, 12
```

### 3. Three-level nesting
```flap
@ i in 0..<2 {
    @ j in 0..<2 {
        @ k in 0..<2 {
            println(i * 100 + j * 10 + k)
        }
    }
}
// Expected: 000, 001, 010, 011, 100, 101, 110, 111
```

### 4. Variable bounds (inner loop depends on outer)
```flap
@ i in 0..<2 {
    count := i + 2
    @ j in 0..<count {
        printf("*")
    }
    printf("\n")
}
// Expected: ** (i=0, count=2)
//           *** (i=1, count=3)
```

### 5. Function calls in loop body
```flap
@ i in 0..<3 {
    printf("i=%v\n", i)
}
// Function calls must not corrupt loop state
```

## Implementation Checklist

- [ ] Remove push/pop r12/r13 logic
- [ ] Allocate stack space BEFORE stores
- [ ] Use consistent rbp-relative offsets
- [ ] Load counter/limit at loop top (not before loop)
- [ ] Store counter after increment
- [ ] Test with 1, 2, 3, 4 nesting levels
- [ ] Test with variable bounds
- [ ] Test with function calls in body

## Edge Cases

1. **Empty range**: 0..<0 should not execute body
2. **Reverse range**: 5..<3 should not execute body
3. **Inclusive vs exclusive**: 0..=2 is {0,1,2}, 0..<2 is {0,1}
4. **Break/continue**: Must jump to correct positions
5. **Large nesting**: 10+ levels should work (stack permitting)

## Performance Considerations

- **Memory ops per iteration**: 2 loads (counter, limit) + 1 store (counter)
- **Trade-off**: More memory ops vs. register corruption safety
- **Optimization opportunity**: Could use registers for innermost loop only
- **Current priority**: Correctness over performance
