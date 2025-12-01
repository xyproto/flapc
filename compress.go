package main

import (
	"bytes"
	"encoding/binary"
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
	stub := []byte{}
	
	// mmap(NULL, decompressedSize, PROT_READ|PROT_WRITE|PROT_EXEC, MAP_PRIVATE|MAP_ANONYMOUS, -1, 0)
	stub = append(stub, 0x48, 0xC7, 0xC0, 0x09, 0x00, 0x00, 0x00) // mov rax, 9 (mmap)
	stub = append(stub, 0x48, 0x31, 0xFF)                         // xor rdi, rdi (NULL)
	stub = append(stub, 0x48, 0xC7, 0xC6)                         // mov rsi, decompressedSize
	binary.LittleEndian.PutUint32(stub[len(stub):len(stub)+4], decompressedSize)
	stub = append(stub, make([]byte, 4)...)
	stub = append(stub, 0x48, 0xC7, 0xC2, 0x07, 0x00, 0x00, 0x00) // mov rdx, 7 (PROT_READ|WRITE|EXEC)
	stub = append(stub, 0x48, 0xC7, 0xC1, 0x22, 0x00, 0x00, 0x00) // mov rcx, 0x22 (MAP_PRIVATE|ANONYMOUS)
	stub = append(stub, 0x4D, 0x31, 0xC0)                         // xor r8, r8 (-1)
	stub = append(stub, 0x49, 0xC7, 0xC1, 0xFF, 0xFF, 0xFF, 0xFF) // mov r9, -1
	stub = append(stub, 0x0F, 0x05)                               // syscall
	
	// rax now contains destination buffer
	stub = append(stub, 0x48, 0x89, 0xC7)                         // mov rdi, rax (dst)
	stub = append(stub, 0x48, 0x8D, 0x35, 0x10, 0x00, 0x00, 0x00) // lea rsi, [rip+16] (src = compressed data after stub)
	stub = append(stub, 0x48, 0xC7, 0xC1)                         // mov rcx, decompressedSize
	binary.LittleEndian.PutUint32(stub[len(stub):len(stub)+4], decompressedSize)
	stub = append(stub, make([]byte, 4)...)
	
	// Decompress loop
	// Format: 0xFF <dist:u16> <len:u8> = copy, else literal
	_ = len(stub) // loop offset
	stub = append(stub, 0x48, 0x85, 0xC9)                         // test rcx, rcx
	stub = append(stub, 0x74, 0x2E)                               // jz end (offset will be adjusted)
	stub = append(stub, 0x48, 0x8A, 0x06)                         // mov al, [rsi]
	stub = append(stub, 0x48, 0xFF, 0xC6)                         // inc rsi
	stub = append(stub, 0x3C, 0xFF)                               // cmp al, 0xFF
	stub = append(stub, 0x75, 0x09)                               // jne literal
	
	// Match case
	stub = append(stub, 0x48, 0x0F, 0xB7, 0x16)                   // movzx rdx, word [rsi]
	stub = append(stub, 0x48, 0x83, 0xC6, 0x02)                   // add rsi, 2
	stub = append(stub, 0x48, 0x8A, 0x1E)                         // mov bl, [rsi]
	stub = append(stub, 0x48, 0xFF, 0xC6)                         // inc rsi
	stub = append(stub, 0x48, 0x29, 0xD1)                         // sub rcx, rdx (decrement remaining)
	// Copy loop
	stub = append(stub, 0x48, 0x89, 0xFA)                         // mov rdx, rdi
	stub = append(stub, 0x48, 0x29, 0xD2)                         // sub rdx, rdx (calculate src offset)
	stub = append(stub, 0x48, 0x8A, 0x02)                         // mov al, [rdx]
	stub = append(stub, 0x48, 0xAA)                               // stosb
	stub = append(stub, 0x48, 0xFF, 0xCB)                         // dec rbx
	stub = append(stub, 0x75, 0xF6)                               // jnz copy_loop
	stub = append(stub, 0xEB, 0xD1)                               // jmp decomp_loop
	
	// Literal case
	stub = append(stub, 0x48, 0xAA)                               // stosb
	stub = append(stub, 0x48, 0xFF, 0xC9)                         // dec rcx
	stub = append(stub, 0xEB, 0xCA)                               // jmp decomp_loop
	
	// End: jump to decompressed code
	stub = append(stub, 0xFF, 0xE7)                               // jmp rdi
	
	return stub
}

func generateARM64DecompressorStub(compressedSize, decompressedSize uint32) []byte {
	return []byte{}
}
