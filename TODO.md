# Flapc TODO - Implementation Status

## Currently Implemented Features ✅

- [x] Sequential range loops (`@ i in 0..<100`)
- [x] Sequential list loops (`@ x in myList`)
- [x] Parallel range loops (`@@ i in 0..<100`)
- [x] Break and continue statements (`ret @N`, `@N++`)
- [x] Parallel map operator (`list || lambda`) - sequential implementation
- [x] Atomic operations (atomic_add, atomic_cas, atomic_load, atomic_store)
- [x] Array/map printing in println
- [x] Basic C FFI via import
- [x] Match expressions
- [x] Lambda expressions
- [x] String operations
- [x] Arena allocators (parsed, runtime pending)
- [x] Defer statements (parsed, runtime pending)
- [x] Hot reload markers (parsed, runtime pending)

## Missing Core Features (From GRAMMAR.md)

### Loop Expressions (Priority: HIGH)
- [ ] Loop expressions returning values: `@ i in 0..<N { expr }`
  - Documented in GRAMMAR.md lines 26-29
  - Currently throws: "loop expressions not yet implemented as expressions"
  - Needed for: Functional programming patterns, result accumulation
  - Test case: `snakegame.flap` expects this to fail

### Parallel Loop Reducers (Priority: HIGH)
- [ ] Parallel reduction: `@@ i in 1..=10000 { i } | a,b | { a + b }`
  - Documented in GRAMMAR.md line 29
  - Currently throws: "parallel loop expressions with reducers not yet implemented"
  - Requires: Tree-reduction algorithm, thread synchronization
  - Test case: `parallel_sum.flap` expects this to fail
  - Use case: Parallel aggregations (sum, max, min, etc.)

### Parallel List Loops (Priority: MEDIUM)
- [ ] Parallel iteration over lists: `@@ x in myList { ... }`
  - Currently throws: "List iteration with parallel loops not yet implemented"
  - Requires: Dynamic work distribution strategy
  - Use case: Parallel data processing

### Dynamic Parallel Range Bounds (Priority: MEDIUM)
- [ ] Support variables in parallel ranges: `@@ i in start..<end { }`
  - Currently requires compile-time constant bounds
  - Requires: Runtime work distribution calculation
  - Use case: Data-dependent parallelism

### Map/List Operators (Priority: LOW)
- [ ] Map union operator for combining maps
  - Currently throws: "map union not yet implemented"
  - Line: parser.go:7878
  - Use case: Merging configuration, state combination

### FFI Enhancements (Priority: MEDIUM)
- [ ] String to C string conversion: `as string` from C char*
  - Currently throws: "conversion not yet implemented"
  - Line: parser.go:9223
  - Requires: flap_cstr_to_string runtime function

- [ ] C array to Flap list: `as list` conversion
  - Currently throws: "'as list' conversion not yet implemented"
  - Line: parser.go:9232
  - Requires: Length parameter support

- [ ] Spawn with pipe syntax: `spawn expr | x | { ... }`
  - Currently throws: "spawn with pipe syntax not yet implemented"
  - Line: parser.go:6282
  - Use case: Process communication, parallel result collection

### Networking Features (Priority: LOW)
- [ ] Send target parsing: host:port format
  - Currently throws: "send target format not yet supported"
  - Line: parser.go:15719
  - Use case: Network communication

- [ ] Proper string message sending/receiving
  - Currently sends placeholder "TEST" message
  - Line: parser.go:15771
  - Requires: Map iteration to extract string bytes

- [ ] Receive error handling and buffer conversion
  - Currently returns 0.0 placeholder
  - Line: parser.go:15975-15977
  - Use case: Network message handling

### Memory Operations (Priority: MEDIUM)
- [ ] Pointer append operator: `ptr ++ value as type`
  - Documented in GRAMMAR.md lines 172-173, 302-393
  - Current status: Unknown (need to check implementation)
  - Use case: Buffer building, binary protocols

- [ ] Multi-precision arithmetic: `+!` add-with-carry
  - Documented in GRAMMAR.md line 173
  - Current status: Not implemented
  - Use case: Arbitrary precision math

### Advanced Features (Documented but Not Implemented)

#### Hot Code Reload (Priority: LOW - Runtime)
- [ ] Function pointer table generation
- [ ] File watching (inotify/kqueue/FSEvents)
- [ ] Incremental recompilation
- [ ] Atomic pointer swaps
- Status: Parser recognizes `hot` keyword, full implementation pending

#### Arena Allocators (Priority: LOW - Runtime)
- [ ] Bump pointer allocation
- [ ] Auto-cleanup on block exit
- [ ] Nested arena support
- Status: Parser recognizes `arena` blocks, runtime implementation pending

#### Defer Statements (Priority: LOW - Runtime)
- [ ] LIFO cleanup execution
- [ ] Scope exit handling
- Status: Parser recognizes `defer` keyword, runtime implementation pending

#### CStruct System (Priority: LOW)
- [ ] C-compatible struct layouts
- [ ] Field offset calculation
- [ ] Packed and aligned modifiers
- [ ] sizeof() support
- Status: Documented in GRAMMAR.md (lines 395-485), implementation unknown

#### Module System (Priority: LOW)
- [ ] Git-based package imports
- [ ] Version specification (@v1.0.0, @latest, @HEAD)
- [ ] Private functions (_prefix convention)
- [ ] Cache management (--update-deps)
- Status: Basic C FFI works, Flap packages pending

## Code Quality & Polish

### Documentation
- [ ] Add memory ordering guarantees for atomic operations to GRAMMAR.md
- [ ] Document loop expression semantics and examples
- [ ] Document reducer syntax and parallel reduction algorithms
- [ ] Add troubleshooting section to README.md
- [ ] Update LANGUAGE.md with complete loop semantics

### Error Handling
- [ ] Improve error messages with line/column numbers
- [ ] Add parse error recovery
- [ ] Clear errors for common syntax mistakes
- [ ] Document all error codes and recovery strategies

### Testing
- [ ] Run test suite through valgrind for memory leaks
- [ ] Add bounds checking for array/memory access
- [ ] Validate pointer dereferences in generated assembly
- [ ] Test on fresh Ubuntu 22.04 VM
- [ ] Comprehensive atomic operations tests with multiple threads
- [ ] Edge case tests for atomic operations

### Performance
- [ ] Profile compiler and optimize slow parsing paths
- [ ] Reduce memory allocations in code generation
- [ ] Review and clean up generated assembly
- [ ] Ensure consistent code style across Go sources

### Platform Compatibility
- [ ] Verify x86-64 code generation on Arch and Ubuntu
- [ ] Test with different kernel versions (5.x, 6.x)
- [ ] Validate syscall numbers for target platforms
- [ ] Document minimal libc dependencies

## Future Releases

### v1.4.0 - Complete Loop & Parallel Features
- Implement loop expressions returning values
- Implement parallel loop reducers with tree reduction
- Support dynamic range bounds in parallel loops
- Parallel list loop support
- Thread-safe printf wrapper
- Make parallel map operator (`||`) actually parallel

### v1.5.0 - Memory & Resource Management
- Complete arena allocator runtime
- Complete defer statement runtime
- Pointer append operator (`++`)
- Multi-precision arithmetic (`+!`)
- Struct return values from C functions

### v1.6.0 - Advanced FFI & Networking
- C callback function pointers
- C++ header parsing with demangling
- String conversions (C ↔ Flap)
- Network message handling (send/receive)
- Connection tracking with timeouts
- SDL3 game example with full event handling

### v2.0.0 - Hot Reload & Advanced Features
- Hot reload runtime implementation
- Function pointer table and swapping
- File watching infrastructure
- Incremental compilation
- Trampolines for deep recursion
- Let bindings for local recursive definitions
- Macro system for metaprogramming
- Windows PE executable support

### v3.0.0 - Tooling & Ecosystem
- LLVM IR backend option
- Incremental compilation with dependency tracking
- Package manager integration
- Language server protocol (LSP)
- Watch mode (--watch) for auto-recompilation
- Debugger integration improvements
