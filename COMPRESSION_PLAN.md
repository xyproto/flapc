# 10-Step Plan for Compressed Self-Extracting Executables

## Goal
Generate compressed executables from Flapc that self-decompress to memory and execute, targeting 4k demoscene intro size.

## 10-Step Implementation Plan

### Step 1: Create Minimal Decompressor Stub (Assembly)
- Write tiny x86-64 assembly decompressor (~150 bytes)
- Use simple RLE or LZ77 variant for minimal code size
- Test standalone: compress a "Hello World" payload and verify decompression

### Step 2: Test Decompressor Stub Standalone
- Create test_decompress.go that:
  - Embeds the decompressor stub
  - Compresses a simple payload
  - Builds executable with stub + compressed data
  - Verifies it runs correctly

### Step 3: Implement Compression Algorithm in Go
- Add lightweight compression to compress.go
- Focus on compression ratio, not speed (offline compression)
- Target: 40-60% size reduction minimum

### Step 4: Create Stub Wrapper Functions
- Add functions in codegen.go to:
  - Emit decompressor stub
  - Calculate compressed payload size
  - Emit compressed data
  - Wire up entry point to stub

### Step 5: Modify ELF Writer for Compressed Mode
- Update codegen_elf_writer.go:
  - Add compressed section after stub
  - Adjust entry point to stub
  - Allocate executable memory in stub via mmap

### Step 6: Modify PE Writer for Compressed Mode
- Update codegen_pe_writer.go:
  - Similar changes for Windows
  - Use VirtualAlloc in stub instead of mmap

### Step 7: Add Compression Flag to Compiler
- Add internal flag to enable compression
- Initially default OFF for testing
- Hook into writeELF/writePE/writeMachO

### Step 8: Test Linux x86-64 Compressed Executables
- Test with simple Flap programs
- Verify size reduction
- Verify correct execution
- Test with SDL3 example

### Step 9: Test Windows x86-64 Compressed Executables
- Same tests with Wine
- Verify Windows-specific stub works

### Step 10: Enable by Default and Validate
- Enable compression by default
- Run full test suite
- Verify all tests pass
- Update documentation

## Success Criteria
- Executables are 40-60% smaller
- All existing tests pass
- SDL3 example works on Linux and Windows
- No runtime overhead beyond initial decompression
- Simple programs can fit in 4k
