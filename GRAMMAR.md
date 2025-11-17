# Flap Grammar Specification

**Version:** 3.0.0  
**Date:** 2025-11-17  
**Status:** Canonical Grammar Reference for Flap 3.0 Release

This document defines the complete formal grammar of the Flap programming language using Extended Backus-Naur Form (EBNF).

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
**Condition:** First element contains `:` (before any `->` or `~>`)

```flap
config = { port: 8080, host: "localhost" }
settings = { "key": value, "other": 42 }
```

### Rule 2: Match Block
**Condition:** Contains `->` or `~>` in the block's scope

There are TWO forms:

#### Form A: Value Match (with expression before `{`)
Evaluates expression, then matches its result against patterns:

```flap
// Match on literal values
x { 
    0 -> "zero"
    5 -> "five"
    ~> "other"
}

// Boolean match
x > 0 {
    1 -> "positive"    // true = 1
    0 -> "zero"        // false = 0
}
```

#### Form B: Guard Match (no expression, uses `|` at line start)
Each branch evaluates its own condition independently:

```flap
// Guard branches with | at line start
{
    | x == 0 -> "zero"
    | x > 0 -> "positive"
    | x < 0 -> "negative"
    ~> "unknown"  // optional default
}
```

**Important:** The `|` is only a guard marker when at the start of a line/clause.
Otherwise `|` is the pipe operator: `data | transform | filter`

### Rule 3: Statement Block
**Condition:** No `->` or `~>` in scope, not a map

```flap
compute = x => {
    temp = x * 2
    result = temp + 10
    result    // Last expression returned
}
```

**Disambiguation order:**
1. Check for `:` → Map literal
2. Check for `->` or `~>` → Match block
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
                | return_statement ;

return_statement = "ret" [ "@" [ integer ] ] [ expression ] ;

cstruct_decl    = "cstruct" identifier "{" { field_decl } "}" ;

field_decl      = identifier "as" c_type [ "," ] ;

class_decl      = "class" identifier [ extend_clause ] "{" { class_member } "}" ;

extend_clause   = { "<>" identifier } ;

class_member    = class_field_decl
                | method_decl ;

class_field_decl = identifier "." identifier "=" expression ;

method_decl     = identifier ":=" lambda_expr ;

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

assignment      = identifier ("=" | ":=" | "<-" | "==>") expression
                | identifier ("+=" | "-=" | "*=" | "/=" | "%=" | "**=") expression
                | indexed_expr "<-" expression ;

indexed_expr    = identifier "[" expression "]" ;

expression_statement = expression [ match_block ] ;

match_block     = "{" ( default_arm
                      | match_clause { match_clause } [ default_arm ]
                      | guard_clause { guard_clause } [ default_arm ] ) "}" ;

match_clause    = expression [ "->" match_target ] ;

guard_clause    = "|" expression "->" match_target ;  // | must be at start of line

default_arm     = "~>" match_target ;

match_target    = jump_target | expression ;

jump_target     = integer ;

block           = "{" { statement { newline } } [ expression ] "}" ;

expression      = pipe_expr ;

pipe_expr       = reduce_expr { ( "|" | "||" | "|||" ) reduce_expr } ;

reduce_expr     = receive_expr ;

receive_expr    = "=>" pipe_expr | pipe_expr ;

send_expr       = or_expr { "<-" or_expr } ;

or_expr         = and_expr { "||" and_expr } ;

and_expr        = comparison_expr { "&&" comparison_expr } ;

comparison_expr = bitwise_or_expr { comparison_op bitwise_or_expr } ;

comparison_op   = "==" | "!=" | "<" | "<=" | ">" | ">=" ;

bitwise_or_expr = bitwise_xor_expr { "|b" bitwise_xor_expr } ;

bitwise_xor_expr = bitwise_and_expr { "^b" bitwise_and_expr } ;

bitwise_and_expr = shift_expr { "&b" shift_expr } ;

shift_expr      = additive_expr { shift_op additive_expr } ;

shift_op        = "<<b" | ">>b" | "<<<b" | ">>>b" ;

additive_expr   = multiplicative_expr { ("+" | "-") multiplicative_expr } ;

multiplicative_expr = power_expr { ("*" | "/" | "%") power_expr } ;

power_expr      = unary_expr { "**" unary_expr } ;

unary_expr      = ( "-" | "!" | "~b" ) unary_expr
                | postfix_expr ;

postfix_expr    = primary_expr { postfix_op } ;

postfix_op      = "[" expression "]"
                | "." ( identifier | integer )
                | "(" [ argument_list ] ")"
                | "!"
                | match_block ;

primary_expr    = identifier
                | number
                | string
                | fstring
                | list_literal
                | map_literal
                | lambda_expr
                | enet_address
                | instance_field
                | "this"
                | "(" expression ")"
                | "??"
                | unsafe_expr
                | arena_expr
                | "???" ;

instance_field  = "." identifier ;

enet_address    = "&" port_or_host_port ;

port_or_host_port = port | [ hostname ":" ] port ;

port            = digit { digit } ;

hostname        = identifier | ip_address ;

ip_address      = digit { digit } "." digit { digit } "." digit { digit } "." digit { digit } ;

arena_expr      = "arena" "{" { statement { newline } } [ expression ] "}" ;

unsafe_expr     = "unsafe" "{" { statement { newline } } [ expression ] "}"
                  [ "{" { statement { newline } } [ expression ] "}" ]
                  [ "{" { statement { newline } } [ expression ] "}" ] ;

lambda_expr     = [ parameter_list ] "=>" lambda_body 
                | "==>" lambda_body ;  // Shorthand for () =>

lambda_body     = block | expression [ match_block ] ;

// Lambda body semantics:
// 1. block: Statement block, map literal, or match block
//    Block type determined by contents:
//    - Contains `:` before arrows → map literal
//    - Contains `->` or `~>` → match block
//    - Otherwise → statement block
//
// 2. expression [ match_block ]: Value match
//    Example: x => x { 0 -> "zero" ~> "other" }
//    Expression is evaluated, result matched against patterns
//
// Match block forms:
//   Value match: expr { pattern -> result }
//   Guard match: { | condition -> result }  (| at line start only)
//
// Examples:
//   x => x { 0 -> "zero" }           // Value match
//   x => { | x > 0 -> "pos" }        // Guard match (| at start)
//   x => { temp = x * 2; temp }      // Statement block
//   x => data | transform            // Pipe operator (| not at start)

parameter_list  = identifier [ "," identifier ]*
                | "(" [ identifier [ "," identifier ]* ] ")" ;

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

## Operators

### Arithmetic Operators

```
+    Addition
-    Subtraction (binary) or negation (unary)
*    Multiplication
/    Division
%    Modulo
**   Exponentiation
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
=     Immutable assignment
:=    Mutable assignment
<-    Update/reassignment (for mutable vars)
==>   No-arg lambda shorthand (alias for = () =>)

+=    Add and assign
-=    Subtract and assign
*=    Multiply and assign
/=    Divide and assign
%=    Modulo and assign
**=   Exponentiate and assign
```

### Other Operators

```
=>    Lambda arrow
->    Match arm
~>    Default match arm
|     Pipe operator
||    Parallel map
|||   Reduce/fold
<-    Send (ENet)
=>    Receive (ENet, prefix)
!     Move operator (postfix)
.     Field access
[]    Indexing
()    Function call
@     Loop
&     Address (ENet)
??    Random number
```

## Operator Precedence

From highest to lowest precedence:

1. **Primary**: `()` `[]` `.` function call, postfix `!`
2. **Unary**: `-` `!` `~b`
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
13. **Send**: `<-`
14. **Receive**: `=>`
15. **Pipe/Reduce**: `|` `||` `|||`
16. **Match**: `{ }` (postfix)
17. **Assignment**: `=` `:=` `<-` `==>` `+=` `-=` `*=` `/=` `%=` `**=`

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
   (x, y) => x + y  // Multiple params
   ```

**Not needed:**
```flap
// Good: no unnecessary parens
x > 0 { -> "positive" ~> "negative" }
result = x + y * z
classify = x => x { 0 -> "zero" ~> "other" }

// Bad: unnecessary parens
result = x > 0 { -> ("positive") ~> ("negative") }
compute = (x) => (x * 2)
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
classify = x => {
    | x > 0 -> "positive"
    | x < 0 -> "negative"
    ~> "zero"
}
```

**Rule:** `|` at the start of a line/clause (after `{` or newline) is a guard marker. Otherwise it's the pipe operator.

#### Arrow Disambiguation

```flap
->   Match arm result
~>   Default match arm
=>   Lambda or receive
==>  No-arg lambda shorthand
```

Context determines meaning:

```flap
f = x => x + 1           // Lambda
msg = => &8080           // Receive
x { 0 -> "zero" }        // Match arm
x { ~> "default" }       // Default arm
greet ==> println("Hi")  // No-arg lambda
```

#### Loop Forms

The `@` symbol introduces loops (one of three forms):

```flap
@ { ... }                  // Infinite loop
@ i in collection { ... }  // For-each loop
@ condition { ... }        // While loop
```

**Loop Control with `ret @`:**

Instead of `break`/`continue` keywords, Flap uses `ret @` with loop labels:

```flap
// Exit current loop
@ i in 0..<100 {
    i > 50 { ret @ }      // Exit loop (equivalent to ret @1)
    i == 42 { ret @ 42 }  // Exit loop with value 42
    println(i)
}

// Nested loops - explicit labels
@ i in 0..<10 {           // This is loop @1
    @ j in 0..<10 {       // This is loop @2
        j == 5 { ret @ }         // Exit inner loop (current loop)
        i == 5 { ret @1 }        // Exit outer loop (loop @1)
        i == 3 and j == 7 { ret @1 42 }  // Exit outer loop with value
        println(i, j)
    }
}

// ret without @ returns from function
compute = n => {
    @ i in 0..<100 {
        i == n { ret i }  // Return from function
        i == 50 { ret @ } // Exit loop only
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
    init := (x, y) ==> {
        .x = x
        .y = y
    }
    
    // Instance methods
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
Point := (x, y) => {
    instance := {}
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
    
    ret instance
}
```

### Instance Fields

Use `.field` inside class methods to access instance state:

```flap
class Counter {
    init := start ==> {
        .count = start
        .history = []
    }
    
    increment := ==> {
        .count <- .count + 1
        .history <- .history :: .count
    }
    
    get := ==> .count
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
    
    init := name ==> {
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
Serializable := {
    to_json: ==> {
        // Convert instance to JSON
    },
    from_json: json => {
        // Parse JSON to instance
    }
}

class Point <> Serializable {
    init := (x, y) ==> {
        .x = x
        .y = y
    }
}

p := Point(10, 20)
json := p.to_json()
```

**Multiple composition** - chain `<>` operators:

```flap
class User <> Serializable <> Validatable <> Timestamped {
    init := name ==> {
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
    
    init := (x, y) ==> {
        .x = x           // Instance field (this instance)
        .y = y
    }
    
    distance_to_origin := ==> {
        .distance(Point.origin)  // Class field access
    }
    
    distance := other => {
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
    init := balance ==> {
        .balance = balance
    }
    
    _validate := amount => {
        amount > 0 && amount <= .balance
    }
    
    withdraw := amount => {
        ._ validate(amount) {
            .balance <- .balance - amount
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
    init := (x, y) ==> {
        .data = call("malloc", Vec2Data.size as uint64)
        unsafe float64 {
            rax <- .data as ptr
            [rax] <- x
            [rax + 8] <- y
        }
    }
    
    magnitude := ==> {
        unsafe float64 {
            rax <- .data as ptr
            xmm0 <- [rax]
            xmm1 <- [rax + 8]
            xmm0 <- xmm0 * xmm0
            xmm1 <- xmm1 * xmm1
            xmm0 <- xmm0 + xmm1
        } | result => sqrt(result)
    }
}
```

### Operator Overloading via Methods

While Flap doesn't have operator overloading syntax, you can define methods with operator-like names:

```flap
class Complex {
    init := (real, imag) ==> {
        .real = real
        .imag = imag
    }
    
    add := other => Complex(.real + other.real, .imag + other.imag)
    mul := other => Complex(
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
class Point <> Serializable <> Validatable {
    // members
}

// Desugars to:
Point := (...) => {
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

Methods that return `instance` (or `.` implicitly) enable chaining:

```flap
class Builder {
    init := ==> {
        .parts = []
    }
    
    add := part ==> {
        .parts <- .parts :: part
        ret .  // Return self
    }
    
    build := ==> .parts
}

result := Builder().add("A").add("B").add("C").build()
```

### No Inheritance

Flap deliberately avoids inheritance hierarchies. Use composition:

```flap
// Instead of inheritance
Drawable := {
    draw: ==> println("Drawing...")
}

Movable := {
    move: (dx, dy) ==> {
        .x <- .x + dx
        .y <- .y + dy
    }
}

class Sprite <> Drawable <> Movable {
    init := (x, y) ==> {
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

## Grammar Extensions for Future Versions

The grammar is designed to be extensible. Potential future additions:

- **Type aliases:** `type Point = { x: float64, y: float64 }`
- **Generics:** `f = <T>(x as T) => x`
- **Macros:** `macro! name { ... }`
- **Modules:** `import "module"`

These extensions must preserve:
1. Universal map type system
2. Minimal syntax philosophy
3. Direct code generation capability

---

**Note:** This grammar is the canonical reference for Flap 3.0. The compiler implementation (lexer.go, parser.go) must match this specification exactly.

**See also:**
- [LANGUAGESPEC.md](LANGUAGESPEC.md) - Complete language semantics
- [TYPE_TRACKING.md](TYPE_TRACKING.md) - Compile-time type system
- [LIBERTIES.md](LIBERTIES.md) - Documentation accuracy guidelines
