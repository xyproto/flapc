# Flap Language Specification

**Version:** 2.0.0  
**Date:** 2025-11-17  
**Status:** Canonical Language Reference

This document describes the complete semantics, behavior, and design philosophy of the Flap programming language. For the formal grammar, see [GRAMMAR.md](GRAMMAR.md).

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

## Table of Contents

- [What Makes Flap Unique](#what-makes-flap-unique)
- [Design Philosophy](#design-philosophy)
- [Type System](#type-system)
- [Variables and Assignment](#variables-and-assignment)
- [Control Flow](#control-flow)
- [Functions and Lambdas](#functions-and-lambdas)
- [Loops](#loops)
- [Parallel Programming](#parallel-programming)
- [ENet Channels](#enet-channels)
- [C FFI](#c-ffi)
- [CStruct](#cstruct)
- [Memory Management](#memory-management)
- [Unsafe Blocks](#unsafe-blocks)
- [Built-in Functions](#built-in-functions)
- [Error Handling](#error-handling)
- [Examples](#examples)

## What Makes Flap Unique

Flap brings together several novel or rare features that distinguish it from other systems programming languages:

### 1. Universal Map Type System

The entire language is built on a single type: `map[uint64]float64`. Every value—numbers, strings, lists, functions—IS this map. This radical simplification enables:
- No type system complexity
- Uniform memory representation
- Natural duck typing
- Simple FFI (cast to native types only at boundaries)

### 2. Direct Machine Code Generation

The compiler emits x86_64, ARM64, and RISCV64 machine code directly from the AST:
- **No intermediate representation** - AST → machine code in one pass
- **No dependencies** - completely self-contained compiler
- **Fast compilation** - no IR translation overhead
- **Small compiler** - ~30k lines of Go
- **Deterministic output** - same code every time

### 3. Blocks: Maps, Matches, and Statements

Blocks `{ ... }` are disambiguated by their contents:

```flap
// Map literal: contains key: value
config = { port: 8080, host: "localhost" }

// Statement block: no -> or ~> arrows
compute = x => {
    temp = x * 2
    result = temp + 10
    result  // last value returned
}

// Value match: expression before {, patterns with ->
classify = x => x {
    0 -> "zero"
    5 -> "five"
    ~> "other"
}

// Guard match: no expression before {, branches with | at line start
classify = x => {
    | x == 0 -> "zero"
    | x > 0 -> "positive"
    ~> "negative"
}
```

**Block disambiguation rules:**
1. Contains `:` (before arrows) → Map literal
2. Contains `->` or `~>` → Match block (value or guard)
3. Otherwise → Statement block

This unifies maps, pattern matching, guards, and function bodies into one syntax.

### 4. Unified Lambda Syntax

All functions use `=>`. Define with `=` (immutable) not `:=` unless reassignment needed:

```flap
// Use = for functions (standard)
square = x => x * 2
add = (x, y) => x + y
compute = x => { temp = x * 2; temp + 10 }
classify = x => x { 0 -> "zero" ~> "other" }
hello ==> println("Hello!")        // ==> shorthand for () =>

// Only use := if function will be reassigned
handler := x => println(x)
handler := x => println("DEBUG:", x)  // reassignment
```

**Convention:** Functions are immutable by default (`=`), only use `:=` when needed.

### 5. Minimal Parentheses

Avoid parentheses unless needed for precedence or grouping:

```flap
// Good: no unnecessary parens
x > 0 { -> "positive" ~> "negative" }
result = x + y * z
classify = x => x { 0 -> "zero" ~> "other" }

// Only use when needed
result = (x + y) * z              // precedence
cond = (x > 0 && y < 10) { ... }  // complex condition grouping
```

### 6. Bitwise Operators with `b` Suffix

All bitwise operations are suffixed with `b` to eliminate ambiguity:

```flap
<<b >>b <<<b >>>b    // Shifts and rotates
&b |b ^b ~b          // Bitwise logic
```

### 7. Explicit String Encoding

```flap
text = "Hello"
bytes = text.bytes   // Map of byte values {0: byte0, 1: byte1, ...}
runes = text.runes   // Map of Unicode code points {0: rune0, 1: rune1, ...}
```

### 8. ENet for All Concurrency

Network-style message passing for concurrency:

```flap
@8080 <- "Hello"     // Send to channel
msg = => @8080       // Receive from channel
```

### 9. Fork-Based Process Model

Parallel loops use `fork()` for true isolation:

```flap
|| i in 0..10 {      // Each iteration in separate process
    compute(i)
}
```

### 10. Pipe Operators for Data Flow

```flap
|    Pipe (transform)
||   Parallel map
|||  Reduce/fold
```

### 11. C FFI via DWARF

Parse C headers automatically using DWARF debug info:

```flap
result = c_function(arg1, arg2)  // Direct C calls
```

### 12. CStruct with Direct Memory Access

```flap
cstruct Point {
    x as float64,
    y as float64
}
p = Point(1.0, 2.0)
p.x  // Direct memory offset access
```

### 13. Tail-Call Optimization Always On

```flap
factorial = (n, acc) => n == 0 {
    -> acc
    ~> factorial(n - 1, acc * n)    // Optimized to loop
}
```

### 14. Cryptographically Secure Random

```flap
x = ??  // Uses OS CSPRNG
```

### 15. Move Operator `!` (Postfix)

```flap
new_owner = old_owner!  // Transfer ownership
```

### 16. Result Type with NaN Error Encoding

```flap
result = risky_operation()
result.error { != "" -> println("Error:", result.error) }
```

### 17. Immutable-by-Default

```flap
x = 42      // Immutable
y := 100    // Mutable (explicit)
```

## Design Philosophy

### Core Principles

1. **Simplicity over complexity**
   - One universal type (map)
   - Minimal syntax
   - Direct code generation

2. **Explicit over implicit**
   - Mutability must be declared (`:=`)
   - String encoding is explicit (`.bytes`, `.runes`)
   - Bitwise ops marked with `b` suffix

3. **Performance without compromise**
   - Direct machine code generation
   - Tail-call optimization
   - Zero-cost abstractions
   - No garbage collection overhead

4. **Safety where it matters**
   - Immutable by default
   - Explicit unsafe blocks
   - Arena allocators
   - Move semantics

5. **Minimal conventions**
   - Functions use `=` not `:=`
   - Avoid unnecessary parentheses
   - Match blocks require explicit condition or guards

## Type System

Flap uses a **universal map type**: `map[uint64]float64`

Every value in Flap IS `map[uint64]float64`:

- **Numbers**: `{0: number_value}`
- **Strings**: `{0: char0, 1: char1, 2: char2, ...}`
- **Lists**: `{0: elem0, 1: elem1, 2: elem2, ...}`
- **Objects**: `{key_hash: value, ...}`
- **Functions**: `{0: code_pointer, 1: closure_data, ...}`

There are no special cases. No "single entry maps", no "byte indices", no "field hashes" — just uint64 keys and float64 values in every case.

### Type Conversions

Use `as` for type casts at FFI boundaries:

```flap
x as int32      // Cast to C int32
ptr as cstr     // Cast to C string pointer
val as float64  // Cast to C double
```

**Supported C types:**
```
int8 int16 int32 int64
uint8 uint16 uint32 uint64
float32 float64
ptr cstr
```

### Duck Typing

Since everything is a map, Flap has structural typing:

```flap
point = { x: 10, y: 20 }
point.x  // Works - map has "x" key

person = { name: "Alice", x: 5 }
person.x  // Also works - different map, same key
```

## Variables and Assignment

### Immutable Assignment (`=`)

Creates immutable binding:

```flap
x = 42
x = 100  // ERROR: cannot reassign immutable variable
```

**Use for:**
- Constants
- Function definitions
- Values that won't change

### Mutable Assignment (`:=`)

Creates mutable binding:

```flap
x := 42
x := 100  // OK: mutable variable
x <- 200  // OK: update with <-
```

**Use for:**
- Loop counters
- Accumulators
- Values that will change

### Update Operator (`<-`)

Updates mutable variables or map elements:

```flap
x := 10
x <- 20      // Update variable

nums := [1, 2, 3]
nums[0] <- 99    // Update list element

config := { port: 8080 }
config.port <- 9000  // Update map field
```

### Function Assignment Convention

**Always use `=` for functions** unless the function variable needs reassignment:

```flap
// Standard (use =)
add = (x, y) => x + y
factorial = n => n { 0 -> 1 ~> n * factorial(n-1) }

// Only use := if reassigning
handler := x => println(x)
handler := x => println("DEBUG:", x)  // reassign
```

### Mutability Semantics

The assignment operator determines both **variable mutability** and **value mutability**:

| Operator | Variable Mutability | Value Mutability |
|----------|---------------------|------------------|
| `=` | Immutable (can't reassign) | Immutable (can't modify contents) |
| `:=` | Mutable (can reassign) | Mutable (can modify contents) |

**Examples:**

```flap
// Immutable binding, immutable value
nums = [1, 2, 3]
nums <- [4, 5, 6]     // ERROR: can't reassign
nums[0] <- 99         // ERROR: can't modify

// Mutable binding, mutable value
vals := [1, 2, 3]
vals <- [4, 5, 6]     // OK: reassign
vals[0] <- 99         // OK: modify
```

## Control Flow

### Match Expressions

Match blocks have two forms: **value match** and **guard match**.

#### Value Match (with expression before `{`)

Evaluates expression, then matches its result against patterns:

```flap
// Match on literal values
x = 5
result = x {
    0 -> "zero"
    5 -> "five"
    10 -> "ten"
    ~> "other"
}

// Match on boolean (1 = true, 0 = false)
result = (x > 0) {
    1 -> "positive"
    0 -> "not positive"
}

// Shorthand with default
result = (x > 10) {
    1 -> "large"
    ~> "small"
}
```

#### Guard Match (no expression, branches with `|` at line start)

Each branch evaluates its own condition:

```flap
// Guard branches with | at line start
classify = x => {
    | x == 0 -> "zero"
    | x > 0 -> "positive"
    | x < 0 -> "negative"
    ~> "unknown"  // optional default
}

// Multiple conditions
category = age => {
    | age < 13 -> "child"
    | age < 18 -> "teen"
    | age < 65 -> "adult"
    ~> "senior"
}
```

**Important:** The `|` is only a guard marker when at the start of a line/clause.
Otherwise `|` is the pipe operator:

```flap
// This is a guard (| at start)
x => { | x > 0 -> "positive" }

// This is a pipe operator (| not at start)
result = data | transform | filter
```

**Key difference:**
- **Value match:** One expression evaluated once, result matched against patterns
- **Guard match:** Each `|` branch (at line start) evaluates independently (short-circuits on first true)

**Default case:** `~>` works in both forms

### Tail Calls

The compiler automatically optimizes tail calls to loops:

```flap
// Explicit tail call with ->
factorial = (n, acc) => n == 0 {
    -> acc
    ~> factorial(n - 1, acc * n)
}

// Tail call in default case
sum_list = (list, acc) => list.length {
    0 -> acc
    ~> sum_list(list[1:], acc + list[0])
}
```

**Tail position rules:**
- Last expression in function body
- After `->` or `~>` in match arm
- In final expression of block

## Functions and Lambdas

### Function Definition

Functions are defined using `=` (immutable by default):

```flap
// Named function
square = x => x * x

// Multiple parameters
add = (x, y) => x + y

// With block body
factorial = n => {
    result := 1
    @ i in 1..n {
        result *= i
    }
    result
}

// No-arg shorthand
greet ==> println("Hello!")  // ==> is () =>
```

### Lambda Expressions

Lambdas use the same syntax:

```flap
// Inline lambda
[1, 2, 3] | x => x * 2

// Multi-line lambda
process = data => {
    cleaned = data | x => x.trim()
    cleaned | x => x.length > 0
}
```

### Closures

Lambdas capture their environment:

```flap
make_counter = start => {
    count := start
    => {
        count <- count + 1
        count
    }
}

counter = make_counter(0)
counter()  // 1
counter()  // 2
```

### Higher-Order Functions

Functions can take and return functions:

```flap
apply_twice = (f, x) => f(f(x))

increment = x => x + 1
result = apply_twice(increment, 10)  // 12
```

## Loops

### Infinite Loop

```flap
@ {
    println("Forever")
}
```

### Counted Loop

```flap
@ 10 {
    println("Hello")
}
```

### Range Loop

```flap
@ i in 0..10 {
    println(i)
}

// With step
@ i in 0..100..10 {  // 0, 10, 20, ...
    println(i)
}
```

### Collection Loop

```flap
nums = [1, 2, 3, 4, 5]
@ n in nums {
    println(n)
}
```

### Loop Control

```flap
@ i in 0..100 {
    i == 50 { break }     // Exit loop
    i % 2 == 0 { continue }  // Skip even numbers
    println(i)
}
```

## Parallel Programming

### Parallel Loops

Use `||` for parallel iteration (each iteration in separate process):

```flap
|| i in 0..10 {
    // Runs in separate forked process
    expensive_computation(i)
}
```

**Implementation:** Uses `fork()` for true OS-level parallelism.

### Parallel Map

```flap
// Sequential map
results = [1, 2, 3] | x => x * 2

// Parallel map  
results = [1, 2, 3] || x => expensive(x)
```

### Reduce/Fold

```flap
// Sum
total = [1, 2, 3, 4, 5] ||| (acc, x) => acc + x

// Max
max_val = [3, 7, 2, 9, 1] ||| (acc, x) => acc > x { -> acc ~> x }

// String concatenation
words = ["Hello", " ", "World"] ||| (acc, s) => acc + s
```

## ENet Channels

Flap uses **ENet-style message passing** for concurrency:

### Send Messages

```flap
@8080 <- "Hello"          // Send to port 8080
@"host:9000" <- data      // Send to remote host
```

### Receive Messages

```flap
msg = => @8080            // Receive from port 8080
data = => @"server:9000"  // Receive from remote
```

### Channel Patterns

```flap
// Worker pattern
worker ==> {
    @ {
        task = => @8080
        result = process(task)
        @8081 <- result
    }
}

// Pipeline pattern
stage1 ==> @ { @8080 <- generate_data() }
stage2 ==> @ { data = => @8080; @8081 <- transform(data) }
stage3 ==> @ { result = => @8081; save(result) }
```

**Note:** ENet channels are compiled directly into machine code that uses ENet library calls.

## C FFI

Flap can call C functions directly using DWARF debug information:

### Calling C Functions

```flap
// Automatically parsed from C headers via DWARF
result = c_malloc(1024)
c_free(result)

// With type casts
size = buffer_size as int32
ptr = c_malloc(size)
```

### Type Mapping

| Flap | C |
|------|---|
| `x as int32` | `int32_t` |
| `x as float64` | `double` |
| `ptr as cstr` | `char*` |
| `ptr as ptr` | `void*` |

### C Library Linking

The compiler links with `-lc` by default. Additional libraries:

```bash
flapc program.flap -o program -L/path/to/libs -lmylib
```

## CStruct

Define C-compatible structures with explicit memory layout:

### Declaration

```flap
cstruct Point {
    x as float64,
    y as float64
}

cstruct Rect {
    top_left as Point,
    width as float64,
    height as float64
}
```

### Usage

```flap
// Create struct
p = Point(3.0, 4.0)

// Access fields (direct memory offset, no overhead)
println(p.x)  // 3.0
p.x <- 10.0   // Update field

// Nested structs
r = Rect(Point(0.0, 0.0), 100.0, 50.0)
println(r.top_left.x)
```

### Memory Layout

CStructs have C-compatible memory layout:
- Fields stored sequentially in memory
- No hidden metadata
- Can be passed to C functions directly
- Access via direct pointer arithmetic

## Memory Management

### Stack vs Heap

- **Stack**: Function local variables, temporaries
- **Heap**: Dynamically allocated data (lists, maps, large objects)

### Arena Allocators

Scoped memory management without GC:

```flap
result = arena {
    data = allocate(1024)
    process(data)
    final_value
}
// All arena memory freed here
```

**Use cases:**
- Request handlers
- Temporary buffers
- Batch processing

### Move Semantics

Transfer ownership with postfix `!`:

```flap
large_data := [1, 2, 3, /* ... */, 1000000]
new_owner = large_data!  // Move, don't copy
// large_data now invalid
```

### Manual Memory

```flap
unsafe ptr {
    ptr = malloc(1024)
    // Use ptr
    free(ptr)
}
```

## Unsafe Blocks

Direct assembly and memory access:

### Syntax

```flap
unsafe return_type {
    // Assembly or low-level operations
} {
    // Optional: on success
} {
    // Optional: on error
}
```

### Examples

```flap
// Direct memory access
value = unsafe float64 {
    rax <- ptr
    rax <- [rax + offset]
}

// Syscall
unsafe {
    rax <- 1        // sys_write
    rdi <- 1        // stdout
    rsi <- msg_ptr
    rdx <- msg_len
    syscall
}

// With error handling
result = unsafe int32 {
    rax <- dangerous_operation()
} {
    println("Success")
    rax
} {
    println("Failed")
    -1
}
```

## Built-in Functions

### I/O

```flap
println(x)           // Print with newline
print(x)            // Print without newline
printa(x)           // Atomic print (thread-safe)
```

### String Operations

```flap
s = "Hello"
s.length            // 5 (number of entries in the map)
s.bytes             // Map of byte values {0: 72.0, 1: 101.0, ...}
s.runes             // Map of Unicode code points
s + " World"        // Concatenation (merges maps)
```

### List Operations

```flap
nums = [1, 2, 3]
nums.length         // 3
nums[0]             // 1
nums[1:]            // [2, 3]
nums + [4, 5]       // [1, 2, 3, 4, 5]
```

### Map Operations

```flap
m = { x: 10, y: 20 }
m.x                 // 10
m.z <- 30           // Add field
keys = m.keys()     // Get all keys
```

### Math Functions

All standard math via C FFI:

```flap
sin(x)
cos(x)
sqrt(x)
pow(x, y)
abs(x)
```

## Error Handling

### Result Type

Operations that can fail return a **Result**, which is `map[uint64]float64` that either:
1. Contains the actual value (success case)
2. Contains an error code string (error case)

```flap
result = risky_operation()

// Check for error
result.error { 
    != "" -> println("Error:", result.error) 
}

// Or use match
result.error {
    "" -> println("Success:", result)
    ~> println("Failed:", result.error)
}
```

### Result Encoding

A Result is detected as error/success at runtime:

1. **Pointer check:** If the value can be interpreted as a valid pointer (address > 0x1000), it's **SUCCESS** (contains actual value)
2. **Error code:** If not a valid pointer, interpret as 4-character error code string

Error codes (4 chars, space-padded):
```
"dv0 " - Division by zero
"idx " - Index out of bounds
"key " - Key not found
"typ " - Type mismatch
"nil " - Null pointer
"mem " - Out of memory
"arg " - Invalid argument
"io  " - I/O error
"net " - Network error
"prs " - Parse error
```

### .error Property

Every value has `.error` accessor:

```flap
x = 10 / 0              // Error result
x.error                 // Returns "dv0" (trailing space stripped)

y = 10 / 2              // Success result  
y.error                 // Returns "" (empty string)

// Common pattern
result.error {
    "" -> proceed(result)
    ~> handle_error(result.error)
}
```

### or! Operator

The `or!` operator provides default values for errors:

```flap
x = 10 / 0              // Error result
safe = x or! 99         // Returns 99 (error case)

y = 10 / 2              // Success result (value 5)
safe2 = y or! 99        // Returns 5 (success case)
```

How it works:
1. Evaluate left side
2. If error: return right side
3. If success: return left side value

## Examples

### Hello World

```flap
println("Hello, World!")
```

### Factorial

```flap
// Iterative
factorial = n => {
    result := 1
    @ i in 1..n {
        result *= i
    }
    result
}

// Tail-recursive
factorial = (n, acc) => n == 0 {
    -> acc
    ~> factorial(n-1, n*acc)
}
```

### FizzBuzz

```flap
@ i in 1..100 {
    result = i % 15 {
        0 -> "FizzBuzz"
        ~> i % 3 {
            0 -> "Fizz"
            ~> i % 5 {
                0 -> "Buzz"
                ~> i
            }
        }
    }
    println(result)
}
```

### Parallel Processing

```flap
data = [1, 2, 3, 4, 5, 6, 7, 8]

// Process in parallel
results = data || x => expensive_computation(x)

// Sum results
total = results ||| (acc, x) => acc + x

println(total)
```

### Web Server (ENet)

```flap
server ==> {
    @ {
        request = => @8080
        response = handle_request(request)
        @8080 <- response
    }
}

server()
```

### C Interop

```flap
cstruct Buffer {
    data as ptr,
    size as int32
}

buf = Buffer(c_malloc(1024), 1024)
c_memset(buf.data, 0, buf.size)
// Use buffer
c_free(buf.data)
```

---

**For grammar details, see [GRAMMAR.md](GRAMMAR.md)**

**For development info, see [DEVELOPMENT.md](DEVELOPMENT.md)**

**For release notes, see [RELEASE_NOTES_2.0.md](RELEASE_NOTES_2.0.md)**
