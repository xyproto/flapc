package main

import (
	"fmt"
	"os"
	"strconv"
)

// MovInstruction handles mov instruction generation using the architecture interface
func (eb *ExecutableBuilder) MovInstruction(dest, val string) error {
	w := eb.TextWriter()
	fmt.Fprint(os.Stderr, fmt.Sprintf("mov %s, %s:", dest, val))

	if err := eb.arch.MovImmediate(w, dest, val); err != nil {
		return err
	}

	return eb.writeValue(w, val)
}

// writeValue writes the immediate value (number or symbol lookup)
func (eb *ExecutableBuilder) writeValue(w Writer, val string) error {
	if n, err := strconv.Atoi(val); err == nil {
		// Direct integer value
		w.WriteUnsigned(uint(n))
	} else {
		// Symbol lookup
		addr := eb.Lookup(val)
		if n, err := strconv.Atoi(addr); err == nil {
			w.WriteUnsigned(uint(n))
		} else {
			return fmt.Errorf("invalid value or symbol: %s", val)
		}
	}

	fmt.Fprintln(os.Stderr)
	return nil
}

// IsValidRegister checks if a register is valid using the architecture interface
func (eb *ExecutableBuilder) IsValidRegister(reg string) bool {
	return eb.arch.IsValidRegister(reg)
}

// RISC-V Architecture Implementation
func (r *Riscv64) MovImmediate(w Writer, dest, val string) error {
	// RISC-V doesn't have a direct mov instruction, use addi rd, x0, imm
	switch dest {
	case "a0", "a1", "a2", "a7":
		w.Write(0x13) // addi instruction
		w.Write(0x08) // Placeholder encoding
		w.Write(0x00)
		w.Write(0x00)
	case "t0", "t1", "t2":
		w.Write(0x13) // addi instruction
		w.Write(0x02) // Different register encoding
		w.Write(0x00)
		w.Write(0x00)
	default:
		return fmt.Errorf("unsupported riscv64 register: %s", dest)
	}
	return nil
}

func (r *Riscv64) IsValidRegister(reg string) bool {
	validRegs := []string{"x0", "x1", "x2", "x3", "x4", "x5", "x6", "x7", "x8", "x9", "x10", "x11", "x12", "x13", "x14", "x15", "x16", "x17", "x18", "x19", "x20", "x21", "x22", "x23", "x24", "x25", "x26", "x27", "x28", "x29", "x30", "x31", "zero", "ra", "sp", "gp", "tp", "t0", "t1", "t2", "s0", "s1", "a0", "a1", "a2", "a3", "a4", "a5", "a6", "a7", "s2", "s3", "s4", "s5", "s6", "s7", "s8", "s9", "s10", "s11", "t3", "t4", "t5", "t6"}
	for _, validReg := range validRegs {
		if reg == validReg {
			return true
		}
	}
	return false
}

func (r *Riscv64) Syscall(w Writer) error {
	w.Write(0x73) // ecall instruction for riscv64
	w.Write(0x00)
	w.Write(0x00)
	w.Write(0x00)
	return nil
}

func (r *Riscv64) ELFMachineType() uint16 {
	return 0xF3 // RISC-V
}

func (r *Riscv64) Name() string {
	return "riscv64"
}

func (eb *ExecutableBuilder) SysWriteRiscv64(what_data string, what_data_len ...string) {
	eb.Emit("mov a7, " + eb.Lookup("SYS_WRITE"))
	eb.Emit("mov a0, " + eb.Lookup("STDOUT"))
	eb.Emit("mov a1, " + what_data)
	if len(what_data_len) == 0 {
		if c, ok := eb.consts[what_data]; ok {
			eb.Emit("mov a2, " + strconv.Itoa(len(c.value)))
		}
	} else {
		eb.Emit("mov a2, " + what_data_len[0])
	}
	eb.Emit("syscall")
}

func (eb *ExecutableBuilder) SysExitRiscv64(code ...string) {
	eb.Emit("mov a7, " + eb.Lookup("SYS_EXIT"))
	if len(code) == 0 {
		eb.Emit("mov a0, 0")
	} else {
		eb.Emit("mov a0, " + code[0])
	}
	eb.Emit("syscall")
}
