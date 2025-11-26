# ARM64 Support Status

## Working ✓
- Simple value returns (e.g., `main = { 42 }`)
- ELF binary generation for ARM64+Linux
- Target flag parsing (`--target arm64-linux`)
- Basic register operations (mov, add, sub, cmp, jump)
- Syscalls (write, exit)
- **String literals with println** ✅ NEW!
- **PC-relative addressing for rodata** ✅ NEW!
- **PLT/GOT infrastructure** ✅ NEW!

## In Progress / Not Working ✗
- **Printf-based I/O** (numbers) - calling convention issue
- C FFI integration (printf hangs, other functions untested)
- Complex expressions with libc calls
- Loops
- Match expressions  
- Lambda functions
- Parallel execution

## Recent Fixes (2025-11-26)
1. ✅ **Rodata preparation**: Added proper symbol collection and buffer writing
2. ✅ **PC relocation patching**: Re-patch after address updates
3. ✅ **PLT call patching**: Added call site patching infrastructure

## Working Example
```flap
println("Hello from ARM64!")
println("Strings work perfectly!")
main = { 42 }
```

## Known Issues
1. **Printf hangs**: Stack alignment or calling convention issue with variadic functions
2. **Number println**: Currently uses printf (which hangs) - needs native conversion

## Next Steps
1. Implement native number-to-string conversion for println
2. Debug printf calling convention issue
3. Add ARM64-specific tests
4. Complete runtime helper functions

## Architecture
- `arm64_backend.go`: Backend interface implementation
- `arm64_codegen.go`: Main code generation
- `arm64_instructions.go`: Instruction encoding
- `codegen_arm64_writer.go`: ELF writer for ARM64 (rodata, PLT, PC relocations)
- `pltgot_aarch64.go`: PLT/GOT support
