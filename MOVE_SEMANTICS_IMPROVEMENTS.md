# Move Semantics Improvements and Extensions

## Current State: Move Operator (`!`)

The `!` postfix operator currently provides explicit ownership transfer:

```flap
large_data := create_buffer()
consume(large_data!)  // Ownership transferred
// large_data is now invalidated
```

**Current Implementation:**
- `!` postfix operator marks value for move
- Compiler invalidates source variable
- Zero-copy transfer for large structures

---

## Proposed Improvements

### 1. Move-by-Default for Temporary Values

**Motivation:** Reduce explicit `!` clutter for obvious cases.

#### Current (verbose):
```flap
result := transform(create_data()!)  // Unnecessary !
```

#### Proposed (implicit):
```flap
result := transform(create_data())   // Auto-moved (temporary)
```

**Rule:** Values that are:
- Function return values (temporaries)
- Not assigned to a variable
- Used immediately in another function call

→ Should be moved automatically (no copy needed)

**Syntax Addition:** None (automatic detection)

---

### 2. Move-Only Types with `movable` Keyword

**Motivation:** Some types should NEVER be copied (e.g., file handles, network sockets, unique ownership).

```flap
movable FileHandle {
    fd as int32
}

open_file := (path as cstr) -> FileHandle {
    fd := call("open", path as ptr, 0 as int32)
    fd < 0 {
        exit(1)  // Error handling
    }
    -> FileHandle { fd: fd }
}

close_file := (handle as FileHandle!) -> {
    call("close", handle.fd as int32)
}

// Usage:
file := open_file("data.txt")
close_file(file!)  // MUST use ! - compile error otherwise
// file cannot be used again (moved)
```

**Benefits:**
- Prevents accidental double-close (safety)
- Compiler enforces ownership semantics
- Clear intent in type system

**Implementation:**
- Add `movable` keyword to cstruct/type definitions
- Compiler error if attempting to copy
- Must use `!` for all transfers

---

### 3. Borrowing with `&` Reference Syntax

**Motivation:** Allow temporary read-only access without transfer.

```flap
print_data := (data as &Buffer) -> {
    // Read-only access to data
    // Cannot modify or move
    println(f"Size: {data.size}")
}

main := () -> {
    buffer := create_buffer()
    print_data(&buffer)  // Borrow (no move)
    print_data(&buffer)  // Can use again!
    consume(buffer!)     // Final move
}
```

**Rules:**
- `&` creates temporary read-only reference
- Cannot modify borrowed value
- Cannot move borrowed value
- Original owner retains ownership
- Reference invalid after owner moves

**Type Signatures:**
```flap
read_only := (data as &Buffer) -> int32    // Borrow
take_ownership := (data as Buffer!) -> int32  // Move
normal := (data as Buffer) -> int32         // Copy (if copyable)
```

---

### 4. Mutable Borrowing with `&mut`

**Motivation:** Allow temporary mutable access without transfer.

```flap
resize_buffer := (buf as &mut Buffer, new_size as int32) -> {
    buf.size = new_size
    buf.data = call("realloc", buf.data as ptr, new_size as uint64)
}

main := () -> {
    buffer := create_buffer()
    resize_buffer(&mut buffer, 1024)  // Mutable borrow
    // buffer is still valid here
    consume(buffer!)  // Final move
}
```

**Rules:**
- Only one `&mut` reference at a time (exclusive access)
- Cannot create `&` while `&mut` exists
- Original owner cannot access while borrowed mutably
- Prevents data races

---

### 5. Lifetime Annotations for Complex Cases

**Motivation:** Make borrow checker work across function boundaries.

```flap
// Simple case (no annotation needed):
get_ptr := (data as &Buffer) -> ptr {
    -> data.ptr  // Lifetime tied to data
}

// Complex case (explicit lifetime):
longest<'a> := (s1 as &'a string, s2 as &'a string) -> &'a string {
    #s1 > #s2 {
        -> s1
    }
    { -> s2 }
}
```

**Syntax:**
- `<'a>` declares lifetime parameter
- `&'a T` ties reference to lifetime
- Compiler ensures returned reference valid

---

### 6. Move Semantics for Collections

**Motivation:** Efficient collection operations.

```flap
// Move elements from one list to another:
source := [1, 2, 3, 4, 5]
dest := []

@ item in source {
    dest.push(item!)  // Move each element
}
// source is now empty (all elements moved)

// Or move entire list:
dest2 := source!  // Entire list moved
```

**Implementation:**
- Collections track moved elements
- Empty slots marked as "moved-out"
- Accessing moved element → compile error

---

### 7. Conditional Moves

**Motivation:** Move only if condition met.

```flap
maybe_consume := (data as Buffer, should_consume as int32) -> {
    should_consume {
        consume(data!)
    }
    // If not consumed, caller retains ownership
}

// Problem: What if we don't know at compile time?
// Solution: Return optional moved flag

consume_if := (data as Buffer!, condition as int32) -> int32 {
    condition {
        consume(data)
        -> 1  // Consumed
    }
    -> 0  // Not consumed
}
```

**Challenge:** Partial moves in control flow require careful tracking.

---

### 8. Move Constructors / Destructors

**Motivation:** Custom move behavior for types.

```flap
cstruct UniquePtr {
    data as ptr

    // Move constructor (called on x!)
    move := (self!) -> UniquePtr {
        result := UniquePtr { data: self.data }
        self.data = 0 as ptr  // Invalidate source
        -> result
    }

    // Destructor (called on scope exit)
    drop := (self!) -> {
        self.data != 0 as ptr {
            call("free", self.data)
        }
    }
}
```

**Automatic Behavior:**
- `move` called when `!` used
- `drop` called when variable goes out of scope
- Similar to Rust's Drop trait

---

### 9. Pattern Matching with Moves

**Motivation:** Destructure and move simultaneously.

```flap
// Tuple destructuring with move:
(x!, y!) := get_pair()  // Both moved

// Struct destructuring with move:
buffer := Buffer { data: ptr, size: 100 }
{ data: d!, size: s } := buffer  // Move data, copy size
// buffer.data is now invalid

// Match with move:
result := some_operation()
result {
    Ok(value!) -> use(value)      // Move on success
    Err(e) -> handle_error(e)     // Copy error
}
```

---

### 10. Move Chains

**Motivation:** Fluent APIs with move semantics.

```flap
builder := BufferBuilder()
    .with_capacity(1024)!
    .with_alignment(16)!
    .build()!

// Each ! transfers ownership through chain
```

**Implementation:**
- Methods return `self!` for chaining
- Final `.build()!` consumes builder

---

## Syntax Summary

| Syntax | Meaning | Example |
|--------|---------|---------|
| `x!` | Move value | `consume(x!)` |
| `&x` | Borrow (read-only) | `read(&x)` |
| `&mut x` | Borrow (mutable) | `modify(&mut x)` |
| `movable T` | Move-only type | `movable FileHandle {}` |
| `T!` | Move-only parameter | `close(f as FileHandle!)` |
| `&'a T` | Lifetime-bound reference | `longest<'a>(s1 as &'a string, s2 as &'a string)` |

---

## Implementation Priority

### Phase 1 (Essential):
1. ✅ **Basic move operator (`!`)** - DONE
2. ⏳ **Move-by-default for temporaries** - High value, low complexity
3. ⏳ **Borrowing (`&` syntax)** - Critical for usability

### Phase 2 (High Value):
4. ⏳ **Mutable borrowing (`&mut`)** - Prevents unnecessary copies
5. ⏳ **Move-only types (`movable`)** - Safety for resources
6. ⏳ **Destructors (`drop`)** - RAII pattern support

### Phase 3 (Advanced):
7. ⏳ **Lifetime annotations** - Complex borrow scenarios
8. ⏳ **Collection move semantics** - Efficient data structure operations
9. ⏳ **Pattern matching moves** - Ergonomic destructuring

---

## Benefits of Full Move Semantics

### 1. Performance:
- **Zero-copy** for large data structures
- **No allocations** for temporary objects
- **Cache-friendly** (data stays in place)

### 2. Safety:
- **Use-after-move prevention** (compile-time error)
- **Double-free prevention** (moved resources can't be freed twice)
- **Data race prevention** (exclusive mutable access via `&mut`)

### 3. Clarity:
- **Explicit ownership** in function signatures
- **Clear intent** (borrow vs. move vs. copy)
- **Self-documenting** code

---

## Example: Before vs. After

### Before (no advanced move semantics):
```flap
process_data := (data as Buffer) -> {
    // Is data copied or moved? Unclear!
    transform(data)
    // Can I still use data? Unclear!
}
```

### After (with borrowing and moves):
```flap
// Read-only access:
inspect_data := (data as &Buffer) -> {
    println(f"Size: {data.size}")
    // Clearly: no ownership transfer
}

// Mutable access:
modify_data := (data as &mut Buffer) -> {
    data.size = 1024
    // Clearly: temporary mutable borrow
}

// Ownership transfer:
consume_data := (data as Buffer!) -> {
    free_buffer(data)
    // Clearly: data is consumed (moved)
}
```

---

## Compatibility

All improvements are **backward-compatible**:
- Existing `!` operator still works
- New syntax is opt-in
- Old code continues to compile

---

## Next Steps

1. ✅ Document current move semantics
2. ⏳ Implement move-by-default for temporaries
3. ⏳ Add borrow checking infrastructure
4. ⏳ Implement `&` and `&mut` syntax
5. ⏳ Add `movable` keyword support
6. ⏳ Implement destructor (`drop`) support
7. ⏳ Full lifetime tracking (if needed)

---

## Conclusion

These improvements would make Flapc's move semantics:
- **As safe as Rust's** (borrow checker prevents use-after-free)
- **As fast as C++** (zero-copy, RAII)
- **More ergonomic than both** (simpler syntax, auto-move temporaries)

The combination of explicit moves (`!`), borrowing (`&`, `&mut`), and move-only types (`movable`) provides a powerful, safe, and efficient ownership system suitable for systems programming, game development, and high-performance applications.
