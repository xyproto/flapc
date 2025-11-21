package main

import (
	"bytes"
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
func (eb *ExecutableBuilder) WritePEHeaderWithImports(entryPointRVA uint32, codeSize, dataSize, idataSize, idataRVA uint32) error {
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
		optionalHeaderSize+3*peSectionHeaderSize+codeSize+dataSize+idataSize, peSectionAlign)
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
	for i := 0; i < 16; i++ {
		if i == 1 { // Import directory
			writeU32(idataRVA)   // Import RVA
			writeU32(idataSize)  // Import size
		} else {
			w.Write8u(0)
		}
	}

	return nil
}

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

// Confidence that this function is working: 75%
func (eb *ExecutableBuilder) WritePE(outputPath string) error {
	// For Windows, we need to generate import tables for C runtime
	// Build import directory for msvcrt.dll (C runtime)
	
	// Standard C runtime functions needed by Flap programs
	libraries := map[string][]string{
		"msvcrt.dll": {
			"printf", "exit", "malloc", "free", "realloc",
			"strlen", "memcpy", "memset", "pow", "fflush",
			"sin", "cos", "sqrt", "fopen", "fclose", "fwrite", "fread",
		},
	}
	
	// Write rodata and data content to buffers first
	rodataSymbols := eb.RodataSection()
	for _, value := range rodataSymbols {
		eb.WriteRodata([]byte(value))
	}
	dataSymbols := eb.DataSection()
	for _, value := range dataSymbols {
		eb.data.Write([]byte(value))
	}
	
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

	// Build import data
	idataVirtualAddr := dataVirtualAddr + alignTo(dataSize, peSectionAlign)
	importData, iatMap, err := BuildPEImportData(libraries, idataVirtualAddr)
	if err != nil {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Warning: Failed to build import data: %v\n", err)
		}
		importData = []byte{} // Empty import section
	}
	
	idataSize := uint32(len(importData))
	idataRawSize := alignTo(idataSize, peFileAlign)
	idataRawAddr := dataRawAddr + dataSize

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Import section: size=%d, RVA=0x%x\n", idataSize, idataVirtualAddr)
		fmt.Fprintf(os.Stderr, "IAT mapping: %d functions\n", len(iatMap))
	}

	// Entry point is at start of .text section
	entryPointRVA := textVirtualAddr

	// Write PE header with import directory info
	if err := eb.WritePEHeaderWithImports(entryPointRVA, codeSize, dataSize, idataSize, idataVirtualAddr); err != nil {
		return err
	}

	// Write section headers
	eb.WritePESectionHeader(".text", codeSize, textVirtualAddr, codeSize, textRawAddr,
		scnCntCode|scnMemExecute|scnMemRead)
	eb.WritePESectionHeader(".data", dataSize, dataVirtualAddr, dataSize, dataRawAddr,
		scnCntInitData|scnMemRead|scnMemWrite)
	eb.WritePESectionHeader(".idata", idataSize, idataVirtualAddr, idataRawSize, idataRawAddr,
		scnCntInitData|scnMemRead) // Import section

	// Pad headers to file alignment
	currentPos := uint32(dosHeaderSize + dosStubSize + peSignatureSize + coffHeaderSize +
		optionalHeaderSize + 3*peSectionHeaderSize)
	padding := int(headerSize - currentPos)
	if padding > 0 {
		eb.ELFWriter().WriteN(0, padding)
	}

	// Assign addresses to all data symbols (strings, constants)
	// For PE, the .data section contains both rodata and data
	rodataAddr := peImageBase + uint64(dataVirtualAddr)
	currentAddr := rodataAddr
	
	// First, rodata symbols (read-only)
	for symbol, value := range rodataSymbols {
		eb.DefineAddr(symbol, currentAddr)
		currentAddr += uint64(len(value))
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "PE rodata: %s at 0x%x\n", symbol, eb.consts[symbol].addr)
		}
	}
	
	// Then, data symbols (writable, like cpu_has_avx512)
	for symbol, value := range dataSymbols {
		eb.DefineAddr(symbol, currentAddr)
		currentAddr += uint64(len(value))
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "PE data: %s at 0x%x\n", symbol, eb.consts[symbol].addr)
		}
	}

	// Patch calls to use IAT (Import Address Table)
	eb.PatchPECallsToIAT(iatMap, uint64(textVirtualAddr), uint64(idataVirtualAddr), peImageBase)

	// Patch PC-relative relocations (LEA instructions for data access)
	textAddrFull := peImageBase + uint64(textVirtualAddr)
	eb.PatchPCRelocations(textAddrFull, rodataAddr, eb.rodata.Len())

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

	// .idata section (imports)
	eb.ELFWriter().WriteBytes(importData)
	if pad := int(idataRawSize) - len(importData); pad > 0 {
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

// Confidence that this function is working: 70%
// BuildPEImportData builds the complete import section data for PE files
// Returns: import data, IAT RVA map (funcName -> RVA), error
func BuildPEImportData(libraries map[string][]string, idataRVA uint32) ([]byte, map[string]uint32, error) {
	// Structure of .idata section:
	// 1. Import Directory Table (IDT) - array of IMAGE_IMPORT_DESCRIPTOR (20 bytes each), null-terminated
	// 2. Import Lookup Tables (ILT) - one per DLL, array of RVAs to hint/name entries
	// 3. Import Address Tables (IAT) - one per DLL, same structure as ILT (loader fills this)
	// 4. Hint/Name Table - hint (uint16) + name (null-terminated string) for each function
	// 5. DLL names - null-terminated strings
	
	if len(libraries) == 0 {
		return nil, nil, fmt.Errorf("no libraries to import")
	}
	
	var buf bytes.Buffer
	iatMap := make(map[string]uint32)
	
	// Calculate offsets
	numLibs := len(libraries)
	idtSize := (numLibs + 1) * 20 // +1 for null terminator
	currentOffset := uint32(idtSize)
	
	// Storage for each library's data
	type libData struct {
		name      string
		functions []string
		iltOffset uint32
		iatOffset uint32
		nameOffset uint32
		hintsOffset uint32
	}
	libsData := make([]libData, 0, numLibs)
	
	// First pass: calculate all offsets
	for libName, funcs := range libraries {
		ld := libData{
			name:      libName,
			functions: funcs,
		}
		
		// ILT offset
		ld.iltOffset = currentOffset
		iltSize := uint32((len(funcs) + 1) * 8) // 8 bytes per entry (64-bit), +1 for null
		currentOffset += iltSize
		
		// IAT offset (same size as ILT)
		ld.iatOffset = currentOffset
		currentOffset += iltSize
		
		libsData = append(libsData, ld)
	}
	
	// Hint/Name table offset
	hintsBaseOffset := currentOffset
	
	// Calculate hint/name entries
	for i := range libsData {
		libsData[i].hintsOffset = currentOffset
		for _, funcName := range libsData[i].functions {
			// 2 bytes (hint) + function name + null terminator
			// Align to 2-byte boundary
			entrySize := 2 + len(funcName) + 1
			if entrySize%2 != 0 {
				entrySize++
			}
			currentOffset += uint32(entrySize)
		}
	}
	
	// DLL names offset
	for i := range libsData {
		libsData[i].nameOffset = currentOffset
		currentOffset += uint32(len(libsData[i].name) + 1) // +1 for null terminator
	}
	
	// Write Import Directory Table
	for _, ld := range libsData {
		// IMAGE_IMPORT_DESCRIPTOR
		binary.Write(&buf, binary.LittleEndian, idataRVA+ld.iltOffset) // OriginalFirstThunk (ILT RVA)
		binary.Write(&buf, binary.LittleEndian, uint32(0))              // TimeDateStamp
		binary.Write(&buf, binary.LittleEndian, uint32(0))              // ForwarderChain
		binary.Write(&buf, binary.LittleEndian, idataRVA+ld.nameOffset) // Name RVA
		binary.Write(&buf, binary.LittleEndian, idataRVA+ld.iatOffset) // FirstThunk (IAT RVA)
	}
	// Null terminator for IDT
	binary.Write(&buf, binary.LittleEndian, [20]byte{})
	
	// Write ILTs and IATs for each library
	hintOffset := hintsBaseOffset
	for _, ld := range libsData {
		// Write ILT
		for _, funcName := range ld.functions {
			// RVA to hint/name entry (bit 63 clear = import by name)
			binary.Write(&buf, binary.LittleEndian, uint64(idataRVA+hintOffset))
			
			// Calculate hint/name entry size for next iteration
			entrySize := 2 + len(funcName) + 1
			if entrySize%2 != 0 {
				entrySize++
			}
			hintOffset += uint32(entrySize)
		}
		// Null terminator for ILT
		binary.Write(&buf, binary.LittleEndian, uint64(0))
	}
	
	// Write IATs (same as ILTs initially, loader will fill them)
	for _, ld := range libsData {
		iatBase := idataRVA + ld.iatOffset
		funcIndex := 0
		// Use this library's hint offset
		hintOffset = ld.hintsOffset
		
		for _, funcName := range ld.functions {
			// Store IAT RVA for this function
			iatRVA := iatBase + uint32(funcIndex*8)
			iatMap[funcName] = iatRVA
			
			// RVA to hint/name entry
			binary.Write(&buf, binary.LittleEndian, uint64(idataRVA+hintOffset))
			
			entrySize := 2 + len(funcName) + 1
			if entrySize%2 != 0 {
				entrySize++
			}
			hintOffset += uint32(entrySize)
			funcIndex++
		}
		// Null terminator for IAT
		binary.Write(&buf, binary.LittleEndian, uint64(0))
	}
	
	// Write Hint/Name Table
	for _, ld := range libsData {
		for _, funcName := range ld.functions {
			// Hint (ordinal, we use 0)
			binary.Write(&buf, binary.LittleEndian, uint16(0))
			// Function name
			buf.WriteString(funcName)
			buf.WriteByte(0) // Null terminator
			// Align to 2-byte boundary
			if (2+len(funcName)+1)%2 != 0 {
				buf.WriteByte(0)
			}
		}
	}
	
	// Write DLL names
	for _, ld := range libsData {
		buf.WriteString(ld.name)
		buf.WriteByte(0) // Null terminator
	}
	
	return buf.Bytes(), iatMap, nil
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

// Confidence that this function is working: 80%
// PatchPECallsToIAT patches call instructions to use the Import Address Table (IAT)
// On Windows, we use indirect calls through the IAT instead of PLT stubs
func (eb *ExecutableBuilder) PatchPECallsToIAT(iatMap map[string]uint32, textVirtualAddr, idataVirtualAddr, imageBase uint64) {
	textBytes := eb.text.Bytes()
	
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Patching %d calls to use IAT\n", len(eb.callPatches))
	}
	
	for _, patch := range eb.callPatches {
		// Extract the function name (remove $stub suffix if present)
		funcName := patch.targetName
		if len(funcName) > 5 && funcName[len(funcName)-5:] == "$stub" {
			funcName = funcName[:len(funcName)-5]
		}
		
		// Check if this is an internal function label
		if targetOffset := eb.LabelOffset(funcName); targetOffset >= 0 {
			// Internal function - use direct relative call
			ripAddr := uint64(patch.position) + 4
			targetAddr := uint64(targetOffset)
			displacement := int64(targetAddr) - int64(ripAddr)
			
			if displacement >= -0x80000000 && displacement <= 0x7FFFFFFF {
				disp32 := uint32(displacement)
				textBytes[patch.position] = byte(disp32 & 0xFF)
				textBytes[patch.position+1] = byte((disp32 >> 8) & 0xFF)
				textBytes[patch.position+2] = byte((disp32 >> 16) & 0xFF)
				textBytes[patch.position+3] = byte((disp32 >> 24) & 0xFF)
				
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "  Patched internal call to %s: displacement=%d\n", funcName, displacement)
				}
			}
			continue
		}
		
		// Look up the function in the IAT
		iatRVA, ok := iatMap[funcName]
		if !ok {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "  Warning: Function %s not found in IAT\n", funcName)
			}
			continue
		}
		
		// For Windows x86-64, we need to replace the CALL rel32 (0xE8 XX XX XX XX)
		// with CALL [RIP+disp32] (0xFF 0x15 XX XX XX XX)
		// This is an indirect call through the IAT
		
		// Calculate the RIP-relative offset to the IAT entry
		// The instruction is 6 bytes: FF 15 XX XX XX XX
		// RIP points to the byte after the instruction when accessing memory
		callPos := patch.position - 1 // Position of the 0xE8 byte
		ripAddr := uint64(callPos) + 6 // RIP after the new 6-byte instruction
		iatAddr := iatRVA // IAT is at idataVirtualAddr + offset, but iatRVA is already the full RVA
		
		displacement := int64(iatAddr) - int64(ripAddr)
		
		if displacement < -0x80000000 || displacement > 0x7FFFFFFF {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "  Warning: IAT displacement too large for %s: %d\n", funcName, displacement)
			}
			continue
		}
		
		// Replace CALL rel32 with CALL [RIP+disp32]
		disp32 := uint32(displacement)
		textBytes[callPos] = 0xFF   // CALL r/m64
		textBytes[callPos+1] = 0x15 // ModR/M: RIP-relative addressing
		textBytes[callPos+2] = byte(disp32 & 0xFF)
		textBytes[callPos+3] = byte((disp32 >> 8) & 0xFF)
		textBytes[callPos+4] = byte((disp32 >> 16) & 0xFF)
		textBytes[callPos+5] = byte((disp32 >> 24) & 0xFF)
		
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "  Patched IAT call to %s: IAT RVA=0x%x, displacement=%d\n", funcName, iatRVA, displacement)
		}
	}
}

// Helper function to write import descriptor
func writePEImportDescriptor(buf []byte, offset int, ilt, iat, name uint32, timeDateStamp uint32) {
	binary.LittleEndian.PutUint32(buf[offset:], ilt)          // RVA to ILT
	binary.LittleEndian.PutUint32(buf[offset+4:], timeDateStamp) // TimeDateStamp
	binary.LittleEndian.PutUint32(buf[offset+8:], 0)          // ForwarderChain
	binary.LittleEndian.PutUint32(buf[offset+12:], name)      // RVA to DLL name
	binary.LittleEndian.PutUint32(buf[offset+16:], iat)       // RVA to IAT
}
