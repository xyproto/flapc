# macOS Mach-O Status

## ✅ Working Features

### Code Signing
- **Self-signing implemented**: Binaries are automatically signed with ad-hoc signatures
- No external `codesign` tool needed
- Generates proper SuperBlob and CodeDirectory structures
- SHA-256 hashes for each 4KB page
- Verified with `codesign -dvv` - shows correct ad-hoc signature

### Binary Structure
- Correct Mach-O headers (magic, CPU type, file type)
- __PAGEZERO segment (required on macOS)
- __TEXT segment with __text and __stubs sections
- __DATA segment with __data section  
- __LINKEDIT segment with symbol tables and code signature
- LC_CODE_SIGNATURE load command
- LC_BUILD_VERSION for macOS 11.0+
- MH_PIE flag for position-independent executables
- Proper segment alignment (16KB pages on ARM64)

### Symbol Naming
- Correct underscore prefix for macOS symbols (_printf, _exit, etc.)
- Proper symbol table generation
- Undefined external symbols marked correctly

## ❌ Known Issues

### Dynamic Linking (SIGBUS Crash)
**Status**: All binaries using dynamic linking crash with SIGBUS (exit code 138)

**Affected**:
- Any program calling `exit()`, `printf()`, `getpid()`, or other libc functions
- Both inline code (`-c` flag) and .flap file compilation
- TestMachOExecutable test

**Not Affected**:
- Binary structure tests (all pass)
- Code signature generation (works perfectly)
- Static ELF binaries on Linux

**Root Cause**: Unknown - possibly related to:
- GOT (Global Offset Table) setup
- PLT (Procedure Linkage Table) stubs
- Chained fixups implementation
- Entry point setup
- dyld loads libraries correctly, crash happens during execution

**Workaround**: None currently for programs using libc functions

## 📊 Test Results

```
PASS: TestMachOMagicNumber
PASS: TestMachOFileType  
PASS: TestMachOCPUTypes
PASS: TestMachOSegments
PASS: TestMachOPageZero
PASS: TestMachOTextSegment
PASS: TestMachOMinimalSize
FAIL: TestMachOExecutable (SIGBUS - pre-existing issue)
PASS: TestMachOFileCommand
PASS: TestMachOPermissions
```

## 🎯 Next Steps

1. **Debug dynamic linking crash**:
   - Use lldb to trace execution
   - Compare with working GCC-generated binary
   - Check GOT/PLT setup
   - Verify chained fixups structure

2. **Alternative**: Implement static linking for macOS
   - Use direct syscalls instead of libc
   - Would work for simple programs
   - Limited functionality (no printf, etc.)

## 📝 Recent Commits

- `dd860f0`: Implement ad-hoc code signature generation ✅
- `259730a`: Fix double underscore in symbol names ✅  
- `21656cd`: Fix __TEXT segment FileSize
- `2d1f1f8`: Add LC_CODE_SIGNATURE load command

