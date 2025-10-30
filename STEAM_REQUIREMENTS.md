# Flapc Requirements for Steam Game Development

## Essential Features for Commercial Games

### Core Language (MUST HAVE)
- [x] Variables and basic arithmetic
- [x] Functions (lambdas)
- [x] Control flow (if/match)
- [x] Loops (sequential only)
- [x] Arrays and maps
- [x] String handling
- [ ] **Proper register allocator** (currently ad-hoc)
- [ ] **Struct types** (for game entities, components)
- [ ] **Stack management** (currently fragile)

### C FFI (MUST HAVE)
- [x] Basic C function calls (SDL3, etc.)
- [ ] **Struct interop** (SDL_Event, SDL_Rect, etc.)
- [ ] **Callback functions** (event handlers)
- [ ] **String conversions** (C ↔ Flap)
- [ ] **Type safety** (prevent segfaults)

### Memory Management (MUST HAVE)
- [x] malloc/free via FFI
- [ ] **Safe pointer operations**
- [ ] **Bounds checking** (debug mode)
- [ ] **Memory leak detection** (valgrind clean)

### Performance (MUST HAVE)
- [ ] **Optimized code generation** (reduce bloat)
- [ ] **Register allocation** (fewer mov instructions)
- [ ] **Common subexpression elimination**
- [ ] **Dead code elimination**
- [ ] **60 FPS game loop** (must be achievable)

### Debugging (MUST HAVE)
- [ ] **DWARF debug info** (gdb/lldb support)
- [ ] **Line number mapping**
- [ ] **Readable error messages**
- [ ] **Stack traces on crash**

### Robustness (MUST HAVE)
- [ ] **No compiler crashes** (graceful errors)
- [ ] **Comprehensive test suite**
- [ ] **Valgrind clean** (no memory leaks in compiler or generated code)
- [ ] **Fuzzing** (random program generation)

## Features to REMOVE (Too Complex / Not Essential)

### Parallel Loops (REMOVE)
- `@@ i in 0..<N { }` - Too complex, rarely needed for games
- Barrier synchronization code is fragile
- Most games are single-threaded or use simple fork/spawn
- **Decision: Remove @@ syntax, keep simple @ loops**

### Parallel Reducers (REMOVE)
- `@@ i in ... { } | a,b | { }` - Very complex, not implemented
- Tree reduction requires sophisticated algorithm
- Games rarely need data parallelism
- **Decision: Remove from grammar entirely**

### Parallel Map (REMOVE or SIMPLIFY)
- `list || lambda` - Currently sequential anyway
- Name is misleading (suggests parallelism)
- **Decision: Either remove or rename to `map` function**

### Loop Expressions (DEFER)
- `@ i in ... { expr }` - Adds complexity
- Can be achieved with manual accumulator
- **Decision: Defer to v2.0, not essential for MVP**

### Hot Reload (DEFER)
- Very complex to implement correctly
- Can use external tools (live++)
- **Decision: Defer to v2.0**

### Arena Allocators (DEFER)
- Runtime complexity
- Can use malloc/free for MVP
- **Decision: Defer to v2.0**

### Networking (DEFER)
- Send/receive primitives incomplete
- Games can use SDL_net or enet via FFI
- **Decision: Defer to v2.0**

## What Steam Games Actually Need

### Typical 2D Game (e.g., Platformer)
```flap
import sdl3 as sdl

// Structs for game state
Player :: struct {
    x: f64,
    y: f64,
    vx: f64,
    vy: f64,
    grounded: i32
}

// Game loop
@ running {
    // Handle input
    @ event in sdl.SDL_PollEvent() {
        event.type == sdl.SDL_QUIT {
            running <- false
        }
    }

    // Update physics
    player.vy <- player.vy + gravity * dt
    player.y <- player.y + player.vy * dt

    // Render
    sdl.SDL_RenderClear(renderer)
    sdl.SDL_RenderCopy(renderer, player_texture, player.x, player.y)
    sdl.SDL_RenderPresent(renderer)
}
```

**Needs:**
- Structs (Player)
- Mutable variables (player.x, player.y)
- C FFI (SDL3)
- Loops (@ event, @ running)
- Fast compilation (< 1s for iteration)

### Typical 3D Game (e.g., FPS)
```flap
import sdl3 as sdl
import opengl as gl

// 3D math
Vec3 :: struct { x: f32, y: f32, z: f32 }
Mat4 :: struct { m: [16]f32 }

// Entity component system
Entity :: struct {
    position: Vec3,
    rotation: Vec3,
    mesh_id: i32
}

entities := []
@ i in 0..<1000 {
    entities <- entities + new_entity(i)
}

// Render loop
@ running {
    @ entity in entities {
        render_mesh(entity.mesh_id, entity.position, entity.rotation)
    }
}
```

**Needs:**
- Arrays of structs
- Array iteration
- OpenGL FFI
- Good performance (60 FPS with 1000 entities)

## Proposed Final Feature Set for v1.0

### Core Language
- Variables (mutable and immutable)
- Functions (named and lambda)
- Structs (named types with fields)
- Arrays and maps
- Strings
- Control flow (if/else, match)
- Loops (@ only, no @@)
- Break and continue

### Type System
- float64 (default)
- i8, i16, i32, i64
- u8, u16, u32, u64
- f32, f64
- cstr (C string pointer)
- cptr (C void pointer)
- Struct types
- Array types

### C FFI
- Import C libraries
- Call C functions (variadic support)
- C struct interop
- String conversions
- Callback functions

### Standard Functions
- printf, println, print
- malloc, free, realloc
- Array operations (len, append, slice)
- Map operations (insert, lookup, delete)
- String operations (concat, substr, split)
- Math functions (sin, cos, sqrt, etc.)

### Memory Operations
- read_i8, read_i16, read_i32, read_i64
- write_i8, write_i16, write_i32, write_i64
- read_f32, read_f64
- write_f32, write_f64
- Bounds checking (debug mode)

### Compiler Features
- Register allocator
- Code optimization
- DWARF debug info
- Fast compilation (< 1s for typical game)
- Clear error messages

### Removed Features
- ❌ Parallel loops (@@)
- ❌ Parallel reducers
- ❌ Loop expressions
- ❌ Hot reload
- ❌ Arena allocators
- ❌ Defer statements
- ❌ Networking primitives
- ❌ Parallel map operator

## Success Criteria

A Flapc game can be published on Steam if:

1. **Compiles to native ELF** (Linux) or PE (Windows eventually)
2. **Links with SDL3/OpenGL/Vulkan** via FFI
3. **Runs at 60 FPS** with typical game workload
4. **No memory leaks** (valgrind clean)
5. **No segfaults** with proper error handling
6. **Fast iteration** (< 1s compile time)
7. **Debuggable** (gdb/lldb with line numbers)
8. **Clear errors** when compilation fails

## Example: Complete Pong Game

This is the complexity level we need to support:

```flap
import sdl3 as sdl

WIDTH := 800
HEIGHT := 600

Paddle :: struct { x: f64, y: f64, w: f64, h: f64, vy: f64 }
Ball :: struct { x: f64, y: f64, vx: f64, vy: f64, size: f64 }

main := => {
    sdl.SDL_Init(sdl.SDL_INIT_VIDEO)
    window := sdl.SDL_CreateWindow("Pong", WIDTH, HEIGHT, 0)
    renderer := sdl.SDL_CreateRenderer(window, 0)

    player := Paddle { x: 50, y: HEIGHT/2, w: 20, h: 100, vy: 0 }
    ai := Paddle { x: WIDTH-70, y: HEIGHT/2, w: 20, h: 100, vy: 0 }
    ball := Ball { x: WIDTH/2, y: HEIGHT/2, vx: 300, vy: 200, size: 10 }

    running := true
    @ running {
        // Input
        keys := sdl.SDL_GetKeyboardState()
        keys[sdl.SDL_SCANCODE_W] {
            player.vy <- -400
        }
        keys[sdl.SDL_SCANCODE_S] {
            player.vy <- 400
        }

        // AI
        ball.y > ai.y {
            ai.vy <- 300
        } else {
            ai.vy <- -300
        }

        // Physics
        dt := 0.016
        player.y <- player.y + player.vy * dt
        ai.y <- ai.y + ai.vy * dt
        ball.x <- ball.x + ball.vx * dt
        ball.y <- ball.y + ball.vy * dt

        // Collision
        ball.y < 0 or ball.y > HEIGHT {
            ball.vy <- -ball.vy
        }

        // Render
        sdl.SDL_SetRenderDrawColor(renderer, 0, 0, 0, 255)
        sdl.SDL_RenderClear(renderer)
        sdl.SDL_SetRenderDrawColor(renderer, 255, 255, 255, 255)
        draw_rect(renderer, player)
        draw_rect(renderer, ai)
        draw_rect(renderer, ball.x, ball.y, ball.size, ball.size)
        sdl.SDL_RenderPresent(renderer)
    }

    sdl.SDL_Quit()
}
```

**This should:**
- Compile in < 1 second
- Run at 60 FPS
- Work with gdb
- Have no memory leaks
- Give clear errors if syntax wrong

## Timeline

**Phase 1: Core Cleanup (1-2 weeks)**
- Remove parallel loop code
- Implement register allocator
- Fix stack management
- Add struct types

**Phase 2: FFI & Interop (1-2 weeks)**
- Struct interop with C
- Callback functions
- String conversions
- Type safety

**Phase 3: Polish & Testing (1 week)**
- Comprehensive test suite
- Valgrind clean
- DWARF debug info
- Error messages

**Phase 4: Documentation (3-4 days)**
- Update GRAMMAR.md
- Update README.md
- Create LEARNINGS.md
- Write example games

**Total: 4-6 weeks to production-ready v1.0**
