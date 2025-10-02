package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// A tiny compiler for x86_64, aarch64, and riscv64 ELF files for Linux

const versionString = "ggg 0.0.1"

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

type RIPRelocation struct {
	offset     uint64 // Offset in text section where displacement is
	symbolName string // Name of symbol being referenced
}

type BufferWrapper struct {
	buf *bytes.Buffer
}

type ExecutableBuilder struct {
	machine                 Machine
	arch                    Architecture
	consts                  map[string]*Const
	dynlinker               *DynamicLinker
	useDynamicLinking       bool
	neededFunctions         []string
	ripRelocations          []RIPRelocation
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

// PatchRIPRelocations patches all RIP-relative LEA instructions with actual offsets
func (eb *ExecutableBuilder) PatchRIPRelocations(textAddr, rodataAddr uint64, rodataSize int) {
	for _, reloc := range eb.ripRelocations {
		// Find the symbol address
		var targetAddr uint64
		if c, ok := eb.consts[reloc.symbolName]; ok {
			// c.addr is already the absolute virtual address
			targetAddr = c.addr
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Symbol %s not found for RIP relocation\n", reloc.symbolName)
			continue
		}

		// Calculate RIP at the point after the LEA instruction
		// The offset field is the last 4 bytes of the instruction
		ripAddr := textAddr + reloc.offset + 4 // +4 because RIP points after the instruction

		// Calculate displacement (signed 32-bit offset from RIP to target)
		displacement := int64(targetAddr) - int64(ripAddr)

		// Check if displacement fits in 32 bits
		if displacement < -0x80000000 || displacement > 0x7FFFFFFF {
			fmt.Fprintf(os.Stderr, "Warning: RIP-relative displacement too large: %d\n", displacement)
			continue
		}

		// Patch the 4-byte displacement in the text section (little-endian)
		textBytes := eb.text.Bytes()
		offset := int(reloc.offset)
		if offset+4 > len(textBytes) {
			fmt.Fprintf(os.Stderr, "Warning: Relocation offset %d out of bounds\n", offset)
			continue
		}

		disp32 := uint32(displacement)
		textBytes[offset] = byte(disp32 & 0xFF)
		textBytes[offset+1] = byte((disp32 >> 8) & 0xFF)
		textBytes[offset+2] = byte((disp32 >> 16) & 0xFF)
		textBytes[offset+3] = byte((disp32 >> 24) & 0xFF)

		fmt.Fprintf(os.Stderr, "Patched RIP relocation: %s at offset 0x%x, target 0x%x, RIP 0x%x, displacement %d\n",
			reloc.symbolName, reloc.offset, targetAddr, ripAddr, displacement)
	}
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
	var result bytes.Buffer
	result.Write(eb.elf.Bytes())
	result.Write(eb.rodata.Bytes())
	result.Write(eb.data.Bytes())
	result.Write(eb.text.Bytes())
	return result.Bytes()
}

func (eb *ExecutableBuilder) Define(symbol, value string) {
	eb.consts[symbol] = &Const{value: value}
}

func (eb *ExecutableBuilder) DefineAddr(symbol string, addr uint64) {
	if c, ok := eb.consts[symbol]; ok {
		c.addr = addr
	}
}

func (eb *ExecutableBuilder) RodataSection() map[string]string {
	rodataSymbols := make(map[string]string)
	for name, c := range eb.consts {
		rodataSymbols[name] = c.value
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
	// PLT is at offset 0x2000 (48 bytes)
	// _start is at offset 0x2030 (16 bytes aligned)
	// text starts after _start
	pltSize := 48
	startSizeAligned := 16 // _start is 14 bytes, aligned to 16
	textOffset := 0x2000 + pltSize + startSizeAligned
	textSize := len(newText)

	fmt.Fprintf(os.Stderr, "  Patching text at offset 0x%x, size %d bytes\n", textOffset, textSize)

	// Replace the text section
	copy(elfBuf[textOffset:textOffset+textSize], newText)

	// Rebuild the ELF buffer
	eb.elf.Reset()
	eb.elf.Write(elfBuf)
}

func main() {
	var machine = flag.String("m", "x86_64", "target machine architecture (x86_64, amd64, arm64, aarch64, riscv64, riscv, rv64)")
	var machineLong = flag.String("machine", "x86_64", "target machine architecture (x86_64, amd64, arm64, aarch64, riscv64, riscv, rv64)")
	var output = flag.String("o", "main", "output executable filename")
	var outputLong = flag.String("output", "main", "output executable filename")
	flag.Parse()

	// Use whichever flag was specified (prefer short form if both given)
	targetMachine := *machine
	if *machineLong != "x86_64" {
		targetMachine = *machineLong
	}

	// Use whichever output flag was specified (prefer short form if both given)
	outputFile := *output
	if *outputLong != "main" {
		outputFile = *outputLong
	}

	// Get input files from remaining arguments
	inputFiles := flag.Args()

	fmt.Fprintf(os.Stderr, "----=[ %s ]=----\n", versionString)

	eb, err := New(targetMachine)
	if err != nil {
		log.Fatalln(err)
	}

	if len(inputFiles) > 0 {
		for _, file := range inputFiles {
			log.Printf("source file: %s", file)
		}
	}

	eb.Define("hello", "Hello, World!\n\x00")

	// Enable dynamic linking for glibc
	eb.useDynamicLinking = true
	eb.neededFunctions = []string{"printf", "exit"}

	if eb.useDynamicLinking && len(eb.neededFunctions) > 0 {
		fmt.Fprintln(os.Stderr, "-> .rodata")
		rodataSymbols := eb.RodataSection()
		estimatedRodataAddr := uint64(0x403000 + 0x100)
		currentAddr := estimatedRodataAddr
		for symbol, value := range rodataSymbols {
			eb.WriteRodata([]byte(value))
			eb.DefineAddr(symbol, currentAddr)
			currentAddr += uint64(len(value))
			fmt.Fprintf(os.Stderr, "%s = %q at ~0x%x (estimated)\n", symbol, value, eb.consts[symbol].addr)
		}

		// Generate text with estimated BSS addresses
		fmt.Fprintln(os.Stderr, "-> .text")
		err := eb.GenerateGlibcHelloWorld()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating glibc hello world: %v\n", err)
			// Fallback to syscalls
			eb.SysWrite("hello")
			eb.SysExit()
		}

		fmt.Fprintln(os.Stderr, "-> ELF generation")

		// Set up complete dynamic sections
		ds := NewDynamicSections()
		ds.AddNeeded("libc.so.6")

		// Add symbols
		for _, funcName := range eb.neededFunctions {
			ds.AddSymbol(funcName, STB_GLOBAL, STT_FUNC)
		}

		gotBase, rodataBaseAddr, err := eb.WriteCompleteDynamicELF(ds, eb.neededFunctions)
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Fprintln(os.Stderr, "-> .rodata (final addresses) and regenerating code")
		currentAddr = rodataBaseAddr
		for symbol, value := range rodataSymbols {
			eb.DefineAddr(symbol, currentAddr)
			currentAddr += uint64(len(value))
			fmt.Fprintf(os.Stderr, "%s = %q at 0x%x\n", symbol, value, eb.consts[symbol].addr)
		}

		eb.text.Reset()
		err = eb.GenerateGlibcHelloWorld()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error regenerating code: %v\n", err)
		}

		textAddr := uint64(0x402040)
		pltBase := uint64(0x402000)
		fmt.Fprintln(os.Stderr, "-> Patching PLT calls in regenerated code")
		eb.patchPLTCalls(ds, textAddr, pltBase, eb.neededFunctions)

		fmt.Fprintln(os.Stderr, "-> Patching RIP-relative relocations in regenerated code")
		rodataSize := eb.rodata.Len()
		eb.PatchRIPRelocations(textAddr, rodataBaseAddr, rodataSize)

		fmt.Fprintln(os.Stderr, "-> Updating ELF with regenerated code")
		eb.patchTextInELF()

		fmt.Fprintf(os.Stderr, "Final GOT base: 0x%x\n", gotBase)

	} else {
		fmt.Fprintln(os.Stderr, "-> .rodata")
		rodataSymbols := eb.RodataSection()
		rodataAddr := baseAddr + headerSize
		currentAddr := uint64(rodataAddr)
		for symbol, value := range rodataSymbols {
			eb.DefineAddr(symbol, currentAddr)
			currentAddr += eb.WriteRodata([]byte(value))
			fmt.Fprintf(os.Stderr, "%s = %q\n", symbol, value)
		}

		fmt.Fprintln(os.Stderr, "-> .text")
		err = eb.GenerateGlibcHelloWorld()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating glibc hello world: %v\n", err)
			eb.SysWrite("hello")
			eb.SysExit()
		}

		if len(eb.dynlinker.Libraries) > 0 {
			eb.WriteDynamicELF()
		} else {
			eb.WriteELFHeader()
		}
	}

	// Output the executable file
	if err := os.WriteFile(outputFile, eb.Bytes(), 0o755); err != nil {
		log.Fatalln(err)
	}
}
