# Flap Compiler TODO

## Current Status

**Version**: 1.0.0 (In Progress)
**Platform**: x86-64 Linux
**Tests Passing**: 178/178 (100%)

---

## Active Work Items (Sorted by Priority)

### 1. Core Compiler Fixes (Highest Priority)

- [x] **Fix test_ffi_malloc crash**: Fixed by using r14 (callee-saved) instead of r11 (2025-10-16) ✓
- [x] **Fix test_ffi_from_c**: Added libm.so.6 for FFI math function calls (2025-10-16) ✓
- [ ] **Remove debug output**: Clean up DEBUG print statements in parser.go loop code
- [ ] **Add .gitignore for executables**: Prevent compiled test binaries from being committed

### 2. Parser Completeness (Complete for 1.0.0 Spec)

- [x] **Implement TOKEN_AS parsing**: Added CastExpr AST node (2025-10-16) ✓
- [x] **Fix JumpExpr bug**: Added IsBreak field to distinguish ret @N vs @N (2025-10-16) ✓
- [x] **Slice syntax parsing**: SliceExpr for [start:end:step] ✓
- [x] **Increment/decrement parsing**: TOKEN_INCREMENT, TOKEN_DECREMENT ✓
- [x] **FMA parsing**: TOKEN_FMA for `*+` operator ✓
- [x] **Error handling parsing**: TOKEN_OR_BANG for `or!` ✓
- [x] **Type annotations parsing**: TOKEN_COLON in assignment ✓

### 3. Code Generation for Parsed Features (Unblock Tests)

- [x] **Implement CastExpr codegen**: Generate code for `x as i32`, `ptr as ptr`, etc. (2025-10-16) ✓
  - Support all cast types: i8-i64, u8-u64, f32-f64, cstr, ptr, number
  - Integer casts: truncate float64 to integer
  - cstr conversion: flap_string_to_cstr runtime function
  - Tested with test_ffi_from_c.flap and test_ffi_malloc.flap
  - Note: 'as string' and 'as list' (C→Flap) not yet needed

- [ ] **Implement SliceExpr codegen**: Generate code for `list[start:end:step]`
  - Support all slice variants: [start:], [:end], [::step], [start:end:step]
  - Return new list/string with sliced elements

- [ ] **Implement PostfixExpr codegen**: Generate code for `x++`, `x--`
  - Number: increment/decrement by 1.0
  - List: append/remove last element (polymorphic)
  - Map: add/remove last entry (polymorphic)

- [ ] **Implement FMA codegen**: Generate VFMADD instruction for `a *+ b`
  - Use AVX2 VFMADD213SD for scalar operations
  - Better precision than separate multiply + add

- [ ] **Implement or! codegen**: Generate conditional exit for `condition or! "msg"`
  - If condition is false/zero, print message to stderr and exit(1)
  - If condition is true/non-zero, continue execution

### 4. FFI Implementation (Enable Foreign Function Interface)

- [x] **Implement call() builtin**: Call C functions with arguments ✓
  - Parse function name as string literal
  - Convert Flap values to C types using cast expressions
  - Handle function pointers from dlsym()
  - Return C values converted back to Flap floats

- [ ] **Implement dlopen/dlsym/dlclose**: Dynamic library loading
  - dlopen(path, flags) returns handle as float64
  - dlsym(handle, symbol) returns function pointer as float64
  - dlclose(handle) closes library

- [x] **Implement read_TYPE builtins**: Read from memory pointers ✓
  - read_i8, read_i16, read_i32, read_i64
  - read_u8, read_u16, read_u32, read_u64
  - read_f32, read_f64
  - Safe indexing: ptr[index] = ptr + (index * sizeof(TYPE))

- [x] **Implement write_TYPE builtins**: Write to memory pointers ✓
  - write_i8, write_i16, write_i32, write_i64
  - write_u8, write_u16, write_u32, write_u64
  - write_f32, write_f64
  - Safe indexing: ptr[index] = ptr + (index * sizeof(TYPE))

- [ ] **Implement sizeof_TYPE builtins**: Return type sizes
  - sizeof_i8() through sizeof_f64()
  - Return size in bytes as float64

- [x] **Fix flap_string_to_cstr bug**: Fixed register usage and alignment (2025-10-16) ✓
  - Used r14 (callee-saved) instead of r11 to avoid malloc clobbering
  - Added 8-byte alignment for string literals in rodata
  - Fixed instruction encoding for r13 memory access

### 5. Builtin Functions (Standard Library)

**I/O Functions:**
- [ ] **Implement readln()**: Read line from stdin, return as Flap string
- [ ] **Implement read_file(path)**: Read entire file, return as Flap string
- [ ] **Implement write_file(path, content)**: Write string to file

**String Functions:**
- [ ] **Implement num(string)**: Parse string to float64 ("42" → 42.0)
- [ ] **Implement split(string, delimiter)**: Split string into list of strings
- [ ] **Implement join(list, delimiter)**: Join list of strings with delimiter
- [ ] **Implement upper(string)**: Convert to uppercase
- [ ] **Implement lower(string)**: Convert to lowercase
- [ ] **Implement trim(string)**: Remove leading/trailing whitespace

**Collection Functions:**
- [ ] **Implement map(f, list)**: Apply function to each element
- [ ] **Implement filter(f, list)**: Keep elements where f(x) is true
- [ ] **Implement reduce(f, list, init)**: Fold list with binary function
- [ ] **Implement keys(map)**: Return list of map keys
- [ ] **Implement values(map)**: Return list of map values
- [ ] **Implement sort(list)**: Sort list in ascending order

**Vector Math Functions:**
- [ ] **Implement dot(v1, v2)**: Dot product of two vectors
- [ ] **Implement cross(v1, v2)**: Cross product of two 3D vectors
- [ ] **Implement magnitude(v)**: Length of vector
- [ ] **Implement normalize(v)**: Unit vector in same direction

### 6. Polymorphic Operators (Type-Aware Behavior)

**String Operations:**
- [ ] **Implement string < and >**: Lexicographic comparison
- [ ] **Implement string slicing**: Use SliceExpr codegen for strings
- [ ] **Implement string subtraction**: Remove characters (set difference)

**List/Map Operations:**
- [ ] **Implement list + list**: Concatenate lists (runtime, not compile-time)
- [ ] **Implement map + map**: Merge maps
- [ ] **Implement list - list**: Set difference (remove elements)
- [ ] **Implement map - map**: Remove keys from first map

### 7. Control Flow Enhancements (Loop Return Values)

- [ ] **Implement LoopExpr**: Allow loops in expression context
  - Parse `x = @+ i in range(10) { i * 2 }` as LoopExpr
  - Use ret @N value to return from loop expression
  - Default return value is 0 (like match expressions)

- [ ] **Test ret @N value**: Verify loops can return values
  - `@+ i in range(10) { i == 5 { -> ret @1 42 } }` returns 42
  - `@+ i in range(10) { i * 2 }` at end returns last value

### 8. Error Reporting Improvements

- [ ] **Add line numbers to runtime errors**: Include source location in error messages
- [ ] **Improve type error messages**: Show expected vs actual types
- [ ] **Check function argument counts**: Report errors for wrong number of arguments
- [ ] **Add undefined variable detection**: Report which variable is undefined

### 9. Architecture Support (ARM64, RISC-V)

**Phase 1: Architecture Abstraction**
- [ ] **Create CodeGenerator interface**: Abstract instruction emission
- [ ] **Extract X86_64CodeGen**: Refactor existing x86-64 code into backend
- [ ] **Implement ARM64CodeGen**: Use arm64_instructions.go encoders
- [ ] **Add architecture selection**: --arch flag to choose backend

**Phase 2: ARM64 Support**
- [ ] **Implement ARM64 register allocation**: x0-x30 (GP), v0-v31 (NEON)
- [ ] **Implement ARM64 calling convention**: AAPCS64 (x0-x7, v0-v7)
- [ ] **Implement ARM64 instruction selection**: FADD, FSUB, FMUL, FDIV, etc.
- [ ] **Test on macOS arm64**: Verify Mach-O executables run on Apple Silicon

**Phase 3: RISC-V Support**
- [ ] **Implement RISC-V register allocation**: x0-x31, f0-f31
- [ ] **Implement RISC-V calling convention**: a0-a7, fa0-fa7
- [ ] **Implement RISC-V instruction selection**: FADD.D, FSUB.D, etc.
- [ ] **Test on RISC-V hardware/emulator**: Verify ELF executables run

### 10. Performance Optimizations (Post-1.0.0)

- [ ] **Implement AVX-512 map lookup**: 8 keys/iteration (4x faster than SSE2)
- [ ] **Add perfect hashing**: For compile-time constant maps
- [ ] **Implement binary search**: For maps with 32+ sorted keys
- [ ] **Optimize CString conversion**: O(n²) → O(n)
- [ ] **Add constant folding**: Evaluate constant expressions at compile time

### 11. Advanced Features (2.0.0)

- [ ] **Multiple lambda dispatch**: `f = x => x * 2, x, y => x + y`
- [ ] **Pattern matching**: Destructuring in match expressions
- [ ] **Method call sugar**: `receiver.fn(args)` desugars to `fn(receiver, args)`
- [ ] **Regex matching**: `text =~ /pattern/`, `text !~ /pattern/`
- [ ] **Gather/scatter operations**: `data@[indices]`, `data@[indices] := values`
- [ ] **SIMD annotations**: `@simd(width=8) { }` for explicit vectorization
- [ ] **Precision annotations**: `@precision(128) { }` for arbitrary precision

---

## Recently Completed (2025-10-16)

### FFI String Conversion Bug Fix (2025-10-16)
- [x] **Fixed flap_string_to_cstr crashes**: Used r14 instead of r11 to avoid malloc clobbering
- [x] **Added string literal alignment**: 8-byte alignment for proper float64 access
- [x] **Fixed r13 instruction encoding**: Corrected memory access instruction generation
- [x] **Added libm.so.6 linking**: Required for FFI calls to math functions (sqrt, etc.)
- [x] **All tests passing**: 178/178 tests now pass (100%)

### FFI Memory Access Implementation
- [x] **Implemented write_TYPE builtins**: All variants (i8-i64, u8-u64, f32, f64)
- [x] **Implemented read_TYPE builtins**: All variants with proper sign/zero extension
- [x] **Created movq.go**: Extracted MOVQ instruction generation
- [x] **Made type keywords contextual**: Can use ptr, i32, string, etc. as variable names
- [x] **Updated LANGUAGE.md**: Documented contextual keywords with examples

### Test Framework Improvements
- [x] **Added wildcard pattern matching**: Support * for variable values (e.g., malloc pointers)
- [x] **Fixed .gitignore**: Allow .result files while ignoring test executables
- [x] **test_ffi_malloc passing**: 171/173 tests now pass (98.8%)

### Jump Control Flow Bug Fix
- [x] **Added IsBreak field to JumpExpr**: Distinguish ret @N (break) from @N (continue)
- [x] **Fixed nested_break_test**: @N now correctly continues loop instead of exiting
- [x] **Updated compileMatchJump**: Handle both break and continue cases

### Cast Operator Parsing
- [x] **Implemented TOKEN_AS parsing**: Added CastExpr AST node
- [x] **Added parsePostfix support**: Parse `expr as type` in postfix position
- [x] **Support all cast types**: i8-i64, u8-u64, f32-f64, cstr, ptr, number, string, list

### Test Framework Improvements
- [x] **Ignore trailing whitespace**: Updated splitLines() to trim trailing spaces/tabs
- [x] **Fixed test_ffi_from_c.flap**: Removed invalid statement blocks in match expressions

### Loop Special Variables
- [x] **Implemented @first, @last, @counter, @i**: All loop state variables working

### Core Compiler
- [x] **Two-pass compilation**: Symbols collected in first pass, code generated in second
- [x] **Forward references**: Functions can be called before definition

---

## Test Status Summary

**Passing**: 178/178 tests (100%) ✓

**Known Failures**: None! All tests passing as of 2025-10-16

**Test Coverage**:
- ✓ All arithmetic operators (+, -, *, /, %, **)
- ✓ All comparison operators (<, <=, >, >=, ==, !=)
- ✓ All logical operators (and, or, xor, not)
- ✓ All bitwise operators (&b, |b, ^b, ~b, <b, >b, <<b, >>b)
- ✓ String operations (concatenation, length, indexing, comparison)
- ✓ List operations (indexing, length, concatenation)
- ✓ Map operations (indexing, length, SIMD lookup)
- ✓ Loop control flow (break, continue, nested loops)
- ✓ Loop special variables (@first, @last, @counter, @i)
- ✓ Match expressions (with guards, defaults, nested)
- ✓ Lambda expressions (single/multiple params, closures)
- ✓ Math functions (sqrt, sin, cos, tan, log, exp, abs, floor, ceil, round)
- ✓ FFI (call(), read_TYPE, write_TYPE, string-to-C conversion)
- ✓ Type casting (i8-i64, u8-u64, f32-f64, cstr, ptr, number)

---

## Notes

- **Philosophy**: Keep x86-64/Linux fully working before adding ARM64/RISC-V
- **Next Target**: macOS arm64 (Apple Silicon) after x86-64 feature-complete
- **Code Quality**: Use ndisasm, gdb, and comparison with tcc for debugging
- **Problem Solving**: Apply Polya's "How to Solve It" techniques for hard problems
