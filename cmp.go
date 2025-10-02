package main

import (
	"fmt"
	"os"
)

// CMP instruction implementation for all architectures
// This is fundamental for implementing the Flap language's comparison operators:
//   - Pattern matching guards: n <= 1 -> 1
//   - Filter expressions: [x in rest]{x < pivot}
//   - Loop conditions: @ entity in entities{health > 0}
//   - Error guards: size > 0 or! "invalid size"
//   - Boolean comparisons: ==, !=, >=, <=, >, <

// CmpRegToReg generates a comparison instruction between two registers
// This sets flags that can be used by conditional branches
// Essential for implementing Flap's comparison operators: >=, <=, >, <, ==, !=
func (o *Out) CmpRegToReg(src1, src2 string) {
	switch o.machine {
	case MachineX86_64:
		o.cmpX86RegToReg(src1, src2)
	case MachineARM64:
		o.cmpARM64RegToReg(src1, src2)
	case MachineRiscv64:
		o.cmpRISCVRegToReg(src1, src2)
	}
}

// CmpRegToImm generates a comparison between a register and an immediate value
// Used for constant comparisons like: x > 0, x <= 1, etc.
func (o *Out) CmpRegToImm(reg string, imm int64) {
	switch o.machine {
	case MachineX86_64:
		o.cmpX86RegToImm(reg, imm)
	case MachineARM64:
		o.cmpARM64RegToImm(reg, imm)
	case MachineRiscv64:
		o.cmpRISCVRegToImm(reg, imm)
	}
}

// x86-64 CMP instruction: CMP src2, src1 (computes src1 - src2 and sets flags)
func (o *Out) cmpX86RegToReg(src1, src2 string) {
	src1Reg, src1Ok := GetRegister(o.machine, src1)
	src2Reg, src2Ok := GetRegister(o.machine, src2)
	if !src1Ok || !src2Ok {
		return
	}

	fmt.Fprintf(os.Stderr, "cmp %s, %s:", src1, src2)

	// REX prefix for 64-bit operation
	rex := uint8(0x48)
	if (src1Reg.Encoding & 8) != 0 {
		rex |= 0x01 // REX.B
	}
	if (src2Reg.Encoding & 8) != 0 {
		rex |= 0x04 // REX.R
	}
	o.Write(rex)

	// CMP opcode (0x39 for r/m64, r64)
	o.Write(0x39)

	// ModR/M byte: 11 (register direct) | reg (src2) | r/m (src1)
	modrm := uint8(0xC0) | ((src2Reg.Encoding & 7) << 3) | (src1Reg.Encoding & 7)
	o.Write(modrm)

	fmt.Fprintln(os.Stderr)
}

// x86-64 CMP with immediate: CMP reg, imm
func (o *Out) cmpX86RegToImm(reg string, imm int64) {
	regInfo, regOk := GetRegister(o.machine, reg)
	if !regOk {
		return
	}

	fmt.Fprintf(os.Stderr, "cmp %s, %d:", reg, imm)

	// REX prefix for 64-bit operation
	rex := uint8(0x48)
	if (regInfo.Encoding & 8) != 0 {
		rex |= 0x01 // REX.B
	}
	o.Write(rex)

	// Check if immediate fits in 8 bits for shorter encoding
	if imm >= -128 && imm <= 127 {
		// CMP r/m64, imm8 (opcode 0x83 /7)
		o.Write(0x83)
		modrm := uint8(0xF8) | (regInfo.Encoding & 7) // ModR/M: 11 111 reg
		o.Write(modrm)
		o.Write(uint8(imm & 0xFF))
	} else {
		// CMP r/m64, imm32 (opcode 0x81 /7)
		o.Write(0x81)
		modrm := uint8(0xF8) | (regInfo.Encoding & 7) // ModR/M: 11 111 reg
		o.Write(modrm)

		// Write 32-bit immediate (sign-extended to 64-bit)
		imm32 := uint32(imm)
		o.Write(uint8(imm32 & 0xFF))
		o.Write(uint8((imm32 >> 8) & 0xFF))
		o.Write(uint8((imm32 >> 16) & 0xFF))
		o.Write(uint8((imm32 >> 24) & 0xFF))
	}

	fmt.Fprintln(os.Stderr)
}

// ARM64 CMP instruction: CMP Xn, Xm (actually SUBS XZR, Xn, Xm)
func (o *Out) cmpARM64RegToReg(src1, src2 string) {
	src1Reg, src1Ok := GetRegister(o.machine, src1)
	src2Reg, src2Ok := GetRegister(o.machine, src2)
	if !src1Ok || !src2Ok {
		return
	}

	fmt.Fprintf(os.Stderr, "cmp %s, %s:", src1, src2)

	// CMP is encoded as SUBS XZR, Xn, Xm
	// Format: sf 1 1 01011 shift(2) 0 Rm(5) imm6(6) Rn(5) Rd(5)
	// sf=1 (64-bit), shift=00, Rd=31 (XZR)
	instr := uint32(0xEB000000) | // SUBS base opcode
		(uint32(src2Reg.Encoding&31) << 16) | // Rm (src2)
		(uint32(src1Reg.Encoding&31) << 5) |  // Rn (src1)
		31                                      // Rd = XZR (discard result)

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	fmt.Fprintln(os.Stderr)
}

// ARM64 CMP with immediate: CMP Xn, #imm (SUBS XZR, Xn, #imm)
func (o *Out) cmpARM64RegToImm(reg string, imm int64) {
	regInfo, regOk := GetRegister(o.machine, reg)
	if !regOk {
		return
	}

	fmt.Fprintf(os.Stderr, "cmp %s, #%d:", reg, imm)

	// ARM64 immediate must be 12-bit unsigned
	if imm < 0 || imm > 4095 {
		fmt.Fprintf(os.Stderr, " (immediate out of range, using 0)")
		imm = 0
	}

	// SUBS XZR, Xn, #imm
	// Format: sf 1 1 10001 shift(2) imm12(12) Rn(5) Rd(5)
	instr := uint32(0xF1000000) | // SUBS immediate base
		(uint32(imm&0xFFF) << 10) |        // imm12
		(uint32(regInfo.Encoding&31) << 5) | // Rn
		31                                    // Rd = XZR

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	fmt.Fprintln(os.Stderr)
}

// RISC-V doesn't have a direct CMP instruction
// Comparison is done with SUB and checking the result
// Or using SLT (set less than) instructions
func (o *Out) cmpRISCVRegToReg(src1, src2 string) {
	src1Reg, src1Ok := GetRegister(o.machine, src1)
	src2Reg, src2Ok := GetRegister(o.machine, src2)
	if !src1Ok || !src2Ok {
		return
	}

	fmt.Fprintf(os.Stderr, "# cmp %s, %s (sub t0, %s, %s):", src1, src2, src1, src2)

	// Use SUB t0, src1, src2 to compare (result in t0)
	// Format: funct7(7) rs2(5) rs1(5) funct3(3) rd(5) opcode(7)
	// SUB: 0100000 rs2 rs1 000 rd 0110011
	instr := uint32(0x40000033) |
		(uint32(src2Reg.Encoding&31) << 20) | // rs2
		(uint32(src1Reg.Encoding&31) << 15) | // rs1
		(5 << 7)                               // rd = t0 (x5)

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	fmt.Fprintln(os.Stderr)
}

// RISC-V CMP with immediate using ADDI with negated immediate
func (o *Out) cmpRISCVRegToImm(reg string, imm int64) {
	regInfo, regOk := GetRegister(o.machine, reg)
	if !regOk {
		return
	}

	fmt.Fprintf(os.Stderr, "# cmp %s, %d (addi t0, %s, %d):", reg, imm, reg, -imm)

	// Use ADDI t0, reg, -imm to compare
	// Format: imm[11:0] rs1 000 rd 0010011
	negImm := -imm
	if negImm < -2048 || negImm > 2047 {
		fmt.Fprintf(os.Stderr, " (immediate out of range)")
		negImm = 0
	}

	instr := uint32(0x13) |
		(uint32(negImm&0xFFF) << 20) |        // imm[11:0]
		(uint32(regInfo.Encoding&31) << 15) | // rs1
		(5 << 7)                               // rd = t0 (x5)

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	fmt.Fprintln(os.Stderr)
}
