package main

import (
	"encoding/binary"
	"fmt"
)

// ARM64 instruction encoding
// ARM64 uses fixed 32-bit little-endian instructions

// ARM64 Register mapping
var arm64GPRegs = map[string]uint32{
	"x0": 0, "x1": 1, "x2": 2, "x3": 3, "x4": 4, "x5": 5, "x6": 6, "x7": 7,
	"x8": 8, "x9": 9, "x10": 10, "x11": 11, "x12": 12, "x13": 13, "x14": 14, "x15": 15,
	"x16": 16, "x17": 17, "x18": 18, "x19": 19, "x20": 20, "x21": 21, "x22": 22, "x23": 23,
	"x24": 24, "x25": 25, "x26": 26, "x27": 27, "x28": 28, "x29": 29, "x30": 30,
	"xzr": 31, "sp": 31, "fp": 29, "lr": 30,
}

var arm64FPRegs = map[string]uint32{
	"v0": 0, "v1": 1, "v2": 2, "v3": 3, "v4": 4, "v5": 5, "v6": 6, "v7": 7,
	"v8": 8, "v9": 9, "v10": 10, "v11": 11, "v12": 12, "v13": 13, "v14": 14, "v15": 15,
	"v16": 16, "v17": 17, "v18": 18, "v19": 19, "v20": 20, "v21": 21, "v22": 22, "v23": 23,
	"v24": 24, "v25": 25, "v26": 26, "v27": 27, "v28": 28, "v29": 29, "v30": 30, "v31": 31,
	// Also support d0-d31 (64-bit), s0-s31 (32-bit)
	"d0": 0, "d1": 1, "d2": 2, "d3": 3, "d4": 4, "d5": 5, "d6": 6, "d7": 7,
	"d8": 8, "d9": 9, "d10": 10, "d11": 11, "d12": 12, "d13": 13, "d14": 14, "d15": 15,
	"d16": 16, "d17": 17, "d18": 18, "d19": 19, "d20": 20, "d21": 21, "d22": 22, "d23": 23,
	"d24": 24, "d25": 25, "d26": 26, "d27": 27, "d28": 28, "d29": 29, "d30": 30, "d31": 31,
}

// ARM64Out wraps Out for ARM64-specific instructions
type ARM64Out struct {
	out *Out
}

// encodeInstr writes a 32-bit ARM64 instruction in little-endian format
func (a *ARM64Out) encodeInstr(instr uint32) {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], instr)
	a.out.writer.WriteBytes(buf[:])
}

// ARM64 Instruction encodings

// ADD (immediate): ADD Xd, Xn, #imm
// opcode: 1001000100 | imm12 | Rn | Rd
func (a *ARM64Out) AddImm64(dest, src string, imm uint32) error {
	rd, ok := arm64GPRegs[dest]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", dest)
	}
	rn, ok := arm64GPRegs[src]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", src)
	}
	if imm > 0xfff {
		return fmt.Errorf("immediate value too large for ADD: %d", imm)
	}

	// ADD (immediate, 64-bit): sf=1, op=0, S=0
	instr := uint32(0x91000000) | (imm << 10) | (rn << 5) | rd
	a.encodeInstr(instr)
	return nil
}

// SUB (immediate): SUB Xd, Xn, #imm
func (a *ARM64Out) SubImm64(dest, src string, imm uint32) error {
	rd, ok := arm64GPRegs[dest]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", dest)
	}
	rn, ok := arm64GPRegs[src]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", src)
	}
	if imm > 0xfff {
		return fmt.Errorf("immediate value too large for SUB: %d", imm)
	}

	// SUB (immediate, 64-bit): sf=1, op=1, S=0
	instr := uint32(0xd1000000) | (imm << 10) | (rn << 5) | rd
	a.encodeInstr(instr)
	return nil
}

// MOV (register): MOV Xd, Xn  (alias for ORR Xd, XZR, Xn)
func (a *ARM64Out) MovReg64(dest, src string) error {
	rd, ok := arm64GPRegs[dest]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", dest)
	}
	rm, ok := arm64GPRegs[src]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", src)
	}

	// ORR (shifted register): ORR Xd, XZR, Xm
	// sf=1, opc=01, N=0, shift=00, Rm, imm6=0, Rn=31 (xzr), Rd
	instr := uint32(0xaa0003e0) | (rm << 16) | rd
	a.encodeInstr(instr)
	return nil
}

// MOVZ (move wide with zero): MOVZ Xd, #imm16, LSL #shift
func (a *ARM64Out) MovImm64(dest string, imm uint64) error {
	rd, ok := arm64GPRegs[dest]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", dest)
	}

	// For values that fit in 16 bits, use MOVZ
	if imm <= 0xffff {
		// MOVZ Xd, #imm16, LSL #0
		instr := uint32(0xd2800000) | (uint32(imm&0xffff) << 5) | rd
		a.encodeInstr(instr)
		return nil
	}

	// For larger values, use MOVZ followed by MOVK instructions
	// MOVZ for lowest 16 bits
	instr := uint32(0xd2800000) | (uint32(imm&0xffff) << 5) | rd
	a.encodeInstr(instr)

	// MOVK for each subsequent 16-bit chunk
	if (imm>>16)&0xffff != 0 {
		instr = uint32(0xf2a00000) | (uint32((imm>>16)&0xffff) << 5) | rd
		a.encodeInstr(instr)
	}
	if (imm>>32)&0xffff != 0 {
		instr = uint32(0xf2c00000) | (uint32((imm>>32)&0xffff) << 5) | rd
		a.encodeInstr(instr)
	}
	if (imm>>48)&0xffff != 0 {
		instr = uint32(0xf2e00000) | (uint32((imm>>48)&0xffff) << 5) | rd
		a.encodeInstr(instr)
	}

	return nil
}

// LDR (literal): LDR Xt, label (PC-relative)
func (a *ARM64Out) LdrLiteral64(dest string, offset int32) error {
	rd, ok := arm64GPRegs[dest]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", dest)
	}

	// offset must be word-aligned and within ±1MB
	if offset%4 != 0 {
		return fmt.Errorf("LDR literal offset must be word-aligned: %d", offset)
	}
	imm19 := offset >> 2
	if imm19 < -(1<<18) || imm19 >= (1<<18) {
		return fmt.Errorf("LDR literal offset out of range: %d", offset)
	}

	// LDR (literal, 64-bit): opc=01
	instr := uint32(0x58000000) | (uint32(imm19&0x7ffff) << 5) | rd
	a.encodeInstr(instr)
	return nil
}

// STR (immediate): STR Xt, [Xn, #offset]
func (a *ARM64Out) StrImm64(src, base string, offset int32) error {
	rt, ok := arm64GPRegs[src]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", src)
	}
	rn, ok := arm64GPRegs[base]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", base)
	}

	// Check if offset fits in unsigned 12-bit scaled immediate (must be multiple of 8)
	if offset < 0 || offset%8 != 0 || offset >= (1<<12)*8 {
		return fmt.Errorf("STR offset out of range or not aligned: %d", offset)
	}

	imm12 := uint32(offset / 8)
	// STR (immediate, unsigned offset, 64-bit)
	instr := uint32(0xf9000000) | (imm12 << 10) | (rn << 5) | rt
	a.encodeInstr(instr)
	return nil
}

// LDR (immediate): LDR Xt, [Xn, #offset]
func (a *ARM64Out) LdrImm64(dest, base string, offset int32) error {
	rt, ok := arm64GPRegs[dest]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", dest)
	}
	rn, ok := arm64GPRegs[base]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", base)
	}

	if offset < 0 || offset%8 != 0 || offset >= (1<<12)*8 {
		return fmt.Errorf("LDR offset out of range or not aligned: %d", offset)
	}

	imm12 := uint32(offset / 8)
	// LDR (immediate, unsigned offset, 64-bit)
	instr := uint32(0xf9400000) | (imm12 << 10) | (rn << 5) | rt
	a.encodeInstr(instr)
	return nil
}

// B (branch): B label
func (a *ARM64Out) Branch(offset int32) error {
	// offset must be word-aligned and within ±128MB
	if offset%4 != 0 {
		return fmt.Errorf("branch offset must be word-aligned: %d", offset)
	}
	imm26 := offset >> 2
	if imm26 < -(1<<25) || imm26 >= (1<<25) {
		return fmt.Errorf("branch offset out of range: %d", offset)
	}

	// B: op=0, imm26
	instr := uint32(0x14000000) | uint32(imm26&0x3ffffff)
	a.encodeInstr(instr)
	return nil
}

// BL (branch with link): BL label
func (a *ARM64Out) BranchLink(offset int32) error {
	if offset%4 != 0 {
		return fmt.Errorf("branch offset must be word-aligned: %d", offset)
	}
	imm26 := offset >> 2
	if imm26 < -(1<<25) || imm26 >= (1<<25) {
		return fmt.Errorf("branch offset out of range: %d", offset)
	}

	// BL: op=1, imm26
	instr := uint32(0x94000000) | uint32(imm26&0x3ffffff)
	a.encodeInstr(instr)
	return nil
}

// RET (return): RET Xn
func (a *ARM64Out) Return(reg string) error {
	rn, ok := arm64GPRegs[reg]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", reg)
	}

	// RET Xn: opc=10, op2=11111, op3=00000, Rn=Xn, op4=00000
	instr := uint32(0xd65f0000) | (rn << 5)
	a.encodeInstr(instr)
	return nil
}

// CBZ (compare and branch if zero): CBZ Xt, label
func (a *ARM64Out) CompareAndBranchZero64(reg string, offset int32) error {
	rt, ok := arm64GPRegs[reg]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", reg)
	}

	if offset%4 != 0 {
		return fmt.Errorf("branch offset must be word-aligned: %d", offset)
	}
	imm19 := offset >> 2
	if imm19 < -(1<<18) || imm19 >= (1<<18) {
		return fmt.Errorf("branch offset out of range: %d", offset)
	}

	// CBZ (64-bit): sf=1, op=0, imm19, Rt
	instr := uint32(0xb4000000) | (uint32(imm19&0x7ffff) << 5) | rt
	a.encodeInstr(instr)
	return nil
}

// CBNZ (compare and branch if non-zero): CBNZ Xt, label
func (a *ARM64Out) CompareAndBranchNonZero64(reg string, offset int32) error {
	rt, ok := arm64GPRegs[reg]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", reg)
	}

	if offset%4 != 0 {
		return fmt.Errorf("branch offset must be word-aligned: %d", offset)
	}
	imm19 := offset >> 2
	if imm19 < -(1<<18) || imm19 >= (1<<18) {
		return fmt.Errorf("branch offset out of range: %d", offset)
	}

	// CBNZ (64-bit): sf=1, op=1, imm19, Rt
	instr := uint32(0xb5000000) | (uint32(imm19&0x7ffff) << 5) | rt
	a.encodeInstr(instr)
	return nil
}

// LDR (immediate, FP/SIMD): LDR Dt, [Xn, #offset]
func (a *ARM64Out) LdrImm64Double(dest, base string, offset int32) error {
	rt, ok := arm64FPRegs[dest]
	if !ok {
		return fmt.Errorf("invalid ARM64 FP register: %s", dest)
	}
	rn, ok := arm64GPRegs[base]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", base)
	}

	if offset%8 != 0 {
		return fmt.Errorf("LDR FP offset not 8-byte aligned: %d", offset)
	}

	// For negative offsets, use LDUR (unscaled) instead
	if offset < 0 {
		if offset < -256 || offset > 255 {
			return fmt.Errorf("LDR FP offset out of range for LDUR: %d", offset)
		}
		// LDUR (unscaled): size=11, V=1, opc=01
		imm9 := uint32(offset) & 0x1ff
		instr := uint32(0xfc400000) | (imm9 << 12) | (rn << 5) | rt
		a.encodeInstr(instr)
		return nil
	}

	if offset >= (1<<12)*8 {
		return fmt.Errorf("LDR FP offset out of range: %d", offset)
	}

	imm12 := uint32(offset / 8)
	// LDR (immediate, unsigned offset, 64-bit FP/SIMD): size=11 (64-bit)
	instr := uint32(0xfd400000) | (imm12 << 10) | (rn << 5) | rt
	a.encodeInstr(instr)
	return nil
}

// STR (immediate, FP/SIMD): STR Dt, [Xn, #offset]
func (a *ARM64Out) StrImm64Double(src, base string, offset int32) error {
	rt, ok := arm64FPRegs[src]
	if !ok {
		return fmt.Errorf("invalid ARM64 FP register: %s", src)
	}
	rn, ok := arm64GPRegs[base]
	if !ok {
		return fmt.Errorf("invalid ARM64 register: %s", base)
	}

	if offset%8 != 0 {
		return fmt.Errorf("STR FP offset not 8-byte aligned: %d", offset)
	}

	// For negative offsets, use STUR (unscaled) instead
	if offset < 0 {
		if offset < -256 || offset > 255 {
			return fmt.Errorf("STR FP offset out of range for STUR: %d", offset)
		}
		// STUR (unscaled): size=11, V=1, opc=00
		imm9 := uint32(offset) & 0x1ff
		instr := uint32(0xfc000000) | (imm9 << 12) | (rn << 5) | rt
		a.encodeInstr(instr)
		return nil
	}

	if offset >= (1<<12)*8 {
		return fmt.Errorf("STR FP offset out of range: %d", offset)
	}

	imm12 := uint32(offset / 8)
	// STR (immediate, unsigned offset, 64-bit FP/SIMD): size=11 (64-bit)
	instr := uint32(0xfd000000) | (imm12 << 10) | (rn << 5) | rt
	a.encodeInstr(instr)
	return nil
}

// B.cond (conditional branch): B.cond label
// Condition codes: EQ, NE, CS/HS, CC/LO, MI, PL, VS, VC, HI, LS, GE, LT, GT, LE, AL
func (a *ARM64Out) BranchCond(cond string, offset int32) error {
	if offset%4 != 0 {
		return fmt.Errorf("branch offset must be word-aligned: %d", offset)
	}
	imm19 := offset >> 2
	if imm19 < -(1<<18) || imm19 >= (1<<18) {
		return fmt.Errorf("branch offset out of range: %d", offset)
	}

	// Map condition codes to their values
	condMap := map[string]uint32{
		"eq": 0x0, "ne": 0x1, "cs": 0x2, "hs": 0x2, "cc": 0x3, "lo": 0x3,
		"mi": 0x4, "pl": 0x5, "vs": 0x6, "vc": 0x7, "hi": 0x8, "ls": 0x9,
		"ge": 0xa, "lt": 0xb, "gt": 0xc, "le": 0xd, "al": 0xe,
	}

	condCode, ok := condMap[cond]
	if !ok {
		return fmt.Errorf("invalid condition code: %s", cond)
	}

	// B.cond: 01010100 | imm19 | 0 | cond
	instr := uint32(0x54000000) | (uint32(imm19&0x7ffff) << 5) | condCode
	a.encodeInstr(instr)
	return nil
}

// TODO: Add more floating-point instructions (FADD, FSUB, FMUL, FDIV, FCVT, etc.)
// TODO: Add SIMD/NEON instructions
// TODO: Add load/store pair instructions (STP, LDP)
// TODO: Add more arithmetic instructions (MUL, UDIV, SDIV, etc.)
// TODO: Add logical instructions (AND, OR, EOR, etc.)
// TODO: Add shift instructions (LSL, LSR, ASR, ROR)
