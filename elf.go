package main

import (
	"fmt"
	"os"
)

const (
	baseAddr   = 0x400000 // virtual base address
	headerSize = 64 + 56  // ELF + header size
)

func (eb *ExecutableBuilder) WriteELFHeader() error {
	w := eb.ELFWriter()
	bssSize := eb.bss.Len()
	codeSize := eb.text.Len()

	// Magic
	w.Write(0x7f)
	w.Write(0x45)  // E
	w.Write(0x4c)  // L
	w.Write(0x46)  // F
	w.Write(2)     // 64-bit
	w.Write(1)     // little endian
	w.Write(1)     // ELF version
	w.Write(3)     // Linux
	w.Write(3)     // ABI version, dynamic linker version
	w.WriteN(0, 7) // zero padding, length of 7
	w.Write2(2)    // object file type: executable

	// Machine type - platform specific
	w.Write2(byte(eb.arch.ELFMachineType()))

	w.Write4(1) // original ELF version (?)

	fmt.Fprintln(os.Stderr)

	entry := uint64(baseAddr + headerSize + bssSize)

	w.Write8u(entry)                    // address of entry point
	w.Write8(0x40)                      // program header table
	const sectionAddr = 0x40 + 0x38     // right after ELF header + program header
	w.Write8u(sectionAddr)              // start of section header table
	w.Write4(0)                         // "interpretation of this field depends on the target architecture"
	w.Write2(64)                        // size of this ELF header
	w.Write2(0x38)                      // size of a program header table entry
	const programHeaderTableEntries = 1 // one LOAD segment
	w.Write2(programHeaderTableEntries) // number of entries in the program header table
	w.Write2(0x40)                      // size of a section header table entry
	const sectionHeaderTableEntries = 0 // 5 for: .text, .data, .bss, .shstrtab, .symtab. Can be 0 for minimal executables.
	w.Write2(sectionHeaderTableEntries)
	const sectionHeaderTableEntryIndex = 0 // .shstrtab at index 3, or 0 if there are no sections
	w.Write2(sectionHeaderTableEntryIndex)

	fmt.Fprintln(os.Stderr)

	// Program header
	w.Write4(1)         // PT_LOAD
	w.Write4(5)         // flags: PF_X | PF_R (executable + readable)
	w.Write8u(0)        // offset in file
	w.Write8u(baseAddr) // virtual address
	w.Write8u(baseAddr) // physical address
	fileSize := uint64(headerSize + bssSize + codeSize)
	w.Write8u(fileSize) // size in file
	w.Write8u(fileSize) // size in memory
	w.Write8u(0x1000)   // alignment

	fmt.Fprintln(os.Stderr)

	return nil
}
