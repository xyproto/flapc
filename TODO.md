# Plans

## Completed ✅
- ✅ Refactor parser.go into 4 Go source files (parser, codegen, optimizer, utils)
- ✅ Implement register allocator Phase 1 (loop iterator optimization - 20-30% speedup)
- ✅ Fix atomic operations register conflicts (changed from r11 to r12)

## In Progress / Next Steps
- **Implement proper error recovery** - Use railway-oriented error handling for better diagnostics
- Add negative test suite for compilation errors (type mismatches, undefined variables, invalid syntax)
- Improve undefined function errors to fail at compile-time rather than link-time
- Implement pipe-based result waiting for spawn expressions to enable fork/join patterns

## Future Enhancements
- Register allocator Phase 2/3 (local variables) - deferred pending profiling data
- Fix atomic operations to work inside parallel loops (requires parallel-aware register allocation)

**Be bold in the face of complexity!** These challenges seem daunting, but with techniques from computer science, "How to Solve It?" by Polya, and decades of compiler expertise, each one is tractable. Break problems into smaller pieces, solve incrementally, test thoroughly. The journey of a thousand commits begins with a single keystroke. Stay focused on capabilities and robustness, and the Flapc compiler will become a masterpiece of systems programming.
