package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

type Compressor struct {
	windowSize int
	minMatch   int
}

func NewCompressor() *Compressor {
	return &Compressor{
		windowSize: 32768,
		minMatch:   4,
	}
}

func (c *Compressor) Compress(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	var compressed bytes.Buffer

	binary.Write(&compressed, binary.LittleEndian, uint32(len(data)))

	pos := 0
	for pos < len(data) {
		bestLen := 0
		bestDist := 0

		searchStart := pos - c.windowSize
		if searchStart < 0 {
			searchStart = 0
		}

		for i := searchStart; i < pos; i++ {
			matchLen := 0
			for matchLen < 255 && pos+matchLen < len(data) && data[i+matchLen] == data[pos+matchLen] {
				matchLen++
			}

			if matchLen >= c.minMatch && matchLen > bestLen {
				bestLen = matchLen
				bestDist = pos - i
			}
		}

		if bestLen >= c.minMatch {
			compressed.WriteByte(0xFF)
			binary.Write(&compressed, binary.LittleEndian, uint16(bestDist))
			compressed.WriteByte(byte(bestLen))
			pos += bestLen
		} else {
			literal := data[pos]
			if literal == 0xFF {
				compressed.WriteByte(0xFF)
				compressed.WriteByte(0x00)
				compressed.WriteByte(0x00)
				compressed.WriteByte(0x01)
			} else {
				compressed.WriteByte(literal)
			}
			pos++
		}
	}

	return compressed.Bytes()
}

func (c *Compressor) Decompress(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return data, nil
	}

	origSize := binary.LittleEndian.Uint32(data[0:4])
	decompressed := make([]byte, 0, origSize)

	pos := 4
	for pos < len(data) {
		if data[pos] == 0xFF {
			if pos+3 >= len(data) {
				break
			}
			dist := binary.LittleEndian.Uint16(data[pos+1 : pos+3])
			length := int(data[pos+3])

			if dist == 0 && length == 1 {
				decompressed = append(decompressed, 0xFF)
			} else {
				start := len(decompressed) - int(dist)
				for i := 0; i < length; i++ {
					decompressed = append(decompressed, decompressed[start+i])
				}
			}
			pos += 4
		} else {
			decompressed = append(decompressed, data[pos])
			pos++
		}
	}

	return decompressed, nil
}

func generateDecompressorStub(arch string, compressedSize, decompressedSize uint32) []byte {
	switch arch {
	case "amd64":
		return generateX64DecompressorStub(compressedSize, decompressedSize)
	case "arm64":
		return generateARM64DecompressorStub(compressedSize, decompressedSize)
	default:
		return nil
	}
}

func generateX64DecompressorStub(compressedSize, decompressedSize uint32) []byte {
	stub := []byte{
		// Save registers we'll use
		0x53,             // push rbx
		0x55,             // push rbp
		0x41, 0x54,       // push r12
		0x41, 0x55,       // push r13
		
		// Allocate memory for decompressed code: mmap(NULL, size, PROT_READ|PROT_WRITE|PROT_EXEC, MAP_PRIVATE|MAP_ANONYMOUS, -1, 0)
		0x48, 0x31, 0xFF, // xor rdi, rdi (addr = NULL)
		0x48, 0xBE,       // movabs rsi, decompressedSize
	}
	stub = append(stub, uint64ToBytes(uint64(decompressedSize))...)
	stub = append(stub, []byte{
		0x48, 0xC7, 0xC2, 0x07, 0x00, 0x00, 0x00, // mov rdx, 7 (PROT_READ|PROT_WRITE|PROT_EXEC)
		0x49, 0xC7, 0xC2, 0x22, 0x00, 0x00, 0x00, // mov r10, 0x22 (MAP_PRIVATE|MAP_ANONYMOUS)
		0x49, 0xC7, 0xC0, 0xFF, 0xFF, 0xFF, 0xFF, // mov r8, -1 (fd)
		0x4D, 0x31, 0xC9,                         // xor r9, r9 (offset = 0)
		0x48, 0xC7, 0xC0, 0x09, 0x00, 0x00, 0x00, // mov rax, 9 (sys_mmap)
		0x0F, 0x05,                               // syscall
		
		// Check for error
		0x48, 0x85, 0xC0,       // test rax, rax
		0x78, 0x10,             // js error (jump if sign flag set)
		
		0x49, 0x89, 0xC4,       // mov r12, rax (save dest pointer)
		
		// Get source pointer (compressed data follows this stub)
		0x48, 0x8D, 0x35, 0x00, 0x00, 0x00, 0x00, // lea rsi, [rip+0] (will patch)
	}...)
	
	// Decompress loop
	stub = append(stub, []byte{
		// rsi = source (compressed data + 4 byte header)
		// r12 = dest (decompressed buffer)
		// Load original size from header
		0x48, 0x8B, 0x0E,       // mov rcx, [rsi] (load 4-byte size, will use lower 32 bits)
		0x48, 0x83, 0xC6, 0x04, // add rsi, 4 (skip header)
		0x4C, 0x89, 0xE7,       // mov rdi, r12 (dest pointer)
		
		// Decompress loop
		// while (rcx > 0) { ... }
		// decompress_loop:
		0x48, 0x85, 0xC9,       // test rcx, rcx
		0x74, 0x3A,             // jz done (jump if zero)
		
		0x8A, 0x06,             // mov al, [rsi] (load byte)
		0x48, 0xFF, 0xC6,       // inc rsi
		
		0x3C, 0xFF,             // cmp al, 0xFF
		0x75, 0x2B,             // jne literal
		
		// Match: load distance and length
		0x0F, 0xB7, 0x1E,       // movzx ebx, word [rsi] (distance)
		0x48, 0x83, 0xC6, 0x02, // add rsi, 2
		0x8A, 0x2E,             // mov bpl, [rsi] (length)
		0x48, 0xFF, 0xC6,       // inc rsi
		
		// Check for literal 0xFF
		0x66, 0x85, 0xDB,       // test bx, bx
		0x75, 0x09,             // jnz copy_match
		0x40, 0x80, 0xFD, 0x01, // cmp bpl, 1
		0x75, 0x04,             // jne copy_match
		0xC6, 0x07, 0xFF,       // mov byte [rdi], 0xFF
		0xEB, 0x15,             // jmp next
		
		// copy_match:
		0x48, 0x89, 0xF8,       // mov rax, rdi
		0x48, 0x29, 0xD8,       // sub rax, rbx (source = dest - distance)
		0x40, 0x0F, 0xB6, 0xED, // movzx ebp, bpl
		// copy_loop:
		0x8A, 0x10,             // mov dl, [rax]
		0x88, 0x17,             // mov [rdi], dl
		0x48, 0xFF, 0xC0,       // inc rax
		0x48, 0xFF, 0xC7,       // inc rdi
		0x48, 0xFF, 0xCD,       // dec rbp
		0x75, 0xF3,             // jnz copy_loop
		0xEB, 0x03,             // jmp next
		
		// literal:
		0x88, 0x07,             // mov [rdi], al
		0x48, 0xFF, 0xC7,       // inc rdi
		
		// next:
		0x48, 0xFF, 0xC9,       // dec rcx
		0xEB, 0xBC,             // jmp decompress_loop
		
		// done:
		// Restore registers
		0x41, 0x5D,             // pop r13
		0x41, 0x5C,             // pop r12
		0x5D,                   // pop rbp
		0x5B,                   // pop rbx
		
		// Jump to decompressed code
		0x49, 0xFF, 0xE4,       // jmp r12
		
		// error:
		0x48, 0xC7, 0xC0, 0x3C, 0x00, 0x00, 0x00, // mov rax, 60 (sys_exit)
		0x48, 0xC7, 0xC7, 0x01, 0x00, 0x00, 0x00, // mov rdi, 1 (error code)
		0x0F, 0x05,                               // syscall
	}...)
	
	return stub
}

func uint64ToBytes(n uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, n)
	return b
}

func generateARM64DecompressorStub(compressedSize, decompressedSize uint32) []byte {
	// TODO: Implement ARM64 decompressor stub
	return []byte{}
}

// WrapWithDecompressor wraps an ELF executable with compression and decompressor stub
func WrapWithDecompressor(originalELF []byte, arch string) ([]byte, error) {
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG: WrapWithDecompressor called for arch=%s, size=%d\n", arch, len(originalELF))
	}
	
	// Extract the actual code sections to compress
	// For simplicity, compress everything after the ELF headers
	
	compressor := NewCompressor()
	
	// Compress the entire ELF
	compressed := compressor.Compress(originalELF)
	
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG: Compressed %d -> %d bytes\n", len(originalELF), len(compressed))
	}
	
	// Generate decompressor stub
	stub := generateDecompressorStub(arch, uint32(len(compressed)), uint32(len(originalELF)))
	if len(stub) == 0 {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG: No decompressor stub for arch %s\n", arch)
		}
		// Compression not supported for this arch, return original
		return originalELF, nil
	}
	
	// Build new ELF with:
	// 1. ELF header
	// 2. Program header (single LOAD segment)
	// 3. Decompressor stub
	// 4. Compressed data
	
	var result bytes.Buffer
	
	// ELF header
	elfHeader := make([]byte, 64)
	// ELF magic
	elfHeader[0] = 0x7F
	elfHeader[1] = 'E'
	elfHeader[2] = 'L'
	elfHeader[3] = 'F'
	elfHeader[4] = 2 // 64-bit
	elfHeader[5] = 1 // Little endian
	elfHeader[6] = 1 // ELF version
	// OS/ABI and padding
	elfHeader[16] = 2 // ET_EXEC
	
	if arch == "amd64" {
		binary.LittleEndian.PutUint16(elfHeader[18:20], 0x3E) // EM_X86_64
	} else if arch == "arm64" {
		binary.LittleEndian.PutUint16(elfHeader[18:20], 0xB7) // EM_AARCH64
	}
	
	binary.LittleEndian.PutUint32(elfHeader[20:24], 1) // EV_CURRENT
	
	// Entry point
	entryPoint := uint64(0x400000 + 64 + 56) // After headers
	binary.LittleEndian.PutUint64(elfHeader[24:32], entryPoint)
	
	// Program header offset
	binary.LittleEndian.PutUint64(elfHeader[32:40], 64)
	
	// Section header offset (none)
	binary.LittleEndian.PutUint64(elfHeader[40:48], 0)
	
	// Flags
	binary.LittleEndian.PutUint32(elfHeader[48:52], 0)
	
	// Header size
	binary.LittleEndian.PutUint16(elfHeader[52:54], 64)
	
	// Program header size and count
	binary.LittleEndian.PutUint16(elfHeader[54:56], 56)
	binary.LittleEndian.PutUint16(elfHeader[56:58], 1)
	
	result.Write(elfHeader)
	
	// Program header (LOAD segment with RWX)
	progHeader := make([]byte, 56)
	binary.LittleEndian.PutUint32(progHeader[0:4], 1) // PT_LOAD
	binary.LittleEndian.PutUint32(progHeader[4:8], 7) // PF_R|PF_W|PF_X
	
	// Offset in file
	binary.LittleEndian.PutUint64(progHeader[8:16], 0)
	
	// Virtual address
	binary.LittleEndian.PutUint64(progHeader[16:24], 0x400000)
	
	// Physical address
	binary.LittleEndian.PutUint64(progHeader[24:32], 0x400000)
	
	// Size in file and memory
	totalSize := uint64(64 + 56 + len(stub) + len(compressed))
	binary.LittleEndian.PutUint64(progHeader[32:40], totalSize)
	binary.LittleEndian.PutUint64(progHeader[40:48], totalSize)
	
	// Alignment
	binary.LittleEndian.PutUint64(progHeader[48:56], 0x1000)
	
	result.Write(progHeader)
	
	// Decompressor stub
	result.Write(stub)
	
	// Compressed data
	result.Write(compressed)
	
	return result.Bytes(), nil
}
