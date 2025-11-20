# Flapc - The Flap Programming Language Compiler

[![Go CI](https://github.com/xyproto/flapc/actions/workflows/ci.yml/badge.svg)](https://github.com/xyproto/flapc/actions/workflows/ci.yml) [![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/flapc)](https://goreportcard.com/report/github.com/xyproto/flapc)

**Flap** is a systems programming language that compiles directly to native machine code without LLVM, IR, or dependencies.

## An example Flap program, using SDL3 to display an image

```go
import sdl3 as sdl

// Window dimensions
width := 620
height := 387

// Initialize SDL3 with video subsystem
println("Initializing SDL3...")

// SDL3 returns true (1) on success, false (0) on failure
// Use or! for railway-oriented error handling
// Note: Blocks that return values are expressions (no exit needed - defer handles cleanup)
sdl.SDL_Init(sdl.SDL_INIT_VIDEO) or! {
    exitf("SDL_Init failed: %s\n", sdl.SDL_GetError())
}

// Use defer to ensure SDL_Quit is called when program exits
// Deferred calls execute in LIFO (Last In, First Out) order
defer sdl.SDL_Quit()

println("Creating window and renderer...")

// Create window - or! checks for null pointer (0) and returns error value if null
window := sdl.SDL_CreateWindow("Hello World!", width, height, sdl.SDL_WINDOW_RESIZABLE) or! {
    exitf("Failed to create window: %s\n", sdl.SDL_GetError())
}

// Defer window cleanup - will execute before SDL_Quit
defer sdl.SDL_DestroyWindow(window)

// Create renderer - clean error handling with or!
renderer := sdl.SDL_CreateRenderer(window, 0) or! {
    exitf("Failed to create renderer: %s\n", sdl.SDL_GetError())
}

// Defer renderer cleanup - will execute before window cleanup
defer sdl.SDL_DestroyRenderer(renderer)

printf("Loading BMP image...\n")

// Load BMP file - or! handles null file pointer
file := sdl.SDL_IOFromFile("img/grumpy-cat.bmp", "rb") or! {
    exitf("Error reading file: %s\n", sdl.SDL_GetError())
}

// Load BMP surface from file
bmp := sdl.SDL_LoadBMP_IO(file, 1) or! {
    exitf("Error creating surface: %s\n", sdl.SDL_GetError())
}

// Defer surface cleanup
defer sdl.SDL_DestroySurface(bmp)

// Create texture from surface
tex := sdl.SDL_CreateTextureFromSurface(renderer, bmp) or! {
    exitf("Error creating texture: %s\n", sdl.SDL_GetError())
}

// Defer texture cleanup - will execute first (LIFO)
defer sdl.SDL_DestroyTexture(tex)

println("Rendering for 2 seconds...")

// Main rendering loop - run for approximately 2 seconds (20 frames * 100ms = 2s)
@ frame in 0..<20 {
    // Clear screen
    sdl.SDL_RenderClear(renderer)

    // Render texture (fills entire window)
    sdl.SDL_RenderTexture(renderer, tex, 0, 0)

    // Present the rendered frame
    sdl.SDL_RenderPresent(renderer)

    // Delay to maintain framerate
    sdl.SDL_Delay(100)
}

println("Done!")
```

## What's special about Flap?

Flap is distinguished by these rare/unique design choices:

### 1. **One Type for Everything**
Everything IS `map[uint64]float64` — numbers, strings, lists, objects, functions. No type system complexity.

### 2. **Direct Machine Code from AST**
AST → x86-64/ARM64/RISC-V in one pass. No IR, no LLVM. Sub-millisecond compilation. ~30k lines of Go.

### 3. **Context-Sensitive Blocks**
`{...}` disambiguated by contents: `{x: 1}` = map, `{x => y}` = match, `{x := 1}` = statements. No `match` keyword needed.

### 4. **Minimal Keyword Design**
`ret @` for loop control (numbered labels `@1`, `@2`), `@` for loops, `->` for lambdas (inferred when obvious), named operators (`and`/`or`/`not`).

### 5. **Built-in Parallelism**
`@@` for parallel loops with automatic barrier sync. Native atomic operations. No thread management.

### 6. **Zero-Cost C FFI**
Direct C library calls: `c.malloc()`, `import sdl3 as sdl`. Automatic header parsing extracts constants: `sdl.SDL_INIT_VIDEO`. CStruct for C-compatible layouts.

### 7. **Immutable by Default**
`x = 42` is immutable, `x := 42` is mutable. Functions use `=` not `:=`.

### 8. **Arena Memory & Move Semantics**
`arena {...}` for scope-based cleanup. `x!` for ownership transfer. No GC pauses.

### 9. **List Methods**
Built-in list operations: `.length`, indexing, slicing. Lists are maps with sequential keys starting at 0.

### 10. **Cryptographic Random Built-in**
`???` generates secure random using OS CSPRNG. No setup required.

### 11. **Tail-Call Optimization Always On**
Recursive functions automatically optimized to loops.

### 12. **Self-Contained & Deterministic**
Zero dependencies, same input → same binary. Entire toolchain in one executable.

---

## Quick Start

### Installation

```bash
git clone https://github.com/xyproto/flapc
cd flapc
go build
sudo install -Dm755 flapc /usr/bin/flapc
```

### Hello World

```bash
echo 'println("Hello, World!")' > hello.flap
flapc hello.flap
./hello
```

### First Real Program

```flap
// Factorial with tail-call optimization
factorial = (n, acc) -> n == 0 {
    => acc
    ~> factorial(n-1, n*acc)
}

@ i in 0..<10 {
    println(factorial(i, 1))
}
```

## Language Features

### Variables and Assignment

```flap
x = 42        // Immutable (preferred - cannot reassign)
y := 100      // Mutable (rare - can reassign)
y <- y + 1    // Update operator
y++           // Increment (mutable vars only)
```

### Functions

```flap
square = x -> x * x
add = (a, b) -> a + b
greet = { println("Hello!") }  // No-arg lambda (inferred from block)
greet2 = -> println("Hello!")  // No-arg lambda (explicit ->)

// Match expression
classify = x -> x { 0 => "zero" | 1 => "one" | ~> "many" }

// Guard match
sign = x -> { | x < 0 => "neg" | x > 0 => "pos" | ~> "zero" }
```

### Loops with Numbered Labels

```flap
@ i in 0..<10 { println(i) }                  // Range
@ item in [1, 2, 3] { println(item) }         // Collection
@@ i in 0..<1000 { process(i) }               // Parallel

// Loop control with numbered labels (@1 = outer, @2 = inner)
@ i in 0..<10 {                   // Loop @1
    @ j in 0..<10 {               // Loop @2
        j == 5 { ret @ }          // Exit @2 (inner)
        i == 5 { ret @1 }         // Exit @1 (outer)
    }
}
```

### Bitwise Operations

```flap
x := 8
y := x <<b 1    // Shift left: 16
z := x >>b 1    // Shift right: 4
a := x <<<b 1   // Rotate left
b := x >>>b 1   // Rotate right

c := x &b 15    // AND
d := x |b 1     // OR
e := x ^b 255   // XOR
f := ~b x       // NOT
```

### C FFI

Direct calls to C libraries with automatic constant extraction:

```flap
// Standard C library
ptr = c.malloc(1024.0)
c.memset(ptr, 0, 1024.0)
c.free(ptr)

// SDL3 for graphics
import sdl3 as sdl

// Constants from SDL3 headers are automatically available
sdl.SDL_Init(sdl.SDL_INIT_VIDEO)  // SDL_INIT_VIDEO = 32
window := sdl.SDL_CreateWindow("Game", 800, 600, 0)
renderer := sdl.SDL_CreateRenderer(window, 0)

@ {
    sdl.SDL_RenderClear(renderer)
    // Game logic here
    sdl.SDL_RenderPresent(renderer)
    sdl.SDL_Delay(16)
}
```

**Automatic Constant Discovery:**
When you `import sdl3 as sdl`, Flapc automatically:
- Finds SDL3 header files using `pkg-config`, and parses them.
- Extracts `#define` constants (SDL_INIT_VIDEO, SDL_WINDOW_SHOWN, etc.)
- Makes them available via namespace syntax: `sdl.SDL_INIT_VIDEO`
- No manual constant definitions needed!

### CStruct - C-Compatible Structures

```flap
cstruct Vec3 {
    x as float32,
    y as float32,
    z as float32
}

// Compiler generates size and offset constants
ptr := c.malloc(Vec3.size)
write_f32(ptr + Vec3.x.offset, 1.0)
write_f32(ptr + Vec3.y.offset, 2.0)
write_f32(ptr + Vec3.z.offset, 3.0)
```

### Arena Allocation

Scope-based memory management:

```flap
arena {
    // All allocations freed at block exit
    buffer := alloc(1024)
    entities := alloc(100 * entity_size)

    // Work with memory...
}  // Everything freed automatically
```

### Parallel Programming

```flap
// Parallel map
@@ i in 0..<1000 {
    results[i] <- expensive_computation(data[i])
}

// Atomic operations
counter := c.malloc(8.0)
atomic_store(counter, 0)

@@ i in 0..<1000 {
    atomic_add(counter, 1)
}

result := atomic_load(counter)  // 1000
```

### Type Casting

```flap
x := 42.7

// Integer types
i := x as int32      // 42
u := x as uint64     // 42

// Float types
f := x as float32    // 42.700000

// Pointer types
ptr := x as ptr
str := "hello" as cstr
```

### Operators

```flap
// Arithmetic
+  -  *  /  %  **  (power)

// Comparison
==  !=  <  <=  >  >=

// Logical
and  or  xor  not

// List operators
::     // Cons (prepend): 1 :: [2, 3] -> [1, 2, 3]
#      // Length: #list
^      // Head: ^list (first element)
_      // Tail: _list (all but first)

// Range
..     // Inclusive: 1..10
..<    // Exclusive: 1..<10

// Special
!      // Move operator: x!
???    // Secure random: 0.0 <= ??? < 1.0
```

### Result/Option Types and Error Handling

Flap uses a **Result type** for operations that can fail, using NaN-boxing to encode errors efficiently:

```flap
// Division returns Result type (can fail with division by zero)
safe_divide = (a, b) -> {
    | b == 0 => error("dv0")  // Create error with code "dv0"
    ~> a / b
}

// Check for errors using .error property
result = safe_divide(10, 2)
result.error {
    "" => println("Success:", result)        // Empty string = no error
    "dv0" => println("Division by zero!")    // Error code
    ~> println("Other error:", result.error)
}

// Use or! operator for default values
safe = safe_divide(10, 0) or! -1  // Returns -1 on error

// Chain multiple or! for fallbacks
x = parse(input) or! default_value or! 0
```

**Standard error codes:**
- `"dv0"` - Division by zero
- `"idx"` - Index out of bounds
- `"key"` - Key not found
- `"mem"` - Out of memory
- `"arg"` - Invalid argument
- `"io"` - I/O error

**Built-in operations that return Results:**
```flap
x = 10 / 0      // Error: "dv0"
y = list[99]    // Error: "idx" (out of bounds)
z = map.missing // Error: "key" (key not found)
```

## Example Programs

### Game Loop

```flap
import sdl3 as sdl

sdl.SDL_Init(sdl.SDL_INIT_VIDEO)
window := sdl.SDL_CreateWindow("Game", 800, 600, 0)
renderer := sdl.SDL_CreateRenderer(window, 0)

running := 1
@ running == 1 {
    arena {
        // Per-frame memory freed automatically
        entities := alloc(entity_count * entity_size)

        @@ i in 0..<entity_count {
            update_entity(entities, i)
        }
    }

    sdl.SDL_RenderClear(renderer)
    render(renderer)
    sdl.SDL_RenderPresent(renderer)
}
```

### Fibonacci Sequence

```flap
fib = n -> n < 2 {
    => n
    ~> fib(n-1) + fib(n-2)
}

@ i in 0..<15 {
    println(fib(i))
}
```

### List Processing

```flap
map = (list, fn) -> {
    result := []
    @ i in 0..<#list {
        result[i] <- fn(list[i])
    }
    ret result
}

numbers := [1, 2, 3, 4, 5]
doubled := map(numbers, x -> x * 2)
```

## Performance Tips

1. **Use parallel loops** for CPU-bound work
2. **Arena allocators** for frame-based memory
3. **Tail calls** are optimized automatically
4. **Match expressions** compile to jump tables
5. **Atomic operations** for lock-free code

## Documentation

- **[LANGUAGESPEC.md](LANGUAGESPEC.md)** - Complete language specification
- **[GRAMMAR.md](GRAMMAR.md)** - Formal grammar (EBNF)
- **[LIBERTIES.md](LIBERTIES.md)** - Documentation accuracy tracking
- **[DEVELOPMENT.md](DEVELOPMENT.md)** - Compiler development guide

## Compilation

```bash
# Basic
flapc program.flap

# Custom output
flapc -o game program.flap

# Single file mode
flapc -s program.flap

# Cross-compilation (experimental)
flapc -arch arm64 program.flap
flapc -arch riscv64 program.flap
```

## Testing

```bash
# Fast tests
go test -short  # ~0.3s

# Full test suite
go test         # ~2s
```

## Current Status

### ✅ Production Ready (v3.0)
- Universal `map[uint64]float64` type system
- Direct x86-64 machine code generation
- Tail-call optimization
- Context-sensitive block disambiguation
- C FFI with automatic type handling
- Arena allocation
- Parallel loops (`@@`) with barrier synchronization
- Atomic operations
- SIMD support (AVX/AVX-512)
- Pattern matching (value and guard forms)

### 🚧 Experimental
- ARM64 backend (basic support)
- RISC-V backend (basic support)
- Advanced optimizations

## General info

* Version: 3.0.0
* License: BSD-3
