# Flap Compiler TODO

## Current Status

**Version**: 1.0.0 (In Progress)
**Platform**: x86-64 Linux
**Tests Passing**: 178/178 (100%) ✓

---

## Active Work Items (Sorted by Priority)

### 1. Builtin Functions (Standard Library)

**I/O Functions:**
- [ ] **Implement readln()**: Read line from stdin, return as Flap string (WIP - stack management issues)
- [ ] **Implement read_file(path)**: Read entire file, return as Flap string (WIP - fclose segfault)
- [x] **Implement write_file(path, content)**: Write string to file ✓ (Working!)

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

### 2. Polymorphic Operators (Type-Aware Behavior)

**String Operations:**
- [ ] **Implement string < and >**: Lexicographic comparison
- [ ] **Implement string slicing**: Use SliceExpr codegen for strings
- [ ] **Implement string subtraction**: Remove characters (set difference)

**List/Map Operations:**
- [ ] **Implement list + list**: Concatenate lists (runtime, not compile-time)
- [ ] **Implement map + map**: Merge maps
- [ ] **Implement list - list**: Set difference (remove elements)
- [ ] **Implement map - map**: Remove keys from first map

### 3. Control Flow Enhancements (Loop Return Values)

- [ ] **Implement LoopExpr**: Allow loops in expression context
  - Parse `x = @+ i in range(10) { i * 2 }` as LoopExpr
  - Use ret @N value to return from loop expression
  - Default return value is 0 (like match expressions)

- [ ] **Test ret @N value**: Verify loops can return values
  - `@+ i in range(10) { i == 5 { -> ret @1 42 } }` returns 42
  - `@+ i in range(10) { i * 2 }` at end returns last value

### 4. Error Reporting Improvements

- [ ] **Add line numbers to runtime errors**: Include source location in error messages
- [ ] **Improve type error messages**: Show expected vs actual types
- [ ] **Check function argument counts**: Report errors for wrong number of arguments
- [ ] **Add undefined variable detection**: Report which variable is undefined

### 5. Architecture Support (ARM64, RISC-V)

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

### 6. Performance Optimizations (Post-1.0.0)

- [ ] **Implement AVX-512 map lookup**: 8 keys/iteration (4x faster than SSE2)
- [ ] **Add perfect hashing**: For compile-time constant maps
- [ ] **Implement binary search**: For maps with 32+ sorted keys
- [ ] **Optimize CString conversion**: O(n²) → O(n)
- [ ] **Add constant folding**: Evaluate constant expressions at compile time

### 7. Advanced Features (2.0.0)

- [ ] **Multiple lambda dispatch**: `f = x => x * 2, x, y => x + y`
- [ ] **Pattern matching**: Destructuring in match expressions
- [ ] **Method call sugar**: `receiver.fn(args)` desugars to `fn(receiver, args)`
- [ ] **Regex matching**: `text =~ /pattern/`, `text !~ /pattern/`
- [ ] **Gather/scatter operations**: `data@[indices]`, `data@[indices] := values`
- [ ] **SIMD annotations**: `@simd(width=8) { }` for explicit vectorization
- [ ] **Precision annotations**: `@precision(128) { }` for arbitrary precision

---

## Test Status Summary

**Passing**: 178/178 tests (100%) ✓

**Known Failures**: None! All tests passing as of 2025-10-17

**Test Coverage**:
- ✓ All arithmetic operators (+, -, *, /, %, **, *+)
- ✓ All comparison operators (<, <=, >, >=, ==, !=)
- ✓ All logical operators (and, or, xor, not)
- ✓ All bitwise operators (&b, |b, ^b, ~b, <b, >b, <<b, >>b)
- ✓ Postfix operators (x++, x-- as statements only)
- ✓ FMA operator (*+ for fused multiply-add)
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
