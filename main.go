package main

import (
	"bytes"
	"log"
	"os"
	"strconv"
)

// A tiny compiler for x86_64 ELF files for Linux

const platform = "x86_64"

type Const struct {
	value string
	addr  uint64
}

type ExecutableBuilder struct {
	platform string
	consts   map[string]*Const

	elf, bss, text bytes.Buffer // The ELF header, .bss and .text sections, as bytes
}

func New() *ExecutableBuilder {
	return &ExecutableBuilder {
		platform = platform,
		consts = make(map[string]*Const),
	}
}

var globalLookup = map[string]} string{
	"SYS_WRITE": "1",
	"SYS_EXIT": "60",
	"STDOUT": "1",
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
	return eb.buf.Bytes()
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
	eb.Emit("mov rax, " + GlobalLookup("SYS_WRITE"))
	eb.Emit("mov rdi, " + GlobalLookup("STDOUT"))
	eb.Emit("mov rsi, " + what_data)
	if len(what_data_len) == 0 {
		if c, ok := eb.consts[what_data]; ok {
			eb.Emit("mov rdx, " + strconv.Itoa(len(c.value)))
		}
	} else {
		eb.Emit("mov rdx, " + what_data_len[0])
	}
	eb.Emit("syscall")
}

func (eb *ExecutableBuilder) SysExit(code ...string) {
	eb.Emit("mov rax, " + GlobalLookup("SYS_EXIT"))
	if len(code) == 0 {
		eb.Emit("mov rdi, 0")
	} else {
		eb.Emit("mov rdi, " + code[0])
	}
	eb.Emit("syscall")
}

func main() {
	eb := NewExecutableBuilder()

	// Define constants
	eb.Define("hello", "Hello, World!\n")

	// Prepare the .bss section
	bssSymbols, bssSize := eb.BssSection()
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
	eb.WriteELFHeader(eb.bss.Len(), eb.text.Len())

	// Output the executable file
	const filename = "hello"
	err := os.WriteFile(filename, eb.Bytes(), 0o755)
	if err != nil {
		log.Fatalln(err)
	}
}
