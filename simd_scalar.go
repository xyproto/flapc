package main

import (
	"fmt"
	"os"
)

// Scalar Double-Precision Floating-Point Operations
// These operate on single float64 values in XMM registers

// AddsdXmm - Add Scalar Double (SSE2)
// addsd xmm, xmm
func (o *Out) AddsdXmm(dst, src string) {
	switch o.machine {
	case MachineX86_64:
		o.addsdX86(dst, src)
	case MachineARM64:
		o.faddScalarARM64(dst, src)
	case MachineRiscv64:
		o.faddScalarRISCV(dst, src)
	}
}

func (o *Out) addsdX86(dst, src string) {
	fmt.Fprintf(os.Stderr, "addsd %s, %s: ", dst, src)

	var dstNum, srcNum int
	fmt.Sscanf(dst, "xmm%d", &dstNum)
	fmt.Sscanf(src, "xmm%d", &srcNum)

	// F2 prefix for scalar double
	o.Write(0xF2)

	// REX if needed
	if dstNum >= 8 || srcNum >= 8 {
		rex := uint8(0x40)
		if dstNum >= 8 {
			rex |= 0x04 // REX.R
		}
		if srcNum >= 8 {
			rex |= 0x01 // REX.B
		}
		o.Write(rex)
	}

	// 0F 58 - ADDSD opcode
	o.Write(0x0F)
	o.Write(0x58)

	// ModR/M
	modrm := uint8(0xC0) | (uint8(dstNum&7) << 3) | uint8(srcNum&7)
	o.Write(modrm)

	fmt.Fprintln(os.Stderr)
}

// SubsdXmm - Subtract Scalar Double
func (o *Out) SubsdXmm(dst, src string) {
	switch o.machine {
	case MachineX86_64:
		o.subsdX86(dst, src)
	}
}

func (o *Out) subsdX86(dst, src string) {
	fmt.Fprintf(os.Stderr, "subsd %s, %s: ", dst, src)

	var dstNum, srcNum int
	fmt.Sscanf(dst, "xmm%d", &dstNum)
	fmt.Sscanf(src, "xmm%d", &srcNum)

	// F2 prefix for scalar double
	o.Write(0xF2)

	// REX if needed
	if dstNum >= 8 || srcNum >= 8 {
		rex := uint8(0x40)
		if dstNum >= 8 {
			rex |= 0x04
		}
		if srcNum >= 8 {
			rex |= 0x01
		}
		o.Write(rex)
	}

	// 0F 5C - SUBSD opcode
	o.Write(0x0F)
	o.Write(0x5C)

	modrm := uint8(0xC0) | (uint8(dstNum&7) << 3) | uint8(srcNum&7)
	o.Write(modrm)

	fmt.Fprintln(os.Stderr)
}

// MulsdXmm - Multiply Scalar Double
func (o *Out) MulsdXmm(dst, src string) {
	switch o.machine {
	case MachineX86_64:
		o.mulsdX86(dst, src)
	}
}

func (o *Out) mulsdX86(dst, src string) {
	fmt.Fprintf(os.Stderr, "mulsd %s, %s: ", dst, src)

	var dstNum, srcNum int
	fmt.Sscanf(dst, "xmm%d", &dstNum)
	fmt.Sscanf(src, "xmm%d", &srcNum)

	// F2 prefix for scalar double
	o.Write(0xF2)

	// REX if needed
	if dstNum >= 8 || srcNum >= 8 {
		rex := uint8(0x40)
		if dstNum >= 8 {
			rex |= 0x04
		}
		if srcNum >= 8 {
			rex |= 0x01
		}
		o.Write(rex)
	}

	// 0F 59 - MULSD opcode
	o.Write(0x0F)
	o.Write(0x59)

	modrm := uint8(0xC0) | (uint8(dstNum&7) << 3) | uint8(srcNum&7)
	o.Write(modrm)

	fmt.Fprintln(os.Stderr)
}

// DivsdXmm - Divide Scalar Double
func (o *Out) DivsdXmm(dst, src string) {
	switch o.machine {
	case MachineX86_64:
		o.divsdX86(dst, src)
	}
}

func (o *Out) divsdX86(dst, src string) {
	fmt.Fprintf(os.Stderr, "divsd %s, %s: ", dst, src)

	var dstNum, srcNum int
	fmt.Sscanf(dst, "xmm%d", &dstNum)
	fmt.Sscanf(src, "xmm%d", &srcNum)

	// F2 prefix for scalar double
	o.Write(0xF2)

	// REX if needed
	if dstNum >= 8 || srcNum >= 8 {
		rex := uint8(0x40)
		if dstNum >= 8 {
			rex |= 0x04
		}
		if srcNum >= 8 {
			rex |= 0x01
		}
		o.Write(rex)
	}

	// 0F 5E - DIVSD opcode
	o.Write(0x0F)
	o.Write(0x5E)

	modrm := uint8(0xC0) | (uint8(dstNum&7) << 3) | uint8(srcNum&7)
	o.Write(modrm)

	fmt.Fprintln(os.Stderr)
}

// ARM64 scalar floating-point operations
func (o *Out) faddScalarARM64(dst, src string) {
	fmt.Fprintf(os.Stderr, "fadd %s, %s (scalar): ", dst, src)
	// FADD Dd, Dn, Dm
	// Implementation would go here
	fmt.Fprintln(os.Stderr)
}

// RISC-V scalar floating-point operations
func (o *Out) faddScalarRISCV(dst, src string) {
	fmt.Fprintf(os.Stderr, "fadd.d %s, %s (scalar): ", dst, src)
	// FADD.D fd, fs1, fs2
	// Implementation would go here
	fmt.Fprintln(os.Stderr)
}
