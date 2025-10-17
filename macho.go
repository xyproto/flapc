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
	LC_SEGMENT_64         = 0x19
	LC_SYMTAB             = 0x2
	LC_DYSYMTAB           = 0xb
	LC_LOAD_DYLINKER      = 0xe
	LC_UUID               = 0x1b
	LC_VERSION_MIN_MACOSX = 0x24
	LC_SOURCE_VERSION     = 0x2A
	LC_MAIN               = 0x80000028
	LC_LOAD_DYLIB         = 0xc
	LC_FUNCTION_STARTS    = 0x26
	LC_DATA_IN_CODE       = 0x29

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
	S_ATTR_PURE_INSTRUCTIONS = 0x80000000
	S_ATTR_SOME_INSTRUCTIONS = 0x00000400
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

// WriteMachO writes a Mach-O executable for macOS
func (eb *ExecutableBuilder) WriteMachO() error {
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

	// Align sizes
	textSizeAligned := (textSize + pageSize - 1) &^ (pageSize - 1)
	rodataSizeAligned := (rodataSize + pageSize - 1) &^ (pageSize - 1)

	// Calculate addresses - __TEXT starts after zero page
	textAddr := zeroPageSize      // __TEXT segment starts at 4GB
	textSectAddr := textAddr      // __text section
	rodataAddr := textAddr + textSizeAligned
	rodataSectAddr := rodataAddr // __rodata section

	// Build load commands in a temporary buffer
	var loadCmdsBuf bytes.Buffer
	ncmds := uint32(0)

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

	// Calculate header + load commands size (preliminary for offsets)
	// We need to know this before writing segments
	headerSize := uint32(binary.Size(MachOHeader64{}))
	// Pre-calculate all load commands to know total size
	prelimLoadCmdsSize := uint32(0)
	prelimLoadCmdsSize += uint32(binary.Size(SegmentCommand64{}))                        // __PAGEZERO
	prelimLoadCmdsSize += uint32(binary.Size(SegmentCommand64{}) + binary.Size(Section64{})) // __TEXT
	if rodataSize > 0 {
		prelimLoadCmdsSize += uint32(binary.Size(SegmentCommand64{}) + binary.Size(Section64{})) // __DATA
	}
	dylinkerPath := "/usr/lib/dyld\x00"
	dylinkerCmdSize := (uint32(binary.Size(LoadCommand{}) + 4 + len(dylinkerPath)) + 7) &^ 7
	prelimLoadCmdsSize += dylinkerCmdSize // LC_LOAD_DYLINKER
	prelimLoadCmdsSize += uint32(binary.Size(EntryPointCommand{})) // LC_MAIN
	if eb.useDynamicLinking {
		dylibPath := "/usr/lib/libSystem.B.dylib\x00"
		dylibCmdSize := (uint32(binary.Size(LoadCommand{}) + 16 + len(dylibPath)) + 7) &^ 7
		prelimLoadCmdsSize += dylibCmdSize // LC_LOAD_DYLIB
	}
	// Skip SYMTAB and DYSYMTAB - they need LINKEDIT segment

	fileHeaderSize := headerSize + prelimLoadCmdsSize
	// Align to page boundary for first segment
	textFileOffset := uint64((fileHeaderSize + uint32(pageSize) - 1) &^ (uint32(pageSize) - 1))
	rodataFileOffset := textFileOffset + textSizeAligned

	// 2. LC_SEGMENT_64 for __TEXT with __text section
	// The __TEXT segment should start at file offset 0 and include headers + code
	{
		// Round up filesize to match vmsize (which includes padding after text)
		textFileSize := textFileOffset + textSize
		if textFileSize > textSizeAligned {
			textFileSize = textSizeAligned
		}

		seg := SegmentCommand64{
			Cmd:      LC_SEGMENT_64,
			CmdSize:  uint32(binary.Size(SegmentCommand64{}) + binary.Size(Section64{})),
			VMAddr:   textAddr,
			VMSize:   textSizeAligned,
			FileOff:  0, // __TEXT starts at beginning of file
			FileSize: textFileSize, // Includes headers + text, but not exceeding vmsize
			MaxProt:  VM_PROT_READ | VM_PROT_EXECUTE,
			InitProt: VM_PROT_READ | VM_PROT_EXECUTE,
			NSects:   1,
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
		ncmds++
	}

	// 3. LC_SEGMENT_64 for __DATA with __data section (rodata goes here)
	if rodataSize > 0 {
		seg := SegmentCommand64{
			Cmd:      LC_SEGMENT_64,
			CmdSize:  uint32(binary.Size(SegmentCommand64{}) + binary.Size(Section64{})),
			VMAddr:   rodataAddr,
			VMSize:   rodataSizeAligned,
			FileOff:  rodataFileOffset,
			FileSize: rodataSize,
			MaxProt:  VM_PROT_READ | VM_PROT_WRITE,
			InitProt: VM_PROT_READ | VM_PROT_WRITE,
			NSects:   1,
			Flags:    0,
		}
		copy(seg.SegName[:], "__DATA")
		binary.Write(&loadCmdsBuf, binary.LittleEndian, &seg)

		// __data section (using for rodata)
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
		ncmds++
	}

	// 4. LC_LOAD_DYLINKER
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

	// 5. LC_MAIN (entry point)
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

	// 6. LC_LOAD_DYLIB for libSystem.B.dylib (required for any macOS executable)
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

	// Skip SYMTAB and DYSYMTAB for now - they need a proper LINKEDIT segment
	// These are optional for simple executables

	// Verify our preliminary calculation was correct
	loadCmdsSize := uint32(loadCmdsBuf.Len())
	if loadCmdsSize != prelimLoadCmdsSize {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Warning: load commands size mismatch: expected %d, got %d\n", prelimLoadCmdsSize, loadCmdsSize)
		}
	}

	// Write Mach-O header
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

	// Write load commands
	buf.Write(loadCmdsBuf.Bytes())

	// Pad to page boundary
	for uint64(buf.Len()) < textFileOffset {
		buf.WriteByte(0)
	}

	// Write __text section
	buf.Write(eb.text.Bytes())

	// Pad to page boundary
	for uint64(buf.Len())%pageSize != 0 {
		buf.WriteByte(0)
	}

	// Write __data section (rodata)
	if rodataSize > 0 {
		buf.Write(eb.rodata.Bytes())
	}

	eb.elf = buf
	return nil
}
