# Flapc TODO - Version 1.6 Release

**Target**: x86-64 Linux game development
**Philosophy**: Minimal, elegant, implementable

---

## 📊 Current Status

**Date**: 2025-10-28
**Version**: 1.6 (preparing for release)

**Recent Completions:**
- ✅ Cross-platform unsafe blocks with register aliases (a→rax/x0/a0, etc.)
- ✅ Unified and per-CPU unsafe syntax
- ✅ Documentation consolidation (README.md + GRAMMAR.md)
- ✅ Parallel loop parser with reducer syntax
- ✅ V4-V6 futex barrier synchronization
- ✅ Full loop body execution in parallel loops
- ✅ External variable access in child threads

**Files**: `register_alias.go`, `GRAMMAR.md`, `atomic.go`, `parallel.go`, `lexer.go`, `parser.go`, `ast.go`

---

## 🔥 Critical Path to 1.6

### 1. Parallel Loops - Finish Reducers

**Current Status**: V5 complete! Parallel loops work with full loop bodies and external variables.

**Next: Step 8 - Reduction Operations**

Implement parallel loop reducers for combining thread-local results:

```flap
sum = @@ i in 0..<N { i * i } | a,b | { a + b }
```

**Subtasks:**
- [x] 8a: Extend AST with Reducer field
- [x] 8b: Parse `| params | { body }` syntax
- [ ] 8c: Generate code for partial result storage (thread-local)
- [ ] 8d: Generate code for atomic combination (LOCK or mutex)
- [ ] 8e: Test with simple addition

**Optional Future Steps:**
- [ ] Step 9: Atomic operations for shared state (LOCK XADD, CMPXCHG)
- [ ] Step 10: Printf support in parallel loops (via wrapper or PIC)
- [ ] V7: Dynamic range bounds (currently constants only)

---

### 2. Hot Reload - Phase 2: True Hot Patching

**Current Status**: Phase 1 complete (smart restart when hot functions change)

**Next: Phase 2 - IPC-based code injection**

**Pending Steps:**
- [ ] Shared memory setup (mmap'd code region + function pointer table)
- [ ] Extract machine code with `ExtractFunctionCode()`
- [ ] IPC signaling (Unix socket)
- [ ] Atomic pointer updates with memory barriers
- [ ] Test: change getValue() from 42 → 100 while running

**Files**: `main.go`, `hotreload.go`, `incremental.go`

---

### 3. Networking Polish

Basic UDP works. Add quality-of-life features:

**Step 1: Error Handling**
- [ ] Check sendto()/recvfrom() return values
- [ ] Print error messages with errno
- [ ] Continue on failure instead of crashing

**Step 2: Parse Received Data**
- [ ] Convert bytes to Flap string
- [ ] Extract sender IP and port from sockaddr_in
- [ ] Store in `msg` and `from` variables

**Step 3: Connection Tracking (Optional)**
- [ ] Hash map for sender addresses
- [ ] Track last_seen timestamp
- [ ] Timeout stale connections (60s)

**Test Case:**
```flap
@ msg, from in ":5000-5010" {
    printf("From %v: %v\n", from, msg)
}
```

**Files**: `parser.go`

---

## 📋 Optional Nice-to-Haves

### Atomic Operations
For thread-safe shared state in parallel loops:
- [ ] `atomic_add(ptr, value)` builtin (LOCK XADD)
- [ ] `atomic_cas(ptr, old, new)` builtin (LOCK CMPXCHG)
- [ ] `mutex_lock(ptr)` / `mutex_unlock(ptr)` (futex)
- [ ] Test: increment shared counter from 4 threads

### Steamworks FFI
For commercial Steam releases:
- [ ] Parse C++ header files
- [ ] Handle name mangling
- [ ] Support callback function pointers
- [ ] Achievement and leaderboard wrappers
- [ ] Test: unlock achievement from Flap

---

## 🎯 1.6 Release Checklist

**Core Features:**
- [x] UDP networking (send/receive)
- [x] Port availability and fallback
- [x] Hot reload infrastructure
- [x] Spawn background processes
- [x] Tail call optimization
- [~] Parallel loops (V5 complete: full loop bodies work, reducers pending)

**Quality:**
- [x] Parallel loops: futex barrier synchronization verified
- [ ] Parallel loops: test reducers with 10k items
- [ ] Hot reload: test changing physics constants live
- [ ] Networking: test 1000 messages/second throughput
- [ ] Clean VM: install and run on fresh Ubuntu 22.04
- [ ] Memory: run valgrind, fix any leaks

**Documentation:**
- [x] README.md comprehensive (3433 lines with all features)
- [x] GRAMMAR.md extracted (1234 lines)
- [x] TODO.md updated
- [ ] Add parallel loop examples to testprograms/
- [ ] Write networking tutorial (client + server)
- [ ] Document hot reload workflow

---

## 🚀 Post-1.6 Ideas

Deferred until after 1.6 ships:

- **ENet Protocol**: Reliable channels, packet ordering, ACKs
- **Trampolines**: Deep recursion without stack overflow
- **Macros**: Compile-time metaprogramming
- **CPS Transform**: All function calls become tail calls
- **Multiplatform**: Windows, macOS ARM, RISC-V
- **Python Syntax**: Colon + indentation alternative
- **Let Bindings**: Local recursive definitions

---

The core is solid. Focus: parallel loop reducers, then release!
