# Flapc TODO

Actionable items sorted by importance for **game development with SDL3 and RayLib5**.
Current focus: x86-64 Linux. Multiplatform support (Windows/macOS/ARM64/RISC-V) deferred.
End goal: Publishable Steam games with Steamworks integration.

Complexity: (LOW/MEDIUM/HIGH/VERY HIGH)

## Critical - Game Development

1. **Implement hot code reload** (HIGH) 🎮 GAMEDEV
   - Goal: Enable rapid iteration for game development - reload code without restarting
   - Use case: Tweak physics parameters, adjust visual effects, modify AI behavior in real-time
   - Current status: `hot` keyword foundation complete (lexer, parser, hotFunctions map)

   **Phase 1: Function Pointer Table & Indirection** (MEDIUM)
   - Add function pointer table to rodata segment
   - Generate pointers for all hot functions at compile time
   - Modify hot function calls to use indirect jumps through table
   - Performance cost: ~1-2 CPU cycles per hot function call (acceptable)
   - Files: `parser.go` (codegen for hot calls), `elf_builder.go` (pointer table)

   **Phase 2: File Watching** (MEDIUM)
   - Implement inotify-based file watcher for Linux
   - Watch all .flap source files loaded during compilation
   - Debounce file changes (500ms delay to batch rapid edits)
   - Trigger recompilation on file modification
   - Files: New `filewatcher.go`, integration in `main.go`

   **Phase 3: Incremental Recompilation** (HIGH)
   - Parse only changed .flap files
   - Extract hot functions from changed files
   - Generate machine code for hot functions only
   - Preserve non-hot functions and program data
   - Files: `parser.go` (incremental mode), new `incremental.go`

   **Phase 4: Code Injection** (HIGH)
   - Allocate executable memory pages with mmap(PROT_READ|PROT_WRITE|PROT_EXEC)
   - Copy new hot function machine code to allocated pages
   - Atomically update function pointer table (single 8-byte write)
   - Add 1-second grace period before munmap of old code (prevent crashes)
   - Files: New `hotreload.go` (mmap/munmap/pointer swap logic)

   **Phase 5: Developer Experience** (LOW)
   - Add `--watch` flag to compiler (enables hot reload mode)
   - Add USR1 signal handler for manual reload trigger
   - Print reload notifications to stderr with timestamps
   - Add configuration file support to save/restore game state
   - Files: `main.go` (flag handling), `config.go` (state persistence)

   **Constraints & Safety**
   - Cannot change function signatures (parameter count/types)
   - Cannot change struct layouts (breaks memory compatibility)
   - Cannot add/remove global variables
   - If recompilation fails, keep old code running (no crashes)
   - Hot reload only works for functions marked with `hot` keyword

   **Testing Strategy**
   - Test: Hot reload simple function (e.g., physics gravity constant)
   - Test: Hot reload lambda with closure (verify env preserved)
   - Test: Rapid file changes (verify debouncing works)
   - Test: Compilation error recovery (old code keeps running)
   - Test: Signal-based manual reload (kill -USR1)

   Files: `hotreload.go`, `filewatcher.go`, `incremental.go`, `parser.go`, `main.go`

2. **Add Steamworks FFI support** (HIGH)
   - Goal: Steam achievements, leaderboards, cloud saves for commercial releases
   - Approach: Extend C FFI to handle Steamworks SDK callbacks and structs
   - Requirements: Handle C++ name mangling, callback function pointers
   - Impact: Required for publishing on Steam
   - Files: New `steamworks.go`, extend FFI in `parser.go`

## Fundamental - Language Features

3. **Add trampoline execution for deep recursion** (MEDIUM)
   - Current: Non-tail-recursive functions can stack overflow
   - Goal: Handle deep recursion without TCO (e.g., tree traversal, Ackermann)
   - Approach: Return thunk (suspended computation), evaluate iteratively
   - Benefits: Enable fibonacci, tree algorithms without stack limits
   - Files: New `trampoline.go`, modify lambda compilation in `parser.go`

4. **Add let bindings for local scope** (MEDIUM)
   - Syntax: `let rec loop = (n, acc) => if n == 0 { acc } else { loop(n-1, acc*n) }`
   - Benefits: StandardML/OCaml-style local recursive definitions
   - Common functional pattern for loop-like constructs
   - Files: `parser.go`, new LetExpr AST node in `ast.go`

5. **Add Python-style colon + indentation** (MEDIUM)
   - Opt-in alternative to braces: `if x > 0:\n    print(x)`
   - Enables: Python/GDScript-like syntax for language packs
   - Approach: Track indentation levels in lexer, emit virtual braces
   - Files: `lexer.go` (indentation tracking), `parser.go` (virtual brace handling)

## Advanced - Optimization

6. **Add CPS (Continuation-Passing Style) transform** (VERY HIGH)
   - Goal: Convert all calls to tail calls internally
   - Benefits: Advanced control flow, no stack growth, efficient coroutines
   - Approach: Transform AST before code generation
   - Example: `f() + g()` → `f((r1) => g((r2) => r1 + r2))`
   - Note: Optional optimization pass, no IR needed
   - Files: New `cps.go`, integrate into compilation pipeline in `parser.go`

## Nice to Have

7. **Add tail call validation in debug mode** (LOW)
   - Warn if tail recursion incorrectly used (e.g., in non-tail position)
   - Helps developers understand when TCO will apply
   - Files: Add validation pass in `parser.go`

8. **Add macro system** (VERY HIGH)
   - Pattern-based code transformation at parse time
   - Enables advanced language packs and metaprogramming
   - Example: `macro when(cond, body) => if cond { body }`
   - Files: New `macro.go`, extend parser

9. **Add custom infix operators** (HIGH)
   - For language packs (e.g., Python's `**` for exponentiation)
   - Requires: Precedence table, associativity rules
   - Files: Extend `parser.go` precedence handling

10. **Add multi-precision arithmetic operators** (MEDIUM)
    - `++` (add with carry) for big integer implementations
    - `<->` (swap/exchange) for in-place algorithms
    - Files: Extend `parser.go` operator handling

## Multiplatform Support (Deferred)

Focus on x86-64 Linux first. These are lower priority.

11. **Add Windows x64 code generation** (HIGH)
    - Goal: Compile to PE/COFF executables for Windows x64
    - Calling convention: Microsoft x64 (rcx, rdx, r8, r9 for first 4 args)
    - Binary format: PE32+ (Portable Executable)
    - Impact: Required for Steam Windows builds
    - Files: New `pe_builder.go`, `codegen_windows.go`

12. **Add Windows ARM64 code generation** (HIGH)
    - Goal: Support Windows on ARM (Surface, future gaming devices)
    - Calling convention: Microsoft ARM64
    - Binary format: PE32+ ARM64
    - Impact: Future-proofing for Windows ARM gaming PCs
    - Files: Extend ARM64 codegen with Windows support

13. **Fix macOS ARM64 runtime issues** (HIGH)
    - Current: Binaries hang before entering main()
    - Problem: dyld/code signing/entitlements
    - Impact: Blocks macOS game distribution
    - Approach: Debug Mach-O generation, test codesigning
    - Files: `macho_builder.go`, startup code

14. **Complete Linux ARM64 support** (MEDIUM)
    - Goal: Raspberry Pi 4+ and Linux ARM gaming devices
    - Current: Basic ARM64 codegen exists
    - Need: Full ELF generation and runtime testing
    - Impact: Enables embedded/portable Linux gaming
    - Files: `arm64.go`, `elf_builder.go`

15. **Complete RISC-V 64-bit support** (LOW)
    - Goal: Future RISC-V gaming handhelds
    - Current: Instruction encoders ready
    - Need: Full codegen implementation
    - Impact: Future-proofing for RISC-V gaming
    - Files: `riscv64.go`
