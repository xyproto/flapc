# The GGG Programming Language



Float. Map. Fly.



### ebnf



program = { statement } ;



statement = assignment | match_assignment | expression ;



assignment = identifier ( "=" | ":=" ) expression ;



match_assignment = identifier "=~" identifier "{" { pattern "->" expression } "}" ;



expression = pipeline_expr | match_expr | lambda_expr | primary_expr ;



pipeline_expr = expression "|" expression ;



match_expr = "~" expression "{" { pattern "->" expression } "}" ;



lambda_expr = "(" [ param_list ] ")" "->" expression ;



pattern = literal | identifier | filtered_pattern | guard_pattern | default_pattern ;



filtered_pattern = identifier "{" filter_expr "}" ;



guard_pattern = identifier [ "|" ] expression ;



default_pattern = "~>" ;



filter_expr = comparison_expr | expression ;



comparison_expr = ( ">=" | "<=" | ">" | "<" | "==" | "!=" | "=~" | "!~" ) expression ;



primary_expr = identifier | literal | list_literal | map_literal |

               comprehension | loop | filtered_expr | default_expr |

               property_access | array_access | head_expr | tail_expr |

               guard_expr | early_return_expr | error_expr | self_ref | object_def |

               function_call | return_stmt | "(" expression ")" ;



comprehension = "[" expression "in" expression "]" [ "{" ( expression | slice_expr ) "}" ] ;



loop = "@" identifier "in" expression "{" { statement } "}" ;



filtered_expr = expression "{" ( expression | slice_expr ) "}" ;



slice_expr = [ expression ] ":" [ expression ] [ ":" expression ] ;



default_expr = expression "or" expression ;



guard_expr = expression "or" "return" [ expression ] ;



early_return_expr = expression "or!" expression ;



error_expr = "!" expression ;



property_access = expression "." identifier | "me" "." identifier ;



array_access = expression "." "[" expression "]" ;



head_expr = "^" expression ;



tail_expr = "_" expression ;



self_ref = "me" ;



object_def = "@" "{" [ object_member { "," object_member } ] "}" ;



object_member = identifier ":" ( expression | method_def ) ;



method_def = "(" [ param_list ] ")" "->" expression ;



function_call = identifier "(" [ arg_list ] ")" ;



return_stmt = "return" [ expression ] ;



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





### Keywords (9)



    in and or not yes no me return or!



### Symbols (5)



    ~ @ # ^ _



### Complete Feature Set



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



### Game Development Ready



```

// Game loop with error handling

GameLoop = @{

    entities: [],

    running: true,



    update: () -> {

        me.running or return "game stopped"



        @ entity in me.entities{health > 0} {

            entity.update() or! "entity update failed"

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



// Process scheduler

schedule_next = (processes) -> {

    ready_processes = [p in processes]{p.state == "ready"}

    ready_processes == no or return no



    # Round-robin scheduling

    next = ^ready_processes

    next.state := "running"

    next.time_slice := 10  # 10ms



    next

}

```



The language aims for maximum expressiveness with minimum complexity, backed by a map[float64]float64 foundation that enables both high performance and elegant abstractions.
