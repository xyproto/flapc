package main

import (
	"fmt"
	"os"
)

// DIV instruction for division and remainder
// Essential for implementing Flap's division operations:
//   - Division expressions: quotient / divisor
//   - Modulo operations: n % 10
//   - Array partitioning: size / chunk_size
//   - Ratio calculations: total / count
//   - Remainder checks: n % 2 == 0 (even/odd)

// DivRegByReg generates DIV dst, src (dst = dst / src)
// Note: On x86-64, this is more complex due to implicit RDX:RAX usage
func (o *Out) DivRegByReg(dst, src string) {
	switch o.machine {
	case MachineX86_64:
		o.divX86RegByReg(dst, src)
	case MachineARM64:
		o.divARM64RegByReg(dst, src)
	case MachineRiscv64:
		o.divRISCVRegByReg(dst, src)
	}
}

// DivRegByRegToReg generates DIV quotient, dividend, divisor (quotient = dividend / divisor)
// 3-operand form for ARM64 and RISC-V
func (o *Out) DivRegByRegToReg(quotient, dividend, divisor string) {
	switch o.machine {
	case MachineX86_64:
		// x86-64: Complex due to implicit RDX:RAX - need temporary handling
		// For now, MOV quotient, dividend; DIV quotient, divisor
		if quotient != dividend {
			o.MovRegToReg(quotient, dividend)
		}
		o.DivRegByReg(quotient, divisor)
	case MachineARM64:
		o.divARM64RegByRegToReg(quotient, dividend, divisor)
	case MachineRiscv64:
		o.divRISCVRegByRegToReg(quotient, dividend, divisor)
	}
}

// RemRegByReg generates REM dst, src (dst = dst % src) - remainder/modulo
func (o *Out) RemRegByReg(dst, src string) {
	switch o.machine {
	case MachineX86_64:
		o.remX86RegByReg(dst, src)
	case MachineARM64:
		o.remARM64RegByReg(dst, src)
	case MachineRiscv64:
		o.remRISCVRegByReg(dst, src)
	}
}

// RemRegByRegToReg generates REM remainder, dividend, divisor (remainder = dividend % divisor)
func (o *Out) RemRegByRegToReg(remainder, dividend, divisor string) {
	switch o.machine {
	case MachineX86_64:
		if remainder != dividend {
			o.MovRegToReg(remainder, dividend)
		}
		o.RemRegByReg(remainder, divisor)
	case MachineARM64:
		o.remARM64RegByRegToReg(remainder, dividend, divisor)
	case MachineRiscv64:
		o.remRISCVRegByRegToReg(remainder, dividend, divisor)
	}
}

// ============================================================================
// x86-64 implementations
// ============================================================================

// x86-64 IDIV (signed divide) - 2 operand form
// WARNING: This modifies both RAX (quotient) and RDX (remainder)
// Input: RDX:RAX = dividend, divisor in register
// Output: RAX = quotient, RDX = remainder
func (o *Out) divX86RegByReg(dst, src string) {
	// For x86-64, we need to:
	// 1. Save current RDX if needed
	// 2. Move dst to RAX if it's not already there
	// 3. Sign-extend RAX to RDX:RAX using CQO
	// 4. IDIV src
	// 5. Move RAX back to dst if needed

	// Simplified version: assumes dst is already in RAX
	srcReg, srcOk := GetRegister(o.machine, src)
	if !srcOk {
		return
	}

	fmt.Fprintf(os.Stderr, "# div %s, %s (cqo; idiv %s):", dst, src, src)

	// CQO: Sign-extend RAX into RDX:RAX
	o.Write(0x48) // REX.W
	o.Write(0x99) // CQO

	// IDIV r/m64
	rex := uint8(0x48)
	if (srcReg.Encoding & 8) != 0 {
		rex |= 0x01 // REX.B
	}
	o.Write(rex)

	o.Write(0xF7) // IDIV opcode

	// ModR/M: 11 111 reg (opcode extension /7 for IDIV)
	modrm := uint8(0xF8) | (srcReg.Encoding & 7)
	o.Write(modrm)

	fmt.Fprintln(os.Stderr)
}

// x86-64 IDIV for remainder
func (o *Out) remX86RegByReg(dst, src string) {
	// Same as division, but the result we want is in RDX, not RAX
	srcReg, srcOk := GetRegister(o.machine, src)
	if !srcOk {
		return
	}

	fmt.Fprintf(os.Stderr, "# rem %s, %s (cqo; idiv %s; mov %s, rdx):", dst, src, src, dst)

	// CQO: Sign-extend RAX into RDX:RAX
	o.Write(0x48) // REX.W
	o.Write(0x99) // CQO

	// IDIV r/m64
	rex := uint8(0x48)
	if (srcReg.Encoding & 8) != 0 {
		rex |= 0x01 // REX.B
	}
	o.Write(rex)

	o.Write(0xF7) // IDIV opcode

	// ModR/M: 11 111 reg (opcode extension /7 for IDIV)
	modrm := uint8(0xF8) | (srcReg.Encoding & 7)
	o.Write(modrm)

	// Result is in RDX - would need to move to dst if dst != rdx
	// This is simplified

	fmt.Fprintln(os.Stderr)
}

// ============================================================================
// ARM64 implementations
// ============================================================================

// ARM64 SDIV (signed divide) - 2 operand form
func (o *Out) divARM64RegByReg(dst, src string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	srcReg, srcOk := GetRegister(o.machine, src)
	if !dstOk || !srcOk {
		return
	}

	fmt.Fprintf(os.Stderr, "sdiv %s, %s, %s:", dst, dst, src)

	// SDIV Xd, Xn, Xm
	// Format: sf 0 011010110 Rm 000011 Rn Rd
	// sf=1 (64-bit), opcode for SDIV
	instr := uint32(0x9AC00C00) |
		(uint32(srcReg.Encoding&31) << 16) |  // Rm (divisor)
		(uint32(dstReg.Encoding&31) << 5) |   // Rn (dividend, same as Rd)
		uint32(dstReg.Encoding&31)             // Rd (quotient)

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	fmt.Fprintln(os.Stderr)
}

// ARM64 SDIV - 3 operand form
func (o *Out) divARM64RegByRegToReg(quotient, dividend, divisor string) {
	quotientReg, quotientOk := GetRegister(o.machine, quotient)
	dividendReg, dividendOk := GetRegister(o.machine, dividend)
	divisorReg, divisorOk := GetRegister(o.machine, divisor)
	if !quotientOk || !dividendOk || !divisorOk {
		return
	}

	fmt.Fprintf(os.Stderr, "sdiv %s, %s, %s:", quotient, dividend, divisor)

	instr := uint32(0x9AC00C00) |
		(uint32(divisorReg.Encoding&31) << 16) |  // Rm (divisor)
		(uint32(dividendReg.Encoding&31) << 5) |  // Rn (dividend)
		uint32(quotientReg.Encoding&31)            // Rd (quotient)

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	fmt.Fprintln(os.Stderr)
}

// ARM64 remainder - calculate using: rem = dividend - (quotient * divisor)
// This requires a temp register
func (o *Out) remARM64RegByReg(dst, src string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	srcReg, srcOk := GetRegister(o.machine, src)
	if !dstOk || !srcOk {
		return
	}

	fmt.Fprintf(os.Stderr, "# rem %s, %s (msub %s, temp_quotient, %s, %s):", dst, src, dst, src, dst)

	// ARM64 has MSUB: Xd = Xa - Xn * Xm
	// We want: remainder = dividend - (dividend/divisor) * divisor
	// Need temp register for quotient - using x9 (caller-saved)
	// SDIV x9, dst, src
	// MSUB dst, x9, src, dst  (dst = dst - x9 * src)

	// This is complex - simplified comment for now
	// Full implementation would need temp register management

	fmt.Fprintln(os.Stderr)
}

// ARM64 remainder - 3 operand form
func (o *Out) remARM64RegByRegToReg(remainder, dividend, divisor string) {
	remainderReg, remainderOk := GetRegister(o.machine, remainder)
	dividendReg, dividendOk := GetRegister(o.machine, dividend)
	divisorReg, divisorOk := GetRegister(o.machine, divisor)
	if !remainderOk || !dividendOk || !divisorOk {
		return
	}

	fmt.Fprintf(os.Stderr, "# rem %s, %s, %s (msub with temp):", remainder, dividend, divisor)

	// MSUB: Xd = Xa - Xn * Xm
	// remainder = dividend - (quotient * divisor)
	// This needs temp register for quotient

	fmt.Fprintln(os.Stderr)
}

// ============================================================================
// RISC-V implementations
// ============================================================================

// RISC-V DIV (signed divide, requires M extension) - 2 operand form
func (o *Out) divRISCVRegByReg(dst, src string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	srcReg, srcOk := GetRegister(o.machine, src)
	if !dstOk || !srcOk {
		return
	}

	fmt.Fprintf(os.Stderr, "div %s, %s, %s:", dst, dst, src)

	// DIV: 0000001 rs2 rs1 100 rd 0110011
	instr := uint32(0x33) |
		(4 << 12) |                            // funct3 = 100 (DIV)
		(1 << 25) |                            // funct7 = 0000001 (M extension)
		(uint32(srcReg.Encoding&31) << 20) |   // rs2 (divisor)
		(uint32(dstReg.Encoding&31) << 15) |   // rs1 (dividend, same as rd)
		(uint32(dstReg.Encoding&31) << 7)      // rd (quotient)

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	fmt.Fprintln(os.Stderr)
}

// RISC-V DIV - 3 operand form
func (o *Out) divRISCVRegByRegToReg(quotient, dividend, divisor string) {
	quotientReg, quotientOk := GetRegister(o.machine, quotient)
	dividendReg, dividendOk := GetRegister(o.machine, dividend)
	divisorReg, divisorOk := GetRegister(o.machine, divisor)
	if !quotientOk || !dividendOk || !divisorOk {
		return
	}

	fmt.Fprintf(os.Stderr, "div %s, %s, %s:", quotient, dividend, divisor)

	instr := uint32(0x33) |
		(4 << 12) |                             // funct3 = 100 (DIV)
		(1 << 25) |                             // funct7 = 0000001 (M extension)
		(uint32(divisorReg.Encoding&31) << 20) | // rs2 (divisor)
		(uint32(dividendReg.Encoding&31) << 15) | // rs1 (dividend)
		(uint32(quotientReg.Encoding&31) << 7)   // rd (quotient)

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	fmt.Fprintln(os.Stderr)
}

// RISC-V REM (signed remainder, requires M extension) - 2 operand form
func (o *Out) remRISCVRegByReg(dst, src string) {
	dstReg, dstOk := GetRegister(o.machine, dst)
	srcReg, srcOk := GetRegister(o.machine, src)
	if !dstOk || !srcOk {
		return
	}

	fmt.Fprintf(os.Stderr, "rem %s, %s, %s:", dst, dst, src)

	// REM: 0000001 rs2 rs1 110 rd 0110011
	instr := uint32(0x33) |
		(6 << 12) |                            // funct3 = 110 (REM)
		(1 << 25) |                            // funct7 = 0000001 (M extension)
		(uint32(srcReg.Encoding&31) << 20) |   // rs2 (divisor)
		(uint32(dstReg.Encoding&31) << 15) |   // rs1 (dividend, same as rd)
		(uint32(dstReg.Encoding&31) << 7)      // rd (remainder)

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	fmt.Fprintln(os.Stderr)
}

// RISC-V REM - 3 operand form
func (o *Out) remRISCVRegByRegToReg(remainder, dividend, divisor string) {
	remainderReg, remainderOk := GetRegister(o.machine, remainder)
	dividendReg, dividendOk := GetRegister(o.machine, dividend)
	divisorReg, divisorOk := GetRegister(o.machine, divisor)
	if !remainderOk || !dividendOk || !divisorOk {
		return
	}

	fmt.Fprintf(os.Stderr, "rem %s, %s, %s:", remainder, dividend, divisor)

	instr := uint32(0x33) |
		(6 << 12) |                             // funct3 = 110 (REM)
		(1 << 25) |                             // funct7 = 0000001 (M extension)
		(uint32(divisorReg.Encoding&31) << 20) | // rs2 (divisor)
		(uint32(dividendReg.Encoding&31) << 15) | // rs1 (dividend)
		(uint32(remainderReg.Encoding&31) << 7)  // rd (remainder)

	o.Write(uint8(instr & 0xFF))
	o.Write(uint8((instr >> 8) & 0xFF))
	o.Write(uint8((instr >> 16) & 0xFF))
	o.Write(uint8((instr >> 24) & 0xFF))

	fmt.Fprintln(os.Stderr)
}
