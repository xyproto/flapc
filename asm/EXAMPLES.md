# Printf Examples

## Basic Usage

```nasm
section .data
    msg db "Hello, %s!", 10, 0
    name db "World", 0

section .text
    global _start

_start:
    lea rdi, [rel msg]
    lea rsi, [rel name]
    call printf
    
    mov rax, 60
    xor rdi, rdi
    syscall

; Include printf implementation here or link separately
```

## Integer Formats

```nasm
; Decimal
lea rdi, [rel fmt_decimal]     ; "%d\n"
mov rsi, -42
call printf

; Unsigned
lea rdi, [rel fmt_unsigned]    ; "%u\n"
mov rsi, 18446744073709551615
call printf

; Hexadecimal (lowercase)
lea rdi, [rel fmt_hex]          ; "%x\n"
mov rsi, 0xDEADBEEF
call printf

; Hexadecimal (uppercase)
lea rdi, [rel fmt_HEX]          ; "%X\n"
mov rsi, 0xABCD
call printf

; Octal
lea rdi, [rel fmt_octal]        ; "%o\n"
mov rsi, 64
call printf

; Binary
lea rdi, [rel fmt_binary]       ; "%b\n"
mov rsi, 42
call printf
```

## Floating Point

```nasm
; Float (6 decimal places)
lea rdi, [rel fmt_float]        ; "%f\n"
movsd xmm0, [rel pi_value]      ; pi_value dq 3.141592653589793
call printf

; Scientific notation
lea rdi, [rel fmt_sci]          ; "%e\n"
movsd xmm0, [rel large_value]   ; large_value dq 1.23e10
call printf

; Multiple floats
lea rdi, [rel fmt_multi]        ; "%f + %f = %f\n"
movsd xmm0, [rel val1]
movsd xmm1, [rel val2]
movsd xmm2, [rel sum]
call printf
```

## Mixed Arguments

```nasm
; Integer and string
lea rdi, [rel fmt]              ; "Count: %d, Name: %s\n"
mov rsi, 100
lea rdx, [rel name_str]
call printf

; Integer and float
lea rdi, [rel fmt]              ; "Int: %d, Float: %f\n"
mov rsi, 42
movsd xmm0, [rel pi]
call printf

; Complex mix
lea rdi, [rel fmt]              ; "%s: %d (0x%x) = %b\n"
lea rsi, [rel label]
mov rdx, 42
mov rcx, 42
mov r8, 42
call printf
```

## Special Types

```nasm
; Boolean
lea rdi, [rel fmt_bool]         ; "Success: %t\n"
mov rsi, 1                      ; true
call printf

; Pointer
lea rdi, [rel fmt_ptr]          ; "Address: %p\n"
lea rsi, [rel some_data]
call printf

; NULL pointer
lea rdi, [rel fmt_ptr]
xor rsi, rsi                    ; Will print "<nil>"
call printf
```

## Format Escaping

```nasm
; Print a literal percent sign
lea rdi, [rel fmt]              ; "Progress: 100%%\n"
call printf
```

## Argument Limits

The implementation supports:
- Up to 5 integer arguments (rsi, rdx, rcx, r8, r9)
- Up to 3 float arguments (xmm0, xmm1, xmm2)

Example with maximum arguments:

```nasm
lea rdi, [rel fmt]              ; "%d %d %d %d %d\n"
mov rsi, 1
mov rdx, 2
mov rcx, 3
mov r8, 4
mov r9, 5
call printf
```

## Complete Example Program

```nasm
section .data
    intro db "=== Printf Demo ===", 10, 0
    int_fmt db "Integer: %d", 10, 0
    float_fmt db "Pi: %f", 10, 0
    mixed_fmt db "%s scored %d points (%.2f average)", 10, 0
    
    player_name db "Alice", 0
    pi_value dq 3.141592653589793
    
section .text
    global _start

_start:
    ; Print intro
    lea rdi, [rel intro]
    call printf
    
    ; Print integer
    lea rdi, [rel int_fmt]
    mov rsi, 42
    call printf
    
    ; Print float
    lea rdi, [rel float_fmt]
    movsd xmm0, [rel pi_value]
    call printf
    
    ; Exit
    mov rax, 60
    xor rdi, rdi
    syscall

; Include printf implementation
%include "printf.asm"
```

## Building

```bash
# Assemble
nasm -f elf64 your_program.asm -o your_program.o

# Link
ld your_program.o -o your_program

# Run
./your_program
```

## Tips

1. **Argument Order**: Integer arguments come before floats in registers
2. **Stack Alignment**: Printf handles stack alignment internally
3. **Register Preservation**: Printf preserves all callee-saved registers
4. **Precision**: Float output is fixed at 6 decimal places
5. **Buffer Size**: Numbers up to 64-bit are fully supported
6. **String Safety**: Ensure all strings are null-terminated
