# Flapc TODO

Actionable items sorted by importance and fundamentality for **game development with SDL3 and RayLib5**.
Current focus: x86-64 Linux. Multiplatform support (Windows/macOS/ARM64/RISC-V) deferred.
End goal: Publishable Steam games with Steamworks integration.

Complexity: (LOW/MEDIUM/HIGH/VERY HIGH)

## Recent Progress

**Test Framework Enhancement** (2025-10-24)
- Added wildcard pattern matching for non-deterministic test output
- Supports `*` wildcard for matching memory addresses, PIDs, timestamps
- Correctly handles ASCII art (doesn't treat `         *` as wildcard)
- Updated 6 test result files with wildcard patterns
- Test pass rate: 303/306 (99%)
- 3 remaining failures are documented segfault issues (#2 below)

## Critical - Compiler Correctness

1. ~~**Change 'ret' keyword to 'retval' and introduce 'reterr'**~~ ✓ COMPLETED
   - Completed in commit 858db49
   - `retval` and `reterr` keywords now implemented

2. **Fix advanced lambda/map features** (HIGH) 🔴 BLOCKED
   - Status: Programs compile successfully but crash immediately at runtime
   - Failing tests (3/306): test_lambda_match, test_map_simple, ex2_list_operations
   - Analysis:
     - `test_map_simple`: Map indexing `m[5] = 10` on scalar value generates invalid code
     - `test_lambda_match`: Recursive lambda with match expressions crashes in closure
     - `ex2_list_operations`: Pipe operator with lambdas segfaults during transformation
   - Note: Stub warnings ("Label $stub not found") are benign - PLT calls work correctly
   - Impact: Blocks advanced functional programming patterns
   - Root cause: Code generation bugs in map operations and complex lambda/match combinations
   - Files: `parser.go` (map indexing codegen, lambda match expressions, pipe operator)

3. ~~**Fix nested loops with loop-local variables**~~ ✓ COMPLETED
   - Fixed in commits 8cbb7b9 and 987e746
   - Stack offset management and iteration counter initialization corrected
   - Test status: 303/306 tests passing (99%)

4. **Implement proper closure capture for nested lambdas** (HIGH)
   - Current: Inner lambdas can't access outer lambda parameters
   - Error: "undefined variable" when lambda returns lambda
   - Impact: Blocks higher-order functional programming
   - Approach: Track captured variables, allocate environment on heap
   - Reference: Already have heap-allocated closures, extend for capture
   - Files: `parser.go:collectCapturedVarsExpr`, lambda compilation

## Critical - Game Development FFI

3. **Enhance unsafe blocks to return register values** (MEDIUM)
   - Current: unsafe blocks execute but don't return expressions
   - Goal: Return register values (rax, xmm0, etc.) as expressions
   - Syntax: `result := unsafe { rax <- 42; rax }`
   - Impact: Essential for low-level game optimizations
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

8. **Implement automatic tail-call optimization** (HIGH)
   - Goal: Detect tail-recursive calls and convert to loops automatically
   - Status: Documented in LANGUAGE.md, not yet implemented
   - Approach: Analyze function to detect tail position calls, emit loop instead of call
   - Benefits: No stack growth for tail recursion, no special keywords needed
   - Files: `parser.go:compileLambdaCall`, add tail-call detection

9. **Implement automatic memoization for pure functions** (MEDIUM)
   - Goal: Automatically cache results of pure recursive functions
   - Status: Documented in LANGUAGE.md, not yet implemented
   - Approach: Detect purity (no side effects), add result cache with function arguments as key
   - Benefits: Efficient fibonacci, dynamic programming, no manual caching
   - Note: Uses arena-based memory allocation for cache
   - Files: `parser.go:compileLambdaCall`, add purity analysis and cache logic

10. **Add trampoline execution for deep recursion** (MEDIUM)
    - Current: Non-tail-recursive functions can stack overflow
    - Goal: Handle deep recursion without TCO (e.g., tree traversal)
    - Approach: Return thunk (suspended computation), evaluate iteratively
    - Benefits: Fibonacci, tree recursion without stack limits
    - Files: New `trampoline.go`, modify lambda returns

11. **Implement precalculated stack frames** (MEDIUM)
    - Current: Dynamic stack allocation causes tracking bugs
    - Goal: Allocate entire frame at function entry (C/C++ style)
    - Benefits: No stack tracking bugs, predictable layout, easier debug
    - Approach: Calculate frame size in collectSymbols, allocate once
    - Files: `parser.go:Compile`, `parser.go:collectSymbols`

12. ~~**Implement infinite loop syntax**~~ ✓ COMPLETED (LOW)
    - Goal: `@ { ... }` without arguments for infinite loops
    - Completed in recent commits
    - Impact: Cleaner game loop syntax
    - Files: `parser.go:parseLoopStatement`

## Advanced - Optimization

13. **Add CPS (Continuation-Passing Style) transform** (VERY HIGH)
    - Goal: Convert all calls to tail calls internally
    - Benefits: Advanced control flow, no stack growth
    - Approach: Transform AST before code generation
    - Example: `f() + g()` → `f((r1) => g((r2) => r1 + r2))`
    - Note: Optional optimization pass, no IR needed
    - Files: New `cps.go`, modify compilation pipeline

14. Whole program optimization.

15. Loop unrolling and other optimization tricks.

## Language Features

16. **Add alias keyword for language packs** (MEDIUM)
    - Syntax: `alias for=@`, `alias break=@-`, `alias continue=@=`
    - Enables: python.flap, gdscript.flap style packs
    - Files: `lexer.go`, `parser.go`, new alias map

17. **Add Python-style colon + indentation** (MEDIUM)
    - Opt-in alternative to braces: `if x > 0:\n    print(x)`
    - Enables: Python/GDScript-like syntax
    - Files: `lexer.go` (indentation tracking), `parser.go`

18. **Add pattern matching on function parameters** (HIGH)
    - Syntax: `factorial := (0) => 1 | (n) => n * factorial(n-1)`
    - StandardML-style elegance
    - Files: `parser.go` lambda parsing, new pattern match system

19. **Add let bindings for local scope** (MEDIUM)
    - Syntax: `let rec loop = (n, acc) => ...`
    - Common functional pattern
    - Files: `parser.go`, new LetExpr type

20. **Extend `inf` keyword for other contexts** (LOW-MEDIUM)
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

21. **Add tail call validation in debug mode** (LOW)
    - Warn if `~tailcall>` used incorrectly
    - Helps developers write correct code

22. **Add approximate equality operator** (LOW)
    - Syntax: `0.3 =0.1= 0.2`
    - Useful for floating-point game physics comparisons

23. **Add macro system** (VERY HIGH)
    - Pattern-based code transformation
    - Enables advanced language packs

24. **Add custom infix operators** (HIGH)
    - For language packs (e.g., Python's `**`)
    - Requires precedence handling

25. **Add multi-precision arithmetic operators** (MEDIUM)
    - `++` (add with carry)
    - `<->` (swap/exchange)

## Multiplatform Support (Deferred)

These items are lower priority. Focus on x86-64 Linux first.

26. **Add Windows x64 code generation** (HIGH)
    - Goal: Compile to PE/COFF executables for Windows x64
    - Calling convention: Microsoft x64 (different from System V)
    - Binary format: PE32+ (Portable Executable)
    - Impact: Required for Steam Windows builds
    - Files: New `pe_builder.go`, `codegen_windows.go`

27. **Add Windows ARM64 code generation** (HIGH)
    - Goal: Support Windows on ARM (Surface, future gaming devices)
    - Calling convention: Microsoft ARM64
    - Binary format: PE32+ ARM64
    - Impact: Future-proofing for Windows ARM gaming PCs
    - Files: Extend ARM64 codegen with Windows support

28. **Fix macOS ARM64 runtime issues** (HIGH)
    - Current: Binaries hang before entering main()
    - Problem: dyld/code signing/entitlements
    - Impact: Blocks macOS game distribution
    - Approach: Debug Mach-O generation, test codesigning
    - Files: `macho_builder.go`, startup code

29. **Complete Linux ARM64 support** (MEDIUM)
    - Goal: Raspberry Pi 4+ and Linux ARM gaming devices
    - Current: Basic ARM64 codegen exists
    - Need: Full ELF generation and runtime testing
    - Impact: Enables embedded/portable Linux gaming
    - Files: `arm64.go`, `elf_builder.go`

30. **Complete RISC-V 64-bit support** (LOW)
    - Goal: Future RISC-V gaming handhelds
    - Current: Instruction encoders ready
    - Need: Full codegen implementation
    - Impact: Future-proofing for RISC-V gaming
    - Files: `riscv64.go`
