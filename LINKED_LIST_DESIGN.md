# Linked List Design for Flap

## Problem Statement

The current array-based list implementation has several issues:
1. **Cons (::) is expensive**: O(n) - copies entire list
2. **Register clobbering**: Large memory copies cause register/stack issues
3. **Not truly functional**: Copying large data structures is error-prone
4. **Unnatural for Lisp-style**: Cons should be O(1)

## Proposed Solution: True Linked Lists

### Memory Layout

**Linked List Node (Cons Cell):**
```
[head: float64][tail: float64]
Size = 16 bytes (fixed)
```

Where:
- `head`: The first element (any float64 value)
- `tail`: Pointer to next node, stored as float64 bits (0.0 = empty/nil)

**Empty List:**
- Represented as NULL pointer (0.0 when stored as float64)
- `[]` literal evaluates to 0.0

### Examples

```
[1, 2, 3] becomes:
  Node1: [1.0][ptr_to_Node2]
  Node2: [2.0][ptr_to_Node3]  
  Node3: [3.0][0.0]

[] = 0.0

1 :: [] = Node1: [1.0][0.0]

2 :: (1 :: []) = Node2: [2.0][ptr_to_Node1]
                 Node1: [1.0][0.0]
```

### Operations Complexity

| Operation | Current (Array) | Linked List |
|-----------|----------------|-------------|
| Cons (::) | O(n) - copy all | O(1) - alloc one node |
| Head (^)  | O(1) | O(1) |
| Tail (_)  | O(n) - copy all | O(1) |
| Index [i] | O(1) | O(i) - walk links |
| Length (#)| O(1) | O(n) - walk links |

### Trade-offs

**Advantages:**
- ✅ Cons is O(1) - just allocate 16 bytes
- ✅ Tail is O(1) - just return the tail pointer
- ✅ True functional semantics (structure sharing)
- ✅ No large memory copies
- ✅ Reduced register pressure
- ✅ Natural for recursive list processing
- ✅ Matches Lisp/Scheme/ML semantics perfectly

**Disadvantages:**
- ❌ Indexing becomes O(n) instead of O(1)
- ❌ Length becomes O(n) instead of O(1)
- ❌ More memory overhead (16 bytes per element vs 8 bytes)
- ❌ Poor cache locality (nodes scattered in memory)

### When to Use Each

**Linked Lists (cons-built):**
```flap
// Functional list building
list := 1 :: 2 :: 3 :: []

// Recursive processing
process := lst => {
    #lst == 0 -> ret
    head := ^lst
    tail := _lst
    println(head)
    process(tail)
}
```

**Array Literals (for random access):**
```flap
// If you need fast indexing, use array literals
// These could remain array-based internally
arr := [1, 2, 3, 4, 5]
println(arr[3])  // O(1) access
```

### Implementation Strategy

**Phase 1: Add Linked List Support**
1. Add `_flap_list_cons_linked` runtime function
2. Update cons operator (`::`) to generate linked list nodes
3. Update head operator (`^`) to read from node.head
4. Update tail operator (`_`) to return node.tail
5. Update length operator (`#`) to walk and count nodes

**Phase 2: Update List Literals**
Option A: Keep array-based for literals `[1,2,3]`
Option B: Convert literals to linked lists at compile time
Option C: Convert literals to linked lists at runtime on first cons

**Phase 3: Update Indexing**
- `list[i]` walks i nodes to find element
- Add warning or deprecation for indexing cons-built lists
- Suggest using head/tail decomposition instead

**Phase 4: Performance Optimizations**
- Cache list length if measured
- Use array repr for large literal lists
- Hybrid approach: small lists = linked, large lists = array

## Implementation Details

### Runtime Functions Needed

```c
// Cons: Create a new linked list node
// Returns pointer to new node with head=element, tail=rest
void* _flap_list_cons_linked(double element, void* rest);

// Head: Get first element
// Returns NaN if list is empty (NULL)
double _flap_list_head_linked(void* list);

// Tail: Get rest of list
// Returns NULL (0.0) if list is empty or single element
void* _flap_list_tail_linked(void* list);

// Length: Count nodes
// Returns 0 for empty list (NULL)
int64_t _flap_list_length_linked(void* list);

// Index: Walk to i-th element
// Returns NaN if index out of bounds
double _flap_list_index_linked(void* list, int64_t index);
```

### Codegen Changes

**Cons Operator (::):**
```go
case "::":
    // Allocate 16-byte node from arena
    // node = alloc(16)
    // node[0] = element (xmm0)
    // node[8] = rest (xmm1 as pointer)
    // return node pointer in xmm0
```

**Head Operator (^):**
```go
case "^":
    // Load pointer from xmm0
    // If NULL, return NaN
    // Otherwise load [ptr+0] into xmm0
```

**Tail Operator (_):**
```go
case "_":
    // Load pointer from xmm0
    // If NULL, return 0.0 (NULL)
    // Otherwise load [ptr+8] into xmm0
```

### Migration Path

**Option 1: Make all lists linked (breaking change)**
- Simplest implementation
- Best performance for functional code
- Breaks code that relies on O(1) indexing

**Option 2: Hybrid approach (compatible)**
- List literals `[...]` stay array-based
- Cons operations `::` create linked lists
- Have both `_flap_list_array_*` and `_flap_list_linked_*` runtime
- Type flag distinguishes them

**Option 3: Lazy conversion**
- Start with array representation
- Convert to linked on first cons operation
- Best of both worlds, but complex

## Recommendation

**Go with Option 1: Pure Linked Lists**

Reasoning:
1. Flap is a functional language - embrace it
2. Indexing is anti-pattern in functional programming
3. Simplifies compiler (one representation)
4. Matches LANGUAGE.md specification perfectly
5. Fixes the current cons/register issues
6. Users who need random access can use arenas + raw pointers

If indexing performance is critical, users can:
- Use maps with explicit indices: `{0: val0, 1: val1, ...}`
- Use arena + pointer arithmetic: `arena { arr := alloc(n*8); arr[i] <- val }`
- Use C arrays via FFI

## Decision

**YES - Make Flap lists always linked lists.**

This aligns with:
- Functional programming principles
- Lisp/Scheme/ML semantics
- The language's immutable-by-default philosophy
- The arena memory management model
- The "explicit over implicit" design principle

Users expecting array-like performance should be steered toward:
- Explicit data structures (maps, arenas, C arrays)
- Understanding that Flap prioritizes functional style
