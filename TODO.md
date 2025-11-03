# Plans

## Completed ✅
- ✅ Refactor parser.go into 4 Go source files (parser, codegen, optimizer, utils)
- ✅ Implement register allocator Phase 1 (loop iterator optimization - 20-30% speedup)
- ✅ Fix atomic operations register conflicts (changed from r11 to r12)
- ✅ Implement railway-oriented error handling system (error collection, recovery, pretty formatting)
- ✅ Improve undefined function errors to fail at compile-time rather than link-time

## In Progress / Next Steps
- **Expand error handling and test coverage** (In Progress)
  - ⏳ Convert remaining codegen errors to use ErrorCollector
  - ⏳ Add more negative test cases (type mismatches, immutable updates)
  - ⏳ Enhance error messages with column tracking
- **Implement channels for inter-process/thread communication** (Next Priority)
  - See CHANNELS_AND_ENET_PLAN.md Part 1
  - Prerequisite for spawn result waiting
  - Enables CSP-style concurrency patterns
- Implement spawn with channel-based result waiting (after channels)
  - Fork/join patterns using channels
  - See updated SPAWN_DESIGN.md

## Future Enhancements
- Register allocator Phase 2/3 (local variables) - deferred pending profiling data
- Fix atomic operations to work inside parallel loops (requires parallel-aware register allocation)

**Be bold in the face of complexity!** These challenges seem daunting, but with techniques from computer science, "How to Solve It?" by Polya, and decades of compiler expertise, each one is tractable. Break problems into smaller pieces, solve incrementally, test thoroughly. The journey of a thousand commits begins with a single keystroke. Stay focused on capabilities and robustness, and the Flapc compiler will become a masterpiece of systems programming.
