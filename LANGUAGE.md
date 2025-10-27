# The Flap Programming Language

### Version 1.6

## Type System

Flap uses `map[uint64]float64` as its unified type representation:

- **Numbers**: `{0: 42.0}`
- **Strings**: `{0: 72.0, 1: 101.0, ...}` (index → character code)
- **Lists**: `{0: 1.0, 1: 2.0, ...}` (index → element val)
- **Maps**: Direct representation
- **Functions**: Pointers stored as float64 values

All values use IEEE 754 double-precision floating point. This single underlying type enables uniform operations and consistent SIMD optimization.

### Result Type

All expressions and functions in Flap implicitly return a **Result** type, which represents either success with val(s) or failure with an err.

**Key Properties:**
- A Result contains **either** vals **or** an err string, never both
- Vals can be zero, one, or multiple (returned as a list)
- The `ret` keyword returns a success result
- The `err` keyword returns an error result
- Pattern matching uses `->` for success (extracts vals) and `~>` for error (extracts err)
- The `or!` operator propagates errs automatically
- No explicit constructors - use `ret`/`err` keywords only

**Examples:**

```flap
// Function returning a single val
divide = (a, b) => {
    b == 0 { err "division by zero" }
    ret a / b
}

// Pattern match on Result
result := divide(10, 2)
result {
    -> val { printf("Result: %v\n", val) }
    ~> err { printf("Error: %v\n", err) }
}

// Multiple return values
parse_coords = text => {
    valid { ret x, y, z }
    ~> err "invalid coordinates"
}

coords := parse_coords("1,2,3")
coords {
    -> x, y, z { printf("x=%v, y=%v, z=%v\n", x, y, z) }
    ~> err { printf("Parse err: %v\n", err) }
}

// Err propagation with or!
safe_divide = (a, b) => {
    result := divide(a, b) or! 0  // Returns 0 if err
    ret result * 2
}

// Loops can return Results
find_item = items => @ i in items {
    i > 100 { ret @ i }  // Success: returns i
    i < 0 { err @ "negative val found" }  // Error with message
}
```

**Benefits:**
- **Explicit err handling**: All errs must be handled or propagated
- **No null/undefined**: Missing vals are explicit Err results
- **Multiple returns**: Functions can return multiple vals naturally
- **Railway-oriented**: `or!` chains operations that might fail
- **Type safety**: Cannot accidentally use err as val

## Compilation

- **Direct code generation**: x86-64, ARM64, RISC-V machine code (no LLVM, no GCC)
- **Binary format**: ELF64 (Linux), Mach-O (macOS)
- **No runtime system**: No garbage collector, no bytecode interpreter
- **SIMD optimization**: Runtime CPUID detection selects SSE2 or AVX-512
- **FFI**: PLT/GOT dynamic linking for C library calls
- **Calling convention**: System V AMD64 ABI

## Design Philosophy

**Avoid Magic Numbers**: Flap prefers explicit keywords and proper types over magic numbers like `-1` for special values:
- ❌ Use `-1` for "infinite", "err", or "missing"
- ✓ Use `inf` keyword for infinite iterations/vals
- ✓ Use explicit err handling (match expressions, err types)
- ✓ Use optional types or nullable representations for missing vals

This makes code more readable and prevents confusion between legitimate negative values and special sentinel values.

## Language Spec

### Variables

Flap has **five distinct assignment operators** to make mutability and updates explicit:

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
@ i in 0..<5 {
    sum := sum + i  // ERROR: variable already defined
    sum <- sum + i  // ✓ Correct: use <- to update
}
```

**The Five Operators:**
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

4. **`=?`** - Define/initialize **immutable** with error propagation
   - If right side is err, return from function with that err
   - If right side has val, assign to immutable variable
   - Railway-oriented assignment for immutable values

5. **`<-?`** - Update **mutable** variable with error propagation
   - If right side is err, return from function with that err
   - If right side has val, update mutable variable
   - Railway-oriented assignment for mutable values

**Why five operators?**
- Prevents accidental variable shadowing bugs (the #1 cause of logic errors in loops)
- Makes mutability explicit at definition site
- Makes mutations explicit at update site
- `=?` and `<-?` eliminate boilerplate error handling
- Compiler catches common mistakes at compile time

**Note on Arrow Symbols:**
Flap uses three distinct arrow operators with different meanings:
- `<-` (left arrow) - Update mutable variable/register: `x <- x + 1`
- `->` (right arrow) - Match result/consequence: `condition { -> result }`
- `~>` (squiggly arrow) - Default/else case: `condition { -> if_true ~> if_false }`

These arrows point in different directions and have completely different purposes, so they're easy to distinguish in context.

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

### Unsafe Blocks (Direct Register Access)

Flap provides `unsafe` blocks for architecture-specific code that requires direct register manipulation. This enables low-level optimization while maintaining portability through architecture-specific implementations.

```flap
// unsafe requires all three architecture blocks
result := unsafe {
    // x86_64 block
    rax <- 42
    rbx <- rax
    rax <- rbx
} {
    // arm64 block
    x0 <- 42
    x1 <- x0
    x0 <- x1
} {
    // riscv64 block
    a0 <- 42
    a1 <- a0
    a0 <- a1
}

// The result is the val in the accumulator register:
// x86_64: rax, arm64: x0, riscv64: a0
printf("Result: %.0f\n", result)  // Output: 42

// Useful for bit manipulation and low-level operations
flags := unsafe {
    rax <- 0xFF
    rcx <- 0b11110000
    rax <- rcx
} {
    x0 <- 0xFF
    x1 <- 0b11110000
    x0 <- x1
} {
    a0 <- 0xFF
    a1 <- 0b11110000
    a0 <- a1
}
```

**Unsafe Block Rules:**
- All three architecture blocks are **required** (x86_64, arm64, riscv64)
- Only register-to-register and immediate-to-register moves are supported
- Immediates can be decimal, hex (`0xFF`), or binary (`0b11110000`)
- The compiler selects the appropriate block for the target architecture
- The return val is the accumulator register (rax/x0/a0) converted to float64
- Use for: low-level bit manipulation, custom SIMD operations, performance-critical paths

**Common x86_64 Registers:** rax, rbx, rcx, rdx, rsi, rdi, r8-r15
**Common ARM64 Registers:** x0-x30
**Common RISC-V Registers:** a0-a7, t0-t6

### Operators

**Arithmetic:** `+` `-` `*` `/` `%` `**` (power)

```flap
// Basic arithmetic
x + y    // Addition
x - y    // Subtraction
x * y    // Multiplication
x / y    // Division
x % y    // Modulo (remainder)
x ** y   // Power (exponentiation)

// Power operator examples
area := radius ** 2              // Square
volume := side ** 3              // Cube
distance := (dx ** 2 + dy ** 2) ** 0.5  // Euclidean distance (sqrt)
growth := principal * (1 + rate) ** years  // Compound interest
```

**Compound Assignment:** `+=` `-=` `*=` `/=` `%=` `**=` (equivalent to `<-`)
```flap
sum := 0
sum += 10     // Equivalent to: sum <- sum + 10
count -= 1    // count <- count - 1
val *= 2      // val <- val * 2
x /= 3        // x <- x / 3
x %= 5        // x <- x % 5
x **= 2       // x <- x ** 2 (square x)
```

**Comparison:** `<` `<=` `>` `>=` `==` `!=`

**Logical:** `and` `or` `xor` `not`

Logical operators provide short-circuit evaluation:
```flap
// and - short-circuit AND (returns 0 if left is false, else right value)
x > 0 and x < 100    // Only evaluates x < 100 if x > 0 is true
valid and process()  // Only calls process() if valid is true

// or - short-circuit OR (returns left if true, else right value)
x < 0 or x > 100     // Only evaluates x > 100 if x < 0 is false
cache or compute()   // Only calls compute() if cache is false/zero

// Common patterns
has_value and { do_something() }  // Execute block only if condition is true
get_cached() or { expensive_computation() }  // Compute only if cache miss

// not - logical negation
not ready  // Returns 1 if ready is 0, else 0
not (x > 10)
```

Note: For bitwise AND/OR, use `&b` and `|b` instead.

**Bitwise:** `&b` `|b` `^b` `~b` (operate on an integer representation of the float)

**Shifts:** `<b` `>b` (shift left/right), `<<b` `>>b` (rotate left/right)

**Special:**
- `++` (add with carry) - Multi-precision arithmetic, works in safe and unsafe blocks
  ```flap
  // Safe mode: automatic carry handling
  low := 0xFFFFFFFFFFFFFFFF  // Max uint64
  high := 0x1
  result_low := low ++ 1      // Wraps to 0, sets carry
  result_high := high ++ 0    // Becomes 2 due to carry
  ```
- `<->` (swap) - Exchange two values, works in safe and unsafe blocks
  ```flap
  // Safe mode: swap variables
  a := 10
  b := 20
  a <-> b  // Now: a=20, b=10
  ```

**Pipeline:**
- `|` (functional composition: `x | f | g` ≡ `g(f(x))`)
- `||` (parallel piping)
- `|||` (reducing, map-filter-reduce)

**List:** `^` (head), `&` (tail), `#` (length), `::` (cons/prepend)

List operators for common operations:
```flap
numbers = [1, 2, 3]

// # - length
len := #numbers       // 3

// ^ - head (first element)
first := ^numbers     // 1

// & - tail (all but first)
rest := &numbers      // [2, 3]

// :: - cons (prepend element to list)
new_list := 0 :: numbers      // [0, 1, 2, 3]
item :: existing_list         // Prepend item

// Common patterns
@ item :: rest in list {
    // Pattern match with cons in loops
}

// Building lists functionally
build_list := (n) => n <= 0 {
    -> []
    ~> n :: build_list(n - 1)  // Prepend n to recursive result
}
```

**Err handling:**
- `or!` (railway-oriented programming / err propagation)
- `and!` (success handler - executes if left has val, not err)
- `err?` (check if expression is an err)
- `val?` (check if expression has val)

**Control flow:** `ret` (return val from function/lambda), `err` (return err from function/lambda)

**Type Casting:** `as` (convert between Flap and C types for FFI)
- To C: `as i8`, `as i16`, `as i32`, `as i64` (signed integers)
- To C: `as u8`, `as u16`, `as u32`, `as u64` (unsigned integers)
- To C: `as f32`, `as f64` (floating point)
- To C: `as cstr` (C null-terminated string)
- To C: `as cptr` (C pointer)
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

// Match without default (implicit 0.0/no)
x > 42 {
    -> 123           // sugar for "-> 123 ~> 0"
}

// Match without default, without the arrow (implicit 0)
x > 42 {
    123           // sugar for "-> 123"
}

// Default-only (preserves condition val when true)
x > 42 {
    ~> 123           // yields 1.0/yes when true, 123 when false
}

// Shorthand: ~> without -> is equivalent to { -> ~> val }
x > 42 { ~> 123 }    // same as { -> ~> 123 }

// Super-shorthand: Single case + default without braces
x > 2 -> 42 ~> 128   // same as { -> 42 ~> 128 }
                     // Only valid for exactly one case + default

// Subject/guard matching
x {
    x < 10 -> 0
    x < 20 -> 1
    ~> 2
}

// Ternary replacement
z = x > 42 { 1 ~> 0 }

// Ternary replacement with yes/no
z = x > 42 { yes ~> no }
```

### Strings

```flap
s := "Hello"         // Creates {0: 72.0, 1: 101.0, ...}

// Indexing returns Result (err if out of bounds)
s[1] {
    -> val { printf("Char: %v\n", val) }  // val = 101.0 (UTF-8 'e')
    ~> err { printf("Out of bounds\n") }
}

char := s[1] or! 0   // 101.0 (in bounds) or 0 if out of bounds

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
length = #numbers    // length operator
head = ^numbers      // first element
tail = &numbers      // all but first

// Indexing returns Result (err if out of bounds)
numbers[0] {
    -> val { printf("First: %v\n", val) }  // val = 1
    ~> err { printf("Err: %v\n", err) }
}

numbers[999] {
    -> val { printf("Found: %v\n", val) }
    ~> err { printf("Out of bounds!\n") }  // This executes
}

// With or! for default values
first := numbers[0] or! 0   // 1 (in bounds)
missing := numbers[999] or! -1  // -1 (out of bounds, returns err, uses default)

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

// Indexing returns Result (err if key doesn't exist)
ages[1] {
    -> val { printf("Age: %v\n", val) }  // val = 25
    ~> err { printf("Key not found\n") }
}

ages[999] {
    -> val { printf("Age: %v\n", val) }
    ~> err { printf("Key 999 not found!\n") }  // This executes
}

// With or! for default values
age := ages[1] or! 0      // 25 (key exists)
missing := ages[999] or! 0  // 0 (key doesn't exist, returns err, uses default)

// Maps preserve insertion order
@ key, val in ages {
    println(f"{key}: {val}")  // Always prints in order: 1: 25, 2: 30, 3: 35
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
- Namespaced function calls (`sdl.SDL_init()`) are supported through dot notation

**Err Handling with Dot Notation:**

When using the `.` operator, if the field doesn't exist or the left side is an err:

```flap
// Accessing non-existent field returns err
player = {health: 100, x: 10}
result := player.asdf  // Returns err: "asdf is not a member of player"

// If left side is already an err, dot operator propagates with new message
x := err "something went wrong"
result := x.field  // Returns err: "field is not a member of x"

// Handle with pattern matching
player.health {
    -> val { printf("Health: %v\n", val) }
    ~> err { printf("Err: %v\n", err) }
}
```

This ensures that field access errs are explicit and can be handled through the Result type system.

### Membership Testing

```flap
10 in numbers {
    -> println("Found!")
    ~> println("Not found")
}

println(10 in numbers { "Found!" ~> "Not found" })

result = 5 in mylist  // returns 1.0 or 0.0
```

### Loops

**The `@` symbol in Flap is loop-related**: it's used for loop iteration, loop control flow, and accessing loop state.

Loops use `@` for iteration, with optional CPU parallelism using a numeric prefix:

```flap
// Sequential loop (default) - max is OPTIONAL (inferred as 5)
@ i in 0..<5 {
    println(i)  // Prints 0, 1, 2, 3, 4
}

// Parallel loop with 4 cores
4 @ i in 0..<1000 {
    compute(i)  // Executes in parallel across 4 CPU cores
}

// Parallel loop with all available cores
cpu_count() @ i in data max 100000 {
    process(i)  // Uses all CPU cores for parallel execution
}

// Range operator - max inferred from literal bounds
@ i in 1..<10 {
    println(i)  // Prints 1, 2, 3, ..., 9
}

// Nested loops (auto-labeled @1, @2, @3, ...)
@ i in 0..<3 {       // @1 - max inferred as 3
    @ j in 0..<3 {   // @2 - max inferred as 3
        printf(f"{i},{j} ")
    }
}

// Explicit max for safety bounds or variable ranges
@ i in 0..<100 max 1000 {
    println(i)  // Runtime check: will err if somehow exceeds 1000 iterations
}

// Variable ranges REQUIRE explicit max
end := 100
@ i in 0..<end max 10000 {
    println(i)
}

// For unbounded range loops, use 'max inf'
@ i in 0..<1000000 max inf {
    println(i)
    i == 100 { ret @ }  // Usually exits via some condition
}

// List iteration with inferred max (when literal)
@ n in [10, 20, 30] {
    println(n)
}

// Variable list iteration requires explicit max
numbers := [10, 20, 30]
@ n in numbers max 1000 {
    println(n)
}

// Infinite loop (no iterator, no max needed)
@ {
    printf("Looping...\n")
    condition { ret @ }  // Exit loop with ret @
}

// Alternative: infinite game loop
@ {
    handle_input()
    update_game()
    render()
}
```

**Max Iteration Safety:**
- Literal ranges (`0..<5`) and list literals: `max` is **optional** (inferred, no runtime overhead)
- Variable ranges/lists: `max` is **required** (runtime checking enforced)
- Explicit `max N`: adds runtime checking to ensure loop doesn't exceed N iterations
- `max inf`: allows unlimited iterations for ranges/lists (use cautiously!)
- Infinite loops (`@ {}`): no iterator, no `max` needed - truly infinite until explicit exit

**CPU Parallelism:**
- Sequential loop: `@ item in collection` (default, single core)
- Parallel loop: `N @ item in collection` (uses N CPU cores)
- Optional prefix: number of cores to use (e.g., `4 @`, `8 @`)
- Dynamic cores: `cpu_count() @ item in list` uses all available cores
- Use for: data processing, image processing, physics simulations, ray tracing
- **Note:** Parallel loops require thread-safe operations (no shared mutable state without synchronization)

**Example:**
```flap
// Sequential processing
@ pixel in pixels max 1000000 {
    process_pixel(pixel)
}

// Parallel processing with 8 cores (8x faster)
8 @ pixel in pixels max 1000000 {
    process_pixel(pixel)  // Executed across 8 threads
}
```

**Loop Control:**
- `ret` - returns from function with val
- `ret val` - returns val from function
- `ret @1`, `ret @2`, `ret @3`, ... - exits loop at nesting level 1, 2, 3, ... and all inner loops
- `ret @1 val` - exits loop and returns val
- `err "message"` - returns err from function (for err handling)
- `@++` - skip this iteration and continue to next (current loop)
- `@1++`, `@2++`, `@3++`, ... - continue to next iteration of loop at nesting level 1, 2, 3, ...

**Loop Variables:**
- `@first` - true on first iteration
- `@last` - true on last iteration
- `@counter` - iteration count (starts at 0)
- `@i` - current element val (same as loop variable)

**Example:**
```flap
@ item in ["a", "b", "c"] {
    @first { printf("[") }
    printf("%v", item)
    @last { printf("]") ~> printf(", ") }
}
// Output: [a, b, c]
// Note: max is optional for list literals
```

### Err Handling

Flap uses Result types and pattern matching for explicit, type-safe err handling. All functions and expressions return Results containing either val(s) or an err.

#### Pattern Matching on Results

The primary way to handle Results is through pattern matching with `->` (success) and `~>` (err):

```flap
// Function that returns Result
parse_int = text => {
    is_number(text) { ret string_to_int(text) }
    ~> err "not a valid integer"
}

// Handle Result with pattern matching
result := parse_int("42")
result {
    -> val { printf("Parsed: %v\n", val) }
    ~> err { printf("Err: %v\n", err) }
}

// Multiple return vals
read_user = id => {
    valid { ret name, age, email }
    ~> err "user not found"
}

user := read_user(123)
user {
    -> name, age, email { printf("%v (%v): %v\n", name, age, email) }
    ~> err { printf("Failed: %v\n", err) }
}
```

#### Err Propagation with `or!`

The `or!` operator provides automatic err propagation (railway-oriented programming). If the left side is an err, the right side determines what happens:

- **String**: Replace err message and propagate: `operation() or! "custom err"`
- **Val**: Return val as default: `operation() or! 0`
- **Block**: Execute block (usually exits): `operation() or! { exit(1) }`

```flap
// Chain operations that might fail
process_file = filename => {
    file := open(filename) or! "cannot open file"
    data := read(file) or! "cannot read data"
    result := parse(data) or! "cannot parse data"
    ret result
}

// or! with default vals
safe_divide = (a, b) => {
    result := divide(a, b) or! 0  // Returns 0 if err
    ret result
}

// or! propagates errs up the call chain
main ==> {
    result := process_file("data.txt") or! {
        printf("File processing failed\n")
        exit(1)
    }
    printf("Success: %v\n", result)
}
```

#### Err Propagation with `=?` and `<-?`

The `=?` and `<-?` operators combine assignment with error propagation - if the right side is an err, the function returns with that err immediately:

```flap
// Without =?: verbose error handling
process_data = input => {
    result1 := parse(input)
    result1.err? {
        err result1  // Must explicitly return error
    }
    data = result1  // Extract value

    result2 := validate(data)
    result2.err? {
        err result2
    }
    validated = result2

    ret validated
}

// With =?: concise and clean
process_data = input => {
    data =? parse(input)       // If err, returns from function
    validated =? validate(data)  // If err, returns from function
    ret validated
}

// With mutable variables using <-?
update_state = new_val => {
    state := get_state()
    state <-? validate(new_val)  // If err, returns from function
    state <-? transform(state)   // If err, returns from function
    save_state(state)
}

// Mix with or! for defaults
load_config = () => {
    port =? read_port() or! 8080     // Use default if err
    host =? read_host() or! "localhost"
    timeout =? read_timeout()  // Propagate err (no default)
    ret {port: port, host: host, timeout: timeout}
}
```

**When to use:**
- Use `=?` / `<-?` when you want to propagate errors immediately
- Use `or!` when you want to provide defaults or custom handling
- `=?` / `<-?` is like Rust's `?` operator but integrated into assignment

#### Success Handler `and!`

The `and!` operator executes the right side if the left side has vals (not an err). This enables "happy path" continuations:

```flap
// Execute block if operation succeeds
parse_config("config.json") and! {
    printf("Config loaded successfully\n")
    start_server()
}

// Chain success handlers
load_data() and! process_data() and! save_results()

// Use with pattern matching
result := fetch_user(123)
result and! -> user { printf("Welcome, %v!\n", user.name) }

// Combined with or! for complete handling
operation()
    and! { printf("Success!\n") }
    or! { printf("Failed!\n") }
```

**Behavior:**
- If left side is a success (has vals), execute right side
- If left side is an err, skip right side and propagate err
- Complement to `or!`: `and!` handles success, `or!` handles err

#### Checking Results: `err?` and `val?`

The `err?` and `val?` operators test whether an expression is an err or has val(s):

```flap
// Check if something is an err
result := parse_int("not a number")
result.err? {
    printf("Got an err: %v\n", result)  // Prints error message
}

// Check if something has val
result := parse_int("42")
result.val? {
    printf("Successfully parsed!\n")
}

// Use in conditionals
operation() {
    val? -> printf("Success!\n")
    err? -> printf("Failed!\n")
}

// Common pattern: handle both cases
result.val? {
    result {
        -> val { process(val) }
    }
}
result.err? {
    result {
        ~> err { log_err(err) }
    }
}

// With or! for compact code
parse_int(input).val? {
    data := parse_int(input) or! 0  // We know it's safe now
    process(data)
}
```

**Behavior:**
- `expr.err?` returns true (1.0) if expr is an err, false (0.0) otherwise
- `expr.val?` returns true (1.0) if expr has val(s), false (0.0) otherwise
- These are complementary: exactly one is always true
- More concise than pattern matching when you just need to check

#### Loop Err Handling

Loops can return Results, allowing early exit with val or err:

```flap
// Exit loop with val
find_first = items => @ i in items {
    i > 100 { ret @ i }  // Returns success with i
}

// Exit loop with err
validate_all = items => @ i in items {
    i < 0 { err @ "negative val found" }
    i > 1000 { err @ "val too large" }
}

// Handle loop result
result := find_first([1, 50, 150, 200])
result {
    -> val { printf("Found: %v\n", val) }
    ~> err { printf("Not found\n") }
}
```

#### Err Handling Patterns

**1. Explicit Handling (pattern matching)**
```flap
result := risky_operation()
result {
    -> val { process(val) }
    ~> err { log_err(err) }
}
```

**2. Propagation (railway-oriented)**
```flap
chained_operation = () => {
    a := step1() or! "step1 failed"
    b := step2(a) or! "step2 failed"
    c := step3(b) or! "step3 failed"
    ret c
}
```

**3. Default Vals**
```flap
val := unsafe_operation() or! default_val
```

**4. Panic on Err**
```flap
val := must_succeed() or! {
    printf("Fatal err\n")
    exit(1)
}
```

**Benefits:**
- **Explicit**: All errs must be handled or propagated
- **Type-safe**: Cannot accidentally use err as val
- **No exceptions**: No hidden control flow or stack unwinding
- **Composable**: `or!` chains operations naturally
- **Railway-oriented**: Success path stays clean and linear

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

// Zero parameters use () => or shorthand ==>
main = () => {
    println("Hello, World!")
}

// Shorthand ==> syntax (equivalent to = () =>)
main ==> {
    println("Hello, World!")
}

// Lambdas can have match blocks
classify = x => x > 0 {
    -> "positive"
    ~> "non-positive"
}

// Block body for complex logic
process = x => {
    temp := x * 2
    result := temp + 10
    result  // Last expression is return val
}
```

#### Piping Blocks to Functions

For callback-heavy APIs, use the pipe operator to pass blocks as arguments:

```flap
// Instead of: func(() => { block })
// Use pipe: { block } | func()

// Traditional style (explicit lambda)
app.key_down("space", () => { player <- jump(player) })

// Pipe style (cleaner, functional)
{ player <- jump(player) } | app.key_down("space")

// The block flows into the function as the last argument
// This makes the data flow explicit and clear

// Multiple handlers
{ initialize() } | on_start
{ cleanup() } | on_stop

// With arguments
{ process(data) } | handler(timeout: 1000)
```

**How it works:**
- The pipe operator `|` can take a block `{ ... }` on the left
- The block is automatically wrapped as `() => { ... }`
- It becomes the last argument to the function on the right
- Makes callback flow explicit: data → function

**Common patterns:**
```flap
// Event handlers (FLAPGAME style)
{ update_game_state() } | app.on_update
{ draw_scene() } | app.on_render

// Input handling
{ player.y <- player.y - speed } | app.key_down("w")
{ player.y <- player.y + speed } | app.key_down("s")
{ shoot(mouse_x, mouse_y) } | app.mouse_click

// Conditional execution (using match, not pipe)
ready {
    -> { do_work() }
    ~> { do_alternative() }
}

// Loop with action
@ running {
    process_frame()
    should_quit { ret @ }
}

// Chained pipes
data | transform | validate | { save(it) } | on_complete
```

**Why pipe instead of suffix blocks:**
- ✅ Clear data flow direction (left to right)
- ✅ Consistent with functional composition
- ✅ No ambiguity with match expressions
- ✅ Explicit about what's being passed
- ✅ Works naturally with Flap's existing pipe operator

### Automatic Recursion Optimization

Flap automatically optimizes recursive function calls without requiring special keywords:

**Tail-Call Optimization (Automatic):**

When a function calls itself as the last operation (tail position), the compiler automatically converts it to a loop:

```flap
// Fibonacci using tail recursion with accumulator pattern
fib := (n, a, b) => n <= 0 {
    -> a
    ~> fib(n - 1, b, a + b)  // Tail call - auto-optimized to loop
}

println(fib(10, 0, 1))  // 55 (no stack growth)
```

**Automatic Memoization (Pure Functions):**

Pure recursive functions (no side effects) are automatically memoized by the compiler:

```flap
// Fibonacci with automatic memoization (pure function)
fib := n => n <= 1 {
    -> n
    ~> fib(n - 1) + fib(n - 2)  // Auto-memoized (pure function)
}

println(fib(10))   // First call builds cache
println(fib(20))   // Reuses cached results (very fast)
```

**How it works:**
- **Tail calls**: Compiler detects when function calls itself in tail position and converts to loop (no stack growth)
- **Pure functions**: Compiler analyzes function for side effects; if pure, adds automatic result caching
- **Zero overhead**: No special syntax required, optimizations happen automatically
- **Smart caching**: Memoization uses arena-based memory allocation

**Note:** Recursive calls use the function name directly. There is no special `me` or `cme` keyword.

### Builtin Functions

**I/O:**
- `println(x)` - print with newline (syscall-based)
- `printf(fmt, ...)` - formatted print (libc-based)
- `exit(code)` - exit program explicitly (syscall-based)
- `cexit(code)` - exit program explicitly (libc-based)

**Note:** Programs automatically exit with code 0 if no explicit exit is present

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
- `%v` - smart val (42.0→"42", 3.14→"3.14")
- `%b` - boolean (0.0→"no", non-zero→"yes")
- `%f` - float
- `%d` - integer
- `%s` - string

**Math:** (using native x87 FPU or SSE2, or C library via FFI)
- `sqrt(x)`, `abs(x)`, `floor(x)`, `ceil(x)`, `round(x)`
- `sin(x)`, `cos(x)`, `tan(x)`
- `asin(x)`, `acos(x)`, `atan(x)`
- `log(x)`, `exp(x)`, `pow(x, y)`

**Note:** Math functions from C libraries (like libm) use proper floating-point calling convention (xmm registers) for accurate results.

## The Unsafe Language

Flap provides `unsafe` blocks for direct register manipulation when you need architecture-specific optimizations. Unlike inline assembly in other languages, Flap's unsafe blocks maintain portability by requiring implementations for all three target architectures.

### Philosophy

Unsafe blocks exist for three reasons:

1. **Performance-critical paths** - Direct register control for hot loops, SIMD operations, or bit manipulation
2. **Low-level operations** - Tasks that require precise control over CPU registers
3. **Maintained portability** - By requiring all three architectures, code remains cross-platform

### Syntax

```flap
result := unsafe {
    // x86_64 block
    register_ops
} {
    // arm64 block
    register_ops
} {
    // riscv64 block
    register_ops
}
```

All three blocks are **mandatory**. The compiler selects the appropriate block for the target architecture at compile time.

### Operations

Unsafe blocks support three operations:

#### 1. Immediate to Register
```flap
rax <- 42          // Load immediate val
rcx <- 0xFF        // Hex literals work
rdx <- 0b1010      // Binary literals work
```

#### 2. Register to Register
```flap
rax <- rbx         // Copy rbx to rax
x0 <- x1          // ARM64 equivalent
a0 <- a1          // RISC-V equivalent
```

#### 3. Stack Operations
```flap
stack <- rax       // Push rax onto stack
rbx <- stack       // Pop from stack into rbx
```

Stack operations follow LIFO (Last In, First Out) order:
```flap
unsafe {
    rax <- 10
    stack <- rax      // Push 10
    rax <- 20
    stack <- rax      // Push 20

    rbx <- stack      // Pop 20 into rbx
    rcx <- stack      // Pop 10 into rcx
} { /* arm64 */ } { /* riscv64 */ }
```

#### 4. Arithmetic Operations

**Addition and Subtraction** (v1.3.0+):
```flap
unsafe {
    rax <- 100
    rbx <- 50
    rax <- rax + rbx   // rax = 150 (register + register)
    rax <- rax + 10    // rax = 160 (register + immediate)
    rax <- rax - 20    // rax = 140 (register - immediate)
    rax <- rax - rbx   // rax = 90  (register - register)
} { /* arm64 */ } { /* riscv64 */ }
```

**Add with Carry** - Multi-precision arithmetic:
```flap
unsafe {
    // 128-bit addition: add two pairs of 64-bit values
    rax <- low1 + low2      // Add lower 64 bits (sets carry flag)
    rdx <- high1 ++ high2   // Add upper 64 bits with carry
} { /* arm64 */ } { /* riscv64 */ }
```

**Exchange/Swap** - Atomic val exchange:
```flap
unsafe {
    rax <- 100
    rbx <- 200
    rax <-> rbx            // Swap: rax=200, rbx=100

    rax <-> [rcx]          // Swap rax with memory at [rcx]
} { /* arm64 */ } { /* riscv64 */ }
```

**Coming in v1.5.0**: Multiply, divide, bitwise operations (AND, OR, XOR), shifts

#### 5. Memory Operations

**Memory Loads** (v1.3.0+):
```flap
unsafe {
    rbx <- 0x1000        // Address to load from
    rax <- [rbx]         // Load 64-bit val from address in rbx
    rcx <- [rbx + 16]    // Load from rbx + 16 offset
} { /* arm64 */ } { /* riscv64 */ }
```

**Sized Memory Loads** (v1.5.0+):
```flap
unsafe {
    rbx <- 0x1000

    // Zero-extend smaller types (unsigned)
    rax <- [rbx] as uint8    // Load byte, zero-extend to 64-bit (0x00000000000000FF)
    rax <- [rbx] as uint16   // Load word, zero-extend to 64-bit (0x000000000000FFFF)
    rax <- [rbx] as uint32   // Load dword, zero-extend to 64-bit (0x00000000FFFFFFFF)

    // Sign-extend smaller types (signed)
    rax <- [rbx] as int8     // Load byte, sign-extend to 64-bit (0xFFFFFFFFFFFFFFFF for -1)
    rax <- [rbx] as int16    // Load word, sign-extend to 64-bit
    rax <- [rbx] as int32    // Load dword, sign-extend to 64-bit

    // Works with offsets too
    rax <- [rbx + 8] as uint8
} { /* arm64 */ } { /* riscv64 */ }
```

**Sized Memory Stores** (v1.5.0+):
```flap
unsafe {
    rbx <- 0x1000

    // Store byte/word/dword from register
    [rbx] <- rax as uint8     // Store low byte of RAX to [rbx]
    [rbx] <- rax as uint16    // Store low word of RAX to [rbx]
    [rbx] <- rax as uint32    // Store low dword of RAX to [rbx]
    [rbx] <- rax as uint64    // Store full 64-bit RAX to [rbx] (default)

    // Works with offsets too
    [rbx + 8] <- rax as uint8

    // Note: Signed types (int8, int16, int32) are treated the same as unsigned for stores
    // The signedness only matters when loading back with sign extension
} { /* arm64 */ } { /* riscv64 */ }
```

### Return Val

Unsafe blocks return the val in the **accumulator register**:
- **x86_64**: `rax`
- **ARM64**: `x0`
- **RISC-V**: `a0`

The val is automatically converted from integer to `float64` (Flap's native type).

```flap
result := unsafe {
    rax <- 42
    rbx <- 100
    rax <- rbx    // rax = 100
} { x0 <- 100 } { a0 <- 100 }

// result is 100.0 (converted to float64)
```

### Common Registers

**x86_64**:
- General purpose: `rax`, `rbx`, `rcx`, `rdx`, `rsi`, `rdi`
- Extended: `r8`, `r9`, `r10`, `r11`, `r12`, `r13`, `r14`, `r15`
- Stack pointer: `rsp` (use with caution)
- Base pointer: `rbp` (use with caution)

**ARM64**:
- General purpose: `x0`-`x30`
- Stack pointer: `sp` (use with caution)
- Zero register: `xzr` (reads as 0)

**RISC-V**:
- Arguments/results: `a0`-`a7`
- Temporaries: `t0`-`t6`
- Saved: `s0`-`s11`
- Stack pointer: `sp` (use with caution)

### Examples

#### Example 1: Bit Manipulation
```flap
// Swap two vals using XOR trick
swapped := unsafe {
    rax <- 42
    rbx <- 100
    rax <- rax      // XOR rax with rbx (not yet implemented)
} {
    x0 <- 42
    x1 <- 100
    // ARM64 implementation
} {
    a0 <- 42
    a1 <- 100
    // RISC-V implementation
}
```

#### Example 2: Stack-based Calculation
```flap
// Calculate: (10 + 20) * 30 using stack
result := unsafe {
    rax <- 10
    stack <- rax
    rax <- 20
    stack <- rax

    // Pop and add (simplified - add instruction not yet supported)
    rbx <- stack
    rcx <- stack
    rax <- 30
} {
    // ARM64 equivalent
} {
    // RISC-V equivalent
}
```

#### Example 3: Color Packing
```flap
// Pack RGBA bytes into single val
packed_color := unsafe {
    rax <- 0xFF      // R
    rcx <- 0x80      // G
    rdx <- 0x40      // B
    rbx <- 0xFF      // A
    // Shift and combine (shift instructions not yet implemented)
    rax <- rax
} { /* arm64 */ } { /* riscv64 */ }
```

### Limitations

Current limitations (will be expanded):
- ✅ Immediate loads
- ✅ Register moves
- ✅ Stack push/pop
- ❌ Arithmetic operations (add, sub, mul, div)
- ❌ Bitwise operations (and, or, xor, shift)
- ❌ Memory loads/stores (beyond stack)
- ❌ Conditional operations

### Safety Considerations

Unsafe blocks bypass Flap's safety guarantees:

⚠️ **You can:**
- Corrupt the stack pointer
- Overwrite important registers
- Create undefined behavior

✅ **Best practices:**
- Keep unsafe blocks small
- Document what each block does
- Test on all three architectures
- Avoid modifying `rsp`/`sp` unless you know what you're doing
- Preserve caller-saved registers if calling functions

### When to Use Unsafe

**Good uses:**
- Performance-critical tight loops
- Custom bit manipulation
- Specialized SIMD operations
- Interfacing with specific CPU features

**Bad uses:**
- General arithmetic (use Flap operators instead)
- String manipulation (use Flap builtins)
- Control flow (use match blocks)
- Anything portable Flap can already do

Remember: **With great power comes great responsibility**. Unsafe blocks give you the keys to the CPU, but also the ability to crash your program spectacularly.

## Grammar

The hand-written recursive-descent parser accepts the following grammar. Newlines separate statements but are otherwise insignificant. `//` starts a line comment. String escape sequences: `\n`, `\t`, `\r`, `\\`, `\"`.

```ebnf
program         = { newline } { statement { newline } } ;

statement       = use_statement
                | import_statement
                | arena_statement
                | defer_statement
                | loop_statement
                | jump_statement
                | assignment
                | expression_statement ;

use_statement   = "use" string ;

import_statement = "import" string ;

arena_statement = "arena" block ;

defer_statement = "defer" expression ;

loop_statement  = "@" block
                | [ expression ] "@" identifier "in" expression [ "max" (number | "inf") ] block ;

jump_statement  = ("ret" | "err") [ "@" number ] [ expression ]
                | "@" number
                | "@++"
                | "@" number "++" ;

assignment      = identifier [ ":" type_annotation ] ("=" | ":=" | "<-") expression
                | identifier ("+=" | "-=" | "*=" | "/=" | "%=") expression ;

type_annotation = ("b" | "f") number ;

expression_statement = expression [ match_block ] ;

match_block     = "{" ( default_arm
                      | match_clause { match_clause } [ default_arm ] ) "}" ;

match_clause    = "->" match_target
                | expression [ "->" match_target ] ;

default_arm     = "~>" match_target ;

match_target    = jump_target | expression ;

jump_target     = ("ret" | "err") [ "@" number ] [ expression ]
                | "@" number ;

block           = "{" { statement { newline } } "}" ;

expression              = or_bang_expr ;

or_bang_expr            = concurrent_gather_expr [ "or!" string ] ;

concurrent_gather_expr  = parallel_expr { "|||" parallel_expr } ;

parallel_expr           = pipe_expr { "||" pipe_expr } ;

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
                        | arena_expr
                        | unsafe_expr
                        | "^" primary_expr
                        | "&" primary_expr ;

arena_expr              = "arena" "{" { statement { newline } } expression "}" ;

unsafe_expr             = "unsafe" "{" { statement { newline } } [ expression ] "}" ;

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

* `@` without arguments creates an infinite loop: `@ { ... }`
* `@` with identifier introduces auto-labeled loops. The loop label is the current nesting depth (1, 2, 3, ...).
* Optional numeric prefix before `@` specifies parallel execution: `4 @ i in list` uses 4 CPU cores
* `@++` continues the current loop (skip this iteration, jump to next).
* `@1++`, `@2++`, `@3++`, ... continues the loop at that nesting level (skip iteration, jump to next).
* `unsafe` blocks can return register values as expressions (e.g., `rax`, `rbx`)
* `ret` returns from the current function with a val. `ret @` exits the current loop. `ret @1`, `ret @2`, `ret @3`, ... exits the loop at that nesting level and all inner loops. `err` returns an err.
* Lambda syntax: `x => expr` or `x, y => expr` (no parentheses around parameters).
* Type casting with `as`: Bidirectional conversion for FFI (e.g., `42 as i32` to C, `c_val as number` from C).
* Match blocks attach to the preceding expression. When omitted, implicit default is `0`.
* A single bare expression inside braces is shorthand for `-> expression`.
* A block with only `~>` leaves the condition's val untouched when true.
* Type annotations use `:b64` or `:f32` syntax for precision control.

## Keywords

```
and and! as cstruct err hot in not or or! ret xor &b |b ^b ~b <b >b >>b <<b
i8 i16 i32 i64 u8 u16 u32 u64 f32 f64 cstr cptr
number string list
packed aligned sizeof
```

**Note:** Type keywords (`i8`, `i16`, `i32`, `i64`, `u8`, `u16`, `u32`, `u64`, `f32`, `f64`, `cstr`, `cptr`, `number`, `string`, `list`) are **contextual keywords** - they are only reserved when used after `as` in type casting expressions. They can be used as variable names in other contexts:

```flap
// Valid: using type keywords as variable names
i32 = 100.0
ptr = call("malloc", 64 as u64)
string = "hello"

// Also valid: using them as type keywords in casts
x = 42 as i32
address = pointer_value as cptr
text = c_string as string
```

## Examples

### Factorial

```flap
factorial := n => n <= 1 {
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
time_val = call("time", 0 as cptr)
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
val = read_f64(ptr, 0)          // Read back float64
int_val = read_i32(ptr, 1)      // Read back int32
call("free", ptr as cptr)

// Working with C structs (safe indexing)
// struct Point { float x; float y; }
point_ptr = call("malloc", 8 as u64)
write_f32(point_ptr, 0, 10.5)   // x field at index 0
write_f32(point_ptr, 1, 20.3)   // y field at index 1
x_val = read_f32(point_ptr, 0)
y_val = read_f32(point_ptr, 1)
call("free", point_ptr as cptr)
```

### C-Compatible Structs

The `cstruct` keyword defines C-compatible struct types for FFI. These structs have explicit memory layout, padding, and alignment compatible with C libraries.

**Syntax:**

```flap
cstruct StructName {
    field1: c_type,
    field2: c_type,
    ...
}

// With modifiers
cstruct StructName packed {  // No padding between fields
    ...
}

cstruct StructName aligned(16) {  // 16-byte alignment
    ...
}
```

**Examples:**

```flap
// Basic C struct
cstruct Vec3 {
    x: f32,
    y: f32,
    z: f32
}  // 12 bytes, standard C layout

// Packed struct (no padding)
cstruct PackedData packed {
    flag: u8,      // 1 byte
    val: u32,      // 4 bytes (no padding before)
    count: u16     // 2 bytes
}  // 7 bytes total (no padding)

// Aligned struct for SIMD
cstruct AlignedVec4 aligned(16) {
    x: f32,
    y: f32,
    z: f32,
    w: f32
}  // 16 bytes, 16-byte aligned

// Struct with pointers and strings
cstruct Entity {
    name: cstr,        // C string pointer
    position: *Vec3,   // Pointer to Vec3
    health: i32,
    flags: u32
}

// Nested structs
cstruct Transform {
    position: Vec3,
    rotation: Vec3,
    scale: Vec3
}
```

**Creating and Using:**

```flap
// Allocate struct memory
entity_ptr := call("malloc", sizeof(Entity) as u64)

// Write fields (field offsets calculated by compiler)
write_cstr(entity_ptr, Entity.name_offset, "Player" as cstr)
write_i32(entity_ptr, Entity.health_offset, 100)
write_u32(entity_ptr, Entity.flags_offset, 0x1)

// Read fields
health := read_i32(entity_ptr, Entity.health_offset)

// Pass to C functions
call("process_entity", entity_ptr as cptr)

// Cleanup
call("free", entity_ptr as cptr)
```

**Benefits:**
- **C-compatible**: Exact same layout as C structs
- **Explicit**: No hidden padding or alignment surprises
- **Safe offsets**: Compiler calculates field offsets
- **FFI-ready**: Works seamlessly with SDL, Raylib, etc.

### Hot Code Reload

Hot code reload allows recompiling and hot-swapping functions in a running program **without restart or re-launch**. The executable modifies its own machine code pages and atomically swaps to the new implementation - essential for rapid gamedev iteration.

**How It Works:**

Unlike traditional hot reload (restart process or dlopen), Flap's hot reload works by:

1. **Function indirection** - Hot functions called through function pointer table
2. **File watching** - Monitor source files with `inotify`/`kqueue`/`FSEvents`
3. **Incremental recompilation** - Recompile only changed functions
4. **Executable memory** - Allocate new pages with `mmap(PROT_EXEC)`
5. **Atomic swap** - Update function pointer (single atomic store)
6. **Self-modification** - Running executable patches itself

**Performance:** ~1-2 CPU cycles overhead per hot function call (indirect branch, branch predictor friendly)

**Marking Functions for Hot Reload:**

```flap
// Mark function as hot-swappable
hot update_player = (state, dt) => {
    state.x <- state.x + state.vx * dt
    state.y <- state.y + state.vy * dt

    // Edit this code while game runs
    state.x < 0 { state.x <- 0 }
    state.x > 800 { state.x <- 800 }

    ret state
}

hot draw_particles = particles => {
    @ p in particles {
        draw_circle(p.x, p.y, p.radius, p.color)
    }
}

// Normal function - no indirection overhead
init_game = () => {
    // Cannot be hot-swapped
    ret GameState { ... }
}
```

**Compilation & Usage:**

```bash
# Start game with file watching enabled
./flapc game.flap --watch -o game &
./game

# Edit hot functions in game.flap
# Compiler detects changes and hot-swaps automatically
# Game continues running with new code!

# Or manual trigger:
kill -USR1 $(pidof game)  # Reload signal
```

**Implementation Details:**

```c
// Compiler generates function table
struct FunctionTable {
    void (*update_player)(State*, f64);
    void (*draw_particles)(List*);
} hot_functions;

// All hot function calls go through table:
hot_functions.update_player(state, dt);  // One indirect jump

// On file change:
// 1. Recompile changed function
// 2. Generate new machine code
// 3. code = mmap(NULL, size, PROT_READ|PROT_WRITE|PROT_EXEC, ...)
// 4. memcpy(code, new_machine_code, size)
// 5. atomic_store(&hot_functions.update_player, code)
// 6. Old code pages freed after grace period
```

**File Watching:**

- **Linux**: `inotify` watches `*.flap` files
- **macOS**: `FSEvents` / `kqueue`
- **FreeBSD**: `kqueue`
- Detects: saves, renames, modifications
- Triggers: incremental recompilation

**What Can Be Hot-Swapped:**

✅ **Allowed:**
- Function bodies (logic changes)
- Algorithms and control flow
- Constants used in functions
- Bug fixes

❌ **Not Allowed:**
- Function signatures (arg count/types locked)
- Struct layouts (field offsets fixed)
- Global variable sizes
- Inlined functions (callers have old code baked in)

**Example Gamedev Workflow:**

```flap
// main.flap
import sdl3 as sdl

main ==> {
    window := sdl.SDL_CreateWindow("Game", 0, 0, 800, 600, 0)
    renderer := sdl.SDL_CreateRenderer(window, -1, 0)

    state := init_game()  // Not hot - runs once
    running := true

    @ running {
        // Hot functions - can edit while running
        state <- hot_update(state, 0.016)
        hot_render(renderer, state)

        sdl.SDL_RenderPresent(renderer)
    }
}

// These can be edited and reloaded live
hot hot_update = (state, dt) => {
    // Change gameplay logic here
    // See changes in ~50ms without restart
    state.player.x <- state.player.x + state.player.vx * dt
    ret state
}

hot hot_render = (renderer, state) => {
    // Tweak colors, positions, effects
    // Instant visual feedback
    sdl.SDL_SetRenderDrawColor(renderer, 255, 0, 0, 255)
    sdl.SDL_RenderClear(renderer)

    // Draw game...
}
```

**Use Cases:**

- **Rapid iteration**: Test gameplay changes in seconds
- **Visual tuning**: Adjust colors, positions, timing instantly
- **Live debugging**: Fix bugs while reproducing issue
- **Immediate feedback**: No wait for compile+restart cycle
- **State preservation**: Keep game state across code changes

**Comparison to Other Systems:**

| System | Method | Overhead | Restart Required |
|--------|--------|----------|------------------|
| **Flap** | Self-modifying code | ~0.1% | No |
| **SBCL (Lisp)** | Symbol table | ~5% | No |
| **Live++** | DLL patching | ~1% | No |
| **Rust hot reload** | dlopen | ~2% | No |
| **C++ recompile** | Full rebuild | 0% | Yes (slow) |

**Limitations:**

- Hot functions have slight overhead (indirect call)
- Cannot change function signatures
- Cannot change struct layouts mid-game
- Platform-specific (requires `mmap` + executable pages)
- Debug symbols may be stale

**Technical Notes:**

- Function table stored in `.data` section (mutable)
- Old code pages kept alive ~1 second for in-flight calls
- Atomic pointer updates ensure no torn reads
- Recompilation is incremental (only changed functions)
- Compatible with debuggers (gdb/lldb attach works)

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

### C Library Imports

**Status (v1.0.0):** C library FFI is fully functional for basic use cases.

Flap can call C library functions directly using a simple import syntax. The compiler automatically handles dynamic linking via PLT/GOT on Linux.

**Syntax:**

```flap
// Import C library (auto-detected by lack of "/" in name)
import sdl3 as sdl
import raylib as rl
import c as libc  // Standard C library

// Import custom .so file (NEW in v1.6.0)
import "/tmp/libmylib.so" as mylib
import "/usr/local/lib/libcustom.so.1" as custom

// Call C functions with namespace prefix
sdl.SDL_Init(0)
window := sdl.SDL_CreateWindow("Game", 100, 100, 800, 600, 0)

// Call custom library functions
result := mylib.my_function(1, 2, 3, 4, 5, 6, 7, 8)  // >6 arguments supported!

// Standard library functions
pid := libc.getpid()
time := libc.time(0)
```

**How It Works:**

1. **Auto-detection**:
   - Strings ending with ".so" or containing ".so." → Custom .so file import
   - Identifiers without "/" → Standard C libraries (sdl3, raylib, c)
   - Strings with "/" → Flap packages (Git)
2. **Symbol extraction**: For custom .so files, symbols are automatically extracted using `nm -D`
3. **Dynamic linking**: C libraries are added to ELF `DT_NEEDED` (e.g., `libSDL3.so`, `libraylib.so`, `libmylib.so`)
4. **PLT calls**: Functions are called through the Procedure Linkage Table
5. **ABI compatibility**: Arguments are marshaled to System V AMD64 calling convention
6. **Stack arguments**: Functions with >6 arguments use stack-based argument passing

**Custom .so File Imports (v1.6.0+):**

```flap
// Import custom library
import "/tmp/libmanyargs.so" as mylib

// Functions are automatically discovered from the .so file
result := mylib.sum7(1, 2, 3, 4, 5, 6, 7)          // 7 arguments
result := mylib.sum10(1, 2, 3, 4, 5, 6, 7, 8, 9, 10) // 10 arguments

// Run with: LD_LIBRARY_PATH=/tmp ./program
```

**Current Limitations (v1.6.0):**

- Arguments are converted to integers (uint32/int64) or float64
- Return values are converted to Flap's `float64`
- Limited support for strings (C strings work via `as cstr` cast)
- No support for structs yet

**Example: SDL3 Game Initialization**

```flap
import sdl3 as sdl

// Initialize SDL
result := sdl.SDL_Init(0x00000020)  // SDL_INIT_VIDEO = 0x20
result > 0 {
    println("SDL_Init failed")
    exit(1)
}

// Create window
window := sdl.SDL_CreateWindow(
    "My Game",  // title (will be supported in v1.4.1)
    100,        // x
    100,        // y
    800,        // width
    600,        // height
    0           // flags
)

// Game loop would go here...

sdl.SDL_Quit()
```

**Example: Standard C Library**

```flap
import c as libc

pid := libc.getpid()
println("Process ID:")
println(pid)

time := libc.time(0)
println("Unix timestamp:")
println(time)
```

**Library Naming:**

The compiler automatically converts library names:
- `sdl3` → `libSDL3.so.0` (SDL3 with version)
- `raylib` → `libraylib.so.5` (RayLib 5)
- `c` → (uses already-linked `libc.so.6`)
- `m` → (math library, link automatically if needed)

**Roadmap (v1.4.1+):**

- String arguments (C char pointers)
- Struct support
- Pointer handling
- Float return values
- pkg-config integration for automatic library discovery
- >6 argument support

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

## Memory Management

**Status (v1.0.0):** Syntax and documentation complete. Full runtime implementation coming in v1.5.0.

Flap introduces syntax for arena allocators and defer statements. The compiler recognizes the keywords and parses the constructs, with full runtime implementation planned for v1.5.0.

### Arena Allocators

Arena allocators provide fast bump-pointer allocation with bulk deallocation. All memory allocated within an `arena` block is automatically freed when the block exits.

**Key Benefits:**
- **Fast allocation**: O(1) bump pointer, no per-allocation overhead
- **Automatic cleanup**: No manual free() calls needed
- **Cache friendly**: Contiguous memory allocation
- **Perfect for**: Temporary data structures, per-frame game allocations, parser ASTs

**Syntax:**

```flap
arena {
    buffer := alloc(1024)        // Allocate 1KB
    particles := alloc(8 * 100)  // Allocate 100 particles (8 bytes each)
    // Use buffer and particles...
}  // All memory automatically freed here
```

**Implementation:**
- Initial size: 4096 bytes (4KB)
- Growth strategy: Double on overflow (4KB → 8KB → 16KB → 32KB...)
- Uses `malloc()` for initial allocation, `realloc()` for growth
- Thread-local arena stack for nested arenas

**Nested Arenas:**

```flap
arena {
    outer_data := alloc(100)

    arena {
        inner_data := alloc(200)
        // Both inner_data and outer_data available
    }  // inner_data freed

    // outer_data still available
}  // outer_data freed
```

**Game Development Example:**

```flap
// Per-frame arena for temporary allocations
game_loop = () -> {
    @ frame in range(1000000) {
        arena {
            // Allocate temporary structures
            visible_entities := alloc(entity_size * max_visible)
            render_commands := alloc(command_size * max_commands)

            // Render frame using temporary data...
            render_frame(visible_entities, render_commands)
        }  // All temporary memory freed - zero fragmentation
    }
}
```

### The `alloc()` Builtin

Allocates memory from the current arena.

**Signature:** `alloc(size: number) -> pointer`

**Parameters:**
- `size`: Number of bytes to allocate

**Returns:** Pointer to allocated memory (as float64)

**Example:**

```flap
arena {
    // Allocate structure
    player := alloc(64)  // 64 bytes

    // Write to memory
    write_f64(player, 0, 100.0)      // health at offset 0
    write_f64(player, 8, 50.0)       // mana at offset 8
    write_f64(player, 16, 250.5)     // x position
    write_f64(player, 24, 128.3)     // y position

    // Read from memory
    health := read_f64(player, 0)
    printf("Player health: %.0f\n", health)
}
```

**Interaction with malloc/free:**

You can still use `malloc()` and `free()` via FFI for manual memory management:

```flap
// Load libc
libc := dlopen("libc.so.6", 2)  // RTLD_NOW
malloc_fn := dlsym(libc, "malloc")
free_fn := dlsym(libc, "free")

// Manual allocation
ptr := call(malloc_fn, 1024 as u64) as cptr
defer call(free_fn, ptr)  // Cleanup with defer

// Use ptr...
```

### Defer Statements

The `defer` keyword schedules an expression to execute at the end of the current scope, regardless of how the scope exits (normal return, early return, or implicit fall-through).

**Execution Order:** LIFO (Last-In-First-Out) - like a stack

**Syntax:**

```flap
defer expression
```

**Example:**

```flap
open_and_process = (filename) -> {
    file := fopen(filename, "r")
    defer fclose(file)  // Always executed

    // If this fails, fclose still called
    data := read_file(file)

    // Process data...
    ret process(data)  // fclose called before return
}
```

**Multiple Defers:**

```flap
process_resources = () -> {
    file1 := fopen("data.txt", "r")
    defer fclose(file1)

    file2 := fopen("config.txt", "r")
    defer fclose(file2)

    connection := connect("localhost", 8080)
    defer disconnect(connection)

    // On exit, calls in order:
    // 1. disconnect(connection)
    // 2. fclose(file2)
    // 3. fclose(file1)
}
```

**With Arena:**

```flap
load_level = (level_file) -> {
    arena {
        // Temporary allocations for loading
        temp_buffer := alloc(1024 * 1024)  // 1MB temp buffer

        file := fopen(level_file, "rb")
        defer fclose(file)  // Called before arena cleanup

        // Load and process...
        level_data := parse_level(file, temp_buffer)

        ret level_data
    }  // Arena freed, then fclose called (LIFO)
}
```

**Common Patterns:**

```flap
// Resource cleanup
handle_request = (request) -> {
    lock := acquire_lock()
    defer release_lock(lock)

    // Critical section...
}

// Profiling
profile_function = () -> {
    start := get_time()
    defer {
        duration := get_time() - start
        printf("Function took %.2fms\n", duration)
    }

    // Function body...
}

// Error handling with manual memory
allocate_and_process = () -> {
    libc := dlopen("libc.so.6", 2)
    malloc_fn := dlsym(libc, "malloc")
    free_fn := dlsym(libc, "free")

    ptr1 := call(malloc_fn, 1024 as u64) as cptr
    defer call(free_fn, ptr1)

    ptr2 := call(malloc_fn, 2048 as u64) as cptr
    defer call(free_fn, ptr2)

    // Both freed automatically, even if early return
}
```

### Best Practices

**Use arenas for:**
- Temporary allocations (per-frame game data)
- Parser/compiler intermediate structures
- Request-scoped data in servers
- Any data with clear lifetime boundaries

**Use defer for:**
- File handles (fopen/fclose)
- Network connections
- Locks (acquire/release)
- Manual memory cleanup (malloc/free)
- Resource handles from C libraries

**Avoid:**
- Long-lived data in arenas (arena blocks should be scoped)
- Mixing arena alloc() with manual free() (undefined behavior)
- Returning pointers from arena blocks (dangling pointer)

**Example: Game Entity System**

```flap
update_physics = (entities, dt) -> {
    arena {
        // Temporary spatial partitioning
        grid := alloc(grid_size * 8)

        // Build acceleration structure
        @ entity in entities {
            cell := get_grid_cell(entity.x, entity.y)
            add_to_cell(grid, cell, entity)
        }

        // Process collisions using grid
        collisions := check_collisions(grid)
        apply_collision_responses(collisions, dt)

    }  // Grid freed automatically
}
```

## Hot Code Reload

The `hot` keyword marks functions as hot-reloadable for live development:

```flap
hot update_player := (player, dt) => {
    player.x <- player.x + player.vx * dt
    player.y <- player.y + player.vy * dt
}

hot render_scene := (scene) => {
    @ entity in scene.entities {
        draw_sprite(entity.sprite, entity.x, entity.y)
    }
}

normal_init := () => {
    // Non-hot functions compile to direct calls (faster)
    load_assets()
}
```

**Current Implementation:**
- Parser recognizes `hot` keyword
- Functions marked as hot are tracked in compiler
- Foundation for runtime code swapping

**Future Capabilities:**
- Function pointer table for indirect calls
- File watching (inotify/kqueue/FSEvents)
- Incremental recompilation of changed functions
- Atomic pointer swaps for seamless updates
- ~1-2 cycle overhead per hot function call

**Use Cases:**
- Game development (iterate on gameplay without restarting)
- Visual tuning (adjust rendering in real-time)
- Shader development
- Live debugging
- Rapid prototyping

**Limitations:**
- Cannot change function signatures at runtime
- Cannot modify struct layouts
- Hot functions have slight indirection overhead

Hot reload enables sub-50ms iteration cycles compared to full recompilation.
