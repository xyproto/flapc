# Things NOT to Do in Flap and Flapc

This document lists tempting but **incorrect** approaches when developing Flap programs or working on the Flapc compiler. These patterns may seem natural coming from other languages, but violate Flap's design philosophy.

---

## Memory Management - DON'T Use Malloc (Except...)

### ❌ DON'T: Use malloc for list/map allocation

**Tempting but wrong:**
```go
// In codegen.go - WRONG!
func (c *Codegen) emitListLiteral(list *ListLiteral) {
    c.EmitCallC("malloc", size)  // NO!
}
```

**Why it's wrong:**
- Violates Flap's arena-based memory philosophy
- Creates fragmentation
- No automatic cleanup
- Defeats the purpose of arena allocators
- Users lose control over memory lifetime

**Correct approach:**
- Lists should use arena allocation
- The main function has an implicit arena
- Explicit arena blocks manage their own memory
- See MEMORY.md for details

### ❌ DON'T: Use malloc for cons operator (::)

**Tempting but wrong:**
```go
// In flap_runtime.go or codegen.go - WRONG!
func _flap_list_cons(item, list) {
    new_list := malloc(...)  // NO!
    // ...
}
```

**Why it's wrong:**
- Cons is a pure functional operation
- Should not have side effects like malloc
- Memory should come from arena
- Creates memory leaks in implicit contexts

**Correct approach:**
- Cons uses arena allocation
- If in arena block, use that arena
- Otherwise, use default global arena (arena0)
- Memory cleanup happens when arena exits

### ✅ DO: Use malloc ONLY for these three cases

1. **User-explicit C FFI**: `c.malloc()` in user code
2. **Arena metadata**: Allocating arena structures in the meta-arena table
3. **Arena growth**: Expanding arena memory with `realloc()`

**Example of correct malloc usage:**
```go
// CORRECT: Allocating arena metadata
arena_meta := malloc(sizeof(ArenaMetadata))

// CORRECT: Growing arena memory
arena_memory := realloc(old_ptr, new_size)

// CORRECT: User explicitly called c.malloc()
ptr := c.malloc(1024)  // User's choice, they manage it
```

---

## Type System - DON'T Add Complex Types

### ❌ DON'T: Add special case types

**Tempting but wrong:**
```go
// WRONG: Adding special "string" or "list" types to type system
type Type int
const (
    TypeFloat64
    TypeString   // NO!
    TypeList     // NO!
    TypeInt32
)
```

**Why it's wrong:**
- Everything is `map[uint64]float64` internally
- Special cases complicate the compiler
- Breaks the unified type system philosophy

**Correct approach:**
- Only one internal type: `map[uint64]float64`
- Type casts (`as int32`) are for C FFI only
- Strings/lists are just maps with specific key patterns

### ❌ DON'T: Add implicit type conversions

**Tempting but wrong:**
```flap
x := 42
y := x + "hello"  // NO! Implicit int->string conversion
```

**Why it's wrong:**
- Flap is explicit over implicit
- User must choose conversions
- Reduces surprises and bugs

**Correct approach:**
```flap
x := 42
y := f"{x}hello"  // Explicit: f-string for concatenation
```

---

## Control Flow - DON'T Add New Keywords

### ❌ DON'T: Add `break` or `continue` keywords

**Tempting but wrong:**
```flap
// NO! Don't add these keywords
@ i in 0..<10 {
    i == 5 {
        break     // WRONG - keyword doesn't exist
    }
}
```

**Why it's wrong:**
- Flap uses `ret @` for loop exit
- Flap uses `@N` for continue to next iteration
- Adding keywords violates minimalism

**Correct approach:**
```flap
@ i in 0..<10 {
    i == 5 {
        ret @     // Exit current loop
    }
}

@ i in 0..<10 {
    i == 5 {
        @         // Continue to next iteration
    }
}
```

### ❌ DON'T: Add `if/else` keywords

**Tempting but wrong:**
```flap
// NO! Don't add if/else keywords
if x > 0 {      // WRONG - 'if' doesn't exist
    println("positive")
} else {
    println("negative")
}
```

**Why it's wrong:**
- Flap uses match blocks with `->` and `~>`
- Conditional expressions are just expressions with blocks
- No need for special keywords

**Correct approach:**
```flap
// Guard match (note the | prefix for conditions)
{
    | x > 0 -> println("positive")
    ~> println("negative")
}

// Or simple conditional
x > 0 {
    println("positive")
    ~> println("negative")
}

// Or with the optional arrow
x > 0 {
    -> println("positive")
    ~> println("negative")
}
```

---

## Match Blocks - DON'T Confuse Syntax

### ❌ DON'T: Use `->` for lambda bodies

**Tempting but wrong:**
```flap
// WRONG: Using -> for lambda
square := x -> x * x   // NO! Wrong arrow
```

**Why it's wrong:**
- `=>` is the lambda arrow (always)
- `->` is for match arms only
- Consistency is key

**Correct approach:**
```flap
square := x => x * x          // Lambda uses =>
result := x {                 // Match uses ->
    0 -> "zero"
    ~> "other"
}
```

### ❌ DON'T: Mix value matching and guards without prefix

**Tempting but wrong:**
```flap
// WRONG: Guard without | prefix
result := x {
    x > 0 -> "positive"    // NO! Needs | prefix for guards
    0 -> "zero"            // This is value matching
}
```

**Why it's wrong:**
- Guards need `|` prefix to distinguish from value matches
- `x > 0` looks like value comparison, not guard evaluation
- Ambiguous without prefix

**Correct approach:**
```flap
result := x {
    | x > 0 -> "positive"  // Guard: | prefix required
    0 -> "zero"            // Value match: no prefix
    ~> "negative"
}
```

---

## Functions - DON'T Use := for Function Definitions

### ❌ DON'T: Declare functions with :=

**Tempting but wrong:**
```flap
// WRONG: Using := for function definition
add := (a, b) => a + b   // NO! Functions should use =
```

**Why it's wrong:**
- Functions are immutable values
- `:=` implies mutability, but functions shouldn't be reassigned
- Inconsistent with Flap philosophy

**Correct approach:**
```flap
// CORRECT: Use = for function definitions
add = (a, b) => a + b

// Only use := if you REALLY need to reassign
counter := x => x + 1    // Mutable
counter <- x => x + 2    // Can reassign (rare case)
```

---

## Operators - DON'T Confuse Bitwise

### ❌ DON'T: Use & | ^ ~ for bitwise operations

**Tempting but wrong:**
```flap
// WRONG: Using symbols without 'b' suffix
x := 5 & 3     // NO! This isn't bitwise AND
x := 5 | 3     // NO! This isn't bitwise OR
```

**Why it's wrong:**
- Flap uses `&b`, `|b`, `^b`, `~b` for bitwise (with 'b' suffix)
- Plain `&`, `|` might be used for other purposes later
- Explicit suffix prevents confusion

**Correct approach:**
```flap
x := 5 &b 3    // Bitwise AND
x := 5 |b 3    // Bitwise OR
x := 5 ^b 3    // Bitwise XOR
x := ~b 5      // Bitwise NOT
```

---

## Loops - DON'T Forget max for Unknown Iteration Count

### ❌ DON'T: Omit max when counter is modified

**Tempting but wrong:**
```flap
// WRONG: Counter modified but no max specified
@ i in 0..<10 {
    i += 1     // NO! Modifying counter requires max
}
```

**Why it's wrong:**
- Compiler can't determine loop termination
- Could be infinite loop
- Safety mechanism requires explicit max

**Correct approach:**
```flap
@ i in 0..<10 max 20 {
    i += 1     // OK with max specified
}

@ msg in channel() max inf {
    process(msg)  // OK: infinite loop explicitly marked
}
```

---

## Error Handling - DON'T Add try/catch

### ❌ DON'T: Add exception handling keywords

**Tempting but wrong:**
```go
// NO! Don't add try/catch to Flap
try {
    x := dangerous_operation()
} catch (error) {
    handle_error(error)
}
```

**Why it's wrong:**
- Flap uses Result type with NaN encoding
- Errors are values, not exceptions
- Explicit error handling with `or!` operator
- No hidden control flow

**Correct approach:**
```flap
// Use Result type and or! operator
x := dangerous_operation()
safe_x := x or! default_value

// Or extract error code
x.error == "dv0 " {
    println("Division by zero!")
}
```

---

## Built-ins - DON'T Add Unnecessary Functions

### ❌ DON'T: Add `is_error()` built-in

**Tempting but wrong:**
```flap
// WRONG: is_error doesn't exist
result := 10 / 0
is_error(result) {   // NO! This function doesn't exist
    println("Error!")
}
```

**Why it's wrong:**
- Use `or!` operator instead
- Use `.error` property to extract error code
- Built-in would be redundant

**Correct approach:**
```flap
result := 10 / 0
safe_result := result or! 0.0   // Provide default for errors

// Or check error code directly
error_code := result.error
error_code != "" {
    println(f"Error: {error_code}")
}
```

### ❌ DON'T: Add `range()` function

**Tempting but wrong:**
```flap
// WRONG: range() function doesn't exist
@ i in range(0, 10) {   // NO!
    println(i)
}
```

**Why it's wrong:**
- Use `..` or `..<` operators directly
- No need for function call overhead
- Operators are more idiomatic

**Correct approach:**
```flap
@ i in 0..<10 {     // Exclusive range
    println(i)
}

@ i in 0..10 {      // Inclusive range
    println(i)
}
```

---

## Naming - DON'T Use Abbreviations

### ❌ DON'T: Use short type names

**Tempting but wrong:**
```flap
// WRONG: Abbreviated type names
x := value as i32    // NO! Use int32
y := value as u64    // NO! Use uint64
z := value as f32    // NO! Use float32
```

**Why it's wrong:**
- Flap uses full names for clarity
- `i32`, `u64`, `f32` are not valid
- Consistency with C type names

**Correct approach:**
```flap
x := value as int32
y := value as uint64
z := value as float32
```

---

## Concurrency - DON'T Mix Paradigms

### ❌ DON'T: Add goroutines or channels

**Tempting but wrong:**
```go
// NO! Don't add Go-style concurrency to Flap
go process_data()           // NO goroutines
ch := make(chan int)        // NO channels
```

**Why it's wrong:**
- Flap uses ENet for all messaging
- Flap uses `spawn` for processes (fork-based)
- Unified IPC/network model with addresses

**Correct approach:**
```flap
// Use spawn for background work
spawn process_data()

// Use ENet addresses for communication
@5000 <- "message"
@ msg, from in @5000 {
    process(msg)
}
```

---

## Parser - DON'T Add Ambiguous Syntax

### ✅ DO: Use `<-` for both update and send/receive

**The `<-` operator is unified:**
```flap
x <- 42            // Update mutable variable
@5000 <- msg       // Send to address
@5000 <- response  // Receive into variable (blocking)
```

**Why it works:**
- Context determines meaning: variable on left = update, address on left/right = send/receive
- Minimal syntax with no ambiguity
- Consistent with channel semantics from other languages
- Single arrow for all data flow operations

**Direction matters:**
```flap
x <- 42            // Data flows INTO x (update)
@5000 <- "hello"   // Data flows INTO @5000 (send)
@5000 <- msg       // Data flows FROM @5000 INTO msg (receive)
```

---

## Compilation - DON'T Add Intermediate Representations

### ❌ DON'T: Generate LLVM IR or other IR

**Tempting but wrong:**
```go
// NO! Don't generate IR
func (c *Codegen) EmitLLVMIR() string {
    // WRONG approach
}
```

**Why it's wrong:**
- Flap compiles directly to machine code
- No IR layer (AST → machine code)
- Adds dependency and complexity
- Slower compilation
- Less control over output

**Correct approach:**
- Emit x86-64, ARM64, or RISC-V machine code directly
- Use existing architecture-specific codegen files
- Keep compilation single-pass

---

## Optimization - DON'T Optimize Prematurely

### ❌ DON'T: Add complex optimizations early

**Tempting but wrong:**
```go
// NO! Don't add these before compiler is complete
- Constant folding (maybe later)
- Dead code elimination (maybe later)
- Inline expansion (maybe later)
- Loop unrolling (probably never)
```

**Why it's wrong:**
- Compiler correctness comes first
- Optimization adds complexity and bugs
- Profile before optimizing
- Simple is better

**Correct approach:**
- Get it working first
- Profile to find bottlenecks
- Optimize hot paths only
- Focus on correctness over speed initially

---

## Testing - DON'T Write Tests to /tmp or /dev/shm

### ❌ DON'T: Create temporary .flap files

**Tempting but wrong:**
```go
// WRONG: Writing test files to filesystem
func TestSomething(t *testing.T) {
    ioutil.WriteFile("/tmp/test.flap", code, 0644)  // NO!
    // ...
}
```

**Why it's wrong:**
- Pollutes filesystem
- Race conditions between tests
- Cleanup issues
- Slower than in-memory

**Correct approach:**
```go
// CORRECT: Use in-memory compilation
func TestSomething(t *testing.T) {
    code := `x := 42\nprintln(x)`
    program := ParseString(code)
    binary := Compile(program)
    result := Execute(binary)
    // ...
}
```

The exception is if a problem is hard to debug, and it's easier to write an executable to /tmp or /dev/shm, view the code with objdump or ndisasm, or debug it with gdb, and then remove the executable afterwards.

---

## Summary: The Flap Way

**DO:**
- Use arena allocators for memory
- Keep everything `map[uint64]float64` internally
- Emit machine code directly
- Use explicit syntax (no magic)
- Follow LANGUAGE.md as source of truth
- Be minimal and orthogonal

**DON'T:**
- Use malloc (except 3 cases)
- Add type system complexity
- Add unnecessary keywords
- Optimize prematurely
- Mix concurrency paradigms
- Create intermediate representations
- Write tests to filesystem

**When in doubt:**
1. Check LANGUAGE.md (source of truth)
2. Check MEMORY.md (for memory questions)
3. Check LEARNINGS.md (for gotchas)
4. Ask: "Is this adding complexity or removing it?"
5. Choose the simpler, more explicit option

---

**Remember:** Flap's power comes from simplicity, not feature count. When tempted to add something conventional, ask: "Does this align with Flap's philosophy?" If not, don't do it.
