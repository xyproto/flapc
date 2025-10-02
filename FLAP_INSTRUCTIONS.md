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

## Future Instructions Needed

To fully implement Flap, additional instructions will be needed:

- **DIV/MOD** - Division and modulo operations
- **MOV variants** - Memory load/store (currently only register-to-register)
- **Floating point** - FADD, FSUB, FMUL, FDIV for float64 operations
- **Load/Store** - Memory access instructions for reading/writing data
- **Shift operations** - SHL, SHR, SAR for bit shifting
- **Negation** - NEG for arithmetic negation
