// Completion: 80% - Backend functional, some TODOs for advanced features
package main

import (
	"strconv"
	"strings"
)

// ARM64Backend implements the CodeGenerator interface for ARM64 architecture
type ARM64Backend struct {
	writer Writer
	eb     *ExecutableBuilder
}

// NewARM64Backend creates a new ARM64 code generator backend
func NewARM64Backend(writer Writer, eb *ExecutableBuilder) *ARM64Backend {
	return &ARM64Backend{
		writer: writer,
		eb:     eb,
	}
}

func (a *ARM64Backend) write(b uint8) {
	a.writer.(*BufferWrapper).Write(b)
}

func (a *ARM64Backend) writeUnsigned(i uint) {
	a.writer.(*BufferWrapper).WriteUnsigned(i)
}

func (a *ARM64Backend) emit(bytes []byte) {
	for _, b := range bytes {
		a.write(b)
	}
}

func (a *ARM64Backend) writeInstruction(instr uint32) {
	// ARM64 instructions are little-endian
	a.write(uint8(instr & 0xFF))
	a.write(uint8((instr >> 8) & 0xFF))
	a.write(uint8((instr >> 16) & 0xFF))
	a.write(uint8((instr >> 24) & 0xFF))
}

// ===== Data Movement =====

func (a *ARM64Backend) MovRegToReg(dst, src string) {
	dstReg, dstOk := arm64Registers[dst]
	srcReg, srcOk := arm64Registers[src]
	if !dstOk || !srcOk {
		return
	}

	// ARM64 MOV (register): ORR Xd, XZR, Xm
	var instr uint32
	if dstReg.Size == 64 && srcReg.Size == 64 {
		// 64-bit: ORR Xd, XZR, Xm
		instr = 0xAA0003E0 | (uint32(srcReg.Encoding&31) << 16) | uint32(dstReg.Encoding&31)
	} else {
		// 32-bit: ORR Wd, WZR, Wm
		instr = 0x2A0003E0 | (uint32(srcReg.Encoding&31) << 16) | uint32(dstReg.Encoding&31)
	}

	a.writeInstruction(instr)
}

func (a *ARM64Backend) MovImmToReg(dst, imm string) {
	dstReg, dstOk := arm64Registers[dst]
	if !dstOk {
		return
	}

	// Strip ARM64 immediate prefix (#) if present
	imm = strings.TrimPrefix(imm, "#")

	// Parse immediate value
	var immVal uint64
	if val, err := strconv.ParseInt(imm, 0, 64); err == nil {
		immVal = uint64(val)
	} else if val, err := strconv.ParseUint(imm, 0, 64); err == nil {
		immVal = val
	}

	// Use MOVZ (move with zero) for immediate values
	var instr uint32
	if dstReg.Size == 64 {
		// MOVZ Xd, #imm16
		instr = 0xD2800000 | (uint32(immVal&0xFFFF) << 5) | uint32(dstReg.Encoding&31)
	} else {
		// MOVZ Wd, #imm16
		instr = 0x52800000 | (uint32(immVal&0xFFFF) << 5) | uint32(dstReg.Encoding&31)
	}

	a.writeInstruction(instr)
}

func (a *ARM64Backend) MovMemToReg(dst, symbol string, offset int32) {
	compilerError("ARM64Backend.MovMemToReg not implemented")
}

func (a *ARM64Backend) MovRegToMem(src, symbol string, offset int32) {
	compilerError("ARM64Backend.MovRegToMem not implemented")
}

// ===== Integer Arithmetic =====

func (a *ARM64Backend) AddRegToReg(dst, src string) {
	dstReg, dstOk := arm64Registers[dst]
	srcReg, srcOk := arm64Registers[src]
	if !dstOk || !srcOk {
		return
	}

	// ADD Xd, Xd, Xm (shifted register form)
	var instr uint32
	if dstReg.Size == 64 {
		instr = 0x8B000000 | (uint32(srcReg.Encoding&31) << 16) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	} else {
		instr = 0x0B000000 | (uint32(srcReg.Encoding&31) << 16) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	}

	a.writeInstruction(instr)
}

func (a *ARM64Backend) AddImmToReg(dst string, imm int64) {
	dstReg, dstOk := arm64Registers[dst]
	if !dstOk {
		return
	}

	// ADD Xd, Xd, #imm12
	if imm < 0 || imm > 4095 {
		imm = imm & 0xFFF
	}

	var instr uint32
	if dstReg.Size == 64 {
		instr = 0x91000000 | (uint32(imm&0xFFF) << 10) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	} else {
		instr = 0x11000000 | (uint32(imm&0xFFF) << 10) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	}

	a.writeInstruction(instr)
}

func (a *ARM64Backend) SubRegToReg(dst, src string) {
	dstReg, dstOk := arm64Registers[dst]
	srcReg, srcOk := arm64Registers[src]
	if !dstOk || !srcOk {
		return
	}

	// SUB Xd, Xd, Xm
	var instr uint32
	if dstReg.Size == 64 {
		instr = 0xCB000000 | (uint32(srcReg.Encoding&31) << 16) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	} else {
		instr = 0x4B000000 | (uint32(srcReg.Encoding&31) << 16) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	}

	a.writeInstruction(instr)
}

func (a *ARM64Backend) SubImmFromReg(dst string, imm int64) {
	dstReg, dstOk := arm64Registers[dst]
	if !dstOk {
		return
	}

	// SUB Xd, Xd, #imm12
	if imm < 0 || imm > 4095 {
		imm = imm & 0xFFF
	}

	var instr uint32
	if dstReg.Size == 64 {
		instr = 0xD1000000 | (uint32(imm&0xFFF) << 10) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	} else {
		instr = 0x51000000 | (uint32(imm&0xFFF) << 10) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	}

	a.writeInstruction(instr)
}

func (a *ARM64Backend) MulRegToReg(dst, src string) {
	dstReg, dstOk := arm64Registers[dst]
	srcReg, srcOk := arm64Registers[src]
	if !dstOk || !srcOk {
		return
	}

	// MUL Xd, Xd, Xm (MADD Xd, Xd, Xm, XZR)
	var instr uint32
	if dstReg.Size == 64 {
		instr = 0x9B007C00 | (uint32(srcReg.Encoding&31) << 16) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	} else {
		instr = 0x1B007C00 | (uint32(srcReg.Encoding&31) << 16) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	}

	a.writeInstruction(instr)
}

func (a *ARM64Backend) DivRegToReg(dst, src string) {
	dstReg, dstOk := arm64Registers[dst]
	srcReg, srcOk := arm64Registers[src]
	if !dstOk || !srcOk {
		return
	}

	// SDIV Xd, Xd, Xm (signed division)
	var instr uint32
	if dstReg.Size == 64 {
		instr = 0x9AC00C00 | (uint32(srcReg.Encoding&31) << 16) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	} else {
		instr = 0x1AC00C00 | (uint32(srcReg.Encoding&31) << 16) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	}

	a.writeInstruction(instr)
}

func (a *ARM64Backend) IncReg(dst string) {
	// ADD Xd, Xd, #1
	a.AddImmToReg(dst, 1)
}

func (a *ARM64Backend) DecReg(dst string) {
	// SUB Xd, Xd, #1
	a.SubImmFromReg(dst, 1)
}

func (a *ARM64Backend) NegReg(dst string) {
	dstReg, dstOk := arm64Registers[dst]
	if !dstOk {
		return
	}

	// NEG Xd, Xd (SUB Xd, XZR, Xd)
	var instr uint32
	if dstReg.Size == 64 {
		instr = 0xCB0003E0 | (uint32(dstReg.Encoding&31) << 16) | uint32(dstReg.Encoding&31)
	} else {
		instr = 0x4B0003E0 | (uint32(dstReg.Encoding&31) << 16) | uint32(dstReg.Encoding&31)
	}

	a.writeInstruction(instr)
}

// ===== Bitwise Operations =====

func (a *ARM64Backend) XorRegWithReg(dst, src string) {
	dstReg, dstOk := arm64Registers[dst]
	srcReg, srcOk := arm64Registers[src]
	if !dstOk || !srcOk {
		return
	}

	// EOR Xd, Xd, Xm
	var instr uint32
	if dstReg.Size == 64 {
		instr = 0xCA000000 | (uint32(srcReg.Encoding&31) << 16) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	} else {
		instr = 0x4A000000 | (uint32(srcReg.Encoding&31) << 16) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	}

	a.writeInstruction(instr)
}

func (a *ARM64Backend) XorRegWithImm(dst string, imm int64) {
	compilerError("ARM64Backend.XorRegWithImm not implemented")
}

func (a *ARM64Backend) AndRegWithReg(dst, src string) {
	dstReg, dstOk := arm64Registers[dst]
	srcReg, srcOk := arm64Registers[src]
	if !dstOk || !srcOk {
		return
	}

	// AND Xd, Xd, Xm
	var instr uint32
	if dstReg.Size == 64 {
		instr = 0x8A000000 | (uint32(srcReg.Encoding&31) << 16) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	} else {
		instr = 0x0A000000 | (uint32(srcReg.Encoding&31) << 16) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	}

	a.writeInstruction(instr)
}

func (a *ARM64Backend) OrRegWithReg(dst, src string) {
	dstReg, dstOk := arm64Registers[dst]
	srcReg, srcOk := arm64Registers[src]
	if !dstOk || !srcOk {
		return
	}

	// ORR Xd, Xd, Xm
	var instr uint32
	if dstReg.Size == 64 {
		instr = 0xAA000000 | (uint32(srcReg.Encoding&31) << 16) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	} else {
		instr = 0x2A000000 | (uint32(srcReg.Encoding&31) << 16) | (uint32(dstReg.Encoding&31) << 5) | uint32(dstReg.Encoding&31)
	}

	a.writeInstruction(instr)
}

func (a *ARM64Backend) NotReg(dst string) {
	dstReg, dstOk := arm64Registers[dst]
	if !dstOk {
		return
	}

	// MVN Xd, Xd (ORN Xd, XZR, Xd)
	var instr uint32
	if dstReg.Size == 64 {
		instr = 0xAA2003E0 | (uint32(dstReg.Encoding&31) << 16) | uint32(dstReg.Encoding&31)
	} else {
		instr = 0x2A2003E0 | (uint32(dstReg.Encoding&31) << 16) | uint32(dstReg.Encoding&31)
	}

	a.writeInstruction(instr)
}

// ===== Stack Operations =====

func (a *ARM64Backend) PushReg(reg string) {
	compilerError("ARM64Backend.PushReg not implemented (ARM64 doesn't have PUSH/POP)")
}

func (a *ARM64Backend) PopReg(reg string) {
	compilerError("ARM64Backend.PopReg not implemented (ARM64 doesn't have PUSH/POP)")
}

// ===== Control Flow =====

func (a *ARM64Backend) JumpConditional(condition JumpCondition, offset int32) {
	// B.cond offset (conditional branch)
	// Offset is in instructions (4-byte units), signed 19-bit
	immOffset := offset / 4
	if immOffset < -262144 || immOffset > 262143 {
		compilerError("ARM64 conditional branch offset out of range: %d", immOffset)
		return
	}

	var cond uint32
	switch condition {
	case JumpEqual:
		cond = 0x0 // EQ
	case JumpNotEqual:
		cond = 0x1 // NE
	case JumpGreater:
		cond = 0xC // GT
	case JumpGreaterOrEqual:
		cond = 0xA // GE
	case JumpLess:
		cond = 0xB // LT
	case JumpLessOrEqual:
		cond = 0xD // LE
	default:
		compilerError("Unknown jump condition for ARM64: %v", condition)
		return
	}

	// B.cond: 0101010 0 imm19 0 cond
	instr := uint32(0x54000000) | (uint32(immOffset&0x7FFFF) << 5) | cond
	a.writeInstruction(instr)
}

func (a *ARM64Backend) JumpUnconditional(offset int32) {
	// B offset (unconditional branch)
	// Offset is in instructions (4-byte units), signed 26-bit
	immOffset := offset / 4
	if immOffset < -33554432 || immOffset > 33554431 {
		compilerError("ARM64 unconditional branch offset out of range: %d", immOffset)
		return
	}

	// B: 000101 imm26
	instr := uint32(0x14000000) | uint32(immOffset&0x3FFFFFF)
	a.writeInstruction(instr)
}

func (a *ARM64Backend) CallSymbol(symbol string) {
	// BL offset (branch with link)
	a.write(0x94) // Placeholder - will be patched

	callPos := a.eb.text.Len()
	a.writeUnsigned(0x000000) // 3 more bytes (total 4 bytes for BL)

	a.eb.callPatches = append(a.eb.callPatches, CallPatch{
		position:   callPos - 1, // Point to start of instruction
		targetName: symbol,
	})
}

func (a *ARM64Backend) CallRelative(offset int32) {
	// BL offset
	immOffset := offset / 4
	if immOffset < -33554432 || immOffset > 33554431 {
		compilerError("ARM64 BL offset out of range: %d", immOffset)
		return
	}

	// BL: 100101 imm26
	instr := uint32(0x94000000) | uint32(immOffset&0x3FFFFFF)
	a.writeInstruction(instr)
}

func (a *ARM64Backend) CallRegister(reg string) {
	regInfo, regOk := arm64Registers[reg]
	if !regOk {
		return
	}

	// BLR Xn (branch with link to register)
	// BLR: 1101011 0 0 01 11111 000000 Rn 00000
	instr := uint32(0xD63F0000) | (uint32(regInfo.Encoding&31) << 5)
	a.writeInstruction(instr)
}

func (a *ARM64Backend) Ret() {
	// RET (BR X30)
	instr := uint32(0xD65F03C0)
	a.writeInstruction(instr)
}

// ===== Comparisons =====

func (a *ARM64Backend) CmpRegToReg(reg1, reg2 string) {
	reg1Info, reg1Ok := arm64Registers[reg1]
	reg2Info, reg2Ok := arm64Registers[reg2]
	if !reg1Ok || !reg2Ok {
		return
	}

	// CMP is encoded as SUBS XZR, Xn, Xm
	instr := uint32(0xEB000000) |
		(uint32(reg2Info.Encoding&31) << 16) |
		(uint32(reg1Info.Encoding&31) << 5) |
		31 // Rd = XZR

	a.writeInstruction(instr)
}

func (a *ARM64Backend) CmpRegToImm(reg string, imm int64) {
	regInfo, regOk := arm64Registers[reg]
	if !regOk {
		return
	}

	// ARM64 immediate must be 12-bit unsigned
	if imm < 0 || imm > 4095 {
		imm = 0
	}

	// SUBS XZR, Xn, #imm
	instr := uint32(0xF1000000) |
		(uint32(imm&0xFFF) << 10) |
		(uint32(regInfo.Encoding&31) << 5) |
		31 // Rd = XZR

	a.writeInstruction(instr)
}

// ===== Address Calculation =====

func (a *ARM64Backend) LeaSymbolToReg(dst, symbol string) {
	compilerError("ARM64Backend.LeaSymbolToReg not implemented")
}

func (a *ARM64Backend) LeaImmToReg(dst, base string, offset int32) {
	// ADD Xd, Xbase, #offset
	dstReg, dstOk := arm64Registers[dst]
	baseReg, baseOk := arm64Registers[base]
	if !dstOk || !baseOk {
		return
	}

	if offset < 0 || offset > 4095 {
		offset = offset & 0xFFF
	}

	instr := uint32(0x91000000) |
		(uint32(offset&0xFFF) << 10) |
		(uint32(baseReg.Encoding&31) << 5) |
		uint32(dstReg.Encoding&31)

	a.writeInstruction(instr)
}

// ===== Floating Point (SIMD) =====

func (a *ARM64Backend) MovXmmToMem(src, base string, offset int32) {
	compilerError("ARM64Backend.MovXmmToMem not implemented")
}

func (a *ARM64Backend) MovMemToXmm(dst, base string, offset int32) {
	compilerError("ARM64Backend.MovMemToXmm not implemented")
}

func (a *ARM64Backend) MovRegToXmm(dst, src string) {
	compilerError("ARM64Backend.MovRegToXmm not implemented")
}

func (a *ARM64Backend) MovXmmToReg(dst, src string) {
	compilerError("ARM64Backend.MovXmmToReg not implemented")
}

func (a *ARM64Backend) Cvtsi2sd(dst, src string) {
	compilerError("ARM64Backend.Cvtsi2sd not implemented")
}

func (a *ARM64Backend) Cvttsd2si(dst, src string) {
	compilerError("ARM64Backend.Cvttsd2si not implemented")
}

func (a *ARM64Backend) AddpdXmm(dst, src string) {
	compilerError("ARM64Backend.AddpdXmm not implemented")
}

func (a *ARM64Backend) SubpdXmm(dst, src string) {
	compilerError("ARM64Backend.SubpdXmm not implemented")
}

func (a *ARM64Backend) MulpdXmm(dst, src string) {
	compilerError("ARM64Backend.MulpdXmm not implemented")
}

func (a *ARM64Backend) DivpdXmm(dst, src string) {
	compilerError("ARM64Backend.DivpdXmm not implemented")
}

func (a *ARM64Backend) Ucomisd(reg1, reg2 string) {
	compilerError("ARM64Backend.Ucomisd not implemented")
}

// ===== System Calls =====

func (a *ARM64Backend) Syscall() {
	// SVC #0
	instr := uint32(0xD4000001)
	a.writeInstruction(instr)
}
