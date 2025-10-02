package main

import "strconv"

func (eb *ExecutableBuilder) SysWriteX86_64(what_data string, what_data_len ...string) {
	eb.Emit("mov rax, " + eb.Lookup("SYS_WRITE"))
	eb.Emit("mov rdi, " + eb.Lookup("STDOUT"))
	eb.Emit("mov rsi, " + what_data)
	if len(what_data_len) == 0 {
		if c, ok := eb.consts[what_data]; ok {
			eb.Emit("mov rdx, " + strconv.Itoa(len(c.value)))
		}
	} else {
		eb.Emit("mov rdx, " + what_data_len[0])
	}
	eb.Emit("syscall")
}

func (eb *ExecutableBuilder) SysExitX86_64(code ...string) {
	eb.Emit("mov rax, " + eb.Lookup("SYS_EXIT"))
	if len(code) == 0 {
		eb.Emit("mov rdi, 0")
	} else {
		eb.Emit("mov rdi, " + code[0])
	}
	eb.Emit("syscall")
}

// RISC-V syscall implementations
func (eb *ExecutableBuilder) SysWriteRiscv64(what_data string, what_data_len ...string) {
	eb.Emit("mov x17, 64")           // write syscall number for RISC-V
	eb.Emit("mov x10, 1")            // stdout file descriptor
	eb.Emit("mov x11, " + what_data) // buffer address
	if len(what_data_len) == 0 {
		if c, ok := eb.consts[what_data]; ok {
			eb.Emit("mov x12, " + strconv.Itoa(len(c.value)))
		}
	} else {
		eb.Emit("mov x12, " + what_data_len[0])
	}
	eb.Emit("ecall")
}

func (eb *ExecutableBuilder) SysExitRiscv64(code ...string) {
	eb.Emit("mov x17, 93") // exit syscall number for RISC-V
	if len(code) == 0 {
		eb.Emit("mov x10, 0")
	} else {
		eb.Emit("mov x10, " + code[0])
	}
	eb.Emit("ecall")
}
