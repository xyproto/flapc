// Completion: 100% - Platform support complete
package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

// GeneratePLT creates PLT (Procedure Linkage Table) stubs for x86_64
func (ds *DynamicSections) GeneratePLT(functions []string, gotBase uint64, pltBase uint64) {
	ds.plt.Reset()
	ds.pltEntries = functions

	// PLT[0] - special resolver stub
	// pushq GOT[1]
	ds.plt.Write([]byte{0xff, 0x35})
	offset1 := uint32(gotBase + 8 - pltBase - 6) // offset to GOT[1]
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "PLT[0] push offset: gotBase=0x%x, pltBase=0x%x, offset=0x%x\n", gotBase, pltBase, offset1)
	}
	binary.Write(&ds.plt, binary.LittleEndian, offset1)

	// jmpq *GOT[2]
	ds.plt.Write([]byte{0xff, 0x25})
	offset2 := uint32(gotBase + 16 - pltBase - 12) // offset to GOT[2]
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "PLT[0] jmp offset: offset=0x%x\n", offset2)
	}
	binary.Write(&ds.plt, binary.LittleEndian, offset2)

	// padding
	ds.plt.Write([]byte{0x0f, 0x1f, 0x40, 0x00})

	// PLT[1..n] - one per function
	for i := range functions {
		pltOffset := pltBase + uint64(ds.plt.Len())
		gotOffset := gotBase + uint64(24+i*8) // GOT[0,1,2] reserved, functions start at GOT[3]

		// jmpq *GOT[n]
		ds.plt.Write([]byte{0xff, 0x25})
		relOffset := int32(gotOffset - pltOffset - 6)
		binary.Write(&ds.plt, binary.LittleEndian, relOffset)

		// pushq $index
		ds.plt.Write([]byte{0x68})
		binary.Write(&ds.plt, binary.LittleEndian, uint32(i))

		// jmpq PLT[0]
		ds.plt.Write([]byte{0xe9})
		jumpBack := int32(pltBase - pltOffset - 16)
		binary.Write(&ds.plt, binary.LittleEndian, jumpBack)
	}
}

// GenerateGOT creates GOT (Global Offset Table) entries
func (ds *DynamicSections) GenerateGOT(functions []string, dynamicAddr uint64, pltBase uint64) {
	ds.got.Reset()

	// GOT[0] = address of _DYNAMIC
	binary.Write(&ds.got, binary.LittleEndian, dynamicAddr)

	// GOT[1] = link_map (filled by dynamic linker)
	binary.Write(&ds.got, binary.LittleEndian, uint64(0))

	// GOT[2] = _dl_runtime_resolve (filled by dynamic linker)
	binary.Write(&ds.got, binary.LittleEndian, uint64(0))

	// GOT[3..n] = PLT stubs (initial values point to PLT push instructions)
	for i := range functions {
		// Point to the push instruction in PLT[i+1]
		// PLT[0] is 16 bytes, each PLT entry is 16 bytes
		pltPushAddr := pltBase + 16 + uint64(i*16) + 6
		binary.Write(&ds.got, binary.LittleEndian, pltPushAddr)
	}
}

// GetPLTOffset returns the offset within PLT for a function
func (ds *DynamicSections) GetPLTOffset(funcName string) int {
	for i, name := range ds.pltEntries {
		if name == funcName {
			// PLT[0] is 16 bytes, each entry is 16 bytes
			return 16 + i*16
		}
	}
	return -1
}

// GeneratePLTCall generates a call instruction to a PLT entry
func GeneratePLTCall(w Writer, funcName string, pltOffset int, currentAddr uint64, pltBase uint64) {
	// Calculate relative offset from current position to PLT entry
	targetAddr := pltBase + uint64(pltOffset)
	// Current position after the call instruction
	nextInstr := currentAddr + 5 // call is 5 bytes

	relOffset := int32(targetAddr - nextInstr)

	// Write call instruction
	w.Write(0xe8) // call rel32
	binary.Write(w.(*BufferWrapper).buf, binary.LittleEndian, relOffset)
}
