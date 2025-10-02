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
	rodataSize := eb.rodata.Len()
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

	// Machine type - machine specific
	w.Write2(byte(eb.arch.ELFMachineType()))

	w.Write4(1) // original ELF version (?)

	fmt.Fprintln(os.Stderr)

	entry := uint64(baseAddr + headerSize + rodataSize)

	w.Write8u(entry)
	w.Write8(0x40)
	const sectionAddr = 0x40 + 0x38
	w.Write8u(sectionAddr)
	w.Write4(0)
	w.Write2(64)
	w.Write2(0x38)
	const programHeaderTableEntries = 1
	w.Write2(programHeaderTableEntries)
	w.Write2(0x40)
	const sectionHeaderTableEntries = 0
	w.Write2(sectionHeaderTableEntries)
	const sectionHeaderTableEntryIndex = 0
	w.Write2(sectionHeaderTableEntryIndex)

	fmt.Fprintln(os.Stderr)

	w.Write4(1)
	w.Write4(7)
	w.Write8u(0)
	w.Write8u(baseAddr)
	w.Write8u(baseAddr)
	fileSize := uint64(headerSize + rodataSize + codeSize)
	w.Write8u(fileSize)
	w.Write8u(fileSize)
	w.Write8u(0x1000)

	fmt.Fprintln(os.Stderr)

	return nil
}
