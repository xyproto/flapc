# The Flap Programming Language spec

### Version 1.0.0

## Language Philosophy

Flap is a functional programming language designed for high-performance numerical computing. Built on a `map[uint64]float64` foundation, it provides elegant abstractions for modern CPU architectures while maintaining simplicity and clarity.

**Core Principle:** Everything is `map[uint64]float64`:
- Numbers are `map[uint64]float64` (e.g., 42 is `{0: 42.0}`)
- Strings are `map[uint64]float64` (character indices → char codes)
- Lists are `map[uint64]float64` (element indices → values)
- Maps are `map[uint64]float64` (keys → values)
- Functions are `map[uint64]float64` (pointers stored as float values)

This unified type system with a single underlying representation enables consistent optimization and uniform operations across all data structures.

## Language Spec

### Variables

```flap
// Immutable (default)
x = 10

// Mutable (requires :=)
y := 20
y := y + 5
```

### Operators

**Arithmetic:** `+` `-` `*` `/` `%` `**` (power)

**Comparison:** `<` `<=` `>` `>=` `==` `!=`

**Logical:** `and` `or` `xor` `not`

**Bitwise:** `&b` `|b` `^b` `~b` (operate on an integer representation of the float)

**Shifts:** `<b` `>b` (shift left/right), `<<b` `>>b` (rotate left/right)

**Pipeline:** `|` (functional composition: `x | f | g` ≡ `g(f(x))`)

**List:** `^` (head), `&` (tail), `#` (length), `::` (cons)

**Error handling:** `or!` (railway-oriented programming / error propagation)

**Control flow:** `ret` (break loop / return value)

**Type Casting:** `as` (convert between Flap and C types for FFI)
- To C: `as i8`, `as i16`, `as i32`, `as i64` (signed integers)
- To C: `as u8`, `as u16`, `as u32`, `as u64` (unsigned integers)
- To C: `as f32`, `as f64` (floating point)
- To C: `as cstr` (null-terminated string)
- To C: `as ptr` (pointer)
- From C: `as number` (any C type → Flap number)
- From C: `as string` (C string → Flap string)
- From C: `as list` (C array → Flap list)

### Match Expressions

Flap uses `match` blocks instead of if/else. A match block attaches to the preceding expression:

```flap
// Simple match (default case optional)
x > 42 {
    -> println("big")
    ~> println("small")
}

// Match without default (implicit 0)
x > 42 {
    -> 123           // sugar for "-> 123"
}

// Default-only (preserves condition value when true)
x > 42 {
    ~> 123           // yields 1.0 when true, 123 when false
}

// Subject/guard matching
x {
    x < 10 -> 0
    x < 20 -> 1
    ~> 2
}

// Ternary replacement
z = x > 42 { 1 ~> 0 }
```

### Strings

```flap
s := "Hello"         // Creates {0: 72.0, 1: 101.0, ...}
char := s[1]         // returns 101.0 (ASCII 'e')
println("Hello")     // String literals optimized for direct output
result := "Hello, " + "World!"  // Compile-time concatenation

// Slicing (Python-style with start:end:step)
s[0:2]               // "He" (indices 0, 1)
s[1:4]               // "ell" (indices 1, 2, 3)
s[::2]               // "Hlo" (every other character)
s[::-1]              // "olleH" (reversed)
s[1:5:2]             // Characters at indices 1, 3
```

### Lists

```flap
numbers = [1, 2, 3]
first = numbers[0]
length = #numbers    // length operator
head = ^numbers      // first element
tail = &numbers      // all but first

// Slicing works on lists too
numbers[0:2]         // [1, 2] (first two elements)
numbers[::2]         // [1, 3] (every other element)
numbers[::-1]        // [3, 2, 1] (reversed)
```

### Maps

```flap
ages = {1: 25, 2: 30, 3: 35}
empty = {}
count = #ages        // returns 3.0
price = ages[1]      // returns 25.0
missing = ages[999]  // returns 0.0 (key doesn't exist)
```

### Membership Testing

```flap
10 in numbers {
    -> println("Found!")
    ~> println("Not found")
}

result = 5 in mylist  // returns 1.0 or 0.0
```

### Loops

Loops use `@+` for auto-labeling by nesting depth (would be "for" in other languages):

```flap
// Basic loop (implicitly labeled @1), 5 in this case means from 0 (inclusive) up to 5 (exclusive), so 0,1,2,3,4
@+ i in 5 {
    println(i)
}

// Nested loops (labeled @1, @2, @3, ...)
@+ i in 3 {       // @1
    @+ j in 3 {   // @2
        printf("%v,%v ", i, j)
    }
}

// Range operator ..<
@+ i in 0..<3 {   // 0, 1, 2
    println(i)
}
```

**Loop Control:**
- `ret` - returns from function
- `ret value` - returns value from function
- `ret @1`, `ret @2`, `ret @3`, ... - exits loop at nesting level 1, 2, 3, ... and all inner loops
- `ret @1 value` - exits loop and returns value
- `@1`, `@2`, `@3`, ... - continues (jumps to top of) loop at nesting level 1, 2, 3, ...

**Loop Variables:**
- `@first` - true on first iteration
- `@last` - true on last iteration
- `@counter` - iteration count (starts at 0)
- `@i` - current element/key

**Example:**
```flap
@+ item in ["a", "b", "c"] {
    @first { printf("[") }
    printf("%v", item)
    @last { printf("]") ~> printf(", ") }
}
// Output: [a, b, c]
```

### Error Handling (Railway-Oriented Programming)

The `or!` operator enables clean error handling using railway-oriented programming:

```flap
// Convention: functions return 0.0 on error, non-zero on success
// or! checks the left side and either continues (success) or exits (error)

// Example: file operations with error handling
file = open("data.txt") or! "Failed to open file"
data = read(file) or! "Failed to read data"
result = process(data) or! "Failed to process data"

// Each operation either succeeds (continues with value) or fails (exits with message)
// This creates a "railway" where success stays on the main track
// and errors branch off to the error handling track (exit)

// Equivalent verbose version without or!:
file = open("data.txt")
file == 0 {
    -> println("Failed to open file") :: exit(1)
}
data = read(file)
data == 0 {
    -> println("Failed to read data") :: exit(1)
}
```

**Benefits:**
- No nested if/else for error checking
- Errors propagate automatically with clear messages
- Success path remains clean and readable
- Similar to Rust's `?` operator or Haskell's Either monad

### Lambdas

```flap
double = x => x * 2
add = x, y => x + y
result = double(5)

// Lambdas can have match blocks
classify = x => x > 0 {
    -> "positive"
    ~> "non-positive"
}
```

### Builtin Functions

**I/O:**
- `println(x)` - print with newline (syscall-based)
- `printf(fmt, ...)` - formatted print (libc-based)
- `exit(code)` - exit program (syscall-based)
- `cexit(code)` - exit program (libc-based)

**FFI:**
- `call(fn_name, ...)` - call C function with type-cast arguments
  - Example: `call("exit", 42 as i32)`
  - Example: `call("printf", "%s\n" as cstr, "hello" as cstr)`
- `dlopen(path, flags)` - load dynamic library, returns handle as number
- `dlsym(handle, symbol)` - get function pointer from library
- `dlclose(handle)` - close dynamic library

**Memory:**
- `read_i8(ptr, index)`, `read_i16(ptr, index)`, `read_i32(ptr, index)`, `read_i64(ptr, index)`
- `read_u8(ptr, index)`, `read_u16(ptr, index)`, `read_u32(ptr, index)`, `read_u64(ptr, index)`
- `read_f32(ptr, index)`, `read_f64(ptr, index)`
- `write_i8(ptr, index, val)`, `write_i16(ptr, index, val)`, `write_i32(ptr, index, val)`, `write_i64(ptr, index, val)`
- `write_u8(ptr, index, val)`, `write_u16(ptr, index, val)`, `write_u32(ptr, index, val)`, `write_u64(ptr, index, val)`
- `write_f32(ptr, index, val)`, `write_f64(ptr, index, val)`
- `sizeof_i8()`, `sizeof_i16()`, `sizeof_i32()`, `sizeof_i64()`, `sizeof_u8()`, `sizeof_u16()`, `sizeof_u32()`, `sizeof_u64()`, `sizeof_f32()`, `sizeof_f64()` - get size of type in bytes

**Format Specifiers:**
- `%v` - smart value (42.0→"42", 3.14→"3.14")
- `%b` - boolean (0.0→"no", non-zero→"yes")
- `%f` - float
- `%d` - integer
- `%s` - string

**Math:** (all using native x87 FPU or SSE2)
- `sqrt(x)`, `abs(x)`, `floor(x)`, `ceil(x)`, `round(x)`
- `sin(x)`, `cos(x)`, `tan(x)`
- `asin(x)`, `acos(x)`, `atan(x)`
- `log(x)`, `exp(x)`

## Grammar

The hand-written recursive-descent parser accepts the following grammar. Newlines separate statements but are otherwise insignificant. `//` starts a line comment. String escape sequences: `\n`, `\t`, `\r`, `\\`, `\"`.

```ebnf
program         = { newline } { statement { newline } } ;

statement       = loop_statement
                | jump_statement
                | assignment
                | expression_statement ;

loop_statement  = "@+" identifier "in" expression block
                | "@" number identifier "in" expression block ;

jump_statement  = "ret" [ "@" number ] [ expression ]
                | "@" number ;

assignment      = identifier [ ":" type_annotation ] ("=" | ":=") expression ;

type_annotation = ("b" | "f") number ;

expression_statement = expression [ match_block ] ;

match_block     = "{" ( default_arm
                      | match_clause { match_clause } [ default_arm ] ) "}" ;

match_clause    = "->" match_target
                | expression [ "->" match_target ] ;

default_arm     = "~>" match_target ;

match_target    = jump_target | expression ;

jump_target     = "ret" [ "@" number ] [ expression ]
                | "@" number ;

block           = "{" { statement { newline } } "}" ;

expression              = or_bang_expr ;

or_bang_expr            = pipe_expr [ "or!" string ] ;

pipe_expr               = logical_or_expr { "|" logical_or_expr } ;

logical_or_expr         = logical_and_expr { ("or" | "xor") logical_and_expr } ;

logical_and_expr        = comparison_expr { "and" comparison_expr } ;

comparison_expr         = range_expr [ (rel_op range_expr) | ("in" range_expr) ] ;

rel_op                  = "<" | "<=" | ">" | ">=" | "==" | "!=" ;

range_expr              = additive_expr [ "..<" additive_expr ] ;

additive_expr           = cons_expr { ("+" | "-") cons_expr } ;

cons_expr               = bitwise_expr { "::" bitwise_expr } ;

bitwise_expr            = multiplicative_expr { ("|b" | "&b" | "^b" | "<b" | ">b" | "<<b" | ">>b") multiplicative_expr } ;

power_expr              = unary_expr { "**" unary_expr } ;

multiplicative_expr     = power_expr { ("*" | "/" | "%" | "*+") power_expr } ;

unary_expr              = ("not" | "-" | "#" | "~b") unary_expr
                        | postfix_expr ;

postfix_expr            = primary_expr { "[" expression "]"
                                       | "(" [ argument_list ] ")"
                                       | "++"
                                       | "--"
                                       | "as" cast_type } ;

cast_type               = "i8" | "i16" | "i32" | "i64"
                        | "u8" | "u16" | "u32" | "u64"
                        | "f32" | "f64"
                        | "cstr" | "ptr"
                        | "number" | "string" | "list" ;

primary_expr            = number
                        | string
                        | identifier
                        | loop_state_var
                        | "(" expression ")"
                        | lambda_expr
                        | list_literal
                        | map_literal
                        | "^" primary_expr
                        | "&" primary_expr ;

loop_state_var          = "@first" | "@last" | "@counter" | "@i" ;

lambda_expr             = parameter_list "=>" lambda_body ;

lambda_body             = expression [ match_block ] ;

parameter_list          = identifier { "," identifier } ;

argument_list           = expression { "," expression } ;

list_literal            = "[" [ expression { "," expression } ] "]" ;

map_literal             = "{" [ map_entry { "," map_entry } ] "}" ;

map_entry               = expression ":" expression ;

identifier              = letter { letter | digit | "_" } ;

number                  = [ "-" ] digit { digit } [ "." digit { digit } ] ;

string                  = '"' { character } '"' ;

character               = printable_char | escape_sequence ;

escape_sequence         = "\\" ( "n" | "t" | "r" | "\\" | '"' ) ;
```

### Grammar Notes

* `@+` introduces auto-labeled loops. The loop label is the current nesting depth (1, 2, 3, ...).
* `@1`, `@2`, `@3`, ... continues the loop at that nesting level by jumping to its top.
* When used in a loop statement (`@1 identifier in expression`), it explicitly labels that loop.
* `ret` returns from the current function. `ret @1`, `ret @2`, `ret @3`, ... exits the loop at that nesting level and all inner loops.
* Lambda syntax: `x => expr` or `x, y => expr` (no parentheses around parameters).
* Type casting with `as`: Bidirectional conversion for FFI (e.g., `42 as i32` to C, `c_value as number` from C).
* Match blocks attach to the preceding expression. When omitted, implicit default is `0`.
* A single bare expression inside braces is shorthand for `-> expression`.
* A block with only `~>` leaves the condition's value untouched when true.
* Type annotations use `:b64` or `:f32` syntax for precision control.

## Keywords

```
and as in not or or! ret xor &b |b ^b ~b <b >b >>b <<b
i8 i16 i32 i64 u8 u16 u32 u64 f32 f64 cstr ptr
number string list
```

**Note:** Type keywords (`i8`, `i16`, `i32`, `i64`, `u8`, `u16`, `u32`, `u64`, `f32`, `f64`, `cstr`, `ptr`, `number`, `string`, `list`) are **contextual keywords** - they are only reserved when used after `as` in type casting expressions. They can be used as variable names in other contexts:

```flap
// Valid: using type keywords as variable names
i32 = 100.0
ptr = call("malloc", 64 as u64)
string = "hello"

// Also valid: using them as type keywords in casts
x = 42 as i32
address = pointer_value as ptr
text = c_string as string
```

## Examples

### Factorial

```flap
factorial = n => n <= 1 {
    -> 1
    ~> n * factorial(n - 1)
}

println(factorial(5))  // 120
```

### FizzBuzz

```flap
@+ i in 1..<101 {
    i % 15 == 0 {
        -> println("FizzBuzz")
    }
    i % 3 == 0 {
        -> println("Fizz")
    }
    i % 5 == 0 {
        -> println("Buzz")
    }
    println(i)
}
```

### List Processing

```flap
sum = list => {
    result := 0
    @+ x in list {
        result := result + x
    }
    result
}

println(sum([1, 2, 3, 4, 5]))  // 15
```

### Filtering

```flap
filter = predicate, list => {
    result := []
    @+ x in list {
        predicate(x) {
            -> result := result ++ x
        }
    }
    result
}

positive = x => x > 0
numbers = [-2, -1, 0, 1, 2]
println(filter(positive, numbers))  // [1, 2]
```

### FFI (Foreign Function Interface)

```flap
// Call C functions with type casting TO C
call("printf", "Hello from C!\n" as cstr)

// Get values FROM C
time_val = call("time", 0 as ptr)
timestamp = time_val as number
printf("Unix time: %f\n", timestamp)

// String conversion FROM C
home_ptr = call("getenv", "HOME" as cstr)
home_str = home_ptr as string
printf("HOME: %s\n", home_str)

// Memory allocation with safe read/write
ptr = call("malloc", 64 as u64)
write_f64(ptr, 0, 42.0)         // Write float64 at index 0
write_i32(ptr, 1, 100)          // Write int32 at index 1
value = read_f64(ptr, 0)        // Read back float64
int_val = read_i32(ptr, 1)      // Read back int32
call("free", ptr as ptr)

// Working with C structs (safe indexing)
// struct Point { float x; float y; }
point_ptr = call("malloc", 8 as u64)
write_f32(point_ptr, 0, 10.5)   // x field at index 0
write_f32(point_ptr, 1, 20.3)   // y field at index 1
x_val = read_f32(point_ptr, 0)
y_val = read_f32(point_ptr, 1)
call("free", point_ptr as ptr)
```

## Automatic Dependency Resolution

Flap uses automatic dependency resolution - no explicit `import` statements needed. When the compiler encounters an unknown function, it automatically fetches and compiles the required code from predefined repositories.

### Example

```flap
// No import needed!
x := -5
y := abs(x)      // Compiler automatically fetches flap_math
println(y)       // Outputs: 5
```

### Cache Management

```bash
# View cached dependencies
ls ~/.cache/flapc/

# Update all dependencies
flapc --update-deps myprogram.flap

# Clear cache
rm -rf ~/.cache/flapc/
```
