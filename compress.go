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
	var stub []byte

	// Save registers
	stub = append(stub, 0x53)       // push rbx
	stub = append(stub, 0x55)       // push rbp
	stub = append(stub, 0x41, 0x54) // push r12
	stub = append(stub, 0x41, 0x55) // push r13
	stub = append(stub, 0x41, 0x56) // push r14

	// Allocate memory: mmap(NULL, size, PROT_READ|PROT_WRITE|PROT_EXEC, MAP_PRIVATE|MAP_ANONYMOUS, -1, 0)
	stub = append(stub, 0x48, 0x31, 0xFF) // xor rdi, rdi
	stub = append(stub, 0x48, 0xBE)       // movabs rsi, decompressedSize
	stub = append(stub, uint64ToBytes(uint64(decompressedSize))...)
	stub = append(stub, 0x48, 0xC7, 0xC2, 0x07, 0x00, 0x00, 0x00) // mov rdx, 7
	stub = append(stub, 0x49, 0xC7, 0xC2, 0x22, 0x00, 0x00, 0x00) // mov r10, 0x22
	stub = append(stub, 0x49, 0xC7, 0xC0, 0xFF, 0xFF, 0xFF, 0xFF) // mov r8, -1
	stub = append(stub, 0x4D, 0x31, 0xC9)                         // xor r9, r9
	stub = append(stub, 0x48, 0xC7, 0xC0, 0x09, 0x00, 0x00, 0x00) // mov rax, 9
	stub = append(stub, 0x0F, 0x05)                               // syscall
	stub = append(stub, 0x48, 0x85, 0xC0)                         // test rax, rax

	// Jump to error handler
	errorJmpPos := len(stub)
	stub = append(stub, 0x78, 0x00) // js error (will patch offset)

	stub = append(stub, 0x49, 0x89, 0xC4) // mov r12, rax (save dest pointer)

	// Get source pointer
	stub = append(stub, 0x48, 0x8D, 0x35) // lea rsi, [rip+offset]
	leaOffsetPos := len(stub)
	stub = append(stub, 0x00, 0x00, 0x00, 0x00) // Placeholder

	// Load original size and setup pointers
	stub = append(stub, 0x8B, 0x0E)             // mov ecx, [rsi]
	stub = append(stub, 0x48, 0x83, 0xC6, 0x04) // add rsi, 4
	stub = append(stub, 0x4C, 0x89, 0xE7)       // mov rdi, r12
	stub = append(stub, 0x4D, 0x89, 0xE5)       // mov r13, r12
	stub = append(stub, 0x49, 0x01, 0xCD)       // add r13, rcx (r13 = end)

	// Main decompress loop
	decompressLoopStart := len(stub)
	stub = append(stub, 0x4C, 0x39, 0xEF) // cmp rdi, r13

	doneJmpPos := len(stub)
	stub = append(stub, 0x73, 0x00) // jae done (will patch)

	stub = append(stub, 0xAC)       // lodsb (al = [rsi++])
	stub = append(stub, 0x3C, 0xFF) // cmp al, 0xFF

	literalJmpPos := len(stub)
	stub = append(stub, 0x75, 0x00) // jne literal (will patch)

	// Match case
	stub = append(stub, 0x66, 0xAD)       // lodsw
	stub = append(stub, 0x0F, 0xB7, 0xD8) // movzx ebx, ax
	stub = append(stub, 0xAC)             // lodsb
	stub = append(stub, 0x0F, 0xB6, 0xD0) // movzx edx, al
	stub = append(stub, 0x66, 0x85, 0xDB) // test bx, bx

	copyMatchJmpPos := len(stub)
	stub = append(stub, 0x75, 0x00) // jnz copy_match (will patch)

	stub = append(stub, 0x83, 0xFA, 0x01) // cmp edx, 1

	copyMatchJmpPos2 := len(stub)
	stub = append(stub, 0x75, 0x00) // jne copy_match (will patch)

	// Escaped 0xFF
	stub = append(stub, 0xC6, 0x07, 0xFF) // mov byte [rdi], 0xFF
	stub = append(stub, 0x48, 0xFF, 0xC7) // inc rdi

	loopBackJmpPos := len(stub)
	stub = append(stub, 0xEB, 0x00) // jmp decompress_loop (will patch)

	// copy_match:
	copyMatchLabel := len(stub)
	stub = append(stub, 0x48, 0x89, 0xF8) // mov rax, rdi
	stub = append(stub, 0x48, 0x29, 0xD8) // sub rax, rbx

	// copy_loop:
	copyLoopStart := len(stub)
	stub = append(stub, 0x8A, 0x08)       // mov cl, [rax]
	stub = append(stub, 0x88, 0x0F)       // mov [rdi], cl
	stub = append(stub, 0x48, 0xFF, 0xC0) // inc rax
	stub = append(stub, 0x48, 0xFF, 0xC7) // inc rdi
	stub = append(stub, 0xFF, 0xCA)       // dec edx

	copyLoopJmpPos := len(stub)
	stub = append(stub, 0x75, 0x00) // jnz copy_loop (will patch)

	loopBackJmpPos2 := len(stub)
	stub = append(stub, 0xEB, 0x00) // jmp decompress_loop (will patch)

	// literal:
	literalLabel := len(stub)
	stub = append(stub, 0xAA) // stosb

	loopBackJmpPos3 := len(stub)
	stub = append(stub, 0xEB, 0x00) // jmp decompress_loop (will patch)

	// done:
	doneLabel := len(stub)
	stub = append(stub, 0x41, 0x5E)       // pop r14
	stub = append(stub, 0x41, 0x5D)       // pop r13
	stub = append(stub, 0x41, 0x5C)       // pop r12
	stub = append(stub, 0x5D)             // pop rbp
	stub = append(stub, 0x5B)             // pop rbx
	stub = append(stub, 0x49, 0xFF, 0xE4) // jmp r12

	// error:
	errorLabel := len(stub)
	stub = append(stub, 0x48, 0xC7, 0xC0, 0x3C, 0x00, 0x00, 0x00) // mov rax, 60
	stub = append(stub, 0x48, 0xC7, 0xC7, 0x01, 0x00, 0x00, 0x00) // mov rdi, 1
	stub = append(stub, 0x0F, 0x05)                               // syscall

	// Patch all jump offsets
	stub[errorJmpPos+1] = byte(errorLabel - (errorJmpPos + 2))
	stub[doneJmpPos+1] = byte(doneLabel - (doneJmpPos + 2))
	stub[literalJmpPos+1] = byte(literalLabel - (literalJmpPos + 2))
	stub[copyMatchJmpPos+1] = byte(copyMatchLabel - (copyMatchJmpPos + 2))
	stub[copyMatchJmpPos2+1] = byte(copyMatchLabel - (copyMatchJmpPos2 + 2))
	stub[loopBackJmpPos+1] = byte(int(decompressLoopStart) - int(loopBackJmpPos+2))
	stub[copyLoopJmpPos+1] = byte(int(copyLoopStart) - int(copyLoopJmpPos+2))
	stub[loopBackJmpPos2+1] = byte(int(decompressLoopStart) - int(loopBackJmpPos2+2))
	stub[loopBackJmpPos3+1] = byte(int(decompressLoopStart) - int(loopBackJmpPos3+2))

	// Patch LEA offset
	ripAfterLea := leaOffsetPos + 4
	compressedDataOffset := len(stub) - ripAfterLea
	binary.LittleEndian.PutUint32(stub[leaOffsetPos:leaOffsetPos+4], uint32(compressedDataOffset))

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
