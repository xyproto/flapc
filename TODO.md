# TODO for the Flap Compiler (flapc)

## High Priority
- [ ] Fix Windows/Wine PE execution: PE files are generated correctly with proper Microsoft x64 calling convention (RCX,RDX,R8,R9 + shadow space), but Wine loader returns STATUS_INVALID_IMAGE_FORMAT (c000007b). Investigation needed: import table structure, section alignments, or Wine-specific requirements. PE structure validates correctly with objdump.
- [ ] Update remaining C FFI call sites to use platform-specific calling convention helpers (getIntArgReg, allocateShadowSpace)

## Features
- [ ] Add back the "import" feature, for being able to import directly from git repos with .flap source code files.
- [ ] Add an internal utility function for sorting a Flap type (map[uint64]float64) by key. This can be needed before calling the head or tail operators.
- [ ] Rename the tail operator from `_` to `¤` and then implement/fix it, and then enable the tests.
- [ ] Fix or implement local variables in lambda bodies, if it's not implemented yet. Example: `f = x -> { y := x + 1; y }`
- [ ] Only use malloc and realloc and free in connection with arenas (internally: malloc the arena, then expand the arena with realloc as needed, then free it).
      Users can use c.malloc, c.realloc and c.free if they need anything else but the arena/alloc functionality.
      codegen.go has TODO comments where code should use the arena instead of malloc/realloc/free. Fully integrate the arena allocator.
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
