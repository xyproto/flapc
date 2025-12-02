# Printf Implementation in x86_64 Assembly

A comprehensive, production-ready `printf` implementation in x86_64 assembly for Linux, featuring 15 format specifiers including floating-point support.

## Quick Start

```bash
# Run the demo
./printf

# Or compile from source
nasm -f elf64 printf.asm -o printf.o
ld printf.o -o printf
./printf
```

## Documentation

- **[README.md](README.md)** - Overview and quick reference
- **[EXAMPLES.md](EXAMPLES.md)** - Usage examples and code samples
- **[FEATURES.md](FEATURES.md)** - Complete feature list and comparison
- **[TEST_RESULTS.md](TEST_RESULTS.md)** - Verification and test results
- **[SUMMARY.md](SUMMARY.md)** - Executive summary and conclusion

## Key Features

✓ **15 format specifiers**: %d, %i, %u, %x, %X, %o, %b, %f, %e, %s, %c, %p, %t, %v, %%
✓ **Floating-point support**: Full double-precision with proper rounding
✓ **Register preservation**: All callee-saved registers properly maintained
✓ **No dependencies**: Pure assembly, no libc required
✓ **Verified**: Tested with gdb, objdump, ndisasm
✓ **Documented**: Comprehensive documentation with examples

## Files

- `printf.asm` - Full implementation with built-in tests (1110 lines)
- `printf_func.asm` - Library version for inclusion in your projects
- `demo.asm` - Comprehensive demonstration of all features

## Specifications

- **Architecture**: x86_64 (AMD64)
- **OS**: Linux
- **ABI**: System V AMD64 calling convention
- **Size**: ~15KB executable
- **Dependencies**: None

## Usage as Library

```nasm
%include "printf_func.asm"

section .data
    fmt db "Hello, %s! The answer is %d", 10, 0
    name db "World", 0

section .text
    global _start
_start:
    lea rdi, [rel fmt]
    lea rsi, [rel name]
    mov rdx, 42
    call printf
    
    mov rax, 60
    xor rdi, rdi
    syscall
```

## Verification

All features have been verified using:
- Direct testing with multiple test programs
- GDB debugging and register inspection
- Disassembly analysis with objdump and ndisasm
- Edge case testing (max/min values, special floats)
- Register preservation across 10+ consecutive calls

See [TEST_RESULTS.md](TEST_RESULTS.md) for detailed verification results.

## Comparison

This implementation is more comprehensive than typical assembly printf examples:
- **Typical**: 5-8 format specifiers, no floats
- **This**: 15 format specifiers, including floats and special values
- **Unique**: Binary and boolean formats from Go
- **Quality**: Professional-grade code with full verification

## License

See parent directory LICENSE file.

## Author

Created as a demonstration of advanced x86_64 assembly programming, combining traditional C printf features with modern Go-inspired format specifiers.
