# Flap Compiler Development Guide

**Version:** 3.0.0  
**Last Updated:** November 19, 2025

This document covers development insights, lessons learned, and implementation details for contributors.

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Hard-Earned Learnings](#hard-earned-learnings)
- [Anti-Patterns (Things NOT to Do)](#anti-patterns-things-not-to-do)
- [Memory Management](#memory-management)
- [Removed Features](#removed-features)
- [Testing](#testing)
- [Contributing](#contributing)

---

## Architecture Overview

### Compilation Pipeline

```
Source Code (.flap)
    ↓
Lexer (lexer.go)
    ↓
Parser (parser.go) → AST
    ↓
Code Generator (codegen.go)
    ↓
Platform-Specific Backend (x86_64_codegen.go, arm64_codegen.go, riscv64_codegen.go)
    ↓
Binary Format Writer (elf.go, macho.go)
    ↓
Executable Binary
```

### Direct Code Generation

Flap compiles directly from AST to machine code with **no intermediate representation**:
- No LLVM
- No IR layer
- Single-pass compilation
- Direct x86_64/ARM64/RISCV64 instruction emission

### Platform Support

| Platform | Status | Backend | Binary Format |
|----------|--------|---------|---------------|
| Linux x86_64 | ✅ Production | x86_64_codegen.go | ELF |
| Linux ARM64 | ✅ Production | arm64_codegen.go | ELF |
| Linux RISCV64 | ✅ Production | riscv64_codegen.go | ELF |
| macOS x86_64 | ✅ Production | x86_64_codegen.go | Mach-O |
| macOS ARM64 | ✅ Production | arm64_codegen.go | Mach-O |
| Windows | ❌ Planned 3.0 | - | PE/COFF |
| WASM | ❌ Planned 3.0 | - | WASM |

---

## Hard-Earned Learnings

### 1. Register Allocation for Syscalls

**Problem:** Syscalls clobber rcx and r11 on x86_64

**Solution:** Always use callee-saved registers (rbx, r12-r15) for loop counters

```go
// ❌ BAD: rcx gets clobbered by syscall
fc.out.XorRegWithReg("rcx", "rcx")
loop:
    // ... syscall here clobbers rcx ...
    fc.out.IncReg("rcx")
    // Loop never exits!

// ✅ GOOD: r14 is callee-saved
fc.out.XorRegWithReg("r14", "r14")
loop:
    // ... syscall here preserves r14 ...
    fc.out.IncReg("r14")
    // Loop works correctly
```

### 2. Lambda Epilogue Consistency

**Problem:** ret statement in lambda must match function epilogue exactly

**Lesson:** After `mov rsp, rbp`, use `SUB rsp, 8` not `ADD rsp, 8` to point to saved rbx

```go
// Lambda prologue:
// push rbp
// mov rbp, rsp
// ... allocate locals ...
// sub rsp, 8    // alignment
// push rbx      // save at rbp-8

// Epilogue must be symmetric:
mov rsp, rbp     // Point to saved rbp
sub rsp, 8       // ✅ Point to saved rbx
pop rbx
pop rbp
ret
```

### 3. String Literals Need Null Terminators

**Problem:** Format strings for printf must be null-terminated

**Solution:** Always append `\x00` to string literal definitions

```go
// ❌ BAD: Garbage after string
fc.eb.Define(labelName, "Hello, World!\n")

// ✅ GOOD: Properly terminated
fc.eb.Define(labelName, "Hello, World!\n\x00")
```

### 4. Lists Are Maps, Not Linked Lists

**Major Redesign (v2.0):** Lists changed from linked lists to maps

**Old (v1.x):**
```
List = [head][tail_ptr] → [head][tail_ptr] → ...
- O(n) indexing
- O(n) length
- Complex traversal
```

**New (v2.0):**
```
List = [count][0][val0][1][val1][2][val2]...
- O(1) indexing
- O(1) length
- Simple offset calculation
```

**Impact:** 400+ lines of code removed, better performance

### 5. Match Expression Jumps

**Lesson:** Match expressions must preserve xmm0 across all jump paths

**Current Bug:** String literals in match results lose pointer value (see FAILURES.md)

### 6. Parallel Loop Synchronization

**Lesson:** Use pthread barriers for proper thread synchronization

```go
// Create barrier before spawning threads
barrier := createBarrier(threadCount)

// Each thread waits at barrier
for tid := 0; tid < threadCount; tid++ {
    spawn { 
        // ... work ...
        barrier.wait()
    }
}

// Main thread waits
barrier.wait()
```

### 7. Offset Calculation Precision

**Lesson:** Off-by-one errors in offset calculations cause subtle bugs

For map format `[count][key0][val0][key1][val1]...`:
- Count at offset 0
- Key0 at offset 8
- Val0 at offset 16
- Val1 at offset 32 (not 24!)

Formula: `offset = 8 + index * 16 + 8` = `16 + index * 16`

---

## Anti-Patterns (Things NOT to Do)

### Language Anti-Patterns

#### 1. Using printf/sprintf Instead of Built-ins

```flap
// ❌ WRONG: Don't import C printf for string ops
use "libc.so.6"
printf("Result: %d\n", x)

// ✅ RIGHT: Use Flap's println
println(x)
```

#### 2. Treating Strings as Null-Terminated

```flap
// ❌ WRONG: Flap strings are maps, not C strings
str := "Hello"
// str is NOT a char* with '\0' at end

// ✅ RIGHT: Use string operations
length := #str
```

#### 3. Trying to Mutate Immutable Variables

```flap
// ❌ WRONG: Variables are immutable by default
x := 10
x = 20  // Compilation error

// ✅ RIGHT: Use mutable variables
x := 10
x <- 20  // Reassignment operator
```

#### 4. Using Traditional Loops Instead of Range Loops

```flap
// ❌ WRONG: Manual counter loops
i := 0
@ {
    i >= 10 -> ret @
    println(i)
    i <- i + 1
}

// ✅ RIGHT: Range loop
@ i in 0..<10 {
    println(i)
}
```

### Compiler Anti-Patterns

#### 1. Using Stack for Temporary Pointers

```go
// ❌ WRONG: String pointer gets lost
fc.out.SubImmFromReg("rsp", 8)
fc.out.MovXmmToMem("xmm0", "rsp", 0)
// ... other operations ...
// Lost track of where pointer is!

// ✅ RIGHT: Use register for pointer
fc.out.MovXmmToMem("xmm0", "rsp", 0)
fc.out.MovMemToReg("rbx", "rsp", 0)
// rbx now reliably holds pointer
```

#### 2. Forgetting to Track Function Calls

```go
// ❌ WRONG: Direct call without tracking
fc.eb.GenerateCallInstruction("printf")

// ✅ RIGHT: Track for PLT generation
fc.trackFunctionCall("printf")
fc.eb.GenerateCallInstruction("printf")
```

#### 3. Assuming Registers Survive Function Calls

```go
// ❌ WRONG: rax/rcx/rdx not preserved
fc.out.MovImmToReg("rax", "42")
fc.eb.GenerateCallInstruction("function")
// rax is now garbage!

// ✅ RIGHT: Use callee-saved or save/restore
fc.out.PushReg("rbx")
fc.out.MovImmToReg("rbx", "42")
fc.eb.GenerateCallInstruction("function")
// rbx still contains 42
fc.out.PopReg("rbx")
```

---

## Memory Management

### Arena Allocation

Flap uses arena (region-based) memory management:

```flap
arena {
    // All allocations in this block use arena
    items := [1, 2, 3, 4, 5]
    data := "Hello"
    // ... more allocations ...
}  // Arena freed here, all allocations released at once
```

**Benefits:**
- O(1) deallocation (free entire arena)
- No individual free calls
- Predictable memory usage
- Cache-friendly allocations

**Implementation:**
- Each arena has a memory pool
- Allocations bump pointer forward
- On arena exit, entire pool released

### Manual Memory Management

Outside arenas, Flap uses manual memory management:

```flap
// Allocate
ptr := malloc(1024)

// Use...

// Must manually free
free(ptr)
```

### Memory Layout

All Flap values use map representation:
```
[count_f64][key0_u64][val0_f64][key1_u64][val1_f64]...
```

- Numbers: count=1, single key-value
- Strings: count=N, keys are indices, values are char codes
- Lists: count=N, keys are sequential 0..N-1
- Maps: count=N, keys and values arbitrary

**Future (v3.0):** Add type byte prefix for runtime type checking

---

## Removed Features

### Removed Syntax (Historical)

These features were removed from Flap over time:

#### 1. Traditional Function Syntax

**Removed:** `func name(args) { body }`  
**Why:** Inconsistent with lambda-first design  
**Replaced with:** `name := (args) => body`

#### 2. Implicit Returns

**Removed:** Last expression as implicit return  
**Why:** Ambiguous in match blocks  
**Replaced with:** Explicit `ret` keyword

#### 3. Nullable Types

**Removed:** `?Type` syntax  
**Why:** Everything is map[uint64]float64, no null  
**Replaced with:** Use 0.0 or empty map as sentinel

#### 4. Class/Object Syntax

**Removed:** `class Name { ... }`  
**Why:** Flap is functional, not object-oriented  
**Replaced with:** Maps with function values

---

## Testing

### Test Organization

```
*_test.go files:
- basic_programs_test.go      Core language features
- string_map_test.go           String operations
- list_programs_test.go        List operations
- lambda_programs_test.go      Lambda expressions
- loop_programs_test.go        Loop constructs
- parallel_programs_test.go    Parallel execution
- arithmetic_comprehensive_test.go  All operators
- cstruct_programs_test.go     C FFI
- enet_test.go                 ENet syntax
- compiler_test.go             Error handling
+ many unit tests              Internals
```

### Running Tests

```bash
# All tests
go test

# Specific test
go test -run TestBasicPrograms

# Verbose
go test -v

# With coverage
go test -cover
```

### Test Results

**Current Status:** 128/128 tests passing (100%)

See TEST_REVIEW.md for complete test coverage analysis.

### Known Issues

See FAILURES.md for the one known edge case (match expressions with string literals).

---

## Contributing

### Code Style

- Follow Go conventions
- Comment complex assembly generation
- Keep functions under 100 lines where possible
- Test new features thoroughly

### Before Submitting PR

1. Run all tests: `go test`
2. Run linter: `go vet ./...`
3. Format code: `go fmt ./...`
4. Update documentation
5. Add test coverage for new features

### Architecture Decisions

- Keep direct code generation (no IR layer)
- Maintain zero-dependency compilation
- Support all three platforms (x86_64, ARM64, RISCV64)
- Follow Flap philosophy (everything is a map)

### Future Work

See TODO30.md for Flap 3.0 roadmap.

---

## Resources

- **Language Spec:** LANGUAGESPEC.md
- **User Guide:** README.md
- **Known Issues:** FAILURES.md
- **Release Notes:** RELEASE_NOTES_2.0.md
- **Roadmap:** TODO30.md

---

**Questions?** Open an issue on GitHub: https://github.com/xyproto/flapc
