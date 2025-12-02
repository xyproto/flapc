; Comprehensive printf demonstration
; Shows all format specifiers and proper register preservation

section .data
    ; Header
    header db "═══════════════════════════════════════", 10
           db "  Printf Implementation Demo", 10
           db "═══════════════════════════════════════", 10, 0
    
    ; Integer formats
    int_header db 10, "Integer Formats:", 10, 0
    fmt_decimal db "  Decimal (%%d):     %d", 10, 0
    fmt_unsigned db "  Unsigned (%%u):   %u", 10, 0
    fmt_hex_low db "  Hex lower (%%x):  %x", 10, 0
    fmt_hex_up db "  Hex upper (%%X):  %X", 10, 0
    fmt_octal db "  Octal (%%o):      %o", 10, 0
    fmt_binary db "  Binary (%%b):     %b", 10, 0
    
    ; Float formats
    float_header db 10, "Floating Point:", 10, 0
    fmt_float db "  Float (%%f):      %f", 10, 0
    fmt_sci db "  Scientific (%%e): %e", 10, 0
    
    ; String/Char formats
    str_header db 10, "String & Character:", 10, 0
    fmt_string db "  String (%%s):     %s", 10, 0
    fmt_char db "  Character (%%c):  %c", 10, 0
    
    ; Special formats
    special_header db 10, "Special Formats:", 10, 0
    fmt_pointer db "  Pointer (%%p):    %p", 10, 0
    fmt_bool db "  Boolean (%%t):    %t", 10, 0
    fmt_escape db "  Escape (%%%%):     100%%", 10, 0
    
    ; Multiple arguments
    multi_header db 10, "Multiple Arguments:", 10, 0
    fmt_multi db "  %s: %d items at $%f each = $%f", 10, 0
    
    ; Register test
    reg_header db 10, "Register Preservation Test:", 10, 0
    reg_before db "  Before: r12=%d r13=%d rbx=%d", 10, 0
    reg_after db "  After:  r12=%d r13=%d rbx=%d", 10, 0
    reg_status db "  Status: ", 0
    reg_pass db "✓ PASS", 10, 0
    reg_fail db "✗ FAIL", 10, 0
    
    ; Footer
    footer db 10, "═══════════════════════════════════════", 10, 0
    
    ; Test data
    test_string db "Hello, World", 0
    product_name db "Widget", 0
    pi_val dq 3.141592653589793
    large_val dq 12345.678
    total_val dq 246.912

section .text
    global _start

_start:
    ; Print header
    lea rdi, [rel header]
    call printf
    
    ; Integer formats
    lea rdi, [rel int_header]
    call printf
    
    lea rdi, [rel fmt_decimal]
    mov rsi, -12345
    call printf
    
    lea rdi, [rel fmt_unsigned]
    mov rsi, 18446744073709551615  ; max uint64
    call printf
    
    lea rdi, [rel fmt_hex_low]
    mov rsi, 0xdeadbeef
    call printf
    
    lea rdi, [rel fmt_hex_up]
    mov rsi, 0xcafebabe
    call printf
    
    lea rdi, [rel fmt_octal]
    mov rsi, 511
    call printf
    
    lea rdi, [rel fmt_binary]
    mov rsi, 42
    call printf
    
    ; Floating point
    lea rdi, [rel float_header]
    call printf
    
    lea rdi, [rel fmt_float]
    movsd xmm0, [rel pi_val]
    call printf
    
    lea rdi, [rel fmt_sci]
    movsd xmm0, [rel large_val]
    call printf
    
    ; String & char
    lea rdi, [rel str_header]
    call printf
    
    lea rdi, [rel fmt_string]
    lea rsi, [rel test_string]
    call printf
    
    lea rdi, [rel fmt_char]
    mov rsi, 'A'
    call printf
    
    ; Special formats
    lea rdi, [rel special_header]
    call printf
    
    lea rdi, [rel fmt_pointer]
    lea rsi, [rel test_string]
    call printf
    
    lea rdi, [rel fmt_bool]
    mov rsi, 1
    call printf
    
    lea rdi, [rel fmt_escape]
    call printf
    
    ; Multiple arguments
    lea rdi, [rel multi_header]
    call printf
    
    lea rdi, [rel fmt_multi]
    lea rsi, [rel product_name]
    mov rdx, 79
    movsd xmm0, [rel pi_val]
    movsd xmm1, [rel total_val]
    call printf
    
    ; Register preservation test
    lea rdi, [rel reg_header]
    call printf
    
    ; Set known values
    mov r12, 11111
    mov r13, 22222
    mov rbx, 33333
    
    ; Print before
    lea rdi, [rel reg_before]
    mov rsi, r12
    mov rdx, r13
    mov rcx, rbx
    call printf
    
    ; Save for comparison
    push r12
    push r13
    push rbx
    
    ; Call printf multiple times
    lea rdi, [rel fmt_decimal]
    mov rsi, 100
    call printf
    
    lea rdi, [rel fmt_float]
    movsd xmm0, [rel pi_val]
    call printf
    
    ; Check if preserved
    pop rax  ; rbx original
    pop rcx  ; r13 original
    pop rdx  ; r12 original
    
    ; Print after
    lea rdi, [rel reg_after]
    mov rsi, r12
    push rax
    push rcx
    mov rdx, r13
    mov rcx, rbx
    call printf
    pop rcx  ; restore r13 original
    pop rax  ; restore rbx original
    
    ; Check preservation
    lea rdi, [rel reg_status]
    call printf
    
    cmp r12, rdx
    jne .reg_fail
    cmp r13, rcx
    jne .reg_fail
    cmp rbx, rax
    jne .reg_fail
    
    lea rdi, [rel reg_pass]
    call printf
    jmp .continue
    
.reg_fail:
    lea rdi, [rel reg_fail]
    call printf
    
.continue:
    ; Print footer
    lea rdi, [rel footer]
    call printf
    
    ; Exit
    mov rax, 60
    xor rdi, rdi
    syscall

%include "printf_func.asm"
