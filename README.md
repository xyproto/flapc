# Flapc - The Flap Programming Language Compiler

[![Go CI](https://github.com/xyproto/flapc/actions/workflows/ci.yml/badge.svg)](https://github.com/xyproto/flapc/actions/workflows/ci.yml) [![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/flapc)](https://goreportcard.com/report/github.com/xyproto/flapc)

**Flap** is a compiled systems programming language that generates native machine code directly—no LLVM, no intermediate representations, just pure compilation speed and simplicity.

**Version:** 2.0.0  
**Platform:** Linux x86-64 (primary), ARM64 & RISC-V (in development)  
**Status:** Production-ready - 250+ tests passing (96.5%)

## What Makes Flap Different?

### Direct Machine Code Generation
- Compiles directly to x86-64/ARM64/RISC-V assembly
- No LLVM, no GCC dependency, no runtime
- Compilation time: **~1ms** for typical programs
- Generates static ELF binaries

### Unified Type System
Everything is `map[uint64]float64` internally:
```flap
42              // Number
"Hello"         // String (map of byte values)
[1, 2, 3]       // List
{x: 10, y: 20}  // Map/Object
[]              // Universal empty value
```

### One Way To Do Things
- **Single lambda arrow**: `=>` (not `=>` and `==>`)
- **Named operators**: `and`/`or`/`not` (not `&&`/`||`/`!`)
- **Explicit casts**: `x as uint64` (not `uint64(x)`)
- **Immutable by default**: Use `:=` only when you need mutation

### Built for Performance
- Automatic tail-call optimization
- AVX/AVX-512 SIMD support
- Parallel loops with barrier synchronization
- Arena allocators for scope-based memory management
- Atomic operations for lock-free algorithms

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
factorial = (n, acc) => n == 0 {
    -> acc
    ~> factorial(n-1, n*acc)
}

@ i in 0..<10 {
    println(factorial(i, 1))
}
```

## Language Features

### Variables and Assignment

```flap
x = 42        // Immutable (cannot reassign)
y := 100      // Mutable (can reassign)
y <- y + 1    // Update operator
y++           // Increment (mutable vars only)
```

### Functions (Lambdas)

```flap
// Single arrow => for all functions
square = x => x * x
add = (a, b) => a + b

// Block body (no arrows = regular function)
process = x => {
    temp := x * 2
    result := temp + 1
    ret result
}

// Match function (has arrows = pattern matching)
classify = x => x {
    0 -> "zero"
    n => n < 0 { -> "negative" ~> "positive" }
}
```

### Loops

```flap
// Range loops
@ i in 0..<10 {
    println(i)
}

// List iteration
items := [1, 2, 3, 4, 5]
@ item in items {
    println(item)
}

// Parallel loops (uses all CPU cores)
@@ i in 0..<1000 {
    process(i)
}

// Infinite loops
@ {
    update()
    render()
}

// Loop control
@ i in 0..<100 {
    i > 50 { ret @ }  // Break from loop
    println(i)
}
```

### Match Expressions

```flap
result = x {
    0 -> "zero"
    1 -> "one"
    ~> "many"      // Default case
}

// Conditional matching
sign = n => n {
    0 -> "zero"
    ~> n < 0 { -> "negative" ~> "positive" }
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

Direct calls to C libraries:

```flap
// Standard C library
ptr = c.malloc(1024.0)
c.memset(ptr, 0, 1024.0)
c.free(ptr)

// SDL3 for graphics
import sdl3 as sdl

sdl.SDL_Init(sdl.SDL_INIT_VIDEO)
window := sdl.SDL_CreateWindow("Game", 800, 600, 0)
renderer := sdl.SDL_CreateRenderer(window, 0)

@ {
    sdl.SDL_RenderClear(renderer)
    // Game logic here
    sdl.SDL_RenderPresent(renderer)
    sdl.SDL_Delay(16)
}
```

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
::     // Cons (prepend): 1 :: [2, 3] => [1, 2, 3]
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
fib = n => n < 2 {
    -> n
    ~> fib(n-1) + fib(n-2)
}

@ i in 0..<15 {
    println(fib(i))
}
```

### List Processing

```flap
map = (list, fn) => {
    result := []
    @ i in 0..<#list {
        result[i] <- fn(list[i])
    }
    ret result
}

numbers := [1, 2, 3, 4, 5]
doubled := map(numbers, x => x * 2)
```

## Performance Tips

1. **Use parallel loops** for CPU-bound work
2. **Arena allocators** for frame-based memory
3. **Tail calls** are optimized automatically
4. **Match expressions** compile to jump tables
5. **Atomic operations** for lock-free code

## Documentation

- **[LANGUAGESPEC.md](LANGUAGESPEC.md)** - Language semantics and design
- **[GRAMMAR.md](GRAMMAR.md)** - Formal grammar specification (EBNF)
- **[DEVELOPMENT.md](DEVELOPMENT.md)** - Compiler development guide
- **[RELEASE_NOTES_2.0.md](RELEASE_NOTES_2.0.md)** - Version 2.0 changes

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

### ✅ Production Ready
- Direct machine code generation
- Tail-call optimization
- C FFI with automatic type handling
- Arena allocation
- Parallel loops
- Atomic operations
- SIMD support (AVX/AVX-512)

### 🚧 In Development
- List update operations
- List cons operator
- ARM64/RISC-V backends
- Advanced optimizations

## Contributing

See [TODO.md](TODO.md) for specific tasks. Pull requests welcome!

## License

BSD 3-Clause License. See [LICENSE](LICENSE) for details.

## Links

- **Repository**: https://github.com/xyproto/flapc
- **Issues**: https://github.com/xyproto/flapc/issues
- **CI/CD**: https://github.com/xyproto/flapc/actions

---

**Flap: Direct compilation, unified types, minimal syntax.**
