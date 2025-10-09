package main

import (
	"fmt"
	"os"
)

// Memory Operations for map[uint64]float64 runtime

// MovRegToMem - Store register to memory [base+offset]
func (o *Out) MovRegToMem(src, base string, offset int) {
	switch o.machine {
	case MachineX86_64:
		o.movRegToMemX86(src, base, offset)
	}
}

func (o *Out) movRegToMemX86(src, base string, offset int) {
	fmt.Fprintf(os.Stderr, "mov [%s+%d], %s: ", base, offset, src)

	srcReg, _ := GetRegister(o.machine, src)
	baseReg, _ := GetRegister(o.machine, base)

	// REX.W for 64-bit operation
	rex := uint8(0x48)
	if srcReg.Encoding >= 8 {
		rex |= 0x04 // REX.R
	}
	if baseReg.Encoding >= 8 {
		rex |= 0x01 // REX.B
	}
	o.Write(rex)

	// 0x89 - MOV r/m64, r64
	o.Write(0x89)

	// ModR/M byte with displacement
	if offset == 0 && (baseReg.Encoding&7) != 5 { // rbp/r13 needs displacement
		modrm := uint8(0x00) | ((srcReg.Encoding & 7) << 3) | (baseReg.Encoding & 7)
		o.Write(modrm)
		if (baseReg.Encoding & 7) == 4 { // rsp/r12 needs SIB
			o.Write(0x24)
		}
	} else if offset < 128 && offset >= -128 {
		modrm := uint8(0x40) | ((srcReg.Encoding & 7) << 3) | (baseReg.Encoding & 7)
		o.Write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			o.Write(0x24)
		}
		o.Write(uint8(offset))
	} else {
		modrm := uint8(0x80) | ((srcReg.Encoding & 7) << 3) | (baseReg.Encoding & 7)
		o.Write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			o.Write(0x24)
		}
		o.WriteUnsigned(uint(offset))
	}

	fmt.Fprintln(os.Stderr)
}

// MovMemToReg - Load from memory [base+offset] to register
func (o *Out) MovMemToReg(dst, base string, offset int) {
	switch o.machine {
	case MachineX86_64:
		o.movMemToRegX86(dst, base, offset)
	}
}

func (o *Out) movMemToRegX86(dst, base string, offset int) {
	fmt.Fprintf(os.Stderr, "mov %s, [%s+%d]: ", dst, base, offset)

	dstReg, _ := GetRegister(o.machine, dst)
	baseReg, _ := GetRegister(o.machine, base)

	rex := uint8(0x48)
	if dstReg.Encoding >= 8 {
		rex |= 0x04
	}
	if baseReg.Encoding >= 8 {
		rex |= 0x01
	}
	o.Write(rex)

	// 0x8B - MOV r64, r/m64
	o.Write(0x8B)

	// ModR/M
	if offset == 0 && (baseReg.Encoding&7) != 5 {
		modrm := uint8(0x00) | ((dstReg.Encoding & 7) << 3) | (baseReg.Encoding & 7)
		o.Write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			o.Write(0x24)
		}
	} else if offset < 128 && offset >= -128 {
		modrm := uint8(0x40) | ((dstReg.Encoding & 7) << 3) | (baseReg.Encoding & 7)
		o.Write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			o.Write(0x24)
		}
		o.Write(uint8(offset))
	} else {
		modrm := uint8(0x80) | ((dstReg.Encoding & 7) << 3) | (baseReg.Encoding & 7)
		o.Write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			o.Write(0x24)
		}
		o.WriteUnsigned(uint(offset))
	}

	fmt.Fprintln(os.Stderr)
}

// ShlImmReg - Shift left by immediate
func (o *Out) ShlImmReg(dst string, imm int) {
	switch o.machine {
	case MachineX86_64:
		o.shlImmX86(dst, imm)
	}
}

func (o *Out) shlImmX86(dst string, imm int) {
	fmt.Fprintf(os.Stderr, "shl %s, %d: ", dst, imm)

	dstReg, _ := GetRegister(o.machine, dst)

	rex := uint8(0x48)
	if dstReg.Encoding >= 8 {
		rex |= 0x01
	}
	o.Write(rex)

	if imm == 1 {
		// D1 /4 - SHL r/m64, 1
		o.Write(0xD1)
		modrm := uint8(0xE0) | (dstReg.Encoding & 7) // /4 = 100 in reg field
		o.Write(modrm)
	} else {
		// C1 /4 ib - SHL r/m64, imm8
		o.Write(0xC1)
		modrm := uint8(0xE0) | (dstReg.Encoding & 7)
		o.Write(modrm)
		o.Write(uint8(imm))
	}

	fmt.Fprintln(os.Stderr)
}

// MovByteRegToMem - Store byte from register to memory [base+offset]
func (o *Out) MovByteRegToMem(src, base string, offset int) {
	switch o.machine {
	case MachineX86_64:
		o.movByteRegToMemX86(src, base, offset)
	}
}

func (o *Out) movByteRegToMemX86(src, base string, offset int) {
	fmt.Fprintf(os.Stderr, "mov byte [%s+%d], %s: ", base, offset, src)

	srcReg, _ := GetRegister(o.machine, src)
	baseReg, _ := GetRegister(o.machine, base)

	// REX prefix for extended registers (no REX.W for byte operation)
	needREX := srcReg.Encoding >= 8 || baseReg.Encoding >= 8 || srcReg.Encoding >= 4
	if needREX {
		rex := uint8(0x40)
		if srcReg.Encoding >= 8 {
			rex |= 0x04 // REX.R
		}
		if baseReg.Encoding >= 8 {
			rex |= 0x01 // REX.B
		}
		o.Write(rex)
	}

	// 0x88 - MOV r/m8, r8
	o.Write(0x88)

	// ModR/M byte with displacement
	if offset == 0 && (baseReg.Encoding&7) != 5 {
		modrm := uint8(0x00) | ((srcReg.Encoding & 7) << 3) | (baseReg.Encoding & 7)
		o.Write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			o.Write(0x24)
		}
	} else if offset < 128 && offset >= -128 {
		modrm := uint8(0x40) | ((srcReg.Encoding & 7) << 3) | (baseReg.Encoding & 7)
		o.Write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			o.Write(0x24)
		}
		o.Write(uint8(offset))
	} else {
		modrm := uint8(0x80) | ((srcReg.Encoding & 7) << 3) | (baseReg.Encoding & 7)
		o.Write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			o.Write(0x24)
		}
		o.WriteUnsigned(uint(offset))
	}

	fmt.Fprintln(os.Stderr)
}

// LeaMemToReg - Load effective address [base+offset] to register
func (o *Out) LeaMemToReg(dst, base string, offset int) {
	switch o.machine {
	case MachineX86_64:
		o.leaMemToRegX86(dst, base, offset)
	}
}

func (o *Out) leaMemToRegX86(dst, base string, offset int) {
	fmt.Fprintf(os.Stderr, "lea %s, [%s+%d]: ", dst, base, offset)

	dstReg, _ := GetRegister(o.machine, dst)
	baseReg, _ := GetRegister(o.machine, base)

	rex := uint8(0x48) // REX.W for 64-bit
	if dstReg.Encoding >= 8 {
		rex |= 0x04 // REX.R
	}
	if baseReg.Encoding >= 8 {
		rex |= 0x01 // REX.B
	}
	o.Write(rex)

	// 0x8D - LEA r64, m
	o.Write(0x8D)

	// ModR/M
	if offset == 0 && (baseReg.Encoding&7) != 5 {
		modrm := uint8(0x00) | ((dstReg.Encoding & 7) << 3) | (baseReg.Encoding & 7)
		o.Write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			o.Write(0x24)
		}
	} else if offset < 128 && offset >= -128 {
		modrm := uint8(0x40) | ((dstReg.Encoding & 7) << 3) | (baseReg.Encoding & 7)
		o.Write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			o.Write(0x24)
		}
		o.Write(uint8(offset))
	} else {
		modrm := uint8(0x80) | ((dstReg.Encoding & 7) << 3) | (baseReg.Encoding & 7)
		o.Write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			o.Write(0x24)
		}
		o.WriteUnsigned(uint(offset))
	}

	fmt.Fprintln(os.Stderr)
}
