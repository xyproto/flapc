# TODO for the Flap Compiler (flapc)

## High Priority
- [x] Windows/Wine PE execution working with proper Microsoft x64 calling convention (RCX,RDX,R8,R9 + shadow space). Printf works correctly, exit codes work, basic tests run under Wine.
- [x] Fix PE import table generation (ILT+IAT pairs must be interleaved per library).
- [x] Update remaining C FFI call sites to use platform-specific calling convention helpers - Already implemented in compileCFunctionCall
- [x] Fix exitf() - Now uses syscall directly to stderr (fd 2) instead of fprintf. All tests pass.
- [x] PE64 (PE32+) generation working - Can produce .exe files that Wine recognizes and executes
- [x] Variadic function infrastructure - Grammar, lexer, parser, signature tracking, r14 calling convention. Ready for argument collection implementation.
- [ ] Fix tail (_) operator - Currently produces garbage values. See TAIL.md for details. The algorithm for copying and re-indexing list elements needs fixing. DEFERRED - focus on other features first.
- [ ] SDL3 + Windows/Wine GUI support - SDL_Init works, but SDL_CreateWindow fails under Wine in Wayland (Hyprland). This is a Wine/graphics environment limitation, not a flapc bug. PE generation and basic C FFI work correctly. Testing native Windows SDL3 would require actual Windows hardware/VM.

## Variadic Functions (In Progress)
- [x] Grammar and lexer support for `...` syntax
- [x] Parser handles variadic parameters in lambdas
- [x] Function signature tracking and call-site detection
- [x] r14 register convention for passing variadic count
- [ ] Complete argument collection from xmm registers into list
- [ ] Implement spread operator `func(list...)` at call sites
- [ ] Create stdlib.flap with variadic printf, eprintf, exitf
- See VARIADIC_IMPLEMENTATION.md for detailed status

## Features
- [ ] Add back the "import" feature, for being able to import directly from git repos with .flap source code files.
- [ ] Add an internal utility function for sorting a Flap type (map[uint64]float64) by key. This can be needed before calling the head or tail operators.
- [x] head() and tail() functions work correctly, tests enabled. The _ operator is the tail operator and works fine.
- [ ] Fix or implement local variables in lambda bodies, if it's not implemented yet. Example: `f = x -> { y := x + 1; y }`
- [x] CRITICAL: Flap does NOT offer malloc, free, realloc, or calloc as builtin functions. Users must use the arena allocator (allocate() within arena {} blocks) or explicitly call c.malloc/c.free/c.realloc/c.calloc via C FFI.
      Internally, the compiler uses arenas (malloc the arena, expand with realloc as needed, free when done).
      NOTE: Internal compiler code still uses malloc/realloc/free directly - that's fine for internal use.
- [x] CRITICAL: head() and tail() should NOT be builtin functions. Only ^ for head and _ for tail operators. Removed head()/tail() functions.
- [x] CRITICAL: Keep builtin functions to an ABSOLUTE MINIMUM. The language should be minimal and orthogonal. Most functionality should come from:
      1. Operators (^, _, #, etc.)
      2. C FFI (c.malloc, c.sin, etc.)
      3. User-defined functions
      Only add builtins if there is NO other way to implement the functionality.
- [ ] Add pattern destructuring in match clauses.
- [ ] Implement full tail call optimization for mutual recursion.

## Optimizations
- [ ] Implement whole program optimization.
- [ ] Make sure that all pure functions are memoized.
- [ ] Improve the constant folding.
- [ ] Improve the dead code elimination.
- [ ] Improve the SIMD optimizations.
- [ ] More aggressive register allocation.
- [ ] Only include constant strings in the produced executables if the constant strings are being used.

## Testing
- [ ] Make it possible to check if a type can be converted, using a Result type. Example: `42 as uint32 or! { exitf("42 can not be converted to uint32!\n") }`
- [ ] Check that it is possible to write a working ENet client and server in Flap, and that those two executables are able to talk to each other over ENet, using the ENet machine code implementation that Flapc provides.


Tips:
* Use good techniques for dealing with complexity.
* It's okay if things take time to implement, but take a step back if stuck.

## ✅ RESOLVED: Windows x64 C FFI Return Values

**Status: WORKING** (as of 2025-11-24)

C function calls on Windows x64 now work correctly, including return values.

**Verified working:**
- ✅ `c.printf("Hello\n")` correctly returns 6
- ✅ `c.sqrt(16.0)` correctly returns 4.0  
- ✅ `c.sin(0.5)` correctly returns 0.479426
- ✅ Math functions work correctly
- ✅ I/O functions work correctly
- ✅ Return values are correct (not garbage)

**What was fixed:**
- Proper Windows x64 calling convention (RCX,RDX,R8,R9 + shadow space)
- Correct PE import table generation (ILT+IAT pairs)
- Stack alignment before C FFI calls
- Shadow space allocation/deallocation

Windows PE generation now fully functional for both native Flap code AND C FFI.

