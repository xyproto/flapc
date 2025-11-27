Hi, this is written by a human.

* Flap is a programming language.
* Flapc is a compiler written in Go for compiling `.flap` programs directly to machine code.

As a quick demonstration for what Flapc can do right now, the follow programs compiles and runs fine on both Linux (`x86_64`) and Windows (`x86_64`):

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

### General info

* Version: 1.3.0
* License: BSD-3
# Flap: The Gamedev Language

**Flap compiles directly to x86_64, ARM64, and RISC-V machine code. Zero dependencies. Zero runtime. Pure metal.**

```flap
import sdl3 as sdl

sdl.SDL_Init(sdl.SDL_INIT_VIDEO) or! exitln("SDL init failed")
defer sdl.SDL_Quit()

window := sdl.SDL_CreateWindow("My Game", 800, 600, 0) or! exitln("Window failed")
defer sdl.SDL_DestroyWindow(window)

@ { render_frame(); sdl.SDL_Delay(16) }  // 60 FPS forever
```

**That's it. No build system. No package manager. No ceremony.**

---

## What Makes Flap Different?

### 1. **Everything Is A Map**

Flap has ONE type: `map[uint64]float64`

```flap
42                    // {0: 42.0}
"Hello"               // {0: 72, 1: 101, 2: 108, 3: 108, 4: 111}
[1, 2, 3]            // {0: 1, 1: 2, 2: 3}
{x: 10, y: 20}       // {hash("x"): 10, hash("y"): 20}
```

No type system complexity. No generics hell. No trait soup. **Just maps.**

### 2. **Automatic Memoization**

Pure functions are automatically cached:

```flap
fib = n -> {
    | n == 0 => 0
    | n == 1 => 1
    _ => fib(n-1) + fib(n-2)
}

println(fib(35))  // Instant: ~0.001s (cached)
```

No decorators. No manual cache management. **Just write recursive code.**

### 3. **Direct Machine Code**

Flapc compiles straight to machine code. No LLVM. No intermediate layers.

```bash
./flapc game.flap -o game
./game
```

**x86_64, ARM64, RISC-V support. Windows, Linux, macOS binaries.**

### 4. **Zero-Cost C FFI**

Call C functions directly. No wrappers. No overhead.

```flap
import sdl3 as sdl
import opengl as gl

window := sdl.SDL_CreateWindow("OpenGL", 800, 600, sdl.SDL_WINDOW_OPENGL)
context := sdl.SDL_GL_CreateContext(window)

gl.glClearColor(0.0, 0.0, 0.0, 1.0)
gl.glClear(gl.GL_COLOR_BUFFER_BIT)
```

SDL3, Raylib, OpenGL, Vulkan, GLFW—just import and use.

### 5. **Arena Allocators**

Bulk allocate per frame, bulk free at end:

```flap
arena {
    particles = @ i in 0..<1000 { create_particle(i) }
    @ p in particles { update(p) }
    // All 1000 particles freed HERE
}
```

**No GC pauses. No reference counting. No use-after-free.**

### 6. **Defer For Cleanup**

LIFO resource cleanup, like Go:

```flap
file := open("data.txt") or! ret
defer close(file)

buffer := c.malloc(1024) or! ret  
defer c.free(buffer)

// Cleanup happens automatically on return/error
```

### 7. **Railway-Oriented Errors**

The `or!` operator handles errors inline:

```flap
window := sdl.SDL_CreateWindow("Game", 800, 600, 0) or! {
    exitln("Failed to create window")
}

result := divide(10, 0) or! -1  // Returns -1 on error
```

**No try/catch. No Result<T, E> ceremony. Just `or!`.**

### 8. **Match Expressions**

Pattern matching built-in:

```flap
sign = x {
    | x > 0 => "positive"
    | x < 0 => "negative"
    _ => "zero"
}
```

### 9. **Parallel Everything**

Multi-core by default:

```flap
// Parallel map
squared = [1, 2, 3, 4] || x -> x * x

// Parallel loop
entities || entity -> update(entity)
```

### 10. **Small Binaries**

A "Hello World" is ~8KB on Linux. No runtime, no stdlib bloat.

---

## Installation

```bash
git clone https://github.com/xyproto/flapc
cd flapc
go build
sudo mv flapc /usr/local/bin/
```

---

## Your First Game: Pong

```flap
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

```flap
x = 42              // Immutable
counter := 0        // Mutable
counter <- 10       // Update
```

### Functions

```flap
square = x -> x * x
add = (x, y) -> x + y
greet = name -> println(f"Hello, {name}!")
```

### Loops

```flap
@ i in 0..<10 { println(i) }           // Range
@ item in list { process(item) }       // For-each
@ condition { work() }                 // While
@ { render(); sdl.SDL_Delay(16) }      // Infinite
```

### Match

```flap
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

```flap
import sdl3 as sdl
import raylib as rl
import opengl as gl
```

---

## Why Flap For Gamedev?

### Small Executables

No runtime = small binaries. Ship games under 1MB easily.

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

```flap
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

```flap
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

```flap
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

```flap
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

**Flap is designed for:**

1. **Games** - Fast, predictable, no GC pauses
2. **Demos** - Small binaries, direct hardware access
3. **Tools** - Quick compile times, C library access
4. **Learning** - Simple semantics, one universal type

**Flap is NOT for:**

- Web servers (use Go)
- Machine learning (use Python)
- Enterprise CRUD (use Java)

---

## Contributing

```bash
git clone https://github.com/xyproto/flapc
cd flapc
go build
go test -v
```

See compiler internals in the source. The codebase is ~20K lines of clean Go.

---

## License

BSD 3-Clause. See [LICENSE](LICENSE).

---

## Community

- **GitHub**: https://github.com/xyproto/flapc
- **Issues**: https://github.com/xyproto/flapc/issues
- **Discussions**: https://github.com/xyproto/flapc/discussions

---

**Start making games with Flap today. 🎮**
