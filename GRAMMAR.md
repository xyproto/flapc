# Flap Grammar Specification

**Version:** 3.0.0
**Date:** 2025-11-17
**Status:** Canonical Grammar Reference for Flap 3.0 Release

This document defines the complete formal grammar of the Flap programming language using Extended Backus-Naur Form (EBNF).

## ⚠️ CRITICAL: The Universal Type

Flap has exactly ONE type: `map[uint64]float64`, an ordered map.

Not "represented as" or "backed by" — every value IS this map:

```flap
42              // {0: 42.0}
"Hello"         // {0: 72.0, 1: 101.0, 2: 108.0, 3: 108.0, 4: 111.0}
[1, 2, 3]       // {0: 1.0, 1: 2.0, 2: 3.0}
{x: 10}         // {hash("x"): 10.0}
[]              // {}
{}              // {}
```

There are NO special types, NO primitives, NO exceptions.
Everything is a map from uint64 to float64.

This is not an implementation detail — this IS Flap.

## Table of Contents

- [Grammar Notation](#grammar-notation)
- [Block Disambiguation Rules](#block-disambiguation-rules)
- [Complete Grammar](#complete-grammar)
- [Lexical Elements](#lexical-elements)
- [Keywords](#keywords)
- [Operators](#operators)
- [Operator Precedence](#operator-precedence)

## Grammar Notation

The grammar uses Extended Backus-Naur Form (EBNF):

| Notation | Meaning |
|----------|---------|
| `=` | Definition |
| `;` | Termination |
| `\|` | Alternation |
| `[ ... ]` | Optional (zero or one) |
| `{ ... }` | Repetition (zero or more) |
| `( ... )` | Grouping |
| `"..."` | Terminal string |
| `letter`, `digit` | Character classes |

## Block Disambiguation Rules

When the parser encounters `{`, it determines the block type by examining contents:

### Rule 1: Map Literal
**Condition:** First element contains `:` (before any `=>` or `~>`)

```flap
config = { port: 8080, host: "localhost" }
settings = { "key": value, "other": 42 }
```

### Rule 2: Match Block
**Condition:** Contains `=>` or `~>` in the block's scope

There are TWO forms:

#### Form A: Value Match (with expression before `{`)
Evaluates expression, then matches its result against patterns:

```flap
// Match on literal values
x {
    0 => "zero"
    5 => "five"
    ~> "other"
}

// Boolean match
x > 0 {
    1 => "positive"    // true = 1
    0 => "zero"        // false = 0
}
```

#### Form B: Guard Match (no expression, uses `|` at line start)
Each branch evaluates its own condition independently:

```flap
// Guard branches with | at line start
{
    | x == 0 => "zero"
    | x > 0 => "positive"
    | x < 0 => "negative"
    ~> "unknown"  // optional default
}
```

**Important:** The `|` is only a guard marker when at the start of a line/clause.
Otherwise `|` is the pipe operator: `data | transform | filter`

### Rule 3: Statement Block
**Condition:** No `=>` or `~>` in scope, not a map

```flap
compute = x -> {
    temp = x * 2
    result = temp + 10
    result    // Last expression returned
}
```

**Disambiguation order:**
1. Check for `:` → Map literal
2. Check for `=>` or `~>` → Match block
3. Otherwise → Statement block

**Match block type:**
- Has expression before `{` → Value match
- No expression, has `|` at line start → Guard match

## Complete Grammar

```ebnf
program         = { statement { newline } } ;

statement       = assignment
                | expression_statement
                | loop_statement
                | unsafe_statement
                | arena_statement
                | parallel_statement
                | cstruct_decl
                | class_decl
                | return_statement
                | defer_statement ;

return_statement = "ret" [ "@" [ integer ] ] [ expression ] ;

defer_statement  = "defer" expression ;

cstruct_decl    = "cstruct" identifier "{" { field_decl } "}" ;

field_decl      = identifier "as" c_type [ "," ] ;

class_decl      = "class" identifier [ extend_clause ] "{" { class_member } "}" ;

extend_clause   = { "<>" identifier } ;

class_member    = class_field_decl
                | method_decl ;

class_field_decl = identifier "." identifier "=" expression ;

method_decl     = identifier "=" lambda_expr ;

c_type          = "int8" | "int16" | "int32" | "int64"
                | "uint8" | "uint16" | "uint32" | "uint64"
                | "float32" | "float64"
                | "ptr" | "cstr" ;

arena_statement = "arena" block ;

loop_statement  = "@" block
                | "@" identifier "in" expression [ "max" expression ] block
                | "@" expression [ "max" expression ] block ;

parallel_statement = "||" identifier "in" expression block ;

unsafe_statement = "unsafe" type_cast block [ block ] [ block ] ;

type_cast       = "int8" | "int16" | "int32" | "int64"
                | "uint8" | "uint16" | "uint32" | "uint64"
                | "float32" | "float64"
                | "number" | "string" | "list" | "address"
                | "packed" | "aligned" ;

assignment      = identifier ("=" | ":=" | "<-") expression
                | identifier ("+=" | "-=" | "*=" | "/=" | "%=" | "**=") expression
                | indexed_expr "<-" expression
                | identifier_list ("=" | ":=" | "<-") expression ;  // Multiple assignment

identifier_list = identifier { "," identifier } ;

indexed_expr    = identifier "[" expression "]" ;

expression_statement = expression [ match_block ] ;

match_block     = "{" ( default_arm
                      | match_clause { match_clause } [ default_arm ]
                      | guard_clause { guard_clause } [ default_arm ] ) "}" ;

match_clause    = expression [ "=>" match_target ] ;

guard_clause    = "|" expression "=>" match_target ;  // | must be at start of line

default_arm     = ( "~>" | "_" "=>" ) match_target ;

match_target    = jump_target | expression ;

jump_target     = integer ;

block           = "{" { statement { newline } } [ expression ] "}" ;

expression      = pipe_expr ;

pipe_expr       = reduce_expr { ( "|" | "||" ) reduce_expr } ;

reduce_expr     = receive_expr ;

receive_expr    = "<=" pipe_expr | or_bang_expr ;

or_bang_expr    = send_expr { "or!" send_expr } ;

send_expr       = or_expr { "<-" or_expr } ;

or_expr         = and_expr { "or" and_expr } ;

xor_expr        = and_expr { "xor" and_expr } ;

and_expr        = comparison_expr { "and" comparison_expr } ;

comparison_expr = bitwise_or_expr { comparison_op bitwise_or_expr } ;

comparison_op   = "==" | "!=" | "<" | "<=" | ">" | ">=" ;

bitwise_or_expr = bitwise_xor_expr { "|b" bitwise_xor_expr } ;

bitwise_xor_expr = bitwise_and_expr { "^b" bitwise_and_expr } ;

bitwise_and_expr = shift_expr { "&b" shift_expr } ;

shift_expr      = additive_expr { shift_op additive_expr } ;

shift_op        = "<<b" | ">>b" | "<<<b" | ">>>b" ;

additive_expr   = multiplicative_expr { ("+" | "-") multiplicative_expr } ;

multiplicative_expr = power_expr { ("*" | "/" | "%") power_expr } ;

power_expr      = unary_expr { ( "**" | "^" ) unary_expr } ;

unary_expr      = ( "-" | "!" | "~b" | "#" ) unary_expr
                | postfix_expr ;

postfix_expr    = primary_expr { postfix_op } ;

postfix_op      = "[" expression "]"
                | "." ( identifier | integer )
                | "(" [ argument_list ] ")"
                | "!"
                | "#"
                | match_block ;

primary_expr    = identifier
                | number
                | string
                | fstring
                | list_literal
                | map_literal
                | lambda_expr
                | enet_address
                | address_value
                | instance_field
                | this_expr
                | "(" expression ")"
                | "??"
                | unsafe_expr
                | arena_expr
                | "???" ;

instance_field  = "." identifier ;

this_expr       = "." [ " " | newline ] ;  // Dot followed by space or newline means "this"

enet_address    = "&" port_or_host_port ;

port_or_host_port = port | [ hostname ":" ] port ;

address_value   = "$" expression ;

port            = digit { digit } ;

hostname        = identifier | ip_address ;

ip_address      = digit { digit } "." digit { digit } "." digit { digit } "." digit { digit } ;

arena_expr      = "arena" "{" { statement { newline } } [ expression ] "}" ;

unsafe_expr     = "unsafe" "{" { statement { newline } } [ expression ] "}"
                  [ "{" { statement { newline } } [ expression ] "}" ]
                  [ "{" { statement { newline } } [ expression ] "}" ] ;

lambda_expr     = [ parameter_list ] "->" lambda_body
                | block ;  // Inferred lambda with no parameters in assignment context

parameter_list  = variadic_params
                | identifier { "," identifier }
                | "(" [ param_decl_list ] ")" ;

param_decl_list = param_decl { "," param_decl } ;

param_decl      = identifier [ "..." ] ;

variadic_params = "(" identifier { "," identifier } "," identifier "..." ")" ;

lambda_body     = block | expression [ match_block ] ;

// Lambda Syntax Rules:
//
// Explicit lambda syntax (always works):
//   x -> x * 2                        // One parameter
//   (x, y) -> x + y                   // Multiple parameters (parens required)
//   (x, y, rest...) -> sum(rest)      // Variadic parameters (last param with ...)
//   -> println("hi")                  // No parameters (explicit ->)
//   x -> { temp = x * 2; temp }       // Block body
//
// Inferred lambda syntax (works ONLY in assignment context):
//   main = { println("hello") }       // Inferred: main = -> { println("hello") }
//   handler = { | x > 0 => "pos" }    // Inferred: handler = -> { | x > 0 => "pos" }
//
// When `->` can be omitted:
//   1. In assignment context: `name = { ... }` or `name := { ... }`
//   2. Right side is a block (not a map literal - map has `:` colons)
//   3. Block contains statements or guard match (| at line start)
//
// When `->` is REQUIRED:
//   1. Lambda has one or more parameters: `x -> x * 2`
//   2. Lambda body is an expression (not a block): `-> 42`
//   3. Lambda is NOT being assigned: `[1, 2, 3] | x -> x * 2`
//   4. Lambda is a function argument: `map(data, x -> x * 2)`
//
// Parentheses rules:
//   - Single parameter: `x -> x * 2` (no parens needed)
//   - Multiple parameters: `(x, y) -> x + y` (parens required)
//   - No parameters with explicit ->: `-> println("hi")` (no parens needed)
//   - No parameters inferred from block: `main = { ... }` (no parens needed)
//
// Block type determination:
//   { x: 10 }                         // Map literal (has `:` before any `=>`)
//   { | x > 0 => "pos" }              // Guard match block (has `|` at line start)
//   { temp = x * 2; temp }            // Statement block (no `:`, no `=>` or `~>`)
//   x { 0 => "zero" ~> "other" }      // Value match (expression before `{`)
//
// Examples:
//   // Function definitions (inferred lambda)
//   main = { println("Hello!") }
//   process = { | x > 0 => "pos" | x < 0 => "neg" }
//
//   // Lambdas with parameters (explicit)
//   square = x -> x * x
//   add = (x, y) -> x + y
//   map_fn = f -> data | f
//
//   // Method definitions in classes (always use `=`)
//   class Point {
//       distance = other -> sqrt((other.x - .x) ** 2 + (other.y - .y) ** 2)
//   }

argument_list   = expression { "," expression } ;

list_literal    = "[" [ expression { "," expression } ] "]" ;

map_literal     = "{" [ map_entry { "," map_entry } ] "}" ;

map_entry       = ( identifier | string ) ":" expression ;

identifier      = letter { letter | digit | "_" } ;

number          = [ "-" ] digit { digit } [ "." digit { digit } ] ;

string          = '"' { character } '"' ;

fstring         = 'f"' { character | "{" expression "}" } '"' ;
```

## Lexical Elements

### Identifiers

Identifiers start with a letter and contain letters, digits, or underscores:

```ebnf
identifier = letter { letter | digit | "_" } ;
letter     = "a" | "b" | ... | "z" | "A" | "B" | ... | "Z" ;
digit      = "0" | "1" | ... | "9" ;
```

**Rules:**
- Case-sensitive
- Can start with letter only (not digit or underscore)
- No length limit
- Can include Unicode letters

**Valid examples:**
```flap
x, count, user_name, myVar, value2, Temperature, λ
```

**Invalid:**
```flap
2count     // starts with digit
_private   // starts with underscore
my-var     // contains hyphen
```

### Numbers

Numbers are `map[uint64]float64` with a single entry at key 0:

```ebnf
number = [ "-" ] digit { digit } [ "." digit { digit } ] ;
```

**Examples:**
```flap
42              // {0: 42.0}
3.14159         // {0: 3.14159}
-17             // {0: -17.0}
0.001           // {0: 0.001}
1000000         // {0: 1000000.0}
-273.15         // {0: -273.15}
```

**Special values:**
- `??` - cryptographically secure random number [0, 1) → `{0: random_value}`
- Result of `0/0` - NaN (used for error encoding) → `{0: NaN}`

**Note:** While the values stored happen to be IEEE 754 doubles, this is an implementation detail. Numbers ARE maps, not primitives.

### Strings

Strings are `map[uint64]float64` where keys are indices and values are character codes:

```ebnf
string = '"' { character } '"' ;
```

**Examples:**
```flap
"Hello"         // {0: 72.0, 1: 101.0, 2: 108.0, 3: 108.0, 4: 111.0}
"A"             // {0: 65.0}
""              // {} (empty map)
```

**Escape sequences:**
- `\n` - newline (character code 10)
- `\t` - tab (character code 9)
- `\r` - carriage return (character code 13)
- `\\` - backslash
- `\"` - quote
- `\xHH` - hex byte
- `\uHHHH` - Unicode code point

**String operations:**
- `.bytes` - get byte array
- `.runes` - get Unicode code point array
- `+` - concatenation
- `[n]` - access byte at index

### F-Strings (Interpolated Strings)

F-strings allow embedded expressions:

```ebnf
fstring = 'f"' { character | "{" expression "}" } '"' ;
```

**Examples:**
```flap
name = "World"
greeting = f"Hello, {name}!"
result = f"2 + 2 = {2 + 2}"
```

### Comments

```flap
// Single-line comment (C++ style)
```

No multi-line comments.

## Keywords

### Reserved Keywords

```
ret arena unsafe cstruct class as max this defer spawn import
```

**Note:** In Flap 3.0, lambda definitions use `->` (thin arrow) and match arms use `=>` (fat arrow), similar to Rust syntax.

**No-argument lambdas** can be written as `-> expr` or inferred from context in assignments: `name = { ... }`

### Contextual Keywords

These are only keywords in specific contexts (e.g., after `as`):

```
int8 int16 int32 int64 uint8 uint16 uint32 uint64 float32 float64
cstr ptr number string list address packed aligned
```

You can use contextual keywords as variable names:

```flap
int32 = 100      // OK - variable named int32
x = y as int32   // OK - int32 as type cast
```

## Memory Management and Builtins

**CRITICAL DESIGN PRINCIPLE:** Flap keeps builtin functions to an ABSOLUTE MINIMUM.

**Memory allocation:**
- NO `malloc`, `free`, `realloc`, or `calloc` as builtins
- Use arena allocators: `allocate()` within `arena {}` blocks (recommended)
- Or use C FFI: `c.malloc`, `c.free`, `c.realloc`, `c.calloc` (explicit)

```flap
// Recommended: arena allocator
result = arena {
    data = allocate(1024)
    process(data)
}

// Alternative: explicit C FFI
ptr := c.malloc(1024)
defer c.free(ptr)
```

**List operations:**
- Use builtin functions: `head(xs)` for first element, `tail(xs)` for remaining elements
- Use `#` length operator (prefix or postfix)

**Why minimal builtins?**
1. **Simplicity:** Less to learn and remember
2. **Orthogonality:** One concept, one way
3. **Extensibility:** Users can define their own functions
4. **Predictability:** No hidden magic

**What IS builtin:**
- Operators: `#`, arithmetic, logic, bitwise
- Control flow: `@`, match blocks, `ret`
- Core I/O: `print`, `println`, `printf` (and error/exit variants)
- List operations: `head()`, `tail()`
- Keywords: `arena`, `unsafe`, `cstruct`, `class`, `defer`, etc.

**Everything else via:**
1. **Operators** for common operations (`#xs` for length)
2. **Builtin functions** for core operations (`head(xs)`, `tail(xs)`)
3. **C FFI** for system functionality (`c.sin`, `c.malloc`, etc.)
4. **User-defined functions** for application logic

## Operators

### Arithmetic Operators

```
+    Addition
-    Subtraction (binary) or negation (unary)
*    Multiplication
/    Division
%    Modulo
**   Exponentiation
^    Exponentiation (alias for **)
```

### Comparison Operators

```
==   Equal
!=   Not equal
<    Less than
<=   Less than or equal
>    Greater than
>=   Greater than or equal
```

### Logical Operators

```
&&   Logical AND (short-circuit)
||   Logical OR (short-circuit)
!    Logical NOT
```

### Bitwise Operators

All bitwise operators use `b` suffix:

```
&b    Bitwise AND
|b    Bitwise OR
^b    Bitwise XOR
~b    Bitwise NOT (unary)
<<b   Left shift
>>b   Arithmetic right shift
<<<b  Rotate left
>>>b  Rotate right
```

### Assignment Operators

```
=     Immutable assignment (cannot reassign variable or modify value)
:=    Mutable assignment (can reassign variable and modify value)
<-    Update/reassignment (for mutable vars)

+=    Add and assign (for lists: append element)
-=    Subtract and assign
*=    Multiply and assign
/=    Divide and assign
%=    Modulo and assign
**=   Exponentiate and assign
```

**Arrow Operator Summary:**

| Operator | Context | Meaning | Example |
|----------|---------|---------|---------|
| `->` | Lambda definition | Lambda arrow | `x -> x * 2` or `-> println("hi")` |
| `=>` | Match block | Match arm | `x { 0 => "zero" ~> "other" }` |
| `~>` | Match block | Default match arm | `x { 0 => "zero" ~> "other" }` |
| `_ =>` | Match block | Default match arm (alias for ~>) | `x { 0 => "zero" _ => "other" }` |
| `=` | Variable binding | Immutable assignment | `x = 42` (standard for functions) |
| `:=` | Variable binding | Mutable assignment | `x := 42` (can reassign later) |
| `<-` | Update/Send | Update mutable var OR send to ENet | `x <- 99` or `&8080 <- msg` |
| `<=` | Comparison | Less than or equal | `x <= 10` |
| `>=` | Comparison | Greater than or equal | `x >= 10` |

**Important Conventions:**
- **Functions/methods** should use `=` (immutable), not `:=`, since they rarely need reassignment
- **Lambda syntax**: `->` always defines a lambda, `=>` always defines a match arm
- **Update operator** `<-` is for updating existing mutable variables or sending to ENet channels
- **Comparison** operators `<=` and `>=` are for comparisons, not assignment or arrows

**Multiple Assignment (Tuple Unpacking):**

```flap
// Functions can return multiple values as a list
a, b = some_function()  // Unpack first two elements
x, y, z := [1, 2, 3]    // Unpack list literal

// Practical example with pop()
new_list, popped_value = pop(old_list)
```

When a function returns a list, multiple assignment unpacks the elements:
- Right side must evaluate to a list/map
- Left side specifies variable names separated by commas
- Variables are assigned elements at indices 0, 1, 2, etc.
- If list has fewer elements than variables, remaining variables get 0
- If list has more elements, extra elements are ignored

### Collection Operators

```
#     Length operator (prefix or postfix)
```

### Other Operators

```
->    Lambda arrow (can be omitted in assignment context with blocks)
=>    Match arm
~>    Default match arm
|     Pipe operator
||    Parallel map
<-    Update/Send (update mutable var OR send to ENet)
<=    Receive (ENet, prefix) OR less-than-or-equal comparison
!     Move operator (postfix)
.     Field access
[]    Indexing
()    Function call (parentheses optional for zero or one argument in some contexts)
@     Loop
&     ENet address (network endpoints)
$     Address value (memory addresses)
??    Random number
or!   Error/null handler (executes right side if left is error or null pointer)
```

## Operator Precedence

From highest to lowest precedence:

1. **Primary**: `()` `[]` `.` function call, postfix `!`, postfix `#`
2. **Unary**: `-` `!` `~b` `#`
3. **Power**: `**`
4. **Multiplicative**: `*` `/` `%`
5. **Additive**: `+` `-`
6. **Shift**: `<<b` `>>b` `<<<b` `>>>b`
7. **Bitwise AND**: `&b`
8. **Bitwise XOR**: `^b`
9. **Bitwise OR**: `|b`
10. **Comparison**: `==` `!=` `<` `<=` `>` `>=`
11. **Logical AND**: `&&`
12. **Logical OR**: `||`
13. **Or-bang**: `or!`
14. **Send**: `<-`
15. **Receive**: `<=`
16. **Pipe**: `|` `||`
17. **Match**: `{ }` (postfix)
18. **Assignment**: `=` `:=` `<-` `+=` `-=` `*=` `/=` `%=` `**=`

**Associativity:**
- Left-associative: All binary operators except `**` and assignments
- Right-associative: `**`, all assignments
- Non-associative: Comparison operators (can't chain)

## Parsing Rules

### Minimal Parentheses Philosophy

Flap minimizes parenthesis usage. Use parentheses only when:

1. **Precedence override needed:**
   ```flap
   (x + y) * z      // Override precedence
   ```

2. **Complex condition grouping:**
   ```flap
   (x > 0 && y < 10) { ... }  // Group condition
   ```

3. **Multiple lambda parameters:**
   ```flap
   (x, y) -> x + y  // Multiple params
   ```

**Not needed:**
```flap
// Good: no unnecessary parens
x > 0 { => "positive" ~> "negative" }
result = x + y * z
classify = x -> x { 0 => "zero" ~> "other" }

// Bad: unnecessary parens
result = x > 0 { => ("positive") ~> ("negative") }
compute = (x) -> (x * 2)
```

### Statement Termination

Statements are terminated by newlines:

```flap
x = 10
y = 20
z = x + y
```

Multiple statements on one line require explicit semicolons:

```flap
x = 10; y = 20; z = x + y
```

### Whitespace Rules

- **Significant newlines**: End statements
- **Insignificant whitespace**: Spaces, tabs (except in strings)
- **Indentation**: Not significant (unlike Python)

### Edge Cases

#### Pipe vs Guard

The `|` character is context-dependent:

```flap
// Pipe operator (| not at line start)
result = data | transform | filter

// Guard marker (| at line start)
classify = x -> {
    | x > 0 => "positive"
    | x < 0 => "negative"
    ~> "zero"
}
```

**Rule:** `|` at the start of a line/clause (after `{` or newline) is a guard marker. Otherwise it's the pipe operator.

#### Arrow Disambiguation

```flap
=>   Match arm result
~>   Default match arm
->   Lambda or receive
```

Context determines meaning:

```flap
f = x -> x + 1             // Lambda with one arg
msg <- &8080               // Receive from channel
x { 0 => "zero" }          // Match arm
x { ~> "default" }         // Default arm
greet = { println("Hi") }  // No-arg lambda
```

#### No-Argument Lambdas

```flap
// Inferred lambda (in assignment context):
greet = { println("Hello!") }            // Inferred: greet = -> { println("Hello!") }
worker = { @ { process_forever() } }     // Inferred: worker = -> { @ { process_forever() } }

// Explicit no-argument lambda:
greet = -> println("Hello!")             // Explicit ->
handler = -> process_events()            // Explicit ->

// With block body:
worker = {                               // Inferred lambda
    @ { process_forever() }
}

// Common use cases:
init = { setup_resources() }             // Inferred (assignment context)
cleanup = { release_all() }              // Inferred (assignment context)
background = { @ { poll_events() } }     // Inferred (assignment context)

// When explicit -> is needed:
callbacks = [-> print("A"), -> print("B")]  // Not in assignment, need explicit ->
process(-> get_data())                      // Function argument, need explicit ->
```

#### Loop Forms

The `@` symbol introduces loops (one of three forms):

```flap
@ { ... }                  // Infinite loop
@ i in collection { ... }  // For-each loop
@ condition { ... }        // While loop
```

**Loop Control with `ret @` and Numbered Labels:**

Instead of `break`/`continue` keywords, Flap uses `ret @` with automatically numbered loop labels.

**Loop Numbering:** Loops are numbered from outermost to innermost:
- `@1` = outermost loop
- `@2` = second level (nested inside @1)
- `@3` = third level (nested inside @2)
- `@` = current/innermost loop

```flap
// Exit current loop
@ i in 0..<100 {
    i > 50 { ret @ }      // Exit current loop (same as ret @1 here)
    i == 42 { ret @ 42 }  // Exit loop with value 42
    println(i)
}

// Nested loops with numbered labels
@ i in 0..<10 {           // Loop @1 (outermost)
    @ j in 0..<10 {       // Loop @2 (inner)
        j == 5 { ret @ }         // Exit loop @2 (innermost)
        i == 5 { ret @1 }        // Exit loop @1 (outer)
        i == 3 and j == 7 { ret @1 42 }  // Exit loop @1 with value
        println(i, j)
    }
}

// ret without @ returns from function (not loop)
compute = n -> {
    @ i in 0..<100 {
        i == n { ret i }  // Return from function
        i == 50 { ret @ } // Exit loop only, continue function
    }
    ret 0
}
```

**Loop `max` Keyword:**

Loops with unknown bounds or modified counters require `max`:

```flap
// Counter modified - needs max
@ i in 0..<10 max 20 {
    i++  // Modified counter
}

// Unknown iterations - needs max
@ msg in read_channel() max inf {
    process(msg)
}
```

#### Defer Statement

The `defer` keyword schedules an expression to execute when the current scope exits (function return, block exit, or error). Deferred expressions execute in **LIFO (Last In, First Out)** order.

**Syntax:**
```ebnf
defer_statement = "defer" expression ;
```

**Examples:**
```flap
// Resource cleanup with defer
init_resources = () -> {
    file := open("data.txt") or! {
        println("Failed to open file")
        ret 0
    }
    defer close(file)  // Always closes when function returns

    buffer := c_malloc(1024) or! {
        println("Out of memory")
        ret 0
    }
    defer c_free(buffer)  // Frees before file closes (LIFO)

    process(file, buffer)
    ret 1
}

// C FFI with defer (SDL3 example)
sdl.SDL_Init(sdl.SDL_INIT_VIDEO) or! {
    println("SDL init failed")
    ret 1
}
defer sdl.SDL_Quit()  // Always called on return

window := sdl.SDL_CreateWindow("Title", 640, 480, 0) or! {
    println("Window creation failed")
    ret 1  // SDL_Quit still called via defer
}
defer sdl.SDL_DestroyWindow(window)  // Executes before SDL_Quit

// More resources...
```

**Execution Order:**
Deferred calls execute in reverse order of declaration (LIFO):
```flap
defer println("1")  // Executes third
defer println("2")  // Executes second
defer println("3")  // Executes first
// Output: 3, 2, 1
```

**When Defer Executes:**
- On function return (`ret`)
- On block exit (normal completion)
- On early return from error handling
- On loop exit with `ret @`

**Best Practices:**
1. Use `defer` immediately after resource acquisition
2. Combine with `or!` for railway-oriented error handling
3. Rely on LIFO order for proper cleanup sequence
4. Use `defer` for C FFI resources (files, sockets, SDL objects)
5. Return from error blocks instead of `exit()` - defer ensures cleanup

**Common Pattern:**
```flap
// Railway-oriented with defer
resource := acquire() or! {
    println("Acquisition failed")
    ret error("acq")
}
defer cleanup(resource)

// Work with resource...
// cleanup always happens, even on error
```

#### Address Operator

The `&` symbol creates ENet addresses (network endpoints):

```flap
&8080                      // Port only: & followed by digits
&localhost:8080            // Host:port: & followed by identifier/IP + :
&192.168.1.1:3000          // IP:port
```

**Examples:**
```flap
// Loops (statement context)
@ { println("Forever") }           // Infinite loop
@ i in [1, 2, 3] { println(i) }    // For-each loop
@ x < 10 { x = x + 1 }             // While loop

// Addresses (expression context)
server = @8080                      // Address literal
client = &localhost:9000            // Address with hostname
remote = &192.168.1.100:3000        // Address with IP

// Unambiguous in context
listen(&8080)                       // Function call with address
@ x > 0 { send(&8080, data) }      // Loop with address inside
```

#### Block vs Map vs Match

Disambiguated by contents (see Block Disambiguation Rules above):

```flap
{ x: 10 }                // Map: contains :
x { 0 -> "zero" }        // Match: contains ->
{ temp = x * 2; temp }   // Statement block: no : or ->
```

## Error Handling and Result Types

Flap uses a **Result type** for operations that can fail. A Result is still `map[uint64]float64`, but with special semantic meaning tracked by the compiler.

### Result Type Design

A Result is encoded as follows:

**Byte Layout:**
```
[type_byte][length][key][value][key][value]...[0x00]
```

**Type Bytes:**
```
0x01 - Flap Number (success)
0x02 - Flap String (success)
0x03 - Flap List (success)
0x04 - Flap Map (success)
0x05 - Flap Address (success)
0xE0 - Error (failure, followed by 4-char error code)
0x10 - C int8
0x11 - C int16
0x12 - C int32
0x13 - C int64
0x14 - C uint8
0x15 - C uint16
0x16 - C uint32
0x17 - C uint64
0x18 - C float32
0x19 - C float64
0x1A - C pointer
0x1B - C string pointer
```

**Success case:**
- Type byte indicates the Flap or C type
- Length field (uint64) indicates number of key-value pairs
- Key-value pairs follow (each pair is uint64 key, float64 value)
- Terminated with 0x00 byte

**Error case:**
- Type byte is 0xE0
- Followed by 4-byte error code (ASCII, space-padded)
- Terminated with 0x00 byte

### Standard Error Codes

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
"ovf " - Overflow
"udf " - Undefined
```

**Note:** Error codes are 4 bytes, space-padded if shorter. The `.error` accessor strips trailing spaces on access.

### The `.error` Accessor

Every value has a `.error` accessor that:
- Returns `""` (empty string) for success values
- Returns the error code string (spaces stripped) for error values

```flap
x = 10 / 2              // Success: returns 5.0
x.error                 // Returns "" (empty)

y = 10 / 0              // Error: division by zero
y.error                 // Returns "dv0" (spaces stripped)

// Typical usage
result.error {
    "" => proceed(result)
    ~> handle_error(result.error)
}
```

### The `or!` Operator

The `or!` operator provides a default value or executes a block when the left side is an error or null:

```flap
// Handle errors
x = 10 / 0              // Error result
safe = x or! 99         // Returns 99 (error case)

y = 10 / 2              // Success result (value 5)
safe2 = y or! 99        // Returns 5 (success case)

// Handle null pointers from C FFI
window := sdl.SDL_CreateWindow("Title", 640, 480, 0) or! {
    println("Failed to create window!")
    sdl.SDL_Quit()
    exit(1)
}

// Inline null check with default
ptr := c_malloc(1024) or! 0  // Returns 0 if allocation failed
```

**Semantics:**
1. Evaluate left operand
2. Check if NaN (error value) OR if value equals 0.0 (null pointer)
3. If NaN or null:
   - And right side is a block: execute block, result in xmm0
   - And right side is an expression: evaluate right side, result in xmm0
4. Otherwise (value is valid): keep left operand value in xmm0
5. Right side is NOT evaluated unless left is NaN/null (lazy/short-circuit evaluation)

**Error Checking:**
- **NaN check**: Compares value with itself using UCOMISD (NaN != NaN)
- **Null check**: Compares value with 0.0 using UCOMISD

**When checking for null (C FFI pointers):**
- All values are float64, so pointer 0 is encoded as 0.0
- `or!` treats 0.0 (null pointer) as a failure case
- Enables railway-oriented programming for C interop
- Works with any C function that returns pointers

**Precedence:** Lower than logical OR, higher than send operator

### Error Propagation Patterns

```flap
// Check and early return
process = input -> {
    step1 = validate(input)
    step1.error { != "" => step1 }  // Return error

    step2 = transform(step1)
    step2.error { != "" => step2 }

    finalize(step2)
}

// Default values with or!
compute = input -> {
    x = parse(input) or! 0
    y = divide(100, x) or! -1
    y * 2
}

// Match on error code
result = risky()
result.error {
    "" => println("Success:", result)
    "dv0" => println("Division by zero")
    "mem" => println("Out of memory")
    ~> println("Unknown error:", result.error)
}
```

### Creating Custom Errors

Use the `error` function to create error Results:

```flap
// Create error with code
err = error("arg")  // Type byte 0xE0 + "arg "

// Or use division by zero for runtime errors
fail = 0 / 0        // Returns error "dv0"
```

### Compiler Type Tracking

The compiler tracks whether a value is a Result type:

```flap
// Compiler knows this returns Result
divide = (a, b) -> {
    b == 0 { ret error("dv0") }
    a / b
}

// Compiler propagates Result type
compute = x -> {
    y = divide(100, x)  // y has Result type
    y or! 0             // Handles potential error
}
```

See [TYPE_TRACKING.md](TYPE_TRACKING.md) for implementation details.

### Result Type Memory Layout

**Success value (number 42):**
```
Bytes: 01 01 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 40 45 00 00 00 00 00 00 00 00
       ↑  ↑----- length=1 ----↑  ↑------- key=0 -------↑  ↑------- value=42.0 ------↑  ↑ term
       type=01 (number)
```

**Error value (division by zero):**
```
Bytes: E0 64 76 30 20 00
       ↑  ↑----- error code "dv0 " -----↑  ↑ term
       type=E0 (error)
```

### `.error` Implementation

The `.error` accessor:
1. Checks type byte (first byte)
2. If 0xE0: extract next 4 bytes as error code string
3. Strip trailing spaces
4. Return error code string
5. Otherwise: return empty string ""

### `or!` Implementation

The `or!` operator:
1. Evaluates left operand
2. Checks type byte
3. If 0xE0: returns right operand
4. Otherwise: returns left operand value (strips type metadata)

## Classes and Object-Oriented Programming

Flap supports classes as syntactic sugar over maps and closures, providing a familiar OOP interface while maintaining the language's fundamental simplicity.

### Core Principles

- **Maps as objects:** Objects are `map[uint64]float64` with conventions
- **Closures as methods:** Methods are lambdas that close over instance data
- **Composition over inheritance:** Use `<>` to compose with behavior maps
- **Dot notation:** `.field` inside methods for instance fields
- **Minimal syntax:** Only one new keyword (`class`)
- **Desugars to regular Flap:** Classes compile to maps and lambdas
- **`this` keyword:** Reference to current instance

### Class Declaration

```flap
class Point {
    // Constructor (implicit)
    init = (x, y) -> {
        .x = x
        .y = y
    }

    // Instance methods
    distance = other -> {
        dx := other.x - .x
        dy := other.y - .y
        sqrt(dx * dx + dy * dy)
    }

    move = (dx, dy) -> {
        .x <- .x + dx
        .y <- .y + dy
    }
}

// Usage
p1 := Point(10, 20)
p2 := Point(30, 40)
dist := p1.distance(p2)
p1.move(5, 5)
```

### Desugaring

Classes desugar to regular Flap code:

```flap
// class Point { ... } becomes:
Point := (x, y) -> {
    instance := {}
    instance["x"] = x
    instance["y"] = y

    instance["distance"] = other -> {
        dx := other["x"] - instance["x"]
        dy := other["y"] - instance["y"]
        sqrt(dx * dx + dy * dy)
    }

    instance["move"] = (dx, dy) -> {
        instance["x"] <- instance["x"] + dx
        instance["y"] <- instance["y"] + dy
    }

    ret instance
}
```

### Instance Fields

Use `.field` inside class methods to access instance state:

```flap
class Counter {
    init = start -> {
        .count = start
        .history = []
    }

    increment = () -> {
        .count <- .count + 1
        .history <- .history :: .count
    }

    get = () -> .count
}

c := Counter(0)
c.increment()
println(c.get())  // 1
```

### Class Fields (Static Members)

Use `ClassName.field` for class-level state:

```flap
class Entity {
    Entity.count = 0
    Entity.all = []

    init = name -> {
        .name = name
        .id = Entity.count
        Entity.count <- Entity.count + 1
        Entity.all <- Entity.all :: instance
    }
}

e1 := Entity("Alice")
e2 := Entity("Bob")
println(Entity.count)  // 2
```

### Composition with `<>`

Extend classes with behavior maps using `<>`:

```flap
Serializable = {
    to_json: {
        // Convert instance to JSON
    },
    from_json: json {
        // Parse JSON to instance
    }
}

class Point <> Serializable {
    init = (x, y) -> {
        .x = x
        .y = y
    }
}

p := Point(10, 20)
json := p.to_json()
```

**Multiple composition** - chain `<>` operators:

```flap
class User {
    <> Serializable
    <> Validatable
    <> Timestamped
    init = name -> {
        .name = name
        .created_at = now()
    }
}
```

### Instance Field Resolution

Inside class methods:
- `.field` → instance field access
- `ClassName.field` → class field access
- `other.field` → other instance field access

```flap
class Point {
    Point.origin = nil  // Class field

    init = (x, y) -> {
        .x = x           // Instance field (this instance)
        .y = y
    }

    distance_to_origin = -> {
        .distance(Point.origin)  // Class field access
    }

    distance = other -> {
        dx := other.x - .x       // Other instance field vs this instance field
        dy := other.y - .y
        sqrt(dx * dx + dy * dy)
    }
}

Point.origin = Point(0, 0)  // Initialize class field
```

### Private Methods Convention

Use underscore prefix for "private" methods (by convention):

```flap
class Account {
    init = balance -> {
        .balance = balance
    }

    _validate = amount -> {
        amount > 0 && amount <- .balance
    }

    withdraw = amount -> {
        ._ validate(amount) {
            .balance -= - amount
            ret 0
        }
        ret -1  // Error
    }
}
```

### Integration with CStruct

Combine classes with CStruct for performance:

```flap
cstruct Vec2Data {
    x as float64,
    y as float64
}

class Vec2 {
    init = (x, y) -> {
        .data = call("malloc", Vec2Data.size as uint64)
        unsafe float64 {
            rax <- .data as ptr
            [rax] <- x
            [rax + 8] <- y
        }
    }

    magnitude = () -> {
        unsafe float64 {
            rax <- .data as ptr
            xmm0 <- [rax]
            xmm1 <- [rax + 8]
            xmm0 <- xmm0 * xmm0
            xmm1 <- xmm1 * xmm1
            xmm0 <- xmm0 + xmm1
        } | result -> sqrt(result)
    }
}
```

### Operator Overloading via Methods

While Flap doesn't have operator overloading syntax, you can define methods with operator-like names:

```flap
class Complex {
    init = (real, imag) -> {
        .real = real
        .imag = imag
    }

    add = other -> Complex(.real + other.real, .imag + other.imag)
    mul = other -> Complex(
        .real * other.real - .imag * other.imag,
        .real * other.imag + .imag * other.real
    )
}

a := Complex(1, 2)
b := Complex(3, 4)
c := a.add(b)
```

### The `<>` Operator

The `<>` operator merges behavior maps into the class:

```ebnf
class_decl      = "class" identifier { "<>" identifier } "{" { class_member } "}" ;
```

Semantically:

```flap
class Point {
    <> Serializable
    <> Validatable
    // members
}

// Desugars to:
Point = (...) -> {
    instance := {}
    // Merge Serializable methods
    @ key in Serializable { instance[key] <- Serializable[key] }
    // Merge Validatable methods
    @ key in Validatable { instance[key] <- Validatable[key] }
    // Add Point-specific members
    // ...
    ret instance
}
```

### Method Chaining

Methods that return `. ` (this) enable chaining:

```flap
class Builder {
    init = () -> {
        .parts = []
    }

    add = part -> {
        .parts <- .parts :: part
        ret .  // Return this (self)
    }

    build = () -> .parts
}

result = Builder().add("A").add("B").add("C").build()
```

### No Inheritance

Flap deliberately avoids inheritance hierarchies. Use composition:

```flap
// Instead of inheritance
Drawable = {
    draw: { println("Drawing...") }
}

Movable := {
    move: (dx, dy) -> {
        .x <- .x + dx
        .y <- .y + dy
    }
}

class {
    <> Sprite
    <> Drawable
    <> Movable
    init = (x, y) -> {
        .x = x
        .y = y
    }
}
```

## Parsing Algorithm

### High-Level Flow

```
1. Tokenize (lexer.go)
   Source → Tokens

2. Parse (parser.go)
   Tokens → AST

3. Type Inference (optional, see TYPE_TRACKING.md)
   AST → AST with type annotations

4. Code Generation (x86_64_codegen.go, arm64_codegen.go, riscv64_codegen.go)
   AST → Machine code

5. Linking (elf.go, macho.go)
   Machine code → Executable
```

### Parser Implementation Notes

**Recursive Descent:**
- Hand-written recursive descent parser
- Operator precedence climbing for expressions
- Look-ahead for block disambiguation

**Error Recovery:**
- Continue parsing after errors when possible
- Collect multiple errors per pass
- Provide helpful error messages with line numbers

**Performance:**
- Single-pass parsing (no separate AST transformation)
- Minimal memory allocation
- Fast compilation (typically <100ms for small programs)

## Implementation Guidelines

**Memory Management:**
- **ALWAYS use arena allocation** instead of malloc/free when possible
- The arena allocator (`flap_arena_alloc`) provides fast bump allocation with automatic growth
- Arena memory is freed in bulk, avoiding fragmentation
- Only use malloc for external C library compatibility

**Register Management:**
- The compiler has a sophisticated register allocator (`RegisterAllocator` in register_allocator.go)
- Real-time register tracking via `RegisterTracker` (register_tracker.go)
- Register spilling when needed via `RegisterSpiller` (register_allocator.go)
- Use these systems instead of ad-hoc register assignment

**Code Generation:**
- Target-independent IR through `Out` abstraction layer
- Backend-specific optimizations in arm64_backend.go, riscv64_backend.go, x86_64_codegen.go
- SIMD operations for parallel loops (AVX-512 on x86_64)

---

**Note:** This grammar is the canonical reference for Flap 3.0. The compiler implementation (lexer.go, parser.go) must match this specification exactly.

**See also:**
- [LANGUAGESPEC.md](LANGUAGESPEC.md) - Complete language semantics
- [TYPE_TRACKING.md](TYPE_TRACKING.md) - Compile-time type system
- [LIBERTIES.md](LIBERTIES.md) - Documentation accuracy guidelines
