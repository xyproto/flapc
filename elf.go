package main

func (o *Out) WriteELF() error {
	size := 0
	size += o.Write(0x7f)
	size += o.Write(0x45)                       // E
	size += o.Write(0x4c)                       // L
	size += o.Write(0x46)                       // F
	size += o.Write(2)                          // 64-bit
	size += o.Write(1)                          // little endian
	size += o.Write(1)                          // ELF version
	size += o.Write(3)                          // Linux
	size += o.Write(3)                          // ABI version, dynamic linker version
	size += o.WriteN(0, 7)                      // zero padding, length of 7
	size += o.Write2(2)                         // object file type: executable
	size += o.Write2(0x3e)                      // machine (AMD x86-64), ARM64 is 0xB7, RISC-V is 0xF3
	size += o.Write4(1)                         // original ELF version (?)
	const startAddr = 0x80                      // ?
	size += o.Write8(startAddr)                 // address of entry point
	size += o.Write8(0x40)                      // program header table
	const sectionAddr = 0xff                    // ?
	size += o.Write8(sectionAddr)               // start of section header table
	size += o.Write4(0)                         // "interpretation of this field depends on the target architecture"
	size += o.Write2(64)                        // size of this ELF header
	size += o.Write2(0x38)                      // size of a program header table entry
	const programHeaderTableEntries = 1         // ?
	size += o.Write2(programHeaderTableEntries) // number of entries in the program header table
	size += o.Write2(0x40)                      // size of a section header table entry
	const sectionHeaderTableEntries = 1         // ?
	size += o.Write2(sectionHeaderTableEntries)
	const sectionHeaderTableEntryIndex = 0 // ?
	size += o.Write2(sectionHeaderTableEntryIndex)
	o.Write(byte(size))

	psize := 0
	psize += o.Write4()

	return nil
}
