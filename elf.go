package main

func (o *Out) WriteELF() error {
	o.Write(0x7f)
	o.Write(0x45)                       // E
	o.Write(0x4c)                       // L
	o.Write(0x46)                       // F
	o.Write(2)                          // 64-bit
	o.Write(1)                          // little endian
	o.Write(1)                          // ELF version
	o.Write(3)                          // Linux
	o.Write(3)                          // ABI version, dynamic linker version
	o.WriteN(0, 7)                      // zero padding, length of 7
	o.Write2(2)                         // object file type: executable
	o.Write2(0x3e)                      // machine (AMD x86-64), ARM64 is 0xB7, RISC-V is 0xF3
	o.Write4(1)                         // original ELF version (?)
	const startAddr = 0x80              // ?
	o.Write8(startAddr)                 // address of entry point
	o.Write8(0x40)                      // program header table
	const sectionAddr = 0xff            // ?
	o.Write8(sectionAddr)               // start of section header table
	o.Write4(0)                         // "interpretation of this field depends on the target architecture"
	o.Write2(64)                        // size of this ELF header
	o.Write2(0x38)                      // size of a program header table entry
	const programHeaderTableEntries = 1 // ?
	o.Write2(programHeaderTableEntries) // number of entries in the program header table
	o.Write2(0x40)                      // size of a section header table entry
	const sectionHeaderTableEntries = 1 // ?
	o.Write2(sectionHeaderTableEntries)
	const sectionHeaderTableEntryIndex = 0 // ?
	o.Write2(sectionHeaderTableEntryIndex)
	o.Write(64)

	return nil
}
