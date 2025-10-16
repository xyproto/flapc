# Flap Compiler Development Learnings

## Key Technical Insights

### 1. C Library vs Linux Syscalls for Direct Code Generation

**Problem**: C library functions (fopen/fread/fclose/getline) caused persistent segfaults due to complex stack management requirements.

**Solution**: Use Linux syscalls directly (open/lseek/read/close).

**Learnings**:
- **Syscalls are simpler**: Fewer registers, no hidden state, no red zone issues
- **Stack alignment with syscalls**: Just ensure rsp is correct before syscall instruction
- **Stack alignment with C library**: Must be 16-byte aligned at CALL, plus handle callee-saved registers
- **Red zone**: 128 bytes below rsp can be clobbered by signal handlers in C code
- **write_file() works**: Uses C library but simpler sequence (fopen→fwrite→fclose)
- **read_file() failed with C library**: Complex sequence (fopen→fseek→ftell→fseek→fread→fclose)

**Syscall Advantages**:
```
syscall open:  3 registers (rdi=path, rsi=flags, rdx=mode)
C fopen:       2 registers but requires 16-byte stack alignment + callee-saved regs
```

### 2. x86-64 Stack Alignment Rules

**Critical Rule**: Stack pointer must be 16-byte aligned immediately before CALL instruction.

**Stack Alignment Calculation**:
```
Entry:              rsp % 16 = 8  (return address from caller)
After push rbp:     rsp % 16 = 0  ✓ Aligned
After push r12:     rsp % 16 = 8
After push r13:     rsp % 16 = 0  ✓ Aligned
After push r14:     rsp % 16 = 8
After push r15:     rsp % 16 = 0  ✓ Aligned
```

**Rule**: Odd number of pushes → aligned. Even number → misaligned.

**Bug Found**: Added `sub rsp, 8` after 4 pushes + rbp = 5 total (40 bytes, aligned). The subtraction caused misalignment!

**Fix**: Removed the extra `sub rsp, 8`. Stack was already aligned after 5 pushes.

### 3. Test-Driven Development (TDD) for System Programming

**Approach**: Start with simplest possible test, increment complexity.

**TDD Steps Used**:
1. **Step 1**: Read empty file (0 bytes) → Passes immediately
2. **Step 2**: Read 1-byte file → Revealed cstr_to_flap_string bug
3. **Step 3**: Read multi-byte file → Same bug persists

**Benefits**:
- **Isolated failures**: Empty file works → bug is in string conversion, not file reading
- **Fast feedback**: Each test takes seconds to write and run
- **Confidence**: Step 1 passing proves syscall approach works
- **Incremental progress**: Can commit working code even with remaining bugs

**Contrast with Previous Approach**:
- Before TDD: Tried to read full file, got segfault, spent time debugging C library stack
- With TDD: Isolated the bug to cstr_to_flap_string within minutes

### 4. Callee-Saved Registers (x86-64 System V ABI)

**Must be preserved across function calls**: rbx, rbp, r12, r13, r14, r15

**Common Bug**: Using r12 without saving it first.

**Pattern for Functions**:
```assembly
; Prologue
push rbp
mov rbp, rsp
push rbx
push r12
push r13
push r14
; ... function body ...
; Epilogue
pop r14
pop r13
pop r12
pop rbx
pop rbp
ret
```

**Wrong**: Moving value into r12 before saving it → corrupts caller's data.

### 5. Direct Machine Code Generation Challenges

**Challenge**: No assembler to catch errors, bugs are runtime segfaults.

**Debugging Techniques Used**:
1. **strace**: Track syscalls (open/read/close)
2. **gdb**: Get backtrace, see where crash occurs
3. **ndisasm**: Disassemble generated code
4. **Incremental testing**: Test each piece separately

**Example Debug Session**:
```bash
# Check if file operations work
strace ./test 2>&1 | grep open
# openat(AT_FDCWD, "/tmp/file.txt", O_RDONLY) = 3  ✓

# Find crash location
gdb -batch -ex run -ex bt ./test
# #0 fclose() ← Crash in C library, not our code

# Conclusion: Stack corruption before fclose
```

### 6. Type System Integration

**Bug**: `read_file()` returned string, but compiler treated it as number.

**Problem**: `println()` checks `getExprType()` to decide how to print:
- **String type**: Print as characters
- **Number type**: Print as float

**Fix**: Add `read_file` and `readln` to string-returning functions:
```go
case *CallExpr:
    if e.Function == "str" || e.Function == "read_file" || e.Function == "readln" {
        return "string"
    }
    return "number"
```

**Learning**: Type inference must be updated when adding new builtins.

### 7. Memory Layout for Flap Strings

**Format**: `map[uint64]float64` represented as:
```
[count: float64][key0: float64][val0: float64][key1: float64][val1: float64]...
```

**For string "AB"**:
```
Offset 0:  count = 2.0
Offset 8:  key = 0.0,   value = 65.0  (A)
Offset 24: key = 1.0,   value = 66.0  (B)
```

**Allocation**: `malloc(8 + length * 16)` bytes.

**Critical Detail**: Each map entry is 16 bytes (key + value), not 8.

### 8. String Conversion C↔Flap

**C String**: Null-terminated byte array `char*`

**Flap String**: Map with character codes as float64 values

**Conversion Algorithm**:
```go
// C → Flap
1. strlen(cstr) → length
2. malloc(8 + length*16) → map
3. map[0] = length (as float64)
4. for i in 0..length:
     map[8+i*16+0] = i (key as float64)
     map[8+i*16+8] = cstr[i] (value as float64)
5. return map pointer (as float64)
```

**Bug Found**: Memory addressing when using r13 as base register requires SIB byte.

**Fix**: Calculate full address in rax: `rax = r13 + offset`, then use `[rax]`.

### 9. Linux Syscall Numbers (x86-64)

**Common Syscalls**:
- 0: `sys_read(fd, buf, count)`
- 1: `sys_write(fd, buf, count)`
- 2: `sys_open(path, flags, mode)`
- 3: `sys_close(fd)`
- 8: `sys_lseek(fd, offset, whence)`

**Constants**:
- O_RDONLY = 0
- O_WRONLY = 1
- O_RDWR = 2
- SEEK_SET = 0
- SEEK_CUR = 1
- SEEK_END = 2

**Syscall Instruction**: `0x0f 0x05`

**Registers**: rax=syscall_number, rdi, rsi, rdx, r10, r8, r9 (note: r10 not rcx!)

### 10. Conditional Library Linking

**Optimization**: Only link libraries when functions are actually used.

**Implementation**:
```go
// Always need libc.so.6 for basic functionality
ds.AddNeeded("libc.so.6")

// Only add libm.so.6 if math functions called via FFI
libmFunctions := map[string]bool{"sqrt": true, "sin": true, ...}
for funcName := range usedFunctions {
    if libmFunctions[funcName] {
        ds.AddNeeded("libm.so.6")
        break
    }
}
```

**Note**: Builtin math functions (sqrt, sin, cos) use hardware instructions, not libm.

**glibc 2.34+**: dlopen/dlsym/dlclose are in libc.so.6, no need for libdl.so.2.

## Development Principles That Worked

### 1. Go-Style Simplicity Over C-Style Complexity

**Example**: Postfix operators (x++, x--) are statements only, not expressions.
- **C**: `y = x++` (returns old value), `y = ++x` (returns new value) → cognitive overhead
- **Go/Flap**: `x++` is a statement, cannot be used in expressions → simple and clear

### 2. Railway-Oriented Error Handling

**Pattern**: `result or! "error message"`
- Continue on success
- Exit/print on failure
- No nested if-else

### 3. Incremental Commits with Clear Documentation

**Pattern**: Each commit documents what works, what doesn't, and why.

**Example**:
```
✅ write_file() working
❌ read_file() segfaulting in fclose
🔍 Next: Try syscall approach
```

### 4. Use the Right Tool for the Job

- **Syscalls**: For simple operations (open/read/close)
- **C library**: For complex operations (formatted I/O, string parsing)
- **Direct hardware**: For performance-critical math (SSE2, AVX2, FMA)

## Common Pitfalls Avoided

1. **Assuming stack is aligned**: Always calculate actual alignment
2. **Using r12-r15 without saving**: Save callee-saved registers first
3. **Complex C call sequences**: Use syscalls when possible
4. **Forgetting type system integration**: Update getExprType for new functions
5. **Batch testing without isolation**: TDD catches bugs earlier

## Performance Notes

### SIMD String Operations (Future Optimization)

Current string lookup: O(n) linear scan through map

Potential optimizations:
- **SSE2**: Compare 2 keys per iteration (2x faster)
- **AVX2**: Compare 4 keys per iteration (4x faster)
- **AVX-512**: Compare 8 keys per iteration (8x faster)

Note: Current implementation prioritizes correctness over performance.

## Test Coverage Status

**Passing**: 178/178 tests (100%) ✓

**New Functionality**:
- ✅ write_file(path, content) - Fully working
- 🚧 read_file(path) - Reads successfully, printing has bug
- 🚧 readln() - Not yet implemented
- ✅ dlopen/dlsym/dlclose - Working
- ✅ sizeof_TYPE builtins - Working
- ✅ Postfix operators (x++, x--) - Working as statements

## Next Steps

1. **Fix cstr_to_flap_string loop bug**: Debug why non-empty strings crash when printed
2. **Implement readln()**: Use syscall read(0, buf, size) with line buffering
3. **String builtin functions**: num(), split(), join(), upper(), lower(), trim()
4. **Collection functions**: map(), filter(), reduce(), keys(), values(), sort()

## Summary

The syscall approach proved that **simplicity beats complexity** in direct code generation. What took hours to debug with C library functions worked immediately with syscalls. TDD provided fast feedback and confidence in incremental progress. Stack alignment is critical and must be calculated carefully, not assumed.
