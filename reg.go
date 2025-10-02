package main

// Register definitions for all supported architectures

type Register struct {
	Name     string
	Size     int   // Size in bits
	Encoding uint8 // Encoding for instruction generation
}

// x86_64 registers
var x86_64Registers = map[string]Register{
	// 64-bit general purpose registers
	"rax": {Name: "rax", Size: 64, Encoding: 0},
	"rcx": {Name: "rcx", Size: 64, Encoding: 1},
	"rdx": {Name: "rdx", Size: 64, Encoding: 2},
	"rbx": {Name: "rbx", Size: 64, Encoding: 3},
	"rsp": {Name: "rsp", Size: 64, Encoding: 4},
	"rbp": {Name: "rbp", Size: 64, Encoding: 5},
	"rsi": {Name: "rsi", Size: 64, Encoding: 6},
	"rdi": {Name: "rdi", Size: 64, Encoding: 7},
	"r8":  {Name: "r8", Size: 64, Encoding: 8},
	"r9":  {Name: "r9", Size: 64, Encoding: 9},
	"r10": {Name: "r10", Size: 64, Encoding: 10},
	"r11": {Name: "r11", Size: 64, Encoding: 11},
	"r12": {Name: "r12", Size: 64, Encoding: 12},
	"r13": {Name: "r13", Size: 64, Encoding: 13},
	"r14": {Name: "r14", Size: 64, Encoding: 14},
	"r15": {Name: "r15", Size: 64, Encoding: 15},

	// 32-bit registers
	"eax": {Name: "eax", Size: 32, Encoding: 0},
	"ecx": {Name: "ecx", Size: 32, Encoding: 1},
	"edx": {Name: "edx", Size: 32, Encoding: 2},
	"ebx": {Name: "ebx", Size: 32, Encoding: 3},
}

// ARM64 registers
var arm64Registers = map[string]Register{
	// 64-bit general purpose registers
	"x0":  {Name: "x0", Size: 64, Encoding: 0},
	"x1":  {Name: "x1", Size: 64, Encoding: 1},
	"x2":  {Name: "x2", Size: 64, Encoding: 2},
	"x3":  {Name: "x3", Size: 64, Encoding: 3},
	"x4":  {Name: "x4", Size: 64, Encoding: 4},
	"x5":  {Name: "x5", Size: 64, Encoding: 5},
	"x6":  {Name: "x6", Size: 64, Encoding: 6},
	"x7":  {Name: "x7", Size: 64, Encoding: 7},
	"x8":  {Name: "x8", Size: 64, Encoding: 8},
	"x9":  {Name: "x9", Size: 64, Encoding: 9},
	"x10": {Name: "x10", Size: 64, Encoding: 10},
	"x11": {Name: "x11", Size: 64, Encoding: 11},
	"x12": {Name: "x12", Size: 64, Encoding: 12},
	"x13": {Name: "x13", Size: 64, Encoding: 13},
	"x14": {Name: "x14", Size: 64, Encoding: 14},
	"x15": {Name: "x15", Size: 64, Encoding: 15},
	"x16": {Name: "x16", Size: 64, Encoding: 16},
	"x17": {Name: "x17", Size: 64, Encoding: 17},
	"x18": {Name: "x18", Size: 64, Encoding: 18},
	"x19": {Name: "x19", Size: 64, Encoding: 19},
	"x20": {Name: "x20", Size: 64, Encoding: 20},
	"x21": {Name: "x21", Size: 64, Encoding: 21},
	"x22": {Name: "x22", Size: 64, Encoding: 22},
	"x23": {Name: "x23", Size: 64, Encoding: 23},
	"x24": {Name: "x24", Size: 64, Encoding: 24},
	"x25": {Name: "x25", Size: 64, Encoding: 25},
	"x26": {Name: "x26", Size: 64, Encoding: 26},
	"x27": {Name: "x27", Size: 64, Encoding: 27},
	"x28": {Name: "x28", Size: 64, Encoding: 28},
	"x29": {Name: "x29", Size: 64, Encoding: 29}, // Frame pointer
	"x30": {Name: "x30", Size: 64, Encoding: 30}, // Link register
	"sp":  {Name: "sp", Size: 64, Encoding: 31},  // Stack pointer

	// 32-bit registers
	"w0": {Name: "w0", Size: 32, Encoding: 0},
	"w1": {Name: "w1", Size: 32, Encoding: 1},
	"w2": {Name: "w2", Size: 32, Encoding: 2},
	"w3": {Name: "w3", Size: 32, Encoding: 3},
}

// RISC-V registers
var riscvRegisters = map[string]Register{
	// General purpose registers
	"x0":  {Name: "x0", Size: 64, Encoding: 0},   // zero
	"x1":  {Name: "x1", Size: 64, Encoding: 1},   // ra (return address)
	"x2":  {Name: "x2", Size: 64, Encoding: 2},   // sp (stack pointer)
	"x3":  {Name: "x3", Size: 64, Encoding: 3},   // gp (global pointer)
	"x4":  {Name: "x4", Size: 64, Encoding: 4},   // tp (thread pointer)
	"x5":  {Name: "x5", Size: 64, Encoding: 5},   // t0
	"x6":  {Name: "x6", Size: 64, Encoding: 6},   // t1
	"x7":  {Name: "x7", Size: 64, Encoding: 7},   // t2
	"x8":  {Name: "x8", Size: 64, Encoding: 8},   // s0/fp
	"x9":  {Name: "x9", Size: 64, Encoding: 9},   // s1
	"x10": {Name: "x10", Size: 64, Encoding: 10}, // a0
	"x11": {Name: "x11", Size: 64, Encoding: 11}, // a1
	"x12": {Name: "x12", Size: 64, Encoding: 12}, // a2
	"x13": {Name: "x13", Size: 64, Encoding: 13}, // a3
	"x14": {Name: "x14", Size: 64, Encoding: 14}, // a4
	"x15": {Name: "x15", Size: 64, Encoding: 15}, // a5
	"x16": {Name: "x16", Size: 64, Encoding: 16}, // a6
	"x17": {Name: "x17", Size: 64, Encoding: 17}, // a7
	"x18": {Name: "x18", Size: 64, Encoding: 18}, // s2
	"x19": {Name: "x19", Size: 64, Encoding: 19}, // s3
	"x20": {Name: "x20", Size: 64, Encoding: 20}, // s4
	"x21": {Name: "x21", Size: 64, Encoding: 21}, // s5
	"x22": {Name: "x22", Size: 64, Encoding: 22}, // s6
	"x23": {Name: "x23", Size: 64, Encoding: 23}, // s7
	"x24": {Name: "x24", Size: 64, Encoding: 24}, // s8
	"x25": {Name: "x25", Size: 64, Encoding: 25}, // s9
	"x26": {Name: "x26", Size: 64, Encoding: 26}, // s10
	"x27": {Name: "x27", Size: 64, Encoding: 27}, // s11
	"x28": {Name: "x28", Size: 64, Encoding: 28}, // t3
	"x29": {Name: "x29", Size: 64, Encoding: 29}, // t4
	"x30": {Name: "x30", Size: 64, Encoding: 30}, // t5
	"x31": {Name: "x31", Size: 64, Encoding: 31}, // t6

	// ABI names
	"zero": {Name: "zero", Size: 64, Encoding: 0},
	"ra":   {Name: "ra", Size: 64, Encoding: 1},
	"sp":   {Name: "sp", Size: 64, Encoding: 2},
	"gp":   {Name: "gp", Size: 64, Encoding: 3},
	"tp":   {Name: "tp", Size: 64, Encoding: 4},
	"t0":   {Name: "t0", Size: 64, Encoding: 5},
	"t1":   {Name: "t1", Size: 64, Encoding: 6},
	"t2":   {Name: "t2", Size: 64, Encoding: 7},
	"s0":   {Name: "s0", Size: 64, Encoding: 8},
	"fp":   {Name: "fp", Size: 64, Encoding: 8}, // Same as s0
	"s1":   {Name: "s1", Size: 64, Encoding: 9},
	"a0":   {Name: "a0", Size: 64, Encoding: 10},
	"a1":   {Name: "a1", Size: 64, Encoding: 11},
	"a2":   {Name: "a2", Size: 64, Encoding: 12},
	"a3":   {Name: "a3", Size: 64, Encoding: 13},
	"a4":   {Name: "a4", Size: 64, Encoding: 14},
	"a5":   {Name: "a5", Size: 64, Encoding: 15},
	"a6":   {Name: "a6", Size: 64, Encoding: 16},
	"a7":   {Name: "a7", Size: 64, Encoding: 17},
}

// GetRegister returns register info for the given machine and register name
func GetRegister(machine Machine, regName string) (Register, bool) {
	switch machine {
	case MachineX86_64:
		reg, ok := x86_64Registers[regName]
		return reg, ok
	case MachineARM64:
		reg, ok := arm64Registers[regName]
		return reg, ok
	case MachineRiscv64:
		reg, ok := riscvRegisters[regName]
		return reg, ok
	default:
		return Register{}, false
	}
}

// IsRegister checks if a string is a valid register name for the given machine
func IsRegister(machine Machine, name string) bool {
	_, ok := GetRegister(machine, name)
	return ok
}
