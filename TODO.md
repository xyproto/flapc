# Flap TODO

## Current Status (2025-12-03)

### All Tests Passing! ✅

**Go test**: PASS (0.746s)

### Working Features ✅
- **Float printing with full precision!** ✅
  - Fully inline assembly implementation (no libc)
  - Direct syscalls for output
  - SSE2 instructions for decimal extraction
  - Supports printf precision specifiers (%.2f, %.6f, etc.)
- **Dynamic library linking FIXED!** ✅
  - DT_NEEDED entries only when C FFI is used
  - PLT/GOT correctly set up for external functions
  - libc only linked when needed
- **Arena allocator FULLY IMPLEMENTED!** ✅
  - Default arena allocated at program start
  - Arena blocks with automatic cleanup on scope exit
  - Used for all internal allocations (strings, lists, etc.)
- **Pure assembly number conversion** ✅
  - `_flap_itoa` implemented in pure x86_64 assembly
  - Integer and float printing without libc
- **Windows SDL3 support WORKING!** ✅
  - SDL3 example compiles and runs on Windows
- **Higher-order functions WORKING!** ✅
  - Functions can be passed as parameters
  - `apply := f, x -> f(x)` works correctly
- **Executable compression READY** 📋
  - Compression infrastructure exists (compress.go, decompressor_stub.go)
  - Not yet integrated into compilation pipeline
  - Foundation ready for 4k demoscene intros
- **All tests passing!** ✅

### Platform Support
- ✅ Linux x86_64: Fully working, no libc required
- ✅ Windows x86_64: Fully working
- 🚧 Linux ARM64: Backend exists (needs testing)
- 🚧 Linux RISC-V64: Backend exists (needs testing)
- ❌ Windows ARM64: Not yet implemented
- ❌ macOS ARM64: Not yet implemented

### Known Limitations
- macOS will need libc for syscalls (no direct syscall support)
- ARM64/RISC-V float printing needs implementation
- Pipeline with lambdas may have edge cases

## Remaining Work

### Code Generation
- Implement float printing for ARM64/RISC-V backends
- Add ARM64/RISC-V compression stubs
- Optimize O(n²) algorithms

### Type System
- Complete type inference
- Ensure C types integrate with Flap's universal type
- Add runtime type checking (optional)

### Standard Library
- Expand minimal runtime
- Add common game utilities
- Document all builtins

### Testing
- Test ARM64 backend on actual hardware
- Test RISC-V backend on actual hardware

## Future Enhancements

### High Priority
- Function composition (`<>` operator) full implementation
- Re-enable optimizer when type system is complete
- Integrate executable compression into compilation pipeline

### Medium Priority
- Hot reload improvements (patch running process via IPC)
- Performance profiling tools
- Interactive REPL

### Low Priority
- WASM target
- WebGPU bindings
- Language server protocol support
- Package manager

---

## Recent Accomplishments (2025-12-03)

### ✅ Completed
1. **Float printing fixed!** - Pure assembly implementation with full precision
2. **All tests passing** - `go test` clean
3. **libc-free Linux builds** - Only uses libc when C FFI is needed
4. **Printf precision support** - %.2f, %.6f, etc. all working
5. **Arena allocator** - Full implementation with proper cleanup
6. **Higher-order functions** - Functions as parameters working
7. **Match return values** - Fixed return value handling
8. **Pure assembly number conversion** - No libc dependencies
9. **Executable compression ready** - aPLib with tiny decompressor stub
10. **Windows support** - SDL3 example working

### 🎯 Next Steps
1. Test ARM64/RISC-V backends on hardware
2. Implement float printing for ARM64/RISC-V
3. Integrate executable compression
4. Re-enable optimizer with type system improvements
