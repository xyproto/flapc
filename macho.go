package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
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
	switch eb.machine {
	case MachineX86_64:
		cpuType = CPU_TYPE_X86_64
		cpuSubtype = CPU_SUBTYPE_X86_64_ALL
	case MachineARM64:
		cpuType = CPU_TYPE_ARM64
		cpuSubtype = CPU_SUBTYPE_ARM64_ALL
	default:
		return fmt.Errorf("unsupported architecture for Mach-O: %s", eb.machine)
	}

	// Page size
	pageSize := uint64(0x4000) // 16KB for ARM64, but 4KB works for x86_64 too

	// Calculate sizes
	textSize := uint64(eb.text.Len())
	rodataSize := uint64(eb.rodata.Len())

	// Align sizes
	textSizeAligned := (textSize + pageSize - 1) &^ (pageSize - 1)
	rodataSizeAligned := (rodataSize + pageSize - 1) &^ (pageSize - 1)

	// Calculate addresses
	textAddr := pageSize     // __TEXT segment starts at one page
	textSectAddr := textAddr // __text section
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
			VMSize:   pageSize,
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

	// 2. LC_SEGMENT_64 for __TEXT with __text section
	{
		seg := SegmentCommand64{
			Cmd:      LC_SEGMENT_64,
			CmdSize:  uint32(binary.Size(SegmentCommand64{}) + binary.Size(Section64{})),
			VMAddr:   textAddr,
			VMSize:   textSizeAligned,
			FileOff:  0, // Will be at start of file after headers
			FileSize: textSize,
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
			Offset:    0, // Will be set later
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
			FileOff:  0, // Will be set later
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
			Offset:    0, // Will be set later
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

	// 7. LC_SYMTAB (empty for now)
	{
		symtab := SymtabCommand{
			Cmd:     LC_SYMTAB,
			CmdSize: uint32(binary.Size(SymtabCommand{})),
			Symoff:  0,
			Nsyms:   0,
			Stroff:  0,
			Strsize: 0,
		}
		binary.Write(&loadCmdsBuf, binary.LittleEndian, &symtab)
		ncmds++
	}

	// 8. LC_DYSYMTAB (empty for now)
	{
		dysymtab := DysymtabCommand{
			Cmd:     LC_DYSYMTAB,
			CmdSize: uint32(binary.Size(DysymtabCommand{})),
		}
		binary.Write(&loadCmdsBuf, binary.LittleEndian, &dysymtab)
		ncmds++
	}

	// Calculate header + load commands size
	headerSize := uint32(binary.Size(MachOHeader64{}))
	loadCmdsSize := uint32(loadCmdsBuf.Len())
	fileHeaderSize := headerSize + loadCmdsSize

	// Align to page boundary for first segment
	fileOffset := uint64((fileHeaderSize + uint32(pageSize) - 1) &^ (uint32(pageSize) - 1))

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
	for uint64(buf.Len()) < fileOffset {
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
