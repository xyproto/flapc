Hi, this is written by a human.

* Flap is a programming language.
* Flapc is a compiler written in Go for compiling `.flap` programs directly to machine code.

As a quick demonstration for what Flapc can do right now, the follow programs compiles and runs fine on both Linux (`x86_64`) and Windows (`x86_64`):

```c
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

### General info

* Version: 1.3.0
* License: BSD-3
