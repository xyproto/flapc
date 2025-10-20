package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

// WriteCompleteDynamicELF generates a fully functional dynamically-linked ELF
// Returns (gotBase, rodataBase, error)
func (eb *ExecutableBuilder) WriteCompleteDynamicELF(ds *DynamicSections, functions []string) (gotBase, rodataAddr, textAddr, pltBase uint64, err error) {
	eb.elf.Reset()
	eb.neededFunctions = functions // Store functions list for later use in patchTextInELF

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG [WriteCompleteDynamicELF start]: rodata buffer size: %d bytes\n", eb.rodata.Len())
	}
	if eb.rodata.Len() > 0 {
		previewLen := 32
		if eb.rodata.Len() < previewLen {
			previewLen = eb.rodata.Len()
		}
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG [WriteCompleteDynamicELF start]: rodata buffer first %d bytes: %q\n", previewLen, eb.rodata.Bytes()[:previewLen])
		}
	}

	rodataSize := eb.rodata.Len()
	codeSize := eb.text.Len()

	// Build all dynamic sections first
	ds.buildSymbolTable()
	ds.buildHashTable()

	// Pre-generate PLT and GOT with dummy values to get correct sizes
	ds.GeneratePLT(functions, 0, 0)
	ds.GenerateGOT(functions, 0, 0)

	// Calculate memory layout
	const pageSize = 0x1000
	const elfHeader = 64
	const progHeaderSize = 56

	// We need: PHDR, INTERP, LOAD(ro), LOAD(rx), LOAD(rw), DYNAMIC
	numProgHeaders := 6
	headersSize := uint64(elfHeader + progHeaderSize*numProgHeaders)

	// Align to page boundary
	alignedHeaders := (headersSize + pageSize - 1) & ^uint64(pageSize-1)

	// Layout sections in memory
	layout := make(map[string]struct {
		offset uint64
		addr   uint64
		size   int
	})

	// Read-only data segment
	currentOffset := uint64(alignedHeaders)
	currentAddr := baseAddr + currentOffset

	// Interpreter string
	interp := eb.getInterpreterPath()
	interpSize := len(interp) + 1
	layout["interp"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{currentOffset, currentAddr, interpSize}
	currentOffset += uint64((interpSize + 7) & ^7) // align to 8 bytes
	currentAddr += uint64((interpSize + 7) & ^7)

	// .dynsym
	layout["dynsym"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{currentOffset, currentAddr, ds.dynsym.Len()}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "dynsym: offset=0x%x, size=%d, aligned=%d\n",
			currentOffset, ds.dynsym.Len(), (ds.dynsym.Len()+7) & ^7)
	}
	currentOffset += uint64((ds.dynsym.Len() + 7) & ^7)
	currentAddr += uint64((ds.dynsym.Len() + 7) & ^7)

	// .dynstr
	layout["dynstr"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{currentOffset, currentAddr, ds.dynstr.Len()}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "dynstr: offset=0x%x, size=%d, aligned=%d, next offset=0x%x\n",
			currentOffset, ds.dynstr.Len(), (ds.dynstr.Len()+7) & ^7,
			currentOffset+uint64((ds.dynstr.Len()+7) & ^7))
	}
	currentOffset += uint64((ds.dynstr.Len() + 7) & ^7)
	currentAddr += uint64((ds.dynstr.Len() + 7) & ^7)

	// .hash
	layout["hash"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{currentOffset, currentAddr, ds.hash.Len()}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "hash: offset=0x%x, size=%d, aligned=%d\n",
			currentOffset, ds.hash.Len(), (ds.hash.Len()+7) & ^7)
	}
	currentOffset += uint64((ds.hash.Len() + 7) & ^7)
	currentAddr += uint64((ds.hash.Len() + 7) & ^7)

	// .rela.plt
	layout["rela"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{currentOffset, currentAddr, ds.rela.Len()}
	currentOffset += uint64((ds.rela.Len() + 7) & ^7)
	currentAddr += uint64((ds.rela.Len() + 7) & ^7)

	// Align to page for executable segment
	currentOffset = (currentOffset + pageSize - 1) & ^uint64(pageSize-1)
	currentAddr = (currentAddr + pageSize - 1) & ^uint64(pageSize-1)

	// .plt (executable)
	pltBase = currentAddr
	layout["plt"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{currentOffset, currentAddr, ds.plt.Len()}
	currentOffset += uint64(ds.plt.Len())
	currentAddr += uint64(ds.plt.Len())

	// ._start (entry point - clears registers and jumps to user code)
	startSize := 14 // Size of our minimal _start function (3*3 bytes clear regs + 5 bytes jmp)
	layout["_start"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{currentOffset, currentAddr, startSize}
	entryPoint := currentAddr // Entry point is _start
	currentOffset += uint64((startSize + 7) & ^7)
	currentAddr += uint64((startSize + 7) & ^7)

	// .text (our code)
	layout["text"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{currentOffset, currentAddr, codeSize}
	currentOffset += uint64((codeSize + 7) & ^7)
	currentAddr += uint64((codeSize + 7) & ^7)

	// Align to page for writable segment
	currentOffset = (currentOffset + pageSize - 1) & ^uint64(pageSize-1)
	currentAddr = (currentAddr + pageSize - 1) & ^uint64(pageSize-1)

	// .dynamic
	dynamicAddr := currentAddr
	layout["dynamic"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{currentOffset, currentAddr, ds.dynamic.Len()}
	currentOffset += uint64((ds.dynamic.Len() + 7) & ^7)
	currentAddr += uint64((ds.dynamic.Len() + 7) & ^7)

	// .got
	gotBase = currentAddr
	layout["got"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{currentOffset, currentAddr, ds.got.Len()}
	currentOffset += uint64((ds.got.Len() + 7) & ^7)
	currentAddr += uint64((ds.got.Len() + 7) & ^7)

	// .rodata
	layout["rodata"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{currentOffset, currentAddr, rodataSize}
	currentAddr += uint64(rodataSize)

	// Now generate PLT and GOT with correct addresses
	ds.GeneratePLT(functions, gotBase, pltBase)
	ds.GenerateGOT(functions, dynamicAddr, pltBase)

	// Add relocations with TEMPORARY addresses - will be updated later
	for i := range functions {
		symIndex := uint32(i + 1) // +1 because null symbol is at index 0
		// GOT entries start after 3 reserved entries (24 bytes)
		gotEntryAddr := gotBase + uint64(24+i*8)
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Adding TEMPORARY relocation for %s: GOT entry at 0x%x, symIndex=%d\n",
				functions[i], gotEntryAddr, symIndex)
		}
		ds.AddRelocation(gotEntryAddr, symIndex, R_X86_64_JUMP_SLOT)
	}

	// Update layout with actual sizes
	layout["rela"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{layout["rela"].offset, layout["rela"].addr, ds.rela.Len()}

	layout["plt"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{layout["plt"].offset, layout["plt"].addr, ds.plt.Len()}

	// Recalculate .text offset now that PLT size is known (including _start)
	startSizeAligned := ((startSize + 7) & ^7)
	textOffset := layout["plt"].offset + uint64(ds.plt.Len()) + uint64(startSizeAligned)
	textAddr = layout["plt"].addr + uint64(ds.plt.Len()) + uint64(startSizeAligned)
	layout["text"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{textOffset, textAddr, layout["text"].size}

	// Note: We don't patch PLT calls here because the code will be regenerated
	// and patched later by the caller (see parser.go:1448 and default.go:59)
	// The initial .text is just used to determine section sizes and addresses

	layout["got"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{layout["got"].offset, layout["got"].addr, ds.got.Len()}

	// Build dynamic section with addresses
	addrs := make(map[string]uint64)
	addrs["hash"] = layout["hash"].addr
	addrs["dynstr"] = layout["dynstr"].addr
	addrs["dynsym"] = layout["dynsym"].addr
	addrs["rela"] = layout["rela"].addr
	addrs["got"] = layout["got"].addr

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Hash layout: offset=0x%x, size=%d\n", layout["hash"].offset, layout["hash"].size)
	}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Rela layout: offset=0x%x, addr=0x%x, size=%d\n",
			layout["rela"].offset, layout["rela"].addr, layout["rela"].size)
	}
	ds.buildDynamicSection(addrs)

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "\n=== Dynamic Section Debug ===\n")
	}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Dynamic section size: %d bytes\n", ds.dynamic.Len())
	}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Needed libraries: %v\n", ds.needed)
	}

	layout["dynamic"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{layout["dynamic"].offset, layout["dynamic"].addr, ds.dynamic.Len()}

	// Recalculate GOT and BSS offsets now that dynamic size is known
	gotOffset := layout["dynamic"].offset + uint64((ds.dynamic.Len()+7) & ^7)
	gotAddr := layout["dynamic"].addr + uint64((ds.dynamic.Len()+7) & ^7)
	layout["got"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{gotOffset, gotAddr, ds.got.Len()}

	// Regenerate PLT with correct GOT address
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Regenerating PLT with correct GOT address 0x%x\n", gotAddr)
	}
	ds.GeneratePLT(functions, gotAddr, pltBase)
	ds.GenerateGOT(functions, dynamicAddr, pltBase)

	// Update DT_PLTGOT in the dynamic section with the correct GOT address
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Updating DT_PLTGOT from 0x%x to 0x%x\n", gotBase, gotAddr)
	}
	ds.updatePLTGOT(gotAddr)

	// Also update the relocations with the correct GOT addresses
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Updating relocations with final GOT base 0x%x\n", gotAddr)
	}
	for i := range functions {
		oldGotEntryAddr := gotBase + uint64(24+i*8)
		newGotEntryAddr := gotAddr + uint64(24+i*8)
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "  %s: reloction 0x%x -> 0x%x\n", functions[i], oldGotEntryAddr, newGotEntryAddr)
		}
		ds.updateRelocationAddress(oldGotEntryAddr, newGotEntryAddr)
	}

	rodataOffset := gotOffset + uint64((ds.got.Len()+7) & ^7)
	rodataAddr = gotAddr + uint64((ds.got.Len()+7) & ^7)
	layout["rodata"] = struct {
		offset uint64
		addr   uint64
		size   int
	}{rodataOffset, rodataAddr, layout["rodata"].size}

	// Entry point is already set to _start above
	// (entryPoint := layout["_start"].addr)

	// Write ELF file
	w := eb.ELFWriter()

	// ELF Header
	w.Write(0x7f)
	w.Write(0x45) // E
	w.Write(0x4c) // L
	w.Write(0x46) // F
	w.Write(2)    // 64-bit
	w.Write(1)    // little endian
	w.Write(1)    // ELF version
	w.Write(3)    // Linux
	w.WriteN(0, 8)
	w.Write2(3) // DYN
	w.Write2(byte(eb.arch.ELFMachineType()))
	w.Write4(1)

	w.Write8u(entryPoint)
	w.Write8u(elfHeader)
	w.Write8u(0) // no section headers
	w.Write4(0)
	w.Write2(byte(elfHeader))
	w.Write2(byte(progHeaderSize))
	w.Write2(byte(numProgHeaders))
	w.Write2(0)
	w.Write2(0)
	w.Write2(0)

	// Program Headers

	// PT_PHDR (must be first, but covered by a LOAD)
	w.Write4(6) // PT_PHDR
	w.Write4(4) // PF_R
	w.Write8u(elfHeader)
	w.Write8u(baseAddr + elfHeader)
	w.Write8u(baseAddr + elfHeader)
	w.Write8u(uint64(progHeaderSize * numProgHeaders))
	w.Write8u(uint64(progHeaderSize * numProgHeaders))
	w.Write8u(8)

	// PT_INTERP
	interpLayout := layout["interp"]
	w.Write4(3) // PT_INTERP
	w.Write4(4) // PF_R
	w.Write8u(interpLayout.offset)
	w.Write8u(interpLayout.addr)
	w.Write8u(interpLayout.addr)
	w.Write8u(uint64(interpLayout.size))
	w.Write8u(uint64(interpLayout.size))
	w.Write8u(1)

	// PT_LOAD #0 (read-only: ELF header, program headers, + all read-only data)
	// This covers the PHDR segment
	roStart := uint64(0) // Start at beginning of file
	roEnd := layout["rela"].offset + uint64(layout["rela"].size)
	roSize := roEnd - roStart
	w.Write4(1) // PT_LOAD
	w.Write4(4) // PF_R
	w.Write8u(roStart)
	w.Write8u(baseAddr + roStart)
	w.Write8u(baseAddr + roStart)
	w.Write8u(roSize)
	w.Write8u(roSize)
	w.Write8u(pageSize)

	// PT_LOAD #1 (executable: plt, text)
	exStart := layout["plt"].offset
	exEnd := layout["text"].offset + uint64(layout["text"].size)
	exSize := exEnd - exStart
	w.Write4(1) // PT_LOAD
	w.Write4(5) // PF_R | PF_X
	w.Write8u(exStart)
	w.Write8u(baseAddr + exStart)
	w.Write8u(baseAddr + exStart)
	w.Write8u(exSize)
	w.Write8u(exSize)
	w.Write8u(pageSize)

	// PT_LOAD #2 (writable: dynamic, got, rodata)
	rwStart := layout["dynamic"].offset
	rwFileSize := layout["rodata"].offset + uint64(layout["rodata"].size) - rwStart
	rwMemSize := rwFileSize

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "\n=== Writable Segment Debug ===\n")
	}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "rwStart (dynamic.offset): 0x%x\n", rwStart)
	}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "GOT offset: 0x%x, size: %d\n", layout["got"].offset, layout["got"].size)
	}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Rodata offset: 0x%x, size: %d\n", layout["rodata"].offset, layout["rodata"].size)
	}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Calculated rwFileSize: 0x%x\n", rwFileSize)
	}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Calculated rwMemSize: 0x%x\n", rwMemSize)
	}

	w.Write4(1) // PT_LOAD
	w.Write4(6) // PF_R | PF_W
	w.Write8u(rwStart)
	w.Write8u(baseAddr + rwStart)
	w.Write8u(baseAddr + rwStart)
	w.Write8u(rwFileSize)
	w.Write8u(rwMemSize)
	w.Write8u(pageSize)

	// PT_DYNAMIC
	w.Write4(2) // PT_DYNAMIC
	w.Write4(6) // PF_R | PF_W
	w.Write8u(layout["dynamic"].offset)
	w.Write8u(layout["dynamic"].addr)
	w.Write8u(layout["dynamic"].addr)
	w.Write8u(uint64(layout["dynamic"].size))
	w.Write8u(uint64(layout["dynamic"].size))
	w.Write8u(8)

	// Pad to aligned header size
	for i := headersSize; i < alignedHeaders; i++ {
		w.Write(0)
	}

	// Write all sections
	writePadded := func(buf *bytes.Buffer, targetSize int) {
		currentPos := eb.elf.Len()
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Writing buffer at offset 0x%x (%d bytes from buffer, padding to %d)\n",
				currentPos, buf.Len(), targetSize)
		}
		w.WriteBytes(buf.Bytes())
		for i := buf.Len(); i < targetSize; i++ {
			w.Write(0)
		}
	}

	// Interpreter
	for i := 0; i < len(interp); i++ {
		w.Write(byte(interp[i]))
	}
	w.Write(0)
	for i := interpSize; i < (interpSize+7)&^7; i++ {
		w.Write(0)
	}

	writePadded(&ds.dynsym, (ds.dynsym.Len()+7)&^7)
	writePadded(&ds.dynstr, (ds.dynstr.Len()+7)&^7)
	writePadded(&ds.hash, (ds.hash.Len()+7)&^7)
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Rela buffer contents (%d bytes): %x\n", ds.rela.Len(), ds.rela.Bytes())
	}
	writePadded(&ds.rela, (ds.rela.Len()+7)&^7)

	// Pad to next page
	currentPos := int(roEnd)
	nextPage := (currentPos + int(pageSize) - 1) & ^(int(pageSize) - 1)
	for i := currentPos; i < nextPage; i++ {
		w.Write(0)
	}

	// PLT and text
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "About to write PLT: expected offset=0x%x, actual buffer position=0x%x, PLT size=%d bytes\n",
			layout["plt"].offset, eb.elf.Len(), ds.plt.Len())
	}
	w.WriteBytes(ds.plt.Bytes())
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "After PLT, about to write _start\n")
	}

	// _start function (minimal entry point that clears registers and jumps to user code)
	// xor rax, rax   ; clear rax
	w.Write(0x48)
	w.Write(0x31)
	w.Write(0xc0)
	// xor rdi, rdi   ; clear rdi (first argument)
	w.Write(0x48)
	w.Write(0x31)
	w.Write(0xff)
	// xor rsi, rsi   ; clear rsi (second argument)
	w.Write(0x48)
	w.Write(0x31)
	w.Write(0xf6)
	// jmp to user code (relative jump)
	w.Write(0xe9) // jmp rel32
	startAddr := layout["_start"].addr
	textAddrForJump := layout["text"].addr
	jumpOffset := int32(textAddrForJump - (startAddr + 14)) // 14 = size of _start code before jmp
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "_start jump: startAddr=0x%x, textAddr=0x%x, jumpOffset=0x%x (%d)\n",
			startAddr, textAddrForJump, uint32(jumpOffset), jumpOffset)
	}
	binary.Write(w.(*BufferWrapper).buf, binary.LittleEndian, jumpOffset)

	// Pad _start to aligned size
	startActualSize := 14 // 9 bytes of xor instructions + 5 bytes jmp
	for i := startActualSize; i < ((startSize + 7) & ^7); i++ {
		w.Write(0)
	}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Finished writing _start (%d bytes padded to %d), about to write text\n", startActualSize, ((startSize + 7) & ^7))
	}

	// Patch PC-relative relocations before writing text section
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "\n=== Patching PC-relative relocations ===\n")
	}
	eb.PatchPCRelocations(layout["text"].addr, layout["rodata"].addr, rodataSize)

	// Patch direct function calls
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "\n=== Patching function calls ===\n")
	}
	eb.PatchCallSites(layout["text"].addr)

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "About to write text: expected offset=0x%x, actual buffer position=0x%x, text size=%d bytes\n",
			layout["text"].offset, eb.elf.Len(), eb.text.Len())
	}
	w.WriteBytes(eb.text.Bytes())
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Finished writing text section\n")
	}
	for i := codeSize; i < (codeSize+7)&^7; i++ {
		w.Write(0)
	}

	// Pad to dynamic section offset
	currentPos = eb.elf.Len()
	targetOffset := int(layout["dynamic"].offset)
	for i := currentPos; i < targetOffset; i++ {
		w.Write(0)
	}

	// Dynamic, GOT
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Writing dynamic section (%d bytes) at file position ~0x%x (expected 0x%x)\n", ds.dynamic.Len(), eb.elf.Len(), layout["dynamic"].offset)
	}
	writePadded(&ds.dynamic, (ds.dynamic.Len()+7)&^7)
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Writing GOT section (%d bytes) at file position ~0x%x\n", ds.got.Len(), eb.elf.Len())
	}
	writePadded(&ds.got, (ds.got.Len()+7)&^7)

	w.WriteBytes(eb.rodata.Bytes())

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG [after writing rodata to ELF]: rodata buffer size: %d bytes\n", eb.rodata.Len())
	}
	if eb.rodata.Len() > 0 {
		previewLen := 32
		if eb.rodata.Len() < previewLen {
			previewLen = eb.rodata.Len()
		}
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG [after writing rodata to ELF]: rodata buffer first %d bytes: %q\n", previewLen, eb.rodata.Bytes()[:previewLen])
		}
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "\n=== Complete Dynamic ELF ===\n")
	}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Entry point: 0x%x\n", entryPoint)
	}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "PLT base: 0x%x (%d bytes)\n", pltBase, ds.plt.Len())
	}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "GOT base: 0x%x (%d bytes)\n", gotAddr, ds.got.Len())
	}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Rodata base: 0x%x (%d bytes)\n", layout["rodata"].addr, rodataSize)
	}
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Functions: %v\n", functions)
	}

	return gotAddr, rodataAddr, textAddr, pltBase, nil
}

func (eb *ExecutableBuilder) getInterpreterPath() string {
	switch eb.platform.Arch {
	case ArchX86_64:
		return "/lib64/ld-linux-x86-64.so.2"
	case ArchARM64:
		return "/lib/ld-linux-aarch64.so.1"
	case ArchRiscv64:
		return "/lib/ld-linux-riscv64-lp64d.so.1"
	default:
		return "/lib64/ld-linux-x86-64.so.2"
	}
}

// patchPLTCalls patches call instructions in .text to use correct PLT offsets
func (eb *ExecutableBuilder) patchPLTCalls(ds *DynamicSections, textAddr uint64, pltBase uint64, functions []string) {
	textBytes := eb.text.Bytes()

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Text bytes (%d total): %x\n", len(textBytes), textBytes)
	}

	switch eb.platform.Arch {
	case ArchX86_64:
		eb.patchX86PLTCalls(textBytes, ds, textAddr, pltBase, functions)
	case ArchARM64:
		eb.patchARM64PLTCalls(textBytes, ds, textAddr, pltBase, functions)
	case ArchRiscv64:
		eb.patchRISCVPLTCalls(textBytes, ds, textAddr, pltBase, functions)
	}

	// Write the patched bytes back
	eb.text.Reset()
	eb.text.Write(textBytes)
}

func (eb *ExecutableBuilder) patchX86PLTCalls(textBytes []byte, ds *DynamicSections, textAddr, pltBase uint64, functions []string) {
	// Search for placeholder call instructions (0xE8 followed by 0x78563412)
	placeholder := []byte{0x78, 0x56, 0x34, 0x12}

	funcIndex := 0
	for i := 0; i < len(textBytes); i++ {
		if i > 0 && i+3 < len(textBytes) && textBytes[i-1] == 0xE8 {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Found 0xE8 at i-1=%d, checking placeholder at i=%d: %x\n", i-1, i, textBytes[i:i+4])
			}
			if bytes.Equal(textBytes[i:i+4], placeholder) {
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "  -> Placeholder matches!\n")
				}
				if funcIndex < len(functions) {
					pltOffset := ds.GetPLTOffset(functions[funcIndex])
					if pltOffset >= 0 {
						targetAddr := pltBase + uint64(pltOffset)
						currentAddr := textAddr + uint64(i)
						relOffset := int32(targetAddr - (currentAddr + 4))

						if VerboseMode {
							fmt.Fprintf(os.Stderr, "Patching x86-64 call #%d (%s): i=%d, currentAddr=0x%x, targetAddr=0x%x, relOffset=%d (0x%x)\n",
								funcIndex, functions[funcIndex], i, currentAddr, targetAddr, relOffset, uint32(relOffset))
						}

						textBytes[i] = byte(relOffset & 0xFF)
						textBytes[i+1] = byte((relOffset >> 8) & 0xFF)
						textBytes[i+2] = byte((relOffset >> 16) & 0xFF)
						textBytes[i+3] = byte((relOffset >> 24) & 0xFF)
					}
					funcIndex++
				}
			}
		}
	}
}

func (eb *ExecutableBuilder) patchARM64PLTCalls(textBytes []byte, ds *DynamicSections, textAddr, pltBase uint64, functions []string) {
	// Search for placeholder BL instructions (0x94000000)
	funcIndex := 0
	for i := 0; i+3 < len(textBytes); i += 4 {
		instr := uint32(textBytes[i]) |
			(uint32(textBytes[i+1]) << 8) |
			(uint32(textBytes[i+2]) << 16) |
			(uint32(textBytes[i+3]) << 24)

		// BL instruction: 100101 imm26
		if (instr&0xFC000000) == 0x94000000 && (instr&0x03FFFFFF) == 0 {
			if funcIndex < len(functions) {
				pltOffset := ds.GetPLTOffset(functions[funcIndex])
				if pltOffset >= 0 {
					targetAddr := pltBase + uint64(pltOffset)
					currentAddr := textAddr + uint64(i)
					offset := int64(targetAddr - currentAddr)

					// BL uses signed 26-bit word offset (multiply by 4)
					wordOffset := offset >> 2
					if wordOffset >= -0x2000000 && wordOffset < 0x2000000 {
						imm26 := uint32(wordOffset) & 0x03FFFFFF
						blInstr := 0x94000000 | imm26

						if VerboseMode {
							fmt.Fprintf(os.Stderr, "Patching ARM64 call #%d (%s): offset=0x%x, currentAddr=0x%x, targetAddr=0x%x, wordOffset=%d\n",
								funcIndex, functions[funcIndex], i, currentAddr, targetAddr, wordOffset)
						}

						textBytes[i] = byte(blInstr & 0xFF)
						textBytes[i+1] = byte((blInstr >> 8) & 0xFF)
						textBytes[i+2] = byte((blInstr >> 16) & 0xFF)
						textBytes[i+3] = byte((blInstr >> 24) & 0xFF)
					}
				}
				funcIndex++
			}
		}
	}
}

func (eb *ExecutableBuilder) patchRISCVPLTCalls(textBytes []byte, ds *DynamicSections, textAddr, pltBase uint64, functions []string) {
	// Search for placeholder JAL instructions (0x000000EF)
	funcIndex := 0
	for i := 0; i+3 < len(textBytes); i += 4 {
		instr := uint32(textBytes[i]) |
			(uint32(textBytes[i+1]) << 8) |
			(uint32(textBytes[i+2]) << 16) |
			(uint32(textBytes[i+3]) << 24)

		// JAL instruction: imm[20|10:1|11|19:12] rd 1101111
		if (instr&0x7F) == 0x6F && (instr&0xFFFFF000) == 0 {
			if funcIndex < len(functions) {
				pltOffset := ds.GetPLTOffset(functions[funcIndex])
				if pltOffset >= 0 {
					targetAddr := pltBase + uint64(pltOffset)
					currentAddr := textAddr + uint64(i)
					offset := int64(targetAddr - currentAddr)

					// JAL uses signed 21-bit offset
					if offset >= -0x100000 && offset < 0x100000 {
						// Encode immediate in JAL format: [20|10:1|11|19:12]
						imm20 := (uint32(offset>>20) & 1) << 31
						imm10_1 := (uint32(offset>>1) & 0x3FF) << 21
						imm11 := (uint32(offset>>11) & 1) << 20
						imm19_12 := (uint32(offset>>12) & 0xFF) << 12
						rd := (instr >> 7) & 0x1F

						jalInstr := imm20 | imm19_12 | imm11 | imm10_1 | (rd << 7) | 0x6F

						if VerboseMode {
							fmt.Fprintf(os.Stderr, "Patching RISC-V call #%d (%s): offset=0x%x, currentAddr=0x%x, targetAddr=0x%x, pcOffset=%d\n",
								funcIndex, functions[funcIndex], i, currentAddr, targetAddr, offset)
						}

						textBytes[i] = byte(jalInstr & 0xFF)
						textBytes[i+1] = byte((jalInstr >> 8) & 0xFF)
						textBytes[i+2] = byte((jalInstr >> 16) & 0xFF)
						textBytes[i+3] = byte((jalInstr >> 24) & 0xFF)
					}
				}
				funcIndex++
			}
		}
	}
}
