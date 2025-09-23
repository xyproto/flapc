package main

import "fmt"

func (x *X86_64) MovImmediate(w Writer, dest, val string) error {
	w.Write(0x48) // REX.W prefix for 64-bit operation
	w.Write(0xc7) // MOV r/m64, imm32 opcode

	switch dest {
	case "rax":
		w.Write(0xc0)
	case "rbx":
		w.Write(0xc3)
	case "rcx":
		w.Write(0xc1)
	case "rdx":
		w.Write(0xc2)
	case "rdi":
		w.Write(0xc7)
	case "rsi":
		w.Write(0xc6)
	case "r8":
		w.Write(0xc0) // With REX.B
	case "r9":
		w.Write(0xc1) // With REX.B
	default:
		return fmt.Errorf("unsupported x86_64 register: %s", dest)
	}
	return nil
}

func (x *X86_64) IsValidRegister(reg string) bool {
	validRegs := []string{"rax", "rbx", "rcx", "rdx", "rdi", "rsi", "rsp", "rbp", "r8", "r9", "r10", "r11", "r12", "r13", "r14", "r15"}
	for _, validReg := range validRegs {
		if reg == validReg {
			return true
		}
	}
	return false
}

func (x *X86_64) Syscall(w Writer) error {
	w.Write(0x0f) // syscall instruction for x86_64
	w.Write(0x05)
	return nil
}

func (x *X86_64) ELFMachineType() uint16 {
	return 0x3e // AMD x86-64
}

func (x *X86_64) Name() string {
	return "x86_64"
}
