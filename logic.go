package main

import (
	"fmt"
	"os"
)

// Logical operations (AND, OR, XOR, NOT) for bitwise manipulation
// Essential for implementing Flap's logical expressions:
//   - Boolean expressions: condition1 and condition2
//   - Bitwise operations: flags & mask, flags | bit, ~flags
//   - Pattern matching: compound conditions
//   - Set operations: intersection (AND), union (OR), symmetric difference (XOR)
//   - Flag manipulation: entity.flags & VISIBLE
//   - Bit testing: value & (1 << n)
//   - Bit inversion: ~b value

// AndRegWithReg generates AND dst, src (dst = dst & src)
func (o *Out) AndRegWithReg(dst, src string) {
	switch o.machine {
	case MachineX86_64:
		o.andX86RegWithReg(dst, src)
	case MachineARM64:
		o.andARM64RegWithReg(dst, src)
	case MachineRiscv64:
		o.andRISCVRegWithReg(dst, src)
	}
}

// AndRegWithImm generates AND dst, imm (dst = dst & imm)
func (o *Out) AndRegWithImm(dst string, imm int32) {
	switch o.machine {
	case MachineX86_64:
		o.andX86RegWithImm(dst, imm)
	case MachineARM64:
		o.andARM64RegWithImm(dst, imm)
	case MachineRiscv64:
		o.andRISCVRegWithImm(dst, imm)
	}
}

// AndRegWithRegToReg generates AND dst, src1, src2 (dst = src1 & src2)
// 3-operand form for ARM64 and RISC-V
func (o *Out) AndRegWithRegToReg(dst, src1, src2 string) {
	switch o.machine {
	case MachineX86_64:
		// x86-64: MOV dst, src1; AND dst, src2
		o.MovRegToReg(dst, src1)
		o.AndRegWithReg(dst, src2)
	case MachineARM64:
		o.andARM64RegWithRegToReg(dst, src1, src2)
	case MachineRiscv64:
		o.andRISCVRegWithRegToReg(dst, src1, src2)
	}
}

// OrRegWithReg generates OR dst, src (dst = dst | src)
func (o *Out) OrRegWithReg(dst, src string) {
	switch o.machine {
	case MachineX86_64:
		o.orX86RegWithReg(dst, src)
	case MachineARM64:
		o.orARM64RegWithReg(dst, src)
	case MachineRiscv64:
		o.orRISCVRegWithReg(dst, src)
	}
}

// OrRegWithImm generates OR dst, imm (dst = dst | imm)
func (o *Out) OrRegWithImm(dst string, imm int32) {
	switch o.machine {
	case MachineX86_64:
		o.orX86RegWithImm(dst, imm)
	case MachineARM64:
		o.orARM64RegWithImm(dst, imm)
	case MachineRiscv64:
		o.orRISCVRegWithImm(dst, imm)
	}
}

// OrRegWithRegToReg generates OR dst, src1, src2 (dst = src1 | src2)
// 3-operand form for ARM64 and RISC-V
func (o *Out) OrRegWithRegToReg(dst, src1, src2 string) {
	switch o.machine {
	case MachineX86_64:
		// x86-64: MOV dst, src1; OR dst, src2
		o.MovRegToReg(dst, src1)
		o.OrRegWithReg(dst, src2)
	case MachineARM64:
		o.orARM64RegWithRegToReg(dst, src1, src2)
	case MachineRiscv64:
		o.orRISCVRegWithRegToReg(dst, src1, src2)
	}
}

// XorRegWithReg generates XOR dst, src (dst = dst ^ src)
func (o *Out) XorRegWithReg(dst, src string) {
	switch o.machine {
	case MachineX86_64:
		o.xorX86RegWithReg(dst, src)
	case MachineARM64:
		o.xorARM64RegWithReg(dst, src)
	case MachineRiscv64:
		o.xorRISCVRegWithReg(dst, src)
	}
}

// XorRegWithImm generates XOR dst, imm (dst = dst ^ imm)
func (o *Out) XorRegWithImm(dst string, imm int32) {
	switch o.machine {
	case MachineX86_64:
		o.xorX86RegWithImm(dst, imm)
	case MachineARM64:
		o.xorARM64RegWithImm(dst, imm)
	case MachineRiscv64:
		o.xorRISCVRegWithImm(dst, imm)
	}
}

// XorRegWithRegToReg generates XOR dst, src1, src2 (dst = src1 ^ src2)
// 3-operand form for ARM64 and RISC-V
func (o *Out) XorRegWithRegToReg(dst, src1, src2 string) {
	switch o.machine {
	case MachineX86_64:
		// x86-64: MOV dst, src1; XOR dst, src2
		o.MovRegToReg(dst, src1)
		o.XorRegWithReg(dst, src2)
	case MachineARM64:
		o.xorARM64RegWithRegToReg(dst, src1, src2)
	case MachineRiscv64:
		o.xorRISCVRegWithRegToReg(dst, src1, src2)
	}
}

// ============================================================================
// x86-64 implementations
// ============================================================================

// x86-64 AND (register-register)
func (o *Out) andX86RegWithReg(dst, src string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	srcReg, srcOk := GetRegister(o.machine, src)
	if !dstOk || !srcOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "and %s, %s:", dst, src)
	}

	// REX prefix for 64-bit operation
	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01 // REX.B
	}
	if (srcReg.Encoding & 8) != 0 {
		rex |= 0x04 // REX.R
	}
	o.Write(rex)

	// AND opcode (0x21 for r/m64, r64)
	o.Write(0x21)

	// ModR/M: 11 (register direct) | reg (src) | r/m (dst)
	modrm := uint8(0xC0) | ((srcReg.Encoding & 7) << 3) | (dstReg.Encoding & 7)
	o.Write(modrm)

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// x86-64 AND with immediate
func (o *Out) andX86RegWithImm(dst string, imm int32) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	if !dstOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "and %s, %d:", dst, imm)
	}

	// REX prefix for 64-bit operation
	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01 // REX.B
	}
	o.Write(rex)

	// Check if immediate fits in 8 bits
	if imm >= -128 && imm <= 127 {
		// AND r/m64, imm8 (opcode 0x83 /4)
		o.Write(0x83)
		modrm := uint8(0xE0) | (dstReg.Encoding & 7) // opcode extension /4
		o.Write(modrm)
		o.Write(uint8(imm & 0xFF))
	} else {
		// AND r/m64, imm32 (opcode 0x81 /4)
		o.Write(0x81)
		modrm := uint8(0xE0) | (dstReg.Encoding & 7) // opcode extension /4
		o.Write(modrm)

		// Write 32-bit immediate
		o.Write(uint8(imm & 0xFF))
		o.Write(uint8((imm >> 8) & 0xFF))
		o.Write(uint8((imm >> 16) & 0xFF))
		o.Write(uint8((imm >> 24) & 0xFF))
	}

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// x86-64 OR (register-register)
func (o *Out) orX86RegWithReg(dst, src string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	srcReg, srcOk := GetRegister(o.machine, src)
	if !dstOk || !srcOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "or %s, %s:", dst, src)
	}

	// REX prefix for 64-bit operation
	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01 // REX.B
	}
	if (srcReg.Encoding & 8) != 0 {
		rex |= 0x04 // REX.R
	}
	o.Write(rex)

	// OR opcode (0x09 for r/m64, r64)
	o.Write(0x09)

	// ModR/M: 11 (register direct) | reg (src) | r/m (dst)
	modrm := uint8(0xC0) | ((srcReg.Encoding & 7) << 3) | (dstReg.Encoding & 7)
	o.Write(modrm)

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// x86-64 OR with immediate
func (o *Out) orX86RegWithImm(dst string, imm int32) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	if !dstOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "or %s, %d:", dst, imm)
	}

	// REX prefix for 64-bit operation
	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01 // REX.B
	}
	o.Write(rex)

	// Check if immediate fits in 8 bits
	if imm >= -128 && imm <= 127 {
		// OR r/m64, imm8 (opcode 0x83 /1)
		o.Write(0x83)
		modrm := uint8(0xC8) | (dstReg.Encoding & 7) // opcode extension /1
		o.Write(modrm)
		o.Write(uint8(imm & 0xFF))
	} else {
		// OR r/m64, imm32 (opcode 0x81 /1)
		o.Write(0x81)
		modrm := uint8(0xC8) | (dstReg.Encoding & 7) // opcode extension /1
		o.Write(modrm)

		// Write 32-bit immediate
		o.Write(uint8(imm & 0xFF))
		o.Write(uint8((imm >> 8) & 0xFF))
		o.Write(uint8((imm >> 16) & 0xFF))
		o.Write(uint8((imm >> 24) & 0xFF))
	}

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// x86-64 XOR (register-register)
func (o *Out) xorX86RegWithReg(dst, src string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	srcReg, srcOk := GetRegister(o.machine, src)
	if !dstOk || !srcOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "xor %s, %s:", dst, src)
	}

	// REX prefix for 64-bit operation
	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01 // REX.B
	}
	if (srcReg.Encoding & 8) != 0 {
		rex |= 0x04 // REX.R
	}
	o.Write(rex)

	// XOR opcode (0x31 for r/m64, r64)
	o.Write(0x31)

	// ModR/M: 11 (register direct) | reg (src) | r/m (dst)
	modrm := uint8(0xC0) | ((srcReg.Encoding & 7) << 3) | (dstReg.Encoding & 7)
	o.Write(modrm)

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// x86-64 XOR with immediate
func (o *Out) xorX86RegWithImm(dst string, imm int32) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	if !dstOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "xor %s, %d:", dst, imm)
	}

	// REX prefix for 64-bit operation
	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01 // REX.B
	}
	o.Write(rex)

	// Check if immediate fits in 8 bits
	if imm >= -128 && imm <= 127 {
		// XOR r/m64, imm8 (opcode 0x83 /6)
		o.Write(0x83)
		modrm := uint8(0xF0) | (dstReg.Encoding & 7) // opcode extension /6
		o.Write(modrm)
		o.Write(uint8(imm & 0xFF))
	} else {
		// XOR r/m64, imm32 (opcode 0x81 /6)
		o.Write(0x81)
		modrm := uint8(0xF0) | (dstReg.Encoding & 7) // opcode extension /6
		o.Write(modrm)

		// Write 32-bit immediate
		o.Write(uint8(imm & 0xFF))
		o.Write(uint8((imm >> 8) & 0xFF))
		o.Write(uint8((imm >> 16) & 0xFF))
		o.Write(uint8((imm >> 24) & 0xFF))
	}

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// ============================================================================
// ARM64 implementations
// ============================================================================

// ARM64 AND (register-register, 2-operand form)
func (o *Out) andARM64RegWithReg(dst, src string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	srcReg, srcOk := GetRegister(o.machine, src)
	if !dstOk || !srcOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "and %s, %s, %s:", dst, dst, src)
	}

	// AND Xd, Xn, Xm (shifted register)
	// Format: sf 0 01010 shift 0 Rm imm6 Rn Rd
	// sf=1 (64-bit), shift=00 (LSL #0), Rm=src, Rn=dst (same as Rd), Rd=dst
	instr := uint32(0x8A000000) |
		(uint32(srcReg.Encoding&31) << 16) | // Rm
		(uint32(dstReg.Encoding&31) << 5) | // Rn (same as Rd for 2-operand)
		uint32(dstReg.Encoding&31) // Rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// ARM64 AND - 3 operand form
func (o *Out) andARM64RegWithRegToReg(dst, src1, src2 string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	src1Reg, src1Ok := GetRegister(o.machine, src1)
	src2Reg, src2Ok := GetRegister(o.machine, src2)
	if !dstOk || !src1Ok || !src2Ok {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "and %s, %s, %s:", dst, src1, src2)
	}

	instr := uint32(0x8A000000) |
		(uint32(src2Reg.Encoding&31) << 16) | // Rm
		(uint32(src1Reg.Encoding&31) << 5) | // Rn
		uint32(dstReg.Encoding&31) // Rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// ARM64 AND with immediate
func (o *Out) andARM64RegWithImm(dst string, imm int32) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	if !dstOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "and %s, %s, #%d:", dst, dst, imm)
	}

	// AND (immediate) uses a complex bitmask encoding
	// For simplicity, we'll use a limited set of encodable immediates
	// Format: sf 1 00100 1 0 immr imms Rn Rd
	// This is a simplified version - full implementation would need bitmask encoding
	instr := uint32(0x92000000) |
		(uint32(dstReg.Encoding&31) << 5) | // Rn (same as Rd)
		uint32(dstReg.Encoding&31) // Rd
	// Note: immr and imms fields would need proper bitmask encoding

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// ARM64 ORR (register-register, 2-operand form)
func (o *Out) orARM64RegWithReg(dst, src string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	srcReg, srcOk := GetRegister(o.machine, src)
	if !dstOk || !srcOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "orr %s, %s, %s:", dst, dst, src)
	}

	// ORR Xd, Xn, Xm (shifted register)
	// Format: sf 0 10101 shift 0 Rm imm6 Rn Rd
	instr := uint32(0xAA000000) |
		(uint32(srcReg.Encoding&31) << 16) | // Rm
		(uint32(dstReg.Encoding&31) << 5) | // Rn (same as Rd for 2-operand)
		uint32(dstReg.Encoding&31) // Rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// ARM64 ORR - 3 operand form
func (o *Out) orARM64RegWithRegToReg(dst, src1, src2 string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	src1Reg, src1Ok := GetRegister(o.machine, src1)
	src2Reg, src2Ok := GetRegister(o.machine, src2)
	if !dstOk || !src1Ok || !src2Ok {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "orr %s, %s, %s:", dst, src1, src2)
	}

	instr := uint32(0xAA000000) |
		(uint32(src2Reg.Encoding&31) << 16) | // Rm
		(uint32(src1Reg.Encoding&31) << 5) | // Rn
		uint32(dstReg.Encoding&31) // Rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// ARM64 ORR with immediate
func (o *Out) orARM64RegWithImm(dst string, imm int32) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	if !dstOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "orr %s, %s, #%d:", dst, dst, imm)
	}

	// ORR (immediate) uses bitmask encoding
	// Format: sf 1 01100 1 0 immr imms Rn Rd
	instr := uint32(0xB2000000) |
		(uint32(dstReg.Encoding&31) << 5) | // Rn (same as Rd)
		uint32(dstReg.Encoding&31) // Rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// ARM64 EOR (XOR) (register-register, 2-operand form)
func (o *Out) xorARM64RegWithReg(dst, src string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	srcReg, srcOk := GetRegister(o.machine, src)
	if !dstOk || !srcOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "eor %s, %s, %s:", dst, dst, src)
	}

	// EOR Xd, Xn, Xm (shifted register)
	// Format: sf 1 01010 shift 0 Rm imm6 Rn Rd
	instr := uint32(0xCA000000) |
		(uint32(srcReg.Encoding&31) << 16) | // Rm
		(uint32(dstReg.Encoding&31) << 5) | // Rn (same as Rd for 2-operand)
		uint32(dstReg.Encoding&31) // Rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// ARM64 EOR - 3 operand form
func (o *Out) xorARM64RegWithRegToReg(dst, src1, src2 string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	src1Reg, src1Ok := GetRegister(o.machine, src1)
	src2Reg, src2Ok := GetRegister(o.machine, src2)
	if !dstOk || !src1Ok || !src2Ok {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "eor %s, %s, %s:", dst, src1, src2)
	}

	instr := uint32(0xCA000000) |
		(uint32(src2Reg.Encoding&31) << 16) | // Rm
		(uint32(src1Reg.Encoding&31) << 5) | // Rn
		uint32(dstReg.Encoding&31) // Rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// ARM64 EOR with immediate
func (o *Out) xorARM64RegWithImm(dst string, imm int32) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	if !dstOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "eor %s, %s, #%d:", dst, dst, imm)
	}

	// EOR (immediate) uses bitmask encoding
	// Format: sf 1 10100 1 0 immr imms Rn Rd
	instr := uint32(0xD2000000) |
		(uint32(dstReg.Encoding&31) << 5) | // Rn (same as Rd)
		uint32(dstReg.Encoding&31) // Rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// ============================================================================
// RISC-V implementations
// ============================================================================

// RISC-V AND (register-register, 2-operand form)
func (o *Out) andRISCVRegWithReg(dst, src string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	srcReg, srcOk := GetRegister(o.machine, src)
	if !dstOk || !srcOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "and %s, %s, %s:", dst, dst, src)
	}

	// AND: 0000000 rs2 rs1 111 rd 0110011
	instr := uint32(0x33) |
		(7 << 12) | // funct3 = 111 (AND)
		(uint32(srcReg.Encoding&31) << 20) | // rs2
		(uint32(dstReg.Encoding&31) << 15) | // rs1 (same as rd)
		(uint32(dstReg.Encoding&31) << 7) // rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// RISC-V AND - 3 operand form
func (o *Out) andRISCVRegWithRegToReg(dst, src1, src2 string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	src1Reg, src1Ok := GetRegister(o.machine, src1)
	src2Reg, src2Ok := GetRegister(o.machine, src2)
	if !dstOk || !src1Ok || !src2Ok {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "and %s, %s, %s:", dst, src1, src2)
	}

	instr := uint32(0x33) |
		(7 << 12) | // funct3 = 111 (AND)
		(uint32(src2Reg.Encoding&31) << 20) | // rs2
		(uint32(src1Reg.Encoding&31) << 15) | // rs1
		(uint32(dstReg.Encoding&31) << 7) // rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// RISC-V ANDI (AND immediate)
func (o *Out) andRISCVRegWithImm(dst string, imm int32) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	if !dstOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "andi %s, %s, %d:", dst, dst, imm)
	}

	// ANDI: imm[11:0] rs1 111 rd 0010011
	instr := uint32(0x13) |
		(7 << 12) | // funct3 = 111 (ANDI)
		(uint32(imm&0xFFF) << 20) | // imm[11:0]
		(uint32(dstReg.Encoding&31) << 15) | // rs1 (same as rd)
		(uint32(dstReg.Encoding&31) << 7) // rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// RISC-V OR (register-register, 2-operand form)
func (o *Out) orRISCVRegWithReg(dst, src string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	srcReg, srcOk := GetRegister(o.machine, src)
	if !dstOk || !srcOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "or %s, %s, %s:", dst, dst, src)
	}

	// OR: 0000000 rs2 rs1 110 rd 0110011
	instr := uint32(0x33) |
		(6 << 12) | // funct3 = 110 (OR)
		(uint32(srcReg.Encoding&31) << 20) | // rs2
		(uint32(dstReg.Encoding&31) << 15) | // rs1 (same as rd)
		(uint32(dstReg.Encoding&31) << 7) // rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// RISC-V OR - 3 operand form
func (o *Out) orRISCVRegWithRegToReg(dst, src1, src2 string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	src1Reg, src1Ok := GetRegister(o.machine, src1)
	src2Reg, src2Ok := GetRegister(o.machine, src2)
	if !dstOk || !src1Ok || !src2Ok {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "or %s, %s, %s:", dst, src1, src2)
	}

	instr := uint32(0x33) |
		(6 << 12) | // funct3 = 110 (OR)
		(uint32(src2Reg.Encoding&31) << 20) | // rs2
		(uint32(src1Reg.Encoding&31) << 15) | // rs1
		(uint32(dstReg.Encoding&31) << 7) // rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// RISC-V ORI (OR immediate)
func (o *Out) orRISCVRegWithImm(dst string, imm int32) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	if !dstOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "ori %s, %s, %d:", dst, dst, imm)
	}

	// ORI: imm[11:0] rs1 110 rd 0010011
	instr := uint32(0x13) |
		(6 << 12) | // funct3 = 110 (ORI)
		(uint32(imm&0xFFF) << 20) | // imm[11:0]
		(uint32(dstReg.Encoding&31) << 15) | // rs1 (same as rd)
		(uint32(dstReg.Encoding&31) << 7) // rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// RISC-V XOR (register-register, 2-operand form)
func (o *Out) xorRISCVRegWithReg(dst, src string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	srcReg, srcOk := GetRegister(o.machine, src)
	if !dstOk || !srcOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "xor %s, %s, %s:", dst, dst, src)
	}

	// XOR: 0000000 rs2 rs1 100 rd 0110011
	instr := uint32(0x33) |
		(4 << 12) | // funct3 = 100 (XOR)
		(uint32(srcReg.Encoding&31) << 20) | // rs2
		(uint32(dstReg.Encoding&31) << 15) | // rs1 (same as rd)
		(uint32(dstReg.Encoding&31) << 7) // rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// RISC-V XOR - 3 operand form
func (o *Out) xorRISCVRegWithRegToReg(dst, src1, src2 string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	src1Reg, src1Ok := GetRegister(o.machine, src1)
	src2Reg, src2Ok := GetRegister(o.machine, src2)
	if !dstOk || !src1Ok || !src2Ok {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "xor %s, %s, %s:", dst, src1, src2)
	}

	instr := uint32(0x33) |
		(4 << 12) | // funct3 = 100 (XOR)
		(uint32(src2Reg.Encoding&31) << 20) | // rs2
		(uint32(src1Reg.Encoding&31) << 15) | // rs1
		(uint32(dstReg.Encoding&31) << 7) // rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// RISC-V XORI (XOR immediate)
func (o *Out) xorRISCVRegWithImm(dst string, imm int32) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	if !dstOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "xori %s, %s, %d:", dst, dst, imm)
	}

	// XORI: imm[11:0] rs1 100 rd 0010011
	instr := uint32(0x13) |
		(4 << 12) | // funct3 = 100 (XORI)
		(uint32(imm&0xFFF) << 20) | // imm[11:0]
		(uint32(dstReg.Encoding&31) << 15) | // rs1 (same as rd)
		(uint32(dstReg.Encoding&31) << 7) // rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// NotReg generates NOT dst (dst = ~dst) - one's complement
func (o *Out) NotReg(dst string) {
	switch o.machine {
	case MachineX86_64:
		o.notX86Reg(dst)
	case MachineARM64:
		o.notARM64Reg(dst)
	case MachineRiscv64:
		o.notRISCVReg(dst)
	}
}

// x86-64 NOT (one's complement negation)
func (o *Out) notX86Reg(dst string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	if !dstOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "not %s:", dst)
	}

	// REX prefix for 64-bit operation
	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01 // REX.B
	}
	o.Write(rex)

	// NOT opcode (0xF7 /2 for r/m64)
	o.Write(0xF7)

	// ModR/M: 11 (register direct) | opcode extension /2 | r/m (dst)
	modrm := uint8(0xD0) | (dstReg.Encoding & 7) // 11 010 xxx
	o.Write(modrm)

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// ARM64 MVN (NOT) - bitwise NOT using MVN (Move NOT)
func (o *Out) notARM64Reg(dst string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	if !dstOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "mvn %s, %s:", dst, dst)
	}

	// MVN Xd, Xm (ORR Xd, XZR, Xm, LSL #0, inverted)
	// This is equivalent to: NOT Xd = ORN Xd, XZR, Xd
	// Format: sf 1 01010 shift 1 Rm imm6 Rn Rd
	// ORN (OR NOT): sf=1, shift=00, N=1 (inverted), Rm=src, Rn=XZR(31), Rd=dst
	instr := uint32(0xAA200000) |
		(uint32(dstReg.Encoding&31) << 16) | // Rm (source, same as dst)
		(uint32(31) << 5) | // Rn = XZR (zero register)
		uint32(dstReg.Encoding&31) // Rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}

// RISC-V NOT (using XORI with -1)
func (o *Out) notRISCVReg(dst string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	if !dstOk {
		return
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "not %s:", dst)
	}

	// RISC-V doesn't have a direct NOT instruction
	// Use XORI dst, dst, -1 (XOR with all 1s)
	// XORI: imm[11:0] rs1 100 rd 0010011
	instr := uint32(0x13) |
		(4 << 12) | // funct3 = 100 (XORI)
		(uint32(0xFFF) << 20) | // imm = -1 (all 1s in 12 bits)
		(uint32(dstReg.Encoding&31) << 15) | // rs1 (same as rd)
		(uint32(dstReg.Encoding&31) << 7) // rd

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
}
