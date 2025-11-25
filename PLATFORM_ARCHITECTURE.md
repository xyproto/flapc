# Platform Architecture Analysis & Recommendations

## Current State (November 2025)

### Supported Targets
Currently Flapc supports:
- **x86_64 + Linux** ✅ (100% complete)
- **x86_64 + Windows** ✅ (100% complete, SDL3 working)
- **ARM64 + Linux** 🚧 (90% complete, needs defer + dynamic linking)
- **ARM64 + macOS** ❌ (not started)
- **ARM64 + Windows** ❌ (not started)
- **RISCV64 + Linux** 🚧 (80% complete, needs testing)

### Current `unsafe` Block Design
```flap
unsafe [type] { x86_64 block } { arm64 block } { riscv64 block }
```

Example:
```flap
result: int64 = unsafe int64 { 
    rax <- 42    # x86_64 block
} { 
    x0 <- 42     # arm64 block
} { 
    a0 <- 42     # riscv64 block
}
```

## Analysis

### Current Architecture
- **58 architecture-specific files** spread across the codebase
- Files organized by ISA (x86_64, arm64, riscv64) not by target (arch+OS)
- `unsafe` blocks have 3 variants (one per ISA)
- Code generation uses `Platform{Arch, OS}` internally
- Binary format (ELF/PE/Mach-O) determined by OS, not arch

### Key Insight: ISA vs Target
The current design **correctly separates ISA from OS**:
- **ISA (x86_64, arm64, riscv64)** → determines registers, instructions
- **OS (Linux, Windows, macOS)** → determines binary format, syscalls, calling convention
- **Target = ISA + OS** → determines full compilation strategy

### Syscalls vs C FFI
- **Syscalls**: ISA-dependent (different registers, numbers)
  - Linux x86_64: `syscall` instruction
  - Linux arm64: `svc #0` instruction
  - Windows: C FFI to kernel32.dll (no syscall instruction)
  
- **C FFI**: Calling convention varies by target
  - x86_64 Linux: System V ABI
  - x86_64 Windows: Microsoft x64 calling convention
  - arm64 Linux/macOS: AAPCS64
  - arm64 Windows: ARM64 Windows ABI

## Recommendation: Keep 3-Block `unsafe` Design

### Rationale

**✅ KEEP 3 BLOCKS (ISA-based)**
```flap
unsafe [type] { x86_64 } { arm64 } { riscv64 }
```

**Advantages:**
1. **Simpler user model**: Users think in terms of CPU architecture
2. **Less redundancy**: Don't repeat code for arm64-linux vs arm64-macos
3. **Clear separation**: Assembly is ISA-specific, not OS-specific
4. **Already working**: x86_64+Linux and x86_64+Windows share x86_64 block

**How it works:**
- User writes assembly for the ISA (registers, instructions)
- Compiler handles OS differences:
  - Binary format (ELF/PE/Mach-O)
  - Calling conventions
  - Syscall vs C FFI
  - Dynamic linking (PLT/GOT vs IAT)

### Alternative Rejected: 6-Block Design

**❌ DON'T DO 6 BLOCKS (target-based)**
```flap
unsafe [type] { 
    x86_64-linux 
} { 
    x86_64-windows 
} { 
    arm64-linux 
} { 
    arm64-macos 
} { 
    arm64-windows 
} { 
    riscv64-linux 
}
```

**Disadvantages:**
1. **Verbose**: 6 blocks for every unsafe expression
2. **Redundant**: arm64-linux and arm64-macos would be nearly identical
3. **Confusing**: Users must know OS calling conventions
4. **Maintenance burden**: More blocks = more test cases

## Implementation Strategy

### File Organization (Current - GOOD)

Keep the current structure:
```
# ISA backends
x86_64_codegen.go       # x86_64 instruction generation
arm64_codegen.go        # arm64 instruction generation
riscv64_codegen.go      # riscv64 instruction generation

# OS-specific binary writers
codegen_elf_writer.go   # ELF (Linux, *BSD)
codegen_pe_writer.go    # PE (Windows)
codegen_macho_writer.go # Mach-O (macOS)

# Platform integration
target.go               # Target{Arch, OS} management
backend.go              # Backend interface
calling_convention.go   # ABI/calling convention logic
```

### Syscall Abstraction

The compiler should abstract syscalls:
```flap
# User code - portable
result = syscall(1, msg_ptr, msg_len)  # write syscall

# Compiler generates:
# - x86_64 Linux: rax=1, rdi=fd, rsi=ptr, rdx=len, syscall
# - arm64 Linux: x8=64, x0=fd, x1=ptr, x2=len, svc #0
# - Windows (any arch): call WriteFile via C FFI
```

### C FFI Abstraction

The compiler handles calling conventions:
```flap
# User code - portable
result: cint = sdl.SDL_Init(flags)

# Compiler generates based on target:
# - x86_64 Linux: System V ABI (rdi, rsi, rdx, rcx, r8, r9)
# - x86_64 Windows: Microsoft x64 (rcx, rdx, r8, r9)
# - arm64 Linux: AAPCS64 (x0-x7)
# - arm64 macOS: Same as arm64 Linux
# - arm64 Windows: ARM64 Windows (x0-x7, different struct passing)
```

## Roadmap for Complete Platform Support

### Phase 1: Complete Current Targets ✅
- [x] x86_64 + Linux
- [x] x86_64 + Windows
- [ ] arm64 + Linux (defer, dynamic linking)
- [ ] riscv64 + Linux (testing)

### Phase 2: ARM64 Expansion 🎯
- [ ] arm64 + macOS (Mach-O, Apple Silicon)
- [ ] arm64 + Windows (PE, Surface devices)

### Phase 3: Testing Infrastructure
- [ ] Test matrix: all targets × all features
- [ ] Cross-compilation validation
- [ ] Binary compatibility checks

## Conclusion

**Decision: Keep 3-block `unsafe` design (x86_64, arm64, riscv64)**

The current architecture is sound. The separation of ISA-specific code generation
from OS-specific binary format/syscall handling is the right design.

### Action Items
1. ✅ Keep current `unsafe` syntax unchanged
2. ✅ Keep current file organization unchanged
3. 🎯 Complete arm64+Linux support (defer, dynamic linking)
4. 🎯 Add arm64+macOS support (new target, reuse arm64 ISA code)
5. 🎯 Add arm64+Windows support (new target, reuse arm64 ISA code)
6. 📝 Document platform differences in LANGUAGESPEC.md

### For "Flap 2075" (50-year support)

The 3-block design scales well:
- New ISA? Add 4th block to `unsafe`
- New OS? Reuse existing ISA blocks, add binary writer
- New ISA+OS? Both of the above

Example future state:
```flap
# If we add wasm32 in 2030:
unsafe { x86_64 } { arm64 } { riscv64 } { wasm32 }

# arm64-windows just works with existing arm64 block:
./flapc --target=arm64-windows program.flap
```

This architecture will serve Flap well for the next 50 years.
