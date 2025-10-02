define msg "Hello, World!\n"

lea rdi, msg
call printf
mov rdi, 0
call exit
