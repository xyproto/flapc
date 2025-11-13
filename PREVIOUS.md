# Previously Removed Syntax

This document tracks language features that were removed from Flap over time, with explanations for why they were removed.

---

## Version 2.0 (November 2025) - Simplification Release

### Port Literals (`:5000`)
**Removed:** November 2025  
**Reason:** Redundant - strings work better

**Old syntax:**
```flap
:5000                    // Port literal
:5000+ <== "msg"         // Next available port
:5000? <== "msg"         // Check if available
```

**New syntax:**
```flap
":5000" <== "msg"        // Just use strings
```

**Why removed:**
- Strings are more flexible ("host:port" works the same)
- No need for special literal syntax
- Simpler parser (one less token type)
- More consistent with general Flap philosophy

---

### Function Arrow `->` (Replaced with `=>`)
**Removed:** November 2025  
**Reason:** Unified all functions to use single arrow

**Old syntax:**
```flap
// Two different arrows
f := x -> x + 1          // Function definition arrow
x { 42 -> "yes" }        // Match clause arrow (kept)
```

**New syntax:**
```flap
// One arrow for functions
f := x => x + 1          // Lambda/function arrow
x { 42 -> "yes" }        // Match clause arrow (kept different)
```

**Why removed:**
- Two arrows (`->` and `=>`) were confusing
- `=>` is now universal for all functions
- Match clauses still use `->` (different context)
- Clearer mental model: `=>` = function, `->` = match

---

### Reduce Keyword in Parallel Loops
**Removed:** October 2025  
**Reason:** Added complexity without clear benefit

**Old syntax:**
```flap
sum := 0
@@ i in 0..<1000 reduce(+) {
    compute(i)
}
```

**New syntax:**
```flap
// Just do it explicitly
results := []
@@ i in 0..<1000 {
    results[i] <- compute(i)
}
sum := results ||| (acc, x) => acc + x
```

**Why removed:**
- Magic behavior - not obvious how it works
- Limited to specific operations (+, *, max, min)
- Explicit reduction is clearer
- Reduce pipe `|||` is more flexible

---

### Yield and Events Model
**Removed:** October 2025  
**Reason:** Too complex, didn't fit Unix process model

**Old syntax:**
```flap
yield               // Yield to scheduler
@ event in events   // Event loop
```

**New syntax:**
```flap
spawn fn()          // Fork-based processes
@@ i in range       // OpenMP-style parallelism
@ msg, from in port // ENet messaging
```

**Why removed:**
- Flap uses fork() model, not green threads
- Events added unnecessary complexity
- ENet provides better concurrency model
- Simpler to understand and implement

---

### Hot Reload Feature
**Removed:** September 2025  
**Reason:** Feature creep, not core to language

**Old syntax:**
```flap
// Compiler watched files and recompiled on changes
```

**Why removed:**
- Not a language feature, belongs in tooling
- Added complexity to compiler
- Users can use external tools (entr, watchexec)
- Focus on core language features

---

### Unsafe Block `ret` Keyword
**Removed:** September 2025  
**Reason:** Inconsistent with rest of language

**Old syntax:**
```flap
unsafe {
    // risky code
    ret 42
}
```

**New syntax:**
```flap
unsafe {
    // risky code
    42
}
```

**Why removed:**
- Unsafe blocks are expressions, not statements
- Last expression is the value (like everything else)
- `ret` was redundant

---

### Infinite Loop Syntax `@-`
**Removed:** September 2025  
**Reason:** Inconsistent syntax

**Old syntax:**
```flap
@- {
    // infinite loop
}
```

**New syntax:**
```flap
@ {
    // infinite loop
}
```

**Why removed:**
- `@-` was arbitrary syntax
- `@` alone is simpler and clearer
- Consistent with `@ var in range` pattern

---

### Multiple Lambda Arrow Syntaxes
**Removed:** November 2025  
**Reason:** Consolidated to single `=>`

**Old syntax:**
```flap
f := x -> x + 1      // Arrow syntax
g := x => x * 2      // Fat arrow syntax
h := (x) ==> x / 2   // Double fat arrow
```

**New syntax:**
```flap
f := x => x + 1      // Only =>
```

**Why removed:**
- Three different syntaxes for same thing
- Confusing and inconsistent
- `=>` is standard in many languages
- Simpler to teach and remember

---

### Guard Prefix `|` for Match
**Removed:** November 2025  
**Reason:** Match expressions simplified

**Old syntax:**
```flap
x {
    | x > 10 -> "big"
    | x < 5 -> "small"
    ~> "medium"
}
```

**New syntax:**
```flap
x {
    x > 10 -> "big"
    x < 5 -> "small"
    ~> "medium"
}
```

**Why removed:**
- Prefix `|` was redundant
- No ambiguity without it
- Cleaner syntax
- Less visual noise

---

### Bitwise Operators Without `b` Suffix (Partial)
**Removed:** November 2025  
**Reason:** Ambiguity with comparison/shift

**Old syntax:**
```flap
<      // Could be bitwise shift OR comparison
>      // Could be bitwise shift OR comparison
```

**New syntax:**
```flap
<<b    // Shift left (bitwise)
>>b    // Shift right (bitwise)
<      // Less than (comparison)
>      // Greater than (comparison)
```

**Why removed:**
- `<` and `>` were ambiguous
- Now all bitwise operations have `b` suffix
- Consistent: `&b`, `|b`, `^b`, `<<b`, `>>b`, `<<<b`, `>>>b`
- No parser confusion

---

## Design Principles for Removals

When removing syntax, we follow these principles:

1. **Simplicity** - Fewer ways to do the same thing
2. **Consistency** - Similar things should look similar
3. **Clarity** - Syntax should be obvious, not clever
4. **Minimalism** - Remove features that don't pull their weight
5. **Unix Philosophy** - Do one thing well

### What We Keep

Features are kept if they:
- Solve a real problem
- Have no simpler alternative
- Are fundamental to the language
- Improve expressiveness significantly

### What We Remove

Features are removed if they:
- Are redundant (another way exists)
- Add complexity without benefit
- Are inconsistent with other features
- Can be done with external tools
- Confuse users or parser

---

## Lessons Learned

1. **Syntax proliferation is bad** - Multiple ways to do the same thing causes confusion
2. **Special cases are costly** - Each special syntax adds parser/compiler complexity
3. **Simplicity wins** - When in doubt, remove rather than add
4. **Be opinionated** - Fewer choices means clearer code
5. **Strings are powerful** - Don't create special literals when strings work

---

**Last Updated:** 2025-11-13  
**Flap Version:** 2.0.0
