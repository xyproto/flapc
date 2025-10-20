# The Flap Programming Language

### Version 1.1.0

## Language Philosophy

Flap is a functional programming language designed for high-performance numerical computing with ergonomic modern syntax. Built on a `map[uint64]float64` foundation, it provides elegant abstractions for modern CPU architectures while maintaining simplicity and clarity.

**Core Principle:** Everything is `map[uint64]float64`:
- Numbers are `map[uint64]float64` (e.g., 42 is `{0: 42.0}`)
- Strings are `map[uint64]float64` (character indices → char codes)
- Lists are `map[uint64]float64` (element indices → values)
- Maps are `map[uint64]float64` (keys → values)
- Functions are `map[uint64]float64` (pointers stored as float values)

This unified type system with a single underlying representation enables consistent optimization and uniform operations across all data structures.

## Why Flap? Three Killer Features

### 1. **Zero-Cost Abstractions with Predictable Performance**
Unlike Python's interpreter overhead, Go's GC pauses, or C++'s template bloat, Flap compiles directly to native machine code with **zero runtime**. Every abstraction (loops, maps, strings) compiles to tight assembly. You get Python-like ergonomics with C-like speed.

```flap
// This compiles to 5 AVX-512 instructions, no loops
numbers = [1, 2, 3, 4, 5]
sum := numbers | fold(+)  // SIMD vectorized automatically
```

### 2. **Railway-Oriented Programming Built-In**
Error handling that's cleaner than Rust's `?`, Go's `if err != nil`, or C++'s exceptions. The `or!` operator creates error handling railways:

```flap
// Clean error propagation - one line per operation
file = open("data.txt") or! "Failed to open file"
data = read(file) or! "Failed to read data"
result = process(data) or! "Failed to process data"

// Compare to Go:
// file, err := os.Open("data.txt")
// if err != nil { return err }
// data, err := ioutil.ReadFile(file)
// if err != nil { return err }
// ... (10+ lines of if err != nil)
```

### 3. **Everything is `map[uint64]float64` - Ultimate Simplicity**
No type system complexity like Rust, no polymorphism confusion like C++, no boxing overhead like Go. One type, infinite flexibility:

```flap
// Same operations work on numbers, strings, lists, maps
len(42)           // 1.0 (number has one element)
len("hello")      // 5.0
len([1,2,3])      // 3.0
len({a: 1, b: 2}) // 2.0

// Everything is indexable
x = 42.5[0]       // 42.5 (number[0] is itself)
x = "hi"[0]       // 104.0 (char code for 'h')
x = [10,20][1]    // 20.0
```

**Bonus:** F-strings, compound assignments, no required semicolons/exit calls, shadowing protection - all the modern niceties without the baggage.

## Language Spec

### Variables

Flap has **three distinct assignment operators** to make mutability and updates explicit:

```flap
// = defines IMMUTABLE variable
x = 10
x <- 20   // ERROR: cannot update immutable variable

// := defines MUTABLE variable
y := 20
y <- y + 5   // ✓ Use <- to update mutable variables
y += 5       // ✓ Compound assignment (also uses <-)

// Immutable variables can be shadowed
x = 10
x = 20    // ✓ Creates new immutable variable (shadows previous)

// Mutable variables CANNOT be shadowed
y := 5
y := 10   // ERROR: variable already defined

// This prevents shadowing bugs in loops
sum := 0
@ i in range(5) {
    sum := sum + i  // ERROR: variable already defined
    sum <- sum + i  // ✓ Correct: use <- to update
}
```

**The Three Operators:**
1. **`=`** - Define/initialize **immutable** variable
   - Can shadow existing immutable variables
   - Cannot shadow mutable variables
   - Cannot be updated with `<-`

2. **`:=`** - Define/initialize **mutable** variable
   - Cannot shadow any existing variable
   - Can be updated with `<-` or compound operators

3. **`<-`** - Update **existing mutable** variable
   - Requires variable to exist
   - Requires variable to be mutable
   - Makes mutations explicit and visible

**Why three operators?**
- Prevents accidental variable shadowing bugs (the #1 cause of logic errors in loops)
- Makes mutability explicit at definition site
- Makes mutations explicit at update site
- Compiler catches common mistakes at compile time

### Constants

Flap supports compile-time constants using an **uppercase naming convention**. Constants are substituted at compile time with zero runtime overhead.

```flap
// Define constants (uppercase identifiers)
PI = 3.14159
SCREEN_WIDTH = 1920
SCREEN_HEIGHT = 1080
MAX_HEALTH = 100

// Use in expressions - substituted at compile time
circumference = 2.0 * PI * 10.0        // PI replaced with 3.14159
pixels = SCREEN_WIDTH * SCREEN_HEIGHT  // Computed at compile time

// Constants can use hex/binary literals
MAX_U8 = 0xFF
BITMASK = 0b11110000

// Useful for game development
player_health = MAX_HEALTH - 25
```

**Constant Rules:**
- Must be all uppercase (e.g., `PI`, `MAX_HEALTH`, `SCREEN_WIDTH`)
- Can be assigned number literals, string literals, or literal lists
- Substituted at parse time (true compile-time constants)
- Zero runtime overhead - values inlined at each use
- Perfect for configuration values, magic numbers, and named constants

**Example with strings and lists:**
```flap
APP_NAME = "MyGame"
VERSION = "1.0.0"
DEFAULT_COLORS = [255, 128, 64]

printf("%s v%s\n", APP_NAME, VERSION)
red = DEFAULT_COLORS[0]
```

### Number Literals

Flap supports decimal, hexadecimal, and binary number literals:

```flap
// Decimal (standard)
x = 255
y = 3.14159

// Hexadecimal (0x prefix)
color = 0xFF00FF      // RGB magenta
mask = 0xDEADBEEF
offset = 0x1000

// Binary (0b prefix)
flags = 0b11110000
permissions = 0b101   // 5 in decimal
```

**Hexadecimal and Binary:**
- Hexadecimal: `0x` or `0X` prefix followed by `[0-9a-fA-F]+`
- Binary: `0b` or `0B` prefix followed by `[01]+`
- Both convert to float64 at compile time
- Useful for bit manipulation, color values, memory addresses
- Current limitation: values should be < 2³¹ due to compiler immediate encoding

### Operators

**Arithmetic:** `+` `-` `*` `/` `%` `**` (power)

**Compound Assignment:** `+=` `-=` `*=` `/=` `%=` (equivalent to `<-`)
```flap
sum := 0
sum += 10     // Equivalent to: sum <- sum + 10
count -= 1    // count = count - 1
value *= 2    // value = value * 2
x /= 3        // x = x / 3
x %= 5        // x = x % 5
```

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

// Shorthand: ~> without -> is equivalent to { -> ~> value }
x > 42 { ~> 123 }    // same as { -> ~> 123 }

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

// F-strings (interpolation with f"..." prefix)
name := "Alice"
age := 30
println(f"Hello, {name}! You are {age} years old.")
println(f"Sum: {a + b}, Product: {a * b}")  // Expressions in {}

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

Maps are **ordered** - they preserve insertion order.

```flap
ages = {1: 25, 2: 30, 3: 35}
empty = {}
count = #ages        // returns 3.0
price = ages[1]      // returns 25.0
missing = ages[999]  // returns 0.0 (key doesn't exist)

// Maps preserve insertion order
@ key, value in ages {
    println(f"{key}: {value}")  // Always prints in order: 1: 25, 2: 30, 3: 35
}
```

#### String Keys and Dot Notation

String identifiers used as keys are **automatically hashed to uint64** at compile time. This enables ergonomic dot notation for field access while maintaining the `map[uint64]float64` data structure.

```flap
// String keys in map literals (identifiers without quotes)
player = {health: 100, x: 10.0, y: 20.0}

// Dot notation for field access (syntax sugar for map indexing)
player.health        // Same as player[hash("health")]
player.x <- 50.0     // Same as player[hash("x")] <- 50.0

// Nested maps with dot notation
player = {
    pos: {x: 100.0, y: 200.0},
    vel: {x: 0.0, y: 0.0},
    health: 100
}

player.pos.x         // Chained access
player.vel.y <- 5.0  // Update nested fields

// Mixed numeric and string keys (both work)
data = {0: "numeric", name: "Alice", age: 30}
data[0]              // Access by numeric key
data.name            // Access by string key
```

**Implementation Details:**
- String keys are hashed at **compile time** using FNV-1a hash algorithm
- Hash values use a 30-bit range (`0x40000000` to `0x7FFFFFFF`) to work within current compiler limitations
- Dot notation (`obj.field`) is syntax sugar that compiles to map indexing (`obj[hash("field")]`)
- At runtime, everything is still `map[uint64]float64` - no new data types
- String keys preserve insertion order just like numeric keys
- Namespaced function calls (`SDL.init()`) are supported through dot notation

### Membership Testing

```flap
10 in numbers {
    -> println("Found!")
    ~> println("Not found")
}

result = 5 in mylist  // returns 1.0 or 0.0
```

### Loops

Loops use `@` for iteration (simplified from `@` in v1.0):

```flap
// Basic loop - iterates from 0 (inclusive) to 5 (exclusive): 0,1,2,3,4
@ i in 5 {
    println(i)
}

// Iterate over range
@ i in range(10) {
    println(i)
}

// Nested loops (auto-labeled @1, @2, @3, ...)
@ i in 3 {       // @1
    @ j in 3 {   // @2
        printf(f"{i},{j} ")
    }
}

// Iterate over lists
numbers = [10, 20, 30]
@ n in numbers {
    println(n)
}

// Range operator ..<
@ i in 0..<3 {   // 0, 1, 2
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
@ item in ["a", "b", "c"] {
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

Lambdas use `=>` arrow syntax (consistent with JavaScript, Rust, C#):

```flap
double = x => x * 2
add = (x, y) => x + y
result = double(5)

// Single parameter doesn't need parentheses
square = x => x * x

// Multiple parameters need parentheses
multiply = (x, y) => x * y

// Lambdas can have match blocks
classify = x => x > 0 {
    -> "positive"
    ~> "non-positive"
}

// Block body for complex logic
process = x => {
    temp := x * 2
    result := temp + 10
    result  // Last expression is return value
}
```

### Builtin Functions

**I/O:**
- `println(x)` - print with newline (syscall-based)
- `printf(fmt, ...)` - formatted print (libc-based)
- `exit(code)` - exit program explicitly (syscall-based)
- `cexit(code)` - exit program explicitly (libc-based)

**Note:** Programs automatically call `exit(0)` at the end if no explicit exit is present

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

loop_statement  = "@" identifier "in" expression block
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

lambda_body             = block | expression [ match_block ] ;

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

* `@` introduces auto-labeled loops. The loop label is the current nesting depth (1, 2, 3, ...).
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
@ i in 1..<101 {
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
    @ x in list {
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
    @ x in list {
        predicate(x) {
            -> result := result + x
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

## Testing Convention

Flap uses a simple, file-based testing convention for writing and running tests.

### Test File Naming

Test files follow the pattern `test_*.flap` and should be placed in a `tests/` directory or alongside the code they test.

### Test Structure

Each test is a separate Flap program that:
1. **Returns 0 on success** - The program should exit with code 0 when all tests pass
2. **Returns non-zero on failure** - Any non-zero exit code indicates test failure
3. **Prints descriptive messages** - Use `printf` or `println` to describe what failed

### Basic Test Example

```flap
// tests/test_math.flap
// Test basic arithmetic operations

// Test addition
result := 2 + 2
result != 4 {
    -> printf("FAIL: 2 + 2 expected 4, got %v\n", result) :: exit(1)
}

// Test multiplication
result := 3 * 4
result != 12 {
    -> printf("FAIL: 3 * 4 expected 12, got %v\n", result) :: exit(1)
}

println("PASS: All tests passed")
exit(0)
```

### Test Helper Functions

Create reusable assertion functions in your test files:

```flap
// Assert that two values are equal
assert_eq = actual, expected, message => actual != expected {
    -> printf("FAIL: %s\n  Expected: %v\n  Got: %v\n", message, expected, actual) :: exit(1)
}

// Assert that a condition is yes (non-zero)
assert = condition, message => condition == 0 {
    -> printf("FAIL: %s\n", message) :: exit(1)
}

// Usage
assert_eq(square(5), 25, "square(5) should equal 25")
assert(5 > 3, "5 should be greater than 3")
```

### Running Tests

Tests can be run individually or in batch:

```bash
# Run a single test
./flapc tests/test_math.flap && ./test_math
echo "Exit code: $?"

# Run all tests in a directory
for test in tests/test_*.flap; do
    name=$(basename "$test" .flap)
    ./flapc "$test" && ./"$name" && echo "✓ $name" || echo "✗ $name"
done
```

### Package Testing Convention

For packages (like flap_math, flap_core):
1. Place tests in a `tests/` subdirectory
2. Each test should import the package and test its public functions
3. Use descriptive test names that indicate what is being tested

```flap
// tests/test_sum.flap
import "github.com/xyproto/flap_core" as core

assert_eq = actual, expected => actual != expected {
    -> printf("Expected %v, got %v\n", expected, actual) :: exit(1)
}

// Test sum with positive numbers
assert_eq(core.sum([1, 2, 3, 4]), 10)

// Test sum with empty list
assert_eq(core.sum([]), 0)

// Test sum with negative numbers
assert_eq(core.sum([-1, -2, -3]), -6)

println("PASS: sum tests")
exit(0)
```

## Module System

Flap supports both explicit imports and automatic dependency resolution.

### Explicit Imports

```flap
// Import with namespace
import "github.com/xyproto/flap_math" as math
result := math.square(5)

// Import with wildcard (into same namespace)
import "github.com/xyproto/flap_core" as *
filtered := filter((x) -> x > 2, [1, 2, 3, 4])

// Version specification
import "github.com/xyproto/flap_math@v1.0.0" as math
import "github.com/xyproto/flap_math@latest" as math
import "github.com/xyproto/flap_math@HEAD" as math
```

### Private Functions

Functions and variables starting with `_` are private and not exported:

```flap
// Public function (exported)
square = (x) -> x * x

// Private helper (not exported)
_validate = (x) -> x > 0 { -> ~> 1 ~> 0 }

// Only square() is available when imported
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
