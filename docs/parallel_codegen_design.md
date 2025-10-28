# Parallel Loop Code Generation Design

## Status: Design Document (Implementation 85% Complete)

This document describes the assembly code generation strategy for parallel loops in Flap.

## Overview

Parallel loops use Linux `clone()` syscalls to spawn worker threads that execute loop iterations concurrently. Each thread receives a work range and executes the loop body for its assigned items.

## Syntax

```flap
@@ i in 0..<100 {     // All CPU cores (detected: 2 cores)
    printf("%v\n", i)
}

4 @ i in 0..<100 {    // Exactly 4 threads
    printf("%v\n", i)
}
```

## Infrastructure (Complete)

### Phase 1-5, 7: Foundation ✅
- **Lexer**: TOKEN_AT_AT for `@@` syntax
- **Parser**: Handles `@@`, `N @`, and `@` variants
- **AST**: LoopStmt.NumThreads field (0=sequential, -1=all cores, N=specific)
- **Thread Spawning**: CloneThread() wrapper around clone() syscall
- **CPU Detection**: GetNumCPUCores() reads /proc/cpuinfo
- **Work Distribution**: CalculateWorkDistribution(), GetThreadWorkRange()
- **Synchronization**: Futex-based Barrier (tested with 4 threads)

## Assembly Code Generation Strategy (Phase 6 & 8)

### Memory Layout

```
Stack Layout for Parallel Loop:
┌─────────────────────────────────┐
│ Barrier structure (8 bytes)     │  rbp-8   (barrier.count: int32)
│                                  │  rbp-12  (barrier.total: int32)
├─────────────────────────────────┤
│ Thread args array                │  rbp-16-(N*24)
│  [N threads * 24 bytes each]     │
│  struct ThreadArgs {             │
│    start: int64    (8 bytes)     │
│    end: int64      (8 bytes)     │
│    loop_body: ptr  (8 bytes)     │
│  }                               │
├─────────────────────────────────┤
│ Thread stacks (allocated)        │
│  [N threads * 1MB each]          │
└─────────────────────────────────┘
```

### Code Generation Steps

#### Step 1: Prologue - Allocate Shared Memory

```asm
; Calculate total stack needed
; barrier (16 bytes) + thread_args (N * 24 bytes) + padding
mov rax, <N * 24 + 16>
sub rsp, rax

; Initialize barrier
mov dword [rbp-8], <N>         ; barrier.count = N
mov dword [rbp-12], <N>        ; barrier.total = N
```

#### Step 2: Calculate Work Ranges

```asm
; For each thread i in 0..<N:
;   start = i * chunk_size
;   end = (i+1) * chunk_size
;   if i == N-1: end += remainder

; Thread 0: [0, 50)
lea rax, [rbp-16-0]            ; &thread_args[0]
mov qword [rax+0], 0           ; start = 0
mov qword [rax+8], 50          ; end = 50
lea rcx, thread_body_start
mov qword [rax+16], rcx        ; loop_body = &body

; Thread 1: [50, 100)
lea rax, [rbp-16-24]           ; &thread_args[1]
mov qword [rax+0], 50          ; start = 50
mov qword [rax+8], 100         ; end = 100
lea rcx, thread_body_start
mov qword [rax+16], rcx        ; loop_body = &body
```

#### Step 3: Spawn Threads

```asm
; For each thread:
;   1. Allocate stack (1MB)
;   2. Call clone() syscall
;   3. Child thread jumps to thread_entry
;   4. Parent continues to next thread

mov r15, 0                     ; Thread counter

.spawn_loop:
    cmp r15, <N>
    jge .spawn_done

    ; Allocate thread stack (1MB)
    mov rax, 9                 ; sys_mmap
    xor rdi, rdi               ; addr = NULL
    mov rsi, 1048576           ; size = 1MB
    mov rdx, 3                 ; PROT_READ | PROT_WRITE
    mov r10, 34                ; MAP_PRIVATE | MAP_ANONYMOUS
    mov r8, -1                 ; fd = -1
    xor r9, r9                 ; offset = 0
    syscall

    ; rax now contains stack base
    mov r14, rax               ; Save stack base
    add rax, 1048576           ; Stack top (grows downward)
    sub rax, 16                ; Alignment

    ; Calculate thread args pointer
    lea rbx, [rbp-16]
    mov rcx, 24
    imul rcx, r15
    sub rbx, rcx               ; rbx = &thread_args[r15]

    ; Store thread args on child stack
    mov rcx, [rbx+0]           ; start
    mov [rax-8], rcx
    mov rcx, [rbx+8]           ; end
    mov [rax-16], rcx
    mov rcx, [rbx+16]          ; loop_body
    mov [rax-24], rcx
    lea rcx, [rbp-8]           ; &barrier
    mov [rax-32], rcx

    ; Call clone()
    mov rax, 56                ; sys_clone
    mov rdi, 0x00010F00        ; CLONE_VM | CLONE_FS | ...
    mov rsi, rax               ; Stack top
    xor rdx, rdx
    xor r10, r10
    xor r8, r8
    syscall

    ; Check if parent or child
    test rax, rax
    jz .child_thread

    ; Parent: save TID and continue
    ; (rax contains child TID)
    inc r15
    jmp .spawn_loop

.spawn_done:
    jmp .wait_for_threads
```

#### Step 4: Thread Entry Point

```asm
.child_thread:
    ; Load thread arguments from stack
    mov rbx, rsp
    mov r12, [rbx-8]           ; start
    mov r13, [rbx-16]          ; end
    mov r14, [rbx-24]          ; loop_body
    mov r15, [rbx-32]          ; &barrier

    ; Execute loop body for range [r12, r13)
    mov rcx, r12               ; i = start
.thread_loop:
    cmp rcx, r13
    jge .thread_loop_done

    ; Set up loop variable
    ; (Convert integer to float64 for iterator)
    cvtsi2sd xmm0, rcx
    movsd [rbp-<iterator_offset>], xmm0

    ; Execute loop body
    ; (This would be the compiled loop body statements)
    call r14                   ; or inline body here

    inc rcx
    jmp .thread_loop

.thread_loop_done:
    ; Decrement barrier counter atomically
    mov rax, -1
    lock xadd dword [r15], eax ; atomic_add(&barrier.count, -1)
    dec eax                    ; eax now has new value

    ; Check if last thread
    test eax, eax
    jnz .thread_wait

    ; Last thread: wake all
    mov rax, 202               ; sys_futex
    mov rdi, r15               ; &barrier.count
    mov rsi, 129               ; FUTEX_WAKE_PRIVATE
    mov rdx, [r15+4]           ; barrier.total
    syscall
    jmp .thread_exit

.thread_wait:
    ; Not last: wait on futex
    mov rax, 202               ; sys_futex
    mov rdi, r15               ; &barrier.count
    mov rsi, 128               ; FUTEX_WAIT_PRIVATE
    mov rdx, [r15]             ; current value
    syscall
    jmp .thread_exit

.thread_exit:
    ; Exit thread
    mov rax, 60                ; sys_exit
    xor rdi, rdi
    syscall
```

#### Step 5: Parent Waits for Completion

```asm
.wait_for_threads:
    ; Parent thread waits on barrier
    lea rdi, [rbp-8]           ; &barrier.count
    mov rax, 202               ; sys_futex
    mov rsi, 128               ; FUTEX_WAIT_PRIVATE
    mov edx, [rdi]             ; current value
    syscall

    ; All threads complete - continue execution
```

#### Step 6: Cleanup

```asm
    ; Deallocate shared memory
    mov rax, <N * 24 + 16>
    add rsp, rax

    ; Continue to next statement
```

## Simplified V1 Implementation

For initial implementation, we can simplify:

### V1: Single Thread Test
```asm
; Just spawn 1 thread to verify clone() works in generated code
; Thread prints "Hello from thread" and exits
; No barrier, no work distribution
```

### V2: Multiple Threads with Simple Body
```asm
; Spawn N threads, each prints its thread ID
; Use simple sleep for synchronization
; No complex loop body execution yet
```

### V3: Work Distribution
```asm
; Spawn N threads with work ranges
; Each thread executes simple printf with its range
; Add barrier synchronization
```

### V4: Full Implementation
```asm
; Spawn N threads with work ranges
; Execute actual loop body in each thread
; Full barrier synchronization
; Cleanup and error handling
```

## Testing Strategy

### Test 1: Single Thread Spawn
```flap
2 @ i in 0..<10 {
    // Empty body - just verify threads spawn
}
```

### Test 2: Simple Printf
```flap
2 @ i in 0..<10 {
    printf("Thread: %v\n", i)
}
```

### Test 3: Work Distribution Verification
```flap
4 @ i in 0..<100 {
    printf("Item %v\n", i)
}
// Should print all numbers 0-99 exactly once
// Order may vary due to parallelism
```

### Test 4: CPU Detection
```flap
@@ i in 0..<1000 {
    // Should use actual CPU count
}
```

## Performance Considerations

### Thread Overhead
- Thread creation: ~50μs per thread
- Context switch: ~3μs
- Futex wake: ~1μs

For loops with <1000 iterations, overhead may exceed benefits.

**Recommendation**: Only parallelize loops with >1000 iterations or expensive body.

### Memory Access Patterns
- False sharing: Ensure threads don't write to adjacent cache lines
- Prefetching: Sequential access within thread is cache-friendly

### Optimal Thread Count
- CPU-bound: N = num_cores
- I/O-bound: N = 2 * num_cores (to hide latency)
- Memory-bound: N = num_memory_channels (avoid bandwidth saturation)

## Error Handling

### Clone Failure
```asm
; If clone() returns -1:
test rax, rax
jns .clone_success
; Handle error: fall back to sequential?
; Or abort with error message?
```

### Stack Allocation Failure
```asm
; If mmap() returns -1:
cmp rax, -1
jne .mmap_success
; Handle error
```

## Future Enhancements

### Dynamic Work Stealing
- Implement work queue instead of static ranges
- Threads that finish early can steal work from busy threads

### NUMA Awareness
- Pin threads to CPU cores
- Allocate memory on local NUMA node

### Auto-Parallelization
- Compiler heuristic: parallelize if iterations > 1000
- Profile-guided: use runtime profiling to decide

## References

- Linux `clone()` man page
- Futex documentation: kernel.org/doc/Documentation/futex-requeue-pi.txt
- pthread_barrier_wait implementation in glibc
- Go runtime scheduler: golang.org/src/runtime/proc.go

## Implementation Status

| Component | Status | Lines | Tests |
|-----------|--------|-------|-------|
| Lexer & Parser | ✅ Complete | 65 | ✅ |
| AST | ✅ Complete | 15 | ✅ |
| Thread Spawning | ✅ Complete | 120 | ✅ |
| Work Distribution | ✅ Complete | 30 | ✅ |
| Futex Barrier | ✅ Complete | 115 | ✅ |
| CPU Detection | ✅ Complete | 40 | ✅ |
| Assembly Codegen | ⏳ Pending | 0 | - |

**Total Complete: 385 lines, 6/8 phases (75%)**

**Remaining: Assembly code generation (~200-300 lines estimated)**
