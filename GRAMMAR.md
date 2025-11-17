# Flap Grammar Specification

**Version:** 2.0.0  
**Date:** 2025-11-17  
**Status:** Canonical Grammar Reference

This document defines the complete formal grammar of the Flap programming language using Extended Backus-Naur Form (EBNF).

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
                | return_statement
                | break_statement
                | continue_statement ;

return_statement = "ret" [ expression ] ;

break_statement  = "break" ;

continue_statement = "continue" ;

cstruct_decl    = "cstruct" identifier "{" { field_decl } "}" ;

field_decl      = identifier "as" c_type [ "," ] ;

c_type          = "int8" | "int16" | "int32" | "int64"
                | "uint8" | "uint16" | "uint32" | "uint64"
                | "float32" | "float64"
                | "ptr" | "cstr" ;

arena_statement = "arena" block ;

loop_statement  = "@" block
                | "@" identifier "in" expression block
                | "@" expression block ;

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
                | "(" expression ")"
                | "??"
                | unsafe_expr
                | arena_expr
                | "???" ;

enet_address    = "@" port_or_host_port ;

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

Numbers are 64-bit IEEE 754 floating-point values:

```ebnf
number = [ "-" ] digit { digit } [ "." digit { digit } ] ;
```

**Examples:**
```flap
42
3.14159
-17
0.001
1000000
-273.15
```

**Special values:**
- `??` - cryptographically secure random number [0, 1)
- Result of `0/0` - NaN (used for error encoding)

### Strings

Strings are UTF-8 encoded text:

```ebnf
string = '"' { character } '"' ;
```

**Escape sequences:**
- `\n` - newline
- `\t` - tab
- `\r` - carriage return
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
ret break continue arena unsafe cstruct as
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
@     Loop / ENet address
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

---

**Note:** This grammar is the canonical reference. The compiler implementation (parser.go) should match this specification exactly.
