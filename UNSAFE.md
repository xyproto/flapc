# Unsafe Blocks: Battlestar Assembly Language

Flap's `unsafe` blocks provide direct register access across x86_64, ARM64, and RISC-V architectures. This Battlestar-inspired sublanguage allows low-level systems programming while maintaining portability.

## Table of Contents
- [Overview](#overview)
- [Register Aliases](#register-aliases)
- [Syntax](#syntax)
- [Operations](#operations)
- [Return Values](#return-values)
- [Examples](#examples)

## Overview

Unsafe blocks execute architecture-specific machine code while integrating seamlessly with Flap's high-level features.

**Unified approach** (recommended - uses register aliases):
```flap
result := unsafe {
    a <- 42      // Load immediate (works on all CPUs)
    b <- 100     // Register aliases: a, b, c
    c <- a + b   // Register arithmetic
    c            // Return value (last expression)
}
```

**Per-CPU approach** (when platform-specific code is needed):
```flap
result := unsafe {
    x86_64 {
        rax <- 42
        rbx <- 100
        rcx <- rax + rbx
        rcx
    }
    arm64 {
        x0 <- 42
        x1 <- 100
        x2 <- x0 + x1
        x2
    }
    riscv64 {
        a0 <- 42
        a1 <- 100
        a2 <- a0 + a1
        a2
    }
}
```

## Register Aliases

Use portable register aliases to write **unified unsafe code** that works across all architectures:

| Alias | x86_64  | ARM64 | RISC-V | Purpose              |
|-------|---------|-------|--------|----------------------|
| `a`   | `rax`   | `x0`  | `a0`   | Accumulator/arg 0    |
| `b`   | `rbx`   | `x1`  | `a1`   | Base register/arg 1  |
| `c`   | `rcx`   | `x2`  | `a2`   | Count register/arg 2 |
| `d`   | `rdx`   | `x3`  | `a3`   | Data register/arg 3  |
| `e`   | `rsi`   | `x4`  | `a4`   | Source index/arg 4   |
| `f`   | `rdi`   | `x5`  | `a5`   | Dest index/arg 5     |
| `s`   | `rsp`   | `sp`  | `sp`   | Stack pointer        |
| `p`   | `rbp`   | `fp`  | `fp`   | Frame pointer        |

**Unified Example:**
```flap
// Works on ALL architectures - no per-CPU blocks needed!
value := unsafe {
    a <- 0x2A    // Load 42 into accumulator
    a            // Return accumulator
}
```

## Syntax

### Per-Architecture Blocks

Specify different implementations for each CPU with labeled blocks:

```flap
result := unsafe {
    x86_64 {
        rax <- 100
        rbx <- rax
        rbx
    }
    arm64 {
        x0 <- 100
        x1 <- x0
        x1
    }
    riscv64 {
        a0 <- 100
        a1 <- a0
        a1
    }
}
```

### Unified Blocks (Recommended)

Use register aliases for portable code:

```flap
result := unsafe {
    a <- 100     // Works everywhere
    b <- a
    b
}
```

## Operations

### Immediate Loads
```flap
a <- 42          // Decimal
a <- 0xFF        // Hexadecimal
a <- 0b1010      // Binary
```

### Register Moves
```flap
a <- 100
b <- a           // Copy a to b
c <- b           // Copy b to c
```

### Arithmetic Operations
```flap
a <- 10
b <- 20
c <- a + b       // Addition
d <- a - b       // Subtraction
e <- a * b       // Multiplication
f <- a / b       // Division (unsigned)
```

### Bitwise Operations
```flap
a <- 0xFF
b <- 0x0F
c <- a & b       // AND
d <- a | b       // OR
e <- a ^ b       // XOR
f <- ~a          // NOT
```

### Shifts and Rotates
```flap
a <- 8
b <- a << 2      // Shift left
c <- a >> 1      // Shift right
d <- a rol 4     // Rotate left
e <- a ror 2     // Rotate right
```

### Memory Access
```flap
// Load from memory
a <- [b]                // Load 64-bit from address in b
a <- [b + 16]           // Load from b + offset
a <- u8 [b]             // Load 8-bit (zero-extended)
a <- u16 [b + 4]        // Load 16-bit + offset
a <- u32 [b]            // Load 32-bit
a <- i8 [b]             // Load signed 8-bit
a <- i16 [b]            // Load signed 16-bit
a <- i32 [b]            // Load signed 32-bit

// Store to memory
[a] <- 42               // Store immediate
[a + 8] <- b            // Store register to offset
[a] <- b as u8          // Store 8-bit
[a] <- b as u16         // Store 16-bit
[a] <- b as u32         // Store 32-bit
```

### Stack Operations
```flap
stack <- a       // Push a onto stack
b <- stack       // Pop stack into b
```

### System Calls
```flap
// Set up syscall arguments, then invoke
a <- 1           // Syscall number (write)
b <- 1           // File descriptor (stdout)
c <- addr        // Buffer address
d <- len         // Buffer length
syscall          // Invoke syscall
```

## Return Values

The **last expression** in an unsafe block is the return value:

```flap
result := unsafe {
    a <- 42
    b <- a * 2
    b            // Returns b (84)
}
```

### Type Casting Returns

Cast return values to C types:

```flap
ptr := unsafe {
    a <- 0x1000
    a as pointer     // Return as pointer
}

text := unsafe {
    a <- string_addr
    a as cstr        // Return as C string
}
```

## Examples

### Example 1: Simple Arithmetic
```flap
sum := unsafe {
    a <- 10
    b <- 32
    c <- a + b
    c
}
printf("Sum: %v\n", sum)  // Output: Sum: 42
```

### Example 2: Bitwise Magic
```flap
// Fast power-of-2 check
is_power_of_2 := unsafe {
    a <- 16
    b <- a - 1
    c <- a & b
    c            // Returns 0 if power of 2
}
```

### Example 3: Memory Manipulation
```flap
// Allocate buffer
buf_size := 1024
buffer := malloc(buf_size)

// Write to buffer
unsafe {
    a <- buffer
    [a] <- 0x4141414141414141 as u64      // Write "AAAAAAAA"
    [a + 8] <- 0x4242424242424242 as u64  // Write "BBBBBBBB"
}

// Read back
first := unsafe {
    a <- buffer
    b <- [a]
    b
}

printf("First 8 bytes: 0x%x\n", first)
free(buffer)
```

### Example 4: System Call (Per-CPU)
```flap
// Write "Hello\n" to stdout - platform-specific syscalls
msg := "Hello\n"

unsafe {
    x86_64 {
        rax <- 1             // sys_write
        rdi <- 1             // stdout
        rsi <- msg as cstr   // buffer
        rdx <- 6             // length
        syscall
    }
    arm64 {
        x8 <- 64             // sys_write on ARM64
        x0 <- 1              // stdout
        x1 <- msg as cstr    // buffer
        x2 <- 6              // length
        syscall
    }
    riscv64 {
        a7 <- 64             // sys_write on RISC-V
        a0 <- 1              // stdout
        a1 <- msg as cstr    // buffer
        a2 <- 6              // length
        syscall
    }
}
```

### Example 5: Unified Cross-Platform Code
```flap
// Same code works on x86_64, ARM64, and RISC-V!
factorial_5 := unsafe {
    a <- 5          // Input
    b <- 1          // Result

    // Loop would go here (simplified)
    c <- a * b
    d <- c * 4
    e <- d * 3
    f <- e * 2
    f              // Return 120
}

printf("5! = %v\n", factorial_5)
```

## Safety Considerations

1. **No Type Safety**: Unsafe blocks bypass Flap's type system
2. **No Bounds Checking**: Memory access is unchecked
3. **Platform Specific**: Code may behave differently across architectures
4. **Manual Stack Management**: Push/pop must be balanced
5. **Syscall Conventions Vary**: Different syscall numbers per OS/arch

Use unsafe blocks only when:
- Performance is critical
- Interfacing with hardware
- Implementing low-level primitives
- Syscalls are required

For most code, use Flap's safe high-level features instead.

## Advanced Topics

### Interfacing with C
```flap
// Call C function that expects pointer
c_func_ptr := unsafe {
    a <- data_buffer
    a as pointer
}
c_function(c_func_ptr)
```

### Custom Allocators
```flap
// Implement bump allocator
alloc := x => unsafe {
    a <- heap_ptr        // Current heap position
    b <- a + x           // New position
    heap_ptr <- b        // Update heap pointer
    a as pointer         // Return old position
}
```

### Atomic Operations
```flap
// LOCK prefix on x86_64
counter := unsafe {
    a <- counter_addr
    b <- [a]
    c <- b + 1
    [a] <- c    // Note: actual atomics need LOCK prefix
    c
}
```

## See Also

- [LANGUAGE.md](LANGUAGE.md) - Full Flap language reference
- [README.md](README.md) - Getting started guide
- [LEARNINGS.md](LEARNINGS.md) - Implementation insights
- [testprograms/unsafe_*.flap](testprograms/) - More examples
