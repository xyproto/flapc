# The Flap Programming Language

**Tagline**: Float. Map. Fly.

## Language Philosophy

Flap is a functional programming language designed for high-performance numerical computing with explicit SIMD parallelism. Built on a `map[uint64]float64` foundation, it provides elegant abstractions for modern CPU architectures while maintaining simplicity and clarity.

**Core Principle:** Everything is either `float64` or `map[uint64]float64`:
- Numbers are `float64`
- Strings are `map[uint64]float64` (character indices → char codes)
- Lists are `map[uint64]float64` (element indices → values)
- Maps are `map[uint64]float64` (keys → values)
- Functions are `float64` (pointers reinterpreted as floats)

This unified type system enables consistent SIMD optimization across all data structures.

## Currently Implemented Subset

The following features are working in the current implementation:

```flap
// Variables (immutable and mutable)
x = 10
y := 20
y := y + 5

// Variable precision numbers
x:b8 = 42         // 8-bit precision (faster, less accurate)
y:b32 = 3.14159   // 32-bit precision (standard float)
z:b64 = 3.14159265358979  // 64-bit precision (standard double)
w:b128 = PI       // 128-bit precision (quad precision)

// Precision blocks affect all variables defined within
@precision(32) {
    quick := 42.0      // Stored as 32-bit float
    fast := PI * 2.0   // PI computed at 32-bit precision
}

@precision(128) {
    accurate := 42.0   // Stored as 128-bit float
    precise := PI * 2.0  // PI computed at 128-bit precision
}

// Arithmetic
result = 10 + 3 * 2 - 1 / 2

// Comparisons (returns 1.0 for true, 0.0 for false)
x < y, x <= y, x > y, x >= y, x == y, x != y

// Match expressions (if/else replacement)
x < y {
    -> println("less")
    ~> println("not less")
}

// Default case is optional (defaults to 0)
x < y {
    -> println("yes")
}

// Strings (stored as map[uint64]float64)
s := "Hello"         // Creates {0: 72.0, 1: 101.0, 2: 108.0, 3: 108.0, 4: 111.0}
char := s[1]         // returns 101.0 (ASCII code for 'e')
println("Hello")     // String literals optimized for direct output
result := "Hello, " + "World!"  // Compile-time concatenation

// Lists (stored as map[uint64]float64)
numbers = [1, 2, 3]
first = numbers[0]
length = #numbers    // length operator

// Maps (native map[uint64]float64)
ages = {1: 25, 2: 30, 3: 35}
empty = {}
count = #ages        // returns 3.0

// Unified indexing (SIMD-optimized for all types)
price = ages[1]      // returns 25.0
missing = ages[999]  // returns 0.0 (key doesn't exist)
result = empty[1]    // returns 0.0 (empty map)
// Note: All indexing operations use SIMD (SSE2/AVX-512) for 2-8× throughput
// Strings, lists, and maps share the same optimized lookup code

// Membership testing with 'in'
10 in numbers {
    -> println("Found!")
    ~> println("Not found")
}

1 in ages {
    -> println("Key exists")
}

result = 5 in mylist  // returns 1.0 or 0.0

// Loops
for i in range(5) {
    println(i)
}

for item in mylist {
    println(item)
}

// Lambdas (up to 6 parameters)
double = (x) -> x * 2
add = (x, y) -> x + y
result = double(5)

// Storing and calling lambdas
f = double
println(f(10))

// Builtin functions
println("text")                     // Print string literal with newline
println(42)                         // Print number with newline
// println(string_var)              // Not yet implemented (TODO)
printf("format %f\n", value)        // Formatted print
printf("Value: %v, Bool: %b\n", x, y)  // %v=smart number, %b=yes/no
range(n)                            // Generate range for loops
// Note: exit() is called automatically at program end
```

## Complete Planned Grammar

The full grammar below shows both implemented and planned features:

```ebnf
program = { statement } ;
statement = assignment | match_statement | expression ;
assignment = identifier [ ":" type_annotation ] ( "=" | ":=" ) expression ;
match_statement = expression "{" "->" expression [ "~>" expression ] "}" ;
expression = parallel_expr | reduction_expr | pipeline_expr |
             lambda_expr | fma_expr | comparison_expr | primary_expr ;
parallel_expr = expression "||" expression ;
reduction_expr = expression "||>" reduction_op ;
pipeline_expr = expression "|" expression ;
lambda_expr = "(" [ param_list ] ")" "->" expression ;
fma_expr = expression "*+" expression "+" expression ;
pattern = literal | identifier | filtered_pattern | guard_pattern | default_pattern ;
filtered_pattern = identifier "{" filter_expr "}" ;
guard_pattern = identifier [ "|" ] expression ;
default_pattern = "~>" ;
filter_expr = comparison_expr | expression ;
comparison_expr = ( ">=" | "<=" | ">" | "<" | "==" | "!=" | "=~" | "!~" ) expression ;
primary_expr = identifier | literal | constant_expr | list_literal | map_literal |
               comprehension | loop | filtered_expr | default_expr |
               property_access | array_access | gather_access | scatter_assign |
               head_expr | tail_expr | guard_expr | early_return_expr |
               error_expr | self_ref | object_def | function_call |
               return_stmt | simd_block | precision_block | "(" expression ")" ;
precision_block = "@" "precision" "(" number ")" "{" { statement } "}" ;
comprehension = "[" expression "for" identifier "in" expression "]" [ "{" ( expression | slice_expr ) "}" ] ;
loop = [ simd_annotation ] "for" identifier "in" expression "{" { statement } "}" ;
filtered_expr = expression "{" ( expression | slice_expr ) "}" ;
slice_expr = [ expression ] ":" [ expression ] [ ":" expression ] ;
default_expr = expression "or" expression ;
guard_expr = expression "or" "return" [ expression ] ;
early_return_expr = expression "or!" expression ;
error_expr = "!" expression ;
property_access = expression "." identifier | "me" "." identifier ;
array_access = expression "." "[" expression "]" ;
gather_access = expression "@" "[" expression "]" ;
scatter_assign = expression "@" "[" expression "]" ":=" expression ;
head_expr = "^" expression ;
tail_expr = "_" expression ;
self_ref = "me" ;
object_def = "@" "{" [ object_member { "," object_member } ] "}" ;
object_member = identifier ":" ( expression | method_def ) ;
method_def = "(" [ param_list ] ")" "->" expression ;
function_call = identifier "(" [ arg_list ] ")" ;
return_stmt = "return" [ expression ] ;
simd_block = "@" simd_annotation "{" { statement } "}" ;
simd_annotation = "simd" [ "(" simd_param { "," simd_param } ")" ] ;
simd_param = "width" "=" ( number | "auto" ) | "aligned" "=" number ;
type_annotation = precision_type | "mask" | "[" type_annotation "]" | identifier ;
precision_type = "b" digit { digit } | "f" digit { digit } ;
reduction_op = "sum" | "product" | "max" | "min" | "any" | "all" ;
list_literal = "[" [ expression { "," expression } ] "]" | "no" ;
map_literal = "{" [ map_entry { "," map_entry } ] "}" ;
map_entry = expression ":" expression ;
type_def = identifier "=" type_expr ;
type_expr = identifier | map_literal | union_type | object_def ;
union_type = variant { "|" variant } ;
variant = identifier [ "{" field_list "}" ] ;
field_list = identifier ":" type_expr { "," identifier ":" type_expr } ;
literal = number | string | regex | "yes" | "no" | "me" ;
constant_expr = "PI" | "E" | "SQRT2" | "LN2" | "LN10" ;
regex = "/" regex_pattern "/" ;
param_list = identifier { "," identifier } ;
arg_list = expression { "," expression } ;
identifier = letter { letter | digit | "_" } ;
number = [ "-" ] digit { digit } [ "." digit { digit } ] ;
string = '"' { character } '"' ;
character = printable_char | escape_sequence ;
escape_sequence = "\" ( "n" | "t" | "r" | "\" | '"' ) ;
letter = "a" | "b" | ... | "z" | "A" | "B" | ... | "Z" ;
digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
```

## Keywords (2)

```
in for
```

## Variable Precision Numbers

Flap supports **variable precision** numeric types, allowing each variable to have its own precision independent of others. This enables mixing precisions for optimal performance and accuracy.

### Precision Type Syntax

Variables can be declared with explicit precision using the `b` (bits) or `f` (float) prefix:

```flap
// Explicit precision types
x:b8 = 42           // 8-bit integer-like precision
y:b16 = 3.14        // 16-bit half-precision float
z:b32 = 3.14159     // 32-bit single-precision float (IEEE 754 binary32)
w:b64 = 3.14159265358979  // 64-bit double-precision float (IEEE 754 binary64)
q:b128 = PI         // 128-bit quad-precision float (IEEE 754 binary128)

// Alternative "f" notation (same as "b")
a:f32 = 1.5         // Same as b32
b:f64 = 2.5         // Same as b64
c:f128 = 3.5        // Same as b128

// Arbitrary precision (multiples of 64)
high:b256 = PI      // 256-bit precision
ultra:b512 = E      // 512-bit precision
extreme:b1024 = SQRT2  // 1024-bit precision
```

### Precision Blocks

The `@precision` annotation sets the default precision for all variables defined within a block:

```flap
// Default precision (64-bit)
x := 42.0           // 64-bit by default

@precision(32) {
    // All variables here default to 32-bit
    fast := 3.14    // 32-bit float
    quick := PI     // PI computed at 32-bit precision

    // Can still override with explicit types
    accurate:b64 = 3.14159265358979  // 64-bit despite block
}

@precision(128) {
    // All variables here default to 128-bit
    precise := E * 2.0   // E computed at 128-bit
    result := sin(x)     // sin() uses 128-bit arithmetic
}
```

### Mixed Precision Arithmetic

Variables of different precisions can be mixed in expressions. The result precision follows these rules:

```flap
x:b32 = 1.5
y:b64 = 2.5
z:b128 = 3.5

// Result precision = max(operand precisions)
a := x + y      // Result is b64 (max of b32, b64)
b := y * z      // Result is b128 (max of b64, b128)
c := x + y + z  // Result is b128 (max of all)

// Explicit result precision
d:b32 = y + z   // Forces 32-bit result (may lose precision)
```

## Mathematical Constants (Precision-Aware)

Mathematical constants automatically adapt to the surrounding precision context:

### Available Constants

```
PI        // π (3.141592653589793...)
E         // e (2.718281828459045...)
SQRT2     // √2 (1.414213562373095...)
LN2       // ln(2) (0.693147180559945...)
LN10      // ln(10) (2.302585092994046...)
```

### Constant Precision Behavior

Constants inherit precision from context:

```flap
// Default 64-bit precision
area := PI * radius * radius  // PI at 64-bit

// Explicit precision
area:b128 = PI * radius * radius  // PI computed at 128-bit

// Block precision
@precision(32) {
    quick := PI * 2.0   // PI at 32-bit
}

@precision(256) {
    ultra := PI * radius * radius  // PI at 256-bit
}
```

### Implementation Strategy

Constants use the most efficient computation method for each precision:

| Precision | Method |
|-----------|--------|
| **8-16 bit** | Precomputed lookup table |
| **32-bit** | x87 FPU (FLDPI, etc.) + round to float32 |
| **64-bit** | x87 FPU instructions (FLDPI, FLDLN2, etc.) |
| **80-bit** | x87 FPU extended precision registers |
| **128-bit** | Double-double arithmetic or quad-precision |
| **256-bit+** | Taylor series, Machin's formula, or other algorithms |

### Precision Inheritance

```flap
@precision(128) {
    // All variables and constants use 128-bit
    circumference := 2.0 * PI * radius

    @precision(64) {
        // Nested block: 64-bit precision
        approx := PI * 2.0

        // Can still access outer scope variables
        result := circumference * 0.5  // Mixed: 128-bit * 64-bit = 128-bit
    }

    // Back to 128-bit
    area := PI * radius * radius
}
```

## Builtin Functions

### I/O Functions (4)

```
println(x)        // Print value to stdout with newline
printf(fmt, ...)  // Formatted print (up to 8 args)
                  // Format specifiers: %f, %d, %s, %v, %b, %%
                  // %v = smart value (42.0→"42", 3.14→"3.14")
                  // %b = boolean (0.0→"no", non-zero→"yes")
exit(code)        // Exit program with code (automatically called at end)
range(n)          // Generate range 0..n-1 for loops
```

### Math Functions (13)

All trigonometric functions use native x87 FPU instructions (no libc dependency):

```
sqrt(x)     // Square root (SQRTSD instruction)
sin(x)      // Sine (FSIN instruction)
cos(x)      // Cosine (FCOS instruction)
tan(x)      // Tangent (FPTAN instruction)
atan(x)     // Arctangent (FPATAN instruction)
asin(x)     // Arcsine (FPATAN + x87 arithmetic)
acos(x)     // Arccosine (FPATAN + x87 arithmetic)

// Planned (x87 FPU instructions available):
abs(x)      // Absolute value (FABS)
floor(x)    // Round down (FRNDINT)
ceil(x)     // Round up (FRNDINT + adjustment)
round(x)    // Round to nearest (FRNDINT)
exp(x)      // e^x (F2XM1 + FSCALE)
log(x)      // Natural logarithm (FYL2X)
pow(x, y)   // x^y (FYL2X + F2XM1 + FSCALE)
```

## Planned Keywords (Not Yet Implemented)

```
and or not yes no me return or!
mask simd sum product max min
```

## Symbols (5)

```
~ @ # ^ _
```

## Complete Feature Set

### Conditional Control Flow

```flap
// Basic match statement (default case optional)
x < y {
    -> println("x is less than y")
}

// Match with both branches
score >= 90 {
    -> grade = "A"
    ~> grade = "B"
}

// Comparison operators: <, <=, >, >=, ==, !=
temperature > 100 {
    -> status = "boiling"
    ~> temperature < 0 {
        -> status = "freezing"
        ~> status = "normal"
    }
}
```

### SIMD-First Design

Flap is built for modern CPUs with explicit SIMD parallelism as a first-class language feature.

```flap
// Explicit parallel operations with || (guaranteed vectorization)
scaled = data || map(x -> x * 2.0)         // 8× parallelism on AVX-512

// Sparse/indexed access with @[] (gather/scatter)
values = map_data@[indices]                 // Single VGATHER instruction
map_data@[indices] := results               // Single VSCATTER instruction

// Mask type for predication
m: mask = values || (x -> x > threshold)    // VCMPPD → k register
filtered = m ? (values || (x -> x * 2)) : values

// Reductions (horizontal operations)
total = values ||> sum                      // Parallel sum
maximum = values ||> max                    // Parallel max

// Fused multiply-add for precision
result = a *+ b + c                         // Single VFMADD (better precision)

// SIMD annotations
@simd(width=8) {
    // Guaranteed to process 8 elements at a time
    results = data || map(process)
}

// Chunk processing
@simd for chunk in data {
    // Each chunk is exactly SIMD width
    chunk || map(x -> x * scale)
}
```

### Clean Error Handling

```flap
// or! for clean error exits
validate_user = (user_data) -> {
    user_data == no or! "no user data"
    not user_data.email or! "email required"
    user_data.email !~ /@/ or! "invalid email format"
    user_data.age < 0 or! "invalid age"

    create_user(user_data)
}

process_file = (filename) -> {
    file_exists(filename) or! "file not found"

    content = read_file(filename) or! "read failed"
    parsed = parse_json(content) or! "invalid json"
    validated = validate_schema(parsed) or! "schema error"

    process_data(validated)
}
```

### Regular Expression Matching

```flap
// =~ for regex match, !~ for no match
text = "hello123"
text =~ "[0-9]+" {
    -> println("contains digits")
    ~> println("no digits")
}

email = "user@example.com"
email =~ "^[a-z]+@[a-z]+\\.[a-z]+$" {
    -> println("valid email")
    ~> println("invalid email")
}
```

### Elegant Self-Reference (Planned)

```flap
// me for clean self-reference in recursive functions
Entity = @{
    x: 0, y: 0,
    health: 100,

    move: (dx, dy) -> {
        me.health <= 0 or! "cannot move dead entity"
        me.x := me.x + dx
        me.y := me.y + dy
        me
    },

    damage: (amount) -> {
        amount <= 0 or! "invalid damage"
        me.health := me.health - amount
        me.health <= 0 and return "destroyed"
        me
    }
}
```

### High-Performance Computing

```flap
// SIMD-accelerated numerical computation
dot_product = (a, b) -> {
    // Vectorized multiply-accumulate
    a || zip(b) || map((x, y) -> x *+ y + 0.0) ||> sum
}

// Parallel filtering with gather
filter_and_transform = (data, indices, threshold) -> {
    // Gather values at indices (VGATHER)
    values = data@[indices]

    // Parallel comparison (VCMPPD)
    m: mask = values || (x -> x > threshold)

    // Masked multiplication (predicated VMULPD)
    m ? (values || (x -> x * 2.0)) : values
}

// Distance calculation (8× parallel)
@simd(width=8)
compute_distances = (entities, target) -> {
    xs = entities.positions@[0:8:x]        // Gather x coords
    ys = entities.positions@[0:8:y]        // Gather y coords

    dxs = xs || (x -> x - target.x)        // VSUBPD
    dys = ys || (y -> y - target.y)        // VSUBPD

    dist_sq = dxs *+ dxs + (dys *+ dys)   // VFMADD
    dist_sq || sqrt                         // VSQRTPD
}
```

### Game Development Ready

```flap
// Game loop with SIMD optimization
GameLoop = @{
    entities: [],
    running: true,

    update: () -> {
        me.running or return "game stopped"

        // Process entities in SIMD chunks
        @simd for chunk in me.entities{health > 0} {
            chunk || map(e -> e.update())
        }

        me.check_collisions()
        me.cleanup_dead_entities()
        me
    },

    check_collisions: () -> {
        for entity in me.entities {
            nearby = [other for other in me.entities]{
                other != entity and
                entity.distance_to(other) < 32
            }

            for other in nearby {
                handle_collision(entity, other)
            }
        }
    },

    cleanup_dead_entities: () -> {
        me.entities := me.entities{entity.health > 0}
    }
}
```

### OS Development Ready

```flap
// Memory manager with robust error handling
MemoryManager = @{
    heap_start: 0x100000,
    heap_size: 0x400000,
    free_list: [],

    init: () -> {
        me.free_list := [@{address: me.heap_start, size: me.heap_size}]
        me
    },

    alloc: (size) -> {
        size > 0 or! "invalid allocation size"
        size <= me.heap_size or! "allocation too large"

        suitable = [block in me.free_list]{block.size >= size}
        suitable == no or! "out of memory"

        block = ^suitable
        me.free_list := me.free_list{b != block}

        // Split block if larger than needed
        block.size > size and {
            remainder = @{
                address: block.address + size,
                size: block.size - size
            }
            me.free_list := me.free_list + [remainder]
        }

        block.address
    },

    free: (address) -> {
        address != 0 or! "null pointer free"
        address >= me.heap_start or! "address before heap"

        // Add to free list and coalesce
        me.add_to_free_list(address)
        me.coalesce()
    }
}
```

## SIMD Performance Characteristics

### Parallel Operator `||`
- Guarantees vectorization (compile error if impossible)
- Processes 8 elements on AVX-512, 4 on AVX2, 2 on SSE2
- Automatically uses best available instruction set

### Gather/Scatter `@[]`
- Single instruction for sparse access (VGATHER/VSCATTER)
- 4-8× faster than serial indexed access
- Critical for map[uint64]float64 workloads

### Mask Type
- Maps to k registers (x86), predicates (ARM64), v0 (RISC-V)
- Enables branchless conditional execution
- First-class predication support

### Reductions `||>`
- Horizontal SIMD operations
- sum, product, max, min, any, all
- Optimal tree reduction implementation

### FMA `*+`
- Fused multiply-add: single instruction, single rounding
- 2× throughput vs separate multiply and add
- Better numerical precision

## Design Goals

The language aims for maximum expressiveness with minimum complexity, backed by a map[uint64]float64 foundation and explicit SIMD semantics that enable both high performance and elegant abstractions.

### Key Principles

1. **Explicit over implicit** - SIMD operations are visible in the code
2. **Performance by default** - Modern instructions used automatically
3. **Simple foundation** - map[uint64]float64 + float64 + functions
4. **Functional style** - Immutability preferred, mutation explicit
5. **No magic** - What you see is what you get

## Automatic Dependency Resolution

Flap uses automatic dependency resolution - there are no explicit `import` statements. When the compiler encounters an unknown function, it automatically fetches and compiles the required code from predefined repositories.

### How It Works

The `flapc` compiler maintains a hard-coded map of function names to Git repository URLs:

```
abs       -> github.com/xyproto/flap_math
sin       -> github.com/xyproto/flap_math
cos       -> github.com/xyproto/flap_math
tan       -> github.com/xyproto/flap_math
InitWindow -> github.com/xyproto/flap_raylib
```

When compiling code that calls an unknown function:

1. **Resolution**: The compiler looks up the function in its repository map
2. **Caching**: It clones the repository to `~/.cache/flapc/<repo-url>/`
3. **Integration**: All `.flap` files from the repository are added to the compilation
4. **Compilation**: The combined code is compiled as a single unit

### Example

```flap
// No import needed!
x := -5
y := abs(x)      // Compiler automatically fetches flap_math
println(y)       // Outputs: 5
```

When this code is compiled:
1. Compiler encounters `abs()`
2. Looks up `abs` → `github.com/xyproto/flap_math`
3. Clones repo to `~/.cache/flapc/github.com/xyproto/flap_math/`
4. Includes all `.flap` files from that repo
5. Compiles everything together

### Benefits

- **Zero boilerplate**: No import statements needed
- **Automatic updates**: Repositories can be re-fetched with `--update-deps`
- **Simple distribution**: Just write functions and push to Git
- **Dependency isolation**: Each repo is versioned and cached separately

### Cache Management

```bash
# View cached dependencies
ls ~/.cache/flapc/

# Update all dependencies
flapc --update-deps myprogram.flap

# Clear cache
rm -rf ~/.cache/flapc/
```

### Creating a Library

To create a Flap library:

1. Create a Git repository
2. Write pure Flap code defining functions
3. Add your functions to the compiler's repository map (via PR or local config)
4. Users can immediately use your functions without imports

Example `flap_math` repository structure:
```
flap_math/
├── abs.flap
├── trig.flap
├── pow.flap
└── README.md
```

Each `.flap` file contains pure Flap function definitions that will be automatically included when needed.
