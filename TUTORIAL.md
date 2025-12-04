* **C67** is a programming language for games, demos, and tools.
* **c67** is a compiler written in Go that compiles `.c67` programs directly to machine code.
* **Status (December 2025)**: All tests passing! Production-ready for x86_64 Linux/Windows.

## What Makes C67 Special?

1. **Compiles to native machine code** - No VM, no interpreter, no LLVM
2. **Compact executables** - Hello World is ~29KB on Linux
3. **No garbage collection** - Arena allocators for predictable performance
4. **Zero-cost C FFI** - Call SDL3, OpenGL, Raylib directly
5. **Pure syscalls** - Linux builds don't need libc (unless you use C FFI)

As a quick demonstration, the following SDL3 program compiles and runs on both Linux and Windows:

```c
import sdl3 as sdl

// Window dimensions
width := 620
height := 387

println("Initializing SDL3...")

// Initialize SDL and exit if there is an error
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

---

# C67 Tutorial

**Version**: 1.5.0
**License**: BSD-3-Clause

## Installation

```bash
go install github.com/xyproto/c67@latest
```

Then add `~/go/bin` to your path, or use `~/go/bin/c67` to run it.

Or install it to ie. `/usr/bin` with:

    sudo install -Dm755 ~/go/bin/c67 /usr/bin/c67

Test it:

```bash
echo 'println("Hello, C67!")' > hello.c67
c67 hello.c67 -o hello
./hello
```

---

## Core Concepts

### 1. **Everything Is A Map**

C67 has ONE universal type: an ordered map from uint64 to float64

```c67
42                    // A number
"Hello"               // A string (map of character codes)
[1, 2, 3]             // A list (map with numeric keys)
{x: 10, y: 20}        // A struct-like map
```

("x" can be hashed to an uint64)

This simplicity means:
- No type annotations needed (usually)
- No generic syntax complexity
- C structs map directly to C67 maps

### 2. **Variables: Immutable by Default**

```c67
x = 42              // Immutable binding
counter := 0        // Mutable variable (:= declares mutable)
counter <- 10       // Update with <-
```

### 3. **Functions Are Values**

```c67
square = x -> x * x
add = (x, y) -> x + y

// Higher-order functions
apply = (f, x) -> f(x)
result = apply(square, 5)  // 25
```

### 4. **Loops with @**

```c67
@ i in 0..<10 { println(i) }           // Range loop (0..9)
@ item in list { process(item) }       // For-each
@ condition { work() }                 // While loop
@ { render(); delay(16) }              // Infinite loop
```

Some loops require `max inf` or `max 123` where 123 is the maximum amount of allowed loops.

Break out with `ret @`:

```c67
@ i in 0..<100 {
    | i > 50 => ret @
    println(i)
}
```

### 5. **Pattern Matching**

```c67
sign = x {
    | x > 0 => "positive"
    | x < 0 => "negative"
    ~> "zero"
}

// Works on maps too
get_x = point {
    | point.x != 0 => point.x
    ~> 0.0
}
```

### 6. **Error Handling with or!**

The `or!` operator checks for null/zero/error and executes alternative:

```c67
window := sdl.SDL_CreateWindow("Game", 800, 600, 0) or! {
    exitln("Failed to create window")
}

// Short form for default values
value := might_fail() or! 42
```

### 7. **Defer for Cleanup**

Resources are freed in LIFO order:

```c67
sdl.SDL_Init(sdl.SDL_INIT_VIDEO)
defer sdl.SDL_Quit()

window := sdl.SDL_CreateWindow("Game", 800, 600, 0)
defer sdl.SDL_DestroyWindow(window)

renderer := sdl.SDL_CreateRenderer(window, 0)
defer sdl.SDL_DestroyRenderer(renderer)

// Cleanup happens automatically: renderer, then window, then SDL
```

### 8. **Arena Allocators**

Bulk allocation and deallocation for performance:

```c67
arena {
    // All allocations in this block use arena memory
    particles = @ i in 0..<1000 { create_particle(i) }
    @ p in particles { update(p) }
}  // All arena memory freed here in one operation
```

**No GC pauses. No individual free() calls. Perfect for games.**

### 9. **Direct C FFI**

Import C libraries directly:

```c67
import sdl3 as sdl
import c

// Call C functions as if they were C67 functions
ptr := c.malloc(1024)
defer c.free(ptr)

// SDL3 functions work the same way
sdl.SDL_Init(sdl.SDL_INIT_VIDEO)
```

c67 parses C headers and generates bindings automatically.

### 10. **Compact, Fast Binaries**

- Hello World: ~29KB on Linux (vs 16KB for dynamically linked GCC, 813KB for static GCC)
- No runtime dependencies (unless you use C FFI)
- Direct syscalls for I/O
- Compiles in seconds

---

## Quick Examples

### Hello World

```c67
println("Hello, C67!")
```

Compile and run:
```bash
c67 hello.c67 -o hello
./hello
```

### Variables and Math

```c67
x = 42
y := x * 2           // Mutable
y <- y + 1           // Update

printf("x=%d, y=%d\n", x, y)
```

### Functions

```c67
factorial = n {
    | n <= 1 => 1
    ~> n * factorial(n - 1)
}

println(factorial(10))  // 3628800
```

### Lists and Loops

```c67
numbers = [1, 2, 3, 4, 5]

// Map over list
squared = numbers | x -> x * x
println(squared)

// Loop with index
@ i in 0..<#numbers {
    printf("numbers[%d] = %f\n", i, numbers[i])
}
```

---

## Building a Game: Pong

```c67
import sdl3 as sdl

// Constants
WIDTH := 800
HEIGHT := 600
PADDLE_SPEED := 5
BALL_SPEED := 4

// Init
sdl.SDL_Init(sdl.SDL_INIT_VIDEO) or! exitln("SDL failed")
defer sdl.SDL_Quit()

window := sdl.SDL_CreateWindow("Pong", WIDTH, HEIGHT, 0) or! exitln("Window failed")
defer sdl.SDL_DestroyWindow(window)

renderer := sdl.SDL_CreateRenderer(window, 0) or! exitln("Renderer failed")
defer sdl.SDL_DestroyRenderer(renderer)

// Game state
paddle1_y := 250.0
paddle2_y := 250.0
ball_x := 400.0
ball_y := 300.0
ball_vx := BALL_SPEED
ball_vy := BALL_SPEED
running := 1

// Main loop
@ {
    // Events
    @ {
        e := sdl.SDL_PollEvent(0)
        | e == 0 => break
        | sdl.SDL_EventType(e) == sdl.SDL_EVENT_QUIT => { running = 0; break }
    }

    // Input
    keys := sdl.SDL_GetKeyboardState(0)
    | keys[sdl.SDL_SCANCODE_W] => paddle1_y -= PADDLE_SPEED
    | keys[sdl.SDL_SCANCODE_S] => paddle1_y += PADDLE_SPEED
    | keys[sdl.SDL_SCANCODE_UP] => paddle2_y -= PADDLE_SPEED
    | keys[sdl.SDL_SCANCODE_DOWN] => paddle2_y += PADDLE_SPEED

    // Physics
    ball_x += ball_vx
    ball_y += ball_vy

    | ball_y <= 0 or ball_y >= HEIGHT => ball_vy = -ball_vy
    | ball_x <= 0 or ball_x >= WIDTH => {
        ball_x = 400.0
        ball_y = 300.0
    }

    // Render
    sdl.SDL_SetRenderDrawColor(renderer, 0, 0, 0, 255)
    sdl.SDL_RenderClear(renderer)
    sdl.SDL_SetRenderDrawColor(renderer, 255, 255, 255, 255)

    // Draw paddles
    sdl.SDL_RenderFillRect(renderer, 10, paddle1_y, 15, 100)
    sdl.SDL_RenderFillRect(renderer, WIDTH - 25, paddle2_y, 15, 100)

    // Draw ball
    sdl.SDL_RenderFillRect(renderer, ball_x, ball_y, 15, 15)

    sdl.SDL_RenderPresent(renderer)
    sdl.SDL_Delay(16)

    | running == 0 => break
}
```

**100 lines. Full game. No build scripts. No dependencies.**

---

## Language Overview

### Variables

```c67
x = 42              // Immutable
counter := 0        // Mutable
counter <- 10       // Update
```

### Functions

```c67
square = x -> x * x
add = (x, y) -> x + y
greet = name -> println(f"Hello, {name}!")
```

### Loops

```c67
@ i in 0..<10 { println(i) }           // Range
@ item in list { process(item) }       // For-each
@ condition { work() }                 // While
@ { render(); sdl.SDL_Delay(16) }      // Infinite
```

### Match

```c67
result = x {
    0 => "zero"
    1 => "one"
    _ => "other"
}

// Guards
classify = {
    | x > 0 => "positive"
    | x < 0 => "negative"
    _ => "zero"
}
```

### Operators

- `->` : Lambda
- `=>` : Match arm
- `<-` : Update
- `|` : Pipe
- `||` : Parallel map
- `or!` : Error handler
- `#` : Length
- `@` : Loop

### Import

```c67
import sdl3 as sdl
import raylib as rl
import opengl as gl
```

---

## Why C67 For Gamedev?

### Compact Executables

No heavyweight runtime means reasonable binary sizes. Ship indie games easily.

### Fast Compile Times

Direct code generation. No LLVM wait times. Edit → compile → run in <1 second.

### Predictable Performance

No GC pauses. No allocator surprises. Arena allocators give frame-perfect timing.

### C Library Access

SDL3, Raylib, GLFW, OpenGL, Vulkan, Dear ImGui—use any C library directly.

### Multi-Platform

Write once, compile for x86_64, ARM64, RISC-V. Linux, Windows, macOS support.

### Low-Level Control

Unsafe blocks for assembly. Direct memory access. SIMD intrinsics.

---

## Examples

### Sprite Rendering

```c67
import sdl3 as sdl

// Load texture
surface := sdl.SDL_LoadBMP("sprite.bmp") or! exitln("Load failed")
defer sdl.SDL_DestroySurface(surface)

texture := sdl.SDL_CreateTextureFromSurface(renderer, surface) or! exitln("Texture failed")
defer sdl.SDL_DestroyTexture(texture)

// Render sprite
@ {
    sdl.SDL_RenderClear(renderer)
    sdl.SDL_RenderTexture(renderer, texture, 0, 0)
    sdl.SDL_RenderPresent(renderer)
    sdl.SDL_Delay(16)
}
```

### Particle System

```c67
Particle = (x, y) -> {x: x, y: y, vx: ??, vy: ??}

arena {
    particles = @ i in 0..<1000 { Particle(400, 300) }

    @ {
        particles = particles | p -> {
            x: p.x + p.vx,
            y: p.y + p.vy,
            vx: p.vx,
            vy: p.vy + 0.1  // Gravity
        }

        @ p in particles { draw_pixel(p.x, p.y) }
    }
}
```

### Sound Effects

```c67
import sdl3_mixer as mix

mix.Mix_OpenAudio(44100, mix.MIX_DEFAULT_FORMAT, 2, 2048) or! exitln("Audio failed")
defer mix.Mix_CloseAudio()

jump_sound := mix.Mix_LoadWAV("jump.wav") or! exitln("Load failed")
defer mix.Mix_FreeChunk(jump_sound)

keys[sdl.SDL_SCANCODE_SPACE] { mix.Mix_PlayChannel(-1, jump_sound, 0) }
```

---

## Platform Support

| Platform         | Status | Notes                      |
| ---------------- | ------ | -------------------------- |
| x86_64 + Linux   | ✅      | Primary development target |
| x86_64 + Windows | ✅      | Via Wine or native         |
| ARM64 + Linux    | ✅      | Raspberry Pi, Apple M1/M2  |
| RISC-V + Linux   | ✅      | SiFive, StarFive boards    |
| x86_64 + macOS   | 🚧     | Mach-O support in progress |
| ARM64 + macOS    | 🚧     | Apple Silicon coming soon  |

---

## Documentation

- **[GRAMMAR.md](GRAMMAR.md)** - Complete language grammar
- **[LANGUAGESPEC.md](LANGUAGESPEC.md)** - Language specification
- **[TODO.md](TODO.md)** - Upcoming features

---

## Performance Tips

```c67
// Use arenas for per-frame allocations
arena {
    bullets = generate_bullets()
    enemies = spawn_wave()
    process_frame(bullets, enemies)
}  // Bulk free here

// Parallel processing
entities || entity -> {
    update_physics(entity)
    check_collisions(entity)
    update_animation(entity)
}

// SIMD for batch operations
positions_x = [x1, x2, x3, x4]
velocities_x = [vx1, vx2, vx3, vx4]
new_x = positions_x + velocities_x  // 4 adds in one instruction
```

---

## Philosophy

**C67 is designed for:**

1. **Games** - Fast, predictable, no GC pauses
2. **Demos** - Small binaries, direct hardware access
3. **Tools** - Quick compile times, C library access
4. **Learning** - Simple semantics, one universal type

**C67 is NOT for:**

- Web servers (use Go)
- Machine learning (use Python)
- Enterprise CRUD (use Java)

---

## Contributing

```bash
git clone https://github.com/xyproto/c67
cd c67
go build
go test -v
```

See compiler internals in the source. The codebase is ~20K lines of clean Go.

---

## License

BSD 3-Clause. See [LICENSE](LICENSE).

---

## Community

- **GitHub**: https://github.com/xyproto/c67
- **Issues**: https://github.com/xyproto/c67/issues
- **Discussions**: https://github.com/xyproto/c67/discussions

---

**Start making games with C67 today. 🎮**
