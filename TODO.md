# Flapc Master TODO - Actionable Roadmap

**Current Status:** v1.7.4 preparation
**Language Status:** Ready for freeze after completing critical items
**Next Major Version:** v2.0 (post-freeze features)

---

## 🚨 IMMEDIATE: v1.7.4 Release (Language Freeze)

### ✅ COMPLETED
1. Move semantics (`!` operator) - DONE
2. Constant folding (all operators) - DONE
3. Dead code elimination - DONE
4. Magic number elimination - DONE
5. Optimization documentation (OPTIMIZATIONS.md) - DONE
6. Game development readiness (GAME_DEVELOPMENT_READINESS.md) - DONE

### 🔥 CRITICAL (Must complete before v1.7.4)
**None remaining** - All critical items completed!

### 📊 QUALITY (Remaining for v1.7.4)
- ⏳ **Test edge cases** (parallel loops: empty range, single iteration, large ranges)
- ⏳ **Improve error messages** (undefined variable suggestions, FFI type mismatches)
- ⏳ **Documentation review** (LANGUAGE.md completeness check)

### ✅ v1.7.4 Release Checklist
- [ ] All tests pass (`go test`)
- [ ] All 363+ testprograms pass
- [ ] LANGUAGE.md marked as frozen
- [ ] README updated with freeze notice
- [ ] Git tag v1.7.4
- [ ] Announce language freeze

**Estimated time to v1.7.4:** 1-2 days (quality polish only)

---

## 📅 POST-FREEZE: v2.0 Major Features

### Priority 1: Advanced Move Semantics (8 weeks)
**Goal:** Rust-level safety with simpler syntax

#### Phase 1: Borrowing (3 weeks)
- [ ] Implement `&T` (immutable borrow) syntax
- [ ] Implement `&mut T` (mutable borrow) syntax
- [ ] Borrow checker (compile-time ownership analysis)
- [ ] Error messages for borrow violations
- [ ] Test suite (concurrent borrows, use-after-move)

#### Phase 2: Move-Only Types (2 weeks)
- [ ] Add `movable` keyword for type definitions
- [ ] Compiler enforcement (prevent copies)
- [ ] RAII pattern with `drop` destructor
- [ ] Examples: FileHandle, UniquePtr

#### Phase 3: Advanced Features (3 weeks)
- [ ] Lifetime annotations (`<'a>`) for complex cases
- [ ] Pattern matching with moves
- [ ] Move chains for fluent APIs
- [ ] Collection move semantics

**Reference:** MOVE_SEMANTICS_IMPROVEMENTS.md

---

### Priority 2: Channels (CSP Concurrency) (7 weeks)
**Goal:** Go-style channels for safe thread communication

#### Phase 1: Channel MVP (3 weeks)
- [ ] Implement `chan()` builtin (create channel)
- [ ] Implement `<-` send operator (`ch <- value`)
- [ ] Implement `<-` receive operator (`value := <-ch`)
- [ ] Implement `close(ch)` builtin
- [ ] Futex-based mutex/condvar for Linux
- [ ] Buffered channels (`chan(capacity)`)

#### Phase 2: Spawn Syntax (1 week)
- [ ] Add `spawn { ... }` syntax (lightweight threads)
- [ ] Compile to single-iteration parallel loop
- [ ] Integration with channel operations
- [ ] Test: producer-consumer pattern

#### Phase 3: Select Statement (2 weeks)
- [ ] Add `select { }` syntax (multiplex channels)
- [ ] Implement channel polling (non-blocking check)
- [ ] Add `timeout()` case
- [ ] Random selection if multiple ready

#### Phase 4: Documentation (1 week)
- [ ] CHANNELS.md - Usage guide
- [ ] CSP_EXAMPLES.md - Patterns (pipelines, workers)
- [ ] Performance benchmarks

**Reference:** CHANNELS_AND_ENET_PLAN.md

---

### Priority 3: Railway Error Handling (7 weeks)
**Goal:** Rust-style Result type with `?` operator

#### Phase 1: Result Type (1 week)
- [ ] Define `Result` cstruct (ok, value, error)
- [ ] Implement `Ok(value)` helper
- [ ] Implement `Err(msg)` helper
- [ ] Manual usage examples (file I/O, networking)

#### Phase 2: `?` Operator (2 weeks)
- [ ] Lexer: Recognize `?` as postfix operator
- [ ] Parser: Parse `expr?` as error propagation
- [ ] Compiler: Desugar to early return
- [ ] Type checker: Ensure function returns Result
- [ ] Test suite (nested propagation, complex control flow)

#### Phase 3: Pattern Matching (3 weeks)
- [ ] Add `match` keyword
- [ ] Implement pattern matching for structs
- [ ] Support wildcards (`_`)
- [ ] Exhaustiveness checking
- [ ] Syntax: `result match { Ok(v) -> ..., Err(e) -> ... }`

#### Phase 4: Combinators (1 week)
- [ ] Implement `map(r, f)` - transform success value
- [ ] Implement `and_then(r, f)` - chain operations
- [ ] Implement `or_else(r, default)` - provide fallback
- [ ] Implement `unwrap(r)` - panic if error
- [ ] Implement `unwrap_or(r, default)` - safe unwrap

**Reference:** RAILWAY_ERROR_HANDLING_PLAN.md

---

### Priority 4: ENet Integration (2 weeks)
**Goal:** Production-ready multiplayer game networking

#### Phase 1: ENet FFI (1 week)
- [ ] Import ENet library via FFI (`import enet as enet`)
- [ ] Extract constants automatically (DWARF)
- [ ] Create example: Simple server
- [ ] Create example: Simple client
- [ ] Test: Connection, send/receive, disconnect

#### Phase 2: High-Level Wrapper (1 week)
- [ ] Implement `enet_server(port, max_clients)`
- [ ] Implement `enet_client(host, port)`
- [ ] Callback-based API (on_connect, on_receive, on_disconnect)
- [ ] Automatic packet management
- [ ] Example: Multiplayer game server

**Reference:** CHANNELS_AND_ENET_PLAN.md

---

## 🚀 FUTURE: v3.0+ (Long-Term Vision)

### Optimization Enhancements
- [ ] **Auto-vectorization** (detect SIMD-friendly loops)
- [ ] **Profile-Guided Optimization** (PGO) support
- [ ] **Escape analysis** (stack allocations for local-only data)
- [ ] **Common Subexpression Elimination** (CSE)
- [ ] **Strength reduction** (x * 2 → x << 1)
- [ ] **Loop-invariant code motion** (LICM)

### Language Features
- [ ] **Tagged unions** (enum-style sum types)
- [ ] **Generics** (type parameters for functions/structs)
- [ ] **Traits/interfaces** (polymorphism without inheritance)
- [ ] **Compile-time execution** (const functions)
- [ ] **Macros** (AST-level metaprogramming)

### Platform Ports
- [ ] **Windows x86_64** (cross-compilation or native)
- [ ] **macOS ARM64** (Apple Silicon)
- [ ] **macOS x86_64** (Intel Macs)
- [ ] **FreeBSD x86_64**
- [ ] **RISC-V 64** (emerging platform)
- [ ] **WebAssembly** (browser games)

### Tooling
- [ ] **Language Server Protocol (LSP)** (editor integration)
- [ ] **Debugger** (GDB integration or custom DWARF)
- [ ] **Package manager** (dependency management)
- [ ] **Build system** (Makefile generator)
- [ ] **Formatter** (code style enforcement)

### Game Development
- [ ] **Asset pipeline** (textures, sounds, models)
- [ ] **Hot reloading** (live code updates)
- [ ] **Profiler** (frame time, memory, allocations)
- [ ] **Crash reporter** (automatic error reporting)

---

## 📝 DOCUMENTATION STATUS

### ✅ Complete
- LANGUAGE.md (language specification)
- GAME_DEVELOPMENT_READINESS.md (Steam game readiness)
- OPTIMIZATIONS.md (compiler optimizations reference)
- MOVE_SEMANTICS_IMPROVEMENTS.md (future move semantics plan)
- CHANNELS_AND_ENET_PLAN.md (concurrency and networking plan)
- RAILWAY_ERROR_HANDLING_PLAN.md (error handling strategy)

### ⏳ Needs Review/Update
- README.md (add freeze notice)
- LANGUAGE.md (mark as frozen after v1.7.4)

### 📝 Planned (Post-Freeze)
- CHANNELS.md (channel usage guide)
- CSP_EXAMPLES.md (concurrency patterns)
- ERROR_HANDLING.md (Result type guide)
- ENET_GUIDE.md (multiplayer networking tutorial)
- MULTIPLAYER_PATTERNS.md (game networking patterns)
- BORROWING.md (ownership and borrowing guide)
- MIGRATION_GUIDE.md (porting code to new features)

---

## 🧪 TESTING STATUS

### Current Coverage
- 363+ integration tests in `testprograms/`
- Unit tests: `go test` passes
- Test types: Features, edge cases, correctness

### Test Gaps (For v1.7.4)
- [ ] Parallel loop edge cases (empty, single, huge)
- [ ] Error message quality tests
- [ ] Negative tests (intentional compile errors)

### Test Gaps (Post-Freeze)
- [ ] Channel tests (send/receive, close, select)
- [ ] Borrow checker tests (use-after-move, concurrent borrows)
- [ ] Result type tests (propagation, nesting)
- [ ] ENet tests (connection, packet loss, bandwidth)

---

## 📊 CURRENT METRICS

### Code Statistics
- Lines of Go code: ~20,000
- Test programs: 363+
- Supported platforms: Linux x86_64 (primary)

### Performance
- Compilation speed: ~8,000-10,000 LOC/s
- Binary size: Small (~20KB for simple programs)
- Runtime performance: Comparable to C (no runtime overhead)

### Optimization Coverage
- ✅ Constant folding (all operators)
- ✅ Dead code elimination
- ✅ Function inlining
- ✅ Loop unrolling
- ✅ Tail call optimization
- ✅ SIMD vectorization (selective)
- ✅ Whole program optimization (WPO)
- ✅ Parallel loops (@@)
- ✅ Atomic operations
- ✅ Arena allocators
- ✅ Move semantics (!)

---

## 🎯 MILESTONES

### v1.7.4 (Q1 2025) - Language Freeze
**Goal:** Stable, frozen language specification
**Status:** 95% complete
**Blockers:** Quality polish (error messages, edge cases)

### v2.0 (Q3 2025) - Safety and Concurrency
**Goal:** Borrowing, channels, error handling
**Features:**
- Advanced move semantics (Rust-level safety)
- CSP channels (Go-style concurrency)
- Railway error handling (Result + ?)
- ENet integration (multiplayer games)

### v3.0 (Q4 2025+) - Platform Expansion
**Goal:** Cross-platform support
**Features:**
- Windows, macOS, FreeBSD ports
- LSP for editor integration
- Package manager
- Advanced optimizations (PGO, auto-vectorization)

---

## 🚧 KNOWN ISSUES

### None Critical
All critical bugs resolved in v1.7.3.

### Minor (Non-Blocking)
- Verbose error messages in some cases
- Test suite race conditions (fixed)

---

## 📞 COMMUNITY & SUPPORT

### Reporting Issues
- GitHub: https://github.com/xyproto/flapc/issues

### Contributing
- After v1.7.4: Language frozen, contributions limited to:
  - Bug fixes
  - Platform ports
  - Optimizations
  - Tooling
  - Documentation

### Discussion
- Language design decisions: Closed after v1.7.4 freeze
- Feature requests: For v2.0+ only

---

## 🎉 SUCCESS CRITERIA

### v1.7.4 Release Ready When:
- [x] All critical bugs fixed
- [x] Optimizations implemented (constant folding, DCE, etc.)
- [ ] Error messages improved
- [ ] Edge cases tested
- [x] Documentation complete
- [ ] Tests passing (go test && integration tests)
- [ ] Freeze notice added to README
- [ ] Git tag v1.7.4 created

### v2.0 Release Ready When:
- [ ] Borrowing fully implemented and tested
- [ ] Channels working (send, receive, select, spawn)
- [ ] Railway error handling operational (Result + ?)
- [ ] ENet integration validated with example game
- [ ] Documentation complete for all new features
- [ ] Migration guide available

---

## 📖 REFERENCES

### Internal Docs
- [LANGUAGE.md](LANGUAGE.md) - Language specification
- [OPTIMIZATIONS.md](OPTIMIZATIONS.md) - Compiler optimizations
- [GAME_DEVELOPMENT_READINESS.md](GAME_DEVELOPMENT_READINESS.md) - Steam readiness
- [MOVE_SEMANTICS_IMPROVEMENTS.md](MOVE_SEMANTICS_IMPROVEMENTS.md) - Borrowing plan
- [CHANNELS_AND_ENET_PLAN.md](CHANNELS_AND_ENET_PLAN.md) - Concurrency plan
- [RAILWAY_ERROR_HANDLING_PLAN.md](RAILWAY_ERROR_HANDLING_PLAN.md) - Error handling plan

### External References
- Rust: Ownership, borrowing, Result type
- Go: Channels, goroutines, select
- F#: Railway-oriented programming
- ENet: https://github.com/lsalzman/enet

---

## 🗓️ TIMELINE ESTIMATES

| Version | Features | Duration | Status |
|---------|----------|----------|--------|
| v1.7.4 | Language freeze | 1-2 days | 95% |
| v2.0 | Safety + Concurrency | 24 weeks | Planned |
| v3.0 | Platforms + Tooling | 12+ weeks | Future |

**Total estimated effort for v2.0:** 6 months (part-time)

---

## ✅ NEXT ACTIONS

### This Week (v1.7.4 Finalization)
1. Test parallel loop edge cases
2. Improve error messages (undefined variables, FFI type mismatches)
3. Review LANGUAGE.md for completeness
4. Run full test suite 10 times (race condition check)
5. Add freeze notice to README
6. Tag v1.7.4 and announce

### Next Month (v2.0 Planning)
1. Finalize borrow checker design
2. Prototype channel implementation
3. Create Result type MVP
4. Set up ENet build system
5. Write migration guide outline

---

## 📞 MAINTAINER NOTES

**Language Design Philosophy:**
- Correctness over convenience
- Explicitness over magic
- Zero-cost abstractions
- No hidden control flow
- Predictable performance

**Backward Compatibility Promise:**
- v1.7.4: Last version with language changes
- v2.0+: Only additions (no removals/changes)
- Old programs continue to work

**Contribution Focus Post-Freeze:**
- Optimizations (yes)
- Bug fixes (yes)
- Ports (yes)
- Tooling (yes)
- Language changes (NO - frozen)

---

**Last Updated:** 2025-10-31
**Maintainer:** Alexander F. Rødseth (@xyproto)
