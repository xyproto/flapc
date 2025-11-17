# Documentation Liberties Taken (Violations of Flap Philosophy)

**Date:** 2025-11-17  
**Commits:** bd30ab6, 2f92e4f, 2489052  
**Files Affected:** GRAMMAR.md, LANGUAGESPEC.md

This document catalogs all incorrect statements introduced in recent documentation that violate Flap's core philosophy: **Everything is `map[uint64]float64`**.

---

## The Truth About Flap Types

### Core Philosophy

**Everything in Flap is `map[uint64]float64`.**

Not "everything is represented as" or "everything maps to" — **everything IS a `map[uint64]float64`**.

### Correct Representations

```flap
42              // {0: 42.0}
"Hello"         // {0: 72.0, 1: 101.0, 2: 108.0, 3: 108.0, 4: 111.0}
[1, 2, 3]       // {0: 1.0, 1: 2.0, 2: 3.0}
{x: 10, y: 20}  // {hash("x"): 10.0, hash("y"): 20.0}
[]              // {} (universal empty map)
```

**Key Points:**
- Numbers are NOT "64-bit IEEE 754 floating-point values" — they are `map[uint64]float64` with one entry `{0: value}`
- Strings are NOT "UTF-8 encoded text" — they are `map[uint64]float64` with indices as keys and character codes as values
- There is no "single entry", "byte index", "field hash" — these are just keys in the universal map
- All values have the SAME underlying type: `map[uint64]float64`

---

## Liberty #1: Numbers as IEEE 754 (GRAMMAR.md:329)

### ❌ What Was Written

```
### Numbers

Numbers are 64-bit IEEE 754 floating-point values:
```

### ✅ What Should Be Written

```
### Numbers

Numbers are `map[uint64]float64` with a single entry at key 0:

Example:
42        // {0: 42.0}
3.14      // {0: 3.14}
-10.5     // {0: -10.5}
```

### Why This Matters

Saying "numbers are IEEE 754 values" suggests they are primitive types, breaking the unified type system. Numbers in Flap are maps, just like everything else.

---

## Liberty #2: Strings as UTF-8 (GRAMMAR.md:351)

### ❌ What Was Written

```
### Strings

Strings are UTF-8 encoded text:
```

### ✅ What Should Be Written

```
### Strings

Strings are `map[uint64]float64` where keys are indices and values are character codes:

Example:
"Hello"   // {0: 72.0, 1: 101.0, 2: 108.0, 3: 108.0, 4: 111.0}
"A"       // {0: 65.0}
""        // {} (empty map)

Note: Character codes happen to align with ASCII/UTF-8 for convenience, 
but strings are NOT "UTF-8 encoded text" — they are ordered maps.
```

### Why This Matters

Saying "strings are UTF-8 encoded text" implies they are a special string type with encoding. They're not. They're maps from indices to character codes, which happen to use Unicode values.

---

## Liberty #3: Type System Descriptions (LANGUAGESPEC.md:254-257)

### ❌ What Was Written

```
Every value in Flap is this map:
- **Numbers**: Map with single entry (0 → float64)
- **Strings**: Map from byte index → byte value
- **Lists**: Map from index → element value
- **Objects**: Map from field hash → field value
- **Functions**: Map containing code pointer and closures
```

### ✅ What Should Be Written

```
Every value in Flap IS `map[uint64]float64`:

- **Numbers**: `{0: number_value}`
- **Strings**: `{0: char0, 1: char1, 2: char2, ...}`
- **Lists**: `{0: elem0, 1: elem1, 2: elem2, ...}`
- **Objects**: `{key_hash: value, ...}`
- **Functions**: `{0: code_pointer, 1: closure_data, ...}`

There are no special cases. No "single entry maps", no "byte indices", 
no "field hashes" — just uint64 keys and float64 values in every case.
```

### Why This Matters

The phrasing "Map with single entry (0 → float64)" suggests numbers contain a float64, which is wrong. Numbers ARE maps. The phrasing "byte index → byte value" and "field hash" suggests special semantics, but there's only one semantic: `map[uint64]float64`.

---

## Liberty #4: UTF-8 Byte Arrays (LANGUAGESPEC.md:133, 845)

### ❌ What Was Written

```flap
bytes = text.bytes   // UTF-8 byte array
```

### ✅ What Should Be Written

```flap
bytes = text.bytes   // Map of byte values {0: byte0, 1: byte1, ...}
```

### Why This Matters

Saying "UTF-8 byte array" implies a special array type. It's a map. Always a map.

---

## Liberty #5: IEEE 754 Compatibility (LANGUAGESPEC.md:907)

### ❌ What Was Written

```
### NaN Encoding

Errors are encoded in NaN values:
- Error string stored in NaN payload
- Zero overhead when no error
- Compatible with IEEE 754
```

### ✅ What Should Be Written

```
### NaN Encoding

Errors are encoded in special map entries:
- Error information stored in map keys/values
- Zero overhead when no error
- Uses the underlying map representation

Note: While the float64 values in the map happen to be IEEE 754,
the error encoding is a MAP operation, not a float operation.
```

### Why This Matters

Error encoding is a map-level feature, not an IEEE 754 feature. Even if the underlying storage uses IEEE 754 floats, the abstraction is pure map.

---

## Liberty #6: Empty Map as "Universal Empty Value"

### ❌ What Was Written (implied)

The documentation correctly states `[]` is an empty map, but doesn't emphasize this strongly enough in context.

### ✅ What Should Be Emphasized

```
[]  // This is {}, the empty map. 
    // Not "empty list", not "empty array", not "null"
    // Just an empty map[uint64]float64
```

### Why This Matters

Many languages have special null/nil/empty values. Flap doesn't. It has one type, and the empty form of that type is `{}`.

---

## Root Cause Analysis

### Why These Liberties Were Taken

1. **Convenience**: Saying "numbers are floats" is shorter than "numbers are maps with one entry"
2. **Familiarity**: IEEE 754 and UTF-8 are familiar concepts
3. **Implementation Detail Leakage**: The maps internally store IEEE 754 doubles, but this is an implementation detail
4. **Pedagogical Shortcut**: Easier to explain "strings are UTF-8" than the map representation

### Why They Violate Flap's Philosophy

Flap's radical simplification is its **defining characteristic**:
- **One type**: `map[uint64]float64`
- **One representation**: Ordered map
- **No exceptions**: Numbers, strings, lists, objects — all the same

These liberties undermine this by:
- Suggesting there are multiple types (numbers, strings, etc.)
- Implying special encoding/representation for different "types"
- Creating conceptual categories that don't exist in Flap

---

## Impact Assessment

### Severity: **HIGH**

These inaccuracies:
1. **Mislead new users** about Flap's core philosophy
2. **Violate the language specification** (everything is map)
3. **Create false mental models** that will cause confusion
4. **Contradict the implementation** (which is map-only)

### Affected Documents

- `GRAMMAR.md` - Lines 329, 351
- `LANGUAGESPEC.md` - Lines 133, 254-257, 845, 907

### Required Action

**All instances must be corrected** to accurately reflect Flap's universal map type system.

---

## Correction Checklist

- [ ] GRAMMAR.md Line 329: Numbers are maps, not IEEE 754 primitives
- [ ] GRAMMAR.md Line 351: Strings are maps, not UTF-8 text
- [ ] LANGUAGESPEC.md Line 254-257: Rewrite type descriptions to emphasize map uniformity
- [ ] LANGUAGESPEC.md Line 133: Remove "UTF-8 byte array" terminology
- [ ] LANGUAGESPEC.md Line 845: Remove "UTF-8 byte array" terminology
- [ ] LANGUAGESPEC.md Line 907: Clarify NaN encoding is map-level, not float-level
- [ ] Add prominent section in both docs: "Everything is map[uint64]float64 (No Exceptions)"

---

## Recommended Fix

Add this section prominently at the start of both GRAMMAR.md and LANGUAGESPEC.md:

```markdown
## ⚠️ CRITICAL: The Universal Type

Flap has exactly ONE type: `map[uint64]float64`

Not "represented as" or "backed by" — every value IS this map:

```flap
42              // {0: 42.0}
"Hello"         // {0: 72.0, 1: 101.0, 2: 108.0, 3: 108.0, 4: 111.0}
[1, 2, 3]       // {0: 1.0, 1: 2.0, 2: 3.0}
{x: 10}         // {hash("x"): 10.0}
[]              // {}
```

There are NO special types, NO primitives, NO exceptions.
Everything is a map from uint64 to float64.

This is not an implementation detail — this IS Flap.
```

---

**Status:** Awaiting correction  
**Priority:** Critical (affects core language understanding)  
**Estimate:** 30 minutes to fix all instances
