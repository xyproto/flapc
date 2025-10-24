package main

import (
	"strconv"
)

// RISCV64Backend implements the CodeGenerator interface for RISC-V 64-bit architecture
type RISCV64Backend struct {
	writer Writer
	eb     *ExecutableBuilder
}

// NewRISCV64Backend creates a new RISC-V 64-bit code generator backend
func NewRISCV64Backend(writer Writer, eb *ExecutableBuilder) *RISCV64Backend {
	return &RISCV64Backend{
		writer: writer,
		eb:     eb,
	}
}

func (r *RISCV64Backend) write(b uint8) {
	r.writer.(*BufferWrapper).Write(b)
}

func (r *RISCV64Backend) writeUnsigned(i uint) {
	r.writer.(*BufferWrapper).WriteUnsigned(i)
}

func (r *RISCV64Backend) emit(bytes []byte) {
	for _, b := range bytes {
		r.write(b)
	}
}

func (r *RISCV64Backend) writeInstruction(instr uint32) {
	// RISC-V instructions are little-endian
	r.write(uint8(instr & 0xFF))
	r.write(uint8((instr >> 8) & 0xFF))
	r.write(uint8((instr >> 16) & 0xFF))
	r.write(uint8((instr >> 24) & 0xFF))
}

// ===== Data Movement =====

func (r *RISCV64Backend) MovRegToReg(dst, src string) {
	dstReg, dstOk := riscvRegisters[dst]
	srcReg, srcOk := riscvRegisters[src]
	if !dstOk || !srcOk {
		return
	}

	// RISC-V MV is implemented as ADDI rd, rs1, 0
	// Format: imm[11:0] | rs1 | 000 | rd | 0010011
	instr := uint32(0x13) | // opcode for ADDI
		(uint32(dstReg.Encoding&31) << 7) | // rd
		(0 << 12) | // funct3 = 000 for ADDI
		(uint32(srcReg.Encoding&31) << 15) | // rs1
		(0 << 20) // immediate = 0

	r.writeInstruction(instr)
}

func (r *RISCV64Backend) MovImmToReg(dst, imm string) {
	dstReg, dstOk := riscvRegisters[dst]
	if !dstOk {
		return
	}

	// Parse immediate value
	var immVal int64
	if val, err := strconv.ParseInt(imm, 0, 64); err == nil {
		immVal = val
	}

	// For simplicity, use ADDI rd, x0, imm for small immediates
	if immVal >= -2048 && immVal <= 2047 {
		// ADDI rd, x0, imm
		instr := uint32(0x13) | // opcode
			(uint32(dstReg.Encoding&31) << 7) | // rd
			(0 << 12) | // funct3 = 000
			(0 << 15) | // rs1 = x0
			(uint32((immVal & 0xFFF)) << 20) // immediate

		r.writeInstruction(instr)
	} else {
		// For larger immediates, would need LUI + ADDI sequence
		// For now, just use ADDI with truncated immediate
		immVal = immVal & 0xFFF
		instr := uint32(0x13) |
			(uint32(dstReg.Encoding&31) << 7) |
			(0 << 12) |
			(0 << 15) |
			(uint32(immVal&0xFFF) << 20)

		r.writeInstruction(instr)
	}
}

func (r *RISCV64Backend) MovMemToReg(dst, symbol string, offset int32) {
	compilerError("RISCV64Backend.MovMemToReg not implemented")
}

func (r *RISCV64Backend) MovRegToMem(src, symbol string, offset int32) {
	compilerError("RISCV64Backend.MovRegToMem not implemented")
}

// ===== Integer Arithmetic =====

func (r *RISCV64Backend) AddRegToReg(dst, src string) {
	dstReg, dstOk := riscvRegisters[dst]
	srcReg, srcOk := riscvRegisters[src]
	if !dstOk || !srcOk {
		return
	}

	// ADD rd, rd, rs2 (rd = rd + rs2)
	// Format: 0000000 rs2 rs1 000 rd 0110011
	instr := uint32(0x33) | // opcode for ADD
		(uint32(dstReg.Encoding&31) << 7) | // rd
		(0 << 12) | // funct3 = 000
		(uint32(dstReg.Encoding&31) << 15) | // rs1 = rd
		(uint32(srcReg.Encoding&31) << 20) | // rs2
		(0 << 25) // funct7 = 0000000

	r.writeInstruction(instr)
}

func (r *RISCV64Backend) AddImmToReg(dst string, imm int64) {
	dstReg, dstOk := riscvRegisters[dst]
	if !dstOk {
		return
	}

	// ADDI rd, rd, imm
	if imm < -2048 || imm > 2047 {
		imm = imm & 0xFFF
	}

	instr := uint32(0x13) |
		(uint32(dstReg.Encoding&31) << 7) |
		(0 << 12) |
		(uint32(dstReg.Encoding&31) << 15) |
		(uint32(imm&0xFFF) << 20)

	r.writeInstruction(instr)
}

func (r *RISCV64Backend) SubRegToReg(dst, src string) {
	dstReg, dstOk := riscvRegisters[dst]
	srcReg, srcOk := riscvRegisters[src]
	if !dstOk || !srcOk {
		return
	}

	// SUB rd, rd, rs2
	// Format: 0100000 rs2 rs1 000 rd 0110011
	instr := uint32(0x33) |
		(uint32(dstReg.Encoding&31) << 7) |
		(0 << 12) |
		(uint32(dstReg.Encoding&31) << 15) |
		(uint32(srcReg.Encoding&31) << 20) |
		(0x20 << 25) // funct7 = 0100000

	r.writeInstruction(instr)
}

func (r *RISCV64Backend) SubImmFromReg(dst string, imm int64) {
	// SUB rd, rd, imm is done as ADDI rd, rd, -imm
	r.AddImmToReg(dst, -imm)
}

func (r *RISCV64Backend) MulRegToReg(dst, src string) {
	dstReg, dstOk := riscvRegisters[dst]
	srcReg, srcOk := riscvRegisters[src]
	if !dstOk || !srcOk {
		return
	}

	// MUL rd, rd, rs2 (requires M extension)
	// Format: 0000001 rs2 rs1 000 rd 0110011
	instr := uint32(0x33) |
		(uint32(dstReg.Encoding&31) << 7) |
		(0 << 12) |
		(uint32(dstReg.Encoding&31) << 15) |
		(uint32(srcReg.Encoding&31) << 20) |
		(1 << 25) // funct7 = 0000001 (M extension)

	r.writeInstruction(instr)
}

func (r *RISCV64Backend) DivRegToReg(dst, src string) {
	dstReg, dstOk := riscvRegisters[dst]
	srcReg, srcOk := riscvRegisters[src]
	if !dstOk || !srcOk {
		return
	}

	// DIV rd, rd, rs2 (requires M extension)
	// Format: 0000001 rs2 rs1 100 rd 0110011
	instr := uint32(0x33) |
		(uint32(dstReg.Encoding&31) << 7) |
		(4 << 12) | // funct3 = 100 for DIV
		(uint32(dstReg.Encoding&31) << 15) |
		(uint32(srcReg.Encoding&31) << 20) |
		(1 << 25) // funct7 = 0000001

	r.writeInstruction(instr)
}

func (r *RISCV64Backend) IncReg(dst string) {
	// ADDI rd, rd, 1
	r.AddImmToReg(dst, 1)
}

func (r *RISCV64Backend) DecReg(dst string) {
	// ADDI rd, rd, -1
	r.AddImmToReg(dst, -1)
}

func (r *RISCV64Backend) NegReg(dst string) {
	dstReg, dstOk := riscvRegisters[dst]
	if !dstOk {
		return
	}

	// NEG rd, rd (SUB rd, x0, rd)
	instr := uint32(0x33) |
		(uint32(dstReg.Encoding&31) << 7) |
		(0 << 12) |
		(0 << 15) | // rs1 = x0
		(uint32(dstReg.Encoding&31) << 20) |
		(0x20 << 25) // funct7 = 0100000 for SUB

	r.writeInstruction(instr)
}

// ===== Bitwise Operations =====

func (r *RISCV64Backend) XorRegWithReg(dst, src string) {
	dstReg, dstOk := riscvRegisters[dst]
	srcReg, srcOk := riscvRegisters[src]
	if !dstOk || !srcOk {
		return
	}

	// XOR rd, rd, rs2
	// Format: 0000000 rs2 rs1 100 rd 0110011
	instr := uint32(0x33) |
		(uint32(dstReg.Encoding&31) << 7) |
		(4 << 12) | // funct3 = 100 for XOR
		(uint32(dstReg.Encoding&31) << 15) |
		(uint32(srcReg.Encoding&31) << 20) |
		(0 << 25)

	r.writeInstruction(instr)
}

func (r *RISCV64Backend) XorRegWithImm(dst string, imm int64) {
	dstReg, dstOk := riscvRegisters[dst]
	if !dstOk {
		return
	}

	// XORI rd, rd, imm
	if imm < -2048 || imm > 2047 {
		imm = imm & 0xFFF
	}

	instr := uint32(0x13) |
		(uint32(dstReg.Encoding&31) << 7) |
		(4 << 12) | // funct3 = 100 for XORI
		(uint32(dstReg.Encoding&31) << 15) |
		(uint32(imm&0xFFF) << 20)

	r.writeInstruction(instr)
}

func (r *RISCV64Backend) AndRegWithReg(dst, src string) {
	dstReg, dstOk := riscvRegisters[dst]
	srcReg, srcOk := riscvRegisters[src]
	if !dstOk || !srcOk {
		return
	}

	// AND rd, rd, rs2
	// Format: 0000000 rs2 rs1 111 rd 0110011
	instr := uint32(0x33) |
		(uint32(dstReg.Encoding&31) << 7) |
		(7 << 12) | // funct3 = 111 for AND
		(uint32(dstReg.Encoding&31) << 15) |
		(uint32(srcReg.Encoding&31) << 20) |
		(0 << 25)

	r.writeInstruction(instr)
}

func (r *RISCV64Backend) OrRegWithReg(dst, src string) {
	dstReg, dstOk := riscvRegisters[dst]
	srcReg, srcOk := riscvRegisters[src]
	if !dstOk || !srcOk {
		return
	}

	// OR rd, rd, rs2
	// Format: 0000000 rs2 rs1 110 rd 0110011
	instr := uint32(0x33) |
		(uint32(dstReg.Encoding&31) << 7) |
		(6 << 12) | // funct3 = 110 for OR
		(uint32(dstReg.Encoding&31) << 15) |
		(uint32(srcReg.Encoding&31) << 20) |
		(0 << 25)

	r.writeInstruction(instr)
}

func (r *RISCV64Backend) NotReg(dst string) {
	// NOT rd is implemented as XORI rd, rd, -1
	r.XorRegWithImm(dst, -1)
}

// ===== Stack Operations =====

func (r *RISCV64Backend) PushReg(reg string) {
	compilerError("RISCV64Backend.PushReg not implemented (RISC-V doesn't have PUSH)")
}

func (r *RISCV64Backend) PopReg(reg string) {
	compilerError("RISCV64Backend.PopReg not implemented (RISC-V doesn't have POP)")
}

// ===== Control Flow =====

func (r *RISCV64Backend) JumpConditional(condition JumpCondition, offset int32) {
	// RISC-V conditional branches use comparison result in t0 (x5)
	// This is a simplified implementation - proper implementation would need
	// to track which register holds the comparison result

	// For now, use BEQ/BNE/BLT/BGE with x0 (zero register)
	// This assumes the comparison result is in a specific register

	var funct3 uint32
	switch condition {
	case JumpEqual:
		funct3 = 0x0 // BEQ
	case JumpNotEqual:
		funct3 = 0x1 // BNE
	case JumpLess:
		funct3 = 0x4 // BLT
	case JumpGreaterOrEqual:
		funct3 = 0x5 // BGE
	default:
		compilerError("Unsupported jump condition for RISC-V: %v", condition)
		return
	}

	// BEQ rs1, rs2, offset (comparing t0 with x0)
	// Format: imm[12] imm[10:5] rs2 rs1 funct3 imm[4:1] imm[11] 1100011
	if offset < -4096 || offset > 4095 {
		compilerError("RISC-V branch offset out of range: %d", offset)
		return
	}

	imm := uint32(offset)
	instr := uint32(0x63) | // opcode for branch
		(funct3 << 12) |
		(5 << 15) | // rs1 = t0 (x5)
		(0 << 20) | // rs2 = x0
		((imm & 0x800) << 20) | // imm[11]
		((imm & 0x1E) << 7) | // imm[4:1]
		((imm & 0x7E0) << 20) | // imm[10:5]
		((imm & 0x1000) << 19) // imm[12]

	r.writeInstruction(instr)
}

func (r *RISCV64Backend) JumpUnconditional(offset int32) {
	// JAL x0, offset (jump and link, discarding return address)
	// Format: imm[20] imm[10:1] imm[11] imm[19:12] rd 1101111
	if offset < -1048576 || offset > 1048575 {
		compilerError("RISC-V JAL offset out of range: %d", offset)
		return
	}

	imm := uint32(offset)
	instr := uint32(0x6F) | // opcode for JAL
		(0 << 7) | // rd = x0 (discard return address)
		((imm & 0xFF000) << 0) | // imm[19:12]
		((imm & 0x800) << 9) | // imm[11]
		((imm & 0x7FE) << 20) | // imm[10:1]
		((imm & 0x100000) << 11) // imm[20]

	r.writeInstruction(instr)
}

func (r *RISCV64Backend) CallSymbol(symbol string) {
	// JAL ra, offset (will be patched)
	r.write(0x6F) // Placeholder - will be patched

	callPos := r.eb.text.Len()
	r.writeUnsigned(0x000000) // 3 more bytes

	r.eb.callPatches = append(r.eb.callPatches, CallPatch{
		position:   callPos - 1,
		targetName: symbol,
	})
}

func (r *RISCV64Backend) CallRelative(offset int32) {
	// JAL ra, offset
	if offset < -1048576 || offset > 1048575 {
		compilerError("RISC-V JAL offset out of range: %d", offset)
		return
	}

	imm := uint32(offset)
	instr := uint32(0x6F) |
		(1 << 7) | // rd = ra (x1)
		((imm & 0xFF000) << 0) |
		((imm & 0x800) << 9) |
		((imm & 0x7FE) << 20) |
		((imm & 0x100000) << 11)

	r.writeInstruction(instr)
}

func (r *RISCV64Backend) CallRegister(reg string) {
	regInfo, regOk := riscvRegisters[reg]
	if !regOk {
		return
	}

	// JALR ra, 0(reg)
	// Format: imm[11:0] rs1 000 rd 1100111
	instr := uint32(0x67) |
		(1 << 7) | // rd = ra (x1)
		(0 << 12) | // funct3 = 000
		(uint32(regInfo.Encoding&31) << 15) | // rs1
		(0 << 20) // imm = 0

	r.writeInstruction(instr)
}

func (r *RISCV64Backend) Ret() {
	// RET is JALR x0, ra, 0
	// Format: imm[11:0] rs1 000 rd 1100111
	instr := uint32(0x67) |
		(0 << 7) | // rd = x0 (zero)
		(0 << 12) | // funct3 = 000
		(1 << 15) | // rs1 = ra (x1)
		(0 << 20) // imm = 0

	r.writeInstruction(instr)
}

// ===== Comparisons =====

func (r *RISCV64Backend) CmpRegToReg(reg1, reg2 string) {
	reg1Info, reg1Ok := riscvRegisters[reg1]
	reg2Info, reg2Ok := riscvRegisters[reg2]
	if !reg1Ok || !reg2Ok {
		return
	}

	// Use SUB t0, reg1, reg2 to compare (result in t0)
	// Format: 0100000 rs2 rs1 000 rd 0110011
	instr := uint32(0x40000033) |
		(5 << 7) | // rd = t0 (x5)
		(0 << 12) | // funct3 = 000
		(uint32(reg1Info.Encoding&31) << 15) | // rs1
		(uint32(reg2Info.Encoding&31) << 20) // rs2

	r.writeInstruction(instr)
}

func (r *RISCV64Backend) CmpRegToImm(reg string, imm int64) {
	regInfo, regOk := riscvRegisters[reg]
	if !regOk {
		return
	}

	// Use ADDI t0, reg, -imm to compare
	negImm := -imm
	if negImm < -2048 || negImm > 2047 {
		negImm = 0
	}

	instr := uint32(0x13) |
		(5 << 7) | // rd = t0 (x5)
		(0 << 12) | // funct3 = 000
		(uint32(regInfo.Encoding&31) << 15) | // rs1
		(uint32(negImm&0xFFF) << 20) // imm[11:0]

	r.writeInstruction(instr)
}

// ===== Address Calculation =====

func (r *RISCV64Backend) LeaSymbolToReg(dst, symbol string) {
	compilerError("RISCV64Backend.LeaSymbolToReg not implemented")
}

func (r *RISCV64Backend) LeaImmToReg(dst, base string, offset int32) {
	// ADDI rd, base, offset
	dstReg, dstOk := riscvRegisters[dst]
	baseReg, baseOk := riscvRegisters[base]
	if !dstOk || !baseOk {
		return
	}

	if offset < -2048 || offset > 2047 {
		offset = offset & 0xFFF
	}

	instr := uint32(0x13) |
		(uint32(dstReg.Encoding&31) << 7) |
		(0 << 12) |
		(uint32(baseReg.Encoding&31) << 15) |
		(uint32(offset&0xFFF) << 20)

	r.writeInstruction(instr)
}

// ===== Floating Point (SIMD) =====

func (r *RISCV64Backend) MovXmmToMem(src, base string, offset int32) {
	compilerError("RISCV64Backend.MovXmmToMem not implemented")
}

func (r *RISCV64Backend) MovMemToXmm(dst, base string, offset int32) {
	compilerError("RISCV64Backend.MovMemToXmm not implemented")
}

func (r *RISCV64Backend) MovRegToXmm(dst, src string) {
	compilerError("RISCV64Backend.MovRegToXmm not implemented")
}

func (r *RISCV64Backend) MovXmmToReg(dst, src string) {
	compilerError("RISCV64Backend.MovXmmToReg not implemented")
}

func (r *RISCV64Backend) Cvtsi2sd(dst, src string) {
	compilerError("RISCV64Backend.Cvtsi2sd not implemented")
}

func (r *RISCV64Backend) Cvttsd2si(dst, src string) {
	compilerError("RISCV64Backend.Cvttsd2si not implemented")
}

func (r *RISCV64Backend) AddpdXmm(dst, src string) {
	compilerError("RISCV64Backend.AddpdXmm not implemented")
}

func (r *RISCV64Backend) SubpdXmm(dst, src string) {
	compilerError("RISCV64Backend.SubpdXmm not implemented")
}

func (r *RISCV64Backend) MulpdXmm(dst, src string) {
	compilerError("RISCV64Backend.MulpdXmm not implemented")
}

func (r *RISCV64Backend) DivpdXmm(dst, src string) {
	compilerError("RISCV64Backend.DivpdXmm not implemented")
}

func (r *RISCV64Backend) Ucomisd(reg1, reg2 string) {
	compilerError("RISCV64Backend.Ucomisd not implemented")
}

// ===== System Calls =====

func (r *RISCV64Backend) Syscall() {
	// ECALL
	instr := uint32(0x00000073)
	r.writeInstruction(instr)
}
