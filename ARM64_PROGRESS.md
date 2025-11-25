# ARM64 Implementation Progress - 2025-11-25

## What Works

✅ Basic arithmetic: `main = 5 + 10` returns exit code 15
✅ Simple values: `main = 42` returns exit code 42  
✅ Variable assignment: `x = 7; main = x` returns 7
✅ Expression evaluation: All arithmetic operations work
✅ Proper _start function that calls user code and exits with return value
✅ ELF generation with correct ARM64 machine type (0xB7)
✅ Dynamic linking structure (PLT/GOT)
✅ Stack frame management
✅ Float<->int conversions for exit codes

## Critical Bugs

🔴 **println/eprint produce no output**
- Functions compile without errors
- Syscalls are emitted (write syscall #64 on Linux ARM64)
- But no text appears on stdout/stderr
- Likely issues:
  - PC-relative string address loading (ADRP/ADD pair)
  - String data not in correct rodata section
  - Syscall parameter passing

🔴 **Blocks don't return values** 
- `main = { 42 }` returns 0 instead of 42
- Affects BOTH x86-64 and ARM64
- Blocks compile and execute
- But return value not captured/propagated
- Needs architectural decision on block semantics

## Testing

Tested on ARM64 Linux (via SSH to port 2222):
- Go compiler works
- flapc builds successfully  
- Simple programs run correctly
- I/O operations fail silently

## Next Steps

1. Fix println/eprint - debug PC relocations and syscall
2. Decide on block semantics and implement consistently
3. Add more built-in functions (math, string ops)
4. Test lambda functions
5. Add C FFI support for external libraries
6. Test more complex programs

