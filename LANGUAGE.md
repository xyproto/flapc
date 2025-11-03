# Flap Language Specification

**Version:** 1.3.0
**Date:** 2025-10-31

This document describes the complete Flap programming language: syntax, semantics, grammar, and behavior.

## Table of Contents

- [Overview](#overview)
- [Type System](#type-system)
- [Grammar](#grammar)
- [Keywords](#keywords)
- [Operators](#operators)
- [Variables and Assignment](#variables-and-assignment)
- [Control Flow](#control-flow)
- [Functions and Lambdas](#functions-and-lambdas)
- [Loops](#loops)
- [Parallel Programming](#parallel-programming)
- [C FFI](#c-ffi)
- [CStruct](#cstruct)
- [Memory Management](#memory-management)
- [Unsafe Blocks](#unsafe-blocks)
- [Built-in Functions](#built-in-functions)
- [Examples](#examples)

## Overview

Flap is a compiled systems programming language with:
- Direct machine code generation (no LLVM, no runtime)
- Unified type system (`map[uint64]float64` for all values)
- Automatic tail-call optimization
- Immutable-by-default semantics
- C FFI for interfacing with existing libraries
- Arena allocators for scope-based memory management
- Parallel loops with barrier synchronization

## Type System

### Unified Representation

Everything in Flap is internally `map[uint64]float64`:

```flap
42              // {0: 42.0}
"Hello"         // {0: 72.0, 1: 101.0, 2: 108.0, 3: 108.0, 4: 111.0}
[1, 2, 3]       // {0: 1.0, 1: 2.0, 2: 3.0}
{x: 10, y: 20}  // {x: 10.0, y: 20.0}
```

### Type Casting

For C FFI, explicit type casts are supported:

```flap
x as int8      // 8-bit signed integer
x as int16     // 16-bit signed integer
x as int32     // 32-bit signed integer
x as int64     // 64-bit signed integer

x as uint8     // 8-bit unsigned integer
x as uint16    // 16-bit unsigned integer
x as uint32    // 32-bit unsigned integer
x as uint64    // 64-bit unsigned integer

x as float32   // 32-bit float
x as float64   // 64-bit float (no-op, native type)

x as ptr       // Raw pointer
x as cstr      // C null-terminated string
```

**Important:** Always use full type names (`int32`, `uint64`, `float32`), never abbreviations (`i32`, `u64`, `f32`).

## Grammar

The hand-written recursive-descent parser accepts the following grammar:

```ebnf
program         = { newline } { statement { newline } } ;

statement       = use_statement
                | import_statement
                | cstruct_decl
                | arena_statement
                | loop_statement
                | jump_statement
                | defer_statement
                | spawn_statement
                | assignment
                | expression_statement ;

use_statement   = "use" string ;

import_statement = "import" string [ "as" identifier ] ;

cstruct_decl    = "cstruct" identifier [ "packed" ] [ "aligned" "(" number ")" ]
                  "{" { field_decl } "}" ;

field_decl      = identifier "as" c_type [ "," ] ;

c_type          = "int8" | "int16" | "int32" | "int64"
                | "uint8" | "uint16" | "uint32" | "uint64"
                | "float32" | "float64"
                | "ptr" | "cstr" ;

arena_statement = "arena" block ;

loop_statement  = "@" block
                | "@" identifier "in" expression block
                | "@@" identifier "in" expression block ;

jump_statement  = "ret" [ expression ]
                | "->" expression ;

spawn_statement = "spawn" expression ;

defer_statement = "defer" expression ;

assignment      = identifier ("=" | ":=" | "<-") expression
                | identifier ("+=" | "-=" | "*=" | "/=" | "%=" | "**=") expression ;

expression_statement = expression [ match_block ] ;

match_block     = "{" ( default_arm
                      | match_clause { match_clause } [ default_arm ] ) "}" ;

match_clause    = expression [ "->" match_target ] ;

default_arm     = "~>" match_target ;

match_target    = jump_target | expression ;

jump_target     = "ret" [ expression ]
                | "->" expression ;

block           = "{" { statement { newline } } "}" ;

expression              = logical_or_expr ;

logical_or_expr         = logical_and_expr { ("or" | "xor") logical_and_expr } ;

logical_and_expr        = comparison_expr { "and" comparison_expr } ;

comparison_expr         = range_expr [ (rel_op range_expr) | ("in" range_expr) ] ;

rel_op                  = "<" | "<=" | ">" | ">=" | "==" | "!=" ;

range_expr              = additive_expr [ "..<" additive_expr ] ;

additive_expr           = cons_expr { ("+" | "-") cons_expr } ;

cons_expr               = bitwise_expr { "::" bitwise_expr } ;

bitwise_expr            = multiplicative_expr { ("|b" | "&b" | "^b" | "<b" | ">b" | "<<b" | ">>b") multiplicative_expr } ;

multiplicative_expr     = power_expr { ("*" | "/" | "%" | "*+") power_expr } ;

power_expr              = unary_expr { "**" unary_expr } ;

unary_expr              = ("not" | "-" | "#" | "~b") unary_expr
                        | postfix_expr ;

postfix_expr            = primary_expr { "[" expression "]"
                                       | "(" [ argument_list ] ")"
                                       | "as" cast_type
                                       | "!" } ;

cast_type               = "int8" | "int16" | "int32" | "int64"
                        | "uint8" | "uint16" | "uint32" | "uint64"
                        | "float32" | "float64"
                        | "cstr" | "ptr"
                        | "number" | "string" | "list" ;

primary_expr            = number
                        | string
                        | fstring
                        | identifier
                        | "(" expression ")"
                        | lambda_expr
                        | list_literal
                        | map_literal
                        | arena_expr
                        | unsafe_expr ;

arena_expr              = "arena" "{" { statement { newline } } [ expression ] "}" ;

unsafe_expr             = "unsafe" "{" { statement { newline } } [ expression ] "}"
                          [ "{" { statement { newline } } [ expression ] "}" ]
                          [ "{" { statement { newline } } [ expression ] "}" ] ;

lambda_expr             = parameter_list "=>" lambda_body ;

lambda_body             = block | expression [ match_block ] ;

parameter_list          = identifier [ "," identifier ]* ;

argument_list           = expression { "," expression } ;

list_literal            = "[" [ expression { "," expression } ] "]" ;

map_literal             = "{" [ map_entry { "," map_entry } ] "}" ;

map_entry               = ( identifier | string ) ":" expression ;

identifier              = letter { letter | digit | "_" } ;

number                  = [ "-" ] digit { digit } [ "." digit { digit } ] ;

string                  = '"' { character } '"' ;

fstring                 = 'f"' { character | "{" expression "}" } '"' ;
```

## Keywords

### Reserved Keywords

```
and as cstruct defer err in not or ret xor spawn arena unsafe import use alloc call
```

### Contextual Keywords

These are only keywords in specific contexts (e.g., after `as`):

```
int8 int16 int32 int64 uint8 uint16 uint32 uint64 float32 float64
cstr ptr number string list packed aligned
```

You can use contextual keywords as variable names:

```flap
int32 := 100      // OK - variable named int32
x := y as int32   // OK - int32 as type cast
```

## Operators

### Arithmetic

```flap
+     // Addition
-     // Subtraction
*     // Multiplication
/     // Division
%     // Modulo
**    // Exponentiation (power)
*+    // Fused multiply-add
```

### Comparison

```flap
==    // Equal
!=    // Not equal
<     // Less than
<=    // Less than or equal
>     // Greater than
>=    // Greater than or equal
```

### Logical

```flap
and   // Logical AND
or    // Logical OR
xor   // Logical XOR
not   // Logical NOT
```

### Bitwise

```flap
&b    // Bitwise AND
|b    // Bitwise OR
^b    // Bitwise XOR
~b    // Bitwise NOT
<b    // Shift left (logical)
>b    // Shift right (logical)
<<b   // Rotate left
>>b   // Rotate right
```

### Assignment

```flap
=     // Immutable assignment (first time only)
:=    // Mutable assignment (can reassign)
<-    // Update operator (for maps/lists)
+=    // Add and assign
-=    // Subtract and assign
*=    // Multiply and assign
/=    // Divide and assign
%=    // Modulo and assign
**=   // Power and assign
```

### Other

```flap
::    // Cons operator (prepend to list)
..<   // Range operator (0..<10 = 0 to 9)
#     // Length operator
!     // Move operator (postfix - transfers value)
```

## Variables and Assignment

### Immutable by Default

```flap
x = 42        // Immutable - can't reassign
x = 100       // Compile error!
```

### Mutable Variables

```flap
y := 42       // Mutable
y = 100       // OK
y += 10       // OK
```

### Update Operator

```flap
list := [1, 2, 3]
list[0] <- 99          // Updates list to [99, 2, 3]

obj := {x: 10, y: 20}
obj.x <- 100           // Updates x to 100
```

### Move Semantics

Move semantics allow you to explicitly transfer values using the `!` postfix operator. This marks the variable as "moved" and enables compile-time use-after-move detection within the same expression.

```flap
// Basic move - transfers x into the expression
x := 42
y := x! + 100     // y = 142, x is marked as moved

// Error: using moved variable in same expression
a := 10
b := a! + a!      // Compile error: a moved twice

// Move with function calls
process := data => data * 2
value := 100
result := process(value!)  // Transfers value to function
```

**Move Semantics Rules:**

1. **Postfix operator**: `x!` moves the value of `x`
2. **Compile-time detection**: Using a moved variable in the same expression causes an error
3. **Soft semantics**: Unlike Rust, Flap's move semantics are "soft" - they only prevent reuse within the same expression/statement
4. **Type preservation**: The moved expression retains its original type
5. **Optimizer-aware**: The optimizer preserves move semantics throughout optimization passes

**When to Use Moves:**

- Explicit value transfers in complex expressions
- Documenting ownership transfer in function calls
- Preventing accidental reuse within an expression

**Example - Complex Expression:**

```flap
compute := (a, b, c) => a * b + c

x := 10
y := 20
z := 30

// Move all values into the computation
result := compute(x!, y!, z!)

// This would error - x was already moved:
// wrong := x! + result
```

**Note:** Flap's move semantics are lighter than Rust's ownership system. Variables can be used in subsequent statements after being moved - the move only prevents reuse within the same expression where the move occurred.

## Control Flow

### Conditional Expressions

```flap
// Condition without else
x > 10 {
    println("Greater than 10")
}

// Condition as expression with match
result := x {
    0 -> "zero"
    1 -> "one"
    ~> "other"
}

// Multiple conditions
x {
    x < 0 -> "negative"
    x == 0 -> "zero"
    x > 0 -> "positive"
}
```

### Tail Calls

```flap
// Explicit tail call with ->
factorial := (n, acc) => n == 0 {
    -> acc                    // Return accumulator
    ~> factorial(n-1, n*acc)  // Tail call (no stack growth)
}

// Multi-way tail calls
fib := n => n {
    0 -> 0
    1 -> 1
    ~> fib(n-1) + fib(n-2)
}
```

### Defer Statements

Defer statements postpone execution of an expression until the enclosing scope exits. Multiple deferred statements execute in LIFO (Last-In-First-Out) order.

```flap
// Basic defer for cleanup
process_file := filename => {
    file := open_file(filename)
    defer close_file(file)  // Executes when scope exits

    // ... work with file ...
    -> result
}

// LIFO execution order
demo ==> {
    defer println("First")   // Executes third
    defer println("Second")  // Executes second
    defer println("Third")   // Executes first
    println("Body")          // Executes immediately
}
// Output: Body\nThird\nSecond\nFirst

// Resource cleanup pattern
safe_alloc := size => {
    ptr := malloc(size)
    defer free(ptr)  // Guaranteed cleanup

    // Use ptr safely...
    write_i32(ptr, 0, 42)
    value := read_i32(ptr, 0)
    -> value
}
```

**Key Properties:**
- Executes at scope exit (function return, block end, early return)
- LIFO order: Last defer executes first
- Always runs even if errors occur before
- Common use: Resource cleanup (files, memory, locks)

## Functions and Lambdas

### Lambda Syntax

```flap
// Single parameter
square := x => x * x

// Multiple parameters
add := (a, b) => a + b

// Block body
complex := (x, y) => {
    temp := x * 2
    result := temp + y
    -> result
}

// With pattern matching
classify := x => x {
    0 -> "zero"
    ~> x > 0 { -> "positive" ~> "negative" }
}
```

### Function Calls

```flap
result := add(10, 20)
println(result)

// C function calls
ptr := call("malloc", 1024 as uint64)
call("free", ptr as ptr)
```

## Loops

### Infinite Loop

```flap
@ {
    update()
    render()
}
```

### Range Loop

```flap
// Exclusive range (0 to 9)
@ i in 0..<10 {
    println(i)
}

// List iteration
items := [1, 2, 3, 4, 5]
@ item in items {
    println(item)
}
```

### Loop Control

```flap
@ i in 0..<100 {
    i % 2 == 0 {
        continue  // Skip even numbers
    }
    i > 50 {
        break     // Stop at 50
    }
    println(i)
}
```

## Parallel Programming

### Parallel Loops

```flap
// Runs on all CPU cores with barrier synchronization
@@ i in 0..<1000 {
    process(i)
}  // Barrier - all threads wait here
```

### Atomic Operations

```flap
counter_ptr := call("malloc", 8 as uint64)
atomic_store(counter_ptr, 0)

@@ i in 0..<1000 {
    atomic_add(counter_ptr, 1)
}

result := atomic_load(counter_ptr)
println(result)  // 1000
```

Available atomic operations:
- `atomic_load(ptr)` - Atomically read value
- `atomic_store(ptr, value)` - Atomically write value
- `atomic_add(ptr, value)` - Atomically add and return old value
- `atomic_cas(ptr, expected, desired)` - Compare-and-swap

### Spawn

```flap
// Fire-and-forget process
spawn background_task()

// Fork and continue
pid := spawn {
    // Child process code
    println("Child running")
}
```

## C FFI

### Importing C Libraries

```flap
import sdl3 as sdl
import opengl as gl
```

The compiler automatically reads DWARF debug info to infer C function signatures.

### Calling C Functions

```flap
// SDL3 example
init_result := sdl.SDL_Init(sdl.SDL_INIT_VIDEO)
window := sdl.SDL_CreateWindow("Game", 800, 600, 0)

// Direct C calls
ptr := call("malloc", 1024 as uint64)
call("memset", ptr as ptr, 0 as int32, 1024 as uint64)
call("free", ptr as ptr)
```

### Type Conversions

```flap
// Flap to C
c_int := flap_num as int32
c_ptr := flap_num as ptr
c_str := flap_string as cstr

// C to Flap (automatic)
flap_num := c_function()
```

## CStruct

C-compatible structure definitions for FFI:

### Basic CStruct

```flap
cstruct Vec3 {
    x as float32
    y as float32
    z as float32
}

// Generated constants:
// Vec3_SIZEOF = 12
// Vec3_x_OFFSET = 0
// Vec3_y_OFFSET = 4
// Vec3_z_OFFSET = 8
```

### Packed Structures

```flap
cstruct NetworkPacket packed {
    magic as uint16
    type as uint8
    length as uint32
}
// No padding between fields
```

### Aligned Structures

```flap
cstruct CacheAligned aligned(64) {
    data as uint64
}
// Aligned to 64-byte boundary
```

### Using CStructs

```flap
// Allocate
ptr := call("malloc", Vec3_SIZEOF as uint64)

// Write fields
write_f32(ptr, Vec3_x_OFFSET as int32, 1.0)
write_f32(ptr, Vec3_y_OFFSET as int32, 2.0)
write_f32(ptr, Vec3_z_OFFSET as int32, 3.0)

// Read fields
x := read_f32(ptr, Vec3_x_OFFSET as int32)
y := read_f32(ptr, Vec3_y_OFFSET as int32)
z := read_f32(ptr, Vec3_z_OFFSET as int32)

// Free
call("free", ptr as ptr)
```

## Memory Management

### Manual Allocation

```flap
// Allocate
ptr := call("malloc", 1024 as uint64)

// Use memory
write_i32(ptr, 0, 42)
value := read_i32(ptr, 0)

// Free
call("free", ptr as ptr)
```

### Arena Allocation

Scope-based automatic memory management:

```flap
arena {
    // All allocations freed at block exit
    buffer := alloc(1024)
    entities := alloc(count * size)

    // Use memory...
    write_i32(buffer, 0, 100)
}  // Everything freed here
```

### Arena in Game Loop

```flap
@ frame in 0..<1000 {
    arena {
        // Per-frame allocations
        visible := alloc(entity_count * 8)
        commands := alloc(command_count * 16)

        // Do frame work...
    }  // Zero fragmentation, instant cleanup
}
```

## Unsafe Blocks

Direct register access for performance-critical code:

### Unified Syntax

Works across x86-64, ARM64, and RISC-V:

```flap
result := unsafe {
    a <- 42        // a = rax (x86), x0 (ARM), a0 (RISC-V)
    b <- 10
    c <- a + b
}  // Returns c (52)
```

Register aliases:
- `a-h`: General purpose registers (a=rax/x0/a0, b=rbx/x1/a1, etc.)

### Per-Architecture Syntax

```flap
value := unsafe {
    rax <- 100     // x86-64 specific
} {
    x0 <- 100      // ARM64 specific
} {
    a0 <- 100      // RISC-V specific
}
```

### Memory Operations

```flap
unsafe {
    rax <- ptr as ptr
    rbx <- 42
    [rax + 0] <- rbx        // Write to memory
    rcx <- [rax + 0]        // Read from memory
}
```

## Built-in Functions

### Output

```flap
print("Hello")           // Print without newline
println("World")         // Print with newline
printf("x=%d\n", x)      // C-style formatted print
```

### String Operations

```flap
len := #str              // String length
concat := str1 + str2    // Concatenation

// F-strings (interpolated strings)
name := "Alice"
age := 30
msg := f"Hello, {name}! You are {age} years old."

// F-strings with expressions
a := 5
b := 7
result := f"Sum: {a + b}, Product: {a * b}"

// Escaping braces in f-strings
text := f"Use {{ and }} for literal braces"
```

### List Operations

```flap
len := #list             // List length
append := list :: item   // Prepend item
```

### Memory Access

```flap
// Write operations
write_i8(ptr, offset, value)
write_i16(ptr, offset, value)
write_i32(ptr, offset, value)
write_i64(ptr, offset, value)
write_u8(ptr, offset, value)
write_u16(ptr, offset, value)
write_u32(ptr, offset, value)
write_u64(ptr, offset, value)
write_f32(ptr, offset, value)
write_f64(ptr, offset, value)

// Read operations
value := read_i8(ptr, offset)
value := read_i16(ptr, offset)
value := read_i32(ptr, offset)
value := read_i64(ptr, offset)
value := read_u8(ptr, offset)
value := read_u16(ptr, offset)
value := read_u32(ptr, offset)
value := read_u64(ptr, offset)
value := read_f32(ptr, offset)
value := read_f64(ptr, offset)
```

### Math Functions

```flap
sqrt(x)      // Square root
sin(x)       // Sine
cos(x)       // Cosine
tan(x)       // Tangent
abs(x)       // Absolute value
floor(x)     // Floor
ceil(x)      // Ceiling
round(x)     // Round
log(x)       // Natural logarithm
exp(x)       // Exponential
pow(x, y)    // Power
```

### System

```flap
exit(code)   // Exit program
```

## Examples

### Factorial

```flap
// Iterative
factorial := n => {
    result := 1
    @ i in 1..<=n {
        result *= i
    }
    -> result
}

// Tail-recursive
factorial := (n, acc) => n == 0 {
    -> acc
    ~> factorial(n-1, n*acc)
}
```

### Fibonacci

```flap
fib := n => n < 2 {
    -> n
    ~> fib(n-1) + fib(n-2)
}
```

### Game Loop with SDL3

```flap
import sdl3 as sdl

sdl.SDL_Init(sdl.SDL_INIT_VIDEO)
window := sdl.SDL_CreateWindow("Game", 800, 600, 0)
renderer := sdl.SDL_CreateRenderer(window, 0)

running := 1
@ running == 1 {
    // Handle input
    event_ptr := alloc(56)
    has_event := sdl.SDL_PollEvent(event_ptr)

    has_event {
        event_type := read_u32(event_ptr, 0)
        event_type == sdl.SDL_EVENT_QUIT {
            running = 0
        }
    }

    // Clear screen
    sdl.SDL_RenderClear(renderer)

    // Draw game
    // ...

    // Present
    sdl.SDL_RenderPresent(renderer)
    sdl.SDL_Delay(16)
}

sdl.SDL_Quit()
```

### Parallel Physics

```flap
cstruct Particle {
    x as float64
    y as float64
    vx as float64
    vy as float64
}

particle_count := 10000
particles := call("malloc", particle_count * Particle_SIZEOF as uint64)

// Initialize particles
@ i in 0..<particle_count {
    offset := i * Particle_SIZEOF
    write_f64(particles, offset + Particle_x_OFFSET as int32, rand())
    write_f64(particles, offset + Particle_y_OFFSET as int32, rand())
    write_f64(particles, offset + Particle_vx_OFFSET as int32, 0)
    write_f64(particles, offset + Particle_vy_OFFSET as int32, 0)
}

// Update loop
@ frame in 0..<1000 {
    @@ i in 0..<particle_count {
        offset := i * Particle_SIZEOF

        // Read position and velocity
        x := read_f64(particles, offset + Particle_x_OFFSET as int32)
        y := read_f64(particles, offset + Particle_y_OFFSET as int32)
        vx := read_f64(particles, offset + Particle_vx_OFFSET as int32)
        vy := read_f64(particles, offset + Particle_vy_OFFSET as int32)

        // Apply gravity
        vy = vy + 0.01

        // Update position
        x = x + vx
        y = y + vy

        // Bounce at edges
        y > 1.0 {
            y = 1.0
            vy = -vy * 0.9
        }

        // Write back
        write_f64(particles, offset + Particle_x_OFFSET as int32, x)
        write_f64(particles, offset + Particle_y_OFFSET as int32, y)
        write_f64(particles, offset + Particle_vx_OFFSET as int32, vx)
        write_f64(particles, offset + Particle_vy_OFFSET as int32, vy)
    }
}
```

## Special Notes

### Tail Call Optimization

Flap automatically performs tail-call optimization. Use `->` to indicate explicit tail returns:

```flap
loop := n => {
    println(n)
    -> loop(n + 1)  // Tail call - no stack growth
}
```

### Newlines as Statement Separators

Newlines separate statements. No semicolons required:

```flap
x := 42
y := 100
println(x + y)
```

### Comments

```flap
// Line comment

/* Multi-line comments not supported yet */
```

### String Escapes

```flap
"\n"   // Newline
"\t"   // Tab
"\r"   // Carriage return
"\\"   // Backslash
"\""   // Quote
```

### Contextual Type Keywords

Type names are only keywords after `as`. You can use them as identifiers elsewhere:

```flap
int32 := 42           // OK - variable
result := x as int32  // OK - type cast
```

---

## Error Handling and Diagnostics

### Railway-Oriented Error System

The Flap compiler uses a railway-oriented error handling system that collects and reports multiple errors instead of stopping at the first one.

**Error Categories:**
1. **Syntax Errors**: Invalid language syntax
2. **Semantic Errors**: Type mismatches, undefined variables
3. **Code Generation Errors**: Register allocation failures, etc.

**Example Error Output:**

```
error: undefined variable 'sum'
  --> example.flap:5:9
   |
 5 |     total <- sum + i
   |              ^^^
   |
help: did you mean 'total'?

error: cannot update immutable variable 'x'
  --> example.flap:8:5
   |
 8 |     x <- x + 1
   |     ^
   |
help: declare 'x' as mutable with ':='
```

### Error Recovery

The compiler attempts to recover from errors and continue parsing to find additional issues:

- **Syntax errors**: Skips to next statement boundary
- **Undefined variables**: Creates placeholder, continues analysis
- **Type errors**: Reports mismatch, continues with expected type

**Maximum Errors**: By default, the compiler stops after collecting 10 errors to avoid overwhelming output.

### Compile-Time vs Runtime Errors

**Compile-time errors** (caught by the compiler):
- Undefined variables
- Type mismatches in known contexts
- Syntax errors
- Immutable variable updates

**Runtime errors** (not caught, will crash):
- Division by zero
- Array index out of bounds
- Null pointer dereference (in unsafe blocks)

Use assertions and defensive programming for runtime safety.

---

**For more examples, see the `testprograms/` directory with 344+ working Flap programs.**
