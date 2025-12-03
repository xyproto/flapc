; SSE Decimal Extraction Test Program
; Tests extracting 6 decimal digits from a float64 value
; Compile: nasm -f elf64 float_decimal.asm && ld -o float_decimal float_decimal.o
; Run: ./float_decimal

global _start

section .data
    test_value: dq 3.14159265358979323846  ; pi
    ten_million: dq 1000000.0
    ten: dq 10.0
    newline: db 10
    
section .bss
    buffer: resb 16
    
section .text
_start:
    ; Load test value into xmm0
    movsd xmm0, [test_value]
    
    ; Print integer part
    cvttsd2si rax, xmm0                 ; rax = 3
    add rax, 48                         ; Convert to ASCII
    mov [buffer], al
    mov rax, 1                          ; sys_write
    mov rdi, 1                          ; stdout
    lea rsi, [buffer]
    mov rdx, 1
    syscall
    
    ; Print decimal point
    mov byte [buffer], '.'
    mov rax, 1
    mov rdi, 1
    lea rsi, [buffer]
    mov rdx, 1
    syscall
    
    ; Extract fractional part
    movsd xmm0, [test_value]            ; xmm0 = 3.14159...
    cvttsd2si rax, xmm0                 ; rax = 3 (integer part)
    cvtsi2sd xmm1, rax                  ; xmm1 = 3.0
    subsd xmm0, xmm1                    ; xmm0 = 0.14159... (fractional part)
    
    ; Multiply by 1000000 to get 6 decimal digits
    movsd xmm1, [ten_million]
    mulsd xmm0, xmm1                    ; xmm0 = 141592.65...
    cvttsd2si rax, xmm0                 ; rax = 141592
    
    ; Extract 6 digits from rax (141592) into buffer
    mov rcx, 10
    
    ; Digit 5 (rightmost)
    xor rdx, rdx
    div rcx                             ; rax = 14159, rdx = 2
    add rdx, 48
    mov [buffer+5], dl
    
    ; Digit 4
    xor rdx, rdx
    div rcx                             ; rax = 1415, rdx = 9
    add rdx, 48
    mov [buffer+4], dl
    
    ; Digit 3
    xor rdx, rdx
    div rcx                             ; rax = 141, rdx = 5
    add rdx, 48
    mov [buffer+3], dl
    
    ; Digit 2
    xor rdx, rdx
    div rcx                             ; rax = 14, rdx = 1
    add rdx, 48
    mov [buffer+2], dl
    
    ; Digit 1
    xor rdx, rdx
    div rcx                             ; rax = 1, rdx = 4
    add rdx, 48
    mov [buffer+1], dl
    
    ; Digit 0 (leftmost)
    add rax, 48
    mov [buffer+0], al
    
    ; Print the 6 digits
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
    
    ; Exit
    mov rax, 60
    xor rdi, rdi
    syscall
