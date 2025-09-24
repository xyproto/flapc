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

func normalizeMachine(machine string) (string, error) {
	switch strings.ToLower(machine) {
	case "x86_64", "amd64":
		return "x86_64", nil
	case "arm64", "aarch64":
		return "aarch64", nil
	case "riscv64", "riscv", "rv64":
		return "riscv64", nil
	}
	return "", fmt.Errorf("unsupported machine architecture: %s (supported: x86_64, amd64, arm64, aarch64, riscv64, riscv, rv64)", machine)
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

type BufferWrapper struct {
	buf *bytes.Buffer
}

type ExecutableBuilder struct {
	machine        string
	arch           Architecture
	consts         map[string]*Const
	elf, bss, text bytes.Buffer // The ELF header, .bss and .text sections, as bytes
}

func (eb *ExecutableBuilder) ELFWriter() Writer {
	return &BufferWrapper{&eb.elf}
}

func (eb *ExecutableBuilder) BSSWriter() Writer {
	return &BufferWrapper{&eb.bss}
}

func (eb *ExecutableBuilder) TextWriter() Writer {
	return &BufferWrapper{&eb.text}
}

func New(machine string) (*ExecutableBuilder, error) {
	arch, err := NewArchitecture(machine)
	if err != nil {
		return nil, err
	}

	return &ExecutableBuilder{
		machine: machine,
		arch:    arch,
		consts:  make(map[string]*Const),
	}, nil
}

var globalLookup = map[string]string{
	"SYS_WRITE": "1",
	"SYS_EXIT":  "60",
	"STDOUT":    "1",
}

func (eb *ExecutableBuilder) Lookup(what string) string {
	if v, ok := globalLookup[what]; ok {
		return v
	}
	if c, ok := eb.consts[what]; ok {
		return strconv.FormatUint(c.addr, 10)
	}
	return "0"
}

func (eb *ExecutableBuilder) Bytes() []byte {
	var result bytes.Buffer
	result.Write(eb.elf.Bytes())
	result.Write(eb.bss.Bytes())
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

func (eb *ExecutableBuilder) BssSection() map[string]string {
	bssSymbols := make(map[string]string)
	for name, c := range eb.consts {
		bssSymbols[name] = c.value
	}
	return bssSymbols
}

func (eb *ExecutableBuilder) BssSize() int {
	size := 0
	for _, data := range eb.BssSection() {
		size += len(data)
	}
	return size
}

func (eb *ExecutableBuilder) WriteBSS(data []byte) uint64 {
	n, _ := eb.bss.Write(data)
	return uint64(n)
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

	platform, err := normalizeMachine(targetMachine)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Fprintf(os.Stderr, "----=[ %s ]=----\n", versionString)

	eb, err := New(platform)
	if err != nil {
		log.Fatalln(err)
	}

	if len(inputFiles) > 0 {
		for _, file := range inputFiles {
			log.Printf("source file: %s", file)
		}
	}

	eb.Define("hello", "Hello, World!\n")

	fmt.Fprintln(os.Stderr, "-> .bss")

	// Prepare the .bss section
	bssSymbols := eb.BssSection()
	bssAddr := baseAddr + headerSize
	currentAddr := uint64(bssAddr)
	for symbol, value := range bssSymbols {
		eb.DefineAddr(symbol, currentAddr)
		currentAddr += eb.WriteBSS([]byte(value))
		fmt.Fprintf(os.Stderr, "%s = %q\n", symbol, value)
	}

	fmt.Fprintln(os.Stderr, "-> .text")

	// Write .text section
	eb.SysWrite("hello")
	eb.SysExit()

	fmt.Fprintln(os.Stderr, "-> ELF header")

	// Write the ELF header
	eb.WriteELFHeader()

	// Output the executable file
	if err := os.WriteFile(outputFile, eb.Bytes(), 0o755); err != nil {
		log.Fatalln(err)
	}
}
