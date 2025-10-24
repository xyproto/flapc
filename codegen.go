package main

// CPU Abstraction Interface - Phase 1 Foundation
//
// This file introduces the CodeGenerator interface to eliminate the 100+
// switch statements scattered across ~80 files in the codebase.
//
// MIGRATION STRATEGY:
//
// Current state: Every Out method looks like this:
//   func (o *Out) SomeInstruction(args...) {
//       switch o.machine.Arch {
//       case ArchX86_64: o.someInstructionX86(args...)
//       case ArchARM64:  o.someInstructionARM64(args...)
//       case ArchRiscv64: o.someInstructionRISCV(args...)
//       }
//   }
//
// Target state:
//   func (o *Out) SomeInstruction(args...) {
//       o.backend.SomeInstruction(args...)
//   }
//
// Migration steps:
// 1. ✓ Define CodeGenerator interface (below)
// 2. Create architecture-specific implementations (X86_64CodeGen, ARM64CodeGen, etc.)
// 3. Move o.xxxX86() methods into X86_64CodeGen (remove prefix)
// 4. Update Out to delegate to backend
// 5. Remove old switch-based methods
//
// Benefits:
// - Clean separation: architecture code is isolated
// - Easy to add new architectures: just implement interface
// - Better testability: can unit test each backend
// - Eliminates duplication: 100+ switch statements removed

type CodeGenerator interface {
	// ===== Data Movement =====
	MovRegToReg(dst, src string)
	MovImmToReg(dst, imm string)
	MovMemToReg(dst, symbol string, offset int32)
	MovRegToMem(src, symbol string, offset int32)
	
	// ===== Integer Arithmetic =====
	AddRegToReg(dst, src string)
	AddImmToReg(dst string, imm int64)
	SubRegToReg(dst, src string)
	SubImmFromReg(dst string, imm int64)
	MulRegToReg(dst, src string)
	DivRegToReg(dst, src string)
	IncReg(dst string)
	DecReg(dst string)
	NegReg(dst string)
	
	// ===== Bitwise Operations =====
	XorRegWithReg(dst, src string)
	XorRegWithImm(dst string, imm int64)
	AndRegWithReg(dst, src string)
	OrRegWithReg(dst, src string)
	NotReg(dst string)
	
	// ===== Stack Operations =====
	PushReg(reg string)
	PopReg(reg string)
	
	// ===== Control Flow =====
	JumpConditional(condition JumpCondition, offset int32)
	JumpUnconditional(offset int32)
	CallSymbol(symbol string)
	CallRelative(offset int32)
	CallRegister(reg string)
	Ret()
	
	// ===== Comparisons =====
	CmpRegToReg(reg1, reg2 string)
	CmpRegToImm(reg string, imm int64)
	
	// ===== Address Calculation =====
	LeaSymbolToReg(dst, symbol string)
	LeaImmToReg(dst, base string, offset int32)
	
	// ===== Floating Point (SIMD) =====
	MovXmmToMem(src, base string, offset int32)
	MovMemToXmm(dst, base string, offset int32)
	MovRegToXmm(dst, src string)
	MovXmmToReg(dst, src string)
	Cvtsi2sd(dst, src string)
	Cvttsd2si(dst, src string)
	AddpdXmm(dst, src string)
	SubpdXmm(dst, src string)
	MulpdXmm(dst, src string)
	DivpdXmm(dst, src string)
	Ucomisd(reg1, reg2 string)
	
	// ===== System Calls =====
	Syscall()
	
	// TODO: Add remaining methods as needed during migration:
	// - Shift operations (ShlRegByImm, ShrRegByImm, etc.)
	// - More SIMD operations (VAddPDVectorToVector, etc.)
	// - AVX-512/SVE2/RVV vector operations
	// - Additional memory addressing modes
}

// ===== Future Implementation Skeletons =====
//
// Each architecture will implement the interface:
//
// type X86_64CodeGen struct {
//     writer Writer
//     eb     *ExecutableBuilder
// }
//
// func (x *X86_64CodeGen) MovRegToReg(dst, src string) {
//     // Move o.movX86RegToReg() implementation here
//     // ... x86-64 specific encoding ...
// }
//
// type ARM64CodeGen struct {
//     writer Writer
//     eb     *ExecutableBuilder
// }
//
// func (a *ARM64CodeGen) MovRegToReg(dst, src string) {
//     // Move o.movARM64RegToReg() implementation here
//     // ... ARM64 specific encoding ...
// }
//
// type RISCV64CodeGen struct {
//     writer Writer
//     eb     *ExecutableBuilder
// }
//
// func (r *RISCV64CodeGen) MovRegToReg(dst, src string) {
//     // Move o.movRISCVRegToReg() implementation here
//     // ... RISC-V specific encoding ...
// }

// NewCodeGenerator creates the appropriate backend for the given architecture
func NewCodeGenerator(arch Arch, w Writer, eb *ExecutableBuilder) CodeGenerator {
	switch arch {
	case ArchX86_64:
		return NewX86_64CodeGen(w, eb)
	case ArchARM64:
		return NewARM64Backend(w, eb)
	case ArchRiscv64:
		return NewRISCV64Backend(w, eb)
	default:
		compilerError("unsupported architecture: %v", arch)
		return nil
	}
}

// ===== Migration Progress Tracker =====
//
// Phase 1: ✓ Interface definition and architecture design
// Phase 2: ✓ Implement X86_64CodeGen (x86_64_codegen.go - 1028 lines)
// Phase 3: ✓ Implement ARM64Backend (arm64_backend.go - implements CodeGenerator interface)
// Phase 4: ✓ Implement RISCV64Backend (riscv64_backend.go - implements CodeGenerator interface)
// Phase 5: [ ] Add backend field to Out struct
// Phase 6: [ ] Update Out methods to use backend
// Phase 7: [ ] Remove old switch-based methods
// Phase 8: [ ] Update tests and verify all architectures work
//
// Estimated effort: Large (80+ files affected, but mechanical refactoring)
// Benefits: Cleaner code, easier to maintain, better testability
