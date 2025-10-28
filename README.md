# Flapc

[![Go CI](https://github.com/xyproto/flapc/actions/workflows/ci.yml/badge.svg)](https://github.com/xyproto/flapc/actions/workflows/ci.yml) [![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/flapc)](https://goreportcard.com/report/github.com/xyproto/flapc)

**Flap compiler** - Generates native x86-64 machine code directly. No LLVM, no GCC, no runtime.

**Built for:** Game development (SDL3/RayLib5), systems programming, concurrent applications
**Platform:** x86-64 Linux (Arch/Debian tested)
**Status:** 435+ tests passing, process spawning working, ENet networking in progress (v1.6.0 development)

## Key Features

**Direct machine code generation** - Lexer → Parser → x86-64 → ELF. No IR. Instant compilation (~1ms).

**Unified type system** - Everything is `map[uint64]float64`. Numbers, strings, lists, objects—all the same runtime representation. SIMD-optimized lookups (AVX-512/SSE2).

```flap
42              // {0: 42.0}
"Hi"            // {0: 72.0, 1: 105.0}
[1, 2, 3]       // {0: 1.0, 1: 2.0, 2: 3.0}
{x: 10, y: 20}  // {x: 10.0, y: 20.0}
```

**Tail-call optimization** - Automatic TCO. Immutable by default (`=`), mutable when needed (`:=`).

**Arena memory** - Scope-based allocation. Perfect for frame-local game buffers.

**C FFI** - Direct PLT/GOT calls to C libraries. Automatic type inference from DWARF debug info.

**Unsafe blocks** - Cross-platform direct register access (x86-64/ARM64/RISC-V). Unified syntax with register aliases or per-CPU blocks.

**Process spawning** - Unix fork()-based concurrency with `spawn` keyword.

**ENet networking** - Port literals (`:5000`, `:worker`) for IPC and networking.

```flap
// Tail recursion
fib = n => n < 2 { -> n ~> fib(n-1) + fib(n-2) }

// Arenas
arena { buffer := alloc(1024) /* ... */ }  // auto-freed

// C FFI
import sdl3 as sdl
window := sdl.SDL_CreateWindow("Game", 800, 600, 0)

// Process spawning
spawn worker()                    // Fire-and-forget
spawn compute(42) | result | {}   // Wait for result (not yet implemented)

// Port literals (ENet)
port := :5000                     // Numeric port
worker_port := :worker            // Named port (hashed to 39639)

// Unsafe (unified - works on all CPUs)
result := unsafe { a <- 42; a }  // a = rax/x0/a0 depending on CPU
```

## Installation

### From Source
```bash
git clone https://github.com/xyproto/flapc
cd flapc
go build
sudo install -Dm755 flapc /usr/bin/flapc
```

### Dependencies
- Runtime: None (static binary generation)
- Build: Go 1.21+ (for compiler only)

## Usage

```bash
# Compile and run
flapc hello.flap
./hello

# Verbose output (see assembly)
flapc -v program.flap

# Specify output
flapc -o game program.flap
```

## Quick Start

```bash
# Install
go build && sudo install -Dm755 flapc /usr/bin/flapc

# Hello world
echo 'println("Hello, World!")' > hello.flap
flapc hello.flap && ./hello

# Game loop
cat > game.flap << 'EOF'
import sdl3 as sdl
@ { update(); render() }  // infinite loop
EOF
flapc game.flap -o game
```

See `testprograms/` for 50+ examples.

## Language Reference

**Syntax**
- Variables: `x = 42` (immutable), `x := 42` (mutable), `x <- 43` (update)
- Loops: `@ i in 0..<10 { }`, `@ { }` (infinite)
- Match: `x > 0 { yes() ~> no() }` (if-else)
- Lambdas: `f = x => x * 2` or `(x, y) => x + y`
- Processes: `spawn worker()` (Unix fork)
- Ports: `:5000`, `:worker` (network/IPC)

**Types** (all `map[uint64]float64` internally)
- Numbers: `42`, `3.14`, `0xFF`, `0b1010`
- Strings: `"text"`, Lists: `[1,2,3]`, Maps: `{x: 10}`

**Memory**
- Manual: `alloc(size)` / `free(ptr)`
- Arena: `arena { ... }` (scope-based)
- Defer: `defer cleanup()` (LIFO)

**C FFI**
- Import: `import sdl3 as sdl`
- Calls: `sdl.SDL_CreateWindow(...)`
- ABI: System V AMD64, PLT/GOT linking
- Signatures: Auto-discovered via DWARF/pkg-config

**Unsafe**
- Unified syntax: `a <- 42` (portable aliases: a, b, c, d, e, f)
- Per-CPU blocks: `unsafe { x86_64 { rax <- 42 } arm64 { x0 <- 42 } riscv64 { a0 <- 42 } }`
- Arithmetic: `c <- a + b`, `d <- a << 2`
- Memory: `a <- [b]`, `[a] <- value`
- Returns: Last expression value

See the [Unsafe Blocks section](#unsafe-blocks-battlestar-assembly) below for complete Battlestar assembly reference and [GRAMMAR.md](GRAMMAR.md) for full grammar.

## Technical Details

**Compilation:** Lexer → Parser → x86-64 → ELF. Two-pass for address resolution. No IR, no external linker. ~12K lines of Go.

**ABI:** System V AMD64. Args in `rdi, rsi, rdx, rcx, r8, r9`. Floats in `xmm0-7`. 16-byte stack alignment.

**SIMD:** Runtime CPUID detection. AVX-512 (8 keys/iter), SSE2 (2 keys/iter), scalar (1 key/iter). All three compiled into every binary.

**Binary:** ELF64. Dynamic link to libc. Direct syscalls for I/O. No GC, no runtime.

**Optimization:** Dead code elimination, constant propagation, loop unrolling, whole-program optimization (2s timeout default).

**Platform:** x86-64 Linux only. ARM64/RISC-V/Windows/macOS planned.

## Known Limitations

**Platform:** x86-64 Linux only. Other platforms in development (ARM64/RISC-V/Windows/macOS).

See [TODO.md](TODO.md) for detailed roadmap.

## Documentation

- [GRAMMAR.md](GRAMMAR.md) - Formal language grammar
- [TODO.md](TODO.md) - Development roadmap

All language features and implementation notes are documented in this README.


---

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

**Network Send:** `<==`
```flap
// Send message to port/address (ENet/UDP)
":5000" <== "hello"              // Send to localhost:5000
":8080" <== "message"             // Send to variable containing port/address
"server.com:5000" <== "data"   // Send to remote host
```

**Comparison:** `<=` (less than or equal to)
```flap
x <= 10   // Less than or equal to
```

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
- `++` (pointer append) - Buffer building with auto-increment:
  ```flap
  // Allocate buffer
  buffer := call("malloc", 1024 as u64) as cptr

  // Write values sequentially with auto-increment
  buffer ++ 42 as uint32      // Write uint32(42) at offset 0, counter becomes 4
  buffer ++ 3.14 as f32       // Write f32(3.14) at offset 4, counter becomes 8
  buffer ++ 255 as uint8      // Write uint8(255) at offset 8, counter becomes 9
  buffer ++ 1000 as uint16    // Write uint16(1000) at offset 9, counter becomes 11

  // The pointer address remains unchanged - only internal counter increments!
  printf("buffer address: %v\n", buffer)  // Same address throughout
  ```

  **How it works:**
  - Compiler maintains **hidden internal counter** for each pointer variable
  - `ptr ++ value as type` writes value at `ptr + counter`
  - Counter automatically increments by `sizeof(type)`
  - **Pointer address itself never changes** - only the counter
  - Counter is compile-time tracked, not stored in memory
  - Perfect for building binary buffers, network packets, file formats

- `+!` (add with carry) - Multi-precision arithmetic:
  ```flap
  // Automatic carry handling
  low := 0xFFFFFFFFFFFFFFFF  // Max uint64
  high := 0x1
  result_low := low +! 1      // Wraps to 0, sets carry
  result_high := high +! 0    // Becomes 2 due to carry
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

**Pointer Field Access:**

The `.` operator also works with C pointers to structs, automatically handling dereferencing (like Go):

```flap
// C struct: struct Point { float x; float y; }
point_ptr := call("malloc", 8 as u64) as cptr

// Direct field access on pointer (no manual dereferencing needed)
point_ptr.x <- 10.5   // Writes to offset 0 (x field)
point_ptr.y <- 20.3   // Writes to offset 4 (y field)

// Read fields
x_val := point_ptr.x  // Reads from offset 0
y_val := point_ptr.y  // Reads from offset 4

// Works with cstruct definitions
cstruct Entity {
    x: f32,
    y: f32,
    health: i32
}

entity := call("malloc", sizeof(Entity) as u64) as cptr
entity.x <- 100.0
entity.y <- 200.0
entity.health <- 100

// No need for -> operator like in C!
```

**Implementation Details:**
- String keys are hashed at **compile time** using FNV-1a hash algorithm
- Hash values use a 30-bit range (`0x40000000` to `0x7FFFFFFF`) to work within current compiler limitations
- For maps: `obj.field` compiles to `obj[hash("field")]`
- For pointers: `ptr.field` compiles to memory read/write at calculated offset
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

// Parallel loop with ALL available cores (shorthand)
@@ i in data max 100000 {
    process(i)  // Uses all CPU cores - shorthand for cpu_count() @
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
- All cores: `@@ item in collection` (shorthand for all available cores)
- Optional prefix: number of cores to use (e.g., `4 @`, `8 @`)
- Use for: data processing, image processing, physics simulations, ray tracing
- **Implementation:** Uses Linux clone() syscalls with futex-based barrier synchronization for thread coordination
- **Status:** Range loops with constant bounds fully functional (V4 complete); dynamic bounds and full loop body execution coming in V5
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

// Parallel processing with all available cores
@@ pixel in pixels max 1000000 {
    process_pixel(pixel)  // Uses all CPU cores automatically
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

### Concurrency and Parallelism

Flap combines **Unix fork()**, **OpenMP** (data parallelism), and **ENet** (networking) into a unified, minimal syntax.

#### Philosophy

**Three models, one syntax:**
1. **Process spawning** - `spawn` keyword for fork()-style processes (Unix)
2. **Parallel loops** - `@@` and `N @` for data parallelism (OpenMP-inspired)
3. **Message passing** - `:port` for ENet communication (IPC + networking unified)

**Design principles:**
- Process-based (fork model) - true parallelism, no shared state bugs
- ENet for all communication - IPC and networking use same syntax
- Network addresses as strings (":5000", "host:port")
- Zero magic - explicit communication only

---

#### Process Spawning with `spawn`

Spawn processes using the `spawn` keyword (Unix fork-based):

```flap
// Fire and forget
spawn worker()

// Wait for single result
spawn compute(42) | result | {
    printf("Result: %v\n", result)
}

// Destructure multiple return values
spawn get_coords() | x, y | {
    printf("Position: (%v, %v)\n", x, y)
}

// Map destructuring (structured data)
spawn fetch_user(id) | {name, age} | {
    printf("%s is %v years old\n", name, age)
}

// Nested destructuring
spawn load_game() | {player: {name, health}, level} | {
    printf("Player %s (HP: %v) on level %v\n", name, health, level)
}

// Pattern matching on results
spawn http_get(url)
    | {status: 200, data} | process_success(data)
    | {status: 404} | printf("Not found\n")
    | {status, error} | printf("Error %v: %s\n", status, error)

// Spawn multiple workers
@ i in 0..<4 {
    spawn worker(:8000 + i)  // Each worker on different port
}
```

**How it works:**
- `spawn` creates new process (Unix fork)
- Copy-on-write memory (cheap)
- Pipe syntax waits for result (blocks until child returns)
- No pipe = fire and forget (child continues after parent exits)
- Destructuring uses same syntax as lambda parameters

---

#### Parallel Loops (Data Parallelism)

Parallelize loops automatically (inspired by OpenMP):

```flap
// Sequential
@ i in 0..<1000 {
    process(i)
}

// Parallel with 4 cores (OpenMP-style)
4 @ i in 0..<1000 {
    process(i)  // Splits work across 4 cores
}

// All cores
@@ i in 0..<1000 {
    process(i)  // Uses all available cores
}

// Parallel with shared array (each index independent)
results := [0] * 1000
@@ i in 0..<1000 {
    results[i] <- expensive(i)  // Safe - no shared state
}
```

**How it works:**
- Work-sharing: runtime divides iterations across cores
- Each core gets chunk of iterations
- Automatic synchronization at loop end
- No shared mutable state (each iteration independent)

---

#### ENet Messaging (IPC + Networking Unified)

All communication uses ENet with `:port` syntax:

**Port Literals:**
```flap
// Numeric port literal (ENet server on localhost)
":5000"       // Port 5000 on localhost

// String port literal (hashed to port number)
:game_server // Hashed to deterministic port number
:worker      // Named ports for clarity
:banana      // Any string works - hashed consistently

// Remote address (string)
"server.com:5000"      // Remote host + port
"192.168.1.100:7777"   // IP address + port

// Next available port (returns actual port number)
port := :5000+  // Tries 5000, 5001, 5002, ... returns actual port
printf("Bound to port: %v\n", port)

@ msg, from in port {
    printf("[Port %v] Got: %s\n", port, msg)
}

// Check if port is available
:5000?       // Returns 1 if available, 0 if in use

// Port with fallback (using or)
port := :5000 or :5001  // Try 5000, if taken use 5001
port := :game_server or :9000+  // Named port with dynamic fallback

// Using ?: for explicit checking
port := :5000? or :3000  // If 5000 available, use it, else 3000
```

**Receiving Messages:**
```flap
// Listen on port and receive messages
@ msg, from in :5000 {
    printf("Got: %s from %s\n", msg, from)
}

// 'msg' is the message string
// 'from' is the sender address (e.g., "127.0.0.1:51234")

// Pattern match on messages
@ msg, from in :5000 {
    msg {
        "ping" -> from <== "pong"
        "quit" -> ret @
        ~> printf("Unknown: %s\n", msg)
    }
}
```

**Sending Messages:**
```flap
// Send to local port
":5000" <== "hello"

// Send to remote host
"server.com:5000" <== "hello"

// Send from variable
target := "192.168.1.100:7777"
target <== "data"

// Broadcast to multiple addresses
addresses := [":8000", ":8001", ":8002", ":8003"]
@ addr in addresses max 100 {
    addr <== "broadcast message"
}
```

---

#### Complete Example: Worker Pool

```flap
// Worker process (listens on its port)
worker := (port) => {
    @ msg, from in port {
        printf("[Worker %v] Got task: %s\n", port, msg)

        // Process task
        result := expensive_computation(msg)

        // Send result back
        from <- result
    }
}

// Master process
main ==> {
    // Spawn 4 workers on next available ports starting at 8000
    worker_ports := []
    @ i in 0..<4 {
        port := :8000+  // Finds next available port
        worker_ports <- worker_ports + [port]
        spawn worker(port)
    }

    // Distribute tasks to workers
    tasks := ["task1", "task2", "task3", "task4", "task5", "task6"]
    @ task in tasks max 100 {
        worker_port := worker_ports[task % 4]
        worker_port <- task
    }

    // Collect results on available port
    master_port := :7000? { :7000 ~> :7001 }
    @ msg, from in master_port max 6 {
        printf("Result: %s from %s\n", msg, from)
    }
}
```

---

#### Complete Example: Distributed Web Scraper

```flap
// Worker: Scrapes URLs and sends results back
scraper := (worker_port, master_addr) => {
    @ msg, from in worker_port {
        url := msg
        printf("[Scraper] Fetching %s\n", url)

        data := http.get(url)
        master_addr <- data
    }
}

// Master: Distributes URLs and collects results
main ==> {
    master_port := :7000
    urls := ["http://example.com", "http://test.com", "http://demo.com"]

    // Spawn scrapers
    @ i in 0..<4 {
        scraper_port := :8000 + i
        spawn scraper(scraper_port, master_port)
    }

    // Distribute URLs
    @ url in urls max 1000 {
        worker := :8000 + (url % 4)
        worker <- url
    }

    // Collect results
    results := []
    @ msg, from in master_port max 1000 {
        results <- results + [msg]
        results.length >= urls.length { ret @ }
    }

    printf("Scraped %v URLs\n", results.length)
}
```

---

#### Complete Example: Multiplayer Game Server

```flap
// Game worker: Handles subset of players
game_worker := (port) => {
    players := {}

    @ msg, from in port {
        parsed := json.parse(msg)

        parsed.type {
            "connect" -> {
                players[from] <- {x: 0, y: 0, health: 100}
                printf("[Worker] Player connected: %s\n", from)
            }
            "move" -> {
                players[from].x <- parsed.x
                players[from].y <- parsed.y

                // Broadcast to all players
                @ addr in players.keys() max 1000 {
                    addr <- json.stringify({
                        type: "update",
                        players: players
                    })
                }
            }
            "disconnect" -> {
                players <- players.remove(from)
            }
        }
    }
}

// Load balancer: Distributes players across workers
main ==> {
    workers := [":8000", ":8001", ":8002", ":8003"]

    // Spawn game workers
    @ worker in workers max 100 {
        spawn game_worker(worker)
    }

    // Accept client connections and route
    @ msg, from in :7777 {
        // Assign player to worker (consistent hashing)
        worker_idx := hash(from) % workers.length
        worker_addr := workers[worker_idx]

        // Forward message to worker
        worker_addr <- msg
    }
}
```

---

#### Complete Example: Distributed Monte Carlo Pi

```flap
// Worker: Computes portion of samples
monte_worker := (port, master_addr, samples) => {
    hits := 0

    @@ i in 0..<samples {
        x := random()
        y := random()
        (x*x + y*y) < 1.0 {
            -> hits <- hits + 1
        }
    }

    // Send result to master
    master_addr <- f"{hits}"
}

// Master: Coordinates workers and computes final result
main ==> {
    total_samples := 10000000
    num_workers := 4
    samples_per_worker := total_samples / num_workers
    master_port := :7000

    // Spawn workers
    @ i in 0..<num_workers {
        worker_port := :8000 + i
        spawn monte_worker(worker_port, master_port, samples_per_worker)
    }

    // Collect results
    total_hits := 0
    @ msg, from in master_port max 100 {
        hits := parse_int(msg)
        total_hits <- total_hits + hits

        // Check if all workers reported
        (@counter + 1) >= num_workers { ret @ }
    }

    pi := 4.0 * total_hits / total_samples
    printf("Pi estimate: %v\n", pi)
}
```

---

#### Network vs Local Communication

```flap
// Local IPC (same machine)
":5000" <== "message"           // Fast - Unix domain socket

// Network (different machine)
"server.com:5000" <== "message"  // ENet over UDP

// Both use same syntax!
target := ":5000"  // or "remote.com:5000"
target <== "data"

// Automatic detection:
// - ":port" → localhost (fast IPC)
// - "host:port" → network (ENet/UDP)
```

---

**Summary:**

**Concurrency primitives:**
- `spawn fn()` - Spawn process (fork model, fire-and-forget)
- `spawn fn() | result | code` - Spawn and wait for result
- `spawn fn() | x, y | code` - Spawn with tuple destructuring
- `spawn fn() | {name, age} | code` - Spawn with map destructuring
- `@@` / `N @` - Parallel loops (OpenMP-inspired)
- `:port` - ENet port literal (first-class value)
- `:port+` - Next available port
- `:port?` - Check if port is available
- `"host:port"` - Remote address (string)
- `@ msg, from in :port` - Receive messages on port
- `:port <- data` - Send message to port

**Why this model?**
- ✅ **Unified** - IPC and networking use identical syntax
- ✅ **Simple** - Just processes, ports, and messages
- ✅ **Safe** - Process isolation prevents race conditions
- ✅ **Scalable** - Works from 1 core to distributed cluster
- ✅ **Clean** - ENet handles reliability, ordering, fragmentation
- ✅ **Fast** - Local ports use Unix sockets, network uses UDP
- ✅ **Zero magic** - Explicit communication only

---

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

## The Unsafe Language: Battlestar Assembly

Flap's `unsafe` blocks provide direct register access across x86_64, ARM64, and RISC-V architectures. This Battlestar-inspired sublanguage allows low-level systems programming while maintaining portability.

### Overview

Unsafe blocks execute architecture-specific machine code while integrating seamlessly with Flap's high-level features.

**Unified approach** (recommended - uses register aliases):
```flap
result := unsafe {
    a <- 42      // Load immediate (works on all CPUs)
    b <- 100     // Register aliases: a, b, c
    c <- a + b   // Register arithmetic
    c            // Return value (last expression)
}
```

**Per-CPU approach** (when platform-specific code is needed):
```flap
result := unsafe {
    x86_64 {
        rax <- 42
        rbx <- 100
        rcx <- rax + rbx
        rcx
    }
    arm64 {
        x0 <- 42
        x1 <- 100
        x2 <- x0 + x1
        x2
    }
    riscv64 {
        a0 <- 42
        a1 <- 100
        a2 <- a0 + a1
        a2
    }
}
```

### Register Aliases

Use portable register aliases to write **unified unsafe code** that works across all architectures:

| Alias | x86_64  | ARM64 | RISC-V | Purpose              |
|-------|---------|-------|--------|----------------------|
| `a`   | `rax`   | `x0`  | `a0`   | Accumulator/arg 0    |
| `b`   | `rbx`   | `x1`  | `a1`   | Base register/arg 1  |
| `c`   | `rcx`   | `x2`  | `a2`   | Count register/arg 2 |
| `d`   | `rdx`   | `x3`  | `a3`   | Data register/arg 3  |
| `e`   | `rsi`   | `x4`  | `a4`   | Source index/arg 4   |
| `f`   | `rdi`   | `x5`  | `a5`   | Dest index/arg 5     |
| `s`   | `rsp`   | `sp`  | `sp`   | Stack pointer        |
| `p`   | `rbp`   | `fp`  | `fp`   | Frame pointer        |

**Unified Example:**
```flap
// Works on ALL architectures - no per-CPU blocks needed!
value := unsafe {
    a <- 0x2A    // Load 42 into accumulator
    a            // Return accumulator
}
```

### Syntax

#### Per-Architecture Blocks

Specify different implementations for each CPU with labeled blocks:

```flap
result := unsafe {
    x86_64 {
        rax <- 100
        rbx <- rax
        rbx
    }
    arm64 {
        x0 <- 100
        x1 <- x0
        x1
    }
    riscv64 {
        a0 <- 100
        a1 <- a0
        a1
    }
}
```

#### Unified Blocks (Recommended)

Use register aliases for portable code:

```flap
result := unsafe {
    a <- 100     // Works everywhere
    b <- a
    b
}
```

### Operations

#### Immediate Loads
```flap
a <- 42          // Decimal
a <- 0xFF        // Hexadecimal
a <- 0b1010      // Binary
```

#### Register Moves
```flap
a <- 100
b <- a           // Copy a to b
c <- b           // Copy b to c
```

#### Arithmetic Operations
```flap
a <- 10
b <- 20
c <- a + b       // Addition
d <- a - b       // Subtraction
e <- a * b       // Multiplication
f <- a / b       // Division (unsigned)
```

#### Bitwise Operations
```flap
a <- 0xFF
b <- 0x0F
c <- a & b       // AND
d <- a | b       // OR
e <- a ^ b       // XOR
f <- ~a          // NOT
```

#### Shifts and Rotates
```flap
a <- 8
b <- a << 2      // Shift left
c <- a >> 1      // Shift right
d <- a rol 4     // Rotate left
e <- a ror 2     // Rotate right
```

#### Memory Access
```flap
// Load from memory
a <- [b]                // Load 64-bit from address in b
a <- [b + 16]           // Load from b + offset
a <- u8 [b]             // Load 8-bit (zero-extended)
a <- u16 [b + 4]        // Load 16-bit + offset
a <- u32 [b]            // Load 32-bit
a <- i8 [b]             // Load signed 8-bit
a <- i16 [b]            // Load signed 16-bit
a <- i32 [b]            // Load signed 32-bit

// Store to memory
[a] <- 42               // Store immediate
[a + 8] <- b            // Store register to offset
[a] <- b as u8          // Store 8-bit
[a] <- b as u16         // Store 16-bit
[a] <- b as u32         // Store 32-bit
```

#### Stack Operations
```flap
stack <- a       // Push a onto stack
b <- stack       // Pop stack into b
```

#### System Calls
```flap
// Set up syscall arguments, then invoke
a <- 1           // Syscall number (write)
b <- 1           // File descriptor (stdout)
c <- addr        // Buffer address
d <- len         // Buffer length
syscall          // Invoke syscall
```

### Return Values

The **last expression** in an unsafe block is the return value:

```flap
result := unsafe {
    a <- 42
    b <- a * 2
    b            // Returns b (84)
}
```

#### Type Casting Returns

Cast return values to C types:

```flap
ptr := unsafe {
    a <- 0x1000
    a as pointer     // Return as pointer
}

text := unsafe {
    a <- string_addr
    a as cstr        // Return as C string
}
```

### Examples

#### Example 1: Simple Arithmetic
```flap
sum := unsafe {
    a <- 10
    b <- 32
    c <- a + b
    c
}
printf("Sum: %v\n", sum)  // Output: Sum: 42
```

#### Example 2: Bitwise Magic
```flap
// Fast power-of-2 check
is_power_of_2 := unsafe {
    a <- 16
    b <- a - 1
    c <- a & b
    c            // Returns 0 if power of 2
}
```

#### Example 3: Memory Manipulation
```flap
// Allocate buffer
buf_size := 1024
buffer := malloc(buf_size)

// Write to buffer
unsafe {
    a <- buffer
    [a] <- 0x4141414141414141 as u64      // Write "AAAAAAAA"
    [a + 8] <- 0x4242424242424242 as u64  // Write "BBBBBBBB"
}

// Read back
first := unsafe {
    a <- buffer
    b <- [a]
    b
}

printf("First 8 bytes: 0x%x\n", first)
free(buffer)
```

#### Example 4: System Call (Per-CPU)
```flap
// Write "Hello\n" to stdout - platform-specific syscalls
msg := "Hello\n"

unsafe {
    x86_64 {
        rax <- 1             // sys_write
        rdi <- 1             // stdout
        rsi <- msg as cstr   // buffer
        rdx <- 6             // length
        syscall
    }
    arm64 {
        x8 <- 64             // sys_write on ARM64
        x0 <- 1              // stdout
        x1 <- msg as cstr    // buffer
        x2 <- 6              // length
        syscall
    }
    riscv64 {
        a7 <- 64             // sys_write on RISC-V
        a0 <- 1              // stdout
        a1 <- msg as cstr    // buffer
        a2 <- 6              // length
        syscall
    }
}
```

#### Example 5: Unified Cross-Platform Code
```flap
// Same code works on x86_64, ARM64, and RISC-V!
factorial_5 := unsafe {
    a <- 5          // Input
    b <- 1          // Result

    // Loop would go here (simplified)
    c <- a * b
    d <- c * 4
    e <- d * 3
    f <- e * 2
    f              // Return 120
}

printf("5! = %v\n", factorial_5)
```

### Safety Considerations

1. **No Type Safety**: Unsafe blocks bypass Flap's type system
2. **No Bounds Checking**: Memory access is unchecked
3. **Platform Specific**: Code may behave differently across architectures
4. **Manual Stack Management**: Push/pop must be balanced
5. **Syscall Conventions Vary**: Different syscall numbers per OS/arch

Use unsafe blocks only when:
- Performance is critical
- Interfacing with hardware
- Implementing low-level primitives
- Syscalls are required

For most code, use Flap's safe high-level features instead.

### Advanced Topics

#### Interfacing with C
```flap
// Call C function that expects pointer
c_func_ptr := unsafe {
    a <- data_buffer
    a as pointer
}
c_function(c_func_ptr)
```

#### Custom Allocators
```flap
// Implement bump allocator
alloc := x => unsafe {
    a <- heap_ptr        // Current heap position
    b <- a + x           // New position
    heap_ptr <- b        // Update heap pointer
    a as pointer         // Return old position
}
```

#### Atomic Operations
```flap
// LOCK prefix on x86_64
counter := unsafe {
    a <- counter_addr
    b <- [a]
    c <- b + 1
    [a] <- c    // Note: actual atomics need LOCK prefix
    c
}
```




---

# Implementation Notes

# Flapc Compiler Learnings

## Stack Alignment in x86-64

### The 16-byte Alignment Rule

The x86-64 System V ABI requires the stack pointer (rsp) to be aligned to 16 bytes **before** making any function call. This is critical when calling external functions like malloc, printf, etc.

### How to Calculate Stack Alignment

When a function is called, the CPU automatically pushes the return address (8 bytes). So at function entry, rsp is misaligned by 8 bytes.

Stack layout after various operations:
- After `call`: +8 bytes (misaligned - now at 8-byte boundary)
- After `push rbp`: +8 bytes (aligned - now at 16-byte boundary)
- After each `push`: +8 bytes per register

**Example calculation:**
```
call instruction         : +8  (total: 8,  misaligned)
push rbp (prologue)      : +8  (total: 16, aligned)
push r12                 : +8  (total: 24, misaligned)
push r13                 : +8  (total: 32, aligned)
push r14                 : +8  (total: 40, misaligned)
push r15                 : +8  (total: 48, aligned)
push rbx                 : +8  (total: 56, misaligned)
push rdi                 : +8  (total: 64, aligned)
```

Before calling malloc or any external function, count your stack usage. If it's misaligned (not a multiple of 16), subtract 8 more bytes from rsp.

### The Bug Pattern

In `flap_string_to_cstr` (parser.go line ~7520), we had:

```go
// BUGGY CODE (removed):
fc.out.SubImmFromReg("rsp", StackSlotSize)  // Sub 8
fc.out.MovXmmToMem("xmm0", "rsp", 0)
fc.out.MovMemToReg("r12", "rsp", 0)
fc.out.AddImmToReg("rsp", StackSlotSize)    // BUG: Added back too early!
```

At this point:
- call (8) + 6 pushes (48) = 56 bytes on stack
- 56 is not a multiple of 16 (misaligned!)
- The `sub rsp, 8` made it 64 bytes (aligned)
- But then we added it back before calling malloc
- malloc was called with misaligned stack → segfault or garbage data

**Fix:** Keep the stack aligned through the malloc call:

```go
// FIXED CODE:
fc.out.SubImmFromReg("rsp", StackSlotSize)  // Sub 8, now aligned
fc.out.MovXmmToMem("xmm0", "rsp", 0)
fc.out.MovMemToReg("r12", "rsp", 0)
// Keep rsp subtracted - restored later at line 7659
```

### General Principle

**Always verify stack alignment before calling external functions:**

1. Count bytes on stack: call(8) + pushes(8*N) + local_space
2. If total % 16 ≠ 0, subtract 8 more from rsp
3. Keep stack aligned until after the call returns
4. Restore rsp after the call completes

### Debugging Stack Alignment

If you see segfaults or garbage data from malloc/printf/etc:
1. Check stack alignment before the call
2. Use gdb: `info registers` and check rsp value
3. rsp & 0xF should equal 0 (bottom 4 bits zero)
4. Use ndisasm to verify generated assembly

### Impact

Incorrect stack alignment causes:
- Segmentation faults in external functions
- Garbage/corrupted return values
- Undefined behavior in SSE/AVX instructions (they require alignment)
- Intermittent bugs that are hard to reproduce

## Helper Function for Aligned malloc Calls

To make stack alignment easier and prevent bugs, we created a helper function:

```go
func (fc *FlapCompiler) callMallocAligned(sizeReg string, pushCount int)
```

**Parameters:**
- `sizeReg`: Register containing the allocation size (will be moved to rdi)
- `pushCount`: Number of registers pushed after the function prologue (not including `push rbp`)

**What it does:**
1. Calculates current stack usage: 16 + (8 * pushCount)
2. Checks if alignment is needed (total % 16 != 0)
3. Moves size to rdi (first argument for malloc)
4. Subtracts 8 from rsp if needed for alignment
5. Calls malloc
6. Restores rsp if it was adjusted
7. Returns allocated pointer in rax

**Usage example:**
```go
// Function with 5 register pushes after prologue
fc.out.PushReg("rbx")
fc.out.PushReg("r12")
fc.out.PushReg("r13")
fc.out.PushReg("r14")
fc.out.PushReg("r15")

// Allocate 512 bytes
fc.out.MovImmToReg("rax", "512")
fc.callMallocAligned("rax", 5) // 5 pushes
// Result is in rax
```

This replaces the manual alignment pattern:
```go
// OLD WAY (manual):
fc.out.SubImmFromReg("rsp", StackSlotSize)  // For alignment
fc.out.MovRegToReg("rdi", "rax")
fc.trackFunctionCall("malloc")
fc.eb.GenerateCallInstruction("malloc")
fc.out.AddImmToReg("rsp", StackSlotSize)  // Restore

// NEW WAY (helper):
fc.callMallocAligned("rax", pushCount)
```

The helper automatically handles alignment, making code clearer and preventing mistakes.

## When Stack Alignment Is Needed

### Main Function Context (Already Aligned)

In the main function generated by Flap, the stack is pre-aligned:
```
_start:
  // RSP is 16-byte aligned (kernel guarantee)
  jmp main

main:
  push rbp           // RSP now at (16n - 8)
  mov rbp, rsp
  // No further adjustment needed
```

After `push rbp`, RSP is at (16n - 8). When we make a C function call:
- `call` pushes return address (8 bytes) → RSP becomes 16n (aligned!)
- Function prologue in C function maintains alignment

So **C function calls from the main function don't need manual alignment**.

### Runtime Helper Functions (Need Alignment)

Runtime helpers we generate (like `flap_string_to_cstr`, `flap_cache_insert`, etc.) have their own prologue and often push registers:

```
flap_helper:
  call             // +8 (RSP = 16n - 8)
  push rbp         // +8 (RSP = 16n)
  mov rbp, rsp
  push r12         // +8 (RSP = 16n - 8)
  push r13         // +8 (RSP = 16n)
  push r14         // +8 (RSP = 16n - 8)
  push r15         // +8 (RSP = 16n)
  push rbx         // +8 (RSP = 16n - 8) -- MISALIGNED!

  // Calling malloc here would crash!
```

After an odd number of pushes (after the prologue), RSP is misaligned. We need to:
1. Count the pushes
2. If count is odd, subtract 8 before calling C functions
3. Restore after the call

**This is where `callMallocAligned(sizeReg, pushCount)` is essential.**

### General Rule

- **Main function → C function**: Already aligned, no action needed
- **Runtime helper → C function**: Must use alignment helper or manually align
- **Runtime helper → runtime helper**: Each function handles its own alignment

The helper function automatically calculates: `(16 + 8*pushCount) % 16 != 0`

## Register Clobbering and the Stack-First Principle

### The Problem

Registers are volatile across function calls. Any XMM register (xmm0-xmm15) or general-purpose register can be clobbered when evaluating sub-expressions that contain function calls.

**Example of the bug pattern:**
```go
// BUGGY CODE (removed from binary operations):
fc.compileExpression(e.Left)           // Result in xmm0
fc.out.MovRegToReg("xmm2", "xmm0")     // Save left in xmm2
fc.compileExpression(e.Right)          // May call functions that clobber xmm2!
fc.out.MovRegToReg("xmm0", "xmm2")     // BUG: xmm2 is corrupted!
```

This manifested in expressions like `n * factorial(n - 1)`, where:
1. `n` is evaluated and stored in xmm2
2. `factorial(n - 1)` is evaluated, which recursively uses xmm registers
3. When control returns, xmm2 contains garbage, not `n`
4. The multiplication uses corrupted values

### The Solution: Stack-First Principle

**Always use the stack for intermediate values across sub-expression evaluations:**

```go
// CORRECT CODE (current implementation):
fc.compileExpression(e.Left)           // Result in xmm0
fc.out.SubImmFromReg("rsp", 16)        // Allocate stack space
fc.out.MovXmmToMem("xmm0", "rsp", 0)   // Save left to stack
fc.compileExpression(e.Right)          // Safe - can use any registers
fc.out.MovRegToReg("xmm1", "xmm0")     // Right in xmm1
fc.out.MovMemToXmm("xmm0", "rsp", 0)   // Restore left from stack
fc.out.AddImmToReg("rsp", 16)          // Clean up
// Now perform operation with xmm0 and xmm1
```

### When Registers Are Safe vs. Unsafe

**Safe to use registers:**
- Within a single basic block with no function calls
- For immediate operations (e.g., `movsd xmm1, xmm0` followed by `addsd xmm0, xmm1`)
- For results that are used immediately before any function call

**Must use stack:**
- Across sub-expression evaluations that might contain function calls
- Across loop iterations where the loop body might call functions
- When the value needs to survive a function call

### General Guidelines

1. **Default to stack-based storage** for any value that needs to persist across sub-expression evaluation
2. **Only optimize to registers** when you can prove no function calls intervene
3. **Document assumptions** when using register storage (e.g., "safe because no calls in this basic block")
4. **Use descriptive comments** like "Save to stack (registers may be clobbered by function calls)"

### x86-64 Calling Convention Register Usage

According to System V ABI, these registers are caller-saved (clobbered by function calls):
- **General purpose**: rax, rcx, rdx, rsi, rdi, r8-r11
- **XMM registers**: xmm0-xmm15 (all volatile)

These are callee-saved (preserved across calls):
- **General purpose**: rbx, rbp, r12-r15
- **XMM registers**: None! All XMM registers are caller-saved

**Implication:** XMM registers are NEVER safe across function calls. Always use stack.

### Performance Considerations

Stack operations are fast (L1 cache) and the slight overhead is negligible compared to:
- The complexity of register liveness analysis
- The difficulty of debugging register corruption bugs
- The risk of subtle, hard-to-reproduce errors

**Premature optimization**: Trying to "optimize" by using registers for intermediate values often leads to bugs that cost far more time to debug than the microseconds saved.

### Code Patterns and Helpers

**Helper function for safe binary operations:**

```go
// Use this helper instead of manually managing registers:
func (fc *FlapCompiler) compileBinaryOpSafe(left, right Expression, operator string)
```

This helper encapsulates the stack-first pattern and should be used whenever possible.

**Comment template for manual implementations:**

When you must manually implement expression compilation with sub-expressions, use this comment pattern:

```go
// Compile left operand
fc.compileExpression(leftExpr)
// Save to stack (registers may be clobbered by sub-expression evaluation)
fc.out.SubImmFromReg("rsp", 16)
fc.out.MovXmmToMem("xmm0", "rsp", 0)
// Compile right operand (safe - can use any registers)
fc.compileExpression(rightExpr)
// Restore left operand from stack
fc.out.MovMemToXmm("xmm1", "rsp", 0)
fc.out.AddImmToReg("rsp", 16)
// Now xmm0 has right, xmm1 has left - ready to use
```

**Red flags to watch for:**

These patterns are potential bugs:
- `fc.out.MovRegToReg("xmm2", "xmm0")` followed by `fc.compileExpression(...)` - xmm2 will likely be clobbered
- Saving to XMM registers (xmm2-xmm15) across `fc.compileExpression()` calls
- Assuming any XMM register preserves its value across function calls
- Using XMM registers for "temporary" storage without checking call paths

**Safe patterns:**

These patterns are safe:
- Stack-based storage: `SubImmFromReg` → `MovXmmToMem` → ... → `MovMemToXmm` → `AddImmToReg`
- Using callee-saved general-purpose registers (rbx, r12-r15) but ONLY in functions you control the prologue/epilogue for
- Register-to-register moves within a single basic block with no function calls

## Nested Loop Implementation Design

**Problem:** Supporting arbitrary depth nested loops where each loop maintains its counter, limit, and iterator variable.

**Failed Approach:** Register-based storage using r12/r13
- Works for 2 levels but fails for 3+ because only saves the immediately outer loop's registers
- Push/pop pattern creates LIFO order: push A's registers → push B's registers → pop B's registers → pop A's registers
- Inner loop restore happens before outer loop completes

**Correct Solution:** Stack-based storage
```
Each loop allocates dedicated stack space (32 bytes, 16-byte aligned):
- [rsp + 0]:  counter (current iteration value)
- [rsp + 8]:  limit (end value)  
- [rsp + 16]: (padding for alignment)

Loop execution:
1. Allocate stack space: sub rsp, 32
2. Store counter/limit to stack
3. Load counter/limit to r12/r13 for loop condition checks
4. Update counter, store back to stack
5. Nested loops allocate their own stack slots
6. Deallocate on exit: add rsp, 32
```

**Key insight:** Each nested loop level has isolated stack slots, preventing interference.

**Files:** `parser.go:4419-4516` (compileRangeLoop function)

## Stack Alignment and Printf Bug

**Problem:** SIGBUS crashes when calling printf after nested loops.

**Root Cause #1:** Range loops allocated 24 bytes (not 16-byte aligned), violating x86-64 ABI requirement.

**Root Cause #2:** Printf had buggy alignment code:
```go
// WRONG - r10 is caller-saved, gets clobbered by printf
fc.out.MovRegToReg("r10", "rsp")
fc.out.AndImm("rsp", -16)
// ... call printf ...
fc.out.MovRegToReg("rsp", "r10")  // BROKEN: r10 was clobbered!
```

**Solution:**
1. Changed range loop allocation from 24 to 32 bytes (16-byte aligned)
2. Removed buggy printf alignment code - no longer needed since stack is always aligned

**Lesson:** Stack must be 16-byte aligned before any function call. Use proper multiples (16, 32, 48, ...) for stack allocations.

## Variable Scoping and Priority Order in Optimization Passes

When resolving variable/parameter references during optimization, the priority order is:

1. **Lambda parameters** - Highest priority, shadows all outer scopes
2. **Loop iterators** - Local to loop scope
3. **Local variables** - Variables defined in the current scope
4. **Outer scope variables** - Variables from enclosing scopes
5. **Constants** - Constant propagation applies last, only if not shadowed

### Constant Propagation and Lambda Scoping

**Critical Rule:** Lambda parameters must shadow outer variables during constant propagation.

**Bug Pattern:**
```flap
x := 10.5              // Outer variable marked as constant
square := x => x * x   // Lambda parameter 'x'
square(4.0)            // WRONG: returns 110.25 (10.5 * 10.5)
                       // RIGHT: should return 16 (4.0 * 4.0)
```

**Cause:** Constant propagation replaced lambda parameter `x` with outer constant `10.5`.

**Solution:** When propagating into lambda bodies, temporarily remove lambda parameters from the constants map:

```go
case *LambdaExpr:
    savedConstants := make(map[string]Expression)
    for _, param := range e.Params {
        if oldVal, existed := cp.constants[param]; existed {
            savedConstants[param] = oldVal
            delete(cp.constants, param)
        }
    }

    // Propagate into body with parameters shadowing outer constants
    if newBody, bodyChanged := cp.propagateInExpr(e.Body); bodyChanged {
        e.Body = newBody
    }

    // Restore outer constants
    for param, oldVal := range savedConstants {
        cp.constants[param] = oldVal
    }
```

### Mutation Tracking in Expressions

Constant propagation must detect mutations that occur within expressions, not just in assignment statements.

**Mutations can occur in:**
- Match expression branches: `n % 2 == 0 { -> n <- n / 2 }`
- Block expressions
- Lambda bodies
- Binary expressions with `<-` operator
- Postfix expressions: `steps++`

**Implementation:** Add `findMutationsInExprWithDepth()` that recursively searches expressions with depth limiting (max 100 levels) to prevent infinite recursion.

**Example requiring mutation tracking:**
```flap
n := 27
n % 2 == 0 {
    -> n <- n / 2      // Mutation in match branch
    ~> n <- (3*n) + 1  // Must be detected
}
```

### Dead Code Elimination Expression Handling

DCE must track variable usage in all expression types:

**Critical expression types:**
- `FStringExpr` - F-string interpolations: `f"Hello {name}"`
- `DirectCallExpr` - Direct function calls
- `NamespacedIdentExpr` - Dot notation: `data.field`
- `PostfixExpr` - Postfix operations: `i++`
- `VectorExpr` - Vector literals
- `LoopExpr` - Loop expressions
- `MultiLambdaExpr` - Pattern matching lambdas

**Bug Pattern:** Variable marked as unused and removed, causing "undefined variable" errors.

**Solution:** Add cases in `markUsedInExpr()` for all expression types that can reference variables.

### Loop Unrolling State Expression Handling

When unrolling loops with loop state expressions (`@i`, `@i1`, `@i2`):

**Loop Level Semantics:**
- `@i` (LoopLevel=0) - Current loop iterator
- `@i1` (LoopLevel=1) - Outermost loop iterator
- `@i2` (LoopLevel=2) - Second level loop iterator
- etc.

**Unrolling Rules:**
1. Only unroll loops with constant bounds and ≤ 8 iterations
2. Check if loop contains nested loops before substitution
3. When unrolling:
   - Replace `@i1` (LoopLevel=1) with iteration value
   - Decrement LoopLevel for `@i2+` (LoopLevel>1)
   - Only replace `@i` (LoopLevel=0) if no nested loops

**Example:**
```flap
@ i in 0..<3 {              // Outer loop
    @ j in 10..<12 {         // Inner loop
        printf("@i1=%v, @i2=%v, @i=%v", @i1, @i2, @i)
    }
}
```

After outer loop unrolls:
- `@i1` → 0, 1, 2 (replaced with values)
- `@i2` → `@i1` (LoopLevel decremented from 2 to 1)
- `@i` → `@i` (stays as-is, will be replaced when inner loop unrolls)

### Recursion Safety

All recursive AST traversals must include depth limiting to prevent stack overflow on malformed or adversarial input.

**Implementation Pattern:**
```go
const maxRecursionDepth = 100

func traverse(node Node, depth int) {
    if depth > maxRecursionDepth {
        return  // Or return error
    }
    // Process node...
    traverse(child, depth+1)
}
```

**Apply to:** findMutations, propagateInExpr, markUsedInExpr, and any other recursive AST traversal.

## macOS ARM64 Execution Issue (2025-10-26)

### Discovery
macOS ARM64 binaries generated by flapc are structurally valid Mach-O files but are killed with SIGKILL (exit code 137) before code execution. Even `codesign` reports "failed strict validation".

### Progress
1. Fixed LINKEDIT segment size calculation - was including 4KB code signature space when none was written
2. Added LC_CODE_SIGNATURE load command with 4KB reserved space (zeros)
3. Binary now has proper structure with filesize matching actual data

### Fixed Issues (Session 2)
1. __LINKEDIT segment now always generated on macOS (not just for dynamic linking)
2. LC_SYMTAB always written (required for code signature)
3. LC_UUID load command added (dyld requirement)
4. LC_BUILD_VERSION updated to match system version (26.0 for Sequoia)
5. LINKEDIT structures properly sized and written

### Current Status
**Simple binaries (no function calls):**
- Binary structure is valid per `otool`
- `codesign` completes without error but reports "no signature" after
- Execution: SIGKILL (137) - macOS kills unsigned binaries
- Issue: codesign can't sign our binaries for unknown reason

**Binaries with dynamic linking:**
- When MH_DYLDLINK flag set, binary loads but crashes with SIGSEGV (139)
- dyld error: tries to read relocation data but derefs NULL pointer
- Crash in `dyld::forEachRebase_Relocations` at address 0x48
- Issue: Dynamic linking enabled but no relocation structures provided

### Key Findings
1. macOS 26 (Sequoia) has stricter requirements than macOS 12
2. Even simple binaries from Clang link to libSystem.B.dylib
3. Clang uses LC_DYLD_CHAINED_FIXUPS (we removed this for lazy binding)
4. When dynamic linking enabled, dyld expects valid relocation data
5. codesign tool can't create signature - binary structure issue?

### Fixed Issues (Session 3) - Dynamic Linking with GOT/Stubs

**Root Cause:** Two critical issues prevented dynamic linking from working:

1. **Incorrect LINKEDIT section order**
   - We wrote: `symtab → strtab → indirect symtab → code signature`
   - Apple expects: `symtab → indirect symtab → strtab → code signature`
   - When `ldid` signed the binary, it assumed Apple's order and overwrote our indirect symbol table with zeros
   - Result: GOT/stub entries pointed to wrong symbols, causing printf to fail

2. **Missing two-level namespace library ordinal**
   - Symbol table N_desc field was set to 0 for undefined symbols
   - Should be: `(library_ordinal << 8)` where libSystem.B.dylib = ordinal 1
   - Result: dyld couldn't find which library provides printf
   - Error: "Symbol not found: _printf, Expected in: <binary>" instead of "Expected in: libSystem"

**The Fix:**

1. Reordered LINKEDIT sections to match Apple's expected layout:
```go
// Correct order in WriteMachO():
// 1. Write symbol table
// 2. Write indirect symbol table (for dynamic linking)
// 3. Write string table
// 4. Reserve code signature space
```

2. Set correct N_desc for undefined symbols:
```go
sym := Nlist64{
    N_strx:  strOffset,
    N_type:  N_UNDF | N_EXT,
    N_sect:  0,
    N_desc:  uint16(1 << 8), // Dylib ordinal 1 (libSystem.B.dylib)
    N_value: 0,
}
```

**Result:** Dynamic linking now works! Binaries using printf execute correctly with exit code 0.

**Files Modified:**
- `macho.go` lines 955-1013: Updated offset calculations for new LINKEDIT order
- `macho.go` lines 1176-1193: Reordered actual writing of LINKEDIT sections
- `macho.go` line 637: Set N_desc with library ordinal for two-level namespace

**Validation:**
```bash
$ ./flapc testprograms/const_test.flap
$ ./const_test
0
$ echo $?
0
```

**Key Learning:** The order of sections in LINKEDIT matters! Tools like `ldid` make assumptions about the layout and will corrupt data if the order doesn't match Apple's conventions. Always check:
1. What offset calculations assume
2. What order data is actually written
3. What external tools (like ldid) expect

---

## Network Send Operator: Evolution from `<-` to `<==` (2025-01-27)

**Problem 1:** Initial ENet design used `<-` for network send (`:5000 <- "message"`), but this created ambiguity with variable updates (`x <- x + 1`). When the left-hand side is a variable holding a port number, parser cannot distinguish:
```flap
port := :5000
port <- "message"  // Send? Or update variable?
```

**Solution 1:** Use `<=` operator for network send operations.

**Problem 2:** Using `<=` for sends created confusion with the comparison operator (`x <= 10`). While technically unambiguous to the parser, it's confusing for developers reading code.

**Final Solution:** Use `<==` operator for network send operations.

**Rationale:**
- **No ambiguity**: `<==` is distinct from both `<-` (variable update) and `<=` (comparison)
- **Visually intuitive**: Three characters suggest "send toward" or "push to"
- **No operator overloading**: Dedicated operator for dedicated purpose
- **Parser simplicity**: Dedicated token (TOKEN_SEND) with no dual purposes

**Implementation:**
```flap
// Variable update
x <- x + 1

// Comparison
if x <= 10 { }

// Network send
:5000 <== "hello"                // Send to port
port <== "message"               // Send to variable containing port
"server.com:5000" <== "data"     // Send to remote address
```

**Files Modified:**
- `lexer.go`: Added TOKEN_SEND for `<==`
- `ast.go`: SendExpr uses `<==` in String()
- `parser.go`: parseSend() checks TOKEN_SEND instead of TOKEN_LE
- `LANGUAGE.md`: Updated all examples and grammar rules

**Key Learning:** When overloading operators, choose combinations that minimize ambiguity. If context-based resolution requires complex lookahead, consider using a different operator entirely.
## Port Addresses: Strings vs. Special Literals (2025-01-27)

**Problem:** Initial design used special port literal syntax (`:5000`, `:worker`) with TOKEN_PORT and PortExpr AST nodes. This required:
- Special lexer handling with bracket depth tracking
- Dedicated AST node type
- Compile-time hashing for named ports
- Complex parsing to avoid conflicts with slice syntax `[0:2]`

**Solution:** Use regular strings for port addresses: `":5000"`, `"localhost:5000"`

**Rationale:**
- **Simpler parser**: No special token or AST node needed
- **Familiar syntax**: Strings are already well-understood
- **Flexible format**: Easy to extend to "host:port" format
- **Less ambiguity**: No conflict with colon in slices/maps
- **Runtime flexibility**: Can support variables holding addresses (future enhancement)

**Implementation:**
```flap
// Before (special syntax)
:5000 <== "hello"
:worker <== "data"

// After (strings)
":5000" <== "hello"
":8080" <== "data"
"server.com:5000" <== "remote"
```

**Files Modified:**
- `lexer.go`: Removed TOKEN_PORT, readPortLiteral(), bracket depth tracking
- `ast.go`: Removed PortExpr type
- `parser.go`: Removed portToNumber(), simplified compileSendExpr to parse string literals
- `LANGUAGE.md`: Updated all examples to use strings

**Key Learning:** Sometimes the simplest solution is to reuse existing language features rather than adding special syntax. Strings are flexible and well-understood - no need for custom literals.

## Futex Barriers and Parallel Loop Synchronization

### The Challenge: Thread Synchronization Without Pthreads

When implementing parallel loops (`@@` and `N @`), we needed a way to synchronize threads without linking to pthread. The goal: spawn threads via raw `clone()` syscalls and coordinate completion using only kernel primitives.

### Atomic Operations: The Foundation

**Learning 1: LOCK XADD is your friend for atomic decrements**

The key insight: futex barriers need atomic counter operations. The x86-64 `LOCK XADD` instruction is perfect for this:

```asm
mov eax, -1
lock xadd [barrier_addr], eax  ; Atomically: tmp=mem; mem+=eax; eax=tmp
dec eax                         ; eax now has new value
test eax, eax                   ; Check if we're last thread
```

**Why LOCK XADD over LOCK DEC?**
- LOCK XADD returns the OLD value, letting us know the NEW value after decrement
- LOCK DEC doesn't return any value, only sets flags (harder to use)
- Pattern: `old = atomic_add(ptr, -1); new = old - 1; if (new == 0) { last_thread(); }`

**Implementation in atomic.go:**
- Emits proper REX prefix for 64-bit registers
- Uses ModR/M encoding for memory operands with displacements
- Supports x86-64 (LOCK XADD), ARM64 (LDADD), RISC-V (AMO instructions)

### Futex Syscalls: Linux Fast Userspace Mutex

**Learning 2: FUTEX_PRIVATE_FLAG gives you free performance**

Futex operation codes:
```go
FUTEX_WAIT = 0          // Block until woken
FUTEX_WAKE = 1          // Wake N threads
FUTEX_PRIVATE_FLAG = 128 // Don't share across processes
```

Always use the PRIVATE variant for thread-only synchronization:
```go
FUTEX_WAIT_PRIVATE = 128  // 0 | 128
FUTEX_WAKE_PRIVATE = 129  // 1 | 128
```

The PRIVATE flag tells the kernel this futex won't be shared across processes, enabling optimizations:
- No need to hash into a global kernel table
- Faster lookup (process-local table only)
- ~10-20% better performance vs non-private futex

### Barrier Pattern: N-Thread Rendezvous

**Learning 3: Initialize counter to N, parent waits too**

Initial attempt (WRONG):
```go
// BUG: Only child decrements, parent never woken
barrier.count = 1
// Child: atomic_dec(count); if (count == 0) wake_parent();
// Parent: wait_on_futex(count);
// Problem: Parent waits on value 1, child sets it to 0 and wakes,
//          but if parent hasn't started waiting yet, wake is lost!
```

Correct pattern:
```go
// All threads participate in the barrier
barrier.count = num_threads + 1  // +1 for parent

// Child threads:
old = atomic_add(&barrier.count, -1)
if (old - 1 == 0) {  // Last one out
    futex(&barrier.count, FUTEX_WAKE_PRIVATE, barrier.total)
} else {
    futex(&barrier.count, FUTEX_WAIT_PRIVATE, expected_value)
}

// Parent:
futex(&barrier.count, FUTEX_WAIT_PRIVATE, current_value)
// Wakes when count reaches 0
```

**Current V4 Implementation:**
For simplicity, V4 spawns 1 child thread and parent waits:
```go
barrier.count = 1  // Only child participates
// Child decrements, wakes parent
// Parent waits until woken
```

This works but is limited. V5 will spawn N children and all will coordinate via the barrier.

### Memory Layout: Passing Data to Threads

**Learning 4: Store arguments on child stack BEFORE clone()**

The child thread needs to know:
- Work range: [start, end)
- Barrier address for synchronization

Solution: Write to child's stack before it starts:
```go
// Allocate 1MB stack
stack_top = mmap(NULL, 1MB, PROT_READ|PROT_WRITE, ...)

// Store arguments at negative offsets from stack top
[stack_top - 24] = start       // 8 bytes
[stack_top - 16] = end         // 8 bytes
[stack_top - 8]  = barrier_ptr // 8 bytes

// Adjust stack pointer
child_stack = stack_top - 24

// Clone with this stack
clone(CLONE_VM|..., child_stack, ...)
```

Child reads them back:
```asm
mov rbp, rsp           ; Set up frame pointer
mov r12, [rbp+0]       ; r12 = start
mov r13, [rbp+8]       ; r13 = end
mov r15, [rbp+16]      ; r15 = barrier_addr
```

### Clone Flags: Minimal Sharing for Threads

**Learning 5: CLONE_VM is mandatory, CLONE_THREAD is optional**

Required flags for threads:
```go
CLONE_VM        = 0x00000100  // Share memory space
CLONE_FS        = 0x00000200  // Share filesystem info
CLONE_FILES     = 0x00000400  // Share file descriptor table
CLONE_SIGHAND   = 0x00000800  // Share signal handlers
CLONE_SYSVSEM   = 0x00040000  // Share SysV semaphores
```

Optional but useful:
```go
CLONE_THREAD    = 0x00010000  // Thread group (same TGID)
```

Without CLONE_THREAD, each clone gets its own process ID but still shares memory. This is fine for our use case and simpler than managing thread groups.

### Debugging Parallel Code

**Learning 6: Use strace -f to trace all threads**

Essential for debugging:
```bash
# See all syscalls from parent and children
strace -f -e trace=clone,futex,exit ./program

# Output shows thread coordination:
clone(...) = 12345
[pid 12344] futex(..., FUTEX_WAIT_PRIVATE, 1) = <blocks>
[pid 12345] futex(..., FUTEX_WAKE_PRIVATE, 1) = 1
[pid 12344] <resumed>) = 0
```

The `<resumed>` line shows when a blocked syscall continues after being woken.

### Performance Considerations

**Learning 7: Thread overhead ~50μs per spawn**

Measured costs:
- Thread creation (mmap + clone): ~50μs
- Context switch: ~3μs
- Futex wake: ~1μs

**Recommendation:** Only parallelize loops with >1000 iterations or expensive bodies.

For a loop with 100 iterations:
- Sequential: 100 × 10μs = 1ms
- Parallel (2 threads): 2 × 50μs (spawn) + 50 × 10μs + 1μs (futex) = 601μs
- Speedup: 1.66× (not 2×) due to overhead

For a loop with 10000 iterations:
- Sequential: 100ms
- Parallel (2 threads): 0.1ms + 50ms + 0.001ms ≈ 50ms
- Speedup: 2× (overhead negligible)

### Key Insights Summary

1. **LOCK XADD over LOCK DEC**: Returns old value, enabling atomic decrement pattern
2. **FUTEX_PRIVATE_FLAG**: Always use for thread-local synchronization (10-20% faster)
3. **Barrier initialization**: Start with N threads, all participate in countdown
4. **Pass data via stack**: Store arguments on child stack before clone()
5. **Minimal clone flags**: CLONE_VM + CLONE_FS + CLONE_FILES is sufficient
6. **strace -f for debugging**: Essential for understanding multi-threaded execution
7. **Overhead analysis**: Thread spawn costs ~50μs, only profitable for heavy loops

### Files Created

- `atomic.go` (87 lines): LOCK XADD instruction for x86-64/ARM64/RISC-V
- `dec.go` (115 lines): DEC instruction for all architectures
- `parser.go` modifications: compileParallelRangeLoop() with V4 futex barriers

### References

- Linux futex(2) man page: https://man7.org/linux/man-pages/man2/futex.2.html
- Intel x86-64 LOCK prefix: Volume 2A, Section 2.1.2
- pthread_barrier_wait implementation in glibc: nptl/pthread_barrier_wait.c
- Go runtime scheduler: runtime/proc.go (similar barrier patterns)

## Why V5 (Full Loop Body Execution) Is Complex

### The Challenge: Separate Stacks Mean Separate Contexts

After successfully implementing V4 (futex barriers working), I attempted V5: executing the actual loop body statements (like `printf("Loop: %v\n", i)`) instead of just printing ASCII digits.

**V5 crashed immediately with SIGSEGV.**

### The Root Problem

Child threads created with `clone()` have their own separate stacks. This creates fundamental architectural challenges:

**What Doesn't Work:**
```go
// V5 attempt (FAILS):
fc.variables[stmt.Iterator] = iteratorOffset  // Register iterator
for _, s := range stmt.Body {
    fc.compileStatement(s)  // Compile loop body
}
```

**Why It Fails:**

1. **Stack-relative addressing breaks**: Parent's local variables are at offsets from parent's `rbp`. Child's `rbp` points to child's stack, so all offsets are wrong.

2. **Function calls need full runtime**: `printf()` and other builtins expect:
   - Proper stack frame setup
   - Correct calling conventions
   - Access to global data structures
   - String constants at correct addresses

3. **Variable access fails**: Loop body may reference parent's variables (e.g., `x <- x + 1`), but those are on parent's stack, inaccessible to child.

### What V4 Does Right

V4 works because it only does simple, self-contained operations:
```asm
; V4: Print ASCII digit (self-contained, no function calls)
mov rax, r14
add rax, 48        ; Convert to ASCII
mov [rsp], rax
mov rax, 1         ; sys_write
mov rdi, 1         ; stdout
mov rsi, rsp       ; buffer
mov rdx, 2         ; length
syscall            ; Direct syscall, no stack frame needed
```

No variables, no function calls, no stack frame dependencies. Just registers and syscalls.

### What V5 Would Actually Require

To support arbitrary loop body execution in child threads, we need:

**1. Shared Memory Arena**
```go
// Allocate shared memory for loop-accessible variables
arena := mmap(NULL, 4096, PROT_READ|PROT_WRITE, MAP_SHARED|MAP_ANONYMOUS)

// Store loop-local variables in shared arena, not on stack
iterator_ptr = arena + 0
temp_var_ptr = arena + 8
...
```

**2. Thread-Safe Built-ins**
- Reimplement `printf` to work from child thread context
- Or: use message passing to parent thread
- Or: use lock-protected shared stdio

**3. Position-Independent Code**
- All addresses must be absolute or RIP-relative
- No rbp-relative addressing for cross-thread data
- Function pointers must be globally accessible

**4. Proper Call Frame Setup**
```asm
; Child needs full function prologue
push rbp
mov rbp, rsp
sub rsp, <frame_size>
; Align stack to 16 bytes for calls
; Set up shadow space (Windows) or red zone (Linux)
```

**5. Variable Remapping**
```go
// Map parent's stack variables to shared memory locations
parentVars := fc.variables  // Save
fc.variables = make(map[string]int)

// Remap to shared memory offsets
fc.variables[stmt.Iterator] = sharedMemOffset(0)
// Other variables would need similar remapping
```

### Complexity Estimation

Implementing full V5 properly would require:
- **Shared memory allocator**: 50-100 lines
- **Thread-safe printf**: 100-200 lines or message queue
- **Variable remapping logic**: 50-100 lines
- **Call frame management**: 30-50 lines
- **Position-independent addressing**: Changes throughout codebase

**Total:** ~300-500 lines + architectural changes

### Current V4 Status

**What Works:**
- Thread spawning with mmap + clone()
- Futex barrier synchronization
- Work distribution across threads
- Simple self-contained operations in loops

**What Doesn't Work:**
- Function calls from child threads
- Accessing parent's local variables
- Complex loop bodies with string formatting
- Cross-thread shared state

### Recommended Path Forward

**Option A: Stay with V4**
- Document current limitations
- V4 is still useful for embarrassingly parallel workloads
- Simple operations (math, array updates) work fine

**Option B: Implement V6 (Multiple Threads) First**
- Spawn N threads instead of 1
- Each gets work range
- Still simple operations only
- Validates parallel performance

**Option C: Full V5 Later**
- Requires shared memory infrastructure
- Significant architectural changes
- Better done after V6 proves performance benefits

### Key Learning

**Don't underestimate separate stack complexity.** What seems like "just compile the loop body" actually requires:
- Shared memory management
- Thread-safe runtime functions
- Position-independent addressing
- Proper ABI compliance

V4's simple approach (direct syscalls, no shared state) is elegant precisely because it avoids these issues.


## Cross-Platform Register Aliases

### Design Decision: Two Syntax Modes

When implementing cross-platform support for unsafe blocks across x86-64, ARM64, and RISC-V, we faced a choice: how should users write portable assembly code?

**Challenge:** Each architecture uses different register names:
- x86-64: `rax`, `rbx`, `rcx`, `rdx`, `rsi`, `rdi`, `rsp`, `rbp`
- ARM64: `x0`, `x1`, `x2`, `x3`, `x4`, `x5`, `sp`, `fp`
- RISC-V: `a0`, `a1`, `a2`, `a3`, `a4`, `a5`, `sp`, `fp`

**Solution:** Support TWO distinct syntaxes:

### Unified Syntax (Recommended)

Single `unsafe { }` block using portable aliases:

```flap
result := unsafe {
    a <- 42      // Resolves to rax/x0/a0 based on target CPU
    b <- 100     // Resolves to rbx/x1/a1
    c <- a + b   // Works everywhere
    c
}
```

**Register alias mapping:**
| Alias | x86-64 | ARM64 | RISC-V | Purpose |
|-------|--------|-------|--------|---------|
| a | rax | x0 | a0 | Accumulator |
| b | rbx | x1 | a1 | Base |
| c | rcx | x2 | a2 | Count |
| d | rdx | x3 | a3 | Data |
| e | rsi | x4 | a4 | Source index |
| f | rdi | x5 | a5 | Destination index |
| s | rsp | sp | sp | Stack pointer |
| p | rbp | fp | fp | Frame pointer |

**Implementation:** `register_alias.go` with `resolveRegisterAlias()` function called during compilation.

### Per-CPU Syntax

Multiple labeled blocks for platform-specific code:

```flap
result := unsafe {
    x86_64 {
        rax <- 42
        rbx <- 100
        rcx <- rax + rbx
        rcx
    }
    arm64 {
        x0 <- 42
        x1 <- 100
        x2 <- x0 + x1
        x2
    }
    riscv64 {
        a0 <- 42
        a1 <- 100
        a2 <- a0 + a1
        a2
    }
}
```

**Use case:** When you need platform-specific behavior (e.g., different syscall numbers, CPU-specific instructions).

### Critical Design Rule: No Mixing

**WRONG:**
```flap
// DON'T DO THIS - mixes unified and per-CPU
result := unsafe {
    a <- 42      // Unified alias
} {
    x0 <- 42     // ARM64 specific
} {
    a0 <- 42     // RISC-V specific
}
```

This creates confusion: are we writing unified code or per-CPU code?

**Correct Options:**

Option 1 - Pure unified:
```flap
result := unsafe { a <- 42; a }
```

Option 2 - Pure per-CPU:
```flap
result := unsafe {
    x86_64 { rax <- 42; rax }
    arm64 { x0 <- 42; x0 }
    riscv64 { a0 <- 42; a0 }
}
```

### Key Learning: Language Design Clarity

**Lesson learned:** When designing cross-platform features, **clarity beats flexibility**.

Having two well-defined modes (unified vs per-CPU) is better than one flexible mode that mixes concerns. Users can:
1. Choose the **unified** approach for 95% of cases
2. Drop to **per-CPU** when they truly need platform-specific behavior

This prevents subtle bugs where users accidentally write non-portable code while thinking they're writing portable code.

### Documentation Impact

Created `UNSAFE.md` (355 lines) documenting both syntaxes with clear examples showing when to use each approach. Every example follows the "no mixing" rule.

### Testing Approach

Test both syntaxes independently:
- `test_alias_simple.flap`: Unified syntax test
- `test_register_alias.flap`: Both syntaxes shown separately

This ensures the separation is maintained in practice, not just documentation.


---

## Roadmap

**Version 1.6.0 (In Progress):**
- ✅ Process spawning with `spawn` keyword (Unix fork)
- ✅ Port literals for ENet (`:5000`, `:worker` with deterministic hashing)
- ⚙️  ENet networking protocol (socket operations, send/receive)
- ⚙️  Parallel loops (`N @` and `@@` for data parallelism - V4 complete with futex barriers)
- 🔜 Hot code reload integration (infrastructure complete)

**Completed in 1.5.x:**
- Tail-call optimization
- Arena memory management
- C FFI with DWARF auto-discovery
- Cross-platform unsafe blocks with register aliases (x86-64/ARM64/RISC-V)
- Pattern matching and lambdas

**Future:**
- Game development tooling (SDL3/RayLib5 examples)
- Multiplatform support (Windows/macOS/ARM64/RISC-V)
- Steamworks integration
- HTTP/WebSocket support

## License

BSD-3-Clause - Commercial use, packaging, modification allowed. No copyleft.

## Contributing

**Bug reports:** Provide minimal test case. See `testprograms/` for examples.

**Development:** Compiler developed with AI assistance (Claude). All code BSD-licensed, auditable, tested.

---

**Version:** 1.6.0-dev
**Refs:** System V ABI, ELF-64 spec, Intel x86-64 manual
