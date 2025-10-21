# Flapc TODO - x86_64 Focus

## Current Sprint (x86_64 only)
- [ ] **SIMD intrinsics** - Vector operations for audio DSP, graphics effects, particle systems
  - [x] vec2() and vec4() constructors
  - [x] VectorExpr AST node and parser support
  - [x] SIMD instruction wrappers (movupd, addpd, subpd, mulpd, divpd)
  - [ ] Vector arithmetic operations (vadd, vsub, vmul, vdiv) - needs debugging
  - [ ] Vector component access (v.x, v.y, v.z, v.w or v[0], v[1], etc.)
  - [ ] Dot product, magnitude, normalize operations
- [ ] **Register allocation improvements** - Better register usage for performance
  - [x] Binary operation optimization (use xmm2 instead of stack spills)
  - [x] Direct register-to-register moves (movq xmm, rax)
  - [x] Keep loop counters in registers (r12/r13 for range loops)
  - [ ] Register allocation for frequently-used variables
  - [ ] Full register allocator with liveness analysis
- [x] **Dead code elimination** - Remove unused variables and unreachable code
- [x] **Constant propagation** - Substitute immutable variables with their constant values
- [ ] **Function inlining** - Auto-inline small functions for performance
- [ ] **Pure function memoization** - Cache results of pure functions (future)

## Future stdlib (architecture-agnostic)

- [ ] **Collections** - Hash map, tree, queue, stack
- [ ] **String manipulation** - split, join, replace, regex
- [ ] **File I/O library** - High-level wrappers for file operations
- [ ] **Network programming** - Sockets, HTTP
- [ ] **JSON parsing and serialization** - Configuration and data exchange
- [ ] **Date/time library** - Timing and scheduling utilities
