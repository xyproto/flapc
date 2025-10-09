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
// Note: Runtime string operations (println(string_var)) not yet implemented

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
@ i in range(5) {
    println(i)
}

@ item in mylist {
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
primary_expr = identifier | literal | list_literal | map_literal |
               comprehension | loop | filtered_expr | default_expr |
               property_access | array_access | gather_access | scatter_assign |
               head_expr | tail_expr | guard_expr | early_return_expr |
               error_expr | self_ref | object_def | function_call |
               return_stmt | simd_block | "(" expression ")" ;
comprehension = "[" expression "in" expression "]" [ "{" ( expression | slice_expr ) "}" ] ;
loop = [ simd_annotation ] "@" identifier "in" expression "{" { statement } "}" ;
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
type_annotation = "mask" | "float64" | "[" type_annotation "]" | identifier ;
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
regex = "/" regex_pattern "/" ;
param_list = identifier { "," identifier } ;
arg_list = expression { "," expression } ;
identifier = letter { letter | digit | "_" } ;
number = digit { digit } [ "." digit { digit } ] ;
string = '"' { character } '"' ;
character = printable_char | escape_sequence ;
escape_sequence = "\" ( "n" | "t" | "r" | "\" | '"' ) ;
letter = "a" | "b" | ... | "z" | "A" | "B" | ... | "Z" ;
digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
```

## Keywords (1)

```
in
```

## Builtin Functions (4)

```
println(x)        // Print value to stdout with newline
printf(fmt, ...)  // Formatted print (up to 8 args)
                  // Format specifiers: %f, %d, %s, %v, %b, %%
                  // %v = smart value (42.0→"42", 3.14→"3.14")
                  // %b = boolean (0.0→"no", non-zero→"yes")
exit(code)        // Exit program with code (automatically called at end)
range(n)          // Generate range 0..n-1 for loops
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
        @ entity in me.entities {
            nearby = [other in me.entities]{
                other != entity and
                entity.distance_to(other) < 32
            }

            @ other in nearby {
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
