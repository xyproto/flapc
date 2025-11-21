package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

// PE (Portable Executable) format constants for Windows x86_64
const (
	// DOS header (stub)
	dosHeaderSize = 64
	dosStubSize   = 128

	// PE headers
	peSignatureSize     = 4
	coffHeaderSize      = 20
	optionalHeaderSize  = 240 // PE32+ (64-bit)
	peSectionHeaderSize = 40

	// Memory layout for PE
	peImageBase       = 0x140000000 // Standard Windows x64 image base
	peSectionAlign    = 0x1000      // 4KB section alignment in memory
	peFileAlign       = 0x200       // 512 byte file alignment
	
	// Section characteristics
	scnMemExecute = 0x20000000
	scnMemRead    = 0x40000000
	scnMemWrite   = 0x80000000
	scnCntCode    = 0x00000020
	scnCntInitData = 0x00000040
)

// Confidence that this function is working: 75%
func (eb *ExecutableBuilder) WritePEHeader(entryPointRVA uint32, codeSize, dataSize uint32) error {
	w := eb.ELFWriter() // Reuse the writer
	
	// Helper functions to write multi-byte values
	writeU16 := func(v uint16) {
		w.WriteBytes([]byte{byte(v), byte(v >> 8)})
	}
	writeU32 := func(v uint32) {
		w.WriteBytes([]byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)})
	}

	// === DOS Header (64 bytes) ===
	writeU16(0x5A4D) // "MZ" signature
	w.WriteN(0, 58)  // Zero bytes 2-59
	// At offset 0x3C (60), write the PE header offset
	peHeaderOffset := uint32(dosHeaderSize + dosStubSize)
	writeU32(peHeaderOffset) // PE header offset

	// === DOS Stub (simple one that just prints "This program cannot be run in DOS mode") ===
	// For simplicity, we'll just pad with zeros (minimal stub)
	stubMsg := []byte("This program requires Windows.\r\n$")
	w.WriteBytes(stubMsg)
	w.WriteN(0, dosStubSize-len(stubMsg))

	// === PE Signature ===
	writeU32(0x00004550) // "PE\0\0"

	// === COFF File Header (20 bytes) ===
	writeU16(0x8664)     // Machine: AMD64
	writeU16(3)          // Number of sections (.text, .data, .idata)
	writeU32(0)          // TimeDateStamp (0 for reproducibility)
	writeU32(0)          // Pointer to symbol table (deprecated)
	writeU32(0)          // Number of symbols (deprecated)
	writeU16(optionalHeaderSize) // Size of optional header
	writeU16(0x0022)     // Characteristics: EXECUTABLE_IMAGE | LARGE_ADDRESS_AWARE

	// === Optional Header (PE32+) ===
	writeU16(0x020B)     // Magic: PE32+ (64-bit)
	w.Write(1)           // Major linker version
	w.Write(0)           // Minor linker version
	writeU32(codeSize)   // Size of code
	writeU32(dataSize)   // Size of initialized data
	writeU32(0)          // Size of uninitialized data
	writeU32(entryPointRVA) // Entry point RVA
	writeU32(0x1000)     // Base of code

	// PE32+ specific fields
	w.Write8u(peImageBase) // Image base
	writeU32(peSectionAlign) // Section alignment
	writeU32(peFileAlign)    // File alignment
	writeU16(6)          // Major OS version
	writeU16(0)          // Minor OS version
	writeU16(0)          // Major image version
	writeU16(0)          // Minor image version
	writeU16(6)          // Major subsystem version
	writeU16(0)          // Minor subsystem version
	writeU32(0)          // Win32 version value (reserved)

	// Calculate image size (aligned to section alignment)
	imageSize := alignTo(dosHeaderSize+dosStubSize+peSignatureSize+coffHeaderSize+
		optionalHeaderSize+3*peSectionHeaderSize+codeSize+dataSize, peSectionAlign)
	writeU32(imageSize) // Size of image

	headersSize := alignTo(dosHeaderSize+dosStubSize+peSignatureSize+coffHeaderSize+
		optionalHeaderSize+3*peSectionHeaderSize, peFileAlign)
	writeU32(headersSize) // Size of headers

	writeU32(0)          // Checksum
	writeU16(3)          // Subsystem: CUI (Console)
	writeU16(0x8160)     // DLL characteristics: NX compatible, dynamic base, terminal server aware
	w.Write8u(0x100000)  // Size of stack reserve
	w.Write8u(0x1000)    // Size of stack commit
	w.Write8u(0x100000)  // Size of heap reserve
	w.Write8u(0x1000)    // Size of heap commit
	writeU32(0)          // Loader flags
	writeU32(16)         // Number of data directories

	// Data directories (16 entries, each 8 bytes: RVA + Size)
	// We'll fill in import directory later, rest are zeros
	for i := 0; i < 16; i++ {
		if i == 1 { // Import directory
			// Will be filled in later when we have actual imports
			writeU32(0) // Import RVA (placeholder)
			writeU32(0) // Import size (placeholder)
		} else {
			w.Write8u(0)
		}
	}

	return nil
}

// Confidence that this function is working: 80%
func (eb *ExecutableBuilder) WritePESectionHeader(name string, virtualSize, virtualAddr, rawSize, rawAddr uint32, characteristics uint32) {
	w := eb.ELFWriter()
	
	writeU16 := func(v uint16) {
		w.WriteBytes([]byte{byte(v), byte(v >> 8)})
	}
	writeU32 := func(v uint32) {
		w.WriteBytes([]byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)})
	}

	// Section name (8 bytes, null-padded)
	nameBytes := []byte(name)
	if len(nameBytes) > 8 {
		nameBytes = nameBytes[:8]
	}
	w.WriteBytes(nameBytes)
	w.WriteN(0, 8-len(nameBytes))

	writeU32(virtualSize)
	writeU32(virtualAddr)
	writeU32(rawSize)
	writeU32(rawAddr)
	writeU32(0) // Pointer to relocations
	writeU32(0) // Pointer to line numbers
	writeU16(0) // Number of relocations
	writeU16(0) // Number of line numbers
	writeU32(characteristics)
}

// Confidence that this function is working: 70%
func (eb *ExecutableBuilder) WritePE(outputPath string) error {
	// For Windows, we need to generate import tables for C runtime
	// For now, we'll create a minimal PE that links to msvcrt.dll

	codeSize := uint32(eb.text.Len())
	dataSize := uint32(eb.rodata.Len() + eb.data.Len())

	// Align sizes to file alignment
	codeSize = alignTo(codeSize, peFileAlign)
	dataSize = alignTo(dataSize, peFileAlign)

	// Calculate section positions
	headerSize := uint32(dosHeaderSize + dosStubSize + peSignatureSize + coffHeaderSize +
		optionalHeaderSize + 3*peSectionHeaderSize)
	headerSize = alignTo(headerSize, peFileAlign)

	textRawAddr := uint32(headerSize)
	textVirtualAddr := uint32(0x1000) // First section after headers

	dataRawAddr := textRawAddr + codeSize
	dataVirtualAddr := textVirtualAddr + alignTo(codeSize, peSectionAlign)

	idataRawAddr := dataRawAddr + dataSize
	idataVirtualAddr := dataVirtualAddr + alignTo(dataSize, peSectionAlign)

	// Entry point is at start of .text section
	entryPointRVA := textVirtualAddr

	// Write PE header
	if err := eb.WritePEHeader(entryPointRVA, codeSize, dataSize); err != nil {
		return err
	}

	// Write section headers
	eb.WritePESectionHeader(".text", codeSize, textVirtualAddr, codeSize, textRawAddr,
		scnCntCode|scnMemExecute|scnMemRead)
	eb.WritePESectionHeader(".data", dataSize, dataVirtualAddr, dataSize, dataRawAddr,
		scnCntInitData|scnMemRead|scnMemWrite)
	eb.WritePESectionHeader(".idata", 0, idataVirtualAddr, 0, idataRawAddr,
		scnCntInitData|scnMemRead) // Import section (minimal for now)

	// Pad headers to file alignment
	currentPos := uint32(dosHeaderSize + dosStubSize + peSignatureSize + coffHeaderSize +
		optionalHeaderSize + 3*peSectionHeaderSize)
	padding := int(headerSize - currentPos)
	if padding > 0 {
		eb.ELFWriter().WriteN(0, padding)
	}

	// Write sections
	// .text section
	eb.ELFWriter().WriteBytes(eb.text.Bytes())
	if pad := int(codeSize) - eb.text.Len(); pad > 0 {
		eb.ELFWriter().WriteN(0, pad)
	}

	// .data section (combine rodata and data)
	eb.ELFWriter().WriteBytes(eb.rodata.Bytes())
	eb.ELFWriter().WriteBytes(eb.data.Bytes())
	if pad := int(dataSize) - eb.rodata.Len() - eb.data.Len(); pad > 0 {
		eb.ELFWriter().WriteN(0, pad)
	}

	// Write to file
	if err := os.WriteFile(outputPath, eb.elf.Bytes(), 0755); err != nil {
		return fmt.Errorf("failed to write PE file: %v", err)
	}

	return nil
}

// alignTo aligns a value to the given alignment
func alignTo(value, align uint32) uint32 {
	return (value + align - 1) & ^(align - 1)
}

// Confidence that this function is working: 60%
func (eb *ExecutableBuilder) WritePEWithImports(outputPath string, imports []string) error {
	// TODO: Implement proper import table generation
	// For now, just create a minimal PE without imports
	// This will need to be expanded to support msvcrt.dll imports (printf, malloc, etc.)
	
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Warning: PE import tables not yet fully implemented\n")
		fmt.Fprintf(os.Stderr, "Required imports: %v\n", imports)
	}

	return eb.WritePE(outputPath)
}

// Confidence that this function is working: 85%
// WritePEImportDirectory writes the import directory table for PE files
func (eb *ExecutableBuilder) WritePEImportDirectory(libraries map[string][]string) ([]byte, error) {
	// libraries maps DLL name to list of function names
	// Returns the import directory data to be placed in .idata section
	
	buf := make([]byte, 0, 4096)
	
	// For each library, we need:
	// - Import Directory Entry (20 bytes)
	// - Import Lookup Table (ILT)
	// - Import Address Table (IAT) 
	// - Hint/Name table
	// - Library name string
	
	// TODO: Implement full import table generation
	// This is complex and requires:
	// 1. Calculate all offsets
	// 2. Build ILT and IAT
	// 3. Build hint/name entries
	// 4. Write library names
	// 5. Null-terminate directory
	
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Building PE import directory for %d libraries\n", len(libraries))
	}
	
	return buf, nil
}

// Confidence that this function is working: 50%
// WritePERelocations writes base relocation table for PE files
func (eb *ExecutableBuilder) WritePERelocations() ([]byte, error) {
	// Base relocations allow the PE loader to adjust addresses if the image
	// is loaded at a different base address
	
	// For simplicity, we'll generate minimal relocations
	// In a full implementation, we'd need to track all absolute addresses
	// and create relocation entries for them
	
	return []byte{}, nil
}

// Helper function to write import descriptor
func writePEImportDescriptor(buf []byte, offset int, ilt, iat, name uint32, timeDateStamp uint32) {
	binary.LittleEndian.PutUint32(buf[offset:], ilt)          // RVA to ILT
	binary.LittleEndian.PutUint32(buf[offset+4:], timeDateStamp) // TimeDateStamp
	binary.LittleEndian.PutUint32(buf[offset+8:], 0)          // ForwarderChain
	binary.LittleEndian.PutUint32(buf[offset+12:], name)      // RVA to DLL name
	binary.LittleEndian.PutUint32(buf[offset+16:], iat)       // RVA to IAT
}
