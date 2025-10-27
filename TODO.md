# Flapc TODO - Version 1.6 Release

Actionable items for **Flap 1.6 release**.
Current focus: x86-64 Linux. Multiplatform support deferred to post-1.6.
End goal: Minimal, elegant, implementable language ready for game development.

Complexity: (LOW/MEDIUM/HIGH/VERY HIGH)

## Critical - Language Core (1.6 Blockers)

1. **Implement ENet networking in assembly** (VERY HIGH) 🔥 CRITICAL FOR 1.6
   - Goal: Built-in reliable UDP networking as fundamental language feature
   - Implementation: Generate ENet protocol machine code directly in Flapc
   - NOT an external library - core part of the compiler like printf/malloc
   - Syntax support: `:port` literals, `@ msg, from in :port` loops, `:port <- data` sending
   - Port literals: `:5000` (numeric), `:game_server` (string, hashed), `:5000+` (next available), `:5000?` (check)
   - Port fallback: `:5000 or :5001` using `or` operator
   - String port hashing: Deterministic hash to port numbers
   - Protocol features: Connection management, reliable/unreliable channels, packet fragmentation
   - Files: New `enet_codegen.go` (protocol implementation), extend `parser.go` (port literals, message loops)

2. **Implement parallel loops runtime** (HIGH) 🔥 CRITICAL FOR 1.6
   - Goal: CPU parallelism with `N @` and `@@` syntax
   - Implementation: Thread pool with work stealing, OpenMP-style execution
   - Syntax: `4 @ item in data max 1000 { }` (4 cores), `@@ item in data max 1000 { }` (all cores)
   - Thread safety: Need atomics, mutex builtins for shared state
   - Work distribution: Chunk-based splitting, load balancing
   - Files: New `parallel.go` (thread pool, work queue), extend `parser.go` (parallel loop codegen)

3. **Implement `fork() &` background processes** (MEDIUM) 🔥 CRITICAL FOR 1.6
   - Goal: Process-based concurrency with `&` operator
   - Implementation: Unix fork() for isolated processes
   - Syntax: `worker_function() &` spawns process in background
   - Process management: Track PIDs, cleanup on exit
   - Files: Extend `parser.go` (fork codegen), new `process.go` (process lifecycle)

4. **Complete live hot reload integration** (HIGH) 🎮 GAMEDEV - CRITICAL FOR 1.6
   - Goal: Live code patching in running game processes (infrastructure 90% complete)
   - Status: Foundation exists but missing final integration step
   - ✅ Complete: Function indirection table, memory allocation, file watching, code extraction
   - ❌ Missing: Live injection into running process

   **Remaining Work:**
   - Wire up watch mode to keep game process running (don't rebuild binary)
   - On file change: Extract only changed hot function machine code
   - Inject extracted code using `HotReloadManager.ReloadHotFunction()`
   - Update function pointer table atomically in running process
   - Handle compilation errors (keep old code running)

   **Integration Steps:**
   1. Modify `watchAndRecompile()` to not restart process
   2. Use `IncrementalState` to detect which hot functions changed
   3. Compile only changed functions to temporary binary
   4. Extract changed function code with `ExtractFunctionCode()`
   5. Call `HotReloadManager.ReloadHotFunction()` to patch live

   **Testing:**
   - Test game with hot physics constant (gravity, jump height)
   - Test hot render function (change colors, sizes)
   - Test compilation error recovery (old code stays active)

   Files: Wire together `main.go`, `hotreload.go`, `incremental.go`, `filewatcher.go`

## Important - Language Features (1.6 Nice-to-Have)

5. **Add Steamworks FFI support** (HIGH) 🎮 GAMEDEV
   - Goal: Steam achievements, leaderboards, cloud saves for commercial releases
   - Approach: Extend C FFI to handle Steamworks SDK callbacks and structs
   - Requirements: Handle C++ name mangling, callback function pointers
   - Impact: Required for publishing on Steam
   - Files: New `steamworks.go`, extend FFI in `parser.go`

## Future Work (Post-1.6)

These features are deferred until after 1.6 release:

6. **Add trampoline execution for deep recursion** (MEDIUM)
   - Handle deep recursion without TCO (tree traversal, Ackermann)
   - Return thunk (suspended computation), evaluate iteratively
   - Files: New `trampoline.go`

7. **Add let bindings for local scope** (LOW)
   - Syntax: `let rec loop = (n, acc) => ...`
   - StandardML/OCaml-style local recursive definitions
   - Files: `parser.go`, new LetExpr in `ast.go`

8. **Add Python-style colon + indentation** (MEDIUM)
   - Alternative syntax for language packs: `if x > 0:\n    print(x)`
   - Track indentation levels in lexer, emit virtual braces
   - Files: `lexer.go`, `parser.go`

9. **Add CPS (Continuation-Passing Style) transform** (VERY HIGH)
   - Convert all calls to tail calls internally
   - Advanced control flow, no stack growth
   - Files: New `cps.go`

10. **Add macro system** (VERY HIGH)
    - Pattern-based code transformation at parse time
    - Enables language packs and metaprogramming
    - Files: New `macro.go`

11. **Add custom infix operators** (HIGH)
    - For language packs (e.g., Python's `**`)
    - Files: Extend `parser.go` precedence

12. **Multiplatform support** (Deferred to post-1.6)
    - Windows x64 (PE/COFF)
    - Windows ARM64
    - macOS ARM64 (mostly complete, needs testing)
    - Linux ARM64 (Raspberry Pi)
    - RISC-V 64-bit
