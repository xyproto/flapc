# x86_64 Assembly printf Implementation

This directory contains a comprehensive implementation of `printf` in x86_64 assembly for Linux, inspired by Go's `fmt.Printf`.

## Features

The `printf` function supports the following format specifiers:

### Integer Formats
- `%d` / `%i` - Signed decimal integer
- `%u` - Unsigned decimal integer  
- `%x` - Hexadecimal (lowercase with 0x prefix)
- `%X` - Hexadecimal (uppercase with 0x prefix)
- `%o` - Octal (with 0 prefix)
- `%b` - Binary (with 0b prefix)

### Floating Point
- `%f` - Decimal floating point (6 decimal places)
- `%e` - Scientific notation
- Special values: `NaN`, `+Inf`, `-Inf`

### Other Types
- `%s` - String (null-terminated)
- `%c` - Character
- `%p` - Pointer (hex format, displays `<nil>` for NULL)
- `%t` - Boolean (`true` or `false`)
- `%v` - Default format (currently same as %d)
- `%%` - Literal percent sign

## Building and Running

```bash
nasm -f elf64 printf.asm -o printf.o
ld printf.o -o printf
./printf
```

## Implementation Details

- Uses Linux syscalls (write: 1, exit: 60)
- Follows System V AMD64 ABI calling convention
- Integer arguments: rsi, rdx, rcx (up to 3 arguments)
- Float arguments: xmm0, xmm1, xmm2 (up to 3 arguments)
- Position-independent code using RIP-relative addressing
- Proper handling of:
  - Negative numbers with minus sign
  - Float precision with rounding
  - Special float values (NaN, ±Infinity)
  - NULL pointers
  - Leading zeros for hex/octal/binary

## Example Output

```
Hello, World!
Integer: 42
String: Assembly
Character: A
Lowercase hex: 0x00000000deadbeef
Multiple: 100 and 50
Float: 3.141593
Scientific: 12345678900.000000e+00
Unsigned: 18446744073709551615
Binary: 0b0000000000000000000000000000000000000000000000000000000000101010
Octal: 00000000000000000000100
Uppercase hex: 0x000000000000ABCD
Pointer: 0x000000000040212e
Bool: true
Bool: false
Multi: int=42, float=3.141593, string=Assembly
```

## Comparison to Go's fmt.Printf

This implementation includes many features similar to Go's `fmt.Printf`:
- Multiple format specifiers with type-appropriate formatting
- Boolean formatting (`%t`)
- Default value formatting (`%v`)
- Pointer formatting with nil detection (`%p`)
- Binary, octal, and hex with appropriate prefixes
- Floating point with consistent precision
- Special float value handling (NaN, Inf)

## Technical Notes

- Float arithmetic uses SSE2 instructions (movsd, addsd, mulsd, subsd)
- Rounding is performed using the `roundsd` instruction (SSE4.1)
- All buffers are statically allocated in the BSS section
- Maximum 128-byte buffer for number conversion
- 64-byte dedicated buffer for float conversion
