# Flap Compiler Development Learnings

## Code Generation

- **Linux syscalls > C library**: Syscalls (open/read/close) are simpler than C library (fopen/fread/fclose) for direct code generation - fewer registers, no hidden state, no red zone issues
- **Stack alignment**: Stack must be 16-byte aligned before CALL instruction; odd number of pushes = aligned, even = misaligned
- **Callee-saved registers**: Must save/restore rbx, rbp, r12-r15 across function calls
- **Function arguments**: Must restore arguments to correct registers (rdi, rsi, rdx) before calling functions

## Language Design

- **Semicolons**: Must tokenize `;` and skip it like newlines to support multiple statements on one line
- **Type inference**: Update `getExprType()` when adding new builtins that return strings

## Debugging

- **strace**: Track syscalls to see file operations
- **gdb**: Get backtraces for crashes
- **ndisasm**: Disassemble generated code
- **TDD approach**: Test empty file → 1-byte → multi-byte incrementally

## Memory Layout

- **Flap strings**: `map[uint64]float64` with format `[count][key0][val0][key1][val1]...`
- **Allocation**: `malloc(8 + length * 16)` bytes (16 bytes per character)
- **C string conversion**: `cstr_to_flap_string` converts null-terminated C strings to Flap maps

## Package System

- **use statement**: Import external Flap files with `use "./path.flap"` or `use "packagename"`
- **TOKEN_USE**: Added lexer support for `use` keyword
- **UseStmt AST node**: Represents import statements in the AST
- **Future**: Load imported files, parse, and merge into main AST before compilation
