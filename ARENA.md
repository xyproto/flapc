# Arena Allocator System

## Overview

Flap uses an arena-based memory allocator for all runtime string, list, and map operations. This provides fast, predictable memory allocation with automatic cleanup at scope boundaries.

## High-Level Design

### Memory Allocation Strategy

**Static Data (Read-Only):**
- String literals: stored in `.rodata` section
- Constant lists: `[1, 2, 3]` stored in `.rodata`
- Constant maps: `{x: 10}` stored in `.rodata`
- Never allocated at runtime, zero overhead

**Dynamic Data (Arena-Allocated):**
- String concatenation: `"hello" + "world"`
- Dynamic lists: `[x, y, z]` where x/y/z are variables
- Runtime map construction
- Function closures (future)
- Variadic argument lists

**User-Controlled Allocation:**
- `c.malloc()` - C malloc, user must call `c.free()`
- `c.realloc()` - C realloc
- `c.free()` - C free
- `alloc()` - Flap builtin (uses malloc internally)

### Arena Lifecycle

```
Program Start:
  └─> Initialize global arena (1MB)
      └─> All runtime operations use this arena
          └─> Arena grows automatically via realloc
Program End:
  └─> Free global arena

Future: arena { ... } blocks:
  Block Start:
    └─> Create scoped arena
        └─> All allocations inside use this arena
  Block End:
    └─> Free scoped arena (all allocations freed at once)
```

### Benefits

1. **Speed**: O(1) bump allocation (just increment a pointer)
2. **Predictability**: Deterministic memory usage
3. **No Fragmentation**: Memory is contiguous within an arena
4. **Batch Deallocation**: Free entire arena at once
5. **Game-Friendly**: Perfect for frame-based allocation patterns

## Machine Code Level

### Data Structures

#### Meta-Arena Structure
```c
// _flap_arena_meta: Global variable holding pointer to arena array
uint64_t _flap_arena_meta;         // Pointer to arena_ptr[]

// Meta-arena array (dynamically allocated)
void* arena_ptrs[];                 // Array of pointers to arena structs

// Meta-arena metadata
uint64_t _flap_arena_meta_cap;     // Capacity of arena_ptrs array
uint64_t _flap_arena_meta_len;     // Number of arenas currently allocated
```

#### Arena Structure
```c
// Individual arena (32 bytes)
struct Arena {
    void*    base;         // [offset 0]  Base pointer to arena buffer
    uint64_t capacity;     // [offset 8]  Total arena size in bytes
    uint64_t used;         // [offset 16] Bytes currently used
    uint64_t alignment;    // [offset 24] Allocation alignment (typically 8)
};
```

### Initialization Sequence

At program start, `initializeMetaArenaAndGlobalArena()` executes:

```assembly
# 1. Allocate meta-arena array (8 bytes for 1 pointer)
mov rdi, 8
call malloc
# rax = pointer to meta-arena array (P1)

# 2. Store meta-arena pointer in global variable
lea rbx, [_flap_arena_meta]
mov [rbx], rax              # _flap_arena_meta = P1

# 3. Allocate arena buffer (1MB)
mov rdi, 1048576
call malloc
# rax = arena buffer (P2)
mov r12, rax

# 4. Allocate arena struct (32 bytes)
mov rdi, 32
call malloc
# rax = arena struct (P3)

# 5. Initialize arena struct
mov [rax + 0], r12          # base = P2
mov rcx, 1048576
mov [rax + 8], rcx          # capacity = 1MB
xor rcx, rcx
mov [rax + 16], rcx         # used = 0
mov rcx, 8
mov [rax + 24], rcx         # alignment = 8

# 6. Store arena struct pointer in meta-arena[0]
lea rbx, [_flap_arena_meta]
mov rbx, [rbx]              # rbx = P1 (meta-arena array)
mov [rbx], rax              # P1[0] = P3 (arena struct)
```

**Memory Layout After Initialization:**
```
_flap_arena_meta --> P1 --> [P3, NULL, NULL, ...]
                             |
                             v
                        Arena Struct {
                          base: P2 --> [1MB buffer]
                          capacity: 1048576
                          used: 0
                          alignment: 8
                        }
```

### Allocation Sequence

When allocating N bytes, `flap_arena_alloc(arena_ptr, size)` executes:

```assembly
# Input: rdi = arena_ptr (P3), rsi = size (N)
# Output: rax = allocated pointer

# 1. Load arena fields
mov r8,  [rdi + 0]          # r8  = base (P2)
mov r9,  [rdi + 8]          # r9  = capacity
mov r10, [rdi + 16]         # r10 = used
mov r11, [rdi + 24]         # r11 = alignment

# 2. Align offset
mov rax, r10                # rax = used
add rax, r11                # rax += alignment
sub rax, 1                  # rax += alignment - 1
mov rcx, r11
sub rcx, 1                  # rcx = alignment - 1
not rcx                     # rcx = ~(alignment - 1)
and rax, rcx                # rax = aligned_offset

# 3. Check capacity
mov rdx, rax
add rdx, rsi                # rdx = aligned_offset + size
cmp rdx, r9                 # if (rdx > capacity)
jg  arena_grow              #   goto grow path

# 4. Fast path: allocate
arena_fast:
mov rax, r8                 # rax = base
add rax, r13                # rax = base + aligned_offset
mov rdx, r13
add rdx, r12                # rdx = aligned_offset + size
mov [rbx + 16], rdx         # arena->used = new_offset
jmp arena_done

# 5. Grow path: realloc buffer
arena_grow:
mov rdi, r9
add rdi, r9                 # rdi = capacity * 2
# ... (grow logic, realloc arena buffer)

arena_done:
# rax = allocated pointer
ret
```

### String Concatenation Example

Concatenating `"hello" + "world"`:

```assembly
# Strings in rodata:
str_1: [5.0][0][104.0][1][101.0][2][108.0][3][108.0][4][111.0]  # "hello"
str_2: [5.0][0][119.0][1][111.0][2][114.0][3][108.0][4][100.0]  # "world"

# Concatenation code:
lea r12, [str_1]            # r12 = left string
lea r13, [str_2]            # r13 = right string

# Load lengths
movsd xmm0, [r12]           # xmm0 = 5.0
cvttsd2si r14, xmm0         # r14 = 5 (left length)
movsd xmm0, [r13]
cvttsd2si r15, xmm0         # r15 = 5 (right length)

# Calculate size: 8 + (left_len + right_len) * 16
mov rbx, r14
add rbx, r15                # rbx = 10 (total length)
mov rax, rbx
shl rax, 4                  # rax = 10 * 16 = 160
add rax, 8                  # rax = 168 (total size)

# Allocate from arena
mov rdi, rax                # rdi = 168 (size)
call callArenaAlloc         # Arena allocation
# rax = pointer to new string

# Copy data (omitted for brevity)
# Result: [10.0][0][104.0][1][101.0]...[9][100.0]  # "helloworld"
```

### Dynamic List Creation Example

Creating `[x, y]` where x=10, y=20:

```assembly
# Calculate size: 8 + (2 * 16) = 40 bytes
mov rdi, 40                 # size = 40

# Allocate from arena
lea r11, [_flap_arena_meta] # Step 1: Get meta-arena variable address
mov r11, [r11]              # Step 2: Load meta-arena pointer (P1)
mov r11, [r11]              # Step 3: Load arena[0] pointer (P3)
mov rsi, rdi                # rsi = size
mov rdi, r11                # rdi = arena_ptr
call flap_arena_alloc       # Allocate
# rax = list pointer

# Store count
mov rcx, 2
cvtsi2sd xmm0, rcx
movsd [rax], xmm0           # list[0] = 2.0 (count)

# Store element 0: key=0, value=10
xor rcx, rcx
mov [rax + 8], rcx          # list[8] = 0 (key)
movsd xmm0, [rbp - 16]      # xmm0 = x = 10.0
movsd [rax + 16], xmm0      # list[16] = 10.0 (value)

# Store element 1: key=1, value=20
mov rcx, 1
mov [rax + 24], rcx         # list[24] = 1 (key)
movsd xmm0, [rbp - 24]      # xmm0 = y = 20.0
movsd [rax + 32], xmm0      # list[32] = 20.0 (value)
```

## Implementation Details

### callArenaAlloc() Function

The `callArenaAlloc()` function in `arena.go` is a helper that:

1. Takes size in `rdi`
2. Loads the global arena pointer
3. Calls `flap_arena_alloc` with proper arguments
4. Returns allocated pointer in `rax`

**Critical Pattern (must load twice):**
```go
// Step 1: Load address of _flap_arena_meta variable
fc.out.LeaSymbolToReg("r11", "_flap_arena_meta")  // r11 = &_flap_arena_meta

// Step 2: Load meta-arena pointer from variable
fc.out.MovMemToReg("r11", "r11", 0)               // r11 = *_flap_arena_meta = P1

// Step 3: Load arena struct pointer from meta-arena[0]
fc.out.MovMemToReg("r11", "r11", 0)               // r11 = P1[0] = P3

// Now r11 = arena struct pointer, ready to pass to flap_arena_alloc
```

**Why TWO loads?**
- First load: dereference `_flap_arena_meta` to get meta-arena array pointer
- Second load: dereference `meta_arena[0]` to get arena struct pointer

### Integration Points

**String Concatenation (`_flap_string_concat`):**
```go
// OLD: fc.eb.GenerateCallInstruction("malloc")
// NEW:
fc.callArenaAlloc()
```

**Dynamic List Creation:**
```go
// OLD: fc.trackFunctionCall("malloc")
//      fc.eb.GenerateCallInstruction("malloc")
// NEW:
fc.callArenaAlloc()
```

**Dynamic Map Creation:**
```go
// Maps with constant keys/values use .rodata
// Maps with runtime keys/values use callArenaAlloc()
```

## Cleanup

At program end, `cleanupAllArenas()` frees all arenas:

```assembly
# Load meta-arena pointer
lea rbx, [_flap_arena_meta]
mov rbx, [rbx]              # rbx = P1

# Loop through arenas
xor r8, r8                  # r8 = index = 0
cleanup_loop:
cmp r8, rcx                 # if (index >= len)
jge cleanup_done            #   exit loop

# Load arena pointer
mov rax, r8
shl rax, 3                  # offset = index * 8
add rax, rbx                # rax = &meta_arena[index]
mov rdi, [rax]              # rdi = arena_ptr
call free                   # Free arena struct (also frees buffer)

inc r8                      # index++
jmp cleanup_loop

cleanup_done:
# Free meta-arena array
mov rdi, rbx
call free
```

## Future: Arena Blocks

Planned syntax for scoped arenas:

```flap
# Global arena (default)
global_data := "stays alive"

arena {
    # Scoped arena - all allocations freed at block exit
    frame_data := "temporary"
    temp_list := [1, 2, 3]
    
    do_work(temp_list)
    
} # <-- All arena allocations freed here

# global_data still valid, frame_data freed
```

**Implementation:**
- Push new arena on arena stack
- Update `currentArena` index
- All allocations use new arena
- Pop arena and free at block exit

## Debugging

**Common Issues:**

1. **Segfault in callArenaAlloc:**
   - Check that meta-arena is initialized before first use
   - Verify TWO loads are performed to get arena struct pointer
   - Ensure `flap_arena_alloc` is being called, not generated inline

2. **Null pointer from allocation:**
   - Arena may be full and realloc failed
   - Check error handling in `flap_arena_alloc`

3. **Corrupted data:**
   - Check alignment is respected
   - Verify arena->used is updated correctly
   - Ensure no double-free scenarios

**Verification:**
```flap
# Test arena allocation
s1 := "hello"
s2 := " world"
s3 := s1 + s2          # Should allocate from arena
printf("%s\n", s3)     # Should print "hello world"

list := [1, 2, 3]      # Should allocate from arena
printf("%f\n", list[0]) # Should print "1.000000"
```

## Performance Characteristics

**Arena Allocation:**
- Time: O(1) - just pointer arithmetic
- Space: Minimal overhead (32 bytes per arena struct)
- Growth: O(n) when realloc needed (rare)

**vs. Malloc:**
- ~10x faster for typical game allocations
- No fragmentation
- Better cache locality
- Batch deallocation is instant

**Best Use Cases:**
- Frame-based game loops (allocate per frame, free at frame end)
- Level loading (allocate for level, free when done)
- String building (concatenate many strings, use once)
- Temporary data structures

## Summary

The arena allocator provides:
- ✅ Fast O(1) allocation
- ✅ Automatic cleanup at scope boundaries
- ✅ Zero fragmentation
- ✅ Predictable memory usage
- ✅ Integration with existing Flap runtime
- ✅ Compatibility with C malloc/free when needed

All Flap runtime operations (string concat, list creation, etc.) now use arena allocation by default, while user code can still use `c.malloc()` for manual memory management when needed.
