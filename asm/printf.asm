; printf implementation in x86_64 assembly (Linux)
; Uses System V AMD64 ABI calling convention
; Supports: %d/%i (int), %u (uint), %s (string), %c (char), %x (hex), %X (HEX),
;           %o (octal), %b (binary), %f (float), %e (scientific), %p (pointer),
;           %t (bool), %v (default), %%

section .data
    hex_digits db "0123456789abcdef", 0
    hex_digits_upper db "0123456789ABCDEF", 0
    true_str db "true", 0
    false_str db "false", 0
    nil_str db "<nil>", 0
    nan_str db "NaN", 0
    inf_str db "+Inf", 0
    neg_inf_str db "-Inf", 0
    
section .bss
    buffer resb 128         ; Buffer for number conversion
    float_buffer resb 64    ; Buffer for float conversion

section .text
    global _start
    global printf

_start:
    ; Test cases
    lea rdi, [rel msg1]
    call printf
    
    lea rdi, [rel msg2]
    mov rsi, 42
    call printf
    
    lea rdi, [rel msg3]
    lea rsi, [rel name]
    call printf
    
    lea rdi, [rel msg4]
    mov rsi, 'A'
    call printf
    
    lea rdi, [rel msg5]
    mov rsi, 0xDEADBEEF
    call printf
    
    lea rdi, [rel msg6]
    mov rsi, 100
    mov rdx, 50
    call printf
    
    ; Float tests
    lea rdi, [rel msg_float]
    movsd xmm0, [rel pi_val]
    call printf
    
    lea rdi, [rel msg_scientific]
    movsd xmm0, [rel sci_val]
    call printf
    
    ; Unsigned test
    lea rdi, [rel msg_unsigned]
    mov rsi, -1
    call printf
    
    ; Binary test
    lea rdi, [rel msg_binary]
    mov rsi, 42
    call printf
    
    ; Octal test
    lea rdi, [rel msg_octal]
    mov rsi, 64
    call printf
    
    ; Upper hex test
    lea rdi, [rel msg_upper_hex]
    mov rsi, 0xABCD
    call printf
    
    ; Pointer test
    lea rdi, [rel msg_pointer]
    lea rsi, [rel name]
    call printf
    
    ; Bool tests
    lea rdi, [rel msg_bool]
    mov rsi, 1
    call printf
    
    lea rdi, [rel msg_bool]
    xor rsi, rsi
    call printf
    
    ; Multiple types
    lea rdi, [rel msg_multi]
    mov rsi, 42
    movsd xmm0, [rel pi_val]
    lea rdx, [rel name]
    call printf
    
    ; Exit
    mov rax, 60
    xor rdi, rdi
    syscall

; printf(format_string, ...)
; rdi = format string
; rsi, rdx, rcx, r8, r9 = additional int arguments
; xmm0-xmm7 = float arguments
printf:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15
    sub rsp, 88             ; Space for xmm registers + r8/r9 + alignment
    
    ; Save integer argument registers
    mov [rbp - 80], r8      ; arg4
    mov [rbp - 88], r9      ; arg5
    
    ; Save XMM registers
    movsd [rbp - 96], xmm0
    movsd [rbp - 104], xmm1
    movsd [rbp - 112], xmm2
    
    mov r12, rdi            ; r12 = format string
    mov r13, rsi            ; r13 = arg1
    mov r14, rdx            ; r14 = arg2
    mov r15, rcx            ; r15 = arg3
    xor rbx, rbx            ; rbx = int arg index
    xor r10, r10            ; r10 = float arg index
    
.loop:
    movzx rax, byte [r12]
    test al, al
    jz .done
    
    cmp al, '%'
    je .format_spec
    
    ; Regular character - print it
    mov rdi, 1
    lea rsi, [r12]
    mov rdx, 1
    mov rax, 1
    syscall
    
    inc r12
    jmp .loop

.format_spec:
    inc r12
    movzx rax, byte [r12]
    
    cmp al, '%'
    je .print_percent
    
    cmp al, 'd'
    je .print_int
    
    cmp al, 'i'
    je .print_int
    
    cmp al, 'u'
    je .print_uint
    
    cmp al, 's'
    je .print_string
    
    cmp al, 'c'
    je .print_char
    
    cmp al, 'x'
    je .print_hex
    
    cmp al, 'X'
    je .print_hex_upper
    
    cmp al, 'o'
    je .print_octal
    
    cmp al, 'b'
    je .print_binary
    
    cmp al, 'f'
    je .print_float
    
    cmp al, 'e'
    je .print_scientific
    
    cmp al, 'p'
    je .print_pointer
    
    cmp al, 't'
    je .print_bool
    
    cmp al, 'v'
    je .print_int
    
    inc r12
    jmp .loop

.print_percent:
    mov rdi, 1
    lea rsi, [r12]
    mov rdx, 1
    mov rax, 1
    syscall
    inc r12
    jmp .loop

.print_int:
    ; Get current argument based on index
    cmp rbx, 0
    je .use_r13
    cmp rbx, 1
    je .use_r14
    cmp rbx, 2
    je .use_r15
    cmp rbx, 3
    je .use_r8
    cmp rbx, 4
    je .use_r9
    jmp .next_after_int

.use_r13:
    mov rax, r13
    jmp .convert_int
.use_r14:
    mov rax, r14
    jmp .convert_int
.use_r15:
    mov rax, r15
    jmp .convert_int
.use_r8:
    mov rax, [rbp - 80]
    jmp .convert_int
.use_r9:
    mov rax, [rbp - 88]
    
.convert_int:
    inc rbx
    push rbx
    call print_integer
    pop rbx
    jmp .next_after_int

.next_after_int:
    inc r12
    jmp .loop

.print_string:
    cmp rbx, 0
    je .str_r13
    cmp rbx, 1
    je .str_r14
    cmp rbx, 2
    je .str_r15
    cmp rbx, 3
    je .str_r8
    cmp rbx, 4
    je .str_r9
    jmp .next_after_str

.str_r13:
    mov rdi, r13
    jmp .do_print_str
.str_r14:
    mov rdi, r14
    jmp .do_print_str
.str_r15:
    mov rdi, r15
    jmp .do_print_str
.str_r8:
    mov rdi, [rbp - 80]
    jmp .do_print_str
.str_r9:
    mov rdi, [rbp - 88]

.do_print_str:
    inc rbx
    push rbx
    call print_string
    pop rbx
    
.next_after_str:
    inc r12
    jmp .loop

.print_char:
    cmp rbx, 0
    je .char_r13
    cmp rbx, 1
    je .char_r14
    cmp rbx, 2
    je .char_r15
    cmp rbx, 3
    je .char_r8
    cmp rbx, 4
    je .char_r9
    jmp .next_after_char

.char_r13:
    mov rax, r13
    jmp .do_print_char
.char_r14:
    mov rax, r14
    jmp .do_print_char
.char_r15:
    mov rax, r15
    jmp .do_print_char
.char_r8:
    mov rax, [rbp - 80]
    jmp .do_print_char
.char_r9:
    mov rax, [rbp - 88]

.do_print_char:
    inc rbx
    push rax
    mov rdi, 1
    lea rsi, [rsp]
    mov rdx, 1
    mov rax, 1
    syscall
    pop rax
    
.next_after_char:
    inc r12
    jmp .loop

.print_hex:
    cmp rbx, 0
    je .hex_r13
    cmp rbx, 1
    je .hex_r14
    cmp rbx, 2
    je .hex_r15
    cmp rbx, 3
    je .hex_r8
    cmp rbx, 4
    je .hex_r9
    jmp .next_after_hex

.hex_r13:
    mov rax, r13
    jmp .do_print_hex
.hex_r14:
    mov rax, r14
    jmp .do_print_hex
.hex_r15:
    mov rax, r15
    jmp .do_print_hex
.hex_r8:
    mov rax, [rbp - 80]
    jmp .do_print_hex
.hex_r9:
    mov rax, [rbp - 88]

.do_print_hex:
    inc rbx
    push rbx
    call print_hex
    pop rbx
    
.next_after_hex:
    inc r12
    jmp .loop

.print_hex_upper:
    cmp rbx, 0
    je .hex_up_r13
    cmp rbx, 1
    je .hex_up_r14
    cmp rbx, 2
    je .hex_up_r15
    jmp .next_after_hex_up

.hex_up_r13:
    mov rax, r13
    jmp .do_print_hex_up
.hex_up_r14:
    mov rax, r14
    jmp .do_print_hex_up
.hex_up_r15:
    mov rax, r15

.do_print_hex_up:
    inc rbx
    push rbx
    call print_hex_upper
    pop rbx
    
.next_after_hex_up:
    inc r12
    jmp .loop

.print_uint:
    cmp rbx, 0
    je .uint_r13
    cmp rbx, 1
    je .uint_r14
    cmp rbx, 2
    je .uint_r15
    jmp .next_after_uint

.uint_r13:
    mov rax, r13
    jmp .do_print_uint
.uint_r14:
    mov rax, r14
    jmp .do_print_uint
.uint_r15:
    mov rax, r15

.do_print_uint:
    inc rbx
    push rbx
    call print_unsigned
    pop rbx
    
.next_after_uint:
    inc r12
    jmp .loop

.print_octal:
    cmp rbx, 0
    je .oct_r13
    cmp rbx, 1
    je .oct_r14
    cmp rbx, 2
    je .oct_r15
    jmp .next_after_oct

.oct_r13:
    mov rax, r13
    jmp .do_print_oct
.oct_r14:
    mov rax, r14
    jmp .do_print_oct
.oct_r15:
    mov rax, r15

.do_print_oct:
    inc rbx
    push rbx
    call print_octal
    pop rbx
    
.next_after_oct:
    inc r12
    jmp .loop

.print_binary:
    cmp rbx, 0
    je .bin_r13
    cmp rbx, 1
    je .bin_r14
    cmp rbx, 2
    je .bin_r15
    jmp .next_after_bin

.bin_r13:
    mov rax, r13
    jmp .do_print_bin
.bin_r14:
    mov rax, r14
    jmp .do_print_bin
.bin_r15:
    mov rax, r15

.do_print_bin:
    inc rbx
    push rbx
    call print_binary
    pop rbx
    
.next_after_bin:
    inc r12
    jmp .loop

.print_float:
    cmp r10, 0
    je .float_xmm0
    cmp r10, 1
    je .float_xmm1
    cmp r10, 2
    je .float_xmm2
    jmp .next_after_float

.float_xmm0:
    movsd xmm0, [rbp - 96]
    jmp .do_print_float
.float_xmm1:
    movsd xmm0, [rbp - 104]
    jmp .do_print_float
.float_xmm2:
    movsd xmm0, [rbp - 112]

.do_print_float:
    inc r10
    push rbx
    push r10
    call print_float
    pop r10
    pop rbx
    
.next_after_float:
    inc r12
    jmp .loop

.print_scientific:
    cmp r10, 0
    je .sci_xmm0
    cmp r10, 1
    je .sci_xmm1
    cmp r10, 2
    je .sci_xmm2
    jmp .next_after_sci

.sci_xmm0:
    movsd xmm0, [rbp - 96]
    jmp .do_print_sci
.sci_xmm1:
    movsd xmm0, [rbp - 104]
    jmp .do_print_sci
.sci_xmm2:
    movsd xmm0, [rbp - 112]

.do_print_sci:
    inc r10
    push rbx
    push r10
    call print_scientific
    pop r10
    pop rbx
    
.next_after_sci:
    inc r12
    jmp .loop

.print_pointer:
    cmp rbx, 0
    je .ptr_r13
    cmp rbx, 1
    je .ptr_r14
    cmp rbx, 2
    je .ptr_r15
    jmp .next_after_ptr

.ptr_r13:
    mov rax, r13
    jmp .do_print_ptr
.ptr_r14:
    mov rax, r14
    jmp .do_print_ptr
.ptr_r15:
    mov rax, r15

.do_print_ptr:
    inc rbx
    push rbx
    call print_pointer
    pop rbx
    
.next_after_ptr:
    inc r12
    jmp .loop

.print_bool:
    cmp rbx, 0
    je .bool_r13
    cmp rbx, 1
    je .bool_r14
    cmp rbx, 2
    je .bool_r15
    jmp .next_after_bool

.bool_r13:
    mov rax, r13
    jmp .do_print_bool
.bool_r14:
    mov rax, r14
    jmp .do_print_bool
.bool_r15:
    mov rax, r15

.do_print_bool:
    inc rbx
    test rax, rax
    jz .print_false
    lea rdi, [rel true_str]
    call print_string
    jmp .next_after_bool
    
.print_false:
    lea rdi, [rel false_str]
    call print_string
    
.next_after_bool:
    inc r12
    jmp .loop

.done:
    add rsp, 88
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

; print_string(str in rdi)
print_string:
    push rbx
    mov rbx, rdi
    xor rdx, rdx
    
.strlen:
    cmp byte [rbx + rdx], 0
    je .do_write
    inc rdx
    jmp .strlen
    
.do_write:
    mov rsi, rdi
    mov rdi, 1
    mov rax, 1
    syscall
    pop rbx
    ret

; print_integer(value in rax)
print_integer:
    push rbx
    push rcx
    push rdx
    
    test rax, rax
    jns .positive
    
    ; Negative number
    push rax
    mov rdi, 1
    lea rsi, [rel minus]
    mov rdx, 1
    mov rax, 1
    syscall
    pop rax
    neg rax
    
.positive:
    lea rbx, [rel buffer + 31]
    mov byte [rbx], 0
    mov rcx, 10
    
.convert_loop:
    xor rdx, rdx
    div rcx
    add dl, '0'
    dec rbx
    mov [rbx], dl
    test rax, rax
    jnz .convert_loop
    
    ; Print the string
    mov rdi, rbx
    call print_string
    
    pop rdx
    pop rcx
    pop rbx
    ret

; print_hex(value in rax)
print_hex:
    push rbx
    push rcx
    push rdx
    push r8
    
    mov r8, rax          ; Save value in r8
    lea rbx, [rel buffer]
    mov byte [rbx], '0'
    mov byte [rbx + 1], 'x'
    add rbx, 2
    
    mov rcx, 60          ; Start from bit position 60
    
.hex_loop:
    mov rax, r8
    shr rax, cl
    and rax, 0xF
    lea rdx, [rel hex_digits]
    movzx rax, byte [rdx + rax]
    mov [rbx], al
    inc rbx
    
    sub rcx, 4
    cmp rcx, 0
    jge .hex_loop
    
    mov byte [rbx], 0
    
    lea rdi, [rel buffer]
    call print_string
    
    pop r8
    pop rdx
    pop rcx
    pop rbx
    ret

; Helper functions for new format specifiers

; print_unsigned(value in rax)
print_unsigned:
    push rbx
    push rcx
    push rdx
    
    lea rbx, [rel buffer + 31]
    mov byte [rbx], 0
    mov rcx, 10
    
.convert_loop:
    xor rdx, rdx
    div rcx
    add dl, '0'
    dec rbx
    mov [rbx], dl
    test rax, rax
    jnz .convert_loop
    
    mov rdi, rbx
    call print_string
    
    pop rdx
    pop rcx
    pop rbx
    ret

; print_hex_upper(value in rax)
print_hex_upper:
    push rbx
    push rcx
    push rdx
    push r8
    
    mov r8, rax
    lea rbx, [rel buffer]
    mov byte [rbx], '0'
    mov byte [rbx + 1], 'x'
    add rbx, 2
    
    mov rcx, 60
    
.hex_loop:
    mov rax, r8
    shr rax, cl
    and rax, 0xF
    lea rdx, [rel hex_digits_upper]
    movzx rax, byte [rdx + rax]
    mov [rbx], al
    inc rbx
    
    sub rcx, 4
    cmp rcx, 0
    jge .hex_loop
    
    mov byte [rbx], 0
    
    lea rdi, [rel buffer]
    call print_string
    
    pop r8
    pop rdx
    pop rcx
    pop rbx
    ret

; print_octal(value in rax)
print_octal:
    push rbx
    push rcx
    push rdx
    push r8
    
    mov r8, rax
    lea rbx, [rel buffer]
    mov byte [rbx], '0'
    inc rbx
    
    test r8, r8
    jz .zero
    
    mov rcx, 63
    
.oct_loop:
    mov rax, r8
    shr rax, cl
    and rax, 7
    add al, '0'
    mov [rbx], al
    inc rbx
    
    sub rcx, 3
    cmp rcx, 0
    jge .oct_loop
    jmp .finish

.zero:
    mov byte [rbx], '0'
    inc rbx
    
.finish:
    mov byte [rbx], 0
    
    lea rdi, [rel buffer]
    call print_string
    
    pop r8
    pop rdx
    pop rcx
    pop rbx
    ret

; print_binary(value in rax)
print_binary:
    push rbx
    push rcx
    push rdx
    push r8
    
    mov r8, rax
    lea rbx, [rel buffer]
    mov byte [rbx], '0'
    mov byte [rbx + 1], 'b'
    add rbx, 2
    
    mov rcx, 63
    
.bin_loop:
    mov rax, r8
    shr rax, cl
    and rax, 1
    add al, '0'
    mov [rbx], al
    inc rbx
    
    dec rcx
    cmp rcx, 0
    jge .bin_loop
    
    mov byte [rbx], 0
    
    lea rdi, [rel buffer]
    call print_string
    
    pop r8
    pop rdx
    pop rcx
    pop rbx
    ret

; print_float(value in xmm0) - prints with 6 decimal places
print_float:
    push rbx
    push rcx
    push rdx
    push r8
    sub rsp, 16
    
    ; Save original value
    movsd [rsp], xmm0
    movsd [rsp + 8], xmm0
    
    ; Check for NaN/Inf
    mov rax, [rsp]
    mov rcx, rax
    shl rcx, 1
    shr rcx, 53
    cmp rcx, 0x7FF
    je .check_nan_inf
    
    ; Check sign
    test rax, rax
    jns .positive
    
    ; Print minus
    mov rdi, 1
    lea rsi, [rel minus]
    mov rdx, 1
    push rax
    mov rax, 1
    syscall
    pop rax
    
    ; Make positive
    mov rcx, 0x7FFFFFFFFFFFFFFF
    and rax, rcx
    mov [rsp], rax
    movsd xmm0, [rsp]
    
.positive:
    ; Round to 6 decimal places (add 0.0000005)
    movsd xmm1, [rel round_factor]
    addsd xmm0, xmm1
    
    ; Convert integer part
    cvttsd2si rax, xmm0
    push rax
    call print_unsigned
    pop rax
    
    ; Print decimal point
    push rax
    mov rdi, 1
    lea rsi, [rel decimal_point]
    mov rdx, 1
    mov rax, 1
    syscall
    pop rax
    
    ; Get fractional part
    cvtsi2sd xmm1, rax
    movsd xmm0, [rsp]
    
    ; Check if negative originally
    mov rax, [rsp]
    test rax, rax
    jns .frac_positive
    
    ; For negative, negate both
    movsd xmm2, [rel neg_one]
    mulsd xmm0, xmm2
    mulsd xmm1, xmm2
    
.frac_positive:
    subsd xmm0, xmm1
    
    ; Multiply by 1000000 for 6 decimal places
    movsd xmm1, [rel million]
    mulsd xmm0, xmm1
    
    ; Round
    roundsd xmm0, xmm0, 0
    cvttsd2si rax, xmm0
    
    ; Handle edge case where rounding gives us 1000000
    cmp rax, 1000000
    jl .print_frac
    mov rax, 999999
    
.print_frac:
    ; Print 6 digits with leading zeros
    lea rbx, [rel float_buffer + 5]
    mov byte [rbx + 1], 0
    mov rcx, 6
    
.frac_loop:
    xor rdx, rdx
    mov r8, 10
    div r8
    add dl, '0'
    mov [rbx], dl
    dec rbx
    dec rcx
    jnz .frac_loop
    
    lea rdi, [rel float_buffer]
    call print_string
    
    add rsp, 16
    pop r8
    pop rdx
    pop rcx
    pop rbx
    ret

.check_nan_inf:
    mov rcx, rax
    shl rcx, 12
    test rcx, rcx
    jnz .print_nan
    
    ; Infinity
    test rax, rax
    js .print_neg_inf
    lea rdi, [rel inf_str]
    call print_string
    jmp .done_special
    
.print_neg_inf:
    lea rdi, [rel neg_inf_str]
    call print_string
    jmp .done_special
    
.print_nan:
    lea rdi, [rel nan_str]
    call print_string
    
.done_special:
    add rsp, 16
    pop r8
    pop rdx
    pop rcx
    pop rbx
    ret

; print_scientific(value in xmm0)
print_scientific:
    push rbx
    push rcx
    push rdx
    push r8
    sub rsp, 16
    
    movsd [rsp], xmm0
    mov rax, [rsp]
    
    ; Check for special values
    mov rcx, rax
    shl rcx, 1
    shr rcx, 53
    cmp rcx, 0x7FF
    je .special
    
    ; Check sign
    test rax, rax
    jns .pos
    
    mov rdi, 1
    lea rsi, [rel minus]
    mov rdx, 1
    push rax
    mov rax, 1
    syscall
    pop rax
    
    mov rcx, 0x7FFFFFFFFFFFFFFF
    and rax, rcx
    mov [rsp], rax
    movsd xmm0, [rsp]
    
.pos:
    ; Get exponent (simplified - just print as float for now)
    call print_float
    lea rdi, [rel exp_suffix]
    call print_string
    
.done:
    add rsp, 16
    pop r8
    pop rdx
    pop rcx
    pop rbx
    ret
    
.special:
    add rsp, 16
    pop r8
    pop rdx
    pop rcx
    pop rbx
    jmp print_float.check_nan_inf

; print_pointer(value in rax)
print_pointer:
    test rax, rax
    jnz .not_nil
    
    lea rdi, [rel nil_str]
    call print_string
    ret
    
.not_nil:
    jmp print_hex

section .data
    msg1 db "Hello, World!", 10, 0
    msg2 db "Integer: %d", 10, 0
    msg3 db "String: %s", 10, 0
    msg4 db "Character: %c", 10, 0
    msg5 db "Lowercase hex: %x", 10, 0
    msg6 db "Multiple: %d and %d", 10, 0
    msg_float db "Float: %f", 10, 0
    msg_scientific db "Scientific: %e", 10, 0
    msg_unsigned db "Unsigned: %u", 10, 0
    msg_binary db "Binary: %b", 10, 0
    msg_octal db "Octal: %o", 10, 0
    msg_upper_hex db "Uppercase hex: %X", 10, 0
    msg_pointer db "Pointer: %p", 10, 0
    msg_bool db "Bool: %t", 10, 0
    msg_multi db "Multi: int=%d, float=%f, string=%s", 10, 0
    
    name db "Assembly", 0
    minus db "-", 0
    decimal_point db ".", 0
    exp_suffix db "e+00", 0
    
    pi_val dq 3.141592653589793
    sci_val dq 1.23456789e10
    round_factor dq 0.0000005
    million dq 1000000.0
    neg_one dq -1.0
