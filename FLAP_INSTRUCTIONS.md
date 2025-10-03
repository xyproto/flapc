# Flap Language Instruction Set

This document describes the assembly instructions implemented for the Flap programming language compiler.

## Overview

The Flap language ("Float. Map. Fly.") is built on a `map[float64]float64` foundation with functional programming features, pattern matching, and elegant error handling. These low-level instructions form the building blocks for the high-level language constructs.

## Implemented Instructions

### 1. CMP - Comparison (`cmp.go`)

**Purpose**: Compare two values and set flags for conditional operations

**Why Essential for Flap**:
- Pattern matching guards: `n <= 1 -> 1`
- Filter expressions: `[x in rest]{x < pivot}`
- Loop conditions: `@ entity in entities{health > 0}`
- Error guards: `size > 0 or! "invalid size"`
- All comparison operators: `==`, `!=`, `>=`, `<=`, `>`, `<`

**Variants**:
- `CmpRegToReg(src1, src2)` - Compare two registers
- `CmpRegToImm(reg, imm)` - Compare register with immediate value

**Architecture Implementation**:
- **x86-64**: `CMP` instruction (sets RFLAGS)
- **ARM64**: `SUBS XZR, Xn, Xm` (subtract to zero register)
- **RISC-V**: `SUB t0, rs1, rs2` (subtract to temp register)

---

### 2. Conditional Jumps (`jmp.go`)

**Purpose**: Branch execution based on comparison results

**Why Essential for Flap**:
- Pattern matching dispatch: `n <= 1 -> 1; ~> n * me(n - 1)`
- Error handling flow: `x or! "error message"`
- Guard expressions: `x or return y`
- Loop filtering: `@ entity in entities{health > 0}`
- Default patterns: `~>` (catch-all case)

**Jump Conditions**:
- `JumpEqual` / `JumpNotEqual` (JE/JNE)
- `JumpGreater` / `JumpGreaterOrEqual` (JG/JGE)
- `JumpLess` / `JumpLessOrEqual` (JL/JLE)
- `JumpAbove` / `JumpAboveOrEqual` (JA/JAE) - unsigned
- `JumpBelow` / `JumpBelowOrEqual` (JB/JBE) - unsigned

**Variants**:
- `JumpConditional(condition, offset)` - Conditional branch
- `JumpUnconditional(offset)` - Unconditional jump

**Architecture Implementation**:
- **x86-64**: `Jcc` instructions (JE, JNE, JG, etc.)
- **ARM64**: `B.cond` (conditional branch)
- **RISC-V**: `BEQ`, `BNE`, `BLT`, `BGE` (branch instructions)

---

### 3. ADD - Addition (`add.go`)

**Purpose**: Arithmetic addition operations

**Why Essential for Flap**:
- Arithmetic expressions: `n + 1`, `x + y`
- Array/pointer arithmetic: `address + offset`
- Index calculations: `me.entities + [i]`
- Increment operations: `me.x := me.x + dx`
- List concatenation: `smaller + [pivot] + larger`

**Variants**:
- `AddRegToReg(dst, src)` - dst = dst + src (2-operand)
- `AddImmToReg(dst, imm)` - dst = dst + imm
- `AddRegToRegToReg(dst, src1, src2)` - dst = src1 + src2 (3-operand)

**Architecture Implementation**:
- **x86-64**: `ADD` instruction (2-operand)
- **ARM64**: `ADD Xd, Xn, Xm` (3-operand)
- **RISC-V**: `ADD rd, rs1, rs2` / `ADDI rd, rs1, imm`

---

### 4. SUB - Subtraction (`sub.go`)

**Purpose**: Arithmetic subtraction operations

**Why Essential for Flap**:
- Arithmetic expressions: `n - 1`
- Decrement operations: `me.health - amount`
- Pointer arithmetic: `end - start`
- Loop counters: `count - 1`
- Used internally by CMP

**Variants**:
- `SubRegFromReg(dst, src)` - dst = dst - src (2-operand)
- `SubImmFromReg(dst, imm)` - dst = dst - imm
- `SubRegFromRegToReg(dst, src1, src2)` - dst = src1 - src2 (3-operand)

**Architecture Implementation**:
- **x86-64**: `SUB` instruction (2-operand)
- **ARM64**: `SUB Xd, Xn, Xm` (3-operand)
- **RISC-V**: `SUB rd, rs1, rs2` / `ADDI rd, rs1, -imm`

---

### 5. MUL - Multiplication (`mul.go`)

**Purpose**: Arithmetic multiplication operations

**Why Essential for Flap**:
- Arithmetic expressions: `n * 2`
- Recursive multiplication: `n * me(n - 1)` in factorial
- Array size calculations: `rows * columns`
- Scaling operations: `value * scale_factor`
- Area/volume calculations

**Variants**:
- `MulRegWithReg(dst, src)` - dst = dst * src (2-operand)
- `MulRegWithImm(dst, imm)` - dst = dst * imm (x86-64 only)
- `MulRegWithRegToReg(dst, src1, src2)` - dst = src1 * src2 (3-operand)

**Architecture Implementation**:
- **x86-64**: `IMUL` instruction (signed multiply)
- **ARM64**: `MUL Xd, Xn, Xm` (actually MADD with XZR)
- **RISC-V**: `MUL rd, rs1, rs2` (requires M extension)

---

### 6. PUSH/POP - Stack Operations (`push.go`)

**Purpose**: Stack management for function calls and local variables

**Why Essential for Flap**:
- Function prologue/epilogue
- Preserving registers across calls
- Local variable storage
- Function parameter passing
- Recursive calls: `factorial =~ n { ... me(n - 1) }`

**Variants**:
- `PushReg(reg)` - Push register onto stack
- `PopReg(reg)` - Pop value from stack into register

**Architecture Implementation**:
- **x86-64**: `PUSH`/`POP` instructions (compact encoding)
- **ARM64**: `STR [SP, #-16]!` / `LDR [SP], #16` (pre/post-indexed)
- **RISC-V**: `ADDI sp, sp, -8; SD` / `LD; ADDI sp, sp, 8`

---

### 7. CALL - Function Calls (`call.go`)

**Purpose**: Function call mechanism

**Why Essential for Flap**:
- Direct function calls: `process_data(validated)`
- Recursive calls: `me(n - 1)` in factorial
- Method calls: `entity.update()`
- Library function calls: `create_user(user_data)`
- Lambda calls: `(x) -> x + 1`

**Variants**:
- `CallRelative(offset)` - Direct call to relative address
- `CallRegister(reg)` - Indirect call through register

**Architecture Implementation**:
- **x86-64**: `CALL rel32` / `CALL r/m64`
- **ARM64**: `BL` (Branch with Link) / `BLR` (register)
- **RISC-V**: `JAL ra, offset` / `JALR ra, reg, 0`

---

### 8. RET - Function Returns (`ret.go`)

**Purpose**: Return from functions

**Why Essential for Flap**:
- Normal returns: `return expression`
- Early returns: `x or return y`
- Guard returns: `me.running or return "game stopped"`
- Error returns: `or! "error message"`
- Implicit returns from pattern matching

**Variants**:
- `Ret()` - Simple return
- `RetImm(popBytes)` - Return with stack cleanup (x86-64 only)

**Architecture Implementation**:
- **x86-64**: `RET` (opcode 0xC3) / `RET imm16`
- **ARM64**: `RET` (actually `BR X30`, return via link register)
- **RISC-V**: `RET` (pseudo-instruction for `JALR x0, ra, 0`)

---

### 9. Logical Operations (`logic.go`)

**Purpose**: Bitwise logical operations

**Why Essential for Flap**:
- Boolean expressions: combining conditions
- Bitwise operations: `flags & mask`, `flags | bit`
- Pattern matching: compound conditions
- Set operations: intersection (AND), union (OR), symmetric difference (XOR)
- Flag manipulation: `entity.flags & VISIBLE`
- Bit testing: `value & (1 << n)`

**Variants**:
- `AndRegWithReg(dst, src)` - dst = dst & src
- `AndRegWithImm(dst, imm)` - dst = dst & imm
- `AndRegWithRegToReg(dst, src1, src2)` - dst = src1 & src2
- Similar variants for OR and XOR

**Architecture Implementation**:
- **x86-64**: `AND`/`OR`/`XOR` instructions with optimized immediate encodings
- **ARM64**: `AND`/`ORR`/`EOR` with shifted register or bitmask immediate
- **RISC-V**: `AND`/`OR`/`XOR` register, `ANDI`/`ORI`/`XORI` immediate

---

### 10. DIV - Division and Remainder (`div.go`)

**Purpose**: Division and modulo operations

**Why Essential for Flap**:
- Division expressions: `quotient / divisor`
- Modulo operations: `n % 10`
- Array partitioning: `size / chunk_size`
- Ratio calculations: `total / count`
- Remainder checks: `n % 2 == 0` (even/odd testing)

**Variants**:
- `DivRegByReg(dst, src)` - dst = dst / src (2-operand)
- `DivRegByRegToReg(quotient, dividend, divisor)` - quotient = dividend / divisor
- `RemRegByReg(dst, src)` - dst = dst % src (remainder/modulo)
- `RemRegByRegToReg(remainder, dividend, divisor)` - remainder = dividend % divisor

**Architecture Implementation**:
- **x86-64**: `IDIV` (signed divide, implicit RDX:RAX usage, produces both quotient and remainder)
- **ARM64**: `SDIV` (signed divide), remainder via `MSUB` (multiply-subtract)
- **RISC-V**: `DIV` and `REM` instructions (requires M extension)

**Notes**:
- x86-64 division is complex: requires sign-extension (CQO), modifies RAX (quotient) and RDX (remainder)
- ARM64 doesn't have direct remainder instruction; calculated as `remainder = dividend - (quotient * divisor)`
- RISC-V provides separate DIV and REM instructions

---

### 11. Load/Store - Memory Access (`loadstore.go`)

**Purpose**: Reading from and writing to memory

**Why Essential for Flap**:
- Variable access: `me.health`, `me.x`
- Array element access: `entities[i]`
- Map value access: `map[key]`
- Struct field access: `player.position.x`
- Stack variable access: local variables
- Global variable access: `game_state`

**Variants**:
- `LoadRegFromMem(dst, base, offset)` - dst = [base + offset]
- `StoreRegToMem(src, base, offset)` - [base + offset] = src

**Architecture Implementation**:
- **x86-64**: `MOV r64, [r64 + disp]` / `MOV [r64 + disp], r64` with 8-bit or 32-bit displacement
- **ARM64**: `LDR Xt, [Xn, #offset]` / `STR Xt, [Xn, #offset]`, or `LDUR`/`STUR` for unscaled offsets
- **RISC-V**: `LD rd, offset(rs1)` / `SD rs2, offset(rs1)` with 12-bit signed offset

**Offset Ranges**:
- **x86-64**: -2,147,483,648 to 2,147,483,647 (32-bit signed)
- **ARM64**: 0 to 32,760 (scaled, 8-byte aligned) or -256 to 255 (unscaled)
- **RISC-V**: -2,048 to 2,047 (12-bit signed)

---

### 12. NEG - Arithmetic Negation (`neg.go`)

**Purpose**: Two's complement negation (arithmetic negative)

**Why Essential for Flap**:
- Unary minus: `-x`, `-value`
- Direction reversal: `-velocity`
- Sign flipping: `-balance`
- Opposite values: `-delta`
- Negating results: `-(a + b)`

**Variants**:
- `NegReg(dst)` - dst = -dst (2-operand)
- `NegRegToReg(dst, src)` - dst = -src (3-operand)

**Architecture Implementation**:
- **x86-64**: `NEG r/m64` (opcode 0xF7 /3)
- **ARM64**: `NEG Xd, Xn` (actually `SUB Xd, XZR, Xn` - subtract from zero)
- **RISC-V**: `NEG rd, rs` (pseudo-instruction for `SUB rd, x0, rs` - subtract from zero)

**Notes**:
- ARM64 and RISC-V implement NEG as subtraction from the zero register
- This is semantically equivalent to two's complement negation
- More efficient than loading 0 and subtracting

---

## Architecture Support

All instructions are implemented for three target architectures:

### x86-64 (Intel/AMD)
- 64-bit operations with REX prefix
- 2-operand format (destination is also source)
- Rich instruction set with immediate optimizations

### ARM64 (AArch64)
- 3-operand format (separate destination)
- Fixed 4-byte instruction encoding
- 12-bit immediate values for ADD/SUB

### RISC-V 64-bit
- 3-operand format
- Fixed 4-byte instruction encoding
- 12-bit signed immediates for ADDI
- No direct CMP (uses SUB with result check)

---

## Example: Flap Factorial Pattern Matching

```
factorial =~ n {
    n <= 1 -> 1
    ~> n * me(n - 1)
}
```

**Assembly Flow** (conceptual):
1. `CMP n, 1` - Compare n with 1
2. `JLE case1` - Jump if n <= 1
3. `SUB temp, n, 1` - Calculate n - 1
4. `CALL factorial` - Recursive call me(n - 1)
5. `MUL result, n, result` - Multiply n * result
6. `JMP end`
7. `case1: MOV result, 1` - Return 1
8. `end: RET`

---

## Testing

Each instruction has comprehensive tests for all architectures:
- `cmp_test.go` - CMP instruction tests
- `arithmetic_test.go` - ADD, SUB, and Jump tests
- All tests verify correct opcode encoding and operand handling

Run tests with: `go test -v -run "TestCmp|TestAdd|TestSub|TestJump"`

---

## Instruction Summary

The current instruction set provides a complete foundation for implementing basic Flap programs:

**Arithmetic**: ADD, SUB, MUL, DIV (with remainder/modulo)
**Logical**: AND, OR, XOR, NEG
**Comparison**: CMP
**Control Flow**: JE, JNE, JG, JGE, JL, JLE, JA, JAE, JB, JBE, JMP
**Functions**: CALL, RET, PUSH, POP
**Memory**: Load/Store with base+offset addressing
**Data Movement**: MOV (register-to-register), LEA (load effective address)

## Future Instructions for Complete Flap Support

To fully implement all Flap language features, additional instructions will be needed:

- **Modern instructions** - SSE4.2 and/or AVX instructions, which should fit well with the Flap architecture
- **Floating point** - FADD, FSUB, FMUL, FDIV or MMX instructions for float64 operations (core Flap type)
- **Shift operations** - SHL, SHR, SAR for bit shifting and power-of-2 operations
- **Conditional moves** - CMOV variants for branchless code generation
- **Atomic operations** - For concurrent Flap programs (LOCK prefix on x86-64, LDXR/STXR on ARM64, LR/SC on RISC-V)
- **Advanced addressing** - Indexed/scaled addressing for efficient array access
- **Byte/word operations** - For working with smaller data types
