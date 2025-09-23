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
	platform string
	consts   map[string]*Const

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

func New(platform string) *ExecutableBuilder {
	return &ExecutableBuilder{
		platform: platform,
		consts:   make(map[string]*Const),
	}
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

func (eb *ExecutableBuilder) SysWrite(what_data string, what_data_len ...string) {
	switch eb.platform {
	case "x86_64":
		eb.Emit("mov rax, " + eb.Lookup("SYS_WRITE"))
		eb.Emit("mov rdi, " + eb.Lookup("STDOUT"))
		eb.Emit("mov rsi, " + what_data)
		if len(what_data_len) == 0 {
			if c, ok := eb.consts[what_data]; ok {
				eb.Emit("mov rdx, " + strconv.Itoa(len(c.value)))
			}
		} else {
			eb.Emit("mov rdx, " + what_data_len[0])
		}
	case "aarch64":
		eb.Emit("mov x8, " + eb.Lookup("SYS_WRITE"))
		eb.Emit("mov x0, " + eb.Lookup("STDOUT"))
		eb.Emit("mov x1, " + what_data)
		if len(what_data_len) == 0 {
			if c, ok := eb.consts[what_data]; ok {
				eb.Emit("mov x2, " + strconv.Itoa(len(c.value)))
			}
		} else {
			eb.Emit("mov x2, " + what_data_len[0])
		}
	case "riscv64":
		eb.Emit("mov a7, " + eb.Lookup("SYS_WRITE"))
		eb.Emit("mov a0, " + eb.Lookup("STDOUT"))
		eb.Emit("mov a1, " + what_data)
		if len(what_data_len) == 0 {
			if c, ok := eb.consts[what_data]; ok {
				eb.Emit("mov a2, " + strconv.Itoa(len(c.value)))
			}
		} else {
			eb.Emit("mov a2, " + what_data_len[0])
		}
	}
	eb.Emit("syscall")
}

func (eb *ExecutableBuilder) SysExit(code ...string) {
	switch eb.platform {
	case "x86_64":
		eb.Emit("mov rax, " + eb.Lookup("SYS_EXIT"))
		if len(code) == 0 {
			eb.Emit("mov rdi, 0")
		} else {
			eb.Emit("mov rdi, " + code[0])
		}
	case "aarch64":
		eb.Emit("mov x8, " + eb.Lookup("SYS_EXIT"))
		if len(code) == 0 {
			eb.Emit("mov x0, 0")
		} else {
			eb.Emit("mov x0, " + code[0])
		}
	case "riscv64":
		eb.Emit("mov a7, " + eb.Lookup("SYS_EXIT"))
		if len(code) == 0 {
			eb.Emit("mov a0, 0")
		} else {
			eb.Emit("mov a0, " + code[0])
		}
	}
	eb.Emit("syscall")
}

func (eb *ExecutableBuilder) WriteBSS(data []byte) {
	eb.bss.Write(data)
}

func main() {
	var machine = flag.String("m", "x86_64", "target machine architecture (x86_64, amd64, arm64, aarch64, riscv64, riscv, rv64)")
	var machineLong = flag.String("machine", "x86_64", "target machine architecture (x86_64, amd64, arm64, aarch64, riscv64, riscv, rv64)")
	var filename = "a.out"

	flag.Parse()

	// Use whichever flag was specified (prefer short form if both given)
	targetMachine := *machine
	if *machineLong != "x86_64" {
		targetMachine = *machineLong
	}

	platform, err := normalizeMachine(targetMachine)
	if err != nil {
		log.Fatalln(err)
	}

	eb := New(platform)

	// Define constants
	eb.Define("hello", "Hello, World!\n")

	// Prepare the .bss section
	bssSymbols := eb.BssSection()
	bssAddr := baseAddr + headerSize
	currentAddr := uint64(bssAddr)
	for symbol, data := range bssSymbols {
		eb.DefineAddr(symbol, currentAddr)
		currentAddr += uint64(len(data))
		eb.WriteBSS([]byte(data))
	}

	// Write .text section
	eb.SysWrite("hello")
	eb.SysExit()

	// Write the ELF header
	eb.WriteELFHeader()

	if err := os.WriteFile(filename, eb.Bytes(), 0o755); err != nil {
		log.Fatalln(err)
	}
}
