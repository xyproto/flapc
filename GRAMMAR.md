# Flap Grammar Specification

**Version:** 3.0.0
**Date:** 2025-11-26
**Status:** Canonical Grammar Reference for Flap 3.0 Release

This document defines the complete formal grammar of the Flap programming language using Extended Backus-Naur Form (EBNF).

## ⚠️ CRITICAL: The Universal Type

Flap has exactly ONE runtime type: `map[uint64]float64`, an ordered map.

Not "represented as" or "backed by" — every value IS this map:

```flap
42              // {0: 42.0}
"Hello"         // {0: 72.0, 1: 101.0, 2: 108.0, 3: 108.0, 4: 111.0}
[1, 2, 3]       // {0: 1.0, 1: 2.0, 2: 3.0}
{x: 10}         // {hash("x"): 10.0}
[]              // {}
{}              // {}
```

**Even C foreign types are stored as maps:**

```flap
// C pointer (0x7fff1234) stored as float64 bits
ptr: cptr = sdl.SDL_CreateWindow(...)  // {0: <pointer_as_float64>}

// C string pointer
err: cstring = sdl.SDL_GetError()      // {0: <char*_as_float64>}

// C int
result: cint = sdl.SDL_Init(...)       // {0: 1.0} or {0: 0.0}
```

There are NO special types, NO primitives, NO exceptions.
Everything is a map from uint64 to float64.

This is not an implementation detail — this IS Flap.

## Type Annotations

Type annotations are **metadata** that specify:
1. **Semantic intent** - what does this map represent?
2. **FFI conversions** - how to marshal at C boundaries
3. **Optimization hints** - compiler optimizations

They do NOT change the runtime representation (always `map[uint64]float64`).

### Native Flap Types
- `num` - number (default type)
- `str` - string (map of char codes)
- `list` - list (map with integer keys)
- `map` - explicit map

### Foreign C Types
- `cstring` - C `char*` (pointer stored as `{0: <ptr>}`)
- `cptr` - C pointer (e.g., `SDL_Window*`)
- `cint` - C `int`/`int32_t`
- `clong` - C `int64_t`/`long`
- `cfloat` - C `float`
- `cdouble` - C `double`
- `cbool` - C `bool`/`_Bool`
- `cvoid` - C `void` (return type only)

Foreign types are used at FFI boundaries to guide marshalling.

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

| Notation          | Meaning                   |             |
|-------------------|---------------------------|-------------|
| `=`               | Definition                |             |
| `;`               | Termination               |             |
| `|`               | Alternation               |             |
| `[ ... ]`         | Optional (zero or one)    |             |
| `{ ... }`         | Repetition (zero or more) |             |
| `( ... )`         | Grouping                  |             |
| `"..."`           | Terminal string           |             |
| `letter`, `digit` | Character classes         |             |

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
(x > 0) {
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

## Import System

Flap's import system provides a unified way to import libraries, git repositories, and local directories.

### Import Resolution Priority

1. **Libraries** (highest priority)
   - System libraries via pkg-config (Linux/macOS)
   - .dll files in current directory or system paths (Windows)
   - Headers in standard include paths

2. **Git Repositories**
   - GitHub, GitLab, Bitbucket
   - SSH or HTTPS URLs
   - Optional version specifiers

3. **Local Directories** (lowest priority)
   - Relative or absolute paths
   - Current directory with `.`

### Import Syntax

```flap
// Library import (uses pkg-config or finds .dll)
import "sdl3" as sdl
import "raylib" as rl

// Git repository import
import "github.com/xyproto/flap-math" as math
import "github.com/xyproto/flap-math@v1.0.0" as math
import "github.com/xyproto/flap-math@latest" as math
import "github.com/xyproto/flap-math@main" as math
import "git@github.com:xyproto/flap-math.git" as math

// Directory import
import "." as local                    // Current directory
import "./subdir" as sub              // Relative path
import "/absolute/path" as abs        // Absolute path

// C library file import
import "/path/to/libmylib.so" as mylib
import "SDL3.dll" as sdl
```

### Import Behavior

- **Libraries**: Searches for library files and headers, parses C headers for FFI
- **Git Repos**: Clones to `~/.cache/flapc/` (respects `XDG_CACHE_HOME`), imports all top-level `.flap` files
- **Directories**: Imports all top-level `.flap` files from the directory
- **Version Specifiers**:
  - `@v1.0.0` - Specific tag
  - `@main` or `@master` - Specific branch
  - `@latest` - Latest tag (or default branch if no tags)
  - No `@` - Uses default branch

## Program Execution Model

Flap programs can be structured in three ways:

### 1. Main Function
When a `main` function is defined, it becomes the program entry point:

```flap
main = { println("Hello!") }     // A lambda that returns the value returned from println (0)
main = 42                        // A Flap number {0: 42.0}
main = () -> { 100 }             // A lambda that returns 100
main = { 100 }                   // A lambda that returns 100
```

**Return value rules:**
- If `main` is set to a number, it is converted to int32 for the exit code
- If `main` returns an empty map `{}` or empty list `[]`: exit code 0
- If `main` is callable (function): called, result becomes exit code
- Return values are implicitly cast to int32 for `_start`

### 2. Main Variable
When a `main` variable (not a function) is defined without top-level code:

```flap
main = 42        // Exit with code 42
main = {}        // Exit with code 0 (empty map)
main = []        // Exit with code 0 (empty list)
```

**Evaluation:**
- The value of `main` becomes the program's exit code
- Non-callable values are used directly

### 3. Top-Level Code
When there's no `main` function or variable, top-level code executes:

```flap
println("Hello!")
x := 42
println(x)
// Last expression or ret determines exit code
```

**Exit code:**
- Last expression value becomes exit code
- `ret` keyword sets explicit exit code
- No explicit return: exit code 0

### Mixed Cases

**Top-level code + main function:**
- Top-level code executes first
- It's the responsibility of top-level code to call `main()`
- If top-level doesn't call `main()`, `main()` is never executed
- Last expression in top-level code provides exit code

```flap
// Top-level setup
x := 100

main = { println(x); 42 }

// main is defined but not called - exit code is 0
// To call: main() must appear in top-level code
```

**Top-level code + main variable:**
- Top-level code executes
- `main` variable is accessible but not special
- Last top-level expression provides exit code

```flap
main = 99

println("Setup")
42  // Exit code is 42, not 99
```

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
                | defer_statement
                | import_statement ;

return_statement = "ret" [ "@" [ integer ] ] [ expression ] ;

defer_statement  = "defer" expression ;

import_statement = "import" import_source [ "as" identifier ] ;

import_source   = string_literal           (* library name, file path, or directory *)
                | git_url [ "@" version_spec ] ; (* git repository with optional version *)

git_url         = identifier { "." identifier } { "/" identifier }  (* github.com/user/repo *)
                | "git@" identifier ":" identifier "/" identifier ".git" ; (* git@github.com:user/repo.git *)

version_spec    = identifier              (* tag, branch, "latest", or semver like "v1.0.0" *)
                | "latest" ;

cstruct_decl    = "cstruct" identifier "{" { field_decl } "}" ;

field_decl      = identifier "as" c_type [ "," ] ;

class_decl      = "class" identifier [ extend_clause ] "{" { class_member } "}" ;

extend_clause   = { "<>" identifier } ;

class_member    = class_field_decl
                | method_decl ;

class_field_decl = identifier "." identifier "=" expression ;

method_decl     = identifier "=" lambda_expr ;

c_type          = "int8" | "int16"   | "int32"   | "int64"
                | "uint8"   | "uint16"  | "uint32" | "uint64"
                | "float32" | "float64"
                | "ptr"     | "cstr" ;

arena_statement = "arena" block ;

loop_statement  = "@" block
                | "@" identifier "in" expression [ "max" expression ] block
                | "@" expression [ "max" expression ] block ;

parallel_statement = "||" identifier "in" expression block ;

unsafe_statement = "unsafe" type_cast block [ block ] [ block ] ;

type_cast       = "int8" | "int16"   | "int32"     | "int64"
                | "uint8"   | "uint16"    | "uint32" | "uint64"
                | "float32" | "float64"
                | "number"  | "string"    | "list"   | "address"
                | "packed"  | "aligned" ;

assignment      = identifier [ ":" type_annotation ] ( "=" | ":=" | "<-" ) expression
                | identifier ( "+=" | "-=" | "*=" | "/=" | "%=" | "**=" ) expression
                | indexed_expr "<-" expression
                | identifier_list ( "=" | ":=" | "<-" ) expression ;  // Multiple assignment

identifier_list = identifier { "," identifier } ;

type_annotation = native_type | foreign_type ;

native_type     = "num" | "str" | "list" | "map" ;

foreign_type    = "cstring" | "cptr"   | "cint"    | "clong"
                | "cfloat" | "cdouble" | "cbool" | "cvoid" ;

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

param_decl      = identifier [ ":" type_annotation ] [ "..." ] ;

variadic_params = "(" identifier [ ":" type_annotation ] { "," identifier [ ":" type_annotation ] } "," identifier [ ":" type_annotation ] "..." ")" ;

lambda_body     = [ "->" type_annotation ] ( block | expression [ match_block ] ) ;

argument_list   = expression { "," expression } ;

list_literal    = "[" [ expression { "," expression } ] "]" ;

map_literal     = "{" [ map_entry { "," map_entry } ] "}" ;

map_entry       = ( identifier | string ) ":" expression ;

identifier      = letter { letter | digit | "_" } ;

number          = [ "-" ] digit { digit } [ "." digit { digit } ] ;

string          = '"' { character } '"' ;

fstring         = 'f"' { character | "{" expression "}" } '"' ;

## Lexical Elements

### Identifiers

Identifiers start with a letter and contain letters, digits, or underscores:

```ebnf
identifier = letter { letter | digit | "_" } ;
letter     = "a"..."z" | "A"..."Z" ;
digit      = "0"..."9" ;
```

**Rules:**
- Case-sensitive
- Can start with a letter only (not a digit or underscore)
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

### Type Keywords

Type annotations use these keywords (context-dependent):

**Native Flap types:**
```
num str list map
```

**Foreign C types:**
```
cstring cptr cint clong cfloat cdouble cbool cvoid
```

**Legacy type cast keywords (for `unsafe` blocks and `cstruct`):**
```
int8 int16 int32 int64 uint8 uint16 uint32 uint64 float32 float64
ptr cstr number string address packed aligned
```

**Usage:**
```flap
// Type annotations (preferred)
x: num = 42
name: str = "Alice"
ptr: cptr = sdl.SDL_CreateWindow(...)

// Type casts in unsafe blocks (legacy)
value = unsafe int32 { ... }
```

Type keywords are contextual - you can still use them as variable names in most contexts:

```flap
num = 100              // OK - variable named num
x: num = num * 2       // OK - type annotation vs variable
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

| Operator | Context           | Meaning                            | Example                            |
|----------|-------------------|------------------------------------|------------------------------------|
| `->`     | Lambda definition | Lambda arrow                       | `x -> x * 2` or `-> println("hi")` |
| `=>`     | Match block       | Match arm                          | `x { 0 => "zero" ~> "other" }`     |
| `~>`     | Match block       | Default match arm                  | `x { 0 => "zero" ~> "other" }`     |
| `_ =>`   | Match block       | Default match arm (alias for ~>)   | `x { 0 => "zero" _ => "other" }`   |
| `=`      | Variable binding  | Immutable assignment               | `x = 42` (standard for functions)  |
| `:=`     | Variable binding  | Mutable assignment                 | `x := 42` (can reassign later)     |
| `<-`     | Update/Send       | Update mutable var OR send to ENet | `x <- 99` or `&8080 <- msg`        |
| `<=`     | Comparison/Receive| Less than or equal OR receive from ENet | `x <= 10` or `msg <= &8080`      |
| `>=`     | Comparison        | Greater than or equal              | `x >= 10`                          |

**Important Conventions:**
- **Functions/methods** should use `=` (immutable), not `:=`, since they rarely need reassignment
- **Lambda syntax**: `->` always defines a lambda, `=>` always defines a match arm
- **Update operator** `<-` is for updating existing mutable variables or sending to ENet channels
- **Receive operator** `<=` is for receiving from ENet channels.
- **Comparison** operators `<=` and `>=` are for comparisons, not assignment or arrows

### Collection Operators

```
#     Length operator (prefix or postfix)
```

### Other Operators

```
|     Pipe operator
||    Parallel map
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
