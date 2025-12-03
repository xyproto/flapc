; Comprehensive SSE Decimal Extraction Test
; Tests multiple float values to verify algorithm
; Expected output:
;   3.141592
;   10.333333
;   0.500000
;   99.999999

global _start

section .data
    test1: dq 3.14159265358979323846      ; pi
    test2: dq 10.33333333333333333333     ; 31/3
    test3: dq 0.5                          ; 1/2
    test4: dq 99.99999999999999999999     ; almost 100
    ten_million: dq 1000000.0
    newline: db 10
    
section .bss
    buffer: resb 16
    
section .text

; Function to print float in xmm0
print_float:
    push rbp
    mov rbp, rsp
    sub rsp, 16
    
    ; Save xmm0 to stack
    movsd [rsp], xmm0
    
    ; Handle negative (though our tests are positive)
    movq rax, xmm0
    test rax, rax
    jns .positive
    mov byte [buffer], '-'
    mov rax, 1
    mov rdi, 1
    lea rsi, [buffer]
    mov rdx, 1
    syscall
    movsd xmm0, [rsp]
    xorpd xmm1, xmm1
    subsd xmm1, xmm0
    movsd xmm0, xmm1
    movsd [rsp], xmm0
    
.positive:
    ; Print integer part
    movsd xmm0, [rsp]
    cvttsd2si rax, xmm0
    
    ; Handle multi-digit integers
    cmp rax, 10
    jl .single_digit
    
    ; Two digit number
    mov rcx, 10
    xor rdx, rdx
    div rcx                             ; rax = tens, rdx = ones
    add rax, 48
    mov [buffer], al
    add rdx, 48
    mov [buffer+1], dl
    mov rax, 1
    mov rdi, 1
    lea rsi, [buffer]
    mov rdx, 2
    syscall
    jmp .decimal_point
    
.single_digit:
    add rax, 48
    mov [buffer], al
    mov rax, 1
    mov rdi, 1
    lea rsi, [buffer]
    mov rdx, 1
    syscall
    
.decimal_point:
    mov byte [buffer], '.'
    mov rax, 1
    mov rdi, 1
    lea rsi, [buffer]
    mov rdx, 1
    syscall
    
    ; Extract fractional part
    movsd xmm0, [rsp]
    cvttsd2si rax, xmm0
    cvtsi2sd xmm1, rax
    movsd xmm0, [rsp]
    subsd xmm0, xmm1                    ; xmm0 = fractional part
    
    ; Multiply by 1000000
    movsd xmm1, [ten_million]
    mulsd xmm0, xmm1
    
    ; Add 0.5 for rounding
    mov rax, 0x3FE0000000000000          ; 0.5 in IEEE 754
    movq xmm1, rax
    addsd xmm0, xmm1
    
    cvttsd2si rax, xmm0
    
    ; Clamp to 999999 in case of rounding overflow
    mov rcx, 999999
    cmp rax, rcx
    jle .no_overflow
    mov rax, rcx
    
.no_overflow:
    ; Extract 6 digits
    mov rcx, 10
    
    xor rdx, rdx
    div rcx
    add rdx, 48
    mov [buffer+5], dl
    
    xor rdx, rdx
    div rcx
    add rdx, 48
    mov [buffer+4], dl
    
    xor rdx, rdx
    div rcx
    add rdx, 48
    mov [buffer+3], dl
    
    xor rdx, rdx
    div rcx
    add rdx, 48
    mov [buffer+2], dl
    
    xor rdx, rdx
    div rcx
    add rdx, 48
    mov [buffer+1], dl
    
    add rax, 48
    mov [buffer+0], al
    
    ; Print 6 digits
    mov rax, 1
    mov rdi, 1
    lea rsi, [buffer]
    mov rdx, 6
    syscall
    
    ; Print newline
    mov rax, 1
    mov rdi, 1
    lea rsi, [newline]
    mov rdx, 1
    syscall
    
    leave
    ret

_start:
    ; Test 1: pi
    movsd xmm0, [test1]
    call print_float
    
    ; Test 2: 10.333...
    movsd xmm0, [test2]
    call print_float
    
    ; Test 3: 0.5
    movsd xmm0, [test3]
    call print_float
    
    ; Test 4: 99.999...
    movsd xmm0, [test4]
    call print_float
    
    ; Exit
    mov rax, 60
    xor rdi, rdi
    syscall
