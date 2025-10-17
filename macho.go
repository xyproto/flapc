package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

// Mach-O constants
const (
	MH_MAGIC_64            = 0xfeedfacf // 64-bit magic number
	MH_CIGAM_64            = 0xcffaedfe // NXSwapInt(MH_MAGIC_64)
	CPU_TYPE_X86_64        = 0x01000007 // x86_64
	CPU_TYPE_ARM64         = 0x0100000c // ARM64
	CPU_SUBTYPE_X86_64_ALL = 0x00000003
	CPU_SUBTYPE_ARM64_ALL  = 0x00000000

	// File types
	MH_EXECUTE = 0x2 // Executable file

	// Flags
	MH_NOUNDEFS = 0x1
	MH_DYLDLINK = 0x4
	MH_PIE      = 0x200000
	MH_TWOLEVEL = 0x80

	// Load commands
	LC_SEGMENT_64          = 0x19
	LC_SYMTAB              = 0x2
	LC_DYSYMTAB            = 0xb
	LC_LOAD_DYLINKER       = 0xe
	LC_UUID                = 0x1b
	LC_VERSION_MIN_MACOSX  = 0x24
	LC_SOURCE_VERSION      = 0x2A
	LC_MAIN                = 0x80000028
	LC_LOAD_DYLIB          = 0xc
	LC_FUNCTION_STARTS     = 0x26
	LC_DATA_IN_CODE        = 0x29
	LC_DYLD_CHAINED_FIXUPS = 0x80000034
	LC_DYLD_EXPORTS_TRIE   = 0x80000033

	// Protection flags
	VM_PROT_NONE    = 0x00
	VM_PROT_READ    = 0x01
	VM_PROT_WRITE   = 0x02
	VM_PROT_EXECUTE = 0x04

	// Section types
	S_REGULAR          = 0x0
	S_ZEROFILL         = 0x1
	S_CSTRING_LITERALS = 0x2
	S_4BYTE_LITERALS   = 0x3
	S_8BYTE_LITERALS   = 0x4
	S_LITERAL_POINTERS = 0x5

	// Section attributes
	S_ATTR_PURE_INSTRUCTIONS   = 0x80000000
	S_ATTR_SOME_INSTRUCTIONS   = 0x00000400
	S_SYMBOL_STUBS             = 0x8
	S_LAZY_SYMBOL_POINTERS     = 0x7
	S_NON_LAZY_SYMBOL_POINTERS = 0x6
)

// MachOHeader64 represents the Mach-O 64-bit header
type MachOHeader64 struct {
	Magic      uint32
	CPUType    uint32
	CPUSubtype uint32
	FileType   uint32
	NCmds      uint32
	SizeOfCmds uint32
	Flags      uint32
	Reserved   uint32
}

// LoadCommand represents a generic Mach-O load command
type LoadCommand struct {
	Cmd     uint32
	CmdSize uint32
}

// SegmentCommand64 represents a 64-bit segment load command
type SegmentCommand64 struct {
	Cmd      uint32
	CmdSize  uint32
	SegName  [16]byte
	VMAddr   uint64
	VMSize   uint64
	FileOff  uint64
	FileSize uint64
	MaxProt  uint32
	InitProt uint32
	NSects   uint32
	Flags    uint32
}

// Section64 represents a 64-bit section within a segment
type Section64 struct {
	SectName  [16]byte
	SegName   [16]byte
	Addr      uint64
	Size      uint64
	Offset    uint32
	Align     uint32
	Reloff    uint32
	Nreloc    uint32
	Flags     uint32
	Reserved1 uint32
	Reserved2 uint32
	Reserved3 uint32
}

// SymtabCommand represents the symbol table load command
type SymtabCommand struct {
	Cmd     uint32
	CmdSize uint32
	Symoff  uint32
	Nsyms   uint32
	Stroff  uint32
	Strsize uint32
}

// DysymtabCommand represents the dynamic symbol table load command
type DysymtabCommand struct {
	Cmd            uint32
	CmdSize        uint32
	ILocalSym      uint32
	NLocalSym      uint32
	IExtDefSym     uint32
	NExtDefSym     uint32
	IUndefSym      uint32
	NUndefSym      uint32
	TOCOff         uint32
	NTOC           uint32
	ModTabOff      uint32
	NModTab        uint32
	ExtRefSymOff   uint32
	NExtRefSyms    uint32
	IndirectSymOff uint32
	NIndirectSyms  uint32
	ExtRelOff      uint32
	NExtRel        uint32
	LocRelOff      uint32
	NLocRel        uint32
}

// EntryPointCommand represents the LC_MAIN load command (entry point)
type EntryPointCommand struct {
	Cmd       uint32
	CmdSize   uint32
	EntryOff  uint64
	StackSize uint64
}

// DylinkerCommand represents the dynamic linker load command
type DylinkerCommand struct {
	Cmd     uint32
	CmdSize uint32
	NameOff uint32
	// Name follows
}

// DylibCommand represents a dynamic library load command
type DylibCommand struct {
	Cmd                  uint32
	CmdSize              uint32
	NameOff              uint32
	Timestamp            uint32
	CurrentVersion       uint32
	CompatibilityVersion uint32
	// Name follows
}

// UUIDCommand represents the UUID load command
type UUIDCommand struct {
	Cmd     uint32
	CmdSize uint32
	UUID    [16]byte
}

// VersionMinCommand represents version minimum load command
type VersionMinCommand struct {
	Cmd     uint32
	CmdSize uint32
	Version uint32
	SDK     uint32
}

// SourceVersionCommand represents source version load command
type SourceVersionCommand struct {
	Cmd     uint32
	CmdSize uint32
	Version uint64
}

// LinkEditDataCommand represents function starts / data in code load command
type LinkEditDataCommand struct {
	Cmd      uint32
	CmdSize  uint32
	DataOff  uint32
	DataSize uint32
}

// Nlist64 represents a 64-bit symbol table entry
type Nlist64 struct {
	N_strx  uint32 // String table index
	N_type  uint8  // Symbol type
	N_sect  uint8  // Section number
	N_desc  uint16 // Description
	N_value uint64 // Symbol value
}

// Symbol type flags
const (
	N_UNDF = 0x0  // Undefined symbol
	N_EXT  = 0x1  // External symbol
	N_TYPE = 0x0e // Type mask
	N_SECT = 0xe  // Defined in section
)

// Chained fixups structures
type DyldChainedFixupsHeader struct {
	FixupsVersion uint32 // 0
	StartsOffset  uint32 // Offset of dyld_chained_starts_in_image
	ImportsOffset uint32 // Offset of imports table
	SymbolsOffset uint32 // Offset of symbol strings
	ImportsCount  uint32 // Number of imported symbols
	ImportsFormat uint32 // DYLD_CHAINED_IMPORT or DYLD_CHAINED_IMPORT_ADDEND (usually 1)
	SymbolsFormat uint32 // 0 for uncompressed
}

type DyldChainedStartsInImage struct {
	SegCount uint32 // Number of segments
	// Followed by seg_count uint32 offsets to dyld_chained_starts_in_segment
}

type DyldChainedStartsInSegment struct {
	Size            uint32 // Size of this structure
	PageSize        uint16 // Page size (0x4000 for 16KB, 0x1000 for 4KB)
	PointerFormat   uint16 // DYLD_CHAINED_PTR_ARM64E, etc. (1 for ARM64E, 3 for ARM64E_KERNEL)
	SegmentOffset   uint64 // Offset in __LINKEDIT to start of segment
	MaxValidPointer uint32 // For PtrAuth
	PageCount       uint16 // Number of pages
	// Followed by page_count uint16 page_start values (0xFFFF means no fixups)
}

type DyldChainedImport struct {
	LibOrdinal uint8  // Library ordinal (1-based, 0 = self, 0xFE = weak, 0xFF = main executable)
	WeakImport uint8  // 0 or 1
	NameOffset uint32 // Offset into symbol strings (24-bit value in bits 0-23)
}

// Chained fixups constants
const (
	DYLD_CHAINED_PTR_ARM64E            = 1
	DYLD_CHAINED_PTR_64                = 2
	DYLD_CHAINED_PTR_ARM64E_KERNEL     = 7
	DYLD_CHAINED_PTR_64_KERNEL_CACHE   = 8
	DYLD_CHAINED_PTR_ARM64E_USERLAND24 = 12

	DYLD_CHAINED_IMPORT          = 1
	DYLD_CHAINED_IMPORT_ADDEND   = 2
	DYLD_CHAINED_IMPORT_ADDEND64 = 3
)

// WriteMachO writes a Mach-O executable for macOS
func (eb *ExecutableBuilder) WriteMachO() error {

	debug := os.Getenv("FLAP_DEBUG") != ""

	var buf bytes.Buffer

	// Determine CPU type
	var cpuType, cpuSubtype uint32
	switch eb.platform.Arch {
	case ArchX86_64:
		cpuType = CPU_TYPE_X86_64
		cpuSubtype = CPU_SUBTYPE_X86_64_ALL
	case ArchARM64:
		cpuType = CPU_TYPE_ARM64
		cpuSubtype = CPU_SUBTYPE_ARM64_ALL
	default:
		return fmt.Errorf("unsupported architecture for Mach-O: %s", eb.platform)
	}

	// Page size
	pageSize := uint64(0x4000) // 16KB for ARM64, but 4KB works for x86_64 too

	// macOS uses a large zero page (4GB) for security
	zeroPageSize := uint64(0x100000000) // 4GB zero page

	// Calculate sizes
	textSize := uint64(eb.text.Len())
	rodataSize := uint64(eb.rodata.Len())

	// Calculate dynamic linking section sizes
	numImports := uint32(len(eb.neededFunctions))
	stubsSize := uint64(0)
	gotSize := uint64(0)
	if eb.useDynamicLinking && numImports > 0 {
		stubsSize = uint64(numImports * 12) // 12 bytes per stub on ARM64
		gotSize = uint64(numImports * 8)    // 8 bytes per GOT entry
	}

	// Align sizes
	textSizeAligned := (textSize + pageSize - 1) &^ (pageSize - 1)
	rodataSizeAligned := (rodataSize + pageSize - 1) &^ (pageSize - 1)
	stubsSizeAligned := (stubsSize + 15) &^ 15 // 16-byte align
	_ = stubsSizeAligned                       // May be used for stub alignment
	_ = (gotSize + 15) &^ 15                   // May be used for GOT alignment

	// Calculate addresses - __TEXT starts after zero page
	textAddr := zeroPageSize         // __TEXT segment starts at 4GB
	textSectAddr := textAddr         // __text section
	stubsAddr := textAddr + textSize // __stubs right after __text

	rodataAddr := textAddr + textSizeAligned
	rodataSectAddr := rodataAddr       // __data section (rodata)
	gotAddr := rodataAddr + rodataSize // __got right after __data

	// Build load commands in a temporary buffer
	var loadCmdsBuf bytes.Buffer
	ncmds := uint32(0)

	// Calculate preliminary load commands size for offset calculations
	headerSize := uint32(binary.Size(MachOHeader64{}))
	prelimLoadCmdsSize := uint32(0)
	prelimLoadCmdsSize += uint32(binary.Size(SegmentCommand64{})) // __PAGEZERO

	// __TEXT segment with sections
	textNSects := uint32(1) // __text
	if eb.useDynamicLinking && numImports > 0 {
		textNSects++ // __stubs
	}
	prelimLoadCmdsSize += uint32(binary.Size(SegmentCommand64{}) + int(textNSects)*binary.Size(Section64{}))

	// __DATA segment with sections (if needed)
	if rodataSize > 0 || (eb.useDynamicLinking && numImports > 0) {
		dataNSects := uint32(0)
		if rodataSize > 0 {
			dataNSects++
		}
		if eb.useDynamicLinking && numImports > 0 {
			dataNSects++ // __got
		}
		prelimLoadCmdsSize += uint32(binary.Size(SegmentCommand64{}) + int(dataNSects)*binary.Size(Section64{}))
	}

	// __LINKEDIT segment (if dynamic linking)
	if eb.useDynamicLinking && numImports > 0 {
		prelimLoadCmdsSize += uint32(binary.Size(SegmentCommand64{}))
	}

	dylinkerPath := "/usr/lib/dyld\x00"
	dylinkerCmdSize := (uint32(binary.Size(LoadCommand{})+4+len(dylinkerPath)) + 7) &^ 7
	prelimLoadCmdsSize += dylinkerCmdSize                          // LC_LOAD_DYLINKER
	prelimLoadCmdsSize += uint32(binary.Size(EntryPointCommand{})) // LC_MAIN

	if eb.useDynamicLinking {
		dylibPath := "/usr/lib/libSystem.B.dylib\x00"
		dylibCmdSize := (uint32(binary.Size(LoadCommand{})+16+len(dylibPath)) + 7) &^ 7
		prelimLoadCmdsSize += dylibCmdSize // LC_LOAD_DYLIB

		if numImports > 0 {
			prelimLoadCmdsSize += uint32(binary.Size(SymtabCommand{}))       // LC_SYMTAB
			prelimLoadCmdsSize += uint32(binary.Size(DysymtabCommand{}))     // LC_DYSYMTAB
			prelimLoadCmdsSize += uint32(binary.Size(LinkEditDataCommand{})) // LC_DYLD_CHAINED_FIXUPS
		}
	}

	fileHeaderSize := headerSize + prelimLoadCmdsSize
	// Align to page boundary for first segment
	textFileOffset := uint64((fileHeaderSize + uint32(pageSize) - 1) &^ (uint32(pageSize) - 1))
	stubsFileOffset := textFileOffset + textSize
	rodataFileOffset := textFileOffset + textSizeAligned
	gotFileOffset := rodataFileOffset + rodataSize

	// Calculate LINKEDIT segment offset and size
	linkeditFileOffset := rodataFileOffset + rodataSizeAligned

	// Build symbol table and string table
	var symtab []Nlist64
	var strtab bytes.Buffer
	strtab.WriteByte(0) // First byte must be null

	if eb.useDynamicLinking && numImports > 0 {
		for _, funcName := range eb.neededFunctions {
			strOffset := uint32(strtab.Len())
			strtab.WriteString(funcName)
			strtab.WriteByte(0)

			sym := Nlist64{
				N_strx:  strOffset,
				N_type:  N_UNDF | N_EXT, // Undefined external symbol
				N_sect:  0,
				N_desc:  0,
				N_value: 0,
			}
			symtab = append(symtab, sym)
		}
	}

	// Build indirect symbol table (maps GOT/stub entries to symbol indices)
	var indirectSymTab []uint32
	if eb.useDynamicLinking && numImports > 0 {
		for i := uint32(0); i < numImports; i++ {
			indirectSymTab = append(indirectSymTab, i) // GOT entries
		}
		for i := uint32(0); i < numImports; i++ {
			indirectSymTab = append(indirectSymTab, i) // Stub entries
		}
	}

	symtabSize := uint32(len(symtab) * binary.Size(Nlist64{}))
	strtabSize := uint32(strtab.Len())
	indirectSymTabSize := uint32(len(indirectSymTab) * 4)

	// Calculate chained fixups size
	var chainedFixupsSize uint32
	if eb.useDynamicLinking && numImports > 0 {
		// DyldChainedFixupsHeader
		chainedFixupsSize += uint32(binary.Size(DyldChainedFixupsHeader{}))
		// DyldChainedStartsInImage + segment offsets (2 segments: __DATA, __LINKEDIT)
		chainedFixupsSize += 4 + (2 * 4) // seg_count + 2 offsets
		// DyldChainedStartsInSegment for __DATA + page starts (1 page)
		chainedFixupsSize += uint32(binary.Size(DyldChainedStartsInSegment{})) + 2 // + 1 page_start
		// Imports table (DyldChainedImport * numImports)
		chainedFixupsSize += 6 * numImports // Each import is 6 bytes
		// Symbol strings (copy from strtab)
		chainedFixupsSize += strtabSize
		// Align to 8 bytes
		chainedFixupsSize = (chainedFixupsSize + 7) &^ 7
	}

	linkeditSize := symtabSize + strtabSize + indirectSymTabSize + chainedFixupsSize

	// 1. LC_SEGMENT_64 for __PAGEZERO (required on macOS)
	{
		seg := SegmentCommand64{
			Cmd:      LC_SEGMENT_64,
			CmdSize:  uint32(binary.Size(SegmentCommand64{})),
			VMAddr:   0,
			VMSize:   zeroPageSize, // 4GB zero page
			FileOff:  0,
			FileSize: 0,
			MaxProt:  VM_PROT_NONE,
			InitProt: VM_PROT_NONE,
			NSects:   0,
			Flags:    0,
		}
		copy(seg.SegName[:], "__PAGEZERO")
		binary.Write(&loadCmdsBuf, binary.LittleEndian, &seg)
		ncmds++
	}

	// 2. LC_SEGMENT_64 for __TEXT with __text and __stubs sections
	{
		// FileSize must not exceed VMSize
		// __TEXT segment maps from file offset 0 (includes headers) to end of stubs
		textSegFileSize := stubsFileOffset + stubsSize
		if textSegFileSize > textSizeAligned {
			textSegFileSize = textSizeAligned // Cap at vmsize
		}

		seg := SegmentCommand64{
			Cmd:      LC_SEGMENT_64,
			CmdSize:  uint32(binary.Size(SegmentCommand64{}) + int(textNSects)*binary.Size(Section64{})),
			VMAddr:   textAddr,
			VMSize:   textSizeAligned,
			FileOff:  0, // __TEXT starts at beginning of file
			FileSize: textSegFileSize,
			MaxProt:  VM_PROT_READ | VM_PROT_EXECUTE,
			InitProt: VM_PROT_READ | VM_PROT_EXECUTE,
			NSects:   textNSects,
			Flags:    0,
		}
		copy(seg.SegName[:], "__TEXT")
		binary.Write(&loadCmdsBuf, binary.LittleEndian, &seg)

		// __text section
		sect := Section64{
			Addr:      textSectAddr,
			Size:      textSize,
			Offset:    uint32(textFileOffset),
			Align:     4,
			Reloff:    0,
			Nreloc:    0,
			Flags:     S_REGULAR | S_ATTR_PURE_INSTRUCTIONS | S_ATTR_SOME_INSTRUCTIONS,
			Reserved1: 0,
			Reserved2: 0,
			Reserved3: 0,
		}
		copy(sect.SectName[:], "__text")
		copy(sect.SegName[:], "__TEXT")
		binary.Write(&loadCmdsBuf, binary.LittleEndian, &sect)

		// __stubs section (if dynamic linking)
		if eb.useDynamicLinking && numImports > 0 {
			stubsSect := Section64{
				Addr:      stubsAddr,
				Size:      stubsSize,
				Offset:    uint32(stubsFileOffset),
				Align:     2, // 2^2 = 4 byte alignment
				Reloff:    0,
				Nreloc:    0,
				Flags:     S_SYMBOL_STUBS | S_ATTR_PURE_INSTRUCTIONS | S_ATTR_SOME_INSTRUCTIONS,
				Reserved1: numImports, // Indirect symbol table index (stubs start after GOT entries)
				Reserved2: 12,         // Stub size (12 bytes per stub)
				Reserved3: 0,
			}
			copy(stubsSect.SectName[:], "__stubs")
			copy(stubsSect.SegName[:], "__TEXT")
			binary.Write(&loadCmdsBuf, binary.LittleEndian, &stubsSect)
		}

		ncmds++
	}

	// 3. LC_SEGMENT_64 for __DATA with __data and __got sections
	if rodataSize > 0 || (eb.useDynamicLinking && numImports > 0) {
		dataNSects := uint32(0)
		if rodataSize > 0 {
			dataNSects++
		}
		if eb.useDynamicLinking && numImports > 0 {
			dataNSects++
		}

		dataSegSize := rodataSizeAligned
		dataFileSize := rodataSize
		if eb.useDynamicLinking && numImports > 0 {
			dataFileSize += gotSize
		}

		seg := SegmentCommand64{
			Cmd:      LC_SEGMENT_64,
			CmdSize:  uint32(binary.Size(SegmentCommand64{}) + int(dataNSects)*binary.Size(Section64{})),
			VMAddr:   rodataAddr,
			VMSize:   dataSegSize,
			FileOff:  rodataFileOffset,
			FileSize: dataFileSize,
			MaxProt:  VM_PROT_READ | VM_PROT_WRITE,
			InitProt: VM_PROT_READ | VM_PROT_WRITE,
			NSects:   dataNSects,
			Flags:    0,
		}
		copy(seg.SegName[:], "__DATA")
		binary.Write(&loadCmdsBuf, binary.LittleEndian, &seg)

		// __data section (rodata)
		if rodataSize > 0 {
			sect := Section64{
				Addr:      rodataSectAddr,
				Size:      rodataSize,
				Offset:    uint32(rodataFileOffset),
				Align:     3, // 2^3 = 8 byte alignment
				Reloff:    0,
				Nreloc:    0,
				Flags:     S_REGULAR,
				Reserved1: 0,
				Reserved2: 0,
				Reserved3: 0,
			}
			copy(sect.SectName[:], "__data")
			copy(sect.SegName[:], "__DATA")
			binary.Write(&loadCmdsBuf, binary.LittleEndian, &sect)
		}

		// __got section (Global Offset Table)
		if eb.useDynamicLinking && numImports > 0 {
			gotSect := Section64{
				Addr:      gotAddr,
				Size:      gotSize,
				Offset:    uint32(gotFileOffset),
				Align:     3, // 2^3 = 8 byte alignment
				Reloff:    0,
				Nreloc:    0,
				Flags:     S_NON_LAZY_SYMBOL_POINTERS,
				Reserved1: 0, // Indirect symbol table index (GOT entries start at 0)
				Reserved2: 0,
				Reserved3: 0,
			}
			copy(gotSect.SectName[:], "__got")
			copy(gotSect.SegName[:], "__DATA")
			binary.Write(&loadCmdsBuf, binary.LittleEndian, &gotSect)
		}

		ncmds++
	}

	// 4. LC_SEGMENT_64 for __LINKEDIT (if dynamic linking)
	if eb.useDynamicLinking && numImports > 0 {
		seg := SegmentCommand64{
			Cmd:      LC_SEGMENT_64,
			CmdSize:  uint32(binary.Size(SegmentCommand64{})),
			VMAddr:   ((linkeditFileOffset + pageSize - 1) &^ (pageSize - 1)) + zeroPageSize,
			VMSize:   uint64((linkeditSize + uint32(pageSize) - 1) &^ (uint32(pageSize) - 1)),
			FileOff:  linkeditFileOffset,
			FileSize: uint64(linkeditSize),
			MaxProt:  VM_PROT_READ,
			InitProt: VM_PROT_READ,
			NSects:   0,
			Flags:    0,
		}
		copy(seg.SegName[:], "__LINKEDIT")
		binary.Write(&loadCmdsBuf, binary.LittleEndian, &seg)
		ncmds++
	}

	// 5. LC_LOAD_DYLINKER
	{
		dylinkerPath := "/usr/lib/dyld\x00"
		cmdSize := uint32(binary.Size(LoadCommand{}) + 4 + len(dylinkerPath))
		cmdSize = (cmdSize + 7) &^ 7 // 8-byte align

		cmd := LoadCommand{
			Cmd:     LC_LOAD_DYLINKER,
			CmdSize: cmdSize,
		}
		binary.Write(&loadCmdsBuf, binary.LittleEndian, &cmd)
		binary.Write(&loadCmdsBuf, binary.LittleEndian, uint32(12)) // name offset
		loadCmdsBuf.WriteString(dylinkerPath)

		// Pad to alignment
		for loadCmdsBuf.Len()%8 != 0 {
			loadCmdsBuf.WriteByte(0)
		}
		ncmds++
	}

	// 6. LC_MAIN (entry point)
	{
		entry := EntryPointCommand{
			Cmd:       LC_MAIN,
			CmdSize:   uint32(binary.Size(EntryPointCommand{})),
			EntryOff:  0, // Entry is at start of __text
			StackSize: 0,
		}
		binary.Write(&loadCmdsBuf, binary.LittleEndian, &entry)
		ncmds++
	}

	// 7. LC_LOAD_DYLIB for libSystem.B.dylib (required for any macOS executable)
	if eb.useDynamicLinking {
		dylibPath := "/usr/lib/libSystem.B.dylib\x00"
		cmdSize := uint32(binary.Size(LoadCommand{}) + 16 + len(dylibPath))
		cmdSize = (cmdSize + 7) &^ 7 // 8-byte align

		cmd := LoadCommand{
			Cmd:     LC_LOAD_DYLIB,
			CmdSize: cmdSize,
		}
		binary.Write(&loadCmdsBuf, binary.LittleEndian, &cmd)
		binary.Write(&loadCmdsBuf, binary.LittleEndian, uint32(24)) // name offset
		binary.Write(&loadCmdsBuf, binary.LittleEndian, uint32(0))  // timestamp
		binary.Write(&loadCmdsBuf, binary.LittleEndian, uint32(0))  // current version
		binary.Write(&loadCmdsBuf, binary.LittleEndian, uint32(0))  // compatibility version
		loadCmdsBuf.WriteString(dylibPath)

		// Pad to alignment
		for loadCmdsBuf.Len()%8 != 0 {
			loadCmdsBuf.WriteByte(0)
		}
		ncmds++
	}

	// 8. LC_SYMTAB (if dynamic linking)
	if eb.useDynamicLinking && numImports > 0 {
		symtabCmd := SymtabCommand{
			Cmd:     LC_SYMTAB,
			CmdSize: uint32(binary.Size(SymtabCommand{})),
			Symoff:  uint32(linkeditFileOffset),
			Nsyms:   uint32(len(symtab)),
			Stroff:  uint32(linkeditFileOffset) + symtabSize,
			Strsize: strtabSize,
		}
		binary.Write(&loadCmdsBuf, binary.LittleEndian, &symtabCmd)
		ncmds++
	}

	// 9. LC_DYSYMTAB (if dynamic linking)
	if eb.useDynamicLinking && numImports > 0 {
		dysymtabCmd := DysymtabCommand{
			Cmd:            LC_DYSYMTAB,
			CmdSize:        uint32(binary.Size(DysymtabCommand{})),
			ILocalSym:      0,
			NLocalSym:      0,
			IExtDefSym:     0,
			NExtDefSym:     0,
			IUndefSym:      0,
			NUndefSym:      uint32(len(symtab)),
			TOCOff:         0,
			NTOC:           0,
			ModTabOff:      0,
			NModTab:        0,
			ExtRefSymOff:   0,
			NExtRefSyms:    0,
			IndirectSymOff: uint32(linkeditFileOffset) + symtabSize + strtabSize,
			NIndirectSyms:  uint32(len(indirectSymTab)),
			ExtRelOff:      0,
			NExtRel:        0,
			LocRelOff:      0,
			NLocRel:        0,
		}
		binary.Write(&loadCmdsBuf, binary.LittleEndian, &dysymtabCmd)
		ncmds++
	}

	// 10. LC_DYLD_CHAINED_FIXUPS (if dynamic linking)
	if debug {
		fmt.Fprintf(os.Stderr, "DEBUG: About to write LC_DYLD_CHAINED_FIXUPS: useDynamicLinking=%v, numImports=%d, chainedFixupsSize=%d\n",
			eb.useDynamicLinking, numImports, chainedFixupsSize)
	}
	if eb.useDynamicLinking && numImports > 0 {
		chainedFixupsCmd := LinkEditDataCommand{
			Cmd:      LC_DYLD_CHAINED_FIXUPS,
			CmdSize:  uint32(binary.Size(LinkEditDataCommand{})),
			DataOff:  uint32(linkeditFileOffset) + symtabSize + strtabSize + indirectSymTabSize,
			DataSize: chainedFixupsSize,
		}
		if debug {
			fmt.Fprintf(os.Stderr, "DEBUG: Writing LC_DYLD_CHAINED_FIXUPS with DataSize=%d, DataOff=%d\n",
				chainedFixupsSize, uint32(linkeditFileOffset)+symtabSize+strtabSize+indirectSymTabSize)
		}
		binary.Write(&loadCmdsBuf, binary.LittleEndian, &chainedFixupsCmd)
		ncmds++
		if debug {
			fmt.Fprintf(os.Stderr, "DEBUG: After writing LC_DYLD_CHAINED_FIXUPS, ncmds=%d\n", ncmds)
		}
	}

	// Verify our preliminary calculation was correct
	loadCmdsSize := uint32(loadCmdsBuf.Len())
	if loadCmdsSize != prelimLoadCmdsSize {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Warning: load commands size mismatch: expected %d, got %d\n", prelimLoadCmdsSize, loadCmdsSize)
		}
	}

	// Write Mach-O header
	if debug {
		fmt.Fprintf(os.Stderr, "DEBUG: About to write Mach-O header with NCmds=%d, SizeOfCmds=%d\n", ncmds, loadCmdsSize)
	}
	header := MachOHeader64{
		Magic:      MH_MAGIC_64,
		CPUType:    cpuType,
		CPUSubtype: cpuSubtype,
		FileType:   MH_EXECUTE,
		NCmds:      ncmds,
		SizeOfCmds: loadCmdsSize,
		Flags:      MH_NOUNDEFS | MH_DYLDLINK | MH_TWOLEVEL,
		Reserved:   0,
	}
	binary.Write(&buf, binary.LittleEndian, &header)
	if debug {
		fmt.Fprintf(os.Stderr, "DEBUG: Wrote Mach-O header\n")
	}

	// Debug: verify what's actually in the buffer
	bufBytes := buf.Bytes()
	if len(bufBytes) >= 32 {
		ncmdsInBuf := binary.LittleEndian.Uint32(bufBytes[16:20])
		sizeofcmdsInBuf := binary.LittleEndian.Uint32(bufBytes[20:24])
		if debug {
			fmt.Fprintf(os.Stderr, "DEBUG: Header in buffer has NCmds=%d, SizeOfCmds=%d (NCmds bytes: %v)\n",
				ncmdsInBuf, sizeofcmdsInBuf, bufBytes[16:20])
		}
	}

	// Write load commands
	buf.Write(loadCmdsBuf.Bytes())

	// Patch bl instructions to call stubs (if dynamic linking)
	if eb.useDynamicLinking && numImports > 0 {
		textBytes := eb.text.Bytes()
		for _, patch := range eb.callPatches {
			// Find which stub this call refers to
			stubIndex := -1
			for i, funcName := range eb.neededFunctions {
				if patch.targetName == funcName+"$stub" {
					stubIndex = i
					break
				}
			}

			if stubIndex >= 0 {
				// Calculate stub address
				thisStubAddr := stubsAddr + uint64(stubIndex*12)
				// Calculate PC-relative offset from call instruction to stub
				callAddr := textSectAddr + uint64(patch.position)
				offset := int64(thisStubAddr-callAddr) / 4 // ARM64 offset in words

				// Patch bl instruction (opcode 0x94 in bits [31:26])
				blInstr := uint32(0x94000000) | (uint32(offset) & 0x03ffffff)
				textBytes[patch.position] = byte(blInstr)
				textBytes[patch.position+1] = byte(blInstr >> 8)
				textBytes[patch.position+2] = byte(blInstr >> 16)
				textBytes[patch.position+3] = byte(blInstr >> 24)
			}
		}
	}

	// Pad to page boundary
	for uint64(buf.Len()) < textFileOffset {
		buf.WriteByte(0)
	}

	// Write __text section
	buf.Write(eb.text.Bytes())

	// Write __stubs section (if dynamic linking)
	if eb.useDynamicLinking && numImports > 0 {
		// Generate stub code for each import
		for i := uint32(0); i < numImports; i++ {
			// ARM64 stub pattern (12 bytes):
			// adrp x16, GOT@PAGE
			// ldr x16, [x16, GOT@PAGEOFF]
			// br x16

			// Calculate GOT entry address for this import
			gotEntryAddr := gotAddr + uint64(i*8)
			stubAddr := stubsAddr + uint64(i*12)

			// Calculate PC-relative offset from stub to GOT entry
			// ADRP: PC-relative page address
			pcRelPage := int64((gotEntryAddr &^ 0xfff) - (stubAddr &^ 0xfff))
			adrpImm := (pcRelPage >> 12) & 0x1fffff
			adrpImmLo := (adrpImm & 0x3) << 29
			adrpImmHi := (adrpImm >> 2) << 5
			adrpInstr := uint32(0x90000010) | uint32(adrpImmLo) | uint32(adrpImmHi) // adrp x16, #page

			// LDR: Load from [x16 + pageoffset]
			pageOffset := (gotEntryAddr & 0xfff) >> 3                   // Divide by 8 for 8-byte loads
			ldrInstr := uint32(0xf9400210) | (uint32(pageOffset) << 10) // ldr x16, [x16, #offset]

			// BR: Branch to x16
			brInstr := uint32(0xd61f0200) // br x16

			binary.Write(&buf, binary.LittleEndian, adrpInstr)
			binary.Write(&buf, binary.LittleEndian, ldrInstr)
			binary.Write(&buf, binary.LittleEndian, brInstr)
		}
	}

	// Pad to page boundary
	for uint64(buf.Len())%pageSize != 0 {
		buf.WriteByte(0)
	}

	// Write __data section (rodata)
	if rodataSize > 0 {
		buf.Write(eb.rodata.Bytes())
	}

	// Write __got section (if dynamic linking)
	if eb.useDynamicLinking && numImports > 0 {
		// GOT entries are initially zero (dyld will fill them at runtime)
		for i := uint32(0); i < numImports; i++ {
			binary.Write(&buf, binary.LittleEndian, uint64(0))
		}
	}

	// Write __LINKEDIT segment (if dynamic linking)
	if eb.useDynamicLinking && numImports > 0 {
		// Pad to linkedit file offset
		for uint64(buf.Len()) < linkeditFileOffset {
			buf.WriteByte(0)
		}

		// Write symbol table
		for _, sym := range symtab {
			binary.Write(&buf, binary.LittleEndian, &sym)
		}

		// Write string table
		buf.Write(strtab.Bytes())

		// Write indirect symbol table
		for _, idx := range indirectSymTab {
			binary.Write(&buf, binary.LittleEndian, idx)
		}

		// Write chained fixups data
		// Calculate offsets within the chained fixups data
		headerSize := uint32(binary.Size(DyldChainedFixupsHeader{}))
		startsOffset := headerSize
		startsImageSize := uint32(4 + (2 * 4))                                     // seg_count + 2 segment offsets
		startsSegmentSize := uint32(binary.Size(DyldChainedStartsInSegment{}) + 2) // + 1 page_start
		importsOffset := startsOffset + startsImageSize + startsSegmentSize
		symbolsOffset := importsOffset + (6 * numImports)

		// 1. Write DyldChainedFixupsHeader
		chainedHeader := DyldChainedFixupsHeader{
			FixupsVersion: 0,
			StartsOffset:  startsOffset,
			ImportsOffset: importsOffset,
			SymbolsOffset: symbolsOffset,
			ImportsCount:  numImports,
			ImportsFormat: DYLD_CHAINED_IMPORT,
			SymbolsFormat: 0, // Uncompressed
		}
		binary.Write(&buf, binary.LittleEndian, &chainedHeader)

		// 2. Write DyldChainedStartsInImage
		segCount := uint32(1) // Only __DATA segment has fixups
		binary.Write(&buf, binary.LittleEndian, segCount)
		// Offset to __DATA segment starts (relative to StartsOffset)
		dataSegStartsOffset := uint32(startsImageSize)
		binary.Write(&buf, binary.LittleEndian, dataSegStartsOffset)

		// 3. Write DyldChainedStartsInSegment for __DATA
		startsSegment := DyldChainedStartsInSegment{
			Size:            uint32(binary.Size(DyldChainedStartsInSegment{}) + 2),
			PageSize:        0x4000, // 16KB pages
			PointerFormat:   DYLD_CHAINED_PTR_ARM64E,
			SegmentOffset:   0,
			MaxValidPointer: 0,
			PageCount:       1, // GOT fits in one page
		}
		binary.Write(&buf, binary.LittleEndian, &startsSegment)
		// Page start: first fixup at offset 0 in GOT
		binary.Write(&buf, binary.LittleEndian, uint16(0))

		// 4. Write imports table (DyldChainedImport entries)
		for i, funcName := range eb.neededFunctions {
			// Find symbol name offset in strtab
			nameOffset := uint32(0)
			searchBytes := []byte(funcName + "\x00")
			strtabBytes := strtab.Bytes()
			for j := 0; j < len(strtabBytes)-len(searchBytes)+1; j++ {
				if bytes.Equal(strtabBytes[j:j+len(searchBytes)], searchBytes) {
					nameOffset = uint32(j)
					break
				}
			}

			// Write DyldChainedImport (6 bytes total)
			libOrdinal := uint8(1) // libSystem.B.dylib is library ordinal 1
			weakImport := uint8(0) // Not a weak import
			binary.Write(&buf, binary.LittleEndian, libOrdinal)
			binary.Write(&buf, binary.LittleEndian, weakImport)
			// NameOffset is 32-bit but only uses lower 24 bits
			binary.Write(&buf, binary.LittleEndian, nameOffset)

			_ = i // Suppress unused variable warning
		}

		// 5. Write symbol strings (copy from strtab)
		buf.Write(strtab.Bytes())

		// 6. Align to 8 bytes
		for buf.Len()%8 != 0 {
			buf.WriteByte(0)
		}
	}

	eb.elf = buf
	return nil
}
