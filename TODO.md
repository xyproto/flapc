# Flap Compiler TODO

## 🎯 Priority Tasks for "Flap 2075" (Next 50 years)

### Platform Support (see PLATFORM_ARCHITECTURE.md)
1. **x86_64 + Linux** ✅ 100% complete
2. **x86_64 + Windows** ✅ 100% complete (SDL3 working!)
3. **ARM64 + Linux** 🚧 90% complete
   - ✅ Linux syscalls (x8, svc #0)
   - ✅ C FFI integration (header parsing, function signatures)
   - ✅ `exitf`/`exitln` support
   - ✅ `or!` operator (railway-oriented error handling)
   - ⏳ defer statement (needed for SDL3 example)
   - ⏳ Dynamic linking / PLT/GOT stubs
4. **RISC-V64 + Linux** 🚧 80% complete, needs testing + validation
5. **ARM64 + macOS** ❌ Not started (will reuse ARM64 ISA code)
6. **ARM64 + Windows** ❌ Not started (will reuse ARM64 ISA code)

### Architecture Decision ✅
**Decision: Keep 3-block `unsafe` design (x86_64, arm64, riscv64)**
- ISA-based blocks, not target-based
- Compiler handles OS differences (binary format, calling conventions, syscalls)
- Scales well: new ISA = add block, new OS = reuse ISA code + add binary writer
- See PLATFORM_ARCHITECTURE.md for full analysis

### Core Language Features
1. **Module System** - Import system complete, needs ecosystem/testing
2. **Type System Refinement** - Core map[uint64]float64 model stable, C types integrated
3. **Standard Library** - Minimal runtime complete, needs expansion

## Import System - ✅ COMPLETED
The import system is now fully implemented with priority-based resolution:
1. Libraries (pkg-config, .dll, .so files) - highest priority
2. Git repositories (with version specifiers)
3. Local directories - lowest priority

Supported syntax:
- `import "sdl3" as sdl` (library)
- `import "github.com/user/repo" as repo` (git)
- `import "github.com/user/repo@v1.0.0" as repo` (git with version)
- `import "github.com/user/repo@latest" as repo` (git latest)
- `import "github.com/user/repo@main" as repo` (git branch)
- `import "git@github.com:user/repo.git" as repo` (SSH format)
- `import "." as local` (current directory)
- `import "./subdir" as sub` (relative path)
- `import "/absolute/path" as abs` (absolute path)

## Parser (parser.go) - 95% Complete
- Track column positions in lexer for better error messages
- Consider re-enabling blocks-as-arguments feature (currently conflicts with match expressions)
- Consider re-enabling struct literal syntax (currently conflicts with lambda match)

## Code Generation (codegen.go) - 90% Complete
- Add explicit cvtsd2ss/cvtss2sd if needed for precision
- Implement flap_cstr_to_string runtime function
- Implement string-to-cstring conversion (requires length parameter)
- Replace malloc with arena allocation in string operations
- Replace malloc with arena allocation in map operations
- Replace malloc with arena allocation in small string creation
- Integrate with or! error handling
- Optimize O(n²) algorithms
- Re-enable depth tracking (requires writable .bss/.data section support)
- Add proper stderr handling for error paths
- Handle "host:port" format in network operations
- Implement proper map iteration to extract string bytes
- Check for errors and convert buffer to string in network operations
- Implement proper transformation for type conversions
- Re-enable Optimizer when type is fully implemented

## Optimizer (optimizer.go) - 85% Complete
- Re-enable integer-only optimizations in integer contexts (unsafe blocks, explicit int types)

## Calling Conventions (calling_convention.go) - 90% Complete
- Implement ARM64 AAPCS calling convention when needed
- Implement RISC-V calling convention when needed

## Hot Reload (main.go) - 95% Complete
- Future enhancement: patch running process via IPC instead of restart

## ARM64 Backend - 85% Complete
### ✅ Completed
- Static ELF executable generation for Linux
- Exit syscall for static builds
- Basic code generation working

### 🚧 In Progress / TODO
- Implement ARM64-specific PLT/GOT stubs for dynamic linking
- Full testing and validation on ARM64 hardware
- ARM64 printf implementation (currently uses runtime helpers)
- ARM64 AAPCS calling convention refinements
- Test malloc/arena allocator on ARM64
- Test C FFI on ARM64
- Complete advanced ARM64 SIMD features

## RISC-V Backend - 80% Complete
- Complete advanced RISC-V features
- Full testing and validation

## Type System
- Continue refining the universal type system (map[uint64]float64)
- Ensure all C types integrate properly with Flap's core type representation

## PE Writer (pe.go) - 90% Complete
- Implement proper import table generation

## Printf (printf.go, printf_helper.go) - 90% Complete
- Calculate proper done offset in printf implementation
- Implement proper float-to-string conversion
- Implement ARM64-specific assembly for printf
- Implement RISC-V-specific assembly for printf
- Implement simplified float-to-string for common cases

## RISC-V Code Generation (riscv64_codegen.go) - 80% Complete
- Load actual address for rodata symbols (not just 0)
- Implement PC-relative load for rodata symbols

## RISC-V Instructions (riscv64_instructions.go) - 80% Complete
- Add CSR instructions

## Test Issues (several_tests.go)
- Fix the wrong printing of ² (superscript 2)
- Fix the "bare match clause must be the only entry in the block" issue

## Future Enhancements
- Additional platform support
- Performance optimizations
- More comprehensive standard library
