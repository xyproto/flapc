# Spawn with Result Waiting Design

## Overview

Currently, `spawn` only supports fire-and-forget process spawning. This design adds communication to enable fork/join patterns where the parent waits for and uses the child's result.

## Implementation Strategy

**Recommendation: Use Channels instead of raw pipes**

After reviewing the existing plans, implementing channels (as described in CHANNELS_AND_ENET_PLAN.md) would provide a better foundation:

1. **Channels are higher-level** - Easier to use than raw pipe syscalls
2. **Thread-safe by design** - Built-in synchronization
3. **More flexible** - Can handle multiple spawns, select statements, timeouts
4. **Consistent with language** - Same primitives for threads and processes
5. **ENet compatibility** - Channels can eventually work over ENet for distributed computing

**Migration Path:**
1. First implement channels (CHANNELS_AND_ENET_PLAN.md Part 1)
2. Use channels for spawn communication
3. Later add ENet backend for distributed spawns

## Proposed Channel-Based Design

### Syntax with Channels

```flap
// Create channel for result
result_ch := chan()

// Spawn with channel communication
spawn {
    result := expensive_computation()
    result_ch <- result  // Send result to channel
}

// Wait for result
value := <-result_ch  // Receive from channel
println("Got result:", value)
```

Or with the pipe syntax sugar:

```flap
// Syntactic sugar: automatically creates channel and waits
result = spawn expensive_computation() | value | {
    println("Computation returned:", value)
    value * 2
}
```

**Desugaring:**
```flap
// The above desugars to:
__spawn_ch_1 := chan()
spawn {
    __spawn_result_1 := expensive_computation()
    __spawn_ch_1 <- __spawn_result_1
}
value := <-__spawn_ch_1
result = {
    println("Computation returned:", value)
    value * 2
}
```

### Benefits Over Raw Pipes

1. **Type-safe** - Channels can be typed (future enhancement)
2. **Buffered** - Can create buffered channels for non-blocking sends
3. **Select support** - Can wait on multiple spawns with `select`
4. **Timeout support** - Built into select statement
5. **Multiple consumers** - Channels support fan-out patterns
6. **Clean semantics** - Send/receive operators are clear

### Example: Multiple Spawns with Select

```flap
ch1 := chan()
ch2 := chan()

spawn { ch1 <- compute_task1() }
spawn { ch2 <- compute_task2() }

// Wait for first to complete
select {
    result := <-ch1 -> {
        println("Task 1 finished first:", result)
    }
    result := <-ch2 -> {
        println("Task 2 finished first:", result)
    }
}
```

### Implementation Requirements

**Prerequisites:**
1. Implement channels (CHANNELS_AND_ENET_PLAN.md Part 1)
   - `chan()` creation
   - `<-` send operator
   - `<-` receive operator
   - `close()` function
   - `select` statement

**Spawn Integration:**
1. Parse pipe syntax: `spawn expr | params | block`
2. Desugar to channel creation + spawn + receive + block
3. Handle multiple parameters (multiple channel sends/receives)

## Original Pipe-Based Design (Deferred)

The original design used raw Unix pipes. This is kept for reference but **deferred** in favor of channels.

## Current Implementation

```flap
// Fire-and-forget: child runs independently
spawn background_task()

// Fire-and-forget with block (currently errors)
spawn computation() | result | {
    println("Got:", result)  // NOT YET IMPLEMENTED
}
```

**Current behavior:**
- `spawn expr` forks a child process
- Child executes `expr` and exits
- Parent continues immediately (no waiting)
- No way to get child's result

## Proposed Implementation

### Syntax

```flap
// Wait for result and use it
result = spawn expensive_computation() | value | {
    println("Computation returned:", value)
    value * 2  // Last expression is block's return value
}
```

### Semantics

1. **Create pipe**: Parent calls `pipe()` syscall to create file descriptors
2. **Fork**: Parent calls `fork()` to create child process
3. **Child path**:
   - Close read end of pipe
   - Execute expression
   - Write result (as float64) to pipe write end
   - Close write end
   - Exit
4. **Parent path**:
   - Close write end of pipe
   - Read result (as float64) from pipe read end
   - Close read end
   - Bind result to parameter name(s)
   - Execute block with bound parameters
   - Block's return value becomes `spawn` expression's value

### Implementation Details

#### Pipe Creation (Linux x86-64)

```assembly
; Create pipe: int pipe(int pipefd[2])
; pipefd[0] = read end, pipefd[1] = write end
sub rsp, 16          ; Allocate space for 2 file descriptors
mov rax, 22          ; pipe syscall number
mov rdi, rsp         ; Pointer to pipefd array
syscall
; Now [rsp] = read fd, [rsp+8] = write fd
```

#### Fork and Communication

```assembly
; Save pipe FDs
mov r13, [rsp]       ; r13 = read fd
mov r14, [rsp+8]     ; r14 = write fd

; Fork
mov rax, 57          ; fork syscall
syscall
test rax, rax
jz .child            ; Jump if child (rax == 0)

; Parent: close write end, read result
.parent:
    ; Close write end
    mov rax, 3       ; close syscall
    mov rdi, r14     ; write fd
    syscall

    ; Read result from pipe
    mov rax, 0       ; read syscall
    mov rdi, r13     ; read fd
    lea rsi, [rbp-X] ; Buffer for result (8 bytes for float64)
    mov rdx, 8       ; Size
    syscall

    ; Close read end
    mov rax, 3       ; close syscall
    mov rdi, r13     ; read fd
    syscall

    ; Load result into xmm0
    movsd xmm0, [rbp-X]

    ; Execute block with result bound to parameter
    ; ...block code here...

    jmp .continue

; Child: close read end, compute, write, exit
.child:
    ; Close read end
    mov rax, 3       ; close syscall
    mov rdi, r13     ; read fd
    syscall

    ; Execute spawned expression (result in xmm0)
    ; ...expression code here...

    ; Write result to pipe
    movsd [rbp-Y], xmm0  ; Store xmm0 to memory
    mov rax, 1           ; write syscall
    mov rdi, r14         ; write fd
    lea rsi, [rbp-Y]     ; Source buffer
    mov rdx, 8           ; Size (float64)
    syscall

    ; Close write end
    mov rax, 3       ; close syscall
    mov rdi, r14     ; write fd
    syscall

    ; Exit child
    mov rax, 60      ; exit syscall
    xor rdi, rdi     ; status 0
    syscall

.continue:
    ; Parent continues here with result in xmm0
```

### Multiple Parameters

For destructuring:

```flap
spawn get_point() | x, y | {
    println("Point:", x, y)
}
```

The child would need to write multiple values:
- Write first value (8 bytes)
- Write second value (8 bytes)
- Parent reads both in order

### Error Handling

Possible errors:
1. **Pipe creation fails**: Report error, don't fork
2. **Fork fails**: Close pipe FDs, report error
3. **Read fails**: Close FDs, report error with suggestion
4. **Write fails**: Child exits with error status

For now, we'll use simple error handling:
- If any syscall fails, print error and exit
- Future: integrate with ErrorCollector for better error reporting

## Testing Strategy

### Basic Test

```flap
// Test: spawn with result
result = spawn { 42 } | value | {
    println("Got:", value)
    value * 2
}
println("Final:", result)  // Should print 84
```

### Multiple Parameters Test

```flap
// Test: multiple return values
spawn {
    write_float(10.0)
    write_float(20.0)
} | x, y | {
    println("x:", x, "y:", y)
    x + y
}
```

### Fork/Join Pattern Test

```flap
// Test: parallel computation with join
a = spawn compute_heavy_1() | result | { result }
b = spawn compute_heavy_2() | result | { result }
total = a + b  // Both spawns have completed by this point
```

## Implementation Plan

1. **Add pipe syscall wrapper** to Out (codegen helper)
2. **Modify compileSpawnStmt**:
   - Check if `stmt.Block != nil`
   - If yes, implement pipe-based communication
   - If no, keep current fire-and-forget behavior
3. **Handle parameter binding**:
   - Create temporary variables for pipe parameters
   - Bind them before executing block
4. **Test with simple cases first**:
   - Single parameter, simple block
   - Then multiple parameters
   - Then complex expressions

## Future Enhancements

- **Timeout support**: `spawn expr | value | timeout(1s) { ... }`
- **Error propagation**: If child process crashes, parent gets error
- **Multiple spawns**: `results = spawn [task1(), task2(), task3()] | values | { ... }`
- **Channel-based communication**: More sophisticated than simple pipes

## Notes

- This design uses anonymous pipes (pipe syscall), not named FIFOs
- Pipes are unidirectional: child writes, parent reads
- Pipes have finite buffer (usually 64KB on Linux)
- For large data, would need multiple read/write calls
- Currently assumes float64 values only (8 bytes each)
