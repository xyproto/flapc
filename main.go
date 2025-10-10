package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// A tiny compiler for x86_64, aarch64, and riscv64 ELF files for Linux

const versionString = "flapc 0.1.0"

// Machine architecture constants
type Machine int

const (
	MachineX86_64 Machine = iota
	MachineARM64
	MachineRiscv64
)

// MachineToString converts machine constant to string representation
func (m Machine) String() string {
	switch m {
	case MachineX86_64:
		return "x86_64"
	case MachineARM64:
		return "aarch64"
	case MachineRiscv64:
		return "riscv64"
	default:
		return "unknown"
	}
}

// StringToMachine converts string representation to machine constant
func StringToMachine(machine string) (Machine, error) {
	switch strings.ToLower(machine) {
	case "x86_64", "amd64":
		return MachineX86_64, nil
	case "aarch64", "arm64":
		return MachineARM64, nil
	case "riscv64", "riscv", "rv64":
		return MachineRiscv64, nil
	default:
		return -1, fmt.Errorf("unsupported architecture: %s", machine)
	}
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
	machine                 Machine
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
				fmt.Fprintf(os.Stderr, "DEBUG PatchPCRelocations: %s using address 0x%x\n", reloc.symbolName, targetAddr)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Symbol %s not found for PC relocation\n", reloc.symbolName)
			continue
		}

		offset := int(reloc.offset)

		switch eb.machine {
		case MachineX86_64:
			eb.patchX86_64PCRel(textBytes, offset, textAddr, targetAddr, reloc.symbolName)
		case MachineARM64:
			eb.patchARM64PCRel(textBytes, offset, textAddr, targetAddr, reloc.symbolName)
		case MachineRiscv64:
			eb.patchRISCV64PCRel(textBytes, offset, textAddr, targetAddr, reloc.symbolName)
		}
	}
}

func (eb *ExecutableBuilder) patchX86_64PCRel(textBytes []byte, offset int, textAddr, targetAddr uint64, symbolName string) {
	// x86-64 RIP-relative: displacement is at offset, instruction ends at offset+4
	if offset+4 > len(textBytes) {
		fmt.Fprintf(os.Stderr, "Warning: Relocation offset %d out of bounds\n", offset)
		return
	}

	ripAddr := textAddr + uint64(offset) + 4 // RIP points after displacement
	displacement := int64(targetAddr) - int64(ripAddr)

	if displacement < -0x80000000 || displacement > 0x7FFFFFFF {
		fmt.Fprintf(os.Stderr, "Warning: x86-64 displacement too large: %d\n", displacement)
		return
	}

	disp32 := uint32(displacement)
	textBytes[offset] = byte(disp32 & 0xFF)
	textBytes[offset+1] = byte((disp32 >> 8) & 0xFF)
	textBytes[offset+2] = byte((disp32 >> 16) & 0xFF)
	textBytes[offset+3] = byte((disp32 >> 24) & 0xFF)

	fmt.Fprintf(os.Stderr, "Patched x86-64 PC relocation: %s at offset 0x%x, target 0x%x, RIP 0x%x, displacement %d\n",
		symbolName, offset, targetAddr, ripAddr, displacement)
}

func (eb *ExecutableBuilder) patchARM64PCRel(textBytes []byte, offset int, textAddr, targetAddr uint64, symbolName string) {
	// ARM64: ADRP at offset, ADD at offset+4
	// ADRP loads page-aligned address (upper 52 bits)
	// ADD adds the low 12 bits
	if offset+8 > len(textBytes) {
		fmt.Fprintf(os.Stderr, "Warning: ARM64 relocation offset %d out of bounds\n", offset)
		return
	}

	instrAddr := textAddr + uint64(offset)

	// Page offset calculation for ADRP
	instrPage := instrAddr & ^uint64(0xFFF)
	targetPage := targetAddr & ^uint64(0xFFF)
	pageOffset := int64(targetPage - instrPage)

	// Check if page offset fits in 21 bits (signed, shifted)
	if pageOffset < -0x100000000 || pageOffset > 0xFFFFFFFF {
		fmt.Fprintf(os.Stderr, "Warning: ARM64 page offset too large: %d\n", pageOffset)
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

	fmt.Fprintf(os.Stderr, "Patched ARM64 PC relocation: %s at offset 0x%x, target 0x%x, page offset %d, low12 0x%x\n",
		symbolName, offset, targetAddr, pageOffset, low12)
}

func (eb *ExecutableBuilder) patchRISCV64PCRel(textBytes []byte, offset int, textAddr, targetAddr uint64, symbolName string) {
	// RISC-V: AUIPC at offset, ADDI at offset+4
	// AUIPC loads upper 20 bits of PC-relative offset
	// ADDI adds the lower 12 bits
	if offset+8 > len(textBytes) {
		fmt.Fprintf(os.Stderr, "Warning: RISC-V relocation offset %d out of bounds\n", offset)
		return
	}

	instrAddr := textAddr + uint64(offset)
	pcOffset := int64(targetAddr) - int64(instrAddr)

	if pcOffset < -0x80000000 || pcOffset > 0x7FFFFFFF {
		fmt.Fprintf(os.Stderr, "Warning: RISC-V offset too large: %d\n", pcOffset)
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

	fmt.Fprintf(os.Stderr, "Patched RISC-V PC relocation: %s at offset 0x%x, target 0x%x, PC 0x%x, offset %d (upper=0x%x, lower=0x%x)\n",
		symbolName, offset, targetAddr, instrAddr, pcOffset, upper, lower)
}

func New(machineStr string) (*ExecutableBuilder, error) {
	machine, err := StringToMachine(machineStr)
	if err != nil {
		return nil, err
	}

	arch, err := NewArchitecture(machine.String())
	if err != nil {
		return nil, err
	}

	return &ExecutableBuilder{
		machine:   machine,
		arch:      arch,
		consts:    make(map[string]*Const),
		dynlinker: NewDynamicLinker(),
	}, nil
}

// getSyscallNumbers returns architecture-specific syscall numbers
func getSyscallNumbers(machine Machine) map[string]string {
	switch machine {
	case MachineX86_64:
		return map[string]string{
			"SYS_WRITE": "1",
			"SYS_EXIT":  "60",
			"STDOUT":    "1",
		}
	case MachineARM64:
		return map[string]string{
			"SYS_WRITE": "64",
			"SYS_EXIT":  "93",
			"STDOUT":    "1",
		}
	case MachineRiscv64:
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
			fmt.Fprintf(os.Stderr, "Warning: Label %s not found for call patch\n", patch.targetName)
			continue
		}

		// Calculate addresses
		// patch.position points to the 4-byte rel32 offset (after the 0xE8 CALL opcode)
		ripAddr := textAddr + uint64(patch.position) + 4 // RIP points after the rel32
		targetAddr := textAddr + uint64(targetOffset)
		displacement := int64(targetAddr) - int64(ripAddr)

		if displacement < -0x80000000 || displacement > 0x7FFFFFFF {
			fmt.Fprintf(os.Stderr, "Warning: Call displacement too large: %d\n", displacement)
			continue
		}

		// Patch the 4-byte rel32 offset
		disp32 := uint32(displacement)
		textBytes[patch.position] = byte(disp32 & 0xFF)
		textBytes[patch.position+1] = byte((disp32 >> 8) & 0xFF)
		textBytes[patch.position+2] = byte((disp32 >> 16) & 0xFF)
		textBytes[patch.position+3] = byte((disp32 >> 24) & 0xFF)

		fmt.Fprintf(os.Stderr, "Patched call to %s at position 0x%x: target offset 0x%x, displacement %d\n",
			patch.targetName, patch.position, targetOffset, displacement)
	}
}

func (eb *ExecutableBuilder) Lookup(what string) string {
	// Check architecture-specific syscall numbers first
	syscalls := getSyscallNumbers(eb.machine)
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
	// For dynamic ELFs, everything is already in eb.elf
	if eb.useDynamicLinking {
		fmt.Fprintf(os.Stderr, "DEBUG Bytes(): Using dynamic ELF, returning eb.elf only (size=%d)\n", eb.elf.Len())
		return eb.elf.Bytes()
	}

	// For static ELFs, concatenate sections
	fmt.Fprintf(os.Stderr, "DEBUG Bytes(): Using static ELF, concatenating sections\n")
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
			fmt.Fprintf(os.Stderr, "DEBUG DefineAddr: %s set to 0x%x\n", symbol, addr)
		}
		c.addr = addr
	} else {
		// Symbol doesn't exist yet - create it first
		if strings.HasPrefix(symbol, "lambda_") {
			fmt.Fprintf(os.Stderr, "DEBUG DefineAddr: Creating missing symbol %s with addr 0x%x\n", symbol, addr)
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
		machine: eb.machine,
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
	fmt.Fprint(os.Stderr, funcName+"@plt:")

	// Generate architecture-specific call instruction with placeholder
	switch eb.machine {
	case MachineX86_64:
		w.Write(0xE8)               // CALL rel32
		w.WriteUnsigned(0x12345678) // Placeholder - will be patched
	case MachineARM64:
		w.WriteUnsigned(0x94000000) // BL placeholder
	case MachineRiscv64:
		w.WriteUnsigned(0x000000EF) // JAL placeholder
	}

	fmt.Fprintln(os.Stderr)
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
	fmt.Fprintf(os.Stderr, "DEBUG patchTextInELF: Copying %d bytes of regenerated code at ELF offset 0x%x\n", textSize, textOffset)
	copy(elfBuf[textOffset:textOffset+textSize], newText)

	// Rebuild the ELF buffer
	eb.elf.Reset()
	eb.elf.Write(elfBuf)
}

// Global flags for controlling output verbosity and dependencies
var VerboseMode bool
var UpdateDepsFlag bool

func main() {
	const defaultOutputFilename = "/tmp/main"

	var machine = flag.String("m", "x86_64", "target machine architecture (x86_64, amd64, arm64, aarch64, riscv64, riscv, rv64)")
	var machineLong = flag.String("machine", "x86_64", "target machine architecture (x86_64, amd64, arm64, aarch64, riscv64, riscv, rv64)")
	var outputFilenameFlag = flag.String("o", defaultOutputFilename, "output executable filename")
	var outputFilenameLongFlag = flag.String("output", defaultOutputFilename, "output executable filename")
	var version = flag.Bool("version", false, "print version information and exit")
	var verbose = flag.Bool("v", false, "verbose mode (show detailed compilation info)")
	var verboseLong = flag.Bool("verbose", false, "verbose mode (show detailed compilation info)")
	var updateDeps = flag.Bool("u", false, "update all dependency repositories from Git")
	var updateDepsLong = flag.Bool("update-deps", false, "update all dependency repositories from Git")
	var codeFlag = flag.String("c", "", "execute Flap code from command line")
	flag.Parse()

	// Set global update-deps flag (use whichever was specified)
	UpdateDepsFlag = *updateDeps || *updateDepsLong

	if *version {
		fmt.Println(versionString)
		os.Exit(0)
	}

	// Set global verbosity flag (use whichever was specified)
	VerboseMode = *verbose || *verboseLong

	// Use whichever flag was specified (prefer short form if both given)
	targetMachine := *machine
	if *machineLong != "x86_64" {
		targetMachine = *machineLong
	}

	// Use whichever output flag was specified (prefer short form if both given)
	outputFilename := *outputFilenameFlag
	if *outputFilenameLongFlag != defaultOutputFilename {
		outputFilename = *outputFilenameLongFlag
	}

	// Get input files from remaining arguments
	inputFiles := flag.Args()

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "----=[ %s ]=----\n", versionString)
	}

	eb, err := New(targetMachine)
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
		if outputFilename == defaultOutputFilename {
			writeToFilename = "/tmp/flapc_inline"
		}

		err = CompileFlap(tmpFilename, writeToFilename)
		if err != nil {
			log.Fatalf("Flap compilation error: %v", err)
		}
		fmt.Fprintf(os.Stderr, "-> Wrote executable: %s\n", writeToFilename)
		return
	}

	if len(inputFiles) > 0 {
		for _, file := range inputFiles {
			log.Printf("source file: %s", file)

			// Check if this is a Flap source file
			if strings.HasSuffix(file, ".flap") {
				fmt.Fprintln(os.Stderr, "-> Compiling Flap source")

				writeToFilename := outputFilename
				if outputFilename == defaultOutputFilename {
					writeToFilename = strings.TrimSuffix(filepath.Base(file), ".flap")
				}

				err := CompileFlap(file, writeToFilename)
				if err != nil {
					log.Fatalf("Flap compilation error: %v", err)
				}
				fmt.Fprintf(os.Stderr, "-> Wrote executable: %s\n", writeToFilename)
				return
			}
		}
	}

	if err := eb.CompileDefaultProgram("/tmp/main"); err != nil {
		log.Fatalf("Flap compilation error: %v", err)
	}

}
