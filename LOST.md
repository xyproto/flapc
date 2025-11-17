# Lost Features in Documentation Refactoring

**Date:** 2025-11-17  
**Context:** Features lost when refactoring from LANGUAGE.md to GRAMMAR.md + LANGUAGESPEC.md

This document tracks features that existed in the original LANGUAGE.md (commit 99bcab1) but were incorrectly removed or changed during the documentation refactoring to GRAMMAR.md and LANGUAGESPEC.md.

---

## 1. Loop Control: `ret @` Instead of `break`/`continue`

**Status:** ❌ INCORRECTLY CHANGED

### What Was Lost

The original Flap design used `ret @` with loop labels instead of `break`/`continue` keywords:

```flap
// Original Flap syntax (CORRECT)
@ i in 0..<100 {
    i > 50 { ret @ }      // Exit current loop
    i == 42 { ret @ 42 }  // Exit loop with value
    println(i)
}

// Nested loops with explicit labels
@ i in 0..<10 {           // This is loop @1
    @ j in 0..<10 {       // This is loop @2
        j == 5 { ret @ }  // Exit inner loop (@2)
        i == 5 and j == 3 { ret @1 }  // Exit outer loop (@1)
        println(f"i={i}, j={j}")
    }
}
```

### What Was Added (WRONG)

GRAMMAR.md and LANGUAGESPEC.md incorrectly added `break` and `continue` keywords:

```flap
// WRONG - not Flap!
@ i in 0..<10 {
    i == 5 { break }
    i % 2 == 0 { continue }
}
```

### Why This Matters

- **Philosophy violation**: Flap uses minimal keywords
- **Consistency**: `ret` already returns from functions, `ret @` returns from loops
- **Power**: Loop labels allow precise control over nested loops
- **Elegance**: No need for two new keywords

### Fix Required

1. Remove `break` and `continue` from keywords
2. Remove `break_statement` and `continue_statement` from grammar
3. Add loop control via `ret @` and `ret @N`
4. Update all loop control examples

---

## 2. Classes and Object-Oriented Programming

**Status:** ❌ PARTIALLY LOST

### What Was Lost

The original LANGUAGE.md had a complete design for classes with the `class` keyword and `<>` composition operator:

```flap
class Point {
    init = (x, y) => {
        .x = x
        .y = y
    }
    
    distance = other => {
        dx := other.x - .x
        dy := other.y - .y
        sqrt(dx * dx + dy * dy)
    }
    
    move = (dx, dy) => {
        .x <- .x + dx
        .y <- .y + dy
    }
}

// Composition with <> operator
Printable = {
    to_s = () => {
        fields := []
        @ key in keys(this) {
            fields <- fields :: f"{key}: {this[key]}"
        }
        join(fields, ", ")
    }
}

class Entity <> Printable {
    init = name => {
        .name = name
    }
}
```

### What Was Preserved

LANGUAGESPEC.md mentions classes but without the full semantics and the `<>` operator is completely missing.

### Features Lost

1. **`<>` composition operator** - for mixing in behavior maps
2. **`.field` dot syntax** - for instance fields inside class methods
3. **`this` keyword** - reference to current instance
4. **Class variables** - `ClassName.variable` for shared state
5. **`init` and `deinit` methods** - constructor/destructor conventions
6. **Complete desugaring rules** - how classes expand to regular Flap code

### Fix Required

1. Add `<>` operator to grammar and spec
2. Document class system completely
3. Add `this` keyword
4. Implement or document class desugaring
5. Add class examples

---

## 3. Loop `max` Keyword for Unbounded Iterations

**Status:** ❌ COMPLETELY LOST

### What Was Lost

Loops without known bounds required the `max` keyword:

```flap
// Counter modified in loop - needs max
@ i in 0..<10 max 20 {
    i++
}

// Unknown iteration count - needs max
@ i in read_from_channel() max inf {
    process(i)
}
```

### Why This Matters

- **Safety**: Prevents infinite loops from compiler optimization bugs
- **Clarity**: Makes unbounded loops explicit
- **Verification**: Compiler can check loop termination

### Fix Required

1. Add `max` keyword to loop grammar
2. Document when `max` is required
3. Add compiler checks for missing `max`

---

## 4. Lambda Shorthand `==>`

**Status:** ⚠️ PARTIALLY LOST

### What Was Lost

The `==>` shorthand for no-argument functions was mentioned in old LANGUAGE.md:

```flap
// No-arg function shorthand
hello ==> println("Hello!")
// Desugars to: hello = () => println("Hello!")
```

### Current Status

LANGUAGESPEC.md mentions it briefly but it's not in GRAMMAR.md properly.

### Fix Required

1. Ensure `==>` is in grammar
2. Document desugaring clearly
3. Add examples

---

## 5. Guard Conditions Without `|` Prefix (Context-Sensitive)

**Status:** ❌ PARTIALLY CHANGED

### What Was Lost

The original spec showed guards could omit `|` in clear contexts:

```flap
// Original: conditions as implicit guards
result := x {
    x < 0 -> "negative"
    x == 0 -> "zero"
    x > 0 -> "positive"
}
```

### What Was Changed

New docs require `|` prefix for all guards:

```flap
// New: explicit | required
result := x {
    | x < 0 -> "negative"
    | x == 0 -> "zero"
    | x > 0 -> "positive"
}
```

### Analysis

The new approach is actually clearer and more consistent. This might be an **improvement** rather than a loss, but should be noted.

---

## 6. Defer Statement LIFO Order

**Status:** ✅ PRESERVED (but moved)

The defer statement and its LIFO execution order was preserved in LANGUAGESPEC.md. This is good.

---

## 7. Arena Allocators

**Status:** ✅ PRESERVED

Arena allocators are still documented. No loss here.

---

## 8. Range Operators Documentation

**Status:** ⚠️ INCOMPLETE

### What Was Lost

Clear explanation of range operators:
- `..<` (exclusive) - End value not included: `0..<10` gives 0,1,2,...,9
- `..` (inclusive) - End value included: `0..10` gives 0,1,2,...,10

### Current Status

Present in GRAMMAR.md but less clearly explained.

### Fix Required

Ensure both documents clearly explain the difference.

---

## Summary of Required Fixes

### Critical (Breaking Changes)

1. **Remove `break` and `continue`** - replace with `ret @` syntax
2. **Add `<>` composition operator** - for class mixins
3. **Add `max` keyword for loops** - safety feature

### Important (Feature Completion)

4. **Complete class system documentation** - with `this`, `.field`, etc.
5. **Add `==>` to grammar** - no-arg function shorthand

### Minor (Clarifications)

6. **Clarify guard syntax** - when `|` is required
7. **Better range operator docs** - `..<` vs `..`

---

## Notes

This refactoring revealed that Flap's philosophy of **minimal keywords** was violated by adding `break` and `continue`. The original `ret @` design was more elegant and powerful.

The class system with `<>` was a significant feature that should not have been lost.
