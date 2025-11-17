# Grammar and Language Spec Improvements for Complete Examples

**Date:** 2025-11-17  
**Purpose:** Enable completion of all example tests in example_test.go

## Current Status

All existing example tests pass, but some are simplified (e.g., TestQuickSort) because they lack certain list/collection operations. This document proposes minimal additions to GRAMMAR.md and LANGUAGESPEC.md to enable complete, idiomatic implementations.

## Analysis of Missing Features

### 1. List/Map Built-in Properties and Methods

**Current State:** Mentioned in LANGUAGESPEC.md but not formalized in GRAMMAR.md

**Needed for:** TestQuickSort, TestListOperations, and future collection algorithms

**Proposal:** Add to grammar:

```ebnf
postfix_op      = "[" expression "]"
                | "." ( identifier | integer )
                | "(" [ argument_list ] ")"
                | "!"
                | match_block
                | property_access ;     // NEW

property_access = "." built_in_property ;  // NEW

built_in_property = "length"              // NEW
                  | "keys"                 // NEW
                  | "values"               // NEW
                  | "size"                 // NEW (alias for length)
                  | "empty" ;              // NEW (check if map is {})
```

**Semantics:**

Since everything is `map[uint64]float64`:

- `.length` → Number of entries in the map (returns {0: count})
- `.keys` → List (map) of keys: {0: key1, 1: key2, ...}
- `.values` → List (map) of values: {0: val1, 1: val2, ...}
- `.size` → Alias for `.length`
- `.empty` → Returns {0: 1.0} if map is {}, else {0: 0.0}

**Examples:**

```flap
numbers = [1, 2, 3, 4, 5]
count = numbers.length      // {0: 5.0}
is_empty = numbers.empty    // {0: 0.0} (false)

config = {port: 8080, host: "localhost"}
keys = config.keys          // {0: hash("port"), 1: hash("host")}
```

### 2. List Construction Operators

**Current State:** `::` (cons operator) mentioned but not in grammar

**Needed for:** Recursive list building, functional programming patterns

**Proposal:** Add to grammar:

```ebnf
additive_expr   = cons_expr { ("+" | "-") cons_expr } ;

cons_expr       = multiplicative_expr { "::" multiplicative_expr } ;  // NEW
```

**Semantics:**

```flap
// Prepend element to list
list = 1 :: [2, 3]       // [1, 2, 3] = {0: 1.0, 1: 2.0, 2: 3.0}

// Build list recursively
build_list = (n, acc) => {
    | n == 0 -> acc
    ~> build_list(n - 1, n :: acc)
}
```

**Implementation:** Map merge operation where left side gets key 0, right side keys shift up by 1.

### 3. List Slicing

**Current State:** Partially mentioned with ranges like `0..<n` in loops

**Needed for:** Sublist extraction, partitioning

**Proposal:** Add to grammar:

```ebnf
postfix_op      = "[" slice_or_index "]"
                | "." ( identifier | integer )
                | "(" [ argument_list ] ")"
                | "!"
                | match_block ;

slice_or_index  = expression [ ".." expression ]    // NEW
                | expression "..<" expression ;      // NEW
```

**Semantics:**

```flap
arr = [1, 2, 3, 4, 5]

arr[0..2]        // [1, 2, 3] - inclusive range {0: 1.0, 1: 2.0, 2: 3.0}
arr[1..<4]       // [2, 3, 4] - half-open range {0: 2.0, 1: 3.0, 2: 4.0}
arr[2..-1]       // [3, 4, 5] - to end
```

**Implementation:** Create new map with subset of key-value pairs, renumbering from 0.

### 4. List Comprehensions (Optional - Higher Priority)

**Current State:** Not present

**Needed for:** Cleaner quicksort, filter/map operations

**Proposal:** Add to grammar:

```ebnf
primary_expr    = identifier
                | number
                | string
                | fstring
                | list_literal
                | list_comprehension    // NEW
                | map_literal
                | lambda_expr
                | enet_address
                | instance_field
                | this_expr
                | "(" expression ")"
                | "??"
                | unsafe_expr
                | arena_expr
                | "???" ;

list_comprehension = "[" expression "|" identifier "in" expression [ "|" expression ] "]" ;  // NEW
```

**Semantics:**

```flap
// Filter
evens = [x | x in numbers | x % 2 == 0]

// Map
doubled = [x * 2 | x in numbers]

// Combined
big_evens = [x | x in numbers | x % 2 == 0 | x > 10]
```

**Alternative simpler syntax (using existing pipe operators):**

```flap
// Using pipe operators (no grammar change needed!)
evens = numbers | filter(x => x % 2 == 0)
doubled = numbers | map(x => x * 2)
```

**Recommendation:** Skip list comprehensions, use pipe operators with built-in `filter`, `map`, `reduce` functions instead.

### 5. Built-in Higher-Order Functions

**Current State:** Not formally specified

**Needed for:** Functional programming patterns, cleaner algorithms

**Proposal:** Add to LANGUAGESPEC.md built-in functions section:

```flap
// Built-in functions for collections
filter(list, predicate)  // Returns new list with elements where predicate(x) is true
map(list, transform)     // Returns new list with transform(x) applied to each element
reduce(list, fn, init)   // Folds list with fn(acc, x), starting from init
fold(list, fn, init)     // Alias for reduce
partition(list, pred)    // Returns {0: matching, 1: non-matching}
take(list, n)            // First n elements
drop(list, n)            // Skip first n elements
reverse(list)            // Reverse order
sort(list)               // Sort by value (ascending)
sort_by(list, fn)        // Sort using fn as comparator
```

**Implementation:** These are compiler intrinsics or can be runtime library functions written in Flap.

**Examples:**

```flap
// Filter using pipe
evens = numbers | filter(x => x % 2 == 0)

// Partition for quicksort
parts = rest | partition(x => x < pivot)
smaller = parts[0]
larger = parts[1]
```

## Complete QuickSort Example

With the proposed additions, TestQuickSort can be implemented fully:

### Option A: Using comprehensions (if added)

```flap
quicksort = arr => {
    | arr.length <= 1 -> arr
    ~> {
        pivot = arr[0]
        rest = arr[1..-1]
        smaller = [x | x in rest | x < pivot]
        larger = [x | x in rest | x >= pivot]
        quicksort(smaller) + [pivot] + quicksort(larger)
    }
}
```

### Option B: Using built-in functions (recommended)

```flap
quicksort = arr => {
    | arr.length <= 1 -> arr
    ~> {
        pivot = arr[0]
        rest = arr[1..<arr.length]
        parts = rest | partition(x => x < pivot)
        smaller = parts[0]
        larger = parts[1]
        quicksort(smaller) + [pivot] + quicksort(larger)
    }
}
```

### Option C: Using cons and recursion (most Flap-like)

```flap
quicksort = arr => {
    | arr.empty -> arr
    | arr.length == 1 -> arr
    ~> {
        pivot = arr[0]
        rest = arr[1..<arr.length]
        
        // Partition manually
        partition_helper = (lst, smaller, larger) => {
            | lst.empty -> {0: smaller, 1: larger}
            ~> {
                head = lst[0]
                tail = lst[1..<lst.length]
                head < pivot {
                    partition_helper(tail, smaller :: head, larger)
                } ~> {
                    partition_helper(tail, smaller, larger :: head)
                }
            }
        }
        
        parts = partition_helper(rest, [], [])
        quicksort(parts[0]) + [pivot] + quicksort(parts[1])
    }
}
```

## Minimal Recommendation

To complete all examples with minimal grammar/spec changes:

### Must Have:

1. **Built-in properties** (`.length`, `.keys`, `.values`, `.empty`)
   - Add formal grammar rules
   - Document semantics clearly
   - Implement in codegen

2. **Slice syntax** (`arr[1..<n]`, `arr[0..n]`)
   - Add grammar rules for postfix_op
   - Implement range extraction in codegen

3. **Cons operator** (`::` for prepending to lists)
   - Add to expression grammar at appropriate precedence
   - Implement map merge with key shifting

### Should Have:

4. **Built-in HOFs** (`filter`, `map`, `reduce`, `partition`)
   - Add to LANGUAGESPEC.md built-ins
   - Implement as compiler intrinsics or stdlib

### Nice to Have (can defer):

5. **List comprehensions**
   - Wait until more user feedback
   - Pipe + HOFs covers 90% of use cases

## Implementation Order

1. **Phase 1:** Built-in properties (`.length`, `.empty`)
   - Lexer: No changes needed
   - Parser: Extend `postfix_op` handling
   - Codegen: Emit map size calculation code

2. **Phase 2:** Slice syntax (`[start..<end]`)
   - Lexer: Recognize `..` and `..<` as tokens
   - Parser: Extend index expression handling
   - Codegen: Emit map subset extraction code

3. **Phase 3:** Cons operator (`::`)
   - Lexer: Recognize `::` token
   - Parser: Add to expression precedence table (between additive and multiplicative)
   - Codegen: Emit map prepend code

4. **Phase 4:** Built-in HOFs
   - Add function definitions to LANGUAGESPEC.md
   - Implement in Flap stdlib (self-hosted)
   - Or add as compiler intrinsics for performance

5. **Phase 5:** Update example_test.go
   - Implement complete quicksort
   - Add more sophisticated examples
   - Demonstrate idiomatic Flap patterns

## Grammar Changes Summary

```ebnf
// In lexical tokens (lexer.go):
CONS       = "::" ;
RANGE      = ".." ;
RANGE_EXCL = "..<" ;

// In expression grammar:
additive_expr   = cons_expr { ("+" | "-") cons_expr } ;
cons_expr       = multiplicative_expr { "::" multiplicative_expr } ;

// In postfix operations:
postfix_op      = "[" slice_or_index "]"
                | "." ( identifier | integer | built_in_property )
                | "(" [ argument_list ] ")"
                | "!"
                | match_block ;

slice_or_index  = expression [ ( ".." | "..<" ) expression ] ;

built_in_property = "length" | "keys" | "values" | "size" | "empty" ;
```

## Testing Strategy

For each new feature, add tests to example_test.go:

```go
func TestListSlicing(t *testing.T) {
    code := `
arr = [1, 2, 3, 4, 5]
slice = arr[1..<4]
printf("%v\n", slice[0])  // Should print 2
`
    output := compileAndRun(t, code)
    if !strings.Contains(output, "2") {
        t.Errorf("Expected '2', got: %s", output)
    }
}

func TestConsOperator(t *testing.T) {
    code := `
list = 1 :: [2, 3]
printf("%v %v %v\n", list[0], list[1], list[2])
`
    output := compileAndRun(t, code)
    if !strings.Contains(output, "1 2 3") {
        t.Errorf("Expected '1 2 3', got: %s", output)
    }
}

func TestBuiltinProperties(t *testing.T) {
    code := `
arr = [1, 2, 3]
printf("%v\n", arr.length)
`
    output := compileAndRun(t, code)
    if !strings.Contains(output, "3") {
        t.Errorf("Expected '3', got: %s", output)
    }
}
```

## Conclusion

The minimal additions needed are:

1. **Built-in properties** - essential for any collection algorithm
2. **Slice syntax** - common idiom in modern languages
3. **Cons operator** - enables functional programming style
4. **Built-in HOFs** - optional but highly valuable

These additions maintain Flap's philosophy:
- ✅ Everything still `map[uint64]float64`
- ✅ Minimal syntax additions
- ✅ No type system complexity
- ✅ Direct codegen possible
- ✅ Consistent with existing patterns

The changes enable complete, idiomatic implementations of common algorithms while preserving Flap's radical simplicity.
