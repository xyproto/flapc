# Flap Programming Language

**Version:** 3.0  
**Compiler:** Flapc 1.5.0  
**Targets:** x86_64 Linux, x86_64 Windows (Wine), ARM64, RISCV64

Flap is a minimalist systems programming language that compiles directly to machine code. It features a revolutionary universal type system based on `map[uint64]float64`, automatic memory management with arena allocators, and seamless C FFI for graphics, games, and systems programming.

### ✨ Automatic Memoization for Pure Functions

Flap automatically detects and memoizes pure single-argument functions, making recursive algorithms blazingly fast with zero effort:

```flap
// Fibonacci - automatically memoized!
fib = n -> {
    | n == 0 => 0
    | n == 1 => 1
    _ => fib(n - 1) + fib(n - 2)
}

println(fib(35))  // Instant: 9227465 (< 0.001s)
```

The compiler detects that `fib` is pure (no side effects) and automatically caches results. No annotations, no manual cache management—just write clean recursive code and let Flap optimize it for you.

---

## Quick Start: Game Programming with SDL3

Flap excels at game development with SDL3. Here's a complete tutorial from "Hello SDL" to a working game.

### Prerequisites

Install SDL3 development libraries:
```bash
# Arch Linux
sudo pacman -S sdl3

# Ubuntu/Debian (when available)
sudo apt install libsdl3-dev

# Build from source (if not in repos)
git clone https://github.com/libsdl-org/SDL
cd SDL && mkdir build && cd build
cmake .. && make -j$(nproc) && sudo make install
```

### Tutorial 1: Window and Image Display

Create `hello_sdl.flap`:

```flap
import sdl3 as sdl

// Initialize SDL3 with video subsystem
sdl.SDL_Init(sdl.SDL_INIT_VIDEO) or! {
    exitf("SDL_Init failed: %s\n", sdl.SDL_GetError())
}
defer sdl.SDL_Quit()

// Create window
width := 800
height := 600
window := sdl.SDL_CreateWindow("My First Game", width, height, sdl.SDL_WINDOW_RESIZABLE) or! {
    exitf("Failed to create window: %s\n", sdl.SDL_GetError())
}
defer sdl.SDL_DestroyWindow(window)

// Create renderer
renderer := sdl.SDL_CreateRenderer(window, 0) or! {
    exitf("Failed to create renderer: %s\n", sdl.SDL_GetError())
}
defer sdl.SDL_DestroyRenderer(renderer)

// Load and display an image
file := sdl.SDL_IOFromFile("assets/logo.bmp", "rb") or! {
    exitf("Error loading image: %s\n", sdl.SDL_GetError())
}

surface := sdl.SDL_LoadBMP_IO(file, 1) or! {
    exitf("Error creating surface: %s\n", sdl.SDL_GetError())
}
defer sdl.SDL_DestroySurface(surface)

texture := sdl.SDL_CreateTextureFromSurface(renderer, surface) or! {
    exitf("Error creating texture: %s\n", sdl.SDL_GetError())
}
defer sdl.SDL_DestroyTexture(texture)

// Render loop (60 FPS for 3 seconds)
@ frame in 0..<180 {
    sdl.SDL_RenderClear(renderer)
    sdl.SDL_RenderTexture(renderer, texture, 0, 0)
    sdl.SDL_RenderPresent(renderer)
    sdl.SDL_Delay(16)  // ~60 FPS
}

println("Done!")
```

Compile and run:
```bash
./flapc hello_sdl.flap -o hello_sdl
./hello_sdl
```

**Key Features Demonstrated:**
- `import sdl3 as sdl` - C library import with namespace
- `or!` operator - Railway-oriented error handling (checks for null/0)
- `defer` - Automatic LIFO cleanup (like Go's defer)
- `@` loop - Numeric range iteration
- Seamless C FFI with SDL3 functions

### Tutorial 2: Interactive Game Loop with Input

Create `pong_simple.flap`:

```flap
import sdl3 as sdl

// Constants
SCREEN_WIDTH := 800
SCREEN_HEIGHT := 600
PADDLE_WIDTH := 15
PADDLE_HEIGHT := 100
BALL_SIZE := 15
PADDLE_SPEED := 5
BALL_SPEED_X := 4
BALL_SPEED_Y := 4

// Initialize SDL
sdl.SDL_Init(sdl.SDL_INIT_VIDEO) or! {
    exitf("SDL_Init failed: %s\n", sdl.SDL_GetError())
}
defer sdl.SDL_Quit()

window := sdl.SDL_CreateWindow("Pong Game", SCREEN_WIDTH, SCREEN_HEIGHT, 0) or! {
    exitf("Failed to create window: %s\n", sdl.SDL_GetError())
}
defer sdl.SDL_DestroyWindow(window)

renderer := sdl.SDL_CreateRenderer(window, 0) or! {
    exitf("Failed to create renderer: %s\n", sdl.SDL_GetError())
}
defer sdl.SDL_DestroyRenderer(renderer)

// Game state (using Flap's universal map type)
paddle1_y := 250.0
paddle2_y := 250.0
ball_x := 400.0
ball_y := 300.0
ball_vel_x := BALL_SPEED_X as number
ball_vel_y := BALL_SPEED_Y as number
running := 1

// Main game loop
@ {
    // Event handling
    event := sdl.SDL_PollEvent(0)
    @ {
        e := sdl.SDL_PollEvent(0)
        | e == 0 => break  // No more events
        | sdl.SDL_EventType(e) == sdl.SDL_EVENT_QUIT => { running = 0; break }
        | ~> {}
    }
    
    // Keyboard input
    keys := sdl.SDL_GetKeyboardState(0)
    | keys[sdl.SDL_SCANCODE_W] != 0 => paddle1_y -= PADDLE_SPEED
    | keys[sdl.SDL_SCANCODE_S] != 0 => paddle1_y += PADDLE_SPEED
    | keys[sdl.SDL_SCANCODE_UP] != 0 => paddle2_y -= PADDLE_SPEED
    | keys[sdl.SDL_SCANCODE_DOWN] != 0 => paddle2_y += PADDLE_SPEED
    
    // Update ball position
    ball_x += ball_vel_x
    ball_y += ball_vel_y
    
    // Ball collision with top/bottom
    | ball_y <= 0 or ball_y >= SCREEN_HEIGHT - BALL_SIZE => ball_vel_y = -ball_vel_y
    
    // Ball collision with paddles
    | ball_x <= PADDLE_WIDTH and ball_y >= paddle1_y and ball_y <= paddle1_y + PADDLE_HEIGHT => {
        ball_vel_x = -ball_vel_x
        ball_x = PADDLE_WIDTH
    }
    | ball_x >= SCREEN_WIDTH - PADDLE_WIDTH - BALL_SIZE and ball_y >= paddle2_y and ball_y <= paddle2_y + PADDLE_HEIGHT => {
        ball_vel_x = -ball_vel_x
        ball_x = SCREEN_WIDTH - PADDLE_WIDTH - BALL_SIZE
    }
    
    // Ball out of bounds (reset)
    | ball_x < 0 or ball_x > SCREEN_WIDTH => {
        ball_x = 400.0
        ball_y = 300.0
        ball_vel_x = BALL_SPEED_X * (???  > 0.5 ? 1 : -1)  // Random direction
        ball_vel_y = BALL_SPEED_Y * (??? > 0.5 ? 1 : -1)
    }
    
    // Render
    sdl.SDL_SetRenderDrawColor(renderer, 0, 0, 0, 255)  // Black background
    sdl.SDL_RenderClear(renderer)
    
    sdl.SDL_SetRenderDrawColor(renderer, 255, 255, 255, 255)  // White
    
    // Draw paddles and ball (using SDL_Rect and SDL_RenderFillRect)
    rect_paddle1 := sdl.SDL_Rect(10, paddle1_y as number, PADDLE_WIDTH, PADDLE_HEIGHT)
    sdl.SDL_RenderFillRect(renderer, rect_paddle1)
    
    rect_paddle2 := sdl.SDL_Rect(SCREEN_WIDTH - PADDLE_WIDTH - 10, paddle2_y as number, PADDLE_WIDTH, PADDLE_HEIGHT)
    sdl.SDL_RenderFillRect(renderer, rect_paddle2)
    
    rect_ball := sdl.SDL_Rect(ball_x as number, ball_y as number, BALL_SIZE, BALL_SIZE)
    sdl.SDL_RenderFillRect(renderer, rect_ball)
    
    sdl.SDL_RenderPresent(renderer)
    sdl.SDL_Delay(16)  // ~60 FPS
    
    | running == 0 => break
}

println("Thanks for playing!")
```

**New Features:**
- `@ { }` - Infinite loop (break with `break`)
- Match expressions with guards (`| condition => action`)
- `???` - Secure random number operator (cryptographically secure)
- C struct construction (SDL_Rect)
- State management with mutable variables

### Tutorial 3: Sprites and Texture Management

For a complete sprite-based game with animations, see `sdl3example.flap` in the Flapc repository. It demonstrates:

- Texture atlases and sprite sheets
- Animation frame management
- Collision detection
- Score and UI rendering
- Sound effects (SDL_Mixer)
- Game state machines

### Flap Features for Game Development

1. **Zero-Cost C FFI**: Direct calls to SDL3, OpenGL, Vulkan
2. **Arena Allocators**: Bulk memory deallocation per frame
3. **Defer**: Automatic resource cleanup (textures, sounds, etc.)
4. **SIMD**: Built-in vector operations for physics
5. **Parallel Loops**: Multi-threaded game logic with `||`
6. **Hot Reloading**: Recompile and reload game code without restart (Unix)

### Performance Tips

```flap
// Use arenas for per-frame allocations
arena {
    particles := []
    @ i in 0..<1000 {
        particles += create_particle(i)
    }
    // All particles freed at once here
}

// Parallel physics updates
entities || entity -> update_physics(entity)

// SIMD vector math for 4 entities at once
positions_x := [x1, x2, x3, x4]
positions_y := [y1, y2, y3, y4]
velocities_x := [vx1, vx2, vx3, vx4]
velocities_y := [vy1, vy2, vy3, vy4]

// Update all 4 positions in parallel with SIMD
new_x := positions_x + velocities_x
new_y := positions_y + velocities_y
```

---

## Installation

### From Source

```bash
git clone https://github.com/xyproto/flapc
cd flapc
go build
sudo mv flapc /usr/local/bin/
```

### Verify Installation

```bash
flapc --version
```

---

## Language Overview

### The Universal Type System

Everything in Flap is `map[uint64]float64`:

```flap
42                    // {0: 42.0}
"Hello"               // {0: 72.0, 1: 101.0, 2: 108.0, ...}
[1, 2, 3]            // {0: 1.0, 1: 2.0, 2: 3.0}
{x: 10, y: 20}       // {hash("x"): 10.0, hash("y"): 20.0}
```

This eliminates type system complexity while maintaining full expressiveness.

### Syntax Highlights

```flap
// Variables (immutable by default)
x = 42
name = "Alice"
list = [1, 2, 3, 4, 5]

// Mutable variables
counter := 0
counter <- counter + 1  // Update with <-

// Functions (lambdas)
square = x -> x * x
add = (x, y) -> x + y
greet = name -> println(f"Hello, {name}!")

// Match expressions (like pattern matching)
sign = x {
    | x > 0 => "positive"
    | x < 0 => "negative"
    ~> "zero"
}

// Loops
@ i in 0..<10 {
    println(i)
}

// Parallel loops (multi-threaded)
results = [1, 2, 3, 4, 5] || x -> x * x

// Error handling with Result types
result = safe_divide(10, 0)
| result.error => println("Division by zero!")
| ~> println(f"Result: {result.value}")
```

### Operators

- `->` : Lambda definition
- `=>` : Match arm
- `<-` : Update mutable variable/list/map
- `|` : Pipe operator (function composition)
- `||` : Parallel map
- `#` : Length operator
- `^` : Head operator (first element)
- `_` : Tail operator (all except first)
- `??` : Random number [0, 1)
- `???` : Cryptographically secure random
- `or!` : Railway-oriented error handling

---

## Documentation

- [GRAMMAR.md](GRAMMAR.md) - Complete formal grammar (EBNF)
- [LANGUAGESPEC.md](LANGUAGESPEC.md) - Language specification
- [DEVELOPMENT.md](DEVELOPMENT.md) - Compiler internals
- [INSTALL.md](INSTALL.md) - Installation guide

---

## Examples

### QuickSort

```flap
quicksort = xs {
    | #xs <= 1 => xs
    | ~> {
        pivot = xs[0]
        rest = _xs
        lesser = rest | x -> x < pivot || quicksort
        greater = rest | x -> x >= pivot || quicksort
        lesser + [pivot] + greater
    }
}

sorted = quicksort([3, 1, 4, 1, 5, 9, 2, 6])
println(sorted)  // [1, 1, 2, 3, 4, 5, 6, 9]
```

### Web Server (with C FFI)

```flap
import libc as c

server = c.socket(c.AF_INET, c.SOCK_STREAM, 0)
c.bind(server, c.sockaddr_in(c.AF_INET, 8080, c.INADDR_ANY), 16)
c.listen(server, 10)

println("Server listening on port 8080")

@ {
    client = c.accept(server, 0, 0)
    response = "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nHello from Flap!"
    c.write(client, response, #response)
    c.close(client)
}
```

### Parallel Data Processing

```flap
// Process 1 million items in parallel across all CPU cores
data = @ i in 0..<1000000 { expensive_computation(i) }

// Parallel map with ||
squared = data || x -> x * x

// Parallel reduce
sum = data || + 0  // Sum all elements in parallel
```

---

## Cross-Platform Support

| Target | Status | Notes |
|--------|--------|-------|
| x86_64 Linux | ✅ Full | Primary development target |
| x86_64 Windows | ✅ Full | Via Wine or native Windows |
| x86_64 macOS | ⚠️ Experimental | Mach-O support |
| ARM64 Linux | ⚠️ Experimental | Raspberry Pi, Apple Silicon |
| RISCV64 Linux | ⚠️ Experimental | RISC-V boards |

Compile for different targets:
```bash
./flapc game.flap -target x86_64-linux -o game_linux
./flapc game.flap -target x86_64-windows -o game.exe
```

---

## Philosophy

Flap is designed for:

1. **Simplicity**: One universal type, minimal syntax
2. **Performance**: Direct machine code generation, no runtime
3. **Safety**: Arena allocators, no manual memory management
4. **Interop**: Seamless C FFI for existing libraries
5. **Expressiveness**: Functional + imperative + parallel

---

## Contributing

Contributions welcome! See [DEVELOPMENT.md](DEVELOPMENT.md) for compiler internals.

```bash
git clone https://github.com/xyproto/flapc
cd flapc
go build
go test
```

---

## License

Flap compiler (flapc) is licensed under the BSD 3-Clause License.
See [LICENSE](LICENSE) for details.

---

## Community

- GitHub: https://github.com/xyproto/flapc
- Issues: https://github.com/xyproto/flapc/issues
- Discussions: https://github.com/xyproto/flapc/discussions

---

**Happy coding with Flap! 🚀**
