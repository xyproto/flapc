# Flapc TODO

Actionable items sorted by importance and fundamentality for **game development with SDL3 and RayLib5**.
Current focus: x86-64 Linux. Multiplatform support (Windows/macOS/ARM64/RISC-V) deferred.
End goal: Publishable Steam games with Steamworks integration.

Complexity: (LOW/MEDIUM/HIGH/VERY HIGH)

## Recent Progress

**Unsafe Block Syntax Simplification** (2025-10-25)
- Removed ret keyword from unsafe blocks - simplified to implicit return
- All tests passing: 306/306 (100%)
- Fixed lambda/map/pipe operator issues that were causing runtime crashes
- Removed obsolete unsafe_ret_cstr_test and unsafe_return_test files

**Test Framework Enhancement** (2025-10-24)
- Added wildcard pattern matching for non-deterministic test output
- Supports `*` wildcard for matching memory addresses, PIDs, timestamps
- Correctly handles ASCII art (doesn't treat `         *` as wildcard)
- Updated 6 test result files with wildcard patterns

## Critical - Compiler Correctness

1. **Fix advanced lambda/map features** (HIGH) ✅ COMPLETE
   - Status: FIXED - All tests passing (306/306)
   - Previously failing tests now pass: test_lambda_match, test_map_simple, ex2_list_operations
   - Fixed issues with map operations, lambda match expressions, and pipe operator

2. **Implement proper closure capture for nested lambdas** (HIGH) ⚠️ PARTIAL
   - Status: 2-level closures work perfectly, 3+ levels need environment chaining
   - Improved: Added nested lambda support to collectCapturedVarsExpr
   - Working: `(a) => (b) => a + b` fully functional
   - Limitation: `(a) => (b) => (c) => a * b + c` captures detected but runtime env chain needed
   - Files: `parser.go:collectCapturedVarsExpr`, lambda compilation

## Critical - Game Development FFI

3. **Enhance unsafe blocks to return register values** (MEDIUM) ✅ COMPLETE
   - Status: ALREADY IMPLEMENTED - unsafe blocks return register values
   - Syntax: `result := unsafe { rax <- 42; rax }` or implicit return
   - Supports: Explicit returns, implicit rax/xmm0 return, computations
   - Files: `parser.go:parseUnsafeExpr`, unsafe codegen

4. **Extend C FFI for SDL3/RayLib5** (HIGH)
   - Current: Only 6 integer arguments, no floats/pointers/structs
   - Need: Float arguments (colors, positions), pointer arguments (structs)
   - Goal: Full SDL3 and RayLib5 API access
   - Impact: Blocks 90% of game development functions
   - Files: `parser.go` FFI handling, ABI conversion

5. **Add Steamworks FFI support** (HIGH)
   - Goal: Steam achievements, leaderboards, cloud saves
   - Approach: C FFI with proper struct/callback handling
   - Impact: Required for commercial Steam releases
   - Files: New `steamworks.go`, FFI extensions

6. **Implement hot code reload** (HIGH) 🎮 GAMEDEV
   - Goal: Recompile and hot-swap functions without restarting executable
   - Method: Self-modifying code via function pointer table + mmap
   - Subtasks:
     a. Add `hot` keyword to lexer/parser (mark hot-swappable functions)
     b. Generate function pointer table for hot functions
     c. Compile hot functions with indirection (call through table)
     d. Implement file watching (inotify/kqueue/FSEvents)
     e. Add incremental recompilation (parse + codegen changed functions only)
     f. Implement mmap(PROT_EXEC) for new code pages
     g. Add atomic pointer swap (update function table)
     h. Add grace period for old code pages (1 second delay before munmap)
     i. Add --watch flag to compiler
     j. Add USR1 signal handler for manual reload trigger
   - Performance: ~1-2 cycle overhead per hot function call (indirect branch)
   - Limitations: Cannot change function signatures or struct layouts
   - Impact: Essential for rapid gamedev iteration (50ms reload vs minutes)
   - Benefits: Fix bugs while game runs, tune visuals in real-time
   - Files: `lexer.go`, `parser.go`, new `hotreload.go`, `filewatcher.go`

7. **Implement cstruct keyword** (HIGH) 🎮 GAMEDEV
   - Goal: Define C-compatible structs with explicit layout
   - Syntax: `cstruct Vec3 { x: f32, y: f32, z: f32 }`
   - Modifiers: `packed` (no padding), `aligned(N)` (alignment)
   - Features:
     a. Calculate struct size and field offsets at compile time
     b. Generate StructName.field_offset constants
     c. Generate sizeof(StructName) constant
     d. Support nested structs and pointers
     e. Support C types (i8/i16/i32/i64/u8/u16/u32/u64/f32/f64/cstr/ptr)
   - Impact: Required for SDL3, RayLib5, physics engines
   - Benefits: Exact C struct layout, no surprises, compiler-calculated offsets
   - Files: `lexer.go`, `parser.go`, new `cstruct.go`

## Fundamental - Enable Core Patterns

6. **Implement automatic tail-call optimization** (HIGH)
   - Goal: Detect tail-recursive calls and convert to loops automatically
   - Status: Documented in LANGUAGE.md, not yet implemented
   - Approach: Analyze function to detect tail position calls, emit loop instead of call
   - Benefits: No stack growth for tail recursion, no special keywords needed
   - Files: `parser.go:compileLambdaCall`, add tail-call detection

7. **Implement automatic memoization for pure functions** (MEDIUM)
   - Goal: Automatically cache results of pure recursive functions
   - Status: Documented in LANGUAGE.md, not yet implemented
   - Approach: Detect purity (no side effects), add result cache with function arguments as key
   - Benefits: Efficient fibonacci, dynamic programming, no manual caching
   - Note: Uses arena-based memory allocation for cache
   - Files: `parser.go:compileLambdaCall`, add purity analysis and cache logic

8. **Add trampoline execution for deep recursion** (MEDIUM)
    - Current: Non-tail-recursive functions can stack overflow
    - Goal: Handle deep recursion without TCO (e.g., tree traversal)
    - Approach: Return thunk (suspended computation), evaluate iteratively
    - Benefits: Fibonacci, tree recursion without stack limits
    - Files: New `trampoline.go`, modify lambda returns

9. **Implement precalculated stack frames** (MEDIUM)
    - Current: Dynamic stack allocation causes tracking bugs
    - Goal: Allocate entire frame at function entry (C/C++ style)
    - Benefits: No stack tracking bugs, predictable layout, easier debug
    - Approach: Calculate frame size in collectSymbols, allocate once
    - Files: `parser.go:Compile`, `parser.go:collectSymbols`

## Advanced - Optimization

10. **Add CPS (Continuation-Passing Style) transform** (VERY HIGH)
    - Goal: Convert all calls to tail calls internally
    - Benefits: Advanced control flow, no stack growth
    - Approach: Transform AST before code generation
    - Example: `f() + g()` → `f((r1) => g((r2) => r1 + r2))`
    - Note: Optional optimization pass, no IR needed
    - Files: New `cps.go`, modify compilation pipeline

11. Whole program optimization.

12. Loop unrolling and other optimization tricks.

## Language Features

13. **Add alias keyword for language packs** (MEDIUM)
    - Syntax: `alias for=@`, `alias break=@-`, `alias continue=@=`
    - Enables: python.flap, gdscript.flap style packs
    - Files: `lexer.go`, `parser.go`, new alias map

14. **Add Python-style colon + indentation** (MEDIUM)
    - Opt-in alternative to braces: `if x > 0:\n    print(x)`
    - Enables: Python/GDScript-like syntax
    - Files: `lexer.go` (indentation tracking), `parser.go`

15. **Add pattern matching on function parameters** (HIGH)
    - Syntax: `factorial := (0) => 1 | (n) => n * factorial(n-1)`
    - StandardML-style elegance
    - Files: `parser.go` lambda parsing, new pattern match system

16. **Add let bindings for local scope** (MEDIUM)
    - Syntax: `let rec loop = (n, acc) => ...`
    - Common functional pattern
    - Files: `parser.go`, new LetExpr type

17. **Extend `inf` keyword for other contexts** (LOW-MEDIUM)
    - Current: Only used for `max inf` in loops
    - Proposed uses:
      a. **Numeric constant**: `x := inf` for IEEE 754 infinity
      b. **Unbounded ranges**: `@ i in 0..<inf max inf { }` for infinite sequences
      c. **Timeout values**: `wait(inf)` to wait indefinitely
      d. **Comparisons**: `x < inf` always true for finite x
      e. **Math operations**: `1 / inf` → 0
    - Design: Map `inf` to `math.Inf(1)` in numeric contexts
    - Impact: Cleaner syntax than using large numbers or special functions
    - Files: `lexer.go`, `parser.go:parseExpression`

## Nice to Have

18. **Add tail call validation in debug mode** (LOW)
    - Warn if `~tailcall>` used incorrectly
    - Helps developers write correct code

19. **Add approximate equality operator** (LOW)
    - Syntax: `0.3 =0.1= 0.2`
    - Useful for floating-point game physics comparisons

20. **Add macro system** (VERY HIGH)
    - Pattern-based code transformation
    - Enables advanced language packs

21. **Add custom infix operators** (HIGH)
    - For language packs (e.g., Python's `**`)
    - Requires precedence handling

22. **Add multi-precision arithmetic operators** (MEDIUM)
    - `++` (add with carry)
    - `<->` (swap/exchange)

## Multiplatform Support (Deferred)

These items are lower priority. Focus on x86-64 Linux first.

23. **Add Windows x64 code generation** (HIGH)
    - Goal: Compile to PE/COFF executables for Windows x64
    - Calling convention: Microsoft x64 (different from System V)
    - Binary format: PE32+ (Portable Executable)
    - Impact: Required for Steam Windows builds
    - Files: New `pe_builder.go`, `codegen_windows.go`

24. **Add Windows ARM64 code generation** (HIGH)
    - Goal: Support Windows on ARM (Surface, future gaming devices)
    - Calling convention: Microsoft ARM64
    - Binary format: PE32+ ARM64
    - Impact: Future-proofing for Windows ARM gaming PCs
    - Files: Extend ARM64 codegen with Windows support

25. **Fix macOS ARM64 runtime issues** (HIGH)
    - Current: Binaries hang before entering main()
    - Problem: dyld/code signing/entitlements
    - Impact: Blocks macOS game distribution
    - Approach: Debug Mach-O generation, test codesigning
    - Files: `macho_builder.go`, startup code

26. **Complete Linux ARM64 support** (MEDIUM)
    - Goal: Raspberry Pi 4+ and Linux ARM gaming devices
    - Current: Basic ARM64 codegen exists
    - Need: Full ELF generation and runtime testing
    - Impact: Enables embedded/portable Linux gaming
    - Files: `arm64.go`, `elf_builder.go`

27. **Complete RISC-V 64-bit support** (LOW)
    - Goal: Future RISC-V gaming handhelds
    - Current: Instruction encoders ready
    - Need: Full codegen implementation
    - Impact: Future-proofing for RISC-V gaming
    - Files: `riscv64.go`
