# The GGG Programming Language



Float. Map. Fly.



### ebnf



program = { statement } ;



statement = assignment | match_assignment | if_statement | expression ;



assignment = identifier [ ":" type_annotation ] ( "=" | ":=" ) expression ;



match_assignment = identifier "=~" identifier "{" { pattern "->" expression } "}" ;



if_statement = "if" comparison_expr { statement } [ "else" { statement } ] "end" ;



expression = parallel_expr | reduction_expr | pipeline_expr | match_expr |
             lambda_expr | fma_expr | comparison_expr | primary_expr ;



parallel_expr = expression "||" expression ;



reduction_expr = expression "||>" reduction_op ;



pipeline_expr = expression "|" expression ;



match_expr = "~" expression "{" { pattern "->" expression } "}" ;



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





### Keywords (18)

    in and or not yes no me return or!
    mask simd sum product max min
    if else end



### Symbols (5)

    ~ @ # ^ _



### Complete Feature Set



#### Conditional Control Flow

Flap uses if/else/end blocks with comparison operators for branching logic.

```
// Basic if statement
if x < y
    println("x is less than y")
end

// If/else branches
if score >= 90
    grade = "A"
else
    grade = "B"
end

// Comparison operators: <, <=, >, >=, ==, !=
if temperature > 100
    status = "boiling"
else
    if temperature < 0
        status = "freezing"
    else
        status = "normal"
    end
end

// Comparisons use UCOMISD (unordered scalar double compare)
// Efficient floating-point comparisons with proper NaN handling
if distance <= threshold
    trigger_event()
end
```



#### SIMD-First Design

Flap is built for modern CPUs with explicit SIMD parallelism as a first-class language feature.

```
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



#### Clean Error Handling

```
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

// Standalone error assertions
allocate_memory = (size) -> {
    size > 0 or! "invalid size"
    size <= MAX_MEMORY or! "size too large"

    ptr = malloc(size)
    ptr != 0 or! "allocation failed"
    ptr
}
```



### Elegant Self-Reference

```
// me for clean self-reference
factorial =~ n {
    n <= 1 -> 1
    ~> n * me(n - 1)
}

quicksort =~ lst {
    l | #l <= 1 -> l
    ~> {
        pivot = ^l
        rest = _l
        smaller = [x in rest]{x < pivot}
        larger = [x in rest]{x >= pivot}
        me(smaller) + [pivot] + me(larger)
    }
}

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

```
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

// Matrix multiplication with FMA
matmul = (A, B) -> {
    @ i in range(rows) {
        @ j in range(cols) {
            // Vectorized dot product with FMA
            row = A[i]
            col = [B[k][j] for k in range(cols)]
            result[i][j] = (row || zip(col) || map((a, b) -> a *+ b + 0.0)) ||> sum
        }
    }
}
```



### Game Development Ready

```
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

// Particle system with parallel updates
ParticleSystem = @{
    positions: [],
    velocities: [],
    lifetimes: [],

    update: (dt) -> {
        // All arrays updated in parallel
        @simd(width=8) {
            // Fused multiply-add for position updates
            me.positions := me.positions || zip(me.velocities) ||
                           map((p, v) -> p *+ v + dt)

            // Parallel lifetime decrement
            me.lifetimes := me.lifetimes || (t -> t - dt)

            // Mask-based filtering (keep alive particles)
            alive: mask = me.lifetimes || (t -> t > 0.0)
            me.positions := alive ? me.positions : []
            me.velocities := alive ? me.velocities : []
            me.lifetimes := alive ? me.lifetimes : []
        }
    }
}
```



### OS Development Ready

```
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

        # Split block if larger than needed
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

        # Add to free list and coalesce
        me.add_to_free_list(address)
        me.coalesce()
    }
}

// Process scheduler with vectorized priority calculation
schedule_next = (processes) -> {
    ready_processes = [p in processes]{p.state == "ready"}
    ready_processes == no or return no

    // Parallel priority calculation
    priorities = ready_processes || map(p -> p.priority * p.wait_time)
    max_priority = priorities ||> max

    # Find process with max priority
    next = ready_processes{p.priority * p.wait_time == max_priority}[0]
    next.state := "running"
    next.time_slice := 10  # 10ms

    next
}
```



### SIMD Performance Characteristics

**Parallel Operator `||`**
- Guarantees vectorization (compile error if impossible)
- Processes 8 elements on AVX-512, 4 on AVX2, 2 on SSE2
- Automatically uses best available instruction set

**Gather/Scatter `@[]`**
- Single instruction for sparse access (VGATHER/VSCATTER)
- 4-8× faster than serial indexed access
- Critical for map[uint64]float64 workloads

**Mask Type**
- Maps to k registers (x86), predicates (ARM64), v0 (RISC-V)
- Enables branchless conditional execution
- First-class predication support

**Reductions `||>`**
- Horizontal SIMD operations
- sum, product, max, min, any, all
- Optimal tree reduction implementation

**FMA `*+`**
- Fused multiply-add: single instruction, single rounding
- 2× throughput vs separate multiply and add
- Better numerical precision



The language aims for maximum expressiveness with minimum complexity, backed by a map[uint64]float64 foundation and explicit SIMD semantics that enable both high performance and elegant abstractions.
