package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// A tiny compiler for x86_64, aarch64, and riscv64 for Linux, macOS, FreeBSD

const versionString = "flapc 1.0.0"

// Architecture type
type Arch int

const (
	ArchX86_64 Arch = iota
	ArchARM64
	ArchRiscv64
)

func (a Arch) String() string {
	switch a {
	case ArchX86_64:
		return "x86_64"
	case ArchARM64:
		return "aarch64"
	case ArchRiscv64:
		return "riscv64"
	default:
		return "unknown"
	}
}

// ParseArch parses an architecture string (like GOARCH values)
func ParseArch(s string) (Arch, error) {
	switch strings.ToLower(s) {
	case "x86_64", "amd64", "x86-64":
		return ArchX86_64, nil
	case "aarch64", "arm64":
		return ArchARM64, nil
	case "riscv64", "riscv", "rv64":
		return ArchRiscv64, nil
	default:
		return 0, fmt.Errorf("unsupported architecture: %s (supported: amd64, arm64, riscv64)", s)
	}
}

// OS type
type OS int

const (
	OSLinux OS = iota
	OSDarwin
	OSFreeBSD
)

func (o OS) String() string {
	switch o {
	case OSLinux:
		return "linux"
	case OSDarwin:
		return "darwin"
	case OSFreeBSD:
		return "freebsd"
	default:
		return "unknown"
	}
}

// ParseOS parses an OS string (like GOOS values)
func ParseOS(s string) (OS, error) {
	switch strings.ToLower(s) {
	case "linux":
		return OSLinux, nil
	case "darwin", "macos":
		return OSDarwin, nil
	case "freebsd":
		return OSFreeBSD, nil
	default:
		return 0, fmt.Errorf("unsupported OS: %s (supported: linux, darwin, freebsd)", s)
	}
}

// Platform represents a target platform (architecture + OS)
type Platform struct {
	Arch Arch
	OS   OS
}

// String returns a string representation like "aarch64" (just the arch for compatibility)
func (p Platform) String() string {
	return p.Arch.String()
}

// FullString returns the full platform string like "arm64-darwin"
func (p Platform) FullString() string {
	archStr := p.Arch.String()
	// Convert aarch64 -> arm64 for cleaner output
	if p.Arch == ArchARM64 {
		archStr = "arm64"
	} else if p.Arch == ArchX86_64 {
		archStr = "amd64"
	}
	return archStr + "-" + p.OS.String()
}

// IsMachO returns true if this platform uses Mach-O format
func (p Platform) IsMachO() bool {
	return p.OS == OSDarwin
}

// IsELF returns true if this platform uses ELF format
func (p Platform) IsELF() bool {
	return p.OS == OSLinux || p.OS == OSFreeBSD
}

// GetDefaultPlatform returns the platform for the current runtime
func GetDefaultPlatform() Platform {
	var arch Arch
	switch runtime.GOARCH {
	case "amd64":
		arch = ArchX86_64
	case "arm64":
		arch = ArchARM64
	case "riscv64":
		arch = ArchRiscv64
	default:
		arch = ArchX86_64 // fallback
	}

	var os OS
	switch runtime.GOOS {
	case "linux":
		os = OSLinux
	case "darwin":
		os = OSDarwin
	case "freebsd":
		os = OSFreeBSD
	default:
		os = OSLinux // fallback
	}

	return Platform{Arch: arch, OS: os}
}

// Deprecated: Use ParseArch and ParseOS separately
func StringToMachine(s string) (Platform, error) {
	// For backward compatibility, try to parse as "arch" or "arch-os"
	parts := strings.Split(s, "-")
	arch, err := ParseArch(parts[0])
	if err != nil {
		return Platform{}, err
	}

	var os OS
	if len(parts) > 1 {
		os, err = ParseOS(parts[1])
		if err != nil {
			return Platform{}, err
		}
	} else {
		os = GetDefaultPlatform().OS
	}

	return Platform{Arch: arch, OS: os}, nil
}

type Writer interface {
	Write(b byte) int
	WriteN(b byte, n int) int
	Write2(b byte) int
	Write4(b byte) int
	Write8(b byte) int
	Write8u(v uint64) int
	WriteBytes(bs []byte) int
	WriteUnsigned(i uint) int
}

type Const struct {
	value string
	addr  uint64
}

type PCRelocation struct {
	offset     uint64 // Offset in text section where relocation data is
	symbolName string // Name of symbol being referenced
}

type CallPatch struct {
	position   int    // Position in text section where the rel32 offset starts
	targetName string // Name of the target symbol
}

type BufferWrapper struct {
	buf *bytes.Buffer
}

type ExecutableBuilder struct {
	platform                Platform
	arch                    Architecture
	consts                  map[string]*Const
	labels                  map[string]int // Maps label names to their offsets in .text
	dynlinker               *DynamicLinker
	useDynamicLinking       bool
	neededFunctions         []string
	pcRelocations           []PCRelocation
	callPatches             []CallPatch
	elf, rodata, data, text bytes.Buffer
}

func (eb *ExecutableBuilder) ELFWriter() Writer {
	return &BufferWrapper{&eb.elf}
}

func (eb *ExecutableBuilder) RodataWriter() Writer {
	return &BufferWrapper{&eb.rodata}
}

func (eb *ExecutableBuilder) DataWriter() Writer {
	return &BufferWrapper{&eb.data}
}

func (eb *ExecutableBuilder) TextWriter() Writer {
	return &BufferWrapper{&eb.text}
}

// PatchPCRelocations patches all PC-relative address loads with actual offsets
func (eb *ExecutableBuilder) PatchPCRelocations(textAddr, rodataAddr uint64, rodataSize int) {
	textBytes := eb.text.Bytes()

	for _, reloc := range eb.pcRelocations {
		// Find the symbol address
		var targetAddr uint64
		if c, ok := eb.consts[reloc.symbolName]; ok {
			targetAddr = c.addr
			if strings.HasPrefix(reloc.symbolName, "str_") {
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "DEBUG PatchPCRelocations: %s using address 0x%x\n", reloc.symbolName, targetAddr)
				}
			}
		} else {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Warning: Symbol %s not found for PC relocation\n", reloc.symbolName)
			}
			continue
		}

		offset := int(reloc.offset)

		switch eb.platform.Arch {
		case ArchX86_64:
			eb.patchX86_64PCRel(textBytes, offset, textAddr, targetAddr, reloc.symbolName)
		case ArchARM64:
			eb.patchARM64PCRel(textBytes, offset, textAddr, targetAddr, reloc.symbolName)
		case ArchRiscv64:
			eb.patchRISCV64PCRel(textBytes, offset, textAddr, targetAddr, reloc.symbolName)
		}
	}
}

func (eb *ExecutableBuilder) patchX86_64PCRel(textBytes []byte, offset int, textAddr, targetAddr uint64, symbolName string) {
	// x86-64 RIP-relative: displacement is at offset, instruction ends at offset+4
	if offset+4 > len(textBytes) {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Warning: Relocation offset %d out of bounds\n", offset)
		}
		return
	}

	ripAddr := textAddr + uint64(offset) + 4 // RIP points after displacement
	displacement := int64(targetAddr) - int64(ripAddr)

	if displacement < -0x80000000 || displacement > 0x7FFFFFFF {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Warning: x86-64 displacement too large: %d\n", displacement)
		}
		return
	}

	disp32 := uint32(displacement)
	textBytes[offset] = byte(disp32 & 0xFF)
	textBytes[offset+1] = byte((disp32 >> 8) & 0xFF)
	textBytes[offset+2] = byte((disp32 >> 16) & 0xFF)
	textBytes[offset+3] = byte((disp32 >> 24) & 0xFF)

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Patched x86-64 PC relocation: %s at offset 0x%x, target 0x%x, RIP 0x%x, displacement %d\n",
			symbolName, offset, targetAddr, ripAddr, displacement)
	}
}

func (eb *ExecutableBuilder) patchARM64PCRel(textBytes []byte, offset int, textAddr, targetAddr uint64, symbolName string) {
	// ARM64: ADRP at offset, ADD at offset+4
	// ADRP loads page-aligned address (upper 52 bits)
	// ADD adds the low 12 bits
	if offset+8 > len(textBytes) {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Warning: ARM64 relocation offset %d out of bounds\n", offset)
		}
		return
	}

	instrAddr := textAddr + uint64(offset)

	// Page offset calculation for ADRP
	instrPage := instrAddr & ^uint64(0xFFF)
	targetPage := targetAddr & ^uint64(0xFFF)
	pageOffset := int64(targetPage - instrPage)

	// Check if page offset fits in 21 bits (signed, shifted)
	if pageOffset < -0x100000000 || pageOffset > 0xFFFFFFFF {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Warning: ARM64 page offset too large: %d\n", pageOffset)
		}
		return
	}

	// Low 12 bits for ADD
	low12 := uint32(targetAddr & 0xFFF)

	// Patch ADRP instruction (bits [23:5] get immlo, bits [30:29] get immhi)
	adrpInstr := uint32(textBytes[offset]) |
		(uint32(textBytes[offset+1]) << 8) |
		(uint32(textBytes[offset+2]) << 16) |
		(uint32(textBytes[offset+3]) << 24)

	pageOffsetShifted := uint32(pageOffset >> 12)
	immlo := (pageOffsetShifted & 0x3) << 29           // bits [1:0] -> [30:29]
	immhi := ((pageOffsetShifted >> 2) & 0x7FFFF) << 5 // bits [20:2] -> [23:5]

	adrpInstr = (adrpInstr & 0x9F00001F) | immlo | immhi

	textBytes[offset] = byte(adrpInstr & 0xFF)
	textBytes[offset+1] = byte((adrpInstr >> 8) & 0xFF)
	textBytes[offset+2] = byte((adrpInstr >> 16) & 0xFF)
	textBytes[offset+3] = byte((adrpInstr >> 24) & 0xFF)

	// Patch ADD instruction (bits [21:10] get imm12)
	addInstr := uint32(textBytes[offset+4]) |
		(uint32(textBytes[offset+5]) << 8) |
		(uint32(textBytes[offset+6]) << 16) |
		(uint32(textBytes[offset+7]) << 24)

	addInstr = (addInstr & 0xFFC003FF) | (low12 << 10)

	textBytes[offset+4] = byte(addInstr & 0xFF)
	textBytes[offset+5] = byte((addInstr >> 8) & 0xFF)
	textBytes[offset+6] = byte((addInstr >> 16) & 0xFF)
	textBytes[offset+7] = byte((addInstr >> 24) & 0xFF)

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Patched ARM64 PC relocation: %s at offset 0x%x, target 0x%x, page offset %d, low12 0x%x\n",
			symbolName, offset, targetAddr, pageOffset, low12)
	}
}

func (eb *ExecutableBuilder) patchRISCV64PCRel(textBytes []byte, offset int, textAddr, targetAddr uint64, symbolName string) {
	// RISC-V: AUIPC at offset, ADDI at offset+4
	// AUIPC loads upper 20 bits of PC-relative offset
	// ADDI adds the lower 12 bits
	if offset+8 > len(textBytes) {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Warning: RISC-V relocation offset %d out of bounds\n", offset)
		}
		return
	}

	instrAddr := textAddr + uint64(offset)
	pcOffset := int64(targetAddr) - int64(instrAddr)

	if pcOffset < -0x80000000 || pcOffset > 0x7FFFFFFF {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Warning: RISC-V offset too large: %d\n", pcOffset)
		}
		return
	}

	// Split into upper 20 bits and lower 12 bits
	// If bit 11 is set, we need to add 1 to upper because ADDI sign-extends
	upper := uint32((pcOffset + 0x800) >> 12)
	lower := uint32(pcOffset & 0xFFF)

	// Patch AUIPC instruction (bits [31:12] get upper 20 bits)
	auipcInstr := uint32(textBytes[offset]) |
		(uint32(textBytes[offset+1]) << 8) |
		(uint32(textBytes[offset+2]) << 16) |
		(uint32(textBytes[offset+3]) << 24)

	auipcInstr = (auipcInstr & 0xFFF) | (upper << 12)

	textBytes[offset] = byte(auipcInstr & 0xFF)
	textBytes[offset+1] = byte((auipcInstr >> 8) & 0xFF)
	textBytes[offset+2] = byte((auipcInstr >> 16) & 0xFF)
	textBytes[offset+3] = byte((auipcInstr >> 24) & 0xFF)

	// Patch ADDI instruction (bits [31:20] get lower 12 bits)
	addiInstr := uint32(textBytes[offset+4]) |
		(uint32(textBytes[offset+5]) << 8) |
		(uint32(textBytes[offset+6]) << 16) |
		(uint32(textBytes[offset+7]) << 24)

	addiInstr = (addiInstr & 0xFFFFF) | (lower << 20)

	textBytes[offset+4] = byte(addiInstr & 0xFF)
	textBytes[offset+5] = byte((addiInstr >> 8) & 0xFF)
	textBytes[offset+6] = byte((addiInstr >> 16) & 0xFF)
	textBytes[offset+7] = byte((addiInstr >> 24) & 0xFF)

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Patched RISC-V PC relocation: %s at offset 0x%x, target 0x%x, PC 0x%x, offset %d (upper=0x%x, lower=0x%x)\n",
			symbolName, offset, targetAddr, instrAddr, pcOffset, upper, lower)
	}
}

func New(machineStr string) (*ExecutableBuilder, error) {
	platform, err := StringToMachine(machineStr)
	if err != nil {
		return nil, err
	}

	arch, err := NewArchitecture(platform.String())
	if err != nil {
		return nil, err
	}

	return &ExecutableBuilder{
		platform:  platform,
		arch:      arch,
		consts:    make(map[string]*Const),
		dynlinker: NewDynamicLinker(),
	}, nil
}

// NewWithPlatform creates an ExecutableBuilder for a specific platform
func NewWithPlatform(platform Platform) (*ExecutableBuilder, error) {
	arch, err := NewArchitecture(platform.String())
	if err != nil {
		return nil, err
	}

	return &ExecutableBuilder{
		platform:  platform,
		arch:      arch,
		consts:    make(map[string]*Const),
		dynlinker: NewDynamicLinker(),
	}, nil
}

// getSyscallNumbers returns platform-specific syscall numbers
func getSyscallNumbers(platform Platform) map[string]string {
	// macOS (Darwin) has different syscall numbers with class prefix 0x2000000
	if platform.OS == OSDarwin {
		switch platform.Arch {
		case ArchX86_64:
			return map[string]string{
				"SYS_WRITE": "33554436", // 0x2000004
				"SYS_EXIT":  "33554433", // 0x2000001
				"STDOUT":    "1",
			}
		case ArchARM64:
			return map[string]string{
				"SYS_WRITE": "4", // 0x4 - Darwin uses lower 24 bits, x16 needs just the number
				"SYS_EXIT":  "1", // 0x1
				"STDOUT":    "1",
			}
		default:
			return map[string]string{}
		}
	}

	// Linux/FreeBSD syscall numbers
	switch platform.Arch {
	case ArchX86_64:
		return map[string]string{
			"SYS_WRITE": "1",
			"SYS_EXIT":  "60",
			"STDOUT":    "1",
		}
	case ArchARM64:
		return map[string]string{
			"SYS_WRITE": "64",
			"SYS_EXIT":  "93",
			"STDOUT":    "1",
		}
	case ArchRiscv64:
		return map[string]string{
			"SYS_WRITE": "64",
			"SYS_EXIT":  "93",
			"STDOUT":    "1",
		}
	default:
		return map[string]string{}
	}
}

// PatchCallSites patches all direct function calls with correct relative offsets
func (eb *ExecutableBuilder) PatchCallSites(textAddr uint64) {
	textBytes := eb.text.Bytes()

	for _, patch := range eb.callPatches {
		// Find the target symbol address (should be a label in the text section)
		targetOffset := eb.LabelOffset(patch.targetName)
		if targetOffset < 0 {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Warning: Label %s not found for call patch\n", patch.targetName)
			}
			continue
		}

		// Calculate addresses
		// patch.position points to the 4-byte rel32 offset (after the 0xE8 CALL opcode)
		ripAddr := textAddr + uint64(patch.position) + 4 // RIP points after the rel32
		targetAddr := textAddr + uint64(targetOffset)
		displacement := int64(targetAddr) - int64(ripAddr)

		if displacement < -0x80000000 || displacement > 0x7FFFFFFF {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Warning: Call displacement too large: %d\n", displacement)
			}
			continue
		}

		// Patch the 4-byte rel32 offset
		disp32 := uint32(displacement)
		textBytes[patch.position] = byte(disp32 & 0xFF)
		textBytes[patch.position+1] = byte((disp32 >> 8) & 0xFF)
		textBytes[patch.position+2] = byte((disp32 >> 16) & 0xFF)
		textBytes[patch.position+3] = byte((disp32 >> 24) & 0xFF)

		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Patched call to %s at position 0x%x: target offset 0x%x, displacement %d\n",
				patch.targetName, patch.position, targetOffset, displacement)
		}
	}
}

func (eb *ExecutableBuilder) Lookup(what string) string {
	// Check architecture-specific syscall numbers first
	syscalls := getSyscallNumbers(eb.platform)
	if v, ok := syscalls[what]; ok {
		return v
	}
	// Then check constants
	if c, ok := eb.consts[what]; ok {
		return strconv.FormatUint(c.addr, 10)
	}
	return "0"
}

func (eb *ExecutableBuilder) Bytes() []byte {
	// For Mach-O format (macOS)
	if eb.platform.IsMachO() {
		if err := eb.WriteMachO(); err != nil {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "ERROR: Failed to write Mach-O: %v\n", err)
			}
			// Fallback to ELF
		} else {
			result := eb.elf.Bytes()
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "DEBUG Bytes(): Using Mach-O format (size=%d)\n", len(result))
				if len(result) >= 824 {
					fmt.Fprintf(os.Stderr, "DEBUG Bytes(): bytes at offset 816: %x\n", result[816:824])
				}
			}
			return result
		}
	}

	// For dynamic ELFs, everything is already in eb.elf
	if eb.useDynamicLinking {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG Bytes(): Using dynamic ELF, returning eb.elf only (size=%d)\n", eb.elf.Len())
		}
		return eb.elf.Bytes()
	}

	// For static ELFs, concatenate sections
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG Bytes(): Using static ELF, concatenating sections\n")
	}
	var result bytes.Buffer
	result.Write(eb.elf.Bytes())
	result.Write(eb.rodata.Bytes())
	result.Write(eb.data.Bytes())
	result.Write(eb.text.Bytes())
	return result.Bytes()
}

func (eb *ExecutableBuilder) Define(symbol, value string) {
	if c, ok := eb.consts[symbol]; ok {
		// Symbol exists - update value but preserve address
		c.value = value
	} else {
		// New symbol
		eb.consts[symbol] = &Const{value: value}
	}
}

func (eb *ExecutableBuilder) DefineAddr(symbol string, addr uint64) {
	if c, ok := eb.consts[symbol]; ok {
		if strings.HasPrefix(symbol, "str_") || strings.HasPrefix(symbol, "lambda_") {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "DEBUG DefineAddr: %s set to 0x%x\n", symbol, addr)
			}
		}
		c.addr = addr
	} else {
		// Symbol doesn't exist yet - create it first
		if strings.HasPrefix(symbol, "lambda_") {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "DEBUG DefineAddr: Creating missing symbol %s with addr 0x%x\n", symbol, addr)
			}
		}
		eb.consts[symbol] = &Const{value: "", addr: addr}
	}
}

func (eb *ExecutableBuilder) MarkLabel(label string) {
	// Mark a position in .text for a label (like a function)
	// Store as empty string - address will be set later based on text position
	if _, ok := eb.consts[label]; !ok {
		eb.consts[label] = &Const{value: ""}
	}

	// Also record the current offset in the text section
	if eb.labels == nil {
		eb.labels = make(map[string]int)
	}
	eb.labels[label] = eb.text.Len()
}

func (eb *ExecutableBuilder) LabelOffset(label string) int {
	// Get the offset of a label in the text section
	if offset, ok := eb.labels[label]; ok {
		return offset
	}
	return -1 // Label not found
}

func (eb *ExecutableBuilder) RodataSection() map[string]string {
	rodataSymbols := make(map[string]string)
	for name, c := range eb.consts {
		// Skip code labels (they have empty values)
		if c.value != "" {
			rodataSymbols[name] = c.value
		}
	}
	return rodataSymbols
}

func (eb *ExecutableBuilder) RodataSize() int {
	size := 0
	for _, data := range eb.RodataSection() {
		size += len(data)
	}
	return size
}

func (eb *ExecutableBuilder) WriteRodata(data []byte) uint64 {
	n, _ := eb.rodata.Write(data)
	return uint64(n)
}

func (eb *ExecutableBuilder) DataSection() map[string]string {
	return make(map[string]string)
}

func (eb *ExecutableBuilder) DataSize() int {
	return 0
}

func (eb *ExecutableBuilder) WriteData(data []byte) uint64 {
	n, _ := eb.data.Write(data)
	return uint64(n)
}

func (eb *ExecutableBuilder) MovInstruction(dst, src string) error {
	out := &Out{
		machine: eb.platform,
		writer:  eb.TextWriter(),
		eb:      eb,
	}
	out.MovInstruction(dst, src)
	return nil
}

// Dynamic library helper methods
func (eb *ExecutableBuilder) AddLibrary(name, sofile string) *DynamicLibrary {
	return eb.dynlinker.AddLibrary(name, sofile)
}

func (eb *ExecutableBuilder) ImportFunction(libName, funcName string) error {
	return eb.dynlinker.ImportFunction(libName, funcName)
}

func (eb *ExecutableBuilder) CallLibFunction(funcName string, args ...string) error {
	return eb.dynlinker.GenerateFunctionCall(eb, funcName, args)
}

// GenerateGlibcHelloWorld generates a hello world program using glibc printf
func (eb *ExecutableBuilder) GenerateGlibcHelloWorld() error {
	// Set up for glibc dynamic linking
	eb.useDynamicLinking = true
	eb.neededFunctions = []string{"printf", "exit"}

	// Add glibc library
	glibc := eb.AddLibrary("glibc", "libc.so.6")

	// Define printf function
	glibc.AddFunction("printf", CTypeInt,
		Parameter{Name: "format", Type: CTypePointer},
	)

	// Define exit function
	glibc.AddFunction("exit", CTypeVoid,
		Parameter{Name: "status", Type: CTypeInt},
	)

	// Import functions
	err := eb.ImportFunction("glibc", "printf")
	if err != nil {
		return err
	}

	err = eb.ImportFunction("glibc", "exit")
	if err != nil {
		return err
	}

	// Generate the function calls (will be patched to use PLT)
	err = eb.CallLibFunction("printf", "hello")
	if err != nil {
		return err
	}

	err = eb.CallLibFunction("exit", "0")
	if err != nil {
		return err
	}

	return nil
}

// GenerateCallInstruction generates a call instruction
// NOTE: This generates placeholder addresses that should be fixed
// when we have complete PLT information
func (eb *ExecutableBuilder) GenerateCallInstruction(funcName string) error {
	w := eb.TextWriter()
	if VerboseMode {
		fmt.Fprint(os.Stderr, funcName+"@plt:")
	}

	// Strip leading underscore if present (for Mach-O compatibility)
	targetName := funcName
	if strings.HasPrefix(funcName, "_") {
		targetName = funcName[1:] // Remove underscore
	}

	// Register the call patch for later resolution
	position := eb.text.Len()
	eb.callPatches = append(eb.callPatches, CallPatch{
		position:   position,
		targetName: targetName + "$stub",
	})

	// Generate architecture-specific call instruction with placeholder
	switch eb.platform.Arch {
	case ArchX86_64:
		w.Write(0xE8)               // CALL rel32
		w.WriteUnsigned(0x12345678) // Placeholder - will be patched
	case ArchARM64:
		w.WriteUnsigned(0x94000000) // BL placeholder
	case ArchRiscv64:
		w.WriteUnsigned(0x000000EF) // JAL placeholder
	}

	if VerboseMode {
		fmt.Fprintln(os.Stderr)
	}
	return nil
}

// patchTextInELF replaces the .text section in the ELF buffer with the current text buffer
func (eb *ExecutableBuilder) patchTextInELF() {
	// The ELF buffer contains: ELF header + program headers + all sections
	// We need to find where the .text section is in the ELF buffer and replace it

	// For now, we'll use a simple approach: the ELF buffer is built in order,
	// so we know the text comes after BSS in the file
	// But actually, in WriteCompleteDynamicELF, the buffers are written in this order:
	// - ELF header + program headers
	// - interpreter, dynsym, dynstr, hash, rela (at page 0x1000)
	// - plt, text (at page 0x2000)
	// - dynamic, got, bss (at page 0x3000)

	// Since the entire elf buffer was already constructed, we need to replace just the text portion
	// The text section starts at file offset 0x2000 + plt_size

	elfBuf := eb.elf.Bytes()
	newText := eb.text.Bytes()

	// Find the text section in the ELF buffer
	// PLT is at offset 0x2000
	// PLT size = 16 bytes (PLT[0]) + 16 bytes per function
	// _start is after PLT (16 bytes aligned)
	// text starts after _start
	pltSize := 16 + (len(eb.neededFunctions) * 16) // Dynamic PLT size based on number of functions
	startSizeAligned := 16                         // _start is 14 bytes, aligned to 16
	textOffset := 0x2000 + pltSize + startSizeAligned
	textSize := len(newText)

	// Replace the text section
	copy(elfBuf[textOffset:textOffset+textSize], newText)

	// No need to rebuild - elfBuf is a slice of eb.elf's internal buffer,
	// so modifications to elfBuf are already reflected in eb.elf
}

// Global flags for controlling output verbosity and dependencies
var VerboseMode bool
var UpdateDepsFlag bool

func main() {
	fmt.Fprintln(os.Stderr, "TRACE: main() entry")
	// Create default output filename in system temp directory
	defaultOutputFilename := filepath.Join(os.TempDir(), "main")
	fmt.Fprintln(os.Stderr, "TRACE: got temp dir")

	// Get default platform
	defaultPlatform := GetDefaultPlatform()
	fmt.Fprintln(os.Stderr, "TRACE: got default platform")
	defaultArchStr := "amd64"
	if defaultPlatform.Arch == ArchARM64 {
		defaultArchStr = "arm64"
	} else if defaultPlatform.Arch == ArchRiscv64 {
		defaultArchStr = "riscv64"
	}
	defaultOSStr := defaultPlatform.OS.String()
	fmt.Fprintln(os.Stderr, "TRACE: about to define flags")

	var archFlag = flag.String("arch", defaultArchStr, "target architecture (amd64, arm64, riscv64)")
	fmt.Fprintln(os.Stderr, "TRACE: defined arch flag")
	var osFlag = flag.String("os", defaultOSStr, "target OS (linux, darwin, freebsd)")
	var targetFlag = flag.String("target", "", "target platform (e.g., arm64-macos, amd64-linux, riscv64-linux)")
	var outputFilenameFlag = flag.String("o", defaultOutputFilename, "output executable filename")
	var outputFilenameLongFlag = flag.String("output", defaultOutputFilename, "output executable filename")
	var versionShort = flag.Bool("V", false, "print version information and exit")
	var version = flag.Bool("version", false, "print version information and exit")
	var verbose = flag.Bool("v", false, "verbose mode (show detailed compilation info)")
	var verboseLong = flag.Bool("verbose", false, "verbose mode (show detailed compilation info)")
	var updateDeps = flag.Bool("u", false, "update all dependency repositories from Git")
	var updateDepsLong = flag.Bool("update-deps", false, "update all dependency repositories from Git")
	var codeFlag = flag.String("c", "", "execute Flap code from command line")
	fmt.Fprintln(os.Stderr, "TRACE: all flags defined, about to parse")
	flag.Parse()
	fmt.Fprintln(os.Stderr, "TRACE: flags parsed")

	// Set global update-deps flag (use whichever was specified)
	UpdateDepsFlag = *updateDeps || *updateDepsLong

	if *version || *versionShort {
		fmt.Println(versionString)
		os.Exit(0)
	}

	// Set global verbosity flag (use whichever was specified)
	VerboseMode = *verbose || *verboseLong

	// Parse target platform
	var targetArch Arch
	var targetOS OS
	var err error

	// If --target is specified, parse it; otherwise use --arch and --os
	if *targetFlag != "" {
		// Parse target string like "arm64-macos" or "amd64-linux"
		parts := strings.Split(*targetFlag, "-")
		if len(parts) != 2 {
			log.Fatalf("Invalid --target format '%s'. Expected format: ARCH-OS (e.g., arm64-macos, amd64-linux)", *targetFlag)
		}

		targetArch, err = ParseArch(parts[0])
		if err != nil {
			log.Fatalf("Invalid architecture in --target '%s': %v", *targetFlag, err)
		}

		// Handle OS aliases (macos -> darwin)
		osStr := parts[1]
		if osStr == "macos" {
			osStr = "darwin"
		}
		targetOS, err = ParseOS(osStr)
		if err != nil {
			log.Fatalf("Invalid OS in --target '%s': %v (use 'darwin' instead of 'macos')", *targetFlag, err)
		}
	} else {
		// Use separate --arch and --os flags
		targetArch, err = ParseArch(*archFlag)
		if err != nil {
			log.Fatalf("Invalid --arch '%s': %v", *archFlag, err)
		}

		targetOS, err = ParseOS(*osFlag)
		if err != nil {
			log.Fatalf("Invalid --os '%s': %v", *osFlag, err)
		}
	}

	targetPlatform := Platform{Arch: targetArch, OS: targetOS}

	// Use whichever output flag was specified (prefer short form if both given)
	outputFilename := *outputFilenameFlag
	outputFlagProvided := false
	if *outputFilenameLongFlag != defaultOutputFilename {
		outputFilename = *outputFilenameLongFlag
		outputFlagProvided = true
	}
	if *outputFilenameFlag != defaultOutputFilename {
		outputFilename = *outputFilenameFlag
		outputFlagProvided = true
	}

	// Get input files from remaining arguments
	inputFiles := flag.Args()

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "----=[ %s ]=----\n", versionString)
	}

	// Warn if no input files provided
	if len(inputFiles) == 0 && *codeFlag == "" {
		fmt.Fprintln(os.Stderr, "flapc: warning: no input files")
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Target platform: %s\n", targetPlatform.FullString())
	}

	eb, err := NewWithPlatform(targetPlatform)
	if err != nil {
		log.Fatalln(err)
	}

	// Handle -c flag for inline code execution
	if *codeFlag != "" {
		// Create a temporary file with the inline code
		tmpFile, err := os.CreateTemp("", "flapc_*.flap")
		if err != nil {
			log.Fatalf("Failed to create temp file: %v", err)
		}
		tmpFilename := tmpFile.Name()
		defer os.Remove(tmpFilename)

		// Write the code to the temp file
		if _, err := tmpFile.WriteString(*codeFlag); err != nil {
			tmpFile.Close()
			log.Fatalf("Failed to write to temp file: %v", err)
		}
		tmpFile.Close()

		// Compile the temp file
		writeToFilename := outputFilename
		inlineFlagProvided := outputFlagProvided
		if outputFilename == defaultOutputFilename {
			writeToFilename = filepath.Join(os.TempDir(), "flapc_inline")
			inlineFlagProvided = false
		}

		err = CompileFlap(tmpFilename, writeToFilename, targetPlatform)
		if err != nil {
			log.Fatalf("Flap compilation error: %v", err)
		}
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "-> Wrote executable: %s\n", writeToFilename)
		} else if !inlineFlagProvided {
			fmt.Println(writeToFilename)
		}
		return
	}

	if len(inputFiles) > 0 {
		for _, file := range inputFiles {
			if VerboseMode {
				log.Printf("source file: %s", file)
			}

			// Check if this is a Flap source file
			if strings.HasSuffix(file, ".flap") {
				if VerboseMode {
					fmt.Fprintln(os.Stderr, "-> Compiling Flap source")
				}

				writeToFilename := outputFilename
				if outputFilename == defaultOutputFilename {
					writeToFilename = strings.TrimSuffix(filepath.Base(file), ".flap")
				}

				err := CompileFlap(file, writeToFilename, targetPlatform)
				if err != nil {
					log.Fatalf("Flap compilation error: %v", err)
				}
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "-> Wrote executable: %s\n", writeToFilename)
				} else if !outputFlagProvided {
					fmt.Println(writeToFilename)
				}
				return
			}
		}
	}

	if err := eb.CompileDefaultProgram(filepath.Join(os.TempDir(), "main")); err != nil {
		log.Fatalf("Flap compilation error: %v", err)
	}

}
