# Flap Language Specification

**Version:** 3.0.0  
**Date:** 2025-11-17  
**Status:** Canonical Language Reference for Flap 3.0 Release

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
- [Classes and Object-Oriented Programming](#classes-and-object-oriented-programming)
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
&8080 <- "Hello"     // Send to channel
msg = => &8080       // Receive from channel
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

Creates immutable binding (cannot reassign variable or modify contents):

```flap
x = 42
x = 100  // ERROR: cannot reassign immutable variable

nums = [1, 2, 3]
nums[0] = 99  // ERROR: cannot modify immutable value
```

**Use for:**
- Constants
- Function definitions (standard practice)
- Values that won't change

### Mutable Assignment (`:=`)

Creates mutable binding (can reassign variable and modify contents):

```flap
x := 42
x := 100  // OK: reassign mutable variable
x <- 200  // OK: update with <-

nums := [1, 2, 3]
nums[0] <- 99  // OK: modify mutable value
```

**Use for:**
- Loop counters
- Accumulators
- Values that will change
- Functions that need reassignment (rare)

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

// No-arg shorthand: ==> desugars to () =>
greet ==> println("Hello!")
// Equivalent to: greet = () => println("Hello!")

hello ==> {
    println("Hello")
    println("World")
}
// Equivalent to: hello = () => { println("Hello"); println("World") }
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

Flap uses `ret @` with loop labels instead of `break`/`continue`:

```flap
// Exit current loop
@ i in 0..<100 {
    i > 50 { ret @ }      // Exit current loop
    i == 42 { ret @ 42 }  // Exit loop with value 42
    println(i)
}

// Nested loops with explicit labels
@ i in 0..<10 {           // Loop @1 (outer)
    @ j in 0..<10 {       // Loop @2 (inner)
        j == 5 { ret @ }         // Exit inner loop (@2)
        i == 5 { ret @1 }        // Exit outer loop (@1)
        i == 3 and j == 7 { ret @1 42 }  // Exit outer loop with value
        println(i, j)
    }
}

// ret without @ returns from function
compute = n => {
    @ i in 0..<100 {
        i == n { ret i }  // Return from function with value
        i == 50 { ret @ } // Exit loop only, continue function
    }
    ret 0  // Return from function
}
```

**Loop Labels:**
- `ret @` or `ret @1` - Exit innermost loop
- `ret @2` - Exit second loop level
- `ret @N value` - Exit loop N with value
- `ret value` - Return from function (not loop)

### Loop `max` Keyword

Loops with unknown bounds or modified counters require `max`:

```flap
// Counter modified in loop
@ i in 0..<10 max 20 {
    i++  // Modified counter - needs max
}

// Unknown iteration count
@ msg in read_channel() max inf {
    process(msg)
}

// Condition-based loop
@ x < threshold max 1000 {
    x = compute_next(x)
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
&8080 <- "Hello"          // Send to port 8080
&"host:9000" <- data      // Send to remote host
```

### Receive Messages

```flap
msg = => &8080            // Receive from port 8080
data = => &"server:9000"  // Receive from remote
```

### Channel Patterns

```flap
// Worker pattern
worker ==> {
    @ {
        task = => &8080
        result = process(task)
        &8081 <- result
    }
}

// Pipeline pattern
stage1 ==> @ { &8080 <- generate_data() }
stage2 ==> @ { data = => &8080; &8081 <- transform(data) }
stage3 ==> @ { result = => &8081; save(result) }
```

**Note:** ENet channels are compiled directly into machine code that uses ENet library calls.

## Classes and Object-Oriented Programming

Flap supports classes as syntactic sugar over maps and closures, following the philosophy that everything is `map[uint64]float64`.

### Design Philosophy

- **Syntactic sugar:** Classes compile to regular maps and lambdas
- **No new types:** Objects are still `map[uint64]float64`
- **Composition:** Use `<>` to extend with behavior maps (no inheritance)
- **Minimal syntax:** Only adds the `class` keyword
- **Transparent:** You can always see what the class desugars to

### Class Declaration

Classes group data and methods together:

```flap
class Point {
    init := (x, y) ==> {
        .x = x
        .y = y
    }
    
    distance := other => {
        dx := other.x - .x
        dy := other.y - .y
        sqrt(dx * dx + dy * dy)
    }
    
    move := (dx, dy) ==> {
        .x <- .x + dx
        .y <- .y + dy
    }
}

// Create instance
p1 := Point(10, 20)
p2 := Point(30, 40)

// Call methods
dist := p1.distance(p2)
p1.move(5, 5)
```

### How Classes Work

A class declaration creates a constructor function:

```flap
// This class:
class Counter {
    init := start ==> {
        .count = start
    }
    
    increment := ==> {
        .count <- .count + 1
    }
}

// Desugars to this:
Counter := start => {
    instance := {}
    instance["count"] = start
    instance["increment"] = () => {
        instance["count"] <- instance["count"] + 1
    }
    ret instance
}
```

### Instance Fields and "this"

Inside methods, `.field` accesses instance fields. The `. ` expression (dot followed by space or newline) means "this":

```flap
class List {
    init = () => {
        .items = []
    }
    
    add = item => {
        .items <- .items :: item
        ret .   // Return this (self) for chaining
    }
    
    size = () => .items.length
}

list = List().add(1).add(2).add(3)  // Method chaining via `. `
println(list.size())  // 3
```

**Key points:**
- `.field` accesses instance field inside methods
- `. ` (dot space or dot newline) means "this" (the current instance)
- Return `. ` for method chaining
- Outside methods, use `instance.field` explicitly
- No `this` keyword - use `. ` instead

```flap
class Account {
    init = balance => {
        .balance = balance
    }
    
    withdraw = amount => {
        amount > .balance {
            ret -1  // Insufficient funds
        }
        .balance <- .balance - amount
        ret 0
    }
    
    deposit = amount => {
        .balance <- .balance + amount
    }
    
    get_balance = () => .balance
}

acc = Account(100)
acc.deposit(50)
println(acc.get_balance())  // 150
```

### Class Fields (Static)

Class fields are shared across all instances:

```flap
class Entity {
    Entity.count = 0
    Entity.all = []
    
    init := name ==> {
        .name = name
        .id = Entity.count
        Entity.count <- Entity.count + 1
        Entity.all <- Entity.all :: instance
    }
    
    get_total := ==> Entity.count
}

e1 := Entity("Alice")
e2 := Entity("Bob")
println(e1.get_total())  // 2
println(Entity.count)    // 2
```

### Composition with `<>`

Extend classes with behavior maps using the `<>` composition operator:

```flap
// Define behavior map
Serializable := {
    to_json: ==> {
        // Serialize instance to JSON string
        keys := this.keys()
        @ i in 0..<keys.length {
            // Build JSON...
        }
    },
    from_json: json => {
        // Parse JSON and populate instance
    }
}

// Extend class with behavior using <>
class User <> Serializable {
    init := (name, email) ==> {
        .name = name
        .email = email
    }
}

user := User("Alice", "alice@example.com")
json := user.to_json()
```

**Multiple composition** - chain `<>` operators:

```flap
class Product <> Serializable <> Validatable <> Timestamped {
    init := (name, price) ==> {
        .name = name
        .price = price
        .created_at = now()
    }
}
```

**How `<>` works:** The `<>` operator merges behavior maps into the class. At runtime, all methods from the behavior maps are copied into the instance during construction, with later maps overriding earlier ones if there are conflicts.

### Method Semantics

**Instance methods** close over the instance:

```flap
class Box {
    init := value ==> {
        .value = value
    }
    
    get := ==> .value
    set := v ==> { .value <- v }
}

b := Box(42)
getter := b.get  // Captures b
println(getter())  // 42
```

**Class methods** don't capture instances:

```flap
class Math {
    Math.PI = 3.14159
    
    // Note: no init, Math is never instantiated
    Math.circle_area = radius => Math.PI * radius * radius
}

area := Math.circle_area(10)
```

### Private Methods (Convention)

Use underscore prefix for "private" methods:

```flap
class Parser {
    init := input ==> {
        .input = input
        .pos = 0
    }
    
    _peek := ==> {
        .pos < .input.length {
            ret .input[.pos]
        }
        ret -1
    }
    
    _advance := ==> {
        .pos <- .pos + 1
    }
    
    parse_number := ==> {
        result := 0
        @ ._ peek() >= 48 && ._peek() <= 57 {
            result <- result * 10 + (._peek() - 48)
            ._advance()
        }
        ret result
    }
}
```

### Method Chaining

Return `. ` (this) to enable chaining:

```flap
class StringBuilder {
    init = () => {
        .parts = []
    }
    
    append = str => {
        .parts <- .parts :: str
        ret .  // Return this (self)
    }
    
    build = () => {
        result := ""
        @ part in .parts {
            result <- result + part
        }
        ret result
    }
}

str = StringBuilder()
    .append("Hello")
    .append(" ")
    .append("World")
    .build()

println(str)  // "Hello World"
```

### Integration with CStruct

Combine classes and CStruct for high performance:

```flap
cstruct Vec3Data {
    x as float64,
    y as float64,
    z as float64
}

class Vec3 {
    init := (x, y, z) ==> {
        .data = call("malloc", Vec3Data.size as uint64)
        
        unsafe float64 {
            rax <- .data as ptr
            [rax] <- x
            [rax + 8] <- y
            [rax + 16] <- z
        }
    }
    
    dot := other => {
        unsafe float64 {
            rax <- .data as ptr
            rbx <- other.data as ptr
            xmm0 <- [rax]
            xmm0 <- xmm0 * [rbx]
            xmm1 <- [rax + 8]
            xmm1 <- xmm1 * [rbx + 8]
            xmm0 <- xmm0 + xmm1
            xmm1 <- [rax + 16]
            xmm1 <- xmm1 * [rbx + 16]
            xmm0 <- xmm0 + xmm1
        }
    }
    
    free := ==> call("free", .data as ptr)
}

v1 := Vec3(1, 2, 3)
v2 := Vec3(4, 5, 6)
println(v1.dot(v2))  // 32.0
v1.free()
v2.free()
```

### No Inheritance

Flap does not support classical inheritance. Use composition:

```flap
// Instead of:
// class Dog extends Animal { ... }

// Do this:
Animal := {
    eat: ==> println("Eating..."),
    sleep: ==> println("Sleeping...")
}

class Dog <> Animal {
    init := name ==> {
        .name = name
    }
    
    bark := ==> println("Woof!")
}

dog := Dog("Rex")
dog.eat()    // From Animal
dog.bark()   // From Dog
```

### When to Use Classes

**Use classes when:**
- You have related data and behavior
- You want familiar OOP syntax
- You need encapsulation (via naming conventions)
- You're building objects with state

**Don't use classes when:**
- Simple data structures (use maps)
- Stateless functions (use plain functions)
- Performance-critical code (use CStruct + functions)

### Examples

**Stack data structure:**

```flap
class Stack {
    init = () => {
        .items = []
    }
    
    push = item => {
        .items <- .items :: item
    }
    
    pop = () => {
        .items.length == 0 {
            ret ??  // Empty
        }
        last := .items[.items.length - 1]
        .items <- .items[0..<(.items.length - 1)]
        ret last
    }
    
    is_empty = () => .items.length == 0
}

s = Stack()
s.push(1)
s.push(2)
s.push(3)
println(s.pop())  // 3
```

**Simple ORM-like class:**

```flap
class Model {
    Model.table = ""
    
    init := data ==> {
        .data = data
    }
    
    save := ==> {
        query := f"INSERT INTO {Model.table} VALUES (...)"
        // Execute query...
    }
    
    delete := ==> {
        id := .data["id"]
        query := f"DELETE FROM {Model.table} WHERE id = {id}"
        // Execute query...
    }
}

class User <> Model {
    Model.table = "users"
    
    init := (name, email) ==> {
        .data = { name: name, email: email }
    }
}

user := User("Alice", "alice@example.com")
user.save()
```

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

**Note:** The Result type is still `map[uint64]float64`, but with special semantic meaning tracked by the compiler. See [TYPE_TRACKING.md](TYPE_TRACKING.md) for implementation details.

Error codes (4 chars, space-padded, trailing spaces stripped on access):
```
"dv0" - Division by zero
"idx" - Index out of bounds
"key" - Key not found
"typ" - Type mismatch
"nil" - Null pointer
"mem" - Out of memory
"arg" - Invalid argument
"io"  - I/O error
"net" - Network error
"prs" - Parse error
```

### .error Property

Every value has `.error` accessor:

```flap
x = 10 / 0              // Error result
x.error                 // Returns "dv0" (trailing spaces stripped)

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

### Error Propagation Pattern

```flap
// Manual propagation
process = input => {
    step1 = validate(input)
    step1.error { != "" -> step1 }  // Return error early
    
    step2 = transform(step1)
    step2.error { != "" -> step2 }
    
    finalize(step2)
}

// With or! for defaults
compute = input => {
    x = parse(input) or! 0     // Use 0 if parse fails
    y = divide(100, x) or! -1  // Use -1 if division fails
    y * 2
}
```

## Compilation and Execution

### Compiler Usage

```bash
# Compile to executable
flapc program.flap -o program

# Compile with C library
flapc program.flap -o program -lm

# Specify target architecture
flapc program.flap -o program -arch arm64
flapc program.flap -o program -arch riscv64

# Hot reload mode (Unix)
flapc --hot program.flap

# Show version
flapc --version
```

### Supported Architectures

- **x86_64** (AMD64) - Primary platform
- **ARM64** (AArch64) - Full support
- **RISCV64** - Full support

### Compilation Process

1. **Lexing**: Source code → tokens
2. **Parsing**: Tokens → AST
3. **Type Inference**: Track semantic types (see TYPE_TRACKING.md)
4. **Code Generation**: AST → machine code (direct, no IR)
5. **Linking**: Produce ELF (Linux), Mach-O (macOS), or PE (Windows)

### Performance Characteristics

- **Compilation**: Fast (no IR overhead)
- **Tail calls**: Always optimized to loops
- **Arithmetic**: SIMD for vectorizable operations
- **Memory**: Arena allocators for predictable patterns
- **Concurrency**: OS-level parallelism via fork()

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

// Tail-recursive (optimized to loop)
factorial = (n, acc) => n == 0 {
    -> acc
    ~> factorial(n-1, n*acc)
}

// Usage
println(factorial(5, 1))  // 120
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

### List Processing

```flap
// Map, filter, reduce
numbers = [1, 2, 3, 4, 5]

// Map: double each number
doubled = numbers | x => x * 2

// Filter: only even numbers
evens = numbers | x => x % 2 == 0 { 1 -> x ~> [] }

// Reduce: sum all numbers
total = numbers ||| (acc, x) => acc + x

println(f"Total: {total}")
```

### Pattern Matching

```flap
// Value match
classify_number = x => x {
    0 -> "zero"
    1 -> "one"
    2 -> "two"
    ~> "many"
}

// Guard match
classify_age = age => {
    | age < 13 -> "child"
    | age < 18 -> "teen"
    | age < 65 -> "adult"
    ~> "senior"
}

// Nested match
check_value = x => x {
    0 -> "zero"
    ~> x > 0 {
        1 -> "positive"
        0 -> "negative"
    }
}
```

### Error Handling

```flap
// Division with error handling
safe_divide = (a, b) => {
    result = a / b
    result.error {
        "" -> f"Result: {result}"
        ~> f"Error: {result.error}"
    }
}

println(safe_divide(10, 2))   // "Result: 5"
println(safe_divide(10, 0))   // "Error: dv0"

// With or! operator
compute = (a, b) => {
    x = a / b or! 1.0     // Default to 1.0 on error
    y = x * 2
    y
}
```

### Parallel Processing

```flap
data = [1, 2, 3, 4, 5, 6, 7, 8]

// Process in parallel
results = data || x => expensive_computation(x)

// Sum results
total = results ||| (acc, x) => acc + x

println(f"Total: {total}")
```

### Web Server (ENet)

```flap
// Simple echo server
server ==> {
    @ {
        request = => &8080
        println(f"Received: {request}")
        &8080 <- f"Echo: {request}"
    }
}

// HTTP-like handler
handle_request = req => {
    method = req.method
    path = req.path
    
    method {
        "GET" -> path {
            "/" -> "Welcome!"
            "/api" -> "{status: ok}"
            ~> "Not found"
        }
        "POST" -> process_post(req)
        ~> "Method not allowed"
    }
}

server()
```

### C Interop

```flap
// Define C struct
cstruct Buffer {
    data as ptr,
    size as int32,
    capacity as int32
}

// Use C functions
create_buffer = size => {
    ptr = c_malloc(size)
    ptr == 0 {
        1 -> Buffer(0, 0, 0)  // Failed
        ~> Buffer(ptr, 0, size)
    }
}

write_buffer = (buf, data) => {
    buf.size + 1 > buf.capacity {
        1 -> buf  // Buffer full
        ~> {
            c_memcpy(buf.data + buf.size, data, 1)
            buf.size <- buf.size + 1
            buf
        }
    }
}

free_buffer = buf => {
    buf.data != 0 { c_free(buf.data) }
}

// Usage
buf := create_buffer(1024)
buf := write_buffer(buf, 65)  // Write 'A'
free_buffer(buf)
```

### Advanced: Custom Allocator

```flap
// Arena allocator pattern
process_requests = requests => {
    arena {
        results := []
        @ req in requests {
            result = handle_request(req)
            results <- results + [result]
        }
        results
    }
    // All arena memory freed here
}
```

### Advanced: Unsafe Assembly

```flap
// Direct syscall (Linux x86_64)
print_fast = msg => {
    len = msg.length
    unsafe {
        rax <- 1         // sys_write
        rdi <- 1         // stdout
        rsi <- msg       // buffer
        rdx <- len       // length
        syscall
    }
}

// Atomic compare-and-swap
cas = (ptr, old, new) => unsafe int32 {
    rax <- old
    lock cmpxchg [ptr], new
} {
    1  // Success
} {
    0  // Failed
}
```

## Design Rationale

### Why One Universal Type?

Traditional languages have complex type hierarchies:
- Primitive types (int, float, char, bool)
- Reference types (objects, arrays, strings)
- Special types (null, undefined, NaN)
- Type conversions and coercions
- Boxing/unboxing overhead

**Flap's approach:** Everything is `map[uint64]float64`

**Benefits:**
1. **Conceptual simplicity:** Learn one type, understand the entire language
2. **Implementation simplicity:** One memory layout, one set of operations
3. **No type coercion bugs:** No implicit conversions to reason about
4. **Uniform FFI:** Cast to C types only at boundaries
5. **Natural duck typing:** If it has the key, it works
6. **Optimization freedom:** Compiler can represent values efficiently while preserving semantics

### Why Direct Machine Code Generation?

Most compilers use intermediate representations (IR):
- LLVM IR (Rust, Swift, Clang)
- JVM bytecode (Java, Kotlin, Scala)
- WebAssembly (many languages)
- Custom IR (Go, V8)

**Flap's approach:** AST → Machine code directly

**Benefits:**
1. **Fast compilation:** No IR translation overhead
2. **Small compiler:** ~30k lines vs hundreds of thousands
3. **No dependencies:** Self-contained, no LLVM/GCC required
4. **Predictable output:** Same code every time
5. **Full control:** Optimize for Flap's semantics, not general-purpose IR

**Trade-offs:**
- More code per architecture (x86_64, ARM64, RISCV64)
- Manual optimization (no LLVM optimization passes)
- More maintenance burden

**Why it's worth it:** Flap's simplicity makes per-architecture code manageable. The universal type system means optimizations work uniformly across all values.

### Why Fork-Based Parallelism?

Many languages use threads or async/await:
- Shared memory (requires synchronization)
- Green threads (runtime complexity)
- Async/await (function coloring problem)

**Flap's approach:** `fork()` + ENet channels

**Benefits:**
1. **True isolation:** Separate address spaces
2. **No data races:** No shared memory to corrupt
3. **OS-level scheduling:** Leverage existing scheduler
4. **Simple mental model:** Process per task
5. **Fault isolation:** One process crash doesn't kill others

**Trade-offs:**
- Higher memory overhead per task
- Process creation cost
- IPC overhead for communication

**Why it's worth it:** Safety and simplicity trump performance for most use cases. For hot paths, use threads in unsafe blocks.

### Why ENet for Concurrency?

Traditional approaches:
- Channels (Go, Rust): Good but language-specific
- Actors (Erlang, Akka): Heavy runtime
- MPI: Complex API

**Flap's approach:** ENet-style network channels

**Benefits:**
1. **Familiar model:** Network programming concepts
2. **Local or remote:** Same API for both
3. **Simple implementation:** Thin wrapper over ENet library
4. **Battle-tested:** ENet proven in game networking
5. **Scales naturally:** From single machine to distributed

**Design:**
```flap
&8080 <- msg           // Send to local port
&"host:9000" <- msg    // Send to remote host
data = => &8080        // Receive from port
```

Clean, minimal, network-inspired.

### Why No Garbage Collection?

GC languages (Java, Go, Python) have:
- Unpredictable pause times
- Memory overhead (GC metadata)
- Performance cliffs (GC pressure)
- Tuning complexity

**Flap's approach:** Arena allocators + move semantics

**Benefits:**
1. **Predictable performance:** No GC pauses
2. **Low overhead:** No GC metadata
3. **Simple reasoning:** Allocation/deallocation explicit
4. **Natural patterns:** Arena for requests, move for ownership

**Trade-offs:**
- Manual memory management
- Potential for leaks (if not careful)
- More cognitive load

**Why it's worth it:** Systems programming demands predictability. Arenas make manual management tractable.

### Why Minimal Syntax?

Many languages accumulate features:
- Multiple ways to do the same thing
- Special case syntax
- Keyword proliferation

**Flap's approach:** Minimal, orthogonal features

**Examples:**
- One loop construct: `@`
- One function syntax: `=>`
- One block syntax: `{ }`
- Disambiguate by contents, not syntax

**Benefits:**
1. **Easy to learn:** Fewer concepts
2. **Easy to parse:** Simpler compiler
3. **Less bikeshedding:** Fewer style debates
4. **Uniform code:** Looks consistent

**Philosophy:** "One obvious way to do it"

### Why Bitwise Operators Need `b` Suffix?

In C-like languages:
```c
if (x & FLAG)  // Bitwise AND - easy to confuse with &&
if (x | FLAG)  // Bitwise OR - easy to confuse with ||
```

**Flap's approach:** Explicit `b` suffix

```flap
x &b FLAG     // Clearly bitwise
x && y        // Clearly logical
x | transform // Clearly pipe
x |b mask     // Clearly bitwise
```

**Benefits:**
1. **No ambiguity:** Obvious at a glance
2. **No precedence confusion:** Different operators, different precedence
3. **Frees `|` for pipes:** Pipe operator feels natural
4. **Consistent:** All bitwise ops have `b` suffix

### Design Principles Summary

1. **Radical simplification:** One type, one way
2. **Explicit over implicit:** No hidden complexity
3. **Performance without compromise:** Direct code generation
4. **Safety where practical:** Arenas, move semantics, immutable-by-default
5. **Minimal syntax:** Orthogonal features, no redundancy
6. **Predictable behavior:** No GC, no hidden allocations
7. **Systems-level control:** Direct assembly when needed
8. **Familiar concepts:** Borrow from proven designs

**Flap is not trying to be:**
- A replacement for application languages (Python, JavaScript)
- A replacement for safe languages (Rust, Ada)
- A general-purpose language for all domains

**Flap is designed for:**
- Systems programming with radical simplicity
- Performance-critical code with predictable behavior
- Programmers who value minimalism over features
- Domains where direct machine control matters

---

## Frequently Asked Questions

### Is Flap practical for real projects?

Yes, but in specific domains:
- Systems utilities
- Network services
- Embedded systems
- Performance-critical components

Not ideal for:
- Large applications (no module system yet)
- GUI applications (no standard library)
- Rapid prototyping (manual memory management)

### How fast is Flap?

Comparable to C for:
- Arithmetic operations
- Memory operations
- System calls

Slower than C for:
- String operations (map overhead)
- Complex data structures (map overhead)

Faster than C for:
- Compilation (direct code generation)
- FFI (no marshalling overhead)

### Is the universal type system really practical?

Yes, with caveats:
- **Numbers:** Zero overhead (compiler optimizes to registers)
- **Small strings:** Some overhead (map allocation)
- **Large data:** Similar to C (heap allocation either way)
- **FFI:** Zero overhead (direct casts at boundaries)

The compiler's type tracking (see TYPE_TRACKING.md) eliminates most overhead.

### Why not use LLVM?

LLVM would give:
- Better optimization
- More architectures
- Proven backend

But cost:
- 500MB+ dependency
- Slow compilation
- Complex integration
- Loss of control

For Flap's goals (fast compilation, small compiler, direct control), hand-written backends win.

### What about memory safety?

Flap is **not memory-safe by default** like Rust.

However:
- Immutable-by-default reduces bugs
- Arena allocators prevent use-after-free
- Move semantics reduce double-free
- No GC means no GC bugs

Trade-off: Less safe than Rust, simpler to use.

### Can I use Flap in production?

Flap 3.0 is ready for:
- Personal projects
- Internal tools
- Experiments
- Performance prototypes

Not yet ready for:
- Mission-critical systems
- Large teams (no module system)
- Long-term maintenance (young language)

Use your judgment.

---

**For grammar details, see [GRAMMAR.md](GRAMMAR.md)**

**For compiler type tracking, see [TYPE_TRACKING.md](TYPE_TRACKING.md)**

**For documentation accuracy, see [LIBERTIES.md](LIBERTIES.md)**

**For development info, see [DEVELOPMENT.md](DEVELOPMENT.md)**

**For known issues, see [FAILURES.md](FAILURES.md)**
