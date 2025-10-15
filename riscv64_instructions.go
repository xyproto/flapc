package main

import (
	"encoding/binary"
	"fmt"
)

// RISC-V64 instruction encoding
// RISC-V uses fixed 32-bit little-endian instructions

// RISC-V Register mapping
var riscvGPRegs = map[string]uint32{
	"zero": 0, "x0": 0,
	"ra": 1, "x1": 1,
	"sp": 2, "x2": 2,
	"gp": 3, "x3": 3,
	"tp": 4, "x4": 4,
	"t0": 5, "x5": 5,
	"t1": 6, "x6": 6,
	"t2": 7, "x7": 7,
	"s0": 8, "fp": 8, "x8": 8,
	"s1": 9, "x9": 9,
	"a0": 10, "x10": 10,
	"a1": 11, "x11": 11,
	"a2": 12, "x12": 12,
	"a3": 13, "x13": 13,
	"a4": 14, "x14": 14,
	"a5": 15, "x15": 15,
	"a6": 16, "x16": 16,
	"a7": 17, "x17": 17,
	"s2": 18, "x18": 18,
	"s3": 19, "x19": 19,
	"s4": 20, "x20": 20,
	"s5": 21, "x21": 21,
	"s6": 22, "x22": 22,
	"s7": 23, "x23": 23,
	"s8": 24, "x24": 24,
	"s9": 25, "x25": 25,
	"s10": 26, "x26": 26,
	"s11": 27, "x27": 27,
	"t3": 28, "x28": 28,
	"t4": 29, "x29": 29,
	"t5": 30, "x30": 30,
	"t6": 31, "x31": 31,
}

var riscvFPRegs = map[string]uint32{
	"ft0": 0, "f0": 0,
	"ft1": 1, "f1": 1,
	"ft2": 2, "f2": 2,
	"ft3": 3, "f3": 3,
	"ft4": 4, "f4": 4,
	"ft5": 5, "f5": 5,
	"ft6": 6, "f6": 6,
	"ft7": 7, "f7": 7,
	"fs0": 8, "f8": 8,
	"fs1": 9, "f9": 9,
	"fa0": 10, "f10": 10,
	"fa1": 11, "f11": 11,
	"fa2": 12, "f12": 12,
	"fa3": 13, "f13": 13,
	"fa4": 14, "f14": 14,
	"fa5": 15, "f15": 15,
	"fa6": 16, "f16": 16,
	"fa7": 17, "f17": 17,
	"fs2": 18, "f18": 18,
	"fs3": 19, "f19": 19,
	"fs4": 20, "f20": 20,
	"fs5": 21, "f21": 21,
	"fs6": 22, "f22": 22,
	"fs7": 23, "f23": 23,
	"fs8": 24, "f24": 24,
	"fs9": 25, "f25": 25,
	"fs10": 26, "f26": 26,
	"fs11": 27, "f27": 27,
	"ft8": 28, "f28": 28,
	"ft9": 29, "f29": 29,
	"ft10": 30, "f30": 30,
	"ft11": 31, "f31": 31,
}

// RiscvOut wraps Out for RISC-V-specific instructions
type RiscvOut struct {
	out *Out
}

// encodeInstr writes a 32-bit RISC-V instruction in little-endian format
func (r *RiscvOut) encodeInstr(instr uint32) {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], instr)
	r.out.writer.WriteBytes(buf[:])
}

// RISC-V Instruction encodings

// R-type: opcode[6:0] | rd[11:7] | funct3[14:12] | rs1[19:15] | rs2[24:20] | funct7[31:25]
func (r *RiscvOut) encodeRType(opcode, funct3, funct7 uint32, rd, rs1, rs2 uint32) uint32 {
	return opcode | (rd << 7) | (funct3 << 12) | (rs1 << 15) | (rs2 << 20) | (funct7 << 25)
}

// I-type: opcode[6:0] | rd[11:7] | funct3[14:12] | rs1[19:15] | imm[31:20]
func (r *RiscvOut) encodeIType(opcode, funct3 uint32, rd, rs1 uint32, imm int32) uint32 {
	return opcode | (rd << 7) | (funct3 << 12) | (rs1 << 15) | (uint32(imm&0xfff) << 20)
}

// S-type: opcode[6:0] | imm[11:7] | funct3[14:12] | rs1[19:15] | rs2[24:20] | imm[31:25]
func (r *RiscvOut) encodeSType(opcode, funct3 uint32, rs1, rs2 uint32, imm int32) uint32 {
	imm_4_0 := uint32(imm & 0x1f)
	imm_11_5 := uint32((imm >> 5) & 0x7f)
	return opcode | (imm_4_0 << 7) | (funct3 << 12) | (rs1 << 15) | (rs2 << 20) | (imm_11_5 << 25)
}

// B-type: opcode[6:0] | imm[11|4:1] | funct3[14:12] | rs1[19:15] | rs2[24:20] | imm[12|10:5]
func (r *RiscvOut) encodeBType(opcode, funct3 uint32, rs1, rs2 uint32, imm int32) uint32 {
	imm_11 := uint32((imm >> 11) & 0x1)
	imm_4_1 := uint32((imm >> 1) & 0xf)
	imm_10_5 := uint32((imm >> 5) & 0x3f)
	imm_12 := uint32((imm >> 12) & 0x1)
	return opcode | (imm_11 << 7) | (imm_4_1 << 8) | (funct3 << 12) | (rs1 << 15) | (rs2 << 20) | (imm_10_5 << 25) | (imm_12 << 31)
}

// U-type: opcode[6:0] | rd[11:7] | imm[31:12]
func (r *RiscvOut) encodeUType(opcode uint32, rd uint32, imm uint32) uint32 {
	return opcode | (rd << 7) | (imm & 0xfffff000)
}

// J-type: opcode[6:0] | rd[11:7] | imm[19:12|11|10:1|20]
func (r *RiscvOut) encodeJType(opcode uint32, rd uint32, imm int32) uint32 {
	imm_19_12 := uint32((imm >> 12) & 0xff)
	imm_11 := uint32((imm >> 11) & 0x1)
	imm_10_1 := uint32((imm >> 1) & 0x3ff)
	imm_20 := uint32((imm >> 20) & 0x1)
	return opcode | (rd << 7) | (imm_19_12 << 12) | (imm_11 << 20) | (imm_10_1 << 21) | (imm_20 << 31)
}

// ADD: add rd, rs1, rs2
func (r *RiscvOut) Add(dest, src1, src2 string) error {
	rd, ok := riscvGPRegs[dest]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", dest)
	}
	rs1, ok := riscvGPRegs[src1]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", src1)
	}
	rs2, ok := riscvGPRegs[src2]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", src2)
	}

	// ADD: opcode=0110011, funct3=000, funct7=0000000
	instr := r.encodeRType(0x33, 0x0, 0x00, rd, rs1, rs2)
	r.encodeInstr(instr)
	return nil
}

// ADDI: addi rd, rs1, imm
func (r *RiscvOut) AddImm(dest, src string, imm int32) error {
	rd, ok := riscvGPRegs[dest]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", dest)
	}
	rs1, ok := riscvGPRegs[src]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", src)
	}

	// Check immediate range (-2048 to 2047)
	if imm < -2048 || imm > 2047 {
		return fmt.Errorf("immediate value out of range for ADDI: %d", imm)
	}

	// ADDI: opcode=0010011, funct3=000
	instr := r.encodeIType(0x13, 0x0, rd, rs1, imm)
	r.encodeInstr(instr)
	return nil
}

// SUB: sub rd, rs1, rs2
func (r *RiscvOut) Sub(dest, src1, src2 string) error {
	rd, ok := riscvGPRegs[dest]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", dest)
	}
	rs1, ok := riscvGPRegs[src1]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", src1)
	}
	rs2, ok := riscvGPRegs[src2]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", src2)
	}

	// SUB: opcode=0110011, funct3=000, funct7=0100000
	instr := r.encodeRType(0x33, 0x0, 0x20, rd, rs1, rs2)
	r.encodeInstr(instr)
	return nil
}

// MV (pseudo-instruction): addi rd, rs, 0
func (r *RiscvOut) Move(dest, src string) error {
	return r.AddImm(dest, src, 0)
}

// LI (load immediate, pseudo-instruction)
func (r *RiscvOut) LoadImm(dest string, imm int64) error {
	rd, ok := riscvGPRegs[dest]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", dest)
	}

	// For small immediates, use ADDI
	if imm >= -2048 && imm <= 2047 {
		return r.AddImm(dest, "zero", int32(imm))
	}

	// For larger immediates, use LUI + ADDI
	// LUI loads upper 20 bits, ADDI adds lower 12 bits
	upper := uint32((imm + 0x800) >> 12) // Add 0x800 for sign extension compensation
	lower := int32(imm & 0xfff)

	// LUI: opcode=0110111
	instr := r.encodeUType(0x37, rd, upper<<12)
	r.encodeInstr(instr)

	// ADDI to add lower bits (if non-zero)
	if lower != 0 {
		instr = r.encodeIType(0x13, 0x0, rd, rd, lower)
		r.encodeInstr(instr)
	}

	return nil
}

// LD: ld rd, offset(rs1)
func (r *RiscvOut) Load64(dest, base string, offset int32) error {
	rd, ok := riscvGPRegs[dest]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", dest)
	}
	rs1, ok := riscvGPRegs[base]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", base)
	}

	if offset < -2048 || offset > 2047 {
		return fmt.Errorf("load offset out of range: %d", offset)
	}

	// LD: opcode=0000011, funct3=011
	instr := r.encodeIType(0x03, 0x3, rd, rs1, offset)
	r.encodeInstr(instr)
	return nil
}

// SD: sd rs2, offset(rs1)
func (r *RiscvOut) Store64(src, base string, offset int32) error {
	rs2, ok := riscvGPRegs[src]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", src)
	}
	rs1, ok := riscvGPRegs[base]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", base)
	}

	if offset < -2048 || offset > 2047 {
		return fmt.Errorf("store offset out of range: %d", offset)
	}

	// SD: opcode=0100011, funct3=011
	instr := r.encodeSType(0x23, 0x3, rs1, rs2, offset)
	r.encodeInstr(instr)
	return nil
}

// JAL: jal rd, offset
func (r *RiscvOut) JumpAndLink(dest string, offset int32) error {
	rd, ok := riscvGPRegs[dest]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", dest)
	}

	// Offset must be even and within ±1MB
	if offset%2 != 0 {
		return fmt.Errorf("JAL offset must be even: %d", offset)
	}
	if offset < -(1<<20) || offset >= (1<<20) {
		return fmt.Errorf("JAL offset out of range: %d", offset)
	}

	// JAL: opcode=1101111
	instr := r.encodeJType(0x6f, rd, offset)
	r.encodeInstr(instr)
	return nil
}

// JALR: jalr rd, offset(rs1)
func (r *RiscvOut) JumpAndLinkRegister(dest, base string, offset int32) error {
	rd, ok := riscvGPRegs[dest]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", dest)
	}
	rs1, ok := riscvGPRegs[base]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", base)
	}

	if offset < -2048 || offset > 2047 {
		return fmt.Errorf("JALR offset out of range: %d", offset)
	}

	// JALR: opcode=1100111, funct3=000
	instr := r.encodeIType(0x67, 0x0, rd, rs1, offset)
	r.encodeInstr(instr)
	return nil
}

// BEQ: beq rs1, rs2, offset
func (r *RiscvOut) BranchEqual(src1, src2 string, offset int32) error {
	rs1, ok := riscvGPRegs[src1]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", src1)
	}
	rs2, ok := riscvGPRegs[src2]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", src2)
	}

	// Offset must be even and within ±4KB
	if offset%2 != 0 {
		return fmt.Errorf("branch offset must be even: %d", offset)
	}
	if offset < -(1<<12) || offset >= (1<<12) {
		return fmt.Errorf("branch offset out of range: %d", offset)
	}

	// BEQ: opcode=1100011, funct3=000
	instr := r.encodeBType(0x63, 0x0, rs1, rs2, offset)
	r.encodeInstr(instr)
	return nil
}

// BNE: bne rs1, rs2, offset
func (r *RiscvOut) BranchNotEqual(src1, src2 string, offset int32) error {
	rs1, ok := riscvGPRegs[src1]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", src1)
	}
	rs2, ok := riscvGPRegs[src2]
	if !ok {
		return fmt.Errorf("invalid RISC-V register: %s", src2)
	}

	if offset%2 != 0 {
		return fmt.Errorf("branch offset must be even: %d", offset)
	}
	if offset < -(1<<12) || offset >= (1<<12) {
		return fmt.Errorf("branch offset out of range: %d", offset)
	}

	// BNE: opcode=1100011, funct3=001
	instr := r.encodeBType(0x63, 0x1, rs1, rs2, offset)
	r.encodeInstr(instr)
	return nil
}

// RET (pseudo-instruction): jalr zero, 0(ra)
func (r *RiscvOut) Return() error {
	return r.JumpAndLinkRegister("zero", "ra", 0)
}

// ECALL: system call
func (r *RiscvOut) Ecall() {
	// ECALL: opcode=1110011, funct3=000, imm=0
	instr := r.encodeIType(0x73, 0x0, 0, 0, 0)
	r.encodeInstr(instr)
}

// TODO: Add floating-point instructions (FADD.D, FSUB.D, FMUL.D, FDIV.D, FCVT, etc.)
// TODO: Add multiply/divide instructions (MUL, MULH, DIV, REM, etc.)
// TODO: Add logical instructions (AND, OR, XOR, etc.)
// TODO: Add shift instructions (SLL, SRL, SRA, etc.)
// TODO: Add atomic instructions (LR, SC, AMO*, etc.)
// TODO: Add CSR instructions
