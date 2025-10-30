# Flapc v1.3.0 - Polish & Robustness Release

## Core Stability & Error Handling
- [ ] Add proper error checking for all sendto()/recvfrom() network calls with errno reporting
- [ ] Replace all panic/crash points with graceful error messages and recovery
- [ ] Run full test suite through valgrind and fix all memory leaks
- [ ] Test installation and execution on fresh Ubuntu 22.04 VM
- [ ] Add bounds checking for all array/memory access operations
- [ ] Validate all pointer dereferences before use

## Parallel Loop Completion
- [ ] Implement thread-local storage for parallel loop partial results
- [ ] Add reducer function syntax parsing: `@@ i in 0..<N { expr } | a,b | { reducer }`
- [ ] Generate atomic combination code using LOCK prefix for x86-64
- [ ] Add mutex-based fallback for non-x86 architectures
- [ ] Support dynamic range bounds instead of compile-time constants only
- [ ] Fix printf() in parallel loops via thread-safe wrapper
- [ ] Create parallel_sum.flap example demonstrating 10k item reduction
- [ ] Create parallel_mandelbrot.flap example with pixel-level parallelism

## Atomic Operations Support
- [ ] Implement `atomic_add(ptr, value)` builtin using LOCK XADD instruction
- [ ] Implement `atomic_cas(ptr, old, new)` builtin using LOCK CMPXCHG
- [ ] Add `atomic_load(ptr)` and `atomic_store(ptr, value)` with memory barriers
- [ ] Create atomic_counter.flap test incrementing shared counter from 4 threads
- [ ] Document memory ordering guarantees in GRAMMAR.md

## Hot Reload Infrastructure
- [ ] Implement function extraction via ExtractFunctionCode() from running binary
- [ ] Set up mmap'd shared memory region for function pointer table
- [ ] Add Unix socket IPC for reload notifications
- [ ] Implement atomic pointer swaps with proper memory barriers
- [ ] Create hot_physics.flap demo changing gravity constant while game runs
- [ ] Add --watch flag to compiler for automatic recompilation on file changes

## Network Message Handling
- [ ] Parse sockaddr_in to extract sender IP address as string
- [ ] Extract sender port number from sockaddr_in structure
- [ ] Convert received byte buffer to proper Flap string type
- [ ] Populate `msg` and `from` variables in network receive loops
- [ ] Implement connection tracking hashmap with sender addresses
- [ ] Add last_seen timestamp tracking for each connection
- [ ] Implement 60-second timeout for stale connections
- [ ] Create network_chat.flap example with proper message handling

## Parser & Language Fixes
- [ ] Fix ambiguity in fork/spawn operator parsing with proper precedence
- [ ] Add clear error messages for common syntax mistakes
- [ ] Improve error reporting with line/column numbers
- [ ] Add recovery from parse errors to continue checking rest of file

## FFI Improvements
- [ ] Add support for C callback function pointers
- [ ] Implement basic C++ header parsing with name demangling
- [ ] Fix string argument passing to C functions (proper null termination)
- [ ] Add struct return value support for C functions
- [ ] Create sdl3_game.flap example with proper event handling

## Testing & Documentation
- [ ] Add integration test for 1000 messages/second network throughput
- [ ] Create comprehensive test for all arithmetic operations in parallel loops
- [ ] Write user guide for parallel programming patterns
- [ ] Document all error codes and recovery strategies
- [ ] Add troubleshooting section to README.md

## Platform Robustness
- [ ] Test and fix all syscalls on ARM64 Linux
- [ ] Verify RISC-V code generation for basic programs
- [ ] Add CI tests for all three architectures
- [ ] Create architecture-specific optimization guide

## Performance & Optimization
- [ ] Profile compiler and optimize slowest parsing paths
- [ ] Reduce memory allocations in hot paths
- [ ] Implement string interning for identifiers
- [ ] Add compilation cache for unchanged functions

---

## Future Releases (Post-1.3.0)

### v1.4.0 - Advanced Features
- ENet protocol implementation (reliable channels, packet ordering)
- Trampolines for deep recursion without stack overflow
- Let bindings for local recursive definitions
- Full Windows support with PE executable generation

### v1.5.0 - Language Extensions
- Macro system for compile-time metaprogramming
- CPS transformation for all function calls
- Python-style syntax with colons and indentation
- Pattern matching enhancements

### v2.0.0 - Major Evolution
- Full LLVM IR backend option
- Incremental compilation with dependency tracking
- Package manager integration
- Language server protocol support