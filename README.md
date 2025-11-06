# Flapc

[![Go CI](https://github.com/xyproto/flapc/actions/workflows/ci.yml/badge.svg)](https://github.com/xyproto/flapc/actions/workflows/ci.yml) [![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/flapc)](https://goreportcard.com/report/github.com/xyproto/flapc)

**Flap** (**fl**oats and m**ap**s) is a compiled systems programming language designed for game development, real-time applications, and high-performance computing.

**Version:** 1.3.0
**Platform:** Linux x86-64 (Arch/Debian tested), with ARM64 and RISC-V support in development
**Status:** 344+ test programs passing, production-ready for single-threaded applications

## Why Flap?

- **Direct machine code generation** - No LLVM, no GCC, no runtime. Lexer → Parser → x86-64 → ELF in ~1ms
- **Simple yet powerful** - Unified type system (`map[uint64]float64` for everything), tail-call optimization, automatic memory management
- **Built for games** - C FFI for SDL3/OpenGL/Vulkan, arena allocators for per-frame memory, parallel loops for physics
- **Zero dependencies** - Generates static binaries with no runtime required

## Quick Start

### Installation

```bash
# From source (requires Go 1.21+)
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

### Your First Program

```flap
// Fibonacci with tail-call optimization
fib := n => n < 2 {
    -> n
    ~> fib(n-1) + fib(n-2)
}

println(fib(10))  // 55
```

## Core Features

### Unified Type System

Everything is `map[uint64]float64` internally. Numbers, strings, lists, objects—same representation, SIMD-optimized.

```flap
x := 42              // {0: 42.0}
name := "Flap"       // {0: 70.0, 1: 108.0, 2: 97.0, 3: 112.0}
list := [1, 2, 3]    // {0: 1.0, 1: 2.0, 2: 3.0}
obj := {x: 10}       // {"x" hashed to uint64: 10.0}
```

### Tail-Call Optimization

Automatic TCO for recursive functions:

```flap
// Infinite loop - no stack overflow
loop := n => {
    println(n)
    -> loop(n + 1)  // tail call
}
loop(0)
```

### Immutable by Default

```flap
x = 42        // Immutable (can't be reassigned)
y := 100      // Mutable (can be reassigned)
y++           // OK (increment operator)
y <- y + 1    // OK (assignment operator)
x = x + 1     // Compile error
```

### Loops

```flap
// Range loops
@ i in 0..<10 {
    println(i)
}

// Infinite loops
@ {
    update()
    render()
}

// Parallel loops (with barrier synchronization)
@@ i in 0..<1000 {
    process(i)  // Runs on all CPU cores
}
```

### C FFI

Direct calls to C libraries with automatic type handling:

```flap
// Call C standard library functions directly
ptr = c.malloc(1024.0)
c.free(ptr)

// Or import libraries with namespaces
import sdl3 as sdl

init_result = sdl.SDL_Init(sdl.SDL_INIT_VIDEO)
window = sdl.SDL_CreateWindow("Game", 800, 600, 0)
renderer = sdl.SDL_CreateRenderer(window, 0)

@ {
    sdl.SDL_RenderClear(renderer)
    // ... game logic ...
    sdl.SDL_RenderPresent(renderer)
    sdl.SDL_Delay(16)
}
```

### CStruct - C-Compatible Structures

```flap
cstruct Vec3 {
    x as float32
    y as float32
    z as float32
}

// Use the generated constants
println(sizeof(Vec3))        // 12
println(offsetof(Vec3, x))   // 0
println(offsetof(Vec3, y))   // 4
println(offsetof(Vec3, z))   // 8

// Allocate and access
ptr = c.malloc(sizeof(Vec3))
write_f32(ptr, offsetof(Vec3, x), 1.0)
write_f32(ptr, offsetof(Vec3, y), 2.0)
write_f32(ptr, offsetof(Vec3, z), 3.0)
c.free(ptr)
```

### Arena Allocation

Scope-based memory management - perfect for per-frame game allocations:

```flap
arena {
    // All allocations freed at block exit
    buffer := alloc(1024)
    entities := alloc(entity_count * entity_size)
    // ... work with memory ...
}  // Everything freed here
```

### Unsafe Blocks

Direct register access for performance-critical code:

```flap
// Unified syntax - works on x86-64, ARM64, RISC-V
result := unsafe {
    a <- 42        // a = rax/x0/a0 depending on CPU
    b <- 10
    a <- a + b
}  // result = 52

// Or specify per-architecture
value := unsafe {
    rax <- 100     // x86-64
} {
    x0 <- 100      // ARM64
} {
    a0 <- 100      // RISC-V
}
```

### Atomic Operations

Lock-free primitives for parallel programming:

```flap
counter_ptr = c.malloc(8.0)
atomic_store(counter_ptr, 0)

@@ i in 0..<1000 {
    atomic_add(counter_ptr, 1)
}

result = atomic_load(counter_ptr)
println(result)  // 1000
c.free(counter_ptr)
```

### Match Expressions

Pattern matching with default case:

```flap
result = x {
    0 -> "zero"
    1 -> "one"
    2 -> "two"
    ~> "many"
}
```

### Lambda Functions

```flap
add := (a, b) => a + b
map := (list, fn) => @ i in 0..<len(list) { fn(list[i]) }

numbers := [1, 2, 3, 4, 5]
doubled := map(numbers, n => n * 2)
```

## Type Casting

Full type names (never abbreviated):

```flap
x := 42.7

// Integer casts
i8_val := x as int8       // 42
i32_val := x as int32     // 42
u64_val := x as uint64    // 42

// Float casts
f32_val := x as float32   // 42.700000
f64_val := x as float64   // 42.700000

// Pointer casts
ptr := x as ptr
cstr := name as cstr      // String to C string
```

## Compilation

```bash
# Basic compilation
flapc program.flap

# Specify output
flapc -o game program.flap

# Single file (don't load other .flap files from directory)
flapc -s program.flap

# Target different architectures
flapc -arch arm64 -os darwin program.flap
flapc -arch riscv64 program.flap

# Fast tests during development
go test -short  # ~0.3s
go test         # ~6s full suite
```

## Examples

### Game Loop with SDL3

```flap
import sdl3 as sdl

sdl.SDL_Init(sdl.SDL_INIT_VIDEO)
window := sdl.SDL_CreateWindow("Game", 800, 600, 0)
renderer := sdl.SDL_CreateRenderer(window, 0)

running := 1
@ running == 1 {
    arena {
        // Per-frame allocations freed automatically
        entities := alloc(1000 * entity_size)

        @@ i in 0..<entity_count {
            update_entity(entities, i)
        }
    }

    sdl.SDL_RenderClear(renderer)
    render_all(renderer)
    sdl.SDL_RenderPresent(renderer)
}

sdl.SDL_Quit()
```

### Parallel Ray Tracing

```flap
@@ y in 0..<height {
    @ x in 0..<width {
        ray := create_ray(x, y)
        color := trace(ray, scene, 0)
        pixels[y * width + x] <- color
    }
}
```

### Physics Simulation

```flap
@@ i in 0..<particle_count {
    old_pos := particles[i]
    velocity := velocities[i]

    // Update position
    new_pos := old_pos + velocity * dt
    particles[i] <- new_pos

    // Simple collision
    new_pos.y < 0 {
        particles[i].y <- 0
        velocities[i].y <- -velocities[i].y * 0.8
    }
}
```

## Documentation

- **LANGUAGE.md** - Complete language specification, grammar, and examples
- **TODO.md** - Development roadmap and specific implementation tasks
- **LEARNINGS.md** - Design decisions and lessons learned
- **testprograms/** - 344+ example programs demonstrating all features

## Current Limitations

- **Register Allocator**: Ad-hoc register usage (planned for v1.4)
- **Debugging**: Limited DWARF info (planned for v1.4)
- **Parallel Features**: Loop expressions with reducers not yet implemented
- **Platform Support**: Primary focus on x86-64 Linux (ARM64/RISC-V experimental)

## Contributing

Contributions welcome! See TODO.md for specific tasks needing implementation.

## License

BSD 3-Clause License. See LICENSE file for details.

## Links

- **Repository**: https://github.com/xyproto/flapc
- **Issues**: https://github.com/xyproto/flapc/issues
- **CI/CD**: https://github.com/xyproto/flapc/actions

---

**Note**: Flap is production-ready for single-threaded game development and systems programming. Parallel features and advanced optimizations are under active development.
