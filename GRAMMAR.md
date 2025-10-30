## Grammar

The hand-written recursive-descent parser accepts the following grammar. Newlines separate statements but are otherwise insignificant. `//` starts a line comment. String escape sequences: `\n`, `\t`, `\r`, `\\`, `\"`.

```ebnf
program         = { newline } { statement { newline } } ;

statement       = use_statement
                | import_statement
                | arena_statement
                | defer_statement
                | loop_statement
                | jump_statement
                | spawn_statement
                | assignment
                | expression_statement ;

use_statement   = "use" string ;

import_statement = "import" string ;

arena_statement = "arena" block ;

defer_statement = "defer" expression ;

loop_statement  = "@" block
                | [ expression ] "@" identifier "in" expression [ "max" (number | "inf") ] block
                | "@@" identifier "in" expression [ "max" (number | "inf") ] block
                | "@" identifier "," identifier "in" expression [ "max" (number | "inf") ] block ;

jump_statement  = ("ret" | "err") [ "@" number ] [ expression ]
                | "@" number
                | "@++"
                | "@" number "++" ;

spawn_statement = "spawn" expression [ "|" spawn_params "|" block ] ;

spawn_params    = identifier { "," identifier }
                | map_destructure ;

map_destructure = "{" identifier [ ":" identifier ] { "," identifier [ ":" identifier ] } "}" ;

assignment      = identifier [ ":" type_annotation ] ("=" | ":=" | "<-") expression
                | identifier ("+=" | "-=" | "*=" | "/=" | "%=") expression ;

type_annotation = ("b" | "f") number ;

expression_statement = expression [ match_block ] ;

match_block     = "{" ( default_arm
                      | match_clause { match_clause } [ default_arm ] ) "}" ;

match_clause    = "->" match_target
                | expression [ "->" match_target ] ;

default_arm     = "~>" match_target ;

match_target    = jump_target | expression ;

jump_target     = ("ret" | "err") [ "@" number ] [ expression ]
                | "@" number ;

block           = "{" { statement { newline } } "}" ;

expression              = or_bang_expr ;

or_bang_expr            = concurrent_gather_expr [ "or!" string ] ;

concurrent_gather_expr  = parallel_expr { "|||" parallel_expr } ;

parallel_expr           = pipe_expr { "||" pipe_expr } ;

pipe_expr               = send_expr { "|" send_expr } ;

send_expr               = logical_or_expr [ "<==" logical_or_expr ] ;

logical_or_expr         = logical_and_expr { ("or" | "xor") logical_and_expr } ;

logical_and_expr        = comparison_expr { "and" comparison_expr } ;

comparison_expr         = range_expr [ (rel_op range_expr) | ("in" range_expr) ] ;

rel_op                  = "<" | "<=" | ">" | ">=" | "==" | "!=" ;

range_expr              = additive_expr [ "..<" additive_expr ] ;

additive_expr           = cons_expr { ("+" | "-") cons_expr } ;

cons_expr               = bitwise_expr { "::" bitwise_expr } ;

bitwise_expr            = multiplicative_expr { ("|b" | "&b" | "^b" | "<b" | ">b" | "<<b" | ">>b") multiplicative_expr } ;

power_expr              = unary_expr { "**" unary_expr } ;

multiplicative_expr     = power_expr { ("*" | "/" | "%" | "*+") power_expr } ;

unary_expr              = ("not" | "-" | "#" | "~b") unary_expr
                        | postfix_expr ;

postfix_expr            = primary_expr { "[" expression "]"
                                       | "(" [ argument_list ] ")"
                                       | "as" cast_type } ;

cast_type               = "i8" | "i16" | "i32" | "i64"
                        | "u8" | "u16" | "u32" | "u64"
                        | "f32" | "f64"
                        | "cstr" | "ptr"
                        | "number" | "string" | "list" ;

primary_expr            = number
                        | string
                        | port_literal
                        | identifier
                        | loop_state_var
                        | "(" expression ")"
                        | lambda_expr
                        | list_literal
                        | map_literal
                        | arena_expr
                        | unsafe_expr
                        | "^" primary_expr
                        | "&" primary_expr ;

port_literal            = ":" ( number | identifier ) [ "+" | "?" ] ;

arena_expr              = "arena" "{" { statement { newline } } expression "}" ;

unsafe_expr             = "unsafe" "{" { statement { newline } } [ expression ] "}" ;

loop_state_var          = "@first" | "@last" | "@counter" | "@i" ;

lambda_expr             = parameter_list "=>" lambda_body ;

lambda_body             = block | expression [ match_block ] ;

parameter_list          = identifier { "," identifier } ;

argument_list           = expression { "," expression } ;

list_literal            = "[" [ expression { "," expression } ] "]" ;

map_literal             = "{" [ map_entry { "," map_entry } ] "}" ;

map_entry               = expression ":" expression ;

identifier              = letter { letter | digit | "_" } ;

number                  = [ "-" ] digit { digit } [ "." digit { digit } ] ;

string                  = '"' { character } '"' ;

character               = printable_char | escape_sequence ;

escape_sequence         = "\\" ( "n" | "t" | "r" | "\\" | '"' ) ;
```

### Grammar Notes

* `@` without arguments creates an infinite loop: `@ { ... }`
* `@` with identifier introduces auto-labeled loops. The loop label is the current nesting depth (1, 2, 3, ...).
* `@ msg, from in :port` receives ENet messages: `msg` is the message string, `from` is sender address
* Optional numeric prefix before `@` specifies parallel execution: `4 @ i in list` uses 4 CPU cores
* `@@` is shorthand for all-cores parallel execution: `@@ i in list` uses all available CPU cores
* Port literals: `:5000` (numeric port), `:worker` (string port, hashed to number)
* Port operations: `:5000+` (next available port, returns actual port number), `:5000?` (check if available)
* `spawn expr` creates new process via fork() - fire and forget (child exits independently)
* `spawn expr | x | { ... }` spawns process and blocks waiting for result (assigned to `x`)
* `spawn expr | a, b, c | { ... }` tuple destructuring for multiple return values
* `spawn expr | {name, age} | { ... }` map destructuring for structured data
* `@++` continues the current loop (skip this iteration, jump to next).
* `@1++`, `@2++`, `@3++`, ... continues the loop at that nesting level (skip iteration, jump to next).
* `++` operator for pointer append: `ptr ++ value as type` writes value at current offset and auto-increments by sizeof(type)
* `+!` operator for add-with-carry in multi-precision arithmetic
* `unsafe` blocks can return register values as expressions (e.g., `rax`, `rbx`)
* `ret` returns from the current function with a val. `ret @` exits the current loop. `ret @1`, `ret @2`, `ret @3`, ... exits the loop at that nesting level and all inner loops. `err` returns an err.
* Lambda syntax: `x => expr` or `x, y => expr` (no parentheses around parameters).
* Type casting with `as`: Bidirectional conversion for FFI (e.g., `42 as i32` to C, `c_val as number` from C).
* Match blocks attach to the preceding expression. When omitted, implicit default is `0`.
* A single bare expression inside braces is shorthand for `-> expression`.
* A block with only `~>` leaves the condition's val untouched when true.
* Type annotations use `:b64` or `:f32` syntax for precision control.

## Keywords

```
and and! as cstruct err hot in not or or! ret xor &b |b ^b ~b <b >b >>b <<b
i8 i16 i32 i64 u8 u16 u32 u64 f32 f64 cstr cptr
number string list
packed aligned sizeof
```

**Note:** Type keywords (`i8`, `i16`, `i32`, `i64`, `u8`, `u16`, `u32`, `u64`, `f32`, `f64`, `cstr`, `cptr`, `number`, `string`, `list`) are **contextual keywords** - they are only reserved when used after `as` in type casting expressions. They can be used as variable names in other contexts:

```flap
// Valid: using type keywords as variable names
i32 = 100.0
ptr = call("malloc", 64 as u64)
string = "hello"

// Also valid: using them as type keywords in casts
x = 42 as i32
address = pointer_value as cptr
text = c_string as string
```

## Examples

### Factorial

```flap
factorial := n => n <= 1 {
    -> 1
    ~> n * factorial(n - 1)
}

println(factorial(5))  // 120
```

### FizzBuzz

```flap
@ i in 1..<101 {
    i % 15 == 0 {
        -> println("FizzBuzz")
    }
    i % 3 == 0 {
        -> println("Fizz")
    }
    i % 5 == 0 {
        -> println("Buzz")
    }
    println(i)
}
```

### List Processing

```flap
sum = list => {
    result := 0
    @ x in list {
        result := result + x
    }
    result
}

println(sum([1, 2, 3, 4, 5]))  // 15
```

### Filtering

```flap
filter = predicate, list => {
    result := []
    @ x in list {
        predicate(x) {
            -> result := result + x
        }
    }
    result
}

positive = x => x > 0
numbers = [-2, -1, 0, 1, 2]
println(filter(positive, numbers))  // [1, 2]
```

### FFI (Foreign Function Interface)

```flap
// Call C functions with type casting TO C
call("printf", "Hello from C!\n" as cstr)

// Get values FROM C
time_val = call("time", 0 as cptr)
timestamp = time_val as number
printf("Unix time: %f\n", timestamp)

// String conversion FROM C
home_ptr = call("getenv", "HOME" as cstr)
home_str = home_ptr as string
printf("HOME: %s\n", home_str)

// Memory allocation with safe read/write
ptr = call("malloc", 64 as u64)
write_f64(ptr, 0, 42.0)         // Write float64 at index 0
write_i32(ptr, 1, 100)          // Write int32 at index 1
val = read_f64(ptr, 0)          // Read back float64
int_val = read_i32(ptr, 1)      // Read back int32
call("free", ptr as cptr)

// Working with C structs (safe indexing)
// struct Point { float x; float y; }
point_ptr = call("malloc", 8 as u64)
write_f32(point_ptr, 0, 10.5)   // x field at index 0
write_f32(point_ptr, 1, 20.3)   // y field at index 1
x_val = read_f32(point_ptr, 0)
y_val = read_f32(point_ptr, 1)
call("free", point_ptr as cptr)
```

### Pointer Append (Buffer Building)

The `++` operator in pointer context provides sequential buffer writing with automatic offset tracking:

```flap
// Build a binary packet
build_packet := (id, x, y, health) => {
    packet := call("malloc", 256 as u64) as cptr

    // Write header
    packet ++ 0xCAFE as uint16      // Magic number at offset 0
    packet ++ 1 as uint8            // Version at offset 2
    packet ++ 0 as uint8            // Flags at offset 3

    // Write payload
    packet ++ id as uint32          // ID at offset 4
    packet ++ x as f32              // X position at offset 8
    packet ++ y as f32              // Y position at offset 12
    packet ++ health as uint16      // Health at offset 16

    // Total size: 18 bytes (compiler knows final offset)
    ret packet
}

// Build a file header
write_bitmap_header := (file, width, height) => {
    header := call("malloc", 54 as u64) as cptr

    // BMP file header (14 bytes)
    header ++ 0x4D42 as uint16      // "BM" signature
    header ++ 54 as uint32          // File size
    header ++ 0 as uint32           // Reserved
    header ++ 54 as uint32          // Pixel data offset

    // DIB header (40 bytes)
    header ++ 40 as uint32          // Header size
    header ++ width as uint32       // Image width
    header ++ height as uint32      // Image height
    header ++ 1 as uint16           // Planes
    header ++ 24 as uint16          // Bits per pixel
    header ++ 0 as uint32           // Compression
    header ++ 0 as uint32           // Image size
    header ++ 2835 as uint32        // X pixels per meter
    header ++ 2835 as uint32        // Y pixels per meter
    header ++ 0 as uint32           // Colors used
    header ++ 0 as uint32           // Important colors

    // Write to file
    call("fwrite", header as cptr, 54 as u64, 1 as u64, file as cptr)
    call("free", header as cptr)
}

// Build network message with mixed types
serialize_player := (player) => {
    buffer := call("malloc", 1024 as u64) as cptr

    // Serialize player data
    buffer ++ player.id as uint64
    buffer ++ player.x as f64
    buffer ++ player.y as f64
    buffer ++ player.z as f64
    buffer ++ player.health as uint32
    buffer ++ player.mana as uint32
    buffer ++ player.level as uint16
    buffer ++ player.flags as uint8

    ret buffer
}

// Read back with manual offsets (if needed)
deserialize_player := (buffer) => {
    player := {
        id: read_u64(buffer, 0),        // offset 0
        x: read_f64(buffer, 8),         // offset 8
        y: read_f64(buffer, 16),        // offset 16
        z: read_f64(buffer, 24),        // offset 24
        health: read_u32(buffer, 32),   // offset 32
        mana: read_u32(buffer, 36),     // offset 36
        level: read_u16(buffer, 40),    // offset 40
        flags: read_u8(buffer, 42)      // offset 42
    }
    ret player
}
```

**Benefits of Pointer Append:**
- No manual offset calculation
- Type-safe (compiler knows size of each type)
- Compiler error if you try to write beyond allocated size
- Clean syntax for building binary data
- Perfect for: network protocols, file formats, game saves, serialization


### C-Compatible Structs

The `cstruct` keyword defines C-compatible struct types for FFI. These structs have explicit memory layout, padding, and alignment compatible with C libraries.

**Syntax:**

```flap
cstruct StructName {
    field1: c_type,
    field2: c_type,
    ...
}

// With modifiers
cstruct StructName packed {  // No padding between fields
    ...
}

cstruct StructName aligned(16) {  // 16-byte alignment
    ...
}
```

**Examples:**

```flap
// Basic C struct
cstruct Vec3 {
    x: f32,
    y: f32,
    z: f32
}  // 12 bytes, standard C layout

// Packed struct (no padding)
cstruct PackedData packed {
    flag: u8,      // 1 byte
    val: u32,      // 4 bytes (no padding before)
    count: u16     // 2 bytes
}  // 7 bytes total (no padding)

// Aligned struct for SIMD
cstruct AlignedVec4 aligned(16) {
    x: f32,
    y: f32,
    z: f32,
    w: f32
}  // 16 bytes, 16-byte aligned

// Struct with pointers and strings
cstruct Entity {
    name: cstr,        // C string pointer
    position: *Vec3,   // Pointer to Vec3
    health: i32,
    flags: u32
}

// Nested structs
cstruct Transform {
    position: Vec3,
    rotation: Vec3,
    scale: Vec3
}
```

**Creating and Using:**

```flap
// Allocate struct memory
entity_ptr := call("malloc", sizeof(Entity) as u64)

// Write fields (field offsets calculated by compiler)
write_cstr(entity_ptr, Entity.name_offset, "Player" as cstr)
write_i32(entity_ptr, Entity.health_offset, 100)
write_u32(entity_ptr, Entity.flags_offset, 0x1)

// Read fields
health := read_i32(entity_ptr, Entity.health_offset)

// Pass to C functions
call("process_entity", entity_ptr as cptr)

// Cleanup
call("free", entity_ptr as cptr)
```

**Benefits:**
- **C-compatible**: Exact same layout as C structs
- **Explicit**: No hidden padding or alignment surprises
- **Safe offsets**: Compiler calculates field offsets
- **FFI-ready**: Works seamlessly with SDL, Raylib, etc.

### Hot Code Reload

Hot code reload allows recompiling and hot-swapping functions in a running program **without restart or re-launch**. The executable modifies its own machine code pages and atomically swaps to the new implementation - essential for rapid gamedev iteration.

**How It Works:**

Unlike traditional hot reload (restart process or dlopen), Flap's hot reload works by:

1. **Function indirection** - Hot functions called through function pointer table
2. **File watching** - Monitor source files with `inotify`/`kqueue`/`FSEvents`
3. **Incremental recompilation** - Recompile only changed functions
4. **Executable memory** - Allocate new pages with `mmap(PROT_EXEC)`
5. **Atomic swap** - Update function pointer (single atomic store)
6. **Self-modification** - Running executable patches itself

**Performance:** ~1-2 CPU cycles overhead per hot function call (indirect branch, branch predictor friendly)

**Marking Functions for Hot Reload:**

```flap
// Mark function as hot-swappable
hot update_player = (state, dt) => {
    state.x <- state.x + state.vx * dt
    state.y <- state.y + state.vy * dt

    // Edit this code while game runs
    state.x < 0 { state.x <- 0 }
    state.x > 800 { state.x <- 800 }

    ret state
}

hot draw_particles = particles => {
    @ p in particles {
        draw_circle(p.x, p.y, p.radius, p.color)
    }
}

// Normal function - no indirection overhead
init_game = () => {
    // Cannot be hot-swapped
    ret GameState { ... }
}
```

**Compilation & Usage:**

```bash
# Start game with file watching enabled
./flapc game.flap --watch -o game &
./game

# Edit hot functions in game.flap
# Compiler detects changes and hot-swaps automatically
# Game continues running with new code!

# Or manual trigger:
kill -USR1 $(pidof game)  # Reload signal
```

**Implementation Details:**

```c
// Compiler generates function table
struct FunctionTable {
    void (*update_player)(State*, f64);
    void (*draw_particles)(List*);
} hot_functions;

// All hot function calls go through table:
hot_functions.update_player(state, dt);  // One indirect jump

// On file change:
// 1. Recompile changed function
// 2. Generate new machine code
// 3. code = mmap(NULL, size, PROT_READ|PROT_WRITE|PROT_EXEC, ...)
// 4. memcpy(code, new_machine_code, size)
// 5. atomic_store(&hot_functions.update_player, code)
// 6. Old code pages freed after grace period
```

**File Watching:**

- **Linux**: `inotify` watches `*.flap` files
- **macOS**: `FSEvents` / `kqueue`
- **FreeBSD**: `kqueue`
- Detects: saves, renames, modifications
- Triggers: incremental recompilation

**What Can Be Hot-Swapped:**

✅ **Allowed:**
- Function bodies (logic changes)
- Algorithms and control flow
- Constants used in functions
- Bug fixes

❌ **Not Allowed:**
- Function signatures (arg count/types locked)
- Struct layouts (field offsets fixed)
- Global variable sizes
- Inlined functions (callers have old code baked in)

**Example Gamedev Workflow:**

```flap
// main.flap
import sdl3 as sdl

main ==> {
    window := sdl.SDL_CreateWindow("Game", 0, 0, 800, 600, 0)
    renderer := sdl.SDL_CreateRenderer(window, -1, 0)

    state := init_game()  // Not hot - runs once
    running := true

    @ running {
        // Hot functions - can edit while running
        state <- hot_update(state, 0.016)
        hot_render(renderer, state)

        sdl.SDL_RenderPresent(renderer)
    }
}

// These can be edited and reloaded live
hot hot_update = (state, dt) => {
    // Change gameplay logic here
    // See changes in ~50ms without restart
    state.player.x <- state.player.x + state.player.vx * dt
    ret state
}

hot hot_render = (renderer, state) => {
    // Tweak colors, positions, effects
    // Instant visual feedback
    sdl.SDL_SetRenderDrawColor(renderer, 255, 0, 0, 255)
    sdl.SDL_RenderClear(renderer)

    // Draw game...
}
```

**Use Cases:**

- **Rapid iteration**: Test gameplay changes in seconds
- **Visual tuning**: Adjust colors, positions, timing instantly
- **Live debugging**: Fix bugs while reproducing issue
- **Immediate feedback**: No wait for compile+restart cycle
- **State preservation**: Keep game state across code changes

**Comparison to Other Systems:**

| System | Method | Overhead | Restart Required |
|--------|--------|----------|------------------|
| **Flap** | Self-modifying code | ~0.1% | No |
| **SBCL (Lisp)** | Symbol table | ~5% | No |
| **Live++** | DLL patching | ~1% | No |
| **Rust hot reload** | dlopen | ~2% | No |
| **C++ recompile** | Full rebuild | 0% | Yes (slow) |

**Limitations:**

- Hot functions have slight overhead (indirect call)
- Cannot change function signatures
- Cannot change struct layouts mid-game
- Platform-specific (requires `mmap` + executable pages)
- Debug symbols may be stale

**Technical Notes:**

- Function table stored in `.data` section (mutable)
- Old code pages kept alive ~1 second for in-flight calls
- Atomic pointer updates ensure no torn reads
- Recompilation is incremental (only changed functions)
- Compatible with debuggers (gdb/lldb attach works)

## Testing Convention

Flap uses a simple, file-based testing convention for writing and running tests.

### Test File Naming

Test files follow the pattern `test_*.flap` and should be placed in a `tests/` directory or alongside the code they test.

### Test Structure

Each test is a separate Flap program that:
1. **Returns 0 on success** - The program should exit with code 0 when all tests pass
2. **Returns non-zero on failure** - Any non-zero exit code indicates test failure
3. **Prints descriptive messages** - Use `printf` or `println` to describe what failed

### Basic Test Example

```flap
// tests/test_math.flap
// Test basic arithmetic operations

// Test addition
result := 2 + 2
result != 4 {
    -> printf("FAIL: 2 + 2 expected 4, got %v\n", result) :: exit(1)
}

// Test multiplication
result := 3 * 4
result != 12 {
    -> printf("FAIL: 3 * 4 expected 12, got %v\n", result) :: exit(1)
}

println("PASS: All tests passed")
```

### Test Helper Functions

Create reusable assertion functions in your test files:

```flap
// Assert that two values are equal
assert_eq = actual, expected, message => actual != expected {
    -> printf("FAIL: %s\n  Expected: %v\n  Got: %v\n", message, expected, actual) :: exit(1)
}

// Assert that a condition is yes (non-zero)
assert = condition, message => condition == 0 {
    -> printf("FAIL: %s\n", message) :: exit(1)
}

// Usage
assert_eq(square(5), 25, "square(5) should equal 25")
assert(5 > 3, "5 should be greater than 3")
```

### Running Tests

Tests can be run individually or in batch:

```bash
# Run a single test
./flapc tests/test_math.flap && ./test_math
echo "Exit code: $?"

# Run all tests in a directory
for test in tests/test_*.flap; do
    name=$(basename "$test" .flap)
    ./flapc "$test" && ./"$name" && echo "✓ $name" || echo "✗ $name"
done
```

### Package Testing Convention

For packages (like flap_math, flap_core):
1. Place tests in a `tests/` subdirectory
2. Each test should import the package and test its public functions
3. Use descriptive test names that indicate what is being tested

```flap
// tests/test_sum.flap
import "github.com/xyproto/flap_core" as core

assert_eq = actual, expected => actual != expected {
    -> printf("Expected %v, got %v\n", expected, actual) :: exit(1)
}

// Test sum with positive numbers
assert_eq(core.sum([1, 2, 3, 4]), 10)

// Test sum with empty list
assert_eq(core.sum([]), 0)

// Test sum with negative numbers
assert_eq(core.sum([-1, -2, -3]), -6)

println("PASS: sum tests")
```

## Module System

Flap supports both explicit imports and automatic dependency resolution.

### Explicit Imports

```flap
// Import with namespace
import "github.com/xyproto/flap_math" as math
result := math.square(5)

// Import with wildcard (into same namespace)
import "github.com/xyproto/flap_core" as *
filtered := filter((x) -> x > 2, [1, 2, 3, 4])

// Version specification
import "github.com/xyproto/flap_math@v1.0.0" as math
import "github.com/xyproto/flap_math@latest" as math
import "github.com/xyproto/flap_math@HEAD" as math
```

### C Library Imports

**Status (v1.0.0):** C library FFI is fully functional for basic use cases.

Flap can call C library functions directly using a simple import syntax. The compiler automatically handles dynamic linking via PLT/GOT on Linux.

**Syntax:**

```flap
// Import C library (auto-detected by lack of "/" in name)
import sdl3 as sdl
import raylib as rl
import c as libc  // Standard C library

// Import custom .so file (NEW in v1.2.0)
import "/tmp/libmylib.so" as mylib
import "/usr/local/lib/libcustom.so.1" as custom

// Call C functions with namespace prefix
sdl.SDL_Init(0)
window := sdl.SDL_CreateWindow("Game", 100, 100, 800, 600, 0)

// Call custom library functions
result := mylib.my_function(1, 2, 3, 4, 5, 6, 7, 8)  // >6 arguments supported!

// Standard library functions
pid := libc.getpid()
time := libc.time(0)
```

**How It Works:**

1. **Auto-detection**:
   - Strings ending with ".so" or containing ".so." → Custom .so file import
   - Identifiers without "/" → Standard C libraries (sdl3, raylib, c)
   - Strings with "/" → Flap packages (Git)
2. **Symbol extraction**: For custom .so files, symbols are automatically extracted using `nm -D`
3. **Dynamic linking**: C libraries are added to ELF `DT_NEEDED` (e.g., `libSDL3.so`, `libraylib.so`, `libmylib.so`)
4. **PLT calls**: Functions are called through the Procedure Linkage Table
5. **ABI compatibility**: Arguments are marshaled to System V AMD64 calling convention
6. **Stack arguments**: Functions with >6 arguments use stack-based argument passing

**Custom .so File Imports (v1.2.0+):**

```flap
// Import custom library
import "/tmp/libmanyargs.so" as mylib

// Functions are automatically discovered from the .so file
result := mylib.sum7(1, 2, 3, 4, 5, 6, 7)          // 7 arguments
result := mylib.sum10(1, 2, 3, 4, 5, 6, 7, 8, 9, 10) // 10 arguments

// Run with: LD_LIBRARY_PATH=/tmp ./program
```

**Current Limitations (v1.2.0):**

- Arguments are converted to integers (uint32/int64) or float64
- Return values are converted to Flap's `float64`
- Limited support for strings (C strings work via `as cstr` cast)
- No support for structs yet

**Example: SDL3 Game Initialization**

```flap
import sdl3 as sdl

// Initialize SDL
result := sdl.SDL_Init(0x00000020)  // SDL_INIT_VIDEO = 0x20
result > 0 {
    println("SDL_Init failed")
    exit(1)
}

// Create window
window := sdl.SDL_CreateWindow(
    "My Game",  // title (will be supported in v1.4.1)
    100,        // x
    100,        // y
    800,        // width
    600,        // height
    0           // flags
)

// Game loop would go here...

sdl.SDL_Quit()
```

**Example: Standard C Library**

```flap
import c as libc

pid := libc.getpid()
println("Process ID:")
println(pid)

time := libc.time(0)
println("Unix timestamp:")
println(time)
```

**Library Naming:**

The compiler automatically converts library names:
- `sdl3` → `libSDL3.so.0` (SDL3 with version)
- `raylib` → `libraylib.so.5` (RayLib 5)
- `c` → (uses already-linked `libc.so.6`)
- `m` → (math library, link automatically if needed)

**Roadmap (v1.4.1+):**

- String arguments (C char pointers)
- Struct support
- Pointer handling
- Float return values
- pkg-config integration for automatic library discovery
- >6 argument support

### Private Functions

Functions and variables starting with `_` are private and not exported:

```flap
// Public function (exported)
square = (x) -> x * x

// Private helper (not exported)
_validate = (x) -> x > 0 { -> ~> 1 ~> 0 }

// Only square() is available when imported
```

### Cache Management

```bash
# View cached dependencies
ls ~/.cache/flapc/

# Update all dependencies
flapc --update-deps myprogram.flap

# Clear cache
rm -rf ~/.cache/flapc/
```

## Memory Management

**Status (v1.0.0):** Syntax and documentation complete. Full runtime implementation coming in v1.5.0.

Flap introduces syntax for arena allocators and defer statements. The compiler recognizes the keywords and parses the constructs, with full runtime implementation planned for v1.5.0.

### Arena Allocators

Arena allocators provide fast bump-pointer allocation with bulk deallocation. All memory allocated within an `arena` block is automatically freed when the block exits.

**Key Benefits:**
- **Fast allocation**: O(1) bump pointer, no per-allocation overhead
- **Automatic cleanup**: No manual free() calls needed
- **Cache friendly**: Contiguous memory allocation
- **Perfect for**: Temporary data structures, per-frame game allocations, parser ASTs

**Syntax:**

```flap
arena {
    buffer := alloc(1024)        // Allocate 1KB
    particles := alloc(8 * 100)  // Allocate 100 particles (8 bytes each)
    // Use buffer and particles...
}  // All memory automatically freed here
```

**Implementation:**
- Initial size: 4096 bytes (4KB)
- Growth strategy: Double on overflow (4KB → 8KB → 16KB → 32KB...)
- Uses `malloc()` for initial allocation, `realloc()` for growth
- Thread-local arena stack for nested arenas

**Nested Arenas:**

```flap
arena {
    outer_data := alloc(100)

    arena {
        inner_data := alloc(200)
        // Both inner_data and outer_data available
    }  // inner_data freed

    // outer_data still available
}  // outer_data freed
```

**Game Development Example:**

```flap
// Per-frame arena for temporary allocations
game_loop = () -> {
    @ frame in range(1000000) {
        arena {
            // Allocate temporary structures
            visible_entities := alloc(entity_size * max_visible)
            render_commands := alloc(command_size * max_commands)

            // Render frame using temporary data...
            render_frame(visible_entities, render_commands)
        }  // All temporary memory freed - zero fragmentation
    }
}
```

### The `alloc()` Builtin

Allocates memory from the current arena.

**Signature:** `alloc(size: number) -> pointer`

**Parameters:**
- `size`: Number of bytes to allocate

**Returns:** Pointer to allocated memory (as float64)

**Example:**

```flap
arena {
    // Allocate structure
    player := alloc(64)  // 64 bytes

    // Write to memory
    write_f64(player, 0, 100.0)      // health at offset 0
    write_f64(player, 8, 50.0)       // mana at offset 8
    write_f64(player, 16, 250.5)     // x position
    write_f64(player, 24, 128.3)     // y position

    // Read from memory
    health := read_f64(player, 0)
    printf("Player health: %.0f\n", health)
}
```

**Interaction with malloc/free:**

You can still use `malloc()` and `free()` via FFI for manual memory management:

```flap
// Load libc
libc := dlopen("libc.so.6", 2)  // RTLD_NOW
malloc_fn := dlsym(libc, "malloc")
free_fn := dlsym(libc, "free")

// Manual allocation
ptr := call(malloc_fn, 1024 as u64) as cptr
defer call(free_fn, ptr)  // Cleanup with defer

// Use ptr...
```

### Defer Statements

The `defer` keyword schedules an expression to execute at the end of the current scope, regardless of how the scope exits (normal return, early return, or implicit fall-through).

**Execution Order:** LIFO (Last-In-First-Out) - like a stack

**Syntax:**

```flap
defer expression
```

**Example:**

```flap
open_and_process = (filename) -> {
    file := fopen(filename, "r")
    defer fclose(file)  // Always executed

    // If this fails, fclose still called
    data := read_file(file)

    // Process data...
    ret process(data)  // fclose called before return
}
```

**Multiple Defers:**

```flap
process_resources = () -> {
    file1 := fopen("data.txt", "r")
    defer fclose(file1)

    file2 := fopen("config.txt", "r")
    defer fclose(file2)

    connection := connect("localhost", 8080)
    defer disconnect(connection)

    // On exit, calls in order:
    // 1. disconnect(connection)
    // 2. fclose(file2)
    // 3. fclose(file1)
}
```

**With Arena:**

```flap
load_level = (level_file) -> {
    arena {
        // Temporary allocations for loading
        temp_buffer := alloc(1024 * 1024)  // 1MB temp buffer

        file := fopen(level_file, "rb")
        defer fclose(file)  // Called before arena cleanup

        // Load and process...
        level_data := parse_level(file, temp_buffer)

        ret level_data
    }  // Arena freed, then fclose called (LIFO)
}
```

**Common Patterns:**

```flap
// Resource cleanup
handle_request = (request) -> {
    lock := acquire_lock()
    defer release_lock(lock)

    // Critical section...
}

// Profiling
profile_function = () -> {
    start := get_time()
    defer {
        duration := get_time() - start
        printf("Function took %.2fms\n", duration)
    }

    // Function body...
}

// Error handling with manual memory
allocate_and_process = () -> {
    libc := dlopen("libc.so.6", 2)
    malloc_fn := dlsym(libc, "malloc")
    free_fn := dlsym(libc, "free")

    ptr1 := call(malloc_fn, 1024 as u64) as cptr
    defer call(free_fn, ptr1)

    ptr2 := call(malloc_fn, 2048 as u64) as cptr
    defer call(free_fn, ptr2)

    // Both freed automatically, even if early return
}
```

### Best Practices

**Use arenas for:**
- Temporary allocations (per-frame game data)
- Parser/compiler intermediate structures
- Request-scoped data in servers
- Any data with clear lifetime boundaries

**Use defer for:**
- File handles (fopen/fclose)
- Network connections
- Locks (acquire/release)
- Manual memory cleanup (malloc/free)
- Resource handles from C libraries

**Avoid:**
- Long-lived data in arenas (arena blocks should be scoped)
- Mixing arena alloc() with manual free() (undefined behavior)
- Returning pointers from arena blocks (dangling pointer)

**Example: Game Entity System**

```flap
update_physics = (entities, dt) -> {
    arena {
        // Temporary spatial partitioning
        grid := alloc(grid_size * 8)

        // Build acceleration structure
        @ entity in entities {
            cell := get_grid_cell(entity.x, entity.y)
            add_to_cell(grid, cell, entity)
        }

        // Process collisions using grid
        collisions := check_collisions(grid)
        apply_collision_responses(collisions, dt)

    }  // Grid freed automatically
}
```

## Hot Code Reload

The `hot` keyword marks functions as hot-reloadable for live development:

```flap
hot update_player := (player, dt) => {
    player.x <- player.x + player.vx * dt
    player.y <- player.y + player.vy * dt
}

hot render_scene := (scene) => {
    @ entity in scene.entities {
        draw_sprite(entity.sprite, entity.x, entity.y)
    }
}

normal_init := () => {
    // Non-hot functions compile to direct calls (faster)
    load_assets()
}
```

**Current Implementation:**
- Parser recognizes `hot` keyword
- Functions marked as hot are tracked in compiler
- Foundation for runtime code swapping

**Future Capabilities:**
- Function pointer table for indirect calls
- File watching (inotify/kqueue/FSEvents)
- Incremental recompilation of changed functions
- Atomic pointer swaps for seamless updates
- ~1-2 cycle overhead per hot function call

**Use Cases:**
- Game development (iterate on gameplay without restarting)
- Visual tuning (adjust rendering in real-time)
- Shader development
- Live debugging
- Rapid prototyping

**Limitations:**
- Cannot change function signatures at runtime
- Cannot modify struct layouts
- Hot functions have slight indirection overhead

Hot reload enables sub-50ms iteration cycles compared to full recompilation.

## Atomic Operations

Flap provides builtin atomic operations for lock-free concurrent programming:

### Functions

**atomic_add(ptr, value)**
- Atomically adds `value` to the memory location pointed to by `ptr`
- Returns the old value before the addition
- Uses x86-64 LOCK XADD instruction

**atomic_cas(ptr, old, new)**
- Compare-and-swap: if `*ptr == old`, set `*ptr = new`
- Returns 1.0 if successful, 0.0 if failed
- Uses x86-64 LOCK CMPXCHG instruction

**atomic_load(ptr)**
- Atomically loads value from memory with acquire semantics
- On x86-64, regular loads are already atomic

**atomic_store(ptr, value)**
- Atomically stores value to memory with release semantics
- Returns the stored value
- On x86-64, regular stores are already atomic

### Example

```flap
// Atomic counter example
counter := alloc(8)  // Allocate 8 bytes
atomic_store(counter, 0)

// Multiple threads can safely increment
old := atomic_add(counter, 1)
printf("Old: %.0f, New: %.0f\n", old, atomic_load(counter))

// Compare and swap for lock-free algorithms
success := atomic_cas(counter, 1, 10)
success == 1 {
    println("Successfully swapped 1 with 10")
}
```

### Memory Ordering

On x86-64, the atomic operations provide:
- **atomic_load**: Acquire semantics
- **atomic_store**: Release semantics
- **atomic_add**: Full sequential consistency
- **atomic_cas**: Full sequential consistency

### Current Limitations

- Only supports 64-bit integer atomics (treated as float64)
- ARM64 and RISC-V implementations pending
- No atomic operations on Flap maps/lists yet

Atomic operations enable efficient parallel algorithms without locks.
