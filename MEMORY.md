# Memory Management in Flap and Flapc

This document explains all aspects of memory management in the Flap programming language and its compiler implementation.

## Philosophy

Flap follows these core memory management principles:

1. **No Garbage Collector** - Manual memory management for predictable performance
2. **Arena Allocators by Default** - Scope-based memory cleanup
3. **C FFI as Escape Hatch** - Direct malloc/free only when explicitly requested by user
4. **Immutable by Default** - Reduces memory management complexity
5. **Explicit, Never Hidden** - All memory operations are visible in code

## Memory Allocation Strategies

### 1. Arena Allocation (Preferred)

**When to use:** Almost always for Flap code

```flap
arena {
    data := alloc(1024)        // Allocate from arena
    more := alloc(256)          // Another allocation
    process(data, more)
}  // All arena memory freed here automatically
```

**How it works:**
- Each `arena { }` block creates a new arena in the meta-arena table
- `alloc(size)` allocates from the current arena
- When the block exits, all arena memory is freed in one operation
- Nested arenas are supported (tracked in meta-arena)

**Global Default Arena (arena0):**
- There is always a global default arena available: `_flap_default_arena_struct`
- Initialized at program startup with 4KB of memory (allocated with malloc)
- Used for implicit allocations like cons operator (`::`) and list building
- The main function has an implicit arena block - all code in main executes within arena0
- Grows automatically via realloc when needed
- This ensures cons and list operations work without requiring explicit arena blocks
- Lists like `[1, 2, 3]` in main use arena0 automatically

**Implementation details:**
- Meta-arena: Global table tracking all active arenas
- Arena structure: `{ base_ptr, current_ptr, capacity }`
- Growing: If arena fills, `realloc()` is called to expand it
- Alignment: All allocations are 8-byte aligned for float64 storage

**Benefits:**
- Zero fragmentation
- Fast allocation (bump pointer)
- Automatic cleanup
- No use-after-free bugs

### 2. C FFI Malloc (Explicit User Request Only)

**When to use:** Only when user explicitly calls C functions

```flap
import libc as c

ptr := c.malloc(1024 as uint64)
// Use ptr...
c.free(ptr as ptr)
```

**Rules:**
- Only available through C FFI: `c.malloc()`, `c.realloc()`, `c.free()`
- User must manually free memory
- Flapc compiler NEVER generates direct malloc calls for user code
- Only used for C interop where C library expects malloc'd memory

### 3. Compiler-Internal Malloc (Implementation Detail)

**When the compiler uses malloc:**

1. **Arena metadata allocation** - When creating a new arena, compiler calls malloc to allocate the arena's memory pool
2. **Arena growth** - When an arena runs out of space, compiler calls realloc to grow it
3. **List/map literals** (temporary, should migrate to arena) - Currently uses malloc for list `[1,2,3]` and map `{a: 10}` literals

**DO NOT use malloc in codegen for:**
- ❌ List literals (should use arena or .rodata for immutable)
- ❌ String literals (use .rodata)
- ❌ Map literals (should use arena or .rodata for immutable)
- ❌ Temporary values (use stack or registers)
- ❌ Closure environments (should use arena)
- ❌ Loop variables (use stack)
- ❌ Function calls (use stack for arguments)

**Correct pattern:**
```go
// WRONG: Direct malloc in codegen
fc.trackFunctionCall("malloc")
fc.eb.GenerateCallInstruction("malloc")

// RIGHT: Use arena allocation (generates code to call alloc())
// Flap code: arena { data := alloc(1024) }
// Compiler generates: call to current arena's bump allocator

// RIGHT: User explicitly requests C malloc
// Flap code: ptr := c.malloc(1024 as uint64)
// Compiler generates: call to libc malloc via C FFI
```

## Internal Compiler Structures

### List Representation

**Memory layout:**
```
[length: float64][element0: float64][element1: float64]...
Size = 8 + (length × 8) bytes
```

**Allocation strategy:**
- **Immutable lists** (`list = [1,2,3]`): Store in `.rodata` section (no malloc)
- **Mutable lists** (`list := [1,2,3]`): Allocate in current arena
- **Dynamic lists** (built with cons): Allocate in current arena

### Map Representation

**Memory layout:**
```
[count: float64][key0: float64][val0: float64][key1: float64][val1: float64]...
Size = 8 + (count × 16) bytes
```

**Allocation strategy:**
- **Immutable maps** (`map = {a: 10}`): Store in `.rodata` section (no malloc)
- **Mutable maps** (`map := {a: 10}`): Allocate in current arena
- **Dynamic maps** (updated at runtime): Allocate in current arena

### String Representation

**Memory layout:**
```
[length: float64][char0: float64][char1: float64]...
Each char is a UTF-8 code point stored as float64
```

**Allocation strategy:**
- **String literals** (`"hello"`): Store in `.rodata` section (no malloc)
- **Dynamic strings** (f-strings, concatenation): Allocate in current arena
- **C strings** (for C FFI): Convert to null-terminated, malloc'd (user calls c.free)

### Cons Operator (`::`)

**Semantic:** Pure function that creates a new list without modifying original

```flap
list1 := [2, 3]
list2 := 1 :: list1    // Creates [1, 2, 3], list1 unchanged
```

**Implementation:**
```
1. Allocate new list from default arena (_flap_default_arena_struct): size = 8 + (old_length + 1) * 8
2. Write new length: old_length + 1
3. Write new element at index 0
4. Copy old elements to indices 1..N
5. Return pointer to new list
```

**Memory Strategy:** The cons operator uses the global default arena (arena0) for allocations. This means cons-built lists persist for the program lifetime unless freed explicitly. For temporary list building, wrap your code in an explicit `arena { }` block - then cons will use that arena instead.

## Meta-Arena System

**Purpose:** Track all active arenas to support nested arena blocks

**Structure:**
```c
// Meta-arena table (global)
struct MetaArena {
    Arena* arenas[256];    // Stack of active arenas
    int depth;             // Current nesting depth
};

// Individual arena
struct Arena {
    void* base;            // Start of arena memory
    void* current;         // Current allocation point
    size_t capacity;       // Total arena size
    Arena* parent;         // Parent arena (for nesting)
};
```

**Operations:**
- `arena_create()` - Malloc initial arena memory, push onto meta-arena stack
- `arena_alloc(size)` - Bump pointer allocation from current arena
- `arena_grow(new_capacity)` - Realloc arena memory if needed
- `arena_destroy()` - Free arena memory, pop from meta-arena stack

## Alignment Requirements

**Why alignment matters:**
- x86-64 requires 16-byte stack alignment before `call` instructions
- SSE/AVX instructions require aligned memory accesses
- Cache line alignment (64 bytes) prevents false sharing in parallel code

**Alignment rules:**
1. **Stack:** Must be 16-byte aligned before function calls
2. **Arena allocations:** 8-byte aligned (for float64)
3. **CStruct fields:** Follow C ABI rules (use `packed` or `aligned` attributes)
4. **List/Map data:** 8-byte aligned (natural for float64 arrays)

## Memory Safety

### Compiler Checks

1. **Use after move:** Detect within same expression (soft move semantics)
2. **Immutable updates:** Reject `list[0] <- x` if `list = [...]` (immutable)
3. **Arena scope:** Reject `alloc()` outside `arena { }` block
4. **Type casts:** Validate pointer casts to prevent invalid memory access

### Runtime Checks (Optional)

- **Bounds checking:** Can enable for array/list access (performance cost)
- **NULL pointer checks:** Optional for C FFI pointers
- **Arena overflow:** Detect and grow arena automatically

### Unsafe Blocks

Direct memory access allowed in `unsafe { }` blocks:

```flap
unsafe {
    rax <- ptr as ptr
    rbx <- 42
    [rax + 0] <- rbx    // Direct memory write
    rcx <- [rax + 8]    // Direct memory read
}
```

**No safety checks in unsafe blocks** - user takes full responsibility.

## Common Patterns

### Pattern 1: Temporary Data

```flap
process := data => {
    arena {
        temp := alloc(1024)
        // Work with temp
        result := compute(temp)
        result  // Return value (not temp pointer!)
    }
    // temp is freed here
}
```

### Pattern 2: Long-Lived Data

```flap
import libc as c

state := {
    buffer := c.malloc(1048576 as uint64),  // 1 MB
    size := 1048576
}

// Use state.buffer...

// Later:
c.free(state.buffer as ptr)
```

### Pattern 3: Per-Frame Game Loop

```flap
@ frame in 0..<1000 {
    arena {
        // All per-frame allocations
        visible := alloc(entity_count * 8)
        commands := alloc(command_count * 16)
        
        // Process frame
        update(visible, commands)
    }
    // Zero-cost cleanup, no fragmentation
}
```

### Pattern 4: Building Lists Immutably

```flap
// Build list with cons (no mutation)
build_list := n => {
    n == 0 {
        -> []
    }
    ~> n :: build_list(n - 1)
}

list := build_list(100)  // [100, 99, 98, ..., 1]
// Each cons allocates in current arena
```

### Pattern 5: Main Function with Implicit Arena

```flap
main ==> {
    // All code in main has implicit arena (arena0)
    x := [1, 2, 3]     // Uses arena0
    y := 4 :: x        // Cons uses arena0
    z := [10, 20, 30]  // Uses arena0
    
    // No need for explicit arena { } in main
    // Everything cleaned up at program exit
}
```

## Migration Plan

**Current state:** Some codegen paths use malloc directly

**Target state:** All user-visible allocations use arenas

**Steps:**
1. ✅ Document memory philosophy (this file)
2. ✅ Fix cons operator to use arena allocation (uses default arena)
3. ⬜ Fix list literals to use arena (mutable) or .rodata (immutable)
4. ⬜ Fix map literals to use arena (mutable) or .rodata (immutable)
5. ⬜ Fix f-string compilation to use arena allocation
6. ⬜ Fix closure environment to use arena allocation
7. ⬜ Add comments where malloc is tempting but wrong
8. ✅ Add arena overflow detection and growth (already implemented in flap_arena_alloc)
9. ⬜ Optimize arena allocation (faster bump pointer)
10. ⬜ Add arena pooling for common sizes

## Performance Considerations

**Arena allocation advantages:**
- Allocation: O(1) bump pointer (vs O(log n) malloc)
- Deallocation: O(1) arena free (vs O(log n) per-object free)
- Cache-friendly: Sequential allocations are spatially local
- No fragmentation: All memory freed together

**Arena allocation disadvantages:**
- Memory usage: Can't free individual objects (not a problem for short-lived data)
- Lifetime: All objects must have same lifetime (use nested arenas for different lifetimes)

**When NOT to use arenas:**
- Long-lived data with complex lifetimes (use C malloc via FFI)
- Data shared across processes (use shared memory via FFI)
- Data that needs to outlive the program (use files)

## Debugging Memory Issues

**Tools:**
- `valgrind --leak-check=full ./program` - Detect memory leaks
- `valgrind --tool=massif ./program` - Memory profiling
- `gdb ./program` - Debug segfaults
- `objdump -d ./program` - Inspect generated assembly

**Common issues:**
1. **Segfault in malloc** - Stack alignment problem (must be 16-byte aligned before call)
2. **Memory leak** - Forgot to free C malloc'd memory
3. **Use after free** - Returned pointer to arena memory after arena freed
4. **Double free** - Called c.free twice on same pointer
5. **Corruption** - Overwrote list/map header (length/count field)

## References

- **Flap LANGUAGE.md**: Language-level memory features (arena, alloc, c.malloc)
- **LEARNINGS.md**: Hard-earned lessons about malloc, stack alignment, pthreads
- **TODO.md**: Current memory-related bugs and improvements
- **codegen.go**: x86-64 code generation including memory operations
- **arm64_codegen.go**: ARM64 code generation including memory operations
- **flap_runtime.go**: Arena allocator implementation (meta-arena, growth, alignment)

---

**Last updated:** 2025-11-14

**Status:** Active document - update when memory strategy changes
