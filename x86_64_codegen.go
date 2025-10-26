package main

import (
	"fmt"
	"strconv"
)

type X86_64CodeGen struct {
	writer Writer
	eb     *ExecutableBuilder
}

func NewX86_64CodeGen(writer Writer, eb *ExecutableBuilder) *X86_64CodeGen {
	return &X86_64CodeGen{
		writer: writer,
		eb:     eb,
	}
}

func (x *X86_64CodeGen) write(b uint8) {
	x.writer.(*BufferWrapper).Write(b)
}

func (x *X86_64CodeGen) writeUnsigned(i uint) {
	x.writer.(*BufferWrapper).WriteUnsigned(i)
}

func (x *X86_64CodeGen) emit(bytes []byte) {
	for _, b := range bytes {
		x.write(b)
	}
}

func (x *X86_64CodeGen) Ret() {
	x.write(0xC3)
}

func (x *X86_64CodeGen) Syscall() {
	x.write(0x0F)
	x.write(0x05)
}

func (x *X86_64CodeGen) CallSymbol(symbol string) {
	x.write(0xE8)

	callPos := x.eb.text.Len()
	x.writeUnsigned(0x00000000)

	x.eb.callPatches = append(x.eb.callPatches, CallPatch{
		position:   callPos,
		targetName: symbol,
	})
}

func (x *X86_64CodeGen) CallRelative(offset int32) {
	x.write(0xE8)
	x.writeUnsigned(uint(offset))
}

func (x *X86_64CodeGen) CallRegister(reg string) {
	r, ok := x86_64Registers[reg]
	if !ok {
		compilerError("Unknown register: %s", reg)
	}
	x.write(0xFF)
	x.write(0xD0 + r.Encoding)
}

func (x *X86_64CodeGen) JumpUnconditional(offset int32) {
	x.write(0xE9)
	x.writeUnsigned(uint(offset))
}

func (x *X86_64CodeGen) JumpConditional(condition JumpCondition, offset int32) {
	var opcode byte
	switch condition {
	case JumpEqual:
		opcode = 0x84
	case JumpNotEqual:
		opcode = 0x85
	case JumpLess:
		opcode = 0x8C
	case JumpLessOrEqual:
		opcode = 0x8E
	case JumpGreater:
		opcode = 0x8F
	case JumpGreaterOrEqual:
		opcode = 0x8D
	case JumpAbove:
		opcode = 0x87
	case JumpAboveOrEqual:
		opcode = 0x83
	case JumpBelow:
		opcode = 0x82
	case JumpBelowOrEqual:
		opcode = 0x86
	default:
		compilerError("Unknown jump condition: %v", condition)
	}
	x.write(0x0F)
	x.write(opcode)
	x.writeUnsigned(uint(offset))
}

// ===== Data Movement =====

func (x *X86_64CodeGen) MovRegToReg(dst, src string) {
	dstIsXMM := (len(dst) >= 3 && dst[:3] == "xmm") || (len(dst) >= 1 && dst[:1] == "v")
	srcIsXMM := (len(src) >= 3 && src[:3] == "xmm") || (len(src) >= 1 && src[:1] == "v")
	if dstIsXMM && srcIsXMM {
		var dstNum, srcNum int
		fmt.Sscanf(dst, "xmm%d", &dstNum)
		fmt.Sscanf(src, "xmm%d", &srcNum)

		x.write(0xF2)

		if dstNum >= 8 || srcNum >= 8 {
			rex := uint8(0x40)
			if dstNum >= 8 {
				rex |= 0x04
			}
			if srcNum >= 8 {
				rex |= 0x01
			}
			x.write(rex)
		}

		x.write(0x0F)
		x.write(0x10)

		modrm := uint8(0xC0) | (uint8(dstNum&7) << 3) | uint8(srcNum&7)
		x.write(modrm)
		return
	}

	dstReg, dstOk := x86_64Registers[dst]
	srcReg, srcOk := x86_64Registers[src]

	if !dstOk || !srcOk {
		return
	}

	needsRex := dstReg.Size == 64 || srcReg.Size == 64 || dstReg.Encoding >= 8 || srcReg.Encoding >= 8
	if needsRex {
		rex := uint8(0x40)
		if dstReg.Size == 64 || srcReg.Size == 64 {
			rex |= 0x08
		}
		if srcReg.Encoding >= 8 {
			rex |= 0x04
		}
		if dstReg.Encoding >= 8 {
			rex |= 0x01
		}
		x.write(rex)
	}

	x.write(0x89)

	modrm := uint8(0xC0) | ((srcReg.Encoding & 7) << 3) | (dstReg.Encoding & 7)
	x.write(modrm)
}

func (x *X86_64CodeGen) MovImmToReg(dst, imm string) {
	dstReg, dstOk := x86_64Registers[dst]
	if !dstOk {
		return
	}

	var immVal uint64
	if val, err := strconv.ParseInt(imm, 0, 64); err == nil {
		immVal = uint64(val)
	} else if val, err := strconv.ParseUint(imm, 0, 64); err == nil {
		immVal = val
	}

	if dstReg.Size == 64 {
		rex := uint8(0x48)
		if dstReg.Encoding >= 8 {
			rex |= 0x01
		}
		x.write(rex)
	}

	x.write(0xC7)

	modrm := uint8(0xC0) | (dstReg.Encoding & 7)
	x.write(modrm)

	x.writeUnsigned(uint(immVal))
}

func (x *X86_64CodeGen) MovMemToReg(dst, symbol string, offset int32) {
	dstReg, dstOk := x86_64Registers[dst]
	if !dstOk {
		return
	}

	rex := uint8(0x48)
	if dstReg.Encoding >= 8 {
		rex |= 0x04
	}
	x.write(rex)

	x.write(0x8B)

	modrm := uint8(0x05) | ((dstReg.Encoding & 7) << 3)
	x.write(modrm)

	displacementOffset := uint64(x.eb.text.Len())
	x.eb.pcRelocations = append(x.eb.pcRelocations, PCRelocation{
		offset:     displacementOffset,
		symbolName: symbol,
	})

	x.writeUnsigned(uint(offset))
}

func (x *X86_64CodeGen) MovRegToMem(src, symbol string, offset int32) {
	srcReg, srcOk := x86_64Registers[src]
	if !srcOk {
		return
	}

	rex := uint8(0x48)
	if srcReg.Encoding >= 8 {
		rex |= 0x04
	}
	x.write(rex)

	x.write(0x89)

	modrm := uint8(0x05) | ((srcReg.Encoding & 7) << 3)
	x.write(modrm)

	displacementOffset := uint64(x.eb.text.Len())
	x.eb.pcRelocations = append(x.eb.pcRelocations, PCRelocation{
		offset:     displacementOffset,
		symbolName: symbol,
	})

	x.writeUnsigned(uint(offset))
}

// ===== Integer Arithmetic =====

func (x *X86_64CodeGen) AddRegToReg(dst, src string) {
	dstReg, dstOk := x86_64Registers[dst]
	srcReg, srcOk := x86_64Registers[src]
	if !dstOk || !srcOk {
		return
	}

	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01
	}
	if (srcReg.Encoding & 8) != 0 {
		rex |= 0x04
	}
	x.write(rex)

	x.write(0x01)

	modrm := uint8(0xC0) | ((srcReg.Encoding & 7) << 3) | (dstReg.Encoding & 7)
	x.write(modrm)
}

func (x *X86_64CodeGen) AddImmToReg(dst string, imm int64) {
	dstReg, dstOk := x86_64Registers[dst]
	if !dstOk {
		return
	}

	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01
	}
	x.write(rex)

	if imm >= -128 && imm <= 127 {
		x.write(0x83)
		modrm := uint8(0xC0) | (dstReg.Encoding & 7)
		x.write(modrm)
		x.write(uint8(imm & 0xFF))
	} else {
		x.write(0x81)
		modrm := uint8(0xC0) | (dstReg.Encoding & 7)
		x.write(modrm)

		imm32 := uint32(imm)
		x.write(uint8(imm32 & 0xFF))
		x.write(uint8((imm32 >> 8) & 0xFF))
		x.write(uint8((imm32 >> 16) & 0xFF))
		x.write(uint8((imm32 >> 24) & 0xFF))
	}
}

func (x *X86_64CodeGen) SubRegToReg(dst, src string) {
	dstReg, dstOk := x86_64Registers[dst]
	srcReg, srcOk := x86_64Registers[src]
	if !dstOk || !srcOk {
		return
	}

	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01
	}
	if (srcReg.Encoding & 8) != 0 {
		rex |= 0x04
	}
	x.write(rex)

	x.write(0x29)

	modrm := uint8(0xC0) | ((srcReg.Encoding & 7) << 3) | (dstReg.Encoding & 7)
	x.write(modrm)
}

func (x *X86_64CodeGen) SubImmFromReg(dst string, imm int64) {
	dstReg, dstOk := x86_64Registers[dst]
	if !dstOk {
		return
	}

	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01
	}
	x.write(rex)

	if imm >= -128 && imm <= 127 {
		x.write(0x83)
		modrm := uint8(0xE8) | (dstReg.Encoding & 7)
		x.write(modrm)
		x.write(uint8(imm & 0xFF))
	} else {
		x.write(0x81)
		modrm := uint8(0xE8) | (dstReg.Encoding & 7)
		x.write(modrm)

		imm32 := uint32(imm)
		x.write(uint8(imm32 & 0xFF))
		x.write(uint8((imm32 >> 8) & 0xFF))
		x.write(uint8((imm32 >> 16) & 0xFF))
		x.write(uint8((imm32 >> 24) & 0xFF))
	}
}

func (x *X86_64CodeGen) MulRegToReg(dst, src string) {
	dstReg, dstOk := x86_64Registers[dst]
	srcReg, srcOk := x86_64Registers[src]
	if !dstOk || !srcOk {
		return
	}

	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x04
	}
	if (srcReg.Encoding & 8) != 0 {
		rex |= 0x01
	}
	x.write(rex)

	x.write(0x0F)
	x.write(0xAF)

	modrm := uint8(0xC0) | ((dstReg.Encoding & 7) << 3) | (srcReg.Encoding & 7)
	x.write(modrm)
}

func (x *X86_64CodeGen) DivRegToReg(dst, src string) {
	srcReg, srcOk := x86_64Registers[src]
	if !srcOk {
		return
	}

	if dst != "rax" {
		x.MovRegToReg("rax", dst)
	}

	x.write(0x48)
	x.write(0x99)

	rex := uint8(0x48)
	if (srcReg.Encoding & 8) != 0 {
		rex |= 0x01
	}
	x.write(rex)
	x.write(0xF7)
	modrm := uint8(0xF8) | (srcReg.Encoding & 7)
	x.write(modrm)

	if dst != "rax" {
		x.MovRegToReg(dst, "rax")
	}
}

func (x *X86_64CodeGen) IncReg(dst string) {
	regInfo, ok := x86_64Registers[dst]
	if !ok {
		return
	}

	rex := uint8(0x48)
	if (regInfo.Encoding & 8) != 0 {
		rex |= 0x01
	}
	x.write(rex)

	x.write(0xFF)

	modrm := uint8(0xC0) | (regInfo.Encoding & 7)
	x.write(modrm)
}

func (x *X86_64CodeGen) DecReg(dst string) {
	regInfo, ok := x86_64Registers[dst]
	if !ok {
		return
	}

	rex := uint8(0x48)
	if (regInfo.Encoding & 8) != 0 {
		rex |= 0x01
	}
	x.write(rex)

	x.write(0xFF)

	modrm := uint8(0xC8) | (regInfo.Encoding & 7)
	x.write(modrm)
}

func (x *X86_64CodeGen) NegReg(dst string) {
	dstReg, dstOk := x86_64Registers[dst]
	if !dstOk {
		return
	}

	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01
	}
	x.write(rex)

	x.write(0xF7)

	modrm := uint8(0xD8) | (dstReg.Encoding & 7)
	x.write(modrm)
}

// ===== Bitwise Operations =====

func (x *X86_64CodeGen) XorRegWithReg(dst, src string) {
	dstReg, dstOk := x86_64Registers[dst]
	srcReg, srcOk := x86_64Registers[src]
	if !dstOk || !srcOk {
		return
	}

	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01
	}
	if (srcReg.Encoding & 8) != 0 {
		rex |= 0x04
	}
	x.write(rex)

	x.write(0x31)

	modrm := uint8(0xC0) | ((srcReg.Encoding & 7) << 3) | (dstReg.Encoding & 7)
	x.write(modrm)
}

func (x *X86_64CodeGen) XorRegWithImm(dst string, imm int64) {
	dstReg, dstOk := x86_64Registers[dst]
	if !dstOk {
		return
	}

	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01
	}
	x.write(rex)

	imm32 := int32(imm)
	if imm32 >= -128 && imm32 <= 127 {
		x.write(0x83)
		modrm := uint8(0xF0) | (dstReg.Encoding & 7)
		x.write(modrm)
		x.write(uint8(imm32 & 0xFF))
	} else {
		x.write(0x81)
		modrm := uint8(0xF0) | (dstReg.Encoding & 7)
		x.write(modrm)

		x.write(uint8(imm32 & 0xFF))
		x.write(uint8((imm32 >> 8) & 0xFF))
		x.write(uint8((imm32 >> 16) & 0xFF))
		x.write(uint8((imm32 >> 24) & 0xFF))
	}
}

func (x *X86_64CodeGen) AndRegWithReg(dst, src string) {
	dstReg, dstOk := x86_64Registers[dst]
	srcReg, srcOk := x86_64Registers[src]
	if !dstOk || !srcOk {
		return
	}

	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01
	}
	if (srcReg.Encoding & 8) != 0 {
		rex |= 0x04
	}
	x.write(rex)

	x.write(0x21)

	modrm := uint8(0xC0) | ((srcReg.Encoding & 7) << 3) | (dstReg.Encoding & 7)
	x.write(modrm)
}

func (x *X86_64CodeGen) OrRegWithReg(dst, src string) {
	dstReg, dstOk := x86_64Registers[dst]
	srcReg, srcOk := x86_64Registers[src]
	if !dstOk || !srcOk {
		return
	}

	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01
	}
	if (srcReg.Encoding & 8) != 0 {
		rex |= 0x04
	}
	x.write(rex)

	x.write(0x09)

	modrm := uint8(0xC0) | ((srcReg.Encoding & 7) << 3) | (dstReg.Encoding & 7)
	x.write(modrm)
}

func (x *X86_64CodeGen) NotReg(dst string) {
	dstReg, dstOk := x86_64Registers[dst]
	if !dstOk {
		return
	}

	rex := uint8(0x48)
	if (dstReg.Encoding & 8) != 0 {
		rex |= 0x01
	}
	x.write(rex)

	x.write(0xF7)

	modrm := uint8(0xD0) | (dstReg.Encoding & 7)
	x.write(modrm)
}

// ===== Stack Operations =====

func (x *X86_64CodeGen) PushReg(reg string) {
	regInfo, regOk := x86_64Registers[reg]
	if !regOk {
		return
	}

	if regInfo.Encoding >= 8 {
		x.write(0x41)
		x.write(0x50 + uint8(regInfo.Encoding&7))
	} else {
		x.write(0x50 + uint8(regInfo.Encoding))
	}
}

func (x *X86_64CodeGen) PopReg(reg string) {
	regInfo, regOk := x86_64Registers[reg]
	if !regOk {
		return
	}

	if regInfo.Encoding >= 8 {
		x.write(0x41)
		x.write(0x58 + uint8(regInfo.Encoding&7))
	} else {
		x.write(0x58 + uint8(regInfo.Encoding))
	}
}

// ===== Comparisons =====

func (x *X86_64CodeGen) CmpRegToReg(reg1, reg2 string) {
	src1Reg, src1Ok := x86_64Registers[reg1]
	src2Reg, src2Ok := x86_64Registers[reg2]
	if !src1Ok || !src2Ok {
		return
	}

	rex := uint8(0x48)
	if (src1Reg.Encoding & 8) != 0 {
		rex |= 0x01
	}
	if (src2Reg.Encoding & 8) != 0 {
		rex |= 0x04
	}
	x.write(rex)

	x.write(0x39)

	modrm := uint8(0xC0) | ((src2Reg.Encoding & 7) << 3) | (src1Reg.Encoding & 7)
	x.write(modrm)
}

func (x *X86_64CodeGen) CmpRegToImm(reg string, imm int64) {
	regInfo, regOk := x86_64Registers[reg]
	if !regOk {
		return
	}

	rex := uint8(0x48)
	if (regInfo.Encoding & 8) != 0 {
		rex |= 0x01
	}
	x.write(rex)

	if imm >= -128 && imm <= 127 {
		x.write(0x83)
		modrm := uint8(0xF8) | (regInfo.Encoding & 7)
		x.write(modrm)
		x.write(uint8(imm & 0xFF))
	} else {
		x.write(0x81)
		modrm := uint8(0xF8) | (regInfo.Encoding & 7)
		x.write(modrm)

		imm32 := uint32(imm)
		x.write(uint8(imm32 & 0xFF))
		x.write(uint8((imm32 >> 8) & 0xFF))
		x.write(uint8((imm32 >> 16) & 0xFF))
		x.write(uint8((imm32 >> 24) & 0xFF))
	}
}

// ===== Address Calculation =====

func (x *X86_64CodeGen) LeaSymbolToReg(dst, symbol string) {
	dstReg, dstOk := x86_64Registers[dst]
	if !dstOk {
		return
	}

	rex := uint8(0x48)
	if dstReg.Encoding >= 8 {
		rex |= 0x04
	}
	x.write(rex)

	x.write(0x8D)

	modrm := uint8(0x05) | ((dstReg.Encoding & 7) << 3)
	x.write(modrm)

	displacementOffset := uint64(x.eb.text.Len())
	x.eb.pcRelocations = append(x.eb.pcRelocations, PCRelocation{
		offset:     displacementOffset,
		symbolName: symbol,
	})

	x.writeUnsigned(0xDEADBEEF)
}

func (x *X86_64CodeGen) LeaImmToReg(dst, base string, offset int32) {
	dstReg, dstOk := x86_64Registers[dst]
	baseReg, baseOk := x86_64Registers[base]

	if !dstOk || !baseOk {
		return
	}

	rex := uint8(0x48)
	if dstReg.Encoding >= 8 {
		rex |= 0x04
	}
	if baseReg.Encoding >= 8 {
		rex |= 0x01
	}
	x.write(rex)

	x.write(0x8D)

	offset64 := int64(offset)
	if offset64 == 0 && (baseReg.Encoding&7) != 5 {
		modrm := uint8(0x00) | ((dstReg.Encoding & 7) << 3) | (baseReg.Encoding & 7)
		x.write(modrm)
	} else if offset64 >= -128 && offset64 <= 127 {
		modrm := uint8(0x40) | ((dstReg.Encoding & 7) << 3) | (baseReg.Encoding & 7)
		x.write(modrm)
		x.write(uint8(offset64 & 0xFF))
	} else {
		modrm := uint8(0x80) | ((dstReg.Encoding & 7) << 3) | (baseReg.Encoding & 7)
		x.write(modrm)
		x.writeUnsigned(uint(offset64 & 0xFFFFFFFF))
	}
}

// ===== Floating Point (SIMD) =====

func (x *X86_64CodeGen) MovXmmToMem(src, base string, offset int32) {
	var xmmNum int
	fmt.Sscanf(src, "xmm%d", &xmmNum)

	baseReg, _ := x86_64Registers[base]

	x.write(0xF2)

	rex := uint8(0x48)
	if xmmNum >= 8 {
		rex |= 0x04
	}
	if baseReg.Encoding >= 8 {
		rex |= 0x01
	}
	x.write(rex)

	x.write(0x0F)
	x.write(0x11)

	offset64 := int64(offset)
	if offset64 == 0 && (baseReg.Encoding&7) != 5 {
		modrm := uint8(0x00) | (uint8(xmmNum&7) << 3) | (baseReg.Encoding & 7)
		x.write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			x.write(0x24)
		}
	} else if offset64 < 128 && offset64 >= -128 {
		modrm := uint8(0x40) | (uint8(xmmNum&7) << 3) | (baseReg.Encoding & 7)
		x.write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			x.write(0x24)
		}
		x.write(uint8(offset64))
	} else {
		modrm := uint8(0x80) | (uint8(xmmNum&7) << 3) | (baseReg.Encoding & 7)
		x.write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			x.write(0x24)
		}
		x.writeUnsigned(uint(offset64))
	}
}

func (x *X86_64CodeGen) MovMemToXmm(dst, base string, offset int32) {
	var xmmNum int
	fmt.Sscanf(dst, "xmm%d", &xmmNum)

	baseReg, _ := x86_64Registers[base]

	x.write(0xF2)

	rex := uint8(0x48)
	if xmmNum >= 8 {
		rex |= 0x04
	}
	if baseReg.Encoding >= 8 {
		rex |= 0x01
	}
	x.write(rex)

	x.write(0x0F)
	x.write(0x10)

	offset64 := int64(offset)
	if offset64 == 0 && (baseReg.Encoding&7) != 5 {
		modrm := uint8(0x00) | (uint8(xmmNum&7) << 3) | (baseReg.Encoding & 7)
		x.write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			x.write(0x24)
		}
	} else if offset64 < 128 && offset64 >= -128 {
		modrm := uint8(0x40) | (uint8(xmmNum&7) << 3) | (baseReg.Encoding & 7)
		x.write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			x.write(0x24)
		}
		x.write(uint8(offset64))
	} else {
		modrm := uint8(0x80) | (uint8(xmmNum&7) << 3) | (baseReg.Encoding & 7)
		x.write(modrm)
		if (baseReg.Encoding & 7) == 4 {
			x.write(0x24)
		}
		x.writeUnsigned(uint(offset64))
	}
}

func (x *X86_64CodeGen) MovRegToXmm(dst, src string) {
	srcReg, srcOk := x86_64Registers[src]
	if !srcOk {
		return
	}

	var xmmNum int
	fmt.Sscanf(dst, "xmm%d", &xmmNum)

	x.write(0x66)

	rex := uint8(0x48)
	if xmmNum >= 8 {
		rex |= 0x04
	}
	if srcReg.Encoding >= 8 {
		rex |= 0x01
	}
	x.write(rex)

	x.write(0x0F)
	x.write(0x6E)

	modrm := uint8(0xC0) | (uint8(xmmNum&7) << 3) | (srcReg.Encoding & 7)
	x.write(modrm)
}

func (x *X86_64CodeGen) MovXmmToReg(dst, src string) {
	dstReg, dstOk := x86_64Registers[dst]
	if !dstOk {
		return
	}

	var xmmNum int
	fmt.Sscanf(src, "xmm%d", &xmmNum)

	x.write(0x66)

	rex := uint8(0x48)
	if dstReg.Encoding >= 8 {
		rex |= 0x04
	}
	if xmmNum >= 8 {
		rex |= 0x01
	}
	x.write(rex)

	x.write(0x0F)
	x.write(0x7E)

	modrm := uint8(0xC0) | ((dstReg.Encoding & 7) << 3) | uint8(xmmNum&7)
	x.write(modrm)
}

func (x *X86_64CodeGen) Cvtsi2sd(dst, src string) {
	srcReg, srcOk := x86_64Registers[src]
	if !srcOk {
		return
	}

	var xmmNum int
	fmt.Sscanf(dst, "xmm%d", &xmmNum)

	x.write(0xF2)

	rex := uint8(0x48)
	if xmmNum >= 8 {
		rex |= 0x04
	}
	if srcReg.Encoding >= 8 {
		rex |= 0x01
	}
	x.write(rex)

	x.write(0x0F)
	x.write(0x2A)

	modrm := uint8(0xC0) | (uint8(xmmNum&7) << 3) | (srcReg.Encoding & 7)
	x.write(modrm)
}

func (x *X86_64CodeGen) Cvttsd2si(dst, src string) {
	dstReg, _ := x86_64Registers[dst]

	var xmmNum int
	fmt.Sscanf(src, "xmm%d", &xmmNum)

	x.write(0xF2)

	rex := uint8(0x48)
	if dstReg.Encoding >= 8 {
		rex |= 0x04
	}
	if xmmNum >= 8 {
		rex |= 0x01
	}
	x.write(rex)

	x.write(0x0F)
	x.write(0x2C)

	modrm := uint8(0xC0) | ((dstReg.Encoding & 7) << 3) | uint8(xmmNum&7)
	x.write(modrm)
}

func (x *X86_64CodeGen) AddpdXmm(dst, src string) {
	var dstNum, srcNum int
	fmt.Sscanf(dst, "xmm%d", &dstNum)
	fmt.Sscanf(src, "xmm%d", &srcNum)

	x.write(0x66)

	if dstNum >= 8 || srcNum >= 8 {
		rex := uint8(0x40)
		if dstNum >= 8 {
			rex |= 0x04
		}
		if srcNum >= 8 {
			rex |= 0x01
		}
		x.write(rex)
	}

	x.write(0x0F)
	x.write(0x58)

	modrm := uint8(0xC0) | (uint8(dstNum&7) << 3) | uint8(srcNum&7)
	x.write(modrm)
}

func (x *X86_64CodeGen) SubpdXmm(dst, src string) {
	var dstNum, srcNum int
	fmt.Sscanf(dst, "xmm%d", &dstNum)
	fmt.Sscanf(src, "xmm%d", &srcNum)

	x.write(0x66)

	if dstNum >= 8 || srcNum >= 8 {
		rex := uint8(0x40)
		if dstNum >= 8 {
			rex |= 0x04
		}
		if srcNum >= 8 {
			rex |= 0x01
		}
		x.write(rex)
	}

	x.write(0x0F)
	x.write(0x5C)

	modrm := uint8(0xC0) | (uint8(dstNum&7) << 3) | uint8(srcNum&7)
	x.write(modrm)
}

func (x *X86_64CodeGen) MulpdXmm(dst, src string) {
	var dstNum, srcNum int
	fmt.Sscanf(dst, "xmm%d", &dstNum)
	fmt.Sscanf(src, "xmm%d", &srcNum)

	x.write(0x66)

	if dstNum >= 8 || srcNum >= 8 {
		rex := uint8(0x40)
		if dstNum >= 8 {
			rex |= 0x04
		}
		if srcNum >= 8 {
			rex |= 0x01
		}
		x.write(rex)
	}

	x.write(0x0F)
	x.write(0x59)

	modrm := uint8(0xC0) | (uint8(dstNum&7) << 3) | uint8(srcNum&7)
	x.write(modrm)
}

func (x *X86_64CodeGen) DivpdXmm(dst, src string) {
	var dstNum, srcNum int
	fmt.Sscanf(dst, "xmm%d", &dstNum)
	fmt.Sscanf(src, "xmm%d", &srcNum)

	x.write(0x66)

	if dstNum >= 8 || srcNum >= 8 {
		rex := uint8(0x40)
		if dstNum >= 8 {
			rex |= 0x04
		}
		if srcNum >= 8 {
			rex |= 0x01
		}
		x.write(rex)
	}

	x.write(0x0F)
	x.write(0x5E)

	modrm := uint8(0xC0) | (uint8(dstNum&7) << 3) | uint8(srcNum&7)
	x.write(modrm)
}

func (x *X86_64CodeGen) Ucomisd(reg1, reg2 string) {
	var xmm1Num, xmm2Num int
	fmt.Sscanf(reg1, "xmm%d", &xmm1Num)
	fmt.Sscanf(reg2, "xmm%d", &xmm2Num)

	x.write(0x66)

	rex := uint8(0)
	if xmm1Num >= 8 || xmm2Num >= 8 {
		rex = 0x40
		if xmm1Num >= 8 {
			rex |= 0x04
		}
		if xmm2Num >= 8 {
			rex |= 0x01
		}
		x.write(rex)
	}

	x.write(0x0F)
	x.write(0x2E)

	modrm := uint8(0xC0) | (uint8(xmm1Num&7) << 3) | uint8(xmm2Num&7)
	x.write(modrm)
}
