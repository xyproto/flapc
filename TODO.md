# Flapc TODO - Actionable Items for v1.3.0 Polish & Robustness

## Critical Fixes
- [ ] Run full test suite through valgrind and fix all memory leaks
- [ ] Add bounds checking for array/memory access in generated code
- [ ] Validate pointer dereferences before use in generated assembly
- [ ] Fix any remaining panic/crash points with graceful error messages

## Error Handling & User Experience
- [ ] Improve error messages with line/column numbers in parser
- [ ] Add parse error recovery to continue checking rest of file
- [ ] Add clear error messages for common syntax mistakes
- [ ] Document all error codes and recovery strategies in documentation

## Testing & Quality Assurance
- [ ] Test installation and execution on fresh Ubuntu 22.04 VM
- [ ] Verify all existing test programs compile and run correctly
- [ ] Create comprehensive test for atomic operations with multiple threads
- [ ] Add edge case tests for atomic operations (overflow, underflow, etc.)

## Documentation
- [ ] Document atomic operations memory ordering guarantees in GRAMMAR.md
- [ ] Add troubleshooting section to README.md
- [ ] Write examples demonstrating proper use of atomic operations
- [ ] Update LANGUAGE.md with complete atomic operations reference

## Performance & Code Quality
- [ ] Profile compiler and optimize slowest parsing paths
- [ ] Reduce memory allocations in hot code generation paths
- [ ] Review and clean up generated assembly for common patterns
- [ ] Ensure consistent code style across all Go source files

## Platform Compatibility
- [ ] Verify x86-64 code generation works on both Arch and Ubuntu
- [ ] Test with different kernel versions (5.x, 6.x)
- [ ] Validate syscall numbers are correct for target platforms
- [ ] Ensure libc dependencies are minimal and documented

---

## Future Releases

### v1.4.0 - Parallel Computing & Concurrency
- Complete parallel loop reducer implementation (codegen for parsed syntax)
- Thread-local storage for parallel loop partial results
- Thread-safe printf wrapper for parallel loops
- Parallel sum and mandelbrot example programs
- Support dynamic range bounds in parallel loops

### v1.5.0 - Advanced Networking
- Parse sockaddr_in to extract sender IP and port
- Network message handling with proper string conversion
- Connection tracking with timeout support
- ENet protocol implementation (reliable channels, packet ordering)
- Network chat example program

### v1.6.0 - FFI & Interoperability
- C callback function pointer support
- C++ header parsing with name demangling
- Struct return value support for C functions
- Complete SDL3 game example with event handling
- Fix string argument passing edge cases

### v2.0.0 - Major Features
- Hot reload infrastructure with function pointer swapping
- Trampolines for deep recursion without stack overflow
- Let bindings for local recursive definitions
- Macro system for compile-time metaprogramming
- Windows support with PE executable generation

### v3.0.0 - Advanced Tooling
- Full LLVM IR backend option
- Incremental compilation with dependency tracking
- Package manager integration
- Language server protocol support
- Watch mode (--watch) for automatic recompilation