# Flapc TODO - x86_64 Focus

## Current Sprint (x86_64 only)
- [ ] **SIMD intrinsics** - Vector operations for audio DSP, graphics effects, particle systems
  - [x] vec2() and vec4() constructors
  - [x] VectorExpr AST node and parser support
  - [x] SIMD instruction wrappers (movupd, addpd, subpd, mulpd, divpd)
  - [ ] Vector arithmetic operations (vadd, vsub, vmul, vdiv) - needs debugging
  - [ ] Vector component access (v.x, v.y, v.z, v.w or v[0], v[1], etc.)
  - [ ] Dot product, magnitude, normalize operations
- [ ] **Register allocation improvements** - Better register usage for performance
  - [x] Binary operation optimization (use xmm2 instead of stack spills)
  - [x] Direct register-to-register moves (movq xmm, rax)
  - [ ] Keep loop counters in registers
  - [ ] Register allocation for frequently-used variables
  - [ ] Full register allocator with liveness analysis
- [ ] **Dead code elimination** - Remove unused code from output
- [ ] **Constant propagation across functions** - Optimize constants through call boundaries
- [ ] **Inline small functions automatically** - Performance optimization

## Future stdlib (architecture-agnostic)

- [ ] **Collections** - Hash map, tree, queue, stack
- [ ] **String manipulation** - split, join, replace, regex
- [ ] **File I/O library** - High-level wrappers for file operations
- [ ] **Network programming** - Sockets, HTTP
- [ ] **JSON parsing and serialization** - Configuration and data exchange
- [ ] **Date/time library** - Timing and scheduling utilities
# ✅ IMPLEMENTED: Multi-Statement Match Blocks

**Status**: Implemented on 2025-10-21

## Implementation Details

Multi-statement match blocks are now supported using the following syntax:

```flap
condition {
    -> {
        stmt1
        stmt2
        stmt3
    }
    ~> {
        other_stmt1
        other_stmt2
    }
}
```

The implementation adds support for parsing statement blocks (`BlockExpr`) in match arms through the `parseMatchTarget()` function in `parser.go`. When a `{` token is encountered after a match arrow (`->` or `~>`), the parser now recognizes it as a block of statements rather than requiring a single expression.

## Original Suggestion

# Key Improvement Suggestion for Flap/Flapc

## The Problem: Limited Control Flow Expressiveness

After writing several non-trivial Flap programs, I've identified the **single most impactful improvement** that would dramatically enhance the language's usability:

### **Add Multi-Statement Match Blocks (or Traditional If/Else)**

## Current Limitations

Flap's match expression syntax is elegant but severely restrictive:

```flap
// Current: Only ONE statement allowed per branch
condition {
    println("true")  // ✓ Works
}

// Current: Multiple statements FAIL
condition {
    x <- 5           // ✗ Error: "bare match clause must be the only entry in the block"
    println(x)
}
```

This limitation makes even simple algorithms painful to express. Examples from my attempted programs:

### Example 1: Sieve of Eratosthenes (IMPOSSIBLE)
```flap
// Wanted to write:
isPrime == 1.0 {
    multiple := i * 2
    @ multiple < N {
        sieve[multiple] = 0.0
        multiple = multiple + i
    }
}

// ERROR: "bare match clause must be the only entry in the block"
```

### Example 2: Collatz Conjecture (WORKAROUND UGLY)
```flap
// Wanted:
@ n > 1 {  // While loop
    isEven {
        n <- n / 2
        println(n)
    }
}

// Had to use:
@ i in 0..<1000 {  // Arbitrary limit!
    n == 1 { ret @1 }  // Manual break
    // Can't even put the conditional logic cleanly...
}
```

## Proposed Solution

**Option A: Multi-Statement Match Blocks** (maintains current syntax)
```flap
condition {
    {  // Explicit block for multiple statements
        stmt1
        stmt2
        stmt3
    }
    ~> {  // Default case
        other_stmt
    }
}
```

**Option B: Traditional If/Else** (more familiar)
```flap
if condition {
    stmt1
    stmt2
} else {
    other_stmt
}
```

**Option C: Do-Block Syntax** (Lisp-inspired)
```flap
condition {
    do {
        stmt1
        stmt2
    }
}
```

## Why This is THE Most Important Improvement

### 1. **Unblocks Complex Algorithms**
Currently impossible or extremely awkward:
- Prime sieving
- Graph traversal
- State machines
- Multi-step conditional logic

### 2. **Reduces Cognitive Load**
Programmers shouldn't spend 80% of their time fighting the syntax to express simple ideas like:
```
if condition:
    do A
    do B
    do C
```

### 3. **Maintains Flap's Philosophy**
You can still keep:
- ✓ Map-based foundation
- ✓ Expression-oriented design
- ✓ Minimal keywords
- ✓ Float64 unification

Just make conditionals **usable**.

### 4. **Real-World Impact**
From my experience writing examples:
- **Fibonacci**: ✓ Works (simple loop, no conditionals)
- **Number Series**: ✓ Works (arithmetic only)
- **Prime Sieve**: ✗ FAILED (needs multi-stmt conditionals)
- **Collatz**: ✗ FAILED (needs while + multi-stmt)
- **Lambda Calc**: ✗ FAILED (lambda bugs + complexity)
- **ASCII Art**: ✗ FAILED (nested loops exploded)

**Success rate: 33%** - and only because I avoided conditionals!

## Implementation Complexity

**Low effort, high impact:**

1. Parser already handles blocks (`{ }`)
2. Match expressions already exist
3. Just need to allow `BlockStmt` inside match arms instead of requiring single `Expression`

Estimated: **~200 lines of code** in parser.go

## Alternative Considered

**"Just use lambdas!"**
```flap
condition {
    (() => {
        stmt1
        stmt2
    })()
}
```

Problems:
- Ugly
- Performance overhead (function call)
- Still restrictive (can't access outer scope mutably)
- Nobody wants to write this

## Conclusion

**Without multi-statement conditionals, Flap remains a toy language.**

With them, it becomes a practical, innovative language that:
- Has a unique float64-based type system
- Compiles to fast native code
- Supports real algorithms
- Is actually pleasant to use

This one change would increase Flap's usability by **10x** while requiring minimal implementation effort.

---

## Appendix: Working Examples

### Example 1: Fibonacci Sequence ✓
**File**: `programs/fibonacci.flap`

```flap
// Fibonacci sequence generator
// Demonstrates: loops, mutable variables, mathematical sequences

n := 20
a := 0.0
b := 1.0

printf("First %v Fibonacci numbers:\n", n)

// Generate and print Fibonacci sequence
@ i in 0..<n {
    printf("%v ", a)

    // Calculate next Fibonacci number
    next := a + b
    a <- b
    b <- next
}

printf("\n")
```

**Output**:
```
First 20 Fibonacci numbers:
0 1 1 2 3 5 8 13 21 34 55 89 144 233 377 610 987 1597 2584 4181
```

### Example 2: Number Series with Statistics ✓
**File**: `examples/ex1_number_series.flap`

```flap
// Generate number series and compute statistics
// Demonstrates: loops, arithmetic, mutable variables

sum := 0.0
count := 20

printf("Squares of first %v numbers:\n", count)

@ i in 1..<(count + 1) {
    square := i * i
    sum <- sum + square
    printf("%v² = %v\n", i, square)
}

average := sum / count
printf("\nSum of squares: %v\n", sum)
printf("Average: %v\n", average)
```

**Output**:
```
Squares of first 20 numbers:
1² = 1
2² = 4
3² = 9
...
20² = 400

Sum of squares: 2870
Average: 143.5
```

### Example 3: Prime Sieve ✗ (ATTEMPTED)
Could not complete due to match block limitations.

---

**Generated with Claude Code** - After extensive hands-on experience with Flap
