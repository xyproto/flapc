package main

import "fmt"

func (a *Aarch64) MovImmediate(w Writer, dest, val string) error {
	switch dest {
	case "x0", "x1", "x2", "x8":
		w.Write(0xd2) // MOV (wide immediate) instruction
		w.Write(0x80) // Placeholder encoding
		w.Write(0x00)
		w.Write(0x08) // Placeholder register encoding
	case "w0", "w1", "w2", "w8": // 32-bit variants
		w.Write(0x52) // MOV (wide immediate) 32-bit
		w.Write(0x80)
		w.Write(0x00)
		w.Write(0x08)
	default:
		return fmt.Errorf("unsupported aarch64 register: %s", dest)
	}
	return nil
}

func (a *Aarch64) IsValidRegister(reg string) bool {
	validRegs := []string{"x0", "x1", "x2", "x3", "x4", "x5", "x6", "x7", "x8", "x9", "x10", "x11", "x12", "x13", "x14", "x15", "x16", "x17", "x18", "x19", "x20", "x21", "x22", "x23", "x24", "x25", "x26", "x27", "x28", "x29", "x30", "sp", "w0", "w1", "w2", "w3", "w4", "w5", "w6", "w7", "w8"}
	for _, validReg := range validRegs {
		if reg == validReg {
			return true
		}
	}
	return false
}

func (a *Aarch64) Syscall(w Writer) error {
	w.Write(0xd4) // svc #0 instruction for aarch64
	w.Write(0x00)
	w.Write(0x00)
	w.Write(0x01)
	return nil
}

func (a *Aarch64) ELFMachineType() uint16 {
	return 0xB7 // ARM64
}

func (a *Aarch64) Name() string {
	return "aarch64"
}
