# Flap Package System Design

## Overview

Flap's package system enables importing external code, including wrappers for C libraries like Raylib5.

## Syntax

### Import Statement

```flap
use raylib5  // Import package from standard location
use "./mylib.flap"  // Import local file
use "github.com/user/repo"  // Import from Git (future)
```

### Package Structure

A Flap package is:
- A single `.flap` file, OR
- A directory with `main.flap` or `<dirname>.flap`

## Package Resolution

1. **Standard Library**: `~/.flap/packages/<name>`
2. **Local Files**: Relative paths starting with `.` or `/`
3. **Git Repositories**: Clone to `~/.flap/cache/<hash>` (future)

## Implementation Plan

### Phase 1: Local File Imports (Current)

```flap
// main.flap
use "./math_helpers.flap"

x := add(5, 3)
println(x)

// math_helpers.flap
add := (a, b) -> a + b
mul := (a, b) -> a * b
```

**Compiler Changes:**
- Add `use` keyword to lexer
- Parse `use` statements before other statements
- Concatenate imported files into single AST
- Handle duplicate definitions (last wins, or error?)

### Phase 2: Package Directory Structure

```
~/.flap/packages/
  raylib5/
    raylib5.flap       # Main package file
    colors.flap        # Optional submodules
    shapes.flap
```

**Usage:**
```flap
use raylib5

InitWindow(800, 600, "Hello")
@ BeginDrawing() {
  ClearBackground(RAYWHITE)
  DrawText("Hello!", 190, 200, 20, LIGHTGRAY)
  EndDrawing()
}
CloseWindow()
```

### Phase 3: Git Integration (Future)

```flap
use "github.com/user/raylib-flap@v1.0.0"
```

## Raylib5 Integration

### Example Wrapper Structure

```flap
// raylib5.flap - Main wrapper file

// Load shared library
raylib := dlopen("libraylib.so.5.0", 2)  // RTLD_NOW

// Core functions
InitWindow := (width, height, title) -> {
  w := i32(width)
  h := i32(height)
  t := cstr(title)
  call(raylib, "InitWindow", w, h, t)
}

CloseWindow := () -> call(raylib, "CloseWindow")

WindowShouldClose := () -> {
  result := call(raylib, "WindowShouldClose")
  result != 0
}

BeginDrawing := () -> call(raylib, "BeginDrawing")
EndDrawing := () -> call(raylib, "EndDrawing")

ClearBackground := (color) -> {
  c := u32(color)
  call(raylib, "ClearBackground", c)
}

DrawText := (text, x, y, fontSize, color) -> {
  t := cstr(text)
  px := i32(x)
  py := i32(y)
  fs := i32(fontSize)
  c := u32(color)
  call(raylib, "DrawText", t, px, py, fs, c)
}

// Color constants
LIGHTGRAY := 0xffc8c8c8
GRAY      := 0xff828282
DARKGRAY  := 0xff505050
YELLOW    := 0xff00f9fd
GOLD      := 0xff00d4ff
ORANGE    := 0xff00a0ff
PINK      := 0xffc86bff
RED       := 0xff0000e6
MAROON    := 0xff000078
GREEN     := 0xff00e600
LIME      := 0xff00ff00
DARKGREEN := 0xff009600
SKYBLUE   := 0xffffe666
BLUE      := 0xffff0000
DARKBLUE  := 0xff7d0000
PURPLE    := 0xffff00c8
VIOLET    := 0xffff0087
DARKPURPLE := 0xff700070
BEIGE     := 0xff83d3d3
BROWN     := 0xff2a547f
DARKBROWN  := 0xff1e3c4c
WHITE     := 0xffffffff
BLACK     := 0xff000000
MAGENTA   := 0xffff00ff
RAYWHITE  := 0xfff5f5f5
```

### Example Usage

```flap
use raylib5

InitWindow(800, 600, "Flap + Raylib")

@ !WindowShouldClose() {
  BeginDrawing()
  ClearBackground(RAYWHITE)
  DrawText("Hello from Flap!", 190, 200, 20, LIGHTGRAY)
  EndDrawing()
}

CloseWindow()
```

## Compiler Implementation

### Lexer Changes

Add `use` keyword:
```go
case "use":
    return Token{Type: TOKEN_USE, Value: value, Line: l.line}
```

### Parser Changes

```go
type UseStmt struct {
    Path string  // "./file.flap" or "raylib5"
}

func (p *Parser) parseUse() *UseStmt {
    p.nextToken() // skip 'use'
    if p.current.Type != TOKEN_STRING {
        p.error("expected string after 'use'")
    }
    path := p.current.Value
    return &UseStmt{Path: path}
}
```

### Compilation Strategy

**Option A: Inline Expansion** (Simple, Current)
- Read imported file
- Parse into AST
- Merge into main AST
- Compile as single unit

**Option B: Module System** (Future)
- Each package compiles to `.flapo` (object file)
- Linker combines objects
- Support for separate compilation

## Testing

```bash
# Create raylib wrapper
mkdir -p ~/.flap/packages/raylib5
cat > ~/.flap/packages/raylib5/raylib5.flap <<EOF
... (wrapper content)
EOF

# Test program
./flapc examples/raylib_hello.flap
./raylib_hello
```

## Future Enhancements

1. **Package Manager**: `flap install raylib5`
2. **Version Pinning**: `use "raylib5@1.0.0"`
3. **Namespaces**: `raylib5.InitWindow()` vs `InitWindow()`
4. **Private/Public**: Control symbol visibility
5. **Documentation**: Inline docs with `//!` comments
