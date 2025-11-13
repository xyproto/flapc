# Flap Language Specification

**Version:** 2.0.0
**Date:** 2025-11-06
**Status:** Final - This is the complete, authoritative language specification

This document describes the complete Flap programming language: syntax, semantics, grammar, and behavior. This specification is intended to be stable and serve as the definitive reference for Flap language implementation.

## Table of Contents

- [What Makes Flap Unique](#what-makes-flap-unique)
- [Overview](#overview)
- [Design Philosophy](#design-philosophy)
- [Type System](#type-system)
- [Lexical Elements](#lexical-elements)
- [Grammar](#grammar)
- [Keywords](#keywords)
- [Operators](#operators)
- [Variables and Assignment](#variables-and-assignment)
- [Control Flow](#control-flow)
- [Functions and Lambdas](#functions-and-lambdas)
- [Loops](#loops)
- [Parallel Programming](#parallel-programming)
- [ENet Channels](#enet-channels-concurrency-via-message-passing)
- [C FFI](#c-ffi)
- [CStruct](#cstruct)
- [Memory Management](#memory-management)
- [Unsafe Blocks](#unsafe-blocks)
- [Built-in Functions](#built-in-functions)
- [Examples](#examples)
- [Classes and Objects](#classes-and-objects)

## What Makes Flap Unique

Flap brings together several novel or rare features that distinguish it from other systems programming languages:

### 1. **Universal Map Type System**
The entire language is built on a single type: `map[uint64]float64`. Every value—numbers, strings, lists, functions—is represented as this hash map. This radical simplification enables:
- No type system complexity
- Uniform memory representation
- Natural duck typing
- Simple FFI (cast to native types only at boundaries)

### 2. **Direct Machine Code Generation (No IR, No LLVM)**
The compiler emits x86_64, ARM64, and RISCV64 machine code directly from the AST:
- **No intermediate representation** - AST → machine code in one pass
- **No dependencies** - completely self-contained compiler
- **Fast compilation** - no IR translation overhead
- **Small compiler** - ~30k lines of Go
- **Deterministic output** - same code every time

### 3. **Match Blocks as Function Bodies**
Every function body `{ ... }` is actually a match expression:
```flap
factorial := n => {
    n == 0 -> 1
    ~> n * factorial(n - 1)
}
```
Lines without `->` become the default case. This unifies:
- Pattern matching
- Function bodies  
- Conditional expressions
- Guard clauses

### 4. **Unified Lambda Syntax (`=>`)**
All functions—named, anonymous, inline—use the same `=>` arrow:
```flap
f := x => x + 1                    // Simple lambda
g := (x, y) => x * y               // Multiple parameters
h := x => { x > 0 -> "pos" }       // With match block
```

### 5. **Bitwise Operators with `b` Suffix**
All bitwise operations are suffixed with `b` to eliminate ambiguity:
```flap
<<b >>b <<<b >>>b    // Shifts and rotates
&b |b ^b ~b          // Logical operations
```
No confusion with `<`, `>`, `&`, `|` in other contexts.

### 6. **Explicit String Encoding (`.bytes`, `.runes`)**
Strings are UTF-8 by default, but you choose how to iterate:
```flap
".bytes"    // Iterate as bytes (uint8)
".runes"    // Iterate as Unicode code points (runes)
```
No hidden cost—you decide the representation.

### 7. **ENet for All Concurrency**
Instead of channels, goroutines, or async/await, Flap uses **ENet** (reliable UDP) for:
- Inter-process communication
- Network messaging  
- All concurrency coordination

Same syntax for local and remote:
```flap
:8080 <= "message"              // Local IPC
:server.com:8080 <= "message"   // Network
msg = => :5000                  // Receive message
```

### 8. **Fork-Based Process Model**
Flap uses Unix `fork()` semantics with `spawn`:
```flap
spawn worker()    // New process (copy-on-write)
```
No threads, no shared memory bugs—just processes and messages.

### 9. **Pipe Operators for Data Flow**
Three pipe operators for different execution models:
```flap
|      // Sequential pipe: value → function
||     // Parallel pipe: map over list in parallel  
|||    // Reduce pipe: fold list to single value
```

### 10. **C FFI via DWARF Debug Info**
Import C libraries without writing bindings:
```flap
import sdl3 as sdl
window := sdl.SDL_CreateWindow("Game", 800, 600, 0)
```
The compiler reads DWARF debug info to infer function signatures automatically.

### 11. **CStruct with Direct Memory Access**
Define C-compatible structs with computed field offsets:
```flap
cstruct Vec3 {
    x as float32,
    y as float32,
    z as float32
}

ptr[Vec3.x.offset] <- 1.0 as float32
```
Access fields by offset—no marshaling overhead.

### 12. **Arena Allocators as Language Feature**
Scoped memory management without garbage collection:
```flap
result := arena {
    data := allocate(1024)
    process(data)
    final_value
}
// All arena memory freed here
```

### 13. **Tail-Call Optimization Always On**
The compiler automatically optimizes tail calls—no special syntax required:
```flap
factorial := (n, acc) => {
    n == 0 -> acc
    ~> factorial(n - 1, acc * n)    // Optimized to loop
}
```

### 14. **Cryptographically Secure Random Operator**
Built-in operator for secure random numbers:
```flap
???    // Returns float64 in [0.0, 1.0) from kernel entropy
```
Uses `getrandom()` syscall—no seeding needed.

### 15. **Move Operator `!` (Postfix)**
Explicit ownership transfer:
```flap
large_data!    // Moves value, original becomes invalid
```
Prevents accidental copies of large structures.

### 16. **Result Type with NaN Error Encoding**
Errors encoded in NaN bit patterns (no Result<T, E> wrapper):
```flap
x := 10 / 0            // Returns NaN with error code
is_error(x) { ... }    // Check for error
```
Zero-cost error handling.

### 17. **Immutable-by-Default**
Variables are immutable unless explicitly mutable:
```flap
x := 5        // Immutable (can't change)
y <- 10       // Mutable (can change)
```

### 18. **Named Logical Operators**
```flap
and or xor not    // Words, not symbols
```
More readable, especially for newcomers.

### 19. **No Garbage Collector**
Manual memory management with:
- Stack allocation (default)
- Arena allocators (scoped)
- C malloc/free (explicit)
- No pauses, no runtime overhead

### 20. **Single-Pass Compilation**
Lexer → Parser → AST → Optimizer → Machine Code → ELF
- No IR layers
- Fast incremental builds
- Predictable compilation times

---

These features combine to create a language that is:
- **Minimal** - Few concepts, orthogonally combined
- **Fast** - No runtime, no GC, direct machine code
- **Explicit** - No magic, no hidden costs
- **Composable** - Small features that work together
- **Pragmatic** - Solves real problems with simple tools

## Overview

Flap is a compiled systems programming language with:
- Direct machine code generation (no LLVM, no runtime)
- Unified type system (`map[uint64]float64` for all values)
- Automatic tail-call optimization
- Immutable-by-default semantics
- C FFI for interfacing with existing libraries
- Arena allocators for scope-based memory management
- Parallel loops with barrier synchronization

## Design Philosophy

Flap follows these core principles:

### Explicit Over Implicit
- **Type casts use `as` keyword**: `x as uint32` not `uint32(x)`
- **No implicit conversions**: Every type change must be explicit
- **Named operators**: `and`, `or`, `not` instead of `&&`, `||`, `!`

### Minimal Keywords
- **No `range` keyword**: Use `0..<10` directly instead of `range(0, 10)`
- **No `function` keyword**: Functions are just lambdas assigned to variables
- **Contextual keywords**: Type names (`int32`, `uint64`) are only keywords after `as`

### Calculated, Not Hardcoded
The Flap compiler (`flapc`) minimizes hardcoded "magic numbers":
- **Sizes calculated**: Use `Type.size` not hardcoded `12`
- **Offsets calculated**: Use `Type.field.offset` not hardcoded `8`
- **Global constants**: Define constants at the top of files when needed
- **Derived values**: Calculate from other constants when possible

### Consistency
- **Assignment operators**: `=` (immutable), `:=` (mutable), `<-` (update)
- **Lambda syntax**: Always `=>` never `->`
- **Match arms**: `->` for explicit jumps/returns, `~>` for default case
- **Loop prefix**: `@` for serial, `@@` for parallel

### Simplicity
- **One way to do things**: Prefer a single, clear approach over multiple alternatives
- **Minimal syntax**: Newlines separate statements, no semicolons
- **Unified type**: Everything is `map[uint64]float64` internally
- **Direct assembly**: No intermediate representations, straight to machine code

## Type System

### Unified Representation

Everything in Flap is internally `map[uint64]float64`:

```flap
42              // {0: 42.0}
"Hello"         // {0: 72.0, 1: 101.0, 2: 108.0, 3: 108.0, 4: 111.0}
[1, 2, 3]       // {0: 1.0, 1: 2.0, 2: 3.0}
{x: 10, y: 20}  // {x: 10.0, y: 20.0}
[]              // {} (universal empty value)
```

**Empty Value `[]`:**

The empty literal `[]` represents a universal empty value in Flap's unified type system. Since all values are `map[uint64]float64` internally, `[]` is simply an empty map. It can be used in any context where an empty list or map is needed:

```flap
// Empty value in different contexts
empty := []                  // Universal empty
list := 1 :: 2 :: []        // Builds list from empty
numbers := []               // Empty that will become list when populated
result := []                // Empty that will become map when keys added

// The type emerges from usage
list[0] <- 42               // Now it's a list (sequential keys)
result["key"] <- 100        // Now it's a map (string/arbitrary keys)
```

### Internal Memory Layout

While conceptually everything is `map[uint64]float64`, the compiler uses two distinct memory layouts for efficiency:

**Lists (Position-Indexed)**
- Syntax: `[1.0, 2.0, 3.0]`
- Memory: `[length: float64][element0: float64][element1: float64]...`
- Size: `8 + (length × 8)` bytes
- Indexing: Position-based offset calculation
- Access: `arr[1]` loads from `base_ptr + 8 + (1 × 8)`

**Maps (Key-Indexed)**
- Syntax: `{0: 10.0, 5: 20.0}` (explicit keys)
- Memory: `[count: float64][key0: float64][val0: float64][key1: float64][val1: float64]...`
- Size: `8 + (count × 16)` bytes
- Indexing: Linear search comparing keys
- Access: `map[5]` scans pairs until key matches

Lists use contiguous position-based storage, while maps use sparse key-value pairs. The distinction allows dense arrays to avoid storing redundant position keys, while true maps can have arbitrary sparse keys like `{100: 1.0, 500: 2.0}`.

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
## Lexical Elements

### Comments

Flap supports single-line comments using `//`:

```flap
// This is a comment
x = 42  // Inline comment after code
```

Multi-line comments are not supported. Use multiple single-line comments instead.

### Numeric Literals

**Integer Literals:**
```flap
42          // Decimal integer
-17         // Negative integer
```

**Floating-Point Literals:**
```flap
3.14        // Standard notation
-0.5        // Negative float
6.022e23    // Scientific notation (6.022 × 10²³)
1.5e-10     // Negative exponent
```

**Notes:**
- All numeric literals are stored as float64 internally
- Integer semantics require explicit casting: `x as int64`
- Underscore separators in numbers are not supported

### String Literals

**Regular Strings:**
```flap
"hello world"
"path/to/file"
""  // Empty string
```

**F-Strings (Interpolated):**
```flap
name = "Alice"
greeting = f"Hello, {name}!"  // "Hello, Alice!"

x = 42
message = f"The answer is {x}"  // "The answer is 42"
```

**Escape Sequences:**
```flap
"\n"   // Newline
"\t"   // Tab
"\""   // Double quote
"\\"   // Backslash
```

### Identifiers

Identifiers (variable names, function names) must:
- Start with a letter or underscore: `_`, `a`-`z`, `A`-`Z`
- Continue with letters, digits, or underscores: `a`-`z`, `A`-`Z`, `0`-`9`, `_`

```flap
valid_name
CamelCase
snake_case
_private
counter123
```

Reserved keywords cannot be used as identifiers (see [Keywords](#keywords) section).


The hand-written recursive-descent parser accepts the following grammar:

```ebnf
program         = { newline } { statement { newline } } ;

statement       = use_statement
                | import_statement
                | cstruct_decl
                | arena_statement
                | loop_statement
                | receive_loop
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

receive_loop    = "@" identifier "," identifier "in" expression [ "max" expression ] block ;

jump_statement  = "ret" [ "@" [ number ] ] [ expression ]
                | "->" expression ;

spawn_statement = "spawn" expression ;

defer_statement = "defer" expression ;

assignment      = identifier ("=" | ":=" | "<-") expression
                | identifier ("+=" | "-=" | "*=" | "/=" | "%=" | "**=") expression
                | indexed_expr "<-" expression ;

indexed_expr    = identifier "[" expression "]" ;

expression_statement = expression [ match_block ] ;

match_block     = "{" ( default_arm
                      | match_clause { match_clause } [ default_arm ] ) "}" ;

match_clause    = expression [ "->" match_target ] ;

default_arm     = "~>" match_target ;

match_target    = jump_target | expression ;

jump_target     = "ret" [ "@" [ number ] ] [ expression ]
                | "->" expression ;

block           = "{" { statement { newline } } "}" ;

expression              = send_expr ;

send_expr               = receive_expr [ "<=" receive_expr ] ;

receive_expr            = "=>" pipe_expr | pipe_expr ;

pipe_expr               = reduce_pipe_expr { "|||" reduce_pipe_expr } ;

reduce_pipe_expr        = parallel_pipe_expr { "||" parallel_pipe_expr } ;

parallel_pipe_expr      = sequential_pipe_expr { "|" sequential_pipe_expr } ;

sequential_pipe_expr    = logical_or_expr ;

logical_or_expr         = logical_and_expr { ("or" | "xor") logical_and_expr } ;

logical_and_expr        = comparison_expr { "and" comparison_expr } ;

comparison_expr         = range_expr [ (rel_op range_expr) | ("in" range_expr) ] ;

rel_op                  = "<" | "<=" | ">" | ">=" | "==" | "!=" ;

range_expr              = additive_expr [ ( ".." | "..<" ) additive_expr ] ;

additive_expr           = cons_expr { ("+" | "-") cons_expr } ;

cons_expr               = bitwise_expr { "::" bitwise_expr } ;

bitwise_expr            = multiplicative_expr { ("|b" | "&b" | "^b" | "<<b" | ">>b" | "<<<b" | ">>>b") multiplicative_expr } ;

multiplicative_expr     = power_expr { ("*" | "/" | "%" | "*+") power_expr } ;

power_expr              = unary_expr { "**" unary_expr } ;

unary_expr              = ("not" | "-" | "#" | "~b" | "^" | "_") unary_expr
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
                        | enet_address
                        | identifier
                        | "(" expression ")"
                        | lambda_expr
                        | list_literal
                        | map_literal
                        | arena_expr
                        | unsafe_expr
                        | "???" ;

enet_address            = ":" ( digit { digit } | identifier [ ":" digit { digit } ] ) ;

arena_expr              = "arena" "{" { statement { newline } } [ expression ] "}" ;

unsafe_expr             = "unsafe" "{" { statement { newline } } [ expression ] "}"
                          [ "{" { statement { newline } } [ expression ] "}" ]
                          [ "{" { statement { newline } } [ expression ] "}" ] ;

lambda_expr             = [ parameter_list ] "=>" lambda_body ;

lambda_body             = block | expression [ match_block ] ;

// Lambda arrow semantics:
// => : Unified arrow for all functions
// Block semantics inferred from content:
//   - Contains -> or ~> : match block
//   - No arrows : statement block

parameter_list          = identifier [ "," identifier ]*
                        | "(" [ identifier [ "," identifier ]* ] ")" ;

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
alias and arena as cstruct defer err has hot import in not or ret spawn unsafe use xor
```

**Special keywords:**
- `ret` - Return from function/loop
- `err` - Return error value (NaN-based Result types)
- `has` - Check if key exists in map
- `hot` - Mark function for hot-reloading
- `alias` - Create type or function aliases

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

**Operator Precedence Note:**

Logical operators (`and`, `or`) have appropriate precedence so that expressions like `a >= 0.3 and a <= 0.7` work without needing parentheses. The comparison operators (`<`, `<=`, `>`, `>=`, `==`, `!=`) bind tighter than logical operators.

### Bitwise

```flap
&b     // Bitwise AND
|b     // Bitwise OR
^b     // Bitwise XOR
~b     // Bitwise NOT
<<b    // Shift left
>>b    // Shift right
<<<b   // Rotate left
>>>b   // Rotate right
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

### Range and List

```flap
::    // Cons operator (item :: list prepends item to list)
..    // Inclusive range (1..10 = 1 to 10)
..<   // Exclusive range (1..<10 = 1 to 9)
#     // Length operator
^     // Head operator (first element of list)
_     // Tail operator (all but first element of list)
```

### Pipe Operators

```flap
|     // Sequential pipe (passes value to next function)
||    // Parallel pipe (maps function over list elements in parallel)
|||   // Reduce pipe (summarizes/folds list to single value)
```

### Concurrency Operators

```flap
<==   // Send operator (sends message to address)
```

All addresses are strings:
```flap
":8080" <== "hello"                  // Send to local port (IPC)
"server.com:8080" <== "data"         // Send to remote address (network)
```

### Other

```flap
!     // Move operator (postfix - transfers value)
???   // Secure random number operator (returns float64 in [0.0, 1.0))
```

**Random Operator `???`:**

The random operator provides cryptographically secure random numbers:
- Returns a float64 value in the range [0.0, 1.0)
- Uses Linux `getrandom()` syscall for secure random bytes
- Does not require seeding (kernel entropy pool is used)
- Suitable for security-sensitive applications
- Each call generates fresh random bytes

Example usage:
```flap
// Secure random value
x := ???                     // 0.0 <= x < 1.0

// Random integer in range
dice := (??? * 6) as int64   // 0, 1, 2, 3, 4, or 5
roll := ((??? * 6) as int64) + 1  // 1, 2, 3, 4, 5, or 6

// Random selection from list
items := [10, 20, 30, 40, 50]
idx := (??? * #items) as int64
chosen := items[idx]
```

## Variables and Assignment

### Immutable by Default

```flap
x = 42        // Immutable - can't reassign
x <- 100      // NOT OK, trying to assign a new value to an immutable variable
x += 10       // NOT OK, trying to modify the value of an aimmutable variable
x = 100       // NOT OK, trying to initialize a new immutable variable with the same name
```

### Mutable Variables

```flap
y := 42       // Mutable
y <- 100      // OK, assignment
y += 10       // OK, modification
y = 100       // NOT OK, trying to initialize a new immutable variable with the same name
```

### Shadowing and Scoping Rules

Flap allows variable shadowing in nested scopes but not in the same scope:

```flap
// Shadowing in nested scopes - ALLOWED
x = 10
println(x)       // 10

{
    x = 20       // OK - shadows outer x in nested scope
    println(x)   // 20
}

println(x)       // 10 - outer x unchanged

// Shadowing in same scope - NOT ALLOWED
a = 5
a = 10           // ERROR: cannot redefine immutable variable in same scope

// Function parameters create new scope
process = x => {
    x = x * 2    // OK - shadows parameter in function body
    ret x
}
```

**Scoping Rules:**
- Each `{ }` block creates a new scope
- Function bodies are their own scope
- Loop bodies are their own scope
- Variables shadow outer scopes but don't modify them
- Mutable variables (`:=`) can be updated with `<-` in same scope

### Update Operator

```flap
list := [1, 2, 3]
list[0] <- 99          // Updates list to [99, 2, 3]
list.0 <- 42           // Updates list to [42, 2, 3]

obj := {x: 10, y: 20}
obj.x <- 100           // Updates x to 100
obj[0] <- 42           // Updates x to 42 (Flap maps are ordered)
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

// Multiple conditions (the presence of -> indicates a match block instead of a function block)
{
    x < 0 -> "negative"
    x == 0 -> "zero"
    x > 0 -> "positive"
}

// Matching a positive and negative value
result := x > 0 {
    -> "positive"
    ~> "not positive"
}

// Optional arrows - arrows can be omitted when intent is clear
result := x > 0 {
    "positive"      // No arrow needed for single positive case
    ~> "not positive"
}

// Mixed syntax - explicit arrow for one branch
x == 0 {
    println("zero")  // No arrow - just execute code
    ~> println("not zero")
}
```

**Match Expression Syntax:**

Match expressions distinguish between **value matching** and **guard conditions**:

1. **Value matching**: `value -> result` - Matches when input equals value
2. **Guard conditions**: `| condition -> result` - Evaluates boolean expression (use `|` prefix)
3. **Default case**: `~> result` - Always uses `~>` arrow
4. **Implicit form**: Expression without arrow (for simple conditionals)

```flap
// Value matching (checks equality)
result := x {
    0 -> "zero"           // Matches when x == 0
    1 -> "one"            // Matches when x == 1
    2 -> "two"            // Matches when x == 2
    ~> "other"            // Default case
}

// Guard conditions (use | prefix for boolean expressions)
result := x {
    | x > 10 -> "large"       // Guard: evaluates x > 10
    | x > 0 -> "positive"     // Guard: evaluates x > 0
    | x == 0 -> "zero"        // Guard: evaluates x == 0
    ~> "negative"             // Default case
}

// Can mix value matching and guards
result := x {
    0 -> "zero"               // Value match
    | x > 0 && x < 10 -> "small positive"  // Guard
    | x >= 10 -> "large"      // Guard
    ~> "negative"             // Default
}

// Simple conditional (no value/guard distinction)
x > 0 {
    println("positive")       // Executes when x > 0
    ~> println("not positive") // Default case
}
```

### Tail Calls

```flap
// Explicit tail call with ->
factorial = (n, acc) => n == 0 {
    -> acc                    // Return accumulator
    ~> factorial(n-1, n*acc)  // Tail call (no stack growth)
}

// Multi-way tail calls
fib = n => n {
    0 -> 0
    1 -> 1
    ~> fib(n-1) + fib(n-2)
}
```

### Defer Statements

Defer statements postpone execution of an expression until the enclosing scope exits. Multiple deferred statements execute in LIFO (Last-In-First-Out) order.

```flap
// Basic defer for cleanup
process_file = filename => {
    file := open_file(filename)
    defer close_file(file)  // Executes when scope exits

    // ... work with file ...
    ret result
}

// LIFO execution order
demo => {
    defer println("First")   // Executes third
    defer println("Second")  // Executes second
    defer println("Third")   // Executes first
    println("Body")          // Executes immediately
}
// Output: Body\nThird\nSecond\nFirst

// Resource cleanup pattern
safe_alloc = size => {
    ptr := c.malloc(size)
    defer c.free(ptr)  // Guaranteed cleanup

    // Use ptr safely...
    ptr[0] <- 42 as int32
    value := ptr[0] as int32
    ret value
}
```

**Key Properties:**
- Executes at scope exit (function return, block end, early return)
- LIFO order: Last defer executes first
- Always runs even if errors occur before
- Common use: Resource cleanup (files, memory, locks)

## Functions and Lambdas

### Lambda Syntax

Flap uses a single lambda arrow `=>` for all functions. The block semantics are automatically inferred from the content:

- **Regular function**: Block contains statements without match arrows (`->` or `~>`)
- **Match function**: Block contains match arrows (`->` or `~>`)

```flap
// Single parameter - regular function
square = x => x * x

// Multiple parameters - regular function
add = (a, b) => a + b

// Block body - regular function (no arrows)
complex = (x, y) => {
    temp := x * 2
    result := temp + y
    ret result
}

// Match function - block contains arrows
classify = x => x {
    0 -> "zero"
    ~> x > 0 { -> "positive" ~> "negative" }
}

// Match function with condition
factorial = (n, acc) => n == 0 {
    -> acc
    ~> factorial(n-1, n*acc)
}

// No-argument regular functions (both forms valid)
hello = () => {
    println("hello")
}

greet => {
    println("Hello, world!")
}
```

**Key Rule:**
- A block `{ ... }` is treated as a **match block** if it contains `->` or `~>` arrows
- Otherwise, it's treated as a **statement block** where the last expression is returned
- This makes the distinction natural and eliminates the need for two arrow types

**Parameter Syntax Rules:**

```flap
// 0 parameters - parentheses optional
f = () => { println("hello") }    // Explicit empty parens
g => { println("hello") }          // No parens, inferred no params

// 1 parameter - parentheses optional
square = x => x * x                // No parens (common style)
double = (x) => x * 2              // With parens (also valid)

// 2+ parameters - parentheses REQUIRED
add = (a, b) => a + b              // Must use parens
multiply = (x, y, z) => x * y * z  // Must use parens

// Multiple parameters without parens is an error
bad = a, b => a + b                // ERROR: parens required for 2+ params
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

The @ symbol is only used in connection with loops in the Flap syntax.

### Infinite Loop

```flap
@ {
    update()
    render()
}
```

### Range Loop

```flap
// Exclusive range (0 to 9) - IMPLEMENTED
@ i in 0..<10 {
    println(i)
}

// Inclusive range (0 to 10)
@ i in 0..10 {
    println(i)
}

// List iteration
items := [1, 2, 3, 4, 5]
@ item in items {
    println(item)
}
```

**Range Operators:**
- `..<` (exclusive) - - End value not included: `0..<10` gives 0,1,2,...,9
- `..` (inclusive) - - End value included: `0..9` would give 0,1,2,...,9

### Loop Control with Labels

Flap uses loop labels instead of `break`/`continue` keywords:

```flap
// Simple loop - exit current loop with ret @
@ i in 0..<100 {
    i > 50 {
        ret @  // Exit current loop (shorthand for ret @1)
    }
    i == 42 {
        ret @ 42  // Exit current loop with value 42
    }
    println(i)
}

// Nested loops - @1 is outer, @2 is inner
@ i in 0..<10 {        // This is loop @1
    @ j in 0..<10 {    // This is loop @2
        j == 5 {
            ret @  // Exit current loop (inner loop, same as ret @2)
        }
        j == 7 {
            ret @ j  // Exit current loop with value j
        }
        i == 5 and j == 3 {
            ret @1  // Exit outer loop (loop 1) specifically
        }
        i == 4 {
            ret @1 42  // Exit outer loop (loop 1) and return the value 42
        }
        println(f"i={i}, j={j}")
    }
}

// ret without @ always returns from function
compute => {
    @ i in 0..<100 {
        i == 50 {
            ret i  // Return from function with value i
        }
        i == 30 {
            ret @  // Exit loop only, continue function
        }
    }
    ret 0  // Return from function
}
```

Loops without a known max value (such as 0..<10 or a list), or if the counter is modified in the loop,
MUST include the `max` keyword and the maximum number of iterations (can be `inf`).

For example:

```
@ i in 0..<10 max 20 {
  i++
}
```

```
@ i in read_from_channel() max inf {
  i++
}
```

**Loop Control:**
- `@N` - Jump to next iteration of loop N (continue)
- `ret` - Return from current function (never exits just a loop)
- `ret value` - Return value from function
- `ret @` - Exit current loop (break)
- `ret @ value` - Exit current loop with a return value
- `ret @N` - Exit loop N specifically (for nested loops)
- `ret @N value` - Exit loop N with a return value

**Important:** `ret` without `@` **always** returns from the function, not from a loop. To exit just a loop, use `ret @` (current loop) or `ret @N` (specific loop).

**Note:** There are no `break` or `continue` keywords. Use `@N` to continue to next iteration of loop N, or `ret @` to exit current loop.

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

### ENet Channels (Concurrency via Message Passing)

Flap uses **ENet** for all inter-process communication and networking. ENet provides fast, reliable UDP-based messaging that unifies IPC and network communication with the same syntax.

**Philosophy**: Everything is a message. Local and remote communication use identical syntax.

#### Address Literals

Addresses use the `:` prefix:

```flap
:8080                    // Port on localhost (IPC - fast)
:localhost:8080          // Explicit localhost
:server.com:8080         // Remote hostname
:192.168.1.100:7777      // IP address + port
```

#### Send Operator: `<=`

Send messages to addresses (write to channel):

```flap
:5000 <= "hello"                    // Send to local port
:server.com:8080 <= "data"          // Send to remote server
:localhost:3000 <= f"value={x}"     // Send with f-string
```

#### Receive Operator: `=>`

Receive messages from addresses (read from channel):

```flap
msg = => :5000           // Blocking receive (waits for one message)
msg = => :8080           // Returns string message
```

#### Receive Loop

Process multiple messages with a loop:

```flap
// Basic receive loop
@ msg, from in :5000 {
    println(f"Got: {msg} from {from}")
}

// Pattern matching on messages
@ msg, from in :8080 {
    msg {
        "ping" -> from <= "pong"
        "quit" -> ret
        ~> from <= "unknown"
    }
}

// Limited iterations
@ msg, from in :9000 max 100 {
    process(msg)
}
```

The receive loop syntax `@ msg, from in address`:
- **msg**: The received message (string)
- **from**: Sender's address as a string (e.g., "127.0.0.1:51234")
- Blocks waiting for messages
- Use `ret` to exit the loop

#### Complete Examples

**Echo Server:**
```flap
@ msg, from in :8080 {
    from <= f"Echo: {msg}"
}
```

**Request-Response:**
```flap
:server.com:5000 <= "get_status"
response = => :5000
println(response)
```

**Concurrent Workers:**
```flap
// Spawn 4 workers listening on :7000
parallel(4, worker => {
    @ task, from in :7000 {
        result := process(task)
        from <= result
    }
})
```

#### ENet Properties

- **Reliable**: Messages delivered in order
- **Fast**: UDP-based, low overhead
- **Unified**: Same syntax for IPC and network
- **Simple**: No setup, no configuration
- **UNIX Philosophy**: Everything is a message stream

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

// Metadata available via dot notation:
// Vec3.size = 12
// Vec3.x.offset = 0
// Vec3.y.offset = 4
// Vec3.z.offset = 8
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
ptr = c.malloc(Vec3.size)

// Write fields using unsafe blocks
unsafe pointer {
    rax <- ptr as pointer
    rbx <- 1.0
    [rax + Vec3.x.offset] <- rbx
    rbx <- 2.0
    [rax + Vec3.y.offset] <- rbx
    rbx <- 3.0
    [rax + Vec3.z.offset] <- rbx
}

// Read fields using unsafe blocks
x := unsafe float64 {
    rax <- ptr as pointer
    rax <- [rax + Vec3.x.offset]
}
y := unsafe float64 {
    rax <- ptr as pointer
    rax <- [rax + Vec3.y.offset]
}
z := unsafe float64 {
    rax <- ptr as pointer
    rax <- [rax + Vec3.z.offset]
}

// Free
c.free(ptr)
```

## Memory Management

### Manual Allocation

```flap
// Allocate
ptr = c.malloc(1024)

// Use memory
ptr[0] <- 42 as int32
value = ptr[0] as int32

// Free
c.free(ptr)
```

### Arena Allocation

Scope-based automatic memory management:

```flap
arena {
    // All allocations freed at block exit
    buffer := alloc(1024)
    entities := alloc(count * size)

    // Use memory...
    buffer[0] <- 100 as int32
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

### Unified Unsafe Syntax (similar to the Battlestar programming language")

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

The CPU order is always the same, first x86_64, then arm64, then riscv64.

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

### Pipe Operators

Flap provides three pipe operators for functional composition and data transformation:

#### Sequential Pipe `|`

Passes a value to the next function (left-to-right function application):

```flap
// Basic piping
x := 10 | double | add_one | square
// Equivalent to: square(add_one(double(10)))

// Piping into lambda
result := 42 | x => x * 2 | x => x + 10
// Equivalent to: (x => x + 10)((x => x * 2)(42))

// Piping with unsafe blocks
value := unsafe float64 {
    rax <- ptr
    rax <- [rax + offset]
} | x => x * 2.0
// Takes result of unsafe block, passes to lambda

// Chaining transformations
data | parse | validate | transform | save
```

#### Parallel Pipe `||`

Maps a function over list elements **in parallel** (like parallel map):

```flap
// Apply function to each element in parallel
results := [1, 2, 3, 4, 5] || x => expensive_computation(x)
// Runs expensive_computation in parallel for each element

// Chaining parallel operations
[1, 2, 3, 4] || x => x * 2 || x => x + 1
// First doubles all in parallel, then adds 1 to all in parallel

// Combining with sequential pipe
data | load_items || process_item | collect_results
```

**Note:** `||` creates parallel threads for each element. Use for CPU-intensive operations where parallelism benefits outweigh thread overhead.

#### Reduce Pipe `|||`

Reduces/folds a list to a single value (summarize operation):

**Status:** Not yet implemented - future feature

```flap
// Sum all elements
total := [1, 2, 3, 4, 5] ||| (acc, x) => acc + x
// Equivalent to fold/reduce: ((((1 + 2) + 3) + 4) + 5)

// Find maximum
max := [3, 7, 2, 9, 1] ||| (acc, x) => acc > x { -> acc ~> x }

// Combine with other pipes
data | load_list || parse_item ||| (acc, x) => acc + x.value
// Load list, parse each in parallel, sum the values

// String concatenation
words := ["Hello", " ", "World"] ||| (acc, s) => acc + s
// Result: "Hello World"
```

**Note:** The reduce pipe `|||` requires a binary function `(accumulator, element) => result`. The first element serves as the initial accumulator value.

### List Operations

**Status:** These operators are specified but not yet fully implemented.

```flap
len := #list             // List length

// Cons operator (::) - pure function, returns new list
list1 := [2, 3]
list2 := 1 :: list1      // Returns [1, 2, 3], list1 unchanged
list3 := 0 :: list2      // Returns [0, 1, 2, 3], list2 unchanged

// Head operator (^) - returns first element
first := ^[1, 2, 3]      // 1.0
second := ^_[1, 2, 3]    // 2.0 (head of tail)

// Tail operator (_) - returns all but first element
rest := _[1, 2, 3]       // [2, 3]
all_but_two := __[1, 2, 3, 4]  // [3, 4] (tail of tail)

// Building lists functionally (like Scheme/ML/LISP)
empty := []
one := 1 :: empty        // [1]
two := 2 :: one          // [2, 1]
three := 3 :: two        // [3, 2, 1]

// Deconstructing lists
process_list := list => {
    #list == 0 {
        ret "empty"
    }
    head := ^list
    tail := _list
    println(f"Head: {head}, Tail: {tail}")
}
```

**Important:** The cons operator `::` follows Standard ML/Scheme/LISP semantics - it is a pure function that constructs and returns a new list without modifying the original list. This maintains immutability.

**Edge Cases:**
```flap
// Empty list operations
^[]    // Returns NaN (error - no head of empty list)
_[]    // Returns [] (tail of empty is empty)
1 :: []  // Returns [1]

// Error checking with or! operator
list := []
head := (^list) or! 0.0  // Returns 0.0 if error
println(head)
```


### Memory Access

```flap
// Write operations (array syntax with type cast)
ptr[offset] <- value as int8
ptr[offset] <- value as int16
ptr[offset] <- value as int32
ptr[offset] <- value as int64
ptr[offset] <- value as uint8
ptr[offset] <- value as uint16
ptr[offset] <- value as uint32
ptr[offset] <- value as uint64
ptr[offset] <- value as float32
ptr[offset] <- value as float64

// Read operations (array syntax with type cast)
value = ptr[offset] as int8
value = ptr[offset] as int16
value = ptr[offset] as int32
value = ptr[offset] as int64
value = ptr[offset] as uint8
value = ptr[offset] as uint16
value = ptr[offset] as uint32
value = ptr[offset] as uint64
value = ptr[offset] as float32
value = ptr[offset] as float64
```

### Memory Allocation

```flap
alloc(size)  // Allocate memory in current arena (only available inside arena blocks)
```

**Important:** `alloc()` is only available inside `arena { }` blocks. When an arena block is entered, a new memory arena is created in the meta arena table, and `alloc()` allocates memory from that arena. All allocations are automatically freed when the arena block exits.

Example:
```flap
arena {
    buffer := alloc(1024)     // Allocate 1KB from arena
    data := alloc(256)        // Allocate 256 bytes from arena
    // Use memory...
}  // All allocations automatically freed here
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
x ** y       // Power
```

### Result Type Operations

```flap
value or! default         // Returns value if success, default if error (unwrap with default)
value.error               // Extracts 4-letter error code as string (e.g., "dv0 ", "nan ")
```

**Result Type Usage:**

Results are float64 values that can represent either success or error. Error Results use NaN encoding with 4-letter error codes.

```flap
// Division by zero returns error Result (NaN)
result := 10 / 0

// Use or! operator to provide default value for errors
safe_value := result or! 0.0
println(safe_value)  // Prints 0.0

// Or extract error code
code := result.error
println(code)  // Prints "dv0 " (division by zero)
```

**Common Error Codes:**
- `"dv0 "` - Division by zero
- `"nan "` - Not a number
- `"sqrt"` - Square root of negative
- `"log "` - Logarithm of non-positive

### System

```flap
exit(code)   // Exit program, uses syscall unless if C functions have been called, then the C exit function is called instead.
```

## Examples

### Factorial

```flap
// Iterative
factorial = n => {
    result := 1
    @ i in 1..n {
        result *= i
    }
    result  // the last thing in a expression block is returned, or `ret` can be used
}

// Tail-recursive
factorial = (n, acc) => n == 0 {
    -> acc
    ~> factorial(n-1, n*acc)
}
```

### Fibonacci

```flap
fib = n => n < 2 {
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
@ running == 1 max inf {
    // Handle input
    event_ptr := alloc(56)
    has_event := sdl.SDL_PollEvent(event_ptr)

    has_event {
        event_type := event_ptr[0] as uint32
        event_type == sdl.SDL_EVENT_QUIT {
            running <- 0
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
particles := c.malloc(particle_count * Particle.size)

// Initialize particles
@ i in 0..<particle_count {
    offset := i * Particle.size
    particles[offset + Particle.x.offset] <- ??? as float64
    particles[offset + Particle.y.offset] <- ??? as float64
    particles[offset + Particle.vx.offset] <- 0.0 as float64
    particles[offset + Particle.vy.offset] <- 0.0 as float64
}

// Update loop
@ frame in 0..<1000 {
    @@ i in 0..<particle_count {      // parallel loop! notice @@
        offset := i * Particle.size

        // Read position and velocity
        x := particles[offset + Particle.x.offset] as float64
        y := particles[offset + Particle.y.offset] as float64
        vx := particles[offset + Particle.vx.offset] as float64
        vy := particles[offset + Particle.vy.offset] as float64

        // Apply gravity
        vy_new := vy + 0.01

        // Update position
        x_new := x + vx
        y_new := y + vy_new

        // Bounce at edges
        y_final := y_new
        vy_final := vy_new
        y_new > 1.0 {
            y_final = 1.0
            vy_final = -vy_new * 0.9
        }

        // Write back
        particles[offset + Particle.x.offset] <- x_new as float64
        particles[offset + Particle.y.offset] <- y_final as float64
        particles[offset + Particle.vx.offset] <- vx as float64
        particles[offset + Particle.vy.offset] <- vy_final as float64
    }
}
```

## Classes and Objects

**Status:** Design proposal - Not yet implemented

**Dependencies:** This proposal requires the following additions to the language:
- `this` keyword - reference to current instance
- `keys(map)` builtin - returns list of map keys
- `join(list, separator)` builtin - joins list elements into string

**Note:** For null/empty values, use `[]` (empty list/map) or Result type with error code.

This section describes a potential object system that fits Flap's design philosophy while maintaining compatibility with the existing unified type system.

### Design Philosophy

The class system builds on Flap's existing features:
- **Maps as objects**: Objects are just the standard Flap type (ordered hash maps from uint64 to float64)
- **Closures as methods**: Methods are lambdas that close over instance data
- **Composition over inheritance**: Use `<>` to extend behavior maps
- **Dot notation for instance fields**: `.field` inside methods, `instance.field` outside
- **No `new` keyword**: Classes are just constructor functions
- **Minimal syntax**: (`<>` and `class` is the only needed syntax in Flap for supporting OOP)

### Class Declaration

Classes are syntactic sugar over maps and closures:

```flap
class Point {
    init = (x, y) => {    // functions named "init" are the constructors, and "deinit" are the deconstructors
        .x = x  // Dot prefix = instance field
        .y = y
    }

    distance = other => {
        dx := other.x - .x
        dy := other.y - .y
        sqrt(dx * dx + dy * dy)
    }

    move = (dx, dy) => {
        .x <- .x + dx
        .y <- .y + dy
    }

    magnitude => {
        sqrt(.x * .x + .y * .y)
    }
}

// Usage
p1 := Point(10, 20)
p2 := Point(30, 40)
dist := p1.distance(p2)
p1.move(5, 5)
mag := p1.magnitude()
```

### Desugaring

The `class` keyword desugars to regular Flap code:

```flap
// Class declaration desugars to:
Point = (x, y) => {
    instance := []   // Empty Flap map/list
    instance["x"] = x
    instance["y"] = y

    instance["distance"] = other => {
        dx := other["x"] - instance["x"]
        dy := other["y"] - instance["y"]
        sqrt(dx * dx + dy * dy)
    }

    instance["move"] = (dx, dy) => {
        instance["x"] <- instance["x"] + dx
        instance["y"] <- instance["y"] + dy
    }

    instance["magnitude"] => {
        sqrt(instance["x"] * instance["x"] + instance["y"] * instance["y"])
    }

    ret instance
}
```

### Instance Variables

Instance variables use dot prefix inside class methods:

```flap
class Counter {
    init = start => {
        .count = start
        .history = []
    }

    increment => {
        .count <- .count + 1
        .history <- .history :: .count
    }

    get => .count

    reset => {
        .count <- 0
        .history <- []
    }
}

cnt := Counter(0)
cnt.increment()
cnt.increment()
println(cnt.get())  // 2
```

### Class Variables

Class-level state accessed via class name:

```flap
class Entity {
    Entity.count = 0
    Entity.all = []

    init = name => {
        .name = name
        .id = Entity.count
        Entity.count += 1
        Entity.all <- Entity.all :: this    // Add the current object to the class list "all" by consing it with "::"
    }
}

e1 := Entity("Alice")
e2 := Entity("Bob")
println(Entity.count)  // 2
println(Entity.all)    // [e1, e2]
```

### Composition with `<>`

Extend classes with behavior maps using `<>`:

```flap
// Reusable behaviors as plain maps
Printable = {
    to_s = () => {
        fields := []
        @ key in keys(this) {
            key[0] != '_' {  // Skip private fields
                fields <- fields :: f"{key}={this[key]}"
            }
        }
        join(fields, ", ")
    }
}

Comparable = {
    eq = other => .x == other.x and .y == other.y,
    lt = other => .x < other.x or (.x == other.x and .y < other.y)
}

class Point {
    <> Printable
    <> Comparable

    init = (x, y) => {
        .x = x
        .y = y
    }

    move = (dx, dy) => {
        .x <- .x + dx
        .y <- .y + dy
    }
}

p := Point(10, 20)
println(p.to_s())  // "x=10, y=20"
println(p.eq(Point(10, 20)))  // true
```

### Method Visibility

Use naming conventions (underscore for private):

```flap
class BankAccount {
    init = balance => {
        .balance = balance
    }

    deposit = amount => {
        amount > 0 {
            .balance <- .balance + amount
        }
    }

    _validate = amount => {
        amount > 0 and amount <= .balance
    }

    withdraw = amount => {
        _validate(amount) {
            .balance <- .balance - amount
            ret amount
        }
        ret 0
    }

    balance => .balance
}
```

### Integration with CStruct

Classes can wrap CStruct for performance:

```flap
cstruct ParticleData {
    x as float64
    y as float64
    vx as float64
    vy as float64
}

class Particle {
    init = (x, y) => {
        .data = call("malloc", ParticleData.size as uint64)

        unsafe pointer {
            rax <- .data as pointer
            rbx <- x
            [rax + ParticleData.x.offset] <- rbx
            rbx <- y
            [rax + ParticleData.y.offset] <- rbx
        } {
            x0 <- .data as pointer
            x1 <- x
            [x0 + ParticleData.x.offset] <- x1
            x1 <- y
            [x0 + ParticleData.y.offset] <- x1
        } {
            a0 <- .data as pointer
            a1 <- x
            [a0 + ParticleData.x.offset] <- a1
            a1 <- y
            [a0 + ParticleData.y.offset] <- a1
        }

        .vx = 0.0
        .vy = 0.0
    }

    update = dt => {
        // Read x using unsafe block, pipe to lambda for computation
        unsafe float64 {
            rax <- .data as pointer
            rax <- [rax + ParticleData.x.offset]
        } {
            x0 <- .data as pointer
            x0 <- [x0 + ParticleData.x.offset]
        } {
            a0 <- .data as pointer
            a0 <- [a0 + ParticleData.x.offset]
        } | x => {
            .vx <- .vx + 0.01 * dt
            new_x := x + .vx * dt

            // Write back new x value
            unsafe pointer {
                rax <- .data as pointer
                rbx <- new_x
                [rax + ParticleData.x.offset] <- rbx
            } {
                x0 <- .data as pointer
                x1 <- new_x
                [x0 + ParticleData.x.offset] <- x1
            } {
                a0 <- .data as pointer
                a1 <- new_x
                [a0 + ParticleData.x.offset] <- a1
            }
        }
    }

    destroy => {
        call("free", .data as ptr)
    }
}
```

### Complete Example

```flap
Serializable = {
    to_json = () => {
        parts := []
        @ key in keys(this) {
            key[0] != '_' {
                parts <- parts :: f"\"{key}\": {this[key]}"
            }
        }
        "{" + join(parts, ", ") + "}"
    }
}

Validatable = {
    valid = () => {
        .x >= 0 and .y >= 0
    }
}

class Point {
    <> Serializable
    <> Validatable

    Point.origin = []

    init = (x, y) => {
        .x = x
        .y = y
    }

    move = (dx, dy) => {
        .x <- .x + dx
        .y <- .y + dy
    }

    distance_to = other => {
        dx := other.x - .x
        dy := other.y - .y
        sqrt(dx * dx + dy * dy)
    }
}

Point.origin = Point(0, 0)

p := Point(10, 20)
p.valid() {
    println(p.to_json())  // {"x": 10, "y": 20}
    println(p.distance_to(Point.origin))  // 22.36...
}
```

### Grammar Extension

The class system extends Flap's grammar:

```ebnf
statement = ...
          | class_decl ;

class_decl = "class" identifier "{" { class_member } "}" ;

class_member = class_var_decl
             | extend_decl
             | method_decl ;

class_var_decl = identifier "." identifier "=" expression ;

extend_decl = "<>" identifier ;

method_decl = identifier "=" lambda_expr ;

lambda_expr = [ parameter_list ] "=>" lambda_body ;

primary_expr = ...
             | "." identifier ;  // Instance field access (inside class)
```

## Special Notes

### Tail Call Optimization

Flap automatically performs tail-call optimization, whenever possible, if there are function calls at the end of a function block.

```flap
loop = n => {
    println(n)
    loop(n + 1)  // Tail call - no stack growth
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
// Single line comment
```

There are no multiline comments.

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
x := 42               // OK - variable
result := x as int32  // OK - type cast to the C type int32, result can then be used when calling C functions without casting
```

---

## Error Handling and Diagnostics

Flap has excellent error messages, like this:

```
error: cannot update immutable variable 'x'
  --> example.flap:8:5
   |
 8 |     x <- x + 1
   |     ^
   |
help: declare 'x' as mutable with ':='
```

## Result Type Error Handling

The error handling system in Flap revolves around the Result type.

### Result Type Representation

The Result type uses float64 values that can represent either success or error:
- **Success values**: Normal float64 values or valid pointers
- **Error values**: Invalid pointer values (like 0xFFFFFFFF) that encode 4-letter error codes using available bits

For example: `"dv0 "` (division by zero), `"nan "` (not a number), `"eof "` (end of file)

### Error Handling Operations

Use the Result type operations to handle errors (see Built-in Functions → Result Type Operations):

```flap
// Division by zero example
result := 10 / 0

// Use or! operator to provide default for errors
safe_value := result or! 0.0
println(safe_value)  // Prints 0.0

// Or extract and check error code
error_code := result.error
println(error_code)  // Prints "dv0 " for division by zero
```

### Design Philosophy

There is **no catch/throw or exception system** in Flap. Error handling is explicit:
- Functions return Result values (NaN-encoded errors)
- Caller handles errors using `or!` operator or `.error` property
- Errors propagate through return values, not exceptions

Many errors are caught at **compile time** when possible (type errors, undefined variables, etc.). Runtime errors use the Result type (NaN encoding) for explicit handling.

