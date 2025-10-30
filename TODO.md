# Flapc v1.0 TODO - Production Ready for Steam Games

## Goal

Create a **production-ready compiler** suitable for commercial game development on Steam. Focus on reliability, debuggability, and C interoperability rather than experimental features.

## Key Design Decisions

1. **Remove parallel loops** - Complex, fragile, rarely needed for games
2. **Add register allocator** - Reduce mov instructions, improve performance
3. **Add struct types** - Essential for game entities and components
4. **Focus on C FFI** - SDL3, OpenGL, Vulkan integration
5. **DWARF debug info** - Full gdb/lldb support
6. **No memory leaks** - Valgrind clean compiler and generated code

## Phase 1: Cleanup & Simplification (Week 1-2)

### Remove Complex/Unfinished Features
- [ ] Remove parallel loop code (`@@` syntax)
  - parser.go:6573-6971 (compileParallelRangeLoop)
  - Delete barrier synchronization code
  - Remove futex syscalls
  - Update tests to remove `@@` examples

- [ ] Remove parallel map operator (`||`)
  - parser.go:9725-9854 (compileParallelExpr)
  - Was sequential anyway, misleading name
  - Update GRAMMAR.md

- [ ] Remove loop expressions
  - parser.go:8954-8967 (LoopExpr case)
  - Remove from AST
  - Update test expectations

- [ ] Remove arena/defer/hot keywords
  - These were never fully implemented
  - Remove from lexer and parser
  - Clean up AST types

- [ ] Remove networking primitives
  - Send/receive incomplete
  - Port literals unused
  - Games can use SDL_net via FFI

- [ ] Simplify expression precedence
  - Remove `|||` (concurrent gather)
  - Remove `||` (parallel)
  - Remove `|` pipe in expression context (keep for match)
  - Remove `or!` error handling
  - Remove `<==` send operator

###  Update Grammar and Lexer
- [ ] Update GRAMMAR.md → use GRAMMAR_V1.md
- [ ] Remove unused tokens from lexer
- [ ] Simplify keyword list
- [ ] Update reserved words

## Phase 2: Core Features (Week 2-3)

### Implement Register Allocator
- [ ] Design register allocation strategy
  - Use graph coloring algorithm
  - Track register liveness
  - Spill to stack when needed
  - Prioritize frequently used variables

- [ ] Create register allocator module
  - `register_allocator.go`
  - Liveness analysis pass
  - Interference graph construction
  - Allocation with spilling

- [ ] Integrate with code generation
  - Replace ad-hoc register usage
  - Generate optimal mov instructions
  - Reduce stack traffic

- [ ] Test and validate
  - Compare code size before/after
  - Benchmark compilation speed
  - Verify correctness with existing tests

### Implement Struct Types
- [ ] Add struct definition parsing
  - `Player :: struct { x: f64, y: f64 }`
  - Field types and layout
  - Nested structs

- [ ] Add struct literal parsing
  - `Player { x: 10, y: 20 }`
  - Field initialization
  - Default values

- [ ] Implement struct code generation
  - Memory layout calculation
  - Field offset computation
  - Alignment handling

- [ ] Add field access (dot notation)
  - `player.x`, `player.y`
  - Nested field access
  - Assignment to fields

- [ ] C struct interop
  - Match C struct layout
  - Pass structs to C functions
  - Receive structs from C

### Improve C FFI
- [ ] String conversions
  - Flap string → C char* (malloc + copy)
  - C char* → Flap string (strlen + copy)
  - Automatic cleanup tracking

- [ ] Callback functions
  - Generate C-compatible function pointers
  - Handle closure environments
  - Event handlers for SDL

- [ ] Variadic function support
  - Handle printf-style functions
  - Type checking for format strings

- [ ] Better type safety
  - Detect type mismatches at compile time
  - Warn on dangerous casts
  - Prevent common segfaults

## Phase 3: Code Quality & Debugging (Week 3-4)

### Generate DWARF Debug Info
- [ ] Add DWARF v4 generation
  - Line number table (.debug_line)
  - Variable info (.debug_info)
  - Frame info (.debug_frame)

- [ ] Source location tracking
  - Map assembly to source lines
  - Track variable scopes
  - Function boundaries

- [ ] Test with gdb/lldb
  - Breakpoints by line number
  - Variable inspection
  - Stack traces
  - Step through code

### Improve Error Messages
- [ ] Add source context to errors
  - Show line with error
  - Underline problematic code
  - Suggest fixes

- [ ] Better type error messages
  - "Expected i32, got f64"
  - Show where types come from
  - Suggest conversions

- [ ] Parse error recovery
  - Continue parsing after error
  - Show multiple errors at once
  - Don't cascade errors

### Stack Management
- [ ] Fix stack alignment issues
  - Ensure 16-byte alignment
  - Track stack depth properly
  - Handle nested calls correctly

- [ ] Add stack overflow detection (debug mode)
  - Check stack pointer bounds
  - Emit guard code
  - Clear error messages

- [ ] Optimize stack usage
  - Reuse stack slots
  - Minimize push/pop
  - Better temporary management

### Memory Safety
- [ ] Add bounds checking (debug mode)
  - Array index validation
  - Pointer dereference checks
  - Buffer overflow detection

- [ ] Null pointer checks (debug mode)
  - Check before dereference
  - Clear error messages
  - Stack trace on failure

- [ ] Memory leak detection
  - Track malloc/free pairs
  - Warn on unfreed memory
  - Integration with valgrind

## Phase 4: Testing & Validation (Week 4-5)

### Comprehensive Test Suite
- [ ] Core language tests
  - Variables and assignment
  - Functions and lambdas
  - Control flow
  - Loops
  - Structs

- [ ] Type system tests
  - All primitive types
  - Type conversions
  - Struct types
  - Arrays and maps

- [ ] C FFI tests
  - Function calls
  - Struct passing
  - String conversion
  - Callbacks

- [ ] Edge cases
  - Empty arrays/maps
  - Deeply nested structures
  - Large numbers
  - Long strings

- [ ] Error handling tests
  - Invalid syntax
  - Type errors
  - Undefined variables
  - Null pointers

### Performance Testing
- [ ] Benchmark compilation speed
  - Small programs (< 100 lines)
  - Medium programs (< 1000 lines)
  - Large programs (< 10000 lines)
  - Target: < 1s for typical game

- [ ] Benchmark runtime performance
  - Tight loops
  - Function calls
  - Array access
  - Struct access
  - Target: Within 10% of C

- [ ] Memory usage
  - Compiler memory usage
  - Generated code size
  - Runtime memory overhead
  - No memory leaks (valgrind)

### Example Games
- [ ] Pong (simple)
  - Basic SDL3 rendering
  - Input handling
  - Game state
  - Collision detection

- [ ] Platformer (medium)
  - Sprite rendering
  - Physics simulation
  - Multiple entities
  - Level loading

- [ ] Top-down shooter (complex)
  - Entity component system
  - Particle effects
  - Audio integration
  - Performance at 60 FPS

### Validation Checklist
- [ ] Compiles cleanly (no warnings)
- [ ] All tests pass
- [ ] No memory leaks (valgrind)
- [ ] No undefined behavior (ubsan)
- [ ] Works with gdb/lldb
- [ ] Runs at 60 FPS for typical game
- [ ] Compiles in < 1s for typical game
- [ ] Clear error messages
- [ ] Example games work

## Phase 5: Documentation & Polish (Week 5-6)

### Update Documentation
- [ ] Replace GRAMMAR.md with GRAMMAR_V1.md
- [ ] Update README.md
  - Clear feature list
  - Installation instructions
  - Quick start guide
  - Example programs
  - C FFI tutorial

- [ ] Create LEARNINGS.md
  - Design decisions and rationale
  - What worked well
  - What didn't work
  - Lessons for future versions
  - Advice for similar projects

- [ ] API documentation
  - Standard library functions
  - Built-in functions
  - Memory operations
  - Type conversions

- [ ] Tutorial series
  - Hello World
  - Basic game loop
  - SDL3 integration
  - Struct usage
  - C FFI guide

### Code Cleanup
- [ ] Remove dead code
  - Unused functions
  - Commented-out sections
  - Debug prints

- [ ] Consistent style
  - gofmt all files
  - Consistent naming
  - Clear comments
  - Function documentation

- [ ] Refactoring
  - Extract common patterns
  - Simplify complex functions
  - Improve readability
  - Better error handling

### Release Preparation
- [ ] Version tagging
  - Tag v1.0.0 release
  - Create GitHub release
  - Write release notes

- [ ] Binary releases
  - Linux x86-64 binary
  - Build instructions
  - Dependencies list

- [ ] Examples repository
  - Separate repo for game examples
  - Pong, platformer, shooter
  - Documented and commented
  - Build instructions

## Success Criteria for v1.0

A game built with Flapc v1.0 can be published on Steam if it:

- ✅ Compiles to native ELF executable
- ✅ Links with SDL3/OpenGL seamlessly
- ✅ Runs at stable 60 FPS
- ✅ Has no memory leaks (valgrind clean)
- ✅ Has no segfaults with proper error handling
- ✅ Compiles in < 1 second for iteration
- ✅ Supports full debugging with gdb/lldb
- ✅ Has clear, actionable error messages
- ✅ Handles structs for game entities
- ✅ Integrates with C libraries effortlessly

## Beyond v1.0 (Future Versions)

### v1.1 - Windows Support
- PE executable generation
- DirectX/Vulkan integration
- Visual Studio debugging

### v1.2 - ARM64 Support
- ARM64 code generation
- Raspberry Pi support
- Mobile potential

### v2.0 - Advanced Features
- Loop expressions (if proven necessary)
- Hot code reload (runtime implementation)
- Arena allocators (runtime implementation)
- Advanced optimization passes

### v3.0 - Tooling
- Language server protocol
- Package manager
- LLVM backend option
- IDE integration

## Current Status

**In Progress:**
- [x] Requirements analysis (STEAM_REQUIREMENTS.md)
- [x] Grammar v1.0 design (GRAMMAR_V1.md)
- [ ] TODO.md update (this file)

**Next Steps:**
1. Remove parallel loop code
2. Implement register allocator
3. Add struct types
4. Generate DWARF debug info
5. Comprehensive testing

**Timeline:**
- Week 1-2: Cleanup and simplification
- Week 2-3: Core features (registers, structs, FFI)
- Week 3-4: Quality and debugging
- Week 4-5: Testing and validation
- Week 5-6: Documentation and release

**Target Release:** 6 weeks from now
