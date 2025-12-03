# Float Decimal Extraction - Assembly Tests

## Working Standalone Assembly

The file `float_decimal.asm` contains a **proven working implementation** of SSE decimal extraction that correctly prints "3.141592" for pi.

### Compile and Run
```bash
nasm -f elf64 float_decimal.asm
ld -o float_decimal float_decimal.o
./float_decimal
# Output: 3.141592
```

## Algorithm

1. **Integer Part**: `cvttsd2si rax, xmm0` - truncate to integer
2. **Fractional Part**: `subsd xmm0, xmm1` where xmm1 = floor(value)
3. **Scale**: Multiply fractional by 1,000,000
4. **Extract Digits**: Repeatedly divide by 10, taking remainders

## Integration Challenge

The same algorithm implemented in printf_syscall.go currently produces incorrect results when integrated into the compiler. The issue appears to be:

- Stack/register preservation during calls to other print functions
- XMM register clobbering by emitSyscallPrintInteger()  
- Buffer initialization or syscall interaction

## Test Results

Standalone assembly: ✅ Works perfectly
Compiler integration: ❌ Produces "3.3     " (spaces indicate buffer issues)

## Next Steps

1. Isolate the decimal extraction into a completely separate function
2. Don't call any other functions that might clobber registers
3. Test with simpler values (0.5, 0.25) to verify math
4. Add explicit register preservation (push/pop xmm registers)

## Files

- `float_decimal.asm` - Simple working test (pi)
- `float_decimal_test.asm` - Multiple test values
- `/printf_syscall.go` - Compiler integration (WIP)
